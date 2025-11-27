.PHONY: swagger build build-no-gui build-gui run run-no-gui test clean

# Генерация Swagger документации
swagger:
	@echo "Generating Swagger documentation..."
	@if [ -f "main_no_gui.go" ]; then \
		swag init -g main_no_gui.go -o ./docs; \
	elif [ -f "cmd/server/main.go" ]; then \
		swag init -g cmd/server/main.go -o ./docs; \
	else \
		echo "Warning: main_no_gui.go or cmd/server/main.go not found, skipping swagger generation"; \
	fi

# Сборка приложения без GUI (production)
build-no-gui:
	@echo "Building application (no GUI)..."
	@mkdir -p ./bin
	CGO_ENABLED=1 go build -tags no_gui -o ./bin/httpserver_no_gui.exe main_no_gui.go

# Сборка приложения с GUI
build-gui:
	@echo "Building application (with GUI)..."
	@mkdir -p ./bin
	CGO_ENABLED=1 go build -o ./bin/httpserver.exe ./cmd/server/main.go

# Сборка по умолчанию (без GUI)
build: build-no-gui

# Запуск приложения без GUI
run-no-gui:
	@echo "Running application (no GUI)..."
	CGO_ENABLED=1 go run -tags no_gui main_no_gui.go

# Запуск приложения с GUI
run-gui:
	@echo "Running application (with GUI)..."
	CGO_ENABLED=1 go run ./cmd/server/main.go

# Запуск приложения (по умолчанию без GUI)
run: run-no-gui

# Запуск тестов
test:
	@echo "Running tests..."
	CGO_ENABLED=1 go test ./...

# Сборка инструмента из папки tools
# Использование: make build-tool TOOL=analyze_cached_metadata
build-tool:
	@if [ -z "$(TOOL)" ]; then \
		echo "Error: TOOL variable is required. Usage: make build-tool TOOL=analyze_cached_metadata"; \
		exit 1; \
	fi
	@echo "Building tool: $(TOOL)..."
	@mkdir -p ./bin
	@CGO_ENABLED=1 go build -tags tool_$(TOOL) -o ./bin/$(TOOL).exe ./tools/$(TOOL).go
	@echo "Built: ./bin/$(TOOL).exe"

# Запуск инструмента из папки tools
# Использование: make run-tool TOOL=analyze_cached_metadata
run-tool:
	@if [ -z "$(TOOL)" ]; then \
		echo "Error: TOOL variable is required. Usage: make run-tool TOOL=analyze_cached_metadata"; \
		exit 1; \
	fi
	@echo "Running tool: $(TOOL)..."
	@CGO_ENABLED=1 go run -tags tool_$(TOOL) ./tools/$(TOOL).go

# Очистка
clean:
	@echo "Cleaning..."
	rm -rf ./bin
	rm -rf ./docs/swagger.json
	rm -rf ./docs/swagger.yaml
	rm -f ./httpserver_no_gui.exe
	rm -f ./httpserver.exe

