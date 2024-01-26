package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v1"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"
	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	composeCmd "github.com/docker/compose/v2/cmd/compose"

	"github.com/docker/compose/v2/cmd/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ComposeApp codegen.ComposeApp

func (a *ComposeApp) StoreInfo(includeApps bool) (*codegen.ComposeAppStoreInfo, error) {
	ex, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]
	if !ok {
		return nil, ErrComposeExtensionNameXCasaOSNotFound
	}

	var storeInfo codegen.ComposeAppStoreInfo
	if err := loader.Transform(ex, &storeInfo); err != nil {
		return nil, err
	}

	// locate main app
	if storeInfo.Main == nil || *storeInfo.Main == "" {
		// if main app is not specified, use the first app
		for _, app := range a.Apps() {
			storeInfo.Main = &app.Name
			break
		}
	}

	if storeInfo.Scheme == nil || *storeInfo.Scheme == "" {
		storeInfo.Scheme = lo.ToPtr(codegen.Http)
	}

	if includeApps {
		apps := map[string]codegen.AppStoreInfo{}

		for _, app := range a.Apps() {
			appStoreInfo, err := app.StoreInfo()
			if err != nil {
				if err == ErrComposeExtensionNameXCasaOSNotFound {
					logger.Info("App does not have x-casaos extension - skipping", zap.String("app", app.Name))
					continue
				}

				return nil, err
			}
			apps[app.Name] = appStoreInfo
		}

		storeInfo.Apps = &apps
	}

	return &storeInfo, nil
}

func (a *ComposeApp) AuthorType() codegen.StoreAppAuthorType {
	storeInfo, err := a.StoreInfo(false)
	if err != nil {
		return codegen.Unknown
	}

	if strings.ToLower(storeInfo.Author) == strings.ToLower(storeInfo.Developer) {
		return codegen.Official
	}

	if strings.ToLower(storeInfo.Author) == strings.ToLower(common.ComposeAppAuthorCasaOSTeam) {
		return codegen.ByCasaos
	}

	return codegen.Community
}

func (a *ComposeApp) SetStoreAppID(storeAppID string) (string, bool) {
	// set store_app_id (by convention is the same as app name at install time if it does not exist)
	extension, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]
	if !ok {
		logger.Info("compose app does not have x-casaos extension - might not be a compose app for CasaOS", zap.String("app", a.Name))
		return "", false
	}

	composeAppStoreInfo, ok := extension.(map[string]interface{})
	if !ok {
		logger.Info("compose app does not have valid x-casaos extension - might not be a compose app for CasaOS", zap.String("app", a.Name))
		return "", false
	}

	value, ok := composeAppStoreInfo[common.ComposeExtensionPropertyNameStoreAppID]
	if ok {
		currentStoreAppID, ok := value.(string)
		if ok {
			logger.Info("compose app already has store_app_id", zap.String("app", a.Name), zap.String("storeAppID", currentStoreAppID))
			return currentStoreAppID, true
		}
	}

	composeAppStoreInfo[common.ComposeExtensionPropertyNameStoreAppID] = storeAppID
	return storeAppID, true
}

func (a *ComposeApp) SetTitle(title, lang string) {
	if a.Extensions == nil {
		a.Extensions = make(map[string]interface{})
	}

	extension, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]
	if !ok {
		extension = map[string]interface{}{}
		a.Extensions[common.ComposeExtensionNameXCasaOS] = extension
	}

	composeAppStoreInfo, ok := extension.(map[string]interface{})
	if !ok {
		logger.Info("compose app does not have valid x-casaos extension - might not be a compose app for CasaOS", zap.String("app", a.Name))
		return
	}

	if _, ok := composeAppStoreInfo[common.ComposeExtensionPropertyNameTitle]; !ok {
		composeAppStoreInfo[common.ComposeExtensionPropertyNameTitle] = map[string]string{}
	}

	titleMap, ok := composeAppStoreInfo[common.ComposeExtensionPropertyNameTitle].(map[string]string)
	if !ok {
		logger.Info("compose app does not have valid title map in its x-casaos extension - might not be a compose app for CasaOS", zap.String("app", a.Name))
		return
	}

	if _, ok := titleMap[lang]; !ok {
		titleMap[lang] = title
	}
}

func (a *ComposeApp) IsUpdateAvailable() bool {
	storeInfo, err := a.StoreInfo(false)
	if err != nil {
		logger.Error("failed to get store info of compose app, thus no update available", zap.Error(err))
		return false
	}

	if storeInfo == nil || storeInfo.StoreAppID == nil || *storeInfo.StoreAppID == "" {
		logger.Error("store info of compose app is not valid, thus no update available")
		return false
	}

	storeComposeApp, err := MyService.V2AppStore().ComposeApp(*storeInfo.StoreAppID)
	if err != nil {
		logger.Error("failed to get store compose app, thus no update available", zap.Error(err))
		return false
	}

	if storeComposeApp == nil {
		logger.Error("store compose app not found, thus no update available", zap.String("storeAppID", *storeInfo.StoreAppID))
		return false
	}

	return a.IsUpdateAvailableWith(storeComposeApp)
}

func (a *ComposeApp) IsUpdateAvailableWith(storeComposeApp *ComposeApp) bool {
	storeComposeAppStoreInfo, err := storeComposeApp.StoreInfo(false)
	if err != nil || storeComposeAppStoreInfo == nil {
		logger.Error("failed to get store info of store compose app, thus no update available", zap.Error(err))
		return false
	}

	mainAppName := *storeComposeAppStoreInfo.Main

	mainApp := a.App(mainAppName)
	if mainApp == nil {
		logger.Error("main app not found in local compose app, thus no update available", zap.String("name", mainAppName))
		return false
	}

	mainAppImage, mainAppTag := docker.ExtractImageAndTag(mainApp.Image)

	if mainAppTag == "latest" {
		logger.Info("main app image tag is latest, thus no update available", zap.String("image", mainApp.Image))
		return false
	}

	storeMainApp := storeComposeApp.App(mainAppName)
	if storeMainApp == nil {
		logger.Error("main app not found in store compose app, thus no update available", zap.String("name", mainAppName))
		return false
	}

	storeMainAppImage, storeMainAppTag := docker.ExtractImageAndTag(storeMainApp.Image)

	if mainAppImage != storeMainAppImage {
		logger.Error("main app image not match for local app and store app, thus no update available", zap.String("local", mainApp.Image), zap.String("store", storeMainApp.Image))
		return false
	}

	if mainAppTag == storeMainAppTag {
		return false
	}

	logger.Info("main apps of local app and store app have different image tag, thus update is available", zap.String("local", mainApp.Image), zap.String("store", storeMainApp.Image))
	return true
}

func (a *ComposeApp) Update(ctx context.Context) error {
	if len(a.ComposeFiles) <= 0 {
		return ErrComposeFileNotFound
	}

	if len(a.ComposeFiles) > 1 {
		logger.Info("warning: multiple compose files found, only the first one will be used", zap.String("compose files", strings.Join(a.ComposeFiles, ",")))
	}

	storeInfo, err := a.StoreInfo(true)
	if err != nil {
		return err
	}

	if storeInfo == nil || storeInfo.StoreAppID == nil || *storeInfo.StoreAppID == "" {
		return ErrStoreInfoNotFound
	}

	storeComposeApp, err := MyService.V2AppStore().ComposeApp(*storeInfo.StoreAppID)
	if err != nil {
		return err
	}

	if storeComposeApp == nil {
		return ErrNotFoundInAppStore
	}

	localComposeAppServices := lo.Map(a.Services, func(service types.ServiceConfig, i int) string { return service.Name })
	storeComposeAppServices := lo.Map(storeComposeApp.Services, func(service types.ServiceConfig, i int) string { return service.Name })

	localAbsentOfStore, storeAbsentOfLocal := lo.Difference(localComposeAppServices, storeComposeAppServices)
	if len(localAbsentOfStore) > 0 {
		logger.Error("local compose app has container apps that are not present in store compose app, thus update is not possible", zap.Strings("absent", localAbsentOfStore))
		return ErrComposeAppNotMatch
	}

	if len(storeAbsentOfLocal) > 0 {
		logger.Error("store compose app has container apps that are not present in local compose app, thus update is not possible", zap.Strings("absent", storeAbsentOfLocal))
		return ErrComposeAppNotMatch
	}

	for _, service := range storeComposeApp.Services {
		localComposeAppService := a.App(service.Name)
		localComposeAppService.Image = service.Image
	}

	newComposeYAML, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	// prepare for message bus events
	eventProperties := common.PropertiesFromContext(ctx)
	eventProperties[common.PropertyTypeAppName.Name] = a.Name

	if err := a.UpdateEventPropertiesFromStoreInfo(eventProperties); err != nil {
		logger.Info("failed to update event properties from store info", zap.Error(err), zap.String("name", a.Name))
	}

	go func(ctx context.Context) {
		go PublishEventWrapper(ctx, common.EventTypeAppUpdateBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeAppUpdateEnd, nil)

		if err := a.PullAndApply(ctx, newComposeYAML); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeAppUpdateError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			logger.Error("failed to update compose app", zap.Error(err), zap.String("name", a.Name))
		}
	}(ctx)

	return nil
}

func (a *ComposeApp) App(name string) *App {
	if name == "" {
		return nil
	}

	for i, service := range a.Services {
		if service.Name == name {
			return (*App)(&a.Services[i])
		}
	}

	return nil
}

func (a *ComposeApp) Apps() map[string]*App {
	apps := make(map[string]*App)

	for i, service := range a.Services {
		apps[service.Name] = (*App)(&a.Services[i])
	}

	return apps
}

func (a *ComposeApp) Containers(ctx context.Context) (map[string][]api.ContainerSummary, error) {
	service, dockerClient, err := apiService()
	if err != nil {
		return nil, err
	}
	defer dockerClient.Close()

	containers, err := service.Ps(ctx, a.Name, api.PsOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	// it is possible a `service` contains multiple containers.
	// See https://docs.docker.com/compose/compose-file/deploy/#replicas
	return lo.GroupBy(containers, func(container api.ContainerSummary) string {
		return container.Service
	}), nil
}

func (a *ComposeApp) Pull(ctx context.Context) error {
	// pull
	serviceNum := len(a.Services)

	for i, app := range a.Services {
		if err := func() error {
			go PublishEventWrapper(ctx, common.EventTypeImagePullBegin, map[string]string{
				common.PropertyTypeImageName.Name: app.Image,
			})

			defer PublishEventWrapper(ctx, common.EventTypeImagePullEnd, map[string]string{
				common.PropertyTypeImageName.Name: app.Image,
			})

			if err := docker.PullImage(ctx, app.Image, func(out io.ReadCloser) {
				pullImageProgress(ctx, out, "INSTALL", serviceNum, i+1)
			}); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeImagePullError, map[string]string{
					common.PropertyTypeImageName.Name: app.Image,
					common.PropertyTypeMessage.Name:   err.Error(),
				})
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
}

func injectEnvVariableToComposeApp(a *ComposeApp) {
	for _, service := range a.Services {
		for k, v := range config.Global {
			// if there is same name var declared in environment in compose yaml
			// we should not reassign a value to it.
			if service.Environment[k] == nil {
				service.Environment[k] = utils.Ptr(v)
			}
		}
	}
}

func (a *ComposeApp) Up(ctx context.Context, service api.Service) error {
	injectEnvVariableToComposeApp(a)

	if err := service.Up(ctx, (*codegen.ComposeApp)(a), api.UpOptions{
		Start: api.StartOptions{
			CascadeStop: true,
			Wait:        true,
		},
	}); err != nil {
		logger.Error("failed to start original compose app", zap.Error(err), zap.String("name", a.Name))
		return err
	}
	return nil
}

func (a *ComposeApp) UpWithCheckRequire(ctx context.Context, service api.Service) error {
	// prepare source path for volumes if not exist
	for i, app := range a.Services {
		for _, volume := range app.Volumes {
			if _, ok := a.Volumes[volume.Source]; ok {
				// this is a internal volume, so skip.
				continue
			}

			path := volume.Source
			if err := file.IsNotExistMkDir(path); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
					common.PropertyTypeMessage.Name: err.Error(),
				})
				return err
			}
		}

		// check if each required device exists
		deviceMapFiltered := []string{}
		for _, deviceMap := range app.Devices {
			devicePath := strings.SplitN(deviceMap, ":", 2)[0]
			if file.CheckNotExist(devicePath) {
				logger.Info("device not found", zap.String("device", devicePath))
				continue
			}
			deviceMapFiltered = append(deviceMapFiltered, deviceMap)
		}
		a.Services[i].Devices = deviceMapFiltered
	}

	if err := a.Up(ctx, service); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})
		return err
	}
	return nil
}

func (a *ComposeApp) PullAndApply(ctx context.Context, newComposeYAML []byte) error {
	// backup current compose file
	currentComposeFile := a.ComposeFiles[0]

	backupComposeFile := currentComposeFile + "." + "bak"
	if err := file.CopySingleFile(currentComposeFile, backupComposeFile, ""); err != nil {
		return err
	}

	// start compose app
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	success := false

	defer func() {
		if !success {
			if err := file.CopySingleFile(backupComposeFile, currentComposeFile, ""); err != nil {
				logger.Error("failed to restore original compose file", zap.Error(err), zap.String("src", backupComposeFile), zap.String("dst", currentComposeFile))
				return
			}

			if err := a.Up(ctx, service); err != nil {
				logger.Error("failed to start original compose app", zap.Error(err), zap.String("name", a.Name))
				return
			}

		}
	}()

	// save new compose file
	if err := file.WriteToFullPath(newComposeYAML, currentComposeFile, 0o600); err != nil {
		return err
	}

	newComposeApp, err := LoadComposeAppFromConfigFile(a.Name, currentComposeFile)
	if err != nil {
		return err
	}

	if err := newComposeApp.Pull(ctx); err != nil {
		return err
	}

	go PublishEventWrapper(ctx, common.EventTypeContainerStartBegin, nil)

	defer PublishEventWrapper(ctx, common.EventTypeContainerStartEnd, nil)

	err = newComposeApp.UpWithCheckRequire(ctx, service)

	success = true

	return err
}

func (a *ComposeApp) Create(ctx context.Context, options api.CreateOptions, service api.Service) error {
	injectEnvVariableToComposeApp(a)
	return service.Create(ctx, (*codegen.ComposeApp)(a), api.CreateOptions{})
}

func (a *ComposeApp) PullAndInstall(ctx context.Context) error {
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	// pull
	if err := a.Pull(ctx); err != nil {
		return err
	}

	// create
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerCreateBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeContainerCreateEnd, nil)

		for i, app := range a.Services {
			// prepare source path for volumes if not exist
			for _, volume := range app.Volumes {
				if _, ok := a.Volumes[volume.Source]; ok {
					// this is a internal volume, so skip.
					continue
				}

				path := volume.Source
				if err := file.IsNotExistMkDir(path); err != nil {
					go PublishEventWrapper(ctx, common.EventTypeContainerCreateError, map[string]string{
						common.PropertyTypeMessage.Name: err.Error(),
					})
					return err
				}
			}

			// check if each required device exists
			deviceMapFiltered := []string{}
			for _, deviceMap := range app.Devices {
				devicePath := strings.SplitN(deviceMap, ":", 2)[0]
				if file.CheckNotExist(devicePath) {
					logger.Info("device not found", zap.String("device", devicePath))
					continue
				}
				deviceMapFiltered = append(deviceMapFiltered, deviceMap)
			}
			a.Services[i].Devices = deviceMapFiltered
		}

		if err := a.Create(ctx, api.CreateOptions{}, service); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerCreateError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	go PublishEventWrapper(ctx, common.EventTypeContainerStartBegin, nil)

	defer PublishEventWrapper(ctx, common.EventTypeContainerStartEnd, nil)

	if err := service.Start(ctx, a.Name, api.StartOptions{
		CascadeStop: true,
		Wait:        true,
	}); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})
		return err
	}

	return nil
}

func (a *ComposeApp) Uninstall(ctx context.Context, deleteConfigFolder bool) error {
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	// stop
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerStopBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeContainerStopEnd, nil)

		if err := service.Stop(ctx, a.Name, api.StopOptions{}); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerStopError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	// remove
	go PublishEventWrapper(ctx, common.EventTypeContainerRemoveBegin, nil)

	defer PublishEventWrapper(ctx, common.EventTypeContainerRemoveEnd, nil)

	if err := service.Down(ctx, a.Name, api.DownOptions{
		RemoveOrphans: true,
		Images:        "all",
		Volumes:       true,
	}); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})

		return err
	}

	if err := file.RMDir(a.WorkingDir); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})
	}

	if !deleteConfigFolder {
		return nil
	}

	for _, app := range a.Services {
		for _, volume := range app.Volumes {
			if strings.Contains(volume.Source, a.Name) {
				path := filepath.Join(strings.Split(volume.Source, a.Name)[0], a.Name)
				if err := file.RMDir(path); err != nil {
					logger.Error("failed to remove compose app config folder", zap.Error(err), zap.String("path", path))

					go PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
						common.PropertyTypeMessage.Name: err.Error(),
					})
				}
			}
		}
	}

	return nil
}

func (a *ComposeApp) Apply(ctx context.Context, newComposeYAML []byte) error {
	// compare new ComposeApp with current ComposeApp
	if getNameFrom(newComposeYAML) != a.Name {
		return ErrComposeAppNotMatch
	}

	newComposeApp, err := NewComposeAppFromYAML(newComposeYAML, true, true)
	if err != nil {
		return err
	}

	if len(a.ComposeFiles) <= 0 {
		return ErrComposeFileNotFound
	}

	if len(a.ComposeFiles) > 1 {
		logger.Info("warning: multiple compose files found, only the first one will be used", zap.String("compose files", strings.Join(a.ComposeFiles, ",")))
	}

	// prepare for message bus events
	eventProperties := common.PropertiesFromContext(ctx)
	eventProperties[common.PropertyTypeAppName.Name] = a.Name

	// prepare for message bus events
	if err := newComposeApp.UpdateEventPropertiesFromStoreInfo(eventProperties); err != nil {
		logger.Info("failed to update event properties from store info", zap.Error(err), zap.String("name", a.Name))
	}

	go func(ctx context.Context) {
		go PublishEventWrapper(ctx, common.EventTypeAppApplyChangesBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeAppApplyChangesEnd, nil)

		if err := a.PullAndApply(ctx, newComposeYAML); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeAppApplyChangesError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			logger.Error("failed to apply changes to compose app", zap.Error(err), zap.String("name", a.Name))
		}
	}(ctx)

	return nil
}

func (a *ComposeApp) SetStatus(ctx context.Context, status codegen.RequestComposeAppStatus) error {
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	eventProperties := common.PropertiesFromContext(ctx)
	eventProperties[common.PropertyTypeAppName.Name] = a.Name

	switch status {
	case codegen.RequestComposeAppStatusStart:
		go func(ctx context.Context) {
			go PublishEventWrapper(ctx, common.EventTypeAppStartBegin, nil)

			defer PublishEventWrapper(ctx, common.EventTypeAppStartEnd, nil)

			// to make sure the container is stopped
			// timeout is 20s
			for index := 0; index < 10; index++ {
				containerSummarys, err := service.Ps(ctx, a.Name, api.PsOptions{
					All: true,
				})
				if err != nil {
					logger.Error("failed to get compose app info", zap.Error(err), zap.String("name", a.Name))
				}
				isContainerExited := true
				for _, containerSummary := range containerSummarys {
					// to make sure every service of the container is stopped
					// I think "exited" can be replace by constant value.
					isContainerExited = isContainerExited && (containerSummary.State == "exited")
				}
				if isContainerExited {
					break
				}
				time.Sleep(2 * time.Second)
			}

			if err := service.Start(ctx, a.Name, api.StartOptions{
				CascadeStop: true,
				Wait:        true,
			}); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeAppStartError, map[string]string{
					common.PropertyTypeMessage.Name: err.Error(),
				})

				logger.Error("failed to start compose app", zap.Error(err), zap.String("name", a.Name))
			}
		}(ctx)
	case codegen.RequestComposeAppStatusStop:
		go func(ctx context.Context) {
			go PublishEventWrapper(ctx, common.EventTypeAppStopBegin, nil)

			defer PublishEventWrapper(ctx, common.EventTypeAppStopEnd, nil)

			if err := service.Stop(ctx, a.Name, api.StopOptions{}); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeAppStopError, map[string]string{
					common.PropertyTypeMessage.Name: err.Error(),
				})

				logger.Error("failed to stop compose app", zap.Error(err), zap.String("name", a.Name))
			}
		}(ctx)
	case codegen.RequestComposeAppStatusRestart:
		go func(ctx context.Context) {
			go PublishEventWrapper(ctx, common.EventTypeAppRestartBegin, nil)

			defer PublishEventWrapper(ctx, common.EventTypeAppRestartEnd, nil)

			if err := service.Restart(ctx, a.Name, api.RestartOptions{}); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeAppRestartError, map[string]string{
					common.PropertyTypeMessage.Name: err.Error(),
				})

				logger.Error("failed to restart compose app", zap.Error(err), zap.String("name", a.Name))
			}
		}(ctx)
	default:
		return ErrInvalidComposeAppStatus
	}

	return nil
}

func (a *ComposeApp) Logs(ctx context.Context, lines int) ([]byte, error) {
	service, dockerClient, err := apiService()
	if err != nil {
		return nil, err
	}
	defer dockerClient.Close()

	var buf bytes.Buffer

	consumer := formatter.NewLogConsumer(ctx, &buf, &buf, false, true, false)

	if err := service.Logs(ctx, a.Name, consumer, api.LogOptions{
		Project:  (*codegen.ComposeApp)(a),
		Services: lo.Map(a.Services, func(s types.ServiceConfig, i int) string { return s.Name }),
		Follow:   false,
		Tail:     lo.If(lines < 0, "all").Else(strconv.Itoa(lines)),
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (a *ComposeApp) GetPortsInUse() (*codegen.ComposeAppValidationErrorsPortsInUse, error) {
	tcpPorts, udpPorts, err := port.ListPortsInUse()
	if err != nil {
		return nil, err
	}

	allPortsInUse := lo.Union(tcpPorts, udpPorts)

	tcpPortInUse := []string{}
	udpPortInUse := []string{}

	for _, s := range a.Services {
		for _, p := range s.Ports {
			if lo.ContainsBy(allPortsInUse, func(portInUse int) bool { return strconv.Itoa(portInUse) == p.Published }) {
				switch strings.ToLower(p.Protocol) {
				case "tcp":
					tcpPortInUse = append(tcpPortInUse, p.Published)
				case "udp":
					udpPortInUse = append(udpPortInUse, p.Published)
				}
			}
		}
	}

	if len(tcpPortInUse) == 0 && len(udpPortInUse) == 0 {
		return nil, nil
	}

	portsInUse := struct {
		TCP *codegen.PortList "json:\"TCP,omitempty\""
		UDP *codegen.PortList "json:\"UDP,omitempty\""
	}{TCP: &tcpPortInUse, UDP: &udpPortInUse}

	return &codegen.ComposeAppValidationErrorsPortsInUse{PortsInUse: &portsInUse}, nil
}

// Try to update AppIcon and AppTitle in given event properties from store info
func (a *ComposeApp) UpdateEventPropertiesFromStoreInfo(eventProperties map[string]string) error {
	if eventProperties == nil {
		return fmt.Errorf("event properties is nil")
	}

	storeInfo, err := a.StoreInfo(false)
	if err != nil {
		return err
	}

	eventProperties[common.PropertyTypeAppIcon.Name] = storeInfo.Icon

	if storeInfo.Title == nil {
		return fmt.Errorf("compose app title not found in store info")
	}

	titles, err := json.Marshal(storeInfo.Title)
	if err != nil {
		return err
	}

	eventProperties[common.PropertyTypeAppTitle.Name] = string(titles)

	return nil
}

func (a *ComposeApp) HealthCheck() (bool, error) {
	storeInfo, err := a.StoreInfo(false)
	if err != nil {
		return false, err
	}

	scheme := "http"
	if storeInfo.Scheme != nil && *storeInfo.Scheme != "" {
		scheme = string(*storeInfo.Scheme)
	}

	hostname := common.Localhost
	if storeInfo.Hostname != nil && *storeInfo.Hostname != "" {
		hostname = *storeInfo.Hostname
	}

	url := fmt.Sprintf(
		"%s://%s:%s/%s",
		scheme,
		hostname,
		storeInfo.PortMap,
		strings.TrimLeft(storeInfo.Index, "/"),
	)

	logger.Info("checking compose app health at the specified web port...", zap.String("name", a.Name), zap.Any("url", url))

	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Accept", "text/html")
	// ignore ssl error
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	response, err := client.R().Get(url)
	if err != nil {
		logger.Error("failed to check container health", zap.Error(err), zap.String("name", a.Name))
		return false, err
	}
	if response.StatusCode() == http.StatusOK || response.StatusCode() == http.StatusUnauthorized {
		return true, nil
	}

	logger.Error("compose app health check failed at the specified web port", zap.Any("name", a.Name), zap.Any("url", url), zap.String("status", fmt.Sprint(response.StatusCode())))
	return false, nil
}

func LoadComposeAppFromConfigFile(appID string, configFile string) (*ComposeApp, error) {
	options := composeCmd.ProjectOptions{
		ProjectDir:  filepath.Dir(configFile),
		ProjectName: appID,
	}

	env := []string{fmt.Sprintf("%s=%s", "AppID", appID)}
	for k, v := range baseInterpolationMap() {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// load project
	project, err := options.ToProject(
		nil,
		nil,
		cli.WithWorkingDirectory(options.ProjectDir), // this has to be the first option, otherwise it will assume the dir where this program is running is the working directory.

		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithEnv(env),
		cli.WithConfigFileEnv,
		cli.WithDefaultConfigPath,
		cli.WithEnvFiles(options.EnvFiles...),
		cli.WithName(options.ProjectName),
	)

	return (*ComposeApp)(project), err
}

func removeRuntime(a *ComposeApp) {
	for i := range a.Services {
		a.Services[i].Runtime = ""
	}
}

var gpuCache *([]external.GPUInfo) = nil

func NewComposeAppFromYAML(yaml []byte, skipInterpolation, skipValidation bool) (*ComposeApp, error) {
	tmpWorkingDir, err := os.MkdirTemp("", "casaos-compose-app-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpWorkingDir)

	// the WEBUI_PORT interpolate will tiger twice. In `pulished` and `port-map`.
	// So we need to promise multiple WEBUI_PORT interpolate is a same value.
	port, err := port.GetAvailablePort("tcp")

	project, err := loader.Load(
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Content: []byte(yaml),
				},
			},
			Environment: map[string]string{},

			// need to set a working dir because loader/normalize.go from github.com/compose-spec/compose-go makes
			// wrong assumption that the working dir is the same as the dir where this program is launched.
			WorkingDir: tmpWorkingDir,
		},
		func(o *loader.Options) {
			o.SkipInterpolation = skipInterpolation
			o.SkipValidation = skipValidation

			o.Interpolate.LookupValue = func(key string) (string, bool) {
				switch key {
				case "WEBUI_PORT":
					fmt.Printf("WEBUI_PORT is not specified, using %d\n", port)
					return strconv.Itoa(port), true
				}

				for k := range baseInterpolationMap() {
					if k == key {
						// example:  TZ => $TZ
						// we didn't want to interpolate base interpolation value.
						// they should be interpolated in LoadComposeAppFromConfig
						return fmt.Sprintf("$%s", k), true
					}
				}
				// the function may can to replace the above code.
				value, ok := os.LookupEnv(key)
				if ok {
					return value, true
				} else {
					return fmt.Sprintf("$%s", key), true
				}
			}

			if getNameFrom(yaml) != "" {
				return
			}

			// fix compose app name
			logger.Info("compose app name is not specified, getting a name from one of our contributors :)")
			projectName := random.Name(nil)
			logger.Info("compose app name is given", zap.String("name", projectName))
			o.SetProjectName(projectName, false)
		},
	)
	if err != nil {
		return nil, err
	}

	composeApp := (*ComposeApp)(project)

	if composeApp.Extensions == nil {
		composeApp.Extensions = map[string]interface{}{}
	}

	storeInfo, err := composeApp.StoreInfo(false)

	if err != nil || storeInfo == nil || storeInfo.Title == nil {
		logger.Info("compose app does not have store info with title set, re-using app name as title", zap.String("app", composeApp.Name))
		composeApp.SetTitle(composeApp.Name, common.DefaultLanguage)
	}

	if config.RemoveRuntimeIfNoNvidiaGPUFlag {
		// if gpuCache is nil, it means it is first time fetching gpu info
		if gpuCache == nil {
			value, err := external.GPUInfoList()
			if err != nil {
				gpuCache = &([]external.GPUInfo{})
			} else {
				gpuCache = &value
			}
		}

		// without nvidia-smi 	// no gpu or first time fetching gpu info failed
		if err != nil || len(*gpuCache) == 0 {
			removeRuntime(composeApp)
		}
	}

	// pass icon information to v1 label for backward compatibility, because we are
	// still using `func getContainerStats()` from `container.go` to get container stats
	// (we are being lazy to upgrade that v1 API to v2 - please help if you can :D)
	if err == nil && storeInfo != nil && storeInfo.Icon != "" {
		for i := range composeApp.Services {
			if composeApp.Services[i].Labels == nil {
				composeApp.Services[i].Labels = map[string]string{}
			}
			composeApp.Services[i].Labels[v1.V1LabelIcon] = storeInfo.Icon
		}
	}

	return composeApp, nil
}

func getNameFrom(composeYAML []byte) string {
	var baseStructure struct {
		Name string `yaml:"name"`
	}

	if err := yaml.Unmarshal(composeYAML, &baseStructure); err != nil {
		return ""
	}

	return baseStructure.Name
}
