package main

import (
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

	hook := test.NewGlobal()
	quit := make(chan struct{})

	go runServer(envTestFileName, quit)

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
	resp, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Signal the main function to stop the server
	close(quit)

	// Optionally, give it some time to shutdown
	time.Sleep(1 * time.Second)
}
