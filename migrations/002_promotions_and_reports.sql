CREATE TABLE IF NOT EXISTS promotions (
    id               INT AUTO_INCREMENT PRIMARY KEY,
    code             VARCHAR(50) NOT NULL UNIQUE,
    title            VARCHAR(100) NOT NULL,
    description      TEXT,
    discount_type    ENUM('percentage', 'fixed') NOT NULL DEFAULT 'fixed',
    discount_value   DECIMAL(10,2) NOT NULL,
    min_order_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    start_date       DATETIME NOT NULL,
    end_date         DATETIME NOT NULL,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE orders ADD COLUMN discount_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00;
ALTER TABLE orders ADD COLUMN promotion_code VARCHAR(50) NULL;
