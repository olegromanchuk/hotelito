package pbx

import "encoding/json"

type PBXProvider interface {
	ProcessPBXRequest(jsonDecoder *json.Decoder) (Room, error)
	ProcessLookupByNumber(number string) (bodyAsBytes []byte, err error)
}

type Room struct {
	PhoneNumber   string `json:"PhoneNumber"`
	RoomCondition string `json:"RoomCondition"`
	HouskeeperID  string `json:"HouskeeperID"`
}
