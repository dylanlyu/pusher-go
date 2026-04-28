package beams

type publishResponse struct {
	PublishID string `json:"publishId"`
}

type errorResponse struct {
	Error       string `json:"error"`
	Description string `json:"description"`
}
