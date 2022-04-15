package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

func convertEnvironmentConfig(config *SourceConfig, filter map[string]string) map[string]string {
	out := make(map[string]string)

	for k, v := range filter {
		if env, ok := os.LookupEnv(k); ok {
			out[v] = env
		}
	}

	return out
}

var DebugEnabled bool
var mergeSplit *regexp.Regexp = regexp.MustCompile(`^([^\.]+)\.(.+)$`)

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
	var wg sync.WaitGroup
	sources := make(map[string]map[string]string)
	for name, source := range rl.FunctionConfig.Sources {
		wg.Add(1)

		go func(name string, source SourceConfig) {
			for k, v := range source.Args {
				source.Args[k] = expandEnvInterface(v)
			}

			flatVars := make(map[string]string)
			flattenToMap(source.Vars, "", flatVars)

			switch source.Type {
			case "Variable":
				sources[name] = flatVars
			case "Environment":
				sources[name] = convertEnvironmentConfig(&source, flatVars)
			case "File":
				sources[name] = filterMap(convertFileConfig(&source), flatVars)
			case "Exec":
				sources[name] = filterMap(convertExecConfig(&source), flatVars)
			case "SecretsManager":
				sources[name] = filterMap(convertSecretsManagerConfig(&source), flatVars)
			case "TerraformState":
				sources[name] = filterMap(convertTerraformStateConfig(&source), flatVars)
			default:
				panic(errors.New("Invalid source type " + source.Type))
			}

			if DebugEnabled {
				fmt.Fprintf(os.Stderr, "Source '%s':\n", name)
				for k, v := range sources[name] {
					fmt.Fprintf(os.Stderr, "\t%s (%d chars)\n", k, len(v))
				}
			}

			wg.Done()
		}(name, source)
	}

	wg.Wait()

	// apply source merges
	for name, merge := range rl.FunctionConfig.Merges {
		if _, found := sources[name]; found {
			panic(fmt.Errorf("merge '%s' is already a source", name))
		}

		flatMerge := make(map[string]string)
		flattenToMap(merge, "", flatMerge)

		if DebugEnabled {
			fmt.Fprintf(os.Stderr, "Merge '%s':\n", name)
		}

		for k, v := range flatMerge {
			split := mergeSplit.FindStringSubmatch(v)
			if len(split) < 3 {
				panic(fmt.Errorf("merge value '%s' was not a reference to a source", v))
			}

			sourceName := split[1]
			sourceKey := split[2]

			if source, ok := sources[sourceName]; ok {
				if value, ok := source[sourceKey]; ok {
					flatMerge[k] = value
				} else {
					panic(fmt.Errorf("merge key '%s' was not found from source '%s'", sourceKey, sourceName))
				}
			} else {
				panic(fmt.Errorf("merge source '%s' was not found", sourceName))
			}

			if DebugEnabled {
				fmt.Fprintf(os.Stderr, "\t%s (%d chars)\n", k, len(flatMerge[k]))
			}
		}

		sources[name] = flatMerge
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
