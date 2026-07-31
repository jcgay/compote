package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pomTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<project>
  <parent>
    <groupId>com.vidal</groupId>
    <artifactId>parent</artifactId>
    <version>%PARENT%</version>
  </parent>
  <groupId>com.vidal</groupId>
  <artifactId>foo</artifactId>
  <version>%PROJECT%</version>
  <dependencies>
    <dependency>
      <groupId>com.vidal</groupId>
      <artifactId>bar</artifactId>
      <version>1.4.0-SNAPSHOT</version>
    </dependency>%EXTRA%
  </dependencies>
</project>
`

func pom(project, parent, extra string) string {
	r := strings.NewReplacer("%PROJECT%", project, "%PARENT%", parent, "%EXTRA%", extra)
	return r.Replace(pomTemplate)
}

// The search&replace trap: theirs' project version equals a dependency
// version. Only the project/parent versions may change, byte-for-byte.
func TestAlignIsSurgical(t *testing.T) {
	ours := []byte(pom("1.5.0-SNAPSHOT", "1.5.0-SNAPSHOT", ""))
	theirs := []byte(pom("1.4.0-SNAPSHOT", "1.4.0-SNAPSHOT", ""))
	got := string(align(ours, theirs))
	want := pom("1.5.0-SNAPSHOT", "1.5.0-SNAPSHOT", "")
	if got != want {
		t.Errorf("align mismatch:\n%s", got)
	}
	if strings.Count(got, "1.4.0-SNAPSHOT") != 1 {
		t.Errorf("dependency version corrupted:\n%s", got)
	}
}

func TestMergeDivergentVersionsKeepsOursAndTheirsChanges(t *testing.T) {
	extra := "\n    <dependency>\n      <groupId>com.vidal</groupId>\n      <artifactId>baz</artifactId>\n      <version>2.0.0</version>\n    </dependency>"
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	base := write("base.xml", pom("1.4.0-SNAPSHOT", "1.4.0-SNAPSHOT", ""))
	oursPath := write("ours.xml", pom("1.5.0-SNAPSHOT", "1.5.0-SNAPSHOT", ""))
	theirs := write("theirs.xml", pom("1.4.1-SNAPSHOT", "1.4.1-SNAPSHOT", extra))

	if code := merge(base, oursPath, theirs); code != 0 {
		t.Fatalf("merge exited %d", code)
	}
	merged, err := os.ReadFile(oursPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(merged)
	if strings.Count(got, "1.5.0-SNAPSHOT") != 2 {
		t.Errorf("expected ours' project+parent versions to win:\n%s", got)
	}
	if !strings.Contains(got, "<artifactId>baz</artifactId>") {
		t.Errorf("theirs' new dependency lost:\n%s", got)
	}
	if strings.Count(got, "1.4.0-SNAPSHOT") != 1 {
		t.Errorf("dependency bar corrupted:\n%s", got)
	}
	if strings.Contains(got, "<<<<<<<") {
		t.Errorf("unexpected conflict markers:\n%s", got)
	}
}

func TestMergeRealConflictOutsideVersionsStillConflicts(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	base := write("base.xml", pom("1.4.0-SNAPSHOT", "1.4.0-SNAPSHOT", ""))
	oursPath := write("ours.xml", strings.Replace(pom("1.5.0-SNAPSHOT", "1.5.0-SNAPSHOT", ""), "<artifactId>bar</artifactId>", "<artifactId>bar-ours</artifactId>", 1))
	theirs := write("theirs.xml", strings.Replace(pom("1.4.1-SNAPSHOT", "1.4.1-SNAPSHOT", ""), "<artifactId>bar</artifactId>", "<artifactId>bar-theirs</artifactId>", 1))

	if code := merge(base, oursPath, theirs); code == 0 {
		t.Fatal("expected a conflict exit code")
	}
	merged, _ := os.ReadFile(oursPath)
	if !strings.Contains(string(merged), "<<<<<<<") {
		t.Errorf("expected conflict markers:\n%s", merged)
	}
}
