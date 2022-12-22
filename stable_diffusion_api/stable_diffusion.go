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
	Seed        int   `json:"seed"`
	AllSeeds    []int `json:"all_seeds"`
	AllSubseeds []int `json:"all_subseeds"`
}

type TextToImageResponse struct {
	Images   []string `json:"images"`
	Seeds    []int    `json:"seeds"`
	Subseeds []int    `json:"subseeds"`
}

type TextToImageRequest struct {
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	RestoreFaces      bool    `json:"restore_faces"`
	EnableHR          bool    `json:"enable_hr"`
	DenoisingStrength float64 `json:"denoising_strength"`
	BatchSize         int     `json:"batch_size"`
	Seed              int     `json:"seed"`
	Subseed           int     `json:"subseed"`
	SubseedStrength   float64 `json:"subseed_strength"`
	SamplerName       string  `json:"sampler_name"`
	CfgScale          float64 `json:"cfg_scale"`
	Steps             int     `json:"steps"`
	NIter             int     `json:"n_iter"`
}

func (api *apiImpl) TextToImage(req *TextToImageRequest) (*TextToImageResponse, error) {
	if req == nil {
		return nil, errors.New("missing request")
	}

	postURL := api.host + "/sdapi/v1/txt2img"

	//req := TextToImageRequest{
	//	Prompt: prompt,
	//	NegativePrompt: "ugly, tiling, poorly drawn hands, poorly drawn feet, poorly drawn face, out of frame, " +
	//		"mutation, mutated, extra limbs, extra legs, extra arms, disfigured, deformed, cross-eye, " +
	//		"body out of frame, blurry, bad art, bad anatomy, blurred, text, watermark, grainy",
	//	Width:             768,
	//	Height:            768,
	//	RestoreFaces:      true,
	//	EnableHR:          true,
	//	DenoisingStrength: 0.7,
	//	BatchSize:         1,
	//	Seed:              -1,
	//	Subseed:           -1,
	//	SubseedStrength:   0,
	//	SamplerName:       "Euler a",
	//	CfgScale:          7,
	//	Steps:             20,
	//	NIter:             4,
	//}

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
		Images:   respStruct.Images,
		Seeds:    infoStruct.AllSeeds,
		Subseeds: infoStruct.AllSubseeds,
	}, nil
}
