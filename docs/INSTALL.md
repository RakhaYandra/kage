# Install

Requires: Linux, Go 1.27+, git.

```
make build        # bin/kage
make install      # ~/.local/bin/kage
kage init         # config + repository
```

Arch rootless: `make release` builds `dist/kage-linux-amd64|arm64` + SHA256SUMS.
