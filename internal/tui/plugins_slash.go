package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iome-sh/iomesh-tui/internal/agentplugins"
)

// resetPluginsSlashSession clears session dogfood marker (tests only).
// SSOT: agentplugins.ResetSoftDogfoodSessionState (s1392 set · s1397 shared with onboard next status/export).
func resetPluginsSlashSession() {
	agentplugins.ResetSoftDogfoodSessionState()
}

func markPluginsSlashDogfoodSession(pass bool) {
	agentplugins.SetSoftDogfoodSessionState(pass)
}

// writePluginsResidualFooter prints ResidualSlashHonesty + s1829 dual-path next-step.
func writePluginsResidualFooter(out io.Writer) {
	fmt.Fprintln(out, agentplugins.ResidualSlashHonesty)
	fmt.Fprintln(out, strings.Join(agentplugins.PluginsNextStepLines(), "\n"))
}

// writePluginsDogfoodResidualFooter prints ResidualDogfoodHonesty + s1829 dual-path next-step
// (error paths that never reach ResidualSlashHonesty).
func writePluginsDogfoodResidualFooter(out io.Writer) {
	fmt.Fprintln(out, agentplugins.ResidualDogfoodHonesty)
	fmt.Fprintln(out, strings.Join(agentplugins.PluginsNextStepLines(), "\n"))
}

// pluginsHelp is bare /plugins and help/? copy (s1392 residual honesty · s1829 next-step).
func pluginsHelp() string {
	return strings.TrimSpace(`usage: /plugins [help|list|validate|smoke|status]  (alias /plugin)
  help|?              this residual-honest usage (also bare /plugins)
  list [dir...]       DiscoverAll fail-open residual (Discover ≠ Connected)
  validate [dir...]   ValidateDirs fail-open residual (OK ≠ install green)
  smoke               soft offline smoke both in-repo samples (aliases dogfood|soft|samples|offline)
  status              residual plugins pulse: samples_ok|samples_missing · dogfood_not_run

smoke = discover/validate only · no MCP dial · PATH residual · soft offline ≠ live smoke
Discover/list ≠ Connected · package load ≠ Memory GA · soft offline smoke ≠ invent Agent Plugins GA
never invent install green / Connected / INSTALL_STORE APPLY · dual_write OFF · book-demo OFF · portal HITL
CLI twin: iomesh plugins list|validate|smoke · continuum: /onboard next plugins · /onboard next status
` + agentplugins.ResidualSlashHonesty + "\n" + strings.Join(agentplugins.PluginsNextStepLines(), "\n"))
}

// handlePluginsList residual-honest /plugins list (s1392 · s1829 next-step).
// DiscoverAll fail-open; residual offline message when no dirs.
func handlePluginsList(out io.Writer, args []string) {
	dirs := nonEmptyArgs(args)
	if len(dirs) == 0 {
		// No dirs: residual-honest offline opt-in message (not invent empty-as-none / Connected).
		fmt.Fprintln(out, agentplugins.FormatListEmptyFooter(false, false))
		fmt.Fprintln(out, "tip: pass package roots: /plugins list examples/agent-plugins/hello-iome")
		fmt.Fprintln(out, "or soft offline smoke: /plugins smoke (both in-repo samples)")
		writePluginsResidualFooter(out)
		return
	}
	plugins, warns := agentplugins.DiscoverAll(dirs)
	for _, w := range warns {
		fmt.Fprintf(out, "plugins: %s\n", w)
	}
	if len(plugins) == 0 {
		fmt.Fprintln(out, agentplugins.FormatListEmptyFooter(false, true))
		writePluginsResidualFooter(out)
		return
	}
	fmt.Fprintln(out, agentplugins.FormatListHeader())
	for _, p := range plugins {
		fmt.Fprintln(out, agentplugins.FormatListRow(agentplugins.PluginToListRow(p)))
		for _, w := range p.Warnings {
			fmt.Fprintf(out, "plugins %s: %s\n", p.Manifest.Name, w)
		}
	}
	fmt.Fprintln(out, "note: Discover ≠ Connected · list ≠ invent Agent Plugins GA · package load ≠ Memory GA")
	writePluginsResidualFooter(out)
}

// handlePluginsValidate residual-honest /plugins validate (s1392 · s1829 next-step).
// ValidateDirs fail-open for TUI display (exit codes are CLI-only).
func handlePluginsValidate(out io.Writer, args []string) {
	dirs := nonEmptyArgs(args)
	if len(dirs) == 0 {
		fmt.Fprintln(out, agentplugins.FormatListEmptyFooter(false, false))
		fmt.Fprintln(out, "tip: pass package roots: /plugins validate examples/agent-plugins/hello-iome")
		fmt.Fprintln(out, "or soft offline smoke: /plugins smoke (both in-repo samples)")
		writePluginsResidualFooter(out)
		return
	}
	outcomes, scanWarns := agentplugins.ValidateDirs(dirs)
	for _, w := range scanWarns {
		fmt.Fprintf(out, "plugins: %s\n", w)
	}
	if len(outcomes) == 0 {
		fmt.Fprintln(out, agentplugins.FormatListEmptyFooter(false, true))
		writePluginsResidualFooter(out)
		return
	}
	for _, o := range outcomes {
		if o.OK {
			fmt.Fprintln(out, agentplugins.FormatValidateOK(o))
			for _, w := range o.Warnings {
				fmt.Fprintf(out, "plugins %s: %s\n", o.Name, w)
			}
		} else {
			fmt.Fprintln(out, agentplugins.FormatValidateFail(o.Path, o.Error))
		}
	}
	okCount := agentplugins.ValidateOKCount(outcomes)
	fmt.Fprintf(out, "validate summary: ok=%d/%d (fail-open TUI · OK ≠ Connected / install APPLY · ≠ invent Agent Plugins GA)\n",
		okCount, len(outcomes))
	writePluginsResidualFooter(out)
}

// handlePluginsDogfood residual-honest soft offline /plugins dogfood (s1392 · s1829 next-step).
// Calls DogfoodSamples with FindModuleRoot; never invents GA/Connected/live dogfood.
func handlePluginsDogfood(out io.Writer) {
	root, err := agentplugins.FindModuleRoot("")
	if err != nil {
		// Fallback: treat cwd as module root (operator may have samples without go.mod walk).
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			fmt.Fprintf(out, "smoke: find module root: %v\n", err)
			writePluginsDogfoodResidualFooter(out)
			markPluginsSlashDogfoodSession(false)
			return
		}
		root = cwd
		fmt.Fprintf(out, "smoke: go.mod not found above cwd; using cwd as module root (%s)\n", cwd)
	}
	outcomes, warns, err := agentplugins.DogfoodSamples(root)
	if err != nil {
		fmt.Fprintf(out, "smoke: %v\n", err)
		writePluginsDogfoodResidualFooter(out)
		markPluginsSlashDogfoodSession(false)
		return
	}
	for _, w := range warns {
		fmt.Fprintf(out, "plugins: %s\n", w)
	}
	for _, o := range outcomes {
		if o.OK {
			fmt.Fprintln(out, agentplugins.FormatValidateOK(o))
			for _, w := range o.Warnings {
				fmt.Fprintf(out, "plugins %s: %s\n", o.Name, w)
			}
		} else {
			fmt.Fprintln(out, agentplugins.FormatValidateFail(o.Path, o.Error))
		}
	}
	fmt.Fprintln(out, agentplugins.FormatDogfoodSummary(outcomes))
	pass := agentplugins.DogfoodPass(outcomes)
	markPluginsSlashDogfoodSession(pass)
	// Residual-honest framing: soft offline ≠ live dogfood ≠ Agent Plugins GA.
	fmt.Fprintln(out, "note: soft offline smoke PASS ≠ invent Agent Plugins GA · residual PASS ≠ live dogfood · Discover ≠ Connected · package load ≠ Memory GA")
	fmt.Fprintln(out, "session marker: "+agentplugins.SoftDogfoodSessionLabel()+" · session soft ≠ live dogfood · board/export evidence ≠ invent Connected")
	// s1397: tip re-run status board + export so session soft state refreshes residual evidence.
	fmt.Fprintln(out, "tip: re-run /onboard next status then /onboard next export — session soft smoke refreshes plugins lane (≠ invent Agent Plugins GA · ≠ live dogfood · board ≠ invent Connected)")
	fmt.Fprintln(out, agentplugins.ResidualDogfoodHonesty)
	writePluginsResidualFooter(out)
}

// handlePluginsStatus residual-honest /plugins status pulse (s1392 · s1397 shared session SSOT · s1829 next-step).
// samples_ok|samples_missing · dogfood_not_run default (session soft marker optional).
// ≠ live dogfood · ≠ invent Agent Plugins GA.
func handlePluginsStatus(out io.Writer) {
	samples := agentplugins.SamplesSoftState("")
	// Session SSOT in agentplugins (s1397) — shared with /onboard next status + export.
	dogfoodState := agentplugins.SoftDogfoodSessionLabel()
	fmt.Fprintln(out, strings.TrimSpace(fmt.Sprintf(`plugins status (residual-honest · s1392+s1521 · soft offline · no MCP dial · not live dogfood):
  samples: %s
  smoke: %s
  note: samples soft-check only · dogfood_not_run default · session soft marker ≠ live dogfood
  · soft offline dogfood ≠ invent Agent Plugins GA · Discover ≠ Connected · package load ≠ Memory GA
  · never invent install green / Connected / INSTALL_STORE APPLY · dual_write OFF · book-demo OFF
  slash: /plugins smoke (aliases dogfood|soft|samples|offline) · /plugins list · /plugins validate
  continuum: /onboard next plugins · /onboard next status · /onboard next export · iomesh plugins smoke
%s
%s`, samples, dogfoodState, agentplugins.ResidualSlashHonesty, strings.Join(agentplugins.PluginsNextStepLines(), "\n"))))
}

func nonEmptyArgs(args []string) []string {
	var out []string
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a != "" {
			out = append(out, a)
		}
	}
	return out
}
