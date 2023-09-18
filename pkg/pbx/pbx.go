package pbx

import "encoding/json"

type PBXProvider interface {
	ProcessPBXRequest(jsonDecoder *json.Decoder) (Room, error)
	ProcessLookupByNumber(number string) (bodyAsBytes []byte)
}

type Room struct {
	PhoneNumber     string `json:"RoomStatusPhoneNumber"`
	RoomCondition   string `json:"RoomCondition"`
	HousekeeperName string `json:"HousekeeperName"`
}
