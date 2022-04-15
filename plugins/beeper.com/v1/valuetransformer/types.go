package main

import (
	"regexp"
)

type Transform struct {
	regex  *regexp.Regexp
	source *map[string]string
}

type ResourceList struct {
	Kind           string                   `yaml:"kind"`
	Items          []map[string]interface{} `yaml:"items"`
	FunctionConfig TransformerConfig        `yaml:"functionConfig"`
}

type TransformTarget struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type TransformConfig struct {
	Source string          `yaml:"source"`
	Regex  string          `yaml:"regex"`
	Target TransformTarget `yaml:"target"`
}

type SourceConfig struct {
	Type string                 `yaml:"type"`
	Args map[string]interface{} `yaml:"args"`
	Vars map[string]interface{} `yaml:"vars"` // filter and remap source data
}

type TransformerConfig struct {
	ApiVersion string                  `yaml:"apiVersion"`
	Kind       string                  `yaml:"kind"`
	Sources    map[string]SourceConfig `yaml:"sources"`
	Merges     map[string]interface{}  `yaml:"merges"`
	Transforms []TransformConfig       `yaml:"transforms"`
}
