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
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

var (
	MyService Services

	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type Services interface {
	AppStoreManagement() *AppStoreManagement

	V2AppStore() AppStore

	// Git() *GitService
	Compose() *ComposeService
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

	v2appStore := AppStore(NewAppStoreManagement())

	return &store{
		gateway: gatewayManagement,
		notify:  external.NewNotifyService(RuntimePath),

		appStoreManagement: NewAppStoreManagement(),

		v2appStore: v2appStore,
		compose:    NewComposeService(),
		docker:     NewDockerService(),
	}
}

type store struct {
	appStoreManagement *AppStoreManagement

	v2appStore AppStore

	// git     *GitService
	compose *ComposeService
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

func (c *store) AppStoreManagement() *AppStoreManagement {
	return c.appStoreManagement
}

func (c *store) V2AppStore() AppStore {
	return c.v2appStore
}

// func (c *store) Git() *GitService {
// 	return c.git
// }

func (c *store) Compose() *ComposeService {
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

func PublishEventInSocket(ctx context.Context, eventType message_bus.EventType, properties map[string]string) (*http.Response, error) {
	socketPath := "/tmp/message-bus.sock"
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	body, err := json.Marshal(properties)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST",
		fmt.Sprintf("http://unix/v2/message_bus/event/%s/%s", eventType.SourceID, eventType.Name),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	return resp, nil
}

func PublishEventWrapper(ctx context.Context, eventType message_bus.EventType, properties map[string]string) {
	if MyService == nil {
		fmt.Println("failed to publish event - messsage bus service not initialized")
		return
	}

	if properties == nil {
		properties = map[string]string{}
	}

	// merge with properties from context
	for k, v := range common.PropertiesFromContext(ctx) {
		properties[k] = v
	}

	resp, err := PublishEventInSocket(ctx, eventType, properties)
	if err != nil {
		logger.Error("failed to publish event", zap.Error(err))

		response, err := MyService.MessageBus().PublishEventWithResponse(ctx, common.AppManagementServiceName, eventType.Name, properties)
		if err != nil {
			logger.Error("failed to publish event", zap.Error(err))
			return
		}
		defer response.HTTPResponse.Body.Close()

		if response.StatusCode() != http.StatusOK {
			logger.Error("failed to publish event", zap.String("status code", response.Status()))
		}
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error("failed to publish event", zap.String("status code", resp.Status))
	}
}
