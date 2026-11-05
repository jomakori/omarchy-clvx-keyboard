# Contributing

`master` is protected: **every change lands through a pull request**
(direct pushes are rejected, including for the repo owner), and PRs merge
only when the `go` and `plugin` CI jobs are green with an up-to-date branch.

Flow:

```sh
git checkout -b feat/short-name
# ... make changes ...
git commit -m "feat: short imperative description"
git push -u origin feat/short-name
gh pr create --fill
# wait for CI, then squash-merge (or enable auto-merge)
```

Commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:`,
`chore:` …) — Release Please turns them into the changelog, version bump,
tag, and GitHub Release automatically on merge.

Gates before opening a PR:

- Go: `gofmt -l .` clean, `go vet ./...`, `go test ./...` in `go/`
- Plugin: `python3 .github/scripts/validate_manifest.py`
- QML: `omarchy plugin validate .` (needs Omarchy; CI does not run this)
