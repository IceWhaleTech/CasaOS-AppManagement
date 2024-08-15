package utils

import "runtime"

func GetCPUArch() string {
	// 获取CPU架构
	cpuArch := runtime.GOARCH

	// 将CPU架构转换为更易读的格式
	var readableCPUArch string
	switch cpuArch {
	case "amd64":
		readableCPUArch = "amd64"
	case "arm":
		readableCPUArch = "arm7"
	case "arm64":
		readableCPUArch = "arm64"
	default:
		readableCPUArch = cpuArch
	}

	return readableCPUArch
}
