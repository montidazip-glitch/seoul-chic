package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kfry_shop/internal/middleware"
	"kfry_shop/internal/models"
	"kfry_shop/internal/session"
)

// AdminHandler holds dependencies for admin routes.
type AdminHandler struct {
	DB   *sql.DB
	Tmpl TemplateRenderer
	Hub  *SSEHub
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	totalOrders, pendingOrders, totalRevenue, _ := models.GetOrderStats(h.DB)
	var productCount, customerCount, promoCount int
	h.DB.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&productCount)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='customer'`).Scan(&customerCount)
	h.DB.QueryRow(`SELECT COUNT(*) FROM promotions WHERE is_active = TRUE`).Scan(&promoCount)

	data := map[string]interface{}{
		"Title":         "แดชบอร์ด Admin",
		"Role":          "admin",
		"TotalOrders":   totalOrders,
		"PendingOrders": pendingOrders,
		"TotalRevenue":  totalRevenue,
		"ProductCount":  productCount,
		"CustomerCount": customerCount,
		"PromoCount":    promoCount,
		"CSRFToken":     middleware.GetCSRFToken(r, w),
		"Flashes":       flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_dashboard.html", data)
}

// ─── Products ────────────────────────────────────────────────────────────────

func (h *AdminHandler) ProductList(w http.ResponseWriter, r *http.Request) {
	products, err := models.ListProducts(h.DB)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Title":     "จัดการสินค้า",
		"Role":      "admin",
		"Products":  products,
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_products.html", data)
}

func (h *AdminHandler) ProductNewGET(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":     "เพิ่มสินค้าใหม่",
		"Role":      "admin",
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_product_form.html", data)
}

func (h *AdminHandler) ProductNewPOST(w http.ResponseWriter, r *http.Request) {
	p := productFromRequest(r)
	if err := models.CreateProduct(h.DB, p); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถเพิ่มสินค้าได้: "+err.Error())
		http.Redirect(w, r, "/admin/products/new", http.StatusFound)
		return
	}
	session.SetFlash(r, w, "success", "เพิ่มสินค้าสำเร็จ!")
	http.Redirect(w, r, "/admin/products", http.StatusFound)
}

func (h *AdminHandler) ProductEditGET(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := models.GetProduct(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := map[string]interface{}{
		"Title":     "แก้ไขสินค้า",
		"Role":      "admin",
		"Product":   p,
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_product_form.html", data)
}

func (h *AdminHandler) ProductEditPOST(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	existing, err := models.GetProduct(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	p := productFromRequest(r)
	p.ID = id
	// If no new image was uploaded and text field was left empty, preserve existing image
	if p.ImagePath == "" {
		p.ImagePath = existing.ImagePath
	}

	if err := models.UpdateProduct(h.DB, p); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถแก้ไขสินค้าได้")
		http.Redirect(w, r, "/admin/products", http.StatusFound)
		return
	}
	session.SetFlash(r, w, "success", "แก้ไขสินค้าสำเร็จ!")
	http.Redirect(w, r, "/admin/products", http.StatusFound)
}

func (h *AdminHandler) ProductDeletePOST(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := models.DeleteProduct(h.DB, id); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถลบสินค้าได้")
	} else {
		session.SetFlash(r, w, "success", "ลบสินค้าสำเร็จ!")
	}
	http.Redirect(w, r, "/admin/products", http.StatusFound)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (h *AdminHandler) OrderList(w http.ResponseWriter, r *http.Request) {
	allOrders, err := models.ListAllOrders(h.DB)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	filterStatus := strings.TrimSpace(r.URL.Query().Get("status"))
	filterType := strings.TrimSpace(r.URL.Query().Get("type"))
	filterPayment := strings.TrimSpace(r.URL.Query().Get("payment_status"))
	searchQuery := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	// Stats counters across ALL orders
	var totalOrders, pendingOrders, preparingOrders, completedOrders, cancelledOrders int
	var totalRevenue float64

	for _, o := range allOrders {
		totalOrders++
		if o.Status == "pending" {
			pendingOrders++
		} else if o.Status == "preparing" || o.Status == "ready" || o.Status == "delivering" {
			preparingOrders++
		} else if o.Status == "completed" {
			completedOrders++
		} else if o.Status == "cancelled" {
			cancelledOrders++
		}

		if o.Status != "cancelled" {
			totalRevenue += o.TotalPrice
		}
	}

	// Filter orders for display
	filtered := make([]models.Order, 0, len(allOrders))
	for _, o := range allOrders {
		if filterStatus != "" && o.Status != filterStatus {
			continue
		}
		if filterType != "" && o.OrderType != filterType {
			continue
		}
		if filterPayment != "" && (o.Payment == nil || o.Payment.PaymentStatus != filterPayment) {
			continue
		}
		if searchQuery != "" {
			idStr := strconv.Itoa(o.ID)
			nameLower := strings.ToLower(o.CustomerName)
			emailLower := strings.ToLower(o.CustomerEmail)
			if !strings.Contains(idStr, searchQuery) &&
				!strings.Contains(nameLower, searchQuery) &&
				!strings.Contains(emailLower, searchQuery) {
				continue
			}
		}
		filtered = append(filtered, o)
	}

	data := map[string]interface{}{
		"Title":           "จัดการและติดตามคำสั่งซื้อ",
		"Role":            "admin",
		"Orders":          filtered,
		"FilterStatus":    filterStatus,
		"FilterType":      filterType,
		"FilterPayment":   filterPayment,
		"SearchQuery":     searchQuery,
		"Statuses":        models.ValidOrderStatuses,
		"TotalOrders":     totalOrders,
		"PendingOrders":   pendingOrders,
		"PreparingOrders": preparingOrders,
		"CompletedOrders": completedOrders,
		"CancelledOrders": cancelledOrders,
		"TotalRevenue":    totalRevenue,
		"CSRFToken":       middleware.GetCSRFToken(r, w),
		"Flashes":         flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_orders.html", data)
}

func (h *AdminHandler) OrderDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	order, err := models.GetOrder(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	prevID, nextID := models.GetPrevAndNextOrderID(h.DB, id)

	data := map[string]interface{}{
		"Title":     "รายละเอียดคำสั่งซื้อ #" + strconv.Itoa(id),
		"Role":      "admin",
		"Order":     order,
		"Statuses":  models.ValidOrderStatuses,
		"PrevID":    prevID,
		"NextID":    nextID,
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_order_detail.html", data)
}

func (h *AdminHandler) OrderUpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	newStatus := r.FormValue("status")
	if err := models.UpdateOrderStatus(h.DB, id, newStatus); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถอัปเดตสถานะได้")
	} else {
		// Broadcast the new status to any SSE clients watching this order
		h.Hub.BroadcastStatus(id, newStatus)
		session.SetFlash(r, w, "success", "อัปเดตสถานะเป็น '"+newStatus+"' เรียบร้อยแล้ว!")
	}
	referer := r.Header.Get("Referer")
	if referer != "" && strings.Contains(referer, "/admin/orders") {
		http.Redirect(w, r, referer, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/admin/orders/"+strconv.Itoa(id), http.StatusFound)
}

func (h *AdminHandler) OrderUpdatePaymentStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	newPaymentStatus := r.FormValue("payment_status")
	if err := models.UpdatePaymentStatus(h.DB, id, newPaymentStatus); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถอัปเดตสถานะการชำระเงินได้")
	} else {
		statusThai := "ชำระเงินแล้ว"
		if newPaymentStatus != "paid" {
			statusThai = "รอชำระเงิน"
		}
		session.SetFlash(r, w, "success", "อัปเดตสถานะการชำระเงินเป็น '"+statusThai+"' เรียบร้อยแล้ว!")
	}
	http.Redirect(w, r, "/admin/orders/"+strconv.Itoa(id), http.StatusFound)
}

// ─── Promotions ──────────────────────────────────────────────────────────────

func (h *AdminHandler) PromotionList(w http.ResponseWriter, r *http.Request) {
	promotions, err := models.GetAllPromotions(h.DB)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Title":      "จัดการโปรโมชั่นและส่วนลด",
		"Role":       "admin",
		"Promotions": promotions,
		"CSRFToken":  middleware.GetCSRFToken(r, w),
		"Flashes":    flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_promotions.html", data)
}

func (h *AdminHandler) PromotionNewGET(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	data := map[string]interface{}{
		"Title":       "สร้างโปรโมชั่นใหม่",
		"Role":        "admin",
		"CSRFToken":   middleware.GetCSRFToken(r, w),
		"Flashes":     flashData(r, w),
		"DefaultStart": now.Format("2006-01-02T15:04"),
		"DefaultEnd":   now.AddDate(0, 1, 0).Format("2006-01-02T15:04"),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_promotion_form.html", data)
}

func (h *AdminHandler) PromotionNewPOST(w http.ResponseWriter, r *http.Request) {
	promo, err := promoFromForm(r)
	if err != nil {
		session.SetFlash(r, w, "error", "ข้อมูลไม่ถูกต้อง: "+err.Error())
		http.Redirect(w, r, "/admin/promotions/new", http.StatusFound)
		return
	}

	if err := models.CreatePromotion(h.DB, promo); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถสร้างโปรโมชั่นได้: "+err.Error())
		http.Redirect(w, r, "/admin/promotions/new", http.StatusFound)
		return
	}

	session.SetFlash(r, w, "success", "สร้างโปรโมชั่น '"+promo.Code+"' สำเร็จ!")
	http.Redirect(w, r, "/admin/promotions", http.StatusFound)
}

func (h *AdminHandler) PromotionEditGET(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	promo, err := models.GetPromotionByID(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":        "แก้ไขโปรโมชั่น #" + promo.Code,
		"Role":         "admin",
		"Promotion":    promo,
		"DefaultStart": promo.StartDate.Format("2006-01-02T15:04"),
		"DefaultEnd":   promo.EndDate.Format("2006-01-02T15:04"),
		"CSRFToken":    middleware.GetCSRFToken(r, w),
		"Flashes":      flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_promotion_form.html", data)
}

func (h *AdminHandler) PromotionEditPOST(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	promo, err := promoFromForm(r)
	if err != nil {
		session.SetFlash(r, w, "error", "ข้อมูลไม่ถูกต้อง: "+err.Error())
		http.Redirect(w, r, "/admin/promotions/edit/"+strconv.Itoa(id), http.StatusFound)
		return
	}
	promo.ID = id

	if err := models.UpdatePromotion(h.DB, promo); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถแก้ไขโปรโมชั่นได้: "+err.Error())
		http.Redirect(w, r, "/admin/promotions/edit/"+strconv.Itoa(id), http.StatusFound)
		return
	}

	session.SetFlash(r, w, "success", "บันทึกการแก้ไขโปรโมชั่นสำเร็จ!")
	http.Redirect(w, r, "/admin/promotions", http.StatusFound)
}

func (h *AdminHandler) PromotionTogglePOST(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	active := r.FormValue("active") == "true"
	if err := models.TogglePromotionActive(h.DB, id, active); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถเปลี่ยนสถานะโปรโมชั่นได้")
	} else {
		statusText := "ปิดใช้งาน"
		if active {
			statusText = "เปิดใช้งาน"
		}
		session.SetFlash(r, w, "success", "เปลี่ยนสถานะเป็น "+statusText+" เรียบร้อยแล้ว")
	}
	http.Redirect(w, r, "/admin/promotions", http.StatusFound)
}

func (h *AdminHandler) PromotionDeletePOST(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := models.DeletePromotion(h.DB, id); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถลบโปรโมชั่นได้")
	} else {
		session.SetFlash(r, w, "success", "ลบโปรโมชั่นสำเร็จ!")
	}
	http.Redirect(w, r, "/admin/promotions", http.StatusFound)
}

// ─── Sales Reports ───────────────────────────────────────────────────────────

func (h *AdminHandler) SalesReport(w http.ResponseWriter, r *http.Request) {
	dateRange := r.URL.Query().Get("range")
	if dateRange == "" {
		dateRange = "month"
	}

	now := time.Now()
	var startDate, endDate time.Time

	switch dateRange {
	case "today":
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	case "week":
		startDate = now.AddDate(0, 0, -7)
		endDate = now
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = now
	case "custom":
		s := r.URL.Query().Get("start_date")
		e := r.URL.Query().Get("end_date")
		var err1, err2 error
		startDate, err1 = time.Parse("2006-01-02", s)
		endDate, err2 = time.Parse("2006-01-02", e)
		if err1 != nil || err2 != nil {
			startDate = now.AddDate(0, 0, -30)
			endDate = now
		} else {
			endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
		}
	default:
		dateRange = "month"
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = now
	}

	startStr := startDate.Format("2006-01-02 15:04:05")
	endStr := endDate.Format("2006-01-02 15:04:05")

	summary, err := models.GetSalesReportSummary(h.DB, startStr, endStr)
	if err != nil {
		http.Error(w, "Failed to load report summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	topProducts, _ := models.GetTopSellingProducts(h.DB, 10, startStr, endStr)
	dailyTrends, _ := models.GetDailySalesTrends(h.DB, startStr, endStr)

	// Prepare data for Chart.js
	trendDates := []string{}
	trendRevenues := []float64{}
	trendOrders := []int{}
	for _, t := range dailyTrends {
		trendDates = append(trendDates, t.Date)
		trendRevenues = append(trendRevenues, t.Revenue)
		trendOrders = append(trendOrders, t.OrderCount)
	}

	topNames := []string{}
	topSold := []int{}
	for _, tp := range topProducts {
		topNames = append(topNames, tp.ProductName)
		topSold = append(topSold, tp.TotalSold)
	}

	trendDatesJSON, _ := json.Marshal(trendDates)
	trendRevenuesJSON, _ := json.Marshal(trendRevenues)
	trendOrdersJSON, _ := json.Marshal(trendOrders)
	topNamesJSON, _ := json.Marshal(topNames)
	topSoldJSON, _ := json.Marshal(topSold)

	data := map[string]interface{}{
		"Title":             "รายงานยอดขายและสถิติ",
		"Role":              "admin",
		"DateRange":         dateRange,
		"StartDate":         startDate.Format("2006-01-02"),
		"EndDate":           endDate.Format("2006-01-02"),
		"Summary":           summary,
		"TopProducts":       topProducts,
		"DailyTrends":       dailyTrends,
		"TrendDatesJSON":    template.JS(trendDatesJSON),
		"TrendRevenuesJSON": template.JS(trendRevenuesJSON),
		"TrendOrdersJSON":   template.JS(trendOrdersJSON),
		"TopNamesJSON":      template.JS(topNamesJSON),
		"TopSoldJSON":       template.JS(topSoldJSON),
		"CSRFToken":         middleware.GetCSRFToken(r, w),
		"Flashes":           flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_reports.html", data)
}

// ─── Customers ───────────────────────────────────────────────────────────────

func (h *AdminHandler) CustomerList(w http.ResponseWriter, r *http.Request) {
	customers, err := models.ListCustomers(h.DB)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Title":     "จัดการลูกค้า",
		"Role":      "admin",
		"Customers": customers,
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_customers.html", data)
}

func (h *AdminHandler) CustomerDeletePOST(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := models.DeleteUser(h.DB, id); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถลบลูกค้าได้")
	} else {
		session.SetFlash(r, w, "success", "ลบลูกค้าสำเร็จ!")
	}
	http.Redirect(w, r, "/admin/customers", http.StatusFound)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func productFromRequest(r *http.Request) *models.Product {
	// Support multipart upload (10 MB)
	_ = r.ParseMultipartForm(10 << 20)

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	stock, _ := strconv.Atoi(r.FormValue("stock"))
	imagePath := strings.TrimSpace(r.FormValue("image_path"))

	// Check if a real image file was uploaded
	file, header, err := r.FormFile("image_file")
	if err == nil && header != nil && header.Filename != "" {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif" {
			uploadDir := filepath.Join("static", "images", "uploads")
			_ = os.MkdirAll(uploadDir, 0755)

			filename := fmt.Sprintf("prod_%d%s", time.Now().UnixNano(), ext)
			destPath := filepath.Join(uploadDir, filename)
			out, err := os.Create(destPath)
			if err == nil {
				defer out.Close()
				_, _ = io.Copy(out, file)
				imagePath = "/static/images/uploads/" + filename
			}
		}
	}

	return &models.Product{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Price:       price,
		Category:    strings.TrimSpace(r.FormValue("category")),
		Stock:       stock,
		ImagePath:   imagePath,
		Status:      r.FormValue("status"),
	}
}

func promoFromForm(r *http.Request) (*models.Promotion, error) {
	r.ParseForm()
	code := strings.TrimSpace(r.FormValue("code"))
	if code == "" {
		return nil, fmt.Errorf("กรุณาระบุรหัสโปรโมชั่น")
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return nil, fmt.Errorf("กรุณาระบุชื่อโปรโมชั่น")
	}
	discountType := r.FormValue("discount_type")
	if discountType != "percentage" && discountType != "fixed" {
		discountType = "fixed"
	}
	discountVal, err := strconv.ParseFloat(r.FormValue("discount_value"), 64)
	if err != nil || discountVal <= 0 {
		return nil, fmt.Errorf("มูลค่าส่วนลดต้องมากกว่า 0")
	}
	if discountType == "percentage" && discountVal > 100 {
		return nil, fmt.Errorf("ส่วนลดแบบเปอร์เซ็นต์ต้องไม่เกิน 100%%")
	}

	minOrder, _ := strconv.ParseFloat(r.FormValue("min_order_amount"), 64)

	startTime, err := time.Parse("2006-01-02T15:04", r.FormValue("start_date"))
	if err != nil {
		startTime = time.Now()
	}
	endTime, err := time.Parse("2006-01-02T15:04", r.FormValue("end_date"))
	if err != nil {
		endTime = startTime.AddDate(0, 1, 0)
	}

	isActive := r.FormValue("is_active") == "1" || r.FormValue("is_active") == "true"

	return &models.Promotion{
		Code:           code,
		Title:          title,
		Description:    strings.TrimSpace(r.FormValue("description")),
		DiscountType:   discountType,
		DiscountValue:  discountVal,
		MinOrderAmount: minOrder,
		StartDate:      startTime,
		EndDate:        endTime,
		IsActive:       isActive,
	}, nil
}

func flashData(r *http.Request, w http.ResponseWriter) map[string][]interface{} {
	return map[string][]interface{}{
		"success": session.GetFlashes(r, w, "success"),
		"error":   session.GetFlashes(r, w, "error"),
	}
}
