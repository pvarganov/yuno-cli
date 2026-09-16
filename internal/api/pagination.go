package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const (
	// defaultPageSize is used when a pager is built without an explicit size.
	defaultPageSize = 100
	// maxPages caps how many requests one PaginateAll call may issue, so a
	// server that never stops advertising a next page cannot hang the CLI.
	maxPages = 1000
)

// itemKeys lists the envelope keys Yuno uses for the items of a list response.
var itemKeys = []string{"items", "data", "results", "content", "records"}

// PageMeta holds the paging hints a list response may carry. Every field is
// optional: Yuno exposes a different subset per resource.
type PageMeta struct {
	HasNext  *bool
	Total    *int
	Page     *int
	PageSize *int
}

// Page is one page of a Yuno list response. The envelope shape differs per
// resource, so the items are looked up under several keys and a bare JSON
// array is accepted as well.
type Page[T any] struct {
	Items []T
	Meta  PageMeta
}

// UnmarshalJSON decodes either a bare array or an envelope object.
func (p *Page[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	if trimmed[0] == '[' {
		var items []T
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return fmt.Errorf("decode page items: %w", err)
		}

		p.Items = items

		return nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &envelope); err != nil {
		return fmt.Errorf("decode page envelope: %w", err)
	}

	items, err := itemsFromEnvelope[T](envelope)
	if err != nil {
		return err
	}

	p.Items = items
	p.Meta = metaFromEnvelope(trimmed, envelope)

	return nil
}

func itemsFromEnvelope[T any](envelope map[string]json.RawMessage) ([]T, error) {
	var firstErr error

	for _, key := range itemKeys {
		raw, ok := envelope[key]
		if !ok {
			continue
		}

		var items []T
		if err := json.Unmarshal(raw, &items); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("decode page items from %q: %w", key, err)
			}

			continue
		}

		return items, nil
	}

	return nil, firstErr
}

// metaFields is the union of every paging hint seen in the Yuno spec.
type metaFields struct {
	HasNext       *bool `json:"has_next"`
	HasMore       *bool `json:"has_more"`
	Last          *bool `json:"last"`
	Total         *int  `json:"total"`
	TotalElements *int  `json:"total_elements"`
	TotalCount    *int  `json:"total_count"`
	TotalItems    *int  `json:"total_items"`
	Page          *int  `json:"page"`
	PageNumber    *int  `json:"page_number"`
	PageSize      *int  `json:"page_size"`
	Size          *int  `json:"size"`
	Limit         *int  `json:"limit"`
}

func metaFromEnvelope(data []byte, envelope map[string]json.RawMessage) PageMeta {
	var fields metaFields
	_ = json.Unmarshal(data, &fields)

	// A nested meta object wins: when both levels carry a hint, the nested one
	// is the one describing the pagination.
	for _, key := range []string{"meta", "pagination"} {
		raw, ok := envelope[key]
		if !ok {
			continue
		}

		var nested metaFields
		if err := json.Unmarshal(raw, &nested); err == nil {
			overlay(&fields, &nested)
		}
	}

	return fields.toPageMeta()
}

func overlay(dst, src *metaFields) {
	dst.HasNext = firstNonNil(src.HasNext, dst.HasNext)
	dst.HasMore = firstNonNil(src.HasMore, dst.HasMore)
	dst.Last = firstNonNil(src.Last, dst.Last)
	dst.Total = firstNonNil(src.Total, dst.Total)
	dst.TotalElements = firstNonNil(src.TotalElements, dst.TotalElements)
	dst.TotalCount = firstNonNil(src.TotalCount, dst.TotalCount)
	dst.TotalItems = firstNonNil(src.TotalItems, dst.TotalItems)
	dst.Page = firstNonNil(src.Page, dst.Page)
	dst.PageNumber = firstNonNil(src.PageNumber, dst.PageNumber)
	dst.PageSize = firstNonNil(src.PageSize, dst.PageSize)
	dst.Size = firstNonNil(src.Size, dst.Size)
	dst.Limit = firstNonNil(src.Limit, dst.Limit)
}

func (f *metaFields) toPageMeta() PageMeta {
	meta := PageMeta{
		HasNext:  firstNonNil(f.HasNext, f.HasMore),
		Total:    firstNonNil(f.Total, f.TotalElements, f.TotalCount, f.TotalItems),
		Page:     firstNonNil(f.Page, f.PageNumber),
		PageSize: firstNonNil(f.PageSize, f.Size, f.Limit),
	}

	if meta.HasNext == nil && f.Last != nil {
		more := !*f.Last
		meta.HasNext = &more
	}

	return meta
}

func firstNonNil[T any](values ...*T) *T {
	for _, v := range values {
		if v != nil {
			return v
		}
	}

	return nil
}

// Pager builds the paging query parameters for successive requests and decides
// whether another request is worth making. Yuno is inconsistent across
// resources, so each list command picks the strategy its endpoint speaks.
type Pager interface {
	// Query returns the paging parameters for the next request.
	Query() url.Values
	// Advance records a fetched page and reports whether more may follow.
	Advance(meta PageMeta, received int) bool
}

// PagePager walks a `page` / `page_size` endpoint. Most Yuno resources count
// pages from zero; the organization ones count from one, which is what first
// records.
type PagePager struct {
	page  int
	size  int
	first int
	// param is the name of the page parameter, `page` unless the endpoint
	// spells it differently.
	param string
}

// pageParam is the name most Yuno endpoints give the page parameter.
const pageParam = "page"

// NewPagePager returns a pager for `page` / `page_size` endpoints whose first
// page is page zero.
func NewPagePager(size int) *PagePager {
	return &PagePager{size: normalizeSize(size), param: pageParam}
}

// NewPageNumberPager returns a pager for `page` / `page_size` endpoints whose
// first page is page one, as the organization endpoints declare.
func NewPageNumberPager(size int) *PagePager {
	return &PagePager{page: 1, size: normalizeSize(size), first: 1, param: pageParam}
}

// NewNamedPagePager returns a pager for an endpoint that spells the page
// parameter differently — `page_number` on the report list — and counts its
// pages from first.
func NewNamedPagePager(param string, first, size int) *PagePager {
	return &PagePager{page: first, size: normalizeSize(size), first: first, param: param}
}

// Query implements Pager.
func (p *PagePager) Query() url.Values {
	name := p.param
	if name == "" {
		name = pageParam
	}

	q := url.Values{}
	q.Set(name, strconv.Itoa(p.page))
	q.Set("page_size", strconv.Itoa(p.size))

	return q
}

// Advance implements Pager.
func (p *PagePager) Advance(meta PageMeta, received int) bool {
	p.page++

	return morePages(meta, received, p.size, (p.page-p.first)*p.size)
}

// OffsetPager walks a `limit` / `offset` endpoint.
type OffsetPager struct {
	offset int
	size   int
}

// NewOffsetPager returns a pager for `limit` / `offset` endpoints.
func NewOffsetPager(size int) *OffsetPager {
	return &OffsetPager{size: normalizeSize(size)}
}

// Query implements Pager.
func (p *OffsetPager) Query() url.Values {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(p.size))
	q.Set("offset", strconv.Itoa(p.offset))

	return q
}

// Advance implements Pager.
func (p *OffsetPager) Advance(meta PageMeta, received int) bool {
	p.offset += received

	return morePages(meta, received, p.size, p.offset)
}

// SizePager walks a `page` / `size` endpoint.
type SizePager struct {
	page int
	size int
}

// NewSizePager returns a pager for `page` / `size` endpoints.
func NewSizePager(size int) *SizePager {
	return &SizePager{size: normalizeSize(size)}
}

// Query implements Pager.
func (p *SizePager) Query() url.Values {
	q := url.Values{}
	q.Set("page", strconv.Itoa(p.page))
	q.Set("size", strconv.Itoa(p.size))

	return q
}

// Advance implements Pager.
func (p *SizePager) Advance(meta PageMeta, received int) bool {
	p.page++

	return morePages(meta, received, p.size, p.page*p.size)
}

func normalizeSize(size int) int {
	if size <= 0 {
		return defaultPageSize
	}

	return size
}

// morePages decides whether another request is worth making, preferring the
// server's own hints and falling back to "the last page was full".
func morePages(meta PageMeta, received, size, fetched int) bool {
	if meta.HasNext != nil {
		return *meta.HasNext
	}

	if meta.Total != nil {
		return fetched < *meta.Total
	}

	return received >= size
}

// PaginateAll fetches every page of a list endpoint and returns the items.
// A limit of zero means "everything"; a positive limit stops as soon as that
// many items are collected and truncates the result to exactly that many.
func PaginateAll[T any](ctx context.Context, c *Client, req Request, pager Pager, limit int) ([]T, error) {
	if pager == nil {
		pager = NewPagePager(0)
	}

	var out []T

	for range maxPages {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("paginate %s %s: %w", req.Method, req.Path, err)
		}

		page, err := Do[Page[T]](ctx, c, pageRequest(req, pager.Query()))
		if err != nil {
			return nil, err
		}

		// Guard against a server that keeps advertising a next page while
		// returning nothing: without this the loop would never end.
		if len(page.Items) == 0 {
			return out, nil
		}

		out = append(out, page.Items...)

		if limit > 0 && len(out) >= limit {
			return out[:limit], nil
		}

		if !pager.Advance(page.Meta, len(page.Items)) {
			return out, nil
		}
	}

	return nil, fmt.Errorf("paginate %s %s: stopped after %d pages", req.Method, req.Path, maxPages)
}

// pageRequest copies req with the pager parameters merged into its query, so
// the caller's own url.Values are never mutated.
func pageRequest(req Request, paging url.Values) Request {
	query := url.Values{}

	for key, values := range req.Query {
		query[key] = append([]string(nil), values...)
	}

	for key, values := range paging {
		query[key] = append([]string(nil), values...)
	}

	req.Query = query

	return req
}
