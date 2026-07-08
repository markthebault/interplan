# Release Process

Interplan uses GitHub Actions, Conventional Commits, and Release Please to automate version numbers, changelogs, Git tags, GitHub Releases, and release binaries.

## Overview

There are three automation layers:

1. **CI**: runs tests and formatting checks on pull requests and pushes to `main`.
2. **Release Please**: watches commits on `main`, computes the next version, and opens or updates a release pull request.
3. **Release binaries**: builds platform binaries and uploads them to the GitHub Release.

## Daily Development Flow

Use normal feature branches and pull requests.

```sh
git checkout -b feature/my-change
# edit files
git commit -m "feat: add layout warning detection"
git push origin feature/my-change
```

Open a pull request and merge it into `main` after CI is green.

After the merge, Release Please runs automatically on `main`.

## Commit Message Rules

Release Please reads commit messages to decide if a release is needed and what version number to use.

Use Conventional Commits:

```text
fix: correct browser reload after artifact edits
feat: add export command
feat!: change poll output contract
```

Version bump rules:

| Commit message | Version bump | Example |
|---|---:|---|
| `fix: ...` | patch | `v0.1.0` -> `v0.1.1` |
| `feat: ...` | minor | `v0.1.1` -> `v0.2.0` |
| `feat!: ...` | major | `v1.2.3` -> `v2.0.0` |
| `BREAKING CHANGE:` in commit body | major | `v1.2.3` -> `v2.0.0` |

Non-release commit types do not normally create a release:

```text
docs: update release process
ci: update GitHub Actions
chore: clean generated files
test: add server tests
refactor: simplify CLI parsing
```

Use non-release types for internal-only changes.

## What Happens After Merging To Main

When commits land on `main`, Release Please checks all unreleased commits.

If there is at least one release-worthy commit, such as `fix:` or `feat:`, it opens or updates a pull request named like:

```text
chore: release v0.2.0
```

That pull request contains:

- `CHANGELOG.md` updates.
- `.release-please-manifest.json` version update.

If new commits are merged to `main` while the release pull request is open, Release Please updates the same release pull request with the latest version and changelog.

## How To Publish A Release

To publish a release, merge the Release Please pull request.

After that merge, Release Please automatically creates:

- a Git tag, for example `v0.2.0`;
- a GitHub Release;
- release notes based on the changelog.

Then the release binary workflow builds and uploads these assets:

```text
interplan_vX.Y.Z_darwin_amd64.tar.gz
interplan_vX.Y.Z_darwin_arm64.tar.gz
interplan_vX.Y.Z_linux_amd64.tar.gz
interplan_vX.Y.Z_linux_arm64.tar.gz
interplan_vX.Y.Z_windows_amd64.zip
checksums.txt
```

The release workflow passes the GitHub release tag into `scripts/build-release.sh`, and the build script embeds it into the binary with Go linker flags. That means a release binary prints the tag it was built from:

```sh
interplan --version
# interplan v0.2.0
```

Snapshot binaries use a `snapshot-<short-sha>` version, and local builds without release flags report `interplan dev`.

## Snapshot Binaries On Main

Every push to `main` runs CI.

If CI passes, the workflow builds snapshot binaries and stores them as temporary GitHub Actions artifacts.

Snapshot artifacts are useful for testing the current `main` branch before publishing a release.

Snapshot artifacts are not official releases and expire after the retention period configured in `.github/workflows/ci.yml`.

## Should I Use GitHub's Manual Release UI?

Do not use GitHub's manual Release UI for normal releases.

Normal releases must go through Release Please so that:

- version numbers are computed from commits;
- `CHANGELOG.md` stays correct;
- tags are created consistently;
- binaries are uploaded by automation.

Use GitHub's manual Release UI only for emergency edits to an already-created release description.

## If A Commit Did Not Use `fix:` Or `feat:`

If a commit does not use a release-worthy type, Release Please will not bump the version for that commit.

Example commit messages that do not trigger a release:

```text
Update README
Refactor server
WIP
ci: update workflow
```

If the change should be released but the commit message was wrong, create a new empty commit with the correct type:

```sh
git commit --allow-empty -m "fix: include previous server correction in release"
git push origin main
```

Use `fix:` for bug fixes and `feat:` for user-visible new features.

## Pre-1.0 Versioning Policy

Interplan currently uses `v0.x.y` versions.

Before `v1.0.0`:

- `fix:` increments the patch version.
- `feat:` increments the minor version.
- breaking changes should use `feat!:` and may move the project to the next major version when the project is stable enough.

Examples:

```text
v0.1.0 -> v0.1.1 for fix
v0.1.1 -> v0.2.0 for feat
```

## Local Checks Before Opening A Pull Request

Run:

```sh
gofmt -w .
go test ./...
go build -o ./bin/interplan ./cmd/interplan
```

Do not commit generated release archives from `dist/`.

## Relevant Files

```text
.github/workflows/ci.yml
.github/workflows/release-please.yml
.github/workflows/release.yml
release-please-config.json
.release-please-manifest.json
scripts/build-release.sh
scripts/changelog.sh
```
