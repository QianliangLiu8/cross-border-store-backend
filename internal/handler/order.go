package handler

import (
	"net/http"

	"digital-shop/internal/service"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orders *service.OrderService
}

func NewOrderHandler(o *service.OrderService) *OrderHandler {
	return &OrderHandler{orders: o}
}

// Create POST /api/orders
func (h *OrderHandler) Create(c *gin.Context) {
	var req service.CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.orders.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Capture POST /api/orders/:id/capture
func (h *OrderHandler) Capture(c *gin.Context) {
	orderID := c.Param("id")
	order, err := h.orders.Capture(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"order":    order,
		"download": h.orders.BuildDownloadURLs(order),
	})
}
