package cloudbeds

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
	"pkg/models"
	"strings"
)

const (
	oauthStateString = "random"
)

var (
	oauthConf *oauth2.Config
	log       *logrus.Logger
)

type Cloudbeds struct {
}

type Room models.Room

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

func handleLogin(w http.ResponseWriter, r *http.Request) {
	setOauth2Config()
	url := oauthConf.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func setOauth2Config() {
	scopes := strings.Split(os.Getenv("SCOPES"), ",")
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  os.Getenv("AUTH_URL"),
			TokenURL: os.Getenv("TOKEN_URL"),
		},
	}
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
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
	//get access token
	return &Cloudbeds{}
}
