package localstacktest

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ssm"
	"log"
	"net"
	"os"
	"time"
)

func GetCustomAWSConfig() *aws.Config {
	localstack_host := os.Getenv("LOCALSTACK_HOST")
	localstack_port := os.Getenv("LOCALSTACK_PORT")
	if localstack_host == "" || localstack_port == "" {
		log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Error getting localstack host and port from env variables. Check localstacktest.go and TestMain()")
		return nil
	}

	customAWSConfig := &aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(fmt.Sprintf("http://%s:%s", localstack_host, localstack_port)),
		S3ForcePathStyle: aws.Bool(true),
		Credentials: credentials.NewStaticCredentials(
			"accessKeyID",
			"secretAccessKey",
			"token",
		)}
	return customAWSConfig
}

// CheckLocalStackHealth checks if localstack is running on LOCALSTACK_HOST:LOCALSTACK_PORT. Used only in case of useDockerLocalstack=true
func CheckLocalStackHealth() error {
	localstack_host := os.Getenv("LOCALSTACK_HOST")
	localstack_port := os.Getenv("LOCALSTACK_PORT")
	if localstack_host == "" || localstack_port == "" {
		errMsg := "💩🤷 Error getting localstack host and port from env variables. Check localstacktest.go:CheckLocalStackHealth and TestMain()"
		return errors.New(errMsg)
	}
	address := fmt.Sprintf("%s:%s", localstack_host, localstack_port)

	// Try to establish a TCP connection with a timeout
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		errDetailed := fmt.Errorf("LocalStack is not running on %s: %v\n", address, err)
		return errDetailed
	}
	_ = conn.Close()
	return nil
}

func SaveValuesToLocalStack(mapOfValues map[string]string, customAWSConfig *aws.Config) {

	//from lambda boilerplate
	//var (
	//	defaultAppName         = "hotelito-app"
	//	defaultEnvironmentType = "dev"
	//	defaultAwsRegion       = "us-east-2"
	//)

	prefix := "hotelito-app/dev/"
	// Initialize a session
	sess, err := session.NewSession(customAWSConfig)

	if err != nil {
		log.Fatalf("💩🤷 LOCALSTACK TEST ERROR:  Error creating session: %v", err)
		return
	}

	// Create SSM service client
	ssmSvc := ssm.New(sess)

	for k, d := range mapOfValues {
		paramName := prefix + k
		paramValue := d

		// Put the parameter
		putParamInput := &ssm.PutParameterInput{
			Name:      aws.String(paramName),
			Value:     aws.String(paramValue),
			Overwrite: aws.Bool(true), // Set to true to update existing parameter
			Type:      aws.String("String"),
		}

		_, err = ssmSvc.PutParameter(putParamInput)
		if err != nil {
			log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Error putting SSM parameter: %v", err)
			return
		}
		log.Printf("Successfully put SSM parameter %s with value %s", paramName, paramValue)
	}

}
func ClearLocalstackAllServices(awsCustomConfig *aws.Config) {
	ClearLocalstackSSMStore(awsCustomConfig)
	ClearLocalstackS3(awsCustomConfig)
}

func ClearLocalstackS3(awsCustomConfig *aws.Config) {
	sess, err := session.NewSession(awsCustomConfig)
	if err != nil {
		log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Failed to create session: %v", err)
	}

	s3Client := s3.New(sess)

	// List all buckets
	listBucketsOutput, _ := s3Client.ListBuckets(&s3.ListBucketsInput{})

	for _, bucket := range listBucketsOutput.Buckets {
		bucketName := aws.StringValue(bucket.Name)

		// List all objects in the bucket
		listObjectsInput := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
		}
		err = s3Client.ListObjectsV2Pages(listObjectsInput,
			func(page *s3.ListObjectsV2Output, lastPage bool) bool {
				for _, obj := range page.Contents {
					objKey := aws.StringValue(obj.Key)

					// Delete each object
					_, err := s3Client.DeleteObject(&s3.DeleteObjectInput{
						Bucket: aws.String(bucketName),
						Key:    aws.String(objKey),
					})
					if err != nil {
						log.Printf("Failed to delete object %s in bucket %s: %v", objKey, bucketName, err)
					} else {
						log.Printf("Deleted object %s in bucket %s", objKey, bucketName)
					}
				}
				return !lastPage
			})

		if err != nil {
			log.Printf("Failed to list objects for bucket %s: %v", bucketName, err)
			continue
		}

		// Delete the bucket
		_, err = s3Client.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			log.Printf("Failed to delete bucket %s: %v", bucketName, err)
		} else {
			log.Printf("Deleted bucket %s", bucketName)
		}
	}
}

func ClearLocalstackSSMStore(awsCustomConfig *aws.Config) {
	sess, err := session.NewSession(awsCustomConfig)
	if err != nil {
		log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Failed to create session: %v", err)
	}

	ssmClient := ssm.New(sess)

	input := &ssm.DescribeParametersInput{}
	var paramsToDelete []*string

	// Fetch all parameter names
	err = ssmClient.DescribeParametersPages(input,
		func(page *ssm.DescribeParametersOutput, lastPage bool) bool {
			for _, param := range page.Parameters {
				paramsToDelete = append(paramsToDelete, param.Name)
			}
			return !lastPage
		})

	if err != nil {
		log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Failed to describe parameters: %v", err)
	}

	// Delete all parameters
	for _, paramName := range paramsToDelete {
		_, err := ssmClient.DeleteParameter(&ssm.DeleteParameterInput{
			Name: paramName,
		})
		if err != nil {
			log.Printf("Failed to delete parameter %s: %v", *paramName, err)
		} else {
			log.Printf("Deleted parameter %s", *paramName)
		}
	}
}

func CreateFilesInS3(awsCustomConfig *aws.Config, bucketName string, fileNames []string) {
	sess, err := session.NewSession(awsCustomConfig)

	if err != nil {
		log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Failed to create session: %v", err)
	}

	// Create S3 service client
	s3Client := s3.New(sess)

	// Define bucket and file name

	// Check if bucket exists
	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err = s3Client.HeadBucket(input)
	if err != nil {
		// Create bucket if it doesn't exist
		_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Failed to create bucket: %v", err)
		}
		log.Printf("Bucket %s created.", bucketName)
	}

	for _, fileName := range fileNames {

		// Create empty JSON file and upload
		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Body:   aws.ReadSeekCloser(bytes.NewReader([]byte("{}"))),
			Bucket: aws.String(bucketName),
			Key:    aws.String(fileName),
		})
		if err != nil {
			log.Fatalf("💩🤷 LOCALSTACK TEST ERROR: Failed to upload file: %v", err)
		} else {
			log.Printf("Successfully created empty file %s in bucket %s", fileName, bucketName)
		}
	}
}

func ClearEnvVars() {
	vars := []string{
		"CLOUDBEDS_CLIENT_ID",
		"CLOUDBEDS_CLIENT_SECRET",
		"CLOUDBEDS_REDIRECT_URL",
		"CLOUDBEDS_AUTH_URL",
		"CLOUDBEDS_TOKEN_URL",
		"CLOUDBEDS_SCOPES",
		"AWS_REGION",
		"AWS_S3_BUCKET_4_MAP_3CXROOMEXT_CLBEDSROOMID",
	}

	for _, v := range vars {
		os.Unsetenv(v)
	}
}
