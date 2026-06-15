# Releasing

Release builds are produced by the `release.yml` workflow (which lives on each
`rev-*` branch, since that's where the client code is) and are triggered by
pushing a **revision-namespaced tag**.

## Tag format

```
rev<N>-v<MAJOR.MINOR.PATCH>
```

The revision number, then the semantic version. The workflow trigger is
`tags: ['rev*-v*']`.

| Revision branch | Example tag |
|---|---|
| `rev-225`   | `rev225-v1.0.0`   |
| `rev-244`   | `rev244-v1.0.0`   |
| `rev-245.2` | `rev245.2-v1.0.0` |
| `rev-254`   | `rev254-v1.0.0`   |
| `rev-274`   | `rev274-v1.2.0`   |

Notes:
- The rev part has **no hyphen** after `rev` (`rev225`, not the branch's
  `rev-225`).
- The **whole tag name** becomes the version string baked into the artifact
  names and the GitHub Release — e.g. `rev274-v1.0.0` produces
  `client-rev274-v1.0.0-linux-amd64.tar.gz`.
- Revision-namespacing means releases of different revisions never collide.

## Cutting a release

1. Check out the revision branch, confirm it's at the commit you want to release
   (usually the tip) and that CI is green:

   ```bash
   git switch rev-274
   ```

2. Create a **lightweight** tag and push it — pushing the tag is what triggers
   the release workflow:

   ```bash
   git tag rev274-v1.0.0
   git push origin rev274-v1.0.0
   ```

> **Use a lightweight tag** (`git tag <name>`), not an annotated one
> (`git tag -a`). An annotated tag is its own object that records a tagger
> name, email, and timestamp **with your local timezone** — exactly the
> metadata this project scrubs out of commit history. A lightweight tag is just
> a ref pointing at a commit, with no embedded date or identity, and the
> workflow only reads the tag *name*. If you must create an annotated tag,
> normalize its date:
> `GIT_COMMITTER_DATE='2026-01-01T00:00:00 +0000' git tag -a rev274-v1.0.0 -m "…"`.

## What the workflow does

On a `rev*-v*` tag push, `release.yml` runs three stages:

1. **test** — the shared CI gate (`make ci`: gofmt check, `go vet`, race tests,
   golangci-lint). A tag that fails the gate **never publishes** artifacts.
2. **build** — builds the client on five native GitHub-hosted runners (no
   cross-compilation): `linux/amd64`, `linux/arm64`, `darwin/amd64`,
   `darwin/arm64`, and `windows/amd64`; each archived (`.tar.gz` on Unix, `.zip`
   on Windows) and uploaded. (`windows/arm64` is intentionally not built — its
   cgo toolchain on hosted runners targets x86-64, and Windows-on-ARM runs the
   `windows/amd64` build via x64 emulation.)
3. **release** — downloads all artifacts, writes `SHA256SUMS`, and creates the
   GitHub Release with auto-generated notes.

## Prerequisites

- The tag must point at a commit **on the matching `rev-*` branch** — the build
  checks out the tagged commit and builds exactly that code.
- The release job needs `contents: write` on the `GITHUB_TOKEN` (already
  declared in the workflow); no other secrets are required.
