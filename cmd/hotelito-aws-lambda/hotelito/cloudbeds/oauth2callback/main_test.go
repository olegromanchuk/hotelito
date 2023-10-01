package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/olegromanchuk/hotelito/cmd/hotelito-aws-lambda/hotelito/localstacktest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {

	os.Setenv("LOCALSTACK_HOST", "localhost")
	os.Setenv("LOCALSTACK_PORT", "4566")

	fmt.Printf("🧪🚀 Tests started: oauth2\n")
	// run tests
	code := m.Run()
	fmt.Printf("🧪✅ Tests finished oauth2\n")

	os.Exit(code)
}

func TestExecute(t *testing.T) {

	log := logrus.New()
	request := events.APIGatewayProxyRequest{}

	//get customAWSConfig from localstacktest
	customAWSConfig := localstacktest.GetCustomAWSConfig()

	// Create the APIGatewayProxyRequest object
	requestWState := events.APIGatewayProxyRequest{
		QueryStringParameters: map[string]string{
			"state": "someRandomString",
		},
		// Fill other necessary fields.
	}

	type args struct {
		log             *logrus.Logger
		request         events.APIGatewayProxyRequest
		customAWSConfig *aws.Config
	}
	tests := []struct {
		name                    string
		args                    args
		setEnvironmentVariables bool
		setupLocalStack         bool
		wantResponseApiGateway  events.APIGatewayProxyResponse
		expectedErrorContains   error
	}{
		{
			name: "error: oauth state not found in store",
			args: args{
				log:             log,
				request:         requestWState,
				customAWSConfig: customAWSConfig,
			},
			setupLocalStack:         false,
			setEnvironmentVariables: true,
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
			},
			expectedErrorContains: errors.New("failed to retrieve oauth state from secret store: ParameterNotFound: Parameter /hotelito-app/dev/someRandomString not found"),
		},

		{
			name: "error case: no vars are set",
			args: args{
				log:             log,
				request:         request,
				customAWSConfig: customAWSConfig,
			},
			setEnvironmentVariables: false,
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
			},
			expectedErrorContains: errors.New("not all required env variables are set. Missed one of"),
		},
		{
			name: "error case: failed to retrieve oauth state",
			args: args{
				log:             log,
				request:         request,
				customAWSConfig: customAWSConfig,
			},
			setEnvironmentVariables: true,
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
			},
			expectedErrorContains: errors.New("failed to retrieve oauth state from secret store: ParameterNotFound: Parameter /hotelito-app/dev not found"),
		},
		{
			name: "error case w localstack",
			args: args{
				log:             log,
				request:         requestWState,
				customAWSConfig: customAWSConfig,
			},
			setupLocalStack:         true,
			setEnvironmentVariables: true,
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
			},
			expectedErrorContains: errors.New("failed to retrieve oauth state from secret store"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unset env variables
			localstacktest.ClearEnvVars()

			if tt.setEnvironmentVariables {
				// Unset env variables after the "success" test case
				os.Setenv("CLOUDBEDS_CLIENT_ID", "test_client_id")
				os.Setenv("CLOUDBEDS_CLIENT_SECRET", "test_client_secret")
				os.Setenv("CLOUDBEDS_REDIRECT_URL", "test_redirect_url")
				os.Setenv("CLOUDBEDS_AUTH_URL", "test_auth_url")
				os.Setenv("CLOUDBEDS_TOKEN_URL", "test_token_url")
				os.Setenv("CLOUDBEDS_SCOPES", "test_scopes")
				os.Setenv("AWS_REGION", "us-east-1")
			}

			if tt.setupLocalStack {
				mapOfValues := map[string]string{"state": "someRandomString"}
				localstacktest.SaveValuesToLocalStack(mapOfValues, tt.args.customAWSConfig)
			}

			gotResponseApiGateway, err := Execute(tt.args.log, tt.args.request, tt.args.customAWSConfig)

			if tt.expectedErrorContains != nil {
				assert.Contains(t, err.Error(), tt.expectedErrorContains.Error())
			} else {
				assert.Equalf(t, tt.wantResponseApiGateway, gotResponseApiGateway, "Execute(%v, %v, %v)", tt.args.log, tt.args.request, tt.args.customAWSConfig)
			}

			// Unset env variables
			localstacktest.ClearEnvVars()
		})
	}
}
