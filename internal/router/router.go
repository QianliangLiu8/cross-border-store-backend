package router

import (
	"digital-shop/internal/config"
	"digital-shop/internal/handler"
	"digital-shop/internal/middleware"
	"digital-shop/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(db *gorm.DB, cfg *config.Config, pp *service.PayPalService, tj *service.TaxJarService) *gin.Engine {
	r := gin.Default()

	orderSvc := service.NewOrderService(db, cfg, pp, tj)
	refundSvc := service.NewRefundService(db, pp)

	authH := handler.NewAuthHandler(db, cfg)
	orderH := handler.NewOrderHandler(orderSvc)
	webhookH := handler.NewWebhookHandler(db, orderSvc, pp)
	downloadH := handler.NewDownloadHandler(db, cfg)
	refundH := handler.NewRefundHandler(refundSvc)

	api := r.Group("/api")
	{
		api.POST("/auth/login", authH.Login)

		api.POST("/orders", orderH.Create)
		api.POST("/orders/:id/capture", orderH.Capture)

		api.POST("/webhook/paypal", webhookH.HandlePayPal)
		api.GET("/download", downloadH.Get)
	}

	admin := api.Group("/admin")
	admin.Use(middleware.AdminAuth(cfg))
	{
		admin.POST("/orders/:id/refund", refundH.Create)
		admin.POST("/refunds/:id/approve", refundH.Approve)
		admin.POST("/refunds/:id/reject", refundH.Reject)
	}

	return r
}
