package awsstore

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/sirupsen/logrus"
	"strings"
)

type AWSSecretsStore struct {
	AccessTokenParamName  string
	RefreshTokenParamName string
	AWSSession            *session.Session
	StorePrefix           string
	Log                   *logrus.Logger
}

func (s *AWSSecretsStore) StoreAccessToken(token string) error {
	ssmSvc := ssm.New(s.AWSSession)

	input := &ssm.PutParameterInput{
		Name:      aws.String(s.AccessTokenParamName),
		Overwrite: aws.Bool(true),
		Type:      aws.String("SecureString"),
		Value:     aws.String(token),
	}

	_, err := ssmSvc.PutParameter(input)
	return err
}

func (s *AWSSecretsStore) RetrieveAccessToken() (string, error) {
	ssmSvc := ssm.New(s.AWSSession)

	input := &ssm.GetParameterInput{
		Name:           aws.String(s.AccessTokenParamName),
		WithDecryption: aws.Bool(true),
	}

	result, err := ssmSvc.GetParameter(input)
	if err != nil {
		return "", err
	}

	return *result.Parameter.Value, nil
}

func (s *AWSSecretsStore) StoreRefreshToken(token string) error {
	ssmSvc := ssm.New(s.AWSSession)

	input := &ssm.PutParameterInput{
		Name:      aws.String(s.RefreshTokenParamName),
		Overwrite: aws.Bool(true),
		Type:      aws.String("SecureString"),
		Value:     aws.String(token),
	}

	_, err := ssmSvc.PutParameter(input)
	return err
}

func (s *AWSSecretsStore) RetrieveRefreshToken() (string, error) {
	ssmSvc := ssm.New(s.AWSSession)

	input := &ssm.GetParameterInput{
		Name:           aws.String(s.RefreshTokenParamName),
		WithDecryption: aws.Bool(true),
	}

	result, err := ssmSvc.GetParameter(input)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "ParameterNotFound: " {
			return "", nil
		}
		return "", err
	}

	return *result.Parameter.Value, nil
}

func (s *AWSSecretsStore) StoreOauthState(state string) error {
	ssmSvc := ssm.New(s.AWSSession)
	s.Log.Debugf("Storing state %s", state)
	//get full name including app name and environment type
	fullParamName := fmt.Sprintf("/%s/%s", s.StorePrefix, state)

	input := &ssm.PutParameterInput{
		Name:      aws.String(fullParamName),
		Overwrite: aws.Bool(true),
		Type:      aws.String("String"),
		Value:     aws.String(state),
	}

	_, err := ssmSvc.PutParameter(input)
	if err != nil {
		s.Log.Errorf("Error storing state %s", err.Error())
	}
	return err
}

func (s *AWSSecretsStore) RetrieveOauthState(state string) (string, error) {
	ssmSvc := ssm.New(s.AWSSession)

	//get full name including app name and environment type
	fullParamName := fmt.Sprintf("/%s/%s", s.StorePrefix, state)

	s.Log.Tracef("Retrieving state %s", fullParamName)
	input := &ssm.GetParameterInput{
		Name:           aws.String(fullParamName),
		WithDecryption: aws.Bool(false),
	}

	result, err := ssmSvc.GetParameter(input)
	if err != nil {
		return "", err
	}

	//remove quotes
	resultRaw := *result.Parameter.Value
	resultString := resultRaw
	//check if string is quoted and strip if yes
	if strings.HasPrefix(resultRaw, "\"") {
		resultString = strings.Trim(resultRaw, "\"")
		s.Log.Tracef("Retrieved %s", resultRaw)
		return resultString, nil
	}

	//clean up. Delete retrieved state
	s.Log.Tracef("Deleting state %s", fullParamName)
	delInput := &ssm.DeleteParameterInput{
		Name: aws.String(fullParamName),
	}

	_, err = ssmSvc.DeleteParameter(delInput)
	if err != nil {
		return "", err
	}

	return resultString, nil
}

func (s *AWSSecretsStore) RetrieveVar(varName string) (varValue string, err error) {
	ssmSvc := ssm.New(s.AWSSession)

	//get full name including app name and environment type
	fullParamName := fmt.Sprintf("/%s/%s", s.StorePrefix, varName)

	input := &ssm.GetParameterInput{
		Name:           aws.String(fullParamName),
		WithDecryption: aws.Bool(true),
	}

	result, err := ssmSvc.GetParameter(input)
	if err != nil {
		return "", err
	}

	//remove quotes
	var resultString string
	resultRaw := *result.Parameter.Value
	//check if string is quoted and strip if yes
	if strings.HasPrefix(resultRaw, "\"") {
		resultString = strings.Trim(resultRaw, "\"")
		s.Log.Tracef("Retrieved %s", resultString)
		return resultString, nil
	}
	resultString = resultRaw

	s.Log.Tracef("Retrieved %s", resultString)
	return resultString, nil
}

func Initialize(log *logrus.Logger, storePrefix string, awsRegion string) (*AWSSecretsStore, error) {
	accessTokenParamName := fmt.Sprintf("/%s/access_token", storePrefix)
	refreshTokenParamName := fmt.Sprintf("/%s/refresh_token", storePrefix)

	// Initialize a session that the SDK uses to load
	// credentials from the shared credentials file. (~/.aws/credentials).
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)
	if err != nil {
		return nil, err
	}

	return &AWSSecretsStore{AccessTokenParamName: accessTokenParamName,
		RefreshTokenParamName: refreshTokenParamName,
		AWSSession:            sess,
		StorePrefix:           storePrefix,
		Log:                   log,
	}, nil
}

func (s *AWSSecretsStore) Close() error {
	s.AWSSession = nil
	return nil
}
