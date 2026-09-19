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

func ReaderNSFWPreference(
	c *gin.Context,
	accounts *account.Service,
	requested *string,
) (work.NSFWPreference, bool) {
	if requested != nil {
		return work.NSFWPreference(*requested), true
	}
	preference, err := accounts.NSFWPreference(c.Request.Context(), api.SessionToken(c))
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the content preference")
		return "", false
	}
	return work.NSFWPreference(preference), true
}

func (h *Handlers) GetWork(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	aliasPageQuery(q)
	params := GetWorkParams{
		DraftedChanges: api.QueryFlag(q, "draftedChanges"),
		Nsfw:           api.QueryText[GetWorkParamsNsfw](q, "nsfw"),
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
	preference, ok := ReaderNSFWPreference(c, h.accounts, requested)
	if !ok {
		return
	}
	read := h.works.Detail
	if params.DraftedChanges != nil && *params.DraftedChanges {
		read = h.works.DraftedChanges
	}
	found, err := read(c.Request.Context(), id, viewerID, preference)
	if errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such work")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the work")
		return
	}
	page, err := ToPage(found, preference)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the work")
		return
	}
	page.InstalledAppVersions, err = h.sends.InstalledAppVersions(c.Request.Context(), found.ID, found.InstallCapabilities)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not read the work")
		return
	}
	if viewerID != nil && !found.IsOwner && found.Lifecycle != work.LifecycleDraft {
		follow, err := h.notifications.FollowOf(c.Request.Context(), *viewerID, found.ID)
		if err != nil {
			api.Refuse(c, http.StatusInternalServerError, "could not read the work")
			return
		}
		shown := notify.ToAPIFollow(follow)
		page.Follow = &shown
	}
	c.JSON(http.StatusOK, page)
}

func ToPage(found Detail, preference work.NSFWPreference) (WorkDetail, error) {
	tags := make([]WorkTag, 0, len(found.Tags))
	for _, tag := range found.Tags {
		tags = append(tags, WorkTag{Label: tag.Label, Value: tag.Value})
	}
	media := ToImages(found.Media)
	blocks, err := block.ToBlocks(found.Type, found.Blocks)
	if err != nil {
		return WorkDetail{}, err
	}
	addable := toAPIAddableBlocks(found.Type, found.IsOwner)
	return WorkDetail{
		DraftedChangesVersion: found.DraftedChangesVersion,
		UnpublishedChanges:    found.UnpublishedChanges,
		Id:                    found.ID,
		Type:                  WorkDetailType(found.Type),
		Name:                  found.Name,
		Blurb:                 found.Blurb,
		Tags:                  tags,
		Creator:               found.Creator,
		Identifier:            found.Identifier,
		ExtensionDependencies: toAPIExtensionDependencies(found.Dependencies),
		InstalledAppVersions:  []string{},
		IsNsfw:                found.IsNSFW,
		Visibility:            WorkDetailVisibility(found.Visibility),
		Lifecycle:             WorkDetailLifecycle(found.Lifecycle),
		IsOwner:               found.IsOwner,
		HasPrivatePrompts:     found.HasPrivatePrompts,
		AllowedApps:           AppNames(found.AllowedApps),
		EligibleApps:          AppNames(found.EligibleApps),
		Downloads:             ToDownloads(found.Downloads),
		AppFormats:            ToAppFormats(found.AppFormats),
		Original:              toAPIOriginalUpload(found.Original),
		CreatedAt:             found.CreatedAt,
		Blocks:                blocks,
		Media:                 media,
		Preview:               found.Preview,
		Readiness:             ToReadiness(found.Readiness),
		PreservedPrompts:      countOrAbsent(found.PreservedPrompts),
		AddableBlocks:         addable,
		NSFWPreference:        WorkDetailNSFWPreference(preference),
		LatestVersion:         toAPILatestVersion(found.LatestVersion),
		Withhold:              toAPIWithhold(found.Withhold),
	}, nil
}

func toAPIExtensionDependencies(dependencies []Dependency) []ExtensionDependency {
	out := make([]ExtensionDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		works := make([]DependencyWork, 0, len(dependency.Works))
		for _, found := range dependency.Works {
			works = append(works, DependencyWork{Id: found.ID, Name: found.Name, Creator: found.Creator})
		}
		out = append(out, ExtensionDependency{Name: dependency.Name, Works: works})
	}
	return out
}

func ToImages(images []work.DetailImage) []WorkImage {
	media := make([]WorkImage, 0, len(images))
	for _, image := range images {
		media = append(media, WorkImage{
			Id:        image.ID,
			Role:      WorkImageRole(image.Role),
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

func toAPILatestVersion(recorded *work.Version) *RecordedVersion {
	if recorded == nil {
		return nil
	}
	served := ToRecordedVersion(*recorded)
	return &served
}

func ToDownloads(offered []format.Offered) []DownloadFormat {
	downloads := make([]DownloadFormat, 0, len(offered))
	for _, one := range offered {
		roles := make([]DownloadRoleVerdict, 0, len(one.Roles))
		for _, role := range one.Roles {
			roles = append(roles, DownloadRoleVerdict{
				Role: string(role.Role), Label: role.Label,
				Verdict:     DownloadRoleVerdictVerdict(role.Verdict),
				Reason:      textOrNil(role.Reason),
				Destination: textOrNil(role.Destination),
				ShownBy:     listOrNil(role.ShownBy),
				Sample:      toAPIDownloadSample(role.Sample),
			})
		}
		downloads = append(downloads, DownloadFormat{
			Format: one.Format, Label: one.Label,
			Recommended: one.Recommended, Roles: roles,
		})
	}
	return downloads
}

// AppNames labels each app id from the app registry
func AppNames(ids []string) []AppName {
	names := make([]AppName, len(ids))
	for i, id := range ids {
		names[i] = AppName{Id: id, Label: format.AppLabel(id)}
	}
	return names
}

func ToAppFormats(apps []format.AppFormat) []AppFormat {
	picked := make([]AppFormat, 0, len(apps))
	for _, app := range apps {
		picked = append(picked, AppFormat{Id: app.ID, Label: app.Label, Format: app.Format})
	}
	return picked
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

func toAPIAddableBlocks(workType string, isOwner bool) *[]AddableBlock {
	if !isOwner {
		return nil
	}
	offers, ok := block.Offers(workType)
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

func toAPIWithhold(found *Withhold) *WorkWithhold {
	if found == nil {
		return nil
	}
	return &WorkWithhold{Reason: found.Reason, At: found.At}
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
