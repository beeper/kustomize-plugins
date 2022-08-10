package main

import (
	"errors"
	"github.com/minio/pkg/wildcard"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
)

type ResourceList struct {
	Kind           string            `yaml:"kind"`
	Items          []map[string]any  `yaml:"items"`
	FunctionConfig TransformerConfig `yaml:"functionConfig"`
}

type Match struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type TransformerConfig struct {
	ApiVersion string  `yaml:"apiVersion"`
	Kind       string  `yaml:"kind"`
	Excludes   []Match `yaml:"excludes"`
	Includes   []Match `yaml:"includes"`
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

func doesMatch(matcher Match, kind string, namespace string, name string) bool {
	if matcher.Kind != "" && kind != matcher.Kind {
		return false
	}

	if matcher.Namespace != "" && namespace != matcher.Namespace {
		return false
	}

	if matcher.Name != "" && !wildcard.Match(strings.ToLower(matcher.Name), strings.ToLower(name)) {
		return false
	}

	return true
}

func main() {
	rl := &ResourceList{}

	stdinDecoder := yaml.NewDecoder(os.Stdin)
	if err := stdinDecoder.Decode(rl); err != nil {
		panic(err)
	}

	if rl.FunctionConfig.Kind != "FilterTransformer" {
		panic(errors.New("unsupported Kind, expected FilterTransformer"))
	}

	if rl.FunctionConfig.ApiVersion != "beeper.com/v1" {
		panic(errors.New("unsupported apiVersion, expected beeper.com/v1"))
	}

	var filteredItems []map[string]any

	for _, res := range rl.Items {
		kind := getString(res, "kind")
		metadata := getMap(res, "metadata")
		name := getString(metadata, "name")
		namespace := getString(metadata, "namespace")

		didMatch := false

		for _, include := range rl.FunctionConfig.Includes {
			if doesMatch(include, kind, namespace, name) {
				didMatch = true
			}
		}

		// Excludes override includes
		for _, exclude := range rl.FunctionConfig.Excludes {
			if doesMatch(exclude, kind, namespace, name) {
				didMatch = false
			}
		}

		if didMatch {
			filteredItems = append(filteredItems, res)
		}
	}

	rl.Items = filteredItems

	encoder := yaml.NewEncoder(os.Stdout)
	if err := encoder.Encode(rl); err != nil {
		panic(err)
	}
	if err := encoder.Close(); err != nil {
		panic(err)
	}
}
