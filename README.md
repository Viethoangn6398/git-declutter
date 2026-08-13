# GitDeclutter

**Know what's safe to delete before you delete it.**

Safely identify and clean stale local Git branches.

A merged pull request does not always mean your current local branch contains no unique work. GitDeclutter combines local Git history, remote tracking state, and GitHub/GitLab pull request metadata to classify every local branch as **SAFE**, **REVIEW**, **KEEP**, or **PROTECTED** — and explains why.

## Install

```bash
go install github.com/kunmi02/git-declutter@latest
```

After installation, Git exposes the binary as a subcommand:

```bash
git declutter scan
```

## Example

```text
$ git declutter scan

GitDeclutter
Repository: payments-api

Scanning 34 local branches...

SAFE TO REMOVE                                    12

✓ feature/oauth
  PR #418 merged · remote deleted · no local changes

✓ fix/payment-timeout
  Fully merged into main


NEEDS REVIEW                                      3

⚠ experiment/cache
  Remote deleted · no merged PR found


KEEP                                              4

✕ feature/refunds
  3 commits exist only locally


PROTECTED                                         3

🔒 main
🔒 release/*
🔒 hotfix/production

12 safe · 3 review · 4 keep · 3 protected
```

## Commands

| Command | Purpose |
| --- | --- |
| `git declutter scan` | Analyze local branches. Never deletes anything. |
| `git declutter why <branch>` | Explain one branch's classification. |
| `git declutter clean` | Interactively remove SAFE branches (recoverable). |
| `git declutter clean --dry-run` | Preview cleanup with no changes. |
| `git declutter clean --safe-only --yes` | Non-interactive SAFE cleanup. |
| `git declutter clean --permanent` | Delete without GitDeclutter recovery refs. |
| `git declutter restore <branch>` | Restore a recoverable branch. |
| `git declutter history` | List recoverable deletions. |
| `git declutter gc` | Expire recovery refs past retention. |
| `git declutter config` | View or change configuration. |

### Scan flags

```bash
git declutter scan --json
git declutter scan --offline
git declutter scan --refresh
git declutter scan --branch feature/foo
```

`--refresh` runs `git fetch --prune` and prints `Refreshing remote references...`. Default scans do not prune remotes.

## Safety model

| Status | Meaning |
| --- | --- |
| **SAFE** | Strong evidence that deleting the local branch will not lose unique work. Eligible for automatic cleanup. |
| **REVIEW** | Stale-looking, but evidence is incomplete. Never selected automatically. |
| **KEEP** | Evidence of active or unique work. Never automatically deleted. |
| **PROTECTED** | Current branch, default branch, worktree branch, or a configured pattern. |

False negatives are acceptable. False positives are not. If GitDeclutter is uncertain, it will not recommend automatic deletion.

A merged PR alone does **not** make a branch SAFE. GitDeclutter also checks whether local commits were added after the PR, whether the branch diverged, whether unique commits exist only locally, and whether the branch is in use by a worktree.

## Recovery

Normal cleanup preserves deleted branch tips for **30 days**:

```text
refs/git-declutter/recovery/<event-id>/<branch>
```

Restore with `git declutter restore`. Permanent deletion (`--permanent` / `--hard`) creates no GitDeclutter undo. Git itself may still retain unreachable objects via reflog and garbage collection; GitDeclutter does not claim physical object deletion.

```bash
git declutter config set recovery.retention 7d
git declutter config set recovery.retention 30d
git declutter config set recovery.retention 90d
git declutter config set recovery.retention forever
```

## Configuration

Global config lives in the OS config directory (`~/Library/Application Support/git-declutter/` on macOS, `~/.config/git-declutter/` on Linux).

Optional repository file:

```yaml
# .gitdeclutter.yml
version: 1
protected:
  - main
  - develop
  - release/*
  - production
recovery:
  enabled: true
  retention: 30d
cleanup:
  requireRemoteDeleted: true
```

Default protected patterns: `main`, `master`, `develop`, `development`, `release/*`, `production`, `prod`.

## Privacy

GitDeclutter is local-first. It does not require an account and does not upload repository contents, source, diffs, or commit messages to GitDeclutter-owned infrastructure.

Provider APIs are contacted only to fetch repository metadata (owner, name, branch names, commit SHAs, pull request state). Authentication uses existing `gh` / `GITHUB_TOKEN` or `glab` / `GITLAB_TOKEN` and is never copied into GitDeclutter config. There is no telemetry.

## Development

```bash
make test
make build
```
