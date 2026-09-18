package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type WorkPageResponse struct {
	ID         string `json:"id"`
	Type       string `json:"kind"`
	Name       string `json:"name"`
	Blurb      string `json:"blurb"`
	Creator    string `json:"creator"`
	IsNSFW     bool   `json:"isNsfw"`
	Visibility string `json:"discovery"`
	CreatedAt  string `json:"createdAt"`
	Tags       []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	} `json:"tags"`
	Media []struct {
		ID        string `json:"id"`
		Role      string `json:"role"`
		IsCover   bool   `json:"isCover"`
		DetailURL string `json:"detailUrl"`
		ThumbURL  string `json:"thumbUrl"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	} `json:"media"`
	Preview        *string `json:"preview"`
	NSFWPreference string  `json:"visibility"`
	Withhold       *struct {
		Reason string    `json:"reason"`
		At     time.Time `json:"at"`
	} `json:"withhold"`
}

func FetchWorkPage(t *testing.T, r http.Handler, path string) WorkPageResponse {
	t.Helper()
	response := Send(t, r, httptest.NewRequest(http.MethodGet, path, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200: %s", path, response.Code, response.Body.String())
	}
	var page WorkPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode asset page: %v", err)
	}
	return page
}

type ListedWork struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListedPage struct {
	Items      []ListedWork `json:"items"`
	NextCursor *struct {
		Before   time.Time `json:"before"`
		BeforeID string    `json:"beforeId"`
	} `json:"nextCursor"`
}

func ListPage(t *testing.T, r *gin.Engine, url string) ListedPage {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200. body: %s", url, rec.Code, rec.Body.String())
	}

	var list ListedPage
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return list
}

func ListItems(t *testing.T, r *gin.Engine, url string) []ListedWork {
	t.Helper()
	return ListPage(t, r, url).Items
}
