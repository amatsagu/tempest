package discord

// https://docs.discord.com/developers/interactions/receiving-and-responding#interaction-response-object-modal
type ResponseModalData struct {
	CustomID   string     `json:"custom_id"`
	Title      string     `json:"title"`
	Components Components `json:"components"`
}

func NewModalResponse(customID, title string, components ...Component) Response {
	return Response{
		Type: ResponseModal,
		Data: ResponseModalData{
			CustomID:   customID,
			Title:      title,
			Components: Components(components),
		},
	}
}
