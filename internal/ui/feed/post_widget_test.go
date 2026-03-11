package feed

import (
	"strings"
	"testing"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
)

func TestRenderPostWithImageEmbed(t *testing.T) {
	t.Parallel()
	post := createTestPost("Look at this!", "alice.bsky.social", "Alice", "at://uri1", "cid1")
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedImages_View: &bsky.EmbedImages_View{
			Images: []*bsky.EmbedImages_ViewImage{
				{Alt: "a cute cat", Fullsize: "https://example.com/cat.jpg", Thumb: "https://example.com/cat_thumb.jpg"},
			},
		},
	}

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "🖼") {
		t.Error("Expected image indicator 🖼 in rendered post")
	}
	if !strings.Contains(rendered, "a cute cat") {
		t.Error("Expected alt text 'a cute cat' in rendered post")
	}
}

func TestRenderPostWithMultipleImages(t *testing.T) {
	t.Parallel()
	post := createTestPost("Photos!", "bob.bsky.social", "Bob", "at://uri2", "cid2")
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedImages_View: &bsky.EmbedImages_View{
			Images: []*bsky.EmbedImages_ViewImage{
				{Alt: "photo 1", Fullsize: "https://example.com/1.jpg", Thumb: "https://example.com/1t.jpg"},
				{Alt: "photo 2", Fullsize: "https://example.com/2.jpg", Thumb: "https://example.com/2t.jpg"},
				{Alt: "photo 3", Fullsize: "https://example.com/3.jpg", Thumb: "https://example.com/3t.jpg"},
			},
		},
	}

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "🖼") {
		t.Error("Expected image indicator 🖼 in rendered post")
	}
	if !strings.Contains(rendered, "3 images") {
		t.Error("Expected '3 images' in rendered post for multiple images")
	}
}

func TestRenderPostWithExternalLinkEmbed(t *testing.T) {
	t.Parallel()
	post := createTestPost("Check this out", "carol.bsky.social", "Carol", "at://uri3", "cid3")
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedExternal_View: &bsky.EmbedExternal_View{
			External: &bsky.EmbedExternal_ViewExternal{
				Title:       "Example Article",
				Description: "An interesting article about testing",
				Uri:         "https://example.com/article",
			},
		},
	}

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "🔗") {
		t.Error("Expected link indicator 🔗 in rendered post")
	}
	if !strings.Contains(rendered, "Example Article") {
		t.Error("Expected link title in rendered post")
	}
	if !strings.Contains(rendered, "An interesting article about testing") {
		t.Error("Expected link description in rendered post")
	}
}

func TestRenderPostWithQuoteEmbed(t *testing.T) {
	t.Parallel()
	post := createTestPost("Great take", "dave.bsky.social", "Dave", "at://uri4", "cid4")
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedRecord_View: &bsky.EmbedRecord_View{
			Record: &bsky.EmbedRecord_View_Record{
				EmbedRecord_ViewRecord: &bsky.EmbedRecord_ViewRecord{
					Author: &bsky.ActorDefs_ProfileViewBasic{
						Did:    "did:plc:quoted",
						Handle: "quoted.bsky.social",
					},
					Uri: "at://quoted/post",
					Value: &util.LexiconTypeDecoder{
						Val: &bsky.FeedPost{
							Text:      "This is the original post",
							CreatedAt: "2024-01-01T00:00:00Z",
						},
					},
				},
			},
		},
	}

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "❝") {
		t.Error("Expected quote indicator ❝ in rendered post")
	}
	if !strings.Contains(rendered, "@quoted.bsky.social") {
		t.Error("Expected quoted author handle in rendered post")
	}
	if !strings.Contains(rendered, "This is the original post") {
		t.Error("Expected quoted text in rendered post")
	}
}

func TestRenderPostWithVideoEmbed(t *testing.T) {
	t.Parallel()
	post := createTestPost("Watch this!", "eve.bsky.social", "Eve", "at://uri5", "cid5")
	altText := "A funny video"
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedVideo_View: &bsky.EmbedVideo_View{
			Alt:      &altText,
			Cid:      "videocid",
			Playlist: "https://example.com/video.m3u8",
		},
	}

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "🎥") {
		t.Error("Expected video indicator 🎥 in rendered post")
	}
	if !strings.Contains(rendered, "A funny video") {
		t.Error("Expected video alt text in rendered post")
	}
}

func TestRenderPostWithRecordMediaEmbed(t *testing.T) {
	t.Parallel()
	post := createTestPost("Quote with pics", "frank.bsky.social", "Frank", "at://uri6", "cid6")
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedRecordWithMedia_View: &bsky.EmbedRecordWithMedia_View{
			Media: &bsky.EmbedRecordWithMedia_View_Media{
				EmbedImages_View: &bsky.EmbedImages_View{
					Images: []*bsky.EmbedImages_ViewImage{
						{Alt: "media image", Fullsize: "https://example.com/img.jpg", Thumb: "https://example.com/thumb.jpg"},
						{Alt: "another", Fullsize: "https://example.com/img2.jpg", Thumb: "https://example.com/thumb2.jpg"},
					},
				},
			},
			Record: &bsky.EmbedRecord_View{
				Record: &bsky.EmbedRecord_View_Record{
					EmbedRecord_ViewRecord: &bsky.EmbedRecord_ViewRecord{
						Author: &bsky.ActorDefs_ProfileViewBasic{
							Did:    "did:plc:quoteduser",
							Handle: "quoteduser.bsky.social",
						},
						Uri: "at://quoted/post2",
						Value: &util.LexiconTypeDecoder{
							Val: &bsky.FeedPost{
								Text:      "The quoted text content",
								CreatedAt: "2024-01-01T00:00:00Z",
							},
						},
					},
				},
			},
		},
	}

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "🖼") {
		t.Error("Expected image indicator 🖼 in record with media embed")
	}
	if !strings.Contains(rendered, "2 images") {
		t.Error("Expected '2 images' in record with media embed")
	}
	if !strings.Contains(rendered, "❝") {
		t.Error("Expected quote indicator ❝ in record with media embed")
	}
	if !strings.Contains(rendered, "@quoteduser.bsky.social") {
		t.Error("Expected quoted author in record with media embed")
	}
}

func TestRenderPostWithNoEmbed(t *testing.T) {
	t.Parallel()
	post := createTestPost("Just text", "grace.bsky.social", "Grace", "at://uri7", "cid7")
	// Embed is nil by default from createTestPost

	rendered := stripAnsi(RenderPost(post, 80, false))

	if !strings.Contains(rendered, "Just text") {
		t.Error("Expected post text in rendered output")
	}
	for _, indicator := range []string{"🖼", "🔗", "❝", "🎥"} {
		if strings.Contains(rendered, indicator) {
			t.Errorf("Expected no embed indicator %s when embed is nil", indicator)
		}
	}
}

func TestRenderPostWithNilEmbedFields(t *testing.T) {
	t.Parallel()

	t.Run("nil external field", func(t *testing.T) {
		t.Parallel()
		post := createTestPost("Test", "test.bsky.social", "Test", "at://nil1", "cid_nil1")
		post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedExternal_View: &bsky.EmbedExternal_View{
				External: nil,
			},
		}
		rendered := stripAnsi(RenderPost(post, 80, false))
		if strings.Contains(rendered, "🔗") {
			t.Error("Expected no link indicator when External is nil")
		}
	})

	t.Run("nil record in quote embed", func(t *testing.T) {
		t.Parallel()
		post := createTestPost("Test", "test.bsky.social", "Test", "at://nil2", "cid_nil2")
		post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedRecord_View: &bsky.EmbedRecord_View{
				Record: nil,
			},
		}
		rendered := stripAnsi(RenderPost(post, 80, false))
		if strings.Contains(rendered, "❝") {
			t.Error("Expected no quote indicator when Record is nil")
		}
	})

	t.Run("nil ViewRecord in quote embed", func(t *testing.T) {
		t.Parallel()
		post := createTestPost("Test", "test.bsky.social", "Test", "at://nil3", "cid_nil3")
		post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedRecord_View: &bsky.EmbedRecord_View{
				Record: &bsky.EmbedRecord_View_Record{
					EmbedRecord_ViewRecord: nil,
				},
			},
		}
		rendered := stripAnsi(RenderPost(post, 80, false))
		if strings.Contains(rendered, "❝") {
			t.Error("Expected no quote indicator when ViewRecord is nil")
		}
	})

	t.Run("video with nil alt", func(t *testing.T) {
		t.Parallel()
		post := createTestPost("Test", "test.bsky.social", "Test", "at://nil4", "cid_nil4")
		post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
			EmbedVideo_View: &bsky.EmbedVideo_View{
				Alt:      nil,
				Cid:      "videocid",
				Playlist: "https://example.com/video.m3u8",
			},
		}
		rendered := stripAnsi(RenderPost(post, 80, false))
		if !strings.Contains(rendered, "🎥") {
			t.Error("Expected video indicator 🎥 even with nil alt")
		}
		if !strings.Contains(rendered, "video") {
			t.Error("Expected 'video' text even with nil alt")
		}
	})
}

func TestTruncateStr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"truncated with ellipsis", "hello world", 5, "hello…"},
		{"multi-byte characters", "こんにちは世界", 3, "こんに…"},
		{"emoji truncation", "🎉🎊🎈🎆", 2, "🎉🎊…"},
		{"empty string", "", 5, ""},
		{"single char", "a", 1, "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncateStr(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
