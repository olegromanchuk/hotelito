package configuration

import (
	"github.com/sirupsen/logrus"
	"os"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {

	// Setup: create config.json file
	roomMapData := `{
		"100": "544559-0",
		"101": "544559-1"
    }`

	err := os.WriteFile("roomid_map_test.json", []byte(roomMapData), 0644)
	if err != nil {
		t.Fatalf("unable to set up test: %v", err)
	}

	type args struct {
		log         *logrus.Logger
		mapFileName string
	}
	tests := []struct {
		name    string
		args    args
		want    *ConfigMap
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.log, tt.args.mapFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() got = %v, want %v", got, tt.want)
			}
		})
	}

	// Teardown: delete roomid_map_test.json file
	err = os.Remove("roomid_map_test.json")
	if err != nil {
		t.Errorf("unable to tear down test: %v", err)
	}
}
