# ตรวจสอบ Execution Policy
try {
    $executionPolicy = Get-ExecutionPolicy
    if ($executionPolicy -eq "Restricted") {
        Write-Host "⚠️ PowerShell Execution Policy is set to Restricted" -ForegroundColor Yellow
        Write-Host "To run this script, please use one of these methods:" -ForegroundColor Yellow
        Write-Host "1. Run PowerShell as Administrator and execute:" -ForegroundColor Yellow
        Write-Host "   Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass" -ForegroundColor Cyan
        Write-Host "   .\installService.ps1" -ForegroundColor Cyan
        Write-Host "2. Or run this command directly:" -ForegroundColor Yellow
        Write-Host "   powershell -ExecutionPolicy Bypass -File .\installService.ps1" -ForegroundColor Cyan
        exit 1
    }
} catch {
    Write-Host "⚠️ Error checking Execution Policy" -ForegroundColor Yellow
    Write-Host "Please run PowerShell as Administrator and execute:" -ForegroundColor Yellow
    Write-Host "Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass" -ForegroundColor Cyan
    Write-Host "Then run the script again." -ForegroundColor Yellow
    exit 1
}

# ตรวจสอบสิทธิ์ Administrator
$isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "โปรดรันสคริปต์นี้ด้วยสิทธิ์ Administrator" -ForegroundColor Red
    exit 1
}

# ตั้งค่าตัวแปร
$ServiceName = "PhiDCNService"
$InstallDir = "C:\Program Files (x86)\Phi DCN Bridge"
$AppPath = "$InstallDir\phi-dcn-bridge.exe"
$WorkingDir = $InstallDir
$CurrentDir = $PSScriptRoot

# สร้างโฟลเดอร์ติดตั้งถ้ายังไม่มี
try {
    if (-not (Test-Path $InstallDir)) {
        Write-Host "กำลังสร้างโฟลเดอร์ $InstallDir..." -ForegroundColor Yellow
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    # ตรวจสอบว่าสร้างโฟลเดอร์สำเร็จหรือไม่
    if (-not (Test-Path $InstallDir)) {
        Write-Host "❌ ไม่สามารถสร้างโฟลเดอร์ $InstallDir ได้" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "❌ เกิดข้อผิดพลาดในการสร้างโฟลเดอร์: $_" -ForegroundColor Red
    exit 1
}

# คัดลอกไฟล์ทั้งหมดจากโฟลเดอร์ปัจจุบันไปยังโฟลเดอร์ติดตั้ง
try {
    Write-Host "กำลังคัดลอกไฟล์ไปยัง $InstallDir..." -ForegroundColor Yellow
    Copy-Item -Path "$CurrentDir\*" -Destination $InstallDir -Recurse -Force
} catch {
    Write-Host "❌ เกิดข้อผิดพลาดในการคัดลอกไฟล์: $_" -ForegroundColor Red
    exit 1
}

# ตรวจสอบและติดตั้ง nssm.exe
$nssmPath = "$InstallDir\nssm.exe"
if (-not (Test-Path $nssmPath)) {
    try {
        Write-Host "กำลังดาวน์โหลด nssm.exe..." -ForegroundColor Yellow
        $nssmUrl = "https://nssm.cc/release/nssm-2.24.zip"
        $tempDir = "$env:TEMP\nssm"
        $zipPath = "$tempDir\nssm.zip"
        
        # สร้างโฟลเดอร์ชั่วคราว
        New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
        
        # ดาวน์โหลดไฟล์
        Invoke-WebRequest -Uri $nssmUrl -OutFile $zipPath
        
        # แยกไฟล์
        Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force
        
        # คัดลอก nssm.exe จากโฟลเดอร์ย่อย
        $nssmExe = Get-ChildItem -Path $tempDir -Recurse -Filter "nssm.exe" | Select-Object -First 1
        if ($nssmExe) {
            Copy-Item -Path $nssmExe.FullName -Destination $nssmPath -Force
        } else {
            Write-Host "❌ ไม่พบไฟล์ nssm.exe ในไฟล์ที่ดาวน์โหลดมา" -ForegroundColor Red
            exit 1
        }
        
        # ลบไฟล์ชั่วคราว
        Remove-Item -Path $tempDir -Recurse -Force
    } catch {
        Write-Host "❌ เกิดข้อผิดพลาดในการดาวน์โหลดหรือติดตั้ง nssm.exe: $_" -ForegroundColor Red
        exit 1
    }
}

# ตรวจสอบว่าไฟล์แอปพลิเคชันมีอยู่หรือไม่
if (-not (Test-Path $AppPath)) {
    Write-Host "❌ ไม่พบไฟล์แอปพลิเคชันที่ $AppPath" -ForegroundColor Red
    exit 1
}

# ติดตั้ง service
try {
    Write-Host "กำลังติดตั้ง service..." -ForegroundColor Yellow
    & $nssmPath install $ServiceName $AppPath
    
    # ตั้งค่า working directory
    Write-Host "กำลังตั้งค่า working directory..." -ForegroundColor Yellow
    & $nssmPath set $ServiceName AppDirectory $WorkingDir
    
    # ตั้งค่าให้รันอัตโนมัติเมื่อ startup
    Write-Host "กำลังตั้งค่าให้รันอัตโนมัติ..." -ForegroundColor Yellow
    & $nssmPath set $ServiceName Start SERVICE_AUTO_START
    
    # ตั้งค่า recovery options
    Write-Host "กำลังตั้งค่า recovery options..." -ForegroundColor Yellow
    & $nssmPath set $ServiceName AppExit Default Restart
    & $nssmPath set $ServiceName AppRestartDelay 5000
    
    # เริ่ม service
    Write-Host "กำลังเริ่ม service..." -ForegroundColor Yellow
    & $nssmPath start $ServiceName
} catch {
    Write-Host "❌ เกิดข้อผิดพลาดในการติดตั้งหรือตั้งค่า service: $_" -ForegroundColor Red
    exit 1
}

# ตรวจสอบสถานะ service
$service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($service -and $service.Status -eq "Running") {
    Write-Host "✅ ติดตั้ง service สำเร็จ!" -ForegroundColor Green
    Write-Host "Service Name: $ServiceName"
    Write-Host "Status: $($service.Status)"
    Write-Host "Start Type: $($service.StartType)"
    Write-Host "Installation Directory: $InstallDir"
} else {
    Write-Host "❌ ไม่สามารถติดตั้ง service ได้" -ForegroundColor Red
    exit 1
} 