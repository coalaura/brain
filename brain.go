package brain

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Brain struct {
	id   string
	name string

	loc    *time.Location
	prompt string

	messages int
	images   int
}

func NewBrain(id, name, prompt string, messages, images int, loc *time.Location) *Brain {
	messages = min(max(messages, 10), 100)

	return &Brain{
		id:   id,
		name: name,

		loc:    loc,
		prompt: prompt,

		messages: messages,
		images:   min(max(messages, 0), messages),
	}
}

func (b *Brain) IsSelf(message *discordgo.Message) bool {
	return b.id == message.Author.ID
}

func (b *Brain) GetPrompt() string {
	if b.prompt == "" {
		return ""
	}

	now := time.Now()

	if b.loc != nil {
		now = now.In(b.loc)
	}

	prompt := strings.ReplaceAll(b.prompt, "{time}", now.Format("3:04 PM"))
	prompt = strings.ReplaceAll(prompt, "{date}", now.Format("Monday, 2nd January 2006"))

	return prompt
}
