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
		return nil, errors.New("missing bot session")
	}

	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("missing stable diffusion API")
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
			log.Printf("Error processing image: %v\n", err)

			errorContent := "I'm sorry, but I had a problem imagining your image."

			_, err = q.botSession.InteractionResponseEdit(q.currentImagine.DiscordInteraction, &discordgo.WebhookEdit{
				Content: &errorContent,
			})

			return
		}

		finishedContent := fmt.Sprintf("<@%s> asked me to reimagine \"%s\", here is what I imagined for them.",
			q.currentImagine.DiscordInteraction.Member.User.ID,
			q.currentImagine.Prompt,
		)

		attachedImages := make([]*discordgo.File, len(resp.Images))

		for idx, image := range resp.Images {
			decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
			if decodeErr != nil {
				log.Printf("Error decoding image: %v\n", decodeErr)
			}

			imageBuf := bytes.NewBuffer(decodedImage)

			attachedImages[idx] = &discordgo.File{
				ContentType: "image/png",
				Name:        "imagine.png",
				Reader:      imageBuf,
			}
		}

		_, err = q.botSession.InteractionResponseEdit(q.currentImagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &finishedContent,
			Files:   attachedImages,
			Components: &[]discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							// Label is what the user will see on the button.
							Label: "Re-roll",
							// Style provides coloring of the button. There are not so many styles tho.
							Style: discordgo.PrimaryButton,
							// Disabled allows bot to disable some buttons for users.
							Disabled: false,
							// CustomID is a thing telling Discord which data to send when this button will be pressed.
							CustomID: "imagine_reroll",
							Emoji: discordgo.ComponentEmoji{
								Name: "🤷",
							},
						},
					},
				},
			},
		})
		if err != nil {
			log.Printf("Error editing interaction: %v\n", err)
		}

		q.mu.Lock()
		defer q.mu.Unlock()

		q.currentImagine = nil
	}()
}
