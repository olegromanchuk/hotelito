package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/olegromanchuk/hotelito/cmd/hotelito-aws-lambda/hotelito/lambda_boilerplate"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/secrets/awsstore"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func HandleCallback(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Printf("[%s] Started HandleCallback", time.Now().String())

	log := lambda_boilerplate.InitializeLogger()
	log.Debug(request)

	responseApiGateway, err := Execute(log, request, nil)
	if err != nil {
		log.Errorf("Error executing handler: %v", err)
		responseApiGateway.StatusCode = http.StatusOK //we need to reply with dignity: 200 to cloudbeds
		responseApiGateway.Body = fmt.Sprintf("Error: %v", err)
	}

	return responseApiGateway, nil
}

// Execute is the main function that handles the request
// customAWSConfig is needed for testing to redirect AWS.SSM traffic to localstack. In production, we pass nil for customAWSConfig.
func Execute(log *logrus.Logger, request events.APIGatewayProxyRequest, customAWSConfig *aws.Config) (responseApiGateway events.APIGatewayProxyResponse, returnError error) {
	responseApiGateway = events.APIGatewayProxyResponse{}

	appName, environmentType, awsRegion := lambda_boilerplate.InitializeVariablesFromEnv(log)
	storePrefix := fmt.Sprintf("%s/%s", appName, environmentType) //hotelito-app-production

	//current secret store - aws env variables
	storeClient, err := awsstore.Initialize(log, storePrefix, awsRegion, customAWSConfig)
	if err != nil {
		return responseApiGateway, err
	}

	//create cloudbeds client
	clbClient, err := cloudbeds.NewClient4CallbackAndInit(log, storeClient)
	if err != nil {
		log.Errorf("Error creating cloudbeds client: %v", err)
		return responseApiGateway, err
	}

	// extract state and code from request
	log.Debugf("Handling callback")
	state := request.QueryStringParameters["state"]
	code := request.QueryStringParameters["code"]

	err = clbClient.HandleOAuthCallback(state, code)
	if err != nil {
		log.Error(err)
		return responseApiGateway, err
	}

	responseApiGateway.StatusCode = http.StatusOK
	responseApiGateway.Body = "Success"

	return responseApiGateway, nil
}

func main() {
	lambda.Start(HandleCallback)
}
