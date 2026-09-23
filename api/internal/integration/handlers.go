package integration

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	integrations *Service
}

func NewHandlers(integrations *Service) *Handlers {
	return &Handlers{integrations: integrations}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/account/discord-channel", d.JSON, h.owned(api.SignedIn, h.read))
	routes.Handle(http.MethodPut, "/v1/account/discord-channel", d.JSON, h.owned(api.Verified, h.connect))
	routes.Handle(http.MethodDelete, "/v1/account/discord-channel", d.JSON, h.owned(api.SignedIn, h.disconnect))
	routes.Handle(http.MethodGet, "/v1/blog/discord-channel", d.JSON, h.blog(h.read))
	routes.Handle(http.MethodPut, "/v1/blog/discord-channel", d.JSON, h.blog(h.connect))
	routes.Handle(http.MethodDelete, "/v1/blog/discord-channel", d.JSON, h.blog(h.disconnect))
}

type channelAction func(c *gin.Context, owner *uuid.UUID)

func (h *Handlers) owned(who func(*gin.Context, string) (api.Account, bool), act channelAction) gin.HandlerFunc {
	return func(c *gin.Context) {
		account, ok := who(c, "connecting Discord")
		if ok {
			act(c, &account.ID)
		}
	}
}

func (h *Handlers) blog(act channelAction) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := api.Admin(c, "connecting the blog to Discord"); ok {
			act(c, nil)
		}
	}
}

func (h *Handlers) read(c *gin.Context, owner *uuid.UUID) {
	connected, err := h.integrations.Connected(c.Request.Context(), owner)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not read the Discord channel.")
		return
	}
	c.JSON(http.StatusOK, DiscordChannel{Connected: connected})
}

func (h *Handlers) connect(c *gin.Context, owner *uuid.UUID) {
	var request DiscordChannelRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.RefuseField(c, http.StatusBadRequest, "address", "Paste a Discord webhook address.")
		return
	}
	err := h.integrations.Connect(c.Request.Context(), owner, request.Address)
	switch {
	case errors.Is(err, ErrNotDiscord):
		api.RefuseField(c, http.StatusBadRequest, "address",
			"That isn't a Discord webhook address. Copy it from the channel's Integrations settings.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not save the Discord channel. Try again.")
	default:
		c.JSON(http.StatusOK, DiscordChannel{Connected: true})
	}
}

func (h *Handlers) disconnect(c *gin.Context, owner *uuid.UUID) {
	if err := h.integrations.Disconnect(c.Request.Context(), owner); err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Illarin could not remove the Discord channel. Try again.")
		return
	}
	c.JSON(http.StatusOK, DiscordChannel{Connected: false})
}
