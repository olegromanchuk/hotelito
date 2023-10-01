package lambda_boilerplate

import (
	"github.com/olegromanchuk/hotelito/internal/logging"
	"github.com/sirupsen/logrus"
	"os"
)

// vars below are used ONLY if env vars are not set (testing only). It is not supposed to happen in production.
var (
	defaultAppName         = "hotelito-app"
	defaultEnvironmentType = "dev"
	defaultAwsRegion       = "us-east-2"
)

func InitializeVariablesFromEnv(log *logrus.Logger) (appName, environmentType, awsRegion string) {
	//get APP_NAME from env
	appName = os.Getenv("APPLICATION_NAME")
	if appName == "" {
		log.Debug("APPLICATION_NAME env variable is not set")
		appName = defaultAppName
	}
	log.Debugf("APPLICATION_NAME: %s", appName)

	environmentType = os.Getenv("ENVIRONMENT")
	if environmentType == "" {
		log.Debug("ENVIRONMENT env variable is not set")
		environmentType = defaultEnvironmentType
	}
	log.Debugf("ENVIRONMENT: %s", environmentType)

	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Debug("AWS_REGION env variable is not set")
		awsRegion = defaultAwsRegion
	}
	log.Debugf("AWS_REGION: %s", awsRegion)

	return appName, environmentType, awsRegion
}

func InitializeLogger() *logrus.Logger {
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

	//custom formatter will add caller name to the logging
	traceID := logging.GenerateTraceID()
	if logLevel >= 5 { //Debug or Trace level
		log.Formatter = &logging.CustomFormatter{CustomFormatter: &logrus.TextFormatter{}, TraceID: traceID}
	}

	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)
	log.Infof("Log level: %s", logLevel)

	return log
}
