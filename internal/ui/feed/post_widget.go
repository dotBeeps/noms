package feed

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/dotBeeps/noms/internal/api/bluesky"
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
func RenderPost(post *bsky.FeedDefs_FeedViewPost, width int, selected bool) string {
	var b strings.Builder

	// Indicator above post (repost or reply)
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

	// Author line: DisplayName @handle · 2h ago
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

	b.WriteString(fmt.Sprintf("%s %s · %s\n", authorName, handle, timeStr))

	// Content with facets (rich text)
	if post.Post.Record != nil && post.Post.Record.Val != nil {
		if record, ok := post.Post.Record.Val.(*bsky.FeedPost); ok {
			segments := bluesky.ParseFacets(record.Text, record.Facets)
			for _, seg := range segments {
				switch seg.Type {
				case bluesky.SegmentMention:
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(seg.Text))
				case bluesky.SegmentLink:
					b.WriteString(lipgloss.NewStyle().Underline(true).Foreground(lipgloss.Color("45")).Render(seg.Text))
				case bluesky.SegmentTag:
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Render(seg.Text))
				default:
					b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(seg.Text))
				}
			}
			b.WriteString("\n")
		}
	}

	if post.Post.Embed != nil {
		if embedStr := renderEmbed(post.Post.Embed, width); embedStr != "" {
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

	res := b.String()

	if selected {
		indicator := lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true).
			Render("▶ ")
		res = indicator + res
	} else {
		res = "  " + res
	}

	// Separator line
	sep := theme.StyleMuted.Render(strings.Repeat("─", max(1, width-4)))
	return res + "\n" + sep + "\n"
}

func renderEmbed(embed *bsky.FeedDefs_PostView_Embed, width int) string {
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
				parts = append(parts, embedBoxStyle.Render(fmt.Sprintf("🖼 %d images", len(rwm.Media.EmbedImages_View.Images))))
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

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "…"
}
