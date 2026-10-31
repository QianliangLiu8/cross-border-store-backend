package main

import (
	"log"
	"os"

	"digital-shop/internal/config"
	"digital-shop/internal/model"
	"digital-shop/internal/router"
	"digital-shop/internal/service"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("mysql connect failed: %v", err)
	}

	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	if err := seed(cfg, db); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	pp, err := service.NewPayPalService(cfg)
	if err != nil {
		log.Fatalf("paypal init failed: %v", err)
	}
	tj := service.NewTaxJarService(cfg.TaxJarAPIKey)

	r := router.New(db, cfg, pp, tj)
	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
	_ = os.Stdout
}

func seed(cfg *config.Config, db *gorm.DB) error {
	var count int64
	db.Model(&model.Admin{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte(cfg.AdminPass), bcrypt.DefaultCost)
		if err := db.Create(&model.Admin{Username: cfg.AdminUser, PasswordHash: string(hash)}).Error; err != nil {
			return err
		}
		log.Printf("seeded admin user: %s", cfg.AdminUser)
	}

	db.Model(&model.ProductVariant{}).Count(&count)
	if count == 0 {
		p := model.Product{ID: "P-EARRING", Name: "合金耳环设计图纸源文件", TaxCode: "20010", Active: true}
		db.Create(&p)
		db.Create(&model.ProductVariant{
			ProductID: "P-EARRING", SKU: "EARRING-HOOK", VariantName: "耳钩款",
			PriceCents: 1200, FileKey: "earring_hook.zip", FileName: "合金耳环_耳钩款.zip", FileSize: 1024,
		})
		db.Create(&model.ProductVariant{
			ProductID: "P-EARRING", SKU: "EARRING-CLIP", VariantName: "耳夹款",
			PriceCents: 1200, FileKey: "earring_clip.zip", FileName: "合金耳环_耳夹款.zip", FileSize: 1024,
		})
		log.Println("seeded sample product + variants")
	}
	return nil
}
