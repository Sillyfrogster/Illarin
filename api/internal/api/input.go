package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PathID reads a UUID from the path and answers 400 when it is not one
func PathID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		RefuseParameter(c, name, err)
		return uuid.UUID{}, false
	}
	return id, true
}

// PathNumber reads a whole number from the path and answers 400 when it is not one
func PathNumber(c *gin.Context, name string) (int, bool) {
	number, err := strconv.Atoi(c.Param(name))
	if err != nil {
		RefuseParameter(c, name, err)
		return 0, false
	}
	return number, true
}

func RefuseParameter(c *gin.Context, name string, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("Invalid format for parameter %s: %v", name, err)})
}

// DecodeOneJSON reads exactly one JSON value with no unknown fields
func DecodeOneJSON(reader io.Reader, destination any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("more than one JSON value")
		}
		return err
	}
	return nil
}

// ReadBoundedJSON reads one JSON object no larger than limit and answers 400 or 413 when it cannot
func ReadBoundedJSON(c *gin.Context, destination any, limit int64, tooLargeMessage string) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		Refuse(c, http.StatusBadRequest, "Send JSON with the application/json content type.")
		return false
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	if err := DecodeOneJSON(body, destination); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Refuse(c, http.StatusRequestEntityTooLarge, tooLargeMessage)
			return false
		}
		Refuse(c, http.StatusBadRequest, "Send one valid JSON object.")
		return false
	}
	return true
}

// RequestSource is the address a request came from, trusting the proxy only on loopback
func RequestSource(c *gin.Context) string {
	host := remoteHost(c.Request.RemoteAddr)
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return host
	}
	forwarded := strings.Split(c.GetHeader("X-Forwarded-For"), ",")
	if len(forwarded) == 0 {
		return host
	}
	candidate := strings.TrimSpace(forwarded[len(forwarded)-1])
	if net.ParseIP(candidate) == nil {
		return host
	}
	return candidate
}

func remoteHost(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}
	return address
}

// Query reads query values in order and keeps the first one that is refused
type Query struct {
	values  url.Values
	refusal string
}

func ReadQuery(c *gin.Context) *Query {
	return &Query{values: c.Request.URL.Query()}
}

// Refused answers 400 for the first refused value and reports whether it did
func (q *Query) Refused(c *gin.Context) bool {
	if q.refusal == "" {
		return false
	}
	c.JSON(http.StatusBadRequest, gin.H{"msg": q.refusal})
	return true
}

func (q *Query) single(name string) (string, bool) {
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

func (q *Query) refuse(name string, err error) {
	q.refusal = fmt.Sprintf("Invalid format for parameter %s: %v", name, err)
}

func QueryRequired(q *Query, name string) string {
	if _, found := q.values[name]; !found && q.refusal == "" {
		q.refusal = "Query argument " + name + " is required, but not found"
		return ""
	}
	value, _ := q.single(name)
	return value
}

func QueryText[T ~string](q *Query, name string) *T {
	value, ok := q.single(name)
	if !ok {
		return nil
	}
	text := T(value)
	return &text
}

func QueryList(q *Query, name string) *[]string {
	values, found := q.values[name]
	if !found || q.refusal != "" {
		return nil
	}
	return &values
}

func queryParsed[T any](q *Query, name string, parse func(string) (T, error)) *T {
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

func QueryNumber(q *Query, name string) *int {
	return queryParsed(q, name, strconv.Atoi)
}

func QueryFlag(q *Query, name string) *bool {
	return queryParsed(q, name, strconv.ParseBool)
}

func QueryID(q *Query, name string) *uuid.UUID {
	return queryParsed(q, name, uuid.Parse)
}

func QueryTime(q *Query, name string) *time.Time {
	return queryParsed(q, name, func(value string) (time.Time, error) {
		if moment, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return moment, nil
		}
		return time.Parse(time.DateOnly, value)
	})
}
