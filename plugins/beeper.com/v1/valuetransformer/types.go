package main

import (
	"regexp"
)

type SourceMap map[string]map[string]string

type Transform struct {
	regex  *regexp.Regexp
	source *map[string]string
}

func getString(r map[string]interface{}, key string) string {
	i := r[key]

	switch v := i.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func getMap(r map[string]interface{}, key string) map[string]interface{} {
	i := r[key]
	switch c := i.(type) {
	case map[interface{}]interface{}:
		nr := make(map[string]interface{})
		for k, v := range c {
			switch kc := k.(type) {
			case string:
				nr[kc] = v
			}
		}
		r[key] = nr
		return nr
	default:
		return make(map[string]interface{})
	}
}

type ResourceList struct {
	Kind           string                    `yaml:"kind"`
	Items          []map[string]interface{}  `yaml:"items"`
	FunctionConfig TransformerConfig         `yaml:"functionConfig"`
}

type TransformTarget struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type TransformConfig struct {
	Source string          `yaml:"source"`
	Regex  string          `yaml:"regex"`
	Target TransformTarget `yaml:"targets"`
}

type SourceConfig struct {
	Type string            `yaml:"type"`
	Path string            `yaml:"path"` // any kind of name/path of the resource
	Vars map[string]string `yaml:"vars"` // filter and remap source data

	// if source uses AWS SDK these are overrides for env
	AwsRoleArn string `yaml:"awsRoleArn"`
	AwsRegion  string `yaml:"awsRegion"`
}

type TransformerConfig struct {
	ApiVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	Sources    map[string]SourceConfig `yaml:"sources"`
	Transforms []TransformConfig       `yaml:"transforms"`
}
