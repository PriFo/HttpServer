# Быстрое исправление команды Claude

## Проблема
Команда `claude` не распознается в PowerShell.

## Решение 1: Для текущей сессии (быстро)

Выполните в PowerShell:

```powershell
. .\init-claude.ps1
```

После этого команда `claude` будет работать в текущей сессии:

```powershell
claude help
claude server
claude build
claude status
```

## Решение 2: Постоянное исправление профиля

Выполните:

```powershell
.\fix-claude-profile.ps1
```

Затем перезапустите PowerShell или выполните:

```powershell
. $PROFILE
```

## Решение 3: Использование напрямую (без настройки)

Вы можете использовать скрипт напрямую:

```powershell
.\claude.ps1 help
.\claude.ps1 server
.\claude.ps1 build
.\claude.ps1 status
```

Или через batch-файл:

```powershell
.\claude.bat help
.\claude.bat server
```

## Рекомендация

Для быстрого старта выполните:

```powershell
. .\init-claude.ps1
```

Это загрузит функцию в текущую сессию, и вы сможете использовать `claude` сразу.

