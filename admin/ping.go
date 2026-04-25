package admin

import (
	"kerbecs/config"

	"github.com/gin-gonic/gin"
)

func Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": config.Name + " v" + config.Version + " is online!",
	})
}
