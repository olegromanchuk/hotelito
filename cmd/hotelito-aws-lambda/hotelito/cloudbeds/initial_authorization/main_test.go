package main

import (
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestHandleInit(t *testing.T) {

	//set env variables
	os.Setenv("CLOUDBEDS_CLIENT_ID", "test_client_id")
	os.Setenv("CLOUDBEDS_CLIENT_SECRET", "test_client_secret")
	os.Setenv("CLOUDBEDS_REDIRECT_URL", "test_redirect_url")
	os.Setenv("CLOUDBEDS_AUTH_URL", "test_auth_url")
	os.Setenv("CLOUDBEDS_TOKEN_URL", "test_token_url")
	os.Setenv("CLOUDBEDS_SCOPES", "test_scopes")

	tests := []struct {
		name                    string
		request                 events.APIGatewayProxyRequest
		expected                events.APIGatewayProxyResponse
		setEnvironmentVariables bool
		expectedError           error
		expectedLogMessage      string
		hasError                bool
	}{
		{
			name:    "error case: no vars are set",
			request: events.APIGatewayProxyRequest{},
			expected: events.APIGatewayProxyResponse{
				StatusCode: 500,
			},
			setEnvironmentVariables: false,
			expectedError:           errors.New("not all required env variables are set"),
			hasError:                true,
		},
		{
			name:    "success case",
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

			ctx := context.Background()
			resp, err := HandleInit(ctx, tt.request)

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
