package common

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
)

// common properties
var (
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

	PropertyTypeNotificationType = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:notification:type", "casaos-ui"),
		Description: utils.Ptr("type of the notification"),
		Example:     utils.Ptr("install, uninstall, update"),
	}
)

var EventTypes = []message_bus.EventType{
	// app
	EventTypeAppInstallBegin, EventTypeAppInstallOK, EventTypeAppInstallError,
	EventTypeAppUninstallBegin, EventTypeAppUninstallOK, EventTypeAppUninstallError,

	// image
	EventTypeImagePullBegin, EventTypeImagePullProgress, EventTypeImagePullOK, EventTypeImagePullError,

	// container
	EventTypeContainerCreateBegin, EventTypeContainerCreateOK, EventTypeContainerCreateError,
	EventTypeContainerStartBegin, EventTypeContainerStartOK, EventTypeContainerStartError,
	EventTypeContainerStopBegin, EventTypeContainerStopOK, EventTypeContainerStopError,
	EventTypeContainerRenameBegin, EventTypeContainerRenameOK, EventTypeContainerRenameError,
	EventTypeContainerRemoveBegin, EventTypeContainerRemoveOK, EventTypeContainerRemoveError,
}

// event types for app
var (
	EventTypeAppInstallBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:install-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppInstallOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:install-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
		},
	}

	EventTypeAppInstallError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:install-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeAppUninstallBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:uninstal-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
		},
	}

	EventTypeAppUninstallOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:uninstall-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
		},
	}

	EventTypeAppUninstallError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:uninstall-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppID,
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}
)

// event types for image
var (
	EventTypeImagePullBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeImagePullProgress = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-progress", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}

	EventTypeImagePullOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeImagePullError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}
)

// event types for container
var (
	EventTypeContainerCreateBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:create-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerCreateOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:create-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerCreateError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:create-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerStartBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:start-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerStartOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:start-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerStartError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:start-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerStopBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:stop-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerStopOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:stop-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerStopError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:stop-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerRenameBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:rename-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerRenameOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:rename-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerRenameError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:rename-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerRemoveBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:remove-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerRemoveOK = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:remove-ok", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeNotificationType,
		},
	}

	EventTypeContainerRemoveError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:remove-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
			PropertyTypeNotificationType,
		},
	}
)
