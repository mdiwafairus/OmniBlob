# Makefile for pwni-file-sync

.PHONY: build run

build:
	go build -o bin/pwni-file-sync ./cmd/pwni-file-sync

run:
	go run ./cmd/pwni-file-sync
