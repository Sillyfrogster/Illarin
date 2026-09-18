package work_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

func TestACardKeepsItsExactBytesWhileItsPictureIsExtracted(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %q: %v", module.ID(), err)
		}
	}
	svc, pool := apitest.WorksWithRegistry(t, registry)
	ownerID := apitest.Owner(t, svc, "card.owner")

	card := pngCardFile(t, `{
		"spec":"chara_card_v2","spec_version":"2.0",
		"data":{
			"name":"Ana",
			"description":"A quiet archivist.",
			"first_mes":"Hello.",
			"creator_notes":"A quiet archivist.",
			"extensions":{"depth_prompt":{"depth":4},"third_party":{"kept":true}}
		}
	}`)
	created := apitest.IngestOne(t, svc, ownerID, "ana.png", card)
	apitest.PublishImported(t, svc, ownerID, created)
	if created.Type != "character" || created.Format != character.V2 {
		t.Fatalf("work = type %q format %q", created.Type, created.Format)
	}
	if created.Name != "Ana" || created.Blurb != "A quiet archivist." {
		t.Fatalf("catalog seed = %q, %q", created.Name, created.Blurb)
	}

	source, err := svc.OpenSource(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	defer source.Close()
	var served bytes.Buffer
	if _, err := served.ReadFrom(source); err != nil {
		t.Fatalf("read source: %v", err)
	}
	if !bytes.Equal(served.Bytes(), card) {
		t.Fatal("the stored card is not the file that was uploaded")
	}

	var coverRole string
	var coverWork uuid.UUID
	err = pool.QueryRow(context.Background(), `
		select media.role, media.work_id
		  from works work
		  join work_media media on media.id = work.cover_media_id
		 where work.id = $1
	`, created.ID).Scan(&coverRole, &coverWork)
	if err != nil {
		t.Fatalf("read cover media: %v", err)
	}
	if coverRole != string(work.MediaAvatar) || coverWork != created.ID {
		t.Fatalf("cover = %s on work %s", coverRole, coverWork)
	}

}

func pngCardFile(t *testing.T, body string) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for x := range 8 {
		for y := range 4 {
			picture.Set(x, y, color.RGBA{R: 90, G: 60, B: 120, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatalf("encode card picture: %v", err)
	}
	file := encoded.Bytes()
	end := len(file) - 12
	return slices.Concat(file[:end], textChunk("chara", body), file[end:])
}

func textChunk(keyword, body string) []byte {
	data := slices.Concat([]byte(keyword), []byte{0}, []byte(body))
	var chunk bytes.Buffer
	_ = binary.Write(&chunk, binary.BigEndian, uint32(len(data)))
	chunk.WriteString("tEXt")
	chunk.Write(data)
	_ = binary.Write(&chunk, binary.BigEndian, crc32.ChecksumIEEE(append([]byte("tEXt"), data...)))
	return chunk.Bytes()
}
