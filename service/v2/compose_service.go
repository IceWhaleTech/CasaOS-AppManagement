package v2

import "github.com/IceWhaleTech/CasaOS-AppManagement/codegen"

type ComposeService struct{}

func (s *ComposeService) PrepareWorkingDirectory(projectName string, composeYAML []byte) error {
	return nil
}

func (s *ComposeService) Install(projectName string, composeYAML []byte) (*codegen.ComposeApp, error) {
	return nil, nil
}

func NewComposeService() *ComposeService {
	return &ComposeService{}
}
