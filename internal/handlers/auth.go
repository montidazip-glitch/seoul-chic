package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"kfry_shop/internal/middleware"
	"kfry_shop/internal/models"
	"kfry_shop/internal/session"
)

// AuthHandler holds dependencies for auth routes.
type AuthHandler struct {
	DB   *sql.DB
	Tmpl TemplateRenderer
}

func (h *AuthHandler) RegisterGET(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":     "สมัครสมาชิก",
		"CSRFToken": middleware.GetCSRFToken(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "register.html", data)
}

func (h *AuthHandler) RegisterPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	// Validation
	if name == "" || email == "" || password == "" {
		session.SetFlash(r, w, "error", "กรุณากรอกข้อมูลให้ครบถ้วน")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}
	if password != confirm {
		session.SetFlash(r, w, "error", "รหัสผ่านไม่ตรงกัน")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}
	if len(password) < 8 {
		session.SetFlash(r, w, "error", "รหัสผ่านต้องมีอย่างน้อย 8 ตัวอักษร")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}
	if models.EmailExists(h.DB, email) {
		session.SetFlash(r, w, "error", "อีเมลนี้ถูกใช้งานแล้ว")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	if err := models.CreateUser(h.DB, name, email, password, "customer"); err != nil {
		session.SetFlash(r, w, "error", "เกิดข้อผิดพลาด กรุณาลองใหม่")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	session.SetFlash(r, w, "success", "สมัครสมาชิกสำเร็จ! กรุณาเข้าสู่ระบบ")
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (h *AuthHandler) LoginGET(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect appropriately
	if id := session.GetUserID(r); id != 0 {
		if session.GetUserRole(r) == "admin" {
			http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		} else {
			http.Redirect(w, r, "/menu", http.StatusFound)
		}
		return
	}
	data := map[string]interface{}{
		"Title":     "เข้าสู่ระบบ",
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "login.html", data)
}

func (h *AuthHandler) AdminLoginGET(w http.ResponseWriter, r *http.Request) {
	// If already logged in as admin, redirect to admin dashboard
	if id := session.GetUserID(r); id != 0 {
		if session.GetUserRole(r) == "admin" {
			http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
			return
		}
	}
	data := map[string]interface{}{
		"Title":     "เข้าสู่ระบบ Admin",
		"CSRFToken": middleware.GetCSRFToken(r, w),
		"Flashes":   flashData(r, w),
	}
	h.Tmpl.ExecuteTemplate(w, "admin_login.html", data)
}

func (h *AuthHandler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	user, err := models.GetUserByEmail(h.DB, email)
	if err != nil || !models.CheckPassword(user.PasswordHash, password) {
		session.SetFlash(r, w, "error", "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	session.SetUser(r, w, user.ID, user.Role)
	session.SetFlash(r, w, "success", "เข้าสู่ระบบสำเร็จ! ยินดีต้อนรับ "+user.Name)

	if user.Role == "admin" {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	} else {
		http.Redirect(w, r, "/menu", http.StatusFound)
	}
}

func (h *AuthHandler) LogoutPOST(w http.ResponseWriter, r *http.Request) {
	session.ClearUser(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}
