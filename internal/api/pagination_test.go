package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

type item struct {
	ID string `json:"id"`
}

// itemsRange builds n items whose ids continue from start.
func itemsRange(start, n int) []item {
	out := make([]item, 0, n)
	for i := range n {
		out = append(out, item{ID: strconv.Itoa(start + i)})
	}

	return out
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Errorf("encode response: %v", err)
	}
}

func ids(items []item) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.ID)
	}

	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestPaginateAllPagePager(t *testing.T) {
	var queries []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		writeJSON(t, w, map[string]any{
			"items":    itemsRange(page*2, 2),
			"has_next": page < 2,
			"page":     page,
		})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	got, err := PaginateAll[item](t.Context(), c, Request{Method: http.MethodGet, Path: "/payments"}, NewPagePager(2), 0)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	want := []string{"0", "1", "2", "3", "4", "5"}
	if !equalStrings(ids(got), want) {
		t.Errorf("items = %v, want %v", ids(got), want)
	}

	if len(queries) != 3 {
		t.Fatalf("requests = %d, want 3", len(queries))
	}

	for i, q := range queries {
		if q.Get("page") != strconv.Itoa(i) {
			t.Errorf("request %d page = %q, want %d", i, q.Get("page"), i)
		}

		if q.Get("page_size") != "2" {
			t.Errorf("request %d page_size = %q, want 2", i, q.Get("page_size"))
		}
	}
}

func TestPaginateAllOffsetPager(t *testing.T) {
	var queries []url.Values

	// The server reports only a total, so the pager must stop on its own.
	const total = 5

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())

		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		n := min(limit, total-offset)

		writeJSON(t, w, map[string]any{
			"data":  itemsRange(offset, n),
			"total": total,
		})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	got, err := PaginateAll[item](t.Context(), c, Request{Method: http.MethodGet, Path: "/recipients"}, NewOffsetPager(2), 0)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	want := []string{"0", "1", "2", "3", "4"}
	if !equalStrings(ids(got), want) {
		t.Errorf("items = %v, want %v", ids(got), want)
	}

	if len(queries) != 3 {
		t.Fatalf("requests = %d, want 3", len(queries))
	}

	for i, wantOffset := range []string{"0", "2", "4"} {
		if queries[i].Get("offset") != wantOffset {
			t.Errorf("request %d offset = %q, want %s", i, queries[i].Get("offset"), wantOffset)
		}

		if queries[i].Get("limit") != "2" {
			t.Errorf("request %d limit = %q, want 2", i, queries[i].Get("limit"))
		}
	}
}

func TestPaginateAllSizePager(t *testing.T) {
	var queries []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		writeJSON(t, w, map[string]any{
			"results": itemsRange(page*3, 3),
			"last":    page == 1,
		})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	got, err := PaginateAll[item](t.Context(), c, Request{Method: http.MethodGet, Path: "/sellers"}, NewSizePager(3), 0)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	if len(got) != 6 {
		t.Errorf("items = %v, want 6 items", ids(got))
	}

	if len(queries) != 2 {
		t.Fatalf("requests = %d, want 2", len(queries))
	}

	if queries[0].Get("size") != "3" || queries[1].Get("page") != "1" {
		t.Errorf("queries = %v, want size=3 and second page=1", queries)
	}
}

func TestPaginateAllStopsOnEmptyPage(t *testing.T) {
	var requests int

	// The server always claims another page exists but stops returning items.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))

		items := itemsRange(0, 0)
		if page == 0 {
			items = itemsRange(0, 2)
		}

		writeJSON(t, w, map[string]any{"items": items, "has_next": true})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	got, err := PaginateAll[item](t.Context(), c, Request{Method: http.MethodGet, Path: "/payments"}, NewPagePager(2), 0)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("items = %v, want 2", ids(got))
	}

	if requests != 2 {
		t.Errorf("requests = %d, want 2 (the empty page must end the loop)", requests)
	}
}

func TestPaginateAllLimit(t *testing.T) {
	newServer := func(requests *int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*requests++

			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			writeJSON(t, w, map[string]any{
				"items":    itemsRange(page*2, 2),
				"has_next": page < 4,
			})
		}))
	}

	tests := []struct {
		name         string
		limit        int
		wantItems    int
		wantRequests int
	}{
		{name: "everything", limit: 0, wantItems: 10, wantRequests: 5},
		{name: "stops early", limit: 3, wantItems: 3, wantRequests: 2},
		{name: "exact page boundary", limit: 4, wantItems: 4, wantRequests: 2},
		{name: "above total", limit: 50, wantItems: 10, wantRequests: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int

			srv := newServer(&requests)
			defer srv.Close()

			c, _ := newTestClient(t, srv, testProfile())

			got, err := PaginateAll[item](t.Context(), c,
				Request{Method: http.MethodGet, Path: "/payments"}, NewPagePager(2), tt.limit)
			if err != nil {
				t.Fatalf("PaginateAll: %v", err)
			}

			if len(got) != tt.wantItems {
				t.Errorf("items = %d, want %d", len(got), tt.wantItems)
			}

			if requests != tt.wantRequests {
				t.Errorf("requests = %d, want %d", requests, tt.wantRequests)
			}
		})
	}
}

func TestPaginateAllContextCancelledMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var requests int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		// Cancel once the first page is served: the loop must not fetch more.
		cancel()

		writeJSON(t, w, map[string]any{"items": itemsRange(0, 2), "has_next": true})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := PaginateAll[item](ctx, c, Request{Method: http.MethodGet, Path: "/payments"}, NewPagePager(2), 0)
	if err == nil {
		t.Fatal("PaginateAll: want error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}

	if requests > 1 {
		t.Errorf("requests = %d, want at most 1 after cancellation", requests)
	}
}

func TestPaginateAllPropagatesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"NOT_FOUND","message":"no such account"}`))
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	_, err := PaginateAll[item](t.Context(), c, Request{Method: http.MethodGet, Path: "/payments"}, NewOffsetPager(2), 0)
	if err == nil {
		t.Fatal("PaginateAll: want error, got nil")
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("error = %v, want *api.Error with 404", err)
	}
}

func TestPaginateAllPreservesCallerQuery(t *testing.T) {
	var got url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		writeJSON(t, w, map[string]any{"items": itemsRange(0, 1), "has_next": false})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	caller := url.Values{"account_id": []string{"acc-1"}}

	req := Request{Method: http.MethodGet, Path: "/payments", Query: caller}
	if _, err := PaginateAll[item](t.Context(), c, req, NewPagePager(5), 0); err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	if got.Get("account_id") != "acc-1" {
		t.Errorf("account_id = %q, want acc-1", got.Get("account_id"))
	}

	if len(caller) != 1 {
		t.Errorf("caller query was mutated: %v", caller)
	}
}

func TestPageUnmarshal(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantItems []string
		wantNext  *bool
		wantTotal *int
	}{
		{name: "bare array", body: `[{"id":"a"},{"id":"b"}]`, wantItems: []string{"a", "b"}},
		{name: "items key", body: `{"items":[{"id":"a"}],"has_next":true}`, wantItems: []string{"a"}, wantNext: ptr(true)},
		{name: "data key", body: `{"data":[{"id":"a"}],"total":7}`, wantItems: []string{"a"}, wantTotal: ptr(7)},
		{name: "has_more alias", body: `{"results":[{"id":"a"}],"has_more":false}`, wantItems: []string{"a"}, wantNext: ptr(false)},
		{name: "last alias", body: `{"content":[{"id":"a"}],"last":true}`, wantItems: []string{"a"}, wantNext: ptr(false)},
		{name: "nested meta", body: `{"records":[{"id":"a"}],"meta":{"has_next":true,"total_elements":9}}`,
			wantItems: []string{"a"}, wantNext: ptr(true), wantTotal: ptr(9)},
		{name: "empty object", body: `{}`, wantItems: nil},
		{name: "null", body: `null`, wantItems: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var page Page[item]
			if err := json.Unmarshal([]byte(tt.body), &page); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			if !equalStrings(ids(page.Items), tt.wantItems) {
				t.Errorf("items = %v, want %v", ids(page.Items), tt.wantItems)
			}

			if !equalPtr(page.Meta.HasNext, tt.wantNext) {
				t.Errorf("has_next = %v, want %v", show(page.Meta.HasNext), show(tt.wantNext))
			}

			if !equalPtr(page.Meta.Total, tt.wantTotal) {
				t.Errorf("total = %v, want %v", show(page.Meta.Total), show(tt.wantTotal))
			}
		})
	}
}

func TestPageUnmarshalInvalidBody(t *testing.T) {
	var page Page[item]

	if err := json.Unmarshal([]byte(`{"items":{"id":"a"}}`), &page); err == nil {
		t.Fatal("Unmarshal: want error for a non-array items key, got nil")
	}

	if err := json.Unmarshal([]byte(`"a string"`), &page); err == nil {
		t.Fatal("Unmarshal: want error for a scalar body, got nil")
	}
}

func TestPagerDefaultSize(t *testing.T) {
	pagers := map[string]Pager{
		"page":   NewPagePager(0),
		"offset": NewOffsetPager(-1),
		"size":   NewSizePager(0),
	}

	for name, p := range pagers {
		q := p.Query()

		found := false

		for _, key := range []string{"page_size", "limit", "size"} {
			if q.Get(key) == strconv.Itoa(defaultPageSize) {
				found = true
			}
		}

		if !found {
			t.Errorf("%s pager query = %v, want default size %d", name, q, defaultPageSize)
		}
	}
}

func TestPaginateAllNilPagerUsesPageStrategy(t *testing.T) {
	var got url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		writeJSON(t, w, map[string]any{"items": itemsRange(0, 1), "has_next": false})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	if _, err := PaginateAll[item](t.Context(), c,
		Request{Method: http.MethodGet, Path: "/payments"}, nil, 0); err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	if got.Get("page") != "0" || got.Get("page_size") != strconv.Itoa(defaultPageSize) {
		t.Errorf("query = %v, want the default page pager parameters", got)
	}
}

func ptr[T any](v T) *T { return &v }

func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return *a == *b
}

func show[T any](v *T) string {
	if v == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%v", *v)
}

func TestNewNamedPagePager_UsesTheGivenParamAndFirstPage(t *testing.T) {
	pager := NewNamedPagePager("page_number", 1, 2)

	if got := pager.Query().Encode(); got != "page_number=1&page_size=2" {
		t.Errorf("unexpected first query: %s", got)
	}

	if !pager.Advance(PageMeta{}, 2) {
		t.Error("a full page should ask for the next one")
	}

	if got := pager.Query().Encode(); got != "page_number=2&page_size=2" {
		t.Errorf("unexpected second query: %s", got)
	}

	if pager.Advance(PageMeta{}, 1) {
		t.Error("a short page ends the walk")
	}
}

func TestPaginateAllSizeNumberPager(t *testing.T) {
	var queries []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		writeJSON(t, w, map[string]any{
			"data":       itemsRange((page-1)*3, 3),
			"pagination": map[string]any{"page": page, "size": 3, "total": 6},
		})
	}))
	defer srv.Close()

	c, _ := newTestClient(t, srv, testProfile())

	got, err := PaginateAll[item](
		t.Context(), c, Request{Method: http.MethodGet, Path: "/checkouts"}, NewSizeNumberPager(3), 0,
	)
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}

	if len(got) != 6 {
		t.Errorf("items = %v, want 6 items", ids(got))
	}

	if len(queries) != 2 {
		t.Fatalf("requests = %d, want 2", len(queries))
	}

	// The first page is page one, and `total` must stop the walk after the
	// second: a zero-based count would ask for a third page that does not exist.
	if queries[0].Get("page") != "1" || queries[1].Get("page") != "2" {
		t.Errorf("queries = %v, want page=1 then page=2", queries)
	}
}
