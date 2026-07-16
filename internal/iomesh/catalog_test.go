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

	// 404 both paths → fail-open
	empty := httptest.NewServer(http.NotFoundHandler())
	defer empty.Close()
	c2 := New(Config{Enabled: true, Endpoint: empty.URL, CatalogPlane: true}, nil)
	res2 := c2.ListCatalog(context.Background(), "q")
	if res2.Source != "fail-open" {
		t.Fatalf("%+v", res2)
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
		_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "x", "layer": "analytical"}})
	}))
	defer srv.Close()
	c := New(Config{Enabled: true, Endpoint: srv.URL, CatalogPlane: true}, nil)
	res := c.ListCatalog(context.Background(), "")
	if res.Source != "mesh" || len(res.Products) != 1 || res.Products[0].ID != "x" {
		t.Fatalf("%+v", res)
	}
}
