package handler

import (
	"net/http"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/usecase"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

type OrderHandler struct {
	orderUC *usecase.OrderUseCase
}

func NewOrderHandler(orderUC *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{orderUC: orderUC}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	tracer := otel.Tracer("handler")
	ctx, span := tracer.Start(c.Request.Context(), "CreateOrderHandler")
	defer span.End()

	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	orderID, err := h.orderUC.CreateOrder(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process order"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"order_id": orderID})
}
