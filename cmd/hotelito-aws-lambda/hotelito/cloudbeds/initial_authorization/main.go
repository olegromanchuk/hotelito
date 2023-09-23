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
	"time"
)

// vars below are used ONLY if env vars are not set (testing only). It is not supposed to happen in production.
var (
	defaultAppName         = "hotelito-app"
	defaultEnvironmentType = "dev"
	defaultAwsRegion       = "us-east-2"
)

func HandleInit(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("[%s] Started HandleInit", time.Now().String())

	log := initializeLogger()
	log.Debug(request)

	responseApiGateway, url, err := Execute(log)
	if err != nil {
		log.Errorf("Error executing handler: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, err
	}

	responseApiGateway.StatusCode = http.StatusFound
	responseApiGateway.Headers = map[string]string{
		"Location": url,
	}

	return responseApiGateway, nil
}

func initializeLogger() *logrus.Logger {
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
	log.Infof("Log level: %s", logLevel)

	return log
}

func Execute(log *logrus.Logger) (responseApiGateway events.APIGatewayProxyResponse, url string, returnError error) {

	appName, environmentType, awsRegion := initializeVariablesFromEnv(log)
	storePrefix := fmt.Sprintf("%s/%s", appName, environmentType) //hotelito-app-production

	//Initialize current secret store - aws env variables
	storeClient, err := awsstore.Initialize(log, storePrefix, awsRegion)
	if err != nil {
		errMsg := fmt.Sprintf("error initializing AWS SSM store with store prefix %s in region %s. Error: %v", storePrefix, awsRegion, err)
		log.Fatal(errMsg)
	}

	storeClient.Log = log //set logger for store client

	//create cloudbeds client
	clbClient, err := cloudbeds.NewClient4CallbackAndInit(log, storeClient)
	if err != nil {
		log.Errorf("Error creating cloudbeds client: %v", err)
		responseApiGateway = events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}
		return responseApiGateway, url, err
	}

	log.Debugf("Handling init")

	//create 3cx client
	//pbx3cxClient := pbx3cx.New(log, configMap)
	pbx3cxClient := &pbx3cx.PBX3CX{} //we do not need full-blown 3cx client for initial authorization
	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)
	url, err = h.Hotel.HandleInitialLogin()
	if err != nil {
		log.Error(err)
		responseApiGateway = events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}
		return responseApiGateway, url, err
	}
	log.Debugf("redirect url: %s", url)

	return responseApiGateway, url, nil
}

func initializeVariablesFromEnv(log *logrus.Logger) (appName, environmentType, awsRegion string) {
	//get APP_NAME from env
	appName = os.Getenv("APPLICATION_NAME")
	if appName == "" {
		log.Debug("APPLICATION_NAME env variable is not set")
		appName = defaultAppName
	}
	log.Debugf("APPLICATION_NAME: %s", appName)

	environmentType = os.Getenv("ENVIRONMENT")
	if environmentType == "" {
		log.Debug("ENVIRONMENT env variable is not set")
		environmentType = defaultEnvironmentType
	}
	log.Debugf("ENVIRONMENT: %s", environmentType)

	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Debug("AWS_REGION env variable is not set")
		awsRegion = defaultAwsRegion
	}
	log.Debugf("AWS_REGION: %s", awsRegion)

	return appName, environmentType, awsRegion
}

func main() {
	lambda.Start(HandleInit)
}
