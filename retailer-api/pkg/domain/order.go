package domain

// OrderStatus определяет состояния заказа
type OrderStatus string

const (
	StatusNew       OrderStatus = "NEW"
	StatusAccepted  OrderStatus = "ACCEPTED"
	StatusAssembled OrderStatus = "ASSEMBLED"
	StatusPaid      OrderStatus = "PAID"
	StatusShipped   OrderStatus = "SHIPPED"
	StatusCompleted OrderStatus = "COMPLETED"
)

// Order представляет заказ
type Order struct {
	ID      string                 `json:"id"`
	Status  OrderStatus            `json:"status"`
	Payload map[string]interface{} `json:"payload"`
}
