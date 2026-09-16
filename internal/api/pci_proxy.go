package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pvarganov/yuno-cli/internal/model"
)

// PCI proxy paths and the headers only the forward endpoint understands.
const (
	pciProxyDestinationPath = "/pci-proxy/destinations"
	pciProxyForwardPath     = "/pci-proxy/forward"

	// HeaderProxyDestinationURL carries the full destination URL of a forwarded
	// request: the proxy has no path of its own.
	HeaderProxyDestinationURL = "yuno-proxy-destination-url"
	// HeaderProxyTimeout sets the destination timeout in seconds.
	HeaderProxyTimeout = "yuno-proxy-timeout"
	// HeaderProxyAuth selects a request signing scheme, e.g. DLOCAL_HMAC.
	HeaderProxyAuth = "yuno-proxy-auth"
	// HeaderProxyAuthSecretKey carries the signing secret of that scheme.
	HeaderProxyAuthSecretKey = "yuno-proxy-auth-secret-key" //nolint:gosec // header name, not a credential
	// HeaderProxyAccountID narrows which allowlisted destinations are permitted.
	HeaderProxyAccountID = "yuno-account-id"
)

// ListPCIProxyDestinations lists the forward proxy allowlist. The endpoint is
// not paginated: it answers with the whole list under `destinations`.
func (c *Client) ListPCIProxyDestinations(ctx context.Context) ([]model.PCIProxyDestination, error) {
	list, err := Do[model.PCIProxyDestinationList](ctx, c, Request{
		Method: http.MethodGet,
		Path:   pciProxyDestinationPath,
	})
	if err != nil {
		return nil, fmt.Errorf("list pci proxy destinations: %w", err)
	}

	return list.Destinations, nil
}

// CreatePCIProxyDestination registers a hostname on the allowlist.
func (c *Client) CreatePCIProxyDestination(ctx context.Context, body any) (*model.PCIProxyDestination, error) {
	destination, err := Do[model.PCIProxyDestination](ctx, c, Request{
		Method: http.MethodPost,
		Path:   pciProxyDestinationPath,
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("create pci proxy destination: %w", err)
	}

	return &destination, nil
}

// EnablePCIProxyDestination puts a destination back in the forward path.
func (c *Client) EnablePCIProxyDestination(ctx context.Context, id string) (*model.PCIProxyDestination, error) {
	return c.switchPCIProxyDestination(ctx, id, "enable")
}

// DisablePCIProxyDestination takes a destination out of the forward path
// without removing it from the allowlist.
func (c *Client) DisablePCIProxyDestination(ctx context.Context, id string) (*model.PCIProxyDestination, error) {
	return c.switchPCIProxyDestination(ctx, id, "disable")
}

// switchPCIProxyDestination flips a destination between ENABLED and DISABLED.
func (c *Client) switchPCIProxyDestination(
	ctx context.Context, id, action string,
) (*model.PCIProxyDestination, error) {
	destination, err := Do[model.PCIProxyDestination](ctx, c, Request{
		Method: http.MethodPost,
		Path:   pciProxyDestinationPath + "/" + url.PathEscape(id) + "/" + action,
	})
	if err != nil {
		return nil, fmt.Errorf("%s pci proxy destination %s: %w", action, id, err)
	}

	return &destination, nil
}

// DeletePCIProxyDestination removes a destination. Yuno answers 204, so the
// body is empty.
func (c *Client) DeletePCIProxyDestination(ctx context.Context, id string) ([]byte, error) {
	data, err := c.DoRaw(ctx, Request{
		Method: http.MethodDelete,
		Path:   pciProxyDestinationPath + "/" + url.PathEscape(id),
	})
	if err != nil {
		return nil, fmt.Errorf("delete pci proxy destination %s: %w", id, err)
	}

	return data, nil
}

// ForwardPCIProxy sends a request through the forward proxy, which resolves the
// `{{vaulted_token...}}` expressions of the body inside Yuno's PCI environment
// and returns the destination response unchanged. The method is forwarded
// verbatim, so anything but POST has to be spelled out by the caller.
func (c *Client) ForwardPCIProxy(
	ctx context.Context, method string, body any, headers map[string]string,
) ([]byte, error) {
	if headers[HeaderProxyDestinationURL] == "" {
		return nil, fmt.Errorf("forward pci proxy request: %s is required", HeaderProxyDestinationURL)
	}

	data, err := c.DoRaw(ctx, Request{
		Method:  method,
		Path:    pciProxyForwardPath,
		Body:    body,
		Headers: headers,
	})
	if err != nil {
		return nil, fmt.Errorf("forward pci proxy request: %w", err)
	}

	return data, nil
}
