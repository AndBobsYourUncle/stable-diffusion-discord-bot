package imagine_queue

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"stable_diffusion_bot/composite_renderer"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/repositories/image_generations"
	"stable_diffusion_bot/stable_diffusion_api"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type queueImpl struct {
	botSession          *discordgo.Session
	stableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	queue               chan *QueueItem
	currentImagine      *QueueItem
	mu                  sync.Mutex
	imageGenerationRepo image_generations.Repository
	compositeRenderer   composite_renderer.Renderer
}

type Config struct {
	StableDiffusionAPI  stable_diffusion_api.StableDiffusionAPI
	ImageGenerationRepo image_generations.Repository
}

func New(cfg Config) (Queue, error) {
	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("missing stable diffusion API")
	}

	if cfg.ImageGenerationRepo == nil {
		return nil, errors.New("missing image generation repository")
	}

	compositeRenderer, err := composite_renderer.New(composite_renderer.Config{})
	if err != nil {
		return nil, err
	}

	return &queueImpl{
		stableDiffusionAPI:  cfg.StableDiffusionAPI,
		imageGenerationRepo: cfg.ImageGenerationRepo,
		queue:               make(chan *QueueItem, 100),
		compositeRenderer:   compositeRenderer,
	}, nil
}

type ItemType int

const (
	ItemTypeImagine ItemType = iota
	ItemTypeReroll
	ItemTypeUpscale
)

type QueueItem struct {
	Prompt             string
	Type               ItemType
	UpscaleIndex       int
	DiscordInteraction *discordgo.Interaction
}

func (q *queueImpl) AddImagine(item *QueueItem) (int, error) {
	q.queue <- item

	linePosition := len(q.queue)

	return linePosition, nil
}

func (q *queueImpl) StartPolling(botSession *discordgo.Session) {
	q.botSession = botSession

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
		defer func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			q.currentImagine = nil
		}()

		if q.currentImagine.Type == ItemTypeUpscale {
			q.processUpscaleImagine(q.currentImagine)

			return
		}

		prompt := q.currentImagine.Prompt

		if q.currentImagine.Type == ItemTypeReroll {
			foundPrompt, err := q.getPromptForReroll(q.currentImagine)
			if err != nil {
				log.Printf("Error getting prompt for reroll: %v", err)

				return
			}

			prompt = foundPrompt
		}

		err := q.processImagineGrid(prompt, q.currentImagine)
		if err != nil {
			log.Printf("Error processing imagine grid: %v", err)

			return
		}
	}()
}

func (q *queueImpl) getPromptForReroll(imagine *QueueItem) (string, error) {
	interactionID := imagine.DiscordInteraction.ID
	messageID := ""

	if imagine.DiscordInteraction.Message != nil {
		messageID = imagine.DiscordInteraction.Message.ID
	}

	log.Printf("Reimagining interaction: %v, Message: %v", interactionID, messageID)

	generation, err := q.imageGenerationRepo.GetByMessage(context.Background(), messageID)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return "", err
	}

	log.Printf("Found generation: %v", generation)

	return generation.Prompt, nil
}

func (q *queueImpl) processImagineGrid(prompt string, imagine *QueueItem) error {
	log.Printf("Processing imagine #%s: %v\n", imagine.DiscordInteraction.ID, prompt)

	newContent := fmt.Sprintf("<@%s> asked me to imagine \"%s\". Currently dreaming it up for them.",
		imagine.DiscordInteraction.Member.User.ID,
		prompt)

	message, err := q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &newContent,
	})
	if err != nil {
		log.Printf("Error editing interaction: %v", err)
	}

	newGeneration := &entities.ImageGeneration{
		InteractionID: imagine.DiscordInteraction.ID,
		MessageID:     message.ID,
		MemberID:      imagine.DiscordInteraction.Member.User.ID,
		SortOrder:     0,
		Prompt:        prompt,
		NegativePrompt: "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
			"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
			"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy",
		Width:             768,
		Height:            768,
		RestoreFaces:      true,
		EnableHR:          true,
		DenoisingStrength: 0.7,
		BatchSize:         1,
		Seed:              -1,
		Subseed:           -1,
		SubseedStrength:   0,
		SamplerName:       "Euler a",
		CfgScale:          7,
		Steps:             20,
		Processed:         false,
	}

	_, err = q.imageGenerationRepo.Create(context.Background(), newGeneration)
	if err != nil {
		log.Printf("Error creating image generation record: %v\n", err)
	}

	resp, err := q.stableDiffusionAPI.TextToImage(&stable_diffusion_api.TextToImageRequest{
		Prompt:            newGeneration.Prompt,
		NegativePrompt:    newGeneration.NegativePrompt,
		Width:             newGeneration.Width,
		Height:            newGeneration.Height,
		RestoreFaces:      newGeneration.RestoreFaces,
		EnableHR:          newGeneration.EnableHR,
		DenoisingStrength: newGeneration.DenoisingStrength,
		BatchSize:         newGeneration.BatchSize,
		Seed:              newGeneration.Seed,
		Subseed:           newGeneration.Subseed,
		SubseedStrength:   newGeneration.SubseedStrength,
		SamplerName:       newGeneration.SamplerName,
		CfgScale:          newGeneration.CfgScale,
		Steps:             newGeneration.Steps,
		NIter:             4,
	})
	if err != nil {
		log.Printf("Error processing image: %v\n", err)

		errorContent := "I'm sorry, but I had a problem imagining your image."

		_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
			Content: &errorContent,
		})

		return err
	}

	finishedContent := fmt.Sprintf("<@%s> asked me to imagine \"%s\", here is what I imagined for them.",
		imagine.DiscordInteraction.Member.User.ID,
		prompt,
	)

	log.Printf("Seeds: %v Subseeds:%v", resp.Seeds, resp.Subseeds)

	imageBufs := make([]*bytes.Buffer, len(resp.Images))

	for idx, image := range resp.Images {
		decodedImage, decodeErr := base64.StdEncoding.DecodeString(image)
		if decodeErr != nil {
			log.Printf("Error decoding image: %v\n", decodeErr)
		}

		imageBuf := bytes.NewBuffer(decodedImage)

		imageBufs[idx] = imageBuf
	}

	for idx := range resp.Seeds {
		subGeneration := &entities.ImageGeneration{
			InteractionID:     newGeneration.InteractionID,
			MessageID:         newGeneration.MessageID,
			MemberID:          newGeneration.MemberID,
			SortOrder:         idx + 1,
			Prompt:            newGeneration.Prompt,
			NegativePrompt:    newGeneration.NegativePrompt,
			Width:             newGeneration.Width,
			Height:            newGeneration.Height,
			RestoreFaces:      newGeneration.RestoreFaces,
			EnableHR:          newGeneration.EnableHR,
			DenoisingStrength: newGeneration.DenoisingStrength,
			BatchSize:         newGeneration.BatchSize,
			Seed:              resp.Seeds[idx],
			Subseed:           resp.Subseeds[idx],
			SubseedStrength:   newGeneration.SubseedStrength,
			SamplerName:       newGeneration.SamplerName,
			CfgScale:          newGeneration.CfgScale,
			Steps:             newGeneration.Steps,
			Processed:         true,
		}

		_, createErr := q.imageGenerationRepo.Create(context.Background(), subGeneration)
		if createErr != nil {
			log.Printf("Error creating image generation record: %v\n", createErr)
		}
	}

	compositeImage, err := q.compositeRenderer.TileImages(imageBufs)
	if err != nil {
		log.Printf("Error tiling images: %v\n", err)

		return err
	}

	_, err = q.botSession.InteractionResponseEdit(imagine.DiscordInteraction, &discordgo.WebhookEdit{
		Content: &finishedContent,
		Files: []*discordgo.File{
			{
				ContentType: "image/png",
				Name:        "imagine.png",
				Reader:      compositeImage,
			},
		},
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
							Name: "🎲",
						},
					},
					discordgo.Button{
						// Label is what the user will see on the button.
						Label: "U1",
						// Style provides coloring of the button. There are not so many styles tho.
						Style: discordgo.SecondaryButton,
						// Disabled allows bot to disable some buttons for users.
						Disabled: false,
						// CustomID is a thing telling Discord which data to send when this button will be pressed.
						CustomID: "imagine_upscale_1",
						Emoji: discordgo.ComponentEmoji{
							Name: "⬆️",
						},
					},
					discordgo.Button{
						// Label is what the user will see on the button.
						Label: "U2",
						// Style provides coloring of the button. There are not so many styles tho.
						Style: discordgo.SecondaryButton,
						// Disabled allows bot to disable some buttons for users.
						Disabled: false,
						// CustomID is a thing telling Discord which data to send when this button will be pressed.
						CustomID: "imagine_upscale_2",
						Emoji: discordgo.ComponentEmoji{
							Name: "⬆️",
						},
					},
					discordgo.Button{
						// Label is what the user will see on the button.
						Label: "U3",
						// Style provides coloring of the button. There are not so many styles tho.
						Style: discordgo.SecondaryButton,
						// Disabled allows bot to disable some buttons for users.
						Disabled: false,
						// CustomID is a thing telling Discord which data to send when this button will be pressed.
						CustomID: "imagine_upscale_3",
						Emoji: discordgo.ComponentEmoji{
							Name: "⬆️",
						},
					},
					discordgo.Button{
						// Label is what the user will see on the button.
						Label: "U4",
						// Style provides coloring of the button. There are not so many styles tho.
						Style: discordgo.SecondaryButton,
						// Disabled allows bot to disable some buttons for users.
						Disabled: false,
						// CustomID is a thing telling Discord which data to send when this button will be pressed.
						CustomID: "imagine_upscale_4",
						Emoji: discordgo.ComponentEmoji{
							Name: "⬆️",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error editing interaction: %v\n", err)

		return err
	}

	return nil
}

func (q *queueImpl) processUpscaleImagine(imagine *QueueItem) {
	interactionID := imagine.DiscordInteraction.ID
	messageID := ""

	if imagine.DiscordInteraction.Message != nil {
		messageID = imagine.DiscordInteraction.Message.ID
	}

	log.Printf("Upscaling image: %v, Message: %v, Upscale Index: %d",
		interactionID, messageID, imagine.UpscaleIndex)

	generation, err := q.imageGenerationRepo.GetByMessageAndSort(context.Background(), messageID, imagine.UpscaleIndex)
	if err != nil {
		log.Printf("Error getting image generation: %v", err)

		return
	}

	log.Printf("Found generation: %v", generation)
}
