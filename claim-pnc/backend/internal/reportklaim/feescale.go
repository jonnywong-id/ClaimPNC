package reportklaim

// FeeBand adalah satu pita tangga fee adjuster.
//
// Sumbernya `POOLDATA.GCNM_FEE_SCALE`, tiga kolom: `INDEX_FEE`, `LOSS_AMOUNT`, `FEE`.
type FeeBand struct {
	// Index adalah urutan pita. Ia dipakai sebagai KUNCI, bukan sekadar pengurut —
	// interpolasi mencari pita `Index-1`, bukan "pita sebelum ini menurut nilai".
	Index int

	// LossAmount adalah batas ATAS nilai kerugian pita ini.
	LossAmount float64

	// Fee adalah fee adjuster pada batas atas itu.
	Fee float64
}

// FeeScale adalah tangga fee adjuster, pengganti `POOLDATA.Get_InterpolasiPNC`.
//
// # Kenapa ia ada di paket domain
//
// `D-02` menetapkan seluruh logika stored procedure dinaikkan ke Go. Yang diambil dari
// basis data hanyalah ISI TANGGANYA — 17 baris yang berubah ketika kebijakan fee berubah.
// Aturan interpolasinya sendiri adalah aturan bisnis dan hidup di sini.
//
// # Kenapa dibaca sekali per laporan
//
// Fungsi aslinya menjalankan tiga kueri SETIAP KALI dipanggil, dan ia dipanggil sekali per
// baris settlement. Pada laporan berisi puluhan ribu baris itu puluhan ribu kali tiga
// kueri. Tangganya kecil dan jarang berubah, jadi ia dibaca sekali lalu dipakai ulang.
type FeeScale struct {
	byIndex map[int]FeeBand

	// top adalah pita pangkal ekstrapolasi: pita ber-Index TopBandIndex.
	//
	// Ia BUKAN pita ber-indeks terbesar. Lihat catatan pada Fee.
	top    FeeBand
	adaTop bool

	loaded bool
}

// FeeBeforeFirstBand adalah fee tetap untuk nilai yang masih di dalam pita pertama.
//
// Angkanya DI-HARDCODE di dalam `Database/GET_INTERPOLASIPNC.fnc` (`ehasil := 1650000`),
// bukan dibaca dari master mana pun. Ia dibawa apa adanya demi kesetaraan perilaku
// (`P-5`), tetapi keberadaannya melanggar `D-15` — nilai bisnis tidak boleh di-hardcode.
//
// Pemindahannya ke master data menuntut kolom baru pada tangga fee, dan itu perubahan
// master yang menempuh `D-63`. Sampai itu diputuskan, ia berdiri di sini sebagai konstanta
// bernama supaya terlihat, bukan tersembunyi di tengah perhitungan.
const FeeBeforeFirstBand = 1650000.0

// TopBandIndex adalah nomor pita yang menjadi pangkal ekstrapolasi.
//
// Angkanya DI-HARDCODE di dalam `Database/GET_INTERPOLASIPNC.fnc`
// (`WHERE INDEX_FEE = 17`), bukan diturunkan dari isi master. Ia dibawa apa adanya demi
// kesetaraan perilaku (`P-5`) dan berdiri sebagai konstanta bernama supaya terlihat.
const TopBandIndex = 17

// feeAboveTopBandRate adalah kemiringan di atas pita pangkal: 2% dari kelebihan nilai.
//
// Sumbernya `GET_INTERPOLASIPNC.fnc`: `ehasil := max_fee + ((data_ex - max_amount) * (2/100))`.
const feeAboveTopBandRate = 0.02

// NewFeeScale membentuk tangga dari baris master.
//
// Daftar kosong menghasilkan tangga yang TIDAK tersedia — bukan tangga kosong yang
// menjawab nol. Tangga tanpa pita tidak dapat menjawab apa pun, dan menjawab nol pada
// kolom fee berarti melaporkan bahwa adjuster tidak dibayar.
func NewFeeScale(bands []FeeBand) *FeeScale {
	if len(bands) == 0 {
		return UnavailableFeeScale()
	}

	s := &FeeScale{byIndex: make(map[int]FeeBand, len(bands)), loaded: true}
	for _, b := range bands {
		s.byIndex[b.Index] = b
	}

	s.top, s.adaTop = s.byIndex[TopBandIndex]

	return s
}

// UnavailableFeeScale adalah tangga yang sumbernya tidak dapat dibaca.
func UnavailableFeeScale() *FeeScale { return &FeeScale{} }

// Available menyatakan apakah tangga ini terisi dari sumbernya.
func (s *FeeScale) Available() bool { return s != nil && s.loaded }

// Count mengembalikan banyaknya pita.
func (s *FeeScale) Count() int {
	if s == nil {
		return 0
	}
	return len(s.byIndex)
}

// Fee menghitung fee adjuster untuk sebuah nilai kerugian.
//
// # Aturannya, dibaca dari GET_INTERPOLASIPNC.fnc
//
//	pita  = pita ber-INDEX_FEE terkecil yang LOSS_AMOUNT-nya >= nilai
//
//	ada pita, dan indeksnya > 1   → interpolasi linear terhadap pita indeks sebelumnya
//	ada pita, dan indeksnya = 1   → FeeBeforeFirstBand (nilai tetap)
//	tidak ada pita                → fee pita terakhir + 2% dari kelebihan nilai
//
// Interpolasinya:
//
//	fee = fee_bawah + (nilai − amount_bawah) × (fee_atas − fee_bawah) / (amount_atas − amount_bawah)
//
// # Pangkal ekstrapolasinya pita bernomor 17, bukan pita terakhir
//
// Fungsi aslinya mengambilnya dengan `WHERE INDEX_FEE = 17` — angka tetap di dalam kode,
// bukan "pita terakhir yang ada". Keduanya sama selama master berisi tepat 17 pita.
//
// Sempat diganti menjadi pita ber-indeks TERBESAR supaya tetap benar bila master bertambah.
// Penggantian itu DICABUT: Work Owner menetapkan 2026-09-25 bahwa cacat yang berasal dari
// Pega dibiarkan seperti Pega.
//
// Akibat yang harus disadari: bila master kelak bertambah menjadi 18 pita, seluruh nilai di
// atas pita ke-18 diekstrapolasi dari pita ke-17 — jawaban yang salah tanpa satu pun tanda.
// Konstanta TopBandIndex berdiri supaya perubahan itu berupa suntingan satu baris, bukan
// pencarian angka 17 di tengah perhitungan.
//
// # Yang dikembalikan saat tidak dapat dihitung
//
// `false`, pada tiga keadaan: tangga tidak tersedia, pita indeks sebelumnya tidak ada, dan
// nilai kerugian negatif. Sistem lama MENGGAGALKAN seluruh kueri pada dua keadaan pertama
// (`NO_DATA_FOUND`); di sini satu baris kehilangan satu sel, dan laporannya tetap terunduh.
//
// Penjaga pembagi nol di bawah tidak dapat dicapai — pita berimpit selalu diserap lebih
// dulu oleh pemilihan indeks terkecil — dan dibiarkan berdiri hanya sebagai jaring bila
// aturan pemilihannya kelak berubah. Lihat uji yang mengunci penalarannya.
func (s *FeeScale) Fee(lossValue float64) (float64, bool) {
	if !s.Available() || lossValue < 0 {
		return 0, false
	}

	upper, found := s.smallestBandCovering(lossValue)
	if !found {
		// Nilainya melampaui seluruh pita: ekstrapolasi dari pita TopBandIndex.
		//
		// Pita itu tidak ada berarti tidak dapat dihitung. Sistem lama melempar
		// NO_DATA_FOUND dan menggagalkan seluruh kueri; di sini satu sel yang kosong.
		if !s.adaTop {
			return 0, false
		}
		return s.top.Fee + (lossValue-s.top.LossAmount)*feeAboveTopBandRate, true
	}
	if upper.Index <= 1 {
		return FeeBeforeFirstBand, true
	}

	lower, ada := s.byIndex[upper.Index-1]
	if !ada {
		return 0, false
	}
	span := upper.LossAmount - lower.LossAmount
	if span == 0 {
		return 0, false
	}
	return lower.Fee + (lossValue-lower.LossAmount)*(upper.Fee-lower.Fee)/span, true
}

// smallestBandCovering mencari pita ber-indeks terkecil yang menampung nilai.
//
// Ia menelusuri seluruh pita alih-alih memakai pencarian biner: tangganya 17 baris, dan
// penelusuran lurus tidak menuntut urutan indeks sejalan dengan urutan LOSS_AMOUNT —
// sebuah asumsi yang tidak dijamin apa pun di master.
func (s *FeeScale) smallestBandCovering(value float64) (FeeBand, bool) {
	var best FeeBand
	found := false
	for _, b := range s.byIndex {
		if b.LossAmount < value {
			continue
		}
		if !found || b.Index < best.Index {
			best, found = b, true
		}
	}
	return best, found
}
