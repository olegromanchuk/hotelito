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

type Contact struct {
	ID          int    `json:"id"`
	FirstName   string `json:"firstname"`
	Company     string `json:"company"`
	MobilePhone string `json:"mobilephone"`
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

	pbx3cx.log.Debugf("Parsing request body from 3CX")

	requestBody, err := pbx3cx.decodeRequestBody(jsonDecoder)
	if err != nil {
		return pbx.Room{}, err
	}

	pbx3cx.log.Debugf("Got %v", requestBody)

	if requestBody.CallType == "Inbound" { //junk. Due to 3CX specific we receive incoming calls also, but we do not need them.
		return room, fmt.Errorf("incoming-call-ignoring")
	}

	if requestBody.CallType == "Outbound" {
		room, err = pbx3cx.processOutboundCall(requestBody)
		if err != nil && err.Error() == "outgoing-regular-call-ignoring" { //regular outbound call is not related to room status
			return room, err
		}
	}
	return room, nil
}

// decodeRequestBody decodes request body from 3CX
func (pbx3cx *PBX3CX) decodeRequestBody(jsonDecoder *json.Decoder) (RequestBody, error) {
	var requestBody RequestBody
	err := jsonDecoder.Decode(&requestBody)
	if err != nil || requestBody.CallType == "" {
		return RequestBody{}, fmt.Errorf("error decoding request body / no callType provided: %s", err)
	}
	return requestBody, nil
}

// processOutboundCall search for room by extension (agent=extension) and returns room information
func (pbx3cx *PBX3CX) processOutboundCall(requestBody RequestBody) (room pbx.Room, err error) {
	pbx3cx.log.Debugf("Processing outbound call from %s to %s", requestBody.Agent, requestBody.Number)

	PhoneNumber4HouseKeeper := requestBody.Number //2222222221
	RoomExtension := requestBody.Agent            //1001

	//creating map of housekeeper's room status numbers (the number he is calling to to update room status), so we can easily find housekeeper and room status by phone number
	//each housekeeper has his own number(s), where he calls. There are two numbers: for clean and dirty rooms.
	//This number is mapped to the housekeeper name and room status
	mapHousekeeperNumbers := make(map[string]configuration.Housekeeper)

	for _, housekeeper := range pbx3cx.configMap.HousekeeperMap {
		mapHousekeeperNumbers[housekeeper.RoomStatusPhoneNumber] = housekeeper
	}

	numberInformation := mapHousekeeperNumbers[PhoneNumber4HouseKeeper] //if not found we got empty map
	//if we got empty map - silently discard this call. It is a regular outbound call is not related to room status
	if numberInformation.RoomStatusPhoneNumber == "" {
		pbx3cx.log.Debugf("housekeeper number not found: %s", PhoneNumber4HouseKeeper)
		return room, fmt.Errorf("outgoing-regular-call-ignoring")
	}

	pbx3cx.log.Debugf("found housekeeper number: %s. Housekeeper: %s. Room condition: %s", numberInformation.RoomStatusPhoneNumber, numberInformation.HousekeeperName, numberInformation.NumberType)
	roomCondition := numberInformation.NumberType

	room = pbx.Room{
		PhoneNumber:     RoomExtension,
		RoomCondition:   roomCondition,
		HousekeeperName: mapHousekeeperNumbers[PhoneNumber4HouseKeeper].HousekeeperName,
	}
	return room, nil
}

// ProcessLookupByNumber returns the []byte that contain contact information with the given number
// This function does not contain any meaningful logic. It just converts input number to the json Contact
// We need it to satisfy 3cx API request for number lookup. 3cx sends API request and expects json with contact information
// if no lookup information is provided back, the next request will not be sent. So, we just take incoming number and generate a dummy contact to satisfy 3cx.
func (pbx3cx *PBX3CX) ProcessLookupByNumber(number string) (bodyAsBytes []byte, err error) {
	return ProcessLookupByNumber(number)
}

func ProcessLookupByNumber(number string) (bodyAsBytes []byte, err error) {
	contact := Contact{
		ID:          12345,
		FirstName:   "dummyFirstName",
		Company:     "dummyCompany",
		MobilePhone: number,
	}

	returnStruct := struct {
		Contact Contact `json:"contact"`
	}{Contact: contact}
	bodyAsBytes, _ = json.Marshal(returnStruct)
	return bodyAsBytes, nil
}
