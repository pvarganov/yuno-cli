package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// reportPath is the base path of the report resource.
const reportPath = "/reports"

// reportPageParam is how the report list spells its page parameter; it counts
// pages from one, unlike the `page` endpoints.
const reportPageParam = "page_number"

// maxDownloadSize caps a streamed report at 2 GiB, so a broken link that never
// ends cannot fill the disk.
const maxDownloadSize = 2 << 30

// CreateReport schedules a report run from an already built JSON body.
func (c *Client) CreateReport(ctx context.Context, body any) (*model.Report, error) {
	report, err := Do[model.Report](ctx, c, Request{
		Method: http.MethodPost,
		Path:   reportPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}

	return &report, nil
}

// ListReports walks `GET /reports/list`, which pages with `page_number` and
// `page_size` starting at page one.
func (c *Client) ListReports(
	ctx context.Context, query url.Values, limit, pageSize int,
) ([]model.Report, error) {
	reports, err := PaginateAll[model.Report](ctx, c, Request{
		Method: http.MethodGet,
		Path:   reportPath + "/list",
		Query:  query,
	}, NewNamedPagePager(reportPageParam, 1, pageSize), limit)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}

	return reports, nil
}

// GetReport retrieves one report run by id.
func (c *Client) GetReport(ctx context.Context, reportID string) (*model.Report, error) {
	report, err := Do[model.Report](ctx, c, Request{
		Method: http.MethodGet,
		Path:   reportPath + "/" + url.PathEscape(reportID),
	})
	if err != nil {
		return nil, fmt.Errorf("get report %s: %w", reportID, err)
	}

	return &report, nil
}

// DownloadReport asks for the pre-signed link of a generated report.
func (c *Client) DownloadReport(ctx context.Context, reportID string) (*model.ReportDownload, error) {
	download, err := Do[model.ReportDownload](ctx, c, Request{
		Method: http.MethodGet,
		Path:   reportPath + "/" + url.PathEscape(reportID) + "/download",
	})
	if err != nil {
		return nil, fmt.Errorf("download report %s: %w", reportID, err)
	}

	return &download, nil
}

// FetchFile streams the body of a pre-signed download link into w. The link
// carries its own credentials, so the Yuno auth headers are deliberately not
// sent: forwarding them to a third-party host would leak the API key.
func (c *Client) FetchFile(ctx context.Context, rawURL string, w io.Writer) (int64, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return 0, fmt.Errorf("parse download link: %w", err)
	}

	if target.Scheme != "http" && target.Scheme != "https" {
		return 0, fmt.Errorf("download link %s: want an http or https url", rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("build download request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch download link: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		return 0, fmt.Errorf("fetch download link: unexpected status %s", resp.Status)
	}

	written, err := io.Copy(w, io.LimitReader(resp.Body, maxDownloadSize))
	if err != nil {
		return written, fmt.Errorf("write downloaded report: %w", err)
	}

	return written, nil
}
