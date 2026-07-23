package iomesh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListCatalog_ProductsAndFailOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/catalog/data-products" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("tenant") != "acme" {
			t.Errorf("tenant=%q", r.URL.Query().Get("tenant"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]string{
				{"id": "ops-incidents", "layer": "operational", "subject": "dept.sre.incidents", "title": "Incidents"},
				{"id": "crm-contacts", "layer": "knowledge", "subject": "dept.sales.contacts", "name": "CRM"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{
		Enabled: true, Endpoint: srv.URL, Tenant: "acme", CatalogPlane: true,
	}, nil)
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "mesh" || len(res.Products) != 2 {
		t.Fatalf("%+v", res)
	}
	out := FormatCatalog(res)
	if !strings.Contains(out, "ops-incidents") || !strings.Contains(out, "operational") {
		t.Fatal(out)
	}
	snip := CatalogSnippet(res, 1)
	if !strings.Contains(snip, "ops-incidents") || !strings.Contains(snip, "list_mesh_catalog") {
		t.Fatal(snip)
	}

	empty := httptest.NewServer(http.NotFoundHandler())
	defer empty.Close()
	c2 := New(Config{Enabled: true, Endpoint: empty.URL, CatalogPlane: true}, nil)
	res2 := c2.ListCatalog(context.Background(), "q")
	if res2.Source != "fail-open" {
		t.Fatalf("%+v", res2)
	}
}

func TestListCatalog_PortalFederation(t *testing.T) {
	// Broker paths 404; portal v17 succeeds with portal field names.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v17/portal/catalog/data-products":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"version": "v17-test",
				"products": []map[string]any{
					{
						"id":              "engineering-github-events",
						"name":            "GitHub Events",
						"mesh_layer":      "operational",
						"subject_pattern": "dept.engineering.github.>",
						"summary":         "GitHub webhook stream",
						"sample_subjects": []string{"dept.engineering.github.push"},
						"lineage":         []string{"github", "connector", "mesh"},
						"status":          "ga",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Endpoint: srv.URL, CatalogPlane: true}, nil)
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "portal" || !strings.Contains(res.Detail, "/v17/") {
		t.Fatalf("%+v", res)
	}
	if len(res.Products) != 1 {
		t.Fatalf("%+v", res)
	}
	p := res.Products[0]
	if p.Layer != "operational" || p.Subject == "" || p.Description == "" {
		t.Fatalf("normalize failed: %+v", p)
	}
	out := FormatCatalog(res)
	if !strings.Contains(out, "engineering-github") || !strings.Contains(out, "source=portal") {
		t.Fatal(out)
	}
}

func TestGetCatalogProduct_Detail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v17/portal/catalog/data-products/engineering-github-events" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "engineering-github-events", "name": "GitHub Events",
				"mesh_layer": "operational", "summary": "detail ok",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL, CatalogPlane: true}, nil)
	p, meta := c.GetCatalogProduct(context.Background(), "engineering-github-events")
	if meta.Source != "portal" || p.ID != "engineering-github-events" {
		t.Fatalf("p=%+v meta=%+v", p, meta)
	}
	d := FormatProductDetail(p, meta)
	if !strings.Contains(d, "detail ok") {
		t.Fatal(d)
	}
}

func TestFormatProductDetail_Fields(t *testing.T) {
	detail := FormatProductDetail(DataProduct{
		ID:          "ops-incidents",
		Name:        "Incidents",
		Title:       "SRE Incidents",
		Layer:       "operational",
		Subject:     "dept.sre.incidents",
		Status:      "ga",
		Department:  "sre",
		Description: "incident stream",
		Lineage:     []string{"pagerduty", "connector", "mesh"},
		Subjects:    []string{"dept.sre.incidents.>", "dept.sre.alerts.>"},
	}, CatalogResult{Source: "portal", Detail: "/v17/portal/catalog/data-products/ops-incidents"})
	for _, want := range []string{
		"iomesh catalog product source=portal detail=/v17/portal/catalog/data-products/ops-incidents",
		"id:          ops-incidents",
		"name:        SRE Incidents",
		"layer:       operational",
		"subject:     dept.sre.incidents",
		"status:      ga",
		"department:  sre",
		"description: incident stream",
		"lineage:",
		"  - pagerduty",
		"  - connector",
		"  - mesh",
		"subjects:",
		"  - dept.sre.incidents.>",
		"  - dept.sre.alerts.>",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
	if strings.Contains(detail, "(none)") {
		t.Fatalf("filled product should not print (none):\n%s", detail)
	}
}

// Sparse product still always-emits every scraper key (honest blanks / (none)).
func TestFormatProductDetail_EmptyAlwaysEmit(t *testing.T) {
	detail := FormatProductDetail(DataProduct{
		ID: "sparse-product",
	}, CatalogResult{Source: "mesh", Detail: ""})
	for _, want := range []string{
		"iomesh catalog product source=mesh detail=\n",
		"id:          sparse-product\n",
		"name:        sparse-product\n", // Title falls back via firstNonEmpty(Title, Name) then Normalize sets Title from ID
		"layer:       \n",
		"subject:     \n",
		"status:      \n",
		"department:  \n",
		"description: \n",
		"lineage:\n",
		"  (none)\n",
		"subjects:\n",
		"  (none)\n",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("missing %q in:\n%s", want, detail)
		}
	}
}

func TestFormatProductDetail_LineageSubjectsCap(t *testing.T) {
	lineage := make([]string, 15)
	subjects := make([]string, 14)
	for i := range lineage {
		lineage[i] = "step-" + string(rune('a'+i))
	}
	for i := range subjects {
		subjects[i] = "subj-" + string(rune('a'+i))
	}
	detail := FormatProductDetail(DataProduct{
		ID: "capped", Lineage: lineage, Subjects: subjects,
	}, CatalogResult{Source: "mesh", Detail: "x"})
	if !strings.Contains(detail, "  … +3 more\n") {
		t.Fatalf("expected lineage cap +3 more:\n%s", detail)
	}
	if !strings.Contains(detail, "  … +2 more\n") {
		t.Fatalf("expected subjects cap +2 more:\n%s", detail)
	}
}

func TestListCatalog_Off(t *testing.T) {
	c := New(Config{Enabled: true, Endpoint: "http://127.0.0.1:1", CatalogPlane: false}, nil)
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "off" {
		t.Fatalf("%+v", res)
	}
}

func TestDecodeCatalogArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/catalog/data-products" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/v1/catalog/products" {
			_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "x", "layer": "analytical"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL, CatalogPlane: true}, nil)
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "mesh" || len(res.Products) != 1 || res.Products[0].ID != "x" {
		t.Fatalf("%+v", res)
	}
}
