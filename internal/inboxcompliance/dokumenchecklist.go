package inboxcompliance

// DocumentChecklistItem adalah satu baris tab **Dokumen** form Compliance Checker.
//
// # Dari mana bentuk ini berasal
//
// `Section/ComplianceChecker-Section.xml` layout **S3**, berjudul `Dokumen`, bersyarat
// `IsTravel` (`pyContainerVisibleWhen`). Ia menyisipkan `Section/UploadDocument_sect.xml`
// (`pyInclude=UploadDocument`, `pyType=SUB_SECTION`) ruleset `GCNMFW`.
//
// Tab ini TIDAK ditemukan dari membaca rule — ia ditemukan karena Work Owner membuka
// layar Pega untuk klaim Travel dan memperlihatkannya. Pembacaan saya sebelumnya berhenti
// di S1, S2, dan S4 karena ketiganya sudah menjelaskan layar PA yang saya lihat. Dicatat
// di sini, bukan dirapikan diam-diam.
//
// # Ia daftar periksa, BUKAN daftar berkas
//
// Perbedaan ini menentukan seluruh rancangannya. Barisnya adalah **kategori dokumen yang
// seharusnya ada** untuk lini bisnis klaim itu — bukan berkas yang sudah diunggah.
// Kategori yang belum punya berkas pun tetap muncul, justru itu gunanya.
//
// Jangan tertukar dengan grid dokumen pada tab Compliance yang dicabut 2026-10-08: yang
// itu daftar berkas, dan Pega memang tidak memuatnya dari basis data.
//
// # Kolomnya, dan sumbernya
//
//	"Kategori"              DETAIL_DOKUMEN    LST_TYPE_DOC_BUSINESS
//	"Wajib Unggah"          STS_WAJIB         idem, lewat CASE yang menyertakan cabang
//	"Minimal Unggah"        MIN_DOC           idem
//	"Total Sudah Diunggah"  dihitung          POOLDATA.DATA_ATTACHFILE
//
// Kuncinya `DOC_TYPE_DT_ID`, dan ia tersambung ke lampiran lewat
// `DATA_ATTACHFILE.CATEGORY` — terbukti dari `GetTypeImage-SQL.xml`, yang mencari
// `DETAIL_DOKUMEN` berdasarkan `DOC_TYPE_DT_ID`, dan `CountUpload-SQL.xml`, yang membaca
// `CATEGORY` dari `DATA_ATTACHFILE`.
//
// Itu sekaligus menjelaskan angka `10064` yang tampil di kolom "Category" grid dokumen
// kami sebelum dicabut: ia **id jenis dokumen**, bukan nama kategori.
type DocumentChecklistItem struct {
	// CategoryID adalah `DOC_TYPE_DT_ID` — kunci yang menyambung baris ini ke lampiran.
	//
	// Tidak digambar di layar; ia dibawa karena tombol Unggah menempelkan berkas pada
	// kategori ini, dan tombol Ubah Kategori memindahkannya.
	CategoryID string

	// CategoryName adalah kolom "Kategori" — `DETAIL_DOKUMEN`.
	CategoryName string

	// MandatoryLabel adalah kolom "Wajib Unggah" — `"Ya"`, `"Tidak"`, atau KOSONG.
	//
	// Teks, bukan boolean, dan itu disengaja. `BrowseRegister_upload-SQL.xml` menghasilkan
	// tiga keadaan, bukan dua:
	//
	//	STS_WAJIB = '0'                                  -> "Tidak"
	//	STS_WAJIB = '1' dan OBJECT_DOC_ID cocok cabang   -> "Ya"
	//	STS_WAJIB = '1' dan tidak cocok, atau NULL       -> "Tidak"
	//	STS_WAJIB NULL atau lainnya                      -> (tidak ada arm) -> KOSONG
	//
	// Keadaan keempat itu bukan karangan: pada layar Pega yang diperlihatkan Work Owner,
	// kolom ini **kosong di seluruh baris** sementara "Minimal Unggah" terisi. Boolean
	// akan memaksa kosong menjadi `false`, yakni "Tidak" — dan itu mengubah arti.
	MandatoryLabel string

	// MinUpload adalah kolom "Minimal Unggah" — `MIN_DOC`.
	//
	// Pointer: kolomnya boleh NULL, dan `0` punya arti berbeda dari "tidak ditentukan".
	MinUpload *int

	// UploadedCount adalah kolom "Total Sudah Diunggah".
	//
	// Dihitung dari `DATA_ATTACHFILE` dengan syarat `IMAGEID IS NOT NULL` — meniru
	// `CountUpload-SQL.xml`. Baris tanpa kunci penyimpanan TIDAK dihitung: berkasnya tidak
	// pernah dapat dibuka, sehingga menghitungnya akan membuat kategori tampak lengkap
	// padahal tidak ada yang bisa dibaca.
	UploadedCount int
}

// DocumentChecklistStage adalah tahap dokumen yang DULU disaring tab ini.
//
// # TIDAK LAGI DIPAKAI MENYARING — dicabut 2026-10-10 atas pengukuran
//
// Nilainya tinggal sebagai penanda pada keluaran `claimpnc -periksa`, supaya cacahan per
// tahap tetap punya rujukan yang terbaca. Kuerinya sendiri tidak lagi menyaring tahap.
//
// Sebabnya bukan ragu melainkan ukur. Basis data dev dibaca langsung:
//
//	seluruh LST_TYPE_DOC_BUSINESS   1848 baris bertahap KOSONG, 2 baris bertahap "Dokumen"
//	lini Travel klaim contoh        6 baris, SELURUHNYA bertahap NULL
//
// Tidak ada satu pun baris bertahap `REGISTER`. Dengan penyaring itu, tab Dokumen akan
// KOSONG untuk setiap klaim Travel — bukan sekadar belum terbukti, melainkan terbukti
// salah. Tanpanya ia mengembalikan keenam kategorinya.
//
// Bila kelak produksi mengisi `TYPE_DOCUMENT`, penyaringnya mungkin perlu kembali.
// Pengukurannya dapat diulang kapan saja: `claimpnc -periksa` mencetak cacahan per tahap.
//
// # Catatan sejarah — kenapa `REGISTER` sempat dipilih
//
// # INI SATU-SATUNYA ASUMSI PADA SELURUH TAB — dan ia belum terbukti
//
// Master `V_LST_DOC_TYPE.TYPE_DOCUMENT` punya enam tahap, terbaca dari keenam rule
// `Browse*_upload`:
//
//	REGISTER · SURVEY · COMMITEE · PAYMENT · SALVAGE · COLLECTING DOCUMENT
//
// **Tidak ada tahap `COMPLIANCE`.** Tahap mana yang dipakai tab ini tertanam di dalam
// Report Definition `BrowseUpRegisterDoc_rd` — dan RD itu **tidak ada di export**
// (`R-16`), padahal dirujuk enam section.
//
// Tiga jalan untuk memastikannya sudah ditempuh dan buntu: RD-nya hilang, section tidak
// mengirim tahap sebagai parameter, dan master jenis dokumen tidak termasuk sembilan CSV
// yang sudah dikirim.
//
// `REGISTER` dipilih atas dua penalaran, dan keduanya penalaran — bukan bacaan:
//
//  1. nama RD-nya sendiri memuatnya: `BrowseUp`**`Register`**`Doc_rd`
//  2. RD yang SAMA dipakai enam layar termasuk Input Register, sehingga tahapnya tidak
//     mungkin berbeda per layar
//
// # Keduanya MELEMAH pada 2026-10-10 — baca ini sebelum memercayainya
//
// Pembacaan ulang `UploadDocument_sect.xml` menemukan ia memuat TUJUH blok, satu per
// tahap dokumen:
//
//	PENDAFTARAN · DOKUMEN TRAVEL · SURVEI · DOKUMEN LAIN-LAIN · PEMBAYARAN · KOMITE · SALVAGE
//
// Travel punya **bloknya sendiri** — `DOKUMEN TRAVEL`, bukan blok `PENDAFTARAN`. Dan RD
// yang sama dirujuk DUA kali: sekali di blok PENDAFTARAN, sekali di blok DOKUMEN TRAVEL.
//
// Karena itu penalaran (1) menerangkan blok pertama, belum tentu yang kedua. Pembedanya
// dicari di sekitar kedua rujukan — tidak ada parameter, dan tidak ada properti repeat
// yang membedakannya. Ia ada di dalam RD yang hilang.
//
// Artinya ini **dugaan**, bukan lagi dugaan beralasan kuat. Kueri pemastian di
// `docs/permintaan-dba-compliance-2.sql` karena itu naik dari berguna menjadi PERLU.
//
// Cara memastikannya ada di `docs/permintaan-dba-compliance-2.sql`: hitung kategori per
// tahap, lalu cocokkan dengan jumlah baris grid di layar Pega. Bila ternyata bukan
// `REGISTER`, yang berubah **satu baris** pada kueri `find_document_checklist`.
const DocumentChecklistStage = "REGISTER"
