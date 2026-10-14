package deploylocal

import (
	"context"
	osexec "os/exec"
	"strings"
	"testing"

	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/spec"
)

// recordingExec captures the scripts the walk runs on the venue, so the emitted host
// commands can be asserted without a live host.
type recordingExec struct {
	sysScripts []string
}

func (e *recordingExec) Venue(context.Context) (string, error) { return "fake://venue", nil }
func (e *recordingExec) RunSystem(_ context.Context, s string, _ []byte) error {
	e.sysScripts = append(e.sysScripts, s)
	return nil
}
func (e *recordingExec) RunUser(_ context.Context, s string, _ []byte) error {
	e.sysScripts = append(e.sysScripts, s)
	return nil
}
func (e *recordingExec) PutFile(_ context.Context, _ string, _ []byte, _ uint32, _ bool) error {
	return nil
}
func (e *recordingExec) GetFile(_ context.Context, _ string, _ bool) ([]byte, error) { return nil, nil }
func (e *recordingExec) RunCapture(_ context.Context, _ string) (string, string, int, error) {
	return "/home/u", "", 0, nil
}
func (e *recordingExec) RunHostStep(_ context.Context, _ spec.InstallStepView, _ []byte) ([]spec.ReverseOp, error) {
	return nil, nil
}

// TestWalkPlans_HostDownloadGuardsAndEmptyTo pins the fix for the host-renderer
// defect filed as sdk#302 and merged as opencharly/sdk#303, from THIS plugin's
// entry point: plugin-deploy-local hands its executor straight to kit.WalkPlans, so
// the host-venue emitted script for the exact plan that broke layer-kubernetes on
// target:local — a `download:` with `extract: sh`, `unless_exists`, and no `to:` —
// must (a) NOT emit an empty `install -d` argument, (b) honour unless_exists, and (c) parse under
// `sh -n`.
func TestWalkPlans_HostDownloadGuardsAndEmptyTo(t *testing.T) {
	rec := &recordingExec{}
	plans := []spec.InstallPlanView{{
		Steps: []spec.InstallStepView{{
			Kind:      "Op",
			Scope:     spec.ScopeSystem,
			CandyName: "kubernetes",
			Op: &spec.Op{
				Download:     "https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3",
				Extract:      "sh",
				UnlessExists: "/usr/bin/helm",
				RunAs:        "root",
			},
		}},
	}}
	if _, err := kit.WalkPlans(context.Background(), rec, plans, kit.WalkOpts{}); err != nil {
		t.Fatalf("WalkPlans: %v", err)
	}
	if len(rec.sysScripts) == 0 {
		t.Fatal("no script emitted for the download step")
	}
	script := strings.Join(rec.sysScripts, "\n")

	// (a) the exact defect: `install -d -m0755 ''` must be gone.
	if strings.Contains(script, "-m0755 ''") || strings.Contains(script, `-m0755 ""`) {
		t.Fatalf("empty-to download emitted install -d with an empty destination:\n%s", script)
	}
	// (b) the host renderer must honour unless_exists.
	if !strings.Contains(script, "if [ -e '/usr/bin/helm' ]; then") {
		t.Fatalf("unless_exists was not honoured by the host renderer:\n%s", script)
	}
	// (c) the emitted script must be valid sh — a wrong gate terminator fails here.
	c := osexec.Command("sh", "-n")
	c.Stdin = strings.NewReader(script)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("emitted script is not valid sh (%v):\n%s\n---\n%s", err, out, script)
	}
}

// TestWalkPlans_HostDownloadExpandsCandyVarInDestination pins the fix for sdk#381 (merged as
// opencharly/sdk#382, tagged v0.2026287.1411) FROM THIS PLUGIN'S ENTRY POINT — the same shape as the
// test above, and for the same reason: this plugin hands its executor straight to kit.WalkPlans, so a
// pin that predates the fix silently renders a destination the shell never expands.
//
// The defect it guards: the renderer double-quoted the URL (so a ${VAR} there expanded) while
// single-quoting the DESTINATION, so a `to:` naming a candy var became a path LITERALLY called
// "${VAR}" — resolved against the deploy's working directory. Measured on a real `target: local`
// deploy: /home/<user>/${FC_TARBALL} appeared, and the next step failed with "Cannot open: No such
// file or directory" for the real path.
//
// If anyone downgrades this module's sdk below the fix, this test is what fails.
func TestWalkPlans_HostDownloadExpandsCandyVarInDestination(t *testing.T) {
	const (
		wantDir  = "/var/cache/charly/firecracker"
		wantFile = wantDir + "/firecracker-v1.17.0-x86_64.tgz"
	)

	rec := &recordingExec{}
	plans := []spec.InstallPlanView{{
		Steps: []spec.InstallStepView{{
			Kind:      "Op",
			Scope:     spec.ScopeSystem,
			CandyName: "layer-firecracker",
			// The candy's own var: map, exactly as the walk receives it (the renderer substitutes
			// THESE names before quoting — the fix under test).
			CandyVars: map[string]string{"FC_TARBALL": wantFile},
			Op: &spec.Op{
				Download: "https://github.com/firecracker-microvm/firecracker/releases/download/v1.17.0/firecracker-v1.17.0-x86_64.tgz",
				Extract:  "none",
				To:       "${FC_TARBALL}",
				Mode:     "0644",
				RunAs:    "root",
			},
		}},
	}}

	if _, err := kit.WalkPlans(context.Background(), rec, plans, kit.WalkOpts{}); err != nil {
		t.Fatalf("WalkPlans: %v", err)
	}

	script := strings.Join(rec.sysScripts, "\n")
	if script == "" {
		t.Fatal("no script emitted for the download step")
	}

	if strings.Contains(script, "${FC_TARBALL}") {
		t.Fatalf("the destination kept its unexpanded var, so the shell writes a file literally named that:\n%s", script)
	}

	if !strings.Contains(script, "install -d -m0755 "+wantDir) {
		t.Fatalf("the destination directory was not created from the substituted value:\n%s", script)
	}

	if !strings.Contains(script, "-o "+wantFile) {
		t.Fatalf("the download did not target the substituted file path:\n%s", script)
	}

	// The rendered script must still be valid sh.
	c := osexec.Command("sh", "-n")
	c.Stdin = strings.NewReader(script)

	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("emitted script is not valid sh (%v):\n%s\n---\n%s", err, out, script)
	}
}
