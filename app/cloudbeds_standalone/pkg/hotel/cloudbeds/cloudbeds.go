package cloudbeds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/secrets"
	"github.com/olegromanchuk/hotelito/pkg/secrets/boltstore"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/url"
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
	oauthConf                  *oauth2.Config
	log                        *logrus.Logger
	loginLoopPreventionCounter = 1
)

type Cloudbeds struct {
	httpClient  *http.Client
	storeClient secrets.SecretsStore
}

type UpdateRoomConditionRequest struct {
	RoomID        string `json:"roomID"`
	RoomCondition string `json:"roomCondition"`
	PropertyID    int    `json:"propertyID,omitempty"`
	DoNotDisturb  bool   `json:"doNotDisturb,omitempty"`
}

/*
	Response: {
	    "success": false,
	    "message": "Parameter roomID is required"
	}
*/
type ResponseGetRooms struct {
	Success bool `json:"success"`
	Data    []struct {
		PropertyID string `json:"propertyID"`
		Rooms      []Room `json:"rooms"`
	} `json:"data"`
	Count   int    `json:"count"`
	Total   int    `json:"total"`
	Message string `json:"message,omitempty"`
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

func (r Room) ToHotelRoom() hotel.Room {
	return hotel.Room{
		RoomID:            r.RoomID,
		RoomName:          r.RoomName,
		RoomDescription:   r.RoomDescription,
		MaxGuests:         r.MaxGuests,
		IsPrivate:         r.IsPrivate,
		RoomBlocked:       r.RoomBlocked,
		RoomTypeID:        r.RoomTypeID,
		RoomTypeName:      r.RoomTypeName,
		RoomTypeNameShort: r.RoomTypeNameShort,
		PhoneNumber:       r.PhoneNumber,
		RoomCondition:     r.RoomCondition,
		RoomOccupied:      r.RoomOccupied,
	}
}

func (p *Cloudbeds) GetRooms() (rooms []hotel.Room, err error) {
	log.Debugf("getting rooms")
	respBody := &ResponseGetRooms{}
	resp, err := p.httpClient.Get("https://hotels.cloudbeds.com/api/v1.1/getRooms")
	if err != nil {
		log.Errorf("request failed with: %s", err)
		return rooms, fmt.Errorf("request failed with: %s", err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but parse return body: %s", err)}
		log.Debugf("success, but parse return body: %s", err)
		return rooms, detailedError
	}

	// check if respBody.Data is set
	if len(respBody.Data) == 0 {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but no rooms found: %s", err)}
		log.Debugf("success, but no rooms found: %s", respBody.Data)
		return rooms, detailedError
	}
	log.Debugf("Response data: %s", respBody.Data)
	log.Debugf("HttpCode: %s", resp.Status)

	if !respBody.Success {
		log.Errorf("Failed to get rooms info: %s\n", respBody.Message)
	}

	log.Debugf("Amount of rooms: %s", len(respBody.Data[0].Rooms))

	for _, room := range respBody.Data[0].Rooms {
		rooms = append(rooms, room.ToHotelRoom())
	}

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

func (p *Cloudbeds) UpdateRoom(roomNumber, housekeepingStatus, housekeeperID string) (msg string, err error) {
	log.Debugf("UpdateRoom %s to %s", roomNumber, housekeepingStatus)

	//get room id
	room := &Room{}
	room.PhoneNumber = roomNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(roomNumber)
	if err != nil {
		log.Error(err)
		return msg, err
	}
	room.RoomID = roomID

	// Update the room condition
	err = p.postHousekeepingStatus(room.RoomID, housekeepingStatus)
	if err != nil {
		log.Error(err)
		return msg, err
	}
	msg = fmt.Sprintf("successfully updated room %s to %s", roomNumber, housekeepingStatus)
	log.Debugf(msg)
	return msg, nil
}

func (p *Cloudbeds) GetRoom(roomNumber string) (hotel.Room, error) {
	log.Infof("get info about room %s", roomNumber)

	//get room id
	room := &Room{}
	room.PhoneNumber = roomNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(roomNumber)
	if err != nil {
		log.Error(err)
		return room.ToHotelRoom(), err
	}
	room.RoomID = roomID

	return room.ToHotelRoom(), nil
}

func (p *Cloudbeds) HandleLogin(w http.ResponseWriter, r *http.Request) {
	setOauth2Config()
	url := oauthConf.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (p *Cloudbeds) login(secretStore secrets.SecretsStore) (statusCodeMsg string, msg string) {

	log.Debugf("Trying to login to Cloudbeds")
	if loginLoopPreventionCounter > 1 {
		log.Debugf("Running login in a loop %d time", loginLoopPreventionCounter)
	}

	// try to retrieve refresh token from secret store
	log.Debugf("Trying to retrieve refresh token from secret store")
	refreshToken, err := secretStore.RetrieveRefreshToken()
	if err != nil {
		log.Fatalf("failed to retrieve refresh token from secret store: %v", err)
	}

	if refreshToken == "" {
		//call oauth2 login
		setOauth2Config()
		msg = fmt.Sprintln("No refresh token found. Please run this link in browser to login to Cloudbeds: ", oauthConf.AuthCodeURL(oauthStateString))
		return fmt.Sprintf("no-refresh-token-found"), msg
	}

	// get new access token via refresh token
	// Make client request using the obtained token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := oauthConf.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Info("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again")
		secretStore.StoreRefreshToken("")
		loginLoopPreventionCounter++
		if loginLoopPreventionCounter <= 2 {
			statusRefresh, msgStatus := p.login(secretStore)
			return statusRefresh, msgStatus
		}
		log.Debugf("Login loop prevention counter: %d", loginLoopPreventionCounter)
		return fmt.Sprintln("failed-to-get-refresh-token"), fmt.Sprintf("failed to get new access token")
	}
	log.Debugf("Issued new access token with len: %v", len(newToken.AccessToken))
	return "ok", ""
}

func (p *Cloudbeds) refreshToken() error {
	log.Debugf("Trying to refresh token")

	//call oauth2 data
	setOauth2Config()

	refreshToken, err := p.storeClient.RetrieveRefreshToken()
	if err != nil {
		log.Fatalf("failed to retrieve refresh token from secret store: %v", err)
	}

	// Make client request using the obtained token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := oauthConf.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Info("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again")
	}

	p.httpClient = oauthConf.Client(context.Background(), token)
	p.storeClient.StoreAccessToken(newToken.AccessToken)

	log.Debugf("Issued new access token with len: %d", len(newToken.AccessToken))
	return nil
}

func (p *Cloudbeds) HandleCallback(w http.ResponseWriter, r *http.Request) {

	log.Debugf("Handling callback")
	state := r.FormValue("state")
	if state != oauthStateString {
		log.Error("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)))
		//http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	log.Debugf("Got auth code: %s", code)
	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		log.Error("oauthConf.Exchange() failed with '%s'\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("oauthConf.Exchange() failed with '%s'\n", err)))
		//http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	log.Debugf("Got access token of length: %d", len(token.AccessToken))

	// get pre-authorized client for future requests
	p.httpClient = oauthConf.Client(context.Background(), token)

	//save access and refresh token to secret store
	log.Debugf("Saving access token to secret store")
	err = p.storeClient.StoreAccessToken(token.AccessToken)
	if err != nil {
		log.Error(err)
	}
	log.Debugf("Saving refresh token to secret store")
	err = p.storeClient.StoreRefreshToken(token.RefreshToken)
	if err != nil {
		log.Error(err)
	}

	log.Infof("Ready for future requests")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Great Success! Ready for future requests. You can close this window now.")))

}

func New(logMain *logrus.Logger) *Cloudbeds {
	log = logMain

	cloudbedsClient := &Cloudbeds{}

	//get access_token

	//current secret store - boltDB
	storeClient, err := boltstore.New()
	if err != nil {
		log.Fatal(err)
	}
	cloudbedsClient.storeClient = storeClient

	//check if access_token is valid. If not - get refresh_token and update access_token
	accessToken, err := cloudbedsClient.storeClient.RetrieveAccessToken()
	if err != nil || accessToken == "" {
		statusCodeMsg, msg := cloudbedsClient.login(cloudbedsClient.storeClient)
		if statusCodeMsg == "no-refresh-token-found" {
			log.Errorln(msg)
			//TODO: send communication message to admin
		}
	}

	token := &oauth2.Token{
		AccessToken: accessToken,
	}
	cloudbedsClient.httpClient = oauthConf.Client(context.Background(), token)

	return cloudbedsClient
}

func (p *Cloudbeds) Close() error {
	err := p.Close()
	if err != nil {
		log.Error(err)
	}
	return nil
}

func (p *Cloudbeds) postHousekeepingStatus(roomID string, roomCondition string) (errorStatusCodeMsg error) {
	apiUrl := "https://hotels.cloudbeds.com/api/v1.1/postHousekeepingStatus"
	log.Infof("Posting housekeeping assignment for room %s with condition: %s", roomID, roomCondition)

	reqBody := UpdateRoomConditionRequest{
		RoomID:        roomID,
		RoomCondition: roomCondition,
	}

	data := url.Values{
		"roomID":        {reqBody.RoomID},
		"roomCondition": {reqBody.RoomCondition},
	}

	log.Debugf("Sending POST data to %s: %v", apiUrl, data)

	// Use the encoded form data to create the request body. client.Post() does not work, so we create a separate request and run client.Do(req)
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.httpClient.Do(req)

	var respBody UpdateRoomConditionResponse
	if err != nil {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("update room status failed with: %s", err)}
		log.Debugf("update room status failed with: %s", err)
		return detailedError
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but parse return body: %s", err)}
		log.Debugf("success, but parse return body: %s", err)
		return detailedError
	}

	//check for errors
	if !respBody.Success { //might be access_token expired. Try to refresh it
		log.Debugf("Failed to update room status: %s Might be access_token expired", respBody.Message)
		err = p.refreshToken()
		if err != nil {
			log.Debugf("Failed to update room status: %s", respBody.Message)
			return err
		}
		err = p.postHousekeepingStatus(roomID, roomCondition)
		if err != nil { //might be access_token expired. Try to refresh it
			log.Debugf("Failed to update room status after token refresh: %s", respBody.Message)
			return err
		} else {
			return nil
		}
	}

	// check if respBody.Data is set
	if respBody.Data.RoomID == "" {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but return body is empty: %s", err)}
		log.Debugf("but return body Data.RoomID is empty: %s", respBody.Data)
		return detailedError
	}

	log.Infof("Room \"%s\" status successfully updated to \"%s\".", respBody.Data.RoomID, respBody.Data.RoomCondition)
	log.Debugf("HttpCode: %s. Response data: %v", resp.Status, respBody.Data)

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

func setOauth2Config() {
	log.Debugf("Setting oauth2 config")
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
