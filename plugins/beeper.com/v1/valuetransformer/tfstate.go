package main

import (
	"encoding/json"
	"errors"
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
	//Resources      []interface{}              `json:"resources"`
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

	flat := make(map[string]string)

	output := getString(config.Args, "output")
	if output != "" {
		if root, ok := tfstate.Outputs[output]; ok {
			switch value := root.Value.(type) {
			case map[interface{}]interface{}:
				flattenToMap(value, "", flat)
			case map[string]interface{}:
				flattenToMap(value, "", flat)
			default:
				panic(errors.New("unsupported output type"))
			}
		} else {
			panic(errors.New("could not find output key"))
		}
	} else {
		raw := make(map[string]interface{})
		for name, output := range tfstate.Outputs {
			raw[name] = output.Value
		}
		flattenToMap(raw, "", flat)
	}

	return flat
}
