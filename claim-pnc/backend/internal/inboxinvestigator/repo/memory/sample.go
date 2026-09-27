package memory

import (
	"time"

	"claim-pnc/internal/inboxinvestigator"
)

// SampleTasks mengembalikan antrean contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh isinya KARANGAN, dan itu wajib
//
// Tidak ada satu pun nomor polis, nama tertanggung, nama peserta, maupun nomor klaim nyata
// di sini. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan itu
// berlaku pada data contoh persis seperti pada dokumen.
//
// Nomor klaimnya memakai kedua format yang hidup berdampingan selama masa paralel —
// `PNC-xxxx` dari Pega dan `PNCN.YY.xxxx` dari sistem baru (`D-71`) — supaya layar terbukti
// menampilkan keduanya tanpa membedakan.
//
// # Yang sengaja diwakili
//
// Kedelapan baris di bawah dipilih supaya setiap keadaan yang dapat dihadapi layar muncul
// sekurang-kurangnya sekali:
//
//	baris 1   lengkap, baru masuk beberapa jam
//	baris 2   lengkap, tanggal survei beberapa hari lalu
//	baris 3   TANPA tanggal survei                    -> kolom kesembilan KOSONG
//	baris 4   TANPA Nama Peserta                      -> klaim yang objeknya belum terisi
//	baris 5   nomor klaim format baru PNCN.YY.xxxx
//	baris 6   Nama Admin kosong                       -> kasus yang dibuat proses, bukan orang
//	baris 7   tanggal survei jauh di belakang
//	baris 8   Tanggal Pendaftaran kosong              -> kolom tanggal yang dapat NULL
//
// Baris 3 yang paling perlu ada: ia satu-satunya yang membuktikan klaim tanpa baris survei
// TETAP tampil di antrean — hanya kolom kesembilannya yang kosong. Tanpa baris itu, jalur
// tersebut tidak akan pernah tercoba.
//
// # Seluruhnya SUDAH merupakan antrean investigator yang belum selesai
//
// Tidak ada baris yang berstatus selesai dan tidak ada yang berada di workbasket lain,
// karena kedua penyaring itu hidup di dalam kueri SQL — bukan di dalam Filter. Repo memori
// karena itu menganggap isinya sudah tersaring; lihat catatan pada Repo.List.
func SampleTasks() []inboxinvestigator.Task {
	return []inboxinvestigator.Task{
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-100241",
			CaseNumber:      "PNC-100241",
			PolicyNumber:    "00.000.2026.00001",
			InsuredName:     "PT Contoh Sejahtera Abadi",
			ParticipantName: "Peserta Contoh Satu",
			BusinessName:    "Personal Accident",
			BranchName:      "Contoh Pusat",
			AdminName:       "ADMINCONTOH1",
			RegisteredAt:    at("2026-09-21T02:15:00Z"),
			SurveyDate:      at("2026-09-22T01:00:00Z"),
		},
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-100238",
			CaseNumber:      "PNC-100238",
			PolicyNumber:    "00.000.2026.00002",
			InsuredName:     "PT Contoh Karya Nusantara",
			ParticipantName: "Peserta Contoh Dua",
			BusinessName:    "Aneka",
			BranchName:      "Contoh Cabang Timur",
			AdminName:       "ADMINCONTOH2",
			RegisteredAt:    at("2026-09-16T03:40:00Z"),
			SurveyDate:      at("2026-09-18T07:30:00Z"),
		},
		{
			Reference:    "ASM-FW-GCNMFW-WORK PNC-100236",
			CaseNumber:   "PNC-100236",
			PolicyNumber: "00.000.2026.00003",
			InsuredName:  "PT Contoh Bahari Lestari",
			// Nama Peserta ada, tetapi surveinya belum dijadwalkan sama sekali.
			ParticipantName: "Peserta Contoh Tiga",
			BusinessName:    "Marine Cargo",
			BranchName:      "Contoh Cabang Barat",
			AdminName:       "ADMINCONTOH1",
			RegisteredAt:    at("2026-09-15T08:05:00Z"),
			// TANPA tanggal survei. Kolom kesembilan wajib tampil KOSONG, barisnya TETAP ada.
			SurveyDate: nil,
		},
		{
			Reference:    "ASM-FW-GCNMFW-WORK PNC-100230",
			CaseNumber:   "PNC-100230",
			PolicyNumber: "00.000.2026.00004",
			InsuredName:  "PT Contoh Graha Utama",
			// Klaim yang objek pertanggungannya belum terisi — T_CLAIM_OBJECTLIST kosong,
			// sehingga subquery mengembalikan NULL.
			ParticipantName: "",
			BusinessName:    "Fire",
			BranchName:      "Contoh Pusat",
			AdminName:       "ADMINCONTOH3",
			RegisteredAt:    at("2026-09-14T06:20:00Z"),
			SurveyDate:      at("2026-09-17T02:45:00Z"),
		},
		{
			// Nomor klaim format BARU (`D-71`), hidup berdampingan dengan format Pega.
			Reference:       "PNCN.26.0007",
			CaseNumber:      "PNCN.26.0007",
			PolicyNumber:    "00.000.2026.00005",
			InsuredName:     "PT Contoh Mitra Andalan",
			ParticipantName: "Peserta Contoh Lima",
			BusinessName:    "Travel",
			BranchName:      "Contoh Cabang Utara",
			AdminName:       "ADMINCONTOH2",
			RegisteredAt:    at("2026-09-19T01:10:00Z"),
			SurveyDate:      at("2026-09-21T03:00:00Z"),
		},
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-100222",
			CaseNumber:      "PNC-100222",
			PolicyNumber:    "00.000.2026.00006",
			InsuredName:     "PT Contoh Sentosa Jaya",
			ParticipantName: "Peserta Contoh Enam",
			BusinessName:    "Personal Accident",
			BranchName:      "Contoh Cabang Selatan",
			// Nama Admin kosong: kasus yang dibuat proses terjadwal, bukan orang. Job
			// harian memang membuat kasus tanpa pengguna sama sekali (`D-57`).
			AdminName:    "",
			RegisteredAt: at("2026-09-12T00:30:00Z"),
			SurveyDate:   at("2026-09-15T04:15:00Z"),
		},
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-099876",
			CaseNumber:      "PNC-099876",
			PolicyNumber:    "00.000.2026.00007",
			InsuredName:     "PT Contoh Persada Mandiri",
			ParticipantName: "Peserta Contoh Tujuh",
			BusinessName:    "Aneka",
			BranchName:      "Contoh Cabang Timur",
			AdminName:       "ADMINCONTOH3",
			RegisteredAt:    at("2026-07-30T07:00:00Z"),
			SurveyDate:      at("2026-08-03T02:00:00Z"),
		},
		{
			Reference:       "ASM-FW-GCNMFW-WORK PNC-099801",
			CaseNumber:      "PNC-099801",
			PolicyNumber:    "00.000.2026.00008",
			InsuredName:     "PT Contoh Dirgantara Prima",
			ParticipantName: "Peserta Contoh Delapan",
			BusinessName:    "Contractors PM",
			BranchName:      "Contoh Cabang Barat",
			AdminName:       "ADMINCONTOH1",
			// Tanggal Pendaftaran kosong. Tidak semestinya terjadi pada baris yang sah,
			// tetapi tidak ada DDL yang membuktikan kolomnya NOT NULL (`R-08`).
			RegisteredAt: nil,
			SurveyDate:   at("2026-09-08T06:00:00Z"),
		},
	}
}

// at membaca waktu contoh, dan panik bila teksnya salah.
//
// Panik di sini aman: teksnya konstanta di dalam berkas ini, bukan masukan pengguna,
// sehingga kesalahannya adalah cacat pemrograman yang harus terlihat saat pertama
// dijalankan.
//
// Seluruhnya UTC, sama seperti yang disimpan basis data (`DB-8`). Pengubahan ke WIB terjadi
// di layar, bukan di sini — dan tidak pernah dengan menambahkan tujuh jam secara manual
// (`F-5`).
func at(text string) *time.Time {
	moment, err := time.Parse(time.RFC3339, text)
	if err != nil {
		panic("inboxinvestigator/memory: waktu contoh tidak dapat dibaca: " + text)
	}
	return &moment
}
