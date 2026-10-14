# Kage — Configuration State Manager

Linux-first CLI: snapshot, diff, restore dotfiles/config via local Git.

## Quickstart

```
go build -o bin/kage ./cmd/kage
./bin/kage init
./bin/kage snapshot --message "initial"
./bin/kage status
./bin/kage log
```

Config: `~/.config/kage/config.yaml`. Repo: `~/.local/share/kage/repository`.

Exit codes: 0 ok, 1 error, 2 changes/drift, 3 conflict/secret-block.
