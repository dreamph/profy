APP=profy

.PHONY: build
build:
	go build -o $(APP) .

.PHONY: print-dev
print-dev: build
	./$(APP) --print-env dev
