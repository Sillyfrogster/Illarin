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

func (h *Handlers) StartLinkRequest(c *gin.Context) {
	noStoreLink(c)
	var request StartLinkRequest
	if !readLinkJSON(c, &request) {
		return
	}
	started, err := h.apps.Start(
		c.Request.Context(),
		api.RequestSource(c),
		startInput(
			request.ApplicationName, request.InstanceName, request.ApplicationVersion,
			request.ProtocolVersion, request.Capabilities, request.AcceptedTargets,
			request.Scopes,
		),
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusCreated, LinkRequest{
		DeviceCode: started.DeviceCode, UserCode: started.UserCode,
		VerificationUrl: started.VerifyURL, ExpiresAt: started.ExpiresAt,
		Interval: int(started.Interval.Seconds()),
	})
}

func (h *Handlers) StartLinkAuthorization(c *gin.Context) {
	noStoreLink(c)
	var request StartLinkAuthorization
	if !readLinkJSON(c, &request) {
		return
	}
	started, err := h.apps.StartAuthorization(
		c.Request.Context(),
		api.RequestSource(c),
		AuthorizationInput{
			StartInput: startInput(
				request.ApplicationName, request.InstanceName, request.ApplicationVersion,
				request.ProtocolVersion, request.Capabilities, request.AcceptedTargets,
				request.Scopes,
			),
			RedirectURI: request.RedirectUri, State: request.State,
			CodeChallenge:       request.CodeChallenge,
			CodeChallengeMethod: string(request.CodeChallengeMethod),
		},
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusCreated, LinkAuthorization{
		AuthorizationUrl: started.URL, ExpiresAt: started.ExpiresAt,
	})
}

func (h *Handlers) PollLinkRequest(c *gin.Context) {
	noStoreLink(c)
	var request PollLinkRequest
	if !readLinkJSON(c, &request) {
		return
	}
	grant, linked, err := h.apps.Poll(
		c.Request.Context(), api.RequestSource(c), request.DeviceCode,
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	if !linked {
		c.JSON(http.StatusOK, PendingLinkPollResult{Status: PendingLinkPollResultStatusPending})
		return
	}
	c.JSON(http.StatusOK, toAPIPollGrant(grant))
}

func (h *Handlers) GetLinkRequest(c *gin.Context) {
	userCode := c.Param("userCode")
	noStoreLink(c)
	creator, ok := api.Verified(c, "reviewing a link")
	if !ok {
		return
	}
	pending, err := h.apps.Pending(c.Request.Context(), creator.ID, string(userCode))
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIPendingDeviceLink(pending))
}

func (h *Handlers) ApproveLinkRequest(c *gin.Context) {
	userCode := c.Param("userCode")
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.Verified(c, "approving a link")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var decision DeviceLinkDecision
	if !readLinkJSON(c, &decision) {
		return
	}
	approved, err := h.apps.Approve(
		c.Request.Context(), creator.ID, string(userCode), decision.ApprovalToken,
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIPendingLink(approved))
}

func (h *Handlers) DenyLinkRequest(c *gin.Context) {
	userCode := c.Param("userCode")
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.Verified(c, "denying a link")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	var decision DeviceLinkDecision
	if !readLinkJSON(c, &decision) {
		return
	}
	if err := h.apps.Deny(
		c.Request.Context(), creator.ID, string(userCode), decision.ApprovalToken,
	); err != nil {
		h.linkingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetLinkAuthorization(c *gin.Context) {
	requestCode := c.Param("requestCode")
	noStoreLink(c)
	creator, ok := api.Verified(c, "reviewing a link")
	if !ok {
		return
	}
	pending, err := h.apps.PendingAuthorization(
		c.Request.Context(), creator.ID, string(requestCode),
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPIPendingLink(pending))
}

func (h *Handlers) ApproveLinkAuthorization(c *gin.Context) {
	requestCode := c.Param("requestCode")
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.Verified(c, "approving a link")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	redirect, err := h.apps.ApproveAuthorization(
		c.Request.Context(), creator.ID, string(requestCode),
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, LinkRedirect{RedirectUrl: redirect.URL})
}

func (h *Handlers) DenyLinkAuthorization(c *gin.Context) {
	requestCode := c.Param("requestCode")
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.Verified(c, "denying a link")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	redirect, err := h.apps.DenyAuthorization(
		c.Request.Context(), creator.ID, string(requestCode),
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, LinkRedirect{RedirectUrl: redirect.URL})
}

func (h *Handlers) ExchangeLinkAuthorization(c *gin.Context) {
	noStoreLink(c)
	var request ExchangeLinkAuthorization
	if !readLinkJSON(c, &request) {
		return
	}
	grant, err := h.apps.Exchange(
		c.Request.Context(), api.RequestSource(c), request.AuthorizationCode,
		request.CodeVerifier, request.RedirectUri,
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPITokenGrant(grant))
}

func (h *Handlers) RefreshInstanceToken(c *gin.Context) {
	noStoreLink(c)
	var request RefreshInstanceToken
	if !readLinkJSON(c, &request) {
		return
	}
	grant, err := h.apps.Refresh(
		c.Request.Context(), api.RequestSource(c), request.RefreshToken,
	)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAPITokenGrant(grant))
}

func (h *Handlers) ListInstances(c *gin.Context) {
	noStoreLink(c)
	creator, ok := api.SignedIn(c, "managing linked instances")
	if !ok {
		return
	}
	found, err := h.apps.List(c.Request.Context(), creator.ID)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	counts, err := h.sends.LibraryCountsByInstance(c.Request.Context(), creator.ID)
	if err != nil {
		h.linkingError(c, err)
		return
	}
	items := make([]ManagedInstance, 0, len(found))
	for _, instance := range found {
		items = append(items, toAPIManagedInstance(instance, counts[instance.ID]))
	}
	c.JSON(http.StatusOK, LinkedInstanceList{Items: items})
}

func (h *Handlers) RevokeInstance(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if !api.FromIllarin(c) {
		return
	}
	noStoreLink(c)
	creator, ok := api.SignedIn(c, "managing linked instances")
	if !ok || !api.RequireBrowser(c, h.apps.BrowserOrigin()) {
		return
	}
	if err := h.apps.Revoke(c.Request.Context(), creator.ID, id); err != nil {
		h.linkingError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetInstance(c *gin.Context) {
	noStoreLink(c)
	instance, ok := h.instance(c, "")
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toAPIInstance(instance))
}

func (h *Handlers) UpdateInstance(c *gin.Context) {
	noStoreLink(c)
	instance, ok := h.instance(c, "")
	if !ok {
		return
	}
	var request UpdateInstance
	if !readLinkJSON(c, &request) {
		return
	}
	updated, err := h.apps.UpdateDeclaration(c.Request.Context(), instance.ID, Declaration{
		ApplicationName: instance.ApplicationName, InstanceName: instance.InstanceName,
		ApplicationVersion: applicationVersion(request.ApplicationVersion),
		ProtocolVersion:    int(request.ProtocolVersion),
		Capabilities:       stringsFromCapabilities(request.Capabilities),
		AcceptedTargets:    stringsFromTargets(request.AcceptedTargets),
	})
	if err != nil {
		h.linkingError(c, err)
		return
	}
	updated.UserID = instance.UserID
	c.JSON(http.StatusOK, toAPIInstance(updated))
}

func (h *Handlers) instance(c *gin.Context, needs Scope) (Instance, bool) {
	found, err := h.apps.Authenticate(c.Request.Context(), bearerToken(c), needs)
	switch {
	case errors.Is(err, ErrInstanceCredential):
		api.Refuse(c, http.StatusUnauthorized, "This access token is not live.")
		return Instance{}, false
	case errors.Is(err, ErrInstanceMissingScope):
		api.Refuse(c, http.StatusForbidden, "This instance was not granted that scope.")
		return Instance{}, false
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not read the linked instance.")
		return Instance{}, false
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

func (h *Handlers) linkingError(c *gin.Context, err error) {
	var delay *PollDelayError
	var limited *RateLimitError
	switch {
	case errors.As(err, &delay):
		c.Header("Retry-After", strconv.Itoa(int(delay.After.Seconds())))
		api.Refuse(c, http.StatusTooManyRequests, "slow_down")
	case errors.Is(err, ErrTooManyCodes):
		c.Header("Retry-After", strconv.Itoa(int(timeHourSeconds)))
		api.Refuse(c, http.StatusTooManyRequests, "Too many codes were entered. Try again later.")
	case errors.As(err, &limited):
		seconds := int((limited.After + time.Second - 1) / time.Second)
		c.Header("Retry-After", strconv.Itoa(seconds))
		api.Refuse(c, http.StatusTooManyRequests, "Too many link requests. Try again later.")
	case errors.Is(err, ErrInvalidName):
		api.Refuse(c, http.StatusBadRequest, "Name the application and this installation in 64 characters or fewer.")
	case errors.Is(err, ErrInvalidDeclaration):
		api.Refuse(c, http.StatusBadRequest, "The instance declaration is not valid.")
	case errors.Is(err, ErrInvalidScopes):
		api.Refuse(c, http.StatusBadRequest, "Ask for asset:receive, library:sync, or both, once each.")
	case errors.Is(err, ErrInvalidRedirect):
		api.Refuse(c, http.StatusBadRequest, "Use an exact 127.0.0.1 or [::1] callback with an explicit port.")
	case errors.Is(err, ErrInvalidPKCE):
		api.Refuse(c, http.StatusBadRequest, "invalid_grant")
	case errors.Is(err, ErrAccessDenied):
		api.Refuse(c, http.StatusBadRequest, "access_denied")
	case errors.Is(err, ErrLinkExpired):
		api.Refuse(c, http.StatusBadRequest, "expired_token")
	case errors.Is(err, ErrLinkRequestNotFound):
		api.Refuse(c, http.StatusNotFound, "No pending link request matches that code.")
	case errors.Is(err, ErrRefreshReuse):
		api.Refuse(c, http.StatusUnauthorized, "This instance was revoked because a replaced refresh token was reused.")
	case errors.Is(err, ErrInstanceCredential):
		api.Refuse(c, http.StatusUnauthorized, "This token is not live.")
	case errors.Is(err, ErrInstanceNotFound):
		api.Refuse(c, http.StatusNotFound, "No live linked instance has that id.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not complete the link request.")
	}
}

const maxLinkBodyBytes = 4 << 10

func readLinkJSON(c *gin.Context, destination any) bool {
	return api.ReadBoundedJSON(c, destination, maxLinkBodyBytes, "The link request is too large.")
}

func noStoreLink(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

const timeHourSeconds = 60 * 60

func startInput(
	applicationName ApplicationName,
	instanceName InstanceName,
	version *ApplicationVersion,
	protocol LinkProtocolVersion,
	capabilities InstanceCapabilities,
	targets AcceptedTargets,
	scopes Scopes,
) StartInput {
	return StartInput{
		Declaration: Declaration{
			ApplicationName: string(applicationName), InstanceName: string(instanceName),
			ApplicationVersion: applicationVersion(version), ProtocolVersion: int(protocol),
			Capabilities:    stringsFromCapabilities(capabilities),
			AcceptedTargets: stringsFromTargets(targets),
		},
		Scopes: copyScopes(scopes),
	}
}

func applicationVersion(value *ApplicationVersion) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func stringsFromCapabilities(values InstanceCapabilities) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}

func stringsFromTargets(values AcceptedTargets) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = string(value)
	}
	return result
}

func copyScopes(scopes []Scope) []Scope {
	return append(make([]Scope, 0, len(scopes)), scopes...)
}

func toAPIPendingLink(pending Pending) PendingLink {
	return PendingLink{
		ApplicationName:    ApplicationName(pending.ApplicationName),
		InstanceName:       InstanceName(pending.InstanceName),
		ApplicationVersion: apiApplicationVersion(pending.ApplicationVersion),
		ProtocolVersion:    LinkProtocolVersion(pending.ProtocolVersion),
		Capabilities:       apiCapabilities(pending.Capabilities),
		AcceptedTargets:    apiTargets(pending.AcceptedTargets),
		Scopes:             copyScopes(pending.Scopes), ExpiresAt: pending.ExpiresAt,
	}
}

func toAPIPendingDeviceLink(pending Pending) PendingDeviceLink {
	base := toAPIPendingLink(pending)
	return PendingDeviceLink{
		ApplicationName: base.ApplicationName, InstanceName: base.InstanceName,
		ApplicationVersion: base.ApplicationVersion,
		ProtocolVersion:    base.ProtocolVersion, Capabilities: base.Capabilities,
		AcceptedTargets: base.AcceptedTargets, Scopes: base.Scopes,
		ExpiresAt: base.ExpiresAt, ApprovalToken: pending.ApprovalToken,
	}
}

func toAPIPollGrant(grant TokenGrant) LinkedLinkPollResult {
	return LinkedLinkPollResult{
		Status: LinkedLinkPollResultStatusLinked, AccessToken: AccessToken(grant.AccessToken),
		AccessTokenExpiresAt: grant.AccessTokenExpiresAt,
		RefreshToken:         RefreshToken(grant.RefreshToken),
		Instance:             toAPIInstance(grant.Instance),
	}
}

func toAPITokenGrant(grant TokenGrant) InstanceTokenGrant {
	return InstanceTokenGrant{
		AccessToken:          grant.AccessToken,
		AccessTokenExpiresAt: grant.AccessTokenExpiresAt,
		RefreshToken:         grant.RefreshToken,
		Instance:             toAPIInstance(grant.Instance),
	}
}

func toAPIInstance(instance Instance) LinkedInstance {
	var version *int
	if instance.ProtocolVersion > 0 {
		protocol := instance.ProtocolVersion
		version = &protocol
	}
	return LinkedInstance{
		Id:                 instance.ID,
		ApplicationName:    instance.ApplicationName,
		InstanceName:       instance.InstanceName,
		ApplicationVersion: apiApplicationVersion(instance.ApplicationVersion),
		ProtocolVersion:    version,
		Capabilities:       append([]string{}, instance.Capabilities...),
		AcceptedTargets:    append([]string{}, instance.AcceptedTargets...),
		Prefix:             instance.Prefix, Scopes: copyScopes(instance.Scopes),
		LinkedAt: instance.LinkedAt, LastSeenAt: instance.LastSeenAt,
		RevokedAt: instance.RevokedAt,
	}
}

func toAPIManagedInstance(
	instance Instance,
	counts LibraryCounts,
) ManagedInstance {
	base := toAPIInstance(instance)
	return ManagedInstance{
		Id: base.Id, ApplicationName: base.ApplicationName,
		InstanceName: base.InstanceName, ApplicationVersion: base.ApplicationVersion,
		ProtocolVersion: base.ProtocolVersion, Capabilities: base.Capabilities,
		AcceptedTargets: base.AcceptedTargets, Prefix: base.Prefix,
		Scopes: base.Scopes, LinkedAt: base.LinkedAt, LastSeenAt: base.LastSeenAt,
		RevokedAt: base.RevokedAt, Installed: counts.Installed,
		UpdatesAvailable: counts.UpdatesAvailable,
	}
}

func apiApplicationVersion(value string) *ApplicationVersion {
	if value == "" {
		return nil
	}
	version := ApplicationVersion(value)
	return &version
}

func apiCapabilities(values []string) InstanceCapabilities {
	result := make(InstanceCapabilities, len(values))
	for index, value := range values {
		result[index] = CapabilityId(value)
	}
	return result
}

func apiTargets(values []string) AcceptedTargets {
	result := make(AcceptedTargets, len(values))
	for index, value := range values {
		result[index] = ExportTargetId(value)
	}
	return result
}
