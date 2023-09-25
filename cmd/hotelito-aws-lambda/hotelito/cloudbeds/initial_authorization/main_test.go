package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
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

	fmt.Printf("🧪🚀 Tests started init_auth\n")
	// run tests
	code := m.Run()
	fmt.Printf("🧪✅ Tests finished init_auth\n")

	// Terminate LocalStack if this is the last package
	if err := localstacktest.StopLocalStack(); err != nil {
		log.Fatalf("Could not terminate LocalStack: %v", err)
	}
	os.Exit(code)
}

func TestExecute(t *testing.T) {

	log := logrus.New()

	//get customAWSConfig from localstacktest
	customAWSConfig := localstacktest.GetCustomAWSConfig()

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
			localstacktest.SaveValuesToLocalStack(mapOfValues, tt.args.customAWSConfig)

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
