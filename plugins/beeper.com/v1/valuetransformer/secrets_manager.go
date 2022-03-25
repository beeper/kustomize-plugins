package main

import (
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func convertSecretsManagerConfig(config *SourceConfig) (map[string]string, error) {
	out := make(map[string]string)
	sess := session.Must(session.NewSession())
	creds := stscreds.NewCredentials(sess, config.AwsRoleArn)
	sm := secretsmanager.New(sess, &aws.Config{Credentials: creds, Region: &config.AwsRegion})

	svi := secretsmanager.GetSecretValueInput{}
	svi.SecretId = &config.Path
	svo, err := sm.GetSecretValue(&svi)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal([]byte(*svo.SecretString), &out); err != nil {
		panic(err)
	}

	return out, nil
}
