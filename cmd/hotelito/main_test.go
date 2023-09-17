package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"testing"
)

func createEnvFile(envTestFileName string) (file *os.File) {
	//create .env file for testing
	file, err := os.Create(envTestFileName)
	if err != nil {
		fmt.Printf("Error creating file %v:. Error: %v", envTestFileName, err)
		panic(err)
	}

	envContent := `ENVIRONMENT=production
APPLICATION_NAME=hotelito-app
LOG_LEVEL=debug
CLOUDBEDS_CLIENT_ID=mycompanyexample_LuPCZsereqdqdXjS
CLOUDBEDS_CLIENT_SECRET=sadfsadkjHKJujewnfw32SDDFFD
CLOUDBEDS_REDIRECT_URL=https://mypublic.api.address/api/v1/callback
CLOUDBEDS_SCOPES=read:hotel,read:reservation,write:reservation,read:room,write:room,read:housekeeping,write:housekeeping
CLOUDBEDS_AUTH_URL=https://hotels.cloudbeds.com/api/v1.1/oauth
CLOUDBEDS_TOKEN_URL=https://hotels.cloudbeds.com/api/v1.1/access_token
HOSPITALITY_PHONE2ROOM_MAP_FILENAME=config.json
HOSPITALITY_API_CONF_FILENAME=cloudbeds_api_params.json
PORT=8080
AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID=hotelito-app-3cxroomextension-cloudbedsroomid
AWS_S3_BUCKET_4_CLBEDS_API_CONF=hotelito-app-3cxroomextension-cloudbedsroomid
STANDALONE_VERSION_BOLT_DB_FILENAME=secrets.db
STANDALONE_VERSION_BOLT_DB_BUCKET_NAME=cloudbeds_creds`

	_, err = file.WriteString(envContent)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		panic(err)
	}

	fmt.Println(".env file has been created.")
	return file
}

func Test_readAuthVarsFromFile(t *testing.T) {
	envTestFileName := ".env_test"
	file := createEnvFile(envTestFileName)
	defer os.Remove(envTestFileName)
	defer file.Close()

	//test that environmental vars are loaded into memory
	t.Run("check that file .env is properly loaded into memory", func(t *testing.T) {
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		readAuthVarsFromFile(envTestFileName, logger)
	})

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		key, expectedValue := parts[0], parts[1]

		actualValue := os.Getenv(key)
		if actualValue != expectedValue {
			t.Errorf("For env variable %s, expected %s but got %s", key, expectedValue, actualValue)
		}
	}
}

func TestInitializeStore(t *testing.T) {

	dbFileName := "test.db"
	tests := []struct {
		name           string
		dbEnv          string
		bucketEnv      string
		expectError    bool
		expectedErrMsg string
		expectedBucket string
	}{
		{
			name:           "valid environment variables",
			dbEnv:          dbFileName,
			bucketEnv:      "test_bucket",
			expectError:    false,
			expectedBucket: "test_bucket",
		},
		{
			name:           "missing db env",
			dbEnv:          "",
			bucketEnv:      "test_bucket",
			expectError:    true,
			expectedErrMsg: "STANDALONE_VERSION_BOLT_DB_FILENAME env variable is not set",
		},
		{
			name:           "missing bucket env",
			dbEnv:          dbFileName,
			bucketEnv:      "",
			expectError:    true,
			expectedErrMsg: "STANDALONE_VERSION_BOLT_DB_BUCKET_NAME env variable is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock environment variables
			os.Setenv("STANDALONE_VERSION_BOLT_DB_FILENAME", tt.dbEnv)
			os.Setenv("STANDALONE_VERSION_BOLT_DB_BUCKET_NAME", tt.bucketEnv)

			store, err := InitializeStore()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErrMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBucket, store.BucketName)
			}
		})
	}
	os.Remove(dbFileName)
}
