# plugin-deploy-local

The `local` deploy substrate for OpenCharly — `target: local` (host:local
direct-shell) and `host: user@machine` (SSH) deployments, served out-of-process
(`deploy:local`).

The plugin is a standalone Go module the charly loader host-builds and serves
over go-plugin gRPC. The host's plugin-side deploy target Invokes it
(`OpExecute`) with the deployment's InstallPlan views (the serializable per-step
IR, with secrets injected, `{{.Home}}` resolved, and each step's teardown ops
captured host-side) plus a venue descriptor, and the host's executor served on
the broker (`ShellExecutor` for `host:local`, `SSHExecutor` for
`host:user@machine`).

## What it provides

| Capability | Surface |
|---|---|
| `deploy:local` | the `local:` deploy substrate (`target: local`, and `host: user@machine` SSH) |

## How it works

The plugin dials back through the SDK executor and hands the plans to the shared
`kit.WalkPlans`:

- **Plugin-renderable steps** (`Op` write/cmd/download, `File`, `ShellHook` + the
  `env.d` managed-block finalizer, `ShellSnippet`, `ServicePackaged`,
  `ServiceCustom`, `RepoChange`) it EXECUTES itself via the F2 reverse legs
  (`RunSystem` / `RunUser` / `PutFile`), echoing the host-computed reverse ops.
- **Host-engine steps** (`Builder` / `LocalPkgInstall` / `SystemPackages` /
  act-`Op` / `ExternalPlugin`) it drives over the `RunHostStep` reverse leg.

It returns the combined teardown ops the host records in the install ledger and
replays at `charly deploy del` (record-and-replay).

## How to use it

Compose the plugin candy in a project's `candy:` list:

```yaml
- '@github.com/opencharly/plugin-deploy-local/candy/plugin-deploy-local:<tag>'
```

Then author a `local:` deploy:

```yaml
my-deploy:
    local:
        from: my-template
        host: local          # or user@machine for SSH
```

## Layout

- `candy/plugin-deploy-local/` — the plugin module: `plugin.go` (the deploy
  provider + `NewProvider()` / `NewMeta()` / `Invoke`), `schema/local.cue` (the
  self-contained `#DeployLocalPlugin`), `host_render_test.go`,
  `schema_serve_test.go`, `cmd/serve/main.go`.
- `charly.yml` — the root project manifest (`discover: candy`).
- `.github/workflows/tag-on-merge.yml` — CalVer tag + `CHANGELOG/` on merge.

## Related

- Owning skill: `/charly-local:local-deploy` — `target: local` deployments, the
  `host:` destination field, the install ledger, and ReverseOp teardown.
- `/charly-internals:plugin` — the plugin/provider model, including the `deploy`
  provider class.
- [`opencharly/charly`](https://github.com/opencharly/charly) — the charly CLI.
