package inboxosclaimpercabang

import (
	"time"

	"claim-pnc/internal/platform/money"
)

// Berkas ini memuat isi popup "Detail" — layar kedua modul ini.
//
// Asalnya `Harness/View_DetailKlaimCabang_Harness-Harness.xml` beserta
// `Section/DetailKlaimCabang_Sect-Section.xml`, yang dibuka tombol Detail pada grid dan diisi
// `Activity/ViewStatusProgressCabang_act-Act.xml` (15 langkah).
//
// # Lima nilai datang dari BARIS GRID, bukan dibaca ulang
//
// Tombol Detail mengirim lima parameter dari baris yang diklik:
//
//	Inskey   = .ClaimNo               nomor klaim
//	reserve  = .EstimationValue       Total Reserve
//	aging    = .AgingKlaim            Aging
//	cob      = .BusinessName          menentukan varian kolom grid objek
//	notepic  = .Keterangan            Note dari PIC
//
// Di sistem baru keempat nilainya **dibaca ulang dari penyimpanan**, bukan diterima dari
// klien. Menerimanya dari klien berarti nilai uang dan umur klaim yang tampil di popup
// ditentukan pengirim permintaan — dan keduanya dapat diubah begitu saja lewat alat biasa.
// Hasilnya sama selama datanya tidak berubah di antara dua permintaan, dan bila berubah,
// yang benar adalah yang dibaca ulang.
//
// # Popup ini WAJIB disaring cabang, meski layar lamanya tidak
//
// Di Pega, popup hanya dapat dibuka dari baris yang sudah tampil, sehingga penyaringan
// cabangnya terjadi dengan sendirinya. Endpoint HTTP tidak punya pembatas itu: siapa pun yang
// tahu sebuah nomor klaim dapat memintanya. Karena itu pembacaan detail menempuh penyaring
// cabang yang SAMA dengan daftarnya — tanpa itu, popup menjadi jalan memutar yang membocorkan
// data antarbadan hukum (`R-20`).

// Detail adalah seluruh isi popup untuk satu klaim.
type Detail struct {
	// ClaimNumber adalah nomor klaim yang diminta — `T_CLAIM_PNC.CLAIMNO`.
	ClaimNumber string

	// ClaimKey adalah kunci internal klaim — `T_CLAIM_PNC.CLAIMID`.
	//
	// Ia tidak digambar, tetapi ia kunci ketiga kueri anak: objek, riwayat progres, dan
	// komunikasi adjuster. Nomor klaim TIDAK dapat dipakai untuk itu — ia tidak unik
	// (1.515 nilai berbeda untuk 1.922 baris), sehingga memakainya akan menggabungkan isi
	// dua klaim yang bernomor sama.
	ClaimKey string

	// BusinessName adalah lini bisnis klaim — menentukan varian kolom grid objek.
	//
	// Di Pega ia datang sebagai parameter `cob` dari baris grid. Di sini dibaca ulang.
	BusinessName string

	// Occupation adalah nilai berlabel "Occupation :".
	//
	// Di Pega ia ditentukan TIGA jalur berurutan, yang belakangan menimpa yang sebelumnya:
	//
	//	langkah 6   ObjectList(idxobj).OccupationName
	//	langkah 8   ObjectList(idxobj).CoverageList(1).DeductibleList(1).OccupationName
	//	langkah 10  hasil GetOccupationName atas ObjectList(idxobj).OccupationID
	//
	// Jalur kedua tidak dapat dibawa: daftar deductible tidak punya tabel yang terbaca dari
	// export. Yang dibawa jalur pertama dan ketiga — nama yang tersimpan pada objek, dan bila
	// kosong barulah dicari lewat kodenya ke tabel `OCCUPATION`.
	Occupation string

	// TotalSumInsured adalah nilai berlabel "Total Sum Insured :".
	//
	// # Ia nilai SATU objek, bukan jumlah seluruh objek — dan itu dibawa apa adanya
	//
	// Nama labelnya menyesatkan. Di Pega, `local.idxobj` disetel di dalam perulangan atas
	// `ObjectList` (langkah 4) sehingga nilainya berakhir pada objek TERAKHIR, dan
	// `TempDetail.TSI` diisi dari objek itu saja. Popup karena itu menampilkan TSI objek
	// terakhir, bukan total.
	//
	// Perilakunya direplikasi (`P-5`) dan dinyatakan lewat DetailPlannedDifferences, bukan
	// dibetulkan diam-diam: membetulkannya mengubah angka yang selama ini dibaca petugas.
	TotalSumInsured money.Money

	// Chronology adalah nilai berlabel "Kronologi :" — `T_CLAIM_PNC.REPORTDESCRIPTION`.
	Chronology string

	// EstimationValue adalah nilai berlabel "Total Reserve :".
	//
	// Dihitung sama persis dengan kolom grid — jumlah estimasi apa adanya, tanpa kurs dan
	// tanpa porsi ASM. Bila dihitung dengan cara ekspor, angka popup akan berbeda dari angka
	// baris yang membukanya, dan pengguna tidak punya cara menjelaskan selisihnya.
	EstimationValue money.Money

	// AgingDays adalah nilai berlabel "Aging :", dalam hari kalender WIB.
	//
	// Diisi pemanggil dari seam Clock, sama seperti pada WorkItem — bukan dihitung di kueri.
	// Alasannya di AgingDaysSince.
	AgingDays int

	// RegisterDate dipakai menghitung AgingDays. Ia tidak digambar.
	RegisterDate *time.Time

	// DominantFactors adalah nilai berlabel "Dominant Factor :", sudah dirangkai.
	//
	// Bentuknya mengikuti kueri lama: dipisah `", "`, dan `"-"` bila kosong.
	DominantFactors string

	// RemarkRecommendation adalah nilai berlabel "Claim Recommendation :" —
	// `T_CLAIM_PNC.REMARKRECOMMENDATION`.
	RemarkRecommendation string

	// ProgressNote adalah nilai berlabel "Note dari PIC :".
	//
	// Di Pega ia parameter `notepic` dari baris grid, yang berisi `p.keterangan` pada catatan
	// progres TERAKHIR. Di sini dibaca ulang dari sumber yang sama.
	ProgressNote string

	// Objects adalah isi grid objek pertanggungan.
	Objects []DetailObject

	// ProgressHistory adalah isi grid riwayat progres, terbaru lebih dulu.
	ProgressHistory []DetailProgress

	// AdjusterMessages adalah isi grid "KOMUNIKASI DENGAN LOSS ADJUSTER".
	AdjusterMessages []DetailMessage
}

// DetailObject adalah satu baris grid objek pertanggungan.
//
// # Kenapa SEMUA kolom dibawa, bukan hanya yang tampil
//
// Grid ini punya TIGA varian kolom di Pega, dipilih menurut lini bisnis:
//
//	varian A   Nama Objek · Lokasi Object                        IsAneka · IsMarineCargo · IsFire
//	varian B   Nama Objek · Pekerjaan · Tanggal lahir            IsPA
//	varian C   Nama Peserta · Status · KTP/Paspor · Tanggal Lahir  IsTravel
//
// Kelima when rule ADA di export dan isinya sudah dibaca — hanya satu yang dapat dipetakan
// tepat:
//
//	IsPA            GroupPanel = "002"                       -> cocok persis dengan cob "PA"
//	IsTravel        Kode Bisnis = "77"                       -> BUKAN GroupPanel; belum cocok
//	IsAneka         Kode Bisnis = 24|18|17|10|07|06
//	IsMarineCargo   Kode Bisnis = 24|18|17|10|07|06           -> IDENTIK dengan IsAneka
//	IsFire          kosong, tanpa syarat sama sekali
//
// Karena tiga yang terakhir tidak dapat saling dibedakan — dan memang mengarah ke varian yang
// sama — seluruh kolom tetap dikirim, dan pemilihan variannya dikerjakan layar. Itu membuat
// tebakan Travel dapat diperbaiki tanpa menyentuh penyimpanan begitu kode bisnis tersedia.
type DetailObject struct {
	// ID adalah `T_CLAIM_OBJECTLIST.OBJECTID`.
	//
	// Ia tidak digambar. Ia kunci yang menautkan coverage ke objeknya, dan sekaligus kunci
	// baris di layar — tanpa kunci yang stabil, baris yang terbuka akan berpindah ketika
	// daftar diurutkan ulang.
	ID string

	// Name adalah `.ObjectName` — berjudul "Nama Objek" atau "Nama Peserta".
	Name string

	// Location adalah `.ObjectLocation`, yang di basis data bernama `LOKASI`.
	//
	// Nama kolomnya memang berbeda dari nama propertinya; `OBJECTLOCATION` tidak ada di
	// `T_CLAIM_OBJECTLIST`. Diverifikasi langsung ke katalog.
	Location string

	// Job adalah `.ObjectJob` — berjudul "Pekerjaan".
	Job string

	// DateOfBirth adalah `.ObjectDateOfBirth` — berjudul "Tanggal lahir"/"Tanggal Lahir".
	//
	// Kedua ejaan itu memang berbeda di layar lama, dan keduanya dibawa apa adanya (`D-13`).
	DateOfBirth *time.Time

	// IDCard adalah `.ObjectIDCard` — berjudul "KTP/Paspor".
	IDCard string

	// ParticipantStatus adalah `.ObjectParticipantStatus` — berjudul "Status".
	ParticipantStatus string

	// Coverages adalah isi panel yang terbuka ketika baris objek dibuka.
	//
	// Kosong berarti objek itu memang tidak punya coverage, bukan berarti belum dimuat:
	// seluruhnya dibaca sekali bersama objeknya, bukan satu permintaan per baris yang
	// dibuka. Lihat DetailCoverage.
	Coverages []DetailCoverage
}

// DetailCoverage adalah satu baris coverage di bawah sebuah objek pertanggungan.
//
// # Grid objek di Pega MEMANG dapat dibuka, dan ini isinya
//
// Mekanismenya bukan `pyExpandable` melainkan **master-detail**, dan itu mudah terlewat:
//
//	Section/DetailKlaimCabang_Sect-Section.xml
//	  pyRowEditing     = masterDetail       (ketiga varian grid objek: :5616, :7193, :9798)
//	  pyEditAction     = ViewObjectItem     (:5625, :7179, :9815)
//	  pyCustomTemplate = pzGridExpandPaneTemplate
//
// `Flow Action/ViewObjectItem-FlowAction.xml` merender `Section/ViewObjectCoverage-Section.xml`,
// yang menggambar `.ObjectCoverageList` dengan **tiga** kolom, berjudul persis:
//
//	Coverage    <- .CoverageNote, dan isinya NAMA jaminan (mis. "FLEXAS")
//	Mata Uang   <- .Currency
//	TSI         <- .SumTSI
//
// Grid progres dan grid komunikasi adjuster TIDAK dapat dibuka; keduanya `readOnly`
// (`:13162`, `:19741`).
//
// # Satu tingkat lagi yang BELUM dibangun
//
// Baris coverage di Pega juga master-detail, membuka `ViewObjectCoverageObjectItem` →
// `Section/ViewObjectItemList-Section.xml`, yang memuat rincian item, conveyance, dan batas
// coverage detail. Tingkat ketiga itu belum dibangun dan menunggu keputusan Work Owner —
// lingkupnya jauh lebih besar daripada tingkat ini.
type DetailCoverage struct {
	// ObjectID menautkan baris ini ke objeknya. Tidak digambar.
	ObjectID string

	// ID adalah `OBJECTCOVERAGEID` — kunci baris, tidak digambar.
	ID string

	// Name adalah `.CoverageNote` — digambar sebagai kolom pertama berjudul "Coverage".
	//
	// Di basis data ia `COVERAGENAME`, BUKAN `REMARKS`. Nama propertinya menyesatkan: yang
	// tersimpan nama jaminan, bukan catatan. Diukur, bukan ditebak — nilai contoh dari
	// layar lama muncul pada 358 baris `COVERAGENAME` dan nol baris `REMARKS`.
	Name string

	// Currency adalah `.Currency` — kolom "Mata Uang".
	Currency string

	// SumTSI adalah `.SumTSI` — kolom "TSI".
	SumTSI money.Money

	// Items adalah isi grid "Object Item" yang terbuka saat baris coverage dibuka.
	Items []DetailItem

	// Spreadings adalah isi grid "List Spreading".
	Spreadings []DetailSpreading

	// CoMembers adalah isi grid "CO MEMBER".
	CoMembers []DetailCoMember
}

// DetailSpreading adalah satu baris grid "List Spreading".
//
// # Ia melekat pada COVERAGE, bukan pada adjustment
//
// Sempat disimpulkan sebaliknya dari `Activity/CalculatedSpredingForClaimKomite-Act.xml`, yang
// memang menghitung per `AdjustmentList(Param.idadjustment)`. **Activity itu milik layar
// Komite, dan popup ini tidak memanggilnya.** Yang mengisi popup adalah `GetObjectFromTable`,
// dan di sana daftarnya melekat pada coverage. Kunci tabelnya sejalan: `CLAIMID + OBJECTID +
// OBJECTCOVERAGEID`.
//
// # Kedua nilai uangnya DIHITUNG
//
// `TSISPREADED` dan `PREMIUMSPREADED` NULL pada seluruh 7.115 baris tabelnya, sehingga
// keduanya tidak mungkin dibaca. Lihat EstimationValue.
type DetailSpreading struct {
	// ObjectID dan CoverageID menautkan baris ini ke coverage-nya. Tidak digambar.
	ObjectID   string
	CoverageID string

	// TreatyName adalah kolom "Tipe Treaty" — `TREATYNAME`, bukan `TREATYTYPE`.
	//
	// Isinya `ORS`, `QS`, `FAC-OUT`, `FSPL`, `OR`. `TREATYTYPE` berisi kodenya (`10007`).
	TreatyName string

	// Currency adalah kolom "Currency" — diambil dari coverage induknya.
	Currency string

	// EstimationValue adalah kolom "Estimasi Value" — JUMLAH estimasi coverage ini.
	//
	// Ia SIMPULAN, bukan bacaan langsung, dan dinyatakan lewat DetailPlannedDifferences.
	// Dasarnya contoh Fire dari Work Owner: estimasi `100 / -100 / 200 / -200 / 100`
	// berjumlah 100, dan layar lama menuliskan Estimasi Value `100,00`.
	EstimationValue money.Money

	// SharePercentScaled adalah persentase dikali 10.000 — `100%` menjadi `1000000`.
	//
	// Bilangan bulat, bukan pecahan: perkalian persentase terhadap nilai uang tidak boleh
	// menempuh float (`I-12`). Pemformatannya menjadi "100,0000%" dikerjakan layar.
	SharePercentScaled int64

	// ResultValue adalah kolom "Result Value" — `EstimationValue x share / 100`.
	ResultValue money.Money
}

// DetailCoMember adalah satu baris grid "CO MEMBER".
//
// Daftarnya berkunci POLIS (`T_COINSLIST.NOPOLIS + PRODKE`), sehingga satu daftar berlaku untuk
// seluruh coverage pada klaim itu. Yang berbeda per coverage hanyalah nilai uangnya, karena ia
// dihitung dari estimasi coverage masing-masing.
type DetailCoMember struct {
	ObjectID   string
	CoverageID string

	// InsurerName adalah kolom "Asuransi" — `COINSNAME`.
	InsurerName string

	// Currency adalah kolom "Currency" — diambil dari coverage induknya.
	//
	// Di Pega ia properti bernama `.ASIS`, yang diisi `Local.currencyshare`. Nama itu tidak
	// mencerminkan isinya sama sekali; yang dibawa artinya, bukan namanya (`D-19`).
	Currency string

	// EstimationValue adalah kolom "Estimasi Value" — jumlah estimasi coverage ini.
	//
	// Di Pega ia ditampung `.pyTotalShippingCost`, properti BAWAAN Pega untuk ongkos kirim
	// yang dipakai ulang. Utang teknis 4.2 dalam bentuknya yang paling ekstrem.
	EstimationValue money.Money

	// SharePercentScaled adalah `PERCENT_SHARE` dikali 10.000.
	SharePercentScaled int64

	// ResultValue adalah kolom "Result Value" — `EstimationValue x share / 100`.
	//
	// Bentuknya sama persis dengan rumus pada jalur Komite
	// (`TSIShare = .PercentShare * Local.grossvalue / 100`), hanya basisnya berbeda.
	ResultValue money.Money
}

// ShareOf menghitung porsi sebuah nilai uang menurut persentase berskala 10.000.
//
// Seluruhnya bilangan bulat. `nilai x share / (100 x 10000)`, dibulatkan ke sen terdekat —
// pembagi 1.000.000 sudah mencakup kedua skala sekaligus. Menempuh float di sini akan membuat
// pembagian share menghasilkan selisih sen yang tidak dapat direproduksi (`I-12`).
func ShareOf(value money.Money, sharePercentScaled int64) money.Money {
	const pembagi = 100 * 10000

	hasil := value.MinorUnits() * sharePercentScaled
	// Pembulatan ke terdekat, dan menyadari nilai NEGATIF: estimasi dapat bernilai minus,
	// dan membulatkan ke arah nol akan menggeser koreksi yang seharusnya saling meniadakan.
	if hasil < 0 {
		return money.FromMinorUnits((hasil - pembagi/2) / pembagi)
	}
	return money.FromMinorUnits((hasil + pembagi/2) / pembagi)
}

// DetailItem adalah satu baris grid "Object Item" — tingkat ketiga.
//
// # Barisnya datang dari ESTIMASI, bukan dari tabel item
//
// `T_CLAIM_OBJECTITEMLIST` memuat **1 baris di seluruh tabel**, sedangkan 22.004 klaim punya
// estimasi. Menarik grid ini dari sana akan menghasilkan nol baris untuk hampir setiap klaim,
// padahal layar lama menampilkan barisnya — dengan sel nama dan deskripsi KOSONG. Contoh PA
// dari Work Owner memperlihatkan persis itu.
//
// Karena itu barisnya dibentuk dari `OBJECTITEMID` yang muncul di estimasi, dan nama beserta
// deskripsinya di-LEFT JOIN — hampir selalu kosong, sebagaimana mestinya.
type DetailItem struct {
	// ObjectID dan CoverageID menautkan baris ini ke coverage-nya. Tidak digambar.
	ObjectID   string
	CoverageID string

	// ID adalah `OBJECTITEMID` — nomor urut item di dalam satu coverage, bukan kode jenis.
	ID string

	// Name adalah kolom "Object Item", Description kolom "Deskripsi Item".
	//
	// Keduanya hampir selalu kosong, dan itu BUKAN cacat: sumbernya memang tidak memuatnya.
	Name        string
	Description string

	// Estimations adalah isi grid "Estimasi" yang terbuka saat baris ini dibuka.
	Estimations []DetailEstimation
}

// DetailEstimation adalah satu baris grid "Estimasi" — tingkat keempat, yang terdalam.
//
// Keenam kolomnya diambil dari layar lama apa adanya (`D-13`).
type DetailEstimation struct {
	// ObjectID, CoverageID, dan ItemID menautkan baris ini ke itemnya. Tidak digambar.
	ObjectID   string
	CoverageID string
	ItemID     string

	// Sequence adalah kolom "Estimasi Ke" — `ESTIMASIID`.
	Sequence string

	// RecordedAt adalah kolom "Tanggal Estimasi".
	RecordedAt *time.Time

	// Type adalah kolom "Tipe Estimasi". Kerap kosong; contoh PA memperlihatkannya kosong.
	Type string

	// Currency adalah kolom "Mata Uang", sudah diterjemahkan dari kodenya.
	Currency string

	// Rate adalah kolom "Nilai Kurs (IDR)".
	Rate money.Money

	// Value adalah kolom "Nilai Estimasi".
	//
	// DAPAT bernilai negatif, dan itu dibawa apa adanya: contoh Fire dari Work Owner memuat
	// `100 / -100 / 200 / -200 / 100`, yakni koreksi yang saling meniadakan. Menyaring yang
	// negatif akan membuat jumlahnya tidak pernah cocok dengan layar lama.
	Value money.Money
}

// DetailProgress adalah satu baris grid riwayat progres.
//
// Asalnya `RDB List/GetCommunicationList-SQL.xml`, dan alias kolomnya adalah contoh paling
// jelas dari utang teknis 4.2 — nama yang dipakai TIDAK mencerminkan isinya:
//
//	A.TGL_INPUT      dialias "ObjekTanggal"   -> RecordedAt
//	A.PNCCASEID      dialias "ClaimID"        -> ClaimNumber
//	B.STS_PROGRESS1  dialias "Comment"        -> Status1
//	C.STS_PROGRESS2  dialias "IdCompliance"   -> Status2
//	A.USER_INPUT     dialias "Email"          -> EnteredBy
//	A.NEXT_FOLLOWUP  dialias "Date"           -> NextFollowUpAt
//	A.STATUS         dialias "AcceptedNo"     -> Status
//	A.KETERANGAN     dialias "BranchCode"     -> Note
//
// Tidak satu pun alias itu dibawa (`D-19`).
type DetailProgress struct {
	// RecordedAt berjudul "TANGGAL INPUT".
	RecordedAt *time.Time

	// ClaimNumber berjudul "NOMOR KLAIM".
	ClaimNumber string

	// Status1 berjudul "STATUS PROGRESS 1".
	//
	// Ia dibaca dari `GCNM_MST_PROGRESS_KLAIM`, BUKAN dari `GCNM_MST_PROGRESS` yang dipakai
	// kueri daftar. Kedua tabel itu berbeda, dan kueri lama memang memakai yang ini di sini.
	Status1 string

	// Status2 berjudul "STATUS PROGRESS 2".
	Status2 string

	// EnteredBy berjudul "USER INPUT".
	EnteredBy string

	// NextFollowUpAt berjudul "TANGGAL NEXT FOLLOWUP".
	NextFollowUpAt *time.Time

	// Status berjudul "STATUS".
	Status string

	// Note berjudul "Keterangan".
	//
	// Judulnya di layar lama tertulis `<b>Keterangan<b>` — tag penutupnya salah ketik,
	// sehingga sisa baris ikut menebal. Yang dibawa teksnya, bukan salah ketiknya.
	Note string
}

// DetailMessage adalah satu baris grid "KOMUNIKASI DENGAN LOSS ADJUSTER".
//
// Asalnya `RDB List/GetInboxKomunikasi_OS_Cabang-SQL.xml`.
type DetailMessage struct {
	// SenderName berjudul "Nama User".
	SenderName string

	// SentAt berjudul "Tanggal Proses".
	SentAt *time.Time

	// Message berjudul "Pesan".
	Message string

	// RepliedAt berjudul "Tanggal Balas".
	RepliedAt *time.Time

	// Reply berjudul "Jawaban".
	Reply string

	// Internal menyatakan pengirimnya petugas teknis, bukan adjuster luar.
	//
	// Kueri lama menghitungnya dengan mencocokkan pengirim ke `MST_USER_TEKNIK` yang aktif,
	// lalu memakainya untuk menyusun urutan baris. Nilainya TIDAK digambar sebagai kolom,
	// tetapi dibawa karena ia satu-satunya yang membedakan pesan keluar dari pesan masuk —
	// dan tanpa itu layar tidak dapat menempatkannya di sisi yang benar.
	Internal bool
}

// DetailPlannedDifferences adalah selisih popup terhadap Pega yang sudah diputuskan.
//
// Terpisah dari PlannedDifferences supaya layar daftar tidak menampilkan catatan yang hanya
// berlaku di popup, dan sebaliknya (`D-54`).
var DetailPlannedDifferences = []string{
	"Kolom \"Estimasi Value\" pada List Spreading dan CO MEMBER adalah JUMLAH estimasi " +
		"coverage itu. Layar lama tidak menyimpan angkanya — kolom TSISPREADED dan " +
		"PREMIUMSPREADED kosong pada seluruh barisnya — sehingga nilainya disimpulkan dari " +
		"kecocokan pada contoh yang diberikan Work Owner, bukan dibaca dari rule.",
	"Nilai \"Total Sum Insured\" adalah TSI objek terakhir pada klaim, bukan jumlah seluruh " +
		"objek. Layar lama menampilkan angka yang sama; labelnya yang menyesatkan.",
	"Varian kolom grid objek dipilih dari lini bisnis. Untuk PA pemetaannya sama persis " +
		"dengan layar lama. Untuk Travel belum: layar lama menyaringnya dengan Kode Bisnis " +
		"\"77\", sedangkan layar ini memakai Group Panel 005, dan keduanya belum dibuktikan " +
		"menunjuk klaim yang sama.",
	"Nilai Total Reserve, Aging, dan Note dari PIC dibaca ulang dari penyimpanan. Layar lama " +
		"menerimanya dari baris yang diklik, sehingga popup dapat menampilkan angka yang " +
		"sudah tidak berlaku bila datanya berubah sejak daftar dimuat.",
}
