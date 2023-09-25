package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/olegromanchuk/hotelito/cmd/hotelito-aws-lambda/hotelito/localstacktest"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := localstacktest.StartLocalStack(); err != nil {
		panic(err)
	}

	fmt.Printf("🧪🚀 Tests started: oauth2\n")
	// run tests
	code := m.Run()
	fmt.Printf("🧪✅ Tests finished oauth2\n")

	// Terminate LocalStack if this is the last package
	if err := localstacktest.StopLocalStack(); err != nil {
		log.Fatalf("Could not terminate LocalStack: %v", err)
	}
	os.Exit(code)
}

func TestExecute(t *testing.T) {

	log := logrus.New()
	request := events.APIGatewayProxyRequest{}

	//get localstack config from env variables
	localstack_host := os.Getenv("LOCALSTACK_HOST")
	localstack_port := os.Getenv("LOCALSTACK_PORT")
	if localstack_host == "" || localstack_port == "" {
		log.Fatalf("💩🤷 Error getting localstack host and port from env variables. Check localstacktest.go and TestMain()")
		return
	}

	customAWSConfig := &aws.Config{
		Region:   aws.String("us-east-1"),
		Endpoint: aws.String(fmt.Sprintf("http://%s:%s", localstack_host, localstack_port)),
		Credentials: credentials.NewStaticCredentials(
			"accessKeyID",
			"secretAccessKey",
			"token",
		)}

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
			os.Clearenv()

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
				saveValuesToLocalStack(mapOfValues, tt.args.customAWSConfig)
			}

			gotResponseApiGateway, err := Execute(tt.args.log, tt.args.request, tt.args.customAWSConfig)

			if tt.expectedErrorContains != nil {
				assert.Contains(t, err.Error(), tt.expectedErrorContains.Error())
			} else {
				assert.Equalf(t, tt.wantResponseApiGateway, gotResponseApiGateway, "Execute(%v, %v, %v)", tt.args.log, tt.args.request, tt.args.customAWSConfig)
			}

			// Unset env variables
			os.Clearenv()
		})
	}
}

func saveValuesToLocalStack(mapOfValues map[string]string, customAWSConfig *aws.Config) {

	// Initialize a session
	sess, err := session.NewSession(customAWSConfig)

	if err != nil {
		log.Fatalf("Error creating session: %v", err)
		return
	}

	// Create SSM service client
	ssmSvc := ssm.New(sess)

	for k, d := range mapOfValues {
		paramName := k
		paramValue := d

		// Put the parameter
		putParamInput := &ssm.PutParameterInput{
			Name:      aws.String(paramName),
			Value:     aws.String(paramValue),
			Overwrite: aws.Bool(true), // Set to true to update existing parameter
			Type:      aws.String("String"),
		}

		_, err = ssmSvc.PutParameter(putParamInput)
		if err != nil {
			log.Fatalf("Error putting SSM parameter: %v", err)
			return
		}
		log.Printf("Successfully put SSM parameter %s with value %s", paramName, paramValue)
	}

}
