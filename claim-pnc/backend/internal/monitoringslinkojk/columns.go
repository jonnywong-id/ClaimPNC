package monitoringslinkojk

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
// ============================================================================
// PEMETAAN SEGMEN — TERTUKAR TERHADAP NAMA BERKAS DI EXPORT
// ============================================================================
//
// Layar Pega PRODUKSI (`clouduniapp`, tangkapan layar Work Owner 2026-10-09):
//
//	Segment Slik D01 -> 38 kolom IDENTITAS DEBITUR (Nomor CIF Debitur … Operasi Data)
//	Segment Slik F06 -> 20 kolom FASILITAS KREDIT  (No Klaim … Keterangan)
//
// Itu juga struktur pelaporan SLIK OJK yang sebenarnya: **D01 debitur, F06 fasilitas**.
//
// Berkas di export menamainya terbalik — ke-38 judul itu ada di `ExportDataSlinkFOG`
// (menyebut F06), dan ke-27 judul fasilitas ada di `ExportDataSlinkD01`. Penamaan terbalik
// yang sudah tercatat untuk rule SQL (`GetDataSlinkAllFOGF06` melayani D01) ternyata
// berlaku juga untuk section dan aktivitas ekspor.
//
// Nama variabel di bawah MENGIKUTI NAMA BERKAS EXPORT, bukan nama segmen layarnya —
// supaya tiap variabel dapat dibaca berdampingan dengan berkas asalnya. Pemetaan ke segmen
// layar terjadi di sini, satu tempat.
//
// Catatan atas `pegadev`: di sana segmen D01 menampilkan kolom FASILITAS, bukan debitur.
// Work Owner menetapkan yang berlaku adalah produksi (2026-10-09).
func Columns(segment Segment) []Column {
	switch segment {
	case SegmentD01:
		// Identitas debitur — daftar milik `ExportDataSlinkFOG`.
		return append([]Column(nil), f06ExportColumns...)
	case SegmentF06:
		// Fasilitas kredit — daftar milik `ExportDataSlinkD01`.
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
		// 38 judul identitas debitur. Lihat catatan pemetaan pada Columns.
		return Headers(f06ExportColumns)
	case SegmentF06:
		// 27 judul fasilitas kredit.
		return Headers(d01ExportColumns)
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
		// 34 slot — SENGAJA tidak sejajar dengan 38 judulnya. Lihat catatan di atas.
		return append([]Column(nil), f06ExportSlots...)
	case SegmentF06:
		// 27 slot, sejajar dengan 27 judulnya.
		return append([]Column(nil), d01ExportColumns...)
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
// # KOREKSI 2026-10-09 — namanya ternyata TIDAK tertukar
//
// Catatan di atas menyimpulkan kedua nama berkas tertukar. Itu akibat pemetaan segmen saya
// sendiri yang keliru, bukan cacat di Pega.
//
// Dengan pemetaan yang benar — D01 memakai `ExportDataSlinkFOG`, F06 memakai
// `ExportDataSlinkD01` — nama berkasnya justru **cocok dengan segmennya**:
//
//	segmen D01 -> ExportDataSlinkFOG -> "Laporan SLIK OJK D01"
//	segmen F06 -> ExportDataSlinkD01 -> "Laporan F06 SLIK OJK"
//
// Kecocokan itu sendiri menjadi bukti tambahan bahwa pemetaan barunya benar: dua
// ketertukaran yang saling meniadakan jauh lebih mungkin berarti satu kesalahan baca
// daripada dua cacat terpisah di sistem lama.
func FileName(segment Segment) string {
	switch segment {
	case SegmentD01:
		return "Laporan SLIK OJK D01"
	case SegmentF06:
		return "Laporan F06 SLIK OJK"
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
// # Koreksi kedua, 2026-10-08 — keduanya TIDAK sepenuhnya identik
//
// Sempat ditulis `f06Columns = d01Columns`. Tangkapan layar kedua membuktikan ada satu
// selisih: grid **D01 punya "Nama Debitur"** sebagai kolom ketiga, grid **F06 tidak**.
//
// Jadi F06 = D01 **tanpa** kolom itu. Ia diturunkan dari d01Columns alih-alih disalin,
// supaya kesembilan belas kolom yang memang sama tidak dapat menyimpang diam-diam.
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
var d01Columns = []Column{
	{Key: "no_klaim", Header: "No Klaim", LegacyProperty: "ClaimID", Source: SourceAvailable},
	{Key: "contract_no", Header: "Contract No", LegacyProperty: "ContractNo", Source: SourceAvailable},

	// Tidak ada "Nama Debitur" di sini.
	//
	// Kolom itu sempat ditambahkan karena tampil di layar `pegadev`. Layar PRODUKSI
	// (`clouduniapp`) tidak memilikinya — pada kedua segmen. Work Owner menetapkan
	// produksi yang berlaku (2026-10-09), jadi kolomnya dicabut.
	//
	// Pemeriksaan ke Oracle juga menunjukkan `OBJECTNAME` kosong pada seluruh baris
	// laporan yang cocok, sehingga kolomnya tidak akan pernah berisi apa pun.
	{Key: "nomor_rekening_fasilitas", Header: "Nomor Rekening Fasilitas", LegacyProperty: "NOMORREKENINGFASILITAS", Source: SourceAvailable},
	{Key: "no_cif_debitur", Header: "No CIF Debitur", LegacyProperty: "NOMORCIFDEBITUR", Source: SourceAvailable},
	{Key: "kode_jenis_fasilitas", Header: "Kode Jenis Fasilitas", LegacyProperty: "KodeJenisFasilitas", Source: SourceAvailable},
	{Key: "sumber_dana", Header: "Sumber Dana", LegacyProperty: "SumberDana", Source: SourceAvailable},
	{Key: "start_polis", Header: "Start Polis", LegacyProperty: "AwalPolis", Source: SourceAvailable},
	{Key: "end_polis", Header: "End Polis", LegacyProperty: "AkhirPolis", Source: SourceAvailable},
	{Key: "suku_bunga", Header: "Suku Bunga", LegacyProperty: "SukuBunga", Source: SourceAvailable, Numeric: true},
	{Key: "kode_valuta", Header: "Kode Valuta", LegacyProperty: "KODEVALUTA", Source: SourceAvailable},

	// Lihat catatan di atas: terisi dari `tanggalkondisi`, bukan dari tanggal pembayaran.
	//
	// Urutannya — SEBELUM "Nilai Mata Uang Asal" — diambil dari tangkapan layar Pega yang
	// berjalan, bukan dari urutan SELECT. Keduanya memang berbeda: di `GetDataSlinkAllFOGF06`
	// alias ini berada tepat sesudah `kodejenisfasilitas`, sedangkan di layar ia berada
	// sesudah "Kode Valuta". Yang menentukan tata letak grid adalah section, bukan kueri.
	{Key: "tanggal_pembayaran", Header: "Tanggal Pembayaran", LegacyProperty: "TanggalPembayaran", Source: SourceAvailable},

	{Key: "nilai_mata_uang_asal", Header: "Nilai Mata Uang Asal", LegacyProperty: "NilaiMataUangAsal", Source: SourceAvailable},

	// Judulnya salah ketik di section lama ("Kelektibilitas"), dan kolom tabelnya salah
	// ketik pula ("KODEKOLEKTABILITAS"). Keduanya dibiarkan apa adanya.
	{Key: "kode_kolektibilitas", Header: "Kode Kelektibilitas", LegacyProperty: "KodeKolektibilitas", Source: SourceAvailable},

	// Judulnya menyebut sendiri isinya: tanggal REGISTRASI klaim, bukan tanggal macet.
	// `GetDataSlinkAllFOG-SQL.xml` mengisi kolom padanannya dari `pnc.REGISTERDATE`.
	{Key: "tanggal_macet", Header: "Tanggal Macet (Regist Date)", LegacyProperty: "TanggalMacet", Source: SourceAvailable},

	{Key: "kode_sebab_macet", Header: "Kode Sebab Macet", LegacyProperty: "KodeSebabMacet", Source: SourceAvailable},
	{Key: "tunggakan", Header: "Tunggakan", LegacyProperty: "TUNGGAKAN", Source: SourceAvailable, Numeric: true},
	{Key: "jumlah_kewajiban", Header: "Nominal / Jumlah Kewajiban", LegacyProperty: "JUMLAHKEWAJIBAN", Source: SourceAvailable, Numeric: true},
	{Key: "tanggal_kondisi", Header: "Tanggal Kondisi", LegacyProperty: "TanggalKondisi", Source: SourceAvailable},
	{Key: "kode_kondisi", Header: "Kode Kondisi", LegacyProperty: "KodeKondisi", Source: SourceAvailable},
	{Key: "keterangan", Header: "Keterangan", LegacyProperty: "Keterangan", Source: SourceAvailable},
}

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
var d01ExportColumns = []Column{
	{Key: "no_klaim", Header: "ClaimID", LegacyProperty: "ClaimID", Source: SourceAvailable},
	{Key: "contract_no", Header: "ContractNo", LegacyProperty: "ContractNo", Source: SourceAvailable},
	{Key: "nama_objek", Header: "ObjectName", LegacyProperty: "ObjectName", Source: SourceMissing},
	{Key: "nomor_rekening_fasilitas", Header: "NOMORREKENINGFASILITAS", LegacyProperty: "NOMORREKENINGFASILITAS", Source: SourceAvailable},
	{Key: "no_cif_debitur", Header: "NOMORCIFDEBITUR", LegacyProperty: "NOMORCIFDEBITUR", Source: SourceAvailable},
	{Key: "kode_jenis_fasilitas", Header: "KodeJenisFasilitas", LegacyProperty: "KodeJenisFasilitas", Source: SourceAvailable},
	{Key: "sumber_dana", Header: "SumberDana", LegacyProperty: "SumberDana", Source: SourceAvailable},
	{Key: "start_polis", Header: "AwalPolis", LegacyProperty: "AwalPolis", Source: SourceAvailable},
	{Key: "end_polis", Header: "AkhirPolis", LegacyProperty: "AkhirPolis", Source: SourceAvailable},
	{Key: "suku_bunga", Header: "SukuBunga", LegacyProperty: "SukuBunga", Source: SourceAvailable, Numeric: true},
	{Key: "kode_valuta", Header: "KODEVALUTA", LegacyProperty: "KODEVALUTA", Source: SourceAvailable},
	{Key: "tanggal_pembayaran", Header: "TanggalPembayaran", LegacyProperty: "TanggalPembayaran", Source: SourceAvailable},
	{Key: "nilai_mata_uang_asal", Header: "NilaiMataUangAsal", LegacyProperty: "NilaiMataUangAsal", Source: SourceAvailable},
	{Key: "kode_kolektibilitas", Header: "KodeKolektibilitas", LegacyProperty: "KodeKolektibilitas", Source: SourceAvailable},
	{Key: "tanggal_macet", Header: "TanggalMacet", LegacyProperty: "TanggalMacet", Source: SourceAvailable},
	{Key: "kode_sebab_macet", Header: "KodeSebabMacet", LegacyProperty: "KodeSebabMacet", Source: SourceAvailable},
	{Key: "tunggakan", Header: "TUNGGAKAN", LegacyProperty: "TUNGGAKAN", Source: SourceAvailable, Numeric: true},
	{Key: "jumlah_kewajiban", Header: "JUMLAHKEWAJIBAN", LegacyProperty: "JUMLAHKEWAJIBAN", Source: SourceAvailable, Numeric: true},
	{Key: "tanggal_kondisi", Header: "TanggalKondisi", LegacyProperty: "TanggalKondisi", Source: SourceAvailable},
	{Key: "kode_kondisi", Header: "KodeKondisi", LegacyProperty: "KodeKondisi", Source: SourceAvailable},
	{Key: "keterangan", Header: "Keterangan", LegacyProperty: "Keterangan", Source: SourceAvailable},
	{Key: "kode_kantor_cabang", Header: "KodeKantorCabang", LegacyProperty: "KodeKantorCabang", Source: SourceAvailable},
	{Key: "operasi_data", Header: "OperasiData", LegacyProperty: "OperasiData", Source: SourceAvailable},

	// Kolom 24 — `Keterangan` untuk kedua kalinya. Ada di sumbernya; lihat catatan di atas.
	{Key: "keterangan", Header: "Keterangan", LegacyProperty: "Keterangan", Source: SourceAvailable},

	{Key: "no_ktp", Header: "NoKTP", LegacyProperty: "NoKTP", Source: SourceAvailable},
	{Key: "npwp_perusahaan", Header: "NPWPPerusahaan", LegacyProperty: "NPWPPerusahaan", Source: SourceAvailable},
	{Key: "no_polis", Header: "PolicyNo", LegacyProperty: "PolicyNo", Source: SourceAvailable},
}

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
var f06ExportColumns = []Column{
	{Key: "nomor_cif_debitur", Header: "Nomor CIF Debitur", LegacyProperty: "ASMClientID", Source: SourceAvailable},
	{Key: "jenis_identitas", Header: "Jenis Identitas", LegacyProperty: "ASMIDType", Source: SourceMissing},
	{Key: "nomor_identitas", Header: "Nomor Identitas", LegacyProperty: "ASMNIK", Source: SourceMissing},
	{Key: "nama_sesuai_identitas", Header: "Nama Sesuai Identitas", LegacyProperty: "ASMIDCard", Source: SourceMissing},
	{Key: "nama_lengkap", Header: "Nama Lengkap", LegacyProperty: "pyFullName", Source: SourceMissing},

	// Judul tanpa properti di export; lihat catatan misalign di atas.
	{Key: "kode_status_pendidikan", Header: "KodeStatusPendidikan", Source: SourceMissing},

	{Key: "jenis_kelamin", Header: "Jenis Kelamin", LegacyProperty: "ASMGender", Source: SourceAvailable},

	// Judul tanpa properti di export.
	{Key: "tempat_lahir", Header: "Tempat Lahir", Source: SourceMissing},

	{Key: "tanggal_lahir", Header: "Tanggal Lahir", LegacyProperty: "ASMDateOfBirth", Source: SourceAvailable},

	// Judul tanpa properti di export. Jangan tertukar dengan `NPWPPerusahaan` pada ekspor
	// D01 — yang ini NPWP DEBITUR.
	{Key: "npwp", Header: "NPWP", Source: SourceMissing},

	{Key: "alamat", Header: "Alamat", LegacyProperty: "ASMAddress", Source: SourceAvailable},

	// Alias menyesatkan: "Kelurahan" terikat properti catatan RW.
	{Key: "kelurahan", Header: "Kelurahan", LegacyProperty: "ASMRWNote", Source: SourceMissing},

	{Key: "kecamatan", Header: "Kecamatan", LegacyProperty: "ASMDistrict", Source: SourceMissing},
	{Key: "kode_kab_kota", Header: "Kode Kab/Kota", LegacyProperty: "ASMCity", Source: SourceMissing},
	{Key: "kode_pos", Header: "Kode Pos", LegacyProperty: "PostalCode", Source: SourceAvailable},
	{Key: "telepon", Header: "Telepon", LegacyProperty: "PhoneNo", Source: SourceAvailable},
	{Key: "nomor_telepon_seluler", Header: "Nomor Telepon Seluler", LegacyProperty: "ASMMobilePhoneList", Source: SourceMissing},
	{Key: "alamat_email", Header: "Alamat Email", LegacyProperty: "EmailAddress", Source: SourceMissing},

	// Alias menyesatkan: "Kode Negara Domisili" terikat properti kewarganegaraan.
	{Key: "kode_negara_domisili", Header: "Kode Negara Domisili", LegacyProperty: "ASMNationality", Source: SourceMissing},

	{Key: "kode_pekerjaan", Header: "Kode Pekerjaan", LegacyProperty: "ASMJobDesc", Source: SourceMissing},

	// Alias paling menyesatkan di segmen ini: "Tempat Bekerja" terikat `ASMNotes`.
	{Key: "tempat_bekerja", Header: "Tempat Bekerja", LegacyProperty: "ASMNotes", Source: SourceMissing},

	{Key: "kode_bidang_usaha_tempat_bekerja", Header: "Kode Bidang Usaha Tempat Bekerja", LegacyProperty: "ASMBusinessField", Source: SourceMissing},
	{Key: "alamat_tempat_bekerja", Header: "Alamat Tempat Bekerja", LegacyProperty: "OrganizationAddress", Source: SourceMissing},
	{Key: "penghasilan_kotor_per_tahun", Header: "Penghasilan Kotor Per Tahun", LegacyProperty: "YearlyGrossIncome", Source: SourceMissing, Numeric: true},
	{Key: "kode_sumber_penghasilan", Header: "Kode Sumber Penghasilan", LegacyProperty: "IncomeSource", Source: SourceMissing},
	{Key: "jumlah_tanggungan", Header: "Jumlah Tanggungan", LegacyProperty: "Tanggungan", Source: SourceMissing, Numeric: true},
	{Key: "kode_hubungan_dengan_pelapor", Header: "Kode Hubungan Dengan Pelapor", LegacyProperty: "ReporterRelationship", Source: SourceMissing},
	{Key: "kode_golongan_debitur", Header: "Kode Golongan Debitur", LegacyProperty: "ReporterGroup", Source: SourceMissing},
	{Key: "status_perkawinan_debitur", Header: "Status Perkawinan Debitur", LegacyProperty: "ASMMaritalStatus", Source: SourceMissing},
	{Key: "nomor_identitas_pasangan", Header: "Nomor Identitas Pasangan", LegacyProperty: "SpouseIDCard", Source: SourceMissing},
	{Key: "nama_pasangan", Header: "Nama Pasangan", LegacyProperty: "SpouseName", Source: SourceMissing},
	{Key: "tanggal_lahir_pasangan", Header: "Tanggal Lahir Pasangan", LegacyProperty: "SpouseDateOfBirth", Source: SourceMissing},

	// Salah ketik `AssetAggreement` disalin apa adanya; itu nama propertinya di Pega.
	{Key: "perjanjian_pisah_harta", Header: "Perjanjian Pisah Harta", LegacyProperty: "AssetAggreement", Source: SourceMissing},

	// Judul di GRID lebih panjang — "Melanggar BMPK/BMPD/BMPP" — sedangkan judul di BERKAS
	// lebih pendek. Yang dipakai di sini judul berkas, karena katalog ini melayani
	// keduanya; selisih dua judul itu dicatat, tidak dihilangkan.
	{Key: "melanggar_bmpk", Header: "Melanggar BMPK", LegacyProperty: "ViolatedBMPK", Source: SourceMissing},
	{Key: "melampaui_bmpk", Header: "Melampaui BMPK", LegacyProperty: "ExceededBMPK", Source: SourceMissing},

	{Key: "nama_ibu_kandung", Header: "Nama Ibu Kandung", LegacyProperty: "ASMMotherName", Source: SourceMissing},

	// Kedua kolom terakhir terisi, tetapi keduanya membawa cacat yang ikut dicatat di
	// berkas .sql: kode kantor cabang adalah konstanta `'001'` di dalam kueri lama
	// (melanggar `D-15`), dan operasi data diturunkan dari pencocokan teks `'%EDM%'`.
	{Key: "kode_kantor_cabang", Header: "Kode Kantor Cabang", LegacyProperty: "KodeKantorCabang", Source: SourceAvailable},
	{Key: "operasi_data", Header: "Operasi Data", LegacyProperty: "OperasiData", Source: SourceAvailable},
}

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
var f06ExportSlots = []Column{
	// Delapan slot pertama: properti CIF yang tidak dihasilkan kueri grid.
	{Header: "ASMClientID", LegacyProperty: "ASMClientID", Source: SourceMissing},
	{Header: "ASMIDType", LegacyProperty: "ASMIDType", Source: SourceMissing},
	{Header: "ASMNIK", LegacyProperty: "ASMNIK", Source: SourceMissing},
	{Header: "ASMIDCard", LegacyProperty: "ASMIDCard", Source: SourceMissing},
	{Header: "pyFullName", LegacyProperty: "pyFullName", Source: SourceMissing},

	// Slot keenam: DUA nama properti yang tersambung tanpa koma di sumbernya. Ia tidak
	// pernah ada sebagai properti, sehingga selalu kosong — dan karena satu entri
	// memakan jatah dua, seluruh slot sesudahnya bergeser terhadap judulnya.
	{Header: "ASMGenderASMDateOfBirth", LegacyProperty: "ASMGenderASMDateOfBirth", Source: SourceMissing},

	// Slot ketujuh — TERISI. Nilainya tercetak di bawah judul "Jenis Kelamin".
	{Key: "alamat", Header: "ASMAddress", LegacyProperty: "ASMAddress", Source: SourceAvailable},

	{Header: "ASMRWNote", LegacyProperty: "ASMRWNote", Source: SourceMissing},
	{Header: "ASMDistrict", LegacyProperty: "ASMDistrict", Source: SourceMissing},
	{Header: "ASMCity", LegacyProperty: "ASMCity", Source: SourceMissing},

	// `PostalCode` dan `PhoneNo` TIDAK berpadanan dengan alias kueri (`ASMZIPCODE`,
	// `TELFAXNUMBER`), sehingga keduanya kosong di berkas meski nilainya ADA di grid.
	{Header: "PostalCode", LegacyProperty: "PostalCode", Source: SourceMissing},
	{Header: "PhoneNo", LegacyProperty: "PhoneNo", Source: SourceMissing},

	{Header: "ASMMobilePhoneList", LegacyProperty: "ASMMobilePhoneList", Source: SourceMissing},
	{Header: "EmailAddress", LegacyProperty: "EmailAddress", Source: SourceMissing},
	{Header: "ASMNationality", LegacyProperty: "ASMNationality", Source: SourceMissing},
	{Header: "ASMJobDesc", LegacyProperty: "ASMJobDesc", Source: SourceMissing},
	{Header: "ASMNotes", LegacyProperty: "ASMNotes", Source: SourceMissing},
	{Header: "ASMBusinessField", LegacyProperty: "ASMBusinessField", Source: SourceMissing},
	{Header: "OrganizationAddress", LegacyProperty: "OrganizationAddress", Source: SourceMissing},
	{Header: "YearlyGrossIncome", LegacyProperty: "YearlyGrossIncome", Source: SourceMissing},
	{Header: "IncomeSource", LegacyProperty: "IncomeSource", Source: SourceMissing},
	{Header: "Tanggungan", LegacyProperty: "Tanggungan", Source: SourceMissing},
	{Header: "ReporterRelationship", LegacyProperty: "ReporterRelationship", Source: SourceMissing},
	{Header: "ReporterGroup", LegacyProperty: "ReporterGroup", Source: SourceMissing},
	{Header: "ASMMaritalStatus", LegacyProperty: "ASMMaritalStatus", Source: SourceMissing},
	{Header: "SpouseIDCard", LegacyProperty: "SpouseIDCard", Source: SourceMissing},
	{Header: "SpouseName", LegacyProperty: "SpouseName", Source: SourceMissing},
	{Header: "SpouseDateOfBirth", LegacyProperty: "SpouseDateOfBirth", Source: SourceMissing},
	{Header: "AssetAggreement", LegacyProperty: "AssetAggreement", Source: SourceMissing},
	{Header: "ViolatedBMPK", LegacyProperty: "ViolatedBMPK", Source: SourceMissing},
	{Header: "ExceededBMPK", LegacyProperty: "ExceededBMPK", Source: SourceMissing},
	{Header: "ASMMotherName", LegacyProperty: "ASMMotherName", Source: SourceMissing},

	// Dua slot terakhir — TERISI. Nilainya tercetak di bawah judul "Perjanjian Pisah
	// Harta" dan "Melanggar BMPK".
	{Key: "kode_kantor_cabang", Header: "KodeKantorCabang", LegacyProperty: "KodeKantorCabang", Source: SourceAvailable},
	{Key: "operasi_data", Header: "OperasiData", LegacyProperty: "OperasiData", Source: SourceAvailable},
}

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
