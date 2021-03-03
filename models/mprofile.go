package models

import (
	"errors"
	"fmt"
	"github.com/zebresel-com/mongodm"
	"strings"
)

type Profile struct {
	mongodm.DocumentBase `json:",inline" bson:",inline"`
	Flows                interface{} `json:"from_device" bson:"from_device" model:"FlowDescription" relation:"1n" autosave:"true"`
	APID                 interface{} `json:"apid" bson:"apid" model:"Identity" relation:"11" autosave:"true"`
	IsCreated            bool        `json:"is-created" bson:"is-created"`
}
type FlowDescription struct {
	mongodm.DocumentBase `json:",inline" bson:",inline"`
	Flow                 string `json:"flow" bson:"flow"`
	Type                 string `json:"type" bson:"type"`
}

func (p Profile) GetMap() (map[string]interface{}, error) {

	profile := make(map[string]interface{})
	allow := make(map[string]interface{})
	flows := make([]string, 0)
	if fs, ok := p.Flows.([]*FlowDescription); ok {
		for _, f := range fs {
			if strings.ToLower(f.Type) == "allow" {
				flows = append(flows, f.Flow)
			}
		}
	} else {
		return nil, errors.New("Unable to parse flows of the profile. ")
	}
	allow["allow"] = flows
	profile["from_device"] = allow
	return profile, nil
}
func (p *Profile) LoadFromMap(profile map[string]interface{}, apid interface{}) error {
	p.IsCreated = true
	p.APID = apid
	flows := make([]*FlowDescription, 0)
	fromDevice := profile["from_device"]
	allow, ok := fromDevice.(map[string]interface{})
	if !ok {
		return errors.New("Unable to reach from_device -> allow")
	}
	allows, ok := allow["allow"].([]string)

	if !ok {
		return errors.New("Unable to parse allow field to array")
	}
	for _, s := range allows {
		flows = append(flows, &FlowDescription{
			Flow: fmt.Sprint(s),
			Type: "allow",
		})
	}
	p.Flows = flows
	return nil
}
