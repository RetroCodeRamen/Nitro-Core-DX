package main

import (
	"strings"
	"testing"
)

func TestHelpHeadingsSkipCodeFences(t *testing.T) {
	src := "# Title\n\n```\n# not a heading\n```\n\n## Real section\n"
	got := helpHeadings(src)
	if len(got) != 2 || got[0].Title != "Title" || got[1].Title != "Real section" || got[1].Level != 2 {
		t.Fatalf("headings = %+v", got)
	}
}

func TestHelpListItemsStaySeparateAfterAParagraph(t *testing.T) {
	src := "" +
		"**Part 1 — The CoreLX Language**\n" +
		"4. [Two Kinds of Numbers](#chapter-1--two-kinds-of-numbers)\n" +
		"5. [Constants](#chapter-2--constants)\n" +
		"\n" +
		"- first\n" +
		"  - nested\n"
	blocks := parseHelpBlocks(src)
	var lists []helpBlock
	for _, block := range blocks {
		if block.Kind == "li" {
			lists = append(lists, block)
		}
	}
	if len(lists) != 4 {
		t.Fatalf("list items = %d, want 4: %+v", len(lists), blocks)
	}
	if lists[0].Marker != "4." || lists[1].Marker != "5." {
		t.Fatalf("markers = %q %q", lists[0].Marker, lists[1].Marker)
	}
	if lists[0].Text != "[Two Kinds of Numbers](#chapter-1--two-kinds-of-numbers)" {
		t.Fatalf("item body = %q", lists[0].Text)
	}
	if lists[3].Level != 1 || lists[3].Text != "nested" {
		t.Fatalf("nested = %+v", lists[3])
	}
	if blocks[0].Kind != "p" {
		t.Fatalf("part title should stay a paragraph, got %+v", blocks[0])
	}
	if !headingMatchesAnchor("Two Kinds of Numbers, and Why One of Them Lies to You", "#chapter-1--two-kinds-of-numbers-and-why-one-of-them-lies-to-you") {
		t.Fatal("anchor should match the heading slug inside the manual link")
	}
}

func TestHelpSectionMarkdownStopsAtSameLevel(t *testing.T) {
	src := "# A\n\nalpha\n\n## Nested\n\nnested\n\n# B\n\nbeta\n"
	headings := helpHeadings(src)
	got := helpSectionMarkdown(src, headings, 0)
	for _, part := range []string{"alpha", "Nested", "nested"} {
		if !strings.Contains(got, part) {
			t.Fatalf("section missing %q in %q", part, got)
		}
	}
	if strings.Contains(got, "beta") {
		t.Fatalf("section ran into the next heading: %q", got)
	}
}
