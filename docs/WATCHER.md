# Watcher & TUI

Watcher is opt-in and report-only by default. It never modifies your files.

```yaml
watcher:
  enabled: true        # default false
  auto_snapshot: false # default false; explicit opt-in only
  quiet_period: 5m     # min gap between auto-snapshots (default 5m)

snapshot:
  schedule: daily      # ""|daily|hourly; informational for timer users
  keep_last: 30        # 0 = keep all; enforced by `kage prune`
```

```
kage watch --once        # reconcile once, exit 0/2
kage watch               # live until Ctrl-C (needs enabled: true)
kage status              # includes "watcher: disabled|idle|active"
kage audit --history     # snapshots + restores + watch events
```

- Events append to `~/.local/share/kage/watch.log` (0600).
- Missed events are covered by reconcile every 30s + `--once`.
- `auto_snapshot: true` commits drift automatically, at most one snapshot
  per `quiet_period`; secret/conflict failures log at most once per minute.
- `kage prune --keep N` drops old metadata records (IDs never reused;
  git objects stay until `git gc` in the repository).

## Scheduled snapshots

```
kage timer install     # copy units + enable --now (needs systemd user)
kage timer status      # active|inactive|not-installed (read-only)
kage timer uninstall   # disable + remove units
```

Manual fallback (no systemctl): copy `systemd/kage-snapshot.{service,timer}`
to `~/.config/systemd/user/` and enable as usual. `KAGE_SYSTEMD_DIR`
overrides the unit directory (tests, containers).

## TUI

```
kage tui
```

Keys: `j/k` or arrows move, `enter` next, `d` unified diff (scroll with
`j/k`, window follows terminal height), `esc` back, `y` confirm restore,
`q` quit. Restore always shows the engine preview first; conflicts and
checkpoints behave exactly like `kage restore`. Diff output is capped at
500 lines and honors `redact_output`.

## Scheduled snapshots

```
mkdir -p ~/.config/systemd/user
cp systemd/kage-snapshot.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now kage-snapshot.timer
```
