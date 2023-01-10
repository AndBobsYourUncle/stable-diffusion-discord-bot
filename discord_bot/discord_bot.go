package discord_bot

import (
	"errors"
	"fmt"
	"log"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/imagine_queue"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type botImpl struct {
	developmentMode    bool
	botSession         *discordgo.Session
	guildID            string
	imagineQueue       imagine_queue.Queue
	registeredCommands []*discordgo.ApplicationCommand
	imagineCommand     string
	removeCommands     bool
}

type Config struct {
	DevelopmentMode bool
	BotToken        string
	GuildID         string
	ImagineQueue    imagine_queue.Queue
	ImagineCommand  string
	RemoveCommands  bool
}

func (b *botImpl) imagineCommandString() string {
	if b.developmentMode {
		return "dev_" + b.imagineCommand
	}

	return b.imagineCommand
}

func (b *botImpl) imagineSettingsCommandString() string {
	if b.developmentMode {
		return "dev_" + b.imagineCommand + "_settings"
	}

	return b.imagineCommand + "_settings"
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

	if cfg.ImagineCommand == "" {
		return nil, errors.New("missing imagine command")
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
		developmentMode:    cfg.DevelopmentMode,
		botSession:         botSession,
		imagineQueue:       cfg.ImagineQueue,
		registeredCommands: make([]*discordgo.ApplicationCommand, 0),
		imagineCommand:     cfg.ImagineCommand,
		removeCommands:     cfg.RemoveCommands,
	}

	err = bot.addImagineCommand()
	if err != nil {
		return nil, err
	}

	err = bot.addImagineSettingsCommand()
	if err != nil {
		return nil, err
	}

	botSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case bot.imagineCommandString():
				bot.processImagineCommand(s, i)
			case bot.imagineSettingsCommandString():
				bot.processImagineSettingsCommand(s, i)
			default:
				log.Printf("Unknown command '%v'", i.ApplicationCommandData().Name)
			}
		case discordgo.InteractionMessageComponent:
			switch customID := i.MessageComponentData().CustomID; {
			case customID == "imagine_reroll":
				bot.processImagineReroll(s, i)
			case strings.HasPrefix(customID, "imagine_upscale_"):
				interactionIndex := strings.TrimPrefix(customID, "imagine_upscale_")

				interactionIndexInt, intErr := strconv.Atoi(interactionIndex)
				if intErr != nil {
					log.Printf("Error parsing interaction index: %v", err)

					return
				}

				bot.processImagineUpscale(s, i, interactionIndexInt)
			case strings.HasPrefix(customID, "imagine_variation_"):
				interactionIndex := strings.TrimPrefix(customID, "imagine_variation_")

				interactionIndexInt, intErr := strconv.Atoi(interactionIndex)
				if intErr != nil {
					log.Printf("Error parsing interaction index: %v", err)

					return
				}

				bot.processImagineVariation(s, i, interactionIndexInt)
			case customID == "imagine_dimension_setting_menu":
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine dimension setting menu")

					return
				}

				sizes := strings.Split(i.MessageComponentData().Values[0], "_")

				width := sizes[0]
				height := sizes[1]

				widthInt, intErr := strconv.Atoi(width)
				if intErr != nil {
					log.Printf("Error parsing width: %v", err)

					return
				}

				heightInt, intErr := strconv.Atoi(height)
				if intErr != nil {
					log.Printf("Error parsing height: %v", err)

					return
				}

				bot.processImagineDimensionSetting(s, i, widthInt, heightInt)
			case customID == "imagine_batch_count_setting_menu":
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine batch count setting menu")

					return
				}

				batchCount := i.MessageComponentData().Values[0]

				batchCountInt, intErr := strconv.Atoi(batchCount)
				if intErr != nil {
					log.Printf("Error parsing batch count: %v", err)

					return
				}

				var batchSizeInt int

				// calculate the corresponding batch size
				switch batchCountInt {
				case 1:
					batchSizeInt = 4
				case 2:
					batchSizeInt = 2
				case 4:
					batchSizeInt = 1
				default:
					log.Printf("Unknown batch count: %v", batchCountInt)

					return
				}

				bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
			case customID == "imagine_batch_size_setting_menu":
				if len(i.MessageComponentData().Values) == 0 {
					log.Printf("No values for imagine batch count setting menu")

					return
				}

				batchSize := i.MessageComponentData().Values[0]

				batchSizeInt, intErr := strconv.Atoi(batchSize)
				if intErr != nil {
					log.Printf("Error parsing batch count: %v", err)

					return
				}

				var batchCountInt int

				// calculate the corresponding batch count
				switch batchSizeInt {
				case 1:
					batchCountInt = 4
				case 2:
					batchCountInt = 2
				case 4:
					batchCountInt = 1
				default:
					log.Printf("Unknown batch size: %v", batchSizeInt)

					return
				}

				bot.processImagineBatchSetting(s, i, batchCountInt, batchSizeInt)
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
	// Delete all commands added by the bot
	if b.removeCommands {
		log.Printf("Removing all commands added by bot...")

		for _, v := range b.registeredCommands {
			log.Printf("Removing command '%v'...", v.Name)

			err := b.botSession.ApplicationCommandDelete(b.botSession.State.User.ID, b.guildID, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

	return b.botSession.Close()
}

func (b *botImpl) addImagineCommand() error {
	log.Printf("Adding command '%s'...", b.imagineCommandString())

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineCommandString(),
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
		log.Printf("Error creating '%s' command: %v", b.imagineCommandString(), err)

		return err
	}

	b.registeredCommands = append(b.registeredCommands, cmd)

	return nil
}

func (b *botImpl) addImagineSettingsCommand() error {
	log.Printf("Adding command '%s'...", b.imagineSettingsCommandString())

	cmd, err := b.botSession.ApplicationCommandCreate(b.botSession.State.User.ID, b.guildID, &discordgo.ApplicationCommand{
		Name:        b.imagineSettingsCommandString(),
		Description: "Change the default settings for the imagine command",
	})
	if err != nil {
		log.Printf("Error creating '%s' command: %v", b.imagineSettingsCommandString(), err)

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

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(
				"I'm dreaming something up for you. You are currently #%d in line.\n<@%s> asked me to imagine \"%s\".",
				position,
				i.Member.User.ID,
				prompt),
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func settingsMessageComponents(settings *entities.DefaultSettings) []discordgo.MessageComponent {
	minValues := 1

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:  "imagine_dimension_setting_menu",
					MinValues: &minValues,
					MaxValues: 1,
					Options: []discordgo.SelectMenuOption{
						{
							Label:   "Size: 512x512",
							Value:   "512_512",
							Default: settings.Width == 512 && settings.Height == 512,
						},
						{
							Label:   "Size: 768x768",
							Value:   "768_768",
							Default: settings.Width == 768 && settings.Height == 768,
						},
					},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:  "imagine_batch_count_setting_menu",
					MinValues: &minValues,
					MaxValues: 1,
					Options: []discordgo.SelectMenuOption{
						{
							Label:   "Batch count: 1",
							Value:   "1",
							Default: settings.BatchCount == 1,
						},
						{
							Label:   "Batch count: 2",
							Value:   "2",
							Default: settings.BatchCount == 2,
						},
						{
							Label:   "Batch count: 4",
							Value:   "4",
							Default: settings.BatchCount == 4,
						},
					},
				},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:  "imagine_batch_size_setting_menu",
					MinValues: &minValues,
					MaxValues: 1,
					Options: []discordgo.SelectMenuOption{
						{
							Label:   "Batch size: 1",
							Value:   "1",
							Default: settings.BatchSize == 1,
						},
						{
							Label:   "Batch size: 2",
							Value:   "2",
							Default: settings.BatchSize == 2,
						},
						{
							Label:   "Batch size: 4",
							Value:   "4",
							Default: settings.BatchSize == 4,
						},
					},
				},
			},
		},
	}
}

func (b *botImpl) processImagineSettingsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	botSettings, err := b.imagineQueue.GetBotDefaultSettings()
	if err != nil {
		log.Printf("error getting default settings for settings command: %v", err)

		return
	}

	messageComponents := settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Title:      "Settings",
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineDimensionSetting(s *discordgo.Session, i *discordgo.InteractionCreate, height, width int) {
	botSettings, err := b.imagineQueue.UpdateDefaultDimensions(width, height)
	if err != nil {
		log.Printf("error updating default dimensions: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating default dimensions...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func (b *botImpl) processImagineBatchSetting(s *discordgo.Session, i *discordgo.InteractionCreate, batchCount, batchSize int) {
	botSettings, err := b.imagineQueue.UpdateDefaultBatch(batchCount, batchSize)
	if err != nil {
		log.Printf("error updating batch settings: %v", err)

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "Error updating batch settings...",
			},
		})
		if err != nil {
			log.Printf("Error responding to interaction: %v", err)
		}

		return
	}

	messageComponents := settingsMessageComponents(botSettings)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "Choose defaults settings for the imagine command:",
			Components: messageComponents,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
