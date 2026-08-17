<?php
defined('BASEPATH') OR exit('No direct script access allowed');

/**
 * PWNI Storage Helper for CodeIgniter 3
 * 
 * Helper untuk menghubungkan FE PWNI, BE PWNI, dan Aplikasi Lain 
 * ke pwni-file-sync Storage Service (menggantikan direct NFS filesystem call).
 */

if (!function_exists('pwni_storage_base_url')) {
    function pwni_storage_base_url() {
        // Ganti dengan IP server storage .134 pada production (misal: http://10.128.150.134:8080)
        return 'http://127.0.0.1:8080';
    }
}

/**
 * Upload file dari CodeIgniter 3 ke pwni-file-sync service
 *
 * @param string $module Contoh: 'lapordiri', 'paspor', 'verifikasi'
 * @param string $referensi_id ID referensi aplikasi (misal: 'LP-2026-001')
 * @param string $file_field_name Nama form input file di $_FILES
 * @return array Response JSON dari storage service
 */
if (!function_exists('pwni_upload_file')) {
    function pwni_upload_file($module, $referensi_id, $file_field_name = 'file') {
        if (!isset($_FILES[$file_field_name]) || empty($_FILES[$file_field_name]['tmp_name'])) {
            return ['success' => false, 'error' => 'No file uploaded in form field: ' . $file_field_name];
        }

        $tmpPath = $_FILES[$file_field_name]['tmp_name'];
        $originalName = $_FILES[$file_field_name]['name'];
        $mimeType = $_FILES[$file_field_name]['type'] ?: mime_content_type($tmpPath);

        $url = pwni_storage_base_url() . '/api/v1/files/upload';

        $cfile = new CURLFile($tmpPath, $mimeType, $originalName);
        $postData = [
            'module' => $module,
            'referensi_id' => $referensi_id,
            'file' => $cfile
        ];

        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_POST, 1);
        curl_setopt($ch, CURLOPT_POSTFIELDS, $postData);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 30); // 30s timeout

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $curlError = curl_error($ch);
        curl_close($ch);

        if ($curlError) {
            return ['success' => false, 'error' => 'cURL Error: ' . $curlError];
        }

        $result = json_decode($response, true);
        return $result ?: ['success' => false, 'error' => 'Invalid response from storage service'];
    }
}

/**
 * Dapatkan URL streaming file berdasarkan bin_id
 *
 * @param int $bin_id
 * @return string URL streaming langsung (bisa dipasang di <img src="..."> atau <a href="...">)
 */
if (!function_exists('pwni_get_file_url')) {
    function pwni_get_file_url($bin_id) {
        return pwni_storage_base_url() . '/api/v1/files/' . intval($bin_id);
    }
}

/**
 * Dapatkan URL streaming file berdasarkan referensi_id & modul
 *
 * @param string $referensi_id
 * @param string $module
 * @return string
 */
if (!function_exists('pwni_get_view_url')) {
    function pwni_get_view_url($referensi_id, $module = '') {
        return pwni_storage_base_url() . '/api/v1/files/view?ref_id=' . urlencode($referensi_id) . ($module ? '&module=' . urlencode($module) : '');
    }
}
