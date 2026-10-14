# Usage

```
kage init
kage track add ~/.config/hypr
kage snapshot -m "before hyprland update"
kage status                  # exit 2 on drift
kage log
kage diff                    # working vs latest; exit 2 on change
kage diff --from 1 --to 2
kage diff --last 2           # history of last 2
kage diff --file .zshrc --summary-only
kage restore --snapshot 12 --file ~/.zshrc --dry-run
kage restore --snapshot 12 --file ~/.zshrc --yes
kage audit                   # exit 0/2/3
```

Exit codes: 0 ok · 1 error · 2 changes/drift/review · 3 conflict/secret-block.
`--json` on snapshot/status/log/audit for automation.
