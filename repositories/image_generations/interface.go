package image_generations

import (
	"context"
	"stable_diffusion_bot/entities"
)

type Repository interface {
	Create(ctx context.Context, generation *entities.ImageGeneration) (*entities.ImageGeneration, error)
	GetByMessage(ctx context.Context, messageID string) (*entities.ImageGeneration, error)
}
