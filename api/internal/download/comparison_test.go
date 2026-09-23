package download_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
)

type formatTable struct {
	Apps    []apitest.AppName `json:"apps"`
	Formats []struct {
		ID          string   `json:"id"`
		Label       string   `json:"label"`
		Type        string   `json:"type"`
		ReadBy      []string `json:"readBy"`
		KeepsUpload bool     `json:"keepsUpload"`
		Fields      []struct {
			Field string `json:"field"`
			Label string `json:"label"`
			Grade string `json:"grade"`
			Note  string `json:"note"`
		} `json:"fields"`
	} `json:"formats"`
}

func TestTheFormatTableComparesEveryWrittenFormatFieldByField(t *testing.T) {
	t.Parallel()
	r, _, _ := harness.NewCharacterUploadRouter(t)

	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/formats", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var table formatTable
	if err := json.Unmarshal(response.Body.Bytes(), &table); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(table.Apps) == 0 {
		t.Fatal("the table names no apps")
	}
	cell := func(formatID, field string) (string, string) {
		for _, column := range table.Formats {
			if column.ID != formatID {
				continue
			}
			for _, one := range column.Fields {
				if one.Field == field {
					return one.Grade, one.Note
				}
			}
			t.Fatalf("%s has no %s row", formatID, field)
		}
		t.Fatalf("%s is not in the table", formatID)
		return "", ""
	}
	if grade, _ := cell("chara_card_v2", "gallery"); grade != "none" {
		t.Errorf("a V2 card's gallery = %s, want none", grade)
	}
	if grade, note := cell("chara_card_v3", "gallery"); grade != "full" || note == "" {
		t.Errorf("a V3 card's gallery = %s %q, want full with a note on where it lands", grade, note)
	}
	if grade, note := cell("charx", "greetings"); grade != "partial" || note == "" {
		t.Errorf("CharX greetings = %s %q, want partial with the reason", grade, note)
	}
	for _, column := range table.Formats {
		if column.ID == "chara_card_v3" && !slices.Equal(column.ReadBy, []string{"sillytavern", "risu", "lumiverse"}) {
			t.Errorf("V3 is read by %v, want every app", column.ReadBy)
		}
		if column.ID == "extension_spindle" && (!column.KeepsUpload || len(column.Fields) != 0) {
			t.Errorf("a Spindle extension should hand the upload back untouched, got %+v", column)
		}
		if column.Type == "" || column.Label == "" {
			t.Errorf("%s lacks a type or label", column.ID)
		}
	}
}

func TestAReaderWithAnAppSetIsToldWhichFormatToDownload(t *testing.T) {
	t.Parallel()
	r, session, works := harness.NewCharacterUploadRouter(t)
	workID := apitest.UploadedCharacterID(t, r, session, works, apitest.PlainCard)
	apitest.PublishCharacter(t, r, session, workID)

	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil)
	request.AddCookie(&http.Cookie{Name: page.AppCookie, Value: "risu"})
	response := apitest.Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var detail struct {
		ReaderApp  *string             `json:"readerApp"`
		AppFormats []apitest.AppFormat `json:"appFormats"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.ReaderApp == nil || *detail.ReaderApp != "risu" {
		t.Fatalf("readerApp = %v, want risu from the browser", detail.ReaderApp)
	}
	chosen := ""
	for _, app := range detail.AppFormats {
		if app.ID == "risu" {
			chosen = app.Format
		}
	}
	if chosen == "" {
		t.Fatalf("no format offered for RisuAI: %+v", detail.AppFormats)
	}
	download := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/download/"+workID+"/"+chosen, nil))
	if download.Code != http.StatusOK || download.Header().Get("X-Illarin-Format") != chosen {
		t.Fatalf("download %s status = %d format %q", chosen, download.Code, download.Header().Get("X-Illarin-Format"))
	}

	anyApp := httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil)
	anyApp.AddCookie(&http.Cookie{Name: page.AppCookie, Value: "any"})
	if err := json.Unmarshal(apitest.Send(t, r, anyApp).Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if detail.ReaderApp != nil {
		t.Errorf("readerApp = %q under any, want none", *detail.ReaderApp)
	}
}
