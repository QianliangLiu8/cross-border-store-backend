package handler

import (
	"net/http"
	"strconv"

	"digital-shop/internal/service"

	"github.com/gin-gonic/gin"
)

type RefundHandler struct {
	refunds *service.RefundService
}

func NewRefundHandler(r *service.RefundService) *RefundHandler {
	return &RefundHandler{refunds: r}
}

// Create POST /api/admin/orders/:id/refund
func (h *RefundHandler) Create(c *gin.Context) {
	orderID := c.Param("id")
	r, err := h.refunds.CreatePending(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, r)
}

// Approve POST /api/admin/refunds/:id/approve
func (h *RefundHandler) Approve(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	adminID, _ := c.Get("admin_id")
	r, err := h.refunds.Approve(c.Request.Context(), uint(id), toString(adminID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, r)
}

type rejectReq struct {
	Reason string `json:"reason"`
}

// Reject POST /api/admin/refunds/:id/reject
func (h *RefundHandler) Reject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req rejectReq
	_ = c.ShouldBindJSON(&req)
	adminID, _ := c.Get("admin_id")
	r, err := h.refunds.Reject(c.Request.Context(), uint(id), toString(adminID), req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, r)
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
