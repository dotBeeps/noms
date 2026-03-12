package feed

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// FormatRelativeTime converts a timestamp to a human-readable relative time string.
// Returns formats like "2m", "1h", "3d", or "Jan 5" for older dates.
func FormatRelativeTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d < 365*24*time.Hour {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	return t.Format("Jan 2, 2006")
}

// RenderPost renders a single post for display in the feed.
// The width parameter controls text wrapping, and selected determines if the post is highlighted.
// ExtractImageURLs returns all image/thumbnail URLs from a post's embed for prefetching.
func ExtractImageURLs(post *bsky.FeedDefs_FeedViewPost) []string {
	if post == nil || post.Post == nil || post.Post.Embed == nil {
		return nil
	}
	embed := post.Post.Embed
	var urls []string

	switch {
	case embed.EmbedImages_View != nil:
		for _, img := range embed.EmbedImages_View.Images {
			if img.Thumb != "" {
				urls = append(urls, img.Thumb)
			}
		}
	case embed.EmbedExternal_View != nil:
		if ext := embed.EmbedExternal_View.External; ext != nil && ext.Thumb != nil && *ext.Thumb != "" {
			urls = append(urls, *ext.Thumb)
		}
	case embed.EmbedVideo_View != nil:
		if vid := embed.EmbedVideo_View; vid.Thumbnail != nil && *vid.Thumbnail != "" {
			urls = append(urls, *vid.Thumbnail)
		}
	case embed.EmbedRecordWithMedia_View != nil:
		if m := embed.EmbedRecordWithMedia_View.Media; m != nil {
			if m.EmbedImages_View != nil {
				for _, img := range m.EmbedImages_View.Images {
					if img.Thumb != "" {
						urls = append(urls, img.Thumb)
					}
				}
			}
			if m.EmbedExternal_View != nil && m.EmbedExternal_View.External != nil {
				if t := m.EmbedExternal_View.External.Thumb; t != nil && *t != "" {
					urls = append(urls, *t)
				}
			}
			if m.EmbedVideo_View != nil && m.EmbedVideo_View.Thumbnail != nil {
				urls = append(urls, *m.EmbedVideo_View.Thumbnail)
			}
		}
	}
	return urls
}

func ExtractAvatarURL(post *bsky.FeedDefs_FeedViewPost) string {
	if post == nil || post.Post == nil || post.Post.Author == nil || post.Post.Author.Avatar == nil {
		return ""
	}
	return *post.Post.Author.Avatar
}

func RenderPost(post *bsky.FeedDefs_FeedViewPost, width int, selected bool, cache *images.Cache, avatarOverrides map[string]string) string {
	var b strings.Builder

	if post.Reason != nil && post.Reason.FeedDefs_ReasonRepost != nil {
		reason := post.Reason.FeedDefs_ReasonRepost
		handle := reason.By.Handle
		b.WriteString(theme.StyleMuted.Render(fmt.Sprintf("⟲ Reposted by @%s", handle)))
		b.WriteString("\n")
	} else if post.Reply != nil && post.Reply.Parent != nil {
		if parent := post.Reply.Parent.FeedDefs_PostView; parent != nil && parent.Author != nil {
			handle := parent.Author.Handle
			b.WriteString(theme.StyleMuted.Render(fmt.Sprintf("↩ Replying to @%s", handle)))
			b.WriteString("\n")
		}
	}

	if post.Post == nil || post.Post.Author == nil {
		return theme.StyleMuted.Render("[unavailable post]")
	}
	displayName := post.Post.Author.Handle
	if post.Post.Author.DisplayName != nil && *post.Post.Author.DisplayName != "" {
		displayName = *post.Post.Author.DisplayName
	}
	authorName := theme.StyleHeader.Render(displayName)
	handle := theme.StyleMuted.Render("@" + post.Post.Author.Handle)

	timeStr := theme.StyleMuted.Render("now")
	if post.Post.Record != nil && post.Post.Record.Val != nil {
		if record, ok := post.Post.Record.Val.(*bsky.FeedPost); ok {
			if t, err := time.Parse(time.RFC3339, record.CreatedAt); err == nil {
				timeStr = theme.StyleMuted.Render(FormatRelativeTime(t))
			}
		}
	}

	nameLine := fmt.Sprintf("%s %s · %s", authorName, handle, timeStr)

	avatarURL := ExtractAvatarURL(post)
	if post != nil && post.Post != nil && post.Post.Author != nil {
		if override, ok := avatarOverrides[post.Post.Author.Did]; ok && override != "" {
			avatarURL = override
		}
	}
	if avatarURL != "" && cache.Enabled() && cache.IsCached(avatarURL) {
		avatarStr := strings.TrimRight(cache.RenderImage(avatarURL, 4, 2), "\n ")
		if avatarStr != "" {
			avatarLines := strings.Split(avatarStr, "\n")
			b.WriteString(avatarLines[0] + " " + nameLine + "\n")
			for i := 1; i < len(avatarLines); i++ {
				b.WriteString(avatarLines[i] + "\n")
			}
		} else {
			b.WriteString(nameLine + "\n")
		}
	} else {
		b.WriteString(nameLine + "\n")
	}

	// Content with facets (rich text), wrapped to account for left border width
	if post.Post.Record != nil && post.Post.Record.Val != nil {
		if record, ok := post.Post.Record.Val.(*bsky.FeedPost); ok {
			segments := bluesky.ParseFacets(record.Text, record.Facets)
			var body strings.Builder
			for _, seg := range segments {
				switch seg.Type {
				case bluesky.SegmentMention:
					body.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(seg.Text))
				case bluesky.SegmentLink:
					body.WriteString(lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("45")).Render(seg.Text))
				case bluesky.SegmentTag:
					body.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Render(seg.Text))
				default:
					body.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(seg.Text))
				}
			}
			contentWidth := max(10, width-2)
			b.WriteString(lipgloss.NewStyle().Width(contentWidth).Render(body.String()))
			b.WriteString("\n")
		}
	}

	if post.Post.Embed != nil {
		if embedStr := strings.TrimRight(renderEmbed(post.Post.Embed, width-2, cache), "\n "); embedStr != "" {
			b.WriteString(embedStr)
			b.WriteString("\n")
		}
	}

	likeCount, repostCount, replyCount := int64(0), int64(0), int64(0)
	if post.Post.LikeCount != nil {
		likeCount = *post.Post.LikeCount
	}
	if post.Post.RepostCount != nil {
		repostCount = *post.Post.RepostCount
	}
	if post.Post.ReplyCount != nil {
		replyCount = *post.Post.ReplyCount
	}

	liked := post.Post.Viewer != nil && post.Post.Viewer.Like != nil && *post.Post.Viewer.Like != ""
	reposted := post.Post.Viewer != nil && post.Post.Viewer.Repost != nil && *post.Post.Viewer.Repost != ""

	likeIcon := "♡"
	likeStyle := theme.StyleMuted
	if liked {
		likeIcon = "♥"
		likeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	}

	repostIcon := "⟲"
	repostStyle := theme.StyleMuted
	if reposted {
		repostStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	}

	engLine := likeStyle.Render(fmt.Sprintf("%s %d", likeIcon, likeCount)) +
		"  " + repostStyle.Render(fmt.Sprintf("%s %d", repostIcon, repostCount)) +
		"  " + theme.StyleMuted.Render(fmt.Sprintf("💬 %d", replyCount))
	b.WriteString(engLine)
	b.WriteString("\n")

	return shared.RenderItemWithBorder(b.String(), selected, width)
}

func renderEmbed(embed *bsky.FeedDefs_PostView_Embed, width int, cache *images.Cache) string {
	if embed == nil {
		return ""
	}

	embedBoxStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("246")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(max(10, width-8))

	switch {
	case embed.EmbedImages_View != nil:
		imgs := embed.EmbedImages_View.Images
		if rendered := renderImages(imgs, width, cache); rendered != "" {
			return rendered
		}
		if len(imgs) == 1 {
			alt := ""
			if imgs[0].Alt != "" {
				alt = ": " + truncateStr(imgs[0].Alt, 60)
			}
			return embedBoxStyle.Render(fmt.Sprintf("🖼 image%s", alt))
		}
		return embedBoxStyle.Render(fmt.Sprintf("🖼 %d images", len(imgs)))

	case embed.EmbedExternal_View != nil:
		ext := embed.EmbedExternal_View.External
		if ext == nil {
			return ""
		}
		if ext.Thumb != nil && *ext.Thumb != "" && cache.Enabled() && cache.IsCached(*ext.Thumb) {
			cols := max(10, width/4)
			rendered := strings.TrimRight(cache.RenderImage(*ext.Thumb, cols, 4), "\n ")
			if rendered != "" {
				rendered += "\n"
			}
			rendered += embedBoxStyle.Render(fmt.Sprintf("🔗 %s", truncateStr(ext.Title, 50)))
			return rendered
		}
		title := truncateStr(ext.Title, 50)
		desc := ""
		if ext.Description != "" {
			desc = "\n" + truncateStr(ext.Description, 80)
		}
		return embedBoxStyle.Render(fmt.Sprintf("🔗 %s%s", title, desc))

	case embed.EmbedRecord_View != nil:
		rec := embed.EmbedRecord_View.Record
		if rec == nil || rec.EmbedRecord_ViewRecord == nil {
			return ""
		}
		vr := rec.EmbedRecord_ViewRecord
		author := ""
		if vr.Author != nil {
			author = "@" + vr.Author.Handle
		}
		text := ""
		if vr.Value != nil {
			if fp, ok := vr.Value.Val.(*bsky.FeedPost); ok {
				text = truncateStr(fp.Text, 80)
			}
		}
		return embedBoxStyle.Render(fmt.Sprintf("❝ %s\n%s", author, text))

	case embed.EmbedVideo_View != nil:
		vid := embed.EmbedVideo_View
		alt := ""
		if vid.Alt != nil && *vid.Alt != "" {
			alt = ": " + truncateStr(*vid.Alt, 60)
		}
		return embedBoxStyle.Render(fmt.Sprintf("🎥 video%s", alt))

	case embed.EmbedRecordWithMedia_View != nil:
		rwm := embed.EmbedRecordWithMedia_View
		var parts []string
		if rwm.Media != nil {
			switch {
			case rwm.Media.EmbedImages_View != nil:
				if rendered := renderImages(rwm.Media.EmbedImages_View.Images, width, cache); rendered != "" {
					parts = append(parts, rendered)
				} else {
					parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("🖼 %d images", len(rwm.Media.EmbedImages_View.Images))))
				}
			case rwm.Media.EmbedExternal_View != nil && rwm.Media.EmbedExternal_View.External != nil:
				parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("🔗 %s", truncateStr(rwm.Media.EmbedExternal_View.External.Title, 50))))
			case rwm.Media.EmbedVideo_View != nil:
				parts = append(parts, embedBoxStyle.Render("🎥 video"))
			}
		}
		if rwm.Record != nil && rwm.Record.Record != nil && rwm.Record.Record.EmbedRecord_ViewRecord != nil {
			vr := rwm.Record.Record.EmbedRecord_ViewRecord
			author := ""
			if vr.Author != nil {
				author = "@" + vr.Author.Handle
			}
			text := ""
			if vr.Value != nil {
				if fp, ok := vr.Value.Val.(*bsky.FeedPost); ok {
					text = truncateStr(fp.Text, 80)
				}
			}
			parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("❝ %s\n%s", author, text)))
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

func renderImages(imgs []*bsky.EmbedImages_ViewImage, width int, cache *images.Cache) string {
	if len(imgs) == 0 || cache == nil || !cache.Enabled() {
		return ""
	}

	var cached []*bsky.EmbedImages_ViewImage
	for _, img := range imgs {
		if cache.IsCached(img.Thumb) {
			cached = append(cached, img)
		}
	}
	if len(cached) == 0 {
		return ""
	}

	switch len(cached) {
	case 1:
		cols := max(10, width*3/4)
		rendered := strings.TrimRight(cache.RenderImage(cached[0].Thumb, cols, 16), "\n ")
		if rendered == "" {
			return ""
		}
		if cached[0].Alt != "" {
			rendered += "\n" + theme.StyleMuted.Render(truncateStr(cached[0].Alt, 60))
		}
		return rendered

	case 2:
		cols := max(8, (width-2)/2)
		img1 := strings.TrimRight(cache.RenderImage(cached[0].Thumb, cols, 10), "\n ")
		img2 := strings.TrimRight(cache.RenderImage(cached[1].Thumb, cols, 10), "\n ")
		if img1 == "" || img2 == "" {
			return ""
		}
		return joinHorizontalRaw(img1, img2, "  ")

	case 3:
		cols := max(8, (width-2)/2)
		img1 := strings.TrimRight(cache.RenderImage(cached[0].Thumb, cols, 8), "\n ")
		img2 := strings.TrimRight(cache.RenderImage(cached[1].Thumb, cols, 8), "\n ")
		img3 := strings.TrimRight(cache.RenderImage(cached[2].Thumb, cols, 8), "\n ")
		if img1 == "" || img2 == "" || img3 == "" {
			return ""
		}
		return joinHorizontalRaw(img1, img2, "  ") + "\n" + img3

	default:
		cols := max(8, (width-2)/2)
		img1 := strings.TrimRight(cache.RenderImage(cached[0].Thumb, cols, 8), "\n ")
		img2 := strings.TrimRight(cache.RenderImage(cached[1].Thumb, cols, 8), "\n ")
		img3 := strings.TrimRight(cache.RenderImage(cached[2].Thumb, cols, 8), "\n ")
		img4 := strings.TrimRight(cache.RenderImage(cached[3].Thumb, cols, 8), "\n ")
		if img1 == "" || img2 == "" || img3 == "" || img4 == "" {
			return ""
		}
		return joinHorizontalRaw(img1, img2, "  ") + "\n" + joinHorizontalRaw(img3, img4, "  ")
	}
}

// joinHorizontalRaw joins two multi-line strings side-by-side with a separator.
// Unlike lipgloss.JoinHorizontal, this does NOT pad lines to equal width,
// avoiding width miscalculation with Kitty Unicode placeholder characters.
func joinHorizontalRaw(left, right, sep string) string {
	leftLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(right, "\n"), "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	var result strings.Builder
	for i := 0; i < maxLines; i++ {
		if i > 0 {
			result.WriteString("\n")
		}
		l, r := "", ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		result.WriteString(l)
		if r != "" {
			result.WriteString(sep)
			result.WriteString(r)
		}
	}
	return result.String()
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}
