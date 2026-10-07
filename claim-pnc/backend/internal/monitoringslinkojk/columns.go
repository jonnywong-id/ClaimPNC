package monitoringslinkojk

import (
	"claim-pnc/internal/platform/tabletext"
)

// Katalog kolom kedua segmen — untuk grid di layar maupun untuk berkas CSV.
//
// ============================================================================
// KENAPA KATALOG, DAN APA YANG DIJAGANYA
// ============================================================================
//
// Satu daftar melayani TIGA pemakai sekaligus — grid di layar, berkas CSV, dan uji.
// Menuliskannya tiga kali berarti tiga daftar yang harus berubah bersamaan, dan yang
// tertinggal tidak menghasilkan satu pun galat: kolom laporan regulator yang diam-diam
// bergeser hanya terlihat oleh OJK.
//
// Presedennya `reportklaim/columns_*.go`, yang menyalin ~700 kolom dari 28 laporan.
//
// ============================================================================
// PENAMAAN DI AREA INI TERBALIK SECARA SISTEMATIS — baca ini sebelum menyunting
// ============================================================================
//
// Empat artefak di area SLIK dinamai dengan segmen yang SALAH. Ini bukan tiga kekeliruan
// terpisah melainkan satu pola, dan siapa pun yang menyunting berkas ini akan
// tersandung bila tidak mengetahuinya lebih dulu:
//
//	Artefak Pega                       Namanya menyebut   Isinya sebenarnya
//	---------------------------------  -----------------  ------------------------------
//	RDB List/GetDataSlinkAllFOGF06     F06                kueri segmen D01 (fasilitas)
//	RDB List/GetDataSlinkAllFOG        FOG                kueri segmen F06 (debitur)
//	Activity/ExportDataSlinkD01        D01                kolom D01, berkas "Laporan F06 SLIK OJK"
//	Activity/ExportDataSlinkFOG        FOG                kolom F06, berkas "Laporan SLIK OJK D01"
//
// Kedua nama berkas unduhan itu **tertukar satu sama lain**: ekspor segmen D01
// menghasilkan berkas bernama F06, dan sebaliknya. Terverifikasi dari elemen `<FileName>`
// masing-masing aktivitas.
//
// Nama berkas di modul ini karena itu **DIBETULKAN**, tidak direplikasi — lihat FileName.
//
// ============================================================================
// JUDUL KOLOM DISALIN APA ADANYA — TERMASUK SALAH KETIKNYA
// ============================================================================
//
// `Sec_SegmentD01_1-Section.xml` menulis **"Kode Kelektibilitas"**; ejaan yang benar
// "Kolektibilitas", dan tabelnya sendiri menamai kolomnya `KODEKOLEKTABILITAS` — ejaan
// ketiga yang juga keliru. Ketiganya dibiarkan apa adanya di tempatnya masing-masing.
//
// Alasannya bukan kesetiaan buta. Berkas CSV ini dibaca ulang oleh berkas kerja dan makro
// yang sudah ada di sisi pelapor; merapikan judul kolom merusak keduanya tanpa ada yang
// meminta. `D-13` pun menetapkan teks yang dilihat pengguna mengikuti layar Pega.

// ColumnSource menyatakan dari mana satu kolom terisi.
//
// Ia ikut dikirim ke layar, dan itu bukan kerapian melainkan syarat agar layar 38 kolom
// dapat dipercaya: pengguna harus dapat membedakan **"kosong karena datanya memang
// kosong"** dari **"kosong karena kolomnya belum punya sumber"**. Keduanya terlihat sama
// persis di sebuah sel, dan pada laporan ke regulator perbedaannya menentukan.
type ColumnSource string

const (
	// SourceAvailable — kolom terisi dari kueri.
	SourceAvailable ColumnSource = "tersedia"

	// SourceMissing — kolom ADA di layar lama tetapi TIDAK punya sumber di export.
	//
	// Bukan kelalaian migrasi; lihat catatan panjang pada f06Columns.
	SourceMissing ColumnSource = "belum-ada-sumber"
)

// Column adalah satu kolom grid sekaligus satu kolom berkas CSV.
type Column struct {
	// Key adalah kunci pada Row dan nama field pada JSON.
	//
	// `snake_case` berbahasa Indonesia, sejalan dengan nama field JSON modul lain —
	// kontrak yang dibaca frontend, dan termasuk pengecualian `D-80`.
	Key string

	// Header adalah judul yang dilihat pengguna, disalin dari `pyCaption` APA ADANYA.
	Header string

	// LegacyProperty adalah nama properti klipboard Pega yang mengisinya, diambil dari
	// `CSVProperties` milik langkah `pxConvertResultsToCSV`.
	//
	// Ia disimpan supaya pertanyaan "kolom ini dari mana" dapat dijawab dengan menunjuk
	// berkasnya, bukan dengan menelusuri ulang export 1,1 MiB. Di modul ini jejak itu
	// lebih dibutuhkan daripada biasanya: aliasnya menyesatkan pada hampir setiap kolom
	// — "Tempat Bekerja" terikat `ASMNotes`, "Kelurahan" terikat `ASMRWNote`.
	//
	// KOSONG berarti kolomnya tidak punya properti sama sekali di export.
	LegacyProperty string

	// Source menyatakan kolomnya terisi atau belum punya sumber.
	Source ColumnSource

	// Numeric menandai kolom bernilai angka, supaya layar meratakannya ke kanan.
	//
	// Nilainya TETAP dikirim sebagai teks — pemformatan uang dan desimal dilakukan di
	// satu tempat, dan mengirimnya sebagai angka JSON akan membuat peramban
	// memformatnya untuk kedua kalinya dengan aturan yang berbeda.
	Numeric bool
}

// Headers mengembalikan judul kolom saja, sesuai urutannya. Dipakai baris pertama CSV.
func Headers(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Header
	}
	return out
}

// Keys mengembalikan kunci kolom saja, sesuai urutannya.
func Keys(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Key
	}
	return out
}

// Columns mengembalikan katalog kolom GRID satu segmen.
//
// Senarai BARU setiap kali, bukan senarai bersama: pemanggil yang mengurutkan atau
// memotongnya tidak boleh mengubah katalog bagi pemanggil berikutnya.
func Columns(segment Segment) []Column {
	switch segment {
	case SegmentD01:
		return append([]Column(nil), d01Columns...)
	case SegmentF06:
		return append([]Column(nil), f06Columns...)
	}
	return nil
}

// ExportHeaders mengembalikan BARIS KEPALA berkas CSV satu segmen.
//
// Ia terpisah dari ExportSlots karena pada segmen F06 **jumlah keduanya berbeda** — 38
// judul berbanding 34 kolom data. Lihat ExportSlots.
//
// # Kenapa D01 berbeda dari gridnya
//
// Karena di sistem lama pun berbeda. Grid D01 menampilkan **20 kolom**, sedangkan
// `ExportDataSlinkD01` menulis **27** — tujuh di antaranya tidak pernah terlihat di layar
// (`ObjectName`, `KodeKantorCabang`, `OperasiData`, `NoKTP`, `NPWPPerusahaan`, `PolicyNo`,
// dan `Keterangan` untuk KEDUA kalinya).
//
// Berkas itu yang dikirim ke OJK; gridnya hanya untuk memantau.
func ExportHeaders(segment Segment) []string {
	switch segment {
	case SegmentD01:
		return Headers(d01ExportColumns)
	case SegmentF06:
		return Headers(f06ExportColumns)
	}
	return nil
}

// ExportSlots mengembalikan KOLOM DATA berkas CSV satu segmen — satu sel per slot.
//
// ============================================================================
// PADA SEGMEN F06 JUMLAHNYA SENGAJA TIDAK SAMA DENGAN JUDULNYA
// ============================================================================
//
// 38 judul, **34** kolom data. Itu BUKAN cacat modul ini melainkan replikasi dari
// `Activity/ExportDataSlinkFOG-Act.xml`, dan replikasinya **diputuskan Work Owner pada
// 2026-09-26** sesudah selisihnya disampaikan beserta akibatnya.
//
// `pxConvertResultsToCSV` memasangkan judul dengan properti menurut POSISI, dan di
// aktivitas itu jumlahnya tidak sama:
//
//	CSVPropHeaders  -> 38 judul
//	CSVProperties   -> 34 nama properti
//
// Tiga judul tidak punya properti sama sekali — "KodeStatusPendidikan", "Tempat Lahir",
// dan "NPWP" — dan satu entri properti adalah dua nama yang TERSAMBUNG tanpa koma:
// `ASMGenderASMDateOfBirth`, yang mestinya `ASMGender` dan `ASMDateOfBirth`. Nama
// sambungan itu tidak pernah ada sebagai properti, sehingga slotnya selalu kosong.
//
// Akibatnya seluruh kolom sejak slot ketujuh **bergeser terhadap judulnya**:
//
//	slot  7  ASMAddress        -> tercetak di bawah judul "Jenis Kelamin"
//	slot 33  KodeKantorCabang  -> tercetak di bawah judul "Perjanjian Pisah Harta"
//	slot 34  OperasiData       -> tercetak di bawah judul "Melanggar BMPK"
//
// Ketiga slot itulah satu-satunya yang terisi. Dua puluh delapan slot lainnya menyebut
// properti CIF (`ASMNIK`, `pyFullName`, `SpouseName`, …) yang **tidak dihasilkan**
// `GetDataSlinkAllFOG`, sehingga kosong.
//
// Modul ini karena itu menghasilkan berkas yang **hampir seluruhnya kosong dan bergeser**
// — persis seperti Pega. Apakah berkas di produksi memang begitu bentuknya masih perlu
// diuji terhadap satu berkas ekspor sungguhan; tercatat di
// `docs/permintaan-artefak-pega.md` §4.2 butir 4.
func ExportSlots(segment Segment) []Column {
	switch segment {
	case SegmentD01:
		return append([]Column(nil), d01ExportColumns...)
	case SegmentF06:
		return append([]Column(nil), f06ExportSlots...)
	}
	return nil
}

// FileName adalah nama berkas CSV hasil ekspor, tanpa akhiran.
//
// ============================================================================
// KEDUANYA TERTUKAR, DAN ITU DIREPLIKASI
// ============================================================================
//
// `ExportDataSlinkD01` — yang menulis kolom segmen D01 — memberi berkasnya nama
// **"Laporan F06 SLIK OJK"**, sedangkan `ExportDataSlinkFOG` yang menulis kolom segmen
// F06 memberinya nama **"Laporan SLIK OJK D01"**. Keduanya tertukar satu sama lain.
//
// Versi pertama modul ini membetulkannya. Work Owner memutuskan pada 2026-09-26 bahwa
// namanya **disamakan persis dengan Pega**, sesudah selisihnya disampaikan beserta
// akibatnya: pelapor menyimpan dua berkas yang saling tertukar namanya di folder yang
// sama, dan yang keliru membukanya tidak punya cara mengetahuinya.
//
// Nilai di bawah karena itu disalin APA ADANYA dari elemen `<FileName>` masing-masing
// aktivitas — termasuk ketertukarannya. Jangan "dirapikan": ia diuji di
// TestFileNamesMatchPega.
func FileName(segment Segment) string {
	switch segment {
	case SegmentD01:
		// Ya, F06 — lihat catatan di atas. Ini nama berkas ekspor segmen D01.
		return "Laporan F06 SLIK OJK"
	case SegmentF06:
		// Ya, D01 — lihat catatan di atas. Ini nama berkas ekspor segmen F06.
		return "Laporan SLIK OJK D01"
	}
	return "Laporan SLIK OJK"
}

// RowKeyColumn adalah kunci teknis yang ikut dibawa setiap baris TANPA menjadi kolom.
//
// # Kenapa ia tidak menjadi kolom
//
// Segmen F06 di layar lama TIDAK punya kolom No Klaim — ke-38 kolomnya seluruhnya field
// debitur, terverifikasi dari `pyCaption` section maupun dari `CSVPropHeaders` ekspornya.
// Menambahkannya berarti menambah kolom yang tidak ada di Pega, dan `D-13` menetapkan
// tata letak mengikuti layar lama.
//
// # Kenapa ia tetap dibawa
//
// Tabel di layar membutuhkan kunci baris yang stabil, dan mengarang kunci dari nomor urut
// membuat baris tertukar begitu halaman berpindah atau urutannya berubah.
const RowKeyColumn = "kunci_baris"

// f06Columns adalah kolom GRID segmen F06 — dan isinya SAMA PERSIS dengan grid D01.
//
// ============================================================================
// KOREKSI 2026-09-27 — sebelumnya diisi kolom BERKAS EKSPOR, dan itu salah
// ============================================================================
//
// Grid F06 sempat dibangun dari 38 judul `CSVPropHeaders` milik
// `Activity/ExportDataSlinkFOG-Act.xml` — daftar CIF nasabah (`ASMNIK`, `pyFullName`,
// `SpouseName`, …). Tangkapan layar Pega yang berjalan membuktikan itu **keliru**: layar
// "Segment Slik F06" menampilkan **kedua puluh kolom yang sama dengan segmen D01**, dan
// tidak satu pun kolom CIF.
//
// Buktinya juga ada di kueri, dan itu yang seharusnya saya baca lebih dulu:
// `RDB List/GetDataSlinkAllFOG-SQL.xml` — kueri yang mengisi grid ini — menghasilkan alias
// `ClaimID`, `ContractNo`, `NOMORREKENINGFASILITAS`, dan seterusnya. Tidak satu pun alias
// CIF ada di sana. Daftar 38 judul itu milik BERKAS EKSPOR, bukan milik grid, dan kini
// bernama f06ExportColumns sesuai perannya.
//
// Akibat koreksi ini, keputusan "bangun 35 kolom, sisanya kosong" yang diambil pada
// 2026-09-26 **tidak lagi berlaku untuk grid** — ia diambil di atas premis saya yang salah.
// Keputusan itu tetap berlaku untuk berkas ekspornya, yang memang 38 judul.
//
// Keduanya sengaja berbagi satu daftar, bukan disalin: bila kelak satu kolom grid berubah,
// dua grid yang di Pega memang identik tidak boleh menjadi berbeda karena salinan yang
// terlewat diperbarui.
var f06Columns = d01Columns

// ============================================================================
// SEGMEN D01 — FASILITAS KREDIT · GRID
// ============================================================================
//
// Kedua puluh kolom di bawah terpetakan **satu-satu** dengan daftar SELECT pada
// `RDB List/GetDataSlinkAllFOGF06-SQL.xml`, sesuai urutannya. Itulah sebabnya segmen ini
// dapat dibangun utuh sementara F06 tidak.
//
// ---------------------------------------------------------------------------
// SATU KOLOM TABEL MENGISI DUA KOLOM LAYAR — dan itu ada di sumbernya
// ---------------------------------------------------------------------------
//
// Kueri lama memilih `tanggalkondisi` DUA KALI dengan alias berbeda:
//
//	tanggalkondisi as "TanggalPembayaran"   -> kolom layar "Tanggal Pembayaran"
//	tanggalkondisi as "TanggalKondisi"      -> kolom layar "Tanggal Kondisi"
//
// Jadi kolom **"Tanggal Pembayaran" tidak menampilkan tanggal pembayaran**; ia
// menampilkan tanggal kondisi, sama persis dengan kolom di sebelahnya. Keduanya akan
// selalu bernilai sama, dan tidak ada apa pun di layar yang menandakannya.
//
// Tabelnya sendiri TIDAK punya kolom tanggal pembayaran — 28 kolom pada
// `InsertDataSlikOJKF06-SQL.xml` tidak memuatnya. Jadi ini bukan salah alias yang dapat
// diperbaiki dengan menunjuk kolom lain; datanya memang tidak disimpan.
//
// Perilakunya **direplikasi**, bukan diperbaiki: `P-5` menetapkan hasil yang benar adalah
// hasil yang sama dengan Pega kecuali 13 butir `D-49`, dan ini bukan salah satunya.
// Mengubahnya berarti mengosongkan satu kolom laporan OJK atas dasar tebakan.
var d01Columns = tabletext.Rows[Column](`
	Key                      | Header                      | LegacyProperty         | Source   | Numeric
	no_klaim                 | No Klaim                    | ClaimID                | tersedia |
	contract_no              | Contract No                 | ContractNo             | tersedia |
	nomor_rekening_fasilitas | Nomor Rekening Fasilitas    | NOMORREKENINGFASILITAS | tersedia |
	no_cif_debitur           | No CIF Debitur              | NOMORCIFDEBITUR        | tersedia |
	kode_jenis_fasilitas     | Kode Jenis Fasilitas        | KodeJenisFasilitas     | tersedia |
	sumber_dana              | Sumber Dana                 | SumberDana             | tersedia |
	start_polis              | Start Polis                 | AwalPolis              | tersedia |
	end_polis                | End Polis                   | AkhirPolis             | tersedia |
	suku_bunga               | Suku Bunga                  | SukuBunga              | tersedia | true
	kode_valuta              | Kode Valuta                 | KODEVALUTA             | tersedia |
	// Lihat catatan di atas: terisi dari 'tanggalkondisi', bukan dari tanggal pembayaran.
	//
	// Urutannya — SEBELUM "Nilai Mata Uang Asal" — diambil dari tangkapan layar Pega yang
	// berjalan, bukan dari urutan SELECT. Keduanya memang berbeda: di 'GetDataSlinkAllFOGF06'
	// alias ini berada tepat sesudah 'kodejenisfasilitas', sedangkan di layar ia berada
	// sesudah "Kode Valuta". Yang menentukan tata letak grid adalah section, bukan kueri.
	tanggal_pembayaran       | Tanggal Pembayaran          | TanggalPembayaran      | tersedia |
	nilai_mata_uang_asal     | Nilai Mata Uang Asal        | NilaiMataUangAsal      | tersedia |
	// Judulnya salah ketik di section lama ("Kelektibilitas"), dan kolom tabelnya salah
	// ketik pula ("KODEKOLEKTABILITAS"). Keduanya dibiarkan apa adanya.
	kode_kolektibilitas      | Kode Kelektibilitas         | KodeKolektibilitas     | tersedia |
	// Judulnya menyebut sendiri isinya: tanggal REGISTRASI klaim, bukan tanggal macet.
	// 'GetDataSlinkAllFOG-SQL.xml' mengisi kolom padanannya dari 'pnc.REGISTERDATE'.
	tanggal_macet            | Tanggal Macet (Regist Date) | TanggalMacet           | tersedia |
	kode_sebab_macet         | Kode Sebab Macet            | KodeSebabMacet         | tersedia |
	tunggakan                | Tunggakan                   | TUNGGAKAN              | tersedia | true
	jumlah_kewajiban         | Nominal / Jumlah Kewajiban  | JUMLAHKEWAJIBAN        | tersedia | true
	tanggal_kondisi          | Tanggal Kondisi             | TanggalKondisi         | tersedia |
	kode_kondisi             | Kode Kondisi                | KodeKondisi            | tersedia |
	keterangan               | Keterangan                  | Keterangan             | tersedia |
`)

// ============================================================================
// SEGMEN D01 — BERKAS CSV
// ============================================================================
//
// Disalin dari `CSVPropHeaders` dan `CSVProperties` milik langkah `pxConvertResultsToCSV`
// pada `Activity/ExportDataSlinkD01-Act.xml`. **27 judul, 27 properti** — keduanya
// sejajar, berbeda dari segmen F06.
//
// Dua hal disalin apa adanya karena keduanya memang ada di sumbernya:
//
//   - **`Keterangan` muncul DUA KALI** (kolom 21 dan 24), dengan properti yang sama.
//     Berkas yang sampai ke pelapor memang punya dua kolom berjudul sama dan berisi sama.
//   - **`ObjectName` tidak akan pernah terisi.** Kolom itu ada di `GetDataSlinkAllFOG`
//     (kueri segmen F06), BUKAN di `GetDataSlinkAllFOGF06` yang mengisi grid D01. Ia
//     karena itu ditandai SourceMissing — kolomnya ada di berkas, nilainya tidak.
var d01ExportColumns = tabletext.Rows[Column](`
	Key                      | Header                 | LegacyProperty         | Source           | Numeric
	no_klaim                 | ClaimID                | ClaimID                | tersedia         |
	contract_no              | ContractNo             | ContractNo             | tersedia         |
	nama_objek               | ObjectName             | ObjectName             | belum-ada-sumber |
	nomor_rekening_fasilitas | NOMORREKENINGFASILITAS | NOMORREKENINGFASILITAS | tersedia         |
	no_cif_debitur           | NOMORCIFDEBITUR        | NOMORCIFDEBITUR        | tersedia         |
	kode_jenis_fasilitas     | KodeJenisFasilitas     | KodeJenisFasilitas     | tersedia         |
	sumber_dana              | SumberDana             | SumberDana             | tersedia         |
	start_polis              | AwalPolis              | AwalPolis              | tersedia         |
	end_polis                | AkhirPolis             | AkhirPolis             | tersedia         |
	suku_bunga               | SukuBunga              | SukuBunga              | tersedia         | true
	kode_valuta              | KODEVALUTA             | KODEVALUTA             | tersedia         |
	tanggal_pembayaran       | TanggalPembayaran      | TanggalPembayaran      | tersedia         |
	nilai_mata_uang_asal     | NilaiMataUangAsal      | NilaiMataUangAsal      | tersedia         |
	kode_kolektibilitas      | KodeKolektibilitas     | KodeKolektibilitas     | tersedia         |
	tanggal_macet            | TanggalMacet           | TanggalMacet           | tersedia         |
	kode_sebab_macet         | KodeSebabMacet         | KodeSebabMacet         | tersedia         |
	tunggakan                | TUNGGAKAN              | TUNGGAKAN              | tersedia         | true
	jumlah_kewajiban         | JUMLAHKEWAJIBAN        | JUMLAHKEWAJIBAN        | tersedia         | true
	tanggal_kondisi          | TanggalKondisi         | TanggalKondisi         | tersedia         |
	kode_kondisi             | KodeKondisi            | KodeKondisi            | tersedia         |
	keterangan               | Keterangan             | Keterangan             | tersedia         |
	kode_kantor_cabang       | KodeKantorCabang       | KodeKantorCabang       | tersedia         |
	operasi_data             | OperasiData            | OperasiData            | tersedia         |
	// Kolom 24 — 'Keterangan' untuk kedua kalinya. Ada di sumbernya; lihat catatan di atas.
	keterangan               | Keterangan             | Keterangan             | tersedia         |
	no_ktp                   | NoKTP                  | NoKTP                  | tersedia         |
	npwp_perusahaan          | NPWPPerusahaan         | NPWPPerusahaan         | tersedia         |
	no_polis                 | PolicyNo               | PolicyNo               | tersedia         |
`)

// ============================================================================
// SEGMEN F06 — DEBITUR INDIVIDU
// ============================================================================
//
// Urutan dan pemetaannya diambil dari `CSVPropHeaders` + `CSVProperties` milik
// `Activity/ExportDataSlinkFOG-Act.xml` — **bukan** dari urutan elemen di dalam section.
//
// Alasannya: urutan elemen pada XML Pega TIDAK mencerminkan urutan kolom di layar,
// sehingga memasangkan judul dengan properti berdasarkan urutan berkas berarti menebak.
// Daftar CSV adalah satu-satunya tempat di export yang menyandingkan keduanya secara
// eksplisit dan berpasangan.
//
// ---------------------------------------------------------------------------
// BERKAS EKSPOR LAMA MISALIGN — 38 judul, 34 properti
// ---------------------------------------------------------------------------
//
// `pxConvertResultsToCSV` memasangkan judul dengan properti menurut POSISI. Pada
// aktivitas itu jumlahnya tidak sama:
//
//	CSVPropHeaders  -> 38 judul
//	CSVProperties   -> 34 nama properti
//
// Tiga judul tidak punya properti sama sekali — **"KodeStatusPendidikan"**,
// **"Tempat Lahir"**, dan **"NPWP"** — dan satu entri properti adalah dua nama yang
// TERSAMBUNG tanpa koma: `ASMGenderASMDateOfBirth`, yang mestinya `ASMGender` dan
// `ASMDateOfBirth`. Akibatnya seluruh kolom sejak posisi keenam **bergeser**: nilai yang
// benar tertulis di bawah judul yang salah.
//
// Di sini berkasnya dibuat **SEJAJAR** — 38 judul, 38 kolom — dan itu selisih terencana
// yang dicatat, bukan perbaikan diam-diam. Alasannya sempit dan dapat diperiksa: ketiga
// puluh kolom tanpa sumber memang kosong, sehingga berkas yang sejajar TIDAK menambah
// satu pun nilai — ia hanya menaruh kedelapan nilai yang ada di bawah judul yang benar.
// Mereplikasi pergeseran berarti sengaja mengirim nilai ke kolom yang salah pada laporan
// regulator.
//
// ---------------------------------------------------------------------------
// KENAPA 30 DARI 38 KOSONG — dan kenapa itu BUKAN kelalaian migrasi
// ---------------------------------------------------------------------------
//
// Ke-34 properti di atas adalah properti **CIF nasabah** (`ASMClientID`, `pyFullName`,
// `ASMNIK`, `SpouseName`, …) yang diisi `Activity/InsertDataSlinkOJKIndividu-Act.xml`
// dari data Customer, bukan dari kueri yang mengisi grid ini.
//
// Grid F06 diisi `GetAllDataSumbisSlink` → `RDB List/GetDataSlinkAllFOG-SQL.xml`, dan
// kueri itu menghasilkan alias yang BERBEDA. Hanya delapan di antaranya berpadanan, dan
// kedelapan itulah yang ditandai SourceAvailable di bawah.
//
// Kedelapan itu **bukan tebakan**: masing-masing dipasangkan karena nama kolom sumbernya
// menyatakan artinya sendiri — `OBJECTGENDER` untuk Jenis Kelamin, `DATEOFBIRTH` untuk
// Tanggal Lahir, `ASMZIPCODE` untuk Kode Pos. Itu pencocokan, bukan penebakan.
//
// Satu kandidat SENGAJA ditolak. `CUSTOMERTYPE` sempat tampak mengisi "Kode Golongan
// Debitur", tetapi daftar CSV membuktikan kolom itu terikat `ReporterGroup` — properti
// yang sama sekali lain. Kolom laporan regulator yang terisi SALAH lebih berbahaya
// daripada yang kosong: yang kosong terlihat, yang salah tidak.
//
// Konsekuensinya bagi produksi, dan ini yang perlu dikonfirmasi ke tim Pega: berkas
// ekspor F06 hari ini kemungkinan besar **hampir seluruhnya kosong**. Dicatat sebagai
// pertanyaan di `docs/permintaan-artefak-pega.md`, bukan sebagai kesimpulan — ia perlu
// diuji terhadap satu berkas ekspor produksi yang sungguhan.
var f06ExportColumns = tabletext.Rows[Column](`
	Key                              | Header                           | LegacyProperty       | Source           | Numeric
	nomor_cif_debitur                | Nomor CIF Debitur                | ASMClientID          | tersedia         |
	jenis_identitas                  | Jenis Identitas                  | ASMIDType            | belum-ada-sumber |
	nomor_identitas                  | Nomor Identitas                  | ASMNIK               | belum-ada-sumber |
	nama_sesuai_identitas            | Nama Sesuai Identitas            | ASMIDCard            | belum-ada-sumber |
	nama_lengkap                     | Nama Lengkap                     | pyFullName           | belum-ada-sumber |
	// Judul tanpa properti di export; lihat catatan misalign di atas.
	kode_status_pendidikan           | KodeStatusPendidikan             |                      | belum-ada-sumber |
	jenis_kelamin                    | Jenis Kelamin                    | ASMGender            | tersedia         |
	// Judul tanpa properti di export.
	tempat_lahir                     | Tempat Lahir                     |                      | belum-ada-sumber |
	tanggal_lahir                    | Tanggal Lahir                    | ASMDateOfBirth       | tersedia         |
	// Judul tanpa properti di export. Jangan tertukar dengan 'NPWPPerusahaan' pada ekspor
	// D01 — yang ini NPWP DEBITUR.
	npwp                             | NPWP                             |                      | belum-ada-sumber |
	alamat                           | Alamat                           | ASMAddress           | tersedia         |
	// Alias menyesatkan: "Kelurahan" terikat properti catatan RW.
	kelurahan                        | Kelurahan                        | ASMRWNote            | belum-ada-sumber |
	kecamatan                        | Kecamatan                        | ASMDistrict          | belum-ada-sumber |
	kode_kab_kota                    | Kode Kab/Kota                    | ASMCity              | belum-ada-sumber |
	kode_pos                         | Kode Pos                         | PostalCode           | tersedia         |
	telepon                          | Telepon                          | PhoneNo              | tersedia         |
	nomor_telepon_seluler            | Nomor Telepon Seluler            | ASMMobilePhoneList   | belum-ada-sumber |
	alamat_email                     | Alamat Email                     | EmailAddress         | belum-ada-sumber |
	// Alias menyesatkan: "Kode Negara Domisili" terikat properti kewarganegaraan.
	kode_negara_domisili             | Kode Negara Domisili             | ASMNationality       | belum-ada-sumber |
	kode_pekerjaan                   | Kode Pekerjaan                   | ASMJobDesc           | belum-ada-sumber |
	// Alias paling menyesatkan di segmen ini: "Tempat Bekerja" terikat 'ASMNotes'.
	tempat_bekerja                   | Tempat Bekerja                   | ASMNotes             | belum-ada-sumber |
	kode_bidang_usaha_tempat_bekerja | Kode Bidang Usaha Tempat Bekerja | ASMBusinessField     | belum-ada-sumber |
	alamat_tempat_bekerja            | Alamat Tempat Bekerja            | OrganizationAddress  | belum-ada-sumber |
	penghasilan_kotor_per_tahun      | Penghasilan Kotor Per Tahun      | YearlyGrossIncome    | belum-ada-sumber | true
	kode_sumber_penghasilan          | Kode Sumber Penghasilan          | IncomeSource         | belum-ada-sumber |
	jumlah_tanggungan                | Jumlah Tanggungan                | Tanggungan           | belum-ada-sumber | true
	kode_hubungan_dengan_pelapor     | Kode Hubungan Dengan Pelapor     | ReporterRelationship | belum-ada-sumber |
	kode_golongan_debitur            | Kode Golongan Debitur            | ReporterGroup        | belum-ada-sumber |
	status_perkawinan_debitur        | Status Perkawinan Debitur        | ASMMaritalStatus     | belum-ada-sumber |
	nomor_identitas_pasangan         | Nomor Identitas Pasangan         | SpouseIDCard         | belum-ada-sumber |
	nama_pasangan                    | Nama Pasangan                    | SpouseName           | belum-ada-sumber |
	tanggal_lahir_pasangan           | Tanggal Lahir Pasangan           | SpouseDateOfBirth    | belum-ada-sumber |
	// Salah ketik 'AssetAggreement' disalin apa adanya; itu nama propertinya di Pega.
	perjanjian_pisah_harta           | Perjanjian Pisah Harta           | AssetAggreement      | belum-ada-sumber |
	// Judul di GRID lebih panjang — "Melanggar BMPK/BMPD/BMPP" — sedangkan judul di BERKAS
	// lebih pendek. Yang dipakai di sini judul berkas, karena katalog ini melayani
	// keduanya; selisih dua judul itu dicatat, tidak dihilangkan.
	melanggar_bmpk                   | Melanggar BMPK                   | ViolatedBMPK         | belum-ada-sumber |
	melampaui_bmpk                   | Melampaui BMPK                   | ExceededBMPK         | belum-ada-sumber |
	nama_ibu_kandung                 | Nama Ibu Kandung                 | ASMMotherName        | belum-ada-sumber |
	// Kedua kolom terakhir terisi, tetapi keduanya membawa cacat yang ikut dicatat di
	// berkas .sql: kode kantor cabang adalah konstanta ''001'' di dalam kueri lama
	// (melanggar 'D-15'), dan operasi data diturunkan dari pencocokan teks ''%EDM%''.
	kode_kantor_cabang               | Kode Kantor Cabang               | KodeKantorCabang     | tersedia         |
	operasi_data                     | Operasi Data                     | OperasiData          | tersedia         |
`)

// f06ExportSlots adalah **34 kolom data** berkas ekspor segmen F06.
//
// Urutannya disalin APA ADANYA dari `CSVProperties` pada `ExportDataSlinkFOG-Act.xml`,
// satu slot per nama properti — termasuk slot keenam yang namanya adalah DUA nama
// tersambung tanpa koma. Lihat ExportSlots untuk akibatnya terhadap bentuk berkasnya.
//
// `Header` di sini adalah nama PROPERTINYA, bukan judul kolom: judulnya datang dari
// ExportHeaders yang jumlahnya berbeda, dan memasangkan keduanya di satu struct justru
// akan menyembunyikan pergeseran yang sedang direplikasi.
//
// `Key` terisi hanya pada slot yang propertinya benar-benar dihasilkan
// `GetDataSlinkAllFOG`. Slot berkunci kosong selalu menghasilkan sel kosong — dan itulah
// yang terjadi pada 31 dari 34 slot.
var f06ExportSlots = tabletext.Rows[Column](`
	Header                  | LegacyProperty          | Source           | Key
	// Delapan slot pertama: properti CIF yang tidak dihasilkan kueri grid.
	ASMClientID             | ASMClientID             | belum-ada-sumber |
	ASMIDType               | ASMIDType               | belum-ada-sumber |
	ASMNIK                  | ASMNIK                  | belum-ada-sumber |
	ASMIDCard               | ASMIDCard               | belum-ada-sumber |
	pyFullName              | pyFullName              | belum-ada-sumber |
	// Slot keenam: DUA nama properti yang tersambung tanpa koma di sumbernya. Ia tidak
	// pernah ada sebagai properti, sehingga selalu kosong — dan karena satu entri
	// memakan jatah dua, seluruh slot sesudahnya bergeser terhadap judulnya.
	ASMGenderASMDateOfBirth | ASMGenderASMDateOfBirth | belum-ada-sumber |
	// Slot ketujuh — TERISI. Nilainya tercetak di bawah judul "Jenis Kelamin".
	ASMAddress              | ASMAddress              | tersedia         | alamat
	ASMRWNote               | ASMRWNote               | belum-ada-sumber |
	ASMDistrict             | ASMDistrict             | belum-ada-sumber |
	ASMCity                 | ASMCity                 | belum-ada-sumber |
	// 'PostalCode' dan 'PhoneNo' TIDAK berpadanan dengan alias kueri ('ASMZIPCODE',
	// 'TELFAXNUMBER'), sehingga keduanya kosong di berkas meski nilainya ADA di grid.
	PostalCode              | PostalCode              | belum-ada-sumber |
	PhoneNo                 | PhoneNo                 | belum-ada-sumber |
	ASMMobilePhoneList      | ASMMobilePhoneList      | belum-ada-sumber |
	EmailAddress            | EmailAddress            | belum-ada-sumber |
	ASMNationality          | ASMNationality          | belum-ada-sumber |
	ASMJobDesc              | ASMJobDesc              | belum-ada-sumber |
	ASMNotes                | ASMNotes                | belum-ada-sumber |
	ASMBusinessField        | ASMBusinessField        | belum-ada-sumber |
	OrganizationAddress     | OrganizationAddress     | belum-ada-sumber |
	YearlyGrossIncome       | YearlyGrossIncome       | belum-ada-sumber |
	IncomeSource            | IncomeSource            | belum-ada-sumber |
	Tanggungan              | Tanggungan              | belum-ada-sumber |
	ReporterRelationship    | ReporterRelationship    | belum-ada-sumber |
	ReporterGroup           | ReporterGroup           | belum-ada-sumber |
	ASMMaritalStatus        | ASMMaritalStatus        | belum-ada-sumber |
	SpouseIDCard            | SpouseIDCard            | belum-ada-sumber |
	SpouseName              | SpouseName              | belum-ada-sumber |
	SpouseDateOfBirth       | SpouseDateOfBirth       | belum-ada-sumber |
	AssetAggreement         | AssetAggreement         | belum-ada-sumber |
	ViolatedBMPK            | ViolatedBMPK            | belum-ada-sumber |
	ExceededBMPK            | ExceededBMPK            | belum-ada-sumber |
	ASMMotherName           | ASMMotherName           | belum-ada-sumber |
	// Dua slot terakhir — TERISI. Nilainya tercetak di bawah judul "Perjanjian Pisah
	// Harta" dan "Melanggar BMPK".
	KodeKantorCabang        | KodeKantorCabang        | tersedia         | kode_kantor_cabang
	OperasiData             | OperasiData             | tersedia         | operasi_data
`)

// ============================================================================
// FORMAT FILE — berkas contoh unggahan
// ============================================================================

// TemplateFileName adalah nama berkas tombol "Format File".
//
// Disalin apa adanya dari `<FileName>` pada
// `Activity/DownloadFileCSVFormatSlikOJK-Act.xml`. Berbeda dari kedua nama berkas ekspor,
// nama INI tidak tertukar — jadi ia tidak dibetulkan.
const TemplateFileName = "Format Auto Klaim Slik OJK ASM"

// TemplateColumns adalah judul kolom berkas contoh unggahan — **25 kolom**.
//
// Disalin APA ADANYA dari `CSVPropHeaders` pada
// `Activity/DownloadFileCSVFormatSlikOJK-Act.xml`. Di aktivitas itu `CSVPropHeaders` dan
// `CSVProperties` **jumlahnya sama** dan isinya identik — berbeda dari `ExportDataSlinkFOG`
// yang misalign. Berkas contohnya karena itu tidak punya cacat yang perlu dibetulkan.
//
// Enam kolom pertama adalah yang diisi pengunggah; sembilan belas sisanya kolom segmen D01
// yang ikut terbawa dan dibiarkan kosong. Keduanya tetap dibawa utuh — lihat TemplateSample.
var TemplateColumns = []string{
	// Enam kolom yang benar-benar diisi pengunggah.
	"PolicyNo",
	"ContractNo",
	"ClaimAmount",
	"ReportType",
	"FlagData",
	"TanggalBayarKlaim",

	// Sembilan belas kolom segmen D01 yang ikut terbawa, seluruhnya kosong pada baris
	// contoh — dan memang begitu di berkas aslinya.
	"NOMORREKENINGFASILITAS",
	"NOMORCIFDEBITUR",
	"KodeJenisFasilitas",
	"SumberDana",
	"AwalPolis",
	"AkhirPolis",
	"SukuBunga",
	"KODEVALUTA",
	"NilaiMataUangAsal",
	"KodeKolektibilitas",
	"TanggalMacet",
	"KodeSebabMacet",
	"TUNGGAKAN",
	"JUMLAHKEWAJIBAN",
	"TanggalKondisi",
	"KodeKondisi",
	"Keterangan",
	"KodeKantorCabang",
	"OperasiData",
}

// TemplateSample adalah satu baris contoh di dalam berkas format — **25 sel**, enam terisi.
//
// Keenam nilainya disalin APA ADANYA dari langkah `Property-Set` pada aktivitas yang sama,
// termasuk keterangan yang ditulis di dalam sel `FlagData` — memang begitu bentuknya di
// berkas aslinya.
//
// # Kenapa sembilan belas sel terakhir KOSONG, bukan dibuang
//
// Aktivitas lama menyusun barisnya lewat `TempsFormaater.pxResults(<LAST>)` dan hanya
// mengisi keenam properti pertama; sembilan belas sisanya tidak pernah di-set. Karena
// jumlah judul dan jumlah properti di sana SAMA, hasilnya adalah baris yang sejajar
// dengan sembilan belas sel kosong — bukan baris yang bergeser.
//
// Versi pertama modul ini MEMOTONGNYA menjadi enam kolom, atas bacaan yang keliru bahwa
// berkasnya tidak sejajar. Itu salah: yang misalign adalah `ExportDataSlinkFOG`, bukan
// berkas format ini. Dikembalikan utuh — pengunggah membandingkan berkasnya dengan
// contoh ini kolom per kolom, dan contoh yang lebih pendek membuat berkas yang benar
// terlihat salah.
//
// Data contohnya karangan Pega, bukan data nasabah: nomor polis dan nomor kontrak di
// bawah adalah contoh yang memang tertulis di dalam rule.
var TemplateSample = []string{
	"12000000000026",
	"093B/SMMF-PAP/XI/2020",
	"22500000",
	"KLAIM",
	"1 (Diisi Jika Pembayaran Ke Investment)",
	"09/12/2024",

	// Sembilan belas sel kosong, sesuai berkas aslinya.
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "",
}

// AvailableColumns mengembalikan kolom yang benar-benar terisi pada satu segmen.
//
// Dipakai uji — bukan untuk menyembunyikan kolom di layar. Di layar seluruh 38 kolom
// tetap tampil, sesuai keputusan Work Owner 2026-09-26.
func AvailableColumns(segment Segment) []Column {
	all := Columns(segment)
	out := make([]Column, 0, len(all))
	for _, c := range all {
		if c.Source == SourceAvailable {
			out = append(out, c)
		}
	}
	return out
}
