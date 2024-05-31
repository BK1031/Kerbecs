package controller

import (
	"kerbecs/config"

	"github.com/gin-gonic/gin"
)

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Kerbecs v" + config.Service.Version + " is online!",
	})
}
