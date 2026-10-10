package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"kfry_shop/internal/middleware"
	"kfry_shop/internal/models"
	"kfry_shop/internal/session"
)

// CustomerHandler holds dependencies for customer routes.
type CustomerHandler struct {
	DB   *sql.DB
	Tmpl TemplateRenderer
}

// Home redirects to /menu.
func (h *CustomerHandler) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/menu", http.StatusFound)
}

// ─── Menu ────────────────────────────────────────────────────────────────────

func (h *CustomerHandler) Menu(w http.ResponseWriter, r *http.Request) {
	products, err := models.ListAvailableProducts(h.DB)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	activePromos, _ := models.GetActivePromotions(h.DB)

	userID := session.GetUserID(r)
	var userName string
	if u, err := models.GetUserByID(h.DB, userID); err == nil {
		userName = u.Name
	}

	data := map[string]interface{}{
		"Title":        "เมนูไก่ทอดเกาหลี",
		"Role":         session.GetUserRole(r),
		"Products":     products,
		"ActivePromos": activePromos,
		"UserName":     userName,
		"CSRFToken":    middleware.GetCSRFToken(r, w),
		"Flashes":      flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "customer_menu.html", data)
}

// ─── Cart (session-backed) ───────────────────────────────────────────────────

func (h *CustomerHandler) CartView(w http.ResponseWriter, r *http.Request) {
	cart := getCart(r)
	var total float64
	for _, item := range cart {
		total += item.UnitPrice * float64(item.Quantity)
	}
	data := map[string]interface{}{
		"Title":     "ตะกร้าสินค้า",
		"Role":      session.GetUserRole(r),
		"Cart":      cart,
		"Total":     total,
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "customer_cart.html", data)
}

func (h *CustomerHandler) CartAdd(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	productID, _ := strconv.Atoi(r.FormValue("product_id"))
	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	if qty <= 0 {
		qty = 1
	}

	p, err := models.GetProduct(h.DB, productID)
	if err != nil || p.Status != "available" {
		session.SetFlash(r, w, "error", "ไม่พบสินค้าหรือสินค้าไม่พร้อมจำหน่าย")
		http.Redirect(w, r, "/menu", http.StatusFound)
		return
	}

	cart := getCart(r)
	found := false
	for i, item := range cart {
		if item.ProductID == productID {
			cart[i].Quantity += qty
			found = true
			break
		}
	}
	if !found {
		cart = append(cart, models.CartItem{
			ProductID:   p.ID,
			ProductName: p.Name,
			Quantity:    qty,
			UnitPrice:   p.Price,
		})
	}
	saveCart(r, w, cart)
	session.SetFlash(r, w, "success", "เพิ่ม '"+p.Name+"' ลงตะกร้าแล้ว!")
	http.Redirect(w, r, "/menu", http.StatusFound)
}

func (h *CustomerHandler) CartUpdate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	productID, _ := strconv.Atoi(r.FormValue("product_id"))
	qty, _ := strconv.Atoi(r.FormValue("quantity"))

	cart := getCart(r)
	newCart := cart[:0]
	for _, item := range cart {
		if item.ProductID == productID {
			if qty > 0 {
				item.Quantity = qty
				newCart = append(newCart, item)
			}
			// qty == 0 means remove item
		} else {
			newCart = append(newCart, item)
		}
	}
	saveCart(r, w, newCart)
	http.Redirect(w, r, "/cart", http.StatusFound)
}

func (h *CustomerHandler) CartClear(w http.ResponseWriter, r *http.Request) {
	saveCart(r, w, []models.CartItem{})
	savePromoCode(r, w, "")
	http.Redirect(w, r, "/cart", http.StatusFound)
}

// ─── Promotions (Checkout) ───────────────────────────────────────────────────

func (h *CustomerHandler) ApplyPromoCode(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := strings.TrimSpace(r.FormValue("promo_code"))
	if code == "" {
		session.SetFlash(r, w, "error", "กรุณาระบุโค้ดส่วนลด")
		http.Redirect(w, r, "/orders/new", http.StatusFound)
		return
	}

	cart := getCart(r)
	var subtotal float64
	for _, item := range cart {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}
	if subtotal <= 0 {
		session.SetFlash(r, w, "error", "ตะกร้าสินค้าว่างเปล่า")
		http.Redirect(w, r, "/orders/new", http.StatusFound)
		return
	}

	promo, err := models.GetPromotionByCode(h.DB, code)
	if err != nil {
		session.SetFlash(r, w, "error", "ไม่พบโค้ดส่วนลดนี้ หรือโค้ดไม่ถูกต้อง")
		http.Redirect(w, r, "/orders/new", http.StatusFound)
		return
	}

	discount, err := promo.CalculateDiscount(subtotal)
	if err != nil {
		session.SetFlash(r, w, "error", err.Error())
		http.Redirect(w, r, "/orders/new", http.StatusFound)
		return
	}

	savePromoCode(r, w, promo.Code)
	session.SetFlash(r, w, "success", fmt.Sprintf("ใช้โค้ด '%s' สำเร็จ! ได้รับส่วนลด ฿%.2f", promo.Code, discount))
	http.Redirect(w, r, "/orders/new", http.StatusFound)
}

func (h *CustomerHandler) RemovePromoCode(w http.ResponseWriter, r *http.Request) {
	savePromoCode(r, w, "")
	session.SetFlash(r, w, "success", "ยกเลิกโค้ดส่วนลดแล้ว")
	http.Redirect(w, r, "/orders/new", http.StatusFound)
}

// ─── Orders ──────────────────────────────────────────────────────────────────

func (h *CustomerHandler) OrderNewGET(w http.ResponseWriter, r *http.Request) {
	cart := getCart(r)
	if len(cart) == 0 {
		session.SetFlash(r, w, "error", "ตะกร้าว่างเปล่า กรุณาเลือกสินค้าก่อน")
		http.Redirect(w, r, "/menu", http.StatusFound)
		return
	}

	var subtotal float64
	for _, item := range cart {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}

	promoCode := getPromoCode(r)
	var discount float64
	var activePromo *models.Promotion
	if promoCode != "" {
		if p, err := models.GetPromotionByCode(h.DB, promoCode); err == nil {
			d, err := p.CalculateDiscount(subtotal)
			if err == nil {
				discount = d
				activePromo = p
			} else {
				// Promotion no longer valid with current cart
				savePromoCode(r, w, "")
				session.SetFlash(r, w, "error", "โค้ดส่วนลด '"+promoCode+"' ไม่ผ่านเงื่อนไข: "+err.Error())
				promoCode = ""
			}
		} else {
			savePromoCode(r, w, "")
			promoCode = ""
		}
	}

	total := subtotal - discount
	if total < 0 {
		total = 0
	}

	data := map[string]interface{}{
		"Title":       "ยืนยันคำสั่งซื้อ",
		"Role":        session.GetUserRole(r),
		"Cart":        cart,
		"Subtotal":    subtotal,
		"Discount":    discount,
		"PromoCode":   promoCode,
		"Promo":       activePromo,
		"Total":       total,
		"CSRFToken":   middleware.GetCSRFToken(r, w),
		"Flashes":     flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "customer_order_confirm.html", data)
}

func (h *CustomerHandler) OrderNewPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	orderType := r.FormValue("order_type")
	deliveryAddr := strings.TrimSpace(r.FormValue("delivery_address"))
	paymentMethod := r.FormValue("payment_method")

	if orderType == "delivery" && deliveryAddr == "" {
		session.SetFlash(r, w, "error", "กรุณาระบุที่อยู่สำหรับจัดส่ง")
		http.Redirect(w, r, "/orders/new", http.StatusFound)
		return
	}

	cart := getCart(r)
	if len(cart) == 0 {
		session.SetFlash(r, w, "error", "ตะกร้าว่างเปล่า")
		http.Redirect(w, r, "/menu", http.StatusFound)
		return
	}

	var subtotal float64
	for _, item := range cart {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}

	promoCode := getPromoCode(r)
	var discount float64
	if promoCode != "" {
		if p, err := models.GetPromotionByCode(h.DB, promoCode); err == nil {
			if d, err := p.CalculateDiscount(subtotal); err == nil {
				discount = d
			}
		}
	}

	customerID := session.GetUserID(r)
	o := &models.Order{
		CustomerID:      customerID,
		OrderType:       orderType,
		DeliveryAddress: deliveryAddr,
		DiscountAmount:  discount,
		PromotionCode:   promoCode,
	}

	orderID, err := models.CreateOrder(h.DB, o, cart, paymentMethod)
	if err != nil {
		session.SetFlash(r, w, "error", "เกิดข้อผิดพลาดในการสร้างคำสั่งซื้อ: "+err.Error())
		http.Redirect(w, r, "/orders/new", http.StatusFound)
		return
	}

	saveCart(r, w, []models.CartItem{}) // clear cart
	savePromoCode(r, w, "")             // clear promo code
	session.SetFlash(r, w, "success", "สั่งซื้อสำเร็จ! หมายเลขคำสั่งซื้อ #"+strconv.Itoa(orderID))
	http.Redirect(w, r, "/orders/"+strconv.Itoa(orderID), http.StatusFound)
}

func (h *CustomerHandler) OrderList(w http.ResponseWriter, r *http.Request) {
	customerID := session.GetUserID(r)
	orders, err := models.ListOrdersByCustomer(h.DB, customerID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Title":   "คำสั่งซื้อของฉัน",
		"Role":    session.GetUserRole(r),
		"Orders":  orders,
		"Flashes": flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "customer_orders.html", data)
}

func (h *CustomerHandler) OrderDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	customerID := session.GetUserID(r)
	order, err := models.GetOrder(h.DB, id)
	if err != nil || order.CustomerID != customerID {
		http.NotFound(w, r)
		return
	}
	data := map[string]interface{}{
		"Title":     "คำสั่งซื้อ #" + strconv.Itoa(id),
		"Role":      session.GetUserRole(r),
		"Order":     order,
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "customer_order_detail.html", data)
}

func (h *CustomerHandler) OrderCancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	customerID := session.GetUserID(r)
	if err := models.CancelOrder(h.DB, id, customerID); err != nil {
		session.SetFlash(r, w, "error", "ไม่สามารถยกเลิกคำสั่งซื้อได้: "+err.Error())
	} else {
		session.SetFlash(r, w, "success", "ยกเลิกคำสั่งซื้อสำเร็จ")
	}
	http.Redirect(w, r, "/orders/"+strconv.Itoa(id), http.StatusFound)
}

// ─── Cart persistence helpers (session-backed JSON) ───────────────────────────

const (
	cartSessionKey      = "cart"
	promoCodeSessionKey = "promo_code"
)

func getCart(r *http.Request) []models.CartItem {
	sess, _ := session.Get(r)
	raw, ok := sess.Values[cartSessionKey].(string)
	if !ok || raw == "" {
		return []models.CartItem{}
	}
	var cart []models.CartItem
	json.Unmarshal([]byte(raw), &cart)
	return cart
}

func saveCart(r *http.Request, w http.ResponseWriter, cart []models.CartItem) {
	data, _ := json.Marshal(cart)
	sess, _ := session.Get(r)
	sess.Values[cartSessionKey] = string(data)
	sess.Save(r, w)
}

func getPromoCode(r *http.Request) string {
	sess, _ := session.Get(r)
	if code, ok := sess.Values[promoCodeSessionKey].(string); ok {
		return code
	}
	return ""
}

func savePromoCode(r *http.Request, w http.ResponseWriter, code string) {
	sess, _ := session.Get(r)
	if code == "" {
		delete(sess.Values, promoCodeSessionKey)
	} else {
		sess.Values[promoCodeSessionKey] = code
	}
	sess.Save(r, w)
}
