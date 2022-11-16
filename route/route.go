package route

import (
	"os"

	v1 "github.com/IceWhaleTech/CasaOS-AppManagement/route/v1"
	"github.com/IceWhaleTech/CasaOS-Common/middleware"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	// check if environment variable is set
	ginMode, success := os.LookupEnv(gin.EnvGinMode)
	if !success {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Cors())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	if ginMode != gin.ReleaseMode {
		r.Use(middleware.WriteLog())
	}

	v1Group := r.Group("/v1")

	v1Group.Use(jwt.ExceptLocalhost())
	{
		v1AppsGroup := v1Group.Group("/apps")
		v1AppsGroup.Use()
		{
			v1AppsGroup.GET("", v1.AppList) // list
			v1AppsGroup.GET("/:id", v1.AppInfo)
		}
	}

	return r
}
