# ติดตั้ง service
nssm.exe install PhiDCNService

# ตั้งค่า path ของ executable
nssm.exe set PhiDCNService Application "C:\Path\To\phi-dcn-windows.exe"

# ตั้งค่า working directory
nssm.exe set PhiDCNService AppDirectory "C:\Path\To"

# ตั้งค่าให้รันอัตโนมัติเมื่อ startup
nssm.exe set PhiDCNService Start SERVICE_AUTO_START

# ตั้งค่า recovery options (restart เมื่อ crash)
nssm.exe set PhiDCNService AppExit Default Restart
nssm.exe set PhiDCNService AppRestartDelay 5000

# เริ่ม service
nssm.exe start PhiDCNService