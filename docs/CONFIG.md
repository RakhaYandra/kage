# Config (`~/.config/kage/config.yaml`, v1)

```yaml
version: 1
repository:
  path: "~/.local/share/kage/repository"
tracking:
  paths: ["~/.bashrc", "~/.config/hypr"]
  exclude: ["**/.env", "**/id_*", "**/*.pem", "**/credentials.json"]
security:
  protect_secrets: true
  redact_output: true
  block_detected_secrets: true
  allow_secrets: []   # explicit, recorded exceptions (home-relative)
snapshot:
  auto_commit: true
  skip_unchanged: true
  preserve_permissions: true
  schedule: ""         # ""|daily|hourly; informational for timer users
  keep_last: 0         # 0 = keep all; enforced by `kage prune`
watcher:
  enabled: false
  auto_snapshot: false
  quiet_period: 5m
```

- Paths must stay inside `$HOME`. Symlinks to outside scope are never snapshotted.
- `kage config validate`, `kage track add|remove|list`. Presets in `presets/`.
