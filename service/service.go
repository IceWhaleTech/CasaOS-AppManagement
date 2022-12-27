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
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/patrickmn/go-cache"
)

var (
	Cache         *cache.Cache
	MyService     Services
	NewVersionApp map[string]string
)

type Services interface {
	AppStore() AppStore
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

	return &store{
		gateway: gatewayManagement,
		notify:  external.NewNotifyService(RuntimePath),

		appStore: NewAppService(),
		docker:   NewDockerService(),
	}
}

type store struct {
	appStore AppStore
	docker   DockerService
	gateway  external.ManagementService
	notify   external.NotifyService
}

func (c *store) Gateway() external.ManagementService {
	return c.gateway
}

func (c *store) Notify() external.NotifyService {
	return c.notify
}

func (c *store) AppStore() AppStore {
	return c.appStore
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
