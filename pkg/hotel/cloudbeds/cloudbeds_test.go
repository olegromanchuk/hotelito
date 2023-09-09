package cloudbeds

import (
	"bytes"
	"fmt"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/secrets"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

type MockHTTPClient struct {
	mock.Mock
}

type MockTokenRefresher struct {
	mock.Mock
}

type MockSecretsStore struct {
	mock.Mock
}

func (m *MockSecretsStore) StoreAccessToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockSecretsStore) StoreRefreshToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockSecretsStore) RetrieveAccessToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSecretsStore) RetrieveRefreshToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSecretsStore) StoreOauthState(state string) error {
	args := m.Called(state)
	return args.Error(0)
}

func (m *MockSecretsStore) RetrieveOauthState(state string) (string, error) {
	args := m.Called(state)
	return args.String(0), args.Error(1)
}

func (m *MockSecretsStore) RetrieveVar(varName string) (varValue string, err error) {
	args := m.Called(varName)
	return args.String(0), args.Error(1)
}

func (m *MockSecretsStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockTokenRefresher) refreshToken() error {
	return fmt.Errorf("refresh token error")
}

func TestRoom_ToHotelRoom(t *testing.T) {
	type fields struct {
		RoomID            string
		RoomName          string
		RoomDescription   string
		MaxGuests         int32
		IsPrivate         bool
		RoomBlocked       bool
		RoomTypeID        int32
		RoomTypeName      string
		RoomTypeNameShort string
		PhoneNumber       string
		RoomCondition     string
		RoomOccupied      bool
	}
	tests := []struct {
		name   string
		fields fields
		want   hotel.Room
	}{
		{
			name: "Test 1",
			fields: fields{
				RoomID:            "544559-1",
				RoomName:          "DQ(2)",
				RoomDescription:   "",
				MaxGuests:         2,
				IsPrivate:         true,
				RoomBlocked:       false,
				RoomTypeID:        544559,
				RoomTypeName:      "Deluxe Queen",
				RoomTypeNameShort: "DQ",
				PhoneNumber:       "",
				RoomCondition:     "",
				RoomOccupied:      false,
			},
			want: hotel.Room{
				RoomID:            "544559-1",
				RoomName:          "DQ(2)",
				RoomDescription:   "",
				MaxGuests:         2,
				IsPrivate:         true,
				RoomBlocked:       false,
				RoomTypeID:        544559,
				RoomTypeName:      "Deluxe Queen",
				RoomTypeNameShort: "DQ",
				PhoneNumber:       "",
				RoomCondition:     "",
				RoomOccupied:      false,
			},
		},
		{
			name: "Test 2",
			fields: fields{
				RoomID:            "544560-9",
				RoomName:          "DK(10)",
				RoomDescription:   "",
				MaxGuests:         2,
				IsPrivate:         true,
				RoomBlocked:       false,
				RoomTypeID:        544560,
				RoomTypeName:      "Deluxe King",
				RoomTypeNameShort: "DK",
				PhoneNumber:       "",
				RoomCondition:     "",
				RoomOccupied:      false,
			},
			want: hotel.Room{
				RoomID:            "544560-9",
				RoomName:          "DK(10)",
				RoomDescription:   "",
				MaxGuests:         2,
				IsPrivate:         true,
				RoomBlocked:       false,
				RoomTypeID:        544560,
				RoomTypeName:      "Deluxe King",
				RoomTypeNameShort: "DK",
				PhoneNumber:       "",
				RoomCondition:     "",
				RoomOccupied:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Room{
				RoomID:            tt.fields.RoomID,
				RoomName:          tt.fields.RoomName,
				RoomDescription:   tt.fields.RoomDescription,
				MaxGuests:         tt.fields.MaxGuests,
				IsPrivate:         tt.fields.IsPrivate,
				RoomBlocked:       tt.fields.RoomBlocked,
				RoomTypeID:        tt.fields.RoomTypeID,
				RoomTypeName:      tt.fields.RoomTypeName,
				RoomTypeNameShort: tt.fields.RoomTypeNameShort,
				PhoneNumber:       tt.fields.PhoneNumber,
				RoomCondition:     tt.fields.RoomCondition,
				RoomOccupied:      tt.fields.RoomOccupied,
			}
			if got := r.ToHotelRoom(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToHotelRoom() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Get implements the HTTPClient interface
func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	args := m.Called(url)
	return args.Get(0).(*http.Response), args.Error(1)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestCloudbeds_GetRooms(t *testing.T) {

	// Define the JSON response
	responseJSON := `{
		"success": true,
		"data": [
			{
				"propertyID": "297652",
				"rooms": [
					{
						"roomID": "544559-0",
						"roomName": "DQ(1)",
						"roomDescription": "",
						"maxGuests": 2,
						"isPrivate": true,
						"roomBlocked": false,
						"roomTypeID": 544559,
						"roomTypeName": "Deluxe Queen",
						"roomTypeNameShort": "DQ"
					}
				]
			}
		],
		"count": 1,
		"total": 1
	}`

	// Create a mock http.Response
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(responseJSON)),
	}

	// Create a mock HTTPClient
	mockClient := new(MockHTTPClient)
	mockClient.On("Get", "https://hotels.cloudbeds.com/api/v1.1/getRooms").Return(resp, nil)

	// Create a Cloudbeds instance with the mock HTTPClient
	cb := &Cloudbeds{
		httpClient: mockClient,
		log:        logrus.New(),
	}

	// Call GetRooms
	rooms, err := cb.GetRooms()

	// Assert no error occurred
	assert.Nil(t, err)

	// Assert the Get function was called exactly once with the expected URL
	mockClient.AssertCalled(t, "Get", "https://hotels.cloudbeds.com/api/v1.1/getRooms")

	// Test that the response was correctly parsed
	expectedRoom := hotel.Room{
		RoomID:            "544559-0",
		RoomName:          "DQ(1)",
		RoomDescription:   "",
		MaxGuests:         2,
		IsPrivate:         true,
		RoomBlocked:       false,
		RoomTypeID:        544559,
		RoomTypeName:      "Deluxe Queen",
		RoomTypeNameShort: "DQ",
		PhoneNumber:       "",
		RoomCondition:     "",
		RoomOccupied:      false,
	}

	assert.Equal(t, expectedRoom, rooms[0], "Expected room: %v, got: %v", expectedRoom, rooms[0])
}

func TestRoom_SearchRoomIDByPhoneNumber(t *testing.T) {

	log := logrus.New()

	//get extensionsInfo from configuration.Extension
	/*
		type Extension struct {
			RoomExtension       string `json:"room_extension"`
			HospitalityRoomID   string `json:"hospitality_room_id"`
			HospitalityRoomName string `json:"hospitality_room_name"`
		}
	*/
	extensionsInfo := []configuration.Extension{
		{
			RoomExtension:       "100",
			HospitalityRoomID:   "544559-0",
			HospitalityRoomName: "DQ(1)",
		},
		{
			RoomExtension:       "101",
			HospitalityRoomID:   "544559-1",
			HospitalityRoomName: "DQ(2)",
		},
	}

	type fields struct {
		RoomID            string
		RoomName          string
		RoomDescription   string
		MaxGuests         int32
		IsPrivate         bool
		RoomBlocked       bool
		RoomTypeID        int32
		RoomTypeName      string
		RoomTypeNameShort string
		PhoneNumber       string
		RoomCondition     string
		RoomOccupied      bool
	}
	type args struct {
		phoneNumber string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				RoomID:            "544559-0",
				RoomName:          "DQ(1)",
				RoomDescription:   "",
				MaxGuests:         2,
				IsPrivate:         true,
				RoomBlocked:       false,
				RoomTypeID:        544559,
				RoomTypeName:      "Deluxe Queen",
				RoomTypeNameShort: "DQ",
				PhoneNumber:       "100",
				RoomCondition:     "",
				RoomOccupied:      false,
			},
			args: args{
				phoneNumber: "100",
			},
			want:    "544559-0",
			wantErr: assert.NoError,
		},
		{
			name: "not found",
			fields: fields{
				RoomID:            "544559-0",
				RoomName:          "DQ(1)",
				RoomDescription:   "",
				MaxGuests:         2,
				IsPrivate:         true,
				RoomBlocked:       false,
				RoomTypeID:        544559,
				RoomTypeName:      "Deluxe Queen",
				RoomTypeNameShort: "DQ",
				PhoneNumber:       "123456789",
				RoomCondition:     "",
				RoomOccupied:      false,
			},
			args: args{
				phoneNumber: "987654321",
			},
			want:    "",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Room{
				RoomID:            tt.fields.RoomID,
				RoomName:          tt.fields.RoomName,
				RoomDescription:   tt.fields.RoomDescription,
				MaxGuests:         tt.fields.MaxGuests,
				IsPrivate:         tt.fields.IsPrivate,
				RoomBlocked:       tt.fields.RoomBlocked,
				RoomTypeID:        tt.fields.RoomTypeID,
				RoomTypeName:      tt.fields.RoomTypeName,
				RoomTypeNameShort: tt.fields.RoomTypeNameShort,
				PhoneNumber:       tt.fields.PhoneNumber,
				RoomCondition:     tt.fields.RoomCondition,
				RoomOccupied:      tt.fields.RoomOccupied,
			}
			got, err := r.SearchRoomIDByPhoneNumber(log, tt.args.phoneNumber, extensionsInfo)
			if !tt.wantErr(t, err, fmt.Sprintf("SearchRoomIDByPhoneNumber(%v)", tt.args.phoneNumber)) {
				return
			}
			assert.Equalf(t, tt.want, got, "SearchRoomIDByPhoneNumber(%v)", tt.args.phoneNumber)
		})
	}
}

func TestCloudbeds_postHousekeepingStatus(t *testing.T) {

	mockRefresher := new(MockTokenRefresher)
	mockRefresher.On("refreshToken").Return("", nil)

	/*type UpdateRoomConditionResponse struct {
		Success bool `json:"success"`
		Data    struct {
			Date          string `json:"date"`
			RoomID        string `json:"roomID"`
			RoomCondition string `json:"roomCondition"`
			DoNotDisturb  bool   `json:"doNotDisturb,omitempty"`
		} `json:"data,omitempty"`
		Message string `json:"message,omitempty"`
	}*/

	type fields struct {
		responseJSON string
		httpStatus   int
	}
	type args struct {
		roomID        string
		roomCondition string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				responseJSON: `{"success":true,"data":{"date":"2020-01-01","roomID":"1","roomCondition":"clean","doNotDisturb":false}}`,
				httpStatus:   http.StatusOK,
			},
			args: args{
				roomID:        "1",
				roomCondition: "clean",
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			fields: fields{
				responseJSON: `{"success":false,"message":"Invalid roomID"}`,
				httpStatus:   http.StatusOK,
			},
			args: args{
				roomID:        "1",
				roomCondition: "clean",
			},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
				return assert.EqualError(t, err, "Failed to update room status: refresh token error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTPClient
			mockClient := new(MockHTTPClient)

			// Create a Cloudbeds instance with the mock HTTPClient
			cb := &Cloudbeds{
				httpClient: mockClient,
				log:        logrus.New(),
				refresher:  mockRefresher,
			}

			mockResponse := &http.Response{
				StatusCode: tt.fields.httpStatus,
				Body:       io.NopCloser(bytes.NewBufferString(tt.fields.responseJSON)),
			}
			mockClient.On("Do", mock.AnythingOfType("*http.Request")).Return(mockResponse, nil)
			err := cb.postHousekeepingStatus(tt.args.roomID, tt.args.roomCondition)
			tt.wantErr(t, err)
		})
	}

}

func TestUpdateRoom(t *testing.T) {
	// Create a mock HTTP server
	postURL := "/api/v1.1/updateRoomCondition"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request is a POST request
		assert.Equal(t, http.MethodPost, r.Method)

		// Check that the request URL matches the expected URL
		assert.Equal(t, postURL, r.URL.Path)

		// Check that the request body contains the expected JSON payload
		expectedBody := "roomCondition=clean&roomID=544559-0"
		bodyBytes, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedBody, string(bodyBytes))

		// Return a successful response
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"success":true,"data":{"date":"2022-01-01","roomID":"123","roomCondition":"clean","doNotDisturb":false}}`))
		assert.NoError(t, err)
	}))
	defer mockServer.Close()

	tests := []struct {
		name          string
		roomExtension string
		condition     string
		expectedMsg   string
		expectErr     bool
		errMsg        string
	}{
		{
			"Valid Room 1",
			"123",
			"clean",
			"Finish UpdateRoom successfully updated room 123 to clean",
			false,
			"",
		},
		{
			"Invalid Room 1",
			"invalid",
			"clean",
			"",
			true,
			"phone number invalid not found",
		},
		{
			"Invalid Room 2",
			"123",
			"dristed",
			"",
			true,
			"room condition dristed is not valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Your existing setup
			cb := &Cloudbeds{
				httpClient:                   mockServer.Client(),
				log:                          logrus.New(),
				apiUrlPostHousekeepingStatus: mockServer.URL + "/api/v1.1/updateRoomCondition",
				roomStatuses:                 []string{"clean", "dirty"},
				configMap: &configuration.ConfigMap{
					ExtensionMap: []configuration.Extension{
						{
							RoomExtension:       "123",
							HospitalityRoomID:   "544559-0",
							HospitalityRoomName: "DQ(1)",
						},
					},
				},
			}

			// Call the UpdateRoom function with test data
			msg, err := cb.UpdateRoom(tt.roomExtension, tt.condition, "John Doe")

			// Check the function's output
			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, msg)
			}
		})
	}

}

// Add your test cases
func TestCloudbeds_setOauth2Config(t *testing.T) {

	// Create a mock SecretStore
	mockSecretStore := new(MockSecretsStore)
	mockSecretStore.On("RetrieveVar", "CLOUDBEDS_SCOPES").Return("", nil)
	mockSecretStore.On("RetrieveVar", "CLOUDBEDS_CLIENT_ID").Return("", nil)
	mockSecretStore.On("RetrieveVar", "CLOUDBEDS_CLIENT_SECRET").Return("", nil)
	mockSecretStore.On("RetrieveVar", "CLOUDBEDS_REDIRECT_URL").Return("", nil)
	mockSecretStore.On("RetrieveVar", "CLOUDBEDS_AUTH_URL").Return("", nil)
	mockSecretStore.On("RetrieveVar", "CLOUDBEDS_TOKEN_URL").Return("", nil)

	//set env variables to satisfy the testable function
	keys := []string{
		"CLOUDBEDS_CLIENT_ID",
		"CLOUDBEDS_CLIENT_SECRET",
		"CLOUDBEDS_REDIRECT_URL",
		"CLOUDBEDS_AUTH_URL",
		"CLOUDBEDS_TOKEN_URL",
		"CLOUDBEDS_SCOPES",
	}

	//set env variables
	os.Setenv("CLOUDBEDS_CLIENT_ID", "test_client_id")
	os.Setenv("CLOUDBEDS_CLIENT_SECRET", "test_client_secret")
	os.Setenv("CLOUDBEDS_REDIRECT_URL", "test_redirect_url")
	os.Setenv("CLOUDBEDS_AUTH_URL", "test_auth_url")
	os.Setenv("CLOUDBEDS_TOKEN_URL", "test_token_url")
	os.Setenv("CLOUDBEDS_SCOPES", "test_scopes")

	type fields struct {
		httpClient  HTTPClient
		storeClient secrets.SecretsStore
		log         *logrus.Logger
		refresher   TokenRefresher
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				httpClient:  &MockHTTPClient{},
				storeClient: mockSecretStore,
				log:         logrus.New(),
				refresher:   &MockTokenRefresher{},
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			fields: fields{
				httpClient:  &MockHTTPClient{},
				storeClient: mockSecretStore,
				log:         logrus.New(),
				refresher:   &MockTokenRefresher{},
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		if tt.name == "error" {
			// Unset env variables after the "success" test case
			for _, key := range keys {
				os.Unsetenv(key)
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			p := &Cloudbeds{
				httpClient:  tt.fields.httpClient,
				storeClient: tt.fields.storeClient,
				log:         tt.fields.log,
				refresher:   tt.fields.refresher,
			}
			tt.wantErr(t, p.setOauth2Config(), "setOauth2Config()")
		})
	}
}

func TestGetRoom(t *testing.T) {
	tests := []struct {
		name           string
		roomNumber     string
		extensionMap   []configuration.Extension
		expectedRoom   hotel.Room
		expectedErrMsg string
	}{
		{
			name:       "Valid room number",
			roomNumber: "123",
			extensionMap: []configuration.Extension{
				{
					RoomExtension:       "123",
					HospitalityRoomID:   "544559-0",
					HospitalityRoomName: "DQ(1)",
				},
			},
			expectedRoom: hotel.Room{
				RoomID:      "544559-0",
				PhoneNumber: "123",
			},
		},
		{
			name:           "Invalid room number",
			roomNumber:     "invalid",
			extensionMap:   []configuration.Extension{},
			expectedErrMsg: "phone number invalid not found",
		},
		{
			name:           "Empty room number",
			roomNumber:     "",
			extensionMap:   []configuration.Extension{},
			expectedErrMsg: "phone number  not found",
		},
		{
			name:       "No matching room extension",
			roomNumber: "999",
			extensionMap: []configuration.Extension{
				{
					RoomExtension:       "123",
					HospitalityRoomID:   "544559-0",
					HospitalityRoomName: "DQ(1)",
				},
			},
			expectedErrMsg: "phone number 999 not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &Cloudbeds{
				log: logrus.New(),
				configMap: &configuration.ConfigMap{
					ExtensionMap: tt.extensionMap,
				},
			}

			actualRoom, err := cb.GetRoom(tt.roomNumber, "map.json")
			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErrMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRoom, actualRoom)
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		expectedErr error
	}{
		{
			name:   "Generate string of length 8",
			length: 8,
		},
		{
			name:   "Generate string of length 16",
			length: 16,
		},
		{
			name:   "Generate string of length 0",
			length: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &Cloudbeds{
				log: logrus.New(),
			}
			str := cb.generateRandomString(tt.length)

			// In our case, the random string is hexadecimal, so each byte generates two characters
			assert.Equal(t, tt.length*2, len(str))
		})
	}
}

type MockSecretStore struct {
	mock.Mock
}

func (m *MockSecretStore) StoreAccessToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockSecretStore) StoreRefreshToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockSecretStore) RetrieveAccessToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSecretStore) RetrieveRefreshToken() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSecretStore) StoreOauthState(state string) error {
	args := m.Called(state)
	return args.Error(0)
}

func (m *MockSecretStore) RetrieveOauthState(state string) (string, error) {
	args := m.Called(state)
	return args.String(0), args.Error(1)
}

func (m *MockSecretStore) RetrieveVar(varName string) (string, error) {
	args := m.Called(varName)
	return args.String(0), args.Error(1)
}

func (m *MockSecretStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Implement other methods if necessary

func TestGetVarFromStoreOrEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		storeValue     string
		storeErr       error
		envValue       string
		expectedResult string
	}{
		{
			name:           "from store",
			storeValue:     "store_value",
			storeErr:       nil,
			envValue:       "env_value",
			expectedResult: "store_value",
		},
		{
			name:           "from environment",
			storeValue:     "",
			storeErr:       nil,
			envValue:       "env_value",
			expectedResult: "env_value",
		},
		{
			name:           "error case",
			storeValue:     "",
			storeErr:       fmt.Errorf("some error"),
			envValue:       "env_value",
			expectedResult: "env_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockStore := new(MockSecretStore)
			mockStore.On("RetrieveVar", mock.Anything).Return(tt.storeValue, tt.storeErr)

			cloudbeds := &Cloudbeds{
				storeClient: mockStore,
				log:         logrus.New(),
			}

			// Set environment variable
			os.Setenv("TEST_VAR", tt.envValue)
			defer os.Unsetenv("TEST_VAR")

			// Execute
			result := cloudbeds.getVarFromStoreOrEnvironment("TEST_VAR")

			// Verify
			assert.Equal(t, tt.expectedResult, result)
			mockStore.AssertExpectations(t)
		})
	}
}

func TestLoadApiConfiguration(t *testing.T) {

	// Create testdata directory if it does not exist
	if _, err := os.Stat("testdata"); os.IsNotExist(err) {
		_ = os.Mkdir("testdata", 0755)
	}

	type APIURLs struct {
		GetRooms               string `json:"getRooms"`
		PostHousekeepingStatus string `json:"postHousekeepingStatus"`
	}

	tests := []struct {
		name           string
		jsonData       string
		apiConfigPath  string
		expectedResult *ApiConfiguration3CX
		wantError      assert.ErrorAssertionFunc
	}{
		{
			name:          "failed to parse config to struct",
			jsonData:      "",
			apiConfigPath: "",
			expectedResult: &ApiConfiguration3CX{
				APIURLs: APIURLs{
					GetRooms:               "",
					PostHousekeepingStatus: "",
				},
				RoomStatuses: nil,
			},
			wantError: assert.Error,
		},
		{
			name: "cant parse config to struct",
			jsonData: `{
		"apiURLs": {
			"getRooms": "https://hotels.cloudbeds.com/api/v1.2/getRooms",
			"postHousekeepingStatus": "https://hotels.cloudbeds.com/api/v1.2/postHousekeepingStatus"
		},
		"roomStatuses": ["clean", "dirty"]
	}`,
			apiConfigPath: "testdata/cloudbeds_api_params.json",
			expectedResult: &ApiConfiguration3CX{
				APIURLs: APIURLs{
					GetRooms:               "https://hotels.cloudbeds.com/api/v1.2/getRooms",
					PostHousekeepingStatus: "https://hotels.cloudbeds.com/api/v1.2/postHousekeepingStatus",
				},
				RoomStatuses: []string{"clean", "dirty"},
			},
			wantError: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			apiConfigPath := "testdata/cloudbeds_api_params.json"
			// Create and write the JSON data to a test file
			err := os.WriteFile("testdata/cloudbeds_api_params.json", []byte(tt.jsonData), 0644)
			if err != nil {
				t.Fatalf("Could not create test file: %v", err)
			}

			// Setup logger and run the test function
			log := logrus.New()
			apiConfiguration, err := loadApiConfiguration(log, tt.apiConfigPath)

			// Assertions
			tt.wantError(t, err)
			assert.Equal(t, tt.expectedResult, apiConfiguration)

			// Clean up: remove the test file
			os.Remove(apiConfigPath)
		})
	}

	//remove test directory if any
	os.Remove("testdata")

}
