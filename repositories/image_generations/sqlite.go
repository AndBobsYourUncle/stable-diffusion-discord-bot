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

func (repo *sqliteRepo) Create(ctx context.Context, note *entities.ImageGeneration) (*entities.ImageGeneration, error) {
	note.CreatedAt = repo.clock.Now()

	res, err := repo.dbConn.ExecContext(ctx, insertGenerationQuery,
		note.InteractionID, note.MemberID, note.SortOrder, note.Prompt,
		note.NegativePrompt, note.Width, note.Height, note.RestoreFaces,
		note.EnableHR, note.DenoisingStrength, note.BatchSize, note.Seed, note.Subseed,
		note.SubseedStrength, note.SamplerName, note.CfgScale, note.Steps, note.Processed, note.CreatedAt)
	if err != nil {
		return nil, err
	}

	lastID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	note.ID = lastID

	return note, nil
}

func (repo *sqliteRepo) Get(ctx context.Context, id int64) (*entities.ImageGeneration, error) {
	return nil, errors.New("not implemented")
}
