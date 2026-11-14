# Kage — Configuration State Manager

Linux-first CLI: snapshot dotfiles/config into a local Git repo, diff state,
restore selectively with checkpoint + conflict checks, audit drift/secrets.

Your distro can reinstall packages; it cannot reinstall the ten years of
dotfiles you tuned by hand. Kage treats them as state, not as files.

## Requirements

- Linux
- Go 1.27+ (to build)
- Git (the snapshot store is a plain Git repository you own)

## Installation

```sh
git clone https://github.com/RakhaYandra/kage && cd kage
make install        # builds bin/kage and installs to ~/.local/bin/kage (override with PREFIX)
```

Release binaries (linux amd64/arm64 + `SHA256SUMS`) are built with `make release`.

## Quickstart

```sh
kage init
kage track add ~/.config/hypr
kage snapshot -m "before hyprland update"
kage status                  # exit 2 on drift
kage diff                    # working tree vs latest snapshot; exit 2 on change
kage restore --snapshot 12 --file ~/.zshrc --dry-run
kage restore --snapshot 12 --file ~/.zshrc --yes
kage audit                   # secrets + drift; exit 0/2/3
```

## Commands

| Command | Effect |
|---|---|
| `kage init` | Create the local repository and config. |
| `kage track add\|remove\|list` | Choose which paths are tracked. |
| `kage snapshot -m MESSAGE` | Snapshot the tracked state into the local Git repo. |
| `kage status` | Drift at a glance. Exit 2 on drift. |
| `kage log` | Snapshot history. |
| `kage diff [--from ID --to ID \| --last N] [--file PATH]` | Diff working vs snapshot(s). Exit 2 on change. |
| `kage restore --snapshot ID --file PATH [--dry-run\|--yes]` | Restore one file with a checkpoint taken first. |
| `kage audit` | Audit drift and secret leakage. Exit 0 clean / 2 findings / 3 blocked. |
| `kage tui` | Interactive picker. |
| `kage watch --once` | Opt-in watcher (see docs/WATCHER.md). |
| `kage timer install` | Daily snapshot timer (systemd user unit). |
| `kage prune --keep N --yes` | Drop old records, keep the last N. |

## Exit codes

`0` ok · `1` error · `2` changes/drift/review · `3` conflict/secret-block

`--json` on `snapshot`, `status`, `log`, `audit` for automation.

## Safety model

- Local-only: no network, no telemetry, snapshots live in a Git repo you own.
- Secrets are blocked by default (audit flags them; nothing leaves the machine).
- `restore` is single-file with a checkpoint taken first; conflicts are
  surfaced, never auto-resolved.
- Nothing is deleted silently; `prune` requires `--yes`.

## Docs

`docs/INSTALL.md` · `docs/CONFIG.md` · `docs/USAGE.md` · `docs/RECOVERY.md` ·
`docs/WATCHER.md`

## License

No `LICENSE` file yet — all rights reserved for now; adding one is an open
decision (an explicit LICENSE choice is pending).
