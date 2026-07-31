# Compote 🍎

*Turns conflicting poms into something smooth.*

A git merge driver for `pom.xml`. It resolves the classic git-flow conflict on
`<version>` (project and parent) by aligning *theirs* onto *ours*, then lets
`git merge-file` merge the rest normally.

Unlike search & replace, it locates `/project/version` and
`/project/parent/version` structurally (streaming XML parser), so dependency
versions that happen to share the same value are never touched. Unlike DOM
rewriting, it splices only the version bytes: the rest of the file stays
byte-identical, so the follow-up merge sees a minimal diff.

## Install

```sh
go install github.com/jcgay/compote@latest
```

## Setup

```ini
# ~/.gitconfig or .git/config
[merge "compote"]
    name = Écrase les poms proprement
    driver = compote %O %A %B
```

```gitattributes
# .gitattributes
pom.xml merge=compote
**/pom.xml merge=compote
```

Note: git only invokes merge drivers on a real 3-way merge — fast-forward
merges bypass them (nothing to resolve anyway).

## Behavior

- Project and parent versions: **ours wins** (pomutils' `OUR` strategy) — the
  git-flow default when merging master/hotfix back into develop.
- Everything else: standard 3-way merge via `git merge-file`; genuine
  conflicts still conflict.
- Unparseable file or missing `<version>`: falls back to a plain
  `git merge-file`.
