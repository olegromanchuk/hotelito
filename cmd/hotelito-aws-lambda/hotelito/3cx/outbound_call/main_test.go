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

var useDockerLocalstack = false //use docker for localstack for fast local debugging
// docker run -d --name localstacktestt --rm -it -p 4566:4566 localstack/localstack:1.4.0
//otherwise localstack is started automatically when tests run

func TestMain(m *testing.M) {

	if useDockerLocalstack { //localsatck is running in docker
		os.Setenv("LOCALSTACK_HOST", "localhost")
		os.Setenv("LOCALSTACK_PORT", "4566")
		err := localstacktest.CheckLocalStackHealth()
		if err != nil {
			errMsg := fmt.Sprintf("useDockerLocalstack is set to true, but localstack is not running. Please start localstack with `docker run -d --name localstacktest --rm -it -p 4566:4566 localstack/localstack` or set useDockerLocalstack to false. Error: %s", err)
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

	if useDockerLocalstack { //localsatck is running in docker
		os.Setenv("LOCALSTACK_HOST", "localhost")
		os.Setenv("LOCALSTACK_PORT", "4566")
	}

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
		createFileInS3BucketFileName []string
		wantResponseApiGateway       events.APIGatewayProxyResponse
		expectedErrorContains        error
	}{
		{
			name: "test1. Error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is not set at all",
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
			name: "test2. Error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is not set in env but set in store. NoSuchBucket",
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

		// Next tests fail because of some problems with testcontainers-localstack. If we run localstack in docker, then it works fine.
		// A problem arise when we try to create a bucket to save a file into S3. Instead of bucket testcontainers-localstack run s3.PutObject.

		//{
		//	name: "test3. Error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is set, but cloudbeds_api_params.json is not available in S3 bucket",
		//	args: args{
		//		log:             log,
		//		request:         emptyRequest,
		//		customAWSConfig: customAWSConfig,
		//	},
		//	setEnvironmentVariables:      false,
		//	setVarsInLocalStack:          true,
		//	createFileInS3BucketFileName: []string{"config.json"},
		//	wantResponseApiGateway: events.APIGatewayProxyResponse{
		//		StatusCode: 500,
		//		Body:       "failed to fetch object: Unable to download item \"cloudbeds_api_params.json\", NoSuchKey: The specified key does not exist.\n\tstatus code: 404",
		//	},
		//},

		//{
		//	name: "test4. Error: AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is set",
		//	args: args{
		//		log:             log,
		//		request:         emptyRequest,
		//		customAWSConfig: customAWSConfig,
		//	},
		//	setEnvironmentVariables:      true,
		//	setVarsInLocalStack:          true,
		//	createFileInS3BucketFileName: []string{"config.json", "cloudbeds_api_params.json"},
		//	wantResponseApiGateway: events.APIGatewayProxyResponse{
		//		StatusCode: 500,
		//		Body:       "failed to fetch object: Unable to download item \"cloudbeds_api_params.json\", NoSuchKey: The specified key does not exist.\n\tstatus code: 404",
		//	},
		//},
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
				mapOfValues := map[string]string{"AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID": awsBucketName}
				localstacktest.SaveValuesToLocalStack(mapOfValues, tt.args.customAWSConfig)
			}
			if len(tt.createFileInS3BucketFileName) > 0 {
				localstacktest.CreateFilesInS3(tt.args.customAWSConfig, awsBucketName, tt.createFileInS3BucketFileName)
			}

			gotResponseApiGateway, _ := Execute(tt.args.log, tt.args.request, tt.args.customAWSConfig)

			assert.Equal(t, tt.wantResponseApiGateway.StatusCode, gotResponseApiGateway.StatusCode)
			assert.Contains(t, gotResponseApiGateway.Body, tt.wantResponseApiGateway.Body)

			// Unset env variables
			os.Clearenv()
		})
	}
}

//func TestFetchS3ObjectAndSaveToFile(t *testing.T) {
//
//	if useDockerLocalstack { //localsatck is running in docker
//		os.Setenv("LOCALSTACK_HOST", "localhost")
//		os.Setenv("LOCALSTACK_PORT", "4566")
//	}
//
//	//get localstack config from env variables
//	customAWSConfig := localstacktest.GetCustomAWSConfig()
//
//	tests := []struct {
//		name            string
//		bucket          string
//		fileName        string
//		content         string
//		awsRegion       string
//		expectError     bool
//		expectedContent string
//	}{
//		{
//			name:            "Success",
//			bucket:          "test-bucket",
//			fileName:        "test-file.txt",
//			content:         "test content",
//			awsRegion:       "us-east-1",
//			expectError:     false,
//			expectedContent: "test content",
//		},
//		// Add more test cases as needed
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Setup
//			log := logrus.New()
//
//			// Connect to LocalStack S3
//			sess, err := session.NewSession(customAWSConfig)
//			require.NoError(t, err)
//
//			s3Client := s3.New(sess)
//
//			// Create a new bucket
//			_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
//				Bucket: aws.String(tt.bucket),
//			})
//			require.NoError(t, err)
//
//			// Upload a new object
//			uploader := s3manager.NewUploader(sess)
//			_, err = uploader.Upload(&s3manager.UploadInput{
//				Bucket: aws.String(tt.bucket),
//				Key:    aws.String(tt.fileName),
//				Body:   io.NopCloser(strings.NewReader(tt.content)),
//			})
//			require.NoError(t, err)
//
//			// Test
//			downloadedFileName, err := fetchS3ObjectAndSaveToFile(log, tt.bucket, tt.fileName, tt.awsRegion, customAWSConfig)
//			if tt.expectError {
//				require.Error(t, err)
//				return
//			}
//			require.NoError(t, err)
//			require.NotEmpty(t, downloadedFileName)
//
//			// Validate content
//			data, err := os.ReadFile(downloadedFileName)
//			require.NoError(t, err)
//			require.Equal(t, tt.expectedContent, string(data))
//
//			// Cleanup
//			err = os.Remove(downloadedFileName)
//			require.NoError(t, err)
//		})
//	}
//}
