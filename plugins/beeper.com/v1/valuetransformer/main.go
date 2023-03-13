package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
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

func resolveIncludes(config *TransformerConfig) {
	includes := map[string]struct{}{}
	nincludes := -1

	// include loop is going to be run until we don't include any new files anymore
	for len(includes) > nincludes {
		nincludes = len(includes)

		for _, includeFile := range config.Includes {
			includeFile = expandEnvInterface(includeFile).(string)

			// no re-including files that we already have
			if _, ok := includes[includeFile]; ok {
				continue
			}

			includes[includeFile] = struct{}{}

			if DebugEnabled {
				fmt.Fprintf(os.Stderr, "Including file: %s\n", includeFile)
			}

			includeConfig := TransformerConfig{}
			readYamlFile(includeFile, &includeConfig)
			mergeConfig(config, &includeConfig)
		}
	}
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
	stdinDecoder := yaml.NewDecoder(os.Stdin)

	// check if we are called as a legacy alpha plugin
	if len(os.Args) > 1 {
		readYamlFile(os.Args[1], &rl.FunctionConfig)

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

	resolveIncludes(&rl.FunctionConfig)

	if rl.FunctionConfig.Kind != "ValueTransformer" {
		panic(errors.New("unsupported Kind, expected ValueTransformer"))
	}

	if rl.FunctionConfig.ApiVersion != "beeper.com/v1" {
		panic(errors.New("unsupported apiVersion, expected beeper.com/v1"))
	}

	// initialize all sources
	var wg sync.WaitGroup
	sources := make(map[string]map[string]string)
	var sourceLock sync.Mutex
	for name, source := range rl.FunctionConfig.Sources {
		wg.Add(1)

		go func(name string, source SourceConfig) {
			for k, v := range source.Args {
				source.Args[k] = expandEnvInterface(v)
			}

			flatVars := make(map[string]string)
			flattenToMapWithJsonify(source.Vars, "", flatVars, false)

			var vars map[string]string

			switch source.Type {
			case "Variable":
				vars = flatVars
			case "Environment":
				vars = convertEnvironmentConfig(&source, flatVars)
			case "File":
				vars = filterMap(convertFileConfig(&source), flatVars)
			case "Exec":
				vars = filterMap(convertExecConfig(&source), flatVars)
			case "SecretsManager":
				vars = filterMap(convertSecretsManagerConfig(&source), flatVars)
			case "TerraformState":
				vars = filterMap(convertTerraformStateConfig(&source), flatVars)
			default:
				panic(errors.New("Invalid source type " + source.Type))
			}

			if DebugEnabled {
				fmt.Fprintf(os.Stderr, "Source '%s':\n", name)
				for k, v := range vars {
					fmt.Fprintf(os.Stderr, "\t%s (%d chars)\n", k, len(v))
				}
			}

			defer sourceLock.Unlock()
			sourceLock.Lock()
			sources[name] = vars

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
		flattenToMapWithJsonify(merge, "", flatMerge, false)

		if DebugEnabled {
			fmt.Fprintf(os.Stderr, "Merge '%s':\n", name)
		}

		for k, v := range flatMerge {
			split := mergeSplit.FindStringSubmatch(os.ExpandEnv(v))
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
				panic(fmt.Errorf("merge source '%s' was not found for key '%s'", sourceName, sourceKey))
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
