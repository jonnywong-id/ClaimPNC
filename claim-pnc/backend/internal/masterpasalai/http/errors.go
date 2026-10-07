package masterpasalaihttp

import "claim-pnc/internal/platform/apierror"

// Kode galat modul ini.
//
// # Hanya SATU, dan itu bukan kekurangan
//
// Modul ini baca-saja: tidak ada isian yang dapat cacat, tidak ada baris yang dapat bentrok,
// dan tidak ada penyimpanan yang dapat gagal separuh. Yang tersisa hanyalah permintaan yang
// bentuknya salah.
//
// Galat PORTAL dipetakan `portalhttp.WithPortalError` yang membungkus penulis galat dari cmd
// — satu pemetaan yang dipakai seluruh modul bisnis, bukan satu tafsiran per modul. Galat
// lain — kegagalan basis data, kegagalan jaringan, cacat pemrograman — diserahkan ke penulis
// bersama, yang menjawab 500 dengan pesan umum dan menaruh rinciannya di log saja.
const (
	CodeMalformedRequest = "permintaan_cacat"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter
