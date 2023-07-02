package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/olegromanchuk/hotelito/internal/handlers"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Print(request)
	//define logger
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(os.Stdout)

	//current secret store - aws env variables
	storeClient, err := awsStore.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	//create cloudbeds client
	clbClient := cloudbeds.New(log, storeClient)

	//create 3cx client
	pbx3cxClient := pbx3cx.New(log)

	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)

	// Your handle login logic here
	log.Debugf("Handling callback")
	state := request.QueryStringParameters["state"]
	code := request.QueryStringParameters["code"]
	err := h.Hotel.HandleCallback(state, code)
	if err != nil {
		log.Error(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}
	log.Debugf("Got auth code: %s state: %s", code, state)
	log.Infof("Ready for future requests")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       "Great Success! Ready for future requests. You can close this window now.",
	}, nil
}

// func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
// 	resp, err := http.Get(DefaultHTTPGetAddress)
// 	if err != nil {
// 		return events.APIGatewayProxyResponse{}, err
// 	}

// 	if resp.StatusCode != 200 {
// 		return events.APIGatewayProxyResponse{}, ErrNon200Response
// 	}

// 	ip, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return events.APIGatewayProxyResponse{}, err
// 	}

// 	if len(ip) == 0 {
// 		return events.APIGatewayProxyResponse{}, ErrNoIP
// 	}

// 	return events.APIGatewayProxyResponse{
// 		Body:       fmt.Sprintf("Hello, %v", string(ip)),
// 		StatusCode: 200,
// 	}, nil
// }

func main() {
	lambda.Start(HandleRequest)
}
