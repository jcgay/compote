// Compote — a git merge driver for pom.xml.
// It aligns /project/version and /project/parent/version of "theirs" onto
// "ours" by byte-level splice (no re-serialization), then delegates the real
// merge to git merge-file.
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
)

// span is the byte range of a version's text content inside a pom.
type span struct {
	start, end int64
	value      string
}

type versions struct {
	project *span // /project/version
	parent  *span // /project/parent/version
}

// scan streams the pom and records where the project and parent versions
// live. Dependency versions are never matched: only the exact element paths
// project/version and project/parent/version qualify.
func scan(data []byte) versions {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var stack []string
	var v versions
	for {
		tok, err := dec.Token()
		if err != nil {
			return v
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
			path := strings.Join(stack, "/")
			if path != "project/version" && path != "project/parent/version" {
				continue
			}
			start := dec.InputOffset()
			next, err := dec.Token()
			if err != nil {
				return v
			}
			switch n := next.(type) {
			case xml.CharData:
				// ponytail: assumes the version is a single CharData token
				// (no entities/CDATA inside <version>), which holds for any
				// real-world pom.
				raw := string(n)
				val := strings.TrimSpace(raw)
				if val == "" {
					continue
				}
				// splice only the non-blank bytes so surrounding padding
				// (spaces, newlines, tabs) survives byte-for-byte.
				lead := int64(strings.Index(raw, val))
				s := &span{
					start: start + lead,
					end:   start + lead + int64(len(val)),
					value: val,
				}
				if path == "project/version" {
					v.project = s
				} else {
					v.parent = s
				}
			case xml.EndElement:
				stack = stack[:len(stack)-1] // empty <version/>, leave it alone
			case xml.StartElement:
				stack = append(stack, n.Name.Local)
			}
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
}

// align rewrites theirs so its project/parent versions match ours, touching
// only the version bytes. Returns theirs unchanged when nothing differs.
func align(ours, theirs []byte) []byte {
	ov, tv := scan(ours), scan(theirs)
	type edit struct {
		sp  *span
		val string
	}
	var edits []edit
	for _, p := range [][2]*span{{ov.project, tv.project}, {ov.parent, tv.parent}} {
		if p[0] != nil && p[1] != nil && p[0].value != p[1].value {
			edits = append(edits, edit{p[1], p[0].value})
		}
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].sp.start > edits[j].sp.start })
	out := theirs
	for _, e := range edits {
		out = slices.Concat(out[:e.sp.start], []byte(e.val), out[e.sp.end:])
	}
	return out
}

// merge runs the driver: %O %A %B. The merged result is written into ours
// (%A) by git merge-file, per the merge-driver contract.
func merge(base, ours, theirs string) int {
	oursData, err1 := os.ReadFile(ours)
	theirsData, err2 := os.ReadFile(theirs)
	if err1 == nil && err2 == nil {
		if aligned := align(oursData, theirsData); !bytes.Equal(aligned, theirsData) {
			// theirs is a temp file created by git for this merge: safe to overwrite.
			if err := os.WriteFile(theirs, aligned, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "compote:", err)
			}
		}
	}
	cmd := exec.Command("git", "merge-file", "-L", "ours", "-L", "base", "-L", "theirs", ours, base, theirs)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() > 0 {
			return 1 // conflicts remain outside the versions
		}
		fmt.Fprintln(os.Stderr, "compote:", err)
		return 2
	}
	return 0
}

var version = "dev" // set by goreleaser via -ldflags

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: compote <base> <ours> <theirs> (git merge driver: %O %A %B)")
		os.Exit(2)
	}
	os.Exit(merge(os.Args[1], os.Args[2], os.Args[3]))
}
