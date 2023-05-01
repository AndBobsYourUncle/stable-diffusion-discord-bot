package default_settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"stable_diffusion_bot/clock"
	"stable_diffusion_bot/entities"
	"stable_diffusion_bot/repositories"
)

const upsertSetting string = `
INSERT OR REPLACE INTO default_settings (member_id, width, height, batch_count, batch_size) VALUES (?, ?, ?, ?, ?);
`

const getSettingByMemberID string = `
SELECT member_id, width, height, batch_count, batch_size FROM default_settings WHERE member_id = ?;
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

func (repo *sqliteRepo) Upsert(ctx context.Context, setting *entities.DefaultSettings) (*entities.DefaultSettings, error) {
	_, err := repo.dbConn.ExecContext(ctx, upsertSetting,
		setting.MemberID, setting.Width, setting.Height, setting.BatchCount, setting.BatchSize)
	if err != nil {
		return nil, err
	}

	return setting, nil
}

func (repo *sqliteRepo) GetByMemberID(ctx context.Context, memberID string) (*entities.DefaultSettings, error) {
	var setting entities.DefaultSettings

	err := repo.dbConn.QueryRowContext(ctx, getSettingByMemberID, memberID).Scan(
		&setting.MemberID, &setting.Width, &setting.Height, &setting.BatchCount, &setting.BatchSize)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.NewNotFoundError(fmt.Sprintf("default setting for member ID %s", memberID))
		}

		return nil, err
	}

	return &setting, nil
}
