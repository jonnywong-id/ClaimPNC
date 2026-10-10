// Package uang menyediakan satu tipe nilai uang untuk seluruh aplikasi.
//
// # Kenapa ada tipe tersendiri, bukan float64
//
// `I-12` menetapkan nilai uang disimpan dengan PRESISI PENUH dan pembulatan hanya
// terjadi saat ditampilkan, dan `docs/Steering/09-DATABASE-STRATEGY.md` §5 menyebutnya
// "tidak bisa ditawar": nilai uang tidak pernah `float`/`double`.
//
// Alasannya bukan kerapian. Modul Komite membandingkan nilai klaim terhadap ambang
// bawah — Rp 50.000.001 melawan Rp 50.000.000 — dan selisih satu rupiah di sana
// menentukan apakah satu jenjang persetujuan ikut atau tidak. Pembulatan floating point
// membuat perbandingan seperti itu gagal secara acak dan tidak dapat direproduksi.
//
// # Bentuk yang dipilih: bilangan bulat satuan terkecil
//
// Uang disimpan sebagai int64 dalam SATUAN TERKECIL (sen), yaitu rupiah dikali 100.
// Dua desimal mengikuti tipe kolom yang ditetapkan Database Strategy §5 —
// `NUMERIC(18,2)` di PostgreSQL, `NUMBER(18,2)` di Oracle.
//
// Batasnya lapang: nilai terbesar yang benar-benar ada di master ambang komite hari ini
// adalah Rp 100.000.000.000 (`Database/emailkomite.csv`, baris ID 4), yaitu 10^13 satuan
// terkecil, sementara int64 menampung sampai sekitar 9,2 × 10^18. Tidak ada nilai klaim
// yang mendekati batas itu.
//
// # Kenapa bukan pustaka desimal pihak ketiga
//
// `docs/Steering/08-TECHNICAL-STRATEGY.md` menetapkan "utamakan pustaka standar; setiap
// dependensi pihak ketiga adalah satu hal lagi yang harus dipelajari tim, dipantau
// keamanannya, dan bisa ditinggalkan pemeliharanya". Untuk kebutuhan yang seluruhnya
// penjumlahan dan perbandingan pada dua desimal, bilangan bulat sudah cukup dan tidak
// menambah satu pun dependensi.
//
// Bila kelak dibutuhkan pembagian berpresisi tinggi — perhitungan share reasuransi enam
// desimal pada `B-4`, atau interpolasi fee adjuster `GET_INTERPOLASIPNC` yang membagi
// dua kali tanpa pembulatan — keputusan itu diambil tersendiri, dan tipe ini tidak
// menghalanginya.
//
// Lapisan platform: tidak mengimpor apa pun dari domain, adapter, maupun transport.
package uang

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Desimal adalah jumlah angka di belakang koma yang disimpan tipe ini.
const Desimal = 2

// faktor adalah pengali dari rupiah ke satuan terkecil.
const faktor int64 = 100

// Nol memudahkan perbandingan terbaca, misalnya `if nilai == uang.Nol`.
const Nol Uang = 0

// ErrFormat dikembalikan bila teks yang diurai bukan nilai uang yang sah.
//
// Ia sengaja satu galat, bukan satu per sebab: pemanggil hanya perlu tahu "ini bukan
// nilai uang", dan rincian sebabnya dibungkus sebagai teks supaya tetap terbaca di log.
var ErrFormat = errors.New("uang: format nilai uang tidak sah")

// Uang adalah nilai uang dalam SATUAN TERKECIL — rupiah dikali 100.
//
// Ia bertipe int64 supaya perbandingan biasa (`<=`, `==`) langsung berlaku dan tidak
// ada cara memakainya secara salah dengan tanpa sengaja memanggil metode yang keliru.
type Uang int64

// DariRupiah membentuk nilai dari rupiah bulat.
//
// Panik bila hasilnya melampaui int64. Itu disengaja: pemanggilnya adalah konstanta di
// dalam kode dan berkas uji, bukan masukan pengguna, sehingga kelebihan di sini adalah
// cacat pemrograman yang harus terlihat saat pertama dijalankan.
func DariRupiah(rupiah int64) Uang {
	if rupiah > math.MaxInt64/faktor || rupiah < math.MinInt64/faktor {
		panic(fmt.Sprintf("uang: %d rupiah melampaui batas yang dapat disimpan", rupiah))
	}
	return Uang(rupiah * faktor)
}

// DariSatuanTerkecil membentuk nilai dari satuan terkecil apa adanya.
func DariSatuanTerkecil(satuan int64) Uang { return Uang(satuan) }

// SatuanTerkecil mengembalikan nilai dalam satuan terkecil.
//
// Inilah bentuk yang disimpan dan dibandingkan. Ia dipakai pengujian dan penyimpanan,
// bukan tampilan.
func (u Uang) SatuanTerkecil() int64 { return int64(u) }

// String mengembalikan bentuk kanonik: angka desimal dengan TEPAT dua angka di belakang
// titik, tanpa pemisah ribuan dan tanpa lambang mata uang.
//
// Bentuk inilah yang dikirim ke peramban dan diterima kembali darinya. Ia sengaja
// bentuk MESIN, bukan bentuk baca: pemisah ribuan, lambang "Rp", dan pemilihan koma
// versus titik adalah urusan tampilan, dan menaruhnya di sini akan membuat nilai yang
// dikirim API bergantung pada selera layar.
//
// Ini sekaligus menjawab utang teknis 4.2 sistem lama: 411 pemakaian `TO_CHAR` membuat
// basis data mengembalikan angka dan tanggal sebagai teks berformat tampilan, sehingga
// pengurutan dan penyaringan diam-diam menjadi pengurutan teks.
func (u Uang) String() string {
	negatif := u < 0
	satuan := int64(u)
	if negatif {
		// Ditangani sebagai nilai positif lalu diberi tanda, supaya MinInt64 tidak
		// membalik tanda saat dinegasikan.
		if satuan == math.MinInt64 {
			// Nilai ini mustahil muncul dari data nyata; ditangani supaya fungsi ini
			// tidak punya satu pun masukan yang membuatnya berperilaku aneh.
			return "-92233720368547758.08"
		}
		satuan = -satuan
	}

	utuh := satuan / faktor
	pecahan := satuan % faktor

	var b strings.Builder
	if negatif {
		b.WriteByte('-')
	}
	b.WriteString(strconv.FormatInt(utuh, 10))
	b.WriteByte('.')
	if pecahan < 10 {
		b.WriteByte('0')
	}
	b.WriteString(strconv.FormatInt(pecahan, 10))
	return b.String()
}

// Urai membaca teks menjadi nilai uang.
//
// Yang diterima: bilangan bulat ("50000000"), bilangan berdesimal paling banyak dua
// angka ("50000000.5", "50000000.50"), dengan tanda minus di depan bila perlu. Spasi
// tepi dibuang.
//
// Yang DITOLAK, dan alasannya: pemisah ribuan ("50.000.000" atau "50,000,000") tidak
// diterima karena artinya berbeda antar bahasa — titik adalah pemisah ribuan di
// Indonesia dan pemisah desimal di Inggris, sehingga menerimanya berarti menebak maksud
// pengirim. Notasi ilmiah ("1e8") juga ditolak karena ia jalan masuk ketidaktepatan
// floating point yang justru dihindari tipe ini.
func Urai(teks string) (Uang, error) {
	bersih := strings.TrimSpace(teks)
	if bersih == "" {
		return 0, fmt.Errorf("%w: kosong", ErrFormat)
	}

	negatif := false
	switch bersih[0] {
	case '-':
		negatif = true
		bersih = bersih[1:]
	case '+':
		bersih = bersih[1:]
	}
	if bersih == "" {
		return 0, fmt.Errorf("%w: hanya berisi tanda", ErrFormat)
	}

	utuhTeks, pecahanTeks, adaTitik := strings.Cut(bersih, ".")
	if !hanyaAngka(utuhTeks) || utuhTeks == "" {
		return 0, fmt.Errorf("%w: %q bukan angka", ErrFormat, teks)
	}
	if adaTitik {
		if !hanyaAngka(pecahanTeks) || pecahanTeks == "" {
			return 0, fmt.Errorf("%w: %q bukan angka", ErrFormat, teks)
		}
		if len(pecahanTeks) > Desimal {
			return 0, fmt.Errorf("%w: %q melebihi %d angka di belakang titik",
				ErrFormat, teks, Desimal)
		}
	}

	utuh, err := strconv.ParseInt(utuhTeks, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q terlalu besar", ErrFormat, teks)
	}
	if utuh > math.MaxInt64/faktor {
		return 0, fmt.Errorf("%w: %q terlalu besar", ErrFormat, teks)
	}

	// Pecahan dinormalkan ke dua angka: "5" berarti 50 satuan terkecil, bukan 5.
	pecahan := int64(0)
	if adaTitik {
		lengkap := pecahanTeks + strings.Repeat("0", Desimal-len(pecahanTeks))
		pecahan, err = strconv.ParseInt(lengkap, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %q", ErrFormat, teks)
		}
	}

	satuan := utuh*faktor + pecahan
	if negatif {
		satuan = -satuan
	}
	return Uang(satuan), nil
}

// DariNilaiSQL mengubah nilai yang datang dari basis data menjadi Uang.
//
// # Kenapa ini tidak sesederhana memindai ke int64
//
// Kolom Oracle bertipe NUMBER dapat sampai ke Go dalam beberapa bentuk yang berbeda,
// bergantung pada presisi kolom dan konfigurasi driver: int64, float64, []byte, string,
// atau tipe angka milik driver yang berperilaku seperti string.
//
// Memindai langsung ke *string pun tidak aman: bila driver menyerahkan float64,
// database/sql memformatnya dengan `strconv.FormatFloat(v, 'g', -1, 64)`, dan 'g'
// menghasilkan notasi ilmiah untuk angka besar — Rp 100.000.000 menjadi "1e+08", yang
// kemudian gagal diurai. Karena itu setiap bentuk ditangani di sini secara eksplisit.
//
// float64 diterima HANYA bila nilainya bilangan bulat dan masih di bawah 2^53, rentang
// yang dijamin tepat. Di luar itu ia ditolak, bukan dibulatkan diam-diam — nilai uang
// yang sudah kehilangan ketepatan tidak boleh diteruskan seolah-olah masih tepat.
func DariNilaiSQL(nilai any) (Uang, error) {
	switch v := nilai.(type) {
	case nil:
		// NULL diperlakukan sebagai nol. Pada master ambang komite, LIMIT_BOTTOM yang
		// kosong berarti "tidak ada batas bawah", dan nol menyatakan itu dengan tepat.
		return 0, nil
	case int64:
		return dariRupiahAman(v)
	case int:
		return dariRupiahAman(int64(v))
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("%w: bukan bilangan", ErrFormat)
		}
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("%w: %v berdesimal dan datang sebagai float64, "+
				"ketepatannya tidak dapat dijamin", ErrFormat, v)
		}
		const batasTepat = 1 << 53
		if v >= batasTepat || v <= -batasTepat {
			return 0, fmt.Errorf("%w: %v melampaui rentang yang float64 masih tepat", ErrFormat, v)
		}
		return dariRupiahAman(int64(v))
	case []byte:
		return Urai(string(v))
	case string:
		return Urai(v)
	default:
		// Tipe angka milik driver yang berperilaku seperti string tercakup di sini.
		// Driver yang dipakai aplikasi ini adalah github.com/sijms/go-ora/v2, yang
		// menyerahkan NUMBER sebagai int64, float64, atau string bergantung pada
		// presisi kolomnya — ketiganya sudah ditangani di atas. Cabang ini menjaga
		// pergantian driver kelak tidak langsung membuat modul ini gagal.
		if teks, bisa := nilai.(fmt.Stringer); bisa {
			return Urai(teks.String())
		}
		return 0, fmt.Errorf("%w: tipe %T tidak dikenali", ErrFormat, nilai)
	}
}

// dariRupiahAman adalah DariRupiah yang MENGEMBALIKAN GALAT alih-alih panik.
//
// Pembedaannya penting dan bukan gaya: DariRupiah dipanggil dengan konstanta di dalam
// kode, sehingga kelebihan di sana adalah cacat pemrograman dan pantas terlihat sebagai
// panik. Yang masuk lewat DariNilaiSQL datang dari BASIS DATA — data dari luar — dan
// data yang aneh tidak boleh menjatuhkan proses yang sedang melayani pengguna lain.
func dariRupiahAman(rupiah int64) (Uang, error) {
	if rupiah > math.MaxInt64/faktor || rupiah < math.MinInt64/faktor {
		return 0, fmt.Errorf("%w: %d rupiah melampaui batas yang dapat disimpan", ErrFormat, rupiah)
	}
	return Uang(rupiah * faktor), nil
}

// hanyaAngka memeriksa seluruh karakter adalah digit ASCII.
//
// Sengaja tidak memakai unicode.IsDigit: digit Arab-Hindi dan sejenisnya lolos
// pemeriksaan itu tetapi tidak dapat diurai strconv, sehingga penerimaannya di sini
// hanya akan menggeser kegagalan ke tempat yang lebih sulit dibaca.
func hanyaAngka(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
