package config

import (
	"fmt"
	"os"
)

var (
	dynamoTableName string
	apiPort         string
	ginMode         string
)

func Init() error {
	var ok bool

	dynamoTableName, ok = os.LookupEnv("DYNAMO_TABLE_NAME")
	if !ok {
		return fmt.Errorf("DYNAMO_TABLE_NAME not set")
	}

	apiPort, ok = os.LookupEnv("API_PORT")
	if !ok {
		apiPort = ":8080"
	}
	if _, set := os.LookupEnv("RELEASE"); set {
		ginMode = "release"
	} else {
		ginMode = "debug"
	}
	return nil
}

func GinMode() string {
	return ginMode
}

func DynamoTableName() string {
	return dynamoTableName
}

func APIPort() string {
	return apiPort
}
