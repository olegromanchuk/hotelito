package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/internal/logging"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/awsstore"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func HandleInit(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println(request)
	//define logger
	log := logrus.New()

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
	log.SetOutput(os.Stdout)
	log.Infof("Log level: %s", logLevelEnv)

	//get APP_NAME from env
	appName := os.Getenv("APPLICATION_NAME")
	if appName == "" {
		log.Debug("APPLICATION_NAME env variable is not set")
		appName = "hotelito-app"
	}
	log.Debugf("APPLICATION_NAME: %s", appName)

	environmentType := os.Getenv("ENVIRONMENT")
	if environmentType == "" {
		log.Debug("ENVIRONMENT env variable is not set")
		environmentType = "dev"
	}
	log.Debugf("ENVIRONMENT: %s", environmentType)

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Debug("AWS_REGION env variable is not set")
		awsRegion = "us-east-2"
	}
	log.Debugf("AWS_REGION: %s", awsRegion)

	storePrefix := fmt.Sprintf("%s/%s", appName, environmentType) //hotelito-app-production
	//current secret store - aws env variables
	storeClient, err := awsstore.Initialize(log, storePrefix, awsRegion)
	if err != nil {
		log.Fatal(err)
	}
	storeClient.Log = log //set logger for store client

	//create cloudbeds client
	clbClient, err := cloudbeds.NewClient4CallbackAndInit(log, storeClient)
	if err != nil {
		log.Errorf("Error creating cloudbeds client: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}

	log.Debugf("Handling init")

	//create 3cx client
	//pbx3cxClient := pbx3cx.New(log, configMap)
	pbx3cxClient := &pbx3cx.PBX3CX{} //we do not need full-blown 3cx client for initial authorization
	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)
	url, err := h.Hotel.HandleInitialLogin()
	if err != nil {
		log.Error(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}
	log.Debugf("redirect url: %s", url)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location": url,
		},
	}, nil
}

func main() {
	lambda.Start(HandleInit)
}
