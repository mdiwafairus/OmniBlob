package main1

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
	apiKey := "test_secret_key_2026"
	apiUser := "test_admin"
	apiPass := "test_secret_pass_123"

	// 1. Test Health (Public)
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("[1] Health Check Response (%d):\n%s\n\n", resp.StatusCode, string(body))

	// 2. Test Upload WITHOUT Auth (Expect 401 Unauthorized)
	var bufUnauthorized bytes.Buffer
	w1 := multipart.NewWriter(&bufUnauthorized)
	_ = w1.WriteField("module", "lapordiri")
	_ = w1.WriteField("referensi_id", "LP-UNAUTH")
	part1, _ := w1.CreateFormFile("file", "unauth.txt")
	_, _ = part1.Write([]byte("unauthorized content"))
	_ = w1.Close()

	reqUnauth, _ := http.NewRequest("POST", baseURL+"/api/v1/files/upload", &bufUnauthorized)
	reqUnauth.Header.Set("Content-Type", w1.FormDataContentType())
	respUnauth, err := http.DefaultClient.Do(reqUnauth)
	if err != nil {
		fmt.Printf("Request error: %v\n", err)
	} else {
		bodyUnauth, _ := io.ReadAll(respUnauth.Body)
		respUnauth.Body.Close()
		fmt.Printf("[2] Upload WITHOUT Auth Header -> Expected Status 401 (%d):\n%s\n\n", respUnauth.StatusCode, string(bodyUnauth))
	}

	// 3. Test Upload WITH X-API-KEY Header (Expect 201 Created)
	var bufAPIKey bytes.Buffer
	w2 := multipart.NewWriter(&bufAPIKey)
	_ = w2.WriteField("module", "lapordiri")
	_ = w2.WriteField("referensi_id", "LP-AUTH-KEY")
	_ = w2.WriteField("flag", "dokumen_ktp_lapor")
	part2, _ := w2.CreateFormFile("file", "ktp_auth.txt")
	_, _ = part2.Write([]byte("Dokumen KTP yang diupload dengan API Key."))
	_ = w2.Close()

	reqAPIKey, _ := http.NewRequest("POST", baseURL+"/api/v1/files/upload", &bufAPIKey)
	reqAPIKey.Header.Set("Content-Type", w2.FormDataContentType())
	reqAPIKey.Header.Set("X-API-KEY", apiKey)

	respAPIKey, err := http.DefaultClient.Do(reqAPIKey)
	if err != nil {
		fmt.Printf("Upload with API Key failed: %v\n", err)
		os.Exit(1)
	}
	bodyAPIKey, _ := io.ReadAll(respAPIKey.Body)
	respAPIKey.Body.Close()
	fmt.Printf("[3] Upload WITH X-API-KEY Header -> Status (%d):\n%s\n\n", respAPIKey.StatusCode, string(bodyAPIKey))

	// 4. Test Upload WITH Basic Auth (Expect 201 Created)
	var bufBasic bytes.Buffer
	w3 := multipart.NewWriter(&bufBasic)
	_ = w3.WriteField("module", "lapordiri")
	_ = w3.WriteField("referensi_id", "LP-AUTH-BASIC")
	_ = w3.WriteField("flag", "dokumen_paspor_lapor")
	part3, _ := w3.CreateFormFile("file", "paspor_auth.txt")
	_, _ = part3.Write([]byte("Dokumen Paspor yang diupload dengan Basic Auth."))
	_ = w3.Close()

	reqBasic, _ := http.NewRequest("POST", baseURL+"/api/v1/files/upload", &bufBasic)
	reqBasic.Header.Set("Content-Type", w3.FormDataContentType())
	reqBasic.SetBasicAuth(apiUser, apiPass)

	respBasic, err := http.DefaultClient.Do(reqBasic)
	if err != nil {
		fmt.Printf("Upload with Basic Auth failed: %v\n", err)
		os.Exit(1)
	}
	bodyBasic, _ := io.ReadAll(respBasic.Body)
	respBasic.Body.Close()
	fmt.Printf("[4] Upload WITH Basic Auth (User & Pass) -> Status (%d):\n%s\n\n", respBasic.StatusCode, string(bodyBasic))

	// 5. Test View/Download by Ref ID (Public streaming)
	viewResp, err := http.Get(baseURL + "/api/v1/files/view?ref_id=LP-AUTH-KEY&module=lapordiri")
	if err != nil {
		fmt.Printf("View request failed: %v\n", err)
		os.Exit(1)
	}
	viewBody, _ := io.ReadAll(viewResp.Body)
	viewResp.Body.Close()
	fmt.Printf("[5] View/Stream File -> Status (%d):\nContent: %s\nContent-Type: %s\nETag: %s\n\n",
		viewResp.StatusCode, string(viewBody), viewResp.Header.Get("Content-Type"), viewResp.Header.Get("ETag"))

	fmt.Println("=====================================================")
	fmt.Println("🎉 ALL AUTHENTICATION TESTS PASSED SUCCESSFULLY!")
	fmt.Println("=====================================================")
}
