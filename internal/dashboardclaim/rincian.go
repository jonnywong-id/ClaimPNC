package dashboardclaim

import (
	"context"
)

// ClaimDetail adalah rincian satu klaim — isi popup yang terbuka saat nomor klaim diklik.
//
// Menggantikan harness `ViewTempDetailClaim`, yang di Pega dibuka lewat `showHarness`
// bertarget `popup` dari kolom nomor klaim pada `DashboardClaimShow_Sec`.
type ClaimDetail struct {
	// Tujuh nilai pertama datang dari TABEL KERJA, bukan dari dokumen JSON.
	//
	// Dipisahkan dengan sengaja: ketujuhnya selalu ada selama klaimnya ada, sedangkan
	// dokumen JSON bisa belum terbentuk. Popup yang menggambar nomor klaim dari dokumen
	// akan tampil kosong seluruhnya pada klaim yang baru didaftarkan.
	ClaimID       string
	ClaimNumber   string
	ProcessStatus string
	ClaimStatus   string
	TechnicalPIC  string
	AdminPNC      string
	RegisteredAt  string

	// GroupPanel dan BusinessType menentukan VARIAN grid yang digambar layar — penjaga
	// IsPA, IsHE, IsFire, dan seterusnya.
	//
	// Dibawa di kepala, bukan disimpulkan layar dari nama bisnis: nama bisnis adalah teks
	// bebas, sedangkan kedua kode ini yang dibandingkan rule Pega.
	GroupPanel   string
	BusinessType string

	// InputterNote adalah "Catatan dari Inputor" — T_CLAIM_PNC.REMARK.
	InputterNote string

	// Registration adalah isian tab "Registrasi", dan SurveyRequest isian tab "Hasil Survey".
	//
	// Keduanya himpunan isian tunggal, bukan daftar — layar lama menggambarnya sebagai satu
	// blok nilai, bukan grid. Karena itu keduanya struct, bukan slice.
	Registration  RegistrationFields
	SurveyRequest SurveyRequestFields

	// PUCL memuat dua dari empat isian tab "Catatan Untuk Analis".
	//
	// Dua sisanya — "Pilihan Compliance" dan "Note PilihanCompliance" — TIDAK ADA kolomnya di
	// mana pun pada skema POOLDATA; pencarian menyeluruh atas `ALL_TAB_COLUMNS` tidak
	// menemukan satu pun padanan. Keduanya karena itu tidak diwakili di sini sama sekali,
	// bukan diwakili field kosong yang menyiratkan kolomnya ada tapi tidak terisi.
	PUCL PUCLFields

	// Ketujuh bagian di bawah adalah ISI popup, dibaca dari TABEL RELASIONAL.
	//
	// # Kenapa bukan dari dokumen JSON lagi
	//
	// Sampai 2026-10-12 seluruhnya berupa satu `map[string]any` hasil urai
	// `POOLDATA.JSON_KLAIM.DATA_JSON` — meniru Pega apa adanya.
	//
	// Kolom itu terisi **721 dari 16.284 baris**, dan hanya **348 dari 2.758** klaim dashboard
	// punya isinya. Popup karena itu tampil kosong bagi mayoritas klaim; di Pega pun demikian.
	//
	// Work Owner memutuskan 2026-10-12: migrasi tidak memakai JSON lagi, langsung ke kolom.
	// Itu juga arah Steering — `R-10` menetapkan data klaim disimpan sebagai tabel relasional,
	// dan JSON hanya untuk snapshot polis (`D-04`).
	//
	// # Kosong bukan galat
	//
	// Tiap bagian boleh kosong, dan kosongnya berarti klaim itu memang belum punya isinya —
	// bukan kegagalan membaca. Yang membedakan keduanya adalah galat yang dikembalikan, bukan
	// daftar yang pendek.
	Receivers   []Receiver
	Items       []InsuredItem
	Coverages   []Coverage
	Settlements []SettlementLine
	Surveys     []SurveyStep
	Attachments []Attachment
	Progress    []ProgressStep
}

// ClaimDetailReader membaca rincian satu klaim.
//
// Seam tersendiri, bukan method pada Repo: ia membaca TUJUH tabel yang tidak disentuh
// kueri daftar mana pun, dan memisahkannya membuat batas itu terlihat di perakitan.
//
// Sampai 2026-10-12 ia satu-satunya pembaca `POOLDATA.JSON_KLAIM`; sejak migrasi lepas
// dari JSON, tabel itu tidak disentuh modul ini sama sekali.
type ClaimDetailReader interface {
	FindClaimDetail(ctx context.Context, claimID string) (ClaimDetail, error)
}
