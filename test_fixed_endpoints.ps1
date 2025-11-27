# Тестирование исправленных эндпоинтов (PowerShell версия)
# Использование: .\test_fixed_endpoints.ps1 [PORT]
# По умолчанию: PORT=9999

param(
    [int]$Port = 9999
)

$BaseUrl = "http://localhost:$Port"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Тестирование исправленных эндпоинтов" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# Функция для тестирования эндпоинта
function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Url,
        [int]$ExpectedStatus = 200
    )
    
    Write-Host -NoNewline "Тест: $Name ... "
    
    try {
        $response = Invoke-WebRequest -Uri $Url -Method Get -TimeoutSec 7 -UseBasicParsing -ErrorAction Stop
        $httpCode = $response.StatusCode
        $body = $response.Content
        
        if ($httpCode -eq $ExpectedStatus) {
            Write-Host "✓ Success $httpCode" -ForegroundColor Green
            if ($body -and $body.Length -gt 0 -and $body -ne "null") {
                $preview = if ($body.Length -gt 200) { $body.Substring(0, 200) + "..." } else { $body }
                Write-Host $preview -ForegroundColor Gray
            }
            return $true
        } else {
            Write-Host "✗ Failed $httpCode (expected $ExpectedStatus)" -ForegroundColor Red
            if ($body) {
                Write-Host "Response: $body" -ForegroundColor Yellow
            }
            return $false
        }
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        Write-Host "✗ Failed $statusCode" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Yellow
        return $false
    }
}

# Тест 1: GET /api/databases/find-project
Write-Host "1. Testing GET /api/databases/find-project" -ForegroundColor Yellow
$filePath = [System.Web.HttpUtility]::UrlEncode("Выгрузка_Номенклатура_ERPWE_Unknown_Unknown_2025_11_20_10_18_55.db")
Test-Endpoint -Name "Find project by database path" `
    -Url "$BaseUrl/api/databases/find-project?file_path=$filePath" `
    -ExpectedStatus 200
Write-Host ""

# Тест 2: GET /api/databases/find-project (несуществующая БД)
Write-Host "2. Testing GET /api/databases/find-project (non-existent DB)" -ForegroundColor Yellow
Test-Endpoint -Name "Find project by non-existent database path" `
    -Url "$BaseUrl/api/databases/find-project?file_path=non_existent.db" `
    -ExpectedStatus 404
Write-Host ""

# Тест 3: GET /api/databases/find-project (без параметра)
Write-Host "3. Testing GET /api/databases/find-project (missing parameter)" -ForegroundColor Yellow
Test-Endpoint -Name "Find project without file_path parameter" `
    -Url "$BaseUrl/api/databases/find-project" `
    -ExpectedStatus 400
Write-Host ""

# Тест 4: GET /api/normalization/status
Write-Host "4. Testing GET /api/normalization/status" -ForegroundColor Yellow
Test-Endpoint -Name "Get normalization status" `
    -Url "$BaseUrl/api/normalization/status" `
    -ExpectedStatus 200
Write-Host ""

# Тест 5: GET /api/dashboard/normalization-status
Write-Host "5. Testing GET /api/dashboard/normalization-status" -ForegroundColor Yellow
Test-Endpoint -Name "Get dashboard normalization status" `
    -Url "$BaseUrl/api/dashboard/normalization-status" `
    -ExpectedStatus 200
Write-Host ""

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Тестирование завершено" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

