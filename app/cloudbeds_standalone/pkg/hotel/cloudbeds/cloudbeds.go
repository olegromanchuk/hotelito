package cloudbeds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/secrets"
	"github.com/olegromanchuk/hotelito/pkg/secrets/boltstore"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
	"strings"
)

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

const (
	oauthStateString = "random"
)

var (
	oauthConf *oauth2.Config
	log       *logrus.Logger
)

type Cloudbeds struct {
}

type ResponseGetRooms struct {
	Success bool `json:"success"`
	Data    []struct {
		PropertyID string `json:"propertyID"`
		Rooms      []Room `json:"rooms"`
	} `json:"data"`
	Count int `json:"count"`
	Total int `json:"total"`
}

type UpdateRoomConditionResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Date          string `json:"date"`
		RoomID        string `json:"roomID"`
		RoomCondition string `json:"roomCondition"`
		DoNotDisturb  bool   `json:"doNotDisturb,omitempty"`
	} `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func (p *Cloudbeds) GetRooms() (rooms []Room, err error) {
	// Provider1's implementation of GetRooms
	return rooms, nil
}

//func (p *Cloudbeds) BookRoom(roomID string, date time.Time) (reservations Reservation, err error) {
//	// Provider1's implementation of BookRoom
//	return nil
//}

func (p *Cloudbeds) CancelReservation(reservationID string) error {
	// Provider1's implementation of CancelReservation
	return nil
}

func (r *Room) SearchRoomIDByPhoneNumber(phoneNumber string) (string, error) {
	type RoomMap map[string]string
	// Read file
	jsonFile, err := os.Open("roomid_map.json")
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	// Initialize a map to store the JSON data in
	var roomMap RoomMap

	// Unmarshal the JSON data into the map
	json.Unmarshal(byteValue, &roomMap)

	// Look up room ID by phone number
	roomID, ok := roomMap[phoneNumber]
	if !ok {
		return "", fmt.Errorf("phone number not found")
	}

	return roomID, nil
}

func (p *Cloudbeds) HandleLogin(w http.ResponseWriter, r *http.Request) {
	setOauth2Config()
	url := oauthConf.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (p *Cloudbeds) HandleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != oauthStateString {
		fmt.Printf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	log.Debugf("Got auth code: %s", code)
	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("oauthConf.Exchange() failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	log.Debugf("Got access token: %s", token)

	os.Setenv("ACCESS_TOKEN", token.AccessToken)

	// Make client request using the obtained token
	client := oauthConf.Client(context.Background(), token)

	err = postHousekeepingAssignment(client, "544559-1", "clean")

	//client := oauthConf.Client(context.Background(), token)
	//allRooms,err := getRooms(client)
	//if err != nil {
	//	fmt.Printf("getRooms() failed with '%s'\n", err)
	//	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	//	return
	//}
}

func New(logMain *logrus.Logger) *Cloudbeds {
	log = logMain

	cloudbedsClient := &Cloudbeds{}

	//get access_token

	//current secret store - boltDB
	boltDb, err := boltstore.New()
	if err != nil {
		log.Fatal(err)
	}
	defer boltDb.Db.Close()

	var store secrets.SecretsStore
	store = boltDb

	accessToken, err := store.RetrieveAccessToken()
	if err != nil {
		cloudbedsClient.Login()
	}
	os.Setenv("CLOUDBEDS_ACCESS_TOKEN", accessToken)

	//check if access_token is valid. If not - get refresh_token and update access_token
	return &Cloudbeds{}
}

func setOauth2Config() {
	scopes := strings.Split(os.Getenv("CLOUDBEDS_SCOPES"), ",")
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("CLOUDBEDS_CLIENT_ID"),
		ClientSecret: os.Getenv("CLOUDBEDS_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("CLOUDBEDS_REDIRECT_URL"),
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv("CLOUDBEDS_AUTH_URL"),
			TokenURL: os.Getenv("CLOUDBEDS_TOKEN_URL"),
		},
	}
}
