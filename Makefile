# Makefile for pwni-file-sync

.PHONY: build run

build:
	go build -o bin/pwni-file-sync ./cmd/file-sync

run:
	go run ./cmd/file-sync
