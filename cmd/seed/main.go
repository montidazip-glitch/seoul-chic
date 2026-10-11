package main

import (
	"log"
	"time"

	"kfry_shop/internal/config"
	"kfry_shop/internal/database"
	"kfry_shop/internal/models"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg.DBDSN)
	defer db.Close()
	database.RunMigrations(db, "migrations")

	// 1. Create default admin account
	if !models.EmailExists(db, "admin@seoulchic.com") {
		if err := models.CreateUser(db, "Admin Seoul Chic", "admin@seoulchic.com", "admin1234", "admin"); err != nil {
			log.Fatalf("Failed to create admin: %v", err)
		}
		log.Println("✅ Admin created: admin@seoulchic.com / admin1234")
	} else {
		log.Println("Admin already exists, skipping user creation.")
	}

	// 2. Create sample products if none exist
	var productCount int
	db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&productCount)
	if productCount == 0 {
		products := []models.Product{
			{Name: "ไก่ทอดดั้งเดิม (Original)", Description: "ไก่ทอดกรอบสูตรต้นตำรับจากโซล รสชาติเข้มข้น", Price: 159, Category: "ไก่ทอด", Stock: 50, ImagePath: "/static/images/uploads/original_chicken.jpg", Status: "available"},
			{Name: "ไก่ทอดซอสพริก (Spicy)", Description: "ไก่ทอดราดซอสพริกเกาหลี หอมหวานเผ็ดกำลังดี", Price: 179, Category: "ไก่ทอด", Stock: 40, ImagePath: "/static/images/uploads/spicy_chicken.jpg", Status: "available"},
			{Name: "ไก่ทอดซอสน้ำผึ้งกระเทียม", Description: "ไก่ทอดราดซอสน้ำผึ้งกระเทียมหอมๆ", Price: 189, Category: "ไก่ทอด", Stock: 35, ImagePath: "/static/images/uploads/honey_garlic_chicken.jpg", Status: "available"},
			{Name: "ชีสบอล (Cheese Ball 5 ชิ้น)", Description: "ชีสบอลสอดไส้ชีสเยิ้ม ทอดกรอบนอก", Price: 79, Category: "ของทานเล่น", Stock: 60, ImagePath: "/static/images/uploads/cheese_ball.jpg", Status: "available"},
			{Name: "เฟรนช์ฟรายส์", Description: "มันฝรั่งทอดกรอบ โรยเกลือทะเล", Price: 59, Category: "ของทานเล่น", Stock: 80, ImagePath: "/static/images/uploads/french_fries.jpg", Status: "available"},
			{Name: "โคลาเย็น", Description: "โคล่าเย็นชื่นใจ", Price: 35, Category: "เครื่องดื่ม", Stock: 100, ImagePath: "/static/images/uploads/cola.jpg", Status: "available"},
			{Name: "ชาเย็นเกาหลี", Description: "ชาเขียวนมเย็นสูตรเกาหลี", Price: 55, Category: "เครื่องดื่ม", Stock: 80, ImagePath: "/static/images/uploads/matcha_latte.jpg", Status: "available"},
		}
		for _, p := range products {
			pp := p
			if err := models.CreateProduct(db, &pp); err != nil {
				log.Printf("Warning: could not create product %s: %v", p.Name, err)
			} else {
				log.Printf("✅ Product created: %s", p.Name)
			}
		}
	}

	// 3. Create sample promotions if none exist
	var promoCount int
	db.QueryRow(`SELECT COUNT(*) FROM promotions`).Scan(&promoCount)
	if promoCount == 0 {
		now := time.Now()
		promotions := []*models.Promotion{
			{
				Code:           "KFRY20",
				Title:          "ลดกระหน่ำ 20% เอาใจสายไก่ทอด",
				Description:    "รับส่วนลดทันที 20% เมื่อสั่งซื้อครบ 200 บาท",
				DiscountType:   "percentage",
				DiscountValue:  20,
				MinOrderAmount: 200,
				StartDate:      now.AddDate(0, 0, -1),
				EndDate:        now.AddDate(0, 1, 0),
				IsActive:       true,
			},
			{
				Code:           "SEOUL50",
				Title:          "ส่วนลดพิเศษ 50 บาท",
				Description:    "ลดทันที 50 บาท ทุกออเดอร์ขั้นต่ำ 150 บาท",
				DiscountType:   "fixed",
				DiscountValue:  50,
				MinOrderAmount: 150,
				StartDate:      now.AddDate(0, 0, -1),
				EndDate:        now.AddDate(0, 1, 0),
				IsActive:       true,
			},
			{
				Code:           "WELCOME10",
				Title:          "ต้อนรับลูกค้าใหม่ ลด 10%",
				Description:    "ลด 10% ไม่มีขั้นต่ำในการสั่งซื้อ",
				DiscountType:   "percentage",
				DiscountValue:  10,
				MinOrderAmount: 0,
				StartDate:      now.AddDate(0, 0, -1),
				EndDate:        now.AddDate(0, 2, 0),
				IsActive:       true,
			},
		}

		for _, pr := range promotions {
			if err := models.CreatePromotion(db, pr); err != nil {
				log.Printf("Warning: could not create promotion %s: %v", pr.Code, err)
			} else {
				log.Printf("✅ Promotion created: %s (%s)", pr.Code, pr.FormattedDiscount())
			}
		}
	}

	log.Println("🎉 Seed complete!")
}
