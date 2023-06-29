package cloudbeds

import (
	"bytes"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
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

	// Setup: create roomid_map.json file
	roomMapData := `{
		"100": "544559-0",
		"101": "544559-1"
    }`

	err := os.WriteFile("roomid_map_test.json", []byte(roomMapData), 0644)
	if err != nil {
		t.Fatalf("unable to set up test: %v", err)
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
			got, err := r.SearchRoomIDByPhoneNumber(tt.args.phoneNumber, "roomid_map_test.json")
			if !tt.wantErr(t, err, fmt.Sprintf("SearchRoomIDByPhoneNumber(%v)", tt.args.phoneNumber)) {
				return
			}
			assert.Equalf(t, tt.want, got, "SearchRoomIDByPhoneNumber(%v)", tt.args.phoneNumber)
		})
	}

	// Teardown: delete roomid_map.json file
	err = os.Remove("roomid_map_test.json")
	if err != nil {
		t.Errorf("unable to tear down test: %v", err)
	}
}

func TestCloudbeds_postHousekeepingStatus(t *testing.T) {

	mockRefresher := new(MockTokenRefresher)
	mockRefresher.On("refreshToken").Return(nil)

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

//
//func TestCloudbeds_postHousekeepingStatus1(t *testing.T) {
//	type fields struct {
//		httpClient  HTTPClient
//		storeClient secrets.SecretsStore
//		log         *logrus.Logger
//	}
//	type args struct {
//		roomID        string
//		roomCondition string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr assert.ErrorAssertionFunc
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Cloudbeds{
//				httpClient:  tt.fields.httpClient,
//				storeClient: tt.fields.storeClient,
//				log:         tt.fields.log,
//			}
//			tt.wantErr(t, p.postHousekeepingStatus(tt.args.roomID, tt.args.roomCondition), fmt.Sprintf("postHousekeepingStatus(%v, %v)", tt.args.roomID, tt.args.roomCondition))
//		})
//	}
//}
