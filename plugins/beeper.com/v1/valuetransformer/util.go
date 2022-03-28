package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

func flattenToMap(i interface{}, path string, out map[string]string) {
	switch v := i.(type) {
	case nil:
		out[path] = ""
	case string:
		out[path] = v
	case []interface{}:
		for i, v := range v {
			k := strconv.Itoa(i)
			kpath := k
			if len(path) > 0 {
				kpath = path + "." + k
			}
			flattenToMap(v, kpath, out)
		}
	case map[interface{}]interface{}:
		for k, v := range v {
			switch kt := k.(type) {
			case string:
				kpath := kt
				if len(path) > 0 {
					kpath = path + "." + kt
				}
				flattenToMap(v, kpath, out)
			default:
				fmt.Fprintf(os.Stderr, "Unhandled map key during flattening: %T, value ignored\n", v)
			}
		}
	case map[string]interface{}:
		for k, v := range v {
			kpath := k
			if len(path) > 0 {
				kpath = path + "." + k
			}
			flattenToMap(v, kpath, out)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unhandled type during flattening: %T, defaulting to %%v\n", v)
		out[path] = fmt.Sprintf("%v", v)
	}
}

func transformInterface(i interface{}, transforms []Transform) interface{} {
	switch t := i.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{})
		for k, v := range t {
			out[k] = transformInterface(v, transforms)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for k, v := range t {
			out[k] = transformInterface(v, transforms)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[interface{}]interface{})
		for k, v := range t {
			out[k] = transformInterface(v, transforms)
		}
		return out
	case string:
		out := t
		for _, transform := range transforms {
			out = transform.regex.ReplaceAllStringFunc(out, func(sk string) string {
				matches := transform.regex.FindStringSubmatch(sk)
				if len(matches) < 2 {
					return sk
				}
				repl, ok := (*transform.source)[matches[1]]
				if !ok {
					return sk
				}
				return repl
			})
		}
		return out
	default:
		if DebugEnabled {
			fmt.Fprintf(os.Stderr, "Unhandled type during transforming: %T, ignored\n", t)
		}
	}
	return i
}

func applyTransforms(resource map[string]interface{}, config *TransformerConfig, sources SourceMap) map[string]interface{} {
	kind := getString(resource, "kind")
	metadata := getMap(resource, "metadata")
	name := getString(metadata, "name")
	namespace := getString(metadata, "namespace")

	transforms := []Transform{}

	for _, t := range config.Transforms {
		if t.Target.Kind != "" && t.Target.Kind != kind {
			continue
		}
		if t.Target.Name != "" && t.Target.Name != name {
			continue
		}
		if t.Target.Namespace != "" && t.Target.Namespace != namespace {
			continue
		}

		source := sources[t.Source]
		if source == nil {
			panic(errors.New("Unknown source " + t.Source))
		}

		var regex *regexp.Regexp
		if t.Regex == "" {
			regex = regexp.MustCompile(`\${([^}]*)}`)
		} else {
			regex = regexp.MustCompile(t.Regex)
		}

		transforms = append(transforms, Transform{regex, &source})

		if DebugEnabled {
			fmt.Fprintf(os.Stderr, "Enabled transform regex '%s' with source '%s' to %s/%s (target was %s/%s in %s)\n", regex.String(), t.Source, kind, name, t.Target.Kind, t.Target.Name, t.Target.Namespace)
		}
	}

	return transformInterface(resource, transforms).(map[string]interface{})
}
