package mcp

import (
	"testing"
)

func TestApplyIOMeshContextHeaders_SetsNonEmpty(t *testing.T) {
	got := ApplyIOMeshContextHeaders(nil, "acme", "org_1", "ws_a", "eng")
	if got[HeaderIOMeshTenant] != "acme" {
		t.Fatalf("tenant=%q", got[HeaderIOMeshTenant])
	}
	if got[HeaderIOMeshOrg] != "org_1" {
		t.Fatalf("org=%q", got[HeaderIOMeshOrg])
	}
	if got[HeaderIOMeshWorkspace] != "ws_a" {
		t.Fatalf("workspace=%q", got[HeaderIOMeshWorkspace])
	}
	if got[HeaderIOMeshDepartment] != "eng" {
		t.Fatalf("department=%q", got[HeaderIOMeshDepartment])
	}
}

func TestApplyIOMeshContextHeaders_SkipsEmpty(t *testing.T) {
	got := ApplyIOMeshContextHeaders(map[string]string{"Authorization": "Bearer x"}, "  ", "", "ws_only", "")
	if _, ok := got[HeaderIOMeshTenant]; ok {
		t.Fatalf("empty tenant must not be sent: %v", got)
	}
	if _, ok := got[HeaderIOMeshOrg]; ok {
		t.Fatalf("empty org must not be sent: %v", got)
	}
	if _, ok := got[HeaderIOMeshDepartment]; ok {
		t.Fatalf("empty department must not be sent: %v", got)
	}
	if got[HeaderIOMeshWorkspace] != "ws_only" {
		t.Fatalf("workspace=%q", got[HeaderIOMeshWorkspace])
	}
	if got["Authorization"] != "Bearer x" {
		t.Fatalf("other headers preserved: %v", got)
	}
}

func TestApplyIOMeshContextHeaders_DepartmentOnly(t *testing.T) {
	got := ApplyIOMeshContextHeaders(nil, "", "", "", "research")
	if got[HeaderIOMeshDepartment] != "research" {
		t.Fatalf("department=%q", got[HeaderIOMeshDepartment])
	}
	if _, ok := got[HeaderIOMeshTenant]; ok {
		t.Fatalf("must not invent tenant: %v", got)
	}
	if _, ok := got[HeaderIOMeshOrg]; ok {
		t.Fatalf("must not invent org: %v", got)
	}
	if _, ok := got[HeaderIOMeshWorkspace]; ok {
		t.Fatalf("must not invent workspace: %v", got)
	}
}

func TestApplyIOMeshContextHeaders_NoOverwrite(t *testing.T) {
	// Explicit server header wins (canonical key).
	in := map[string]string{
		HeaderIOMeshOrg:        "org_explicit",
		HeaderIOMeshDepartment: "dept_explicit",
		"X-Custom":             "keep",
	}
	got := ApplyIOMeshContextHeaders(in, "tenant_cfg", "org_from_iomesh", "ws_cfg", "dept_from_iomesh")
	if got[HeaderIOMeshOrg] != "org_explicit" {
		t.Fatalf("must not overwrite Org: %q", got[HeaderIOMeshOrg])
	}
	if got[HeaderIOMeshDepartment] != "dept_explicit" {
		t.Fatalf("must not overwrite Department: %q", got[HeaderIOMeshDepartment])
	}
	if got[HeaderIOMeshTenant] != "tenant_cfg" {
		t.Fatalf("tenant=%q", got[HeaderIOMeshTenant])
	}
	if got[HeaderIOMeshWorkspace] != "ws_cfg" {
		t.Fatalf("workspace=%q", got[HeaderIOMeshWorkspace])
	}
	if got["X-Custom"] != "keep" {
		t.Fatalf("custom=%q", got["X-Custom"])
	}
}

func TestApplyIOMeshContextHeaders_NoOverwriteCaseInsensitive(t *testing.T) {
	in := map[string]string{
		"x-iomesh-org":        "org_lower",
		"x-iomesh-department": "dept_lower",
	}
	got := ApplyIOMeshContextHeaders(in, "t", "org_would_overwrite", "w", "dept_would_overwrite")
	if got["x-iomesh-org"] != "org_lower" {
		t.Fatalf("case-insensitive preserve: %v", got)
	}
	if got["x-iomesh-department"] != "dept_lower" {
		t.Fatalf("case-insensitive department preserve: %v", got)
	}
	// Canonical key must not be double-set either.
	if v, ok := got[HeaderIOMeshOrg]; ok && v != "org_lower" {
		t.Fatalf("unexpected Org key: %q", v)
	}
	if v, ok := got[HeaderIOMeshDepartment]; ok && v != "dept_lower" {
		t.Fatalf("unexpected Department key: %q", v)
	}
}

func TestApplyIOMeshContextHeaders_AllEmptyLeavesNil(t *testing.T) {
	if got := ApplyIOMeshContextHeaders(nil, "", "  ", "", "  "); got != nil {
		t.Fatalf("expected nil when nothing to inject, got %v", got)
	}
}

func TestApplyIOMeshContextHeaders_DefaultUnchangedMap(t *testing.T) {
	// Simulates inject=false path: caller does not call Apply — headers stay as configured.
	// When Apply is called with empty context, original map is returned unchanged.
	in := map[string]string{"Authorization": "Bearer static"}
	got := ApplyIOMeshContextHeaders(in, "", "", "", "")
	if len(got) != 1 || got["Authorization"] != "Bearer static" {
		t.Fatalf("unchanged=%v", got)
	}
}
