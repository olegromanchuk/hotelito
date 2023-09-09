package configuration

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	log := logrus.New()

	// Success scenario
	t.Run("success", func(t *testing.T) {
		// Create a temporary file and write JSON content into it.
		tmpFile, err := os.CreateTemp("", "prefix-")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		configMap := &ConfigMap{
			// Your fields here
		}
		bytes, err := json.Marshal(configMap)
		require.NoError(t, err)
		_, err = tmpFile.Write(bytes)
		require.NoError(t, err)

		result, err := New(log, tmpFile.Name(), "clBedsApiConfigFile")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "clBedsApiConfigFile", result.ApiCfgFileName)
		// Add more assertions to verify the contents of result.
	})

	// Failure scenario: File does not exist
	t.Run("file does not exist", func(t *testing.T) {
		_, err := New(log, "nonexistentfile", "")
		require.Error(t, err)
	})

	// Failure scenario: Malformed JSON
	t.Run("malformed json", func(t *testing.T) {
		// Create a temporary file and write malformed JSON content into it.
		tmpFile, err := os.CreateTemp("", "prefix-")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(`{"malformed json"`))
		require.NoError(t, err)

		_, err = New(log, tmpFile.Name(), "")
		require.Error(t, err)
	})
}
