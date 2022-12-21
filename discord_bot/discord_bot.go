package discord_bot

import (
	"fmt"
	"log"
	"stable_diffusion_bot/imagine_queue"
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
	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err = botSession.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
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

	bot.botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.ApplicationCommandData().Name {
		case "imagine":
			bot.processImagineCommand(s, i)
		default:
			log.Printf("Unknown command '%v'", i.ApplicationCommandData().Name)
		}
	})

	log.Println("Adding commands...")

	err = bot.addImagineCommand()
	if err != nil {
		return nil, err
	}

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
	log.Println("Removing commands...")

	for _, v := range b.registeredCommands {
		err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.guildID, v.ID)
		if err != nil {
			log.Printf("Error deleting '%v' command: %v", v.Name, err)
		}
	}

	return b.botSession.Close()
}

func (b *botImpl) addImagineCommand() error {
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

func (b *botImpl) processImagineCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Access options in the order provided by the user.
	options := i.ApplicationCommandData().Options

	// Or convert the slice into a map
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	var position int
	var queueError error
	var prompt string

	// Get the value from the option map.
	// When the option exists, ok = true
	if option, ok := optionMap["prompt"]; ok {
		prompt = option.StringValue()

		position, queueError = b.imagineQueue.AddImagine(&imagine_queue.QueueItem{
			Prompt:             prompt,
			DiscordInteraction: i.Interaction,
		})
		if queueError != nil {
			log.Printf("Cannot add imagine to queue: %v\n", queueError)
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
