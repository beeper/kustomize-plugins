package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type TerraformOutput struct {
	Value interface{} `json:"value"`
	Type  interface{} `json:"type"`
}

type TerraformState struct {
	Version          int                        `json:"version"`
	TerraformVersion string                     `json:"terraform_version"`
	Serial           int                        `json:"serial"`
	Lineage          string                     `json:"lineage"`
	Outputs          map[string]TerraformOutput `json:"outputs"`
	//Resources        []interface{}              `json:"resources"`
}

func convertTerraformStateConfig(config *SourceConfig) map[string]string {
	data, err := readFile(config)
	if err != nil {
		panic(err)
	}

	tfstate := TerraformState{}
	if err := json.Unmarshal(data, &tfstate); err != nil {
		panic(err)
	}

	out := make(map[string]string)

	for name, output := range tfstate.Outputs {
		if typ, ok := output.Type.(string); ok {
			switch typ {
			case "string":
				if val, ok := output.Value.(string); ok {
					out[name] = val
				}
			case "number":
				if val, ok := output.Value.(float64); ok {
					out[name] = fmt.Sprintf("%f", val)
				}
			case "bool":
				if val, ok := output.Value.(bool); ok {
					out[name] = strconv.FormatBool(val)
				}
			}
		}
	}

	return out
}
