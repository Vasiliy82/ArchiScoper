package domain

type Order struct {
	ID      string                 `json:"order_id"`
	Payload map[string]interface{} `json:"payload"`
}
