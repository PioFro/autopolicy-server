package models

import (
	"encoding/json"
	"github.com/zebresel-com/mongodm"
	)

type Identity struct {
	mongodm.DocumentBase `json:",inline" bson:",inline"`
	Manufacturer string `json:"manufacturer" bson:"manufacturer"`
	DeviceModel string `json:"device" bson:"device"`
	Revision string `json:"revision" bson:"revision"`
	Version string `json:"version" bson:"version"`
	IsCreated string `json:"is-created" bson:"is-created"`
}

func (apid *Identity) FillFromMap(id map[string]string){
	apid.Manufacturer = id["manufacturer"]
	apid.DeviceModel = id["device"]
	apid.Revision = id["revision"]
	apid.Version=id["version"]
	apid.IsCreated = id["is-created"]
}
func (apid Identity) GetRegularMap(device string, mac string, port string)(map[string]string) {
	var id = make(map[string]string)
	id["@switch"] = device
	id["@port"] = port
	id["@mac"] = mac
	jme, _ := json.Marshal(apid)
	m := make(map[string]string)
	json.Unmarshal(jme,&m)
	for i := range m{
		if i!="id" {
			id[i] = m[i]
		}
	}
	return id
}