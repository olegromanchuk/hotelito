package awsstore

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type AWSSecretsStore struct {
	AccessTokenParamName  string
	RefreshTokenParamName string
	AWSSession            *session.Session
	StorePrefix           string
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

	//get full name including app name and environment type
	fullParamName := fmt.Sprintf("/%s/%s", s.StorePrefix, state)

	input := &ssm.PutParameterInput{
		Name:      aws.String(fullParamName),
		Overwrite: aws.Bool(true),
		Type:      aws.String("String"),
		Value:     aws.String(state),
	}

	_, err := ssmSvc.PutParameter(input)
	return err
}

func (s *AWSSecretsStore) RetrieveOauthState(state string) (string, error) {
	ssmSvc := ssm.New(s.AWSSession)

	//get full name including app name and environment type
	fullParamName := fmt.Sprintf("/%s/%s", s.StorePrefix, state)

	input := &ssm.GetParameterInput{
		Name:           aws.String(fullParamName),
		WithDecryption: aws.Bool(false),
	}

	result, err := ssmSvc.GetParameter(input)
	if err != nil {
		return "", err
	}

	value := *result.Parameter.Value

	//clean up. Delete retrieved state
	delInput := &ssm.DeleteParameterInput{
		Name: aws.String(state),
	}

	_, err = ssmSvc.DeleteParameter(delInput)
	if err != nil {
		return "", err
	}

	return value, nil
}

func Initialize(storePrefix string, awsRegion string) (*AWSSecretsStore, error) {
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
		StorePrefix:           storePrefix}, nil
}

func (s *AWSSecretsStore) Close() error {
	s.AWSSession = nil
	return nil
}
