package v2

import "github.com/IceWhaleTech/CasaOS-AppManagement/codegen"

type AppManagement struct{}

func NewAppManagement() codegen.ServerInterface {
	return &AppManagement{}
}
