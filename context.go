package brain

import (
	"fmt"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/revrost/go-openrouter"
)

func (b *Brain) Build(s *discordgo.Session, channelId string) ([]openrouter.ChatCompletionMessage, error) {
	var (
		index    = NewIndex(b.messages)
		messages = make([]openrouter.ChatCompletionMessage, 0, b.messages)
	)

	messages = append(messages, openrouter.SystemMessage(b.GetPrompt()))

	fresh, err := s.ChannelMessages(channelId, b.messages, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed retrieving messages: %v", err)
	}

	important := make(map[string]bool, b.images)

	for i := 0; i < min(b.images, len(fresh)); i++ {
		message := fresh[i]

		important[message.ID] = true

		if ref := message.ReferencedMessage; ref != nil {
			important[ref.ID] = true
		}
	}

	// Discord messages are in reverse order (newest first)
	// so we need to reverse them to get the correct order
	// for the context
	slices.Reverse(fresh)

	for _, message := range fresh {
		completion := formatMessage(b, index, message, important[message.ID])

		if completion != nil {
			messages = append(messages, *completion)
		}
	}

	return messages, nil
}
