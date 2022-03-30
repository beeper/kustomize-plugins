package main

import (
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func convertSecretsManagerConfig(config *SourceConfig) map[string]string {
	name := getString(config.Args, "name")
	if len(name) == 0 {
		panic(errors.New("no secret name given"))
	}

	awsRoleArn := getString(config.Args, "AwsRoleArn")
	awsRegion := getString(config.Args, "AwsRegion")

	sess := session.Must(session.NewSession())
	creds := stscreds.NewCredentials(sess, awsRoleArn)
	sm := secretsmanager.New(sess, &aws.Config{Credentials: creds, Region: &awsRegion})

	svi := secretsmanager.GetSecretValueInput{}
	svi.SecretId = &name
	svo, err := sm.GetSecretValue(&svi)
	if err != nil {
		panic(err)
	}

	out := make(map[string]string)

	if err := json.Unmarshal([]byte(*svo.SecretString), &out); err != nil {
		panic(err)
	}

	return out
}
