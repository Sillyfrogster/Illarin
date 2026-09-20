package integration

import "github.com/gin-gonic/gin"

func refuseBlog(c *gin.Context, status int, code BlogErrorCode, message string) {
	c.AbortWithStatusJSON(status, BlogError{Error: message, Code: code})
}

func refuseField(
	c *gin.Context,
	status int,
	code BlogErrorCode,
	message string,
	field string,
) {
	c.AbortWithStatusJSON(status, BlogError{Error: message, Code: code, Field: &field})
}
