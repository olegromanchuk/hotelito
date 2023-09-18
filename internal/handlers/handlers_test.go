package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
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

func (m *MockPBXProvider) ProcessLookupByNumber(number string) []byte {
	args := m.Called(number)
	return args.Get(0).([]byte)
}

type MockHospitalityProvider struct {
	mock.Mock
}

func (m *MockHospitalityProvider) GetRooms() ([]hotel.Room, error) {
	args := m.Called()
	return args.Get(0).([]hotel.Room), args.Error(1)
}

func (m *MockHospitalityProvider) GetRoom(roomNumber string, mapFileName string) (hotel.Room, error) {
	args := m.Called(roomNumber, mapFileName)
	return args.Get(0).(hotel.Room), args.Error(1)
}

func (m *MockHospitalityProvider) HandleInitialLogin() (url string, err error) {
	args := m.Called()
	return args.Get(0).(string), args.Error(1)
}

func (m *MockHospitalityProvider) HandleOAuthCallback(state, code string) (err error) {
	args := m.Called(state, code)
	return args.Error(1)
}

//func (m *MockHospitalityProvider) UpdateRoom(roomNumber, housekeepingStatus, housekeeperName string) (msg string, err error) {
//	if roomNumber == "error" {
//		return "", errors.New("some error")
//	}
//	return fmt.Sprintf("Finish UpdateRoom successfully updated room %s to %s", roomNumber, housekeepingStatus), nil
//}

func (m *MockHospitalityProvider) UpdateRoom(roomNumber, housekeepingStatus, housekeeperName string) (msg string, err error) {
	args := m.Called(roomNumber, housekeepingStatus, housekeeperName)
	return args.Get(0).(string), args.Error(1)
}

func TestHandler_HandleSetHousekeepingStatus(t *testing.T) {
	// Setup Logger
	log := logrus.New()
	log.Out = io.Discard // discard log output for test

	// Test cases
	tests := []struct {
		name           string
		roomPhone      string
		roomCondition  string
		housekeeperID  string
		responseString string
		responseError  error
		expectedError  bool
		expCode        int
		expMessage     string
	}{
		{
			name:           "Successful UpdateRoom",
			roomPhone:      "1001",
			roomCondition:  "clean",
			housekeeperID:  "1",
			responseString: "Finish UpdateRoom successfully updated room 1001 to clean",
			responseError:  nil,
			expectedError:  false,
			expCode:        http.StatusOK,
			expMessage:     "Finish UpdateRoom successfully updated room 1001 to clean",
		}, {
			name:           "Failed UpdateRoom",
			roomPhone:      "error",
			roomCondition:  "clean",
			housekeeperID:  "1",
			responseString: "test error",
			responseError:  errors.New("test error"),
			expectedError:  true,
			expCode:        http.StatusInternalServerError,
			expMessage:     "test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Handler with mock providers
			h := &Handler{
				Log:   log,
				PBX:   &MockPBXProvider{},
				Hotel: &MockHospitalityProvider{},
			}

			r, _ := http.NewRequest("GET", "/some/url", nil)
			w := httptest.NewRecorder()

			vars := map[string]string{
				"roomPhoneNumber":    tt.roomPhone,
				"housekeepingStatus": tt.roomCondition,
				"housekeeperID":      tt.housekeeperID,
			}

			muxContext := mux.SetURLVars(r, vars)
			h.Hotel.(*MockHospitalityProvider).On("UpdateRoom", vars["roomPhoneNumber"], vars["housekeepingStatus"], vars["housekeeperID"]).Return(tt.responseString, tt.responseError)

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
		})
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
		name                                  string
		fields                                fields
		responseErrorProcessLookupByNumberErr error
		responseErrorProcessLookupByNumber    string
		args                                  args
		expectedBody                          string
		expectedHttpCode                      int
	}{
		{
			name: "Succesful lookup",
			fields: fields{
				Log:   logrus.New(),
				PBX:   &MockPBXProvider{},
				Hotel: &MockHospitalityProvider{},
			},
			responseErrorProcessLookupByNumberErr: nil,
			responseErrorProcessLookupByNumber:    "12345",
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("GET", "/test/url?Number=14523", nil),
			},
			expectedBody:     `12345`,
			expectedHttpCode: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Log:   tt.fields.Log,
				PBX:   tt.fields.PBX,
				Hotel: tt.fields.Hotel,
			}

			h.PBX.(*MockPBXProvider).On("ProcessLookupByNumber", mock.Anything).Return([]byte(tt.responseErrorProcessLookupByNumber), tt.responseErrorProcessLookupByNumberErr)

			h.Handle3cxLookup(tt.args.w, tt.args.r)
			assert.Equal(t, tt.expectedHttpCode, tt.args.w.(*httptest.ResponseRecorder).Code)
			assert.Equal(t, tt.expectedBody, tt.args.w.(*httptest.ResponseRecorder).Body.String())
		})
	}
}

func TestHandler_Handle3cxCallInfo(t *testing.T) {
	log := logrus.New()
	log.Out = io.Discard // discard log output for test

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}

	tests := []struct {
		name           string
		pbxMock        pbx.PBXProvider
		hotelMock      *MockHospitalityProvider
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Successful lookup",
			pbxMock: func() pbx.PBXProvider {
				m := new(MockPBXProvider)
				m.On("ProcessPBXRequest", mock.Anything).Return(pbx.Room{PhoneNumber: "123"}, nil)
				return m
			}(),
			hotelMock: func() *MockHospitalityProvider {
				m := new(MockHospitalityProvider)
				m.On("UpdateRoom", mock.Anything, mock.Anything, mock.Anything).Return("Updated", nil)
				return m
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/3cxCallInfo", strings.NewReader(`{"Number": "14523", "CallType": "Inbound"}`)),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Updated",
		},
		{
			name: "Ignore incoming call",
			pbxMock: func() pbx.PBXProvider {
				m := new(MockPBXProvider)
				m.On("ProcessPBXRequest", mock.Anything).Return(pbx.Room{}, errors.New("incoming-call-ignoring"))
				return m
			}(),
			hotelMock: new(MockHospitalityProvider), // No methods expected to be called
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/3cxCallInfo", strings.NewReader(`{"Number": "14523", "CallType": "Inbound"}`)),
			},
			expectedStatus: http.StatusOK, // Status code not set
			expectedBody:   "",
		},

		{
			name: "failed update room",
			pbxMock: func() pbx.PBXProvider {
				m := new(MockPBXProvider)
				m.On("ProcessPBXRequest", mock.Anything).Return(pbx.Room{PhoneNumber: "123"}, nil)
				return m
			}(),
			hotelMock: func() *MockHospitalityProvider {
				m := new(MockHospitalityProvider)
				m.On("UpdateRoom", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("test error"))
				return m
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/3cxCallInfo", strings.NewReader(`{"Number": "14523", "CallType": "Inbound"}`)),
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "test error",
		},
		{
			name: "empty room number",
			pbxMock: func() pbx.PBXProvider {
				m := new(MockPBXProvider)
				m.On("ProcessPBXRequest", mock.Anything).Return(pbx.Room{PhoneNumber: ""}, nil)
				return m
			}(),
			hotelMock: func() *MockHospitalityProvider {
				m := new(MockHospitalityProvider)
				m.On("UpdateRoom", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("test error"))
				return m
			}(),
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest(http.MethodPost, "/3cxCallInfo", strings.NewReader(`{"Number": "14523", "CallType": "Inbound"}`)),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Log:   log,
				PBX:   tt.pbxMock,
				Hotel: tt.hotelMock,
			}

			h.Handle3cxCallInfo(tt.args.w, tt.args.r)

			res := tt.args.w.(*httptest.ResponseRecorder)
			assert.Equal(t, tt.expectedStatus, res.Code)
			assert.Equal(t, tt.expectedBody, res.Body.String())
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

func TestHandleManualLogin(t *testing.T) {
	// Define the test cases as a slice of structs
	testCases := []struct {
		name               string
		username           string
		password           string
		expected           string
		expectedCodeStatus int
	}{
		{
			name:               "valid credentials",
			username:           "testuser",
			password:           "testpass",
			expected:           "",
			expectedCodeStatus: http.StatusTemporaryRedirect,
		},
		{
			name:               "invalid credentials",
			username:           "invaliduser",
			password:           "invalidpass",
			expected:           "invalid credentials",
			expectedCodeStatus: http.StatusInternalServerError,
		},
	}

	// Loop over the test cases and run the tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new request with a POST method and a JSON body
			logger := logrus.New()
			logger.Out = io.Discard // discard log output for test
			reqBody := fmt.Sprintf(`{"username": "%s", "password": "%s"}`, tc.username, tc.password)
			req, err := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create a new ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Create a new mock HospitalityProvider
			mockProvider := &MockHospitalityProvider{}

			// Create a new Handler instance with the mock provider
			h := &Handler{
				Hotel: mockProvider,
				Log:   logger,
			}

			// Set up the mock HandleInitialLogin function to return a user ID
			userID := "123"
			if tc.username == "testuser" && tc.password == "testpass" {
				mockProvider.On("HandleInitialLogin").Return(userID, nil)
			} else {
				mockProvider.On("HandleInitialLogin").Return("", errors.New("invalid credentials"))
			}

			// Call HandleManualLogin with the request
			h.HandleManualLogin(rr, req)

			// Check the response status code
			if status := rr.Code; status != tc.expectedCodeStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, http.StatusOK)
			}

			// Check the response body
			if rr.Body.String() != tc.expected {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.expected)
			}

			// Check that the mock HandleInitialLogin function was called
			mockProvider.AssertCalled(t, "HandleInitialLogin")
		})
	}
}

func TestHandleCallback(t *testing.T) {
	// Define the test cases as a slice of structs
	testCases := []struct {
		name               string
		code               string
		expected           string
		expectedCodeStatus int
	}{
		{
			name:               "valid code",
			code:               "validcode",
			expected:           "Great Success! Ready for future requests. You can close this window now.",
			expectedCodeStatus: http.StatusOK,
		},
		{
			name:               "invalid code",
			code:               "invalidcode",
			expected:           "invalid code",
			expectedCodeStatus: http.StatusInternalServerError,
		},
	}

	// Loop over the test cases and run the tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new request with a GET method and a query parameter
			req, err := http.NewRequest("GET", "/callback", nil)
			if err != nil {
				t.Fatal(err)
			}
			q := req.URL.Query()
			q.Add("code", tc.code)
			req.URL.RawQuery = q.Encode()

			// Create a new ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Create a new mock HospitalityProvider
			mockProvider := &MockHospitalityProvider{}

			// Create a new Handler instance with the mock provider
			h := &Handler{
				Hotel: mockProvider,
				Log:   logrus.New(),
			}

			// Set up the mock HandleCallback function to return a user ID or an error depending on the code
			userID := "123"
			if tc.code == "validcode" {
				mockProvider.On("HandleOAuthCallback", "", tc.code).Return(userID, nil)
			} else {
				mockProvider.On("HandleOAuthCallback", "", tc.code).Return("", errors.New("invalid code"))
			}

			// Call HandleCallback with the request
			h.HandleCallback(rr, req)

			// Check the response status code
			if status := rr.Code; status != tc.expectedCodeStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedCodeStatus)
			}

			// Check the response body
			if rr.Body.String() != tc.expected {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.expected)
			}

			// Check that the mock HandleCallback function was called with the correct code
			mockProvider.AssertCalled(t, "HandleOAuthCallback", "", tc.code)
		})
	}
}

func TestHandleGetRooms(t *testing.T) {
	mockLogger := logrus.New()
	mockLogger.SetLevel(logrus.PanicLevel) // Set log level to panic to suppress logs during testing

	testCases := []struct {
		name         string
		mockRooms    []hotel.Room
		mockError    error
		expectedCode int
		expectedBody string
	}{
		{
			name: "Successful Room Retrieval",
			mockRooms: []hotel.Room{
				{PhoneNumber: "101"},
				{PhoneNumber: "102"},
			},
			mockError:    nil,
			expectedCode: http.StatusOK,
			expectedBody: "amount of rooms: 2",
		},
		{
			name:         "Provider Error",
			mockRooms:    nil,
			mockError:    errors.New("some error"),
			expectedCode: http.StatusInternalServerError,
			expectedBody: "some error",
		},
		{
			name:         "Empty Room List",
			mockRooms:    []hotel.Room{},
			mockError:    nil,
			expectedCode: http.StatusOK,
			expectedBody: "amount of rooms: 0",
		},
		// Add other failed or edge cases here...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockProvider := new(MockHospitalityProvider)
			mockProvider.On("GetRooms").Return(tc.mockRooms, tc.mockError)

			req, err := http.NewRequest("GET", "/rooms", nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			handler := NewHandler(mockLogger, nil, mockProvider)
			handler.HandleGetRooms(recorder, req)

			res := recorder.Result()
			defer res.Body.Close()

			assert.Equal(t, tc.expectedCode, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestHandleMain(t *testing.T) {
	mockLogger := logrus.New()
	mockLogger.SetLevel(logrus.PanicLevel) // Set log level to panic to suppress logs during testing

	testCases := []struct {
		name         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Basic Test",
			expectedCode: http.StatusOK,
			expectedBody: `<a href="/login">Login with OAuth2 Provider</a>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/main", nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			handler := NewHandler(mockLogger, nil, nil)
			handler.HandleMain(recorder, req)

			res := recorder.Result()
			defer res.Body.Close()

			assert.Equal(t, tc.expectedCode, res.StatusCode)

			bodyBytes, _ := io.ReadAll(res.Body)
			bodyString := string(bodyBytes)
			assert.Equal(t, tc.expectedBody, bodyString)
		})
	}
}
