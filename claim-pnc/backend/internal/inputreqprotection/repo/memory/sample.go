package memory

import (
	"time"

	"claim-pnc/internal/inputreqprotection"
)

// sampleProtections adalah data contoh untuk pengembangan lokal.
//
// # Seluruhnya KARANGAN
//
// `D-69` melarang nomor polis, nama tertanggung, dan nomor klaim sungguhan ditulis di
// berkas yang di-commit. Nomor polis di bawah memakai pola yang jelas-jelas tidak nyata,
// dan tidak satu pun nama orang muncul.
//
// # Yang sengaja diwakili
//
// Keenam baris memilih keadaan yang masing-masing menguji hal berbeda di layar, bukan
// sekadar mengisi tabel:
//
//	OPC-201       nomor WARISAN Pega, bentuknya berbeda dari terbitan aplikasi ini
//	OPCN.26.0001  terbitan aplikasi ini, belum tertaut klaim -> MASIH dapat disunting
//	OPCN.26.0002  sudah tertaut klaim -> TERKUNCI, tautannya mati di layar
//	OPCN.26.0003  tipe '2' PREMI -> muncul di antrean PncCollection, bukan di antrean lain
//	OPCN.26.0004  tipe '7' perubahan DOL -> membawa detail perubahan tanggal
//	OPCN.26.0005  tipe '8' perubahan Cause of Loss -> membawa detail penyebab kerugian
//
// Tidak ada satu pun baris yang sudah diakseptasi. Itu disengaja: baris semacam itu TIDAK
// AKAN TAMPIL di layar ini (penyaringnya `AcceptStatus IS NULL`), sehingga
// menambahkannya hanya akan membuat data contoh tampak lebih banyak daripada yang terlihat.
// Baris terakseptasi dibuat di uji yang memang memeriksa penyaring itu.
func sampleProtections() []inputreqprotection.Protection {
	wib := time.FixedZone("WIB", 7*60*60)

	at := func(day, hour int) time.Time {
		return time.Date(2026, time.September, day, hour, 0, 0, 0, wib)
	}

	dolBefore := time.Date(2026, time.August, 14, 0, 0, 0, 0, wib)
	dolAfter := time.Date(2026, time.August, 17, 0, 0, 0, 0, wib)

	return []inputreqprotection.Protection{
		{
			Number:         "OPC-201",
			PolicyNumber:   "99.001.2026.00000001",
			ClaimNumber:    "",
			Type:           "1",
			InputDate:      at(14, 9),
			Note:           "Permintaan buka proteksi dari cabang contoh.",
			AcceptStatus:   inputreqprotection.AcceptPending,
			CreatedBy:      "ADMINCONTOH",
			CreatedAt:      at(14, 9),
			ClaimReference: "",
		},
		{
			Number:         "OPCN.26.0001",
			PolicyNumber:   "99.001.2026.00000002",
			ClaimNumber:    "",
			Type:           "1",
			InputDate:      at(16, 10),
			Note:           "Belum ditautkan ke klaim, masih dapat disunting pemohon.",
			AcceptStatus:   inputreqprotection.AcceptPending,
			CreatedBy:      "ADMINCONTOH",
			CreatedAt:      at(16, 10),
			ClaimReference: "",
		},
		{
			Number:         "OPCN.26.0002",
			PolicyNumber:   "99.001.2026.00000003",
			ClaimNumber:    "PNCN.26.0007",
			Type:           "1",
			InputDate:      at(17, 8),
			Note:           "Sudah tertaut klaim, tautan barisnya mati di layar.",
			AcceptStatus:   inputreqprotection.AcceptPending,
			CreatedBy:      "ADMINCONTOH",
			CreatedAt:      at(17, 8),
			ClaimReference: "KLAIM-CONTOH-0007",
		},
		{
			Number:         "OPCN.26.0003",
			PolicyNumber:   "99.001.2026.00000004",
			ClaimNumber:    "PNCN.26.0008",
			Type:           inputreqprotection.TypePremium,
			InputDate:      at(18, 11),
			Note:           "Proteksi klaim PREMI; diakseptasi peran penagihan premi.",
			AcceptStatus:   inputreqprotection.AcceptPending,
			CreatedBy:      "ADMINCONTOH",
			CreatedAt:      at(18, 11),
			ClaimReference: "KLAIM-CONTOH-0008",
		},
		{
			Number:         "OPCN.26.0004",
			PolicyNumber:   "99.001.2026.00000005",
			ClaimNumber:    "PNCN.26.0009",
			Type:           inputreqprotection.TypeChangeLossDate,
			InputDate:      at(19, 14),
			Note:           "Permintaan perubahan Tanggal Kejadian.",
			AcceptStatus:   inputreqprotection.AcceptPending,
			CreatedBy:      "TEKNIKCONTOH",
			CreatedAt:      at(19, 14),
			ClaimReference: "KLAIM-CONTOH-0009",
			ChangeDetail: inputreqprotection.ChangeDetail{
				LossDateBefore: &dolBefore,
				LossDateAfter:  &dolAfter,
				ObjectName:     "Objek contoh",
				BranchName:     "Cabang contoh",
			},
		},
		{
			Number:         "OPCN.26.0005",
			PolicyNumber:   "99.001.2026.00000006",
			ClaimNumber:    "PNCN.26.0010",
			Type:           inputreqprotection.TypeChangeCauseOfLoss,
			InputDate:      at(22, 9),
			Note:           "Permintaan perubahan Penyebab Kerugian.",
			AcceptStatus:   inputreqprotection.AcceptPending,
			CreatedBy:      "TEKNIKCONTOH",
			CreatedAt:      at(22, 9),
			ClaimReference: "KLAIM-CONTOH-0010",
			ChangeDetail: inputreqprotection.ChangeDetail{
				CauseOfLossID:       "12002",
				CauseOfLossMasterID: "COL-CONTOH-1",
				ObjectName:          "Objek contoh",
				BranchName:          "Cabang contoh",
			},
		},
	}
}
