package inboxinvestigator

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Investigation adalah hasil investigasi satu pekerjaan — isi formulir yang dibuka saat
// Nomor Case ditekan.
//
// # Asalnya
//
// `Flow Action/InputInvestigator-FA.xml` menunjuk section
// `InputClaimInvestigasiDetail` (`:49`). Ketiga puluh medan di bawah adalah SELURUH isian
// yang dapat diubah pengguna di sana, dibaca PER SEL dari section itu — bukan dari
// kedekatan baris, metode yang sudah terbukti menghasilkan pasangan mustahil.
//
// # Yang TIDAK ada di sini, dan masing-masing punya sebabnya
//
//	Nama Tertanggung · Nama Peserta · Tanggal Masuk · Total Pengajuan
//	    BACA-SAJA. Keempatnya milik klaim dan sudah punya rumah; formulir hanya
//	    menampilkannya supaya penyelidik tahu klaim mana yang sedang dikerjakannya.
//
//	Akan Dikirim ke Analyst (`.ClaimData.UserTeknis`)
//	    SELALU TERSEMBUNYI di layar lama — kedua selnya ber-`pyCondition = 1=2`. Tidak
//	    dibangun, bukan karena lingkup melainkan karena ia memang tidak pernah tampil.
//
//	Pertanyaan dari Analyst · Tanggal Keluar · penanda rawat inap
//	    Milik KLAIM, bukan milik baris investigasi. Kolomnya belum dipastikan ada setelah
//	    `T_CLAIM_PNC` dipangkas dari 97 menjadi 82 kolom; lihat
//	    `docs/ddl/tc_pnc_investigasi.sql` §3.
//
// # Kenapa seluruhnya string, termasuk yang bernilai "1"/"0" dan "true"/"false"
//
// Karena begitulah nilainya tersimpan hari ini, terukur dari `POOLDATA.JSON_KLAIM`.
// Mengubahnya menjadi bool atau int di sini berarti menerjemahkan dua kali — sekali masuk,
// sekali keluar — dan berkas Export Data Investigation menuliskannya APA ADANYA (`P-5`).
// Satu terjemahan yang memang ada hanyalah Jenis Rumah Sakit; lihat ExportRow.
type Investigation struct {
	// ClaimRef adalah `pzInsKey` klaim — penghubung ke Task.Reference.
	ClaimRef string

	// SurveyIndex dan Index adalah indeks page list `SurveyResults(n).SurveyList(m)`.
	// Keduanya ikut supaya satu klaim dapat punya lebih dari satu baris investigasi,
	// persis seperti di Pega. Hari ini keduanya selalu 1.
	SurveyIndex int
	Index       int

	// InvestigatedAt — "Tanggal Investigasi".
	//
	// Diisi OTOMATIS saat formulir dibuka, bukan oleh pengguna:
	// `Activity/PresetInvestigation-Act.xml` satu langkah,
	// `TanggalInvestigasi := @CurrentDateTime()`. Lihat Service.OpenInvestigation.
	InvestigatedAt *time.Time

	// Investigated — "Dapat Diinvestigasi", `"1"` atau `"0"`.
	//
	// Ia juga penyaring berkas Export Data Investigation; lihat ExportFilter.Investigated.
	Investigated string

	// HospitalKindCode — "Tempat Kejadian" / "RS/Klinik yang di Survei", `"1"` atau `"0"`.
	//
	// Ia MENGENDALIKAN tiga isian lain; lihat Visible.
	HospitalKindCode string

	// HospitalName — "Nama Rumah Sakit". Tampil hanya bila HospitalKindCode `"1"`.
	HospitalName string

	// OtherPlaceName — "Nama Tempat Lainnya". Tampil hanya bila HospitalKindCode `"0"`.
	OtherPlaceName string

	// HospitalAddress — "Alamat RS/Klinik/Lokasi Kejadian".
	HospitalAddress string

	// MedicalRecordNumber — "Nomor Rekam Medik". **DATA MEDIS** (`FR-R2`).
	MedicalRecordNumber string

	// PatientName — "Nama Peserta" pada panel rincian. **DATA NASABAH**.
	PatientName string

	// DateOfBirth — "Tanggal Lahir". **DATA NASABAH**.
	DateOfBirth *time.Time

	// BirthDateVerified — "Verifikasi Tanggal Lahir".
	BirthDateVerified string

	// BirthDateNote — "Keterangan Tambahan".
	BirthDateNote string

	// PatientRegistered — "Peserta Terdaftar Di RS", `"1"` atau `"0"`.
	PatientRegistered string

	// RegistrationNote — "Keterangan Tambahan" pada pendaftaran peserta.
	RegistrationNote string

	// TreatmentStart — "Tanggal Perawatan/Kejadian". **DATA MEDIS**.
	TreatmentStart *time.Time

	// TreatmentEnd — "Tanggal Selesai Perawatan". Tampil hanya bila klaimnya rawat inap.
	TreatmentEnd *time.Time

	// BillTotal — "Total Pengajuan" pada panel rincian.
	BillTotal string

	// BillSettled — status pelunasan tagihan.
	BillSettled string

	// Keempat medan berikut adalah SATU kelompok pilihan "Yang Melakukan Pembayaran",
	// bukan empat pertanyaan yang berdiri sendiri. Nilainya `"true"`/`"false"`.
	PaidByPatient      string
	PaidByCompany      string
	PaidByOtherInsurer string
	NoPayment          string

	// OtherInsurer — "Nama Asuransi/Perusahaan Lain". Tampil hanya bila PaidByCompany atau
	// PaidByOtherInsurer bernilai `"true"`.
	OtherInsurer string

	// ReceiptConfirmation — "Konfirmasi Model Kwitansi", `"1"` atau `"0"`.
	ReceiptConfirmation string

	// Empat medan kontak.
	HospitalPIC string // "Nama PIC RS" / "Nama PIC Yang Dapat Dihubungi"
	CallerName  string // "Nama Penelepon"
	StaffName   string // "Nama PIC Yang Dapat Dihubungi" pada panel rincian
	PhoneArea   string // "Kode"
	Phone       string // "Nomor Telepon Yang Dapat Dihubungi"
	PhoneExt    string // "No. Ext."

	// Remarks — "Hasil Investigasi". Ejaan medan di basis data `REMAKS`, mengikuti property
	// Pega; yang diperbaiki hanya nama di kode (`D-80`).
	Remarks string
}

// Visible menyatakan isian mana yang tampil pada keadaan sekarang.
//
// # Kenapa ini hidup di domain, bukan di layar
//
// Keenam syaratnya dibaca dari `pyCondition` pembungkus tiap sel di section lama — ia
// ATURAN, bukan selera tata letak. Aturan yang hidup di satu tempat adalah inti alasan
// migrasi ini dikerjakan (`04-FUTURE-ARCHITECTURE.md` §2).
//
// Menaruhnya di komponen React berarti ia tidak dapat diuji tanpa merender layar, dan
// berarti pula lapisan simpan tidak punya cara mengetahui isian mana yang seharusnya
// kosong.
type Visible struct {
	// HospitalName tampil bila tempat kejadiannya rumah sakit.
	HospitalName bool

	// OtherPlaceName tampil bila BUKAN rumah sakit.
	OtherPlaceName bool

	// HospitalAddress tampil bila tempat kejadiannya rumah sakit.
	//
	// Perhatikan syaratnya di layar lama: `SelectRS=='1'` — dengan tanda kutip — sementara
	// "Nama Rumah Sakit" memakai `SelectRS==1` tanpa kutip. Keduanya nilai yang sama, dan
	// perbedaan penulisannya tidak dibawa.
	HospitalAddress bool

	// TreatmentEnd tampil bila klaimnya rawat inap (`ClaimData.SelectRawatInap`).
	TreatmentEnd bool

	// OtherInsurer tampil bila pembayarnya perusahaan atau asuransi lain.
	OtherInsurer bool
}

// VisibleFor menghitung keadaan tampil dari isian yang sedang dipegang.
//
// inpatient datang dari klaim (`ClaimData.SelectRawatInap`), bukan dari formulir — itulah
// sebabnya ia parameter, bukan medan Investigation.
func VisibleFor(one Investigation, inpatient bool) Visible {
	hospital := one.HospitalKindCode == "1"
	return Visible{
		HospitalName:    hospital,
		OtherPlaceName:  !hospital,
		HospitalAddress: hospital,
		TreatmentEnd:    inpatient,
		OtherInsurer: one.PaidByCompany == "true" ||
			one.PaidByOtherInsurer == "true",
	}
}

// ErrInvestigationInvalid menandai formulir yang tidak dapat disimpan.
//
// Galat KLIEN: yang salah adalah isiannya, bukan kemampuan menyimpannya.
var ErrInvestigationInvalid = errors.New("isian investigasi tidak sah")

// ErrInvestigationStoreMissing menandai tabel penyimpanannya belum ada.
//
// Ia DIBEDAKAN dari galat basis data biasa, dan itu bukan kerapian: selama
// `POOLDATA.TC_PNC_INVESTIGASI` belum dibuat DBA, setiap simpan akan gagal — dan petugas
// berhak membaca sebabnya sebagai "belum disiapkan administrator", bukan sebagai
// "sistem rusak" yang membuatnya mencoba berulang kali.
var ErrInvestigationStoreMissing = errors.New("tabel hasil investigasi belum tersedia")

// Validate memeriksa formulir sebelum disimpan.
//
// # Yang diperiksa, dan yang SENGAJA tidak
//
// Layar lama tidak memasang satu pun `pyRequired` pada isian investigasi — seluruh 59 selnya
// dapat dikosongkan. Yang diperiksa di sini karena itu hanya DUA hal, dan keduanya bukan
// selera melainkan akibat langsung dari aturan lain:
//
//  1. **"Dapat Diinvestigasi" wajib dipilih.** Nilainya menentukan isi berkas Export Data
//     Investigation (`ExportFilter.Investigated`), dan baris yang nilainya kosong tidak akan
//     pernah muncul di berkas mana pun — hilang tanpa satu pun tanda.
//
//  2. **Isian yang tidak tampil tidak boleh terisi.** Nama Rumah Sakit yang terisi pada
//     klaim bertempat kejadian "bukan rumah sakit" adalah nilai yang tidak pernah dapat
//     dilihat maupun dikoreksi pengguna.
//
// Butir kedua MEMBERSIHKAN, bukan menolak: lihat Clean.
func (i Investigation) Validate() error {
	switch strings.TrimSpace(i.Investigated) {
	case InvestigatedYes, InvestigatedNo:
	default:
		return fmt.Errorf("%w: pilihan \"Dapat Diinvestigasi\" wajib diisi",
			ErrInvestigationInvalid)
	}
	if strings.TrimSpace(i.ClaimRef) == "" {
		return fmt.Errorf("%w: pekerjaan yang diinvestigasi tidak disebut",
			ErrInvestigationInvalid)
	}
	return nil
}

// Clean memangkas spasi dan MENGOSONGKAN isian yang tidak tampil.
//
// Pengosongan itu disengaja. Pengguna yang mengisi "Nama Rumah Sakit" lalu mengubah tempat
// kejadian menjadi "bukan rumah sakit" meninggalkan nilai yang tidak lagi terlihat di
// layarnya; menyimpannya berarti menyimpan sesuatu yang tidak dapat dikoreksi siapa pun.
func (i Investigation) Clean(inpatient bool) Investigation {
	trimmed := i
	trimmed.ClaimRef = strings.TrimSpace(i.ClaimRef)
	trimmed.Investigated = strings.TrimSpace(i.Investigated)
	trimmed.HospitalKindCode = strings.TrimSpace(i.HospitalKindCode)
	trimmed.HospitalName = strings.TrimSpace(i.HospitalName)
	trimmed.OtherPlaceName = strings.TrimSpace(i.OtherPlaceName)
	trimmed.HospitalAddress = strings.TrimSpace(i.HospitalAddress)
	trimmed.MedicalRecordNumber = strings.TrimSpace(i.MedicalRecordNumber)
	trimmed.PatientName = strings.TrimSpace(i.PatientName)
	trimmed.BirthDateVerified = strings.TrimSpace(i.BirthDateVerified)
	trimmed.BirthDateNote = strings.TrimSpace(i.BirthDateNote)
	trimmed.PatientRegistered = strings.TrimSpace(i.PatientRegistered)
	trimmed.RegistrationNote = strings.TrimSpace(i.RegistrationNote)
	trimmed.BillTotal = strings.TrimSpace(i.BillTotal)
	trimmed.BillSettled = strings.TrimSpace(i.BillSettled)
	trimmed.PaidByPatient = strings.TrimSpace(i.PaidByPatient)
	trimmed.PaidByCompany = strings.TrimSpace(i.PaidByCompany)
	trimmed.PaidByOtherInsurer = strings.TrimSpace(i.PaidByOtherInsurer)
	trimmed.NoPayment = strings.TrimSpace(i.NoPayment)
	trimmed.OtherInsurer = strings.TrimSpace(i.OtherInsurer)
	trimmed.ReceiptConfirmation = strings.TrimSpace(i.ReceiptConfirmation)
	trimmed.HospitalPIC = strings.TrimSpace(i.HospitalPIC)
	trimmed.CallerName = strings.TrimSpace(i.CallerName)
	trimmed.StaffName = strings.TrimSpace(i.StaffName)
	trimmed.PhoneArea = strings.TrimSpace(i.PhoneArea)
	trimmed.Phone = strings.TrimSpace(i.Phone)
	trimmed.PhoneExt = strings.TrimSpace(i.PhoneExt)
	trimmed.Remarks = strings.TrimSpace(i.Remarks)

	if trimmed.SurveyIndex < 1 {
		trimmed.SurveyIndex = 1
	}
	if trimmed.Index < 1 {
		trimmed.Index = 1
	}

	shown := VisibleFor(trimmed, inpatient)
	if !shown.HospitalName {
		trimmed.HospitalName = ""
	}
	if !shown.OtherPlaceName {
		trimmed.OtherPlaceName = ""
	}
	if !shown.HospitalAddress {
		trimmed.HospitalAddress = ""
	}
	if !shown.TreatmentEnd {
		trimmed.TreatmentEnd = nil
	}
	if !shown.OtherInsurer {
		trimmed.OtherInsurer = ""
	}
	return trimmed
}

// Perpindahan status yang dilakukan saat formulir disimpan.
//
// Seluruhnya dibaca dari `Activity/SetStatusInvestigator_Act-Act.xml`, dan seluruhnya
// bermuara ke `POOLDATA.T_CLAIM_PNC` — tabel yang SUDAH ditulis aplikasi ini. Tidak ada
// kepemilikan baru yang perlu dinegosiasikan untuk bagian ini (`P-1`).
const (
	// SurveyStatusInvestigated adalah `SurveyResults(1).SurveyStatus := 5`.
	SurveyStatusInvestigated = "5"

	// PNCStatusInvestigated adalah `PNCStatus := 5`.
	PNCStatusInvestigated = "5"

	// ClaimStatusAnalyst adalah `ClaimData.StatusClaim := "1151"`.
	//
	// Kode `1151` berarti **Analyst** pada master `v_sts_claim` (33 kode `1134`–`1166`).
	// Ia DIBACA dari master, bukan ditebak dari pemakaiannya — dokumen-dokumen awal sempat
	// menyimpulkan `1151` berarti Investigator, dan itu terbukti salah (`R-06`).
	ClaimStatusAnalyst = "1151"
)

// Transition adalah akibat menyimpan formulir terhadap klaimnya.
//
// Ia dikembalikan, bukan dikerjakan diam-diam, supaya lapisan transport dapat menyebutkan
// kepada pengguna apa yang baru saja terjadi — pekerjaan itu HILANG dari antrean setelah
// disimpan, dan tanpa keterangan itu ia terbaca seperti data yang lenyap.
type Transition struct {
	// ClaimRef adalah pekerjaan yang berpindah.
	ClaimRef string

	// SurveyStatus, PNCStatus, dan ClaimStatus adalah nilai yang ditulis.
	SurveyStatus string
	PNCStatus    string
	ClaimStatus  string

	// At adalah waktu perpindahan — satu nilai untuk `InvestTfDate` maupun
	// `AnalystTransferDate`, supaya keduanya tidak dapat berbeda sedetik.
	At time.Time
}

// InvestigationRepo adalah seam ke penyimpanan hasil investigasi.
//
// Ia TERPISAH dari Repo, dan itu disengaja: Repo hanya membaca, sedangkan yang ini
// MENULIS. Memisahkannya membuat jalur tulis tidak dapat dipakai tanpa disadari oleh kode
// yang hanya bermaksud menampilkan antrean.
type InvestigationRepo interface {
	// Load mengembalikan hasil investigasi yang sudah tersimpan untuk satu pekerjaan.
	//
	// Pekerjaan yang belum pernah diinvestigasi mengembalikan Investigation kosong beserta
	// found=false — BUKAN galat. Formulir yang belum pernah diisi adalah keadaan yang
	// diharapkan, dan menjawabnya dengan galat akan membuat setiap pembukaan pertama
	// tampil merah.
	Load(ctx context.Context, claimRef string) (one Investigation, found bool, err error)

	// Save menyimpan formulir DAN memindahkan klaimnya, dalam satu transaksi.
	//
	// Keduanya satu transaksi karena keduanya satu peristiwa: klaim yang berpindah ke
	// Analyst tanpa hasil investigasinya tersimpan adalah klaim yang tidak dapat
	// ditindaklanjuti siapa pun.
	Save(ctx context.Context, one Investigation, move Transition, by string) error
}

// InvestigationRepoSelector memilih InvestigationRepo milik satu portal entitas.
//
// Alasannya sama dengan RepoSelector: portal yang tidak dikenal WAJIB menghasilkan galat,
// bukan jatuh ke portal utama. Pada jalur TULIS akibatnya lebih berat lagi — hasil
// investigasi satu badan hukum akan tersimpan di basis data badan hukum lain (`R-20`).
type InvestigationRepoSelector func(portalAlias string) (InvestigationRepo, error)
