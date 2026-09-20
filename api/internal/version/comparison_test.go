package version_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func changesUnder(t *testing.T, groups []version.ChangeGroup, subject string) []version.Change {
	t.Helper()
	for _, group := range groups {
		if group.Subject == subject {
			return group.Changes
		}
	}
	t.Fatalf("no %s changes in %+v", subject, groups)
	return nil
}

func changeTypes(changes []version.Change) []version.ChangeType {
	types := make([]version.ChangeType, 0, len(changes))
	for _, change := range changes {
		types = append(types, change.Type)
	}
	return types
}

func subjectsOf(groups []version.ChangeGroup) []string {
	subjects := make([]string, 0, len(groups))
	for _, group := range groups {
		subjects = append(subjects, group.Subject)
	}
	return subjects
}

func number(value float64) *block.Value { return &block.Value{Number: &value} }

func TestComparisonDefaultsToTheVersionBeforeThePublishedOne(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "compare.owner")
	open := func(work.Version) string { return "" }

	if _, err := version.NewService(svc.Pool(), svc).Compare(ctx, version.ComparisonRequest{WorkID: id, Access: open}); !errors.Is(err, work.ErrNoEarlierVersion) {
		t.Fatalf("first version compared against nothing: %v", err)
	}
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	publishUpdate(t, svc, owner, id, "Rewrote the description")
	apitest.SaveDescription(t, svc, owner, id, "Third description")
	nsfw := false
	if err := apitest.Pages(svc).SetDetails(ctx, page.Details{
		OwnerID: owner, WorkID: id, Name: "Renamed", Blurb: "A changed pitch", IsNSFW: &nsfw,
	}, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	publishUpdate(t, svc, owner, id, "Rewrote it again")

	latest, err := version.NewService(svc.Pool(), svc).Compare(ctx, version.ComparisonRequest{WorkID: id, Access: open})
	if err != nil {
		t.Fatal(err)
	}
	if latest.From.Number != 2 || latest.To.Number != 3 {
		t.Fatalf("default comparison ran %d against %d", latest.From.Number, latest.To.Number)
	}
	changes := changesUnder(t, latest.Groups, string(block.RoleDescription))
	if len(changes) != 1 || changes[0].Before != "Second description" ||
		changes[0].After != "Third description" {
		t.Fatalf("changes = %+v", changes)
	}
	renamed := changesUnder(t, latest.Groups, version.MetadataSubject)
	if len(renamed) != 2 || renamed[0].Name != "Name" ||
		renamed[0].Before != "Published name" || renamed[0].After != "Renamed" ||
		renamed[1].Name != "Blurb" || renamed[1].Before != "" || renamed[1].After != "A changed pitch" {
		t.Fatalf("metadata changes = %+v", renamed)
	}
	chosen, err := version.NewService(svc.Pool(), svc).Compare(ctx, version.ComparisonRequest{WorkID: id, From: 1, To: 3, Access: open})
	if err != nil {
		t.Fatal(err)
	}
	changes = changesUnder(t, chosen.Groups, string(block.RoleDescription))
	if len(changes) != 1 || changes[0].Before != "Published description" {
		t.Fatalf("chosen comparison = %+v", changes)
	}
}

func TestComparisonNeedsAccessRulesAndExplainsAVersionItCannotOpen(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "gated.owner")
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	publishUpdate(t, svc, owner, id, "Rewrote the description")

	if _, err := version.NewService(svc.Pool(), svc).Compare(ctx, version.ComparisonRequest{WorkID: id}); !errors.Is(err, version.ErrAccessRequired) {
		t.Fatalf("comparison ran without access rules: %v", err)
	}
	withheld, err := version.NewService(svc.Pool(), svc).Compare(ctx, version.ComparisonRequest{WorkID: id, Access: func(version work.Version) string {
		if version.Number == 1 {
			return "That version was unpublished."
		}
		return ""
	}})
	if err != nil {
		t.Fatal(err)
	}
	if withheld.Unavailable != "That version was unpublished." || len(withheld.Groups) != 0 {
		t.Fatalf("withheld comparison = %+v", withheld)
	}
}

func publishUpdate(t *testing.T, svc *work.Service, owner, id uuid.UUID, summary string) {
	t.Helper()
	if _, _, err := version.NewService(svc.Pool(), svc).PublishVersion(context.Background(), version.PublishRequest{
		OwnerID: owner, WorkID: id, Summary: summary,
	}, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the update: %v", err)
	}
}

func TestReplacementPreviewUsesStableItemIDs(t *testing.T) {
	t.Parallel()
	shared, removed, added := block.NewItemID(), block.NewItemID(), block.NewItemID()
	working := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: shared, Text: "Same"}, {ID: removed, Text: "Old"}}},
	}}}}
	incoming := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: shared, Text: "Updated"}, {ID: added, Text: "New"}}},
	}}}}
	groups := version.CompareContentKeyed(working, incoming, nil)
	var additions, removals, updates int
	for _, group := range groups {
		if group.Subject != string(block.RoleGreetings) {
			continue
		}
		for _, change := range group.Changes {
			switch change.Type {
			case version.ChangeAdded:
				additions++
			case version.ChangeRemoved:
				removals++
			case version.ChangeEdited:
				updates++
			}
		}
	}
	if additions != 1 || removals != 1 || updates != 1 {
		t.Fatalf("stable-item changes = %+v", groups)
	}
}

func TestReplacementPreviewShowsTheWordingOnBothSides(t *testing.T) {
	t.Parallel()
	item := block.NewItemID()
	working := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: item, Name: "Opening", Text: "Old wording"}}},
	}}}}
	incoming := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: item, Name: "Opening", Text: "New wording"}}},
	}}}}
	groups := version.CompareContentKeyed(working, incoming, nil)
	if len(groups) != 1 || len(groups[0].Changes) != 1 {
		t.Fatalf("groups = %+v", groups)
	}
	change := groups[0].Changes[0]
	if change.Type != version.ChangeEdited || change.Before != "Old wording" || change.After != "New wording" {
		t.Fatalf("change = %+v, want the wording on both sides", change)
	}
}
