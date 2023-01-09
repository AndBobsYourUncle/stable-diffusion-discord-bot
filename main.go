package main

import (
	"context"
	"flag"
	"log"
	"stable_diffusion_bot/databases/sqlite"
	"stable_diffusion_bot/discord_bot"
	"stable_diffusion_bot/imagine_queue"
	"stable_diffusion_bot/repositories/default_settings"
	"stable_diffusion_bot/repositories/image_generations"
	"stable_diffusion_bot/stable_diffusion_api"
)

// Bot parameters
var (
	guildID            = flag.String("guild", "", "Guild ID. If not passed - bot registers commands globally")
	botToken           = flag.String("token", "", "Bot access token")
	apiHost            = flag.String("host", "", "Host for the Automatic1111 API")
	imagineCommand     = flag.String("imagine", "imagine", "Imagine command name. Default is \"imagine\"")
	removeCommandsFlag = flag.Bool("remove", false, "Delete all commands when bot exits")
	devModeFlag        = flag.Bool("dev", false, "Start in development mode, using \"dev_\" prefixed commands instead")
)

func main() {
	flag.Parse()

	if guildID == nil || *guildID == "" {
		log.Fatalf("Guild ID flag is required")
	}

	if botToken == nil || *botToken == "" {
		log.Fatalf("Bot token flag is required")
	}

	if apiHost == nil || *apiHost == "" {
		log.Fatalf("API host flag is required")
	}

	if imagineCommand == nil || *imagineCommand == "" {
		log.Fatalf("Imagine command flag is required")
	}

	devMode := false

	if devModeFlag != nil && *devModeFlag {
		devMode = *devModeFlag

		log.Printf("Starting in development mode.. all commands prefixed with \"dev_\"")
	}

	removeCommands := false

	if removeCommandsFlag != nil && *removeCommandsFlag {
		removeCommands = *removeCommandsFlag
	}

	stableDiffusionAPI, err := stable_diffusion_api.New(stable_diffusion_api.Config{
		Host: *apiHost,
	})
	if err != nil {
		log.Fatalf("Failed to create Stable Diffusion API: %v", err)
	}

	ctx := context.Background()

	sqliteDB, err := sqlite.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create sqlite database: %v", err)
	}

	generationRepo, err := image_generations.NewRepository(&image_generations.Config{DB: sqliteDB})
	if err != nil {
		log.Fatalf("Failed to create image generation repository: %v", err)
	}

	defaultSettingsRepo, err := default_settings.NewRepository(&default_settings.Config{DB: sqliteDB})
	if err != nil {
		log.Fatalf("Failed to create default settings repository: %v", err)
	}

	imagineQueue, err := imagine_queue.New(imagine_queue.Config{
		StableDiffusionAPI:  stableDiffusionAPI,
		ImageGenerationRepo: generationRepo,
		DefaultSettingsRepo: defaultSettingsRepo,
	})
	if err != nil {
		log.Fatalf("Failed to create imagine queue: %v", err)
	}

	bot, err := discord_bot.New(discord_bot.Config{
		DevelopmentMode: devMode,
		BotToken:        *botToken,
		GuildID:         *guildID,
		ImagineQueue:    imagineQueue,
		ImagineCommand:  *imagineCommand,
		RemoveCommands:  removeCommands,
	})
	if err != nil {
		log.Fatalf("Error creating Discord bot: %v", err)
	}

	bot.Start()

	log.Println("Gracefully shutting down.")
}
