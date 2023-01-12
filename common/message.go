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
)

var EventTypes = []message_bus.EventType{
	// app
	EventTypeAppInstallBegin, EventTypeAppInstallEnd, EventTypeAppInstallError,
	EventTypeAppUninstallBegin, EventTypeAppUninstallEnd, EventTypeAppUninstallError,

	// image
	EventTypeImagePullBegin, EventTypeImagePullProgress, EventTypeImagePullEnd, EventTypeImagePullError,

	// container
	EventTypeContainerCreateBegin, EventTypeContainerCreateEnd, EventTypeContainerCreateError,
	EventTypeContainerStartBegin, EventTypeContainerStartEnd, EventTypeContainerStartError,
	EventTypeContainerStopBegin, EventTypeContainerStopEnd, EventTypeContainerStopError,
	EventTypeContainerRenameBegin, EventTypeContainerRenameEnd, EventTypeContainerRenameError,
	EventTypeContainerRemoveBegin, EventTypeContainerRemoveEnd, EventTypeContainerRemoveError,
}

// event types for app
var (
	EventTypeAppInstallBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-begin",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppInstallEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-end",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppInstallError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:install-error",
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
			PropertyTypeMessage,
		},
	}

	EventTypeAppUninstallBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     "app:uninstal-begin",
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
