package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/olegromanchuk/hotelito/pkg/hotel"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	log          = logrus.New()
	clbClient    *cloudbeds.Cloudbeds
	pbx3cxClient *pbx3cx.PBX3CX
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

type callerHook struct{}

func (hook *callerHook) Levels() []logrus.Level {
	// Set levels on which the hook to be fired.
	return logrus.AllLevels
}

func (hook *callerHook) Fire(entry *logrus.Entry) error {
	// You can modify any field of the entry here,
	// or use the entry to send logs to other places.
	entry.Data["caller"] = entry.Caller
	return nil
}

func main() {
	// The default level is info.
	log.SetLevel(logrus.DebugLevel)
	//log.SetReportCaller(true)
	//log.AddHook(&callerHook{})

	// Set output of logs to Stdout
	log.SetOutput(os.Stdout)

	readAuthVarsFromFile()

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(loggingMiddleware)

	//   ---------------------- Cloudbed parts ----------------------
	//create cloudbeds client
	clbClient = cloudbeds.New(log)
	defer clbClient.Close()

	//create 3cx client
	pbx3cxClient = pbx3cx.New(log)
	defer clbClient.Close()

	//auth urls
	api.HandleFunc("/", handleMain).Methods("GET")
	api.HandleFunc("/login", clbClient.HandleLogin).Methods("GET")
	api.HandleFunc("/callback", clbClient.HandleCallback).Methods("GET")

	//test/troubleshooting urls
	//update housekeeping status
	// test data: "544559-0", "clean"
	api.HandleFunc("/housekeepings/{roomPhoneNumber}/{housekeepingStatus}/{housekeeperID}", handleSetHousekeepingStatus).Methods("POST")

	//3cx call info receiver
	api.HandleFunc("/3cx/outbound_call", handle3cxCallInfo).Methods("POST")

	http.Handle("/", api)

	//http.HandleFunc("/", handleMain)
	//http.HandleFunc("/login", handleLogin)
	//http.HandleFunc("/callback", handleCallback)
	//http.HandleFunc("/rooms", handleUpdateRoom)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handle3cxCallInfo(w http.ResponseWriter, r *http.Request) {
	var pbxClient pbx.PBXProvider
	pbxClient = pbx3cxClient

	decoder := json.NewDecoder(r.Body)
	log.Debugf("Received 3cx call info")
	room, err := pbxClient.ProcessPBXRequest(decoder)
	if err != nil {
		log.Error(err)
		return
	}
	if room.PhoneNumber == "" {
		log.Error("Room phone number is empty")
		return
	}
	log.Debugf("Room phone number: %s", room.PhoneNumber)
	//get provider
	var hotelProvider hotel.HospitalityProvider
	hotelProvider = clbClient
	msg, err := hotelProvider.UpdateRoom(room.PhoneNumber, room.RoomCondition, room.HouskeeperID)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
}

func handleSetHousekeepingStatus(w http.ResponseWriter, r *http.Request) {
	log.Debugf("handleSetHousekeepingStatus")

	// Get the housekeeping info from the URL
	vars := mux.Vars(r)
	roomPhoneNumber := vars["roomPhoneNumber"]
	housekeepingStatus := vars["housekeepingStatus"]
	housekeeperID := vars["housekeeperID"]

	//get provider
	var hotelProvider hotel.HospitalityProvider

	log.Debugf("roomPhoneNumber: %s, housekeepingStatus: %s, housekeeperID: %s", roomPhoneNumber, housekeepingStatus, housekeeperID)
	hotelProvider = clbClient
	msg, err := hotelProvider.UpdateRoom(roomPhoneNumber, housekeepingStatus, housekeeperID)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(msg))
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
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}

	// Return the status code, response body, and nil error
	return resp.StatusCode, string(bodyBytes), nil
}

func readAuthVarsFromFile() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}
