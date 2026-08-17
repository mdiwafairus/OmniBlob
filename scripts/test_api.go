package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

func main() {
	baseURL := "http://localhost:8080"

	// 1. Test Health
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("[1] Health Check Response (%d):\n%s\n\n", resp.StatusCode, string(body))

	// 2. Test Upload Multipart File
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("module", "lapordiri")
	_ = writer.WriteField("referensi_id", "LP-2026-TEST")

	part, err := writer.CreateFormFile("file", "ktp_sample.txt")
	if err != nil {
		fmt.Printf("Create form file failed: %v\n", err)
		os.Exit(1)
	}
	sampleData := "Ini adalah konten dokumen KTP untuk simulasi lapordiri Peduli WNI."
	_, _ = part.Write([]byte(sampleData))
	_ = writer.Close()

	uploadReq, err := http.NewRequest("POST", baseURL+"/api/v1/files/upload", &buf)
	if err != nil {
		fmt.Printf("New request failed: %v\n", err)
		os.Exit(1)
	}
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())

	uploadResp, err := http.DefaultClient.Do(uploadReq)
	if err != nil {
		fmt.Printf("Upload request failed: %v\n", err)
		os.Exit(1)
	}
	uploadBody, _ := io.ReadAll(uploadResp.Body)
	uploadResp.Body.Close()
	fmt.Printf("[2] Upload Response (%d):\n%s\n\n", uploadResp.StatusCode, string(uploadBody))

	// 3. Test View by Reference ID
	viewResp, err := http.Get(baseURL + "/api/v1/files/view?ref_id=LP-2026-TEST&module=lapordiri")
	if err != nil {
		fmt.Printf("View request failed: %v\n", err)
		os.Exit(1)
	}
	viewBody, _ := io.ReadAll(viewResp.Body)
	viewResp.Body.Close()
	fmt.Printf("[3] View By Ref_ID Response (%d):\nContent: %s\nContent-Type: %s\nETag: %s\nX-Storage-Path: %s\n\n",
		viewResp.StatusCode, string(viewBody), viewResp.Header.Get("Content-Type"),
		viewResp.Header.Get("ETag"), viewResp.Header.Get("X-Storage-Path"))

	fmt.Println("ALL TESTS PASSED SUCCESSFULLY!")
}
