package httpapi

import "testing"

func TestExtractContactMemoryDraftsSkipsRecallQuestions(t *testing.T) {
	drafts := extractContactMemoryDrafts("Tadi aku suka apa?", 5, 800)
	if len(drafts) != 0 {
		t.Fatalf("expected no memory drafts for recall question, got %#v", drafts)
	}
}

func TestExtractContactMemoryDraftsKeepsExplicitPreference(t *testing.T) {
	drafts := extractContactMemoryDrafts("Aku suka coklat.", 5, 800)
	if len(drafts) != 1 {
		t.Fatalf("expected one memory draft, got %#v", drafts)
	}
	if drafts[0].MemoryType != "preference" || drafts[0].NormalizedValue != "suka coklat" {
		t.Fatalf("unexpected draft: %#v", drafts[0])
	}
}

func TestExtractContactMemoryDraftsTrimsTrailingQuestion(t *testing.T) {
	drafts := extractContactMemoryDrafts("Aku suka coklat, ada rekomendasi?", 5, 800)
	if len(drafts) != 1 {
		t.Fatalf("expected one memory draft, got %#v", drafts)
	}
	if drafts[0].NormalizedValue != "suka coklat" {
		t.Fatalf("expected trailing question to be trimmed, got %#v", drafts[0])
	}
}

func TestExtractContactMemoryDraftsSkipsBudgetQuestion(t *testing.T) {
	drafts := extractContactMemoryDrafts("Budget aku berapa ya?", 5, 800)
	if len(drafts) != 0 {
		t.Fatalf("expected no memory drafts for budget question, got %#v", drafts)
	}
}
