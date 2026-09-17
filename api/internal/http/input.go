package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const workingCopyVersionHeader = "X-Working-Copy-Version"

// pathID reads a UUID from the path and answers 400 when it is not one
func pathID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		refuseParameter(c, name, err)
		return uuid.UUID{}, false
	}
	return id, true
}

// pathNumber reads a whole number from the path and answers 400 when it is not one
func pathNumber(c *gin.Context, name string) (int, bool) {
	number, err := strconv.Atoi(c.Param(name))
	if err != nil {
		refuseParameter(c, name, err)
		return 0, false
	}
	return number, true
}

// workingCopyVersion reads the working copy version the creator last saw
func workingCopyVersion(c *gin.Context) (int64, bool) {
	values := c.Request.Header.Values(workingCopyVersionHeader)
	if len(values) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Header parameter " + workingCopyVersionHeader + " is required, but not found"})
		return 0, false
	}
	if len(values) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("Expected one value for %s, got %d", workingCopyVersionHeader, len(values))})
		return 0, false
	}
	version, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		refuseParameter(c, workingCopyVersionHeader, err)
		return 0, false
	}
	return version, true
}

// fromIllarin answers 403 unless the request carries exactly one X-Illarin-Request header
func fromIllarin(c *gin.Context) bool {
	if len(c.Request.Header.Values(browserMutationHeader)) == 1 {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "Open this action from Illarin and try again."})
	return false
}

func refuseParameter(c *gin.Context, name string, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("Invalid format for parameter %s: %v", name, err)})
}

// queryReader reads query values in order and keeps the first one that is refused
type queryReader struct {
	values  url.Values
	refusal string
}

func readQuery(c *gin.Context) *queryReader {
	return &queryReader{values: c.Request.URL.Query()}
}

// refused answers 400 for the first refused value and reports whether it did
func (q *queryReader) refused(c *gin.Context) bool {
	if q.refusal == "" {
		return false
	}
	c.JSON(http.StatusBadRequest, gin.H{"msg": q.refusal})
	return true
}

func (q *queryReader) single(name string) (string, bool) {
	values, found := q.values[name]
	if !found || q.refusal != "" {
		return "", false
	}
	if len(values) != 1 {
		q.refusal = fmt.Sprintf("Invalid format for parameter %s: multiple values for single value parameter '%s'", name, name)
		return "", false
	}
	return values[0], true
}

func (q *queryReader) refuse(name string, err error) {
	q.refusal = fmt.Sprintf("Invalid format for parameter %s: %v", name, err)
}

func queryRequired(q *queryReader, name string) string {
	if _, found := q.values[name]; !found && q.refusal == "" {
		q.refusal = "Query argument " + name + " is required, but not found"
		return ""
	}
	value, _ := q.single(name)
	return value
}

func queryText[T ~string](q *queryReader, name string) *T {
	value, ok := q.single(name)
	if !ok {
		return nil
	}
	text := T(value)
	return &text
}

func queryList(q *queryReader, name string) *[]string {
	values, found := q.values[name]
	if !found || q.refusal != "" {
		return nil
	}
	return &values
}

func queryParsed[T any](q *queryReader, name string, parse func(string) (T, error)) *T {
	value, ok := q.single(name)
	if !ok {
		return nil
	}
	parsed, err := parse(value)
	if err != nil {
		q.refuse(name, err)
		return nil
	}
	return &parsed
}

func queryNumber(q *queryReader, name string) *int {
	return queryParsed(q, name, strconv.Atoi)
}

func queryFlag(q *queryReader, name string) *bool {
	return queryParsed(q, name, strconv.ParseBool)
}

func queryID(q *queryReader, name string) *uuid.UUID {
	return queryParsed(q, name, uuid.Parse)
}

func queryTime(q *queryReader, name string) *time.Time {
	return queryParsed(q, name, func(value string) (time.Time, error) {
		if moment, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return moment, nil
		}
		return time.Parse(time.DateOnly, value)
	})
}
