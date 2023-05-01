package entities

import "time"

type ImageGeneration struct {
	ID                int64     `json:"id"`
	InteractionID     string    `json:"interaction_id"`
	MessageID         string    `json:"message_id"`
	MemberID          string    `json:"member_id"`
	SortOrder         int       `json:"sort_order"`
	Prompt            string    `json:"prompt"`
	NegativePrompt    string    `json:"negative_prompt"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
	RestoreFaces      bool      `json:"restore_faces"`
	EnableHR          bool      `json:"enable_hr"`
	HiresWidth        int       `json:"hires_width"`
	HiresHeight       int       `json:"hires_height"`
	DenoisingStrength float64   `json:"denoising_strength"`
	BatchCount        int       `json:"batch_count"`
	BatchSize         int       `json:"batch_size"`
	Seed              int       `json:"seed"`
	Subseed           int       `json:"subseed"`
	SubseedStrength   float64   `json:"subseed_strength"`
	SamplerName       string    `json:"sampler_name"`
	CfgScale          float64   `json:"cfg_scale"`
	Steps             int       `json:"steps"`
	Processed         bool      `json:"processed"`
	CreatedAt         time.Time `json:"created_at"`
}
