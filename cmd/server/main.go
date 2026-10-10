package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"kfry_shop/internal/config"
	"kfry_shop/internal/database"
	"kfry_shop/internal/handlers"
	"kfry_shop/internal/middleware"
	"kfry_shop/internal/session"
)

func main() {
	// ── 1. Load config ──────────────────────────────────────────────────────
	cfg := config.Load()

	// ── 2. Connect to DB & run migrations ───────────────────────────────────
	db := database.Connect(cfg.DBDSN)
	defer db.Close()
	database.RunMigrations(db, "migrations")
	database.SeedDefaultData(db)

	// ── 3. Initialise session store ─────────────────────────────────────────
	session.Init(cfg.SessionSecret)

	// ── 4. Initialise template engine ───────────────────────────────────────
	tmplEngine, err := handlers.NewTemplateEngine(
		"templates/layout/layout.html",
		templateFuncs(),
		"templates/auth",
		"templates/admin",
		"templates/customer",
	)
	if err != nil {
		log.Fatalf("Failed to initialize template engine: %v", err)
	}

	// ── 5. Initialise SSE hub ───────────────────────────────────────────────
	hub := handlers.NewSSEHub()
	go hub.Run()

	// ── 6. Wire up handlers ─────────────────────────────────────────────────
	auth := &handlers.AuthHandler{DB: db, Tmpl: tmplEngine}
	admin := &handlers.AdminHandler{DB: db, Tmpl: tmplEngine, Hub: hub}
	cust := &handlers.CustomerHandler{DB: db, Tmpl: tmplEngine}

	// ── 7. Build router ─────────────────────────────────────────────────────
	r := mux.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CSRFMiddleware)

	// Static files
	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))),
	)

	// Public routes
	r.HandleFunc("/", cust.Home).Methods("GET")
	r.HandleFunc("/menu", cust.Menu).Methods("GET")
	r.HandleFunc("/register", auth.RegisterGET).Methods("GET")
	r.HandleFunc("/register", auth.RegisterPOST).Methods("POST")
	r.HandleFunc("/login", auth.LoginGET).Methods("GET")
	r.HandleFunc("/login", auth.LoginPOST).Methods("POST")
	r.HandleFunc("/admin/login", auth.AdminLoginGET).Methods("GET")
	r.HandleFunc("/logout", auth.LogoutPOST).Methods("POST")

	// Customer routes (require auth)
	custR := r.NewRoute().Subrouter()
	custR.Use(middleware.RequireAuth)
	custR.HandleFunc("/cart", cust.CartView).Methods("GET")
	custR.HandleFunc("/cart/add", cust.CartAdd).Methods("POST")
	custR.HandleFunc("/cart/update", cust.CartUpdate).Methods("POST")
	custR.HandleFunc("/cart/clear", cust.CartClear).Methods("POST")
	custR.HandleFunc("/cart/apply-promo", cust.ApplyPromoCode).Methods("POST")
	custR.HandleFunc("/cart/remove-promo", cust.RemovePromoCode).Methods("POST")
	custR.HandleFunc("/orders", cust.OrderList).Methods("GET")
	custR.HandleFunc("/orders/new", cust.OrderNewGET).Methods("GET")
	custR.HandleFunc("/orders/new", cust.OrderNewPOST).Methods("POST")
	custR.HandleFunc("/orders/{id:[0-9]+}", cust.OrderDetail).Methods("GET")
	custR.HandleFunc("/orders/{id:[0-9]+}/cancel", cust.OrderCancel).Methods("POST")
	custR.HandleFunc("/events/{id:[0-9]+}", hub.SSEHandler).Methods("GET")

	// Admin routes (require auth + admin role)
	adminR := r.PathPrefix("/admin").Subrouter()
	adminR.Use(middleware.RequireAuth, middleware.RequireAdmin)
	adminR.HandleFunc("/dashboard", admin.Dashboard).Methods("GET")
	adminR.HandleFunc("/products", admin.ProductList).Methods("GET")
	adminR.HandleFunc("/products/new", admin.ProductNewGET).Methods("GET")
	adminR.HandleFunc("/products/new", admin.ProductNewPOST).Methods("POST")
	adminR.HandleFunc("/products/{id:[0-9]+}/edit", admin.ProductEditGET).Methods("GET")
	adminR.HandleFunc("/products/{id:[0-9]+}/edit", admin.ProductEditPOST).Methods("POST")
	adminR.HandleFunc("/products/{id:[0-9]+}/delete", admin.ProductDeletePOST).Methods("POST")
	adminR.HandleFunc("/orders", admin.OrderList).Methods("GET")
	adminR.HandleFunc("/orders/{id:[0-9]+}", admin.OrderDetail).Methods("GET")
	adminR.HandleFunc("/orders/{id:[0-9]+}/status", admin.OrderUpdateStatus).Methods("POST")
	adminR.HandleFunc("/promotions", admin.PromotionList).Methods("GET")
	adminR.HandleFunc("/promotions/new", admin.PromotionNewGET).Methods("GET")
	adminR.HandleFunc("/promotions/new", admin.PromotionNewPOST).Methods("POST")
	adminR.HandleFunc("/promotions/{id:[0-9]+}/edit", admin.PromotionEditGET).Methods("GET")
	adminR.HandleFunc("/promotions/{id:[0-9]+}/edit", admin.PromotionEditPOST).Methods("POST")
	adminR.HandleFunc("/promotions/{id:[0-9]+}/toggle", admin.PromotionTogglePOST).Methods("POST")
	adminR.HandleFunc("/promotions/{id:[0-9]+}/delete", admin.PromotionDeletePOST).Methods("POST")
	adminR.HandleFunc("/reports", admin.SalesReport).Methods("GET")
	adminR.HandleFunc("/customers", admin.CustomerList).Methods("GET")
	adminR.HandleFunc("/customers/{id:[0-9]+}/delete", admin.CustomerDeletePOST).Methods("POST")

	// ── 8. Start server ─────────────────────────────────────────────────────
	addr := ":" + cfg.Port
	log.Printf("🍗 K-Fry Shop running at http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

// templateFuncs returns custom template functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"mul": func(a float64, b int) float64 { return a * float64(b) },
		"add": func(a, b int) int { return a + b },
		"imageURL": func(p string) string {
			if p == "" {
				return ""
			}
			if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
				return p
			}
			return "/static/images/" + p
		},
		"formatMoney": func(v float64) string { return fmt.Sprintf("%.2f", v) },
		"formatDate": func(t time.Time) string { return t.Format("02/01/2006 15:04") },
		"formatDateOnly": func(t time.Time) string { return t.Format("02/01/2006") },
		"formatInputDate": func(t time.Time) string { return t.Format("2006-01-02T15:04") },
		"statusColor": func(status string) string {
			switch status {
			case "pending":
				return "warning"
			case "preparing":
				return "info"
			case "ready":
				return "primary"
			case "delivering":
				return "secondary"
			case "completed":
				return "success"
			case "cancelled":
				return "danger"
			default:
				return "light"
			}
		},
		"statusThai": func(status string) string {
			m := map[string]string{
				"pending":    "รอดำเนินการ",
				"preparing":  "กำลังเตรียม",
				"ready":      "พร้อมส่ง/รับ",
				"delivering": "กำลังจัดส่ง",
				"completed":  "เสร็จสิ้น",
				"cancelled":  "ยกเลิกแล้ว",
			}
			if v, ok := m[status]; ok {
				return v
			}
			return status
		},
	}
}
