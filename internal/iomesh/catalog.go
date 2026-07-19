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
// Fields accept both broker (/v1/catalog) and portal (/v17/portal/catalog) JSON shapes.
type DataProduct struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Summary     string   `json:"summary,omitempty"` // portal
	Subject     string   `json:"subject,omitempty"`
	SubjectPat  string   `json:"subject_pattern,omitempty"` // portal
	Subjects    []string `json:"subjects,omitempty"`
	SampleSubs  []string `json:"sample_subjects,omitempty"` // portal
	Layer       string   `json:"layer,omitempty"`           // operational | knowledge | analytical
	MeshLayer   string   `json:"mesh_layer,omitempty"`      // portal alias
	Freshness   string   `json:"freshness,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	Department  string   `json:"department,omitempty"`
	Status      string   `json:"status,omitempty"`
	Lineage     []string `json:"lineage,omitempty"`
}

// Normalize copies portal aliases into the common fields used by Format/Snippet.
func (p *DataProduct) Normalize() {
	if p.Layer == "" {
		p.Layer = p.MeshLayer
	}
	if p.Subject == "" {
		p.Subject = p.SubjectPat
	}
	if p.Description == "" {
		p.Description = p.Summary
	}
	if len(p.Subjects) == 0 && len(p.SampleSubs) > 0 {
		p.Subjects = p.SampleSubs
	}
	if p.Title == "" {
		p.Title = firstNonEmpty(p.Name, p.ID)
	}
}

// CatalogResult is a fail-open catalog list.
type CatalogResult struct {
	Products []DataProduct
	// Source: mesh | portal | fail-open | off
	Source string
	// Detail is a short operator note (error or path used).
	Detail string
}

// catalogPath is one discovery route with classification for Source.
type catalogPath struct {
	Path   string
	Source string // mesh | portal
}

// defaultCatalogPaths: broker first, then portal / control-plane edge federation.
func defaultCatalogPaths() []catalogPath {
	return []catalogPath{
		{Path: "/v1/catalog/data-products", Source: "mesh"},
		{Path: "/v1/catalog/products", Source: "mesh"},
		// Portal (I/O Mesh control plane) — public list when IOMESH_ENDPOINT points at CP/console edge.
		{Path: "/v17/portal/catalog/data-products", Source: "portal"},
		{Path: "/v16/portal/catalog/marketing/data-products", Source: "portal"},
	}
}

// ListCatalog fetches data products from the mesh catalog plane and/or portal federation.
// Tries broker /v1/catalog/* then portal /v17|/v16 paths (404 → next; all fail → fail-open).
func (c *Client) ListCatalog(ctx context.Context, query string) CatalogResult {
	if c == nil || !c.Enabled() {
		return CatalogResult{Source: "off", Detail: "mesh disabled"}
	}
	if !c.cfg.CatalogPlane {
		return CatalogResult{Source: "off", Detail: "catalog plane disabled"}
	}
	return c.listCatalogPaths(ctx, strings.TrimSpace(query))
}

// GetCatalogProduct fetches one product by id (portal detail or list filter fallback).
func (c *Client) GetCatalogProduct(ctx context.Context, id string) (DataProduct, CatalogResult) {
	id = strings.TrimSpace(id)
	empty := CatalogResult{Source: "off", Detail: "mesh disabled"}
	if c == nil || !c.Enabled() || !c.cfg.CatalogPlane || id == "" {
		return DataProduct{}, empty
	}
	// Prefer portal detail routes.
	detailPaths := []catalogPath{
		{Path: "/v17/portal/catalog/data-products/" + url.PathEscape(id), Source: "portal"},
		{Path: "/v1/catalog/data-products/" + url.PathEscape(id), Source: "mesh"},
	}
	for _, cp := range detailPaths {
		u := strings.TrimRight(c.cfg.Endpoint, "/") + cp.Path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		c.auth(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		products, detail, ok := readCatalogResponse(resp, cp.Path)
		_ = resp.Body.Close()
		if !ok || len(products) == 0 {
			_ = detail
			continue
		}
		p := products[0]
		p.Normalize()
		return p, CatalogResult{Products: products, Source: cp.Source, Detail: cp.Path}
	}
	// Fallback: list + filter by id.
	list := c.ListCatalog(ctx, id)
	for _, p := range list.Products {
		p.Normalize()
		if p.ID == id || p.Name == id {
			return p, CatalogResult{Products: []DataProduct{p}, Source: list.Source, Detail: list.Detail + " (list filter)"}
		}
	}
	return DataProduct{}, CatalogResult{Source: "fail-open", Detail: "product not found: " + id}
}

func (c *Client) listCatalogPaths(ctx context.Context, query string) CatalogResult {
	var lastDetail string
	for _, cp := range defaultCatalogPaths() {
		u := strings.TrimRight(c.cfg.Endpoint, "/") + cp.Path
		vals := url.Values{}
		if query != "" {
			// Broker q= ; portal often uses free-text or mesh_layer=
			vals.Set("q", query)
			if query == "operational" || query == "knowledge" || query == "analytical" {
				vals.Set("mesh_layer", query)
			}
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
			c.logger.Debug("iomesh catalog: request failed (fail-open)", "err", err, "path", cp.Path)
			lastDetail = err.Error()
			continue
		}
		products, detail, ok := readCatalogResponse(resp, cp.Path)
		_ = resp.Body.Close()
		if !ok {
			lastDetail = detail
			continue
		}
		for i := range products {
			products[i].Normalize()
		}
		return CatalogResult{Products: products, Source: cp.Source, Detail: cp.Path}
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
	// Single product object (detail endpoint).
	var one DataProduct
	if err := json.Unmarshal(raw, &one); err == nil && (one.ID != "" || one.Name != "") {
		// Avoid treating list envelopes as single product.
		var probe map[string]json.RawMessage
		if json.Unmarshal(raw, &probe) == nil {
			if _, hasProducts := probe["products"]; !hasProducts {
				if _, hasItems := probe["items"]; !hasItems {
					if _, hasDP := probe["data_products"]; !hasDP {
						one.Normalize()
						return []DataProduct{one}, nil
					}
				}
			}
		}
	}
	var arr []DataProduct
	if err := json.Unmarshal(raw, &arr); err == nil && (len(arr) > 0 || string(raw) == "[]") {
		return arr, nil
	}
	var obj struct {
		Version      string        `json:"version"`
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
		// Empty products array with version envelope is still success.
		if obj.Version != "" || string(raw) == "{}" {
			return nil, nil
		}
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
	fmt.Fprintf(&b, "%-24s %-12s %-28s %s\n", "ID", "LAYER", "SUBJECT", "TITLE/NAME")
	for i, p := range res.Products {
		p.Normalize()
		if i >= 50 {
			fmt.Fprintf(&b, "… (%d more)\n", len(res.Products)-50)
			break
		}
		id := firstNonEmpty(p.ID, p.Name)
		title := firstNonEmpty(p.Title, p.Name, p.Description, p.Summary)
		subj := p.Subject
		if subj == "" && len(p.Subjects) > 0 {
			subj = p.Subjects[0]
		}
		fmt.Fprintf(&b, "%-24s %-12s %-28s %s\n",
			truncateRunes(id, 24), truncateRunes(p.Layer, 12), truncateRunes(subj, 28), truncateRunes(title, 48))
	}
	return b.String()
}

// FormatProductDetail is a multi-line view for one product (agent / CLI).
func FormatProductDetail(p DataProduct, meta CatalogResult) string {
	p.Normalize()
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh catalog product source=%s detail=%s\n", meta.Source, meta.Detail)
	fmt.Fprintf(&b, "id:          %s\n", firstNonEmpty(p.ID, p.Name))
	fmt.Fprintf(&b, "name:        %s\n", firstNonEmpty(p.Title, p.Name))
	fmt.Fprintf(&b, "layer:       %s\n", p.Layer)
	fmt.Fprintf(&b, "subject:     %s\n", p.Subject)
	if p.Status != "" {
		fmt.Fprintf(&b, "status:      %s\n", p.Status)
	}
	if p.Department != "" {
		fmt.Fprintf(&b, "department:  %s\n", p.Department)
	}
	if d := firstNonEmpty(p.Description, p.Summary); d != "" {
		fmt.Fprintf(&b, "description: %s\n", d)
	}
	if len(p.Lineage) > 0 {
		b.WriteString("lineage:\n")
		for _, step := range p.Lineage {
			fmt.Fprintf(&b, "  - %s\n", step)
		}
	}
	if len(p.Subjects) > 0 {
		b.WriteString("subjects:\n")
		for i, s := range p.Subjects {
			if i >= 12 {
				fmt.Fprintf(&b, "  … +%d more\n", len(p.Subjects)-12)
				break
			}
			fmt.Fprintf(&b, "  - %s\n", s)
		}
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
	src := res.Source
	if src == "" {
		src = "mesh"
	}
	fmt.Fprintf(&b, "Governed data products (catalog source=%s):\n", src)
	for i, p := range res.Products {
		p.Normalize()
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
		if t := firstNonEmpty(p.Title, p.Description, p.Summary); t != "" {
			line += " — " + truncateRunes(t, 80)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
