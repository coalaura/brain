package brain

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/revrost/go-openrouter"
)

type codeRegion struct {
	Start int
	End   int
}

func findMarkdownCodeRegions(text string) []codeRegion {
	var regions []codeRegion

	inline := regexp.MustCompile(`\x60[^\x60\n]+?\x60`)

	for _, match := range inline.FindAllStringIndex(text, -1) {
		regions = append(regions, codeRegion{
			Start: match[0],
			End:   match[1],
		})
	}

	fenced := regexp.MustCompile(`(?m)^\x60\x60\x60[^\n]*\n(.*?\n)^\x60\x60\x60\s*$`)

	for _, match := range fenced.FindAllStringIndex(text, -1) {
		regions = append(regions, codeRegion{
			Start: match[0],
			End:   match[1],
		})
	}

	return regions
}

func isInsideCodeBlock(pos int, regions []codeRegion) bool {
	for _, region := range regions {
		if pos >= region.Start && pos < region.End {
			return true
		}
	}

	return false
}

func splitImagePairs(text string, attachments []*discordgo.MessageAttachment) []openrouter.ChatMessagePart {
	code := findMarkdownCodeRegions(text)

	rgx := regexp.MustCompile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)

	var (
		index int
		parts []openrouter.ChatMessagePart
	)

	push := func(str, end int) {
		if str > end {
			return
		}

		rest := text[str:end]

		if rest == "" {
			return
		}

		total := len(parts)

		if total > 0 && parts[total-1].Type == openrouter.ChatMessagePartTypeText {
			parts[total-1].Text += rest

			return
		}

		parts = append(parts, openrouter.ChatMessagePart{
			Type: openrouter.ChatMessagePartTypeText,
			Text: rest,
		})
	}

	for {
		location := rgx.FindStringSubmatchIndex(text[index:])
		if location == nil {
			push(index, len(text))

			break
		}

		start := index + location[0]
		end := index + location[1]

		if isInsideCodeBlock(start, code) {
			push(index, end)

			index = end

			continue
		}

		url := text[start:end]

		if !isImage(url) {
			push(index, end)

			index = end

			continue
		}

		if start > index {
			push(index, start)
		}

		parts = append(parts, openrouter.ChatMessagePart{
			Type: openrouter.ChatMessagePartTypeImageURL,
			ImageURL: &openrouter.ChatMessageImageURL{
				Detail: openrouter.ImageURLDetailAuto,
				URL:    url,
			},
		})

		index = end
	}

	for _, attachment := range attachments {
		if !isImage(attachment.URL) {
			continue
		}

		parts = append(parts, openrouter.ChatMessagePart{
			Type: openrouter.ChatMessagePartTypeImageURL,
			ImageURL: &openrouter.ChatMessageImageURL{
				Detail: openrouter.ImageURLDetailAuto,
				URL:    attachment.URL,
			},
		})
	}

	return parts
}
