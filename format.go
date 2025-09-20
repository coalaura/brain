package brain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/revrost/go-openrouter"
	"mvdan.cc/xurls/v2"
)

type Message struct {
	ID string

	Role       string
	Author     string
	Time       string
	Content    string
	Images     []string
	ReplyingTo string
}

func NewCompletionMessage(id string) *Message {
	return &Message{
		ID: id,
	}
}

func (m *Message) Full(index int, indices map[string]int) openrouter.ChatCompletionMessage {
	message := openrouter.ChatCompletionMessage{
		Role: m.Role,
	}

	text := fmt.Sprintf("[%d] %s", index, m.Author)

	if m.ReplyingTo != "" {
		ref, ok := indices[m.ReplyingTo]
		if ok {
			text += fmt.Sprintf(" (reply to %d)", ref)
		}
	}

	text += fmt.Sprintf(" (%s)", m.Time)

	if m.Content != "" {
		text += fmt.Sprintf(": %s", m.Content)
	}

	if len(m.Images) == 0 {
		message.Content.Text = text

		return message
	}

	message.Content.Multi = []openrouter.ChatMessagePart{
		{
			Type: openrouter.ChatMessagePartTypeText,
			Text: text,
		},
	}

	for _, image := range m.Images {
		message.Content.Multi = append(message.Content.Multi, openrouter.ChatMessagePart{
			Type: openrouter.ChatMessagePartTypeImageURL,
			ImageURL: &openrouter.ChatMessageImageURL{
				URL:    image,
				Detail: openrouter.ImageURLDetailAuto,
			},
		})
	}

	return message
}

func FormatMessage(brain *Brain, index *Index, message *discordgo.Message, includeImages bool) *openrouter.ChatCompletionMessage {
	var (
		self    = brain.IsSelf(message)
		content = message.ContentWithMentionsReplaced()
	)

	// Set role
	result := &openrouter.ChatCompletionMessage{
		Role: openrouter.ChatMessageRoleUser,
	}

	if brain.IsSelf(message) {
		result.Role = openrouter.ChatMessageRoleAssistant
	}

	// Format prefix
	prefix := fmt.Sprintf(
		"[%d] %s",
		index.SetById(message.ID),
		FormatAuthor(message.Author),
	)

	if ref := message.ReferencedMessage; ref != nil {
		idx, ok := index.FindById(ref.ID)
		if ok {
			prefix += fmt.Sprintf(" (reply to %d)", idx)
		}
	}

	prefix += fmt.Sprintf(" (%s): ", FormatTimestamp(message.Timestamp, brain.loc))

	// Set content
	if message.Flags&discordgo.MessageFlagsIsVoiceMessage != 0 {
		result.Content.Text = "(Voice Message)"
	} else {
		// Add embeds to content
		for _, embed := range message.Embeds {
			formatted := FormatEmbed(embed)
			if formatted == "" {
				continue
			}

			if content != "" {
				content += "\n"
			}

			content += fmt.Sprintf("(Embed: %s)", formatted)
		}

		// Load images (if needed)
		if includeImages && !self {
			pairs := SplitImagePairs(content, message.Attachments)

			if len(pairs) == 1 && pairs[0].Type == openrouter.ChatMessagePartTypeText {
				result.Content.Text = pairs[0].Text
			} else {
				result.Content.Multi = LoadImagePairs(pairs)
			}
		} else {
			result.Content.Text = FormatMessageContent(content)
		}
	}

	if result.Content.Text == "" && len(result.Content.Multi) == 0 {
		return nil
	}

	if len(result.Content.Multi) != 0 {
		first := result.Content.Multi[0]

		if first.Type == openrouter.ChatMessagePartTypeText {
			result.Content.Multi[0].Text = prefix + first.Text
		} else {
			result.Content.Multi = append([]openrouter.ChatMessagePart{
				{
					Type: openrouter.ChatMessagePartTypeText,
					Text: prefix,
				},
			}, result.Content.Multi...)
		}
	} else {
		result.Content.Text = prefix + result.Content.Text
	}

	return result
}

func FormatAuthor(author *discordgo.User) string {
	if author.GlobalName != "" {
		return author.GlobalName
	}

	return author.Username
}

func FormatTimestamp(timestamp time.Time, loc *time.Location) string {
	if loc != nil {
		timestamp = timestamp.In(loc)
	}

	diff := time.Since(timestamp)

	if diff < time.Minute {
		return "just now"
	} else if diff < 2*time.Minute {
		return "a minute ago"
	} else if diff < time.Hour {
		return fmt.Sprintf("%.0f minutes ago", diff.Minutes())
	} else if diff < 2*time.Hour {
		return "an hour ago"
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%.0f hours ago", diff.Hours())
	} else if diff < 48*time.Hour {
		return "yesterday"
	}

	return fmt.Sprintf("%.0f days ago", diff.Hours()/24)
}

func FormatEmbed(embed *discordgo.MessageEmbed) string {
	data := make(map[string]any)

	if embed.Author != nil && embed.Author.Name != "" {
		data["author"] = embed.Author.Name
	}

	if embed.Title != "" {
		data["title"] = embed.Title
	}

	if embed.Description != "" {
		data["description"] = embed.Description
	}

	jsn, _ := json.Marshal(data)

	if len(jsn) <= 2 {
		return ""
	}

	return string(jsn)
}

func FormatMessageContent(content string) string {
	if content == "" {
		return ""
	}

	// multiple newlines
	rgx := regexp.MustCompile(`\n{2,}`)
	content = rgx.ReplaceAllString(content, "\n")

	// multiple spaces
	rgx = regexp.MustCompile(` {2,}`)
	content = rgx.ReplaceAllString(content, " ")

	// emoji strings
	rgx = regexp.MustCompile(`(?i)<a?(:\w+:)\d+>?`)
	content = rgx.ReplaceAllString(content, "$1")

	// AI loves the em dash
	rgx = regexp.MustCompile(`\s*[–—―;]\s*`)
	content = rgx.ReplaceAllString(content, ", ")

	// and ·
	content = strings.ReplaceAll(content, "·", "-")

	// commas before conjunctions
	rgx = regexp.MustCompile(`,\s+(and|or)`)
	content = rgx.ReplaceAllString(content, " $1")

	// let ‘em
	rgx = regexp.MustCompile(`([^\w])‘([a-z]+)`)
	content = rgx.ReplaceAllString(content, "$1$2")

	// you’re vibin’
	rgx = regexp.MustCompile(`([a-z]+)’([^\w])`)
	content = rgx.ReplaceAllString(content, "$1$2")

	// nobody uses unicode single quotes
	rgx = regexp.MustCompile(`[’‘‚‛‹›]`)
	content = rgx.ReplaceAllString(content, "'")

	// or unicode double quotes
	rgx = regexp.MustCompile(`[«»“”„‟⹂]`)
	content = rgx.ReplaceAllString(content, "\"")

	// remove trailing period, nobody has time for that
	rgx = regexp.MustCompile(`(?m)\.$`)
	content = rgx.ReplaceAllString(content, "")

	// putting markdown image links
	rgx = regexp.MustCompile(`(?m)\s*!\[[\w -]+]\([\w.]+\)`)
	content = rgx.ReplaceAllString(content, "")

	return strings.TrimSpace(content)
}

func SplitMessage(content string, attachments []*discordgo.MessageAttachment) (string, []string) {
	var images []string

	for _, attachment := range attachments {
		if !IsImage(attachment.URL) {
			continue
		}

		images = append(images, attachment.URL)
	}

	seen := make(map[string]bool)

	content = xurls.Strict().ReplaceAllStringFunc(content, func(match string) string {
		uri := match

		// Special case for tenor gif links
		if strings.HasPrefix(uri, "https://tenor.com/view/") {
			uri += ".gif"
		}

		if !IsImage(uri) {
			return match
		}

		if seen[uri] || strings.HasPrefix(uri, "https://cdn.discordapp.com/emojis") {
			return ""
		}

		seen[uri] = true

		images = append(images, uri)

		return ""
	})

	return FormatMessageContent(content), images
}
