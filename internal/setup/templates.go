package setup

import (
	"fmt"
	"strings"
)

// Profile selects which managed fragment sections to emit.
type Profile string

const (
	ProfileLocalMemory Profile = "local-memory"
	ProfilePlugins     Profile = "plugins"
	ProfileMesh        Profile = "mesh"
	ProfilePlatformMCP Profile = "platform-mcp"
	ProfileAll         Profile = "all"
)

// InitOptions configures managed TOML generation.
// Residual honesty: dual_write always false; secrets as env names only.
type InitOptions struct {
	// MemoryHTTPURL preferred attach (default http://127.0.0.1:8080/mcp).
	MemoryHTTPURL string
	// MemoryServer name for [[mcp.servers]] + [memory].server (default iomesh-memory-mcp).
	MemoryServer string
	// MemoryTenant default tenant (default default).
	MemoryTenant string
	// UseStdioMemory if true, emit command=iomesh-memory-mcp instead of URL.
	UseStdioMemory bool
	// PluginsDirs absolute or ~/ paths for [plugins].dirs.
	PluginsDirs []string
	// MeshEndpoint optional mesh base URL.
	MeshEndpoint string
	// MeshTenant optional.
	MeshTenant string
	// MeshAPIKeyEnv env var name only (default IOMESH_API_KEY).
	MeshAPIKeyEnv string
	// PlatformMCPURL streamable HTTP from portal Agent/MCP panel.
	PlatformMCPURL string
	// PlatformMCPName server name (default aion-platform).
	PlatformMCPName string
	// PlatformTokenEnv env name for Bearer (default AION_TOKEN).
	PlatformTokenEnv string
	// AutoRecall / AutoIngest for [memory].
	AutoRecall bool
	AutoIngest bool
	// Pull opt-in fields (stream/consumer for continuous pull / CLI; pull_continuous default false).
	PullStream   string
	PullConsumer string
}

// DefaultInitOptions returns residual-honest defaults (dual_write off · local primary).
func DefaultInitOptions() InitOptions {
	return InitOptions{
		MemoryHTTPURL:    "http://127.0.0.1:8080/mcp",
		MemoryServer:     "iomesh-memory-mcp",
		MemoryTenant:     "default",
		MeshAPIKeyEnv:    "IOMESH_API_KEY",
		PlatformMCPName:  "aion-platform",
		PlatformTokenEnv: "AION_TOKEN",
		AutoRecall:       true,
		AutoIngest:       false,
		PullStream:       "EVENTS",
		PullConsumer:     "tui-local-palace",
	}
}

// BuildManagedFragment returns TOML for the setup-managed block (no markers).
func BuildManagedFragment(profiles []Profile, opt InitOptions) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("setup: no profiles")
	}
	want := map[Profile]bool{}
	for _, p := range profiles {
		p = Profile(strings.TrimSpace(string(p)))
		if p == ProfileAll {
			want[ProfileLocalMemory] = true
			want[ProfilePlugins] = true
			want[ProfileMesh] = true
			want[ProfilePlatformMCP] = true
			continue
		}
		switch p {
		case ProfileLocalMemory, ProfilePlugins, ProfileMesh, ProfilePlatformMCP:
			want[p] = true
		default:
			return "", fmt.Errorf("setup: unknown profile %q (local-memory|plugins|mesh|platform-mcp|all)", p)
		}
	}
	def := DefaultInitOptions()
	if strings.TrimSpace(opt.MemoryHTTPURL) == "" {
		opt.MemoryHTTPURL = def.MemoryHTTPURL
	}
	if strings.TrimSpace(opt.MemoryServer) == "" {
		opt.MemoryServer = def.MemoryServer
	}
	if strings.TrimSpace(opt.MemoryTenant) == "" {
		opt.MemoryTenant = def.MemoryTenant
	}
	if strings.TrimSpace(opt.MeshAPIKeyEnv) == "" {
		opt.MeshAPIKeyEnv = def.MeshAPIKeyEnv
	}
	if strings.TrimSpace(opt.PlatformMCPName) == "" {
		opt.PlatformMCPName = def.PlatformMCPName
	}
	if strings.TrimSpace(opt.PlatformTokenEnv) == "" {
		opt.PlatformTokenEnv = def.PlatformTokenEnv
	}
	if strings.TrimSpace(opt.PullStream) == "" {
		opt.PullStream = def.PullStream
	}
	if strings.TrimSpace(opt.PullConsumer) == "" {
		opt.PullConsumer = def.PullConsumer
	}

	var b strings.Builder
	b.WriteString("# setup lifecycle fragment — dual_write OFF · not Memory GA · secrets via env only\n")
	b.WriteString("# catalog ≠ Connected · portal HITL for OAuth/install · agent MCP cannot write installs\n\n")

	// Features / MCP enable when any MCP-related profile
	if want[ProfileLocalMemory] || want[ProfilePlatformMCP] || want[ProfilePlugins] {
		b.WriteString("[features]\n")
		b.WriteString("mcp = true\n")
		b.WriteString("skills = true\n\n")
		b.WriteString("[mcp]\n")
		b.WriteString("enabled = true\n\n")
	}

	if want[ProfileMesh] {
		b.WriteString("[iomesh]\n")
		b.WriteString("enabled = true\n")
		if ep := strings.TrimSpace(opt.MeshEndpoint); ep != "" {
			fmt.Fprintf(&b, "endpoint = %q\n", ep)
		} else {
			b.WriteString("# endpoint = \"https://your-mesh.example\"  # set after portal signup\n")
		}
		if t := strings.TrimSpace(opt.MeshTenant); t != "" {
			fmt.Fprintf(&b, "tenant = %q\n", t)
		} else {
			b.WriteString("# tenant = \"dept.yourorg\"\n")
		}
		fmt.Fprintf(&b, "api_key_env = %q  # set env; never commit secret values\n", opt.MeshAPIKeyEnv)
		b.WriteString("emit_dept_streams = true\n")
		b.WriteString("context_plane = true\n\n")
	}

	if want[ProfilePlatformMCP] {
		name := opt.PlatformMCPName
		b.WriteString("[[mcp.servers]]\n")
		fmt.Fprintf(&b, "name = %q\n", name)
		if u := strings.TrimSpace(opt.PlatformMCPURL); u != "" {
			fmt.Fprintf(&b, "url = %q\n", u)
		} else {
			b.WriteString("# url = \"https://…/mcp\"  # copy from portal Settings → Agent/MCP\n")
			b.WriteString("url = \"https://example.invalid/mcp\"  # replace before use\n")
		}
		fmt.Fprintf(&b, "oauth_token_env = %q  # env name only\n", opt.PlatformTokenEnv)
		b.WriteString("mutating = false  # discovery tools residual; installs stay portal HITL\n\n")
	}

	if want[ProfileLocalMemory] {
		name := opt.MemoryServer
		b.WriteString("[[mcp.servers]]\n")
		fmt.Fprintf(&b, "name = %q\n", name)
		if opt.UseStdioMemory {
			b.WriteString("command = \"iomesh-memory-mcp\"\n")
			b.WriteString("args = [\"-palace-root\", \"~/.iomesh/palace\", \"-tenant\", \"" + opt.MemoryTenant + "\"]\n")
		} else {
			fmt.Fprintf(&b, "url = %q\n", opt.MemoryHTTPURL)
			b.WriteString("allow_loopback = true\n")
		}
		b.WriteString("mutating = true\n\n")

		b.WriteString("[memory]\n")
		b.WriteString("enabled = true\n")
		fmt.Fprintf(&b, "server = %q\n", name)
		fmt.Fprintf(&b, "tenant = %q\n", opt.MemoryTenant)
		fmt.Fprintf(&b, "auto_recall = %v\n", opt.AutoRecall)
		fmt.Fprintf(&b, "auto_ingest = %v\n", opt.AutoIngest)
		b.WriteString("dual_write = false  # OFF · local-primary · setup never invents Memory GA\n")
		fmt.Fprintf(&b, "pull_stream = %q\n", opt.PullStream)
		fmt.Fprintf(&b, "pull_consumer = %q  # required for continuous pull\n", opt.PullConsumer)
		// pull_continuous default false: in-session opt-in via /setup pull start or set true + reload/restart.
		// CLI iomesh memory pull remains a valid path either way.
		b.WriteString("pull_continuous = false  # opt-in continuous pull · /setup pull start or set true · CLI iomesh memory pull still valid\n")
		b.WriteString("\n")
	}

	if want[ProfilePlugins] {
		b.WriteString("[plugins]\n")
		b.WriteString("enabled = true\n")
		dirs := opt.PluginsDirs
		if len(dirs) == 0 {
			// Placeholder — operator should point at sample or custom package roots.
			b.WriteString("# dirs = [\"/absolute/path/to/iomesh-tui/examples/agent-plugins\"]\n")
			b.WriteString("dirs = []  # set package root(s); Discover ≠ Connected · not Agent Plugins GA\n")
		} else {
			b.WriteString("dirs = [")
			for i, d := range dirs {
				if i > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "%q", d)
			}
			b.WriteString("]\n")
		}
		b.WriteString("# package map ≠ Connected · dual_write OFF · not Memory GA\n")
	}

	return b.String(), nil
}

// ParseProfiles splits comma/space profile list.
func ParseProfiles(s string) []Profile {
	s = strings.TrimSpace(s)
	if s == "" {
		return []Profile{ProfileLocalMemory}
	}
	var out []Profile
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '|'
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, Profile(part))
	}
	return out
}
