# Recovery

Every restore prints a preview and makes a checkpoint commit first:

```
kage restore --snapshot 12 --file ~/.zshrc --dry-run   # preview
kage restore --snapshot 12 --file ~/.zshrc --yes       # apply
# output: restored 1 file(s) (checkpoint ab12cd3; undo: kage restore --commit <hash>)
```

- Undo: `kage restore --commit <checkpoint-hash> --file <p> --yes`.
- Conflict (uncommitted changes + old target): exit 3, nothing written.
  Resolve with `kage snapshot -m "save work"` then restore again.
- Secret block: exit 3. Remove, exclude, or record in `security.allow_secrets`.
- Broken repo: `kage init --repair`. History of restores: `~/.local/share/kage/restore.log`.
- After restore, reload the app yourself (kage never restarts apps).
