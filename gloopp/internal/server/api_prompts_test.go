package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/auth"
	"code.byted.org/lihuanyu.0w0/gloop/internal/events"
	"code.byted.org/lihuanyu.0w0/gloop/internal/fsstore"
)

func TestPromptsDoesNotRequireEngine(t *testing.T) {
	root, err := fsstore.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open root: %v", err)
	}
	s := &Server{
		root:  root,
		bus:   events.NewBus(),
		Token: &auth.TokenStore{Token: testToken},
	}

	rr := httptest.NewRecorder()
	s.buildRouter().ServeHTTP(rr, authedReq(http.MethodGet, "/api/prompts", ""))

	if rr.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	body := decodeBody(t, rr)
	items, ok := body["items"].([]any)
	if body["ok"] != true || !ok || len(items) == 0 {
		t.Fatalf("unexpected prompts body: %#v", body)
	}

	first, ok := items[0].(map[string]any)
	if !ok || first["name"] == "" {
		t.Fatalf("unexpected first prompt: %#v", items[0])
	}
	if first["source"] == "" {
		t.Fatalf("prompt should have source field: %#v", first)
	}

	// show 接口
	rr = httptest.NewRecorder()
	s.buildRouter().ServeHTTP(rr, authedReq(http.MethodGet, "/api/prompts/"+first["name"].(string), ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	detail := decodeBody(t, rr)
	item, ok := detail["item"].(map[string]any)
	if detail["ok"] != true || !ok || item["content"] == "" {
		t.Fatalf("unexpected prompt detail: %#v", detail)
	}
}
