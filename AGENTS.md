# AGENTS.md

This file provides guidance to coding agents (e.g. Claude Code, claude.ai/code) when working with code in this repository.

## Repository purpose

Go module `kubeops.dev/openshifter` — a Kubernetes operator that **makes a plain Kubernetes cluster behave more like OpenShift** for pod security purposes. On every namespace it annotates `pod-security.kubernetes.io/enforce=restricted` (so workloads default to PSA restricted), and on every pod create/update it validates that the requesting user has the OpenShift-style `use-scc` authorization. Lets you run OpenShift-style "Security Context Constraints" guardrails on plain Kubernetes.

The README is a Kubebuilder scaffold stub; treat this file as the source of truth.

The produced binary is the controller manager from `cmd/main.go`.

## Architecture

- `cmd/main.go` — entry point; controller-runtime manager bootstrap.
- `internal/controller/`:
  - `namespace_controller.go` — `NamespaceReconciler` ensures every namespace stays PSA-restricted (skips entries in `tracker.NSSkipList`).
  - `suite_test.go`, `namespace_controller_test.go` — envtest harness.
- `internal/webhook/`:
  - `nsAnnotator.go` — mutating webhook on `Namespace` create/update that sets `pod-security.kubernetes.io/enforce=restricted` unless skip-listed or already has a Pod Security label.
  - `podValidator.go` — validating webhook on `Pod` create/update that denies pods unless the requesting user passes the `use-scc` authorizer check (only on OpenShift-managed clusters, per `kmodules.xyz/client-go/cluster.IsOpenShiftManaged`).
- `internal/tracker/tracker.go` — the **`NSSkipList`** set of namespaces that bypass PSA enforcement (system namespaces, etc.). User contract; do not flip silently.
- `config/` — Kubebuilder Kustomize bundles: `cert-manager/`, `default/`, `manager/`, `network-policy/`, `prometheus/`, `rbac/`, `webhook/`.
- `test/` — e2e / integration helpers.
- `ns.yaml` — captured cluster `Namespace` set used for fixture testing.
- `PROJECT` — Kubebuilder metadata.
- `Dockerfile` — release image.
- `Makefile` — Kubebuilder-style harness with local Go toolchain. Tools install into `bin/`.
- `vendor/` — checked-in deps.

## Common commands

This repo uses a **local Go toolchain** (Kubebuilder Makefile pattern), not the AppsCode Docker harness.

- `make help` — list targets.
- `make build` (alias `make all`) — `manifests generate fmt vet`, then build manager.
- `make generate` — controller-gen DeepCopy generation.
- `make manifests` — controller-gen CRDs / RBAC / webhook manifests.
- `make fmt`, `make vet` — standard.
- `make lint` / `make lint-fix` — golangci-lint (auto-installs).
- `make test` — `manifests generate fmt vet envtest`, then Go tests.
- `make test-e2e` — spin up Kind and run e2e tests.
- `make run` — run the controller against `~/.kube/config` locally.
- `make docker-build` — build the manager image.

Run a single Go test:

```
go test ./internal/controller/... -run TestName -v
```

## Conventions

- Module path is `kubeops.dev/openshifter` (vanity URL). Imports must use that.
- License: Apache-2.0 (the Kubebuilder scaffold header is in place on most files).
- Sign off commits (`git commit -s`).
- The annotation key `pod-security.kubernetes.io/enforce` is a **Kubernetes-standard** label; don't replace it with a project-local key.
- The `use-scc` authorization verb and `IsOpenShiftManaged` gating live in `internal/webhook/podValidator.go`; do not bypass either without an opt-in flag.
- `tracker.NSSkipList` is the project's escape hatch — add system namespaces there rather than scattering `if name == "..."` checks across reconcilers.
- This is a **Kubebuilder project** (`PROJECT` file present). Use `kubebuilder` to scaffold new APIs/controllers; don't hand-create files that `PROJECT` should track.
