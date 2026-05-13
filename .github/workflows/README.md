# GitHub Actions — `mgm-calendar-backend`

Three workflows ship in this repo:

| Workflow | Triggers | What it does |
|---|---|---|
| `ci.yml` | PRs · pushes to any branch except `main` / `staging` | `go vet` + `go build` + `go test -race` + a `docker buildx build` smoke (no push). |
| `staging.yml` | Push to `staging` · manual *Run workflow* | Test gate → multi-arch (`linux/amd64`, `linux/arm64`) build → push to DockerHub with tag `staging`. |
| `production.yml` | Push to `main` · `v*` tags · manual | Test gate → multi-arch build → push to DockerHub with `latest`, `production`, semver, and sha tags. |

`main` and `staging` don't run `ci.yml` because their own workflows include the same test gate as a prerequisite job — no duplicate runs.

## One-time setup

In **Settings → Secrets and variables → Actions**:

| Kind | Name | Value |
|---|---|---|
| Repository secret | `DOCKERHUB_USERNAME` | DockerHub account name. Image will be pushed as `DOCKERHUB_USERNAME/mgm-calendar-backend`. |
| Repository secret | `DOCKERHUB_TOKEN` | DockerHub access token (Account Settings → Security → New Access Token, Read & Write). |

Optional but recommended: define two **Environments** named `staging` and `production` (Settings → Environments). The publish jobs reference them via `environment:`, which lets you add required reviewers, restrict which branches can deploy, or move the secrets into the environment rather than the repo.

## Image tags

| Trigger | Tags pushed |
|---|---|
| Push to `staging` | `staging`, `staging-<sha>` |
| Push to `main` | `latest`, `production`, `sha-<short>` |
| Tag `v1.4.0` on `main` | `1.4.0`, `1.4`, `sha-<short>` |

## Local equivalents

Reproduce CI on your machine:

```bash
go vet ./...
go build ./...
go test -race ./...
docker buildx build .
```

## Releasing

```bash
# Promote staging → main via your normal PR flow first.
git checkout main && git pull
git tag v0.1.0
git push --tags
```

The `production.yml` workflow watches `v*` tags and publishes the semver tags alongside `latest`.
