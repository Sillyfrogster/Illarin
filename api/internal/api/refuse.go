package api

import "github.com/gin-gonic/gin"

// Refuse answers with the error body the site reads
func Refuse(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// RefuseField answers with the error body and names the field that caused it
func RefuseField(c *gin.Context, status int, field, message string) {
	c.JSON(status, gin.H{"error": message, "field": field})
}
