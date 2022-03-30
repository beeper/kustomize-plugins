package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

func convertEnvironmentConfig(config *SourceConfig) map[string]string {
	out := make(map[string]string)

	for k, v := range config.Vars {
		if env, ok := os.LookupEnv(k); ok {
			out[v] = env
		}
	}

	return out
}

func convertVariableConfig(config *SourceConfig) map[string]string {
	out := make(map[string]string)

	for k, v := range config.Vars {
		out[k] = v
	}

	return out
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
	sources := make(map[string]map[string]string)
	for name, source := range rl.FunctionConfig.Sources {

		for k, v := range source.Args {
			source.Args[k] = expandEnvInterface(v)
		}

		switch source.Type {
		case "Variable":
			sources[name] = convertVariableConfig(&source)
		case "Environment":
			sources[name] = convertEnvironmentConfig(&source)
		case "File":
			sources[name] = filterMap(convertFileConfig(&source), source.Vars)
		case "Exec":
			sources[name] = filterMap(convertExecConfig(&source), source.Vars)
		case "SecretsManager":
			sources[name] = filterMap(convertSecretsManagerConfig(&source), source.Vars)
		case "TerraformState":
			sources[name] = filterMap(convertTerraformStateConfig(&source), source.Vars)
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
