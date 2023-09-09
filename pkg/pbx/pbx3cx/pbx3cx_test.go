package pbx3cx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestPBX3CX_ProcessPBXRequest(t *testing.T) {

	type fields struct {
		log       *logrus.Logger
		configMap *configuration.ConfigMap
	}
	type args struct {
		jsonDecoder *json.Decoder
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantRoom pbx.Room
		wantErr  bool
	}{
		{
			name: "Test Inbound Call",
			fields: fields{
				log:       logrus.New(),
				configMap: &configuration.ConfigMap{},
			},
			args: args{
				jsonDecoder: json.NewDecoder(bytes.NewBufferString(`{"CallType": "Inbound"}`)),
			},
			wantRoom: pbx.Room{},
			wantErr:  true,
		},
		{
			name: "Test Outbound Call",
			fields: fields{
				log: logrus.New(),
				configMap: &configuration.ConfigMap{
					ExtensionMap: []configuration.Extension{
						{
							RoomExtension:       "101",
							HospitalityRoomID:   "1",
							HospitalityRoomName: "101",
						},
					},
					HousekeeperMap: []configuration.Housekeeper{
						{
							RoomStatusPhoneNumber: "2222222221",
							HousekeeperName:       "John Doe",
							NumberType:            "dirty",
						},
					},
				},
			},
			args: args{
				jsonDecoder: json.NewDecoder(bytes.NewBufferString(`{"CallType": "Outbound", "Number": "2222222221", "Agent": "101"}`)),
			},
			wantRoom: pbx.Room{
				PhoneNumber:     "101",
				RoomCondition:   "dirty",
				HousekeeperName: "John Doe",
			},
			wantErr: false,
		},
		{
			name: "Test Outbound Call - regular call",
			fields: fields{
				log: logrus.New(),
				configMap: &configuration.ConfigMap{
					ExtensionMap: []configuration.Extension{
						{
							RoomExtension:       "101",
							HospitalityRoomID:   "1",
							HospitalityRoomName: "101",
						},
					},
					HousekeeperMap: []configuration.Housekeeper{
						{
							RoomStatusPhoneNumber: "2222222221",
							HousekeeperName:       "John Doe",
							NumberType:            "dirty",
						},
					},
				},
			},
			args: args{
				jsonDecoder: json.NewDecoder(bytes.NewBufferString(`{"CallType": "Outbound", "Number": "987654321", "Agent": "404"}`)),
			},
			wantRoom: pbx.Room{
				PhoneNumber:     "",
				RoomCondition:   "",
				HousekeeperName: "",
			},
			wantErr: true,
		},
		{
			name: "Test bad decoder",
			fields: fields{
				log:       logrus.New(),
				configMap: &configuration.ConfigMap{},
			},
			args: args{
				jsonDecoder: json.NewDecoder(bytes.NewBufferString(`{"asdasdasdas": "asdasdasdasdas"}`)),
			},
			wantRoom: pbx.Room{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pbx3cx := &PBX3CX{
				log:       tt.fields.log,
				configMap: tt.fields.configMap,
			}
			gotRoom, err := pbx3cx.ProcessPBXRequest(tt.args.jsonDecoder)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessPBXRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRoom, tt.wantRoom) {
				t.Errorf("ProcessPBXRequest() gotRoom = %v, want %v", gotRoom, tt.wantRoom)
			}
		})
	}
}

func TestNew(t *testing.T) {
	logger := logrus.New()
	configMapInfo := &configuration.ConfigMap{}

	pbx3cx := New(logger, configMapInfo)

	if pbx3cx.log != logger {
		t.Errorf("Unexpected logger object")
	}

	if pbx3cx.configMap != configMapInfo {
		t.Errorf("Unexpected ConfigMap object")
	}
}

func TestPBX3CX_processOutboundCall(t *testing.T) {
	type fields struct {
		log       *logrus.Logger
		configMap *configuration.ConfigMap
	}
	type args struct {
		requestBody RequestBody
	}

	tests := []struct {
		name       string
		fields     fields
		args       args
		wantRoom   pbx.Room
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "Test Outbound Call",
			fields: fields{
				log: logrus.New(),
				configMap: &configuration.ConfigMap{
					ExtensionMap: []configuration.Extension{
						{
							RoomExtension:       "101",
							HospitalityRoomID:   "1",
							HospitalityRoomName: "101",
						},
					},
					HousekeeperMap: []configuration.Housekeeper{
						{
							RoomStatusPhoneNumber: "2222222221",
							HousekeeperName:       "John Doe",
							NumberType:            "dirty",
						},
					},
				},
			},
			args: args{
				requestBody: RequestBody{
					Number:   "2222222221",
					Agent:    "101",
					CallType: "Outbound",
				},
			},
			wantRoom: pbx.Room{
				PhoneNumber:     "101",
				RoomCondition:   "dirty",
				HousekeeperName: "John Doe",
			},
			wantErr:    false,
			wantErrMsg: "",
		},
		{
			name: "Test Outbound Call - regular call",
			fields: fields{
				log: logrus.New(),
				configMap: &configuration.ConfigMap{
					ExtensionMap: []configuration.Extension{
						{
							RoomExtension:       "101",
							HospitalityRoomID:   "1",
							HospitalityRoomName: "101",
						},
					},
					HousekeeperMap: []configuration.Housekeeper{
						{
							RoomStatusPhoneNumber: "2222222221",
							HousekeeperName:       "John Doe",
							NumberType:            "dirty",
						},
					},
				},
			},
			args: args{
				requestBody: RequestBody{
					Number:   "987654321",
					Agent:    "404",
					CallType: "Outbound",
				},
			},
			wantRoom: pbx.Room{
				PhoneNumber:     "",
				RoomCondition:   "",
				HousekeeperName: "",
			},
			wantErr:    true,
			wantErrMsg: "outgoing-regular-call-ignoring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pbx3cx := &PBX3CX{
				log:       tt.fields.log,
				configMap: tt.fields.configMap,
			}
			gotRoom, err := pbx3cx.processOutboundCall(tt.args.requestBody)
			if (err != nil) != tt.wantErr {
				t.Errorf("processOutboundCall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				if err.Error() != tt.wantErrMsg {
					t.Errorf("processOutboundCall() error = %v, wantErrMsg %v", err.Error(), tt.wantErrMsg)
					return
				}
				if !reflect.DeepEqual(gotRoom, tt.wantRoom) {
					t.Errorf("processOutboundCall() gotRoom = %v, want %v", gotRoom, tt.wantRoom)
				}
			}
		},
		)
	}

}

func TestProcessLookupByNumber(t *testing.T) {

	pbx3cxClient := &PBX3CX{
		log:       logrus.New(),
		configMap: &configuration.ConfigMap{},
	}

	// Test case 2: valid number
	t.Run("returns contact", func(t *testing.T) {
		number := "1234567890"
		expectedContact := Contact{
			ID:          12345,
			FirstName:   "dummyFirstName",
			Company:     "dummyCompany",
			MobilePhone: number,
		}

		expectedBody, _ := json.Marshal(struct {
			Contact Contact `json:"contact"`
		}{Contact: expectedContact})

		body, err := pbx3cxClient.ProcessLookupByNumber(number)
		assert.Nil(t, err)
		assert.Equal(t, expectedBody, body)
	})

}

func TestPBX3CX_decodeRequestBody(t *testing.T) {
	type fields struct {
		log       *logrus.Logger
		configMap *configuration.ConfigMap
	}
	type args struct {
		jsonDecoder *json.Decoder
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    RequestBody
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Test decodeRequestBody",
			fields: fields{
				log:       logrus.New(),
				configMap: &configuration.ConfigMap{},
			},
			args: args{
				jsonDecoder: json.NewDecoder(bytes.NewBufferString(`{"CallType": "Outbound", "Number": "2222222221", "Agent": "101"}`)),
			},
			want: RequestBody{
				Number:   "2222222221",
				Agent:    "101",
				CallType: "Outbound",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Test bad decoder",
			fields: fields{
				log:       logrus.New(),
				configMap: &configuration.ConfigMap{},
			},
			args: args{
				jsonDecoder: json.NewDecoder(bytes.NewBufferString(`{"asdasdasdas": "asdasdasdasdas"}`)),
			},
			want:    RequestBody{},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pbx3cx := &PBX3CX{
				log:       tt.fields.log,
				configMap: tt.fields.configMap,
			}
			got, err := pbx3cx.decodeRequestBody(tt.args.jsonDecoder)
			if !tt.wantErr(t, err, fmt.Sprintf("decodeRequestBody(%v)", tt.args.jsonDecoder)) {
				return
			}
			assert.Equalf(t, tt.want, got, "decodeRequestBody(%v)", tt.args.jsonDecoder)
		})
	}
}
