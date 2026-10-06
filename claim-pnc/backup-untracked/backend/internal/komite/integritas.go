package komite

import (
	"fmt"
	"sort"

	"claim-pnc/internal/platform/uang"
)

// Tingkat menyatakan seberapa berat sebuah temuan.
type Tingkat string

const (
	// TingkatCacat berarti masternya SALAH dan hasil penjenjangan tidak dapat
	// dipercaya: rentang yang tumpang tindih, berlubang, atau terbalik.
	TingkatCacat Tingkat = "cacat"

	// TingkatPeringatan berarti masternya tidak salah, tetapi ada sesuatu yang pantas
	// dilihat Work Owner sebelum dipakai memutuskan uang.
	TingkatPeringatan Tingkat = "peringatan"
)

// Jenis temuan. Dipakai layar untuk mengelompokkan dan menjelaskan, bukan untuk
// mencocokkan teks pesan.
const (
	JenisBatasTerbalik = "batas_terbalik"
	JenisTumpangTindih = "tumpang_tindih"
	JenisBerlubang     = "berlubang"
	JenisJenjangGanda  = "jenjang_ganda"
	JenisAtapTangga    = "atap_tangga"
	JenisTanpaJenjang  = "tanpa_jenjang"
)

// Temuan adalah satu hal yang ditemukan pada master ambang.
type Temuan struct {
	Tingkat Tingkat
	Jenis   string

	Lini Lini

	// Pita terisi hanya untuk lini yang memakainya.
	Pita string

	// Pesan sudah berbahasa Indonesia dan siap ditampilkan apa adanya.
	Pesan string

	// IDAmbang menunjuk baris master yang terlibat, supaya temuan dapat ditelusuri ke
	// datanya alih-alih hanya dibaca sebagai keluhan.
	IDAmbang []string
}

// PeriksaIntegritas memeriksa kesehatan master ambang komite dan mengembalikan seluruh
// temuannya.
//
// # Inilah satu-satunya tempat LIMIT_TOP dipakai
//
// `D-47` menetapkan perannya dengan tegas: `LIMIT_TOP` BUKAN penyaring pemilih baris —
// memakainya untuk memilih akan mengembalikan tepat satu baris dan menghapus
// penjenjangan seluruhnya — melainkan alat memeriksa apakah tangganya tersusun rapi.
//
// # Kenapa pengelompokannya berbeda antar lini
//
// Untuk lini BERPITA (Non-MBU), tangganya dikelompokkan per pita: dua deret yang berdiri
// sendiri, dan akumulasi tidak pernah menyeberang di antaranya.
//
// Untuk lini lain, tangganya SATU deret utuh dan TYPE_KOMITE tidak boleh dipakai
// mengelompokkan. Bila dipaksa demikian, tangga PA akan terbelah menjadi dua potongan
// yang tampak berlubang parah — padahal pada PA kolom itu membedakan PA reguler dari
// PA TKI, bukan pita nilai (`D-70`).
//
// Inilah yang dimaksud catatan `D-47` bahwa validasi ini "hanya dapat dijalankan per
// lini, bukan lintas lini".
func PeriksaIntegritas(ambang []Ambang, kebijakan Kebijakan) []Temuan {
	kelompok := map[string][]Ambang{}
	var urutanKelompok []string

	for _, a := range ambang {
		a = a.Bersih()
		if !a.JenjangPersetujuan() {
			continue
		}

		kunci := string(a.Lini)
		if _, berpita := kebijakan.Pita[a.Lini]; berpita {
			kunci = string(a.Lini) + "\x00" + a.JenisKomite
		}
		if _, ada := kelompok[kunci]; !ada {
			urutanKelompok = append(urutanKelompok, kunci)
		}
		kelompok[kunci] = append(kelompok[kunci], a)
	}

	sort.Strings(urutanKelompok)

	var temuan []Temuan
	for _, kunci := range urutanKelompok {
		temuan = append(temuan, periksaSatuKelompok(kelompok[kunci], kebijakan)...)
	}

	if len(urutanKelompok) == 0 {
		temuan = append(temuan, Temuan{
			Tingkat: TingkatCacat,
			Jenis:   JenisTanpaJenjang,
			Pesan: "Master ambang komite tidak memuat satu pun jenjang persetujuan yang " +
				"aktif. Tidak ada klaim yang dapat melewati komite.",
		})
	}

	return temuan
}

// periksaSatuKelompok memeriksa satu tangga — satu lini, atau satu pita di dalam lini.
func periksaSatuKelompok(baris []Ambang, kebijakan Kebijakan) []Temuan {
	if len(baris) == 0 {
		return nil
	}

	lini := baris[0].Lini
	aturan, berpita := kebijakan.Pita[lini]
	pita := ""
	if berpita {
		pita = baris[0].JenisKomite
	}

	// Disalin lebih dulu: sort mengubah senarai di tempat, dan mengurutkan milik
	// pemanggil akan mengubah data yang bukan milik fungsi ini.
	tangga := append([]Ambang(nil), baris...)
	sort.SliceStable(tangga, func(i, j int) bool {
		if tangga[i].BatasBawah != tangga[j].BatasBawah {
			return tangga[i].BatasBawah < tangga[j].BatasBawah
		}
		return tangga[i].ID < tangga[j].ID
	})

	buat := func(tingkat Tingkat, jenis, pesan string, id ...string) Temuan {
		return Temuan{
			Tingkat:  tingkat,
			Jenis:    jenis,
			Lini:     lini,
			Pita:     pita,
			Pesan:    pesan,
			IDAmbang: id,
		}
	}

	var temuan []Temuan

	// 1 — batas terbalik. Diperiksa lebih dulu karena baris yang terbalik membuat
	// pemeriksaan kesinambungan di bawahnya menghasilkan pesan yang menyesatkan.
	for _, a := range tangga {
		if a.BatasAtas < a.BatasBawah && a.BatasAtas != uang.Nol {
			temuan = append(temuan, buat(TingkatCacat, JenisBatasTerbalik,
				fmt.Sprintf("Batas atas (%s) lebih kecil daripada batas bawah (%s).",
					a.BatasAtas, a.BatasBawah),
				a.ID))
		}
	}

	// 2 — kesinambungan antar anak tangga.
	//
	// Anak tangga berikutnya seharusnya mulai TEPAT satu rupiah di atas batas atas anak
	// tangga sebelumnya. Satuannya rupiah, bukan sen, karena seluruh isi master ditulis
	// dalam rupiah bulat dan tangganya memang melangkah demikian —
	// 50.000.000 diikuti 50.000.001.
	satuRupiah := uang.DariRupiah(1)
	for i := 1; i < len(tangga); i++ {
		sebelum, sesudah := tangga[i-1], tangga[i]

		// Baris ber-batas atas nol diperlakukan sebagai "tidak berbatas atas" dan
		// dilewati: pada master yang berlaku, BONDING memakai 0/0 sebagai penanda
		// bahwa tangganya memang tidak bertingkat.
		if sebelum.BatasAtas == uang.Nol {
			continue
		}

		switch {
		case sesudah.BatasBawah <= sebelum.BatasAtas:
			temuan = append(temuan, buat(TingkatCacat, JenisTumpangTindih,
				fmt.Sprintf("Rentang %s–%s dan %s–%s saling menindih. "+
					"Nilai di antaranya masuk dua jenjang sekaligus.",
					sebelum.BatasBawah, sebelum.BatasAtas,
					sesudah.BatasBawah, sesudah.BatasAtas),
				sebelum.ID, sesudah.ID))

		case sesudah.BatasBawah > sebelum.BatasAtas+satuRupiah:
			temuan = append(temuan, buat(TingkatCacat, JenisBerlubang,
				fmt.Sprintf("Ada lubang antara %s dan %s. "+
					"Tidak ada jenjang yang secara khusus mewakili nilai di antaranya.",
					sebelum.BatasAtas, sesudah.BatasBawah),
				sebelum.ID, sesudah.ID))
		}
	}

	// 3 — jenjang ganda.
	//
	// Bukan cacat: `D-52` mencatat DEGREE hanya menentukan urutan dan boleh berulang.
	// Tetapi kueri lama mengurutkan dengan `ORDER BY DEGREE` saja, sehingga saat seri
	// urutannya ditentukan basis data dan dapat berubah antar eksekusi. Itu pantas
	// dilihat, karena ia satu-satunya sumber perbedaan urutan terhadap Pega.
	perJenjang := map[int][]string{}
	var urutJenjang []int
	for _, a := range tangga {
		if _, ada := perJenjang[a.Jenjang]; !ada {
			urutJenjang = append(urutJenjang, a.Jenjang)
		}
		perJenjang[a.Jenjang] = append(perJenjang[a.Jenjang], a.ID)
	}
	sort.Ints(urutJenjang)
	for _, j := range urutJenjang {
		if len(perJenjang[j]) > 1 {
			temuan = append(temuan, buat(TingkatPeringatan, JenisJenjangGanda,
				fmt.Sprintf("Ada %d baris ber-jenjang %d. Urutan menyetujui di antara "+
					"keduanya tidak ditentukan master, sehingga sistem lama dapat "+
					"mengurutkannya berbeda-beda.", len(perJenjang[j]), j),
				perJenjang[j]...))
		}
	}

	// 4 — atap tangga.
	//
	// Pernyataannya harus tepat, dan di sinilah tiket `TKT-B07-001` terlalu jauh: ia
	// menulis klaim di atas atap "tidak punya penyetuju sama sekali". Di bawah aturan
	// kumulatif itu TIDAK BENAR — klaim sebesar apa pun tetap memenuhi seluruh batas
	// bawah, sehingga justru mendapat SELURUH penyetuju pada tangga itu.
	//
	// Yang sebenarnya terjadi: tangganya berhenti membedakan. `D-52` sudah menyatakan
	// hal yang sama — "klaim PA atau Travel di atas nilai itu tetap mendapat 4 dan 3
	// jenjang karena model kumulatif, tetapi tidak ada baris yang secara eksplisit
	// mencakupnya".
	atap := uang.Nol
	var idAtap []string
	takBerbatas := false
	for _, a := range tangga {
		if a.BatasAtas == uang.Nol {
			takBerbatas = true
			continue
		}
		if a.BatasAtas > atap {
			atap = a.BatasAtas
			idAtap = []string{a.ID}
		}
	}

	// Pada pita bawah, atap yang sama dengan batas pita justru BENAR — di atas nilai itu
	// klaim memang berpindah ke pita berikutnya. Melaporkannya akan menjadi peringatan
	// palsu yang muncul setiap kali layar dibuka.
	atapWajar := berpita && pita == aturan.Bawah && atap == aturan.Batas

	if !takBerbatas && !atapWajar && len(idAtap) > 0 {
		temuan = append(temuan, buat(TingkatPeringatan, JenisAtapTangga,
			fmt.Sprintf("Tangga berhenti di %s. Klaim di atas nilai itu tetap "+
				"mendapat seluruh %d penyetuju karena aturannya kumulatif, tetapi "+
				"tidak ada jenjang yang secara khusus mewakilinya.",
				atap, len(tangga)),
			idAtap...))
	}

	return temuan
}
