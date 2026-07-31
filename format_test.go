package main

import (
	"strings"
	"testing"
)

// A pom with everything that breaks naive tooling: comments, namespaces,
// mixed tabs/spaces, whitespace-padded versions, a property holding the same
// value. align must reproduce it byte-for-byte, versions excepted.
const uglyPom = `<?xml version="1.0" encoding="UTF-8"?>
<!-- corporate header, do not remove -->
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
	<parent>
		<groupId>com.vidal</groupId>
		<artifactId>parent</artifactId>
		<version>
			1.4.0-SNAPSHOT
		</version><!-- inline comment -->
	</parent>
  <groupId>com.vidal</groupId>
  <artifactId>foo</artifactId>
  <version>  1.4.0-SNAPSHOT  </version>
  <properties>
    <bar.version>1.4.0-SNAPSHOT</bar.version>
  </properties>
</project>
`

const minimalOurs = `<project><parent><version>1.5.0-SNAPSHOT</version></parent><version>1.5.0-SNAPSHOT</version></project>`

func TestAlignPreservesFormatting(t *testing.T) {
	got := string(align([]byte(minimalOurs), []byte(uglyPom)))
	// only the parent and project versions (the first two occurrences) change;
	// padding, comments, tabs and the property stay byte-identical.
	want := strings.Replace(uglyPom, "1.4.0-SNAPSHOT", "1.5.0-SNAPSHOT", 2)
	if got != want {
		t.Errorf("formatting altered:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAlignLeavesUntouchedWhenNothingToDo(t *testing.T) {
	for name, theirs := range map[string]string{
		"same versions":  uglyPom,
		"no version":     "<project><artifactId>foo</artifactId></project>",
		"empty version":  "<project><version/></project>",
		"not xml at all": "certainly not a pom",
	} {
		ours := minimalOurs
		if name == "same versions" {
			ours = uglyPom
		}
		if got := string(align([]byte(ours), []byte(theirs))); got != theirs {
			t.Errorf("%s: input altered:\n%s", name, got)
		}
	}
}
