package docker_test

import (
	"fmt"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
)

func TestGetDir(t *testing.T) {
	fmt.Println(docker.GetDir("", "config"))
}
