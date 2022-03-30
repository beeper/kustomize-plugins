package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strconv"

	"gopkg.in/yaml.v2"
)

func convertExecConfig(config *SourceConfig) map[string]string {
	buffer := bytes.Buffer{}

	var path string
	var command []string

	switch c := config.Args["command"].(type) {
	case string:
		path = "/bin/sh"
		command = []string{"/bin/sh", "-c", c}
	case []interface{}:
		command = make([]string, len(c))
		for i, t := range c {
			switch v := t.(type) {
			case string:
				command[i] = v
			case int:
				command[i] = strconv.Itoa(v)
			}
		}
		var err error
		if path, err = exec.LookPath(command[0]); err != nil {
			panic(err)
		}
	case []string:
		command = c
	default:
		panic(errors.New("missing command for exec"))
	}

	process := exec.Cmd{
		Path:   path,
		Args:   command,
		Stdout: &buffer,
		Stderr: os.Stderr,
	}

	if err := process.Run(); err != nil {
		panic(err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(buffer.Bytes(), &raw); err != nil {
		panic(err)
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

	return out
}
