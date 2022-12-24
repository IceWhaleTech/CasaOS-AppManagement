package main

import (
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v3"
)

func YAML(composeApp *types.Project) ([]byte, error) {
	return yaml.Marshal(composeApp)
}
