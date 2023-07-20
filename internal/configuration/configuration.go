package configuration

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

// Extension represents the extension mapping
type Extension struct {
	RoomExtension       string `json:"room_extension"`
	HospitalityRoomID   string `json:"hospitality_room_id"`
	HospitalityRoomName string `json:"hospitality_room_name"`
}

// Housekeeper represents the housekeeper mapping
type Housekeeper struct {
	PhoneNumber     string `json:"phone_number"`
	HousekeeperName string `json:"housekeeper_name"`
	NumberType      string `json:"number_type"`
}

// ConfigMap contains arrays of Extension and Housekeeper
type ConfigMap struct {
	ExtensionMap   []Extension   `json:"extension_map"`
	HousekeeperMap []Housekeeper `json:"housekeeper_map"`
}

func New(log *logrus.Logger, mapFileName string) (*ConfigMap, error) {
	configMapInfo := &ConfigMap{}
	//get configuration from mapFileName
	//process configuration file
	file, err := os.Open(mapFileName)
	if err != nil {
		errMsg := fmt.Errorf("Error opening config file: ", err.Error())
		log.Errorf(errMsg.Error())
		return configMapInfo, errMsg
	}
	defer file.Close()
	byteValue, _ := io.ReadAll(file)
	err = json.Unmarshal(byteValue, configMapInfo)
	if err != nil {
		errMsg := fmt.Errorf("error unmarshalling config file %s: %s", mapFileName, err.Error())
		log.Errorf(errMsg.Error())
		return configMapInfo, errMsg
	}
	return configMapInfo, nil
}
