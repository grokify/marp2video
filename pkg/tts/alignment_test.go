package tts

import (
	"testing"
	"time"

	omnistt "github.com/grokify/videoascode/pkg/omnivoice/stt"
)

func TestSplitIntoWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "Two types of AI users",
			expected: []string{"Two", "types", "of", "AI", "users"},
		},
		{
			name:     "with punctuation",
			input:    "Hello, world! How are you?",
			expected: []string{"Hello,", "world!", "How", "are", "you?"},
		},
		{
			name:     "multiple spaces",
			input:    "Hello   world",
			expected: []string{"Hello", "world"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single word",
			input:    "AI",
			expected: []string{"AI"},
		},
		{
			name:     "with newlines",
			input:    "Line one\nLine two",
			expected: []string{"Line", "one", "Line", "two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoWords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitIntoWords(%q) = %v (len=%d), want %v (len=%d)",
					tt.input, result, len(result), tt.expected, len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitIntoWords(%q)[%d] = %q, want %q",
						tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestStripPunctuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no punctuation", "hello", "hello"},
		{"trailing comma", "hello,", "hello"},
		{"trailing period", "world.", "world"},
		{"leading quote", "\"hello", "hello"},
		{"both ends", "\"hello,\"", "hello"},
		{"all punctuation", "...", "..."},
		{"numbers", "123", "123"},
		{"mixed", "AI!", "AI"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripPunctuation(tt.input)
			if result != tt.expected {
				t.Errorf("stripPunctuation(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWordsMatch(t *testing.T) {
	tests := []struct {
		name     string
		original string
		stt      string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"case difference", "AI", "ai", true},
		{"punctuation", "hello,", "hello", true},
		{"both punctuation", "hello!", "hello?", true},
		{"different words", "hello", "world", false},
		{"Claude Code single", "Claude", "claude", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wordsMatch(tt.original, tt.stt)
			if result != tt.expected {
				t.Errorf("wordsMatch(%q, %q) = %v, want %v",
					tt.original, tt.stt, result, tt.expected)
			}
		})
	}
}

func TestAlignTranscriptionWithOriginal(t *testing.T) {
	// Create a mock STT result
	sttResult := &omnistt.TranscriptionResult{
		Text:     "two types of ai users are emerging",
		Language: "en-US",
		Duration: 3 * time.Second,
		Segments: []omnistt.Segment{
			{
				Text:      "two types of ai users are emerging",
				StartTime: 0,
				EndTime:   3 * time.Second,
				Words: []omnistt.Word{
					{Text: "two", StartTime: 0, EndTime: 200 * time.Millisecond, Confidence: 0.95},
					{Text: "types", StartTime: 250 * time.Millisecond, EndTime: 500 * time.Millisecond, Confidence: 0.98},
					{Text: "of", StartTime: 520 * time.Millisecond, EndTime: 600 * time.Millisecond, Confidence: 0.99},
					{Text: "ai", StartTime: 650 * time.Millisecond, EndTime: 900 * time.Millisecond, Confidence: 0.85},
					{Text: "users", StartTime: 950 * time.Millisecond, EndTime: 1200 * time.Millisecond, Confidence: 0.97},
					{Text: "are", StartTime: 1250 * time.Millisecond, EndTime: 1400 * time.Millisecond, Confidence: 0.99},
					{Text: "emerging", StartTime: 1450 * time.Millisecond, EndTime: 2000 * time.Millisecond, Confidence: 0.96},
				},
			},
		},
	}

	originalText := "Two types of AI users are emerging"

	result := alignTranscriptionWithOriginal(sttResult, originalText)

	// Check that the text is now properly capitalized
	if result.Text != originalText {
		t.Errorf("aligned text = %q, want %q", result.Text, originalText)
	}

	// Check that timestamps are preserved
	if len(result.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(result.Segments))
	}

	words := result.Segments[0].Words
	if len(words) != 7 {
		t.Fatalf("expected 7 words, got %d", len(words))
	}

	// Check specific word alignments
	expectedWords := []string{"Two", "types", "of", "AI", "users", "are", "emerging"}
	for i, expected := range expectedWords {
		if words[i].Text != expected {
			t.Errorf("word[%d] = %q, want %q", i, words[i].Text, expected)
		}
	}

	// Verify timestamps were preserved from STT
	if words[0].StartTime != 0 {
		t.Errorf("first word start time = %v, want 0", words[0].StartTime)
	}
	if words[3].Text != "AI" {
		t.Errorf("word[3] = %q, want \"AI\"", words[3].Text)
	}
}

func TestAlignTranscriptionWithOriginal_MismatchedCounts(t *testing.T) {
	// Create a mock STT result with fewer words
	sttResult := &omnistt.TranscriptionResult{
		Text:     "hello world",
		Language: "en-US",
		Duration: 1 * time.Second,
		Segments: []omnistt.Segment{
			{
				Text: "hello world",
				Words: []omnistt.Word{
					{Text: "hello", StartTime: 0, EndTime: 400 * time.Millisecond},
					{Text: "world", StartTime: 500 * time.Millisecond, EndTime: 900 * time.Millisecond},
				},
			},
		},
	}

	// Original has more words - alignment should fall back to STT
	originalText := "Hello beautiful world today"

	result := alignTranscriptionWithOriginal(sttResult, originalText)

	// Should return original STT result when alignment fails
	if result.Text != "hello world" {
		t.Errorf("expected fallback to STT text, got %q", result.Text)
	}
}

func TestAlignWordsWithTiming(t *testing.T) {
	tests := []struct {
		name      string
		orig      []string
		stt       []omnistt.Word
		expectOK  bool
		wantWords []string
	}{
		{
			name: "exact 1:1 match",
			orig: []string{"Hello", "World"},
			stt:  []omnistt.Word{{Text: "hello"}, {Text: "world"}},

			expectOK:  true,
			wantWords: []string{"Hello", "World"},
		},
		{
			name: "STT split hyphenated word (command-line)",
			orig: []string{"a", "command-line", "tool"},
			stt:  []omnistt.Word{{Text: "a"}, {Text: "command"}, {Text: "line"}, {Text: "tool"}},

			expectOK:  true,
			wantWords: []string{"a", "command-line", "tool"},
		},
		{
			name: "STT split brand into token+word (marp2video)",
			orig: []string{"Welcome", "to", "marp2video,"},
			stt:  []omnistt.Word{{Text: "Welcome"}, {Text: "to"}, {Text: "MARP2"}, {Text: "Video"}},

			expectOK:  true,
			wantWords: []string{"Welcome", "to", "marp2video,"},
		},
		{
			name: "STT merged two words (web site -> website)",
			orig: []string{"visit", "the", "web", "site", "now"},
			stt:  []omnistt.Word{{Text: "visit"}, {Text: "the"}, {Text: "website"}, {Text: "now"}},

			expectOK:  true,
			wantWords: []string{"visit", "the", "web site", "now"},
		},
		{
			name: "too many mismatches falls back",
			orig: []string{"completely", "different", "original", "words"},
			stt:  []omnistt.Word{{Text: "alpha"}, {Text: "beta"}},

			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := alignWordsWithTiming(tt.orig, tt.stt)
			if ok != tt.expectOK {
				t.Fatalf("alignWordsWithTiming ok = %v, want %v", ok, tt.expectOK)
			}
			if !tt.expectOK {
				return
			}
			if len(got) != len(tt.wantWords) {
				t.Fatalf("got %d words, want %d: %v", len(got), len(tt.wantWords), got)
			}
			for i, w := range tt.wantWords {
				if got[i].Text != w {
					t.Errorf("word[%d] = %q, want %q", i, got[i].Text, w)
				}
			}
		})
	}
}

// TestAlignWordsWithTiming_SpanTiming verifies that when STT splits an original
// word, the aligned word spans from the first token's start to the last token's
// end.
func TestAlignWordsWithTiming_SpanTiming(t *testing.T) {
	orig := []string{"marp2video"}
	stt := []omnistt.Word{
		{Text: "MARP2", StartTime: 100 * time.Millisecond, EndTime: 400 * time.Millisecond},
		{Text: "Video", StartTime: 400 * time.Millisecond, EndTime: 900 * time.Millisecond},
	}

	got, ok := alignWordsWithTiming(orig, stt)
	if !ok || len(got) != 1 {
		t.Fatalf("alignWordsWithTiming ok=%v words=%d, want ok=true words=1", ok, len(got))
	}
	if got[0].StartTime != 100*time.Millisecond {
		t.Errorf("start = %v, want 100ms", got[0].StartTime)
	}
	if got[0].EndTime != 900*time.Millisecond {
		t.Errorf("end = %v, want 900ms (span across both split tokens)", got[0].EndTime)
	}
}
