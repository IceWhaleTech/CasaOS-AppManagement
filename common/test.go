package common

import _ "embed"

//go:embed fixtures/sample.docker-compose.yaml
var SampleComposeAppYAML string

//go:embed fixtures/sample-vanilla.docker-compose.yaml
var SampleVanillaComposeAppYAML string
