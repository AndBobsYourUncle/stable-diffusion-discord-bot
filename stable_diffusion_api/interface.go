package stable_diffusion_api

type StableDiffusionAPI interface {
	TextToImage(prompt string) (*TextToImageResponse, error)
}
