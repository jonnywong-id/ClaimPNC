package memory

import (
	"time"

	"claim-pnc/internal/inboxprogressclaim"
)

// SampleOwner adalah login petugas pemilik rekap contoh.
//
// Ia dipakai adapter memori saja. Pada basis data sungguhan, pemiliknya adalah petugas yang
// benar-benar tercatat sebagai PIC klaimnya.
const SampleOwner = "ADMINKLAIM"

// day membentuk tanggal UTC tanpa jam, supaya contoh terbaca dan saringan tanggalnya mudah
// diperiksa dengan mata.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// SampleClaims adalah baris klaim contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap penyaring memperoleh baris yang membuatnya dapat
// dibuktikan bekerja.
//
//   - Dua baris SUDAH jatuh tempo dan dua belum, sehingga region Next Follow Up dapat
//     dibedakan dari Outstanding tanpa mengubah apa pun.
//   - Satu baris tindak lanjut terakhirnya BELUM PERNAH dicatat, sehingga baris yang bocor
//     ke Next Follow Up karena perbandingan terhadap nilai kosong langsung terlihat.
//   - Satu klaim berada di DUA posisi sekaligus, sehingga penggabungan antarposisi —
//     pengganti `GET_POSISI_PROGRESS_PNC` — terlihat hasilnya.
//   - Satu baris tidak punya posisi berjalan sama sekali.
//   - Nama PIC berbeda-beda, sehingga kotak cari yang lupa menelusuri kolom PIC akan
//     terlihat sebagai pencarian yang tidak menemukan apa-apa.
//   - `ProcessDate` berjarak dan satu baris dibiarkan kosong, sehingga urutannya —
//     termasuk penempatan baris tanpa tanggal di belakang — dapat diperiksa.
var SampleClaims = []ClaimRecord{
	{
		Item: inboxprogressclaim.ClaimRow{
			ClaimNumber:  "PNCN.26.0101",
			PolicyNumber: "16.001.2026.00101",
			InsuredName:  "PT Bina Usaha Contoh",
			RegisterDate: day(2026, time.August, 3),
			LossDate:     day(2026, time.July, 28),
			LGBNote:      "Kerugian kebakaran gudang",
			TechnicalPIC: SampleOwner,
			Positions: []inboxprogressclaim.Position{
				{
					Name:         "SURVEY",
					Status1:      "Survey Berjalan",
					Status2:      "Menunggu laporan surveyor",
					NextFollowUp: day(2026, time.September, 18),
				},
				{
					Name:         "KOMITE",
					Status1:      "Menunggu Komite",
					Status2:      "Berkas dilengkapi",
					NextFollowUp: day(2026, time.September, 19),
				},
			},
			EarliestFollowUp: day(2026, time.September, 18),
			ProcessDate:      day(2026, time.August, 4),
			ProdKe:           "1",
		},
		// Sudah lewat tenggat pada tanggal contoh — muncul di Next Follow Up.
		LastFollowUpByPIC: day(2026, time.September, 19),
	},
	{
		Item: inboxprogressclaim.ClaimRow{
			ClaimNumber:  "PNCN.26.0102",
			PolicyNumber: "16.001.2026.00102",
			InsuredName:  "Koperasi Maju Contoh",
			RegisterDate: day(2026, time.August, 11),
			LossDate:     day(2026, time.August, 9),
			LGBNote:      "Kehilangan kargo dalam pengangkutan",
			TechnicalPIC: "BUDITEKNIK",
			Positions: []inboxprogressclaim.Position{
				{
					Name:         "REGISTER",
					Status1:      "Registrasi Lengkap",
					Status2:      "Dokumen diterima",
					NextFollowUp: day(2026, time.September, 21),
				},
			},
			EarliestFollowUp: day(2026, time.September, 21),
			ProcessDate:      day(2026, time.August, 12),
			ProdKe:           "2",
		},
		// Tepat pada tanggal contoh — jatuh tempo HARI INI, jadi ikut muncul.
		LastFollowUpByPIC: day(2026, time.September, 21),
	},
	{
		Item: inboxprogressclaim.ClaimRow{
			ClaimNumber:  "PNCN.26.0103",
			PolicyNumber: "16.001.2026.00103",
			InsuredName:  "CV Sumber Contoh",
			RegisterDate: day(2026, time.September, 1),
			LossDate:     day(2026, time.August, 27),
			LGBNote:      "Kerusakan alat berat",
			TechnicalPIC: "SITITEKNIK",
			Positions: []inboxprogressclaim.Position{
				{
					Name:         "AKSEPTASI",
					Status1:      "Menunggu Akseptasi",
					Status2:      "",
					NextFollowUp: day(2026, time.October, 2),
				},
			},
			EarliestFollowUp: day(2026, time.October, 2),
			ProcessDate:      day(2026, time.September, 2),
			ProdKe:           "1",
		},
		// Belum jatuh tempo — tidak muncul di Next Follow Up.
		LastFollowUpByPIC: day(2026, time.October, 2),
	},
	{
		Item: inboxprogressclaim.ClaimRow{
			ClaimNumber:      "PNCN.26.0104",
			PolicyNumber:     "16.001.2026.00104",
			InsuredName:      "PT Anugerah Contoh",
			RegisterDate:     day(2026, time.September, 9),
			LossDate:         day(2026, time.September, 5),
			LGBNote:          "",
			TechnicalPIC:     SampleOwner,
			Positions:        []inboxprogressclaim.Position{},
			EarliestFollowUp: nil,
			// Sengaja tanpa tanggal proses: barisnya harus jatuh di BELAKANG saat diurutkan.
			ProcessDate: nil,
			ProdKe:      "3",
		},
		// Belum pernah ada catatan tindak lanjut — tidak boleh muncul di Next Follow Up.
		LastFollowUpByPIC: nil,
	},
}

// SamplePICs adalah rekap contoh.
//
// Ketiga barisnya memakai login yang sama tetapi lini bisnis berbeda, sehingga penyaring
// lini bisnis yang lupa dipasang akan langsung terlihat sebagai tiga baris yang muncul
// bersamaan. Satu baris memakai login lain, sehingga penyaring kepemilikan pun terbukti.
var SamplePICs = []PICRecord{
	{
		Item: inboxprogressclaim.PICSummary{
			PIC:           SampleOwner,
			ClaimCount:    12,
			UpdateCount:   37,
			DueTodayCount: 2,
			OnTimeCount:   30,
			LateCount:     5,
		},
		Business:     inboxprogressclaim.BusinessNonMBU,
		RegisteredAt: day(2026, time.August, 3),
	},
	{
		Item: inboxprogressclaim.PICSummary{
			PIC:           SampleOwner,
			ClaimCount:    4,
			UpdateCount:   9,
			DueTodayCount: 1,
			OnTimeCount:   8,
			LateCount:     1,
		},
		Business:     inboxprogressclaim.BusinessTravel,
		RegisteredAt: day(2026, time.September, 1),
	},
	{
		Item: inboxprogressclaim.PICSummary{
			PIC:           SampleOwner,
			ClaimCount:    3,
			UpdateCount:   6,
			DueTodayCount: 0,
			OnTimeCount:   6,
			LateCount:     0,
		},
		Business:     inboxprogressclaim.BusinessPA,
		RegisteredAt: day(2026, time.July, 20),
	},
	{
		Item: inboxprogressclaim.PICSummary{
			PIC:           "BUDITEKNIK",
			ClaimCount:    7,
			UpdateCount:   15,
			DueTodayCount: 3,
			OnTimeCount:   11,
			LateCount:     4,
		},
		Business:     inboxprogressclaim.BusinessNonMBU,
		RegisteredAt: day(2026, time.August, 3),
	},
}

// NewSampleStore membentuk pembaca berisi baris contoh.
//
// Salinan, bukan senarai aslinya: dua portal yang memakai adapter memori tidak boleh
// berbagi baris yang sama, karena perubahan pada satu akan terlihat di yang lain — persis
// kebocoran antarentitas yang `R-20` peringatkan.
func NewSampleStore() *Store {
	claims := make([]ClaimRecord, len(SampleClaims))
	copy(claims, SampleClaims)

	pics := make([]PICRecord, len(SamplePICs))
	copy(pics, SamplePICs)

	return NewStore(claims, pics)
}
