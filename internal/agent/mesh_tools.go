package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

var meshToolNames = []string{
	"list_mesh_catalog",
	"get_mesh_catalog_product",
	"mesh_status",
}

// UnregisterMeshTools removes catalog mesh tools. Used by Runtime.ReplaceMesh.
func (r *ToolRegistry) UnregisterMeshTools() {
	if r == nil || r.funcs == nil {
		return
	}
	for _, name := range meshToolNames {
		delete(r.funcs, name)
		delete(r.meta, name)
	}
}

// RegisterMeshTools adds read-only mesh catalog helpers when catalog plane is on.
func (r *ToolRegistry) RegisterMeshTools(mesh *iomesh.Client) {
	if mesh == nil || !mesh.CatalogEnabled() {
		return
	}
	r.register("list_mesh_catalog", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_mesh_catalog",
			Description: "List governed I/O Mesh data products from the catalog plane or portal federation (fail-open). Optional query or mesh_layer filter (operational|knowledge|analytical).",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"optional search filter or mesh_layer"}},"additionalProperties":false}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			Query string `json:"query"`
		}
		if args != "" && args != "{}" {
			_ = json.Unmarshal([]byte(args), &p)
		}
		res := mesh.ListCatalog(ctx, p.Query)
		return iomesh.FormatCatalog(res), nil
	})

	r.register("get_mesh_catalog_product", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "get_mesh_catalog_product",
			Description: "Fetch one governed data product by id (portal detail or list fallback). Fail-open if missing.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"product id"}},"required":["id"],"additionalProperties":false}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal([]byte(args), &p); err != nil || strings.TrimSpace(p.ID) == "" {
			return "", fmt.Errorf("id required")
		}
		prod, meta := mesh.GetCatalogProduct(ctx, p.ID)
		if meta.Source == "fail-open" || meta.Source == "off" {
			return iomesh.FormatProductDetail(prod, meta), nil
		}
		return iomesh.FormatProductDetail(prod, meta), nil
	})

	r.register("mesh_status", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "mesh_status",
			Description: "Show I/O Mesh client status (endpoint, context/catalog/policy flags) and local LLM usage meter for this process.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}, func(ctx context.Context, args string) (string, error) {
		_ = ctx
		_ = args
		return fmt.Sprintf("%s\n%s", mesh.StatusLine(), iomesh.FormatUsage(mesh.Usage())), nil
	})
}
