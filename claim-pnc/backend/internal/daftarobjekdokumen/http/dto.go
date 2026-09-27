// Package daftarobjekdokumenhttp adalah lapisan transport modul Daftar Objek Dokumen.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `daftarobjekdokumenhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package daftarobjekdokumenhttp

import "claim-pnc/internal/daftarobjekdokumen"

// DocumentObjectDTO adalah bentuk satu baris objek dokumen yang dikirim ke peramban.
//
// Terpisah dari daftarobjekdokumen.DocumentObject supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field berbahasa Indonesia karena ia KONTRAK API, bukan nama internal (`D-80`).
type DocumentObjectDTO struct {
	// ID adalah kolom ID, berlabel "ID" di kolom pertama grid.
	ID string `json:"id"`

	// Description memetakan kolom KET_DOC_OBJ.
	//
	// # Kenapa namanya `objek_dokumen`, bukan `keterangan`
	//
	// Field JSON mengikuti LABEL LAYAR, supaya namanya mencerminkan isinya — perlakuan
	// yang ditetapkan pada modul Master Penyebab Kerugian setelah label yang dikarang
	// terbukti menyesatkan. Labelnya di Pega adalah **"Daftar Objek Dokumen"**, baik pada
	// judul kolom grid (`Section/BrowseDocumentObject-Section.xml:6715`) maupun pada isian
	// formnya (`:11801`).
	//
	// Kata "Daftar" di depan label itu milik LAYARNYA — ia daftar objek dokumen — sedangkan
	// nilai satu barisnya adalah satu objek dokumen. Karena itu fieldnya `objek_dokumen`,
	// bukan `daftar_objek_dokumen`.
	Description string `json:"objek_dokumen"`

	// OldID memetakan kolom OLD_ID: penomoran sebelum sistem ini dibangun.
	//
	// # Dikirim, tetapi TIDAK ditampilkan layar
	//
	// Grid Pega hanya memuat dua kolom — ID dan Daftar Objek Dokumen. Layar baru
	// mengikutinya apa adanya, sehingga kolom ini tidak tampil di mana pun.
	//
	// Ia tetap dikirim karena lapisan data mengikuti `BrowseVLstDocObj_RD`, yang MEMUATNYA
	// — sama seperti lapisan layar mengikuti section-nya. Keduanya artefak Pega yang
	// berbeda, dan keduanya diikuti pada tempatnya masing-masing.
	OldID string `json:"id_lama"`

	// Businesses adalah daftar bisnis yang memakai objek dokumen ini.
	//
	// Pada jawaban DAFTAR ia selalu kosong, dan itu disengaja: grid layar hanya menampilkan
	// ID dan keterangannya, sehingga menariknya untuk seluruh baris berarti satu kueri yang
	// hasilnya tidak pernah dilihat siapa pun. Layar memuatnya saat baris dibuka untuk
	// disunting.
	Businesses []BusinessDTO `json:"bisnis"`
}

// BusinessDTO adalah satu lini bisnis.
//
// Bentuknya sengaja SAMA PERSIS dengan BusinessDTO milik modul Master COL Simas Online:
// keduanya membaca POOLDATA.BUSINESS, dan frontend memakai satu tipe `Business` untuk
// keduanya.
type BusinessDTO struct {
	// ID BOLEH KOSONG pada pemetaan yang namanya diketik bebas dan tidak ada di master.
	// Layar harus menyiapkan keadaan itu; ia bukan tanda data rusak.
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ListResponse adalah jawaban GET /api/master/objek-dokumen.
type ListResponse struct {
	DocumentObject []DocumentObjectDTO `json:"objek_dokumen"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	// Angkanya adalah isi master ENTITAS YANG MENJAWAB — bukan angka yang sama untuk
	// seluruh aplikasi.
	Total int `json:"total"`

	// Portal menyebut entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia dikirim pada setiap jawaban, bukan diandaikan sama dengan yang diminta: satu
	// aplikasi melayani empat badan hukum dengan basis data terpisah (`ADR-0030`), dan
	// "data siapa ini" tidak boleh hanya ditebak dari keadaan layar (`R-20`).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban untuk satu baris: ambil, tambah, dan ubah.
type SingleResponse struct {
	DocumentObject DocumentObjectDTO `json:"objek_dokumen"`
	Portal         string            `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan dan penyuntingan.
//
// ID tidak ada di sini, dan itu disengaja. Pada penambahan ia diterbitkan penyimpanan; pada
// penyuntingan ia diambil dari jalur, bukan dari badan — dua sumber untuk satu nilai berarti
// keduanya dapat berbeda, dan yang mana yang menang menjadi pertanyaan yang tidak perlu ada.
//
// `id_lama` pun tidak diterima: ia jejak sejarah yang ditulis sebelum sistem ini ada, bukan
// isian yang boleh disunting. Layar Pega pun tidak menyediakan isiannya.
type SaveRequest struct {
	Description string `json:"objek_dokumen"`

	// Businesses berisi NAMA bisnis, bukan ID-nya.
	//
	// Nama yang dikirim karena itulah yang diketik dan dilihat petugas di layar Pega — sel
	// gridnya terikat `.Note` — dan karena nama yang diketik bebas memang tidak punya ID.
	// Server yang menyelesaikannya menjadi ID dengan mencocokkan ke master; nama yang tidak
	// cocok tetap diterima dan disimpan tanpa ID.
	Businesses []string `json:"bisnis"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul master lainnya — `{kode, pesan}` — ditambah `detail` untuk
// pelanggaran per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan
// mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(row daftarobjekdokumen.DocumentObject) DocumentObjectDTO {
	return DocumentObjectDTO{
		ID:          row.ID,
		Description: row.Description,
		OldID:       row.OldID,
		Businesses:  toBusinessListDTO(row.Businesses),
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim sebagai
// `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri, dan satu
// layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []daftarobjekdokumen.DocumentObject) []DocumentObjectDTO {
	result := make([]DocumentObjectDTO, 0, len(list))
	for _, row := range list {
		result = append(result, toDTO(row))
	}
	return result
}

func toBusinessListDTO(list []daftarobjekdokumen.Business) []BusinessDTO {
	result := make([]BusinessDTO, 0, len(list))
	for _, b := range list {
		result = append(result, BusinessDTO{ID: b.ID, Name: b.Name})
	}
	return result
}
