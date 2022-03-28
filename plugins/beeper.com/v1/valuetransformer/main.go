package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

func convertEnvironmentConfig(config *SourceConfig) (map[string]string, error) {
	out := make(map[string]string)

	for k, v := range config.Vars {
		if env, ok := os.LookupEnv(k); ok {
			out[v] = env
		}
	}

	return out, nil
}

func convertVariableConfig(config *SourceConfig) (map[string]string, error) {
	out := make(map[string]string)

	for k, v := range config.Vars {
		out[k] = v
	}

	return out, nil
}

var DebugEnabled bool

func main() {
	rl := &ResourceList{}
	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(input, rl); err != nil {
		panic(err)
	}

	if rl.FunctionConfig.Kind != "ValueTransformer" {
		panic(errors.New("unsupported Kind, expected ValueTransformer"))
	}

	if rl.FunctionConfig.ApiVersion != "beeper.com/v1" {
		panic(errors.New("unsupported apiVersion, expected beeper.com/v1"))
	}

	envDebug := strings.ToUpper(os.Getenv("VALUETRANSFORMER_DEBUG"))
	if len(envDebug) > 0 && (envDebug[0] == '1' || envDebug[0] == 'T') {
		DebugEnabled = true
		fmt.Fprintf(os.Stderr, "- WARNING - ValueTransformer debugging enabled - WARNING -\n")
		fmt.Fprintf(os.Stderr, "Input:\n%s\n", input)
	}

	// initialize all sources
	sources := make(SourceMap)
	for name, source := range rl.FunctionConfig.Sources {

		// allow overriding source config from env, could do dynamically with reflection?
		source.Path = os.ExpandEnv(source.Path)
		source.AwsRoleArn = os.ExpandEnv(source.AwsRoleArn)
		source.AwsRegion = os.ExpandEnv(source.AwsRegion)

		switch source.Type {
		case "File":
			sources[name], _ = convertFileConfig(&source)
		case "Environment":
			sources[name], _ = convertEnvironmentConfig(&source)
		case "Variable":
			sources[name], _ = convertVariableConfig(&source)
		case "SecretsManager":
			sources[name], _ = convertSecretsManagerConfig(&source)
		case "TerraformState":
			sources[name], _ = convertTerraformStateConfig(&source)
		default:
			panic(errors.New("Invalid source type " + source.Type))
		}

		if DebugEnabled {
			fmt.Fprintf(os.Stderr, "Source '%s':\n", name)
			for k, v := range sources[name] {
				fmt.Fprintf(os.Stderr, "\t%s (%d chars)\n", k, len(v))
			}
		}
	}

	newItems := make([]map[string]interface{}, len(rl.Items))
	for i, item := range rl.Items {
		newItems[i] = applyTransforms(item, &rl.FunctionConfig, sources)
	}
	rl.Items = newItems

	output, err := yaml.Marshal(rl)
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(output)
}
