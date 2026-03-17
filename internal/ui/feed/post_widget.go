package feed

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
	"github.com/dotBeeps/noms/internal/ui/images"
	"github.com/dotBeeps/noms/internal/ui/shared"
	"github.com/dotBeeps/noms/internal/ui/theme"
)

// Style factory functions for facet segment rendering in the hot post render path.
func mentionSegStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorMention) }
func linkSegStyle() lipgloss.Style {
	return lipgloss.NewStyle().Underline(true).Foreground(theme.ColorLink)
}
func tagSegStyle() lipgloss.Style  { return lipgloss.NewStyle().Foreground(theme.ColorTag) }
func textSegStyle() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.ColorText) }

// renderStyled applies a lipgloss style to text, splitting on newlines first to prevent
// lipgloss's alignTextHorizontal from adding trailing whitespace to empty lines. Without this,
// a text segment ending with "\n" (e.g. "word\n") causes lipgloss to pad the empty line after
// the newline to match the widest line width — those spaces then appear before the next segment.
func renderStyled(style lipgloss.Style, text string) string {
	if !strings.Contains(text, "\n") {
		return style.Render(text)
	}
	parts := strings.Split(text, "\n")
	for i, p := range parts {
		parts[i] = style.Render(p)
	}
	return strings.Join(parts, "\n")
}

func imageDims(url string, maxCols, maxRows, minRows int, cache images.ImageRenderer) (cols, rows int) {
	if cache == nil {
		return maxCols, minRows
	}
	w, h, ok := cache.Dimensions(url)
	if !ok || w == 0 || h == 0 {
		return maxCols, minRows
	}
	cols = maxCols
	rows = maxCols * h / (w * 2)
	if rows > maxRows {
		cols = maxRows * w * 2 / h
		rows = maxRows
	}
	if rows < minRows {
		rows = minRows
	}
	return max(cols, 1), max(rows, 1)
}

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

// RenderPostContent builds the post's visual content without border wrapping.
// Used by thread view which applies its own border treatment.
// The width parameter is the content area width (caller accounts for border overhead).
func RenderPostContent(post *bsky.FeedDefs_FeedViewPost, width int, cache images.ImageRenderer, avatarOverrides map[string]string, likeAnim, repostAnim float64) string {
	var b strings.Builder
	contentWidth := max(10, width)

	if post.Reason != nil && post.Reason.FeedDefs_ReasonRepost != nil {
		reason := post.Reason.FeedDefs_ReasonRepost
		handle := reason.By.Handle
		b.WriteString(theme.StyleMuted().Render(fmt.Sprintf("⟲ Reposted by @%s", handle)))
		b.WriteString("\n")
	} else if post.Reply != nil && post.Reply.Parent != nil {
		if parent := post.Reply.Parent.FeedDefs_PostView; parent != nil && parent.Author != nil {
			handle := parent.Author.Handle
			b.WriteString(theme.StyleMuted().Render(fmt.Sprintf("↩ Replying to @%s", handle)))
			b.WriteString("\n")
		}
	}

	if post.Post == nil || post.Post.Author == nil {
		return theme.StyleMuted().Render("[unavailable post]")
	}
	displayName := post.Post.Author.Handle
	if post.Post.Author.DisplayName != nil && *post.Post.Author.DisplayName != "" {
		displayName = ansi.Strip(*post.Post.Author.DisplayName)
	}
	authorName := theme.StyleHeader().Render(displayName)
	handle := theme.StyleMuted().Render("@" + post.Post.Author.Handle)

	timeStr := theme.StyleMuted().Render("now")
	if post.Post.Record != nil && post.Post.Record.Val != nil {
		if record, ok := post.Post.Record.Val.(*bsky.FeedPost); ok {
			if t, err := time.Parse(time.RFC3339, record.CreatedAt); err == nil {
				timeStr = theme.StyleMuted().Render(FormatRelativeTime(t))
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

	avatarStr := ""
	if cache != nil && cache.Enabled() && post.Post.Author.Avatar != nil {
		if cache.IsCached(avatarURL) {
			avatarStr = strings.TrimRight(cache.RenderImage(avatarURL, shared.AvatarCols, shared.AvatarRows), "\n ")
			if avatarStr == "" {
				avatarStr = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
			}
		} else {
			avatarStr = shared.RenderPlaceholder(shared.AvatarCols, shared.AvatarRows)
		}
	}

	avatarContentWidth := max(10, width-shared.AvatarCols-1)

	bodyLines := []string{}
	if post.Post.Record != nil && post.Post.Record.Val != nil {
		if record, ok := post.Post.Record.Val.(*bsky.FeedPost); ok {
			segments := bluesky.ParseFacets(record.Text, record.Facets)
			var body strings.Builder
			for _, seg := range segments {
				segText := ansi.Strip(seg.Text)
				switch seg.Type {
				case bluesky.SegmentMention:
					body.WriteString(renderStyled(mentionSegStyle(), segText))
				case bluesky.SegmentLink:
					body.WriteString(renderStyled(linkSegStyle(), segText))
				case bluesky.SegmentTag:
					body.WriteString(renderStyled(tagSegStyle(), segText))
				default:
					body.WriteString(renderStyled(textSegStyle(), segText))
				}
			}
			if avatarStr != "" {
				bodyLines = strings.Split(lipgloss.NewStyle().Width(avatarContentWidth).Render(body.String()), "\n")
			} else {
				bodyLines = strings.Split(lipgloss.NewStyle().Width(contentWidth).Render(body.String()), "\n")
			}
		}
	}

	if avatarStr != "" {
		paddedNameLine := lipgloss.NewStyle().Width(avatarContentWidth).Render(nameLine)
		completeContent := paddedNameLine + "\n" + strings.Join(bodyLines, "\n")
		b.WriteString(shared.JoinWithGutter(avatarStr, completeContent, " ", shared.AvatarCols))
		b.WriteString("\n")
	} else {
		b.WriteString(nameLine)
		b.WriteString("\n")
		if len(bodyLines) > 0 {
			b.WriteString(strings.Join(bodyLines, "\n"))
			b.WriteString("\n")
		}
	}

	if post.Post.Embed != nil {
		embedWidth := width
		if avatarStr != "" {
			embedWidth = avatarContentWidth
		}
		if embedStr := strings.TrimRight(renderEmbed(post.Post.Embed, embedWidth, cache), "\n "); embedStr != "" {
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
	likeStyle := theme.StyleMuted()
	if liked && likeAnim > 0.05 {
		likeStyle = lipgloss.NewStyle().Foreground(theme.ColorHighlight)
	} else if liked {
		likeStyle = lipgloss.NewStyle().Foreground(theme.ColorAccent)
	}

	repostIcon := "↻"
	repostStyle := theme.StyleMuted()
	if reposted && repostAnim > 0.05 {
		repostStyle = lipgloss.NewStyle().Foreground(theme.ColorHighlight)
	} else if reposted {
		repostStyle = lipgloss.NewStyle().Foreground(theme.ColorSuccess)
	}

	engLine := likeStyle.Render(fmt.Sprintf("%s %d", likeIcon, likeCount)) +
		"  " + repostStyle.Render(fmt.Sprintf("%s %d", repostIcon, repostCount)) +
		"  " + theme.StyleMuted().Render(fmt.Sprintf("↩ %d", replyCount))
	b.WriteString(engLine)
	b.WriteString("\n \n")

	return b.String()
}

// RenderPost renders a complete bordered post panel.
func RenderPost(post *bsky.FeedDefs_FeedViewPost, width int, selected bool, cache images.ImageRenderer, avatarOverrides map[string]string) string {
	return RenderPostAnimated(post, width, selected, cache, avatarOverrides, 0, 0)
}

// RenderPostAnimated renders a post with like/repost pulse animation values (0–1).
func RenderPostAnimated(post *bsky.FeedDefs_FeedViewPost, width int, selected bool, cache images.ImageRenderer, avatarOverrides map[string]string, likeAnim, repostAnim float64) string {
	content := RenderPostContent(post, width-2, cache, avatarOverrides, likeAnim, repostAnim)
	return shared.RenderItemWithBorder(content, selected, width)
}

func renderEmbed(embed *bsky.FeedDefs_PostView_Embed, width int, cache images.ImageRenderer) string {
	if embed == nil {
		return ""
	}

	embedBoxStyle := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Background(theme.ColorSurface).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorBorder).
		Padding(0, 1).
		Width(max(10, width-8))

	switch {
	case embed.EmbedImages_View != nil:
		imgs := embed.EmbedImages_View.Images
		if rendered := renderImages(imgs, width, cache); rendered != "" {
			if len(imgs) == 1 {
				alt := ""
				if imgs[0].Alt != "" {
					alt = ": " + shared.TruncateStr(imgs[0].Alt, 60)
				}
				return rendered + "\n" + embedBoxStyle.Render(fmt.Sprintf("🖼 image%s", alt))
			}
			return rendered + "\n" + embedBoxStyle.Render(fmt.Sprintf("🖼 %d images", len(imgs)))
		}
		if len(imgs) == 1 {
			alt := ""
			if imgs[0].Alt != "" {
				alt = ": " + shared.TruncateStr(imgs[0].Alt, 60)
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
			rendered += embedBoxStyle.Render(fmt.Sprintf("🔗 %s", shared.TruncateStr(ext.Title, 50)))
			return rendered
		}
		title := shared.TruncateStr(ext.Title, 50)
		desc := ""
		if ext.Description != "" {
			desc = "\n" + shared.TruncateStr(ext.Description, 80)
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
				text = shared.TruncateStr(ansi.Strip(fp.Text), 80)
			}
		}
		return embedBoxStyle.Render(fmt.Sprintf("❝ %s\n%s", author, text))

	case embed.EmbedVideo_View != nil:
		vid := embed.EmbedVideo_View
		alt := ""
		if vid.Alt != nil && *vid.Alt != "" {
			alt = ": " + shared.TruncateStr(*vid.Alt, 60)
		}
		return embedBoxStyle.Render(fmt.Sprintf("🎥 video%s", alt))

	case embed.EmbedRecordWithMedia_View != nil:
		rwm := embed.EmbedRecordWithMedia_View
		var parts []string
		if rwm.Media != nil {
			switch {
			case rwm.Media.EmbedImages_View != nil:
				imgs := rwm.Media.EmbedImages_View.Images
				if rendered := renderImages(imgs, width, cache); rendered != "" {
					parts = append(parts, rendered)
					parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("🖼 %d images", len(imgs))))
				} else {
					parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("🖼 %d images", len(imgs))))
				}
			case rwm.Media.EmbedExternal_View != nil && rwm.Media.EmbedExternal_View.External != nil:
				parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("🔗 %s", shared.TruncateStr(rwm.Media.EmbedExternal_View.External.Title, 50))))
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
					text = shared.TruncateStr(ansi.Strip(fp.Text), 80)
				}
			}
			parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("❝ %s\n%s", author, text)))
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

func renderImages(imgs []*bsky.EmbedImages_ViewImage, width int, cache images.ImageRenderer) string {
	if len(imgs) == 0 {
		return ""
	}

	renderOrPlaceholder := func(url string, cols, rows int) string {
		if cache != nil && cache.Enabled() && url != "" && cache.IsCached(url) {
			rendered := strings.TrimRight(cache.RenderImage(url, cols, rows), "\n ")
			if rendered != "" {
				return rendered
			}
		}
		return shared.RenderPlaceholder(cols, rows)
	}

	if len(imgs) == 1 {
		maxCols := min(max(10, width*3/4), 60)
		cols, rows := imageDims(imgs[0].Thumb, maxCols, 28, 8, cache)
		rendered := renderOrPlaceholder(imgs[0].Thumb, cols, rows)
		if imgs[0].Alt != "" {
			rendered += "\n" + theme.StyleMuted().Render(shared.TruncateStr(imgs[0].Alt, 60))
		}
		return rendered
	}

	// Multi-image: 2 per row, wrapping. Each image keeps its own aspect ratio.
	// perImgMaxCols matches half the single-image cap so widths feel consistent.
	perImgMaxCols := min(max(8, (width-2)/2), 30)
	const multiMaxRows = 14
	const multiMinRows = 4

	var rowStrings []string
	for i := 0; i < len(imgs); i += 2 {
		if i+1 < len(imgs) {
			cols1, rows1 := imageDims(imgs[i].Thumb, perImgMaxCols, multiMaxRows, multiMinRows, cache)
			cols2, rows2 := imageDims(imgs[i+1].Thumb, perImgMaxCols, multiMaxRows, multiMinRows, cache)
			img1 := renderOrPlaceholder(imgs[i].Thumb, cols1, rows1)
			img2 := renderOrPlaceholder(imgs[i+1].Thumb, cols2, rows2)
			rowStrings = append(rowStrings, shared.JoinHorizontalRaw(img1, img2, "  "))
		} else {
			cols, rows := imageDims(imgs[i].Thumb, perImgMaxCols, multiMaxRows, multiMinRows, cache)
			rowStrings = append(rowStrings, renderOrPlaceholder(imgs[i].Thumb, cols, rows))
		}
	}
	return strings.Join(rowStrings, "\n")
}
