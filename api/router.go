package router

import (
	"fmt"
	"github.com/hellodex/HelloSecurity/api/interceptor"
	"log"
	"net/http"

	general "github.com/hellodex/HelloSecurity/api/http"
	"github.com/hellodex/HelloSecurity/config"

	"github.com/gin-gonic/gin"
)

type Option func(*gin.RouterGroup)

var options = []Option{}

func Include(opts ...Option) {
	options = append(options, opts...)
}

func Init() *gin.Engine {
	Include(general.Routers)

	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.GET("/index", helloHandler) //Default welcome api

	apiGroup := r.Group("/spwapi", interceptor.HttpInterceptor()) // total interceptor stack
	for _, opt := range options {
		opt(apiGroup)
	}
	r.Run(fmt.Sprintf(":%d", config.GetConfig().Http.Port))
	return r
}

func helloHandler(c *gin.Context) {
	log.Println("hello")
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello dalink",
	})
}
