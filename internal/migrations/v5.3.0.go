package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V5_3_0 adds autoresponder campaign type and related tables.
func V5_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	// Add 'autoresponder' to campaign_type enum.
	_, err := db.Exec(`
		DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'autoresponder' AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'campaign_type')) THEN
				ALTER TYPE campaign_type ADD VALUE 'autoresponder';
			END IF;
		END $$;
	`)
	if err != nil {
		return err
	}

	// Create autoresponder_history table to track sent autoresponders.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS autoresponder_history (
			id BIGSERIAL PRIMARY KEY,
			campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
			list_id INTEGER NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
			sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(campaign_id, subscriber_id, list_id)
		);
		CREATE INDEX IF NOT EXISTS idx_ar_history_lookup ON autoresponder_history(campaign_id, subscriber_id, list_id);
	`)
	if err != nil {
		return err
	}

	// Add ar_trigger_on_confirm column to campaigns table.
	_, err = db.Exec(`
		ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS ar_trigger_on_confirm BOOLEAN NOT NULL DEFAULT true;
	`)
	if err != nil {
		return err
	}

	return nil
}
