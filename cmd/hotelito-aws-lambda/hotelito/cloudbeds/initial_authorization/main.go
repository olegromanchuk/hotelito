package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/olegromanchuk/hotelito/cmd/hotelito-aws-lambda/hotelito/lambda_boilerplate"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/awsstore"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func HandleInit(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Printf("[%s] Started HandleInit", time.Now().String())

	log := lambda_boilerplate.InitializeLogger()
	log.Debug(request)

	responseApiGateway, err := Execute(log)
	if err != nil {
		log.Errorf("Error executing handler: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, err
	}

	return responseApiGateway, nil
}

func Execute(log *logrus.Logger) (responseApiGateway events.APIGatewayProxyResponse, returnError error) {

	appName, environmentType, awsRegion := lambda_boilerplate.InitializeVariablesFromEnv(log)
	storePrefix := fmt.Sprintf("%s/%s", appName, environmentType) //hotelito-app-production

	//Initialize current secret store - aws env variables
	storeClient, err := awsstore.Initialize(log, storePrefix, awsRegion, nil)
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
		return responseApiGateway, err
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
		responseApiGateway = events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}
		return responseApiGateway, err
	}
	log.Debugf("redirect url: %s", url)

	responseApiGateway.StatusCode = http.StatusFound
	responseApiGateway.Headers = map[string]string{
		"Location": url,
	}

	return responseApiGateway, nil
}

func main() {
	lambda.Start(HandleInit)
}
