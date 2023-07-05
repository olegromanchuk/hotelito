package cloudbeds

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/secrets"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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

var (
	oauthConf                  *oauth2.Config
	loginLoopPreventionCounter = 1
)

// HTTPClient is needed for mocking http requests in tests. This is the only reason to create this interface. Original http.Client implements this interface
type HTTPClient interface {
	Get(url string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}

// Cloudbeds is used to make requests to Cloudbeds API. httpClient contains pre-authorized http.Client that is set by the package oauth2 during authorization process. In tests we just mock this client to imitate cloudbeds API responses.
type Cloudbeds struct {
	httpClient  HTTPClient
	storeClient secrets.SecretsStore
	log         *logrus.Logger
	refresher   TokenRefresher
}

// TokenRefresher is needed for mocking http requests in tests. This is the only reason to create this interface. Cloudbeds implements this interface
type TokenRefresher interface {
	refreshToken() error
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
	p.log.Debugf("getting rooms")
	respBody := &ResponseGetRooms{}
	resp, err := p.httpClient.Get("https://hotels.cloudbeds.com/api/v1.1/getRooms")
	if err != nil {
		p.log.Errorf("request failed with: %s", err)
		return rooms, fmt.Errorf("request failed with: %s", err)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but parse return body: %s", err)}
		p.log.Debugf("success, but parse return body: %s", err)
		return rooms, detailedError
	}

	//check for errors
	if !respBody.Success { //might be access_token expired. Try to refresh it
		p.log.Debugf("Failed to update room status: %s Might be access_token expired", respBody.Message)
		err = p.refresher.refreshToken()
		if err != nil {
			p.log.Debugf("Failed to update room status: %s", respBody.Message)
			return rooms, err
		}
		rooms, err = p.GetRooms()
		if err != nil { //might be access_token expired. Try to refresh it
			p.log.Debugf("Failed to update room status after token refresh: %s", respBody.Message)
			return rooms, err
		} else {
			return rooms, nil
		}
	}

	// check if respBody.Data is set
	if len(respBody.Data) == 0 {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but no rooms found: %s", err)}
		p.log.Debugf("success, but no rooms found: %v", respBody.Data)
		return rooms, detailedError
	}
	p.log.Debugf("Response data: %v", respBody.Data)
	p.log.Debugf("HttpCode: %s", resp.Status)

	p.log.Debugf("Amount of rooms: %d", len(respBody.Data[0].Rooms))

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
	p.log.Debugf("Start UpdateRoom %s to %s", roomNumber, housekeepingStatus)

	//get room id
	room := &Room{}
	room.PhoneNumber = roomNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(roomNumber, os.Getenv("CLOUDBEDS_PHONE2ROOM_MAP_FILENAME"))
	if err != nil {
		p.log.Error(err)
		return msg, err
	}
	room.RoomID = roomID

	// Update the room condition
	err = p.postHousekeepingStatus(room.RoomID, housekeepingStatus)
	if err != nil {
		p.log.Error(err)
		return msg, err
	}
	msg = fmt.Sprintf("Finish UpdateRoom successfully updated room %s to %s", roomNumber, housekeepingStatus)
	p.log.Debugf(msg)
	return msg, nil
}

func (p *Cloudbeds) GetRoom(roomNumber string) (hotel.Room, error) {
	p.log.Infof("get info about room %s", roomNumber)

	//get room id
	room := &Room{}
	room.PhoneNumber = roomNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(roomNumber, os.Getenv("CLOUDBEDS_PHONE2ROOM_MAP_FILENAME"))
	if err != nil {
		p.log.Error(err)
		return room.ToHotelRoom(), err
	}
	room.RoomID = roomID

	return room.ToHotelRoom(), nil
}

// handleLogin helper function to handle login. Just redirect to oauth2 provider login page
func (p *Cloudbeds) HandleManualLogin() (url string, err error) {
	p.setOauth2Config()
	oauthStateString := p.generateRandomString(10)
	url = oauthConf.AuthCodeURL(oauthStateString)
	if url == "" {
		return url, fmt.Errorf("failed to retrieve oauth2 url. Check .env file and make sure that all variables related to CLOUDBEDS are set. Refer to .env_example")
	}
	//save "state" for future validation by the callback function
	err = p.storeClient.StoreOauthState(oauthStateString)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (p *Cloudbeds) login(secretStore secrets.SecretsStore) (statusCodeMsg string, msg string) {
	p.log.Debugf("Trying to login to Cloudbeds")
	if loginLoopPreventionCounter > 1 {
		p.log.Debugf("Running login in a loop %d time", loginLoopPreventionCounter)
	}
	oauthStateString := p.generateRandomString(10) // adjust the length as per your needs

	// try to retrieve refresh token from secret store
	p.log.Debugf("Trying to retrieve refresh token from secret store")
	refreshToken, err := secretStore.RetrieveRefreshToken()
	if err != nil {
		p.log.Fatalf("failed to retrieve refresh token from secret store: %v", err)
	}

	if refreshToken == "" {
		//call oauth2 login
		p.setOauth2Config()
		msg = fmt.Sprintln("No refresh token found. Please run this link in browser to login to Cloudbeds: ", oauthConf.AuthCodeURL(oauthStateString))
		//save "state" for future validation by the callback function
		err = p.storeClient.StoreOauthState(oauthStateString)
		if err != nil {
			errMsg := fmt.Sprintf("failed to store oauth state: %v", err)
			p.log.Errorf(errMsg)
			return "", ""
		}
		return "no-refresh-token-found", msg
	}

	// get new access token via refresh token
	// Make client request using the obtained token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := oauthConf.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		p.log.Info("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again")
		err := secretStore.StoreRefreshToken("")
		if err != nil {
			p.log.Fatalf("failed to clear refresh token from secret store: %v", err)
			return "", ""
		}
		loginLoopPreventionCounter++
		if loginLoopPreventionCounter <= 2 {
			statusRefresh, msgStatus := p.login(secretStore)
			return statusRefresh, msgStatus
		}
		p.log.Debugf("Login loop prevention counter: %d", loginLoopPreventionCounter)
		return fmt.Sprintln("failed-to-get-refresh-token"), "failed to get new access token"
	}
	p.log.Debugf("Issued new access token with len: %v", len(newToken.AccessToken))
	return "ok", ""
}

func (p *Cloudbeds) refreshToken() error {
	p.log.Debugf("Trying to refresh token")

	//call oauth2 data
	p.setOauth2Config()

	refreshToken, err := p.storeClient.RetrieveRefreshToken()
	if err != nil {
		p.log.Fatalf("failed to retrieve refresh token from secret store: %v", err)
	}

	// Make client request using the obtained token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := oauthConf.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		p.log.Info("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again")
	}

	p.httpClient = oauthConf.Client(context.Background(), token)
	err = p.storeClient.StoreAccessToken(newToken.AccessToken)
	if err != nil {
		return err
	}

	p.log.Debugf("Issued new access token with len: %d", len(newToken.AccessToken))
	return nil
}

func (p *Cloudbeds) HandleCallback(state, code string) (err error) {
	p.log.Debugf("Handling callback in cloudbeds")
	oauthStateString, err := p.storeClient.RetrieveOauthState(state)
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve oauth state from secret store: %v. Possibly state does not exist or stale. Try to login again", err.Error())
		p.log.Debug(errMsg)
		return errors.New(errMsg)
	}
	if state != oauthStateString {
		p.log.Errorf("invalid oauth state, expected '%s', got '%s'", oauthStateString, state)
		return fmt.Errorf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
	}

	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		p.log.Debugf("oauthConf.Exchange() failed with '%s'", err)
		return fmt.Errorf("oauthConf.Exchange() failed with '%s'\n", err)
	}
	p.log.Debugf("Got access token of length: %d", len(token.AccessToken))

	// get pre-authorized client for future requests
	p.httpClient = oauthConf.Client(context.Background(), token)

	//save access and refresh token to secret store
	p.log.Debugf("Saving access token to secret store")
	err = p.storeClient.StoreAccessToken(token.AccessToken)
	if err != nil {
		p.log.Error(err)
	}
	p.log.Debugf("Saving refresh token to secret store")
	err = p.storeClient.StoreRefreshToken(token.RefreshToken)
	if err != nil {
		p.log.Error(err)
	}
	return nil
}

func New(log *logrus.Logger, secretStore secrets.SecretsStore) *Cloudbeds {
	log.Debugf("Creating new Cloudbeds client")
	cloudbedsClient := &Cloudbeds{
		log: log,
	}
	cloudbedsClient.refresher = cloudbedsClient //refresher is an interface! This feint with ears is needed to point refreshToken method to itself. Now call p.refresher.refreshToken() will call refreshToken method of Cloudbeds struct
	//refresher was created as interface to make the code more testable

	//get access_token

	cloudbedsClient.storeClient = secretStore

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
	err := p.storeClient.Close()
	if err != nil {
		p.log.Error(err)
	}
	return nil
}

func (p *Cloudbeds) postHousekeepingStatus(roomID string, roomCondition string) (errorStatusCodeMsg error) {
	apiUrl := "https://hotels.cloudbeds.com/api/v1.1/postHousekeepingStatus"
	p.log.Infof("Posting housekeeping assignment for room %s with condition: %s", roomID, roomCondition)

	reqBody := UpdateRoomConditionRequest{
		RoomID:        roomID,
		RoomCondition: roomCondition,
	}

	data := url.Values{
		"roomID":        {reqBody.RoomID},
		"roomCondition": {reqBody.RoomCondition},
	}

	p.log.Debugf("Sending POST data to %s: %v", apiUrl, data)

	// Use the encoded form data to create the request body. client.Post() does not work, so we create a separate request and run client.Do(req)
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		p.log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("update room status failed with: %s", err)}
		p.log.Debugf("update room status failed with: %s", err)
		return detailedError
	}
	defer resp.Body.Close()

	var respBody UpdateRoomConditionResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but parse return body: %s", err)}
		p.log.Debugf("success, but parse return body: %s", err)
		return detailedError
	}

	//check for errors
	if !respBody.Success { //might be access_token expired. Try to refresh it
		p.log.Debugf("Failed to update room status: %s Might be access_token expired", respBody.Message)
		err = p.refresher.refreshToken()
		if err != nil {
			errMsg := fmt.Sprintf("Failed to update room status: %s", err.Error())
			p.log.Debugf(errMsg)
			return errors.New(errMsg)
		}
		err = p.postHousekeepingStatus(roomID, roomCondition)
		if err != nil { //might be access_token expired. Try to refresh it
			errMsg := fmt.Sprintf("Failed to update room status after token refresh: %s", err.Error())
			p.log.Debugf(errMsg)
			return errors.New(errMsg)
		} else {
			return nil
		}
	}

	// check if respBody.Data is set
	if respBody.Data.RoomID == "" {
		detailedError := &hotel.DetailedError{Msg: err, Details: fmt.Sprintf("success, but return body is empty: %s", err)}
		p.log.Debugf("but return body Data.RoomID is empty: %v", respBody.Data)
		return detailedError
	}

	p.log.Infof("Room '%s' status successfully updated to '%s'.", respBody.Data.RoomID, respBody.Data.RoomCondition)
	p.log.Debugf("HttpCode: %s. Response data: %v", resp.Status, respBody.Data)

	return nil
}

func (r *Room) SearchRoomIDByPhoneNumber(phoneNumber string, mapFileName string) (string, error) {
	type RoomMap map[string]string
	// Read file
	jsonFile, err := os.Open(mapFileName)
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	// Initialize a map to store the JSON data in
	var roomMap RoomMap

	// Unmarshal the JSON data into the map
	err = json.Unmarshal(byteValue, &roomMap)
	if err != nil {
		return "", err
	}

	// Look up room ID by phone number
	roomID, ok := roomMap[phoneNumber]
	if !ok {
		return "", fmt.Errorf("phone number not found")
	}

	return roomID, nil
}

// setOauth2Config sets oauth2 config from env variables
func (p *Cloudbeds) setOauth2Config() {
	p.log.Debugf("Setting oauth2 config")
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

func (p *Cloudbeds) generateRandomString(length int) string {
	bytes := make([]byte, length)
	p.log.Debugf("Generating random string of length %d", length)
	// Seed the random number generator with the current time
	rand.Seed(time.Now().UnixNano())

	if _, err := rand.Read(bytes); err != nil {
		p.log.Fatal(err)
	}
	p.log.Debugf("Generated random string: %s", hex.EncodeToString(bytes))
	return hex.EncodeToString(bytes)
}
