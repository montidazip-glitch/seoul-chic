package models

import (
	"database/sql"
	"time"
)

// Payment represents a row in the payments table.
type Payment struct {
	ID            int
	OrderID       int
	PaymentMethod string
	PaymentStatus string
	PaidAt        time.Time
}

// GetPayment retrieves the payment record for a given order.
func GetPayment(db *sql.DB, orderID int) (*Payment, error) {
	p := &Payment{}
	err := db.QueryRow(
		`SELECT id, order_id, payment_method, payment_status, COALESCE(paid_at, '0001-01-01')
		 FROM payments WHERE order_id = ?`, orderID,
	).Scan(&p.ID, &p.OrderID, &p.PaymentMethod, &p.PaymentStatus, &p.PaidAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// UpdatePaymentStatus marks a payment as paid and records the timestamp.
func UpdatePaymentStatus(db *sql.DB, orderID int, status string) error {
	if status == "paid" {
		_, err := db.Exec(
			`UPDATE payments SET payment_status = 'paid', paid_at = NOW() WHERE order_id = ?`,
			orderID,
		)
		return err
	}
	_, err := db.Exec(
		`UPDATE payments SET payment_status = ? WHERE order_id = ?`,
		status, orderID,
	)
	return err
}
