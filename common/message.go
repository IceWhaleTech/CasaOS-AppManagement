package common

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
)

var (
	EventTypeAppInstalling       message_bus.EventType
	EventTypeAppInstalled        message_bus.EventType
	EventTypeAppFailedInstalling message_bus.EventType

	EventTypeAppUninstalling       message_bus.EventType
	EventTypeAppUninstalled        message_bus.EventType
	EventTypeAppFailedUninstalling message_bus.EventType
)

func init() {
	EventTypeAppInstalling = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             fmt.Sprintf("%s:app:installing", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppInstalled = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             fmt.Sprintf("%s:app:installed", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppFailedInstalling = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             fmt.Sprintf("%s:app:failed-installing", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppUninstalling = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             fmt.Sprintf("%s:app:uninstalling", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppUninstalled = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             fmt.Sprintf("%s:app:uninstalled", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppFailedUninstalling = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             fmt.Sprintf("%s:app:failed-uninstalling", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{},
	}
}
