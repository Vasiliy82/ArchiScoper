package handler

import (
	"net/http"

	"github.com/Vasiliy82/ArchiScoper/retailer-api/internal/usecase"
	"github.com/Vasiliy82/ArchiScoper/retailer-api/pkg/tracing"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderUC *usecase.OrderUseCase
}

func NewOrderHandler(orderUC *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{orderUC: orderUC}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	// Используем нашу библиотеку трейсинга вместо raw OpenTelemetry API
	ctx, span := tracing.StartPresentation(c.Request.Context(), "CreateOrder", tracing.SubLayerHTTP)
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
