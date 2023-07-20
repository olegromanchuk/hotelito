package main

import (
	"context"
	"fmt"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/internal/logging"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/awsstore"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleLookupByNumber(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	configMap := &configuration.ConfigMap{} //we do not use configuration in this lambda. Empty struct is ok
	//create cloudbeds client
	clbClient, err := cloudbeds.New(log, storeClient, configMap)
	if err != nil {
		log.Errorf("Error creating cloudbeds client: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}

	// exctract state and code from request
	log.Debugf("Handling lookup by number request: %v", request)
	number := request.QueryStringParameters["Number"]

	//option via handler interface. Helpful for testing
	//create 3cx client
	pbx3cxClient := pbx3cx.New(log, configMap)
	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)
	jsonAsBytes, err := h.PBX.ProcessLookupByNumber(number)
	if err != nil {
		log.Error(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonAsBytes),
	}, nil
}

func main() {
	lambda.Start(HandleLookupByNumber)
}
