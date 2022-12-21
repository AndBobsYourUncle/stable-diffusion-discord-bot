package discord_bot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/png_info_extractor"
	"stable_diffusion_bot/stable_diffusion_api"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	botSession         *discordgo.Session
	guildID            string
	imagineQueue       imagine_queue.Queue
	registeredCommands []*discordgo.ApplicationCommand
}

type Config struct {
	BotToken           string
	GuildID            string
	StableDiffusionAPI stable_diffusion_api.StableDiffusionAPI
}

func New(cfg Config) (Bot, error) {
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	if cfg.GuildID == "" {
		return nil, errors.New("missing guild ID")
	}

	if cfg.StableDiffusionAPI == nil {
		return nil, errors.New("missing stable diffusion API")
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

	imagineQueue, err := imagine_queue.New(imagine_queue.Config{
		BotSession:         botSession,
		StableDiffusionAPI: cfg.StableDiffusionAPI,
	})
	if err != nil {
		return nil, err
	}

	bot := &botImpl{
		botSession:         botSession,
		imagineQueue:       imagineQueue,
		registeredCommands: make([]*discordgo.ApplicationCommand, 0),
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
				bot.processImagineMessageComponent(s, i)
			default:
				log.Printf("Unknown message component '%v'", i.MessageComponentData().CustomID)
			}
		}
	})

	return bot, nil
}

func (b *botImpl) Start() {
	b.imagineQueue.StartPolling()

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

func (b *botImpl) processImagineMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("Message component interaction: %v", i.MessageComponentData())

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "I'm reimagining that for you...",
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}

	prompt := ""

	for _, attachment := range i.Message.Attachments {
		log.Printf("Message URL: %v", attachment.URL)

		response, pngErr := http.Get(attachment.URL)
		if pngErr != nil {
			log.Printf("Error getting image: %v", pngErr)

			return
		}

		defer response.Body.Close()

		attachmentReader := bufio.NewReader(response.Body)

		attachmentBytes, pngErr := io.ReadAll(attachmentReader)
		if pngErr != nil {
			log.Printf("Error reading attachment: %v", pngErr)
		}

		pngExtractor, pngErr := png_info_extractor.New(png_info_extractor.Config{PngData: attachmentBytes})
		if pngErr != nil {
			log.Printf("Error extracting PNG data: %v", pngErr)
		}

		pngInfo, pngErr := pngExtractor.ExtractDiffusionInfo()
		if pngErr != nil {
			log.Printf("Error extracting PNG data: %v", pngErr)
		}

		prompt = pngInfo.Prompt

		break
	}

	_, queueError := b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
		Prompt:             prompt,
		DiscordInteraction: i.Interaction,
	})
	if queueError != nil {
		log.Printf("Error adding imagine to queue: %v\n", queueError)
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
