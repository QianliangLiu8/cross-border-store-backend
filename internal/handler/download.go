package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"digital-shop/internal/config"
	"digital-shop/internal/model"
	"digital-shop/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DownloadHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewDownloadHandler(db *gorm.DB, cfg *config.Config) *DownloadHandler {
	return &DownloadHandler{db: db, cfg: cfg}
}

// Get GET /api/download?order_id=&sku=&expires=&sig=
func (h *DownloadHandler) Get(c *gin.Context) {
	orderID := c.Query("order_id")
	sku := c.Query("sku")
	sig := c.Query("sig")

	var expires int64
	if _, err := fmt.Sscanf(c.Query("expires"), "%d", &expires); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expires"})
		return
	}

	if err := util.VerifyDownload(h.cfg.DownloadSecret, orderID, sku, expires, sig); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// 确认订单已支付，且该 sku 属于订单
	var item model.OrderItem
	if err := h.db.Where("order_id = ? AND variant_sku = ?", orderID, sku).First(&item).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "order item not found"})
		return
	}
	var order model.Order
	if err := h.db.Where("id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "order not found"})
		return
	}
	if order.Status == "PENDING_PAYMENT" || order.Status == "CANCELLED" {
		c.JSON(http.StatusForbidden, gin.H{"error": "order not paid"})
		return
	}

	var variant model.ProductVariant
	if err := h.db.Where("sku = ?", sku).First(&variant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "variant not found"})
		return
	}

	// 记录下载日志
	h.db.Create(&model.DownloadLog{
		OrderID:    orderID,
		VariantSKU: sku,
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
	})

	filePath := filepath.Join(h.cfg.FileStoragePath, variant.FileKey)
	if _, err := os.Stat(filePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file missing"})
		return
	}
	c.FileAttachment(filePath, variant.FileName)
}
