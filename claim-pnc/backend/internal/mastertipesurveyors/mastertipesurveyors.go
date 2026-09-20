// Package mastertipesurveyors adalah inti modul Master Tipe Surveyors (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Setiap klaim yang disurvei ditugaskan kepada seorang **Surveyor**, dan setiap surveyor
// digolongkan ke dalam satu **Tipe** — Internal Surveyor, Loss Adjuster, Expert, atau
// Survey Agent. Daftar TIPE itulah yang dikelola modul ini. Daftar orangnya berada satu
// tingkat di bawah dan bukan lingkup paket ini:
//
//	Tipe Surveyor   POOLDATA.M_SURVEYORS   <- yang dikerjakan paket ini
//	  └─ Surveyor   POOLDATA.D_SURVEYORS   <- anak, merujuk M_SURVEY_ID
//
// Anak-nya belum dibangun; ia butir menu tersendiri ("Master Surveyors", `MENU_ID 15`,
// harness `DetailSurveyorsInbox`).
//
// # Kenapa tipe surveyor bukan sekadar daftar pilihan
//
// Nilainya menyetir percabangan nyata di sistem lama, bukan hanya mengisi dropdown.
// Tiga kueri memilih surveyor dengan mematok kodenya langsung:
//
//	RDB List/BrowseSurveyorTypeLossAdjuster-SQL.xml   m_survey_id in ('1002')
//	RDB List/BrowseSurveyorTypeExpert-SQL.xml         m_survey_id in ('1003')
//	RDB List/BrowseSurveyorTypeSurveyAgent-SQL.xml    m_survey_id in ('1004')
//
// Itulah sebabnya KODE tidak pernah boleh berubah setelah terbit: mengubahnya memutus
// ketiga kueri di atas sekaligus setiap baris D_SURVEYORS yang menyimpannya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega dan dari katalog basis data yang berjalan,
// bukan dikarang:
//
//	Harness/SurveyorsInbox-Harness.xml          layar "Master Petugas Survei"
//	Section/GridSurveyors-Section.xml            kerangka layar; tombol Tambah & Refresh
//	Section/BrowseSuveryors-Section.xml          grid; M_SURVEY_ID pyEditOptions=Read-only
//	Report Definition/BrowseVMSurveyors_RD       tiga field, pyMaxRecords=500
//	Report Definition/SelectVMSurveyors_RD       ambil satu baris untuk disunting
//	Activity/SetSurveryorsValue_act-Act.xml      aksi Ubah: salin ke TempSurveyors
//	Activity/CNMInsertSurveyors_act-Act.xml      aksi Simpan; sentinel "UnknownID"
//	RDB List/UpdateMSurveyors-SQL.xml            memanggil POOLDATA.PEGA_M_SURVEYORS
//	Database/PEGA_M_SURVEYORS.prc                source procedure-nya
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastertipesurveyors

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxDescriptionLength membatasi panjang deskripsi tipe surveyor.
//
// DITETAPKAN WORK OWNER 2026-09-19. Sebelumnya ia asumsi; sekarang ia keputusan.
//
// Dua hal yang diketahui pasti, dan keduanya tidak memberi angka — itulah sebabnya
// angkanya harus diputuskan, bukan dibaca:
//
//   - Kolom POOLDATA.M_SURVEYORS.DESCRIPTION bertipe VARCHAR2(4000) — dibaca langsung
//     dari ALL_TAB_COLUMNS pada 2026-09-19. Ia tidak membatasi apa pun yang berguna:
//     nama tipe sepanjang 4000 karakter akan merusak setiap grid yang menampilkannya.
//   - Layar Pega tidak membatasi panjang sama sekali (`pyMaxLength` kosong).
//
// Keempat nilai yang benar-benar ada hari ini terpanjang 17 karakter ("INTERNAL
// SURVEYOR"). Seratus dipilih karena ia angka yang sama dengan master lain yang sudah
// dibangun (`masterstatus.MaxLabelLength`, `masterstatusprogres.MaxNameLength`),
// sehingga batas yang dilihat pengguna seragam antarlayar master.
//
// Pemeriksaan portal ASM pada 2026-09-19 memastikan batas ini tidak menjebak baris yang
// sudah ada: nol baris melebihi 100 karakter, sehingga setiap tipe yang ada sekarang
// tetap dapat disunting tanpa ditolak validasi. Portal lain belum dapat diperiksa —
// kredensialnya belum terisi — dan pemeriksaannya ikut dalam permintaan ke DBA bersama
// migrasi 0003.
//
// Angka yang sama diulang di frontend supaya pengguna tahu sebelum mengirim; server
// tetap yang berwenang. Bila angka ini berubah, KEDUA tempat wajib ikut berubah.
const MaxDescriptionLength = 100

// SurveyorType adalah satu baris master tipe surveyor.
//
// Ketiga field memetakan ke kolom yang dibaca Pega lewat POOLDATA.V_M_SURVEYORS, dan
// namanya mengikuti `D-19` — bukan nama kolomnya. Pemetaannya ada di repo/sqlstore,
// satu tempat saja.
type SurveyorType struct {
	// Code adalah M_SURVEY_ID. Dibuat sistem saat tipe ditambahkan dan TIDAK PERNAH
	// berubah sesudahnya — layar Pega pun menandainya `pyEditOptions=Read-only`.
	Code string

	// Description adalah DESCRIPTION, teks yang dibaca pengguna: "INTERNAL SURVEYOR",
	// "LOSS ADJUSTER", dan seterusnya.
	Description string

	// LegacyCode adalah OLD_M_SURVEY_ID, jejak penomoran sistem sebelumnya.
	//
	// KOSONG pada seluruh empat baris yang ada hari ini — diperiksa langsung ke basis
	// data. Ia tetap dibawa karena ia bagian dari kontrak view yang dibaca Pega, dan
	// karena master entitas lain belum tentu sama. Tidak pernah diisi untuk tipe baru:
	// ia jejak sejarah, bukan field yang dikelola.
	LegacyCode string
}

// Clean mengembalikan salinan dengan spasi tepi dibuang.
//
// Perapian ini nyata gunanya, bukan kerapian kosmetik: M_SURVEY_ID dan OLD_M_SURVEY_ID
// keduanya bertipe CHAR(4), dan Oracle memadatkan nilai CHAR dengan spasi tanpa memberi
// tanda apa pun. Membiarkannya membuat perbandingan kode dan tampilannya membawa spasi
// yang tidak dimaksudkan siapa pun.
func (t SurveyorType) Clean() SurveyorType {
	return SurveyorType{
		Code:        strings.TrimSpace(t.Code),
		Description: strings.TrimSpace(t.Description),
		LegacyCode:  strings.TrimSpace(t.LegacyCode),
	}
}

// DescriptionKey adalah bentuk deskripsi yang dipakai menguji keunikan.
//
// Perbandingan mengabaikan besar-kecil huruf dan spasi tepi. Alasannya bukan selera:
// `docs/Steering/11-SECURITY.md` §3.1 mencatat tiga nama access group Pega muncul dalam
// dua kapitalisasi berbeda karena perbandingan rule lama tidak konsisten soal itu.
// "Expert" dan "EXPERT" adalah tipe yang sama bagi pengguna, dan harus diperlakukan sama
// oleh sistem.
//
// Indeks unik di basis data memakai ekspresi yang SAMA (`UPPER(TRIM(DESCRIPTION))` pada
// migrasi 0003), sehingga pemeriksaan di sini dan penegakan di sana tidak dapat berbeda
// pendapat.
func DescriptionKey(description string) string {
	return strings.ToUpper(strings.TrimSpace(description))
}

// Repo adalah seam ke penyimpanan master tipe surveyor SATU portal.
//
// Satu instans Repo selalu terikat pada satu basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat penyaringan baris (`ADR-0030` Opsi 1). Tidak
// ada satu pun kueri di pengisinya yang menyaring berdasarkan entitas, dan memang tidak
// boleh ada.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Antarmukanya berbicara dalam istilah domain — List, Get, Insert,
// Update — bukan istilah SQL.
type Repo interface {
	// List mengembalikan seluruh tipe surveyor, terurut menurut kode.
	List(ctx context.Context) ([]SurveyorType, error)

	// Get mengembalikan satu tipe. ErrNotFound bila kodenya tidak ada.
	Get(ctx context.Context, code string) (SurveyorType, error)

	// Insert menyimpan tipe baru dan mengembalikannya LENGKAP DENGAN KODE yang dibuat
	// penyimpanan. Kode tidak pernah datang dari pemanggil.
	Insert(ctx context.Context, description string) (SurveyorType, error)

	// Update mengubah deskripsi tipe yang sudah ada. Kode dan kode lama tidak ikut
	// berubah. ErrNotFound bila kodenya tidak ada.
	Update(ctx context.Context, code, description string) (SurveyorType, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// CheckDescription mengumpulkan SELURUH pelanggaran aturan deskripsi, bukan berhenti pada
// yang pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama menampilkan
// seluruh pesan validasi sekaligus, dan mengembalikannya satu per satu akan membuat
// pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// Keunikan TIDAK diperiksa di sini: ia menuntut membaca penyimpanan, sedangkan fungsi ini
// murni dan dapat diuji tanpa apa pun. Pemeriksaannya ada di usecase.
func CheckDescription(description string) []Violation {
	clean := strings.TrimSpace(description)
	var violation []Violation

	if clean == "" {
		// Perbedaan yang DISENGAJA dari sistem lama, mengikuti keputusan Work Owner untuk
		// Master Status Klaim (2026-09-17) yang menetapkan isian kosong ditolak.
		//
		// Ia bukan kerapian: tipe surveyor berdeskripsi kosong akan tampil sebagai baris
		// kosong di setiap dropdown pemilihan surveyor, dan petugas tidak punya cara
		// menduga tipe mana yang sedang dipilihnya.
		violation = append(violation, Violation{
			Field:   FieldDescription,
			Message: "Tipe surveyor wajib diisi.",
		})
	}

	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if length := utf8.RuneCountInString(clean); length > MaxDescriptionLength {
		violation = append(violation, Violation{
			Field:   FieldDescription,
			Message: "Tipe surveyor paling panjang " + strconv.Itoa(MaxDescriptionLength) + " karakter.",
		})
	}

	return violation
}

// ErrNoSite dikembalikan bila kode tidak dapat dibentuk karena tabel situs tidak memuat
// baris aktif. Ia dideklarasikan di domain supaya transport dapat membedakannya dari
// kegagalan basis data biasa dan menjawabnya dengan pesan yang dapat ditindaklanjuti.
var ErrNoSite = errors.New("mastertipesurveyors: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
