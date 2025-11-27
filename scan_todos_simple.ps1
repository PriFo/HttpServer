# Простой скрипт для поиска TODO/FIXME/HACK в коде
# Использование: .\scan_todos_simple.ps1

Write-Host "🔍 Поиск TODO/FIXME/HACK в коде..." -ForegroundColor Cyan
Write-Host ""

$results = @()
$extensions = @("*.go", "*.ts", "*.tsx", "*.js", "*.jsx", "*.ps1", "*.sh")
$excludeDirs = @("node_modules", ".git", ".next", "dist", "build", "vendor", ".todos", "logs", "tmp", "checkpoints", "exports")

foreach ($ext in $extensions) {
    Get-ChildItem -Path . -Filter $ext -Recurse -ErrorAction SilentlyContinue | 
        Where-Object { 
            $excluded = $false
            foreach ($exDir in $excludeDirs) {
                if ($_.FullName -like "*\$exDir\*") {
                    $excluded = $true
                    break
                }
            }
            -not $excluded
        } | ForEach-Object {
            $content = Get-Content $_.FullName -ErrorAction SilentlyContinue
            $lineNum = 0
            foreach ($line in $content) {
                $lineNum++
                $todoMatch = $line -match 'TODO'
                $fixmeMatch = $line -match 'FIXME'
                $hackMatch = $line -match 'HACK'
                $bugMatch = $line -match 'BUG'
                $noteMatch = $line -match 'NOTE'
                
                if ($todoMatch -or $fixmeMatch -or $hackMatch -or $bugMatch -or $noteMatch) {
                    $priority = "LOW"
                    if ($line -match 'CRITICAL' -or $line -match 'panic' -or $line -match 'not.*implemented') {
                        $priority = "CRITICAL"
                    } elseif ($line -match 'HIGH' -or $line -match 'implement' -or $line -match 'not.*implemented') {
                        $priority = "HIGH"
                    } elseif ($line -match 'MEDIUM' -or $line -match 'optimize' -or $line -match 'refactor') {
                        $priority = "MEDIUM"
                    }
                    
                    $type = "TODO"
                    if ($fixmeMatch) { $type = "FIXME" }
                    elseif ($hackMatch) { $type = "HACK" }
                    elseif ($bugMatch) { $type = "BUG" }
                    elseif ($noteMatch) { $type = "NOTE" }
                    
                    $results += [PSCustomObject]@{
                        File = $_.FullName.Replace((Get-Location).Path + "\", "")
                        Line = $lineNum
                        Type = $type
                        Priority = $priority
                        Content = $line.Trim()
                    }
                }
            }
        }
}

Write-Host "📊 Найдено задач: $($results.Count)" -ForegroundColor Green
Write-Host ""

# Группируем по приоритету
$byPriority = $results | Group-Object Priority | Sort-Object Name -Descending

foreach ($group in $byPriority) {
    $color = switch ($group.Name) {
        "CRITICAL" { "Red" }
        "HIGH" { "Yellow" }
        "MEDIUM" { "Cyan" }
        default { "Gray" }
    }
    Write-Host "  $($group.Name): $($group.Count)" -ForegroundColor $color
}

Write-Host ""
Write-Host "📋 Детальный список:" -ForegroundColor Cyan
Write-Host ""

# Выводим результаты
$results | Sort-Object Priority, File, Line | ForEach-Object {
    $color = switch ($_.Priority) {
        "CRITICAL" { "Red" }
        "HIGH" { "Yellow" }
        "MEDIUM" { "Cyan" }
        default { "Gray" }
    }
    
    Write-Host "[$($_.Type)] [$($_.Priority)]" -ForegroundColor $color -NoNewline
    Write-Host " $($_.File):$($_.Line)" -ForegroundColor White
    Write-Host "  $($_.Content)" -ForegroundColor Gray
    Write-Host ""
}

# Сохраняем в файл
$reportFile = "TODO_SCAN_REPORT_$(Get-Date -Format 'yyyyMMdd_HHmmss').txt"
$results | Format-Table -AutoSize | Out-File $reportFile -Encoding UTF8
Write-Host "📄 Отчет сохранен: $reportFile" -ForegroundColor Green
