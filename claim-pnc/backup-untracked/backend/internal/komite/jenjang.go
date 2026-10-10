package komite

import (
	"sort"

	"claim-pnc/internal/platform/uang"
)

// BatasPitaNonMBUBawaan adalah nilai yang memisahkan pita 1 dari pita 2 pada lini
// Non-MBU: sampai Rp 100.000.000 masuk pita "1", di atasnya masuk pita "2" (`D-52`).
//
// # Kenapa angkanya ada di kode, padahal D-15 melarang nilai bisnis di-hardcode
//
// Ia TIDAK dibaca langsung oleh mesin penjenjangan. Ia hanya nilai bawaan yang dipakai
// membentuk Kebijakan, dan Kebijakan itu DIPASOK DARI LUAR — sehingga menggantinya kelak
// dengan baris master `F-4` tidak menyentuh satu baris pun aturan di berkas ini.
//
// Kenapa belum menjadi master hari ini: tidak ada tabel yang memuatnya. Di sistem lama
// pita dipilih dengan MEMBANDINGKAN NAMA ORANG — `Activity/SetEmailKomite-Act.xml`
// step 10, 12, dan 14 mencocokkan `UserTeknis` dengan tiga nama tertentu, lalu memaksa
// nilai pembandingnya melewati ambang. `D-52` mencabut cara itu dan menggantinya dengan
// pita yang diturunkan dari nilai klaim, sekaligus menghapus tiga dari 24 Operator ID
// yang di-hardcode.
const BatasPitaNonMBUBawaan = 100_000_000

// Pita yang berlaku pada lini Non-MBU, sesuai isi kolom TYPE_KOMITE di master.
const (
	PitaBawah = "1"
	PitaAtas  = "2"
)

// KebijakanPita menyatakan bagaimana sebuah lini memilih pita nilai sebelum jenjang
// diakumulasi.
type KebijakanPita struct {
	// Batas adalah nilai TERTINGGI yang masih masuk pita bawah. Nilai tepat di batas
	// masuk pita bawah; satu satuan di atasnya sudah masuk pita atas.
	Batas uang.Uang

	Bawah string
	Atas  string
}

// Pilih mengembalikan pita untuk sebuah nilai klaim.
func (k KebijakanPita) Pilih(nilai uang.Uang) string {
	if nilai <= k.Batas {
		return k.Bawah
	}
	return k.Atas
}

// Kebijakan mengumpulkan aturan pita untuk lini yang memakainya.
//
// Lini yang TIDAK ada di dalamnya tidak mengenal pita sama sekali, dan jenjangnya
// diakumulasi langsung atas seluruh baris lini tersebut.
type Kebijakan struct {
	Pita map[Lini]KebijakanPita
}

// KebijakanBawaan mengembalikan kebijakan yang berlaku hari ini: HANYA Non-MBU yang
// memakai pita.
//
// # Kenapa hanya Non-MBU, dan kenapa ini tidak boleh diseragamkan
//
// `D-70` membatasinya setelah diperiksa terhadap isi master. Dihitung dari
// `Database/emailkomite.csv`, memberlakukan pemilihan pita ke semua lini akan
// menghasilkan:
//
//	PA Rp 5.000.000        1 penyetuju  →  0 penyetuju   klaim mandek
//	PA Rp 75.000.000       3 penyetuju  →  2 penyetuju
//	PA Rp 150.000.000      4 penyetuju  →  2 penyetuju
//	Travel Rp 150.000.000  3 penyetuju  →  0 penyetuju   klaim mandek
//
// Sebabnya terlihat di data: pada PA, kolom TYPE_KOMITE berselang-seling 2 · 1 · 1 · 2
// menaiki tangga — ia membedakan PA reguler dari PA TKI, bukan pita nilai.
func KebijakanBawaan() Kebijakan {
	return Kebijakan{
		Pita: map[Lini]KebijakanPita{
			LiniNonMBU: {
				Batas: uang.DariRupiah(BatasPitaNonMBUBawaan),
				Bawah: PitaBawah,
				Atas:  PitaAtas,
			},
		},
	}
}

// Penyetuju adalah satu orang yang harus menyetujui, beserta tempatnya dalam antrean.
type Penyetuju struct {
	// Urutan adalah posisi menyetujui, 1 sampai jumlah penyetuju. Ia SELALU berurutan
	// tanpa lompatan, berbeda dari Jenjang.
	Urutan int

	// Jenjang adalah DEGREE dari master. Ia dapat berulang antar baris dan dapat
	// melompat; ia bukan penomoran antrean.
	Jenjang int

	Nama       string
	OperatorID string

	// BatasBawah adalah ambang yang membuat orang ini ikut. Dikirim ke layar supaya
	// pengguna dapat melihat ALASAN seseorang masuk daftar, bukan hanya hasilnya.
	BatasBawah uang.Uang

	// SedangAbsen dibawa apa adanya dari master. Ia TIDAK menyaring siapa pun — lihat
	// catatan pada Ambang.SedangAbsen.
	SedangAbsen bool

	// IDAmbang menunjuk baris master asalnya, supaya hasil hitungan dapat ditelusuri
	// balik ke datanya saat ada yang meragukannya.
	IDAmbang string
}

// Penjenjangan adalah hasil perhitungan untuk satu nilai klaim pada satu lini.
type Penjenjangan struct {
	Nilai uang.Uang
	Lini  Lini

	// Pita terisi HANYA untuk lini yang memakainya. Kosong berarti lini ini memang
	// tidak mengenal pita — bukan berarti pitanya gagal dihitung.
	Pita string

	// BerpitaNilai membedakan kedua keadaan di atas secara tegas, supaya layar tidak
	// perlu menebak arti Pita yang kosong.
	BerpitaNilai bool

	// Penyetuju berurutan sesuai antrean menyetujui. Panjangnya adalah jumlah jenjang.
	Penyetuju []Penyetuju

	// UrutanTidakPasti menyala bila ada dua penyetuju atau lebih ber-DEGREE SAMA.
	//
	// Kenapa ini dilaporkan: kueri sistem lama mengurutkan dengan `ORDER BY DEGREE`
	// saja, sehingga saat DEGREE seri, urutannya ditentukan basis data dan dapat
	// berubah antar eksekusi. Keadaan itu BENAR-BENAR ADA pada master yang berlaku —
	// Non-MBU pita 1 memiliki dua baris ber-DEGREE 1 (ID 7 dan ID 1).
	//
	// Modul ini mengurutkan secara pasti (lihat Tentukan), dan menyalakan penanda ini
	// supaya perbedaan urutan terhadap Pega pada kasus seri tidak terbaca sebagai cacat
	// saat uji kesetaraan.
	UrutanTidakPasti bool
}

// JumlahJenjang adalah banyaknya persetujuan yang dibutuhkan.
func (p Penjenjangan) JumlahJenjang() int { return len(p.Penyetuju) }

// TanpaPenyetuju berarti tidak satu pun jenjang cocok untuk nilai ini.
//
// Ia dilaporkan sebagai KEADAAN, bukan galat, karena pemanggil yang berbeda menanganinya
// berbeda: layar simulasi menampilkannya sebagai peringatan yang mencolok, sedangkan
// alur klaim kelak harus menolak melanjutkan. Menjadikannya galat akan memaksa layar
// simulasi menampilkan kegagalan padahal yang terjadi adalah temuan.
func (p Penjenjangan) TanpaPenyetuju() bool { return len(p.Penyetuju) == 0 }

// Tentukan menghitung siapa saja yang harus menyetujui sebuah nilai klaim.
//
// # Langkahnya, sama persis dengan sistem lama
//
//  1. Ambil hanya baris yang merupakan jenjang persetujuan — STS_AKTIF dan STS_ADJ
//     keduanya menyala.
//  2. Ambil hanya baris lini yang diminta.
//  3. Bila lini memakai pita: pilih pitanya dari nilai klaim, lalu saring
//     TYPE_KOMITE = pita itu. Akumulasi kemudian berjalan DI DALAM pita itu saja dan
//     tidak pernah menyeberang.
//  4. Ambil hanya baris yang BatasBawah <= nilai klaim. Inilah akumulasinya.
//  5. Urutkan menurut Jenjang.
//
// LIMIT_TOP tidak dipakai pada satu langkah pun. Pemakaiannya hanya di integritas.go.
//
// # Kenapa pengurutannya lebih pasti daripada Pega, dan kenapa itu disengaja
//
// Kueri lama memakai `ORDER BY DEGREE` saja. Pada master yang berlaku, Non-MBU pita 1
// memiliki DUA baris ber-DEGREE 1, sehingga urutan keduanya diserahkan kepada basis data
// dan dapat berbeda antar eksekusi.
//
// Di sini seri dipecahkan secara pasti: BatasBawah lebih kecil lebih dulu — yang secara
// bisnis memang masuk akal, karena jenjang berambang lebih rendah adalah yang menyetujui
// lebih awal — lalu ID sebagai pemecah terakhir supaya hasilnya tidak pernah bergantung
// pada urutan baris yang datang dari penyimpanan.
//
// Ini TIDAK mengubah SIAPA yang menyetujui, hanya URUTANNYA saat seri, dan setiap kali
// hal itu terjadi ia dilaporkan lewat UrutanTidakPasti.
func Tentukan(nilai uang.Uang, lini Lini, ambang []Ambang, kebijakan Kebijakan) (Penjenjangan, error) {
	lini = lini.Bersih()

	if err := GalatValidasiBaru(PeriksaMasukan(nilai, lini)); err != nil {
		return Penjenjangan{}, err
	}

	hasil := Penjenjangan{Nilai: nilai, Lini: lini}

	// Langkah 1 dan 2 — jenjang persetujuan pada lini yang diminta.
	adaLini := false
	sejalur := make([]Ambang, 0, len(ambang))
	for _, a := range ambang {
		a = a.Bersih()
		if a.Lini != lini {
			continue
		}
		adaLini = true
		if !a.JenjangPersetujuan() {
			continue
		}
		sejalur = append(sejalur, a)
	}
	if !adaLini {
		// Dibedakan dari "ada tetapi tidak ada yang cocok": yang ini salah ketik atau
		// lini baru yang belum diisi, dan pemanggil pantas diberi tahu bedanya.
		return Penjenjangan{}, ErrLiniTidakDikenal
	}

	// Langkah 3 — pita, hanya untuk lini yang memakainya.
	if aturan, berpita := kebijakan.Pita[lini]; berpita {
		hasil.BerpitaNilai = true
		hasil.Pita = aturan.Pilih(nilai)

		disaring := sejalur[:0:0]
		for _, a := range sejalur {
			if a.JenisKomite == hasil.Pita {
				disaring = append(disaring, a)
			}
		}
		sejalur = disaring
	}

	// Langkah 4 — akumulasi menurut batas bawah.
	cocok := sejalur[:0:0]
	for _, a := range sejalur {
		if a.BatasBawah <= nilai {
			cocok = append(cocok, a)
		}
	}

	// Langkah 5 — urutan yang pasti.
	sort.SliceStable(cocok, func(i, j int) bool {
		if cocok[i].Jenjang != cocok[j].Jenjang {
			return cocok[i].Jenjang < cocok[j].Jenjang
		}
		if cocok[i].BatasBawah != cocok[j].BatasBawah {
			return cocok[i].BatasBawah < cocok[j].BatasBawah
		}
		return cocok[i].ID < cocok[j].ID
	})

	hasil.Penyetuju = make([]Penyetuju, 0, len(cocok))
	terpakai := map[int]bool{}
	for i, a := range cocok {
		if terpakai[a.Jenjang] {
			hasil.UrutanTidakPasti = true
		}
		terpakai[a.Jenjang] = true

		hasil.Penyetuju = append(hasil.Penyetuju, Penyetuju{
			Urutan:      i + 1,
			Jenjang:     a.Jenjang,
			Nama:        a.Nama,
			OperatorID:  a.OperatorID,
			BatasBawah:  a.BatasBawah,
			SedangAbsen: a.SedangAbsen,
			IDAmbang:    a.ID,
		})
	}

	return hasil, nil
}

// PeriksaMasukan mengumpulkan SELURUH pelanggaran pada masukan, bukan berhenti pada yang
// pertama.
func PeriksaMasukan(nilai uang.Uang, lini Lini) []Pelanggaran {
	var pelanggaran []Pelanggaran

	if nilai < 0 {
		pelanggaran = append(pelanggaran, Pelanggaran{
			Field: FieldNilai,
			Pesan: "Nilai klaim tidak boleh kurang dari nol.",
		})
	}
	if lini.Bersih() == "" {
		pelanggaran = append(pelanggaran, Pelanggaran{
			Field: FieldLini,
			Pesan: "Lini bisnis wajib dipilih.",
		})
	}

	return pelanggaran
}

// DaftarLini mengembalikan seluruh lini yang punya jenjang persetujuan di master,
// terurut.
//
// Dipakai layar untuk mengisi pilihan lini. Daftarnya datang DARI DATA, tidak pernah
// dari daftar tetap di dalam kode (`D-15`) — lini yang ditambahkan ke master langsung
// muncul tanpa rilis ulang.
func DaftarLini(ambang []Ambang) []Lini {
	terlihat := map[Lini]bool{}
	for _, a := range ambang {
		a = a.Bersih()
		if a.JenjangPersetujuan() {
			terlihat[a.Lini] = true
		}
	}

	hasil := make([]Lini, 0, len(terlihat))
	for l := range terlihat {
		hasil = append(hasil, l)
	}
	sort.Slice(hasil, func(i, j int) bool { return hasil[i] < hasil[j] })
	return hasil
}
