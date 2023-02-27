package downloadHelper

import (
	"context"

	"github.com/hashicorp/go-getter"
)

func Download(src string, dst string) error {
	backgroundCtx := context.Background()
	client := &getter.Client{
		Ctx:     backgroundCtx,
		Src:     src,
		Dst:     dst,
		Mode:    getter.ClientModeAny,
		Options: []getter.ClientOption{},
	}

	return client.Get()
}
