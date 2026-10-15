// Package masterreashttp adalah lapisan transport modul Master Reas.
//
// Namanya mengikuti `D-81`: folder modul memakai nama modul bisnis apa adanya
// ("Master Reas"), dan paket transportnya menambahkan akhiran `http` tanpa tanda hubung
// karena Go tidak mengizinkannya. Pola yang sama dipakai `masterloginhttp`,
// `masterspareparthttp`, dan `masterbengkelhttp`.
//
// Lapisan Transport — ia boleh tahu Domain, dan dilarang tahu SQL maupun nama tabel.
package masterreashttp

import "claim-pnc/internal/masterreas"

// MemberDTO adalah satu member reasuransi sebagaimana dilihat klien.
//
// # Kenapa nama field JSON-nya bahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Nama tipe, field Go, dan variabel di modul ini
// seluruhnya bahasa Inggris; yang tetap Indonesia hanyalah yang dipakai di luar kode — dan
// nama field JSON termasuk di dalamnya, karena mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama.
//
// # Namanya mengikuti ISI kolom, bukan caption layar
//
// Berbeda dari modul master lain, di sini tidak ada caption untuk diikuti: section grid
// `BrowseListMemberReas` tidak ada di export (`R-16`), sehingga label kolom layar lamanya
// tidak diketahui. Yang dipakai karena itu adalah nama kolomnya sendiri — bukan alias
// klipboard Pega, yang justru menyesatkan (`email as "City"`).
type MemberDTO struct {
	// ReinsurerID adalah kolom REINSURERID.
	ReinsurerID string `json:"kode_reas"`

	// ReinsurerName adalah kolom REINSURERNAME.
	ReinsurerName string `json:"nama_reas"`

	// Login adalah kolom LOGIN — identitas mitra reasuransi saat masuk.
	Login string `json:"login"`

	// Email adalah kolom EMAIL — tujuan pemberitahuan PLA/DLA.
	Email string `json:"email"`

	// Country adalah kolom COUNTRY.
	Country string `json:"negara"`

	// Type adalah kolom TYPE — karakter pertama nomor dokumen PLA/DLA yang dilayani baris
	// ini. Artinya dalam bahasa bisnis belum tercatat di mana pun; lihat masterreas.Member.
	Type string `json:"tipe"`

	// IsFallback menyatakan baris ini adalah baris CADANGAN (`TYPE = '1'`).
	//
	// Ia TURUNAN, bukan kolom — dihitung server supaya layar tidak perlu mengetahui bahwa
	// `'1'` punya arti khusus. Menaruh pengetahuan itu di layar berarti dua tempat harus
	// ikut berubah bila kelak artinya berubah.
	IsFallback bool `json:"cadangan"`
}

// ListResponse adalah jawaban daftar.
//
// Alias portal ikut dikirim, sama seperti modul master lain: satu aplikasi melayani empat
// badan hukum dengan basis data terpisah, dan layar menyebut terang-terangan data siapa
// yang sedang ditampilkan (`ADR-0030`, `R-20`).
type ListResponse struct {
	Member []MemberDTO `json:"member_reas"`
	Portal string      `json:"portal"`
}

// SaveRequest adalah badan permintaan ubah surel.
//
// # Kunci ikut di BADAN, bukan di jalur URL
//
// Kunci alami tabel ini TIGA kolom — `REINSURERID` + `REINSURERNAME` + `TYPE` — dan ketiganya
// dibutuhkan untuk menunjuk satu baris (`Database/UPDATEREAS.prc`). Memaksakannya ke jalur
// URL menuntut tiga segmen yang harus dikodekan, dan salah satunya — `TYPE` — boleh kosong,
// sehingga jalurnya akan memuat segmen kosong yang mudah tertelan router.
//
// Karena itu `PUT /master/reas` tidak berparameter jalur: seluruh kuncinya dikirim di sini.
//
// # Hanya `email` yang dapat DIUBAH
//
// Ketiga kolom kunci dikirim untuk MENUNJUK baris, bukan untuk diubah. `login` dan `negara`
// tidak dapat dikirim sama sekali — `UPDATEREAS` pada baris yang sudah ada tidak pernah
// menyentuh keduanya, dan `LOGIN` menentukan klaim mana yang dilihat seorang mitra.
type SaveRequest struct {
	ReinsurerID   string `json:"kode_reas"`
	ReinsurerName string `json:"nama_reas"`
	Type          string `json:"tipe"`
	Email         string `json:"email"`
}

// ViolationDTO adalah satu pelanggaran isian.
type ViolationDTO struct {
	Field   string `json:"isian"`
	Message string `json:"pesan"`
}

// SingleResponse adalah jawaban satu baris — dipakai jalur ubah.
type SingleResponse struct {
	Member MemberDTO `json:"member_reas"`
	Portal string    `json:"portal"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — supaya klien tidak menghadapi dua
// bentuk galat yang berbeda. Lihat catatan pada berkas errors.go.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// toKey dan toInput memisahkan bagian PENUNJUK baris dari bagian yang DIUBAH.
//
// Pemisahannya bukan kerapian: lapisan aplikasi menerima keduanya sebagai argumen yang
// berbeda, sehingga tidak ada jalan bagi nilai kunci untuk tanpa sengaja ikut tersimpan.
func (r SaveRequest) toKey() masterreas.Key {
	return masterreas.Key{
		ReinsurerID:   r.ReinsurerID,
		ReinsurerName: r.ReinsurerName,
		Type:          r.Type,
	}
}

func (r SaveRequest) toInput() masterreas.Input {
	return masterreas.Input{Email: r.Email}
}

// toDTO memetakan satu baris domain menjadi bentuk yang dikirim.
func toDTO(one masterreas.Member) MemberDTO {
	return MemberDTO{
		ReinsurerID:   one.ReinsurerID,
		ReinsurerName: one.ReinsurerName,
		Login:         one.Login,
		Email:         one.Email,
		Country:       one.Country,
		Type:          one.Type,
		IsFallback:    one.IsFallback(),
	}
}

// toListDTO memetakan seluruh baris.
//
// Senarai kosong dikembalikan sebagai `[]`, bukan `null`: klien yang memetakan hasilnya
// tanpa memeriksa nil akan gagal pada `null`, dan daftar yang memang kosong adalah keadaan
// yang wajar di sini — entitas yang belum pernah mengirim PLA/DLA tidak punya satu pun
// baris.
func toListDTO(list []masterreas.Member) []MemberDTO {
	result := make([]MemberDTO, 0, len(list))
	for _, one := range list {
		result = append(result, toDTO(one))
	}
	return result
}
