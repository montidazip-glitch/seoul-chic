package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Promotion struct {
	ID             int       `json:"id"`
	Code           string    `json:"code"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	DiscountType   string    `json:"discount_type"` // "percentage" or "fixed"
	DiscountValue  float64   `json:"discount_value"`
	MinOrderAmount float64   `json:"min_order_amount"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

// FormattedDiscount returns a display string like "20%" or "฿50.00"
func (p *Promotion) FormattedDiscount() string {
	if p.DiscountType == "percentage" {
		return fmt.Sprintf("%.0f%%", p.DiscountValue)
	}
	return fmt.Sprintf("฿%.2f", p.DiscountValue)
}

// IsCurrentlyValid checks if promotion is active and within date range
func (p *Promotion) IsCurrentlyValid() bool {
	if !p.IsActive {
		return false
	}
	now := time.Now()
	return !now.Before(p.StartDate) && !now.After(p.EndDate)
}

// CalculateDiscount calculates the discount amount based on subtotal
func (p *Promotion) CalculateDiscount(subtotal float64) (float64, error) {
	if !p.IsCurrentlyValid() {
		return 0, errors.New("โปรโมชั่นนี้หมดอายุหรือไม่สามารถใช้งานได้แล้ว")
	}
	if subtotal < p.MinOrderAmount {
		return 0, fmt.Errorf("ยอดสั่งซื้อขั้นต่ำสำหรับโปรโมชั่นนี้คือ ฿%.2f", p.MinOrderAmount)
	}

	var discount float64
	if p.DiscountType == "percentage" {
		discount = (subtotal * p.DiscountValue) / 100.0
	} else {
		discount = p.DiscountValue
	}

	// Discount cannot exceed subtotal
	if discount > subtotal {
		discount = subtotal
	}
	return discount, nil
}

// CreatePromotion inserts a new promotion into the database
func CreatePromotion(db *sql.DB, p *Promotion) error {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	query := `
		INSERT INTO promotions (code, title, description, discount_type, discount_value, min_order_amount, start_date, end_date, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := db.Exec(query, p.Code, p.Title, p.Description, p.DiscountType, p.DiscountValue, p.MinOrderAmount, p.StartDate, p.EndDate, p.IsActive)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)
	return nil
}

// GetPromotionByID retrieves a promotion by its ID
func GetPromotionByID(db *sql.DB, id int) (*Promotion, error) {
	query := `
		SELECT id, code, title, description, discount_type, discount_value, min_order_amount, start_date, end_date, is_active, created_at
		FROM promotions WHERE id = ?
	`
	p := &Promotion{}
	err := db.QueryRow(query, id).Scan(
		&p.ID, &p.Code, &p.Title, &p.Description, &p.DiscountType, &p.DiscountValue, &p.MinOrderAmount,
		&p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetPromotionByCode retrieves a promotion by its code (case-insensitive)
func GetPromotionByCode(db *sql.DB, code string) (*Promotion, error) {
	query := `
		SELECT id, code, title, description, discount_type, discount_value, min_order_amount, start_date, end_date, is_active, created_at
		FROM promotions WHERE UPPER(code) = UPPER(?)
	`
	p := &Promotion{}
	err := db.QueryRow(query, strings.TrimSpace(code)).Scan(
		&p.ID, &p.Code, &p.Title, &p.Description, &p.DiscountType, &p.DiscountValue, &p.MinOrderAmount,
		&p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetAllPromotions retrieves all promotions ordered by created_at DESC
func GetAllPromotions(db *sql.DB) ([]*Promotion, error) {
	query := `
		SELECT id, code, title, description, discount_type, discount_value, min_order_amount, start_date, end_date, is_active, created_at
		FROM promotions ORDER BY created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []*Promotion
	for rows.Next() {
		p := &Promotion{}
		err := rows.Scan(
			&p.ID, &p.Code, &p.Title, &p.Description, &p.DiscountType, &p.DiscountValue, &p.MinOrderAmount,
			&p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		promotions = append(promotions, p)
	}
	return promotions, rows.Err()
}

// GetActivePromotions returns promotions that are active and currently within validity period
func GetActivePromotions(db *sql.DB) ([]*Promotion, error) {
	query := `
		SELECT id, code, title, description, discount_type, discount_value, min_order_amount, start_date, end_date, is_active, created_at
		FROM promotions 
		WHERE is_active = TRUE AND start_date <= NOW() AND end_date >= NOW()
		ORDER BY discount_value DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var promotions []*Promotion
	for rows.Next() {
		p := &Promotion{}
		err := rows.Scan(
			&p.ID, &p.Code, &p.Title, &p.Description, &p.DiscountType, &p.DiscountValue, &p.MinOrderAmount,
			&p.StartDate, &p.EndDate, &p.IsActive, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		promotions = append(promotions, p)
	}
	return promotions, rows.Err()
}

// UpdatePromotion updates an existing promotion
func UpdatePromotion(db *sql.DB, p *Promotion) error {
	p.Code = strings.ToUpper(strings.TrimSpace(p.Code))
	query := `
		UPDATE promotions 
		SET code = ?, title = ?, description = ?, discount_type = ?, discount_value = ?, min_order_amount = ?, start_date = ?, end_date = ?, is_active = ?
		WHERE id = ?
	`
	_, err := db.Exec(query, p.Code, p.Title, p.Description, p.DiscountType, p.DiscountValue, p.MinOrderAmount, p.StartDate, p.EndDate, p.IsActive, p.ID)
	return err
}

// TogglePromotionActive updates the is_active status of a promotion
func TogglePromotionActive(db *sql.DB, id int, active bool) error {
	query := `UPDATE promotions SET is_active = ? WHERE id = ?`
	_, err := db.Exec(query, active, id)
	return err
}

// DeletePromotion removes a promotion by ID
func DeletePromotion(db *sql.DB, id int) error {
	query := `DELETE FROM promotions WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}
