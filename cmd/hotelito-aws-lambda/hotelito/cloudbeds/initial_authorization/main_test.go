package main

import (
	"context"
	"errors"
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
	ctx := context.Background()
	if err := localstacktest.StartLocalStack(ctx); err != nil {
		panic(err)
	}

	// run tests
	code := m.Run()

	if err := localstacktest.StopLocalStack(ctx); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestExecute(t *testing.T) {

	log := logrus.New()

	customAWSConfig := &aws.Config{
		Region:   aws.String("us-east-1"),
		Endpoint: aws.String("http://localhost:4566"),
		Credentials: credentials.NewStaticCredentials(
			"accessKeyID",
			"secretAccessKey",
			"token",
		)}

	type args struct {
		log             *logrus.Logger
		request         events.APIGatewayProxyRequest
		customAWSConfig *aws.Config
	}

	tests := []struct {
		name                    string
		args                    args
		request                 events.APIGatewayProxyRequest
		expected                events.APIGatewayProxyResponse
		setEnvironmentVariables bool
		expectedError           error
		expectedLogMessage      string
		hasError                bool
	}{
		{
			name: "error case: no vars are set",
			args: args{
				log:             log,
				request:         events.APIGatewayProxyRequest{},
				customAWSConfig: customAWSConfig,
			},
			request: events.APIGatewayProxyRequest{},
			expected: events.APIGatewayProxyResponse{
				StatusCode: 500,
			},
			setEnvironmentVariables: false,
			expectedError:           errors.New("not all required env variables are set. Missed one of"),
			hasError:                true,
		},
		{
			name: "success case",
			args: args{
				log:             log,
				request:         events.APIGatewayProxyRequest{},
				customAWSConfig: customAWSConfig,
			},
			request: events.APIGatewayProxyRequest{
				// Populate fields as needed for this test case
			},
			expected: events.APIGatewayProxyResponse{
				// Populate fields that you expect to be returned in a successful case
				StatusCode: 302,
				Headers: map[string]string{
					"Location": "test_auth_url?client_id=test_client_id&redirect_uri=test_redirect_url&response_type=code&scope=test_scopes",
				},
			},
			setEnvironmentVariables: true,
			expectedError:           nil,
			expectedLogMessage:      "AWS_REGION env variable is not set",
			hasError:                false,
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
			}

			mapOfValues := map[string]string{"state": "someRandomString"}
			saveValuesToLocalStack(mapOfValues, tt.args.customAWSConfig)

			resp, err := Execute(tt.args.log, tt.args.customAWSConfig)

			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.Contains(t, resp.Headers["Location"], tt.expected.Headers["Location"])
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
