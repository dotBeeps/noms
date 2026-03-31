package feed

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/dotBeeps/noms/internal/ui/shared"
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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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

// TestRenderPostExternalThumbPlaceholderWhenRenderEmpty verifies that when
// an external link card's thumbnail is cached but RenderImage returns ""
// (e.g. LazyRenderer, non-near-visible Kitty), a placeholder is shown
// instead of silently dropping the thumbnail.
func TestRenderPostExternalThumbPlaceholderWhenRenderEmpty(t *testing.T) {
	t.Parallel()
	thumbURL := "https://example.com/thumb.jpg"
	post := createTestPost("Check this link", "test.bsky.social", "Tester", "at://uri_ext", "cid_ext")
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedExternal_View: &bsky.EmbedExternal_View{
			External: &bsky.EmbedExternal_ViewExternal{
				Title: "Cool Article",
				Uri:   "https://example.com/article",
				Thumb: &thumbURL,
			},
		},
	}

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return "" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))

	// Should contain placeholder box characters (┌ or [····]) not just the text link
	hasPlaceholder := strings.Contains(rendered, "┌") || strings.Contains(rendered, "[····]")
	if !hasPlaceholder {
		t.Errorf("Expected placeholder for cached-but-empty-render external thumbnail, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Cool Article") {
		t.Error("Expected link title in rendered post")
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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

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
		rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))
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
		rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))
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
		rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))
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
		rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))
		if !strings.Contains(rendered, "🎥") {
			t.Error("Expected video indicator 🎥 even with nil alt")
		}
		if !strings.Contains(rendered, "video") {
			t.Error("Expected 'video' text even with nil alt")
		}
	})
}

func TestExtractAvatarURL(t *testing.T) {
	t.Parallel()

	t.Run("returns avatar URL when present", func(t *testing.T) {
		t.Parallel()
		post := createTestPost("Hello", "test.bsky.social", "Test", "at://uri", "cid")
		avatar := "https://cdn.bsky.app/img/avatar/test.jpg"
		post.Post.Author.Avatar = &avatar
		if got := ExtractAvatarURL(post); got != avatar {
			t.Errorf("ExtractAvatarURL() = %q, want %q", got, avatar)
		}
	})

	t.Run("returns empty when no avatar", func(t *testing.T) {
		t.Parallel()
		post := createTestPost("Hello", "test.bsky.social", "Test", "at://uri", "cid")
		if got := ExtractAvatarURL(post); got != "" {
			t.Errorf("ExtractAvatarURL() = %q, want empty", got)
		}
	})

	t.Run("returns empty for nil post", func(t *testing.T) {
		t.Parallel()
		if got := ExtractAvatarURL(nil); got != "" {
			t.Errorf("ExtractAvatarURL(nil) = %q, want empty", got)
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
			result := shared.TruncateStr(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("shared.TruncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

type stubImageRenderer struct {
	enabled  bool
	isCached func(url string) bool
	render   func(url string, cols, rows int) string
}

func (s *stubImageRenderer) Enabled() bool { return s.enabled }
func (s *stubImageRenderer) IsCached(url string) bool {
	if s.isCached == nil {
		return false
	}
	return s.isCached(url)
}
func (s *stubImageRenderer) RenderImage(url string, cols, rows int) string {
	if s.render == nil {
		return ""
	}
	return s.render(url, cols, rows)
}
func (s *stubImageRenderer) FetchAvatar(url string) tea.Cmd         { return nil }
func (s *stubImageRenderer) InvalidateTransmissions()               {}
func (s *stubImageRenderer) Dimensions(url string) (int, int, bool) { return 0, 0, false }

func TestRenderPostAvatarLeftWhenCached(t *testing.T) {
	t.Parallel()
	post := createTestPost("Some post content", "avtest.bsky.social", "AvTestUser", "at://uri_av1", "cid_av1")
	avatarURL := "https://example.com/avatar_cached.jpg"
	post.Post.Author.Avatar = &avatarURL

	const avatarPixels = "XAVT\nXAVT"
	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return avatarPixels },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))

	if !strings.Contains(rendered, "XAVT") {
		t.Fatal("Expected avatar string 'XAVT' in rendered post when avatar is cached")
	}
	if !strings.Contains(rendered, "AvTestUser") {
		t.Fatal("Expected author display name 'AvTestUser' in rendered post")
	}
	avtrIdx := strings.Index(rendered, "XAVT")
	nameIdx := strings.Index(rendered, "AvTestUser")
	if avtrIdx >= nameIdx {
		t.Errorf("Expected avatar (index %d) to appear before author name (index %d) — avatar-left layout broken", avtrIdx, nameIdx)
	}
}

func TestRenderPostPlaceholderWhenUncached(t *testing.T) {
	t.Parallel()
	post := createTestPost("Placeholder post", "pltest.bsky.social", "PlTestUser", "at://uri_pl1", "cid_pl1")
	avatarURL := "https://example.com/avatar_uncached.jpg"
	post.Post.Author.Avatar = &avatarURL

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return false },
		render:   func(url string, cols, rows int) string { return "" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))

	if !strings.Contains(rendered, "[····]") {
		t.Errorf("Expected placeholder '[····]' in rendered post when avatar is not cached; got:\n%s", rendered)
	}
}

func TestRenderPostNoLeadingBlankLine(t *testing.T) {
	t.Parallel()
	post := createTestPost("Padding test post", "padtest.bsky.social", "PaddingDisplayName", "at://uri_pad1", "cid_pad1")

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))
	lines := strings.Split(rendered, "\n")

	nameLineIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "PaddingDisplayName") {
			nameLineIdx = i
			break
		}
	}
	if nameLineIdx != 0 {
		t.Errorf("Expected 'PaddingDisplayName' on the first line (no leading blank). Got line index %d in:\n%s", nameLineIdx, rendered)
	}
}

func TestRenderPostNewFooterIconsPresent(t *testing.T) {
	t.Parallel()
	post := createTestPost("Footer icon test", "footer.bsky.social", "FooterUser", "at://uri_fi1", "cid_fi1")

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

	for _, icon := range []string{"♡", "↻", "↩"} {
		if !strings.Contains(rendered, icon) {
			t.Errorf("Expected new footer icon %q to be present in rendered post", icon)
		}
	}
}

func TestRenderPostOldFooterIconsAbsent(t *testing.T) {
	t.Parallel()
	post := createTestPost("Old icon check", "oldicon.bsky.social", "OldIconUser", "at://uri_oi1", "cid_oi1")

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

	for _, oldIcon := range []string{"⟲", "💬", "♥"} {
		if strings.Contains(rendered, oldIcon) {
			t.Errorf("Expected old footer icon %q to be absent in rendered post, but it was found", oldIcon)
		}
	}
}

func TestRenderPostNilCacheFallback(t *testing.T) {
	t.Parallel()
	post := createTestPost("Nil cache content here", "nilcache.bsky.social", "NilCacheUser", "at://uri_nc1", "cid_nc1")
	avatarURL := "https://example.com/avatar.jpg"
	post.Post.Author.Avatar = &avatarURL

	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

	if !strings.Contains(rendered, "Nil cache content here") {
		t.Error("Expected post text to appear in rendered output with nil cache")
	}
	if !strings.Contains(rendered, "NilCacheUser") {
		t.Error("Expected author name to appear in rendered output with nil cache")
	}
}

func TestPostWidgetAvatarIndentAllRows(t *testing.T) {
	t.Parallel()

	post := createTestPost("L1\nL2\nL3\nL4\nL5", "indent.bsky.social", "IndentUser", "at://indent", "cid_indent")
	avatarURL := "https://example.com/avatar_indent.jpg"
	post.Post.Author.Avatar = &avatarURL

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return "AAAAAA\nBBBBBB\nCCCCCC" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))
	lines := strings.Split(rendered, "\n")

	for _, marker := range []string{"L3", "L4", "L5"} {
		found := false
		for _, line := range lines {
			idx := strings.Index(line, marker)
			if idx == -1 {
				continue
			}
			found = true
			prefix := line[:idx]
			if !strings.HasSuffix(prefix, strings.Repeat(" ", shared.AvatarCols+1)) {
				t.Fatalf("expected line containing %q to include >=%d trailing gutter spaces before content; line=%q", marker, shared.AvatarCols+1, line)
			}
			break
		}
		if !found {
			t.Fatalf("expected to find marker %q in rendered output", marker)
		}
	}
}

func TestPostWidgetNoAvatarNoIndent(t *testing.T) {
	t.Parallel()

	post := createTestPost("BodyLineNoAvatar", "noindent.bsky.social", "NoIndentUser", "at://noindent", "cid_noindent")
	rendered := stripAnsi(RenderPost(post, 80, false, nil, nil))

	for _, line := range strings.Split(rendered, "\n") {
		if !(strings.Contains(line, "NoIndentUser") || strings.Contains(line, "BodyLineNoAvatar")) {
			continue
		}
		if strings.Contains(line, strings.Repeat(" ", shared.AvatarCols+1)+"BodyLineNoAvatar") {
			t.Fatalf("unexpected avatar gutter indentation in no-avatar render: %q", line)
		}
	}
}

func TestPostWidgetEmbedWidthReducedWithAvatar(t *testing.T) {
	t.Parallel()

	post := createTestPost("Embed width check", "embedwidth.bsky.social", "EmbedWidthUser", "at://embedwidth", "cid_embedwidth")
	avatarURL := "https://example.com/avatar_embed.jpg"
	post.Post.Author.Avatar = &avatarURL
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedExternal_View: &bsky.EmbedExternal_View{
			External: &bsky.EmbedExternal_ViewExternal{
				Title:       strings.Repeat("T", 120),
				Description: strings.Repeat("D", 180),
				Uri:         "https://example.com/very-long-link",
			},
		},
	}

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return "AVATAR\nAVATAR\nAVATAR" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))
	for _, line := range strings.Split(rendered, "\n") {
		trimmed := strings.TrimLeft(line, "▎ ")
		trimmed = strings.TrimRight(trimmed, " ")
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "🔗") || strings.Contains(trimmed, "TTTT") || strings.Contains(trimmed, "DDDD") || strings.Contains(trimmed, "╭") || strings.Contains(trimmed, "│") || strings.Contains(trimmed, "╰") {
			if got := len([]rune(trimmed)); got > 71 {
				t.Fatalf("embed-related line width exceeded 71 with avatar: got=%d line=%q", got, trimmed)
			}
		}
	}
}

func TestPostWidgetAvatarPresentInOutput(t *testing.T) {
	t.Parallel()

	post := createTestPost("Avatar appears", "avatarpresent.bsky.social", "AvatarPresentUser", "at://avpresent", "cid_avpresent")
	avatarURL := "https://example.com/avatar_present.jpg"
	post.Post.Author.Avatar = &avatarURL

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return "VISIBLE_AVATAR" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))
	if !strings.Contains(rendered, "VISIBLE_AVATAR") {
		t.Fatalf("expected avatar marker in rendered output, got: %q", rendered)
	}
}

func TestEmbedIndentedToGutterWhenAvatarPresent(t *testing.T) {
	t.Parallel()

	post := createTestPost("Post with embed", "embedtest.bsky.social", "EmbedUser", "at://uri_em1", "cid_em1")
	avatarURL := "https://example.com/avatar_embed.jpg"
	post.Post.Author.Avatar = &avatarURL
	post.Post.Embed = &bsky.FeedDefs_PostView_Embed{
		EmbedExternal_View: &bsky.EmbedExternal_View{
			External: &bsky.EmbedExternal_ViewExternal{
				Title: "Test Link Title",
				Uri:   "https://example.com",
			},
		},
	}

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return "AVEMB\nAVEMB\nAVEMB" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))
	lines := strings.Split(rendered, "\n")

	indent := strings.Repeat(" ", shared.GutterWidth)
	embedLineFound := false
	for _, line := range lines {
		if strings.Contains(line, "Test Link Title") {
			embedLineFound = true
			// Each line starts with "▎ " (border + padding), then our content.
			// After stripping that prefix, check GutterWidth spaces precede the embed box.
			afterBorder := strings.TrimPrefix(line, "▎ ")
			if afterBorder == line {
				t.Errorf("embed line missing border prefix: %q", line)
				continue
			}
			if !strings.HasPrefix(afterBorder, indent) {
				t.Errorf("embed line not indented by GutterWidth (%d) after border: got %q", shared.GutterWidth, afterBorder)
			}
		}
	}
	if !embedLineFound {
		t.Fatal("expected embed title 'Test Link Title' in rendered output")
	}
}

func TestPostWidgetBodyWrappedToAvatarWidth(t *testing.T) {
	t.Parallel()

	post := createTestPost(strings.Repeat("Q", 170), "wrap.bsky.social", "WrapUser", "at://wrap", "cid_wrap")
	avatarURL := "https://example.com/avatar_wrap.jpg"
	post.Post.Author.Avatar = &avatarURL

	stub := &stubImageRenderer{
		enabled:  true,
		isCached: func(url string) bool { return true },
		render:   func(url string, cols, rows int) string { return "AVWRAP\nAVWRAP\nAVWRAP" },
	}

	rendered := stripAnsi(RenderPost(post, 80, false, stub, nil))
	lines := strings.Split(rendered, "\n")

	foundBody := false
	for _, line := range lines {
		idx := strings.Index(line, "QQQ")
		if idx == -1 {
			continue
		}
		foundBody = true
		bodyPart := line[idx:]
		bodyPart = strings.TrimRight(bodyPart, " ")
		if got := len([]rune(bodyPart)); got > 71 {
			t.Fatalf("wrapped body line exceeds 71 cols with avatar: got=%d line=%q", got, bodyPart)
		}
	}

	if !foundBody {
		t.Fatal("expected wrapped body content lines in rendered output")
	}
}
