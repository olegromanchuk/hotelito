package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/internal/logging"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/boltstore"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Define the flag
	configFileName := flag.String("config", ".env", "Path to the config file")

	// Parse the flags
	flag.Parse()

	quit := make(chan struct{})
	runServer(*configFileName, quit)
}

func runServer(envFileName string, quit chan struct{}) {

	//define logger
	log := logrus.New()

	//load .env variables into environment
	readAuthVarsFromFile(envFileName, log)

	// The default level is debug.
	logLevelEnv := os.Getenv("LOG_LEVEL")
	if logLevelEnv == "" {
		logLevelEnv = "debug"
	}

	logLevel, err := logrus.ParseLevel(logLevelEnv)
	if err != nil {
		logLevel = logrus.DebugLevel
	}

	//custom formatter will add caller name to the logging
	traceID := logging.GenerateTraceID()
	if logLevel >= 5 { //Debug or Trace level
		log.Formatter = &logging.CustomFormatter{CustomFormatter: &logrus.TextFormatter{}, TraceID: traceID}
	}

	log.SetLevel(logLevel)

	// Set output of logs to Stdout
	log.SetOutput(os.Stdout)
	log.Infof("Log level: %s", logLevelEnv)

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(loggingMiddleware)

	//parse config.json
	mapFileName := os.Getenv("HOSPITALITY_PHONE2ROOM_MAP_FILENAME")
	CloudbedsApiConfFileName := os.Getenv("HOSPITALITY_API_CONF_FILENAME")

	configMap, err := configuration.New(log, mapFileName, CloudbedsApiConfFileName)
	if err != nil {
		log.Fatal(err) //TODO: add error handling. Try to load previous version of configMap
	}

	//   ---------------------- Cloudbed parts ----------------------

	//current secret store - boltDB
	storeClient, err := InitializeStore()
	if err != nil {
		log.Fatal(err)
	}

	//create cloudbeds client
	clbClient, err := cloudbeds.New(log, storeClient, configMap)
	if err != nil {
		log.Fatal(err)
	}
	defer clbClient.Close()

	//create 3cx client
	pbx3cxClient := pbx3cx.New(log, configMap)
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

	port := ":" + os.Getenv("PORT")
	if port == ":" {
		log.Warn("PORT env variable is not set. Using default port 8080")
		port = ":8080"
	}

	server := &http.Server{Addr: port}

	go func() {
		fmt.Printf("Starting server on port %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
}

func readAuthVarsFromFile(fileName string, log *logrus.Logger) {
	log.Warnf("Loading .env file: %s", fileName)
	err := godotenv.Load(fileName)
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func InitializeStore() (*boltstore.BoltDBStore, error) {

	//get file name from env
	dbFileName := os.Getenv("STANDALONE_VERSION_BOLT_DB_FILENAME")
	if dbFileName == "" {
		return nil, fmt.Errorf("STANDALONE_VERSION_BOLT_DB_FILENAME env variable is not set")
	}

	bucketName := os.Getenv("STANDALONE_VERSION_BOLT_DB_BUCKET_NAME")
	if bucketName == "" {
		return nil, fmt.Errorf("STANDALONE_VERSION_BOLT_DB_BUCKET_NAME env variable is not set")
	}

	storeClient, err := boltstore.Initialize(dbFileName, bucketName)
	if err != nil {
		errMsg := fmt.Sprintf("error initializing bolt store: %s", err)
		return nil, errors.New(errMsg)
	}

	//check returned store object has BucketName is set
	if storeClient.BucketName == "" {
		return nil, fmt.Errorf("error durin store initialization. storeClient.BucketName variable is not set")
	}

	return storeClient, nil
}
