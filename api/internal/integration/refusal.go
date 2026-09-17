package integration

import "github.com/gin-gonic/gin"

func refusePublication(c *gin.Context, status int, code PublicationErrorCode, message string) {
	c.AbortWithStatusJSON(status, PublicationError{Error: message, Code: code})
}

func refuseField(
	c *gin.Context,
	status int,
	code PublicationErrorCode,
	message string,
	field string,
) {
	c.AbortWithStatusJSON(status, PublicationError{Error: message, Code: code, Field: &field})
}
