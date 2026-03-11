package bluesky

import (
	"net/url"
	"regexp"
	"strings"

	bsky "github.com/bluesky-social/indigo/api/bsky"
)

// SegmentType identifies the kind of rich text segment.
type SegmentType int

const (
	SegmentText    SegmentType = iota // Plain text
	SegmentMention                    // @mention
	SegmentLink                       // URL link
	SegmentTag                        // #hashtag
)

// Segment represents a styled portion of rich text.
type Segment struct {
	Text string
	Type SegmentType
	// Value holds the associated data: DID for mentions, URI for links, tag for hashtags.
	Value string
}

// ParseFacets converts raw facets from a post into styled text segments.
// IMPORTANT: facet byte indices are UTF-8 byte offsets, not rune indices.
func ParseFacets(text string, facets []*bsky.RichtextFacet) []Segment {
	textBytes := []byte(text)
	textLen := int64(len(textBytes))

	if len(facets) == 0 {
		return []Segment{{Text: text, Type: SegmentText}}
	}

	// Sort facets by byte start (they should already be sorted, but be safe)
	sorted := make([]*bsky.RichtextFacet, len(facets))
	copy(sorted, facets)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].Index.ByteStart < sorted[j-1].Index.ByteStart; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	var segments []Segment
	var cursor int64

	for _, facet := range sorted {
		if facet.Index == nil {
			continue
		}
		start := facet.Index.ByteStart
		end := facet.Index.ByteEnd

		// Clamp to text bounds
		if start < 0 {
			start = 0
		}
		if end > textLen {
			end = textLen
		}
		if start >= end || start > textLen {
			continue
		}

		// Add plain text before this facet
		if cursor < start {
			segments = append(segments, Segment{
				Text: string(textBytes[cursor:start]),
				Type: SegmentText,
			})
		}

		facetText := string(textBytes[start:end])
		seg := Segment{Text: facetText, Type: SegmentText}

		// Determine segment type from first feature
		if len(facet.Features) > 0 {
			feat := facet.Features[0]
			if feat.RichtextFacet_Mention != nil {
				seg.Type = SegmentMention
				seg.Value = feat.RichtextFacet_Mention.Did
			} else if feat.RichtextFacet_Link != nil {
				seg.Type = SegmentLink
				seg.Value = feat.RichtextFacet_Link.Uri
			} else if feat.RichtextFacet_Tag != nil {
				seg.Type = SegmentTag
				seg.Value = feat.RichtextFacet_Tag.Tag
			}
		}

		segments = append(segments, seg)
		cursor = end
	}

	// Add trailing plain text
	if cursor < textLen {
		segments = append(segments, Segment{
			Text: string(textBytes[cursor:]),
			Type: SegmentText,
		})
	}

	return segments
}

// Regular expressions for detecting facets in text.
var (
	// Matches @handle patterns (e.g. @user.bsky.social)
	mentionRegex = regexp.MustCompile(`(?:^|\s)(@([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)`)
	// Matches URLs (http:// or https://)
	urlRegex = regexp.MustCompile(`https?://[^\s\)>\]]+`)
	// Matches #hashtag patterns
	tagRegex = regexp.MustCompile(`(?:^|\s)(#[^\s#\p{P}][^\s#]*)`)
)

// DetectFacets auto-detects @mentions, URLs, and #hashtags in text.
// Returns facets with byte-correct indices.
// Note: mention DIDs will be empty strings — the caller must resolve handles to DIDs.
func DetectFacets(text string) []*bsky.RichtextFacet {
	textBytes := []byte(text)
	var facets []*bsky.RichtextFacet

	// Detect mentions
	for _, match := range mentionRegex.FindAllIndex(textBytes, -1) {
		// Find the actual '@' start position (skip leading whitespace)
		start := match[0]
		for start < match[1] && textBytes[start] != '@' {
			start++
		}
		mention := string(textBytes[start:match[1]])
		handle := strings.TrimPrefix(mention, "@")

		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(start),
				ByteEnd:   int64(match[1]),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Mention: &bsky.RichtextFacet_Mention{
					LexiconTypeID: "app.bsky.richtext.facet#mention",
					Did:           handle, // Caller should resolve to actual DID
				},
			}},
		})
	}

	// Detect URLs
	for _, match := range urlRegex.FindAllIndex(textBytes, -1) {
		rawURL := string(textBytes[match[0]:match[1]])
		// Clean trailing punctuation that's likely not part of the URL
		rawURL = strings.TrimRight(rawURL, ".,;:!?")
		cleanEnd := match[0] + len([]byte(rawURL))

		// Validate URL
		if _, err := url.Parse(rawURL); err != nil {
			continue
		}

		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(match[0]),
				ByteEnd:   int64(cleanEnd),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Link: &bsky.RichtextFacet_Link{
					LexiconTypeID: "app.bsky.richtext.facet#link",
					Uri:           rawURL,
				},
			}},
		})
	}

	// Detect hashtags
	for _, match := range tagRegex.FindAllIndex(textBytes, -1) {
		// Find the actual '#' start position (skip leading whitespace)
		start := match[0]
		for start < match[1] && textBytes[start] != '#' {
			start++
		}
		tag := string(textBytes[start:match[1]])
		tagValue := strings.TrimPrefix(tag, "#")

		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(start),
				ByteEnd:   int64(match[1]),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{{
				RichtextFacet_Tag: &bsky.RichtextFacet_Tag{
					LexiconTypeID: "app.bsky.richtext.facet#tag",
					Tag:           tagValue,
				},
			}},
		})
	}

	return facets
}
