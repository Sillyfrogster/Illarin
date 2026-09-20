package upload

import (
	"bytes"
	"context"
	"errors"
	"image/color"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func originalFileOwner(t *testing.T, svc *Service, handle string) uuid.UUID {
	t.Helper()
	ownerID := uuid.New()
	if _, err := svc.pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, $2)`, ownerID, handle); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	return ownerID
}

func ingestOne(t *testing.T, svc *Service, ownerID uuid.UUID, filename string, file []byte) work.Work {
	t.Helper()
	operation, err := svc.AcceptIngest(context.Background(), IngestInput{
		OwnerID: ownerID, Filename: filename, File: bytes.NewReader(file),
	})
	if err != nil {
		t.Fatalf("AcceptIngest: %v", err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v; want true, nil", processed, err)
	}
	operation, err = svc.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if operation.Work == nil {
		t.Fatalf("ingest did not create a work: %+v", operation)
	}
	return *operation.Work
}

func publishImported(t *testing.T, svc *Service, ownerID uuid.UUID, created work.Work) {
	t.Helper()
	name := created.Name
	if name == "" {
		name = "Test asset"
	}
	nsfw := false
	if err := works(svc).SetDetails(context.Background(), page.Details{
		OwnerID: ownerID, WorkID: created.ID, Name: name, Blurb: created.Blurb, IsNSFW: &nsfw,
	}, currentCandidate(t, svc, created.ID)); err != nil {
		t.Fatalf("SetDetails imported work: %v", err)
	}
	if _, err := works(svc).Publish(context.Background(), ownerID, created.ID, currentCandidate(t, svc, created.ID)); err != nil {
		t.Fatalf("Publish imported work: %v", err)
	}
}

func addRevision(
	t *testing.T,
	svc *Service,
	ownerID, workID uuid.UUID,
	filename string,
	file []byte,
) Operation {
	t.Helper()
	operation, err := svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
		OwnerID: ownerID, WorkID: workID, Filename: filename, File: bytes.NewReader(file),
	}, currentCandidate(t, svc, workID))
	if err != nil {
		t.Fatalf("AcceptOriginalFile: %v", err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v; want true, nil", processed, err)
	}
	got, err := svc.GetIngest(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetIngest: %v", err)
	}
	if got.Status == IngestPreview {
		got, err = svc.AcceptReplacement(context.Background(), ownerID, workID, operation.ID, currentCandidate(t, svc, workID), nil, false)
		if err != nil {
			t.Fatalf("AcceptReplacement: %v", err)
		}
	}
	return got
}

func TestANewOriginalFileChangesTheDraftedChangesAndKeepsThePublishedOne(t *testing.T) {
	t.Parallel()
	registry := registryWithModule(t, recognizedModule{parsed: format.Parsed{
		Type: "character", Format: "recognized", Header: format.Header{Name: "Seeded", Blurb: "Seeded blurb"},
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Description"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}}},
		},
	}})
	svc, pool := newTestServiceWithRegistry(t, registry)
	ownerID := originalFileOwner(t, svc, "revision.owner")
	created := ingestOne(t, svc, ownerID, "card.json", []byte(`{"spec":"x","take":1}`))
	publishImported(t, svc, ownerID, created)

	operation := addRevision(t, svc, ownerID, created.ID, "card.json", []byte(`{"spec":"x","take":2}`))
	if operation.Status != IngestSuccess || operation.Work == nil {
		t.Fatalf("revision operation = %+v, want success", operation)
	}
	if operation.Work.ID != created.ID {
		t.Fatalf("revision made work %s, want %s", operation.Work.ID, created.ID)
	}
	if operation.Work.OriginalFileID == created.OriginalFileID {
		t.Fatal("the work still points at its first revision")
	}
	if operation.Work.Name != created.Name || operation.Work.Blurb != created.Blurb {
		t.Fatalf("the details were re-seeded: %+v", operation.Work)
	}

	var revisions int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from work_original_files where work_id = $1`, created.ID).Scan(&revisions); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 2 {
		t.Fatalf("revision count = %d, want 2", revisions)
	}

	source, err := svc.works.OpenSource(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	defer source.Close()
	var served bytes.Buffer
	if _, err := served.ReadFrom(source); err != nil {
		t.Fatalf("read source: %v", err)
	}
	if served.String() != `{"spec":"x","take":1}` {
		t.Fatalf("source = %s, want the published revision's bytes", served.String())
	}
}

func TestAnOriginalFileResolvingToADifferentTypeIsRejected(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range []format.Module{
		typeModule{id: "as_character", workType: "character"},
		typeModule{id: "as_lorebook", workType: "lorebook"},
	} {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register: %v", err)
		}
	}
	svc, pool := newTestServiceWithRegistry(t, registry)
	ownerID := originalFileOwner(t, svc, "kind.owner")
	created := ingestOne(t, svc, ownerID, "card.json", []byte(`{"spec":"as_character"}`))

	operation := addRevision(t, svc, ownerID, created.ID, "book.json", []byte(`{"spec":"as_lorebook"}`))
	if operation.Status != IngestFailed {
		t.Fatalf("revision status = %s, want failed", operation.Status)
	}
	if operation.Failure == nil || operation.Failure.Reason != "wrong_type" {
		t.Fatalf("revision failure = %+v, want wrong_type", operation.Failure)
	}

	var workType string
	var currentOriginalFileID uuid.UUID
	var revisions int
	err := pool.QueryRow(context.Background(), `
		select type, original_file_id,
		       (select count(*) from work_original_files where work_id = works.id)
		  from works where id = $1
	`, created.ID).Scan(&workType, &currentOriginalFileID, &revisions)
	if err != nil {
		t.Fatalf("read work: %v", err)
	}
	if workType != "character" || currentOriginalFileID != created.OriginalFileID || revisions != 1 {
		t.Fatalf("work changed: type %s, current %s, revisions %d", workType, currentOriginalFileID, revisions)
	}
}

func TestAReplacementFileBecomesTheWorksOrigin(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	svc, pool := newTestServiceWithRegistry(t, registry)
	ownerID := originalFileOwner(t, svc, "replacement.origin.owner")
	created := ingestOne(t, svc, ownerID, "card-v2.json", []byte(`{
		"spec":"chara_card_v2","spec_version":"2.0",
		"data":{"name":"Ana","description":"Before","first_mes":"Hello","character_version":"v2"}
	}`))
	operation := addRevision(t, svc, ownerID, created.ID, "card-v3.json", []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"After","first_mes":"Hello","character_version":"v3"}
	}`))
	if operation.Status != IngestSuccess {
		t.Fatalf("replacement = %+v, want success", operation)
	}
	var origin, version string
	if err := pool.QueryRow(context.Background(), `
		select origin_format, work_version from works where id = $1
	`, created.ID).Scan(&origin, &version); err != nil {
		t.Fatalf("read origin: %v", err)
	}
	if origin != character.V3 || version != "v3" {
		t.Fatalf("origin and header = %q, %q; want %q, v3", origin, version, character.V3)
	}
}

func TestAnUnrecognisedOriginalFileIsRefusedWithoutChangingTheWork(t *testing.T) {
	t.Parallel()
	registry := registryWithModule(t, typeModule{id: "as_character", workType: "character"})
	svc, pool := newTestServiceWithRegistry(t, registry)
	ownerID := originalFileOwner(t, svc, "unsupported.revision.owner")
	created := ingestOne(t, svc, ownerID, "card.json", []byte(`{"spec":"as_character"}`))

	operation := addRevision(t, svc, ownerID, created.ID, "mystery.bin", []byte("nothing matches this"))
	if operation.Status != IngestFailed || operation.Work != nil {
		t.Fatalf("revision operation = %+v, want failed without a work", operation)
	}
	if operation.Failure == nil || operation.Failure.Reason != string(format.FailureUnsupportedFormat) {
		t.Fatalf("revision failure = %+v, want unsupported_format", operation.Failure)
	}

	var currentOriginalFileID uuid.UUID
	var revisions int
	err := pool.QueryRow(context.Background(), `
		select original_file_id,
		       (select count(*) from work_original_files where work_id = works.id)
		  from works where id = $1
	`, created.ID).Scan(&currentOriginalFileID, &revisions)
	if err != nil {
		t.Fatalf("read work: %v", err)
	}
	if currentOriginalFileID != created.OriginalFileID || revisions != 1 {
		t.Fatalf("work changed: current %s, revisions %d", currentOriginalFileID, revisions)
	}
}

func TestOnlyTheOwnerOfALiveWorkCanAddAnOriginalFile(t *testing.T) {
	t.Parallel()
	registry := registryWithModule(t, typeModule{id: "as_character", workType: "character"})
	svc, pool := newTestServiceWithRegistry(t, registry)
	ownerID := originalFileOwner(t, svc, "guard.owner")
	created := ingestOne(t, svc, ownerID, "card.json", []byte(`{"spec":"as_character"}`))

	_, err := svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
		OwnerID: uuid.New(), WorkID: created.ID, Filename: "card.json",
		File: bytes.NewReader([]byte(`{"spec":"as_character"}`)),
	}, currentCandidate(t, svc, created.ID))
	if !errors.Is(err, work.ErrNotFound) {
		t.Fatalf("stranger revision error = %v, want work.ErrNotFound", err)
	}

	if _, err := pool.Exec(context.Background(), `
		update works set withheld_at = now(), withheld_by = $2, withheld_reason = 'held'
		 where id = $1
	`, created.ID, ownerID); err != nil {
		t.Fatalf("withhold work: %v", err)
	}
	_, err = svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
		OwnerID: ownerID, WorkID: created.ID, Filename: "card.json",
		File: bytes.NewReader([]byte(`{"spec":"as_character"}`)),
	}, currentCandidate(t, svc, created.ID))
	if !errors.Is(err, work.ErrWorkFrozen) {
		t.Fatalf("withheld revision error = %v, want work.ErrAssetFrozen", err)
	}
}

func TestReimportedMediaFillsTheWork(t *testing.T) {
	t.Parallel()
	registry := registryWithModule(t, recognizedModule{parsed: format.Parsed{
		Type: "character", Format: "recognized",
		Media: []format.Media{{Role: work.MediaAvatar, ImageID: 0}},
	}})
	svc, pool := newTestServiceWithRegistry(t, registry)
	ownerID := originalFileOwner(t, svc, "scoped.owner")
	first := archiveWithImage(t, testPNG(t, 40, 20, color.White))
	created := ingestOne(t, svc, ownerID, "card.charx", first)
	added, err := svc.works.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaGallery,
		File: bytes.NewReader(testPNG(t, 50, 25, color.Gray{Y: 128})),
	}, currentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("add creator media: %v", err)
	}

	second := archiveWithImage(t, testPNG(t, 60, 30, color.Black))
	operation := addRevision(t, svc, ownerID, created.ID, "card.charx", second)
	if operation.Status != IngestSuccess || operation.Work == nil {
		t.Fatalf("revision operation = %+v, want success", operation)
	}
	media, err := svc.works.ListMedia(context.Background(), created.ID, &ownerID)
	if err != nil {
		t.Fatalf("list reimported media: %v", err)
	}
	if len(media) != 2 {
		t.Fatalf("media rows = %d, want creator media and the reimported image", len(media))
	}
	foundCreatorMedia := false
	for _, image := range media {
		if image.WorkID != created.ID {
			t.Fatalf("reimported media belongs to %s, want %s", image.WorkID, created.ID)
		}
		foundCreatorMedia = foundCreatorMedia || image.ID == added.ID
	}
	if !foundCreatorMedia {
		t.Fatalf("creator media %s disappeared on reimport", added.ID)
	}

	var previewWork uuid.UUID
	var width int
	err = pool.QueryRow(context.Background(), `
		select media.work_id, media.width
		  from works work
		  join work_media media on media.id = work.cover_media_id
		 where work.id = $1
	`, created.ID).Scan(&previewWork, &width)
	if err != nil {
		t.Fatalf("read cover media: %v", err)
	}
	if previewWork != created.ID {
		t.Fatalf("cover belongs to work %s, want %s", previewWork, created.ID)
	}
	if width != 60 {
		t.Fatalf("cover media is %d wide, want the reimported picture", width)
	}
}

type typeModule struct {
	id       string
	workType string
}

func (m typeModule) ID() string { return m.id }
func (m typeModule) Declaration() format.Declaration {
	return testReaderDeclaration(m.id, m.workType)
}

func (m typeModule) Match(file format.Inspection) (format.Match, bool) {
	for _, payload := range file.Payloads {
		if spec, _ := payload.String("spec"); spec == m.id {
			return format.AuthoritativeMatch(payload, "spec")
		}
	}
	return format.Match{}, false
}

func (m typeModule) Parse(context.Context, format.Inspection, format.Match) (format.Parsed, error) {
	return format.Parsed{Type: m.workType, Format: m.id}, nil
}
