package utils

import "runtime"

func GetCPUArch() string {
	return runtime.GOARCH
}
