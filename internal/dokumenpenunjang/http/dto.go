package dokumenpenunjanghttp

import (
	"time"

	"claim-pnc/internal/dokumenpenunjang"
)

// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field berbahasa Indonesia karena ia KONTRAK yang dibaca frontend — salah satu dari
// lima pengecualian `D-80`.

// dokumenDTO adalah satu dokumen penunjang sebagaimana dilihat layar.
type dokumenDTO struct {
	// ID adalah `IMAGEID`, kunci yang diterbitkan layanan penyimpanan.
	ID string `json:"id"`

	// NamaBerkas adalah nama TERSIMPAN — sudah dibersihkan dari tanda baca, termasuk titik
	// ekstensinya. Ia karena itu tidak sama dengan nama yang diketik pengguna.
	//
	// Perbedaan itu ditiru dari Pega (`P-5`) dan bukan cacat kami; lihat
	// `dokumenpenunjang.BersihkanNamaBerkas`.
	NamaBerkas string `json:"nama_berkas"`

	// JenisDokumen adalah `TYPEIMAGE`; kosong bila tidak dipilih.
	JenisDokumen string `json:"jenis_dokumen"`

	// URL adalah alamat berkasnya. KOSONG bila belum terisi di metadata — layanan
	// mengisinya beberapa saat setelah unggah, sehingga kosong bukan berarti gagal.
	URL string `json:"url"`

	// TanggalUnggah dan BerlakuSampai sudah diformat untuk ditampilkan, bukan waktu mentah.
	TanggalUnggah string `json:"tanggal_unggah"`
	BerlakuSampai string `json:"berlaku_sampai"`

	// Kedaluwarsa dihitung DI SERVER, bukan di peramban.
	//
	// Menghitungnya di peramban menyandarkan hasilnya pada jam komputer pengguna — yang
	// dapat salah berjam-jam, dan membuat dua orang melihat status berbeda untuk dokumen
	// yang sama.
	Kedaluwarsa bool `json:"kedaluwarsa"`
}

// ListResponse membungkus daftar dokumen.
//
// Dibungkus `data`, bukan larik telanjang, mengikuti amplop `09-API-STRATEGY.md` §3 —
// sehingga penambahan `meta` kelak tidak merusak klien.
type ListResponse struct {
	Data []dokumenDTO `json:"data"`
}

// ItemResponse membungkus satu dokumen.
type ItemResponse struct {
	Data dokumenDTO `json:"data"`
}

// dariDokumen mengubah satu dokumen menjadi bentuk yang dibaca layar.
func dariDokumen(
	d dokumenpenunjang.Document,
	location *time.Location,
	sekarang time.Time,
) dokumenDTO {
	return dokumenDTO{
		ID:            d.ImageID,
		NamaBerkas:    d.FileName,
		JenisDokumen:  d.DocumentType,
		URL:           d.URL,
		TanggalUnggah: formatWaktu(d.UploadedAt, location),
		BerlakuSampai: formatWaktu(d.ExpiresAt, location),
		Kedaluwarsa:   d.Kedaluwarsa(sekarang),
	}
}

// dariDaftar mengubah sekumpulan dokumen.
//
// Mengembalikan larik KOSONG, bukan nil, supaya JSON-nya `[]` dan bukan `null`. Layar yang
// menerima `null` harus menjaganya sendiri, dan satu layar yang lupa akan galat saat
// klaimnya belum punya dokumen — keadaan yang justru paling sering.
func dariDaftar(
	daftar []dokumenpenunjang.Document,
	location *time.Location,
	sekarang time.Time,
) []dokumenDTO {
	hasil := make([]dokumenDTO, 0, len(daftar))
	for _, d := range daftar {
		hasil = append(hasil, dariDokumen(d, location, sekarang))
	}
	return hasil
}

// formatWaktu menampilkan waktu dalam zona tampilan.
//
// Waktu yang tidak ada menjadi teks kosong, bukan "01/01/0001" — `time.Time` nol yang
// terformat apa adanya terbaca sebagai tanggal sungguhan oleh pengguna.
func formatWaktu(t *time.Time, location *time.Location) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.In(location).Format("02/01/2006 15:04")
}
