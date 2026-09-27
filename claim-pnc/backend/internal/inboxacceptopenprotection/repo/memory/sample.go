package memory

import (
	"time"

	"claim-pnc/internal/inboxacceptopenprotection"
)

// sampleProtections adalah data contoh untuk pengembangan lokal.
//
// Seluruhnya KARANGAN (`D-69`).
//
// # Yang sengaja diwakili
//
//	OPCN.26.0002  NON PREMI, lengkap          -> tampil di antrean NON PREMI
//	OPCN.26.0003  PREMI (tipe '2'), lengkap   -> tampil di antrean PREMI saja
//	OPCN.26.0004  NON PREMI, lengkap          -> menguji paginasi dan urutan
//	OPCN.26.0006  BELUM tertaut klaim         -> TIDAK tampil di antrean mana pun
//	OPCN.26.0007  sudah disetujui             -> TIDAK tampil; menguji penyaring status
//
// Dua baris terakhir ada justru karena ia TIDAK BOLEH tampil. Data contoh yang seluruhnya
// memenuhi syarat tidak membuktikan penyaringnya bekerja.
func sampleProtections() []inboxacceptopenprotection.Protection {
	wib := time.FixedZone("WIB", 7*60*60)

	at := func(day, hour int) time.Time {
		return time.Date(2026, time.September, day, hour, 0, 0, 0, wib)
	}

	policyStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, wib)
	policyEnd := time.Date(2026, time.December, 31, 0, 0, 0, 0, wib)
	approvedAt := at(20, 15)

	return []inboxacceptopenprotection.Protection{
		{
			Number:       "OPCN.26.0002",
			PolicyNumber: "99.001.2026.00000003",
			ClaimNumber:  "PNCN.26.0007",
			Type:         "1",
			InputDate:    at(17, 8),
			Note:         "Lengkap dan menunggu akseptasi.",
			CreatedBy:    "ADMINCONTOH",
			AcceptStatus: inboxacceptopenprotection.AcceptPending,
			InsuredName:  "Tertanggung Contoh",
			PolicyStart:  &policyStart,
			PolicyEnd:    &policyEnd,
		},
		{
			Number:       "OPCN.26.0003",
			PolicyNumber: "99.001.2026.00000004",
			ClaimNumber:  "PNCN.26.0008",
			Type:         inboxacceptopenprotection.TypePremium,
			InputDate:    at(18, 11),
			Note:         "Proteksi klaim PREMI; hanya tampil di antrean penagihan premi.",
			CreatedBy:    "ADMINCONTOH",
			AcceptStatus: inboxacceptopenprotection.AcceptPending,
			InsuredName:  "Tertanggung Contoh",
			PolicyStart:  &policyStart,
			PolicyEnd:    &policyEnd,
		},
		{
			Number:       "OPCN.26.0004",
			PolicyNumber: "99.001.2026.00000005",
			ClaimNumber:  "PNCN.26.0009",
			Type:         "7",
			InputDate:    at(19, 14),
			Note:         "Permintaan perubahan Tanggal Kejadian.",
			CreatedBy:    "TEKNIKCONTOH",
			AcceptStatus: inboxacceptopenprotection.AcceptPending,
			InsuredName:  "Tertanggung Contoh",
			PolicyStart:  &policyStart,
			PolicyEnd:    &policyEnd,
		},
		{
			// Belum tertaut klaim: masih milik pemohon, belum layak diakseptasi.
			Number:       "OPCN.26.0006",
			PolicyNumber: "99.001.2026.00000007",
			ClaimNumber:  "",
			Type:         "1",
			InputDate:    at(21, 9),
			Note:         "Masih rancangan; belum tertaut klaim.",
			CreatedBy:    "ADMINCONTOH",
			AcceptStatus: inboxacceptopenprotection.AcceptPending,
		},
		{
			// Sudah diputuskan: hilang dari antrean, tetapi tetap dapat dibuka lewat nomor.
			Number:       "OPCN.26.0007",
			PolicyNumber: "99.001.2026.00000008",
			ClaimNumber:  "PNCN.26.0012",
			Type:         "1",
			InputDate:    at(20, 10),
			Note:         "Sudah disetujui.",
			CreatedBy:    "ADMINCONTOH",
			AcceptStatus: inboxacceptopenprotection.AcceptApproved,
			AcceptedAt:   &approvedAt,
			AcceptedBy:   "KOLEKSICONTOH",
			InsuredName:  "Tertanggung Contoh",
			PolicyStart:  &policyStart,
			PolicyEnd:    &policyEnd,
		},
	}
}
