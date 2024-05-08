// Package hotel is a package that provides an interface for a hotel.
// It is a common interface for different hospitality providers
package hotel

/*
	Response: {
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
	                },
	                {
	                    "roomID": "544559-1",
	                    "roomName": "DQ(2)",
	                    "roomDescription": "",
	                    "maxGuests": 2,
	                    "isPrivate": true,
	                    "roomBlocked": false,
	                    "roomTypeID": 544559,
	                    "roomTypeName": "Deluxe Queen",
	                    "roomTypeNameShort": "DQ"
	                },
	                {
	                    "roomID": "544559-2",
	                    "roomName": "DQ(3)",
	                    "roomDescription": "",
	                    "maxGuests": 2,
	                    "isPrivate": true,
	                    "roomBlocked": false,
	                    "roomTypeID": 544559,
	                    "roomTypeName": "Deluxe Queen",
	                    "roomTypeNameShort": "DQ"
	                },

	{
	                    "roomID": "544560-9",
	                    "roomName": "DK(10)",
	                    "roomDescription": "",
	                    "maxGuests": 2,
	                    "isPrivate": true,
	                    "roomBlocked": false,
	                    "roomTypeID": 544560,
	                    "roomTypeName": "Deluxe King",
	                    "roomTypeNameShort": "DK"
	                }
	            ]
	        }
	    ],
	    "count": 20,
	    "total": 20
	}
*/

// Room is a struct that represents a room in a hospitality provider
type Room struct {
	RoomID            string `json:"roomID"`
	RoomName          string `json:"roomName"`
	RoomDescription   string `json:"roomDescription"`
	MaxGuests         int32  `json:"maxGuests"`
	IsPrivate         bool   `json:"isPrivate"`
	RoomBlocked       bool   `json:"roomBlocked"`
	RoomTypeID        int32  `json:"roomTypeID"`
	RoomTypeName      string `json:"roomTypeName"`
	RoomTypeNameShort string `json:"roomTypeNameShort"`
	PhoneNumber       string `json:"phoneNumber,omitempty"`
	RoomCondition     string `json:"RoomCondition,omitempty"`
	RoomOccupied      bool   `json:"RoomOccupied,omitempty"`
}

// HospitalityProvider is an interface that represents a hospitality provider
type HospitalityProvider interface {
	GetRooms() ([]Room, error)
	GetRoom(roomNumber string, mapFileName string) (Room, error)
	UpdateRoom(roomNumber, housekeepingStatus, housekeeperName string) (msg string, err error)
	HandleOAuthCallback(state, code string) (err error)
	HandleInitialLogin() (url string, err error)
}

// DetailedError is a struct that represents an error with a status code and details
type DetailedError struct {
	Msg               error
	StatusCodeMessage string
	Details           string
}

// Error returns the error message
func (e *DetailedError) Error() string { return e.Msg.Error() }
