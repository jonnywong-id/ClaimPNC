// Package daftartipedokumenbisnis adalah inti modul Daftar Tipe Dokumen Bisnis
// (`F-4`, MENU_ID 42, MENU_PROGRAM `DetTypeDocumenBisnis`).
//
// # Apa yang dimodelkan di sini
//
// Satu baris menjawab satu pertanyaan: **dokumen apa yang harus diunggah, untuk lini
// bisnis mana, pada tahap klaim mana.** Ia bukan daftar dokumen melainkan ATURAN
// KELENGKAPAN, dan pembacanya bukan layar ini melainkan enam layar unggah dokumen di
// sepanjang perjalanan klaim.
//
// Tahapnya ditentukan DOCUMENT_TYPE_ID yang menunjuk `V_LST_DOC_TYPE`. Keenam nilainya
// terbaca langsung dari keenam kueri pembacanya, bukan ditebak:
//
//	REGISTER              RDB List/BrowseRegister_upload-SQL.xml
//	SURVEY                RDB List/BrowseSurvey_upload-SQL.xml
//	COMMITEE              RDB List/BrowseCommitee_upload-SQL.xml   (salah ketik di data)
//	PAYMENT               RDB List/BrowsePayment_upload-SQL.xml
//	SALVAGE               RDB List/BrowseSalvage_upload-SQL.xml
//	COLLECTING DOCUMENT   RDB List/BrowseCollectingDoc_upload-SQL.xml
//
// # Batas modul: ini master TURUNAN, bukan salah satu masternya
//
// Empat master lain dirujuk baris ini, dan TIDAK SATU PUN dimiliki paket ini:
//
//	V_LST_DOC_TYPE       <- internal/daftartipedokumen        MENU_ID 40  tahap klaim
//	V_LST_DET_TYPE_DOC   <- internal/daftardetailtipedokumen  MENU_ID 41  rincian dokumen
//	V_LST_DOC_OBJ        <- internal/daftarobjekdokumen       MENU_ID 43  objek dokumen
//	BUSINESS             <- GISFW (`D-03`)                                lini bisnis
//
// Keempatnya hanya DIBACA, lewat seam yang sengaja tidak punya satu pun operasi tulis
// (`P-1`). Batas itu ditegakkan bentuk antarmuka, bukan ingatan orang yang menulis kode
// berikutnya.
//
// # Wajib di sini TIDAK berarti wajib di klaim
//
// Ini bagian yang paling mudah salah dibaca, dan akibatnya uang. `Mandatory` bukan
// jawaban akhir — ia disaring lagi oleh daftar jaminan baris ini saat klaim berjalan.
// `RDB List/BrowseRegisterCvg-SQL.xml` membuktikannya:
//
//	WHEN sts_wajib = '1' AND (select count(a.coverageid) from coverage_doc_business a
//	     where a.id=b.id and a.businessid=b.businessid
//	       and coverageid in (...)) > 0   THEN 'Ya'
//	WHEN sts_wajib = '1' AND (...) = 0    THEN 'Tidak'
//
// Artinya baris ber-`STS_WAJIB=1` yang daftar jaminannya KOSONG tidak pernah menjadi
// wajib pada klaim mana pun. Daftar jaminan karena itu bukan pelengkap layar melainkan
// bagian dari aturannya, dan itulah sebab ia ikut dibangun di modul ini.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/DetTypeDocumenBisnis-Harness.xml                       layar
//	Section/DetTypeDocumenBisnis_Portal-Section.xml                judul, tombol Refresh
//	Section/BrowseListDetailTypeDocumentBusiness_sect-Section.xml  grid dan tombolnya
//	Section/InputListDetailTypeDocumentBusiness_sect-Section.xml   form dan labelnya
//	RDB List/BrowseLSTDetailDocument_sql-SQL.xml                   grid tingkat pertama
//	RDB List/Select_TYPE_DOCUMENT-SQL.xml                          grid tingkat kedua
//	RDB List/UpdateDetTypeDocBusiness_SQL-SQL.xml                  pemetaan parameter simpan
//	Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc                    jalur tulis yang sebenarnya
//	Activity/InsertDetailTypeDocumentBusiness_act-Act.xml          jalur tambah
//	Activity/UpdateDetailTypeDocumentBusiness_act-Act.xml          jalur muat untuk ubah dan salin
//	Activity/SetAllBusiness-Act.xml                                tombol pilih semua bisnis
//
// Berbeda dari modul Daftar Detail Dokumen Travel, jalur TULIS layar ini ADA di export.
// Nama tabel, nama kolom, dan bentuk ID karena itu DIBACA, bukan diturunkan dari nama
// view — satu-satunya hal yang masih harus dipastikan DBA adalah lebar kolomnya.
//
// # EMPAT TEMUAN atas layar lama yang HARUS dibaca sebelum menilai kesetaraan
//
// Ketiganya yang pertama adalah cacat pada sistem lama, bukan pada modul ini, dan
// seluruhnya ditemukan dengan menelusuri precondition setiap langkah — bukan dengan
// membaca komentarnya.
//
//  1. JALUR "TAMBAH" TIDAK MENULIS APA PUN sebagaimana ia diekspor. Langkah 3
//     `InsertDetailTypeDocumentBusiness_act` menuntut `pyLabel=="tambah"`, sementara
//     langkah anaknya yang memanggil basis data menuntut `pyLabel="update"||"copy"` —
//     kedua syarat itu tidak dapat benar bersamaan. Keduanya ditambahkan pada hari yang
//     sama, 2022-06-03. Yang dibangun modul ini adalah perilaku yang DIMAKSUDKAN,
//     sebagaimana tertulis pada komentar langkahnya sendiri ("kalo tambah > bisa banyak
//     bisnis & banyak dokumen"), bukan perilaku yang benar-benar berjalan.
//
//     Akibatnya untuk gerbang 1: uji kesetaraan pada jalur Tambah akan menunjukkan
//     SELISIH — sistem lama tidak menyimpan, modul ini menyimpan. Itu selisih yang
//     diketahui di muka, bukan kejutan, dan penyelesaiannya menunggu Work Owner: apakah
//     layar lama memang sudah rusak sejak 2022, atau ada jalur lain yang tidak ikut
//     terekspor.
//
//  2. JALUR "COPY" MEMBUKA LAYAR KOSONG. `UpdateDetailTypeDocumentBusiness_act` memuat
//     barisnya hanya bila `Tipe=="update"`, dan melewati perulangannya bila
//     `Tipe=="COPY"` — huruf besar seluruhnya — sementara tombolnya mengirim `"copy"`.
//     Salah kapitalisasi yang jelas tidak disengaja.
//
//  3. TIDAK ADA PEMERIKSAAN GANDA pada tabel utamanya. Procedure hanya memeriksa
//     keberadaan pada baris jaminan (`:53`, `:64`), tidak pada baris aturan. Menyimpan
//     dokumen yang sama dua kali untuk satu bisnis menghasilkan dua baris. Modul ini
//     MENIRUNYA (`P-5`) — menambahkan pemeriksaan berarti menolak penyimpanan yang di
//     sistem lama berhasil.
//
//  4. LIMA KODE BISNIS DI-HARDCODE pada tombol "Tamban semua bisnis NONMBU"
//     (`Activity/SetAllBusiness-Act.xml:984`): kelimanya DIKECUALIKAN dari pemilihan
//     massal karena merupakan lini MBU. Salah satunya, kode lini Personal Accident, juga
//     menjadi sakelar yang mengganti tombol Simpan menjadi "Simpan Doc PA"
//     (`Section/InputListDetailTypeDocumentBusiness_sect-Section.xml:8849`).
//
//     Kelimanya TIDAK dibawa sebagai konstanta (`D-15`). Modul ini tidak memasang
//     penyaring massal sama sekali — pemilihan bisnis seluruhnya di tangan petugas — dan
//     jalur "Simpan Doc PA" tidak dibangun; lihat di bawah.
//
// # Yang sengaja TIDAK dibangun, dan kenapa
//
// "Simpan Doc PA" (`Activity/InsertDocumentPendukungPA_-Act.xml`) adalah satu-satunya
// jalur yang mengisi COVERAGE_DOC_BUSINESS.NOKLAIM. Ia tidak dibangun, dan alasannya
// terbaca dari export itu sendiri: properti yang mengisinya, `DFT_KLAIMTYPE_ID`, tidak
// muncul di satu pun rule lain di seluruh export — tidak ada yang mengisinya, sehingga
// cabang itu tidak pernah berjalan sebagaimana diekspor. Kolom NOKLAIM pun tidak dibaca
// siapa pun.
//
// Seluruh kueri jaminan modul ini karena itu menyaring `NOKLAIM IS NULL`, dan baris
// ber-NOKLAIM — bila ada di produksi — dibiarkan apa adanya, tidak dibaca dan tidak
// disentuh.
//
// Satu artefak lagi yang HILANG dari export (`R-16`): aksi lokal `DetailCoverageDoc`, yakni
// popup tempat petugas menambahkan jaminan. Bentuk penyimpanannya tetap diketahui — ia
// memanggil procedure yang sama, dan cabangnya terbaca — tetapi bentuk LAYARNYA tidak.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package daftartipedokumenbisnis

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Business adalah satu lini bisnis, dibaca dari POOLDATA.BUSINESS.
//
// Ia mengisi grid tingkat pertama layar — `BrowseLSTDetailDocument_sql` mengembalikan
// bisnis yang SUDAH punya baris aturan, bukan seluruh bisnis yang ada.
type Business struct {
	// ID adalah BUSINESS.ID, tersimpan sebagai LST_TYPE_DOC_BUSINESS.BUSINESSID.
	ID string

	// Name adalah BUSINESS.NOTE, nama yang dibaca petugas.
	//
	// Perhatikan `BrowseLSTDetailDocument_sql` mengalias-namakannya `DOCUMENT_TYPE_ID` —
	// alias yang menyesatkan dan tidak dibawa (`D-19`). Isinya nama bisnis, bukan kode
	// tipe dokumen.
	Name string

	// ExcludedFromBulkSelect menandai bisnis yang TIDAK ikut terpilih oleh tombol "Pilih
	// semua".
	//
	// Kelimanya lini MBU, dan di Pega kodenya ditulis langsung di dalam rule
	// (`Activity/SetAllBusiness-Act.xml:984`) sebagai syarat yang mengeluarkan baris dari
	// perulangan. Di sini daftarnya datang dari konfigurasi (`D-15`), sehingga perilakunya
	// sama tetapi nilainya bukan lagi konstanta di dalam kode.
	//
	// Bisnis bertanda ini TETAP dapat dipilih satu per satu — di Pega pun demikian, karena
	// pengecualiannya hanya berlaku pada tombol pemilihan massal, bukan pada daftarnya.
	ExcludedFromBulkSelect bool
}

// DocumentRule adalah satu aturan kelengkapan dokumen — satu baris
// POOLDATA.LST_TYPE_DOC_BUSINESS.
type DocumentRule struct {
	// ID adalah kunci baris. Diterbitkan penyimpanan dan tidak pernah diisi pengguna.
	//
	// Bentuknya `kode_situs || lpad(urutan, 4, '0')`
	// (`Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22`) — pola yang sama dengan seluruh
	// master per entitas lain, dan sebab modul ini portal-aware.
	ID string

	// BusinessID adalah BUSINESSID, rujukan ke POOLDATA.BUSINESS.
	//
	// Ia TIDAK PERNAH BERUBAH setelah baris dibuat. Itu bukan pilihan modul ini
	// melainkan perilaku sistem lama: UPDATE di `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:39-41`
	// menyentuh delapan kolom dan BUSINESSID bukan salah satunya. Sebuah aturan dokumen
	// karena itu tidak dapat dipindahkan ke lini bisnis lain — ia disalin, bukan
	// dipindah, dan itulah guna tombol Copy di layar lama.
	BusinessID string

	// BusinessName adalah BUSINESS.NOTE hasil join, bukan kolom baris ini.
	//
	// Hanya terisi pada pembacaan; tidak pernah ditulis.
	BusinessName string

	// DocumentTypeID adalah DOCUMENT_TYPE_ID, rujukan ke V_LST_DOC_TYPE — TAHAP klaim
	// tempat dokumen ini diminta.
	DocumentTypeID string

	// DocumentTypeName adalah V_LST_DOC_TYPE.TYPE_DOCUMENT hasil join.
	//
	// Nilainya salah satu dari enam tahap yang disebut di kepala paket. Ia yang dipakai
	// keenam kueri unggah untuk memilih dokumen mana yang muncul di layarnya.
	DocumentTypeName string

	// ObjectDocID adalah OBJECT_DOC_ID, rujukan ke V_LST_DOC_OBJ.
	//
	// Boleh kosong. `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:944`
	// mengosongkannya dengan sengaja ketika keterangan objeknya kosong:
	//
	//	@If(.OBJ_DOC_DESC=="","",.OLD_ID)
	//
	// Pada tahap COMMITEE kolom ini ikut menentukan kewajiban dokumen
	// (`BrowseCommitee_upload-SQL.xml`), sehingga mengosongkannya di sana BUKAN sekadar
	// merapikan data.
	ObjectDocID string

	// ObjectDocName adalah V_LST_DOC_OBJ.KET_DOC_OBJ hasil join.
	ObjectDocName string

	// DetailTypeDocID adalah DOC_TYPE_DT_ID, rujukan ke V_LST_DET_TYPE_DOC.
	DetailTypeDocID string

	// DetailDocument adalah DETAIL_DOKUMEN, nama dokumen yang dibaca petugas di layar
	// unggah.
	//
	// # Nilai "-" MENYEMBUNYIKAN baris ini
	//
	// Keenam kueri unggah menyaring `AND (b.DETAIL_DOKUMEN != '-')`. Sebuah baris yang
	// isinya tepat satu tanda hubung karena itu tidak pernah muncul di layar klaim mana
	// pun, meski barisnya tetap ada dan tetap tampil di layar master ini.
	//
	// Ia ditiru apa adanya (`P-5`) dan TIDAK diubah menjadi sakelar "aktif/non-aktif":
	// mengubahnya berarti menebak bahwa setiap "-" yang ada hari ini memang dimaksudkan
	// sebagai penyembunyian, padahal ia sama mungkinnya sekadar isian kosong yang diisi
	// tanda hubung.
	DetailDocument string

	// Mandatory adalah STS_WAJIB.
	//
	// Tersimpan sebagai TEKS '1' atau '0', bukan angka — keenam kueri pembacanya
	// membandingkannya dengan literal berkutip (`sts_wajib = '1'`).
	//
	// BACA kepala paket sebelum memakai nilai ini: wajib di sini belum tentu wajib di
	// klaim.
	Mandatory bool

	// MinDocument adalah MIN_DOC, jumlah berkas paling sedikit yang harus diunggah.
	MinDocument int

	// Coverages adalah daftar jaminan yang membuat baris ini benar-benar wajib.
	//
	// Hanya terisi pada pembacaan SATU baris, tidak pada daftar — grid layar lama pun
	// tidak menampilkannya, dan menariknya untuk seluruh baris berarti satu kueri yang
	// hasilnya tidak pernah dilihat siapa pun.
	Coverages []Coverage
}

// Coverage adalah satu jaminan pada sebuah aturan dokumen — satu baris
// POOLDATA.COVERAGE_DOC_BUSINESS.
//
// # Kenapa hanya kodenya
//
// Tabelnya memang hanya menyimpan itu. Keempat kolomnya — ID, BUSINESSID, COVERAGEID,
// NOKLAIM — tidak memuat nama jaminan, dan tidak ada view yang menambahkannya. Nama yang
// dilihat petugas karena itu harus dicarikan dari master jaminan saat ditampilkan, bukan
// dibaca dari baris ini.
type Coverage struct {
	// ID adalah COVERAGEID.
	ID string
}

// Input adalah nilai satu baris aturan yang dikirim pengguna.
//
// Terpisah dari DocumentRule karena tiga hal di sana tidak pernah berasal dari pengguna:
// ID diterbitkan penyimpanan, kedua nama hasil join, dan daftar jaminan ditambahkan lewat
// jalurnya sendiri.
type Input struct {
	DocumentTypeID  string
	ObjectDocID     string
	DetailTypeDocID string
	DetailDocument  string
	Mandatory       bool
	MinDocument     int
}

// Clean memangkas spasi di kedua ujung setiap isian teks.
//
// # Ini bukan validasi
//
// Satu-satunya aturan isian yang benar-benar ada di layar lama adalah kewajiban memilih
// bisnis — `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:208` menyiapkan pesan
// "Nama Bisnis belum di isi" dan menghentikan penyimpanan. Aturan itu hidup di
// BatchInput.Validate, bukan di sini, karena yang dijaganya adalah pilihan bisnis dan
// bukan isi barisnya.
//
// Selebihnya tidak ada: tidak satu pun isian di
// `Section/InputListDetailTypeDocumentBusiness_sect-Section.xml` bertanda `pyRequired`
// bernilai true, dan tidak ada Validate rule yang dipasang padanya. Nama dokumen kosong
// diterima, kode yang tidak ada di master diterima. Perlakuan yang sama sudah dipakai
// Master Dokumen Travel dan Daftar Tipe Dokumen.
//
// Pemangkasan spasi tetap dilakukan, dan alasannya bukan kerapian: kolom CHAR berlebar
// tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, sehingga tanpa
// memangkas saat menulis, apa yang disimpan dan apa yang dibaca kembali dapat berbeda —
// dan selisih itu tidak terlihat di layar karena spasi tidak tampak.
func (i Input) Clean() Input {
	clean := Input{
		DocumentTypeID:  strings.TrimSpace(i.DocumentTypeID),
		ObjectDocID:     strings.TrimSpace(i.ObjectDocID),
		DetailTypeDocID: strings.TrimSpace(i.DetailTypeDocID),
		DetailDocument:  strings.TrimSpace(i.DetailDocument),
		Mandatory:       i.Mandatory,
		MinDocument:     i.MinDocument,
	}

	// MIN_DOC negatif tidak punya arti apa pun — "paling sedikit minus satu berkas" bukan
	// aturan yang dapat dipenuhi maupun dilanggar. Ia diratakan menjadi nol, bukan
	// ditolak, supaya perlakuannya tetap sejalan dengan "tanpa validasi". Sama dengan
	// MINUNGGAH pada Daftar Detail Dokumen Travel.
	if clean.MinDocument < 0 {
		clean.MinDocument = 0
	}
	return clean
}

// BatchInput adalah satu penyimpanan dari layar Tambah.
//
// # Kenapa bentuknya jamak kali jamak
//
// Karena begitulah layar lamanya bekerja, dan itu tertulis apa adanya sebagai komentar
// langkah di `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml`:
//
//	"kalo tambah > bisa banyak bisnis & banyak dokumen"
//	"kalo edit / copy > 1 bisnis"
//
// Petugas memilih beberapa lini bisnis sekaligus — atau seluruhnya, lewat tombol "Tamban
// semua bisnis NONMBU" (`Activity/SetAllBusiness-Act.xml`) — lalu menyusun beberapa baris
// dokumen, dan yang tersimpan adalah PERKALIAN keduanya. Aturan kelengkapan dokumen
// memang berulang nyaris sama di banyak lini bisnis, sehingga memaksa petugas
// memasukkannya satu per satu akan mengubah pekerjaan sepuluh menit menjadi sepuluh jam.
//
// Menyederhanakannya menjadi satu bisnis per penyimpanan bukan penyederhanaan melainkan
// penghapusan fitur.
type BatchInput struct {
	// BusinessIDs adalah lini bisnis yang dituju, satu atau lebih.
	BusinessIDs []string

	// Rules adalah baris aturan yang akan dibuat pada SETIAP bisnis di atas.
	Rules []Input
}

// Clean memangkas dan membuang pilihan bisnis yang kosong serta baris aturan yang
// seluruh isiannya kosong.
//
// Bisnis yang sama disebut dua kali dipadatkan menjadi satu. Tanpa itu, petugas yang
// tidak sengaja memilih satu bisnis dua kali akan memperoleh dua baris kembar yang tidak
// dapat dibedakan di grid mana pun.
func (b BatchInput) Clean() BatchInput {
	clean := BatchInput{}

	seen := make(map[string]struct{}, len(b.BusinessIDs))
	for _, id := range b.BusinessIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, already := seen[id]; already {
			continue
		}
		seen[id] = struct{}{}
		clean.BusinessIDs = append(clean.BusinessIDs, id)
	}

	for _, rule := range b.Rules {
		rule = rule.Clean()
		// Baris yang SELURUH isiannya kosong dibuang, dan itu bukan validasi melainkan
		// pembacaan maksud: grid di form selalu menyisakan baris kosong yang baru
		// ditambahkan tetapi belum diisi. Menyimpannya berarti menulis aturan dokumen
		// tanpa dokumen.
		if rule.DocumentTypeID == "" && rule.ObjectDocID == "" && rule.DetailTypeDocID == "" &&
			rule.DetailDocument == "" && rule.MinDocument == 0 && !rule.Mandatory {
			continue
		}
		clean.Rules = append(clean.Rules, rule)
	}
	return clean
}

// Validate menegakkan SATU-SATUNYA aturan isian yang benar-benar ada di layar lama.
//
// `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:208-209` menyiapkan
// `Local.MsgErr := "Nama Bisnis belum di isi"` lalu menghentikan langkahnya. Pesannya
// ditiru apa adanya, termasuk ejaan "di isi" yang terpisah, karena petugas yang hafal
// layar lama membaca kalimat yang sama persis (`D-13`).
//
// # Kenapa baris dokumen kosong TIDAK ditolak
//
// Versi pertama modul ini menolaknya, dengan alasan mengganti kegagalan senyap menjadi
// kegagalan yang terbaca. Itu PENYIMPANGAN yang saya buat sendiri: di Pega, penyimpanan
// tanpa baris dokumen sekadar memutari perulangannya nol kali lalu menutup layar. Work
// Owner menetapkan 2026-09-23 layar ini mengikuti Pega apa adanya, sehingga penolakan itu
// dicabut — penyimpanan tanpa baris dokumen berhasil dan tidak menyimpan apa pun.
func (b BatchInput) Validate() error {
	if len(b.BusinessIDs) == 0 {
		return ErrBusinessRequired
	}
	return nil
}

var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("daftartipedokumenbisnis: aturan dokumen tidak ditemukan")

	// ErrBusinessRequired: tidak satu pun lini bisnis dipilih.
	//
	// Satu-satunya aturan isian yang ditiru dari layar lama — lihat BatchInput.Validate.
	ErrBusinessRequired = errors.New("daftartipedokumenbisnis: nama bisnis belum diisi")
)

// SequenceDigits adalah lebar nomor urut pada ID baris.
//
// Empat, dibaca langsung dari `Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22`:
//
//	id_site || lpad(to_Char(LST_TYPE_DOC_BUSINESS_SEQ.nextval),4,'0')
//
// Sama dengan Daftar Tipe Dokumen, dan berbeda dari Daftar Detail Dokumen Travel yang
// memakai lima. Angkanya tidak pernah boleh diseragamkan antarmodul: masing-masing
// mengikuti procedure-nya sendiri, dan menyeragamkannya akan menerbitkan ID berbentuk
// lain daripada yang sudah ada di tabelnya.
const SequenceDigits = 4

// FormatID menyusun ID dari kode situs dan nomor urut.
//
// Nomor yang melampaui lebar padding dikembalikan apa adanya, meniru LPAD Oracle yang
// tidak memotong nilai yang lebih panjang dari lebarnya. Penyisipan ke-10000 karena itu
// menghasilkan ID satu karakter lebih panjang — keadaan yang perlu diketahui DBA sebelum
// terjadi, bukan sesudah, dan dicatat di migrasinya.
func FormatID(site string, sequence int64) string {
	digits := strconv.FormatInt(sequence, 10)
	for len(digits) < SequenceDigits {
		digits = "0" + digits
	}
	return site + digits
}

// Repo adalah seam ke penyimpanan aturan dokumen SATU portal.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Satu instans selalu terikat pada satu basis data entitas —
// pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat kueri (`ADR-0030`).
//
// Tidak ada Delete, dan itu bukan kelalaian: layar lama tidak punya tombol hapus sama
// sekali — tujuh tombolnya Tambah, Ubah, Copy, Detail, Simpan, Simpan Doc PA, dan Tamban
// semua bisnis NONMBU — dan `D-66` melarang penghapusan fisik data bernilai bisnis.
type Repo interface {
	// ListBusinesses mengembalikan lini bisnis yang SUDAH punya aturan dokumen, terurut
	// menurut namanya. Ia mengisi grid tingkat pertama.
	ListBusinesses(ctx context.Context) ([]Business, error)

	// ListByBusiness mengembalikan aturan dokumen milik satu lini bisnis, terurut
	// menurut ID seperti `Select_TYPE_DOCUMENT` (`ORDER BY a.id ASC`).
	//
	// Daftar KOSONG bukan galat: bisnis yang belum punya aturan memang belum punya
	// baris, dan layar Tambah menuju ke sana.
	ListByBusiness(ctx context.Context, businessID string) ([]DocumentRule, error)

	// Get mengembalikan satu aturan LENGKAP dengan daftar jaminannya.
	Get(ctx context.Context, id string) (DocumentRule, error)

	// InsertBatch menyisipkan perkalian bisnis kali baris aturan, dan mengembalikan
	// seluruh baris yang benar-benar tersimpan.
	//
	// Seluruhnya dalam SATU transaksi. `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc` melakukan
	// sebaliknya — ia dipanggil sekali per baris, masing-masing dengan COMMIT-nya
	// sendiri di `UpdateDetTypeDocBusiness_SQL`, sehingga kegagalan di tengah
	// meninggalkan sebagian bisnis terisi dan sebagian tidak, tanpa satu pun tanda.
	// `D-68` memindahkan kepemilikan transaksi ke Go persis untuk kelas kegagalan ini.
	InsertBatch(ctx context.Context, input BatchInput, by Editor) ([]DocumentRule, error)

	// Update mengubah satu baris aturan; ErrNotFound bila barisnya hilang di antara
	// pemuatan layar dan penyimpanan.
	//
	// BUSINESSID tidak ikut berubah — lihat DocumentRule.BusinessID.
	Update(ctx context.Context, id string, input Input, by Editor) (DocumentRule, error)

	// AddCoverage menambahkan satu jaminan pada sebuah aturan, dan tidak melakukan apa
	// pun bila jaminan itu sudah ada.
	//
	// # Kenapa menambah, bukan mengganti seluruh daftarnya
	//
	// Karena begitulah sistem lama menyimpannya, dan bentuknya tegas di
	// `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:51-61`:
	//
	//	select count(id) into countcvg from POOLDATA.COVERAGE_DOC_BUSINESS
	//	 where coverageid = tCvg and id = IDPega;
	//	if countcvg = 0 then insert ... end if;
	//
	// Tidak ada DELETE di mana pun terhadap tabel itu — tidak di procedure, tidak di satu
	// pun rule SQL. Modul Daftar Detail Dokumen Travel mengganti daftar anaknya secara
	// menyeluruh, dan MENIRUNYA DI SINI akan salah: penggantian menyeluruh menuntut
	// penghapusan, dan penghapusan jaminan mengubah dokumen yang tadinya wajib menjadi
	// tidak wajib pada klaim yang sedang berjalan.
	AddCoverage(ctx context.Context, id string, coverageID string) (DocumentRule, error)
}

// Editor adalah jejak siapa menyimpan dan kapan.
//
// Keduanya ditulis sistem lama juga — `InsertDetailTypeDocumentBusiness_act` mengisi
// `InputData.USER_EDIT := OperatorID.pyUserIdentifier` (`:574`) dan
// `InputData.TGL_EDIT := @getCurrentTimeStamp()` (`:516`) — sehingga meniadakannya akan
// mengosongkan kolom yang hari ini terisi.
type Editor struct {
	// Identity mengisi USER_EDIT. Kosong tidak menghalangi penyimpanan: ia hanya
	// mengisi kolom jejak, dan menolak penyimpanan karenanya akan membuat petugas
	// kehilangan isian yang sudah diketik demi kolom yang tidak dilihat siapa pun di
	// layar.
	Identity string

	// At mengisi EDIT_DATE, datang dari seam Clock (`F-5`) dan bukan dari jam basis
	// data. Tanpa itu waktunya mengikuti zona waktu server basis data (`R-12`) dan
	// penyimpanan tidak dapat diuji secara deterministik.
	At time.Time
}

// BusinessRepo adalah seam BACA-SAJA ke master lini bisnis (POOLDATA.BUSINESS).
//
// Tabelnya dimiliki GISFW (`D-03`). Modul Master COL Simas Online sudah memakai seam
// berbentuk sama atas tabel yang sama; keduanya punya salinannya sendiri alih-alih saling
// mengimpor, karena yang dibagi adalah tabelnya, bukan kodenya.
type BusinessRepo interface {
	// List mengembalikan SELURUH lini bisnis — bukan hanya yang sudah punya aturan
	// dokumen. Ia mengisi pilihan pada layar Tambah, dan di sanalah bisnis yang belum
	// punya satu baris pun justru paling sering dituju.
	List(ctx context.Context) ([]Business, error)
}

// DocumentTypeRepo adalah seam BACA-SAJA ke master tahap dokumen (V_LST_DOC_TYPE).
//
// Tabelnya dimiliki modul Daftar Tipe Dokumen, MENU_ID 40 (`P-1`).
type DocumentTypeRepo interface {
	List(ctx context.Context) ([]Reference, error)
}

// DetailTypeDocRepo adalah seam BACA-SAJA ke master rincian dokumen
// (V_LST_DET_TYPE_DOC).
//
// Tabelnya dimiliki modul Daftar Detail Tipe Dokumen, MENU_ID 41 (`P-1`).
//
// Seam-nya tetap milik modul ini, bukan diimpor dari sana: modul tidak saling mengimpor
// tipenya, sehingga perubahan di satu modul tidak merambat ke modul lain. Yang dibagi
// adalah tabelnya, bukan kodenya — perlakuan yang sama dipakai DocumentRepo pada modul
// Daftar Detail Dokumen Travel.
type DetailTypeDocRepo interface {
	List(ctx context.Context) ([]Reference, error)
}

// ObjectDocRepo adalah seam BACA-SAJA ke master objek dokumen (V_LST_DOC_OBJ).
type ObjectDocRepo interface {
	List(ctx context.Context) ([]Reference, error)
}

// Reference adalah satu pilihan pada isian yang merujuk master lain.
//
// Satu tipe untuk tiga seam, bukan tiga tipe kembar: ketiganya benar-benar berbentuk
// sama — sebuah kode, sebuah nama, dan pada salah satunya sebuah induk.
type Reference struct {
	ID   string
	Name string

	// ParentID mengikat pilihan ini pada pilihan lain, dan HANYA terisi pada daftar
	// rincian dokumen.
	//
	// Di sana ia DOC_TYPE_ID — tahap dokumen pemilik rincian itu. Ia ada karena isian
	// Detail Dokumen di layar lama TIDAK menampilkan seluruh master: autocomplete-nya
	// `BrowseVLstDetTypeDoc_RD` menerima parameter `idDocument` yang diisi tahap dokumen
	// yang sudah dipilih pada baris yang sama, sehingga daftarnya menyempit mengikutinya.
	//
	// Menghilangkannya akan menampilkan rincian milik tahap lain sebagai pilihan yang
	// sah — dan baris yang tersimpan karenanya tidak akan pernah muncul di layar unggah
	// mana pun, karena keenam kueri unggah menyaring menurut tahap.
	//
	// Penyaringannya dikerjakan LAYAR atas daftar yang sudah di tangan, bukan server per
	// permintaan. Sebabnya sama dengan daftar jaminan pada modul Daftar Detail Dokumen
	// Travel: grid di form dapat memuat banyak baris dengan tahap berbeda-beda, dan
	// menyaring di server berarti satu permintaan per baris grid setiap kali tahapnya
	// berganti.
	ParentID string
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca saat
// permintaan datang — bukan diputuskan sekali ketika aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis aturan dokumen
// satu badan hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// BusinessRepoSelector memilih BusinessRepo milik satu portal entitas.
type BusinessRepoSelector func(portalAlias string) (BusinessRepo, error)

// DocumentTypeRepoSelector memilih DocumentTypeRepo milik satu portal entitas.
type DocumentTypeRepoSelector func(portalAlias string) (DocumentTypeRepo, error)

// DetailTypeDocRepoSelector memilih DetailTypeDocRepo milik satu portal entitas.
type DetailTypeDocRepoSelector func(portalAlias string) (DetailTypeDocRepo, error)

// ObjectDocRepoSelector memilih ObjectDocRepo milik satu portal entitas.
type ObjectDocRepoSelector func(portalAlias string) (ObjectDocRepo, error)
