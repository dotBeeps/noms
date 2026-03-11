package bluesky

import (
	"testing"

	bsky "github.com/bluesky-social/indigo/api/bsky"
)

func TestParseFacetsNoFacets(t *testing.T) {
	segments := ParseFacets("Hello world", nil)
	if len(segments) != 1 {
		t.Fatalf("len(segments) = %d, want 1", len(segments))
	}
	if segments[0].Text != "Hello world" {
		t.Errorf("text = %q", segments[0].Text)
	}
	if segments[0].Type != SegmentText {
		t.Errorf("type = %d, want SegmentText", segments[0].Type)
	}
}

func TestParseFacetsMention(t *testing.T) {
	text := "Hello @alice.bsky.social!"
	textBytes := []byte(text)

	// @alice.bsky.social starts at byte 6, ends at byte 24
	atStart := indexOf(textBytes, '@')
	atEnd := atStart + len("@alice.bsky.social")

	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(atStart),
				ByteEnd:   int64(atEnd),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Mention: &bsky.RichtextFacet_Mention{
					LexiconTypeID: "app.bsky.richtext.facet#mention",
					Did:           "did:plc:alice",
				},
			}},
		},
	}

	segments := ParseFacets(text, facets)
	if len(segments) != 3 {
		t.Fatalf("len(segments) = %d, want 3", len(segments))
	}

	if segments[0].Text != "Hello " {
		t.Errorf("seg[0] = %q, want 'Hello '", segments[0].Text)
	}
	if segments[1].Type != SegmentMention {
		t.Errorf("seg[1] type = %d, want SegmentMention", segments[1].Type)
	}
	if segments[1].Value != "did:plc:alice" {
		t.Errorf("seg[1] value = %q, want did:plc:alice", segments[1].Value)
	}
	if segments[1].Text != "@alice.bsky.social" {
		t.Errorf("seg[1] text = %q", segments[1].Text)
	}
	if segments[2].Text != "!" {
		t.Errorf("seg[2] = %q, want '!'", segments[2].Text)
	}
}

func TestParseFacetsLink(t *testing.T) {
	text := "Check https://example.com out"
	textBytes := []byte(text)
	urlStart := indexOf(textBytes, 'h')
	urlEnd := urlStart + len("https://example.com")

	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(urlStart),
				ByteEnd:   int64(urlEnd),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Link: &bsky.RichtextFacet_Link{
					LexiconTypeID: "app.bsky.richtext.facet#link",
					Uri:           "https://example.com",
				},
			}},
		},
	}

	segments := ParseFacets(text, facets)
	if len(segments) != 3 {
		t.Fatalf("len(segments) = %d, want 3", len(segments))
	}
	if segments[1].Type != SegmentLink {
		t.Errorf("seg[1] type = %d, want SegmentLink", segments[1].Type)
	}
	if segments[1].Value != "https://example.com" {
		t.Errorf("seg[1] value = %q", segments[1].Value)
	}
}

func TestParseFacetsTag(t *testing.T) {
	text := "Hello #test world"
	textBytes := []byte(text)
	tagStart := indexOf(textBytes, '#')
	tagEnd := tagStart + len("#test")

	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(tagStart),
				ByteEnd:   int64(tagEnd),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Tag: &bsky.RichtextFacet_Tag{
					LexiconTypeID: "app.bsky.richtext.facet#tag",
					Tag:           "test",
				},
			}},
		},
	}

	segments := ParseFacets(text, facets)
	if len(segments) != 3 {
		t.Fatalf("len(segments) = %d, want 3", len(segments))
	}
	if segments[1].Type != SegmentTag {
		t.Errorf("seg[1] type = %d, want SegmentTag", segments[1].Type)
	}
	if segments[1].Value != "test" {
		t.Errorf("seg[1] value = %q", segments[1].Value)
	}
}

func TestParseFacetsEmojiByteOffsets(t *testing.T) {
	// CRITICAL: Emoji text to verify byte offset correctness.
	// 👋 is 4 bytes in UTF-8.
	text := "Hello 👋 @user.bsky.social check"
	textBytes := []byte(text)

	// Find the byte position of '@'
	atIdx := -1
	for i, b := range textBytes {
		if b == '@' {
			atIdx = i
			break
		}
	}
	if atIdx < 0 {
		t.Fatal("could not find '@' in text bytes")
	}

	mentionEnd := atIdx + len("@user.bsky.social")

	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(atIdx),
				ByteEnd:   int64(mentionEnd),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Mention: &bsky.RichtextFacet_Mention{
					LexiconTypeID: "app.bsky.richtext.facet#mention",
					Did:           "did:plc:user",
				},
			}},
		},
	}

	segments := ParseFacets(text, facets)
	if len(segments) != 3 {
		t.Fatalf("len(segments) = %d, want 3", len(segments))
	}

	// Verify the emoji is in the first segment
	if segments[0].Text != "Hello 👋 " {
		t.Errorf("seg[0] = %q, want 'Hello 👋 '", segments[0].Text)
	}
	if segments[1].Text != "@user.bsky.social" {
		t.Errorf("seg[1] = %q, want '@user.bsky.social'", segments[1].Text)
	}
	if segments[1].Type != SegmentMention {
		t.Errorf("seg[1] type = %d, want SegmentMention", segments[1].Type)
	}
	if segments[2].Text != " check" {
		t.Errorf("seg[2] = %q, want ' check'", segments[2].Text)
	}
}

func TestParseFacetsFullEmojiText(t *testing.T) {
	// Full test from task: "Hello 👋 @user.bsky.social check https://example.com #test"
	text := "Hello 👋 @user.bsky.social check https://example.com #test"
	textBytes := []byte(text)

	// Build facets with correct byte offsets
	atIdx := byteIndexOf(textBytes, '@')
	mentionEnd := atIdx + len("@user.bsky.social")

	httpIdx := byteIndexOf(textBytes, 'h')
	// Find the 'h' that starts "https://", not "Hello"
	for i := httpIdx; i < len(textBytes); i++ {
		if textBytes[i] == 'h' && i+5 < len(textBytes) && string(textBytes[i:i+5]) == "https" {
			httpIdx = i
			break
		}
	}
	urlEnd := httpIdx + len("https://example.com")

	hashIdx := byteIndexOf(textBytes, '#')
	tagEnd := hashIdx + len("#test")

	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{ByteStart: int64(atIdx), ByteEnd: int64(mentionEnd)},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Mention: &bsky.RichtextFacet_Mention{Did: "did:plc:user"},
			}},
		},
		{
			Index: &bsky.RichtextFacet_ByteSlice{ByteStart: int64(httpIdx), ByteEnd: int64(urlEnd)},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Link: &bsky.RichtextFacet_Link{Uri: "https://example.com"},
			}},
		},
		{
			Index: &bsky.RichtextFacet_ByteSlice{ByteStart: int64(hashIdx), ByteEnd: int64(tagEnd)},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Tag: &bsky.RichtextFacet_Tag{Tag: "test"},
			}},
		},
	}

	segments := ParseFacets(text, facets)
	if len(segments) != 6 {
		for i, s := range segments {
			t.Logf("seg[%d] type=%d text=%q value=%q", i, s.Type, s.Text, s.Value)
		}
		t.Fatalf("len(segments) = %d, want 6", len(segments))
	}

	// Reconstruct text from segments to verify no bytes lost
	var reconstructed string
	for _, s := range segments {
		reconstructed += s.Text
	}
	if reconstructed != text {
		t.Errorf("reconstructed = %q, want %q", reconstructed, text)
	}
}

func TestDetectFacetsMentions(t *testing.T) {
	text := "Hello @alice.bsky.social and @bob.test"
	facets := DetectFacets(text)

	mentionCount := 0
	for _, f := range facets {
		if len(f.Features) > 0 && f.Features[0].RichtextFacet_Mention != nil {
			mentionCount++
			// Verify byte offsets are correct
			textBytes := []byte(text)
			extracted := string(textBytes[f.Index.ByteStart:f.Index.ByteEnd])
			if extracted[0] != '@' {
				t.Errorf("mention should start with @, got %q", extracted)
			}
		}
	}
	if mentionCount != 2 {
		t.Errorf("found %d mentions, want 2", mentionCount)
	}
}

func TestDetectFacetsURLs(t *testing.T) {
	text := "Visit https://example.com and http://test.org/path"
	facets := DetectFacets(text)

	linkCount := 0
	for _, f := range facets {
		if len(f.Features) > 0 && f.Features[0].RichtextFacet_Link != nil {
			linkCount++
			textBytes := []byte(text)
			extracted := string(textBytes[f.Index.ByteStart:f.Index.ByteEnd])
			if extracted != f.Features[0].RichtextFacet_Link.Uri {
				t.Errorf("extracted URL %q != feature URI %q", extracted, f.Features[0].RichtextFacet_Link.Uri)
			}
		}
	}
	if linkCount != 2 {
		t.Errorf("found %d links, want 2", linkCount)
	}
}

func TestDetectFacetsHashtags(t *testing.T) {
	text := "Check out #golang and #atproto"
	facets := DetectFacets(text)

	tagCount := 0
	for _, f := range facets {
		if len(f.Features) > 0 && f.Features[0].RichtextFacet_Tag != nil {
			tagCount++
			textBytes := []byte(text)
			extracted := string(textBytes[f.Index.ByteStart:f.Index.ByteEnd])
			if extracted[0] != '#' {
				t.Errorf("tag should start with #, got %q", extracted)
			}
		}
	}
	if tagCount != 2 {
		t.Errorf("found %d tags, want 2", tagCount)
	}
}

func TestDetectFacetsEmojiByteOffsets(t *testing.T) {
	// Verify byte offsets are correct with multi-byte emoji characters.
	text := "Hello 👋 @user.bsky.social check https://example.com #test"
	facets := DetectFacets(text)
	textBytes := []byte(text)

	for _, f := range facets {
		extracted := string(textBytes[f.Index.ByteStart:f.Index.ByteEnd])

		if f.Features[0].RichtextFacet_Mention != nil {
			if extracted != "@user.bsky.social" {
				t.Errorf("mention extracted = %q, want @user.bsky.social", extracted)
			}
		}
		if f.Features[0].RichtextFacet_Link != nil {
			if extracted != "https://example.com" {
				t.Errorf("link extracted = %q, want https://example.com", extracted)
			}
		}
		if f.Features[0].RichtextFacet_Tag != nil {
			if extracted != "#test" {
				t.Errorf("tag extracted = %q, want #test", extracted)
			}
		}
	}
}

func TestDetectFacetsEmptyText(t *testing.T) {
	facets := DetectFacets("")
	if len(facets) != 0 {
		t.Errorf("expected 0 facets for empty text, got %d", len(facets))
	}
}

func TestParseFacetsOutOfBounds(t *testing.T) {
	text := "short"
	facets := []*bsky.RichtextFacet{
		{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: 0,
				ByteEnd:   1000, // Way past end
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Link: &bsky.RichtextFacet_Link{Uri: "https://x.com"},
			}},
		},
	}

	// Should not panic
	segments := ParseFacets(text, facets)
	if len(segments) == 0 {
		t.Error("expected at least one segment")
	}
}

// Helper to find first byte index of a character.
func indexOf(b []byte, ch byte) int {
	for i, c := range b {
		if c == ch {
			return i
		}
	}
	return -1
}

// byteIndexOf finds the first occurrence of a byte in the slice.
func byteIndexOf(b []byte, ch byte) int {
	for i, c := range b {
		if c == ch {
			return i
		}
	}
	return -1
}
