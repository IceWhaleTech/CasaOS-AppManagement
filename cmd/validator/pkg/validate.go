package pkg

import "github.com/IceWhaleTech/CasaOS-AppManagement/service"

func VaildDockerCompose(yaml []byte) error {
	_, err := service.NewComposeAppFromYAML(yaml, false, false)

	return err
}
