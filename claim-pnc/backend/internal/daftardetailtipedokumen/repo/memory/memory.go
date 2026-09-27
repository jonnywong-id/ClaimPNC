// Package memory adalah pengisi kedua seam modul Daftar Detail Tipe Dokumen yang hidup
// di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Ia juga yang
// memungkinkan aplikasi dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul
// auth, portal, dan seluruh modul master yang sudah ada.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara ID
// diterbitkan, cara daftar bisnis DIGANTI SELURUHNYA saat disimpan, dan cara keterangan
// master DIISI LEWAT PENCARIAN alih-alih disimpan di baris — kalau tidak, uji yang lulus
// di sini tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/daftardetailtipedokumen"
)

// siteCode adalah kode situs tiruan, bagian pertama setiap ID.
//
// Di Oracle ia dibaca dari POOLDATA.M_SITE_DATABASE dan BERBEDA di tiap entitas
// (`Database/PEGA_LST_DET_TYPE_DOC.prc:11`). Di sini ia tetap, karena pengembangan tanpa
// basis data tidak punya tabel itu — yang ditiru adalah BENTUK ID-nya, bukan nilainya.
const siteCode = "10"

// Repo menyimpan detail tipe dokumen satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// `ADR-0030` dan `R-20`.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus, dan
	// penerbitan ID harus berjalan satu per satu — persis seperti transaksi pada adapter
	// SQL.
	mutex     sync.Mutex
	rows      map[string]daftardetailtipedokumen.DetailType
	sequence  int64
	failure   error
	reference *ReferenceRepo
}

// NewRepo membentuk repo berisi baris yang diberikan.
//
// Nomor urut dimulai dari nomor tertinggi yang sudah terpakai, supaya penambahan pertama
// menghasilkan ID berikutnya yang wajar dan bukan yang bentrok.
func NewRepo(rows ...daftardetailtipedokumen.DetailType) *Repo {
	r := &Repo{rows: make(map[string]daftardetailtipedokumen.DetailType, len(rows))}
	for _, row := range rows {
		clean := cleanDetail(row)
		r.rows[clean.ID] = clean
		if n := sequenceFromID(clean.ID); n > r.sequence {
			r.sequence = n
		}
	}
	return r
}

// UseReferences menyambungkan repo ini ke daftar master, supaya keterangan yang di Oracle
// datang dari join view ikut terisi di sini.
//
// Tanpa ini, baris yang baru disimpan akan tampil TANPA nama tipe dokumen dan TANPA nama
// bisnis — dan perbedaan itu hanya muncul saat pengembangan tanpa basis data, yakni
// tempat yang paling mudah disalahartikan sebagai cacat aplikasi.
func (r *Repo) UseReferences(reference *ReferenceRepo) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.reference = reference
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh rincian TANPA daftar bisnisnya, terurut menurut ID.
func (r *Repo) List(_ context.Context) ([]daftardetailtipedokumen.DetailType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]daftardetailtipedokumen.DetailType, 0, len(r.rows))
	for _, row := range r.rows {
		// Daftar bisnis sengaja dibuang di sini, meniru kueri daftar yang memang tidak
		// membacanya. Membiarkannya ikut akan membuat uji layar lulus di sini lalu gagal
		// terhadap Oracle, karena di sana daftarnya benar-benar kosong.
		row.Businesses = nil
		result = append(result, r.withReferences(row))
	}

	// Urutannya mengikuti `BrowseVLstDetTypeDoc_RD-RD.xml`: ID menaik. Perbandingan teks,
	// bukan angka — sama seperti ORDER BY pada kolom bertipe teks.
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu rincian lengkap dengan daftar bisnisnya.
func (r *Repo) Get(_ context.Context, id string) (daftardetailtipedokumen.DetailType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftardetailtipedokumen.DetailType{}, r.failure
	}

	row, exists := r.rows[strings.TrimSpace(id)]
	if !exists {
		return daftardetailtipedokumen.DetailType{}, daftardetailtipedokumen.ErrNotFound
	}
	return r.withReferences(copyDetail(row)), nil
}

// InsertNew menerbitkan ID lalu menyimpan barisnya beserta daftar bisnisnya.
//
// Tidak ada pemeriksaan bahwa kode rujukannya ada di master, dan itu bukan kelalaian:
// layar Pega pun tidak memeriksanya — keempat isiannya autocomplete yang tetap menerima
// ketikan di luar daftar. Adapter memori yang lebih ketat daripada adapter SQL akan
// membuat uji lulus di sini lalu gagal di sana.
//
// Editor diterima tetapi tidak disimpan: DetailType memang tidak memuat jejak simpan,
// karena grid maupun form layar lama tidak menampilkannya. Di Oracle keduanya tetap
// ditulis ke kolom TGL_EDIT dan USER_EDIT.
func (r *Repo) InsertNew(
	_ context.Context,
	input daftardetailtipedokumen.Input,
	_ daftardetailtipedokumen.Editor,
) (daftardetailtipedokumen.DetailType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftardetailtipedokumen.DetailType{}, r.failure
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk kasus
	// tabel yang sudah memuat ID berbentuk lain, supaya baris baru tidak menabraknya.
	var id string
	for {
		r.sequence++
		id = daftardetailtipedokumen.FormatID(siteCode, r.sequence)
		if _, taken := r.rows[id]; !taken {
			break
		}
	}

	row := detailFrom(id, input)
	r.rows[id] = row
	return r.withReferences(copyDetail(row)), nil
}

// Update mengganti isi satu rincian beserta SELURUH daftar bisnisnya.
func (r *Repo) Update(
	_ context.Context,
	id string,
	input daftardetailtipedokumen.Input,
	_ daftardetailtipedokumen.Editor,
) (daftardetailtipedokumen.DetailType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftardetailtipedokumen.DetailType{}, r.failure
	}

	key := strings.TrimSpace(id)
	if _, exists := r.rows[key]; !exists {
		return daftardetailtipedokumen.DetailType{}, daftardetailtipedokumen.ErrNotFound
	}

	// Baris lama DIBUANG seluruhnya, termasuk daftar bisnisnya, lalu disusun ulang dari
	// isian yang dikirim. Inilah yang ditiru adapter SQL dengan menghapus lalu menyisip
	// ulang baris bisnis di dalam satu transaksi.
	row := detailFrom(key, input)
	r.rows[key] = row
	return r.withReferences(copyDetail(row)), nil
}

// withReferences mengisi keterangan yang di Oracle datang dari join view.
//
// # Hanya DUA yang diisi di sini, dan pembedaannya penting
//
// `DocumentTypeName` dan nama bisnis pada setiap baris anak memang hasil join —
// keduanya tidak pernah tersimpan. Kedua keterangan lain, `CauseOfLossDescription` dan
// `ObjectDocumentDescription`, **tersimpan di barisnya sendiri** dan karena itu TIDAK
// disentuh: menimpanya dari master akan membuang keterangan yang diketik bebas, yaitu
// tepat isian yang `pyAllowFreeFormInput=true` izinkan di layar lama.
//
// Pemanggilnya sudah memegang mutex; metode ini TIDAK mengambilnya sendiri.
//
// Kode yang tidak ada di master dibiarkan berketerangan KOSONG, bukan diganti tanda apa
// pun — itulah yang dilakukan LEFT JOIN pada adapter SQL, dan baris seperti itu memang
// harus tetap tampil supaya petugas dapat memperbaikinya.
func (r *Repo) withReferences(row daftardetailtipedokumen.DetailType) daftardetailtipedokumen.DetailType {
	if r.reference == nil {
		return row
	}
	row.DocumentTypeName = r.reference.documentTypeName(row.DocumentTypeID)
	for index := range row.Businesses {
		row.Businesses[index].BusinessName = r.reference.businessName(row.Businesses[index].BusinessID)
	}
	return row
}

// detailFrom menyusun baris tersimpan dari isian yang dikirim layar.
//
// Kedua keterangan yang TERSIMPAN ikut disalin apa adanya. Yang TIDAK diisi di sini
// hanyalah `DocumentTypeName` dan nama bisnis pada baris anak — keduanya hasil join, dan
// diisikan withReferences saat dibaca.
func detailFrom(id string, input daftardetailtipedokumen.Input) daftardetailtipedokumen.DetailType {
	row := daftardetailtipedokumen.DetailType{
		ID:                        id,
		DocumentTypeID:            input.DocumentTypeID,
		Detail:                    input.Detail,
		InsuredStatus:             input.InsuredStatus,
		CauseOfLossID:             input.CauseOfLossID,
		CauseOfLossDescription:    input.CauseOfLossDescription,
		ObjectDocumentID:          input.ObjectDocumentID,
		ObjectDocumentDescription: input.ObjectDocumentDescription,
		Risk:                      input.Risk,
	}
	for _, business := range input.Businesses {
		row.Businesses = append(row.Businesses, daftardetailtipedokumen.BusinessRule{
			BusinessID:  business.BusinessID,
			Mandatory:   business.Mandatory,
			MinDocument: business.MinDocument,
		})
	}
	return row
}

// cleanDetail memangkas isian baris yang diberikan sebagai isi awal.
func cleanDetail(row daftardetailtipedokumen.DetailType) daftardetailtipedokumen.DetailType {
	clean := daftardetailtipedokumen.DetailType{
		ID:                        strings.TrimSpace(row.ID),
		DocumentTypeID:            strings.TrimSpace(row.DocumentTypeID),
		DocumentTypeName:          strings.TrimSpace(row.DocumentTypeName),
		Detail:                    strings.TrimSpace(row.Detail),
		InsuredStatus:             strings.TrimSpace(row.InsuredStatus),
		CauseOfLossID:             strings.TrimSpace(row.CauseOfLossID),
		CauseOfLossDescription:    strings.TrimSpace(row.CauseOfLossDescription),
		ObjectDocumentID:          strings.TrimSpace(row.ObjectDocumentID),
		ObjectDocumentDescription: strings.TrimSpace(row.ObjectDocumentDescription),
		Risk:                      strings.TrimSpace(row.Risk),
	}
	for _, business := range row.Businesses {
		clean.Businesses = append(clean.Businesses, daftardetailtipedokumen.BusinessRule{
			BusinessID:   strings.TrimSpace(business.BusinessID),
			BusinessName: strings.TrimSpace(business.BusinessName),
			Mandatory:    business.Mandatory,
			MinDocument:  business.MinDocument,
		})
	}
	return clean
}

// copyDetail menyalin baris beserta senarai bisnisnya.
//
// Salinan senarainya WAJIB, bukan kehati-hatian berlebihan: tanpa itu pemanggil memegang
// senarai yang sama dengan yang tersimpan, dan mengubah satu elemennya akan mengubah isi
// repo tanpa melewati Update sama sekali. Adapter SQL tidak punya kelemahan itu karena
// ia selalu menyusun senarai baru dari hasil kueri.
func copyDetail(row daftardetailtipedokumen.DetailType) daftardetailtipedokumen.DetailType {
	clone := row
	if row.Businesses != nil {
		clone.Businesses = append([]daftardetailtipedokumen.BusinessRule(nil), row.Businesses...)
	}
	return clone
}

// sequenceFromID membaca kembali nomor urut dari sebuah ID, supaya penambahan berikutnya
// melanjutkan dan tidak mengulang nomor yang sudah dipakai.
//
// Bagian kode situs di depannya dilewati. ID yang bukan berbentuk itu dijawab 0 dan
// karena itu tidak mempengaruhi nomor berikutnya — baris lama dapat memuat apa saja, dan
// melewatinya lebih baik daripada salah menafsirkannya.
func sequenceFromID(id string) int64 {
	trimmed := strings.TrimSpace(id)
	if !strings.HasPrefix(trimmed, siteCode) {
		return 0
	}
	n, err := strconv.ParseInt(strings.TrimPrefix(trimmed, siteCode), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// ReferenceRepo menyimpan keempat daftar master di memori.
//
// Terpisah dari Repo karena ia mengisi seam yang berbeda, dan seam itu BACA-SAJA.
type ReferenceRepo struct {
	mutex           sync.Mutex
	documentTypes   []daftardetailtipedokumen.DocumentTypeOption
	causesOfLoss    []daftardetailtipedokumen.CauseOfLossOption
	objectDocuments []daftardetailtipedokumen.ObjectDocumentOption
	businesses      []daftardetailtipedokumen.Business
	failure         error
}

// NewReferenceRepo membentuk pembaca keempat master rujukan.
func NewReferenceRepo(
	documentTypes []daftardetailtipedokumen.DocumentTypeOption,
	causesOfLoss []daftardetailtipedokumen.CauseOfLossOption,
	objectDocuments []daftardetailtipedokumen.ObjectDocumentOption,
	businesses []daftardetailtipedokumen.Business,
) *ReferenceRepo {
	return &ReferenceRepo{
		documentTypes:   append([]daftardetailtipedokumen.DocumentTypeOption(nil), documentTypes...),
		causesOfLoss:    append([]daftardetailtipedokumen.CauseOfLossOption(nil), causesOfLoss...),
		objectDocuments: append([]daftardetailtipedokumen.ObjectDocumentOption(nil), objectDocuments...),
		businesses:      append([]daftardetailtipedokumen.Business(nil), businesses...),
	}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
//
// Jalur itu penting dan bukan sekadar kelengkapan: kegagalan membaca daftar master TIDAK
// BOLEH menghalangi penyimpanan. Keempat kodenya boleh diketik sendiri, sehingga yang
// hilang saat daftarnya gagal dimuat hanyalah kenyamanan memilih — perlakuan yang sama
// dengan daftar bisnis pada modul Master COL Simas Online.
func (r *ReferenceRepo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// ListDocumentTypes mengembalikan pilihan ID Tipe Dokumen, terurut menurut namanya.
func (r *ReferenceRepo) ListDocumentTypes(_ context.Context) ([]daftardetailtipedokumen.DocumentTypeOption, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	result := append([]daftardetailtipedokumen.DocumentTypeOption(nil), r.documentTypes...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// ListCausesOfLoss mengembalikan pilihan Dokumen kolom ID, terurut menurut keterangannya.
func (r *ReferenceRepo) ListCausesOfLoss(_ context.Context) ([]daftardetailtipedokumen.CauseOfLossOption, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	result := append([]daftardetailtipedokumen.CauseOfLossOption(nil), r.causesOfLoss...)
	sort.Slice(result, func(i, j int) bool { return result[i].Description < result[j].Description })
	return result, nil
}

// ListObjectDocuments mengembalikan pilihan Objek Dokumen, terurut menurut keterangannya.
func (r *ReferenceRepo) ListObjectDocuments(_ context.Context) ([]daftardetailtipedokumen.ObjectDocumentOption, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	result := append([]daftardetailtipedokumen.ObjectDocumentOption(nil), r.objectDocuments...)
	sort.Slice(result, func(i, j int) bool { return result[i].Description < result[j].Description })
	return result, nil
}

// ListBusinesses mengembalikan pilihan ID Bisnis, terurut menurut namanya.
func (r *ReferenceRepo) ListBusinesses(_ context.Context) ([]daftardetailtipedokumen.Business, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}
	result := append([]daftardetailtipedokumen.Business(nil), r.businesses...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// Kedua pencarian di bawah meniru join pada view, dan sengaja TIDAK memeriksa r.failure:
// keterangan yang tidak dapat dicari cukup kosong, sama seperti LEFT JOIN yang
// menghasilkan NULL. Menggagalkan pembacaan daftar karena master rujukannya bermasalah
// akan menyembunyikan baris yang justru perlu diperbaiki petugas.
//
// HANYA DUA, bukan empat: keterangan penyebab kerugian dan keterangan objek dokumen
// TERSIMPAN di barisnya sendiri dan tidak pernah dicari dari master. Menambahkan
// pencarian untuk keduanya akan menimpa keterangan yang diketik bebas.

func (r *ReferenceRepo) documentTypeName(id string) string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, row := range r.documentTypes {
		if row.ID == id {
			return row.Name
		}
	}
	return ""
}

func (r *ReferenceRepo) businessName(id string) string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	for _, row := range r.businesses {
		if row.ID == id {
			return row.Name
		}
	}
	return ""
}

var (
	_ daftardetailtipedokumen.Repo          = (*Repo)(nil)
	_ daftardetailtipedokumen.ReferenceRepo = (*ReferenceRepo)(nil)
)
