# Watcher & TUI

Watcher is opt-in and report-only by default. It never modifies your files.

```yaml
watcher:
  enabled: true        # default false
  auto_snapshot: false # default false; explicit opt-in only
```

```
kage watch --once        # reconcile once, exit 0/2
kage watch               # live until Ctrl-C (needs enabled: true)
kage status              # includes "watcher: disabled|idle|active"
kage audit --history     # snapshots + restores + watch events
```

- Events append to `~/.local/share/kage/watch.log` (0600).
- Missed events are covered by reconcile every 30s + `--once`.
- `auto_snapshot: true` commits drift automatically; secret blocks are
  logged, never retried in a loop (snapshot skips when unchanged).

## TUI

```
kage tui
```

Keys: `j/k` or arrows move, `enter` next, `esc` back, `y` confirm restore,
`q` quit. Restore always shows the engine preview first; conflicts and
checkpoints behave exactly like `kage restore`.

## Scheduled snapshots

```
mkdir -p ~/.config/systemd/user
cp systemd/kage-snapshot.{service,timer} ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now kage-snapshot.timer
```
