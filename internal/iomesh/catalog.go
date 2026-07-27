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
// Text path is unchanged by s735; scrapers prefer FormatCatalogJSON / CatalogPrint.
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

// DataProductPrint is a CLI-side print DTO for mesh catalog --json product rows.
// Always-emit scraper fields (no omitempty). Wire DataProduct stays lean with
// omitempty. Call after Normalize() via NewDataProductPrint.
//
// s735: mold PubPrint s732 + StreamMessagesPrint s720 + KVKeysPrint s714.
// Peer aion s734 residual. Beta catalog · offline unit ≠ live APPLY · empty/0/[]
// honest · dual_write OFF · not full mesh RBAC GA · DTO ≠ invent catalog/product
// success · fail-open source honest · wire omitempty ≠ print always-emit.
type DataProductPrint struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Subject     string   `json:"subject"`
	Layer       string   `json:"layer"`
	Status      string   `json:"status"`
	Department  string   `json:"department"`
	Subjects    []string `json:"subjects"` // empty [] not null
	Lineage     []string `json:"lineage"`  // empty [] not null
}

// CatalogPrint is a CLI-side print DTO for mesh catalog --json list envelope.
// Always emits source / detail / query / count / products (empty [] not null)
// so scrapers see a stable envelope without omitempty gaps. Wire CatalogResult
// stays lean (no JSON tags). Source is honest (mesh|portal|fail-open|off).
//
// s735: mold StreamMessagesPrint s720 + KVKeysPrint s714. Peer aion s734.
// Beta catalog · offline unit ≠ live APPLY · empty/0/[] honest · dual_write OFF
// · not full mesh RBAC GA · DTO ≠ invent catalog success · portal federation
// not invent GA.
type CatalogPrint struct {
	Source   string             `json:"source"`
	Detail   string             `json:"detail"`
	Query    string             `json:"query"`
	Count    int                `json:"count"`
	Products []DataProductPrint `json:"products"` // empty [] not null
}

// NewDataProductPrint builds a product print DTO. Normalizes portal aliases
// first; maps common fields; nil subjects/lineage become []string{} so JSON
// emits [] not null. Empty strings are honest when unset.
func NewDataProductPrint(p DataProduct) DataProductPrint {
	p.Normalize()
	subjects := p.Subjects
	if subjects == nil {
		subjects = []string{}
	}
	lineage := p.Lineage
	if lineage == nil {
		lineage = []string{}
	}
	return DataProductPrint{
		ID:          p.ID,
		Name:        p.Name,
		Title:       p.Title,
		Description: p.Description,
		Subject:     p.Subject,
		Layer:       p.Layer,
		Status:      p.Status,
		Department:  p.Department,
		Subjects:    subjects,
		Lineage:     lineage,
	}
}

// NewCatalogPrint builds a catalog list print envelope. count is len(products);
// nil Products become []DataProductPrint{}. query is the operator filter as
// passed to ListCatalog (empty string honest when unset).
func NewCatalogPrint(res CatalogResult, query string) CatalogPrint {
	products := make([]DataProductPrint, 0, len(res.Products))
	for _, p := range res.Products {
		products = append(products, NewDataProductPrint(p))
	}
	return CatalogPrint{
		Source:   res.Source,
		Detail:   res.Detail,
		Query:    query,
		Count:    len(products),
		Products: products,
	}
}

// FormatCatalogJSON returns indented JSON for stage CI / scrapers.
// Always emits all CatalogPrint / DataProductPrint fields without omitempty gaps.
func FormatCatalogJSON(p CatalogPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"catalog json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// CatalogProductPrint — single-product detail always-emit envelope (s744).
// Reuses DataProductPrint for nested product. found=false honest when missing.
//
// s744: mold CatalogPrint s735 + PubPrint s732. Peer aion s743 residual.
// Beta catalog · offline unit ≠ live APPLY · dual_write OFF · not full mesh
// RBAC GA · empty/0/[]/false honest · DTO ≠ invent catalog/product success ·
// fail-open · found=false honest · s735 list ≠ product detail residual ·
// portal federation not invent GA · no invent GA.
type CatalogProductPrint struct {
	Source  string           `json:"source"`
	Detail  string           `json:"detail"`
	ID      string           `json:"id"` // requested id (empty honest)
	Found   bool             `json:"found"`
	Product DataProductPrint `json:"product"` // empty fields + [] subjects/lineage when not found
}

// NewCatalogProductPrint builds a single-product detail print envelope.
// id is the operator-requested product id (empty string honest when unset).
// When found is false, product is an empty DataProductPrint (empty strings +
// subjects/lineage []) — do not invent success from partial wire fields.
// Source/Detail come from GetCatalogProduct meta (mesh|portal|fail-open|off).
func NewCatalogProductPrint(id string, p DataProduct, meta CatalogResult, found bool) CatalogProductPrint {
	var product DataProductPrint
	if found {
		product = NewDataProductPrint(p)
	} else {
		// Empty always-emit product: no invent; subjects/lineage [] not null.
		product = NewDataProductPrint(DataProduct{})
	}
	return CatalogProductPrint{
		Source:  meta.Source,
		Detail:  meta.Detail,
		ID:      id,
		Found:   found,
		Product: product,
	}
}

// FormatCatalogProductJSON returns indented JSON for stage CI / scrapers.
// Always emits all CatalogProductPrint / DataProductPrint fields without
// omitempty gaps. Mold FormatCatalogJSON s735.
func FormatCatalogProductJSON(p CatalogProductPrint) string {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return `{"error":"catalog product json marshal failed"}` + "\n"
	}
	return string(b) + "\n"
}

// FormatProductDetail is a multi-line view for one product (agent / CLI).
// Pure helper with no network I/O.
//
// Always emits optional knobs for scrapers: status, department, description,
// lineage, subjects (empty string / blank when unset; empty lineage/subjects →
// "  (none)"). Header always includes detail= even when meta.Detail is empty.
// Description uses firstNonEmpty(Description, Summary). Lineage and subjects
// lists are capped at 12 items with "… +N more".
func FormatProductDetail(p DataProduct, meta CatalogResult) string {
	p.Normalize()
	var b strings.Builder
	fmt.Fprintf(&b, "iomesh catalog product source=%s detail=%s\n", meta.Source, meta.Detail)
	fmt.Fprintf(&b, "id:          %s\n", firstNonEmpty(p.ID, p.Name))
	fmt.Fprintf(&b, "name:        %s\n", firstNonEmpty(p.Title, p.Name))
	fmt.Fprintf(&b, "layer:       %s\n", p.Layer)
	fmt.Fprintf(&b, "subject:     %s\n", p.Subject)
	fmt.Fprintf(&b, "status:      %s\n", p.Status)
	fmt.Fprintf(&b, "department:  %s\n", p.Department)
	fmt.Fprintf(&b, "description: %s\n", firstNonEmpty(p.Description, p.Summary))
	b.WriteString("lineage:\n")
	if len(p.Lineage) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for i, step := range p.Lineage {
			if i >= 12 {
				fmt.Fprintf(&b, "  … +%d more\n", len(p.Lineage)-12)
				break
			}
			fmt.Fprintf(&b, "  - %s\n", step)
		}
	}
	b.WriteString("subjects:\n")
	if len(p.Subjects) == 0 {
		b.WriteString("  (none)\n")
	} else {
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
