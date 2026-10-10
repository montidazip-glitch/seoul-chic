# Implementation Plan: ระบบจัดการร้านขายไก่ทอดเกาหลี

## 1. Project Overview

สร้างเว็บ *ระบบจัดการร้านขายไก่ทอดเกาหลี (K-Fry Shop Management System)* สำหรับเจ้าของร้านและลูกค้า เพื่อจัดการเมนูสินค้า คำสั่งซื้อ และติดตามสถานะการสั่งซื้อ โดยลูกค้าสามารถเลือกวิธีรับสินค้าได้ทั้งรับที่ร้านและจัดส่ง

## 2. Technology Stack

* Frontend: HTML + CSS + Bootstrap + JavaScript
* Backend: Go (Golang)
* Database: MySQL / MariaDB
* Security: OWASP Top 10

---

## 3. User Roles

### Admin (เจ้าของร้าน)

* สร้างเมนูสินค้า
* แก้ไขข้อมูลเมนูสินค้า
* เพิ่ม/ลบสินค้า
* จัดการข้อมูลลูกค้า
* ดูรายการคำสั่งซื้อทั้งหมด
* สร้าง/แก้ไขข้อมูลโปรโมชั่น
* เปลี่ยนสถานะคำสั่งซื้อ
* ดูรายงานยอดขาย

### Customer (ลูกค้า)

* สมัครสมาชิกและเข้าสู่ระบบ
* ดูรายการเมนูไก่ทอดเกาหลี
* ดูรายละเอียดและราคาสินค้า
* สั่งซื้อสินค้า
* เลือกวิธีรับสินค้า (รับที่ร้าน/จัดส่ง)
* ดูสถานะคำสั่งซื้อของตนเอง
* ยกเลิกคำสั่งซื้อของตนเองตามเงื่อนไขที่กำหนด

---

## 4. Core Features

### Authentication

* Register
* Login
* Logout
* Password Hashing
* Session Management

### Shop Management

Products  
├── Product Information  
├── Product Categories  
├── Product Prices  
└── Product Availability  

Orders  
├── Customer Information  
├── Order Items  
├── Payment Information  
└── Order Status  

### Order Management

* Create Order
* View Order
* Edit Order
* Cancel/Delete Order
* Track Order Status
* View Order History

### Dashboard

Admin:

จำนวนสินค้า  
จำนวนคำสั่งซื้อ  
จำนวนลูกค้า  
คำสั่งซื้อที่รอดำเนินการ  
ยอดขายรวม  

Customer:

รายการเมนูสินค้า  
คำสั่งซื้อของฉัน  
คำสั่งซื้อที่กำลังดำเนินการ  
คำสั่งซื้อที่เสร็จสิ้น  
คำสั่งซื้อที่ยกเลิก  

---

## 5. Suggested Database

users
- id
- name
- email
- password_hash
- role

products
- id
- name
- description
- price
- category
- stock
- image_path
- status

orders
- id
- customer_id
- order_type
- delivery_address
- total_price
- status
- created_at

order_items
- id
- order_id
- product_id
- quantity
- unit_price

payments
- id
- order_id
- payment_method
- payment_status
- paid_at

Relationship หลัก:

Admin  
   └── Products  
         ├── Product Information  
         ├── Product Categories  
         └── Product Availability  

Customer  
   └── Orders  
         ├── Order Items  
         │     └── Products  
         ├── Payments  
         └── Order Status  

---

## 6. Security (OWASP Top 10)

* Password Hashing เพื่อจัดเก็บรหัสผ่านอย่างปลอดภัย
* Session Management เพื่อจัดการการเข้าสู่ระบบ
* Access Control เพื่อป้องกันลูกค้าเข้าถึงหน้าผู้ดูแลระบบ
* SQL Injection Prevention โดยใช้ Prepared Statements
* XSS Prevention โดยตรวจสอบและ Escape ข้อมูลก่อนแสดงผล
* CSRF Protection เพื่อป้องกันการส่งคำขอปลอม
* Input Validation เพื่อตรวจสอบข้อมูลที่รับจากผู้ใช้
* Security Headers เพื่อเพิ่มความปลอดภัยของเว็บไซต์
* Error Handling เพื่อไม่เปิดเผยข้อมูลภายในระบบ
* HTTPS เพื่อเข้ารหัสข้อมูลระหว่างรับส่งผ่านเครือข่าย

---

## 7. ข้อมูลบัญชีผู้ดูแลระบบ (Admin Credentials)

ระบบได้ทำการสร้างบัญชีผู้ดูแลระบบเริ่มต้นให้อัตโนมัติ:

| รายการ | ค่าเริ่มต้น |
|---|---|
| **อีเมล (Email)** | `admin@seoulchic.com` |
| **รหัสผ่าน (Password)** | `admin1234` |
| **สิทธิ์ (Role)** | `admin` |
| **ลิงก์เข้าสู่ระบบ Admin** | `/admin/login` |

*(ในหน้า `/admin/login` มีตัวอย่าง user admin และ password พร้อมปุ่ม "คลิกเพื่อกรอกอัตโนมัติ" ให้ทดลองเข้าระบบได้ทันที)*

---

## 8. วิธีการรันบนเครื่อง Localhost

### วิธีที่ 1: ดับเบิลคลิกไฟล์ `run_server.bat` (ง่ายที่สุด)
1. ดับเบิลคลิกที่ไฟล์ [run_server.bat](file:///d:/go/SEOUL%20CHIC/run_server.bat)
2. สคริปต์จะตรวจสอบ MySQL และเปิดเซิร์ฟเวอร์ให้อัตโนมัติ
3. เปิดเบราว์เซอร์ไปที่ `http://localhost:8080`

### วิธีที่ 2: รันผ่านคำสั่ง Go
1. เปิด **XAMPP Control Panel** แล้วกด **Start** ที่บริการ **MySQL** (พอร์ต 3306)
2. รันคำสั่ง:
   ```bash
   go run ./cmd/server
   ```
3. ระบบจะสร้างฐานข้อมูล `kfry_shop`, Migrations และ Seed ข้อมูล Admin ให้อัตโนมัติ

---

## 9. วิธีการนำเว็บขึ้นโฮสต์ (Deploy to Cloud / Hosting)

โปรเจกต์รองรับการ Deploy ทั้งแบบ **Docker** และ **Platform as a Service (Render, Railway, Fly.io, VPS)**:

1. **Environment Variables ที่ต้องตั้งค่าบนโฮสต์:**
   * `PORT`: พอร์ตที่โฮสต์กำหนด (ระบบจะตรวจจับอัตโนมัติ)
   * `DB_DSN` หรือ `DATABASE_URL`: Connection String ของ MySQL (รองรับทั้งฟอร์แมต `user:pass@tcp(host:port)/dbname` และ `mysql://user:pass@host:port/dbname`)
   * `SESSION_SECRET`: สตริงสุ่มสำหรับ Session ความยาว 32 ตัวอักษรขึ้นไป
2. **การ Deploy ด้วย Dockerfile:**
   * ใช้ไฟล์ [Dockerfile](file:///d:/go/SEOUL%20CHIC/Dockerfile) ที่เตรียมไว้ สามารถเชื่อมต่อกับ Git Repo แล้ว Deploy บน Render, Railway หรือ Docker Container ได้ทันที