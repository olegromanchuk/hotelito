package lambda_boilerplate

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func TestInitializeVariablesFromEnv(t *testing.T) {
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	// Test with no environment variables set
	os.Clearenv()
	appName, environmentType, awsRegion := InitializeVariablesFromEnv(logger)

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

	appName, environmentType, awsRegion = InitializeVariablesFromEnv(logger)
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
			logger := InitializeLogger()

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
