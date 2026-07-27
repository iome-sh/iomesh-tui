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

// s735: CatalogPrint JSON always-emits envelope + product keys (no omitempty gaps).
func TestCatalogPrint_JSONAlwaysEmitKeys(t *testing.T) {
	t.Parallel()

	// Empty catalog (nil products) still emits source/detail/query/count/products; products=[].
	emptyDTO := NewCatalogPrint(CatalogResult{Source: "off", Detail: "mesh disabled"}, "")
	emptyJS := FormatCatalogJSON(emptyDTO)
	var emptyObj map[string]any
	if err := json.Unmarshal([]byte(emptyJS), &emptyObj); err != nil {
		t.Fatalf("empty marshal/unmarshal: %v\n%s", err, emptyJS)
	}
	for _, key := range []string{"source", "detail", "query", "count", "products"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["source"] != "off" || emptyObj["detail"] != "mesh disabled" {
		t.Fatalf("empty envelope identity: %s", emptyJS)
	}
	if emptyObj["query"] != "" {
		t.Fatalf("empty query want \"\"; got %v\n%s", emptyObj["query"], emptyJS)
	}
	if emptyObj["count"].(float64) != 0 {
		t.Fatalf("count want 0; got %v\n%s", emptyObj["count"], emptyJS)
	}
	prods, ok := emptyObj["products"].([]any)
	if !ok {
		t.Fatalf("products want array not null: %s", emptyJS)
	}
	if len(prods) != 0 {
		t.Fatalf("products want []; got %v\n%s", prods, emptyJS)
	}
	// products must be [] not null in raw JSON.
	if strings.Contains(emptyJS, `"products": null`) {
		t.Fatalf("products must not be null: %s", emptyJS)
	}

	// Sparse single product: all DataProductPrint keys; empty strings honest; subjects/lineage [].
	sparseDTO := NewCatalogPrint(CatalogResult{
		Source:   "fail-open",
		Detail:   "no catalog path succeeded",
		Products: []DataProduct{{ID: "sparse-product"}},
	}, "ops")
	sparseJS, err := json.Marshal(sparseDTO)
	if err != nil {
		t.Fatal(err)
	}
	var sparseObj map[string]any
	if err := json.Unmarshal(sparseJS, &sparseObj); err != nil {
		t.Fatal(err)
	}
	if sparseObj["query"] != "ops" || sparseObj["count"].(float64) != 1 {
		t.Fatalf("sparse envelope: %s", sparseJS)
	}
	sparseProds, ok := sparseObj["products"].([]any)
	if !ok || len(sparseProds) != 1 {
		t.Fatalf("sparse products: %s", sparseJS)
	}
	row, ok := sparseProds[0].(map[string]any)
	if !ok {
		t.Fatalf("product row: %s", sparseJS)
	}
	for _, key := range []string{
		"id", "name", "title", "description", "subject", "layer",
		"status", "department", "subjects", "lineage",
	} {
		if _, ok := row[key]; !ok {
			t.Fatalf("product missing key %q: %s", key, sparseJS)
		}
	}
	// Title falls back via Normalize (Name|ID); other strings empty-honest.
	if row["id"] != "sparse-product" {
		t.Fatalf("id: %s", sparseJS)
	}
	if row["title"] != "sparse-product" {
		t.Fatalf("title after Normalize want id fallback; got %v\n%s", row["title"], sparseJS)
	}
	if row["name"] != "" || row["description"] != "" || row["subject"] != "" ||
		row["layer"] != "" || row["status"] != "" || row["department"] != "" {
		t.Fatalf("empty strings not honest: %s", sparseJS)
	}
	for _, arrKey := range []string{"subjects", "lineage"} {
		arr, ok := row[arrKey].([]any)
		if !ok {
			t.Fatalf("%s want array not null: %s", arrKey, sparseJS)
		}
		if len(arr) != 0 {
			t.Fatalf("%s want []; got %v\n%s", arrKey, arr, sparseJS)
		}
	}
	// Raw JSON: subjects/lineage must be [] not null (omitempty would drop or nil→null).
	if strings.Contains(string(sparseJS), `"subjects":null`) || strings.Contains(string(sparseJS), `"lineage":null`) {
		t.Fatalf("subjects/lineage must not be null: %s", sparseJS)
	}

	// Populated product: all keys + non-empty arrays.
	popDTO := NewDataProductPrint(DataProduct{
		ID:          "ops-incidents",
		Name:        "Incidents",
		Title:       "SRE Incidents",
		Description: "incident stream",
		Subject:     "dept.sre.incidents",
		Layer:       "operational",
		Status:      "ga",
		Department:  "sre",
		Subjects:    []string{"dept.sre.incidents.>", "dept.sre.alerts.>"},
		Lineage:     []string{"pagerduty", "mesh"},
	})
	popJS, err := json.Marshal(popDTO)
	if err != nil {
		t.Fatal(err)
	}
	var popObj map[string]any
	if err := json.Unmarshal(popJS, &popObj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"id", "name", "title", "description", "subject", "layer",
		"status", "department", "subjects", "lineage",
	} {
		if _, ok := popObj[key]; !ok {
			t.Fatalf("populated missing key %q: %s", key, popJS)
		}
	}
	if popObj["id"] != "ops-incidents" || popObj["name"] != "Incidents" ||
		popObj["title"] != "SRE Incidents" || popObj["layer"] != "operational" ||
		popObj["status"] != "ga" || popObj["department"] != "sre" {
		t.Fatalf("populated scalars: %s", popJS)
	}
	subs := popObj["subjects"].([]any)
	lin := popObj["lineage"].([]any)
	if len(subs) != 2 || len(lin) != 2 {
		t.Fatalf("populated arrays: %s", popJS)
	}
	// Wire DataProduct still has omitempty — print DTO must not.
	wireJS, err := json.Marshal(DataProduct{ID: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wireJS), `"status"`) {
		// status omitempty on wire — if present with empty it would still show;
		// ensure lean wire does omit empty optional fields.
		t.Fatalf("unexpected: lean wire should omit empty status: %s", wireJS)
	}
	// Print DTO always includes status even when empty.
	printJS, err := json.Marshal(NewDataProductPrint(DataProduct{ID: "x"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(printJS), `"status"`) {
		t.Fatalf("print DTO must always-emit status: %s", printJS)
	}
}

// s744: CatalogProductPrint JSON always-emits detail envelope + nested product keys.
func TestCatalogProductPrint_JSONAlwaysEmitKeys(t *testing.T) {
	t.Parallel()

	// Not-found / empty: found=false; all envelope keys; product empty + subjects/lineage [].
	emptyDTO := NewCatalogProductPrint("missing-id", DataProduct{}, CatalogResult{
		Source: "fail-open",
		Detail: "product not found: missing-id",
	}, false)
	emptyJS := FormatCatalogProductJSON(emptyDTO)
	var emptyObj map[string]any
	if err := json.Unmarshal([]byte(emptyJS), &emptyObj); err != nil {
		t.Fatalf("empty marshal/unmarshal: %v\n%s", err, emptyJS)
	}
	for _, key := range []string{"source", "detail", "id", "found", "product"} {
		if _, ok := emptyObj[key]; !ok {
			t.Fatalf("empty json missing key %q: %s", key, emptyJS)
		}
	}
	if emptyObj["source"] != "fail-open" || emptyObj["detail"] != "product not found: missing-id" {
		t.Fatalf("empty envelope identity: %s", emptyJS)
	}
	if emptyObj["id"] != "missing-id" {
		t.Fatalf("requested id want missing-id; got %v\n%s", emptyObj["id"], emptyJS)
	}
	if emptyObj["found"] != false {
		t.Fatalf("found want false; got %v\n%s", emptyObj["found"], emptyJS)
	}
	prod, ok := emptyObj["product"].(map[string]any)
	if !ok {
		t.Fatalf("product want object not null: %s", emptyJS)
	}
	for _, key := range []string{
		"id", "name", "title", "description", "subject", "layer",
		"status", "department", "subjects", "lineage",
	} {
		if _, ok := prod[key]; !ok {
			t.Fatalf("empty product missing key %q: %s", key, emptyJS)
		}
	}
	if prod["id"] != "" || prod["name"] != "" || prod["title"] != "" {
		t.Fatalf("not-found must not invent product identity: %s", emptyJS)
	}
	for _, arrKey := range []string{"subjects", "lineage"} {
		arr, ok := prod[arrKey].([]any)
		if !ok {
			t.Fatalf("%s want array not null: %s", arrKey, emptyJS)
		}
		if len(arr) != 0 {
			t.Fatalf("%s want []; got %v\n%s", arrKey, arr, emptyJS)
		}
	}
	if strings.Contains(emptyJS, `"subjects": null`) || strings.Contains(emptyJS, `"lineage": null`) ||
		strings.Contains(emptyJS, `"subjects":null`) || strings.Contains(emptyJS, `"lineage":null`) {
		t.Fatalf("subjects/lineage must not be null: %s", emptyJS)
	}
	if strings.Contains(emptyJS, `"product": null`) || strings.Contains(emptyJS, `"product":null`) {
		t.Fatalf("product must not be null: %s", emptyJS)
	}

	// found=false with off source (mesh disabled honesty).
	offDTO := NewCatalogProductPrint("any", DataProduct{ID: "should-not-emit"}, CatalogResult{
		Source: "off", Detail: "mesh disabled",
	}, false)
	offJS := FormatCatalogProductJSON(offDTO)
	var offObj map[string]any
	if err := json.Unmarshal([]byte(offJS), &offObj); err != nil {
		t.Fatal(err)
	}
	if offObj["found"] != false || offObj["source"] != "off" {
		t.Fatalf("off not-found: %s", offJS)
	}
	offProd := offObj["product"].(map[string]any)
	if offProd["id"] != "" {
		t.Fatalf("found=false must ignore partial product: %s", offJS)
	}

	// Populated product: found=true; all keys + non-empty arrays.
	popDTO := NewCatalogProductPrint("ops-incidents", DataProduct{
		ID:          "ops-incidents",
		Name:        "Incidents",
		Title:       "SRE Incidents",
		Description: "incident stream",
		Subject:     "dept.sre.incidents",
		Layer:       "operational",
		Status:      "ga",
		Department:  "sre",
		Subjects:    []string{"dept.sre.incidents.>", "dept.sre.alerts.>"},
		Lineage:     []string{"pagerduty", "mesh"},
	}, CatalogResult{
		Source: "portal",
		Detail: "/v17/portal/catalog/data-products/ops-incidents",
	}, true)
	popJS := FormatCatalogProductJSON(popDTO)
	var popObj map[string]any
	if err := json.Unmarshal([]byte(popJS), &popObj); err != nil {
		t.Fatalf("populated: %v\n%s", err, popJS)
	}
	if popObj["found"] != true {
		t.Fatalf("found want true: %s", popJS)
	}
	if popObj["id"] != "ops-incidents" || popObj["source"] != "portal" {
		t.Fatalf("populated envelope: %s", popJS)
	}
	popProd := popObj["product"].(map[string]any)
	for _, key := range []string{
		"id", "name", "title", "description", "subject", "layer",
		"status", "department", "subjects", "lineage",
	} {
		if _, ok := popProd[key]; !ok {
			t.Fatalf("populated product missing key %q: %s", key, popJS)
		}
	}
	if popProd["id"] != "ops-incidents" || popProd["name"] != "Incidents" ||
		popProd["title"] != "SRE Incidents" || popProd["layer"] != "operational" ||
		popProd["status"] != "ga" || popProd["department"] != "sre" {
		t.Fatalf("populated product scalars: %s", popJS)
	}
	subs := popProd["subjects"].([]any)
	lin := popProd["lineage"].([]any)
	if len(subs) != 2 || len(lin) != 2 {
		t.Fatalf("populated arrays: %s", popJS)
	}

	// Sparse found product: empty strings honest; subjects/lineage [].
	sparseDTO := NewCatalogProductPrint("sparse-product", DataProduct{ID: "sparse-product"}, CatalogResult{
		Source: "mesh", Detail: "/v1/catalog/data-products/sparse-product",
	}, true)
	sparseJS, err := json.Marshal(sparseDTO)
	if err != nil {
		t.Fatal(err)
	}
	var sparseObj map[string]any
	if err := json.Unmarshal(sparseJS, &sparseObj); err != nil {
		t.Fatal(err)
	}
	if sparseObj["found"] != true {
		t.Fatalf("sparse found: %s", sparseJS)
	}
	sparseProd := sparseObj["product"].(map[string]any)
	if sparseProd["id"] != "sparse-product" {
		t.Fatalf("sparse id: %s", sparseJS)
	}
	if sparseProd["title"] != "sparse-product" {
		t.Fatalf("title after Normalize want id fallback; got %v\n%s", sparseProd["title"], sparseJS)
	}
	for _, arrKey := range []string{"subjects", "lineage"} {
		arr, ok := sparseProd[arrKey].([]any)
		if !ok {
			t.Fatalf("%s want array not null: %s", arrKey, sparseJS)
		}
		if len(arr) != 0 {
			t.Fatalf("%s want []; got %v\n%s", arrKey, arr, sparseJS)
		}
	}
	if strings.Contains(string(sparseJS), `"subjects":null`) || strings.Contains(string(sparseJS), `"lineage":null`) {
		t.Fatalf("subjects/lineage must not be null: %s", sparseJS)
	}

	// Empty requested id honest when New called with "".
	blankID := NewCatalogProductPrint("", DataProduct{}, CatalogResult{Source: "off", Detail: "mesh disabled"}, false)
	blankJS := FormatCatalogProductJSON(blankID)
	var blankObj map[string]any
	if err := json.Unmarshal([]byte(blankJS), &blankObj); err != nil {
		t.Fatal(err)
	}
	if blankObj["id"] != "" || blankObj["found"] != false {
		t.Fatalf("blank id envelope: %s", blankJS)
	}
}
