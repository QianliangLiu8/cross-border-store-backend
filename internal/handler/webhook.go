package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"digital-shop/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WebhookHandler struct {
	db     *gorm.DB
	orders *service.OrderService
	paypal *service.PayPalService
}

func NewWebhookHandler(db *gorm.DB, o *service.OrderService, pp *service.PayPalService) *WebhookHandler {
	return &WebhookHandler{db: db, orders: o, paypal: pp}
}

type paypalWebhook struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	Resource  json.RawMessage `json:"resource"`
}

type paypalCaptureResource struct {
	ID       string `json:"id"`
	CustomID string `json:"custom_id"`
	Status   string `json:"status"`
}

// HandlePayPal POST /api/webhook/paypal
func (h *WebhookHandler) HandlePayPal(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	// 把 body 回填，供 paypal SDK 验签
	c.Request.Body = io.NopCloser(bytesReader(body))

	if err := h.paypal.VerifyWebhook(c.Request.Context(), c.Request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var evt paypalWebhook
	if err := json.Unmarshal(body, &evt); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// 幂等记录
	log := &model.PayPalWebhookLog{Provider: "paypal", EventID: evt.ID, EventType: evt.EventType}
	if err := h.db.Create(log).Error; err != nil {
		// 唯一约束冲突 = 重复事件，直接 200 返回
		c.Status(http.StatusOK)
		return
	}

	switch evt.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		var res paypalCaptureResource
		_ = json.Unmarshal(evt.Resource, &res)
		if res.CustomID != "" {
			_, _ = h.orders.applyPaidState(c.Request.Context(), res.CustomID, res.ID)
			_ = h.orders.MarkDelivered(c.Request.Context(), res.CustomID)
		}
	}

	c.Status(http.StatusOK)
}

// small helper to avoid importing bytes at the top
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
