package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
)

// OK sends a 200 JSON response with the given data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// Created sends a 201 JSON response with the given data.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Message sends a 200 JSON response with a message field.
func Message(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// Error writes a JSON error response and aborts the handler chain.
func Error(c *gin.Context, statusCode int, err error) {
	commonerrors.AbortWithError(c, statusCode, err)
}

// ErrorFromErr maps the error to an HTTP status and writes a JSON error response.
func ErrorFromErr(c *gin.Context, err error) {
	status := commonerrors.HTTPStatusFromError(err)
	commonerrors.AbortWithError(c, status, err)
}
