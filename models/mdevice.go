package models

import "github.com/zebresel-com/mongodm"

type Device struct {
	mongodm.DocumentBase `json:",inline" bson:",inline"`

	SwitchId string       `json:"@switch" bson:"@switch"`
	Ports interface{}      `json:"ports" bson:"ports" model:"OccupiedPort" relation:"1n" autosave:"true"`
}
type OccupiedPort struct {
	mongodm.DocumentBase `json:",inline" bson:",inline"`
	Port string `json:"@port" bson:"@port"`
	MAC string `json:"@mac" bson:"@mac"`
	APID string `json:"apid" bson:"apid"`
	ProfileId string `json:"profileid" bson:"profileid"`
}
