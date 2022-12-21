package png_info_extractor

type Extractor interface {
	ExtractDiffusionInfo() (*PNGInfo, error)
}
