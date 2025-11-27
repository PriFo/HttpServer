#!/usr/bin/env pwsh
# PowerShell парсер ошибок компиляции Go и генератор промптов

param(
    [string]$ErrorFile = "build_errors_full.log"
)

if (-not (Test-Path $ErrorFile)) {
    Write-Host "❌ Файл $ErrorFile не найден!" -ForegroundColor Red
    Write-Host "Сначала выполните сборку проекта:" -ForegroundColor Yellow
    Write-Host "  go build ./... 2>&1 | Tee-Object -FilePath $ErrorFile" -ForegroundColor Yellow
    Write-Host "Или запустите: .\build-and-fix.ps1" -ForegroundColor Yellow
    exit 1
}

# Проверяем, что файл не пустой
$fileContent = Get-Content $ErrorFile -Raw
if ([string]::IsNullOrWhiteSpace($fileContent)) {
    Write-Host "⚠️  Файл $ErrorFile пуст. Проект собрался без ошибок!" -ForegroundColor Green
    exit 0
}

Write-Host "Парсинг ошибок из $ErrorFile..." -ForegroundColor Cyan

# Читаем файл с ошибками
$content = Get-Content $ErrorFile -Raw -Encoding UTF8
$lines = $content -split "`n"

# Структура для хранения ошибок
$errorsByFile = @{}
$currentFile = $null
$currentError = $null
$currentPackage = $null
$errorContext = @()

foreach ($line in $lines) {
    $trimmed = $line.Trim()
    
    if ([string]::IsNullOrWhiteSpace($trimmed)) {
        continue
    }
    
    # Заголовок пакета: # package_name
    if ($trimmed -match '^#\s+(.+)$') {
        $currentPackage = $matches[1]
        continue
    }
    
    # Паттерн для ошибки: file.go:line:column: message
    if ($trimmed -match '^([^:]+\.go):(\d+):(\d+):\s*(.+)$') {
        # Сохраняем предыдущую ошибку
        if ($currentFile -and $currentError) {
            if (-not $errorsByFile.ContainsKey($currentFile)) {
                $errorsByFile[$currentFile] = @()
            }
            
            $errorObj = @{
                Line = [int]$currentError.Line
                Column = [int]$currentError.Column
                Message = $currentError.Message
                Context = if ($errorContext.Count -gt 0) { ($errorContext -join "`n") } else { $null }
                Package = $currentPackage
            }
            $errorsByFile[$currentFile] += $errorObj
        }
        
        # Начинаем новую ошибку
        $currentFile = $matches[1]
        $currentError = @{
            Line = $matches[2]
            Column = $matches[3]
            Message = $matches[4]
        }
        $errorContext = @()
    }
    elseif (($trimmed -match '^\s+') -and $currentError) {
        # Дополнительный контекст ошибки
        $errorContext += $trimmed
    }
    elseif ($trimmed -eq "too many errors") {
        # Специальная строка "too many errors"
        if ($currentFile -and $currentError) {
            if (-not $errorsByFile.ContainsKey($currentFile)) {
                $errorsByFile[$currentFile] = @()
            }
            
            $errorObj = @{
                Line = [int]$currentError.Line
                Column = [int]$currentError.Column
                Message = "$($currentError.Message) (too many errors)"
                Context = if ($errorContext.Count -gt 0) { ($errorContext -join "`n") } else { $null }
                Package = $currentPackage
            }
            $errorsByFile[$currentFile] += $errorObj
        }
        $currentFile = $null
        $currentError = $null
        $errorContext = @()
    }
}

# Сохраняем последнюю ошибку
if ($currentFile -and $currentError) {
    if (-not $errorsByFile.ContainsKey($currentFile)) {
        $errorsByFile[$currentFile] = @()
    }
    
    $errorObj = @{
        Line = [int]$currentError.Line
        Column = [int]$currentError.Column
        Message = $currentError.Message
        Context = if ($errorContext.Count -gt 0) { ($errorContext -join "`n") } else { $null }
        Package = $currentPackage
    }
    $errorsByFile[$currentFile] += $errorObj
}

if ($errorsByFile.Count -eq 0) {
    Write-Host "✅ Ошибок компиляции не найдено! Проект собирается без ошибок." -ForegroundColor Green
    exit 0
}

# Нормализуем пути (заменяем обратные слеши на прямые)
function Normalize-Path {
    param([string]$path)
    return $path -replace '\\', '/'
}

# Функция для объяснения типа ошибки
function Get-ErrorExplanation {
    param([string]$errorMsg)
    
    $errorLower = $errorMsg.ToLower()
    
    if ($errorLower -match 'undefined') {
        return @"
Ошибка "undefined" означает, что компилятор не может найти определение символа (переменной, функции, типа, пакета).

Тонкости этой ошибки:
- Символ может быть не объявлен в текущем пакете или в импортированных пакетах
- Неправильный импорт пакета - возможно, пакет импортирован, но символ в нем не существует или имеет другое имя
- Символ может быть объявлен в другом файле того же пакета, но не экспортирован (начинается с маленькой буквы) - в Go экспортируются только символы с заглавной буквы
- Опечатка в имени символа - очень частая причина
- Файл с определением может не быть включен в компиляцию (не в той директории или не имеет правильного package)
- Тип может быть определен в другом пакете, но не импортирован
- Может быть проблема с циклическими зависимостями между пакетами
"@
    }
    
    if ($errorLower -match 'imported and not used') {
        return @"
Ошибка "imported and not used" означает, что пакет импортирован, но ни один его символ не используется в коде.

Тонкости этой ошибки:
- В Go неиспользуемые импорты считаются ошибкой компиляции
- Импорт может быть оставлен после рефакторинга, когда код, использующий пакет, был удален
- Может быть опечатка в имени импортируемого пакета
- Пакет может быть импортирован с алиасом, но алиас не используется
- Может быть импорт для side-effect (например, _ "package"), но забыт подчеркивающий символ
"@
    }
    
    if ($errorLower -match 'cannot use') {
        return @"
Ошибка "cannot use" означает несовместимость типов - переменная одного типа используется там, где ожидается другой тип.

Тонкости этой ошибки:
- Go строго типизированный язык, неявные преобразования типов ограничены
- Компилятор показывает, какой тип передан и какой ожидается
- Может потребоваться явное преобразование типа
- Может быть проблема с интерфейсами - тип не реализует требуемый интерфейс
- Может быть проблема с указателями - передается значение вместо указателя или наоборот
"@
    }
    
    if ($errorLower -match 'too many errors') {
        return @"
Сообщение "too many errors" означает, что компилятор обнаружил слишком много ошибок в файле и прекратил дальнейший анализ.

Тонкости этой ошибки:
- Это не отдельная ошибка, а индикатор того, что в файле накопилось слишком много ошибок
- Компилятор останавливается после определенного количества ошибок (обычно 10)
- Нужно исправить первые ошибки, чтобы увидеть остальные
- Часто первые ошибки вызывают каскад последующих (например, неопределенный тип вызывает ошибки везде, где он используется)
- Рекомендуется исправлять ошибки последовательно, начиная с первых
"@
    }
    
    return @"
Общая ошибка компиляции Go. 

Тонкости компиляции Go:
- Go компилятор строгий и требует явного объявления всех символов
- Порядок объявления важен - нельзя использовать символ до его объявления
- Экспорт символов зависит от регистра первой буквы
- Импорты должны использоваться, иначе это ошибка
- Типы должны совпадать точно, неявные преобразования ограничены
- Все переменные должны быть использованы (кроме _)
"@
}

# Генерируем промпты
$prompts = @()
$fileNames = $errorsByFile.Keys | Sort-Object

foreach ($filePath in $fileNames) {
    $errors = $errorsByFile[$filePath]
    $filePathNormalized = Normalize-Path $filePath
    
    # Собираем детали ошибок
    $errorDetails = @()
    for ($i = 0; $i -lt $errors.Count; $i++) {
        $err = $errors[$i]
        $detail = "Ошибка $($i + 1):`n"
        $detail += "  Строка: $($err.Line), Колонка: $($err.Column)`n"
        $detail += "  Сообщение компилятора: $($err.Message)`n"
        if ($err.Package) {
            $detail += "  Пакет: $($err.Package)`n"
        }
        if ($err.Context) {
            $detail += "  Дополнительный контекст:`n$($err.Context)`n"
        }
        $errorDetails += $detail
    }
    
    # Определяем типы ошибок для объяснений
    $explanations = @()
    $seenTypes = @{}
    foreach ($err in $errors) {
        $explanation = Get-ErrorExplanation -errorMsg $err.Message
        $explanationKey = $explanation.Substring(0, [Math]::Min(50, $explanation.Length))
        if (-not $seenTypes.ContainsKey($explanationKey)) {
            $seenTypes[$explanationKey] = $true
            $explanations += $explanation
        }
    }
    
    $prompt = @"
Исправь все ошибки компиляции в файле ``$filePathNormalized``.

## Файл для исправления:
``$filePathNormalized``

## Ошибки компилятора в этом файле:

$($errorDetails -join "`n")

## Объяснение типов ошибок (тонкости):

$($explanations -join "`n`n")

## Важные примечания:

- Некоторые зависимые файлы могли создаваться отдельно, т.е. Handler может зависеть от repository, но repository мог не существовать на момент написания handler

- Могут быть простые ошибки в наименовании файлов при импорте

- Но может быть действительно отсутствующий файл - убедись на 150%, что файл действительно отсутствует, прежде чем создавать его

- Проверь все импорты и убедись, что они указывают на существующие файлы с правильными именами

- Проверь после исправления компиляцию файла напрямую и, если есть ошибки - исправь их. Делай циклы "Проверка компиляции - исправление ошибки" пока не будет ошибок. Если после компиляции файла остались ошибки - сообщи об этом.

## Задача:

Проанализируй каждую ошибку компилятора, определи её причину, подумай над решением и исправь все ошибки в файле. 

ВАЖНО: Не исправляй файл построчно - исправь все ошибки в файле за один раз полностью. После исправления проверь компиляцию файла с помощью команды в терминале (PowerShell): ``go build ./path/to/file.go`` или ``go build ./package/path``. Если после исправления остались ошибки - исправь их в цикле до полного устранения всех ошибок компиляции.
"@
    
    $prompts += @{
        File = $filePathNormalized
        ErrorsCount = $errors.Count
        Prompt = $prompt
    }
}

# Сохраняем план
$planFile = "build_fix_plan.md"
$planContent = @"
# План исправления ошибок компиляции

Всего файлов с ошибками: $($prompts.Count)
Всего ошибок: $(($prompts | Measure-Object -Property ErrorsCount -Sum).Sum)
Всего промптов: $($prompts.Count)

**Примечание:** Шаги можно выполнять параллельно, так как каждый шаг исправляет отдельный файл.

---

"@

for ($i = 0; $i -lt $prompts.Count; $i++) {
    $item = $prompts[$i]
    $planContent += @"

## Шаг $($i + 1): Исправление ``$($item.File)``

**Количество ошибок в файле:** $($item.ErrorsCount)

**Промпт:**

``````
$($item.Prompt)
``````

---

"@
}

$planContent | Out-File -FilePath $planFile -Encoding UTF8 -NoNewline

Write-Host "`nПлан сохранен в $planFile" -ForegroundColor Green

# Сохраняем отдельные промпты
$promptsDir = "build_fix_prompts"
if (-not (Test-Path $promptsDir)) {
    New-Item -ItemType Directory -Path $promptsDir | Out-Null
}

for ($i = 0; $i -lt $prompts.Count; $i++) {
    $item = $prompts[$i]
    $safeName = [System.IO.Path]::GetFileNameWithoutExtension($item.File) -replace '[\\/]', '_'
    $promptFile = Join-Path $promptsDir "step_$($("{0:D2}" -f ($i + 1)))_$safeName.txt"
    $item.Prompt | Out-File -FilePath $promptFile -Encoding UTF8 -NoNewline
}

Write-Host "Отдельные промпты сохранены в директории $promptsDir/" -ForegroundColor Green

Write-Host "`nСтатистика:" -ForegroundColor Cyan
Write-Host "  - Файлов с ошибками: $($prompts.Count)" -ForegroundColor Yellow
Write-Host "  - Всего ошибок: $(($prompts | Measure-Object -Property ErrorsCount -Sum).Sum)" -ForegroundColor Yellow
Write-Host "  - Промптов создано: $($prompts.Count)" -ForegroundColor Yellow
