package common

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
)

var (
	// common properties
	PropertyTypeAppID = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:app:id", AppManagementServiceName),
		Description: utils.Ptr("id of the app which could be a container id, a snap id or the id of any other forms of app"),
		Example:     utils.Ptr("855084f79fc89bea4de5111c69621b3329ecf0a1106863a7a83bbdef01d33b9e (this is a container id)"),
	}

	PropertyTypeAppName = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:app:name", AppManagementServiceName),
		Description: utils.Ptr("name of the app which could be a container image name including version, a snap name or the name of any other forms of app"),
		Example:     utils.Ptr("hello-world:latest (this is the name of a container image"),
	}

	PropertyTypeMessage = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:message", AppManagementServiceName),
		Description: utils.Ptr("message at different levels, typically for error"),
	}

	// event types for container app
	EventTypeContainerAppInstalling = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:installing", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeContainerAppInstalled = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:installed", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
		},
	}

	EventTypeContainerAppInstallFailed = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:install-failed", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerAppUninstalling = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:uninstalling", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
		},
	}

	EventTypeContainerAppUninstalled = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:uninstalled", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
		},
	}

	EventTypeContainerAppUninstallFailed = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:uninstall-failed", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}
)
