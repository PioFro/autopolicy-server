/*
 * Autopolicy PoC
 * Copyright (C) 2020-2020 IITiS PAN Gliwice <https://www.iitis.pl/>
 * Author: Piotr Frohlich <pfrohlich@iitis.pl>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
package main

import (
	"ap-server/models"
	"errors"
	mongodm "github.com/zebresel-com/mongodm"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

type MongoDB struct {
	con *mongodm.Connection
	devices *mongodm.Model
	ports *mongodm.Model
	apids *mongodm.Model
	profiles *mongodm.Model
	flows *mongodm.Model
}

func NewMongoDB(S *Server) *MongoDB{


	dbConfig := &mongodm.Config{
		DatabaseHosts: []string{S.opts.db},
		DatabaseName: "apserver",
	}
	con,err  := mongodm.Connect(dbConfig)
	if err !=nil{
		dieErr("Database Connection",errors.New("Connection to database failed."))
	}
	con.Register(&models.Device{},"devices")
	con.Register(&models.Profile{},"profiles")
	con.Register(&models.Identity{},"identities")
	con.Register(&models.OccupiedPort{},"ports")
	con.Register(&models.FlowDescription{},"flows")

	return &MongoDB{
		con:      con,
		devices:  con.Model("Device"),
		ports:    con.Model("OccupiedPort"),
		apids:    con.Model("Identity"),
		profiles: con.Model("Profile"),
		flows: con.Model("FlowDescription"),
	}
}

func (db *MongoDB) GetPortOnSwitch(id string, mac string, port string) (*models.OccupiedPort,error){
	device,err := db.GetSwitchById(id,false)
	if err!=nil{
		return nil, err
	}
	isSomeoneOnMyPort:=false
	if ps, ok := device.Ports.([]*models.OccupiedPort); ok{
		dbg(2,"APID verification","Got ports for switch",db.ports,device)
		for _,p := range ps{
			// For each port check if we've already seen such mac-port pair
			if p.Port==port {
				// Check if port isn't occupied by some other mac
				if p.MAC == mac {
					// Update updatable fields of AP id. See tags in the model
					dbg(1, "APID verification","Resubmission of the identity",id)
					dbg(2, "APID verification", "Checking if only updatable fields are updated")
					return p,nil
				}else {
					isSomeoneOnMyPort = true
				}
			}
		}
	}
	if isSomeoneOnMyPort{return nil, err_downgrade}
	return nil, errors.New("Mac/port not found on switch")
}

func (db *MongoDB) GetSwitchById(id string, createOnNotFound bool) (*models.Device,error){
	mdevice := &models.Device{}
	err:=db.devices.FindOne(bson.M{"@switch":id}).Populate("Ports").Exec(mdevice)
	// Check if we have such switch in the database
	if err!=nil{
		if !createOnNotFound {
			return nil, err
		}else if _, ok := err.(*mongodm.NotFoundError); ok {
			//no records were found
			dbg(2, "Get switch by Id","Device with id not found. Creating device with id ",id)
			db.devices.New(mdevice)
			mdevice.SwitchId = id
			mdevice.Ports = make([]*models.OccupiedPort,0)
			// Create such switch with 0 ports connected
			err := mdevice.Save()
			if err != nil{
				dbgErr(1,"Get switch by Id",err)
				return nil, err
			}
			dbg(2, "Get switch by Id","Creatied device with id ",id)
		} else if err != nil {
			// DB connection or assertion failed
			dbgErr(1,"Get switch by Id",err)
			return nil, err
		}
	}
	return mdevice,nil
	}
//Checks if such an Id isn't already on the port. Or if the port is already taken by some other mac.
func (db *MongoDB) Verify(id Identity) (Identity, error) {
	mac := id["@mac"]
	device := id["@switch"]
	port := id["@port"]
	mdevice := &models.Device{}
	apid := &models.Identity{}
	//err:=devices.FindOne(bson.M{"@switch":device}).Populate("Ports").Exec(mdevice)
	mdevice,err := db.GetSwitchById(device,true)
	if err != nil{
		dbgErr(1,"APID verification",err)
		return nil, err
	}
	// Check if the switch has any ports already connected
	connectedPort,err:=db.GetPortOnSwitch(device,mac,port)
	if connectedPort!=nil{
		db.apids.FindId(bson.ObjectIdHex(connectedPort.APID)).Exec(apid)
		return apid.GetRegularMap(device,mac,port),nil
	}
	if err == err_downgrade{
		return nil, err_downgrade
	}
	// New mac on existing device. There is no such APID before in the database. Proceed with adding it
	if ps, ok := mdevice.Ports.([]*models.OccupiedPort); ok{
		dbg(3, "APID verification","Creating APID record",id)
		query := id.getQuery()
		// See if there is such a APID in the database (some other similar device)
		err := db.apids.FindOne(query).Exec(apid)
		if _, ok := err.(*mongodm.NotFoundError); ok {
			db.apids.New(apid)
			apid.FillFromMap(id)
			apid.Save()
		}else if err != nil{
			dbgErr(1,"APID verification",err)
		}
		// Create and save new port for device
		newPort := &models.OccupiedPort{}
		db.ports.New(newPort)
		newPort.Port = port
		newPort.MAC = mac
		// Strip the hex id to only hex
		newPort.APID = strings.Replace(strings.Replace(apid.Id.String(),"ObjectIdHex(\"","",-1),"\")","",-1)
		// Placeholder for profile
		newPort.ProfileId = "TBA"
		newPort.Save()
		// Add port to the device
		mdevice.Ports = append(ps,newPort)
		errsave := mdevice.Save()
		if errsave!=nil{
			return nil, errsave
		}
	}
	return id,nil
}
func (db *MongoDB) Authorize(id Identity) (pf Profile, err error) {
	// Get APID for port
	mac := id["@mac"]
	device := id["@switch"]
	port := id["@port"]

	mport,err := db.GetPortOnSwitch(device,mac,port)
	if err!=nil{
		return nil, err
	}
	// Search for the apid for the port/mac/device triplet
	apid := &models.Identity{}
	err = db.apids.FindId(bson.ObjectIdHex(mport.APID)).Exec(apid)
	// Did not found referred to identity
	if err!= nil{
		return nil, err
	}
	profile:=&models.Profile{}
	// Search for Profile for this APID
	// Try insert new profile
	err = db.profiles.FindOne(bson.M{"apid":apid.Id}).Populate("Flows").Exec(profile)
	if err !=nil{
		dbgErr(1,"Profile verification",err)
		return nil, ProfileNotFound
	}
	if fs, ok := profile.Flows.([]*models.FlowDescription); ok{
		dbg(1,"Profile verification","Able to gather flows",fs)
		return profile.GetMap()
	}
	// No profile? -> post to profiling
	return nil, errors.New("Unable to populate flows for profile with id "+ profile.Id.String())
}
func (db *MongoDB)AddProfile(profile Profile, id Identity)(Profile,error){
	mac := id["@mac"]
	device := id["@switch"]
	port := id["@port"]
	mport,err := db.GetPortOnSwitch(device,mac,port)
	if err!=nil{
		return nil, err
	}
	// Search for the apid for the port/mac/device triplet
	apid := &models.Identity{}
	err = db.apids.FindId(bson.ObjectIdHex(mport.APID)).Exec(apid)
	// Did not found referred to identity
	if err!= nil{
		return nil, err
	}
	mprofile := &models.Profile{}
	db.profiles.New(mprofile)
	err = mprofile.LoadFromMap(profile, apid.Id)
	if err!=nil{
		return nil, err
	}
	if fds, ok :=mprofile.Flows.([]*models.FlowDescription);ok{
		for _,fd:= range fds{
			db.flows.New(fd)
			fd.Save()
		}
	}
	mprofile.Save()
	return nil, nil
}