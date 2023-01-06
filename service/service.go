/*@Author: LinkLeong link@icewhale.com
 *@Date: 2022-07-12 09:48:56
 *@LastEditors: LinkLeong
 *@LastEditTime: 2022-09-02 22:10:05
 *@FilePath: /CasaOS/service/service.go
 *@Description:
 *@Website: https://www.casaos.io
 *Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package service

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	v1 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v1"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v2"
	"github.com/IceWhaleTech/CasaOS-Common/external"
)

var MyService Services

type Services interface {
	V1AppStore() v1.AppStore
	V2AppStore() *v2.AppStore

	// Git() *GitService
	Compose() *v2.ComposeService
	Docker() DockerService
	Gateway() external.ManagementService
	Notify() external.NotifyService
	MessageBus() *message_bus.ClientWithResponses
}

func NewService(RuntimePath string) Services {
	gatewayManagement, err := external.NewManagementService(RuntimePath)
	if err != nil && len(RuntimePath) > 0 {
		panic(err)
	}

	v2appStore, err := v2.NewAppStore()
	if err != nil {
		panic(err)
	}

	return &store{
		gateway: gatewayManagement,
		notify:  external.NewNotifyService(RuntimePath),

		v1appStore: v1.NewAppService(),
		v2appStore: v2appStore,
		compose:    v2.NewComposeService(),
		docker:     NewDockerService(),
		// git:        NewGitService(),
	}
}

type store struct {
	v1appStore v1.AppStore
	v2appStore *v2.AppStore

	// git     *GitService
	compose *v2.ComposeService
	docker  DockerService
	gateway external.ManagementService
	notify  external.NotifyService
}

func (c *store) Gateway() external.ManagementService {
	return c.gateway
}

func (c *store) Notify() external.NotifyService {
	return c.notify
}

func (c *store) V1AppStore() v1.AppStore {
	return c.v1appStore
}

func (c *store) V2AppStore() *v2.AppStore {
	return c.v2appStore
}

// func (c *store) Git() *GitService {
// 	return c.git
// }

func (c *store) Compose() *v2.ComposeService {
	return c.compose
}

func (c *store) Docker() DockerService {
	return c.docker
}

func (c *store) MessageBus() *message_bus.ClientWithResponses {
	client, _ := message_bus.NewClientWithResponses("", func(c *message_bus.Client) error {
		// error will never be returned, as we always want to return a client, even with wrong address,
		// in order to avoid panic.
		//
		// If we don't avoid panic, message bus becomes a hard dependency, which is not what we want.

		messageBusAddress, err := external.GetMessageBusAddress(config.CommonInfo.RuntimePath)
		if err != nil {
			c.Server = "message bus address not found"
			return nil
		}

		c.Server = messageBusAddress
		return nil
	})

	return client
}
