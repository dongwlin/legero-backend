package httpx

import "github.com/gin-gonic/gin"

func JSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

func NoContent(c *gin.Context) {
	c.Status(204)
}
