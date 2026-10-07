package memory

import (
	_ "embed"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// sampleRows adalah contoh data untuk mode PENYIMPANAN=memori dan pengujian.
//
// # Seluruhnya KARANGAN, dan itu disengaja
//
// Tidak satu pun baris di bawah disalin dari basis data mana pun. `D-69` melarang data
// nasabah masuk ke berkas yang di-commit, dan walau surveyor bukan nasabah, nama orang
// sungguhan beserta nomor telepon dan alamatnya tidak punya alasan berada di repositori.
//
// Alamat surel memakai domain `contoh.invalid` — akhiran `.invalid` dicadangkan RFC 2606
// dan dijamin tidak pernah dapat diselesaikan, sehingga percobaan pengiriman surel dari
// lingkungan pengembangan tidak mungkin tiba di kotak surat siapa pun.
//
// # Kenapa contohnya mencakup keempat posisi persetujuan
//
// Supaya kelima tab layar dapat dibuka dan dilihat isinya tanpa harus membuat data lebih
// dulu — termasuk tab "Antrean Komite Saya", yang tanpa baris berstatus menunggu akan
// selalu tampak kosong dan membuat orang menyangka layarnya rusak.
//
// Kode tipe merujuk keempat baris nyata pada POOLDATA.M_SURVEYORS, yang sudah diverifikasi
// modul induk pada 2026-09-19:
//
//	1001  INTERNAL SURVEYOR
//	1002  LOSS ADJUSTER
//	1003  EXPERT
//	1004  SURVEY AGENT
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Surveyor internal: satu-satunya golongan yang WAJIB punya login aplikasi.
// Loss adjuster eksternal: tanpa login aplikasi, dan itu SAH.
func sampleRows() []mastersurveyors.Surveyor {
	return sampledata.Must[[]mastersurveyors.Surveyor](sampleJSON, "sampleRows")
}
