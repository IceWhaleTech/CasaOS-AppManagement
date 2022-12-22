package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
)

type AppManagement struct{}

const MIMEApplicationYAML = "application/yaml"

func NewAppManagement() codegen.ServerInterface {
	return &AppManagement{}
}
