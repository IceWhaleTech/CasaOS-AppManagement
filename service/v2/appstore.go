package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v2"

	_ "embed"
)

var (
	//go:embed fixtures/sample.docker-compose.yaml
	sampleDockerComposeYAML string

	catalog = map[string]composeApp{}
)

type composeApp struct {
	YAML    string
	Project *types.Project
}

type ProjectExtension struct {
	ID string `mapstructure:"id"`
}

func init() {
	project, err := loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: []byte(sampleDockerComposeYAML),
			},
		},
	})
	if err != nil {
		panic(err)
	}
	if ex, ok := project.Extensions["x-casaos"]; ok {
		var projectEx ProjectExtension
		if err := loader.Transform(ex, &projectEx); err != nil {
			panic(err)
		}

		catalog[projectEx.ID] = composeApp{
			YAML:    sampleDockerComposeYAML,
			Project: project,
		}

	} else {
		panic("invalid project extension")
	}
}

func GetAppInfo(id codegen.StoreAppID) error {
	composeYAML := GetAppComposeYAML(id)

	var compose interface{}

	if err := yaml.Unmarshal([]byte(*composeYAML), &compose); err != nil {
		return err
	}

	return nil
}

func GetAppComposeYAML(id codegen.StoreAppID) *string {
	if v, ok := catalog[id]; ok {
		return &v.YAML
	}

	return nil
}
