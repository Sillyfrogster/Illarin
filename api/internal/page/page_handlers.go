package page

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/notify"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ReaderVisibility(
	c *gin.Context,
	accounts *account.Service,
	requested *string,
) (work.ContentVisibility, bool) {
	if requested != nil {
		return work.ContentVisibility(*requested), true
	}
	preference, err := accounts.NSFWVisibility(c.Request.Context(), api.SessionToken(c))
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the content preference")
		return "", false
	}
	return work.ContentVisibility(preference), true
}

func (h *Handlers) GetAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := GetAssetParams{
		WorkingCopy: api.QueryFlag(q, "workingCopy"),
		Nsfw:        api.QueryText[GetAssetParamsNsfw](q, "nsfw"),
	}
	if q.Refused(c) {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	var requested *string
	if params.Nsfw != nil {
		value := string(*params.Nsfw)
		requested = &value
	}
	visibility, ok := ReaderVisibility(c, h.accounts, requested)
	if !ok {
		return
	}
	read := h.works.Detail
	if params.WorkingCopy != nil && *params.WorkingCopy {
		read = h.works.WorkingCopy
	}
	found, err := read(c.Request.Context(), id, viewerID, visibility)
	if errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such asset")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the asset")
		return
	}
	page, err := ToPage(found, visibility)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the asset")
		return
	}
	page.InstalledAppVersions, err = h.deliveries.InstalledAppVersions(c.Request.Context(), found.ID, found.InstallCapabilities)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the asset")
		return
	}
	if viewerID != nil && !found.IsOwner && found.Lifecycle != work.LifecycleDraft {
		watch, err := h.notifications.WatchOf(c.Request.Context(), *viewerID, found.ID)
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "could not read the asset")
			return
		}
		shown := notify.ToAPIWatch(watch)
		page.Watch = &shown
	}
	c.JSON(http.StatusOK, page)
}

func ToPage(found Detail, visibility work.ContentVisibility) (AssetDetail, error) {
	tags := make([]AssetTag, 0, len(found.Tags))
	for _, tag := range found.Tags {
		tags = append(tags, AssetTag{Label: tag.Label, Value: tag.Value})
	}
	media := ToImages(found.Media)
	blocks, err := block.ToBlocks(found.Kind, found.Blocks)
	if err != nil {
		return AssetDetail{}, err
	}
	addable := toAPIAddableBlocks(found.Kind, found.IsOwner)
	return AssetDetail{
		WorkingCopyVersion:    found.WorkingCopyVersion,
		UnpublishedChanges:    found.UnpublishedChanges,
		Id:                    found.ID,
		Kind:                  AssetDetailKind(found.Kind),
		Name:                  found.Name,
		Blurb:                 found.Blurb,
		Tags:                  tags,
		Creator:               found.Creator,
		Identifier:            found.Identifier,
		ExtensionDependencies: toAPIExtensionDependencies(found.Dependencies),
		InstalledAppVersions:  []string{},
		IsNsfw:                found.IsNSFW,
		Discovery:             AssetDetailDiscovery(found.Discovery),
		Lifecycle:             AssetDetailLifecycle(found.Lifecycle),
		IsOwner:               found.IsOwner,
		LinkedInstallOnly:     found.LinkedInstallOnly,
		AllowedApps:           apiAllowedApps(found.AllowedApps),
		EligibleApps:          apiEligibleApps(found.EligibleApps),
		Downloads:             ToDownloads(found.Downloads),
		AppTargets:            ToAppTargets(found.AppTargets),
		Original:              toAPIOriginalUpload(found.Original),
		CreatedAt:             found.CreatedAt,
		Blocks:                blocks,
		Media:                 media,
		Preview:               found.Preview,
		Readiness:             ToReadiness(found.Readiness),
		SealedBlocks:          countOrAbsent(found.SealedBlocks),
		AddableBlocks:         addable,
		Visibility:            AssetDetailVisibility(visibility),
		LatestUpdate:          toAPILatestUpdate(found.LatestUpdate),
		Withhold:              toAPIWithhold(found.Withhold),
	}, nil
}

func toAPIExtensionDependencies(dependencies []Dependency) []ExtensionDependency {
	out := make([]ExtensionDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		assets := make([]DependencyAsset, 0, len(dependency.Assets))
		for _, found := range dependency.Assets {
			assets = append(assets, DependencyAsset{Id: found.ID, Name: found.Name, Creator: found.Creator})
		}
		out = append(out, ExtensionDependency{Name: dependency.Name, Assets: assets})
	}
	return out
}

func ToImages(images []work.DetailImage) []AssetImage {
	media := make([]AssetImage, 0, len(images))
	for _, image := range images {
		media = append(media, AssetImage{
			Id:        image.ID,
			Role:      AssetImageRole(image.Role),
			IsCover:   image.IsCover,
			DetailUrl: image.DetailURL,
			ThumbUrl:  image.ThumbURL,
			Width:     image.Width,
			Height:    image.Height,
			Bytes:     int(image.Bytes),
		})
	}
	return media
}

func toAPILatestUpdate(recorded *work.Version) *RecordedVersion {
	if recorded == nil {
		return nil
	}
	served := ToRecordedVersion(*recorded)
	return &served
}

func apiAllowedApps(apps []string) []AssetDetailAllowedApps {
	result := make([]AssetDetailAllowedApps, len(apps))
	for i, app := range apps {
		result[i] = AssetDetailAllowedApps(app)
	}
	return result
}

func apiEligibleApps(apps []string) []AssetDetailEligibleApps {
	result := make([]AssetDetailEligibleApps, len(apps))
	for i, app := range apps {
		result[i] = AssetDetailEligibleApps(app)
	}
	return result
}

func ToDownloads(targets []format.Target) []DownloadTarget {
	downloads := make([]DownloadTarget, 0, len(targets))
	for _, target := range targets {
		roles := make([]DownloadRoleVerdict, 0, len(target.Roles))
		for _, role := range target.Roles {
			roles = append(roles, DownloadRoleVerdict{
				Role: string(role.Role), Label: role.Label,
				Verdict:     DownloadRoleVerdictVerdict(role.Verdict),
				Reason:      textOrNil(role.Reason),
				Destination: textOrNil(role.Destination),
				ShownBy:     listOrNil(role.ShownBy),
				Sample:      toAPIDownloadSample(role.Sample),
			})
		}
		downloads = append(downloads, DownloadTarget{
			Format: target.Format, Label: target.Label,
			Recommended: target.Recommended, Roles: roles,
		})
	}
	return downloads
}

func ToAppTargets(apps []format.AppTarget) []AppTarget {
	targets := make([]AppTarget, 0, len(apps))
	for _, app := range apps {
		targets = append(targets, AppTarget{Id: app.ID, Label: app.Label, Format: app.Format})
	}
	return targets
}

func toAPIDownloadSample(sample block.Sample) DownloadSample {
	converted := DownloadSample{Count: sample.Count}
	if len(sample.Texts) > 0 {
		texts := sample.Texts
		converted.Texts = &texts
	}
	if len(sample.Images) > 0 {
		images := make([]uuid.UUID, 0, len(sample.Images))
		for _, image := range sample.Images {
			images = append(images, image)
		}
		converted.Images = &images
	}
	return converted
}

func toAPIOriginalUpload(found *work.OriginalUpload) *OriginalUpload {
	if found == nil {
		return nil
	}
	return &OriginalUpload{
		Label: found.Label, MediaType: found.MediaType, ArrivedAt: found.ArrivedAt,
	}
}

func listOrNil(values []string) *[]string {
	if len(values) == 0 {
		return nil
	}
	return &values
}

func textOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func toAPIAddableBlocks(kind string, isOwner bool) *[]AddableBlock {
	if !isOwner {
		return nil
	}
	offers, ok := block.Offers(kind)
	if !ok {
		return nil
	}
	addable := make([]AddableBlock, 0, len(offers))
	for _, offer := range offers {
		choices := make([]AddableBlockChoice, 0, len(offer.Choices))
		for _, choice := range offer.Choices {
			choices = append(choices, AddableBlockChoice{
				Label: choice.Label, Type: block.ElementType(choice.Type),
			})
		}
		addable = append(addable, AddableBlock{
			Definition: string(offer.Definition),
			Title:      offer.Title,
			Summary:    offer.Summary,
			Group:      AddableBlockGroup(offer.Group),
			GroupTitle: offer.Group.Title(),
			Repeatable: offer.Repeatable,
			Choices:    choices,
		})
	}
	return &addable
}

func toAPIWithhold(found *Withhold) *AssetWithhold {
	if found == nil {
		return nil
	}
	return &AssetWithhold{Reason: found.Reason, At: found.At}
}

func countOrAbsent(count int) *int {
	if count == 0 {
		return nil
	}
	return &count
}

func ToRecordedVersion(recorded work.Version) RecordedVersion {
	return RecordedVersion{
		Id: recorded.ID, Number: recorded.Number,
		RecordedAt: recorded.RecordedAt, Initial: recorded.Initial,
		VersionLabel: recorded.VersionLabel,
		Summary:      recorded.Summary, Notes: recorded.Notes,
		NotesEditedAt:         recorded.NotesEditedAt,
		WithdrawnAt:           recorded.WithdrawnAt,
		WithdrawalExplanation: textOrNil(recorded.WithdrawalExplanation),
	}
}
