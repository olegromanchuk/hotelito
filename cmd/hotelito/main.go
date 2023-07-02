package main

import (
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/boltstore"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func main() {

	//define logger
	log := logrus.New()
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

	//current secret store - boltDB
	storeClient, err := boltstore.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	//create cloudbeds client
	clbClient := cloudbeds.New(log, storeClient)
	defer clbClient.Close()

	//create 3cx client
	pbx3cxClient := pbx3cx.New(log)
	defer clbClient.Close()

	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)

	//auth urls
	api.HandleFunc("/", h.HandleMain).Methods("GET")
	api.HandleFunc("/login", h.HandleManualLogin).Methods("GET")
	api.HandleFunc("/callback", h.HandleCallback).Methods("GET")

	//test/troubleshooting urls
	//update housekeeping status
	// test data: "544559-0", "clean"
	api.HandleFunc("/housekeepings/{roomPhoneNumber}/{housekeepingStatus}/{housekeeperID}", h.HandleSetHousekeepingStatus).Methods("POST")
	api.HandleFunc("/getRooms", h.HandleGetRooms).Methods("GET")

	//3cx call info receiver
	api.HandleFunc("/3cx/lookupbynumber", h.Handle3cxLookup).Methods("GET")
	api.HandleFunc("/3cx/outbound_call", h.Handle3cxCallInfo).Methods("POST")

	http.Handle("/", api)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func readAuthVarsFromFile() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}
