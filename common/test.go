package common

import _ "embed"

//go:embed fixtures/sample-appfile-export.json
var SampleLegacyAppfileExportJSON string

//go:embed fixtures/sample-category-list.json
var SampleCategoryListJSON string

//go:embed fixtures/sample.docker-compose.yaml
var SampleComposeAppYAML string

//go:embed fixtures/latest.docker-compose.yaml
var LatestComposeAppYAML string

//go:embed fixtures/sd.docker-compose.yaml
var SDComposeAppYAML string

//go:embed fixtures/sample-vanilla.docker-compose.yaml
var SampleVanillaComposeAppYAML string
