package models

import (
	"database/sql"
	"time"
)

// Product represents a row in the products table.
type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
	Category    string
	Stock       int
	ImagePath   string
	Status      string
	CreatedAt   time.Time
}

// ListProducts returns all products ordered by created_at DESC (admin view).
func ListProducts(db *sql.DB) ([]Product, error) {
	rows, err := db.Query(
		`SELECT id, name, description, price, category, stock, COALESCE(image_path,''), status, created_at
		 FROM products ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

// ListAvailableProducts returns only status='available' products (customer menu).
func ListAvailableProducts(db *sql.DB) ([]Product, error) {
	rows, err := db.Query(
		`SELECT id, name, description, price, category, stock, COALESCE(image_path,''), status, created_at
		 FROM products WHERE status = 'available' ORDER BY category, name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProducts(rows)
}

// GetProduct returns a single product by ID.
func GetProduct(db *sql.DB, id int) (*Product, error) {
	p := &Product{}
	err := db.QueryRow(
		`SELECT id, name, description, price, category, stock, COALESCE(image_path,''), status, created_at
		 FROM products WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.ImagePath, &p.Status, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// CreateProduct inserts a new product record.
func CreateProduct(db *sql.DB, p *Product) error {
	_, err := db.Exec(
		`INSERT INTO products (name, description, price, category, stock, image_path, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Description, p.Price, p.Category, p.Stock, p.ImagePath, p.Status,
	)
	return err
}

// UpdateProduct modifies an existing product record.
func UpdateProduct(db *sql.DB, p *Product) error {
	_, err := db.Exec(
		`UPDATE products SET name=?, description=?, price=?, category=?, stock=?, image_path=?, status=?
		 WHERE id=?`,
		p.Name, p.Description, p.Price, p.Category, p.Stock, p.ImagePath, p.Status, p.ID,
	)
	return err
}

// DeleteProduct removes a product by ID.
func DeleteProduct(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM products WHERE id = ?`, id)
	return err
}

func scanProducts(rows *sql.Rows) ([]Product, error) {
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.ImagePath, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}
