package pbx3cx

import (
	"encoding/json"
	"fmt"
	"github.com/olegromanchuk/hotelito/internal/configuration"
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
	log       *logrus.Logger
	configMap *configuration.ConfigMap
}

// New creates new PBX3CX client
func New(log *logrus.Logger, configMapInfo *configuration.ConfigMap) *PBX3CX {
	log.Debugf("Creating new PBX3CX client")

	pbx3cx := &PBX3CX{
		log:       log,
		configMap: configMapInfo,
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
	pbx3cx.log.Debugf("Processing outbound call from %s to %s", requestBody.Agent, requestBody.Number)

	PhoneNumber4HouseKeeper := requestBody.Number //2222222221
	RoomExtension := requestBody.Agent            //1001

	//creating map of housekeeper numbers, so we can easily find housekeeper and room status by phone number
	mapHousekeeperNumbers := make(map[string]configuration.Housekeeper)
	for _, housekeeper := range pbx3cx.configMap.HousekeeperMap {
		mapHousekeeperNumbers[housekeeper.PhoneNumber] = housekeeper
	}

	numberInformation := mapHousekeeperNumbers[PhoneNumber4HouseKeeper]
	pbx3cx.log.Debugf("found housekeeper number: %s. Housekeeper: %s. Room condition: %s", numberInformation.PhoneNumber, numberInformation.HousekeeperName, numberInformation.NumberType)
	roomCondition := numberInformation.NumberType

	room = pbx.Room{
		PhoneNumber:     RoomExtension,
		RoomCondition:   roomCondition,
		HousekeeperName: mapHousekeeperNumbers[PhoneNumber4HouseKeeper].HousekeeperName,
	}
	return room, nil
}

func (pbx3cx *PBX3CX) ProcessLookupByNumber(number string) (bodyAsBytes []byte, err error) {
	type Contact struct {
		ID          int    `json:"id"`
		FirstName   string `json:"firstname"`
		Company     string `json:"company"`
		MobilePhone string `json:"mobilephone"`
	}
	contact := Contact{
		ID:          12345,
		FirstName:   "testFirstName",
		Company:     "testCompany",
		MobilePhone: number,
	}

	returnStruct := struct {
		Contact Contact `json:"contact"`
	}{Contact: contact}

	bodyAsBytes, err = json.Marshal(returnStruct)
	if err != nil {
		return bodyAsBytes, err
	}
	return bodyAsBytes, nil
}
