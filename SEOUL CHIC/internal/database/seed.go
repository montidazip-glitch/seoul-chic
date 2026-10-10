package database

import (
	"database/sql"
	"log"
	"time"

	"kfry_shop/internal/models"
)

// SeedDefaultData ensures default admin, sample products, and promotions exist.
func SeedDefaultData(db *sql.DB) {
	// 1. Ensure default admin and demo customer account exist
	if !models.EmailExists(db, "admin@seoulchic.com") {
		if err := models.CreateUser(db, "Admin Seoul Chic", "admin@seoulchic.com", "admin1234", "admin"); err != nil {
			log.Printf("[seed] Warning: Failed to create default admin: %v", err)
		} else {
			log.Println("✅ [seed] Default Admin created: admin@seoulchic.com / admin1234")
		}
	}

	if !models.EmailExists(db, "customer@seoulchic.com") {
		if err := models.CreateUser(db, "ลูกค้าทดสอบ (Demo Customer)", "customer@seoulchic.com", "customer1234", "customer"); err != nil {
			log.Printf("[seed] Warning: Failed to create demo customer: %v", err)
		} else {
			log.Println("✅ [seed] Demo Customer created: customer@seoulchic.com / customer1234")
		}
	}

	// 2. Sample products if empty
	var productCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&productCount); err == nil && productCount == 0 {
		products := []models.Product{
			{Name: "ไก่ทอดดั้งเดิม (Original)", Description: "ไก่ทอดกรอบสูตรต้นตำรับจากโซล รสชาติเข้มข้น กรอบนอกนุ่มใน", Price: 159, Category: "ไก่ทอด", Stock: 50, Status: "available"},
			{Name: "ไก่ทอดซอสพริก (Spicy Yangnyeom)", Description: "ไก่ทอดราดซอสพริกเกาหลีเข้มข้น หอมหวานเผ็ดกำลังดี", Price: 179, Category: "ไก่ทอด", Stock: 40, Status: "available"},
			{Name: "ไก่ทอดซอสน้ำผึ้งกระเทียม (Honey Garlic)", Description: "ไก่ทอดราดซอสน้ำผึ้งแท้และกระเทียมเจียวหอมกรุ่น", Price: 189, Category: "ไก่ทอด", Stock: 35, Status: "available"},
			{Name: "ชีสบอล (Cheese Ball 5 ชิ้น)", Description: "ชีสบอลสอดไส้มอสซาเรลลาชีสเยิ้ม ทอดสีทองกรอบนอก", Price: 79, Category: "ของทานเล่น", Stock: 60, Status: "available"},
			{Name: "เฟรนช์ฟรายส์", Description: "มันฝรั่งแท่งทอดกรอบ โรยเกลือทะเล เสิร์ฟร้อนๆ", Price: 59, Category: "ของทานเล่น", Stock: 80, Status: "available"},
			{Name: "โคล่าเย็น", Description: "เครื่องดื่มน้ำอัดลมเย็นสดชื่น", Price: 35, Category: "เครื่องดื่ม", Stock: 100, Status: "available"},
			{Name: "ชาเขียวนมเกาหลี", Description: "ชาเขียวนมเย็นสไตล์คาเฟ่เกาหลี หอมละมุน", Price: 55, Category: "เครื่องดื่ม", Stock: 80, Status: "available"},
		}
		for _, p := range products {
			pp := p
			if err := models.CreateProduct(db, &pp); err != nil {
				log.Printf("[seed] Warning: could not create product %s: %v", p.Name, err)
			}
		}
		log.Println("✅ [seed] Initial sample products loaded.")
	}

	// 3. Sample promotions if empty
	var promoCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM promotions`).Scan(&promoCount); err == nil && promoCount == 0 {
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
				log.Printf("[seed] Warning: could not create promotion %s: %v", pr.Code, err)
			}
		}
		log.Println("✅ [seed] Initial promotions loaded.")
	}
}
