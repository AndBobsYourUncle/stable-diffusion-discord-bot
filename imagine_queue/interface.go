package imagine_queue

import "github.com/bwmarrin/discordgo"

type Queue interface {
	AddImagine(item *QueueItem) (int, error)
	StartPolling(botSession *discordgo.Session)
	GetDefaultBotWidth() (int, error)
	GetDefaultBotHeight() (int, error)
	UpdateDefaultDimensions(width, height int) error
}
