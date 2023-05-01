package sqlite

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

const dbFile string = "sd_discord_bot.sqlite"

const getCurrentMigration string = `PRAGMA user_version;`
const setCurrentMigration string = `PRAGMA user_version = ?;`

const createGenerationTableIfNotExistsQuery string = `
CREATE TABLE IF NOT EXISTS image_generations (
id INTEGER NOT NULL PRIMARY KEY,
interaction_id TEXT NOT NULL,
message_id TEXT NOT NULL,
member_id TEXT NOT NULL,
sort_order INTEGER NOT NULL,
prompt TEXT NOT NULL,
negative_prompt TEXT NOT NULL,
width INTEGER NOT NULL,
height INTEGER NOT NULL,
restore_faces INTEGER NOT NULL,
enable_hr INTEGER NOT NULL,
denoising_strength REAL NOT NULL,
batch_size INTEGER NOT NULL,
seed INTEGER NOT NULL,
subseed INTEGER NOT NULL,
subseed_strength REAL NOT NULL,
sampler_name TEXT NOT NULL,
cfg_scale REAL NOT NULL,
steps INTEGER NOT NULL,
processed INTEGER NOT NULL,
created_at DATETIME NOT NULL
);`

const createInteractionIndexIfNotExistsQuery string = `
CREATE INDEX IF NOT EXISTS generation_interaction_index 
ON image_generations(interaction_id);
`

const createMessageIndexIfNotExistsQuery string = `
CREATE INDEX IF NOT EXISTS generation_interaction_index 
ON image_generations(message_id);
`

const addHiresFirstPassDimensionColumnsQuery string = `
ALTER TABLE image_generations ADD COLUMN firstphase_width INTEGER NOT NULL DEFAULT 0;
ALTER TABLE image_generations ADD COLUMN firstphase_height INTEGER NOT NULL DEFAULT 0;
`

const dropHiresFirstPassDimensionColumnsQuery string = `
ALTER TABLE image_generations DROP COLUMN firstphase_width;
ALTER TABLE image_generations DROP COLUMN firstphase_height;
`

const addHiresResizeColumnsQuery string = `
ALTER TABLE image_generations ADD COLUMN hires_width INTEGER NOT NULL DEFAULT 0;
ALTER TABLE image_generations ADD COLUMN hires_height INTEGER NOT NULL DEFAULT 0;
`

const createDefaultSettingsTableIfNotExistsQuery string = `
CREATE TABLE IF NOT EXISTS default_settings (
member_id TEXT NOT NULL PRIMARY KEY,
width INTEGER NOT NULL,
height INTEGER NOT NULL
);`

const addSettingsBatchColumnsQuery string = `
ALTER TABLE default_settings ADD COLUMN batch_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE default_settings ADD COLUMN batch_size INTEGER NOT NULL DEFAULT 0;
`

const addGenerationBatchSizeColumnQuery string = `
ALTER TABLE image_generations ADD COLUMN batch_count INTEGER NOT NULL DEFAULT 0;
`

type migration struct {
	migrationName  string
	migrationQuery string
}

var migrations = []migration{
	{migrationName: "create generation table", migrationQuery: createGenerationTableIfNotExistsQuery},
	{migrationName: "add generation interaction index", migrationQuery: createInteractionIndexIfNotExistsQuery},
	{migrationName: "add generation message index", migrationQuery: createMessageIndexIfNotExistsQuery},
	{migrationName: "add hires firstpass columns", migrationQuery: addHiresFirstPassDimensionColumnsQuery},
	{migrationName: "drop hires firstpass columns", migrationQuery: dropHiresFirstPassDimensionColumnsQuery},
	{migrationName: "add hires resize columns", migrationQuery: addHiresResizeColumnsQuery},
	{migrationName: "create default settings table", migrationQuery: createDefaultSettingsTableIfNotExistsQuery},
	{migrationName: "add settings batch columns", migrationQuery: addSettingsBatchColumnsQuery},
	{migrationName: "add generation batch count column", migrationQuery: addGenerationBatchSizeColumnQuery},
}

func New(ctx context.Context) (*sql.DB, error) {
	filename, err := DBFilename()
	if err != nil {
		return nil, err
	}

	err = touchDBFile(filename)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return nil, err
	}

	err = migrate(ctx, db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	var currentMigration int

	row := db.QueryRowContext(ctx, getCurrentMigration)

	err := row.Scan(&currentMigration)
	if err != nil {
		return err
	}

	requiredMigration := len(migrations)

	log.Printf("Current DB version: %v, required DB version: %v\n", currentMigration, requiredMigration)

	if currentMigration < requiredMigration {
		for migrationNum := currentMigration + 1; migrationNum <= requiredMigration; migrationNum++ {
			err = execMigration(ctx, db, migrationNum)
			if err != nil {
				log.Printf("Error running migration %v '%v'\n", migrationNum, migrations[migrationNum-1].migrationName)

				return err
			}
		}
	}

	return nil
}

func execMigration(ctx context.Context, db *sql.DB, migrationNum int) error {
	log.Printf("Running migration %v '%v'\n", migrationNum, migrations[migrationNum-1].migrationName)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	//nolint
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, migrations[migrationNum-1].migrationQuery)
	if err != nil {
		return err
	}

	setQuery := strings.Replace(setCurrentMigration, "?", strconv.Itoa(migrationNum), 1)

	_, err = tx.ExecContext(ctx, setQuery)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func DBFilename() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return dir + "/" + dbFile, nil
}

func touchDBFile(filename string) error {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		file, createErr := os.Create(filename)
		if createErr != nil {
			return createErr
		}

		closeErr := file.Close()
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}
