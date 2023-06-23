package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const ()

var (
	//oauthConf = &oauth2.Config{
	//	ClientID:     "improcom_LuPCZYx8TKrtyq72035DMXjS",
	//	ClientSecret: "PoUAGK5bYMSvBCE1y7Zm9eODfwi6zkXH",
	//	RedirectURL:  "https://bae1-72-89-122-10.ngrok-free.app/callback",
	//	Scopes:       []string{"read:hotel", "read:reservation", "write:reservation", "read:room", "write:room", "read:housekeeping", "write:housekeeping"},
	//	Endpoint: oauth2.Endpoint{
	//		AuthURL:  "https://hotels.cloudbeds.com/api/v1.1/oauth",
	//		TokenURL: "https://hotels.cloudbeds.com/api/v1.1/access_token",
	//	},
	//}

	log = logrus.New()
)

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

func main() {
	// The default level is info.
	log.SetLevel(logrus.DebugLevel)

	// Set output of logs to Stdout
	log.SetOutput(os.Stdout)

	readAuthVarsFromFile()

	r := mux.NewRouter()

	//   ---------------------- Cloudbed parts ----------------------
	//create cloudbeds client
	clbClient := cloudbeds.New(log)

	//auth urls
	r.HandleFunc("/", handleMain).Methods("GET")
	r.HandleFunc("/login", clbClient.HandleLogin).Methods("GET")
	r.HandleFunc("/callback", clbClient.HandleCallback).Methods("GET")

	//test/troubleshooting urls
	r.HandleFunc("/housekeepings/{roomPhoneNumber}/{housekeepingStatus}/{housekeeperID}", handleHousekeepingAssignment).Methods("GET")

	//3cx call info receiver
	r.HandleFunc("/3cx/callback/{callinfo3cx}", handle3cxCallInfo).Methods("POST")

	http.Handle("/", r)

	//http.HandleFunc("/", handleMain)
	//http.HandleFunc("/login", handleLogin)
	//http.HandleFunc("/callback", handleCallback)
	//http.HandleFunc("/rooms", handleUpdateRoom)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handle3cxCallInfo(w http.ResponseWriter, r *http.Request) {
	data3cx := mux.Vars(r)
	log.Debugf("received 3cx Call Info: ", data3cx)
}

func handleHousekeepingAssignment(w http.ResponseWriter, r *http.Request) {
	log.Info("handleHousekeepingAssignment")

	//get client
	client = getClient

	// Get the housekeeping info from the URL
	vars := mux.Vars(r)
	roomPhoneNumber := vars["roomPhoneNumber"]
	housekeepingStatus := vars["housekeepingStatus"]
	housekeeperID := vars["housekeeperID"]

	room := &Room{}
	room.PhoneNumber = roomPhoneNumber
	roomID, err := room.SearchRoomIDByPhoneNumber(roomPhoneNumber)
	if err != nil {
		log.Error(err)
		return
	}
	room.RoomID = roomID

	// Update the room condition
	postHousekeepingAssignment(room.RoomID, housekeepingStatus)

	// Redirect to the main page
	http.Redirect(w, r, "/", http.StatusSeeOther)

}

func handleMain(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<a href="/login">Login with OAuth2 Provider</a>`)
}

func callCloudbedsAPI(method, URL, data string) (int, string, error) {
	// Prepare the request
	var req *http.Request
	var err error

	// Prepare the data for the POST and DELETE requests
	switch method {
	case "POST", "DELETE":
		formData := url.Values{}
		pairs := strings.Split(data, "&")
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			if len(kv) != 2 {
				return 0, "", errors.New("invalid data format")
			}
			formData.Set(kv[0], kv[1])
		}

		req, err = http.NewRequest(method, URL, strings.NewReader(formData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	case "GET":
		req, err = http.NewRequest(method, URL, nil)
	default:
		return 0, "", fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return 0, "", err
	}

	// Add the authorization header
	token := os.Getenv("ACCESS_TOKEN")
	req.Header.Add("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}

	// Return the status code, response body, and nil error
	return resp.StatusCode, string(bodyBytes), nil
}

func postHousekeepingAssignment(client *http.Client, roomID string, roomCondition string) error {
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

	log.Debugf("Sending POST data to %s: %s", apiUrl, data)

	// Use the encoded form data to create the request body. client.Post() does not work, so we create separate request abd run client.Do(req)
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	var respBody UpdateRoomConditionResponse
	if err != nil {
		log.Fatalf("Request failed with: %s", err)
	} else {
		defer resp.Body.Close()
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			fmt.Println("Error:", err)
			return err
		}
		if err != nil {
			log.Fatalf("Failed to read response: %s", err)
		}
		if respBody.Message != "" {
			log.Debugf("Response message: %s", respBody.Message)
		}
		// check if respBody.Data is set
		if respBody.Data.RoomID != "" {
			log.Debugf("Response data: %s", respBody.Data)
		}
		log.Debugf("HttpCode: %s", resp.Status)
	}

	if respBody.Success {
		fmt.Println("Room status successfully updated.")
	} else {
		fmt.Printf("Failed to update room status: %s\n", respBody.Message)
	}

	return nil
}

func getRooms(client *http.Client) (allRooms ResponseGetRooms, err error) {
	resp, err := client.Get("https://hotels.cloudbeds.com/api/v1.1/getRooms")
	if err != nil {
		log.Fatalf("Request failed with: %s", err)
	} else {
		defer resp.Body.Close()

		err = json.NewDecoder(resp.Body).Decode(&allRooms)
		if err != nil {
			log.Fatalf("Failed to read response: %s", err)
		}
		log.Debugf("Response: %s", allRooms)
		return allRooms, nil
	}
	return allRooms, nil
}

func readAuthVarsFromFile() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}
