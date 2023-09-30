package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/olegromanchuk/hotelito/cmd/hotelito-aws-lambda/hotelito/lambda_boilerplate"
	"github.com/olegromanchuk/hotelito/internal/configuration"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/awsstore"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleProcessOutboundCall(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	fmt.Printf("[%s] Started HandleProcessOutboundCall", time.Now().String())

	log := lambda_boilerplate.InitializeLogger()
	log.Debug(request)

	responseApiGateway, err := Execute(log, request, nil)
	if err != nil {
		log.Errorf("Error executing handler: %v", err)
		responseApiGateway.StatusCode = http.StatusOK //we need to reply with dignity: 200 to cloudbeds
		responseApiGateway.Body = fmt.Sprintf("Error: %v", err)
	}

	return responseApiGateway, nil
}

func Execute(log *logrus.Logger, request events.APIGatewayProxyRequest, customAWSConfig *aws.Config) (responseApiGateway events.APIGatewayProxyResponse, returnError error) {
	appName, environmentType, awsRegion := lambda_boilerplate.InitializeVariablesFromEnv(log)
	storePrefix := fmt.Sprintf("%s/%s", appName, environmentType) //hotelito-app-production

	//current secret store - aws env variables
	storeClient, err := awsstore.Initialize(log, storePrefix, awsRegion, customAWSConfig)
	if err != nil {
		return responseApiGateway, err
	}

	awsBucketName := os.Getenv("AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID")
	if awsBucketName == "" {
		//get from awsstore if localenv is empty
		log.Debug("AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID env variable is not set. Trying store")
		awsBucketName, err = storeClient.RetrieveVar("AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID")
		if err != nil {
			errMsg := fmt.Sprintf("failed to retrieve AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID from store: %v", err)
			log.Error(errMsg)
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       errMsg,
			}, nil
		}
	}
	log.Debugf("AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID: %s", awsBucketName)

	log.Debugf("Fetching config.json from S3 bucket %s", awsBucketName)
	//get information about mapping: room extension -- cloudbeds room ID
	//fetchS3ObjectAndSaveToFile is a helper function to fetch object from S3 and save it to file
	mapFullFileName, err := fetchS3ObjectAndSaveToFile(log, awsBucketName, "config.json", awsRegion, customAWSConfig) // Replace with your bucket name and the file name
	if err != nil || mapFullFileName == "" {
		errMsg := fmt.Sprintf("failed to fetch object: %v. Check if AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is set and S3 bucket with config.json exists", err)
		log.Error(errMsg)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       errMsg,
		}, nil
	}

	log.Debugf("Fetching cloudbeds_api_params.json from S3 bucket %s", awsBucketName)
	//get information about mapping: room extension -- cloudbeds room ID
	//fetchS3ObjectAndSaveToFile is a helper function to fetch object from S3 and save it to file
	clBedsApiConfigFile, err := fetchS3ObjectAndSaveToFile(log, awsBucketName, "cloudbeds_api_params.json", awsRegion, customAWSConfig) // Replace with your bucket name and the file name
	if err != nil || mapFullFileName == "" {
		errMsg := fmt.Sprintf("failed to fetch object: %v. Check if AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID is set and S3 bucket with cloudbeds_api_params.json exists", err)
		log.Error(errMsg)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       errMsg,
		}, nil
	}

	//parse config.json
	configMap, err := configuration.New(log, mapFullFileName, clBedsApiConfigFile)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       err.Error(),
		}, nil
	}

	//create cloudbeds client
	clbClient, err := cloudbeds.New(log, storeClient, configMap)
	if err != nil {
		log.Errorf("Error creating cloudbeds client: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}

	//option via handler interface. Helpful for testing
	//create 3cx client
	pbx3cxClient := pbx3cx.New(log, configMap)
	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)

	body := request.Body
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			log.Errorf("Error decoding base64 string: %v", err)
		}
		body = string(decoded)
	}

	log.Debugf("Request body: %s", body)
	decoder := json.NewDecoder(strings.NewReader(body))
	room, err := h.PBX.ProcessPBXRequest(decoder)
	if err != nil {
		if err.Error() == "incoming-call-ignoring" { //ignore incoming calls. Specific of 3CX. 3CX sends 2 request for each call: incoming(through loopback) and outgoing
			h.Log.Debugf("Ignoring incoming call")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
		}
		if err.Error() == "outgoing-regular-call-ignoring" { //regular outbound call is not related to room status
			h.Log.Debugf("Ignoring regular outgoing call")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
		}
		h.Log.Error(err)
		log.Error(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}
	if room.PhoneNumber == "" {
		h.Log.Error("Room phone number is empty")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}
	h.Log.Debugf("Room phone number: %s", room.PhoneNumber)

	//get provider
	hotelProvider := h.Hotel

	msg, err := hotelProvider.UpdateRoom(room.PhoneNumber, room.RoomCondition, room.HousekeeperName)
	if err != nil {
		h.Log.Error(err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error: %v", err),
		}, nil
	}
	h.Log.Debugf("Message from provider: %s", msg)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       msg,
	}, nil

}

// fetchS3ObjectAndSaveToFile is a helper function to fetch object from S3 and save it to file
func fetchS3ObjectAndSaveToFile(log *logrus.Logger, bucket, fileName string, awsRegion string, customAWSConfig *aws.Config) (filename string, err error) {

	awsConfig := awsstore.PrepareAWSConfig(awsRegion, customAWSConfig)
	fullFileName := fmt.Sprintf("/tmp/%s", fileName)
	sess, err := session.NewSession(awsConfig)

	if err != nil {
		return "", err
	}

	downloader := s3manager.NewDownloader(sess)
	log.Tracef("Downloading %s from bucket %s", fileName, bucket)
	file, err := os.Create(fullFileName) //save file to current directory. Exists only for current lambda execution
	if err != nil {
		errMsg := fmt.Sprintf("Unable to open file %q for writing - %v", fileName, err)
		log.Error(errMsg)
		return "", errors.New(errMsg)
	}

	defer file.Close()

	bytesDownloaded, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(fileName),
		})
	if err != nil {
		errMsg := fmt.Sprintf("Unable to download item %q, %v", fileName, err)
		log.Error(errMsg)
		return "", errors.New(errMsg)
	}
	log.Tracef("Stored to %s from bucket %s, %d bytes", fullFileName, bucket, bytesDownloaded)
	return fullFileName, nil
}

func main() {
	lambda.Start(HandleProcessOutboundCall)
}
