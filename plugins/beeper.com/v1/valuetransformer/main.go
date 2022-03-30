package main

import (
	"errors"
	"fmt"
	"io"
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
	envDebug := strings.ToUpper(os.Getenv("VALUETRANSFORMER_DEBUG"))
	if len(envDebug) > 0 && (envDebug[0] == '1' || envDebug[0] == 'T') {
		DebugEnabled = true
		fmt.Fprintf(os.Stderr, "- WARNING - ValueTransformer debugging enabled - WARNING -\n")
	}

	rl := &ResourceList{}

	legacy := false
	var input []byte
	stdinDecoder := yaml.NewDecoder(os.Stdin)

	// check if we are called as a legacy alpha plugin
	if len(os.Args) > 1 {
		configFile, err := os.Open(os.Args[1])
		if err != nil {
			panic(err)
		}

		configDecoder := yaml.NewDecoder(configFile)
		if err := configDecoder.Decode(&rl.FunctionConfig); err != nil {
			panic(err)
		}

		for {
			item := make(map[string]interface{})
			if err := stdinDecoder.Decode(&item); err != nil {
				if err == io.EOF {
					break
				}
				panic(err)
			}

			rl.Items = append(rl.Items, item)
		}

		// enable legacy output
		legacy = true
	} else {
		if err := stdinDecoder.Decode(rl); err != nil {
			panic(err)
		}
	}

	if DebugEnabled {
		fmt.Fprintf(os.Stderr, "Input:\n%s\n", input)
	}

	if rl.FunctionConfig.Kind != "ValueTransformer" {
		panic(errors.New("unsupported Kind, expected ValueTransformer"))
	}

	if rl.FunctionConfig.ApiVersion != "beeper.com/v1" {
		panic(errors.New("unsupported apiVersion, expected beeper.com/v1"))
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

	encoder := yaml.NewEncoder(os.Stdout)
	if legacy {
		for _, item := range rl.Items {
			if err := encoder.Encode(item); err != nil {
				panic(err)
			}
		}
	} else {
		if err := encoder.Encode(rl); err != nil {
			panic(err)
		}
	}
	if err := encoder.Close(); err != nil {
		panic(err)
	}
}
