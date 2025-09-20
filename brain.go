package brain

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Timezone  *time.Location
	ImageSize uint
}

type Brain struct {
	id   string
	name string

	cfg    Config
	prompt string

	messages int
	images   int
}

var Defaults = Config{
	Timezone:  nil,
	ImageSize: 1024,
}

func NewBrain(id, name, prompt string, messages, images int, cfg Config) *Brain {
	messages = min(max(messages, 10), 100)
	images = min(max(images, 0), messages)

	return &Brain{
		id:   id,
		name: name,

		cfg:    cfg,
		prompt: prompt,

		messages: messages,
		images:   images,
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

	if b.cfg.Timezone != nil {
		now = now.In(b.cfg.Timezone)
	}

	prompt := strings.ReplaceAll(b.prompt, "{time}", now.Format("3:04 PM"))
	prompt = strings.ReplaceAll(prompt, "{date}", now.Format("Monday, 2nd January 2006"))

	return prompt
}
