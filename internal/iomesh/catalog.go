package iomesh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DataProduct is a governed catalog entry (data product / stream surface).
type DataProduct struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Subjects    []string `json:"subjects,omitempty"`
	Layer       string   `json:"layer,omitempty"` // operational | knowledge | analytical
	Freshness   string   `json:"freshness,omitempty"`
	Owner       string   `json:"owner,omitempty"`
}

// CatalogResult is a fail-open catalog list.
type CatalogResult struct {
	Products []DataProduct
	// Source: mesh | empty | fail-open | off
	Source string
	// Detail is a short operator note (error or path used).
	Detail string
}

// ListCatalog fetches data products from the mesh catalog plane.
// Tries GET /v1/catalog/data-products then /v1/catalog/products (404 → fail-open empty).
func (c *Client) ListCatalog(ctx context.Context, query string) CatalogResult {
	if c == nil || !c.Enabled() {
		return CatalogResult{Source: "off", Detail: "mesh disabled"}
	}
	if !c.cfg.CatalogPlane {
		return CatalogResult{Source: "off", Detail: "catalog plane disabled"}
	}
	return c.listCatalogPaths(ctx, strings.TrimSpace(query))
}

func (c *Client) listCatalogPaths(ctx context.Context, query string) CatalogResult {
	paths := []string{"/v1/catalog/data-products", "/v1/catalog/products"}
	var lastDetail string
	for _, path := range paths {
		u := strings.TrimRight(c.cfg.Endpoint, "/") + path
		vals := url.Values{}
		if query != "" {
			vals.Set("q", query)
		}
		if c.cfg.Tenant != "" {
			vals.Set("tenant", c.cfg.Tenant)
		}
		if enc := vals.Encode(); enc != "" {
			u += "?" + enc
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			lastDetail = err.Error()
			continue
		}
		c.auth(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.logger.Debug("iomesh catalog: request failed (fail-open)", "err", err, "path", path)
			lastDetail = err.Error()
			continue
		}
		products, detail, ok := readCatalogResponse(resp, path)
		_ = resp.Body.Close()
		if !ok {
			lastDetail = detail
			continue
		}
		return CatalogResult{Products: products, Source: "mesh", Detail: path}
	}
	if lastDetail == "" {
		lastDetail = "no catalog path succeeded"
	}
	return CatalogResult{Source: "fail-open", Detail: lastDetail}
}

func readCatalogResponse(resp *http.Response, path string) ([]DataProduct, string, bool) {
	if resp.StatusCode == http.StatusNotFound {
		return nil, path + " 404", false
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Sprintf("%s http %d", path, resp.StatusCode), false
	}
	products, err := decodeCatalogBody(resp)
	if err != nil {
		return nil, "decode: " + err.Error(), false
	}
	return products, path, true
}

func decodeCatalogBody(resp *http.Response) ([]DataProduct, error) {
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var arr []DataProduct
	if err := json.Unmarshal(raw, &arr); err == nil && (len(arr) > 0 || string(raw) == "[]") {
		return arr, nil
	}
	var obj struct {
		Products     []DataProduct `json:"products"`
		Items        []DataProduct `json:"items"`
		DataProducts []DataProduct `json:"data_products"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	switch {
	case len(obj.Products) > 0:
		return obj.Products, nil
	case len(obj.Items) > 0:
		return obj.Items, nil
	case len(obj.DataProducts) > 0:
		return obj.DataProducts, nil
	default:
		return nil, nil
	}
}

// FormatCatalog renders a compact table for CLI / agent tools.
func FormatCatalog(res CatalogResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh catalog source=%s", res.Source)
	if res.Detail != "" {
		fmt.Fprintf(&b, " detail=%s", res.Detail)
	}
	b.WriteByte('\n')
	if len(res.Products) == 0 {
		b.WriteString("(no data products)\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%-20s %-12s %-28s %s\n", "ID", "LAYER", "SUBJECT", "TITLE/NAME")
	for i, p := range res.Products {
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(res.Products)-50)
			break
		}
		id := firstNonEmpty(p.ID, p.Name)
		title := firstNonEmpty(p.Title, p.Name, p.Description)
		subj := p.Subject
		if subj == "" && len(p.Subjects) > 0 {
			subj = p.Subjects[0]
		}
		fmt.Fprintf(&b, "%-20s %-12s %-28s %s\n",
			truncateRunes(id, 20), truncateRunes(p.Layer, 12), truncateRunes(subj, 28), truncateRunes(title, 48))
	}
	return b.String()
}

// CatalogSnippet builds a short system-prompt block for composition (fail-open empty).
func CatalogSnippet(res CatalogResult, max int) string {
	if max <= 0 {
		max = 12
	}
	if len(res.Products) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Governed data products (catalog):\n")
	for i, p := range res.Products {
		if i >= max {
			fmt.Fprintf(&b, "… +%d more (use list_mesh_catalog)\n", len(res.Products)-max)
			break
		}
		id := firstNonEmpty(p.ID, p.Name)
		subj := p.Subject
		if subj == "" && len(p.Subjects) > 0 {
			subj = p.Subjects[0]
		}
		line := "- " + id
		if p.Layer != "" {
			line += " [" + p.Layer + "]"
		}
		if subj != "" {
			line += " subject=" + subj
		}
		if t := firstNonEmpty(p.Title, p.Description); t != "" {
			line += " — " + truncateRunes(t, 80)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
