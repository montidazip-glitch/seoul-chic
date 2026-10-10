@echo off
chcp 65001 >nul
echo ========================================================
echo  🍗 Seoul Chic — K-Fry Shop Management System
echo ========================================================
echo.

REM 1. Check if MySQL is running
netstat -ano | findstr ":3306" >nul
if %errorlevel% neq 0 (
    echo [i] กำลังตรวจสอบบริการ MySQL...
    if exist "C:\xampp\mysql_start.bat" (
        echo [i] เริ่มต้น MySQL จาก XAMPP อัตโนมัติ...
        start "" /b "C:\xampp\mysql_start.bat"
        timeout /t 3 /nobreak >nul
    ) else (
        echo [!] คำเตือน: ยังไม่ได้เปิดบริการ MySQL บนพอร์ต 3306
        echo     กรุณาเปิด XAMPP Control Panel แล้วกด 'Start' ที่ MySQL
        echo.
    )
) else (
    echo [✓] MySQL กำลังทำงานอยู่บนพอร์ต 3306
)

echo.
echo [i] กำลังเริ่มรันเซิร์ฟเวอร์ Seoul Chic...
echo [i] บัญชี Admin สำหรับเข้าสู่ระบบ:
echo     Email:    admin@seoulchic.com
echo     Password: admin1234
echo.
echo เข้าใช้งานได้ที่: http://localhost:8080
echo.
go run ./cmd/server
pause
