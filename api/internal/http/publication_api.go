package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/credential"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
)

const publicationAPIPrefix = "/v1/publication/"

const idempotencyKeyHeader = "Idempotency-Key"

const minIdempotencyKey = 8

const maxIdempotencyKey = 200

const idempotencySlack = 1 << 20

const publicationBearerKey = "publicationBearer"

func (h *Handlers) publicationAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		value := bearerToken(c)
		if _, ok := credential.Read(value, credential.Publication); !ok {
			c.Next()
			return
		}
		if !strings.HasPrefix(c.FullPath(), publicationAPIPrefix) {
			refusePublication(c, http.StatusUnauthorized, CodeUnauthenticated,
				"A publication token reaches Illarin's publication routes and nothing else.")
			return
		}
		if c.GetHeader("Origin") != "" {
			refusePublication(c, http.StatusUnauthorized, CodeUnauthenticated,
				"A publication token belongs to server-side tooling, not to a browser.")
			return
		}
		bearing, err := h.publications.Bearing(c.Request.Context(), value)
		if err != nil {
			refuseBearer(c, err)
			return
		}
		c.Set(publicationBearerKey, bearing)
		err = h.publications.Take(c.Request.Context(), bearing.Token.ID, operationOf(c))
		if err != nil {
			refusePace(c, err)
			return
		}
		h.replayable(c, bearing)
	}
}

func publicationBearing(c *gin.Context) (publication.Bearer, bool) {
	held, ok := c.Get(publicationBearerKey)
	if !ok {
		return publication.Bearer{}, false
	}
	bearing, ok := held.(publication.Bearer)
	return bearing, ok
}

func (h *Handlers) replayable(c *gin.Context, bearing publication.Bearer) {
	key := c.GetHeader(idempotencyKeyHeader)
	if key == "" || safeMethod(c.Request.Method) {
		c.Next()
		return
	}
	length := utf8.RuneCountInString(key)
	if length < minIdempotencyKey || length > maxIdempotencyKey ||
		key != strings.TrimSpace(key) {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"Send an idempotency key of 8 to 200 characters.", idempotencyKeyHeader)
		return
	}
	operation := routeKey(c.Request.Method, c.FullPath())
	c.Request.Body = http.MaxBytesReader(
		c.Writer, c.Request.Body, h.maxUploadBytes+idempotencySlack,
	)
	attempt, err := h.publications.ClaimAttempt(
		c.Request.Context(), bearing.Token.ID, operation, key,
	)
	if err != nil {
		refusePublication(c, http.StatusInternalServerError, CodeServerError,
			"Could not read the idempotency key.")
		return
	}
	if !attempt.Fresh {
		answerEarlierAttempt(c, attempt)
		return
	}
	h.runOnce(c, bearing, operation, key)
}

func (h *Handlers) runOnce(
	c *gin.Context,
	bearing publication.Bearer,
	operation string,
	key string,
) {
	sent := &hashedBody{ReadCloser: c.Request.Body, sum: sha256.New()}
	kept := &keptResponse{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
	c.Request.Body = sent
	c.Writer = kept
	defer func() {
		c.Writer = kept.ResponseWriter
		after := context.WithoutCancel(c.Request.Context())
		_, drained := io.Copy(io.Discard, sent)
		if drained != nil || kept.overflowed ||
			kept.Status() >= http.StatusInternalServerError {
			_ = h.publications.ReleaseAttempt(after, bearing.Token.ID, operation, key)
			return
		}
		_ = h.publications.FinishAttempt(
			after, bearing.Token.ID, operation, key,
			sent.sum.Sum(nil), kept.Status(), kept.answer(),
		)
	}()
	c.Next()
}

func answerEarlierAttempt(c *gin.Context, attempt publication.Attempt) {
	if attempt.Running {
		refusePublication(c, http.StatusConflict, CodeIdempotencyInProgress,
			"The first request with this idempotency key has not finished yet.")
		return
	}
	sum := sha256.New()
	if _, err := io.Copy(sum, c.Request.Body); err != nil {
		refuseField(c, http.StatusBadRequest, CodeInvalid,
			"The request body could not be read.", idempotencyKeyHeader)
		return
	}
	if !bytes.Equal(sum.Sum(nil), attempt.Fingerprint) {
		refuseField(c, http.StatusConflict, CodeIdempotencyMismatch,
			"This idempotency key was already used for a different request.",
			idempotencyKeyHeader)
		return
	}
	c.Abort()
	c.Data(attempt.Status, "application/json; charset=utf-8", attempt.Response)
}

func operationOf(c *gin.Context) string {
	switch {
	case safeMethod(c.Request.Method):
		return publication.OperationRead
	case strings.HasSuffix(c.FullPath(), "/media"):
		return publication.OperationUpload
	default:
		return publication.OperationWrite
	}
}

func retryAfterSeconds(after time.Duration) string {
	return strconv.Itoa(int((after + time.Second - 1) / time.Second))
}

func safeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead ||
		method == http.MethodOptions
}

func refuseBearer(c *gin.Context, err error) {
	switch {
	case errors.Is(err, publication.ErrTokenExpired):
		refusePublication(c, http.StatusUnauthorized, CodeTokenExpired,
			"This publication token has expired.")
	case errors.Is(err, publication.ErrTokenRevoked):
		refusePublication(c, http.StatusUnauthorized, CodeTokenRevoked,
			"This publication token has been revoked.")
	case errors.Is(err, publication.ErrGrantRevoked):
		refusePublication(c, http.StatusUnauthorized, CodeGrantRevoked,
			"The approval behind this publication token is no longer active.")
	case errors.Is(err, publication.ErrTokenCredential):
		refusePublication(c, http.StatusUnauthorized, CodeUnauthenticated,
			"This publication token is not live.")
	default:
		refusePublication(c, http.StatusInternalServerError, CodeServerError,
			"Could not check the publication token.")
	}
}

func refusePace(c *gin.Context, err error) {
	var limited publication.TooManyRequests
	if !errors.As(err, &limited) {
		refusePublication(c, http.StatusInternalServerError, CodeServerError,
			"Could not check the publication token.")
		return
	}
	c.Header("Retry-After", retryAfterSeconds(limited.After))
	refusePublication(c, http.StatusTooManyRequests, CodeRateLimited,
		"This publication token is going faster than the "+limited.Operation+
			" limit allows. Wait and try again.")
}

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

type hashedBody struct {
	io.ReadCloser
	sum hash.Hash
}

func (b *hashedBody) Read(p []byte) (int, error) {
	read, err := b.ReadCloser.Read(p)
	if read > 0 {
		b.sum.Write(p[:read])
	}
	return read, err
}

type keptResponse struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	overflowed bool
}

func (k *keptResponse) Write(p []byte) (int, error) {
	k.keep(p)
	return k.ResponseWriter.Write(p)
}

func (k *keptResponse) WriteString(s string) (int, error) {
	k.keep([]byte(s))
	return k.ResponseWriter.WriteString(s)
}

func (k *keptResponse) answer() []byte {
	if k.body.Len() == 0 {
		return []byte{}
	}
	return k.body.Bytes()
}

func (k *keptResponse) keep(p []byte) {
	if k.body.Len()+len(p) > publication.MaxAttemptResponse {
		k.overflowed = true
		return
	}
	k.body.Write(p)
}
