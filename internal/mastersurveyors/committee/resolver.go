// Package committee memenuhi seam mastersurveyors.CommitteeResolver.
//
// # Apa yang sebenarnya disimpan di kolom KOMITE
//
// Sistem lama mengisinya begini, pada langkah 7 `CNMInsertDetailSurveyors_act`:
//
//	TempDetailSurveyors.KOMITE := TempRDBSearchEmailKomite.pxResults(1).BUSINESS_CODE
//
// Nama `BUSINESS_CODE` MENYESATKAN, dan itu terbukti — bukan dugaan. Kueri yang mengisi
// halaman itu mengaliaskan kolomnya:
//
//	RDB List/EmailKomiteAdjuster_sql-SQL.xml
//	    SELECT EMAIL        as BRANCH_CODE,
//	           DEGREE       as BRANCH_NAME,
//	           OPERATOR_ID  as BUSINESS_CODE     <- INI yang masuk ke kolom KOMITE
//	      FROM POOLDATA.EMAILKOMITE
//	     WHERE STS_AKTIF = '1' ...
//	     ORDER BY DEGREE
//
// Jadi KOMITE berisi **OPERATOR_ID** anggota komite — bukan kode bisnis, bukan email,
// bukan derajat. Ketiga alias di kueri itu ketiganya salah arti sekaligus, persis pola
// utang teknis yang `docs/Steering/03-CURRENT-ARCHITECTURE.md` §4.2 catat.
//
// Mengetahui ini penting untuk layar: tab "Antrean Komite Saya" menyaring dengan
// membandingkan identitas pengguna yang sedang masuk terhadap kolom KOMITE, dan
// perbandingan itu hanya benar bila keduanya sama-sama Operator ID.
//
// # Yang TIDAK diketahui, dan karena itu tidak dikarang
//
// Kueri mana persisnya yang dipakai jalur surveyor TIDAK DAPAT DIBACA dari export.
// `GetKomiteApproval` meneruskan ke `SetEmailKomite`, yang memilih di antara beberapa
// kueri EMAILKOMITE berdasarkan `TYPE_BUSINESS` dan `TYPE_KOMITE` — dan nilai keduanya
// untuk jalur surveyor berasal dari halaman yang tidak ikut diekspor (`R-16`).
//
// Ketiga penyaring uang pada kueri adjuster (`LIMIT_BOTTOM <= nilai`) juga tidak dapat
// berlaku di sini: master surveyor TIDAK PUNYA nilai uang untuk dibandingkan. Penjenjangan
// kumulatif `D-47` karena itu tidak berlaku pada modul ini — yang dibutuhkan hanya SATU
// komite penentu, dan `pxResults(1)` di sistem lama memang hanya mengambil baris pertama.
//
// Pengisi seam di sini mengambil bentuk paling sederhana yang setia pada yang terbaca:
// baris aktif, diurutkan DEGREE, ambil OPERATOR_ID pertama. Bila Tim Pega kelak mengirim
// rule-nya dan penyaringnya ternyata lebih sempit, yang berubah hanya berkas ini.
//
// Lapisan Adapter — memenuhi interface yang dideklarasikan Domain.
package committee

import (
	"context"
	"strings"

	"claim-pnc/internal/mastersurveyors"
)

// Fixed adalah pengisi seam yang selalu mengembalikan identitas yang sama.
//
// Dipakai mode PENYIMPANAN=memori dan pengujian, supaya alur persetujuan dapat dijalankan
// tanpa basis data sama sekali.
type Fixed struct {
	// Identity adalah Operator ID komite yang dikembalikan. Boleh kosong — itu keadaan
	// sah yang berarti "tidak ada komite yang cocok", dan usecase menanganinya.
	Identity string
}

// Resolve mengembalikan identitas tetap, tanpa melihat surveyor maupun portal.
func (f Fixed) Resolve(_ context.Context, _ string, _ mastersurveyors.Surveyor) (string, error) {
	return strings.TrimSpace(f.Identity), nil
}
