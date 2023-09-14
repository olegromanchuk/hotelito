package awsstore

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestStoreAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name             string
		token            string
		mockReturnOutput *ssm.PutParameterOutput
		mockReturnErr    error
		expectedErr      error
	}{
		{
			name:             "Successful PutParameter call",
			token:            "testToken",
			mockReturnOutput: &ssm.PutParameterOutput{},
			mockReturnErr:    nil,
			expectedErr:      nil,
		},
		{
			name:             "PutParameter returns an error",
			token:            "testToken",
			mockReturnOutput: nil,
			mockReturnErr:    errors.New("PutParameter error"),
			expectedErr:      errors.New("PutParameter error"),
		},
		{
			name:             "Empty token",
			token:            "",
			mockReturnOutput: &ssm.PutParameterOutput{},
			mockReturnErr:    nil,
			expectedErr:      nil, // or an error, if you expect one for an empty token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := NewMockstorageManager(ctrl)

			// Setup your AWSSecretsStore
			store := &AWSSecretsStore{
				AccessTokenParamName: "testParam",
				SSM:                  mockSM,
			}

			expectedInput := &ssm.PutParameterInput{
				Name:      aws.String("testParam"),
				Overwrite: aws.Bool(true),
				Type:      aws.String("SecureString"),
				Value:     aws.String(tt.token),
			}

			mockSM.EXPECT().PutParameter(expectedInput).Return(tt.mockReturnOutput, tt.mockReturnErr)

			err := store.StoreAccessToken(tt.token)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestRetrieveAccessToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name             string
		mockReturnOutput *ssm.GetParameterOutput
		mockReturnErr    error
		expectedValue    string
		expectedErr      error
	}{
		{
			name:             "Successful GetParameter call",
			mockReturnOutput: &ssm.GetParameterOutput{Parameter: &ssm.Parameter{Value: aws.String("testValue")}},
			mockReturnErr:    nil,
			expectedValue:    "testValue",
			expectedErr:      nil,
		},
		{
			name:             "GetParameter returns an error",
			mockReturnOutput: nil,
			mockReturnErr:    errors.New("GetParameter error"),
			expectedValue:    "",
			expectedErr:      errors.New("GetParameter error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := NewMockstorageManager(ctrl)

			store := &AWSSecretsStore{
				AccessTokenParamName: "testParam",
				SSM:                  mockSM,
			}

			expectedInput := &ssm.GetParameterInput{
				Name:           aws.String("testParam"),
				WithDecryption: aws.Bool(true),
			}

			mockSM.EXPECT().GetParameter(expectedInput).Return(tt.mockReturnOutput, tt.mockReturnErr)

			value, err := store.RetrieveAccessToken()
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestStoreRefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		token         string
		mockReturnErr error
		expectedErr   error
	}{
		{
			name:          "Successful PutParameter call",
			token:         "refreshToken",
			mockReturnErr: nil,
			expectedErr:   nil,
		},
		{
			name:          "PutParameter returns an error",
			token:         "refreshToken",
			mockReturnErr: errors.New("PutParameter error"),
			expectedErr:   errors.New("PutParameter error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := NewMockstorageManager(ctrl)

			store := &AWSSecretsStore{
				RefreshTokenParamName: "refreshParam",
				SSM:                   mockSM,
			}

			expectedInput := &ssm.PutParameterInput{
				Name:      aws.String("refreshParam"),
				Overwrite: aws.Bool(true),
				Type:      aws.String("SecureString"),
				Value:     aws.String(tt.token),
			}

			mockSM.EXPECT().PutParameter(expectedInput).Return(nil, tt.mockReturnErr)

			err := store.StoreRefreshToken(tt.token)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestRetrieveRefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name            string
		mockReturnValue *ssm.GetParameterOutput
		mockReturnErr   error
		expectedValue   string
		expectedErr     error
	}{
		{
			name: "Successful GetParameter",
			mockReturnValue: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String("refreshToken"),
				},
			},
			mockReturnErr: nil,
			expectedValue: "refreshToken",
			expectedErr:   nil,
		},
		{
			name:            "GetParameter returns ParameterNotFound",
			mockReturnValue: nil,
			mockReturnErr:   errors.New("ParameterNotFound: "),
			expectedValue:   "",
			expectedErr:     nil,
		},
		{
			name:            "GetParameter returns generic error",
			mockReturnValue: nil,
			mockReturnErr:   errors.New("Some other error"),
			expectedValue:   "",
			expectedErr:     errors.New("Some other error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := NewMockstorageManager(ctrl)

			store := &AWSSecretsStore{
				RefreshTokenParamName: "refreshParam",
				SSM:                   mockSM,
			}

			expectedInput := &ssm.GetParameterInput{
				Name:           aws.String("refreshParam"),
				WithDecryption: aws.Bool(true),
			}

			mockSM.EXPECT().GetParameter(expectedInput).Return(tt.mockReturnValue, tt.mockReturnErr)

			value, err := store.RetrieveRefreshToken()

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestStoreOauthState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name          string
		mockReturnErr error
		expectedErr   error
	}{
		{
			name:          "Successful PutParameter",
			mockReturnErr: nil,
			expectedErr:   nil,
		},
		{
			name:          "Unsuccessful PutParameter",
			mockReturnErr: errors.New("Some error"),
			expectedErr:   errors.New("Some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := NewMockstorageManager(ctrl)
			logger := logrus.New()
			logger.SetOutput(io.Discard)
			logger.SetLevel(logrus.DebugLevel)
			hook := test.NewLocal(logger)

			store := &AWSSecretsStore{
				StorePrefix: "prefix",
				Log:         logger,
				SSM:         mockSM,
			}

			state := "testState"
			fullParamName := "/prefix/testState"

			expectedInput := &ssm.PutParameterInput{
				Name:      aws.String(fullParamName),
				Overwrite: aws.Bool(true),
				Type:      aws.String("String"),
				Value:     aws.String(state),
			}

			mockSM.EXPECT().PutParameter(expectedInput).Return(nil, tt.mockReturnErr)

			err := store.StoreOauthState(state)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
				assert.Len(t, hook.Entries, 2)
				assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)
				assert.Equal(t, logrus.ErrorLevel, hook.Entries[1].Level)
			} else {
				assert.Nil(t, err)
				assert.Len(t, hook.Entries, 1)
				assert.Equal(t, logrus.DebugLevel, hook.LastEntry().Level)
			}
		})
	}
}

func TestRetrieveOauthState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		getErr      error
		deleteErr   error
		paramValue  string
		expected    string
		expectedErr error
	}{
		{
			name:        "Successful Get and Delete",
			getErr:      nil,
			deleteErr:   nil,
			paramValue:  "testState",
			expected:    "testState",
			expectedErr: nil,
		},
		{
			name:        "Get Error",
			getErr:      errors.New("Get Error"),
			paramValue:  "",
			expected:    "",
			expectedErr: errors.New("Get Error"),
		},
		{
			name:        "Delete Error",
			getErr:      nil,
			deleteErr:   errors.New("Delete Error"),
			paramValue:  "testState",
			expected:    "",
			expectedErr: errors.New("Delete Error"),
		},
		{
			name:        "Quoted String",
			getErr:      nil,
			deleteErr:   nil,
			paramValue:  "\"testState\"",
			expected:    "testState",
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSM := NewMockstorageManager(ctrl)
			logger := logrus.New()
			logger.SetOutput(io.Discard)
			logger.SetLevel(logrus.TraceLevel)
			hook := test.NewLocal(logger)

			store := &AWSSecretsStore{
				StorePrefix: "prefix",
				Log:         logger,
				SSM:         mockSM,
			}

			state := tt.paramValue
			fullParamName := fmt.Sprintf("/prefix/%s", tt.paramValue)

			getInput := &ssm.GetParameterInput{
				Name:           aws.String(fullParamName),
				WithDecryption: aws.Bool(false),
			}

			getOutput := &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String(tt.paramValue),
				},
			}

			deleteInput := &ssm.DeleteParameterInput{
				Name: aws.String(fullParamName),
			}

			mockSM.EXPECT().GetParameter(getInput).Return(getOutput, tt.getErr)
			if tt.getErr == nil {
				mockSM.EXPECT().DeleteParameter(deleteInput).Return(nil, tt.deleteErr)
			}

			result, err := store.RetrieveOauthState(state)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expected, result)
			}

			assert.True(t, len(hook.Entries) > 0)
			assert.Equal(t, logrus.TraceLevel, hook.LastEntry().Level)
		})
	}
}

func TestRetrieveVar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSM := NewMockstorageManager(ctrl)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.TraceLevel)
	hook := test.NewLocal(logger)

	store := &AWSSecretsStore{
		StorePrefix: "testPrefix",
		SSM:         mockSM,
		Log:         logger,
	}

	testCases := []struct {
		desc             string
		varName          string
		mockReturnOutput *ssm.GetParameterOutput
		mockReturnErr    error
		expectedValue    string
		expectedErr      error
	}{
		{
			desc:    "Successful retrieval with quotes",
			varName: "testVar",
			mockReturnOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String("\"testValue\""),
				},
			},
			expectedValue: "testValue",
		},
		{
			desc:    "Successful retrieval without quotes",
			varName: "testVar",
			mockReturnOutput: &ssm.GetParameterOutput{
				Parameter: &ssm.Parameter{
					Value: aws.String("testValue"),
				},
			},
			expectedValue: "testValue",
		},
		{
			desc:          "Failure due to error",
			varName:       "testVar",
			mockReturnErr: errors.New("some error"),
			expectedErr:   errors.New("some error"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			expectedInput := &ssm.GetParameterInput{
				Name:           aws.String(fmt.Sprintf("/%s/%s", store.StorePrefix, tt.varName)),
				WithDecryption: aws.Bool(true),
			}

			mockSM.EXPECT().GetParameter(expectedInput).Return(tt.mockReturnOutput, tt.mockReturnErr)

			value, err := store.RetrieveVar(tt.varName)

			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedErr, err)
			assert.True(t, len(hook.Entries) > 0)
			assert.Equal(t, logrus.TraceLevel, hook.LastEntry().Level)
		})
	}
}

func TestInitialize(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	testCases := []struct {
		desc                  string
		storePrefix           string
		awsRegion             string
		AccessTokenParamName  string
		RefreshTokenParamName string
		mockSession           *session.Session
		mockErr               error
		expectErr             bool
	}{
		{
			desc:                  "Successful initialization",
			storePrefix:           "test",
			awsRegion:             "us-west-2",
			AccessTokenParamName:  "/test/access_token",
			RefreshTokenParamName: "/test/refresh_token",
			mockSession:           &session.Session{},
			expectErr:             false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.desc, func(t *testing.T) {
			// Mock session.NewSession function
			store, err := Initialize(logger, tt.storePrefix, tt.awsRegion)

			assert.Nil(t, err)
			assert.NotNil(t, store)
			assert.Equal(t, tt.storePrefix, store.StorePrefix)
			assert.Equal(t, tt.AccessTokenParamName, store.AccessTokenParamName)
			assert.Equal(t, tt.RefreshTokenParamName, store.RefreshTokenParamName)
			assert.Equal(t, tt.storePrefix, store.StorePrefix)
			assert.Equal(t, tt.awsRegion, *store.AWSSession.Config.Region)
		})
	}
}
