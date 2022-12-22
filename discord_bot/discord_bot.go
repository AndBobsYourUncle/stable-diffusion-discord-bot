package discord_bot

import (
	"errors"
	"fmt"
	"log"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/repositories/image_generations"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	botSession          *discordgo.Session
	guildID             string
	imagineQueue        imagine_queue.Queue
	imageGenerationRepo image_generations.Repository
	registeredCommands  []*discordgo.ApplicationCommand
}

type Config struct {
	BotToken            string
	GuildID             string
	ImagineQueue        imagine_queue.Queue
	ImageGenerationRepo image_generations.Repository
}

func New(cfg Config) (Bot, error) {
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	if cfg.GuildID == "" {
		return nil, errors.New("missing guild ID")
	}

	if cfg.ImagineQueue == nil {
		return nil, errors.New("missing imagine queue")
	}

	if cfg.ImageGenerationRepo == nil {
		return nil, errors.New("missing image generation repo")
	}

	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = botSession.Open()
	if err != nil {
		return nil, err
	}

	bot := &botImpl{
		botSession:          botSession,
		imagineQueue:        cfg.ImagineQueue,
		imageGenerationRepo: cfg.ImageGenerationRepo,
		registeredCommands:  make([]*discordgo.ApplicationCommand, 0),
	}

	err = bot.addImagineCommand()
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case "imagine":
				bot.processImagineCommand(s, i)
			default:
				log.Printf("Unknown command '%v'", i.ApplicationCommandData().Name)
			}
		case discordgo.InteractionMessageComponent:
			switch i.MessageComponentData().CustomID {
			case "imagine_reroll":
				bot.processImagineReroll(s, i)
			case "imagine_upscale_1":
				bot.processImagineUpscale(s, i, 1)
			case "imagine_upscale_2":
				bot.processImagineUpscale(s, i, 2)
			case "imagine_upscale_3":
				bot.processImagineUpscale(s, i, 3)
			case "imagine_upscale_4":
				bot.processImagineUpscale(s, i, 4)
			case "imagine_variation_1":
				bot.processImagineVariation(s, i, 1)
			case "imagine_variation_2":
				bot.processImagineVariation(s, i, 2)
			case "imagine_variation_3":
				bot.processImagineVariation(s, i, 3)
			case "imagine_variation_4":
				bot.processImagineVariation(s, i, 4)
			default:
				log.Printf("Unknown message component '%v'", i.MessageComponentData().CustomID)
			}
		}
	})

	return bot, nil
}

func (b *botImpl) Start() {
	b.imagineQueue.StartPolling(b.botSession)

	err := b.teardown()
	if err != nil {
		log.Printf("Error tearing down bot: %v", err)
	}
}

func (b *botImpl) teardown() error {
	return b.botSession.Close()
}

func (b *botImpl) addImagineCommand() error {
	log.Printf("Adding command 'imagine'...")

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        "imagine",
		Description: "Ask the bot to imagine something",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "prompt",
				Description: "The text prompt to imagine",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", cmd.Name, err)

		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) processImagineReroll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeReroll,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm reimagining that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineUpscale(s *discordgo.Session, i *discordgo.InteractionCreate, upscaleIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeUpscale,
		InteractionIndex:   upscaleIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm upscaling that for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineVariation(s *discordgo.Session, i *discordgo.InteractionCreate, variationIndex int) {
	position, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Type:               imagine_queue.ItemTypeVariation,
		InteractionIndex:   variationIndex,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("I'm imagining more variations for you... You are currently #%d in line.", position),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var position int
	var queueError error
	var prompt string

	if option, ok := optionMap["prompt"]; ok {
		prompt = option.StringValue()

		position, queueError = b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
			Prompt:             prompt,
			Type:               imagine_queue.ItemTypeImagine,
			DiscordInteraction: i.Interaction,
		})
		if queueError != nil {
			log.Printf("Error adding imagine to queue: %v\n", queueError)
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		// Ignore type for now, they will be discussed in "responses"
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(
				"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \"%s\".",
				position,
				i.Member.User.ID,
				prompt),
		},
	})
}
