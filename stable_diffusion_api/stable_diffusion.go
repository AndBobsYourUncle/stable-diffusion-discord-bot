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
	Seed int `json:"seed"`
}

type TextToImageResponse struct {
	Images []string `json:"images"`
	Seed   int      `json:"seed"`
}

func (api *apiImpl) TextToImage(prompt string) (*TextToImageResponse, error) {
	postURL := api.host + "/sdapi/v1/txt2img"

	var jsonData = []byte(`{
	  "prompt": "` + prompt + `",
	  "seed": -1,
	  "sampler_name": "Euler a",
	  "batch_size": 1,
	  "steps": 20,
	  "cfg_scale": 7,
	  "width": 768,
	  "height": 768,
	  "restore_faces": true,
	  "negative_prompt": "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy",
	  "sampler_index": "Euler a"
	}`)

	request, err := http.NewRequest("POST", postURL, bytes.NewBuffer(jsonData))
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

	return &TextToImageResponse{Images: respStruct.Images, Seed: infoStruct.Seed}, nil
}
