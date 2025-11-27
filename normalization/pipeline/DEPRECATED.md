# ⚠️ DEPRECATED: Pipeline Processor

Этот файл `processor.go` помечен как устаревший и не используется в текущей кодовой базе.

## Статус

- **Статус:** DEPRECATED
- **Дата пометки:** 2025-11-20
- **Причина:** Нормализация реализована через `Normalizer` в `normalization/normalizer.go`

## Рекомендация

Этот код должен быть удален, если pipeline-based processing не планируется в будущем.

## Альтернатива

Используйте `normalization.Normalizer` для нормализации данных.

## Удаление

Если вы уверены, что этот код не нужен, выполните:

```bash
rm normalization/pipeline/processor.go
rm normalization/pipeline/DEPRECATED.md  # после удаления файла
```

