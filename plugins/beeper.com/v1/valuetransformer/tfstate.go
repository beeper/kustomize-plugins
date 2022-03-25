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

func convertTerraformStateConfig(config *SourceConfig) (map[string]string, error) {
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
		oname := name
		if len(config.Vars) > 0 {
			var ok bool
			if oname, ok = config.Vars[name]; !ok {
				continue
			}
		}
		if typ, ok := output.Type.(string); ok {
			switch typ {
			case "string":
				if val, ok := output.Value.(string); ok {
					out[oname] = val
				}
			case "number":
				if val, ok := output.Value.(float64); ok {
					out[oname] = fmt.Sprintf("%f", val)
				}
			case "bool":
				if val, ok := output.Value.(bool); ok {
					out[oname] = strconv.FormatBool(val)
				}
			}
		}
	}

	return out, nil
}
