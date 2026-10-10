package models

import (
	"database/sql"
	"fmt"
	"time"
)

// Order represents a row in the orders table.
type Order struct {
	ID              int
	CustomerID      int
	CustomerName    string
	CustomerEmail   string
	OrderType       string
	DeliveryAddress string
	TotalPrice      float64
	DiscountAmount  float64
	PromotionCode   string
	Status          string
	CreatedAt       time.Time
	Items           []OrderItem
	Payment         *Payment
	Subtotal        float64
	TotalItems      int
}

// OrderItem represents a row in the order_items table.
type OrderItem struct {
	ID              int
	OrderID         int
	ProductID       int
	ProductName     string
	ProductCategory string
	ProductImage    string
	Quantity        int
	UnitPrice       float64
	Subtotal        float64
}

// CartItem is used transiently during checkout (not stored in DB directly).
type CartItem struct {
	ProductID   int
	ProductName string
	Quantity    int
	UnitPrice   float64
}

// ValidOrderStatuses defines the allowed status progression for orders.
var ValidOrderStatuses = []string{"pending", "preparing", "ready", "delivering", "completed", "cancelled"}

// CreateOrder inserts an order with its items and payment in a single transaction.
func CreateOrder(db *sql.DB, o *Order, items []CartItem, paymentMethod string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Calculate subtotal
	var subtotal float64
	for _, item := range items {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}

	total := subtotal - o.DiscountAmount
	if total < 0 {
		total = 0
	}
	o.TotalPrice = total

	// Insert order
	res, err := tx.Exec(
		`INSERT INTO orders (customer_id, order_type, delivery_address, total_price, discount_amount, promotion_code, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
		o.CustomerID, o.OrderType, o.DeliveryAddress, o.TotalPrice, o.DiscountAmount, o.PromotionCode,
	)
	if err != nil {
		return 0, err
	}

	orderID64, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	orderID := int(orderID64)

	// Insert order items
	for _, item := range items {
		_, err = tx.Exec(
			`INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES (?, ?, ?, ?)`,
			orderID, item.ProductID, item.Quantity, item.UnitPrice,
		)
		if err != nil {
			return 0, err
		}
	}

	// Insert payment record
	_, err = tx.Exec(
		`INSERT INTO payments (order_id, payment_method, payment_status) VALUES (?, ?, 'pending')`,
		orderID, paymentMethod,
	)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return orderID, nil
}

// GetOrder retrieves a full order (with items and payment) by ID.
func GetOrder(db *sql.DB, id int) (*Order, error) {
	o := &Order{}
	err := db.QueryRow(
		`SELECT o.id, o.customer_id, u.name, COALESCE(u.email, ''), o.order_type, COALESCE(o.delivery_address,''),
		        o.total_price, COALESCE(o.discount_amount, 0), COALESCE(o.promotion_code, ''),
		        o.status, o.created_at
		 FROM orders o JOIN users u ON u.id = o.customer_id
		 WHERE o.id = ?`, id,
	).Scan(&o.ID, &o.CustomerID, &o.CustomerName, &o.CustomerEmail, &o.OrderType, &o.DeliveryAddress,
		&o.TotalPrice, &o.DiscountAmount, &o.PromotionCode, &o.Status, &o.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Fetch items
	rows, err := db.Query(
		`SELECT oi.id, oi.order_id, oi.product_id, p.name, COALESCE(p.category, ''), COALESCE(p.image_path, ''), oi.quantity, oi.unit_price
		 FROM order_items oi JOIN products p ON p.id = oi.product_id
		 WHERE oi.order_id = ?`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rawSubtotal float64
	var totalQty int
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.ProductCategory, &item.ProductImage, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, err
		}
		item.Subtotal = item.UnitPrice * float64(item.Quantity)
		rawSubtotal += item.Subtotal
		totalQty += item.Quantity
		o.Items = append(o.Items, item)
	}
	o.Subtotal = rawSubtotal
	o.TotalItems = totalQty

	// Fetch payment
	p := &Payment{}
	err = db.QueryRow(
		`SELECT id, order_id, payment_method, payment_status, COALESCE(paid_at, '0001-01-01')
		 FROM payments WHERE order_id = ?`, id,
	).Scan(&p.ID, &p.OrderID, &p.PaymentMethod, &p.PaymentStatus, &p.PaidAt)
	if err == nil {
		o.Payment = p
	}

	return o, nil
}

// ListOrdersByCustomer returns all orders for a given customer.
func ListOrdersByCustomer(db *sql.DB, customerID int) ([]Order, error) {
	rows, err := db.Query(
		`SELECT id, customer_id, order_type, COALESCE(delivery_address,''), total_price, 
		        COALESCE(discount_amount, 0), COALESCE(promotion_code, ''), status, created_at
		 FROM orders WHERE customer_id = ? ORDER BY created_at DESC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrders(rows)
}

// ListAllOrders returns all orders for the admin view with items & payment details.
func ListAllOrders(db *sql.DB) ([]Order, error) {
	rows, err := db.Query(
		`SELECT o.id, o.customer_id, u.name, COALESCE(u.email, ''), o.order_type, COALESCE(o.delivery_address,''),
		        o.total_price, COALESCE(o.discount_amount, 0), COALESCE(o.promotion_code, ''),
		        o.status, o.created_at,
		        COALESCE(p.payment_method, 'cash'), COALESCE(p.payment_status, 'pending')
		 FROM orders o 
		 JOIN users u ON u.id = o.customer_id
		 LEFT JOIN payments p ON p.order_id = o.id
		 ORDER BY o.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var payMethod, payStatus string
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.CustomerName, &o.CustomerEmail, &o.OrderType,
			&o.DeliveryAddress, &o.TotalPrice, &o.DiscountAmount, &o.PromotionCode,
			&o.Status, &o.CreatedAt, &payMethod, &payStatus); err != nil {
			return nil, err
		}
		o.Payment = &Payment{
			OrderID:       o.ID,
			PaymentMethod: payMethod,
			PaymentStatus: payStatus,
		}
		orders = append(orders, o)
	}

	// Fetch order items summary for each order
	for i := range orders {
		itemRows, err := db.Query(
			`SELECT oi.id, oi.order_id, oi.product_id, p.name, COALESCE(p.category, ''), COALESCE(p.image_path, ''), oi.quantity, oi.unit_price
			 FROM order_items oi JOIN products p ON p.id = oi.product_id
			 WHERE oi.order_id = ?`, orders[i].ID,
		)
		if err == nil {
			var rawSubtotal float64
			var totalQty int
			for itemRows.Next() {
				var item OrderItem
				if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.ProductCategory, &item.ProductImage, &item.Quantity, &item.UnitPrice); err == nil {
					item.Subtotal = item.UnitPrice * float64(item.Quantity)
					rawSubtotal += item.Subtotal
					totalQty += item.Quantity
					orders[i].Items = append(orders[i].Items, item)
				}
			}
			itemRows.Close()
			orders[i].Subtotal = rawSubtotal
			orders[i].TotalItems = totalQty
		}
	}

	return orders, nil
}

// GetPrevAndNextOrderID returns the previous and next order IDs relative to currentID for quick navigation.
func GetPrevAndNextOrderID(db *sql.DB, currentID int) (prevID int, nextID int) {
	_ = db.QueryRow(`SELECT id FROM orders WHERE id < ? ORDER BY id DESC LIMIT 1`, currentID).Scan(&prevID)
	_ = db.QueryRow(`SELECT id FROM orders WHERE id > ? ORDER BY id ASC LIMIT 1`, currentID).Scan(&nextID)
	return
}

// UpdateOrderStatus changes an order's status.
func UpdateOrderStatus(db *sql.DB, id int, status string) error {
	_, err := db.Exec(`UPDATE orders SET status = ? WHERE id = ?`, status, id)
	return err
}

// CancelOrder allows a customer to cancel their order only if it is still pending.
func CancelOrder(db *sql.DB, id, customerID int) error {
	res, err := db.Exec(
		`UPDATE orders SET status = 'cancelled'
		 WHERE id = ? AND customer_id = ? AND status = 'pending'`,
		id, customerID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("order cannot be cancelled (not found, not yours, or no longer pending)")
	}
	return nil
}

// GetOrderStats returns aggregate stats for the admin dashboard.
func GetOrderStats(db *sql.DB) (totalOrders, pendingOrders int, totalRevenue float64, err error) {
	db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&totalOrders)
	db.QueryRow(`SELECT COUNT(*) FROM orders WHERE status = 'pending'`).Scan(&pendingOrders)
	db.QueryRow(`SELECT COALESCE(SUM(total_price),0) FROM orders WHERE status != 'cancelled'`).Scan(&totalRevenue)
	return
}

func scanOrders(rows *sql.Rows) ([]Order, error) {
	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.OrderType, &o.DeliveryAddress,
			&o.TotalPrice, &o.DiscountAmount, &o.PromotionCode, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// ── Sales Reporting Models & Queries ─────────────────────────────────────────

type SalesReportSummary struct {
	TotalRevenue    float64
	TotalOrders     int
	CompletedOrders int
	CancelledOrders int
	TotalDiscounts  float64
	AverageOrderVal float64
}

type TopProduct struct {
	ProductID   int     `json:"product_id"`
	ProductName string  `json:"product_name"`
	Category    string  `json:"category"`
	TotalSold   int     `json:"total_sold"`
	TotalSales  float64 `json:"total_sales"`
}

type DailySale struct {
	Date       string  `json:"date"`
	OrderCount int     `json:"order_count"`
	Revenue    float64 `json:"revenue"`
}

// GetSalesReportSummary computes total revenue, counts, and averages in the date range.
func GetSalesReportSummary(db *sql.DB, startDate, endDate string) (*SalesReportSummary, error) {
	s := &SalesReportSummary{}
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN status != 'cancelled' THEN total_price ELSE 0 END), 0),
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status != 'cancelled' THEN discount_amount ELSE 0 END), 0)
		FROM orders
		WHERE created_at >= ? AND created_at <= ?
	`
	err := db.QueryRow(query, startDate, endDate).Scan(
		&s.TotalRevenue,
		&s.TotalOrders,
		&s.CompletedOrders,
		&s.CancelledOrders,
		&s.TotalDiscounts,
	)
	if err != nil {
		return nil, err
	}

	validOrders := s.TotalOrders - s.CancelledOrders
	if validOrders > 0 {
		s.AverageOrderVal = s.TotalRevenue / float64(validOrders)
	}
	return s, nil
}

// GetTopSellingProducts retrieves the most ordered products in the date range.
func GetTopSellingProducts(db *sql.DB, limit int, startDate, endDate string) ([]*TopProduct, error) {
	query := `
		SELECT 
			p.id, p.name, COALESCE(p.category, 'General'),
			SUM(oi.quantity) as total_sold,
			SUM(oi.quantity * oi.unit_price) as total_sales
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN products p ON p.id = oi.product_id
		WHERE o.status != 'cancelled' AND o.created_at >= ? AND o.created_at <= ?
		GROUP BY p.id, p.name, p.category
		ORDER BY total_sold DESC
		LIMIT ?
	`
	rows, err := db.Query(query, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*TopProduct
	for rows.Next() {
		tp := &TopProduct{}
		if err := rows.Scan(&tp.ProductID, &tp.ProductName, &tp.Category, &tp.TotalSold, &tp.TotalSales); err != nil {
			return nil, err
		}
		products = append(products, tp)
	}
	return products, rows.Err()
}

// GetDailySalesTrends retrieves revenue and order count grouped by date.
func GetDailySalesTrends(db *sql.DB, startDate, endDate string) ([]*DailySale, error) {
	query := `
		SELECT 
			DATE_FORMAT(created_at, '%Y-%m-%d') as sale_date,
			COUNT(*) as order_count,
			COALESCE(SUM(total_price), 0) as daily_revenue
		FROM orders
		WHERE status != 'cancelled' AND created_at >= ? AND created_at <= ?
		GROUP BY DATE_FORMAT(created_at, '%Y-%m-%d')
		ORDER BY sale_date ASC
	`
	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []*DailySale
	for rows.Next() {
		ds := &DailySale{}
		if err := rows.Scan(&ds.Date, &ds.OrderCount, &ds.Revenue); err != nil {
			return nil, err
		}
		trends = append(trends, ds)
	}
	return trends, rows.Err()
}
