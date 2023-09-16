package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type MockPBXProvider struct {
	mock.Mock
}

func (m *MockPBXProvider) ProcessPBXRequest(jsonDecoder *json.Decoder) (pbx.Room, error) {
	args := m.Called(jsonDecoder)
	return args.Get(0).(pbx.Room), args.Error(1)
}

func (m *MockPBXProvider) ProcessLookupByNumber(number string) ([]byte, error) {
	args := m.Called(number)
	return args.Get(0).([]byte), args.Error(1)
}

type MockHospitalityProvider struct {
	mock.Mock
}

func (m *MockHospitalityProvider) GetRooms() ([]hotel.Room, error) {
	return []hotel.Room{}, nil
}

func (m *MockHospitalityProvider) GetRoom(roomNumber string, mapFileName string) (hotel.Room, error) {
	return hotel.Room{}, nil
}

func (m *MockHospitalityProvider) HandleInitialLogin() (url string, err error) {
	return url, nil
}

func (m *MockHospitalityProvider) HandleOAuthCallback(state, code string) (err error) {
	return nil
}

func (m *MockHospitalityProvider) UpdateRoom(roomNumber, housekeepingStatus, housekeeperName string) (msg string, err error) {
	if roomNumber == "error" {
		return "", errors.New("some error")
	}
	return fmt.Sprintf("Finish UpdateRoom successfully updated room %s to %s", roomNumber, housekeepingStatus), nil
}

func TestHandler_HandleSetHousekeepingStatus(t *testing.T) {
	// Setup Logger
	log := logrus.New()
	log.Out = io.Discard // discard log output for test

	// Setup Handler with mock providers
	h := &Handler{
		Log:   log,
		PBX:   &MockPBXProvider{},
		Hotel: &MockHospitalityProvider{},
	}

	// Test cases
	tests := []struct {
		roomPhone     string
		roomCondition string
		housekeeperID string
		expCode       int
		expMessage    string
	}{
		{"1001", "clean", "1", http.StatusOK, "Finish UpdateRoom successfully updated room 1001 to clean"},
		{"error", "clean", "1", http.StatusInternalServerError, "some error"},
	}

	for _, tt := range tests {
		r, _ := http.NewRequest("GET", "/some/url", nil)
		w := httptest.NewRecorder()

		vars := map[string]string{
			"roomPhoneNumber":    tt.roomPhone,
			"housekeepingStatus": tt.roomCondition,
			"housekeeperID":      tt.housekeeperID,
		}

		muxContext := mux.SetURLVars(r, vars)

		// Act
		h.HandleSetHousekeepingStatus(w, muxContext)

		// Assert
		if w.Code != tt.expCode {
			t.Errorf("Expected HTTP status code %d, got %d", tt.expCode, w.Code)
		}

		body := strings.TrimSpace(w.Body.String())
		if body != tt.expMessage {
			t.Errorf("Expected message '%s', got '%s'", tt.expMessage, body)
		}
	}
}

func TestHandler_Handle3cxLookup(t *testing.T) {

	// Setup Logger
	log := logrus.New()
	log.Out = io.Discard // discard log output for test

	type fields struct {
		Log   *logrus.Logger
		PBX   pbx.PBXProvider
		Hotel hotel.HospitalityProvider
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		expectedBody string
	}{
		{
			"Succesful lookup",
			fields{
				Log:   logrus.New(),
				PBX:   &pbx3cx.PBX3CX{},
				Hotel: &MockHospitalityProvider{},
			},
			args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("GET", "/test/url?Number=14523", nil),
			},
			`{"contact":{"id":12345,"firstname":"dummyFirstName","company":"dummyCompany","mobilephone":"14523"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Log:   tt.fields.Log,
				PBX:   tt.fields.PBX,
				Hotel: tt.fields.Hotel,
			}

			h.Handle3cxLookup(tt.args.w, tt.args.r)
			assert.Equal(t, http.StatusOK, tt.args.w.(*httptest.ResponseRecorder).Code)
			assert.Equal(t, tt.expectedBody, tt.args.w.(*httptest.ResponseRecorder).Body.String())
		})
	}
}

func TestHandler_Handle3cxCallInfo(t *testing.T) {

	// Setup Logger
	log := logrus.New()
	log.Out = io.Discard // discard log output for test

	pbx3cxInstance := pbx3cx.New(log, nil)

	type fields struct {
		Log   *logrus.Logger
		PBX   pbx.PBXProvider
		Hotel *MockHospitalityProvider
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		requestBody  string
		expectedBody string
	}{
		{
			"Succesful lookup",
			fields{
				Log:   logrus.New(),
				PBX:   pbx3cxInstance,
				Hotel: &MockHospitalityProvider{},
			},
			args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("GET", "/test/url?Number=14523&CallType=Inbound", nil),
			},
			`{
    "Number": "123456789",
    "CallType": "Inbound",
    "CallDirection": "Inbound",
    "Name": "John Doe",
    "Agent": "Agent007",
    "AgentFirstName": "James",
    "DateTime": "2023-09-08T12:34:56Z"
  }`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Log:   tt.fields.Log,
				PBX:   tt.fields.PBX,
				Hotel: tt.fields.Hotel,
			}

			var requestBody pbx3cx.RequestBody
			err := json.Unmarshal([]byte(tt.requestBody), &requestBody)
			if err != nil {
				t.Fatalf("Error unmarshaling JSON: %v", err)
			}

			h.Handle3cxCallInfo(tt.args.w, tt.args.r)
			fmt.Print(tt.args.w.(*httptest.ResponseRecorder).Body.String())
			assert.Equal(t, http.StatusOK, tt.args.w.(*httptest.ResponseRecorder).Code)
			assert.Equal(t, tt.expectedBody, tt.args.w.(*httptest.ResponseRecorder).Body.String())
		})
	}
}

func TestNewHandler(t *testing.T) {
	mockLog := logrus.New()
	mockPBX := new(MockPBXProvider)
	mockHotel := new(MockHospitalityProvider)

	handler := NewHandler(mockLog, mockPBX, mockHotel)

	assert.NotNil(t, handler)
	assert.IsType(t, &Handler{}, handler)

	// Validate that the dependencies are correctly injected
	assert.Equal(t, mockLog, handler.Log)
	assert.Equal(t, mockPBX, handler.PBX)
	assert.Equal(t, mockHotel, handler.Hotel)
}
