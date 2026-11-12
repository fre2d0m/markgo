package mdeditor

import "testing"

func TestCountTextHandlesChineseAndContractions(t *testing.T) {
	stats := CountText("你好 world don't 123")
	if stats.Words != 5 || stats.Characters != 18 {
		t.Fatalf("statistics: %#v", stats)
	}
}

func TestAnalysisCountsVisibleTextRatherThanLinkDestinations(t *testing.T) {
	analysis := Analyze("# 标题\n\n[Hello world](https://example.com) **bold**\n\n```go\nprint(1)\n```\n")
	if analysis.Stats.Words != 7 {
		t.Fatalf("words: %d", analysis.Stats.Words)
	}
}
