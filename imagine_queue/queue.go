package imagine_queue

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"stable_diffusion_bot/stable_diffusion_api"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type queueImpl struct {
	botSession         *discordgo.Session
	stableDiffusionAPI stable_diffusion_api.StableDiffusionAPI
	queue              chan *QueueItem
	currentImagine     *QueueItem
	mu                 sync.Mutex
}

type Config struct {
	BotSession         *discordgo.Session
	StableDiffusionAPI stable_diffusion_api.StableDiffusionAPI
}

func New(cfg Config) (Queue, error) {
	if cfg.BotSession == nil {
		return nil, errors.New("bot session is nil")
	}

	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("stable diffusion API is nil")
	}

	return &queueImpl{
		botSession:         cfg.BotSession,
		stableDiffusionAPI: cfg.StableDiffusionAPI,
		queue:              make(chan *QueueItem, 100),
	}, nil
}

type QueueItem struct {
	Prompt             string
	DiscordInteraction *discordgo.Interaction
}

func (q *queueImpl) AddImagine(item *QueueItem) (int, error) {
	q.queue <- item

	linePosition := len(q.queue)

	if q.currentImagine == nil {
		q.pullNextInQueue()
	}

	return linePosition, nil
}

func (q *queueImpl) StartPolling() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	stopPolling := false

	for {
		select {
		case <-stop:
			stopPolling = true
		case <-time.After(1 * time.Second):
			if q.currentImagine == nil {
				q.pullNextInQueue()
			}
		}

		if stopPolling {
			break
		}
	}

	log.Printf("Polling stopped...\n")
}

func (q *queueImpl) pullNextInQueue() {
	if len(q.queue) > 0 {
		element := <-q.queue

		q.mu.Lock()
		defer q.mu.Unlock()

		q.currentImagine = element

		q.processCurrentImagine()
	}
}

func (q *queueImpl) processCurrentImagine() {
	go func() {
		log.Printf("Processing imagine: %v\n", q.currentImagine.Prompt)

		newContent := fmt.Sprintf("<@%s> asked me to imagine \"%s\". Currently dreaming it up for them.",
			q.currentImagine.DiscordInteraction.Member.User.ID,
			q.currentImagine.Prompt)

		q.botSession.InteractionResponseEdit(q.currentImagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &newContent,
		})

		resp, err := q.stableDiffusionAPI.TextToImage(q.currentImagine.Prompt)
		if err != nil {
			log.Printf("Error processing imagine: %v\n", err)
		}

		finishedContent := fmt.Sprintf("<@%s>, here is what I imagined for \"%s\".",
			q.currentImagine.DiscordInteraction.Member.User.ID,
			q.currentImagine.Prompt)

		decodedImage, err := base64.StdEncoding.DecodeString(resp.Images[0])
		if err != nil {
			log.Printf("Error decoding imagine: %v\n", err)
		}

		bytesio := bytes.NewBuffer(decodedImage)

		q.botSession.InteractionResponseEdit(q.currentImagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &finishedContent,
			Files: []*discordgo.File{
				{
					ContentType: "image/png",
					Name:        "imagine.png",
					Reader:      bytesio,
				},
			},
		})

		q.mu.Lock()
		defer q.mu.Unlock()

		q.currentImagine = nil
	}()
}
