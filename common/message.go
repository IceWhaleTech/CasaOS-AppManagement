package common

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
)

// common properties
var (
	PropertyTypeMessage = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:message", AppManagementServiceName),
		Description: utils.Ptr("message at different levels, typically for error"),
	}
)

// app properties
var (
	PropertyTypeAppName = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:app:name", AppManagementServiceName),
		Description: utils.Ptr("name of the app which could be a container image name including version, a snap name or the name of any other forms of app"),
		Example:     utils.Ptr("hello-world:latest (this is the name of a container image"),
	}

	PropertyTypeAppIcon = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:app:icon", AppManagementServiceName),
		Description: utils.Ptr("url of app icon"),
		Example:     utils.Ptr("https://cdn.jsdelivr.net/gh/IceWhaleTech/CasaOS-AppStore@main/Apps/Syncthing/icon.png"),
	}
)

// container properties
var (
	PropertyTypeContainerID = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:container:id", AppManagementServiceName),
		Description: utils.Ptr("ID of the container"),
		Example:     utils.Ptr("855084f79fc89bea4de5111c69621b3329ecf0a1106863a7a83bbdef01d33b9e"),
	}

	PropertyTypeContainerName = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:container:name", AppManagementServiceName),
		Description: utils.Ptr("name of the container"),
		Example:     utils.Ptr("hello-world"),
	}
)

// image properties
var (
	PropertyTypeImageName = message_bus.PropertyType{
		Name:        fmt.Sprintf("%s:image:name", AppManagementServiceName),
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
		Name:     fmt.Sprintf("%s:app:install-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppInstallEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:install-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
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
			PropertyTypeAppName,
		},
	}

	EventTypeAppUninstallEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:uninstall-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeAppName,
		},
	}

	EventTypeAppUninstallError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:app:uninstall-error", AppManagementServiceName),
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
		Name:     fmt.Sprintf("%s:image:pull-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeImageName,
		},
	}

	EventTypeImagePullProgress = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-progress", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeImageName,

			PropertyTypeMessage,
		},
	}

	EventTypeImagePullEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeImageName,
		},
	}

	EventTypeImagePullError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:image:pull-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeImageName,

			PropertyTypeMessage,
		},
	}
)

// event types for container
var (
	EventTypeContainerCreateBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:create-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerCreateEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:create-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerCreateError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:create-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerName,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerStartBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:start-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStartEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:start-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStartError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:start-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerStopBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:stop-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStopEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:stop-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerStopError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:stop-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerRenameBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:rename-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerRenameEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:rename-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
		},
	}

	EventTypeContainerRenameError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:rename-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeContainerName,
			PropertyTypeMessage,
		},
	}

	EventTypeContainerRemoveBegin = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:remove-begin", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerRemoveEnd = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:remove-end", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
		},
	}

	EventTypeContainerRemoveError = message_bus.EventType{
		SourceID: AppManagementServiceName,
		Name:     fmt.Sprintf("%s:container:remove-error", AppManagementServiceName),
		PropertyTypeList: []message_bus.PropertyType{
			PropertyTypeContainerID,
			PropertyTypeMessage,
		},
	}
)
