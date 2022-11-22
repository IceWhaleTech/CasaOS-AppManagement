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
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

var (
	Cache         *cache.Cache
	MyService     Repository
	NewVersionApp map[string]string
)

type Repository interface {
	App() AppService
	Docker() DockerService
	Gateway() external.ManagementService
}

func NewService(db *gorm.DB, RuntimePath string) Repository {
	gatewayManagement, err := external.NewManagementService(RuntimePath)
	if err != nil && len(RuntimePath) > 0 {
		panic(err)
	}

	return &store{
		gateway: gatewayManagement,
		app:     NewAppService(db),
		docker:  NewDockerService(),
	}
}

type store struct {
	app     AppService
	docker  DockerService
	gateway external.ManagementService
}

func (c *store) Gateway() external.ManagementService {
	return c.gateway
}

func (c *store) App() AppService {
	return c.app
}

func (c *store) Docker() DockerService {
	return c.docker
}
