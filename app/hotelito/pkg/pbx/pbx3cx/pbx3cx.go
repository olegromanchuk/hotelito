package pbx3cx

import (
	"encoding/json"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	oauthConf                  *oauth2.Config
	log                        *logrus.Logger
	loginLoopPreventionCounter = 1
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
}

func New(logMain *logrus.Logger) *PBX3CX {
	log = logMain //global variable
	log.Debugf("Creating new PBX3CX client")
	pbx3cx := &PBX3CX{}
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
	log.Debugf("Parsing request body from 3CX")
	log.Debugf("Got %v", requestBody)
	if requestBody.CallType == "Outbound" {
		room, err = processOutboundCall(requestBody)
		if err != nil {
			errMsg := fmt.Errorf("Error processing outbound call: %s", err)
			return room, errMsg
		}
	}
	return room, nil
}

func processOutboundCall(requestBody RequestBody) (room pbx.Room, err error) {
	log.Debugf("Processing outbound call to %s", requestBody.Number)
	var roomCondition string
	houskeeperID := "1" //TODO: get houskeeperID from cloudbeds

	switch requestBody.Number {

	case "2222222501":
		roomCondition = "clean"
		break
	case "2222222502":
		roomCondition = "dirty"
		break
	}

	room = pbx.Room{
		PhoneNumber:   requestBody.Agent,
		RoomCondition: roomCondition,
		HouskeeperID:  houskeeperID,
	}
	return room, nil
}
