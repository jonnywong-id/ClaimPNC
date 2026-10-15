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

// SampleExportRows mengembalikan isi berkas Export Data Investigation untuk pengembangan.
//
// # Seluruh isinya KARANGAN
//
// Berkas ini memuat ALAMAT RUMAH SAKIT dan NOMOR REKAM MEDIS — data medis yang `FR-R2`
// batasi aksesnya. Tidak ada satu pun nilai nyata di sini, dan larangan `D-69` berlaku
// lebih keras pada kelompok ini daripada pada kolom mana pun di layar.
//
// # Yang sengaja diwakili
//
//	baris 1   sudah diinvestigasi, rumah sakit, seluruh kolom terisi
//	baris 2   sudah diinvestigasi, NON rumah sakit, nomor rekam medis kosong
//	baris 3   sudah diinvestigasi, di luar rentang contoh paling umum
//	baris 4   BELUM diinvestigasi  -> hanya muncul bila dropdown dipilih "0"
//	baris 5   sudah diinvestigasi, SelectRS kosong -> terbaca "NON Rumah Sakit"
//
// Baris 4 dan 5 yang paling perlu ada. Yang pertama membuktikan dropdown "Pilih
// Investigation" benar-benar menyaring; yang kedua membuktikan nilai kosong diterjemahkan
// sama dengan "0", persis `@if` sistem lama (lihat ExportRow.HospitalKindLabel).
func SampleExportRows() []inboxinvestigator.ExportRow {
	return []inboxinvestigator.ExportRow{
		{
			InvestigatedAt:      at("2026-09-21T02:15:00Z"),
			HospitalAddress:     "Jalan Contoh Nomor 1, Kota Contoh",
			PaidByOtherInsurer:  "false",
			PaidByPatient:       "true",
			PaidByCompany:       "false",
			NoPayment:           "false",
			Investigated:        inboxinvestigator.InvestigatedYes,
			ReceiptConfirmation: "1",
			MedicalRecordNumber: "RM-CONTOH-0001",
			PhoneCalled:         "021-0000000",
			PatientRegistered:   "1",
			Remarks:             "Keterangan contoh baris pertama.",
			HospitalKindCode:    "1",
		},
		{
			InvestigatedAt:      at("2026-09-18T03:00:00Z"),
			HospitalAddress:     "Jalan Contoh Nomor 2, Kota Contoh",
			PaidByOtherInsurer:  "true",
			PaidByPatient:       "false",
			PaidByCompany:       "false",
			NoPayment:           "false",
			Investigated:        inboxinvestigator.InvestigatedYes,
			ReceiptConfirmation: "0",
			MedicalRecordNumber: "",
			PhoneCalled:         "021-0000001",
			PatientRegistered:   "0",
			Remarks:             "Keterangan contoh baris kedua.",
			HospitalKindCode:    "0",
		},
		{
			InvestigatedAt:      at("2026-08-02T04:30:00Z"),
			HospitalAddress:     "Jalan Contoh Nomor 3, Kota Contoh",
			PaidByOtherInsurer:  "false",
			PaidByPatient:       "false",
			PaidByCompany:       "true",
			NoPayment:           "false",
			Investigated:        inboxinvestigator.InvestigatedYes,
			ReceiptConfirmation: "1",
			MedicalRecordNumber: "RM-CONTOH-0003",
			PhoneCalled:         "021-0000002",
			PatientRegistered:   "1",
			Remarks:             "Keterangan contoh baris ketiga.",
			HospitalKindCode:    "1",
		},
		{
			InvestigatedAt:      at("2026-09-19T01:00:00Z"),
			HospitalAddress:     "Jalan Contoh Nomor 4, Kota Contoh",
			PaidByOtherInsurer:  "false",
			PaidByPatient:       "false",
			PaidByCompany:       "false",
			NoPayment:           "true",
			Investigated:        inboxinvestigator.InvestigatedNo,
			ReceiptConfirmation: "0",
			MedicalRecordNumber: "",
			PhoneCalled:         "",
			PatientRegistered:   "0",
			Remarks:             "Belum diinvestigasi.",
			HospitalKindCode:    "0",
		},
		{
			InvestigatedAt:      at("2026-09-15T07:45:00Z"),
			HospitalAddress:     "Jalan Contoh Nomor 5, Kota Contoh",
			PaidByOtherInsurer:  "false",
			PaidByPatient:       "true",
			PaidByCompany:       "false",
			NoPayment:           "false",
			Investigated:        inboxinvestigator.InvestigatedYes,
			ReceiptConfirmation: "1",
			MedicalRecordNumber: "RM-CONTOH-0005",
			PhoneCalled:         "021-0000004",
			PatientRegistered:   "1",
			Remarks:             "SelectRS kosong pada baris ini.",
			HospitalKindCode:    "",
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
