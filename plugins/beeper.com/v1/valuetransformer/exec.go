package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"gopkg.in/yaml.v3"
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
		fmt.Println("error running", process.Path, process.Args)
		panic(err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(buffer.Bytes(), &raw); err != nil {
		panic(err)
	}

	flat := make(map[string]string)
	flattenToMap(raw, "", flat)
	return flat
}
