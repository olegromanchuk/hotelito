package main

import (
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestMainFunction(t *testing.T) {

	//create .env file for testing
	envTestFileName := ".env_test"
	file := createEnvFile(envTestFileName)
	defer os.Remove(envTestFileName)
	defer file.Close()

	//create test_config.json file for testing
	configTestFileName := "test_config.json"
	fileConfig := createTestConfigFile(configTestFileName)
	defer os.Remove(configTestFileName)
	defer fileConfig.Close()

	//create test_cloudbeds_api_params.json file for testing
	configApiParams := "test_cloudbeds_api_params.json"
	fileApiConfig := createAPIParamsConfigFile(configApiParams)
	defer os.Remove(configApiParams)
	defer fileApiConfig.Close()

	hook := test.NewGlobal()
	quit := make(chan struct{})

	go runServer(envTestFileName, logrus.StandardLogger(), quit)

	time.Sleep(2 * time.Second)

	found := false
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "Starting server on port") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected log message not found")

	// Perform your HTTP tests here.
	resp, err := http.Get("http://localhost:8080/api/v1/healthcheck")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body := resp.Body
	defer body.Close()
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, body)
	assert.Equal(t, "OK", buf.String())

	// Signal the main function to stop the server
	close(quit)

	// Optionally, give it some time to shutdown
	time.Sleep(1 * time.Second)
}
