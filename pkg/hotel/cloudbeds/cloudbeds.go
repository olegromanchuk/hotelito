package cloudbeds

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/secrets"
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

type RoomFromConfig struct {
	RoomExtension       string `json:"room_extension"`
	HospitalityRoomID   string `json:"hospitality_room_id"`
	HospitalityRoomName string `json:"hospitality_room_name"`
}

var (
	loginLoopPreventionCounter = 1
)

// HTTPClient is needed for mocking http requests in tests. This is the only reason to create this interface. Original http.Client implements this interface
type HTTPClient interface {
	Get(url string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}

// introduced to make Login() more testable. Now we can mock oauthConf
type OauthConfInterface interface {
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	TokenSource(ctx context.Context, t *oauth2.Token) oauth2.TokenSource
	Client(ctx context.Context, t *oauth2.Token) *http.Client
}

// Cloudbeds is used to make requests to Cloudbeds API. httpClient contains pre-authorized http.Client that is set by the package oauth2 during authorization process. In tests we just mock this client to imitate cloudbeds API responses.
type Cloudbeds struct {
	httpClient                   HTTPClient
	storeClient                  secrets.SecretsStore
	log                          *logrus.Logger
	refresher                    TokenRefresher
	oauthConf                    OauthConfInterface
	configMap                    *configuration.ConfigMap
	apiUrlPostHousekeepingStatus string
	apiUrlGetRooms               string
	roomStatuses                 []string
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

type ApiConfiguration3CX struct {
	APIURLs struct {
		GetRooms               string `json:"getRooms"`
		PostHousekeepingStatus string `json:"postHousekeepingStatus"`
	} `json:"apiURLs"`
	RoomStatuses []string `json:"roomStatuses"`
}

func (p *Cloudbeds) GetRooms() (rooms []hotel.Room, err error) {
	p.log.Debugf("getting rooms")
	apiUrl := p.apiUrlGetRooms
	//TODO - move urlConfiguration to configMap and load from separate cloudbeds_api_url.txt config file
	if apiUrl == "" {
		apiUrl = "https://hotels.cloudbeds.com/api/v1.2/getRooms" // default value
	}

	respBody := &ResponseGetRooms{}
	resp, err := p.httpClient.Get(apiUrl)
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
		rooms, err = p.GetRooms() //recursive call. Not testable
		if err != nil {           //might be access_token expired. Try to refresh it
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

func (p *Cloudbeds) UpdateRoom(roomExtensionNumber, housekeepingStatus, housekeeperName string) (msg string, err error) {
	p.log.Debugf("Start UpdateRoom %s to %s for %s", roomExtensionNumber, housekeepingStatus, housekeeperName)

	//get room id
	room := &Room{}
	room.PhoneNumber = roomExtensionNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(p.log, roomExtensionNumber, p.configMap.ExtensionMap)
	if err != nil {
		p.log.Error(err)
		return msg, err
	}
	room.RoomID = roomID

	if !p.checkIfRoomConditionValid(housekeepingStatus) {
		errMsg := fmt.Sprintf("room condition %s is not valid", housekeepingStatus)
		p.log.Error(errMsg)
		return "", errors.New(errMsg)
	}

	// Update the room condition
	err = p.postHousekeepingStatus(room.RoomID, housekeepingStatus)
	if err != nil {
		p.log.Error(err)
		return msg, err
	}
	msg = fmt.Sprintf("Finish UpdateRoom successfully updated room %s to %s", roomExtensionNumber, housekeepingStatus)
	p.log.Debugf(msg)
	return msg, nil
}

func (p *Cloudbeds) GetRoom(roomNumber string, mapFileName string) (hotel.Room, error) {
	p.log.Infof("get info about room %s", roomNumber)

	//get room id
	room := &Room{}
	room.PhoneNumber = roomNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(p.log, roomNumber, p.configMap.ExtensionMap)
	if err != nil {
		p.log.Error(err)
		return room.ToHotelRoom(), err
	}
	room.RoomID = roomID

	return room.ToHotelRoom(), nil
}

// handleLogin helper function to handle login. Just redirect to oauth2 provider login page
func (p *Cloudbeds) HandleInitialLogin() (url string, errReturn error) {
	err := p.setOauth2Config()
	if err != nil {
		return "", err
	}
	oauthStateString := p.generateRandomString(10)
	//oauth2 Config.AuthCodeURL() returns url that could be used for redirecting to oauth2 provider login page
	url = p.oauthConf.AuthCodeURL(oauthStateString)
	p.log.Debugf("got url for redirect %s", url)
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

// login helper function to handle login. Just redirect to oauth2 provider login page
func (p *Cloudbeds) login(secretStore secrets.SecretsStore) (statusCodeMsg string, msg string, errorInfo error) {
	p.log.Debugf("Trying to login to Cloudbeds")
	if loginLoopPreventionCounter > 1 {
		p.log.Debugf("Running login in a loop %d time", loginLoopPreventionCounter)
	}
	oauthStateString := p.generateRandomString(10) // adjust the length as per your needs

	// try to retrieve refresh token from secret store
	p.log.Debugf("Trying to retrieve refresh token from secret store")
	refreshToken, err := secretStore.RetrieveRefreshToken()
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve refresh token from secret store: %v", err)
		p.log.Error(errMsg)
		return "fatal-error", errMsg, errors.New(errMsg)
	}

	if refreshToken == "" {
		//call oauth2 login
		err = p.setOauth2Config()
		if err != nil {
			errMsg := fmt.Sprintf("failed to set oauth2 config: %v", err)
			p.log.Error(errMsg)
			return "", "errMsg", nil
		}
		msg = fmt.Sprintln("No refresh token found. Please run this link in browser to login to Cloudbeds: ", p.oauthConf.AuthCodeURL(oauthStateString))
		//save "state" for future validation by the callback function
		err = p.storeClient.StoreOauthState(oauthStateString)
		if err != nil {
			errMsg := fmt.Sprintf("failed to store oauth state: %v", err)
			p.log.Errorf(errMsg)
			return "", "", nil
		}
		return "no-refresh-token-found", msg, nil
	}

	// get new access token via refresh token
	// Make client request using the obtained token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := p.oauthConf.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		p.log.Info("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again")
		err := secretStore.StoreRefreshToken("")
		if err != nil {
			p.log.Fatalf("failed to clear refresh token from secret store: %v", err)
			return "", "", nil
		}
		loginLoopPreventionCounter++
		if loginLoopPreventionCounter <= 2 {
			statusRefresh, msgStatus, _ := p.login(secretStore)
			return statusRefresh, msgStatus, nil
		}
		p.log.Debugf("Login loop prevention counter: %d", loginLoopPreventionCounter)
		return fmt.Sprintln("failed-to-get-refresh-token"), "failed to get new access token", nil
	}
	p.log.Debugf("Issued new access token with len: %v", len(newToken.AccessToken))
	return "ok", "", nil
}

// refreshToken helper function to refresh token and store new access token to secret store
func (p *Cloudbeds) refreshToken() error {
	p.log.Debugf("Trying to refresh token")

	//call oauth2 data
	err := p.setOauth2Config()
	if err != nil {
		return err
	}

	refreshToken, err := p.storeClient.RetrieveRefreshToken()
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve refresh token from secret store: %v", err)
		p.log.Error(errMsg)
		return errors.New(errMsg)
	}

	// Make client request using the obtained token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := p.oauthConf.TokenSource(context.Background(), token)
	newToken, err := tokenSource.Token()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again. Error: %v", err.Error())
		p.log.Error(errMsg)
		return errors.New(errMsg)
	}

	if newToken == nil {
		errMsg := "failed to get new access token. Looks like refresh token is stale. Clearing it and try to login again"
		p.log.Error(errMsg)
		return errors.New(errMsg)
	}

	p.httpClient = p.oauthConf.Client(context.Background(), token)
	err = p.storeClient.StoreAccessToken(newToken.AccessToken)
	if err != nil {
		return err
	}

	p.log.Debugf("Issued new access token with len: %d", len(newToken.AccessToken))
	return nil
}

func (p *Cloudbeds) HandleOAuthCallback(state, code string) (err error) {
	p.log.Debugf("Handling oauth callback in cloudbeds. State: %s, Code: %s", state, code)
	oauthStateString, err := p.storeClient.RetrieveOauthState(state)
	if err != nil {
		errMsg := fmt.Sprintf("failed to retrieve oauth state from secret store: %v. Possibly state does not exist or stale. Try to login again", err.Error())
		p.log.Debug(errMsg)
		return errors.New(errMsg)
	}
	if state != oauthStateString {
		errMsg := fmt.Sprintf("invalid oauth state, expected '%s', got '%s'", oauthStateString, state)
		p.log.Error(errMsg)
		return errors.New(errMsg)
	}

	token, err := p.oauthConf.Exchange(context.Background(), code)
	if err != nil {
		p.log.Debugf("oauthConf.Exchange() failed with '%s'", err)
		return fmt.Errorf("oauthConf.Exchange() failed with '%s'\n", err)
	}
	p.log.Debugf("Got access token of length: %d", len(token.AccessToken))

	// get pre-authorized client for future requests (doesn't make a lot of sense for aws version)
	p.httpClient = p.oauthConf.Client(context.Background(), token)

	//save access and refresh token to secret store
	p.log.Debugf("Saving access token to secret store")
	err = p.storeClient.StoreAccessToken(token.AccessToken)
	if err != nil {
		p.log.Error(err)
		return err
	}
	p.log.Debugf("Saving refresh token to secret store")
	err = p.storeClient.StoreRefreshToken(token.RefreshToken)
	if err != nil {
		p.log.Error(err)
		return err
	}
	return nil
}

func New(log *logrus.Logger, secretStore secrets.SecretsStore, configMapInfo *configuration.ConfigMap) (*Cloudbeds, error) {
	log.Debugf("Creating new Cloudbeds client")

	apiConfigurationFileName := configMapInfo.ApiCfgFileName
	apiConfiguration, err := loadApiConfiguration(log, apiConfigurationFileName)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	cloudbedsClient := &Cloudbeds{
		log:         log,
		oauthConf:   &oauth2.Config{},
		storeClient: secretStore,
		configMap:   configMapInfo,
	}

	//get current api parameters for cloudbeds from config file
	cloudbedsClient.apiUrlPostHousekeepingStatus = apiConfiguration.APIURLs.PostHousekeepingStatus
	cloudbedsClient.apiUrlGetRooms = apiConfiguration.APIURLs.GetRooms
	cloudbedsClient.roomStatuses = apiConfiguration.RoomStatuses

	err = cloudbedsClient.setOauth2Config()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	cloudbedsClient.refresher = cloudbedsClient //refresher is an interface! This feint with ears is needed to point refreshToken method to itself. Now call p.refresher.refreshToken() will call refreshToken method of Cloudbeds struct
	//refresher was created as interface to make the code more testable

	//get access_token
	//check if access_token is valid. If not - get refresh_token and update access_token
	accessToken, err := cloudbedsClient.storeClient.RetrieveAccessToken()
	if err != nil || accessToken == "" {
		statusCodeMsg, msg, err := cloudbedsClient.login(cloudbedsClient.storeClient)
		if err != nil {
			log.Errorln(msg)
			return nil, err
		}
		if statusCodeMsg == "no-refresh-token-found" {
			log.Errorln(msg)
			//TODO: send communication message to admin
		}
	}

	token := &oauth2.Token{
		AccessToken: accessToken,
	}
	cloudbedsClient.httpClient = cloudbedsClient.oauthConf.Client(context.Background(), token)

	return cloudbedsClient, nil
}

func loadApiConfiguration(log *logrus.Logger, apiConfigurationFileName string) (apiConfiguration *ApiConfiguration3CX, err error) {
	apiConfiguration = &ApiConfiguration3CX{}
	file, err := os.Open(apiConfigurationFileName)
	if err != nil {
		errMsg := fmt.Sprintf("error opening config file: %s", err.Error())
		log.Errorf(errMsg)
		return apiConfiguration, errors.New(errMsg)
	}
	defer file.Close()
	byteValue, _ := io.ReadAll(file)
	err = json.Unmarshal(byteValue, apiConfiguration)
	if err != nil {
		errMsg := fmt.Errorf("error unmarshalling config file %s: %s", apiConfigurationFileName, err.Error())
		log.Errorf(errMsg.Error())
		return apiConfiguration, errMsg
	}
	return apiConfiguration, nil
}

func NewClient4CallbackAndInit(log *logrus.Logger, secretStore secrets.SecretsStore) (*Cloudbeds, error) {
	log.Debugf("Creating new Cloudbeds client")
	cloudbedsClient := &Cloudbeds{
		log:         log,
		storeClient: secretStore,
	}
	cloudbedsClient.refresher = cloudbedsClient //refresher is an interface! This feint with ears is needed to point refreshToken method to itself. Now call p.refresher.refreshToken() will call refreshToken method of Cloudbeds struct
	//refresher was created as interface to make the code more testable

	err := cloudbedsClient.setOauth2Config()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return cloudbedsClient, nil
}

func (p *Cloudbeds) Close() error {
	err := p.storeClient.Close()
	if err != nil {
		errMsg := fmt.Sprintf("failed to close secret store: %s", err.Error())
		p.log.Error(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

func (p *Cloudbeds) postHousekeepingStatus(roomID string, roomCondition string) (errorStatusCodeMsg error) {
	apiUrl := p.apiUrlPostHousekeepingStatus
	//TODO - move urlConfiguration to configMap and load from separate cloudbeds_api_url.txt config file
	if apiUrl == "" {
		apiUrl = "https://hotels.cloudbeds.com/api/v1.1/postHousekeepingStatus" // default value
	}

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

func (p *Cloudbeds) checkIfRoomConditionValid(roomCondition string) bool {
	p.log.Debugf("Checking if room condition %s is valid", roomCondition)
	for _, status := range p.roomStatuses {
		if status == roomCondition {
			p.log.Debugf("Room condition %s is valid", roomCondition)
			return true
		}
	}
	p.log.Debugf("Room condition %s is not valid", roomCondition)
	return false
}

func (r *Room) SearchRoomIDByPhoneNumber(log *logrus.Logger, phoneNumber string, extensionsInfo []configuration.Extension) (string, error) {

	// create map for easy search
	roomMap := make(map[string]configuration.Extension)
	for _, extension := range extensionsInfo {
		roomMap[extension.RoomExtension] = extension
	}

	// Look up room ID by phone number
	log.Tracef("Looking up room ID by phone number %s", phoneNumber)
	room, ok := roomMap[phoneNumber]
	if !ok {
		errMsg := fmt.Sprintf("phone number %s not found", phoneNumber)
		log.Error(errMsg)
		return "", errors.New(errMsg)
	}
	log.Tracef("Found room name: %s, ID: %s for phone number: %s", room.HospitalityRoomName, room.HospitalityRoomID, phoneNumber)

	return room.HospitalityRoomID, nil
}

// setOauth2Config sets oauth2 config from env variables
func (p *Cloudbeds) setOauth2Config() error {
	p.log.Debugf("Setting oauth2 config")

	//first - trying to get vars from secretStore. If no luck - try from env
	//if no luck again - return error

	scopes := strings.Split(p.getVarFromStoreOrEnvironment("CLOUDBEDS_SCOPES"), ",")

	oauthConfig := &oauth2.Config{
		ClientID:     p.getVarFromStoreOrEnvironment("CLOUDBEDS_CLIENT_ID"),
		ClientSecret: p.getVarFromStoreOrEnvironment("CLOUDBEDS_CLIENT_SECRET"),
		RedirectURL:  p.getVarFromStoreOrEnvironment("CLOUDBEDS_REDIRECT_URL"),
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.getVarFromStoreOrEnvironment("CLOUDBEDS_AUTH_URL"),
			TokenURL: p.getVarFromStoreOrEnvironment("CLOUDBEDS_TOKEN_URL"),
		},
	}
	//check that all env variables are set
	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" || oauthConfig.RedirectURL == "" || oauthConfig.Scopes == nil || oauthConfig.Endpoint.AuthURL == "" || oauthConfig.Endpoint.TokenURL == "" {
		errMsg := fmt.Errorf("Not all required env variables are set. Missed one of: CLOUDBEDS_CLIENT_ID, CLOUDBEDS_CLIENT_SECRET, CLOUDBEDS_REDIRECT_URL, CLOUDBEDS_SCOPES, CLOUDBEDS_AUTH_URL, CLOUDBEDS_TOKEN_URL")
		p.log.Error(errMsg.Error())
		return errMsg
	}

	p.oauthConf = oauthConfig
	return nil
}

// getVarFromStoreOrEnvironment returns variable from secret store or environment if store is empty
func (p *Cloudbeds) getVarFromStoreOrEnvironment(varName string) (result string) {
	p.log.Tracef("Getting variable '%s' from store or environment", varName)
	result, err := p.storeClient.RetrieveVar(varName)
	if err != nil || result == "" {
		result = os.Getenv(varName)
		if err != nil {
			p.log.Errorf("Got error while trying to get variable '%s' from environment: %s", varName, err)
		}
		//obfuscate secret CLOUDBEDS_CLIENT_SECRET
		if varName == "CLOUDBEDS_CLIENT_SECRET" {
			//print last 4 symbols of CLOUDBEDS_CLIENT_SECRET
			if len(result) > 4 {
				result = "************" + result[len(result)-4:]
			} else {
				result = "****"
			}
		}
		p.log.Debugf("The store is empty. Got variable '%s' from environment. Result: '%s'", varName, result)
		return
	}
	p.log.Debugf("Got variable '%s' from store. Result: '%s'", varName, result)
	return
}

func (p *Cloudbeds) generateRandomString(length int) string {
	bytes := make([]byte, length)
	p.log.Debugf("Generating random string of length %d", length)

	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		p.log.Errorf(err.Error())
		bytes = []byte("default")
	}

	randomStr := hex.EncodeToString(bytes)
	p.log.Debugf("Generated random string: %s", randomStr)
	return randomStr
}
