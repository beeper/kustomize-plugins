package main

import (
	"bytes"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

func convertExecConfig(config *SourceConfig) (map[string]string, error) {
	buffer := bytes.Buffer{}

	process := exec.Cmd{
		Path: "/bin/sh",
		Args: []string{
			"/bin/sh",
			"-c",
			config.Path,
		},
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

	return out, nil
}
