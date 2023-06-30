package pbx3cx

import (
	"encoding/json"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
	"github.com/sirupsen/logrus"
)

type RequestBody struct {
	Number         string `json:"Number"`
	CallType       string `json:"CallType"`
	CallDirection  string `json:"CallDirection"`
	Name           string `json:"Name"`
	Agent          string `json:"Agent"`
	AgentFirstName string `json:"AgentFirstName"`
	DateTime       string `json:"DateTime"`
}

type PBX3CX struct {
	log *logrus.Logger
}

func New(log *logrus.Logger) *PBX3CX {
	log.Debugf("Creating new PBX3CX client")
	pbx3cx := &PBX3CX{
		log: log,
	}
	return pbx3cx
}

func (pbx3cx *PBX3CX) ProcessPBXRequest(jsonDecoder *json.Decoder) (room pbx.Room, err error) {
	var requestBody RequestBody
	err = jsonDecoder.Decode(&requestBody)
	if err != nil {
		errMsg := fmt.Errorf("Error decoding request body: %s", err)
		return room, errMsg
	}
	if requestBody.CallType == "Inbound" { //junk. Due to 3CX specific we receive incoming calls also, but we do not need them.
		return room, fmt.Errorf("incoming-call-ignoring")
	}
	pbx3cx.log.Debugf("Parsing request body from 3CX")
	pbx3cx.log.Debugf("Got %v", requestBody)
	if requestBody.CallType == "Outbound" {
		room, err = pbx3cx.processOutboundCall(requestBody)
		if err != nil {
			errMsg := fmt.Errorf("Error processing outbound call: %s", err)
			return room, errMsg
		}
	}
	return room, nil
}

func (pbx3cx *PBX3CX) processOutboundCall(requestBody RequestBody) (room pbx.Room, err error) {
	pbx3cx.log.Debugf("Processing outbound call to %s", requestBody.Number)
	var roomCondition string
	houskeeperID := "1" //TODO: get houskeeperID from cloudbeds

	switch requestBody.Number {
	case "2222222501":
		roomCondition = "clean"
	case "2222222502":
		roomCondition = "dirty"
	}

	room = pbx.Room{
		PhoneNumber:   requestBody.Agent,
		RoomCondition: roomCondition,
		HouskeeperID:  houskeeperID,
	}
	return room, nil
}
