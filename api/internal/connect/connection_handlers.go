package connect

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) StartConnectionRequest(c *gin.Context) {
	noStore(c)
	var request StartConnectionRequest
	if !readConnectJSON(c, &request) {
		return
	}
	started, err := h.apps.Start(c.Request.Context(), api.RequestSource(c), StartInput{
		AppName: request.AppName, Name: request.Name,
		Capabilities: capabilitiesOf(
			request.AppVersion, request.ProtocolVersion, request.Capabilities, request.AcceptedFormats,
		),
		Permissions: request.Permissions,
	})
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, ConnectionRequest{
		DeviceCode: started.DeviceCode, UserCode: started.UserCode,
		VerificationUrl: started.VerifyURL, ExpiresAt: started.ExpiresAt,
		Interval: int(started.Interval.Seconds()),
	})
}

func (h *Handlers) StartConnectionAuthorization(c *gin.Context) {
	noStore(c)
	var request StartConnectionAuthorization
	if !readConnectJSON(c, &request) {
		return
	}
	started, err := h.apps.StartAuthorization(c.Request.Context(), api.RequestSource(c), AuthorizationInput{
		StartInput: StartInput{
			AppName: request.AppName, Name: request.Name,
			Capabilities: capabilitiesOf(
				request.AppVersion, request.ProtocolVersion, request.Capabilities, request.AcceptedFormats,
			),
			Permissions: request.Permissions,
		},
		RedirectURI: request.RedirectUri, State: request.State,
		CodeChallenge: request.CodeChallenge, CodeChallengeMethod: request.CodeChallengeMethod,
	})
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, ConnectionAuthorization{
		AuthorizationUrl: started.URL, UserCode: started.UserCode, ExpiresAt: started.ExpiresAt,
	})
}

func (h *Handlers) PollConnectionRequest(c *gin.Context) {
	noStore(c)
	var request PollConnectionRequest
	if !readConnectJSON(c, &request) {
		return
	}
	issued, connected, err := h.apps.Poll(
		c.Request.Context(), api.RequestSource(c), request.DeviceCode,
	)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	if !connected {
		c.JSON(http.StatusOK, PendingPoll{Status: "pending"})
		return
	}
	c.JSON(http.StatusOK, ConnectedPoll{
		Status: connectedStatus(c), AccessToken: issued.AccessToken,
		AccessTokenExpiresAt: issued.AccessTokenExpiresAt,
		RefreshToken:         issued.RefreshToken,
		ConnectedApp:         toAPIConnectedApp(issued.ConnectedApp),
	})
}

func (h *Handlers) GetConnectionRequest(c *gin.Context) {
	noStore(c)
	creator, ok := api.Verified(c, "reviewing a connection request")
	if !ok {
		return
	}
	pending, err := h.apps.Pending(c.Request.Context(), creator.ID, c.Param("userCode"))
	if err != nil {
		h.connectionError(c, err)
		return
	}
	h.showPendingCode(c, pending)
}

func (h *Handlers) showPendingCode(c *gin.Context, pending Pending) {
	shown := toAPIPendingConnection(pending)
	c.JSON(http.StatusOK, PendingCodeConnection{
		AppName: shown.AppName, Name: shown.Name, AppVersion: shown.AppVersion,
		ProtocolVersion: shown.ProtocolVersion, Capabilities: shown.Capabilities,
		AcceptedFormats: shown.AcceptedFormats, Permissions: shown.Permissions,
		ExpiresAt: shown.ExpiresAt, ApprovalToken: pending.ApprovalToken,
	})
}

func (h *Handlers) ApproveConnectionRequest(c *gin.Context) {
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.Verified(c, "approving a connection request")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var decision ConnectionDecision
	if !readConnectJSON(c, &decision) {
		return
	}
	approved, err := h.apps.Approve(
		c.Request.Context(), creator.ID, c.Param("userCode"), decision.ApprovalToken,
	)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIPendingConnection(approved))
}

func (h *Handlers) DenyConnectionRequest(c *gin.Context) {
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.Verified(c, "denying a connection request")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var decision ConnectionDecision
	if !readConnectJSON(c, &decision) {
		return
	}
	if err := h.apps.Deny(
		c.Request.Context(), creator.ID, c.Param("userCode"), decision.ApprovalToken,
	); err != nil {
		h.connectionError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetConnectionAuthorization(c *gin.Context) {
	noStore(c)
	creator, ok := api.Verified(c, "reviewing a connection request")
	if !ok {
		return
	}
	pending, err := h.apps.PendingAuthorization(
		c.Request.Context(), creator.ID, c.Param("requestCode"), c.Query("userCode"),
	)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	h.showPendingCode(c, pending)
}

func (h *Handlers) ApproveConnectionAuthorization(c *gin.Context) {
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.Verified(c, "approving a connection request")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var decision ConnectionDecision
	if !readConnectJSON(c, &decision) {
		return
	}
	redirect, err := h.apps.ApproveAuthorization(
		c.Request.Context(), creator.ID, c.Param("requestCode"), decision.ApprovalToken,
	)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, ConnectionRedirect{RedirectUrl: redirect.URL})
}

func (h *Handlers) DenyConnectionAuthorization(c *gin.Context) {
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.Verified(c, "denying a connection request")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var decision ConnectionDecision
	if !readConnectJSON(c, &decision) {
		return
	}
	redirect, err := h.apps.DenyAuthorization(
		c.Request.Context(), creator.ID, c.Param("requestCode"), decision.ApprovalToken,
	)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, ConnectionRedirect{RedirectUrl: redirect.URL})
}

func (h *Handlers) ExchangeConnectionAuthorization(c *gin.Context) {
	noStore(c)
	var request ExchangeConnectionAuthorization
	if !readConnectJSON(c, &request) {
		return
	}
	issued, err := h.apps.Exchange(
		c.Request.Context(), api.RequestSource(c), request.AuthorizationCode,
		request.CodeVerifier, request.RedirectUri,
	)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIAppCredentials(issued))
}

func (h *Handlers) RefreshAppCredentials(c *gin.Context) {
	noStore(c)
	var request RefreshAppCredentials
	if !readConnectJSON(c, &request) {
		return
	}
	issued, err := h.apps.Refresh(c.Request.Context(), api.RequestSource(c), request.RefreshToken)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIAppCredentials(issued))
}

func (h *Handlers) ListConnectedApps(c *gin.Context) {
	noStore(c)
	creator, ok := api.SignedIn(c, "managing connected apps")
	if !ok {
		return
	}
	found, err := h.apps.List(c.Request.Context(), creator.ID)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	counts, err := h.sends.LibraryCountsByApp(c.Request.Context(), creator.ID)
	if err != nil {
		h.connectionError(c, err)
		return
	}
	items := make([]ManagedConnectedApp, 0, len(found))
	for _, app := range found {
		items = append(items, toAPIManagedConnectedApp(app, counts[app.ID]))
	}
	c.JSON(http.StatusOK, ConnectedAppList{Items: items})
}

func (h *Handlers) RevokeConnectedApp(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if !api.FromIllarin(c) {
		return
	}
	noStore(c)
	creator, ok := api.SignedIn(c, "managing connected apps")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	if err := h.apps.Revoke(c.Request.Context(), creator.ID, id); err != nil {
		h.connectionError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetConnectedApp(c *gin.Context) {
	noStore(c)
	app, ok := h.connectedApp(c, "")
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toAPIConnectedApp(app))
}

func (h *Handlers) UpdateCapabilities(c *gin.Context) {
	noStore(c)
	app, ok := h.connectedApp(c, "")
	if !ok {
		return
	}
	var request UpdateCapabilities
	if !readConnectJSON(c, &request) {
		return
	}
	updated, err := h.apps.UpdateCapabilities(c.Request.Context(), app, capabilitiesOf(
		request.AppVersion, request.ProtocolVersion, request.Capabilities, request.AcceptedFormats,
	))
	if err != nil {
		h.connectionError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIConnectedApp(updated))
}

func (h *Handlers) connectedApp(c *gin.Context, needs Permission) (ConnectedApp, bool) {
	found, err := h.apps.Authenticate(c.Request.Context(), bearerToken(c), needs)
	switch {
	case errors.Is(err, ErrNotLive):
		api.Refuse(c, http.StatusUnauthorized, "This access token is not live.")
		return ConnectedApp{}, false
	case errors.Is(err, ErrMissingPermission):
		api.Refuse(c, http.StatusForbidden, "This connected app was not granted that permission.")
		return ConnectedApp{}, false
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not read the connected app.")
		return ConnectedApp{}, false
	}
	return found, true
}

func bearerToken(c *gin.Context) string {
	scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func (h *Handlers) connectionError(c *gin.Context, err error) {
	var delay *PollDelayError
	var limited *RateLimitError
	switch {
	case errors.As(err, &delay):
		c.Header("Retry-After", strconv.Itoa(int(delay.After.Seconds())))
		api.Refuse(c, http.StatusTooManyRequests, "slow_down")
	case errors.Is(err, ErrTooManyCodes):
		c.Header("Retry-After", strconv.Itoa(int(time.Hour.Seconds())))
		api.Refuse(c, http.StatusTooManyRequests, "Too many codes were entered. Try again later.")
	case errors.As(err, &limited):
		seconds := int((limited.After + time.Second - 1) / time.Second)
		c.Header("Retry-After", strconv.Itoa(seconds))
		api.Refuse(c, http.StatusTooManyRequests, "Too many connection requests. Try again later.")
	case errors.Is(err, ErrInvalidName):
		api.Refuse(c, http.StatusBadRequest, "Give appName and name, each 64 characters or fewer.")
	case errors.Is(err, ErrInvalidCapabilities):
		api.Refuse(c, http.StatusBadRequest, "The capabilities are not valid.")
	case errors.Is(err, ErrInvalidPermissions):
		api.Refuse(c, http.StatusBadRequest, "Ask for work:receive, library:sync, or both, once each.")
	case errors.Is(err, ErrInvalidRedirect):
		api.Refuse(c, http.StatusBadRequest, "Use an exact 127.0.0.1 or [::1] callback with an explicit port.")
	case errors.Is(err, ErrInvalidPKCE):
		api.Refuse(c, http.StatusBadRequest, "invalid_grant")
	case errors.Is(err, ErrAccessDenied):
		api.Refuse(c, http.StatusBadRequest, "access_denied")
	case errors.Is(err, ErrRequestExpired):
		api.Refuse(c, http.StatusBadRequest, "expired_token")
	case errors.Is(err, ErrRequestNotFound):
		api.Refuse(c, http.StatusNotFound, "No pending connection request matches that code.")
	case errors.Is(err, ErrRefreshReuse):
		api.Refuse(c, http.StatusUnauthorized, "This connected app was revoked because a replaced refresh token was reused.")
	case errors.Is(err, ErrNotLive):
		api.Refuse(c, http.StatusUnauthorized, "This token is not live.")
	case errors.Is(err, ErrAppNotFound):
		api.Refuse(c, http.StatusNotFound, "No live connected app has that id.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not complete the connection request.")
	}
}

const maxConnectBodyBytes = 4 << 10

func readConnectJSON(c *gin.Context, destination any) bool {
	return api.ReadBoundedJSON(c, destination, maxConnectBodyBytes, "The request is too large.")
}

func noStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

func capabilitiesOf(appVersion *string, protocol int, declared, formats []string) Capabilities {
	version := ""
	if appVersion != nil {
		version = *appVersion
	}
	return Capabilities{
		AppVersion: version, ProtocolVersion: protocol,
		Declared: declared, AcceptedFormats: formats,
	}
}

func optionalVersion(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func toAPIPendingConnection(pending Pending) PendingConnection {
	return PendingConnection{
		AppName: pending.AppName, Name: pending.Name,
		AppVersion:      optionalVersion(pending.AppVersion),
		ProtocolVersion: pending.ProtocolVersion,
		Capabilities:    append([]string{}, pending.Declared...),
		AcceptedFormats: append([]string{}, pending.AcceptedFormats...),
		Permissions:     append([]Permission{}, pending.Permissions...),
		ExpiresAt:       pending.ExpiresAt,
	}
}

func toAPIAppCredentials(issued Credentials) AppCredentials {
	return AppCredentials{
		AccessToken:          issued.AccessToken,
		AccessTokenExpiresAt: issued.AccessTokenExpiresAt,
		RefreshToken:         issued.RefreshToken,
		ConnectedApp:         toAPIConnectedApp(issued.ConnectedApp),
	}
}

func toAPIConnectedApp(app ConnectedApp) ConnectedAppDetail {
	var protocol *int
	if app.ProtocolVersion > 0 {
		version := app.ProtocolVersion
		protocol = &version
	}
	return ConnectedAppDetail{
		Id: app.ID, AppName: app.AppName, Name: app.Name,
		AppVersion: optionalVersion(app.AppVersion), ProtocolVersion: protocol,
		Capabilities:    append([]string{}, app.Declared...),
		AcceptedFormats: append([]string{}, app.AcceptedFormats...),
		Prefix:          app.Prefix, Permissions: append([]Permission{}, app.Permissions...),
		ConnectedAt: app.ConnectedAt, LastSeenAt: app.LastSeenAt, RevokedAt: app.RevokedAt,
	}
}

func toAPIManagedConnectedApp(app ConnectedApp, counts LibraryCounts) ManagedConnectedApp {
	shown := toAPIConnectedApp(app)
	return ManagedConnectedApp{
		Id: shown.Id, AppName: shown.AppName, Name: shown.Name, AppVersion: shown.AppVersion,
		ProtocolVersion: shown.ProtocolVersion, Capabilities: shown.Capabilities,
		AcceptedFormats: shown.AcceptedFormats, Prefix: shown.Prefix,
		Permissions: shown.Permissions, ConnectedAt: shown.ConnectedAt,
		LastSeenAt: shown.LastSeenAt, RevokedAt: shown.RevokedAt,
		Installed: counts.Installed, UpdatesAvailable: counts.UpdatesAvailable,
	}
}
