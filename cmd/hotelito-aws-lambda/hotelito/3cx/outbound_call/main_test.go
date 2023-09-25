package main

import (
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

var useDockerLocalstack = true //use docker for localstack for local debugging
// docker run -d --name localstacktestt --rm -it -p 4566:4566 localstack/localstack

func TestMain(m *testing.M) {

	if useDockerLocalstack { //localsatck is running in docker
		os.Setenv("LOCALSTACK_HOST", "localhost")
		os.Setenv("LOCALSTACK_PORT", "4566")
		err := localstacktest.CheckLocalStackHealth()
		if err != nil {
			errMsg := fmt.Sprintf("useDockerLocalstack is set to true, but localstack is not running. Please start localstack with `docker run -d --name localstacktestt --rm -it -p 4566:4566 localstack/localstack` or set useDockerLocalstack to false. Error: %s", err)
			panic(errMsg)
		}
		code := m.Run()
		os.Exit(code)
	}

	//start localstacktest automatically. Option for CI/CD
	if err := localstacktest.StartLocalStack(); err != nil {
		panic(err)
	}

	fmt.Printf("🧪🚀 Tests started: outbound_call\n")
	// run tests
	code := m.Run()
	fmt.Printf("🧪✅ Tests finished outbound_call\n")

	// Terminate LocalStack if this is the last package
	if err := localstacktest.StopLocalStack(); err != nil {
		log.Fatalf("Could not terminate LocalStack: %v", err)
	}
	os.Exit(code)

}

func TestExecute(t *testing.T) {

	awsBucketName := "testbucket"
	log := logrus.New()
	emptyRequest := events.APIGatewayProxyRequest{}

	//get localstack config from env variables
	customAWSConfig := localstacktest.GetCustomAWSConfig()

	//// Create the APIGatewayProxyRequest object
	//requestWState := events.APIGatewayProxyRequest{
	//	QueryStringParameters: map[string]string{
	//		"state": "someRandomString",
	//	},
	//	// Fill other necessary fields.
	//}

	type args struct {
		log             *logrus.Logger
		request         events.APIGatewayProxyRequest
		customAWSConfig *aws.Config
	}
	tests := []struct {
		name                         string
		args                         args
		setEnvironmentVariables      bool
		setVarsInLocalStack          bool
		expectedStatusCode           int
		createFileInS3BucketFileName map[string]string
		wantResponseApiGateway       events.APIGatewayProxyResponse
		expectedErrorContains        error
	}{
		{
			name: "error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is not set at all",
			args: args{
				log:             log,
				request:         emptyRequest,
				customAWSConfig: customAWSConfig,
			},
			setEnvironmentVariables: false,
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "failed to retrieve AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID from store: ParameterNotFound: Parameter /hotelito-app/dev/AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID not found.",
			},
		},

		{
			name: "error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is not set in env but set in store",
			args: args{
				log:             log,
				request:         emptyRequest,
				customAWSConfig: customAWSConfig,
			},
			setEnvironmentVariables: false,
			setVarsInLocalStack:     true,
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "failed to fetch object: Unable to download item \"config.json\", NoSuchBucket: The specified bucket does not exist\n\tstatus code: 404",
			},
		},

		{
			name: "error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is set",
			args: args{
				log:             log,
				request:         emptyRequest,
				customAWSConfig: customAWSConfig,
			},
			setEnvironmentVariables:      true,
			setVarsInLocalStack:          true,
			createFileInS3BucketFileName: map[string]string{awsBucketName: "config.json"},
			wantResponseApiGateway: events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "failed to fetch object: Unable to download item \"cloudbeds_api_params.json\", NoSuchBucket: The specified bucket does not exist\n\tstatus code: 404",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unset env variables
			os.Clearenv()
			localstacktest.ClearLocalstackAllServices(tt.args.customAWSConfig)

			if tt.setEnvironmentVariables {
				// Unset env variables after the "success" test case
				os.Setenv("CLOUDBEDS_CLIENT_ID", "test_client_id")
				os.Setenv("CLOUDBEDS_CLIENT_SECRET", "test_client_secret")
				os.Setenv("CLOUDBEDS_REDIRECT_URL", "test_redirect_url")
				os.Setenv("CLOUDBEDS_AUTH_URL", "test_auth_url")
				os.Setenv("CLOUDBEDS_TOKEN_URL", "test_token_url")
				os.Setenv("CLOUDBEDS_SCOPES", "test_scopes")
				os.Setenv("AWS_REGION", "us-east-1")
				os.Setenv("AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID", awsBucketName)
			}

			if tt.setVarsInLocalStack {
				mapOfValues := map[string]string{"AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID": "test_bucket"}
				localstacktest.SaveValuesToLocalStack(mapOfValues, tt.args.customAWSConfig)
			}
			if len(tt.createFileInS3BucketFileName) > 0 {
				localstacktest.CreateFileInS3(tt.args.customAWSConfig, awsBucketName, tt.createFileInS3BucketFileName[awsBucketName])
			}

			gotResponseApiGateway, _ := Execute(tt.args.log, tt.args.request, tt.args.customAWSConfig)

			assert.Equal(t, tt.wantResponseApiGateway.StatusCode, gotResponseApiGateway.StatusCode)
			assert.Contains(t, gotResponseApiGateway.Body, tt.wantResponseApiGateway.Body)

			// Unset env variables
			os.Clearenv()
		})
	}
}
