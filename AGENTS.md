# AGENTS.md — plugin-deploy-local

Standalone out-of-tree DEPLOY plugin repo serving the `local` deploy substrate
(`deploy:local`). The plugin is a Go module at `candy/plugin-deploy-local/`
(module path `github.com/opencharly/plugin-deploy-local/candy/plugin-deploy-local`);
the root `charly.yml` only declares `discover: candy` so the repo is a project
and its candy is scanned.

Canonical files:

- `candy/plugin-deploy-local/charly.yml` — the `plugin-deploy-local:` candy
  entity (`plugin:` block, `plan:` check).
- `candy/plugin-deploy-local/plugin.go` — the deploy provider (`NewProvider()` /
  `NewMeta()` / the `Invoke` plan walk via `kit.WalkPlans`).
- `candy/plugin-deploy-local/schema/local.cue` — the self-contained
  `#DeployLocalPlugin`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.
- `README.md` — user overview only; never agent guidance.

## Load these skills first (R0)

- `/charly-local:local-deploy` — `target: local` deployments, the Ansible-style
  `host:` destination field, the managed ssh_config fragment, the install ledger,
  and ReverseOp teardown. Load before changing the deploy walk. This candy
  carries no `skill:` entity of its own; the gap is tracked in
  [opencharly/opencharly#291](https://github.com/opencharly/opencharly/issues/291).
- `/charly-internals:plugin` — the plugin authoring reference: the `plugin:`
  block, the `deploy` provider class, the per-plugin CUE-schema contract.
- `/charly-internals:git-workflow` — before any git/PR action.

## Build / validate / test

- `go build ./...` in `candy/plugin-deploy-local/` — compile the plugin module.
- `go test ./...` in `candy/plugin-deploy-local/` — the plugin's Go tests (the
  host render + schema-serve seams).
- `charly box validate` at the repo root — the structural check (the candy +
  `plugin:` block, CUE schema).
- The merge gate is the **org-wide** `charly/pr-validator` (required check
  `validate / validate`, defined in `opencharly/.github`); this repo has **no**
  per-repo candy gate.
- The changed path is exercised by any `target: local` deploy.

## Modify this repo

- Edit the `plugin-deploy-local:` candy entity, the Go source, and
  `schema/local.cue` **together** — the schema is the served declaration surface.
- The substrate carries no authored `plugin_input` (`InputDef: ""`): its input is
  the InstallPlan views. Keep the plan walk in the shared `kit.WalkPlans` — one
  implementation shared with the other substrate plugins (R3), not a local copy.

## Landing

- PR-only. Every change lands through a pull request; the org-required
  `charly/pr-validator` validates the diff and body and arms native auto-merge on
  PASS. Direct pushes to `main` are blocked.
- History lives in `CHANGELOG/` (written by `tag-on-merge` at merge time); the PR
  body IS the changelog.
- The authoritative rulebook is the umbrella `AGENTS.md` in
  `opencharly/opencharly` and `charly/AGENTS.md` in the charly repo. Do not
  restate its rules here.
