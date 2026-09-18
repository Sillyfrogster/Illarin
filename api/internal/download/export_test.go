package download_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/download"
	"github.com/google/uuid"
)

func targetLine(t *testing.T, menu []apitest.DownloadTarget, formatID string) apitest.DownloadTarget {
	t.Helper()
	for _, target := range menu {
		if target.Format == formatID {
			return target
		}
	}
	t.Fatalf("%s is not on the menu: %+v", formatID, menu)
	return apitest.DownloadTarget{}
}

func losses(target apitest.DownloadTarget) []apitest.RoleVerdict {
	lost := make([]apitest.RoleVerdict, 0, len(target.Roles))
	for _, role := range target.Roles {
		if role.Verdict != "carried" {
			lost = append(lost, role)
		}
	}
	return lost
}

func TestTheLossReportIsCheckedAgainstTheWorkAndNotTheFormat(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)

	plain := apitest.DownloadMenu(t, r, session, workID)
	if len(plain) != 3 {
		t.Fatalf("menu = %+v, want all three character formats", plain)
	}
	if lost := losses(targetLine(t, plain, "chara_card_v2")); len(lost) != 0 {
		t.Fatalf("CCv2 reported %+v for a card that has none of what it drops", lost)
	}

	apitest.GiveExpressions(t, r, session, workID)

	withImages := apitest.DownloadMenu(t, r, session, workID)
	lost := losses(targetLine(t, withImages, "chara_card_v2"))
	if len(lost) != 1 || lost[0].Role != "expressions" || lost[0].Verdict != "dropped" {
		t.Fatalf("CCv2 losses = %+v, want the expressions dropped", lost)
	}
	if lost[0].Sample.Count != 1 || len(lost[0].Sample.Images) != 1 {
		t.Errorf("sample = %+v, want the picture that is at stake", lost[0].Sample)
	}
	if len(withImages) != 3 {
		t.Fatalf("menu = %+v, want the lossy target still offered", withImages)
	}
}

func TestTheRecommendationIsTheFormatWhoseImagesReachEveryApp(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.GiveExpressions(t, r, session, workID)
	apitest.GivePictures(t, r, session, workID, "gallery", "gallery")

	menu := apitest.DownloadMenu(t, r, session, workID)
	recommended := ""
	for _, target := range menu {
		if target.Recommended {
			recommended = target.Format
		}
	}
	if recommended != "charx" {
		t.Fatalf("recommended = %q, want the format whose images every app reads", recommended)
	}
	if len(losses(targetLine(t, menu, "charx"))) >
		len(losses(targetLine(t, menu, "chara_card_v3"))) {
		t.Fatal("CharX lost more here, so least loss would have chosen CCv3 anyway")
	}
	inline := roleVerdictNamed(t, targetLine(t, menu, "chara_card_v3"), "gallery")
	if inline.Verdict != "carried" || inline.Destination == "" {
		t.Fatalf("gallery verdict = %+v, want carried with a destination note", inline)
	}
	archived := roleVerdictNamed(t, targetLine(t, menu, "charx"), "gallery")
	if archived.Verdict != "carried" || archived.Destination != "" {
		t.Fatalf("gallery verdict = %+v, want carried with nothing to warn about", archived)
	}
}

func roleVerdictNamed(t *testing.T, target apitest.DownloadTarget, role string) apitest.RoleVerdict {
	t.Helper()
	for _, found := range target.Roles {
		if found.Role == role {
			return found
		}
	}
	t.Fatalf("%s has no verdict for %s: %+v", target.Format, role, target.Roles)
	return apitest.RoleVerdict{}
}

func TestEachDownloadIsNamedAfterItsFormat(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.PublishCharacter(t, r, session, workID)

	seen := make(map[string]string)
	for _, target := range []string{"chara_card_v2", "chara_card_v3", "charx"} {
		download := apitest.Send(t, r, httptest.NewRequest(
			http.MethodGet, "/download/"+workID+"/"+target, nil,
		))
		if download.Code != http.StatusOK {
			t.Fatalf("%s status = %d: %s", target, download.Code, download.Body.String())
		}
		filename := download.Header().Get("Content-Disposition")
		if held, taken := seen[filename]; taken {
			t.Fatalf("%s and %s both arrive as %s", held, target, filename)
		}
		seen[filename] = target
	}
}

func TestTheDownloadMenuReadsTheSameForItsOwnerAndAStranger(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.PublishCharacter(t, r, session, workID)

	owner, stranger := apitest.DownloadMenu(t, r, session, workID), apitest.DownloadMenu(t, r, nil, workID)
	if !json.Valid(mustJSON(t, owner)) || string(mustJSON(t, owner)) != string(mustJSON(t, stranger)) {
		t.Fatalf("the owner reads %s and a reader reads %s",
			mustJSON(t, owner), mustJSON(t, stranger))
	}
}

func TestTheOriginalUploadStandsApartAndOnlyWhereThereIsOne(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)

	uploaded := apitest.FetchStartedWork(t, r, session, workID)
	if uploaded.Original == nil {
		t.Fatal("an uploaded card has no original upload group")
	}
	if uploaded.Original.Label != "Character Card V3" || uploaded.Original.ArrivedAt == "" {
		t.Fatalf("original = %+v, want it labelled by what it is and when it came", uploaded.Original)
	}
	for _, target := range uploaded.Downloads {
		if target.Format == "raw" {
			t.Fatal("the upload was listed beside the generated downloads")
		}
	}

	built := apitest.StartCharacter(t, r, session)
	fromNothing := apitest.FetchStartedWork(t, r, session, built.ID)
	if fromNothing.Original != nil {
		t.Fatalf("a work built from nothing carries %+v", fromNothing.Original)
	}
	if len(fromNothing.Downloads) != 3 {
		t.Fatalf("menu = %+v, want all three character targets", fromNothing.Downloads)
	}
}

func TestTheSummaryIsWrittenWithTheChangeAndPublishingComputesNothing(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewCharacterIngestRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)

	before := apitest.SummaryComputedAt(t, pool, workID)
	apitest.GiveExpressions(t, r, session, workID)
	afterEdit := apitest.SummaryComputedAt(t, pool, workID)
	if !afterEdit.After(before) {
		t.Fatal("editing a block left the export summary where it was")
	}

	apitest.PublishCharacter(t, r, session, workID)
	if afterPublish := apitest.SummaryComputedAt(t, pool, workID); !afterPublish.Equal(afterEdit) {
		t.Fatal("publishing recomputed the export summary")
	}
}

func TestHidingABlockLeavesTheDownloadAlone(t *testing.T) {
	t.Parallel()
	r, session, works, pool := harness.NewCharacterIngestRouterWithPool(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.GiveExpressions(t, r, session, workID)
	apitest.PublishCharacter(t, r, session, workID)

	before := apitest.SummaryComputedAt(t, pool, workID)
	page := apitest.FetchStartedWork(t, r, session, workID)
	arrangement := make([]apitest.ArrangedBlock, 0, len(page.Blocks))
	for _, holder := range page.Blocks {
		arrangement = append(arrangement, apitest.ArrangedBlock{
			ID: holder.ID, Hidden: holder.Definition == "expressions", Width: holder.Width,
		})
	}
	arranged := apitest.ArrangeBlocks(t, r, session, workID, arrangement)
	if arranged.Code != http.StatusOK {
		t.Fatalf("hide the expressions: %d %s", arranged.Code, arranged.Body.String())
	}
	if after := apitest.SummaryComputedAt(t, pool, workID); !after.Equal(before) {
		t.Fatal("hiding a block moved the export half of the summary")
	}

	export, err := download.NewService(works.Pool(), works).OpenExport(
		context.Background(), uuid.MustParse(workID), nil, "chara_card_v3", nil,
	)
	if err != nil {
		t.Fatalf("export a card with a hidden block: %v", err)
	}
	if !apitest.ContainsBytes(export.Body, []byte("emotion")) {
		t.Fatal("a hidden block's content did not travel in the download")
	}
}

func TestEachAppIsOfferedTheFormatItsImagesReach(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.GivePictures(t, r, session, workID, "gallery", "gallery")

	offered := appTargetsFor(t, r, session, workID)
	if len(offered) == 0 {
		t.Fatal("no application was offered a format")
	}
	for _, app := range offered {
		if app.Format != "charx" {
			t.Errorf("%s = %q, want the format whose gallery it shows", app.ID, app.Format)
		}
	}
}

func TestAnAppIsNamedBesideTheDestinationItShows(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterIngestRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.GivePictures(t, r, session, workID, "gallery", "gallery")

	menu := apitest.DownloadMenu(t, r, session, workID)
	inline := roleVerdictNamed(t, targetLine(t, menu, "chara_card_v3"), "gallery")
	if inline.Destination == "" {
		t.Fatal("the inline gallery lost its destination note")
	}
	if !slices.Equal(inline.ShownBy, []string{"risu"}) {
		t.Fatalf("shownBy = %v, want the one app that unpacks it", inline.ShownBy)
	}
	archived := roleVerdictNamed(t, targetLine(t, menu, "charx"), "gallery")
	if len(archived.ShownBy) != 0 {
		t.Fatalf("shownBy = %v, want nothing beside a gallery every app reads", archived.ShownBy)
	}
}

func appTargetsFor(
	t *testing.T, r http.Handler, session *http.Cookie, workID string,
) []apitest.AppTarget {
	t.Helper()
	request := apitest.Authorized(httptest.NewRequest(http.MethodGet, "/v1/assets/"+workID, nil), session)
	response := apitest.Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the work: status = %d: %s", response.Code, response.Body.String())
	}
	var page apitest.StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the work: %v", err)
	}
	return page.AppTargets
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode for comparison: %v", err)
	}
	return encoded
}
