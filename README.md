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
brew install jcgay/jcgay/compote
```

or

```sh
go install github.com/jcgay/compote@latest
```

## Build

```sh
mise install    # installs the pinned Go
mise run check  # test + vet + gofmt
mise run build
```

## Setup

Declare the driver once, globally:

```sh
git config --global merge.compote.name "Écrase les poms proprement"
git config --global merge.compote.driver "compote %O %A %B"
```

Then bind it to `pom.xml`, either **globally for all your repos** (git reads
`~/.config/git/attributes` by default):

```sh
mkdir -p ~/.config/git
echo 'pom.xml merge=compote' >> ~/.config/git/attributes
```

or **per repo**, committed so teammates who installed compote benefit too:

```gitattributes
# .gitattributes
pom.xml merge=compote
```

A pattern without a slash matches at any depth, so one line covers every
module of a multi-module build. Both setups are safe to mix: a committed
`.gitattributes` takes precedence over the global file, and teammates
without compote silently fall back to the standard 3-way merge.

Note: git only invokes merge drivers on a real 3-way merge — fast-forward
merges bypass them (nothing to resolve anyway).

## Behavior

- Project and parent versions: **ours wins** (pomutils' `OUR` strategy) — the
  git-flow default when merging master/hotfix back into develop.
- Everything else: standard 3-way merge via `git merge-file`; genuine
  conflicts still conflict.
- Unparseable file or missing `<version>`: falls back to a plain
  `git merge-file`.
