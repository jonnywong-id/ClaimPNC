package memory

import (
	"time"

	"claim-pnc/internal/inboxservicecenter"
)

// SampleOwner adalah login PIC pemilik baris contoh.
//
// Keempat tab menyaring `PIC` menurut pengguna yang login, sehingga tanpa nilai ini tidak
// ada satu baris pun yang tampil saat pengembangan lokal. Pada basis data sungguhan, PIC-nya
// adalah petugas yang benar-benar ditunjuk menangani barisnya.
const SampleOwner = "PICSERVICECENTER"

// sampleOther adalah PIC lain, dipakai membuktikan penyaring kepemilikan bekerja.
const sampleOther = "PICLAIN"

// day membentuk tanggal UTC tanpa jam, supaya contohnya terbaca dan urutannya mudah
// diperiksa dengan mata.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// SampleClaims adalah klaim portal rekanan contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa seluruh isinya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama nasabah, nomor klaim, dan IMEI di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya. IMEI contoh sengaja TIDAK 15 digit yang sah, supaya ia tidak dapat
// tertukar dengan nomor perangkat sungguhan.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap penyaring memperoleh baris yang membuktikannya
// bekerja.
//
//   - Keempat tab terisi, termasuk DUA baris pada tab Rejected — satu berkode `3` (REJECT)
//     dan satu berkode `2` (TLO) — supaya penyaing `IN ('2','3')` terbukti membawa keduanya.
//   - Satu baris ber-PIC ORANG LAIN, sehingga penyaring kepemilikan yang lupa dipasang akan
//     langsung terlihat sebagai baris yang bocor.
//   - Satu baris TANPA tanggal input, supaya urutan `INPUTDATE DESC` yang salah menaruh
//     NULL langsung terlihat.
//   - Satu baris yang hanya dapat ditemukan lewat NOPOLIS/QQNAME/IMEI dan satu yang ID-nya
//     khas, supaya cacat pencarian irisan (lihat Limitations) terbukti direplikasi — bukan
//     tidak sengaja diperbaiki.
func SampleClaims() []inboxservicecenter.ServiceClaim {
	return []inboxservicecenter.ServiceClaim{
		// ------------------------------------------------ tab Registrasi SC (NULL)
		{
			ID:             "SC-000101",
			RepairID:       "1000101",
			ClaimNumber:    "PNCN.26.0101",
			PolicyNumber:   "90-001-2026-00000101",
			CustomerName:   "Contoh Nasabah Satu",
			Type:           "Gadget",
			TechnicalPIC:   SampleOwner,
			InputDate:      day(2026, time.September, 22),
			IMEI:           "IMEI-CONTOH-001",
			ApprovalStatus: "", // belum pernah diajukan ke komite
			RepairStatus:   "1",
			Owner:          SampleOwner,
		},
		{
			ID:             "SC-000102",
			RepairID:       "1000102",
			ClaimNumber:    "PNCN.26.0102",
			PolicyNumber:   "90-001-2026-00000102",
			CustomerName:   "Contoh Nasabah Dua",
			Type:           "Gadget",
			TechnicalPIC:   SampleOwner,
			InputDate:      nil, // sengaja kosong — menguji urutan NULL pada DESC
			IMEI:           "IMEI-CONTOH-002",
			ApprovalStatus: "",
			RepairStatus:   "2",
			Owner:          SampleOwner,
		},
		{
			// Baris milik PIC LAIN. Ia tidak boleh pernah tampil bagi SampleOwner.
			ID:             "SC-000103",
			RepairID:       "1000103",
			ClaimNumber:    "PNCN.26.0103",
			PolicyNumber:   "90-001-2026-00000103",
			CustomerName:   "Contoh Nasabah Tiga",
			Type:           "Gadget",
			TechnicalPIC:   sampleOther,
			InputDate:      day(2026, time.September, 23),
			IMEI:           "IMEI-CONTOH-003",
			ApprovalStatus: "",
			RepairStatus:   "1",
			Owner:          sampleOther,
		},

		// ------------------------------------------------ tab Waiting Approval ("0")
		{
			ID:                "SC-000201",
			RepairID:          "1000201",
			ClaimNumber:       "PNCN.26.0201",
			PolicyNumber:      "90-001-2026-00000201",
			CustomerName:      "Contoh Nasabah Empat",
			Type:              "Gadget",
			TechnicalPIC:      SampleOwner,
			InputDate:         day(2026, time.September, 18),
			IMEI:              "IMEI-CONTOH-004",
			ApprovalStatus:    inboxservicecenter.ApprovalPending,
			RepairStatus:      "6",
			Owner:             SampleOwner,
			CommitteeApprover: "KOMITECONTOH",
		},

		// ------------------------------------------------ tab Approved ("1")
		{
			ID:                "SC-000301",
			RepairID:          "1000301",
			ClaimNumber:       "PNCN.26.0301",
			PolicyNumber:      "90-001-2026-00000301",
			CustomerName:      "Contoh Nasabah Lima",
			Type:              "Gadget",
			TechnicalPIC:      SampleOwner,
			InputDate:         day(2026, time.September, 12),
			IMEI:              "IMEI-CONTOH-005",
			ApprovalStatus:    inboxservicecenter.ApprovalApproved,
			RepairStatus:      "7",
			Owner:             SampleOwner,
			CommitteeApprover: "KOMITECONTOH",
		},

		// ------------------------------------------------ tab Rejected ("2" dan "3")
		{
			// TLO — kode "2". Ia masuk tab Rejected bersama kode "3".
			ID:                "SC-000401",
			RepairID:          "1000401",
			ClaimNumber:       "PNCN.26.0401",
			PolicyNumber:      "90-001-2026-00000401",
			CustomerName:      "Contoh Nasabah Enam",
			Type:              "Gadget",
			TechnicalPIC:      SampleOwner,
			InputDate:         day(2026, time.September, 9),
			IMEI:              "IMEI-CONTOH-006",
			ApprovalStatus:    inboxservicecenter.ApprovalTotalLoss,
			RepairStatus:      "3",
			Owner:             SampleOwner,
			CommitteeApprover: "KOMITECONTOH",
		},
		{
			// REJECT — kode "3".
			ID:                "SC-000402",
			RepairID:          "1000402",
			ClaimNumber:       "PNCN.26.0402",
			PolicyNumber:      "90-001-2026-00000402",
			CustomerName:      "Contoh Nasabah Tujuh",
			Type:              "Gadget",
			TechnicalPIC:      SampleOwner,
			InputDate:         day(2026, time.September, 5),
			IMEI:              "IMEI-CONTOH-007",
			ApprovalStatus:    inboxservicecenter.ApprovalRejected,
			RepairStatus:      "3",
			Owner:             SampleOwner,
			CommitteeApprover: "KOMITECONTOH",
		},
	}
}
