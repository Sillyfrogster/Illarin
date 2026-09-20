package work_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func privateMediaQuery(svc *work.Service, id uuid.UUID, key string) string {
	signed, _ := url.Parse(svc.ImageAddress(id, "grid", false, true))
	return signed.Query().Get(key)
}

func TestMediaURLsUseTheSmoothBlurCacheVersion(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	svc := &work.Service{}
	if got := svc.ImageAddress(id, "grid", false, false); got != "/media/"+id.String()+"/grid/2" {
		t.Fatalf("clear URL = %q", got)
	}
	if got := svc.ImageAddress(id, "grid", true, false); got != "/media/"+id.String()+"/grid_blurred/2" {
		t.Fatalf("blurred URL = %q", got)
	}
	if got := svc.ImageAddress(id, "og", true, false); got != "/media/"+id.String()+"/og_blurred/2" {
		t.Fatalf("blurred embed URL = %q", got)
	}
}

func TestCreatorAddedMediaKeepsNativeDimensionsAndPreGeneratesVariants(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ownerID := uuid.New()
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "theme.bin",
		File: bytes.NewReader([]byte("theme")), Name: "Theme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	source := testPNG(t, 1200, 600, color.RGBA{R: 12, G: 34, B: 56, A: 255})

	added, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaGallery,
		File: bytes.NewReader(source),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("AddMedia: %v", err)
	}
	if added.ID == uuid.Nil {
		t.Fatal("media id is empty")
	}
	if added.WorkID != created.ID {
		t.Fatalf("media work = %v, want %s", added.WorkID, created.ID)
	}
	if added.Width != 1200 || added.Height != 600 {
		t.Fatalf("dimensions = %dx%d, want native 1200x600", added.Width, added.Height)
	}
	if added.DerivativeVersion != mediaproc.DerivativeVersion {
		t.Fatalf("derivative version = %d, want %d", added.DerivativeVersion, mediaproc.DerivativeVersion)
	}

	var blobID uuid.UUID
	var digestBytes []byte
	var storedWork uuid.UUID
	err = pool.QueryRow(context.Background(), `
		select media.blob_id, blob.sha256, media.work_id
		  from work_media media
		  join blobs blob on blob.id = media.blob_id
		 where media.id = $1
	`, added.ID).Scan(&blobID, &digestBytes, &storedWork)
	if err != nil {
		t.Fatalf("read media row: %v", err)
	}
	if storedWork != created.ID {
		t.Fatalf("stored media work = %v, want %s", storedWork, created.ID)
	}
	var digest [sha256.Size]byte
	copy(digest[:], digestBytes)
	for _, variant := range mediaproc.VariantNames() {
		opened, err := svc.Store().OpenDerivative(context.Background(), storage.DerivativeID{
			SourceDigest: digest, Variant: variant, Version: mediaproc.DerivativeVersion,
		})
		if err != nil {
			t.Fatalf("open pre-generated %s derivative: %v", variant, err)
		}
		opened.Close()
	}

	canonical, err := svc.Store().Open(context.Background(), blobID)
	if err != nil {
		t.Fatalf("open canonical media: %v", err)
	}
	var canonicalBytes bytes.Buffer
	if _, err := canonicalBytes.ReadFrom(canonical); err != nil {
		t.Fatalf("read canonical media: %v", err)
	}
	canonical.Close()
	if !bytes.Equal(canonicalBytes.Bytes(), source) {
		t.Fatal("canonical media was changed while making variants")
	}
}

func TestAddingAReplacementMintsANewImmutableMediaRecord(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	ownerID := uuid.New()
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "theme.bin",
		File: bytes.NewReader([]byte("theme")), Name: "Theme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	first, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatar,
		File: bytes.NewReader(testPNG(t, 20, 10, color.Black)),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("Add first media: %v", err)
	}
	second, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatar,
		File: bytes.NewReader(testPNG(t, 30, 15, color.White)),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("Add replacement media: %v", err)
	}
	if first.ID == second.ID {
		t.Fatal("replacement reused the old media id")
	}
	if first.Width != 20 || first.Height != 10 {
		t.Fatalf("first media mutated to %dx%d", first.Width, first.Height)
	}
	var coverID uuid.UUID
	if err := svc.Pool().QueryRow(context.Background(),
		`select cover_media_id from works where id = $1`, created.ID,
	).Scan(&coverID); err != nil {
		t.Fatalf("read cover: %v", err)
	}
	if coverID != second.ID {
		t.Fatalf("cover = %s, want replacement %s", coverID, second.ID)
	}
}

func TestAReplacementDisplayPictureRetiresTheOneBeforeIt(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	ownerID := uuid.New()
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Type: "lorebook", Filename: "book.bin",
		File: bytes.NewReader([]byte("book")), Name: "Book",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	var newest uuid.UUID
	for range 3 {
		added, err := svc.AddMedia(context.Background(), work.AddMediaInput{
			OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatar,
			File: bytes.NewReader(testPNG(t, 20, 10, color.Black)),
		}, apitest.CurrentCandidate(t, svc, created.ID))
		if err != nil {
			t.Fatalf("Add display picture: %v", err)
		}
		newest = added.ID
	}
	rows, err := svc.Pool().Query(context.Background(),
		`select id from work_media where work_id = $1 and is_current`, created.ID)
	if err != nil {
		t.Fatalf("read current media: %v", err)
	}
	defer rows.Close()
	current := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan current media: %v", err)
		}
		current = append(current, id)
	}
	if len(current) != 1 || current[0] != newest {
		t.Fatalf("current pictures = %v, want the newest one alone (%s)", current, newest)
	}
}

func TestAlternateAvatarCoversUntilAPrimaryTakesItsPlace(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	ownerID := uuid.New()
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "theme.bin",
		File: bytes.NewReader([]byte("theme")), Name: "Theme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatarAlt,
		File: bytes.NewReader(testPNG(t, 20, 10, color.Black)),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("Add alternate avatar: %v", err)
	}
	alternate, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatarAlt,
		File: bytes.NewReader(testPNG(t, 30, 15, color.White)),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("Add second alternate avatar: %v", err)
	}
	var coverID uuid.UUID
	if err := svc.Pool().QueryRow(context.Background(),
		`select cover_media_id from works where id = $1`, created.ID,
	).Scan(&coverID); err != nil {
		t.Fatalf("read alternate cover: %v", err)
	}
	if coverID != alternate.ID {
		t.Fatalf("cover = %s, want latest alternate %s", coverID, alternate.ID)
	}
	primary, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatar,
		File: bytes.NewReader(testPNG(t, 40, 20, color.Gray{Y: 128})),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("Add primary avatar: %v", err)
	}
	if _, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaAvatarAlt,
		File: bytes.NewReader(testPNG(t, 50, 25, color.White)),
	}, apitest.CurrentCandidate(t, svc, created.ID)); err != nil {
		t.Fatalf("Add alternate after primary: %v", err)
	}
	if err := svc.Pool().QueryRow(context.Background(),
		`select cover_media_id from works where id = $1`, created.ID,
	).Scan(&coverID); err != nil {
		t.Fatalf("read cover: %v", err)
	}
	if coverID != primary.ID || coverID == alternate.ID {
		t.Fatalf("cover = %s, want primary %s", coverID, primary.ID)
	}
}

func TestMediaVariantRegeneratesABoundedCacheMiss(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	ownerID := uuid.New()
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "theme.bin",
		File: bytes.NewReader([]byte("theme")), Name: "Theme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	added, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaGallery,
		File: bytes.NewReader(testPNG(t, 320, 180, color.White)),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("AddMedia: %v", err)
	}
	if err := svc.Store().ClearDerivatives(context.Background()); err != nil {
		t.Fatalf("clear derivative cache: %v", err)
	}

	download, err := svc.MediaVariant(context.Background(), work.MediaRequest{
		MediaID: added.ID, Variant: "grid", Version: mediaproc.DerivativeVersion,
		ViewerID: &ownerID, Expires: privateMediaQuery(svc, added.ID, "expires"),
		Signature: privateMediaQuery(svc, added.ID, "signature"),
	})
	if err != nil {
		t.Fatalf("MediaVariant cache miss: %v", err)
	}
	if download.InternalRedirect == "" || download.MediaType != mediaproc.DerivativeType {
		t.Fatalf("download = %+v", download)
	}

	for _, request := range []struct {
		variant string
		version uint32
	}{
		{variant: "1200x630", version: mediaproc.DerivativeVersion},
		{variant: "grid", version: mediaproc.DerivativeVersion + 1},
	} {
		_, err := svc.MediaVariant(context.Background(), work.MediaRequest{
			MediaID: added.ID, Variant: request.variant, Version: request.version,
		})
		if !errors.Is(err, work.ErrMediaNotFound) {
			t.Fatalf("MediaVariant(%q, %d) error = %v, want ErrMediaNotFound",
				request.variant, request.version, err)
		}
	}
}

func TestCreatorCannotAddMediaToSomebodyElsesWork(t *testing.T) {
	t.Parallel()
	svc, _ := apitest.Works(t)
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: uuid.New(), Type: "theme", Filename: "theme.bin",
		File: bytes.NewReader([]byte("theme")), Name: "Theme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: uuid.New(), WorkID: created.ID, Role: work.MediaGallery,
		File: bytes.NewReader(testPNG(t, 20, 10, color.White)),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if !errors.Is(err, work.ErrMediaNotFound) {
		t.Fatalf("AddMedia error = %v, want ErrMediaNotFound", err)
	}
}

func TestUploadStoresExtractedMediaOnTheWork(t *testing.T) {
	t.Parallel()
	archive := archiveWithImage(t, testPNG(t, 90, 45, color.White))
	registry := apitest.RegistryWith(t, apitest.RecognizedModule{Parsed: format.Parsed{
		Type: "character", Format: "recognized",
		Media: []format.Media{{Role: "expression", ImageID: 0}},
	}})
	svc, pool := apitest.WorksWithRegistry(t, registry)
	ownerID := uuid.New()
	if _, err := pool.Exec(context.Background(),
		`insert into users (id, username) values ($1, 'media.extractor')`, ownerID); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	operation, err := apitest.Uploads(svc).AcceptUpload(context.Background(), upload.UploadInput{
		OwnerID: ownerID, Filename: "card.charx", File: bytes.NewReader(archive),
	})
	if err != nil {
		t.Fatalf("AcceptUpload: %v", err)
	}
	processed, err := apitest.Uploads(svc).ProcessNextUpload(context.Background())
	if err != nil || !processed {
		t.Fatalf("ProcessNextUpload = %v, %v; want true, nil", processed, err)
	}
	operation, err = apitest.Uploads(svc).GetUpload(context.Background(), ownerID, operation.ID)
	if err != nil {
		t.Fatalf("GetUpload: %v", err)
	}
	if operation.Work == nil {
		t.Fatal("upload did not create a work")
	}

	var workID uuid.UUID
	var role string
	var width, height int
	err = pool.QueryRow(context.Background(), `
		select work_id, role, width, height
		  from work_media
		 where work_id = $1
	`, operation.Work.ID).Scan(&workID, &role, &width, &height)
	if err != nil {
		t.Fatalf("read extracted media: %v", err)
	}
	if workID != operation.Work.ID {
		t.Fatalf("extracted media work = %v, want %v", workID, operation.Work.ID)
	}
	if role != "expression" || width != 90 || height != 45 {
		t.Fatalf("extracted media = %s %dx%d", role, width, height)
	}
}

func TestConcurrentCacheMissesShareOneBoundedRender(t *testing.T) {
	t.Parallel()
	pool := testdb.Connect(t)
	store, err := storage.NewStore(pool, t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	processor := &blockingMediaProcessor{
		renderStarted: make(chan struct{}),
		releaseRender: make(chan struct{}),
	}
	settings := work.DefaultUploadSettings()
	settings.MediaWorkers = 1
	svc := work.NewServiceWithMediaProcessor(
		pool, apitest.RegistryWith(t, apitest.OpaqueModule{}), store, settings, processor,
	)
	ownerID := uuid.New()
	created, err := apitest.Uploads(svc).Create(context.Background(), upload.CreateInput{
		OwnerID: ownerID, Type: "theme", Filename: "theme.bin",
		File: bytes.NewReader([]byte("theme")), Name: "Theme",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	added, err := svc.AddMedia(context.Background(), work.AddMediaInput{
		OwnerID: ownerID, WorkID: created.ID, Role: work.MediaGallery,
		File: bytes.NewReader([]byte("encoded image")),
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil {
		t.Fatalf("AddMedia: %v", err)
	}

	const requests = 8
	start := make(chan struct{})
	errors := make(chan error, requests)
	var ready sync.WaitGroup
	ready.Add(requests)
	for range requests {
		go func() {
			ready.Done()
			<-start
			_, err := svc.MediaVariant(context.Background(), work.MediaRequest{
				MediaID: added.ID, Variant: "grid", Version: mediaproc.DerivativeVersion,
				ViewerID: &ownerID, Expires: privateMediaQuery(svc, added.ID, "expires"),
				Signature: privateMediaQuery(svc, added.ID, "signature"),
			})
			errors <- err
		}()
	}
	ready.Wait()
	close(start)
	select {
	case <-processor.renderStarted:
	case err := <-errors:
		t.Fatalf("MediaVariant returned before any render started: %v", err)
	}
	close(processor.releaseRender)
	for range requests {
		if err := <-errors; err != nil {
			t.Fatalf("MediaVariant: %v", err)
		}
	}
	if calls := processor.renderCalls.Load(); calls != 1 {
		t.Fatalf("render calls = %d, want one shared cache job", calls)
	}
}

type blockingMediaProcessor struct {
	renderCalls   atomic.Int32
	renderStarted chan struct{}
	releaseRender chan struct{}
	startedOnce   sync.Once
}

func (p *blockingMediaProcessor) Prepare(context.Context, io.Reader) (mediaproc.Prepared, error) {
	return mediaproc.Prepared{Width: 20, Height: 10}, nil
}

func (p *blockingMediaProcessor) Render(
	context.Context,
	io.Reader,
	string,
) (mediaproc.Derivative, error) {
	p.renderCalls.Add(1)
	p.startedOnce.Do(func() { close(p.renderStarted) })
	<-p.releaseRender
	return mediaproc.Derivative{Variant: "grid", Bytes: []byte("rendered")}, nil
}

func (p *blockingMediaProcessor) ComposeLinkCard(
	context.Context,
	io.Reader,
	string,
) (mediaproc.Derivative, error) {
	return mediaproc.Derivative{}, errors.New("unexpected social preview")
}

func (p *blockingMediaProcessor) DerivativeType() string { return "image/png" }

func testPNG(t *testing.T, width, height int, fill color.Color) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.Set(x, y, fill)
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return encoded.Bytes()
}

func archiveWithImage(t *testing.T, picture []byte) []byte {
	t.Helper()
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	for _, entry := range []struct{ name, body string }{
		{name: "card.json", body: `{"spec":"chara_card_v3"}`},
		{name: "assets/icon/main.png", body: string(picture)},
	} {
		writer, err := archive.Create(entry.name)
		if err != nil {
			t.Fatalf("create archive entry: %v", err)
		}
		if _, err := io.WriteString(writer, entry.body); err != nil {
			t.Fatalf("write archive entry: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return file.Bytes()
}
