package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/yaml.v2"
)

func readFile(config *SourceConfig) ([]byte, error) {
	u, err := url.Parse(config.Path)

	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "s3":
		sess := session.Must(session.NewSession())
		creds := stscreds.NewCredentials(sess, config.AwsRoleArn)

		bucket := s3.New(sess, &aws.Config{Credentials: creds, Region: &config.AwsRegion})
		goi := s3.GetObjectInput{}
		goi.Bucket = &u.Host
		goi.Key = &u.Path

		goo, err := bucket.GetObject(&goi)
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(goo.Body)
		if err != nil {
			return nil, err
		}

		return data, nil
	case "":
		return os.ReadFile(config.Path)
	default:
		return nil, errors.New("unsupported URL scheme: " + u.Scheme)
	}
}

func convertFileConfig(config *SourceConfig) (map[string]string, error) {
	data, err := readFile(config)
	if err != nil {
		panic(err)
	}

	var raw map[string]interface{}
	switch {
	case strings.HasSuffix(config.Path, ".yml"), strings.HasSuffix(config.Path, ".yaml"):
		if err := yaml.Unmarshal(data, &raw); err != nil {
			panic(err)
		}
	case strings.HasSuffix(config.Path, ".json"):
		if err := json.Unmarshal(data, &raw); err != nil {
			panic(err)
		}
	default:
		panic(errors.New("unsupported variable file type"))
	}

	flat := make(map[string]string)
	flattenToMap(raw, "", flat)
	out := flat

	// filter/rebuild out map if we have vars set
	if len(config.Vars) > 0 {
		out = make(map[string]string)
		for k, v := range config.Vars {
			if ov, ok := flat[k]; ok {
				out[v] = ov
			}
		}
	}

	return out, nil
}
