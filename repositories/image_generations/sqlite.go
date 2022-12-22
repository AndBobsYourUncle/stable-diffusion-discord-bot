package image_generations

import (
	"context"
	"database/sql"
	"errors"
	"stable_diffusion_bot/clock"
	"stable_diffusion_bot/entities"
)

const insertGenerationQuery string = `
INSERT INTO image_generations (interaction_id, member_id, sort_order, prompt, negative_prompt, width, height, restore_faces, enable_hr, denoising_strength, batch_size, seed, subseed, subseed_strength, sampler_name, cfg_scale, steps, processed, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`

const getGenerationByInteractionIDQuery string = `
SELECT id, interaction_id, member_id, sort_order, prompt, negative_prompt, width, height, restore_faces, enable_hr, denoising_strength, batch_size, seed, subseed, subseed_strength, sampler_name, cfg_scale, steps, processed, created_at FROM image_generations WHERE interaction_id = ?;
`

type sqliteRepo struct {
	dbConn *sql.DB
	clock  clock.Clock
}

type Config struct {
	DB *sql.DB
}

func NewRepository(cfg *Config) (Repository, error) {
	if cfg.DB == nil {
		return nil, errors.New("missing DB parameter")
	}

	newRepo := &sqliteRepo{
		dbConn: cfg.DB,
		clock:  clock.NewClock(),
	}

	return newRepo, nil
}

func (repo *sqliteRepo) Create(ctx context.Context, generation *entities.ImageGeneration) (*entities.ImageGeneration, error) {
	generation.CreatedAt = repo.clock.Now()

	res, err := repo.dbConn.ExecContext(ctx, insertGenerationQuery,
		generation.InteractionID, generation.MemberID, generation.SortOrder, generation.Prompt,
		generation.NegativePrompt, generation.Width, generation.Height, generation.RestoreFaces,
		generation.EnableHR, generation.DenoisingStrength, generation.BatchSize, generation.Seed, generation.Subseed,
		generation.SubseedStrength, generation.SamplerName, generation.CfgScale, generation.Steps, generation.Processed, generation.CreatedAt)
	if err != nil {
		return nil, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	generation.ID = lastID

	return generation, nil
}

func (repo *sqliteRepo) GetByInteraction(ctx context.Context, interactionID string) (*entities.ImageGeneration, error) {
	var generation entities.ImageGeneration

	err := repo.dbConn.QueryRowContext(ctx, getGenerationByInteractionIDQuery, interactionID).Scan(
		&generation.ID, &generation.InteractionID, &generation.MemberID, &generation.SortOrder, &generation.Prompt,
		&generation.NegativePrompt, &generation.Width, &generation.Height, &generation.RestoreFaces,
		&generation.EnableHR, &generation.DenoisingStrength, &generation.BatchSize, &generation.Seed, &generation.Subseed,
		&generation.SubseedStrength, &generation.SamplerName, &generation.CfgScale, &generation.Steps, &generation.Processed, &generation.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &generation, nil
}
