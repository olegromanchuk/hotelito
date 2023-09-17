package cloudbeds

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/secrets"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

// GoMockHTTPClient is a mock of HTTPClient interface.
type GoMockHTTPClient struct {
	ctrl     *gomock.Controller
	recorder *GoMockHTTPClientGoMockRecorder
}

// GoMockHTTPClientGoMockRecorder is the mock recorder for GoMockHTTPClient.
type GoMockHTTPClientGoMockRecorder struct {
	mock *GoMockHTTPClient
}

// NewMockHTTPClient creates a new mock instance.
func NewMockHTTPClient(ctrl *gomock.Controller) *GoMockHTTPClient {
	mock := &GoMockHTTPClient{ctrl: ctrl}
	mock.recorder = &GoMockHTTPClientGoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *GoMockHTTPClient) EXPECT() *GoMockHTTPClientGoMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *GoMockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", req)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Do indicates an expected call of Do.
func (mr *GoMockHTTPClientGoMockRecorder) Do(req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*GoMockHTTPClient)(nil).Do), req)
}

// Get mocks base method.
func (m *GoMockHTTPClient) Get(url string) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", url)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *GoMockHTTPClientGoMockRecorder) Get(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*GoMockHTTPClient)(nil).Get), url)
}

// GoMockTokenRefresher is a mock of TokenRefresher interface.
type GoMockTokenRefresher struct {
	ctrl     *gomock.Controller
	recorder *GoMockTokenRefresherGoMockRecorder
}

// GoMockTokenRefresherGoMockRecorder is the mock recorder for GoMockTokenRefresher.
type GoMockTokenRefresherGoMockRecorder struct {
	mock *GoMockTokenRefresher
}

// NewMockTokenRefresher creates a new mock instance.
func NewMockTokenRefresher(ctrl *gomock.Controller) *GoMockTokenRefresher {
	mock := &GoMockTokenRefresher{ctrl: ctrl}
	mock.recorder = &GoMockTokenRefresherGoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *GoMockTokenRefresher) EXPECT() *GoMockTokenRefresherGoMockRecorder {
	return m.recorder
}

// refreshToken mocks base method.
func (m *GoMockTokenRefresher) refreshToken() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "refreshToken")
	ret0, _ := ret[0].(error)
	return ret0
}

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

type MockOauthConfInterface struct {
	OauthConfInterface
	mock.Mock
}

func (m *MockOauthConfInterface) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	args := m.Called(state, opts)
	return args.String(0)
}

func (m *MockOauthConfInterface) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	args := m.Called(ctx, code, opts)
	return args.Get(0).(*oauth2.Token), args.Error(1)
}

func (m *MockOauthConfInterface) TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource {
	args := m.Called(ctx, t)
	return args.Get(0).(oauth2.TokenSource) // Note the type assertion to oauth2.TokenSource
}

func (m *MockOauthConfInterface) Client(ctx context.Context, t *oauth2.Token) *http.Client {
	args := m.Called(ctx, t)
	return args.Get(0).(*http.Client)
}

func TestCloudbeds_Room_ToHotelRoom(t *testing.T) {
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

func cleanupTestFiles(filename string) {
	if err := os.Remove(filename); err != nil {
		log.Fatalf("Failed to clean up test file: %s", err)
	}
}

func cleanUpEnvVars() {
	keys := []string{
		"CLOUDBEDS_CLIENT_ID",
		"CLOUDBEDS_CLIENT_SECRET",
		"CLOUDBEDS_REDIRECT_URL",
		"CLOUDBEDS_AUTH_URL",
		"CLOUDBEDS_TOKEN_URL",
		"CLOUDBEDS_SCOPES",
	}
	for _, key := range keys {
		os.Unsetenv(key)
	}
}

func TestCloudbeds_GetRooms(t *testing.T) {
	cleanUpEnvVars()
	//expected error for Bad JSON Format
	typeValue := reflect.TypeOf(true) // Type of boolean since 'success' is expected to be a bool

	expectedUnmarshalTypeError := &json.UnmarshalTypeError{
		Value:  "string",           // The incorrect value that we tried to unmarshal
		Type:   typeValue,          // The expected type
		Offset: 23,                 // Byte offset in input where error occurs
		Struct: "ResponseGetRooms", // Struct that has the problematic field
		Field:  "success",          // The problematic field
	}

	// expected hotel.DetailedError
	expectedError := &hotel.DetailedError{
		Msg: expectedUnmarshalTypeError,
		Details: fmt.Sprintf("success, but parse return body: %s",
			errors.New("json: cannot unmarshal string into Go struct field ResponseGetRooms.success of type bool")),
	}

	generalMockUrl := "https://hotels.cloudbeds.com/api/v1.1/getRooms"

	testCases := []struct {
		desc             string
		mockResp         string
		mockStatus       int
		mockError        error
		mockUrl          string
		expectedResponse []hotel.Room
		expectedError    error
		expectError      bool
	}{
		{
			desc: "Successful Response",
			mockResp: `{
				"success": true,
				"data": [{"propertyID": "297652", "rooms": [{"roomID": "544559-0"}]}]
			}`,
			mockStatus: http.StatusOK,
			mockError:  nil,
			mockUrl:    generalMockUrl,
			expectedResponse: []hotel.Room{
				{
					RoomID: "544559-0",
				},
			},
			expectedError: nil,
			expectError:   false,
		},
		{
			desc: "Failed Response",
			mockResp: `{
				"success": false,
				"message": "Something went wrong"
			}`,
			mockStatus:       http.StatusOK,
			mockError:        nil,
			mockUrl:          generalMockUrl,
			expectedResponse: []hotel.Room{},
			expectedError:    errors.New("refresh token error"),
			expectError:      true,
		},
		{
			desc: "Bad JSON Format",
			mockResp: `{
				"success": "true",
				"data": "bad_data"
			}`,
			mockStatus: http.StatusOK,
			mockError:  nil,
			mockUrl:    generalMockUrl,
			expectedResponse: []hotel.Room{
				{
					RoomID: "544559-0",
				},
			},
			expectedError: expectedError,
			expectError:   true,
		},
		{
			desc: "Not found API URL",
			mockResp: `{
				"success": "true",
				"data": "bad_data"
			}`,
			mockStatus: http.StatusNotFound,
			mockError:  errors.New("Not found"),
			mockUrl:    "https://some_garbage",
			expectedResponse: []hotel.Room{
				{
					RoomID: "544559-0",
				},
			},
			expectedError: errors.New("request failed with: Not found"),
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockClient := new(MockHTTPClient)
			mockRefresher := new(MockTokenRefresher)
			mockRefresher.On("refreshToken").Return("", nil)
			cb := &Cloudbeds{
				httpClient:     mockClient,
				log:            logrus.New(),
				apiUrlGetRooms: tc.mockUrl,
				refresher:      mockRefresher,
			}

			resp := &http.Response{
				StatusCode: tc.mockStatus,
				Body:       io.NopCloser(bytes.NewBufferString(tc.mockResp)),
			}
			mockClient.On("Get", cb.apiUrlGetRooms).Return(resp, tc.mockError)

			testResult, err := cb.GetRooms()

			if tc.expectError {
				assert.NotNil(t, err, "Expected error but got none")
				assert.Equal(t, tc.expectedError, err)
			} else {
				assert.Nil(t, err, "Expected no error but got one")
				assert.Equal(t, tc.expectedResponse, testResult)
			}

			mockClient.AssertExpectations(t)
		})
	}

}

func TestCloudbeds_Room_SearchRoomIDByPhoneNumber(t *testing.T) {
	cleanUpEnvVars()
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
	cleanUpEnvVars()
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

func TestCloudbeds_UpdateRoom(t *testing.T) {
	cleanUpEnvVars()
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
	cleanUpEnvVars()
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

func TestCloudbeds_GetRoom(t *testing.T) {
	cleanUpEnvVars()
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

func TestCloudbeds_GenerateRandomString(t *testing.T) {
	cleanUpEnvVars()
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

// Implement other methods if necessary

func TestCloudbeds_GetVarFromStoreOrEnvironment(t *testing.T) {
	cleanUpEnvVars()
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
			mockStore := new(MockSecretsStore)
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

func TestCloudbeds_LoadApiConfiguration(t *testing.T) {
	cleanUpEnvVars()
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

func TestHandleInitialLogin(t *testing.T) {
	//define variables to speed up
	cleanUpEnvVars()

	tests := []struct {
		name                 string
		mockOauth2Config     error
		mockOauthState       string
		mockStoreErr         error
		setEnvVars           bool
		expectedMockOauthURL string
		expectErrMsg         string
		expectErr            bool
	}{
		{
			name:                 "successful case",
			mockOauth2Config:     nil,
			mockOauthState:       "randomState",
			mockStoreErr:         nil,
			expectedMockOauthURL: "test_auth_url?client_id=test_client_id&redirect_uri=test_redirect_url&response_type=code&scope=test_scopes&state",
			setEnvVars:           true,
			expectErrMsg:         "",
			expectErr:            false,
		},
		{
			name:                 "error: not all vars are set",
			mockOauth2Config:     nil,
			mockOauthState:       "randomState",
			mockStoreErr:         nil,
			setEnvVars:           false,
			expectedMockOauthURL: "http://auth.url",
			expectErrMsg:         "Not all required env variables are set. Missed one of: CLOUDBEDS_CLIENT_ID, CLOUDBEDS_CLIENT_SECRET, CLOUDBEDS_REDIRECT_URL, CLOUDBEDS_SCOPES, CLOUDBEDS_AUTH_URL, CLOUDBEDS_TOKEN_URL",
			expectErr:            true,
		},
		{
			name:                 "error: oauth2 config",
			mockOauth2Config:     errors.New("config error"),
			mockOauthState:       "randomState",
			mockStoreErr:         errors.New("store error"),
			setEnvVars:           true,
			expectedMockOauthURL: "http://auth.url",
			expectErrMsg:         "store error",
			expectErr:            true,
		},
		// Add more test cases here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSecretStore := new(MockSecretsStore)
			mockLogger := logrus.New()
			mockLogger.SetOutput(io.Discard)

			cleanUpEnvVars()

			if tt.setEnvVars {
				//set env variables
				os.Setenv("CLOUDBEDS_CLIENT_ID", "test_client_id")
				os.Setenv("CLOUDBEDS_CLIENT_SECRET", "test_client_secret")
				os.Setenv("CLOUDBEDS_REDIRECT_URL", "test_redirect_url")
				os.Setenv("CLOUDBEDS_AUTH_URL", "test_auth_url")
				os.Setenv("CLOUDBEDS_TOKEN_URL", "test_token_url")
				os.Setenv("CLOUDBEDS_SCOPES", "test_scopes")
			}

			cloudbeds := &Cloudbeds{
				storeClient: mockSecretStore,
				log:         mockLogger,
			}

			//Bypass the RetrieveVar call.
			//Trade-off here: we do not to check internal implementation of the function thus making the test less fragile in case of tested function refactoring.
			//Another option would be to do .Once(), .Twice() etc. on the mockSecretStore. That would make the test more fragile in case of refactoring and depended on the internal implementation of tested function (i.e. call of "RetrieveVar"). However, it would check that the function is called with the correct parameters and it will not leave leftovers in case of "RetrieveVar" is removed from tested function.
			mockSecretStore.On("RetrieveVar", mock.Anything).Return("", nil)
			if tt.setEnvVars {
				mockSecretStore.On("StoreOauthState", mock.Anything).Return(tt.mockStoreErr)
			}

			url, err := cloudbeds.HandleInitialLogin()

			if tt.expectErr {
				assert.NotNil(t, err)
				assert.Equal(t, tt.expectErrMsg, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Contains(t, url, tt.expectedMockOauthURL)
			}

			mockSecretStore.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	cleanUpEnvVars()

	tests := []struct {
		name                     string
		mockRetrieveRefreshToken string
		mockRetrieveRefreshErr   error
		mockStoreOauthStateErr   error
		oauthTockenSource        string
		expectedStatusCodeMsg    string
		expectedMsg              string
		expectErrMsg             string
		expectErr                bool
	}{
		{
			name:                     "Case 1: Valid Refresh Token",
			mockRetrieveRefreshToken: "some_valid_token",
			mockRetrieveRefreshErr:   nil,
			mockStoreOauthStateErr:   nil,
			oauthTockenSource:        "someURL",
			expectedStatusCodeMsg:    "ok",
			expectedMsg:              "",
			expectErrMsg:             "",
			expectErr:                false,
		},
		{
			name:                     "Case 2: Invalid Refresh Token",
			mockRetrieveRefreshToken: "",
			mockRetrieveRefreshErr:   errors.New("no-refresh-token-found"),
			mockStoreOauthStateErr:   nil,
			expectedStatusCodeMsg:    "fatal-error",
			expectedMsg:              "failed to retrieve refresh token from secret store: no-refresh-token-found",
			expectErrMsg:             "failed to retrieve refresh token from secret store: no-refresh-token-found",
			expectErr:                true,
		},
		// Add more cases
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSecretStore := new(MockSecretsStore)
			mockLogger := logrus.New()
			mockLogger.SetOutput(io.Discard)
			mockOauthConf := new(MockOauthConfInterface)

			tokenSource := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: "some-access-token"},
			)

			cloudbeds := &Cloudbeds{
				storeClient: mockSecretStore,
				log:         mockLogger,
				oauthConf:   mockOauthConf,
			}

			mockSecretStore.On("RetrieveRefreshToken").Return(tt.mockRetrieveRefreshToken, tt.mockRetrieveRefreshErr)
			mockSecretStore.On("StoreOauthState", mock.Anything).Return(tt.mockStoreOauthStateErr)
			mockOauthConf.On("AuthCodeURL", mock.Anything, mock.Anything).Return("someAuthCodeURL")
			mockOauthConf.On("TokenSource", mock.Anything, mock.AnythingOfType("*oauth2.Token")).Return(tokenSource)

			statusCodeMsg, msg, err := cloudbeds.login(mockSecretStore)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedStatusCodeMsg, statusCodeMsg)
			assert.Equal(t, tt.expectedMsg, msg)

		})
	}
}

func TestCloudbeds_RefreshToken(t *testing.T) {
	cleanUpEnvVars()

	tests := []struct {
		name              string
		expectedError     error
		setOauth2Config   error
		refreshTokenErr   error
		newTokenErr       error
		retrieveVarString string
		storeAccessToken  error
		setEnv            bool
	}{
		//do not test success. Too complex to mock
		{
			name:              "setOauth2Config error",
			setOauth2Config:   errors.New("someError"),
			expectedError:     errors.New("Not all required env variables are set. Missed one of: CLOUDBEDS_CLIENT_ID, CLOUDBEDS_CLIENT_SECRET, CLOUDBEDS_REDIRECT_URL, CLOUDBEDS_SCOPES, CLOUDBEDS_AUTH_URL, CLOUDBEDS_TOKEN_URL"),
			refreshTokenErr:   nil,
			retrieveVarString: "",
		},
		{
			name:              "RetrieveRefreshToken error",
			setOauth2Config:   nil,
			expectedError:     errors.New("failed to retrieve refresh token from secret store: someError"),
			refreshTokenErr:   errors.New("someError"),
			retrieveVarString: "some_value",
		},
		{
			name:              "failed protocol schema",
			setOauth2Config:   nil,
			expectedError:     errors.New("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again. Error: Post \"some_value\": unsupported protocol scheme \"\""),
			refreshTokenErr:   nil,
			retrieveVarString: "some_value",
		},
		{
			name:              "host lookup failed",
			setOauth2Config:   nil,
			expectedError:     errors.New("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again. Error: Post"),
			refreshTokenErr:   nil,
			retrieveVarString: "https://some_value",
		},
		{
			name:              "host available, but oauth2 returns error",
			setOauth2Config:   nil,
			expectedError:     errors.New("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again. Error: oauth2"),
			refreshTokenErr:   nil,
			retrieveVarString: "https://google.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger := logrus.New()
			logger.SetOutput(io.Discard)
			mockStoreClient := new(MockSecretsStore)
			mockOauthConf := new(MockOauthConfInterface)
			p := &Cloudbeds{
				log:         logger,
				storeClient: mockStoreClient,
				oauthConf:   mockOauthConf,
			}

			mockStoreClient.On("RetrieveRefreshToken").Return("someRefreshToken", tt.refreshTokenErr)
			mockOauthConf.On("TokenSource", mock.Anything, mock.Anything).Return(mock.Anything)
			mockOauthConf.On("Client", mock.Anything, mock.Anything).Return(&http.Client{})
			mockStoreClient.On("StoreAccessToken", mock.Anything).Return(tt.storeAccessToken)
			mockStoreClient.On("RetrieveVar", mock.Anything).Return(tt.retrieveVarString, nil)

			err := p.refreshToken()

			assert.Contains(t, err.Error(), tt.expectedError.Error())
		})
	}
}

func TestHandleOAuthCallback(t *testing.T) {
	cleanUpEnvVars()
	tests := []struct {
		name            string
		state           string
		code            string
		mockState       string
		mockStateErr    error
		mockToken       *oauth2.Token
		mockTokenErr    error
		mockStoreAT     error
		mockStoreRT     error
		mockHttpClient  *http.Client
		exchangeOptions []oauth2.AuthCodeOption
		expectedError   string
	}{
		{
			name:           "RetrieveOauthState fails",
			state:          "someState",
			code:           "someCode",
			mockStateErr:   errors.New("state error"),
			mockToken:      &oauth2.Token{AccessToken: "some_token"},
			mockHttpClient: &http.Client{},
			expectedError:  "failed to retrieve oauth state",
		},
		{
			name:           "state doesn't match",
			state:          "someState",
			code:           "someCode",
			mockState:      "otherState",
			mockToken:      &oauth2.Token{AccessToken: "some_token"},
			mockHttpClient: &http.Client{},
			expectedError:  "invalid oauth state",
		},
		{
			name:            "Exchange fails",
			state:           "someState",
			code:            "someCode",
			mockState:       "someState",
			mockToken:       &oauth2.Token{AccessToken: "some_token"},
			mockTokenErr:    errors.New("exchange error"),
			mockHttpClient:  &http.Client{},
			exchangeOptions: []oauth2.AuthCodeOption{},
			expectedError:   "oauthConf.Exchange() failed",
		},
		{
			name:            "StoreAccessToken fails",
			state:           "someState",
			code:            "someCode",
			mockState:       "someState",
			mockToken:       &oauth2.Token{AccessToken: "some_token"},
			mockStoreAT:     errors.New("store access token error"),
			mockHttpClient:  &http.Client{},
			exchangeOptions: []oauth2.AuthCodeOption{},
			expectedError:   "store access token error",
		},
		{
			name:            "StoreRefreshToken fails",
			state:           "someState",
			code:            "someCode",
			mockState:       "someState",
			mockToken:       &oauth2.Token{AccessToken: "some_token"},
			mockStoreRT:     errors.New("store refresh token error"),
			mockHttpClient:  &http.Client{},
			exchangeOptions: []oauth2.AuthCodeOption{},
			expectedError:   "store refresh token error",
		},
		{
			name:            "success",
			state:           "someState",
			code:            "someCode",
			mockState:       "someState",
			mockToken:       &oauth2.Token{AccessToken: "some_token"},
			mockHttpClient:  &http.Client{},
			exchangeOptions: []oauth2.AuthCodeOption{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := new(MockSecretsStore)
			mockOauth := new(MockOauthConfInterface)
			logger := logrus.New()
			logger.SetOutput(io.Discard)

			p := &Cloudbeds{
				storeClient: mockStore,
				oauthConf:   mockOauth,
				log:         logger,
			}

			mockStore.On("RetrieveOauthState", tt.state).Return(tt.mockState, tt.mockStateErr)
			mockOauth.On("Exchange", context.Background(), tt.code, []oauth2.AuthCodeOption(nil)).Return(tt.mockToken, tt.mockTokenErr)
			mockStore.On("StoreAccessToken", mock.Anything).Return(tt.mockStoreAT)
			mockStore.On("StoreRefreshToken", mock.Anything).Return(tt.mockStoreRT)
			mockOauth.On("Client", context.Background(), mock.AnythingOfType("*oauth2.Token")).Return(tt.mockHttpClient)

			err := p.HandleOAuthCallback(tt.state, tt.code)

			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestNew(t *testing.T) {

	apiConfFileName := "test_cloudbeds_api_params.json"
	createCloudbedsApiConfigFile(apiConfFileName)

	tests := []struct {
		name                    string
		loadConfigErr           error
		retrieveAccessTokenErr  error
		retrieveRefreshTokenErr error
		retrieveVarValue        string
		retrieveVarErr          error
		loginMsg                string
		loginStatusCode         string
		apiConfFileName         string
		setEnvVars              bool
		expectedErr             string
	}{
		{
			name:            "loadApiConfiguration error",
			loadConfigErr:   errors.New("failed to load config"),
			expectedErr:     "no such file or directory",
			apiConfFileName: "file_not_exists.json",
		},
		{
			name:             "setOauth2Config error",
			expectedErr:      "Not all required env variables are set",
			retrieveVarValue: "",
			retrieveVarErr:   nil,
			setEnvVars:       false,
			apiConfFileName:  apiConfFileName,
		},
		{
			name:            "success",
			setEnvVars:      true,
			apiConfFileName: apiConfFileName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanUpEnvVars()
			if tt.setEnvVars {
				//set env variables
				os.Setenv("CLOUDBEDS_CLIENT_ID", "test_client_id")
				os.Setenv("CLOUDBEDS_CLIENT_SECRET", "test_client_secret")
				os.Setenv("CLOUDBEDS_REDIRECT_URL", "test_redirect_url")
				os.Setenv("CLOUDBEDS_AUTH_URL", "test_auth_url")
				os.Setenv("CLOUDBEDS_TOKEN_URL", "test_token_url")
				os.Setenv("CLOUDBEDS_SCOPES", "test_scopes")
			}

			logger := logrus.New()
			logger.SetOutput(io.Discard)
			mockStoreClient := new(MockSecretsStore)
			mockConfig := &configuration.ConfigMap{ApiCfgFileName: tt.apiConfFileName}

			mockStoreClient.On("RetrieveAccessToken").Return("some_token", tt.retrieveAccessTokenErr)
			mockStoreClient.On("RetrieveRefreshToken").Return("some_token", tt.retrieveRefreshTokenErr)
			mockStoreClient.On("RetrieveVar", mock.Anything).Return(tt.retrieveVarValue, tt.retrieveVarErr) // Replace "some_value" and nil with whatever you want the function to return

			client, err := New(logger, mockStoreClient, mockConfig)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
	cleanUpEnvVars()
	cleanupTestFiles(apiConfFileName)
}

func createCloudbedsApiConfigFile(filename string) {
	jsonStr := `{
  "apiURLs": {
    "getRooms": "https://hotels.cloudbeds.com/api/v1.2/getRooms",
    "postHousekeepingStatus": "https://hotels.cloudbeds.com/api/v1.2/postHousekeepingStatus"
  },
  "roomStatuses": [
    "clean",
    "dirty"
  ]
}`

	err := os.WriteFile(filename, []byte(jsonStr), 0644)
	if err != nil {
		log.Fatalf("Writing to file failed: %s", err)
	}
}

func TestNewClient4CallbackAndInit(t *testing.T) {
	cleanUpEnvVars()
	mockLog := logrus.New()
	mockLog.SetOutput(io.Discard)

	testCases := []struct {
		name             string
		accessToken      string
		accessTokenError error
		expectedError    error
		expectedClient   bool
	}{
		{
			name:             "Success case: setOauth2Config() doesn't return an error",
			accessToken:      "some-access-token",
			accessTokenError: nil,
			expectedError:    nil,
			expectedClient:   true,
		},
		{
			name:             "Error case: setOauth2Config() returns an error",
			accessToken:      "",
			accessTokenError: errors.New("some error"),
			expectedError:    errors.New("Not all required env variables are set. Missed one of"),
			expectedClient:   false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockSecretStore := new(MockSecretsStore)
			mockSecretStore.On("RetrieveVar", mock.Anything).Return(tt.accessToken, tt.accessTokenError)
			cloudbedsClient, err := NewClient4CallbackAndInit(mockLog, mockSecretStore)
			if tt.expectedError != nil {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.Nil(t, err)
			}

			if tt.expectedClient {
				assert.NotNil(t, cloudbedsClient)
			} else {
				assert.Nil(t, cloudbedsClient)
			}

		})
	}
}

func TestClose(t *testing.T) {
	testCases := []struct {
		name                  string
		closeError            error
		expectedLogErrorValue error
	}{
		{
			name:                  "Success case: Close() doesn't return an error",
			closeError:            nil,
			expectedLogErrorValue: nil,
		},
		{
			name:                  "Error case: Close() returns an error",
			closeError:            errors.New("some error"),
			expectedLogErrorValue: errors.New("failed to close secret store: some error"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := logrus.New()
			mockSecretStore := new(MockSecretsStore)
			// Using a log hook to capture log entries for assertion
			hook := test.NewLocal(mockLog)

			mockSecretStore.On("Close").Return(tt.closeError)

			cloudbedsClient := &Cloudbeds{
				log:         mockLog,
				storeClient: mockSecretStore,
			}

			err := cloudbedsClient.Close()

			if tt.expectedLogErrorValue != nil {
				assert.Equal(t, 1, len(hook.Entries))
				assert.Equal(t, err, tt.expectedLogErrorValue)
				assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
				assert.Contains(t, hook.LastEntry().Message, tt.expectedLogErrorValue.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, 0, len(hook.Entries))
			}
		})
	}
}
