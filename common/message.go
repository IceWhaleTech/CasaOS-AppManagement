package common

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
)

// common properties
var (
	PropertyTypeMessage = message_bus.PropertyType{
		Name:        "message",
		Description: utils.Ptr("message at different levels, typically for error"),
	}
)

// app properties
var (
	PropertyTypeAppName = message_bus.PropertyType{
		Name:        "app:name",
		Description: utils.Ptr("name of the app which could be a container image name including version, a snap name or the name of any other forms of app"),
		Example:     utils.Ptr("hello-world:latest (this is the name of a container image"),
	}

	PropertyTypeAppTitle = message_bus.PropertyType{
		Name:        "app:title",
		Description: utils.Ptr("titles of the app in different languages - serialized as JSON"),
		Example:     utils.Ptr("{\"en_us\":\"OpenSpeedTest\"}"),
	}

	PropertyTypeAppIcon = message_bus.PropertyType{
		Name:        "app:icon",
		Description: utils.Ptr("icon of the app"),
		Example:     utils.Ptr("https://example.com/icon.png"),
	}

	PropertyTypeAppProgress = message_bus.PropertyType{
		Name:        "app:progress",
		Description: utils.Ptr("progress of the app"),
		Example:     utils.Ptr("64"),
	}
)

// container properties
var (
	PropertyTypeContainerID = message_bus.PropertyType{
		Name:        "docker:container:id",
		Description: utils.Ptr("ID of the container"),
		Example:     utils.Ptr("855084f79fc89bea4de5111c69621b3329ecf0a1106863a7a83bbdef01d33b9e"),
	}

	PropertyTypeContainerName = message_bus.PropertyType{
		Name:        "docker:container:name",
		Description: utils.Ptr("name of the container"),
		Example:     utils.Ptr("hello-world"),
	}
)

// image properties
var (
	PropertyTypeImageName = message_bus.PropertyType{
		Name:        "docker:image:name",
		Description: utils.Ptr("name of the image"),
		Example:     utils.Ptr("hello-world:latest"),
	}

	PropertyTypeImageUpdated = message_bus.PropertyType{
		Name:        "docker:image:updated",
		Description: utils.Ptr("true if image is updated"),
	}
)

var EventTypes = []message_bus.EventType{
	// app-store
	EventTypeAppStoreRegisterBegin, EventTypeAppStoreRegisterEnd, EventTypeAppStoreRegisterError,

	// app
	EventTypeAppInstallBegin, EventTypeAppInstallProgress, EventTypeAppInstallEnd, EventTypeAppInstallError,
	EventTypeAppUninstallBegin, EventTypeAppUninstallEnd, EventTypeAppUninstallError,
	EventTypeAppUpdateBegin, EventTypeAppUpdateEnd, EventTypeAppUpdateError,
	EventTypeAppApplyChangesBegin, EventTypeAppApplyChangesEnd, EventTypeAppApplyChangesError,
	EventTypeAppStartBegin, EventTypeAppStartEnd, EventTypeAppStartError,
	EventTypeAppStopBegin, EventTypeAppStopEnd, EventTypeAppStopError,
	EventTypeAppRestartBegin, EventTypeAppRestartEnd, EventTypeAppRestartError,

	// image
	EventTypeImagePullBegin, EventTypeImagePullProgress, EventTypeImagePullEnd, EventTypeImagePullError,

	// container
	EventTypeContainerCreateBegin, EventTypeContainerCreateEnd, EventTypeContainerCreateError,
	EventTypeContainerStartBegin, EventTypeContainerStartEnd, EventTypeContainerStartError,
	EventTypeContainerStopBegin, EventTypeContainerStopEnd, EventTypeContainerStopError,
	EventTypeContainerRenameBegin, EventTypeContainerRenameEnd, EventTypeContainerRenameError,
	EventTypeContainerRemoveBegin, EventTypeContainerRemoveEnd, EventTypeContainerRemoveError,
}

// event types for app-store
var (
	EventTypeAppStoreRegisterBegin = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             "app-store:register-begin",
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppStoreRegisterEnd = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             "app-store:register-end",
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppStoreRegisterError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app-store:register-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
		},
	}
)

// event types for app
var (
	EventTypeAppInstallBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeAppIcon,
		},
	}

	EventTypeAppInstallProgress = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-progress",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeAppProgress,
		},
	}

	EventTypeAppInstallEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeAppIcon,
		},
	}

	EventTypeAppInstallError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeAppIcon,
			PropertyTypeMessage,
		},
	}

	EventTypeAppUninstallBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:uninstall-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppUninstallEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:uninstall-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppUninstallError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:uninstall-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeAppUpdateBegin = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             "app:update-begin",
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppUpdateEnd = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             "app:update-end",
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppUpdateError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:update-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
		},
	}

	EventTypeAppApplyChangesBegin = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             "app:apply-changes-begin",
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppApplyChangesEnd = message_bus.EventType{
		SourceID:         AppManagementServiceName,
		Name:             "app:apply-changes-end",
		PropertyTypeList: []message_bus.PropertyType{},
	}

	EventTypeAppApplyChangesError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:apply-changes-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeMessage,
		},
	}

	EventTypeAppStartBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:start-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppStartEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:start-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppStartError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:start-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeAppStopBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:stop-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppStopEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:stop-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppStopError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:stop-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeAppRestartBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:restart-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppRestartEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:restart-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppRestartError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:restart-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}
)

// event types for image
var (
	EventTypeImagePullBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:pull-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeImagePullProgress = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:pull-progress",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeImagePullEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:pull-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeImageUpdated,
		},
	}

	EventTypeImagePullError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:pull-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeImageRemoveBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:remove-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeImageRemoveEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:remove-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeImageRemoveError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:image:remove-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}
)

// event types for container
var (
	EventTypeContainerCreateBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:create-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerCreateEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:create-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerCreateError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:create-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerName,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerStartBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:start-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStartEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:start-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStartError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:start-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerStopBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:stop-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStopEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:stop-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStopError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:stop-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerRenameBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:rename-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerRenameEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:rename-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerRenameError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:rename-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerRemoveBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:remove-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerRemoveEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:remove-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerRemoveError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "docker:container:remove-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeMessage,
		},
	}
)
