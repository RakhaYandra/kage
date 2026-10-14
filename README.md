# Kage — Configuration State Manager

Linux-first CLI: snapshot dotfiles/config into a local Git repo, diff state,
selectively restore with checkpoint + conflict checks, audit drift/secrets.

```
kage init
kage snapshot -m "initial"
kage diff
kage restore --snapshot 1 --file ~/.zshrc --dry-run
kage audit
kage tui                          # interactive picker
kage watch --once                 # opt-in watcher (see docs/WATCHER.md)
```

- Config: `~/.config/kage/config.yaml` · Repo: `~/.local/share/kage/repository`
- Exit codes: 0 ok, 1 error, 2 changes/drift, 3 conflict/secret-block
- Local-only. Secrets blocked by default, never sent anywhere.

Docs: `docs/INSTALL.md`, `docs/CONFIG.md`, `docs/USAGE.md`, `docs/RECOVERY.md`, `docs/WATCHER.md`.
