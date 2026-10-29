package main

import "testing"

func TestFindEditorSpanForwardWrapsAndRespectsCase(t *testing.T) {
	text := "Alpha beta ALPHA"
	start, end, wrapped, ok := findEditorSpan(text, "alpha", 0, true, false)
	if !ok || wrapped || text[start:end] != "Alpha" {
		t.Fatalf("first match = %d:%d wrapped=%v ok=%v", start, end, wrapped, ok)
	}
	start, end, wrapped, ok = findEditorSpan(text, "alpha", end, true, false)
	if !ok || wrapped || text[start:end] != "ALPHA" {
		t.Fatalf("second match = %q wrapped=%v ok=%v", text[start:end], wrapped, ok)
	}
	start, end, wrapped, ok = findEditorSpan(text, "alpha", end, true, false)
	if !ok || !wrapped || text[start:end] != "Alpha" {
		t.Fatalf("wrapped match = %q wrapped=%v ok=%v", text[start:end], wrapped, ok)
	}
	_, _, _, ok = findEditorSpan(text, "alpha", 0, true, true)
	if ok {
		t.Fatal("case-sensitive search should miss Alpha")
	}
	start, end, _, ok = findEditorSpan(text, "ALPHA", 0, true, true)
	if !ok || text[start:end] != "ALPHA" {
		t.Fatalf("case-sensitive match = %q ok=%v", text[start:end], ok)
	}
	_, _, _, ok = findEditorSpan(text, "", 0, true, false)
	if ok {
		t.Fatal("empty query should not match")
	}
}

func TestFindEditorSpanBackwardWraps(t *testing.T) {
	text := "one two one"
	start, end, wrapped, ok := findEditorSpan(text, "one", 4, false, true)
	if !ok || wrapped || text[start:end] != "one" || start != 0 {
		t.Fatalf("backward = %d:%d wrapped=%v ok=%v", start, end, wrapped, ok)
	}
	start, _, wrapped, ok = findEditorSpan(text, "one", 0, false, true)
	if !ok || !wrapped || start != 8 {
		t.Fatalf("backward wrap = %d wrapped=%v ok=%v", start, wrapped, ok)
	}
}

func TestReplaceAllDoesNotRescanReplacement(t *testing.T) {
	next, count := replaceAllEditorMatches("aaa", "aa", "aa", true)
	if next != "aaa" || count != 1 {
		t.Fatalf("replace = %q count=%d", next, count)
	}
	next, count = replaceAllEditorMatches("foo foo", "foo", "bar", false)
	if next != "bar bar" || count != 2 {
		t.Fatalf("replace = %q count=%d", next, count)
	}
	next, count = replaceAllEditorMatches("Über über", "über", "ue", false)
	if next != "ue ue" || count != 2 {
		t.Fatalf("fold replace = %q count=%d", next, count)
	}
}

func TestEditorGotoRow(t *testing.T) {
	if _, _, ok := editorGotoRow(10, 0); ok {
		t.Fatal("line 0 should be rejected")
	}
	row, clamped, ok := editorGotoRow(10, 4)
	if !ok || clamped || row != 3 {
		t.Fatalf("line 4 = row %d clamped=%v ok=%v", row, clamped, ok)
	}
	row, clamped, ok = editorGotoRow(10, 40)
	if !ok || !clamped || row != 9 {
		t.Fatalf("past end = row %d clamped=%v ok=%v", row, clamped, ok)
	}
}
