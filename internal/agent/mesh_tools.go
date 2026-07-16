package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/iome-sh/iomesh-tui/internal/iomesh"
	"github.com/iome-sh/iomesh-tui/internal/router"
)

// RegisterMeshTools adds read-only mesh catalog helpers when catalog plane is on.
func (r *ToolRegistry) RegisterMeshTools(mesh *iomesh.Client) {
	if mesh == nil || !mesh.CatalogEnabled() {
		return
	}
	r.register("list_mesh_catalog", false, router.Tool{
		Type: "function",
		Function: router.ToolFunction{
			Name:        "list_mesh_catalog",
			Description: "List governed I/O Mesh data products from the catalog plane (fail-open if mesh/catalog unavailable). Optional query filter.",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"optional search filter"}},"additionalProperties":false}`),
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
