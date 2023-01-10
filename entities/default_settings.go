package entities

type DefaultSettings struct {
	MemberID   string `json:"member_id"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	BatchCount int    `json:"batch_count"`
	BatchSize  int    `json:"batch_size"`
}
