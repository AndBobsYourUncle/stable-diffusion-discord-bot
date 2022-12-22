package imagine_queue

import "github.com/bwmarrin/discordgo"

type Queue interface {
	AddImagine(item *QueueItem) (int, error)
	StartPolling(botSession *discordgo.Session)
}
