package stable_diffusion_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type apiImpl struct {
	host string
}

type Config struct {
	Host string
}

func New(cfg Config) (StableDiffusionAPI, error) {
	if cfg.Host == "" {
		return nil, errors.New("missing host")
	}

	return &apiImpl{
		host: cfg.Host,
	}, nil
}

type jsonTextToImageResponse struct {
	Images []string `json:"images"`
	Info   string   `json:"info"`
}

type jsonInfoResponse struct {
	Seed     int   `json:"seed"`
	AllSeeds []int `json:"all_seeds"`
}

type TextToImageResponse struct {
	Images []string `json:"images"`
	Seed   int      `json:"seed"`
	Seeds  []int    `json:"all_seeds"`
}

type textToImageRequest struct {
	Prompt         string `json:"prompt"`
	Seed           int    `json:"seed"`
	SamplerName    string `json:"sampler_name"`
	BatchSize      int    `json:"batch_size"`
	NIter          int    `json:"n_iter"`
	Steps          int    `json:"steps"`
	CfgScale       int    `json:"cfg_scale"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	RestoreFaces   bool   `json:"restore_faces"`
	NegativePrompt string `json:"negative_prompt"`
	SamplerIndex   string `json:"sampler_index"`
}

func (api *apiImpl) TextToImage(prompt string) (*TextToImageResponse, error) {
	postURL := api.host + "/sdapi/v1/txt2img"

	req := textToImageRequest{
		Prompt:       prompt,
		Seed:         -1,
		SamplerName:  "Euler a",
		BatchSize:    1,
		NIter:        4,
		Steps:        20,
		CfgScale:     7,
		Width:        768,
		Height:       768,
		RestoreFaces: true,
		NegativePrompt: "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
			"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
			"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy",
		SamplerIndex: "Euler a",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", postURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)

	respStruct := &jsonTextToImageResponse{}

	err = json.Unmarshal(body, respStruct)
	if err != nil {
		return nil, err
	}

	infoStruct := &jsonInfoResponse{}

	err = json.Unmarshal([]byte(respStruct.Info), infoStruct)
	if err != nil {
		return nil, err
	}

	return &TextToImageResponse{
		Images: respStruct.Images,
		Seed:   infoStruct.Seed,
		Seeds:  infoStruct.AllSeeds,
	}, nil
}
