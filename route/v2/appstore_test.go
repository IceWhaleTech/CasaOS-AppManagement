package v2_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/route/v2"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/compose-spec/compose-go/types"
)

func TestFilterCatalogByCategory(t *testing.T) {
	logger.LogInitConsoleOnly()

	catalog := map[string]*service.ComposeApp{}

	filteredCatalog := v2.FilterCatalogByCategory(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 0)

	catalog["test"] = &service.ComposeApp{
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: map[string]interface{}{
				"main_app": "test",
			},
		},
		Services: []types.ServiceConfig{
			{
				Name: "test",
				Extensions: map[string]interface{}{
					common.ComposeExtensionNameXCasaOS: map[string]interface{}{
						"category": "test",
					},
				},
			},
		},
	}

	filteredCatalog = v2.FilterCatalogByCategory(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 1)

	catalog["test2"] = &service.ComposeApp{
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: map[string]interface{}{
				"main_app": "test2",
			},
		},
		Services: []types.ServiceConfig{
			{
				Name: "test2",
				Extensions: map[string]interface{}{
					common.ComposeExtensionNameXCasaOS: map[string]interface{}{
						"category": "test2",
					},
				},
			},
		},
	}

	filteredCatalog = v2.FilterCatalogByCategory(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 1)
}

func TestFilterCatalogByAuthorType(t *testing.T) {
	logger.LogInitConsoleOnly()

	catalog := map[string]*service.ComposeApp{}

	filteredCatalog := v2.FilterCatalogByAuthorType(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.ByCasaos)
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Official)
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Community)
	assert.Equal(t, len(filteredCatalog), 0)

	catalog["test"] = &service.ComposeApp{
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: map[string]interface{}{
				"main_app": "test",
			},
		},
		Services: []types.ServiceConfig{
			{
				Name: "test",
				Extensions: map[string]interface{}{
					common.ComposeExtensionNameXCasaOS: map[string]interface{}{
						"author": common.ComposeAppAuthorCasaOSTeam,
					},
				},
			},
		},
	}

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.ByCasaos)
	assert.Equal(t, len(filteredCatalog), 1)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Official)
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Community)
	assert.Equal(t, len(filteredCatalog), 0)

	catalog["test2"] = &service.ComposeApp{
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: map[string]interface{}{
				"main_app": "test2",
			},
		},
		Services: []types.ServiceConfig{
			{
				Name: "test2",
				Extensions: map[string]interface{}{
					common.ComposeExtensionNameXCasaOS: map[string]interface{}{
						"author":    "test2",
						"developer": "test2",
					},
				},
			},
		},
	}

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.ByCasaos)
	assert.Equal(t, len(filteredCatalog), 1)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Official)
	assert.Equal(t, len(filteredCatalog), 1)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Community)
	assert.Equal(t, len(filteredCatalog), 0)

	catalog["test3"] = &service.ComposeApp{
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: map[string]interface{}{
				"main_app": "test2",
			},
		},
		Services: []types.ServiceConfig{
			{
				Name: "test2",
				Extensions: map[string]interface{}{
					common.ComposeExtensionNameXCasaOS: map[string]interface{}{
						"author":    "test3",
						"developer": "syncthing",
					},
				},
			},
		},
	}

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, "test")
	assert.Equal(t, len(filteredCatalog), 0)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.ByCasaos)
	assert.Equal(t, len(filteredCatalog), 1)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Official)
	assert.Equal(t, len(filteredCatalog), 1)

	filteredCatalog = v2.FilterCatalogByAuthorType(catalog, codegen.Community)
	assert.Equal(t, len(filteredCatalog), 1)
}
