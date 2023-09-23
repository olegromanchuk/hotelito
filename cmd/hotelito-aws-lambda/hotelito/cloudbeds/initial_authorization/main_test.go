package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io"
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

func TestInitializeVariablesFromEnv(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	// Test with no environment variables set
	os.Clearenv()
	appName, environmentType, awsRegion := initializeVariablesFromEnv(logger)

	assert.Equal(t, defaultAppName, appName)
	assert.Equal(t, defaultEnvironmentType, environmentType)
	assert.Equal(t, defaultAwsRegion, awsRegion)

	// Check if debug logs are generated correctly
	assert.Equal(t, 6, len(hook.Entries))
	assert.Equal(t, "APPLICATION_NAME env variable is not set", hook.Entries[0].Message)
	assert.Equal(t, "ENVIRONMENT env variable is not set", hook.Entries[2].Message)
	assert.Equal(t, "AWS_REGION env variable is not set", hook.Entries[4].Message)

	// Clear log entries
	hook.Reset()

	// Test with environment variables set
	os.Setenv("APPLICATION_NAME", "test-app")
	os.Setenv("ENVIRONMENT", "test-env")
	os.Setenv("AWS_REGION", "us-west-1")

	appName, environmentType, awsRegion = initializeVariablesFromEnv(logger)
	assert.Equal(t, "test-app", appName)
	assert.Equal(t, "test-env", environmentType)
	assert.Equal(t, "us-west-1", awsRegion)

	// No debug log should be generated for missing env variables
	for _, entry := range hook.Entries {
		assert.NotEqual(t, "APPLICATION_NAME env variable is not set", entry.Message)
		assert.NotEqual(t, "ENVIRONMENT env variable is not set", entry.Message)
		assert.NotEqual(t, "AWS_REGION env variable is not set", entry.Message)
	}
}

func TestInitializeLogger(t *testing.T) {
	tests := []struct {
		name          string
		envLogLevel   string
		expectedLevel logrus.Level
		expectedMsg   string
	}{
		{
			"Default log level",
			"",
			logrus.DebugLevel,
			"Log level: debug",
		},
		{
			"Info log level",
			"info",
			logrus.InfoLevel,
			"Log level: info",
		},
		{
			"Invalid log level",
			"garbage",
			logrus.DebugLevel, // Fallback to debug
			"Log level: debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing env vars for this sub-test
			os.Clearenv()

			// Redirect stdout to a buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set the env var for this sub-test
			if tt.envLogLevel != "" {
				os.Setenv("LOG_LEVEL", tt.envLogLevel)
			}

			// Initialize logger
			logger := initializeLogger()

			// Close and restore stdout, read the captured output
			w.Close()
			os.Stdout = oldStdout
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			captured := buf.String()

			// Use assertions
			assert.Equal(t, tt.expectedLevel, logger.Level)
			assert.Contains(t, captured, tt.expectedMsg)
		})
	}
}
