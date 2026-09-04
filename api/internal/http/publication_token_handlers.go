package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ListPublicationTokens(c *gin.Context, id types.UUID) {
	current, ok := h.verifiedAccount(c, "reading publication tokens")
	if !ok {
		return
	}
	issued, err := h.publications.GrantTokens(c.Request.Context(), current.ID, uuid.UUID(id))
	if err != nil {
		h.publicationTokenError(c, err)
		return
	}
	c.JSON(http.StatusOK, PublicationTokenList{Tokens: toAPITokens(issued)})
}

func (h *Handlers) IssuePublicationToken(c *gin.Context, id types.UUID) {
	current, ok := h.verifiedAccount(c, "making a publication token")
	if !ok {
		return
	}
	var request IssuePublicationTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		refusePublication(c, http.StatusBadRequest, CodeInvalid, "Send the token as JSON.")
		return
	}
	made, err := h.publications.IssueToken(
		c.Request.Context(), current.ID, uuid.UUID(id), publication.TokenEdit{
			Name:      request.Name,
			ExpiresAt: request.ExpiresAt,
		},
	)
	if err != nil {
		h.publicationTokenError(c, err)
		return
	}
	c.JSON(http.StatusCreated, IssuedPublicationToken{
		Token: toAPIToken(made.Token),
		Value: made.Value,
	})
}

func (h *Handlers) RevokePublicationToken(c *gin.Context, id types.UUID) {
	current, ok := h.verifiedAccount(c, "revoking a publication token")
	if !ok {
		return
	}
	if err := h.publications.RevokeToken(c.Request.Context(), current.ID, uuid.UUID(id)); err != nil {
		h.publicationTokenError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetPublicationCredential(c *gin.Context) {
	bearing, ok := h.publicationBearer(c)
	if !ok {
		return
	}
	listed, err := h.withHolders(c, []publication.Grant{bearing.Grant})
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, PublicationCredential{
		Token: toAPIToken(bearing.Token),
		Grant: listed[0],
	})
}

// publicationBearer answers the grant a publication token authenticates as.
// The token was checked before any handler ran, so a session cookie cannot
// stand in for a token and a token cannot stand in for a session.
func (h *Handlers) publicationBearer(c *gin.Context) (publication.Bearer, bool) {
	bearing, ok := publicationBearing(c)
	if !ok {
		refusePublication(c, http.StatusUnauthorized, CodeUnauthenticated,
			"Send a publication token as a bearer credential.")
		return publication.Bearer{}, false
	}
	return bearing, true
}

func (h *Handlers) publicationTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, publication.ErrTokenNotFound):
		refusePublication(c, http.StatusNotFound, CodeNotFound, "No such publication token.")
	case errors.Is(err, publication.ErrNotTokenOwner):
		refusePublication(c, http.StatusForbidden, CodeForbidden,
			"Only the approved contributor or Illarin's publication authority can do that.")
	default:
		h.publicationError(c, err)
	}
}

func toAPITokens(issued []publication.Token) []PublicationToken {
	listed := make([]PublicationToken, 0, len(issued))
	for _, one := range issued {
		listed = append(listed, toAPIToken(one))
	}
	return listed
}

func toAPIToken(found publication.Token) PublicationToken {
	return PublicationToken{
		Id:         types.UUID(found.ID),
		GrantId:    types.UUID(found.GrantID),
		Name:       found.Name,
		Prefix:     found.Prefix,
		CreatedAt:  found.CreatedAt,
		ExpiresAt:  found.ExpiresAt,
		LastUsedAt: found.LastUsedAt,
		RevokedAt:  found.RevokedAt,
		Active:     found.Live(time.Now()),
	}
}
