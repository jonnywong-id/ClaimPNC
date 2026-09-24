package memory

import (
	"time"

	"claim-pnc/internal/mastersurveyors"
)

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
func sampleRows() []mastersurveyors.Surveyor {
	decided := time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)
	created := time.Date(2026, 9, 10, 2, 0, 0, 0, time.UTC)

	return []mastersurveyors.Surveyor{
		{
			// Surveyor internal: satu-satunya golongan yang WAJIB punya login aplikasi.
			ID:              "1000001",
			TypeCode:        mastersurveyors.InternalTypeCode,
			TypeDescription: "INTERNAL SURVEYOR",
			Name:            "Surveyor Contoh Satu",
			Address:         "Jalan Contoh Nomor 1",
			PostalCode:      "12345",
			State:           "DKI Jakarta",
			Phone:           "0210000001",
			Email:           "surveyor.satu@contoh.invalid",
			BranchCode:      "001",
			BranchName:      "Kantor Pusat",
			AppLogin:        "SURVEYORSATU",
			Status:          mastersurveyors.StatusApproved,
			Committee:       "KOMITECONTOH",
			DecidedAt:       &decided,
			Note:            "Disetujui sebagai contoh data pengembangan.",
			CreatedBy:       "SISTEM",
			CreatedAt:       created,
		},
		{
			// Loss adjuster eksternal: tanpa login aplikasi, dan itu SAH.
			ID:              "1000002",
			TypeCode:        "1002",
			TypeDescription: "LOSS ADJUSTER",
			Name:            "Adjuster Contoh Dua",
			Address:         "Jalan Contoh Nomor 2",
			PostalCode:      "40123",
			State:           "Jawa Barat",
			Phone:           "0220000002",
			Fax:             "0220000012",
			Email:           "adjuster.dua@contoh.invalid",
			OtherContact:    "Narahubung contoh",
			BranchCode:      "002",
			BranchName:      "Cabang Bandung",
			Status:          mastersurveyors.StatusPending,
			Committee:       "KOMITECONTOH",
			CreatedBy:       "SISTEM",
			CreatedAt:       created,
		},
		{
			ID:              "1000003",
			TypeCode:        "1003",
			TypeDescription: "EXPERT",
			Name:            "Expert Contoh Tiga",
			Address:         "Jalan Contoh Nomor 3",
			State:           "Jawa Timur",
			Phone:           "0310000003",
			Email:           "expert.tiga@contoh.invalid",
			BranchCode:      "003",
			BranchName:      "Cabang Surabaya",
			Status:          mastersurveyors.StatusPending,
			Committee:       "KOMITELAIN",
			CreatedBy:       "SISTEM",
			CreatedAt:       created,
		},
		{
			ID:              "1000004",
			TypeCode:        "1004",
			TypeDescription: "SURVEY AGENT",
			Name:            "Agen Contoh Empat",
			Address:         "Jalan Contoh Nomor 4",
			State:           "Bali",
			Phone:           "0361000004",
			Email:           "agen.empat@contoh.invalid",
			BranchCode:      "004",
			BranchName:      "Cabang Denpasar",
			Status:          mastersurveyors.StatusRejected,
			Committee:       "KOMITECONTOH",
			DecidedAt:       &decided,
			Note:            "Ditolak sebagai contoh data pengembangan.",
			CreatedBy:       "SISTEM",
			CreatedAt:       created,
		},
	}
}
