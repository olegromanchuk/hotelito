package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/olegromanchuk/hotelito/internal/handlers"
	"github.com/olegromanchuk/hotelito/pkg/hotel/cloudbeds"
	"github.com/olegromanchuk/hotelito/pkg/pbx/pbx3cx"
	"github.com/olegromanchuk/hotelito/pkg/secrets/awsstore"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleProcessOutboundCall(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Print(request)
	//define logger
	log := logrus.New()
	// The default level is debug.
	logLevelEnv := os.Getenv("LOG_LEVEL")
	if logLevelEnv == "" {
		logLevelEnv = "debug"
	}
	logLevel, err := logrus.ParseLevel(logLevelEnv)
	if err != nil {
		logLevel = logrus.DebugLevel
	}
	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)
	log.Infof("Log level: %s", logLevelEnv)

	//get APP_NAME from env
	appName := os.Getenv("APPLICATION_NAME")
	if appName == "" {
		log.Debug("APPLICATION_NAME env variable is not set")
		appName = "hotelito-app"
	}
	log.Debugf("APPLICATION_NAME: %s", appName)

	environmentType := os.Getenv("ENVIRONMENT")
	if environmentType == "" {
		log.Debug("ENVIRONMENT env variable is not set")
		environmentType = "dev"
	}
	log.Debugf("ENVIRONMENT: %s", environmentType)

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Debug("AWS_REGION env variable is not set")
		awsRegion = "us-east-2"
	}
	log.Debugf("AWS_REGION: %s", awsRegion)

	storePrefix := fmt.Sprintf("%s/%s", appName, environmentType) //hotelito-app-production
	//current secret store - aws env variables
	storeClient, err := awsstore.Initialize(storePrefix, awsRegion)
	if err != nil {
		log.Fatal(err)
	}

	//create cloudbeds client
	clbClient := cloudbeds.NewClient4CallbackAndInit(log, storeClient)

	//option via handler interface. Helpful for testing
	//create 3cx client
	pbx3cxClient := pbx3cx.New(log)
	//define handlers
	h := handlers.NewHandler(log, pbx3cxClient, clbClient)

	body := request.Body
	if request.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			// handle err
		}
		body = string(decoded)
	}

	log.Debugf("Request body: %s", body)
	decoder := json.NewDecoder(strings.NewReader(body))
	room, err := h.PBX.ProcessPBXRequest(decoder)
	if err != nil {
		if err.Error() == "incoming-call-ignoring" { //ignore incoming calls. Specific of 3CX. 3CX sends 2 request for each call: incoming(through loopback) and outgoing
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

	//get information about mapping: room extension -- cloudbeds room ID
	//fetchS3ObjectAndSaveToFile is a helper function to fetch object from S3 and save it to file
	err = fetchS3ObjectAndSaveToFile(os.Getenv("AWS_BUCKET_MAP_3CXROOMEXTENSION_CLOUDBEDSROOMID"), "roomid_map.json") // Replace with your bucket name and the file name
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch object: %v. Check if AWS_BUCKET_MAP_3CXROOMEXTENSION_CLOUDBEDSROOMID is set and S3 bucket with roomid_map.json exists", err)
		log.Debug(errMsg)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       errMsg,
		}, nil
	}

	msg, err := hotelProvider.UpdateRoom(room.PhoneNumber, room.RoomCondition, room.HouskeeperID, "roomid_map.json")
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
func fetchS3ObjectAndSaveToFile(bucket, fileName string) error {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION"))},
	)

	if err != nil {
		return err
	}

	downloader := s3manager.NewDownloader(sess)

	file, err := os.Create("roomid_map.json") //save file to current directory. Exists only for current lambda execution
	if err != nil {
		return fmt.Errorf("Unable to open empty file %q for writing - %v", fileName, err)
	}

	defer file.Close()

	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(fileName),
		})
	if err != nil {
		return fmt.Errorf("Unable to download and save the file %q from bucket %q - %v", fileName, bucket, err)
	}
	return nil
}

func main() {
	lambda.Start(HandleProcessOutboundCall)
}
