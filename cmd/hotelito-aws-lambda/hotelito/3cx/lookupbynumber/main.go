package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/olegromanchuk/hotelito/cmd/hotelito-aws-lambda/hotelito/lambda_boilerplate"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"net/http"
)

func HandleLookupByNumber(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	//define logger
	log := lambda_boilerplate.InitializeLogger()
	log.Debug(request)

	// extract state and code from request
	log.Debugf("Handling lookup by number request: %v", request)
	number := request.QueryStringParameters["Number"]

	jsonAsBytes := pbx3cx.ProcessLookupByNumber(number) //returns dummy contact with "number"

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(jsonAsBytes),
	}, nil
}

func main() {
	lambda.Start(HandleLookupByNumber)
}
