package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/masterpenolakan"
)

// RepoKomite menyimpan Master Penolakan Komite satu portal di memory.
//
// Ia terpisah dari Repo dengan alasan yang sama seperti di adapter SQL: tabelnya tidak
// sekerabat, dan tidak ada satu pun kolom yang menghubungkannya.
type RepoKomite struct {
	mutex   sync.Mutex
	rows    []masterpenolakan.CommitteeRejection
	failure error
}

// NewRepoKomite membentuk repo berisi baris yang diberikan.
func NewRepoKomite(rows ...masterpenolakan.CommitteeRejection) *RepoKomite {
	copied := make([]masterpenolakan.CommitteeRejection, len(rows))
	copy(copied, rows)
	return &RepoKomite{rows: copied}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *RepoKomite) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh baris, terurut menurut ID seperti kueri lama.
func (r *RepoKomite) List(_ context.Context) ([]masterpenolakan.CommitteeRejection, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterpenolakan.CommitteeRejection, len(r.rows))
	copy(result, r.rows)
	// `ORDER BY IDMASTER` di sini adalah pengurutan ANGKA, bukan teks seperti pada Repo:
	// kolomnya bertipe NUMBER (`INSERTMASTERREJECTEDKOMITE.prc:1`). Perbedaan itu nyata
	// dan terlihat begitu tabel memuat lebih dari sembilan baris — "111" sebelum "9"
	// secara teks, tetapi sesudahnya secara angka.
	sort.SliceStable(result, func(i, j int) bool { return lessNumeric(result[i].ID, result[j].ID) })
	return result, nil
}

// Get mengembalikan satu baris berdasarkan ID-nya.
func (r *RepoKomite) Get(_ context.Context, id string) (masterpenolakan.CommitteeRejection, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpenolakan.CommitteeRejection{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, rejection := range r.rows {
		if rejection.ID == wanted {
			return rejection, nil
		}
	}
	return masterpenolakan.CommitteeRejection{}, masterpenolakan.ErrKomiteNotFound
}

// InsertNew menurunkan ID dari isi yang tersimpan lalu menambahkan barisnya.
//
// Termasuk aturan "baris pertama bernomor 111" pada tabel yang masih kosong, sama seperti
// adapter SQL — kalau tidak, uji terhadap tabel kosong akan lulus di sini dan gagal di
// produksi.
func (r *RepoKomite) InsertNew(_ context.Context, input masterpenolakan.InputKomite) (masterpenolakan.CommitteeRejection, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpenolakan.CommitteeRejection{}, r.failure
	}

	id := masterpenolakan.FormatIDKomite(masterpenolakan.FirstIDKomite)
	if len(r.rows) > 0 {
		used := make([]string, 0, len(r.rows))
		for _, rejection := range r.rows {
			used = append(used, rejection.ID)
		}
		id = masterpenolakan.NextSequence(used, masterpenolakan.FormatIDKomite)
	}

	fresh := masterpenolakan.CommitteeRejection{ID: id, Note: input.Note}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// Update menyimpan perubahan catatan pada baris yang sudah ada.
func (r *RepoKomite) Update(_ context.Context, id string, input masterpenolakan.InputKomite) (masterpenolakan.CommitteeRejection, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpenolakan.CommitteeRejection{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for i, rejection := range r.rows {
		if rejection.ID == wanted {
			// ID tidak ikut ditimpa dari luar: ia kunci baris, bukan isian.
			r.rows[i].Note = input.Note
			return r.rows[i], nil
		}
	}
	return masterpenolakan.CommitteeRejection{}, masterpenolakan.ErrKomiteNotFound
}

// lessNumeric membandingkan dua ID sebagai angka, dan jatuh ke perbandingan teks bila
// salah satunya tidak dapat ditafsirkan.
//
// Jalur cadangan itu ada karena kolomnya NUMBER menurut procedure lama, tetapi DDL-nya
// belum diterima (R-08) — dan baris lama dapat memuat apa saja bila ternyata tipenya
// bukan itu. Mengurutkannya salah lebih baik daripada panik.
func lessNumeric(left, right string) bool {
	leftNumber, leftErr := strconv.Atoi(strings.TrimSpace(left))
	rightNumber, rightErr := strconv.Atoi(strings.TrimSpace(right))
	if leftErr == nil && rightErr == nil {
		return leftNumber < rightNumber
	}
	return left < right
}

// SampleListKomite adalah isi awal POOLDATA.MST_REJECTED_KOMITE untuk pengembangan.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI, dengan alasan yang sama seperti SampleParents:
// isi tabelnya tidak ikut dikirim dan DDL-nya belum diterima (R-08).
//
// Nomor awalnya sengaja 111, bukan 1 — itulah nomor yang diterbitkan
// `Database/INSERTMASTERREJECTEDKOMITE.prc:9` pada tabel kosong, dan membuat contohnya
// bermula dari sana membuat keanehan itu terlihat saat pengembangan alih-alih ditemukan
// di produksi.
func SampleListKomite() []masterpenolakan.CommitteeRejection {
	return []masterpenolakan.CommitteeRejection{
		{ID: "111", Note: "NILAI KLAIM DI BAWAH RISIKO SENDIRI"},
		{ID: "112", Note: "DOKUMEN PENDUKUNG TIDAK MEYAKINKAN"},
		{ID: "113", Note: "OBJEK TIDAK TERCANTUM DALAM POLIS"},
	}
}

var _ masterpenolakan.RepoKomite = (*RepoKomite)(nil)
