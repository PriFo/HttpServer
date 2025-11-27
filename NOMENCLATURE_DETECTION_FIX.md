# Nomenclature Detection Fix

## Problem

Databases with "Номенклатура" (Nomenclature) in their filenames were showing 0 nomenclature records and all records counted as counterparties. This caused the pre-normalization statistics page to display incorrect data:

**Before Fix:**
- БухгалтерияДляКазахстана **Номенклатура**: 0 nomenclature, 16,065 counterparties ❌
- УправлениеПредприятиемДляКазахстана **Номенклатура**: 0 nomenclature, 995 counterparties ❌
- ERPWE **Номенклатура**: 0 nomenclature, 10,818 counterparties ❌

## Root Cause

Two `countDatabaseRecords` functions had fallback logic that treated ALL `catalog_items` table records as counterparties when the dedicated `counterparties` table didn't exist. The functions ignored:
1. Database filename type indicators (Номенклатура vs Контрагенты)
2. Catalog name from the `catalogs` table

## Solution

Modified both functions to intelligently determine data type:

### 1. `server/system_scanner.go:230` - `countDatabaseRecords`
### 2. `server/handlers/normalization.go:2335` - `countDatabaseRecords`

**Detection Strategy (in order):**
1. Check if `nomenclature_items` or `counterparties` tables exist (standard structure)
2. If `catalog_items` table exists:
   - **First**: Check `catalogs` table for catalog name ("Номенклатура" or "Контрагенты")
   - **Second**: Parse database filename using `database.ParseDatabaseFileInfo()`
   - **Third**: Count records in appropriate category based on determined type

## Changes Made

### Added Import
```go
import "path/filepath"
```

### Detection Logic
```go
// Определяем тип данных по каталогу или имени файла
dataType := ""

// Сначала пробуем определить по таблице catalogs
var catalogsTableExists bool
err = conn.QueryRowContext(ctx, `
    SELECT EXISTS (
        SELECT 1 FROM sqlite_master 
        WHERE type='table' AND name='catalogs'
    )
`).Scan(&catalogsTableExists)

if err == nil && catalogsTableExists {
    // Проверяем, есть ли каталог "Номенклатура"
    var nomenclatureCatalogExists bool
    err = conn.QueryRowContext(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM catalogs 
            WHERE name = 'Номенклатура' OR name LIKE '%оменклатур%'
        )
    `).Scan(&nomenclatureCatalogExists)
    if err == nil && nomenclatureCatalogExists {
        dataType = "nomenclature"
    } else {
        // Проверяем, есть ли каталог "Контрагенты"
        var counterpartyCatalogExists bool
        err = conn.QueryRowContext(ctx, `
            SELECT EXISTS (
                SELECT 1 FROM catalogs 
                WHERE name = 'Контрагенты' OR name LIKE '%онтрагент%'
            )
        `).Scan(&counterpartyCatalogExists)
        if err == nil && counterpartyCatalogExists {
            dataType = "counterparties"
        }
    }
}

// Если не удалось определить по каталогу, используем имя файла
if dataType == "" {
    fileName := filepath.Base(dbFilePath)
    fileInfo := database.ParseDatabaseFileInfo(fileName)
    dataType = fileInfo.DataType
}

// Подсчитываем записи в зависимости от типа данных
var catalogItemsCount int64
err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_items").Scan(&catalogItemsCount)
if err == nil {
    if dataType == "nomenclature" {
        nomenclatureCount = catalogItemsCount
    } else {
        counterpartyCount = catalogItemsCount
    }
}
```

## Expected Result

**After Fix:**
- БухгалтерияДляКазахстана **Номенклатура**: 16,065 nomenclature, 0 counterparties ✅
- УправлениеПредприятиемДляКазахстана **Номенклатура**: 995 nomenclature, 0 counterparties ✅
- ERPWE **Номенклатура**: 10,818 nomenclature, 0 counterparties ✅
- Total nomenclature: ~27,878 records (instead of 0)

## Testing

To verify the fix:

1. **Restart the server** to apply the changes
2. Navigate to the normalization preview statistics page
3. Check that databases with "Номенклатура" in their names now show:
   - **Non-zero nomenclature counts**
   - **Zero or correct counterparty counts**
4. Verify the total statistics at the top show correct nomenclature count

## Files Modified

1. `server/system_scanner.go` - Updated `countDatabaseRecords()` function
   - Added `path/filepath` import
   - Added catalog-based and filename-based type detection
   - Added conditional counting based on data type

2. `server/handlers/normalization.go` - Updated `countDatabaseRecords()` function
   - Added catalog-based and filename-based type detection
   - Added conditional counting based on data type
   - Enhanced existing catalog checking logic

## Verification Commands

```powershell
# Build to check for compilation errors
go build ./database

# Check if database utils are accessible
go list -f '{{.Dir}}' -m httpserver

# Restart server and test the statistics endpoint
curl http://localhost:9999/api/clients/{clientId}/projects/{projectId}/normalization/preview-stats
```

## Impact

- ✅ Fixes incorrect record categorization
- ✅ Provides accurate pre-normalization statistics
- ✅ No breaking changes to existing functionality
- ✅ Backwards compatible (works with both old and new database structures)

## Date

2025-11-25

