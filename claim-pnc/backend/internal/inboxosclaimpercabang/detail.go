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
