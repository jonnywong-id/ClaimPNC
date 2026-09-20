// Package memory adalah pengisi seam masterpenolakan.Repo dan RepoKomite yang hidup di
// dalam memory.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat kedua seam nyata, bukan hipotetis. Ia juga yang memungkinkan aplikasi
// dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul auth dan portal.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara nomor baru
// diturunkan, dan urutan langkah penyimpanannya — kalau tidak, uji yang lulus di sini
// tidak membuktikan apa pun tentang adapter SQL.
package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/masterpenolakan"
)

// Repo menyimpan Master Penolakan Klaim satu portal di memory.
//
// Kedua tingkatnya disimpan di dalam satu instans, sama seperti adapter SQL menyimpannya
// di dalam satu basis data: penambahan tingkat 2 dapat menerbitkan baris tingkat 1 baru,
// dan keduanya harus terlihat oleh pembacaan berikutnya.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// ADR-0030 dan R-20.
type Repo struct {
	// mutex melindungi kedua senarai sekaligus. Penambahan menyentuh keduanya dalam satu
	// tarikan napas, dan menguncinya terpisah akan membiarkan pembaca melihat baris
	// tingkat 2 yang induknya belum ada — keadaan yang di adapter SQL dicegah transaksi.
	mutex   sync.Mutex
	parents []masterpenolakan.RejectionStatus
	rows    []masterpenolakan.RejectionStatus2
	failure error
}

// NewRepo membentuk repo berisi baris yang diberikan.
func NewRepo(parents []masterpenolakan.RejectionStatus, rows ...masterpenolakan.RejectionStatus2) *Repo {
	copiedParent := make([]masterpenolakan.RejectionStatus, len(parents))
	copy(copiedParent, parents)

	copiedRows := make([]masterpenolakan.RejectionStatus2, len(rows))
	copy(copiedRows, rows)

	return &Repo{parents: copiedParent, rows: copiedRows}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// ListParent mengembalikan seluruh Status Penolakan 1, terurut seperti kueri baru.
func (r *Repo) ListParent(_ context.Context) ([]masterpenolakan.RejectionStatus, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterpenolakan.RejectionStatus, len(r.parents))
	copy(result, r.parents)
	// `ORDER BY NOTE_ST ASC, ID_ST ASC` — ditiru apa adanya supaya daftar pilihan yang
	// terlihat saat pengembangan sama dengan yang terlihat di produksi.
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// List mengembalikan seluruh Status Penolakan 2, terurut seperti kueri lama.
func (r *Repo) List(_ context.Context) ([]masterpenolakan.RejectionStatus2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterpenolakan.RejectionStatus2, len(r.rows))
	copy(result, r.rows)
	// `ORDER BY ID_ST ASC, ID_ND ASC` pada basis data adalah pengurutan TEKS karena kedua
	// kolomnya bertipe teks. Ditiru apa adanya, termasuk keanehannya — "10" mendahului
	// "9" — supaya urutan di layar tidak berubah saat berpindah dari memori ke Oracle.
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ParentID != result[j].ParentID {
			return result[i].ParentID < result[j].ParentID
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

// Get mengembalikan satu Status Penolakan 2 berdasarkan ID-nya.
func (r *Repo) Get(_ context.Context, id string) (masterpenolakan.RejectionStatus2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpenolakan.RejectionStatus2{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	for _, rejection := range r.rows {
		if rejection.ID == wanted {
			return rejection, nil
		}
	}
	return masterpenolakan.RejectionStatus2{}, masterpenolakan.ErrNotFound
}

// InsertNew menyimpan satu Status Penolakan 2 baru beserta induknya bila diminta.
//
// Urutannya sengaja sama dengan adapter SQL: induk diselesaikan LEBIH DULU, sebelum nomor
// tingkat 2 diturunkan. Uji yang lulus di sini karena itu membuktikan sesuatu tentang
// adapter SQL, bukan hanya tentang dirinya sendiri.
func (r *Repo) InsertNew(_ context.Context, s masterpenolakan.Submission) (masterpenolakan.RejectionStatus2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpenolakan.RejectionStatus2{}, r.failure
	}

	parent, err := r.resolveParent(s.Input)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	fresh := masterpenolakan.RejectionStatus2{
		ID:          masterpenolakan.NextSequence(r.usedIDs(), masterpenolakan.FormatID2),
		Name:        s.Name,
		ParentID:    parent.ID,
		ParentName:  parent.Name,
		Status:      masterpenolakan.StatusPending,
		SubmittedBy: s.By,
		SubmittedAt: s.At.UTC(),
	}
	r.rows = append(r.rows, fresh)
	return fresh, nil
}

// Update menyimpan perubahan pada Status Penolakan 2 yang sudah ada.
//
// Ia MENGEMBALIKAN baris ke antrean persetujuan, persis seperti adapter SQL: status
// menjadi menunggu dan waktu pengajuan disetel ulang, sementara ketiga jejak persetujuan
// dibiarkan apa adanya.
func (r *Repo) Update(_ context.Context, id string, s masterpenolakan.Submission) (masterpenolakan.RejectionStatus2, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterpenolakan.RejectionStatus2{}, r.failure
	}

	wanted := strings.TrimSpace(id)
	position := -1
	for i, rejection := range r.rows {
		if rejection.ID == wanted {
			position = i
			break
		}
	}
	if position < 0 {
		return masterpenolakan.RejectionStatus2{}, masterpenolakan.ErrNotFound
	}

	// Induk diselesaikan SESUDAH barisnya ditemukan, sama seperti adapter SQL. Kalau
	// dibalik, permintaan yang menyebut ID tidak dikenal akan sempat menerbitkan baris
	// tingkat 1 baru yang tidak pernah dipakai siapa pun.
	parent, err := r.resolveParent(s.Input)
	if err != nil {
		return masterpenolakan.RejectionStatus2{}, err
	}

	// ID tidak ikut ditimpa dari luar: ia kunci baris, bukan isian. Ketiga jejak
	// persetujuan juga tidak disentuh.
	r.rows[position].Name = s.Name
	r.rows[position].ParentID = parent.ID
	r.rows[position].ParentName = parent.Name
	r.rows[position].Status = masterpenolakan.StatusPending
	r.rows[position].SubmittedBy = s.By
	r.rows[position].SubmittedAt = s.At.UTC()

	return r.rows[position], nil
}

// resolveParent menyediakan induk yang akan dirujuk, menerbitkannya bila diminta baru.
//
// Pemanggil sudah harus memegang mutex.
func (r *Repo) resolveParent(input masterpenolakan.Input) (masterpenolakan.RejectionStatus, error) {
	if !input.WantsNewParent() {
		wanted := strings.TrimSpace(input.ParentID)
		for _, parent := range r.parents {
			if parent.ID == wanted {
				return parent, nil
			}
		}
		return masterpenolakan.RejectionStatus{}, fmt.Errorf("%w: %q", masterpenolakan.ErrParentNotFound, input.ParentID)
	}

	used := make([]string, 0, len(r.parents))
	for _, parent := range r.parents {
		used = append(used, parent.ID)
	}

	fresh := masterpenolakan.RejectionStatus{
		ID:   masterpenolakan.NextSequence(used, masterpenolakan.FormatID),
		Name: input.ParentName,
	}
	r.parents = append(r.parents, fresh)
	return fresh, nil
}

// usedIDs mengumpulkan seluruh ID tingkat 2 yang sudah terpakai.
//
// Pemanggil sudah harus memegang mutex.
func (r *Repo) usedIDs() []string {
	used := make([]string, 0, len(r.rows))
	for _, rejection := range r.rows {
		used = append(used, rejection.ID)
	}
	return used
}

// SampleParents adalah isi awal POOLDATA.MST_PENOLAKAN_KLAIM_1 untuk pengembangan.
//
// PERINGATAN — INI BUKAN DATA PRODUKSI. Isi sebenarnya tidak ada di export: tidak ada
// berkas CSV-nya di `Database/` seperti halnya `v_sts_claim.csv` dan `m_portal_pnc.csv`,
// dan DDL-nya pun belum diterima (R-08).
//
// Nama-nama di bawah karena itu SUSUNAN SENDIRI, dipilih dari alasan penolakan yang
// benar-benar dikenal di domain ini — polis lapse, kerugian yang dikecualikan, dan
// pelaporan yang lewat batas waktu (`02-BUSINESS-UNDERSTANDING.md` §3.1). Ia tidak boleh
// dipakai sebagai dasar uji kesetaraan gerbang 1, dan harus diganti isi tabel yang
// sebenarnya begitu DBA mengirimkannya.
func SampleParents() []masterpenolakan.RejectionStatus {
	return []masterpenolakan.RejectionStatus{
		{ID: "1", Name: "POLIS TIDAK BERLAKU"},
		{ID: "2", Name: "KERUGIAN DIKECUALIKAN POLIS"},
		{ID: "3", Name: "PELAPORAN MELEWATI BATAS WAKTU"},
		{ID: "4", Name: "DOKUMEN TIDAK DILENGKAPI"},
	}
}

// SampleList adalah isi awal POOLDATA.MST_PENOLAKAN_KLAIM_2 untuk pengembangan.
//
// PERINGATAN yang sama seperti SampleParents berlaku di sini.
//
// Ketiga keadaan persetujuan sengaja terwakili — menunggu, disetujui, dan ditolak —
// supaya kolom "Status Aproval" di layar dapat dilihat dalam ketiga bentuknya tanpa
// menunggu layar Inbox Manager dibangun. Nama penyetuju di bawah karangan, bukan pegawai
// nyata, mengikuti aturan yang sama seperti daftar pengguna di provider tiruan.
func SampleList() []masterpenolakan.RejectionStatus2 {
	approved := sampleTime(2026, 9, 12)
	rejected := sampleTime(2026, 9, 15)

	return []masterpenolakan.RejectionStatus2{
		{
			ID: "1", Name: "PREMI BELUM DIBAYAR SAMPAI TANGGAL KEJADIAN",
			ParentID: "1", ParentName: "POLIS TIDAK BERLAKU",
			Status: masterpenolakan.StatusApproved, SubmittedBy: "adminpnc",
			SubmittedAt: sampleTime(2026, 9, 10),
			ApprovedBy:  "manageradmin", ApprovedAt: &approved,
			ApprovalNote: "Sesuai ketentuan polis.",
		},
		{
			ID: "2", Name: "PERIODE PERTANGGUNGAN SUDAH BERAKHIR",
			ParentID: "1", ParentName: "POLIS TIDAK BERLAKU",
			Status: masterpenolakan.StatusPending, SubmittedBy: "adminpnc",
			SubmittedAt: sampleTime(2026, 9, 16),
		},
		{
			ID: "3", Name: "KERUGIAN AKIBAT KEAUSAN",
			ParentID: "2", ParentName: "KERUGIAN DIKECUALIKAN POLIS",
			Status: masterpenolakan.StatusRejected, SubmittedBy: "pictekniks",
			SubmittedAt: sampleTime(2026, 9, 11),
			ApprovedBy:  "manageradmin", ApprovedAt: &rejected,
			ApprovalNote: "Perlu dirinci per jenis objek.",
		},
		{
			ID: "4", Name: "LAPORAN LEWAT 7 HARI SEJAK TANGGAL KEJADIAN",
			ParentID: "3", ParentName: "PELAPORAN MELEWATI BATAS WAKTU",
			Status: masterpenolakan.StatusPending, SubmittedBy: "pictekniks",
			SubmittedAt: sampleTime(2026, 9, 17),
		},
	}
}

// sampleTime menyusun waktu contoh dalam UTC.
//
// Ia UTC, bukan WIB, karena itulah yang disimpan seluruh aplikasi — konversi ke WIB
// terjadi sekali saja, di tempat waktu ditampilkan (`08-TECHNICAL-STRATEGY.md` §4.4).
// Menyusun contohnya dalam WIB akan membuat layar pengembangan menampilkan waktu yang
// bergeser tujuh jam dari yang diharapkan, lalu menyesatkan siapa pun yang memeriksanya.
func sampleTime(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 3, 0, 0, 0, time.UTC)
}

var _ masterpenolakan.Repo = (*Repo)(nil)
