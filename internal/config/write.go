package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Managed block markers for setup lifecycle (s1525).
// Content between markers is owned by `iomesh setup`; user edits outside are preserved.
const (
	ManagedBegin = "# BEGIN iomesh-setup-managed"
	ManagedEnd   = "# END iomesh-setup-managed"
)

// WriteToPath marshals cfg to TOML and writes path atomically (tmp + rename).
// Creates parent directories as needed. File mode 0o600 (user secrets may live nearby).
// Residual honesty: caller must keep dual_write false for product defaults.
func WriteToPath(path string, cfg *Config) error {
	if path == "" {
		return fmt.Errorf("config: write path empty")
	}
	if cfg == nil {
		return fmt.Errorf("config: write nil config")
	}
	// Never invent dual_write ON on write path defaults — pin honesty.
	// (Caller may still set DualWrite true explicitly for audit opt-in.)
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	header := fmt.Sprintf("# iomesh-tui config written %s (UTC)\n# dual_write honesty: product default OFF · setup never invents Memory GA\n",
		time.Now().UTC().Format(time.RFC3339))
	return atomicWrite(path, append([]byte(header), data...))
}

// WriteUser writes cfg to UserConfigPath().
func WriteUser(cfg *Config) error {
	path, err := UserConfigPath()
	if err != nil {
		return err
	}
	return WriteToPath(path, cfg)
}

// WriteSetupManagedFragment replaces (or appends) the managed block in path with fragment body.
// fragment should be TOML only (no markers). Existing content outside markers is preserved.
// dual_write must not be forced true by fragment (checked loosely via needle).
func WriteSetupManagedFragment(path, fragment string) error {
	if path == "" {
		return fmt.Errorf("config: write path empty")
	}
	frag := strings.TrimSpace(fragment)
	if frag == "" {
		return fmt.Errorf("config: empty managed fragment")
	}
	if strings.Contains(strings.ToLower(frag), "dual_write") && strings.Contains(frag, "true") {
		// Residual honesty: refuse writing dual_write = true via setup managed path.
		if strings.Contains(frag, "dual_write = true") || strings.Contains(frag, "dual_write=true") {
			return fmt.Errorf("config: setup managed fragment must not set dual_write = true (local-primary honesty)")
		}
	}
	block := ManagedBegin + "\n" +
		"# Owned by `iomesh setup` — re-run setup to refresh; edit outside this block freely.\n" +
		"# dual_write OFF · not Memory GA · secrets via env refs only · catalog ≠ Connected\n" +
		frag + "\n" +
		ManagedEnd + "\n"

	var existing []byte
	if data, err := os.ReadFile(path); err == nil {
		existing = data
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("config: read %s: %w", path, err)
	}

	var out string
	if len(existing) == 0 {
		out = "# iomesh-tui config (setup-managed fragment present)\n" + block
	} else {
		s := string(existing)
		begin := strings.Index(s, ManagedBegin)
		end := strings.Index(s, ManagedEnd)
		if begin >= 0 && end > begin {
			// Replace inclusive of end marker line.
			endLine := end + len(ManagedEnd)
			// consume trailing newline after end marker
			for endLine < len(s) && (s[endLine] == '\n' || s[endLine] == '\r') {
				endLine++
			}
			out = s[:begin] + block + s[endLine:]
		} else {
			// Append managed block.
			if !strings.HasSuffix(s, "\n") {
				s += "\n"
			}
			out = s + "\n" + block
		}
	}
	return atomicWrite(path, []byte(out))
}

// WriteSetupManagedUser writes managed fragment to the user config path.
func WriteSetupManagedUser(fragment string) (path string, err error) {
	path, err = UserConfigPath()
	if err != nil {
		return "", err
	}
	return path, WriteSetupManagedFragment(path, fragment)
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".iomesh-config-*.tmp")
	if err != nil {
		return fmt.Errorf("config: temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("config: chmod: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: rename: %w", err)
	}
	cleanup = false
	return nil
}
