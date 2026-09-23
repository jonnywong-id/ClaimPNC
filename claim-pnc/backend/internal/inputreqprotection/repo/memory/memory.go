// Package memory memenuhi seam inputreqprotection.Repo di dalam proses, tanpa basis data.
//
// Ia dipakai dua keadaan: pengembangan lokal (`PENYIMPANAN=memori`) dan seluruh pengujian.
// Keduanya berjalan tanpa Oracle dan tanpa jaringan.
//
// Untuk modul ini ia punya satu kegunaan tambahan yang tidak dimiliki modul lain: tabel
// `POOLDATA.T_CLAIM_OPENPROTECTION` BELUM ADA — Work Owner yang membuatnya. Sampai itu
// terjadi, penyimpanan ini adalah satu-satunya yang dapat menjalankan modul, dan
// aturan-aturan bisnisnya sudah dapat diuji sepenuhnya di sini.
//
// # Kenapa penyaringannya ditulis ulang, bukan disederhanakan
//
// Godaannya besar untuk membuat adapter memori "asal jalan" — mengembalikan semua baris dan
// membiarkan layar menyaring. Itu akan membuat pengujian membuktikan hal yang salah: yang
// teruji menjadi penyaring di layar, bukan penyaring yang sesungguhnya berjalan di
// produksi.
//
// Karena itu aturan di sini menirukan bentuk kueri yang direncanakan sedekat mungkin, dan
// setiap perbedaan yang tidak terhindarkan disebut di komentarnya.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inputreqprotection"
)

// Repo menyimpan proteksi di memori.
type Repo struct {
	// mu menjaga seluruh keadaan.
	//
	// Ia BUKAN kemewahan: penerbitan nomor membaca lalu menaikkan pencacah, dan dua
	// permintaan yang tiba bersamaan tanpa kunci akan menerbitkan nomor yang sama. Itu
	// persis kegagalan yang `SELECT … FOR UPDATE` cegah di sisi Oracle.
	mu sync.Mutex

	protections []inputreqprotection.Protection

	// counters memetakan tahun ke nomor urut terakhir yang terbit pada tahun itu.
	//
	// Bentuknya menirukan tabel pencacah `CPNC_NOMOR_KLAIM (TAHUN, TERAKHIR)` yang dipakai
	// nomor klaim, bukan sequence global — karena `OPCN.YY.xxxx` membawa tahun di dalam
	// nomornya, dan tanpa reset tahunan segmen itu tidak membedakan apa pun.
	counters map[int]int64
}

// NewRepo membentuk repo kosong.
func NewRepo() *Repo {
	return &Repo{counters: map[int]int64{}}
}

// NewRepoWithSamples membentuk repo berisi proteksi contoh.
//
// Data contohnya KARANGAN. `D-69` melarang nomor polis, nama tertanggung, dan nomor klaim
// sungguhan ditulis di berkas yang di-commit.
func NewRepoWithSamples() *Repo {
	r := NewRepo()
	r.protections = sampleProtections()
	return r
}

// Add menambahkan proteksi apa adanya. Dipakai pengujian.
func (r *Repo) Add(protections ...inputreqprotection.Protection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.protections = append(r.protections, protections...)
}

// List membaca satu halaman proteksi yang belum diakseptasi.
func (r *Repo) List(ctx context.Context, f inputreqprotection.Filter) (inputreqprotection.Page, error) {
	if err := ctx.Err(); err != nil {
		return inputreqprotection.Page{}, err
	}

	f = f.Normalize()

	r.mu.Lock()
	defer r.mu.Unlock()

	matched := make([]inputreqprotection.Protection, 0, len(r.protections))
	for _, p := range r.protections {
		// Penyaring layar ini hanya SATU, dan ia bukan pilihan pengguna melainkan definisi
		// layarnya: `InboxReqOpenProtection_RD` menyaring `.AcceptStatus IS NULL`.
		if p.Accepted() {
			continue
		}
		if !matches(p, f.Search) {
			continue
		}
		matched = append(matched, p)
	}

	// Pengurutan DESC mengikuti RD rujukan. Nomor dipakai sebagai pemutus seri supaya
	// urutannya tetap sama pada dua pemanggilan dengan waktu pembuatan yang identik —
	// tanpa itu, paginasi dapat menampilkan satu baris dua kali dan melewatkan baris lain.
	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].CreatedAt.Equal(matched[j].CreatedAt) {
			return matched[i].CreatedAt.After(matched[j].CreatedAt)
		}
		return matched[i].Number > matched[j].Number
	})

	total := len(matched)
	if f.Offset >= total {
		return inputreqprotection.Page{Protections: []inputreqprotection.Protection{}, Total: total}, nil
	}

	end := f.Offset + f.Limit
	if end > total {
		end = total
	}

	page := make([]inputreqprotection.Protection, end-f.Offset)
	copy(page, matched[f.Offset:end])

	return inputreqprotection.Page{Protections: page, Total: total}, nil
}

// Get membaca satu proteksi menurut nomornya.
func (r *Repo) Get(ctx context.Context, number string) (inputreqprotection.Protection, error) {
	if err := ctx.Err(); err != nil {
		return inputreqprotection.Protection{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	index := r.indexOf(number)
	if index < 0 {
		return inputreqprotection.Protection{}, inputreqprotection.ErrNotFound
	}
	return r.protections[index], nil
}

// HasDuplicate menyatakan sudah ada proteksi dengan polis dan tipe yang sama pada hari yang
// sama.
//
// Perbandingan hari dilakukan di ZONA yang sama dengan kunci, bukan di UTC. Membandingkan
// di UTC akan menggeser batas hari tujuh jam, sehingga proteksi yang dibuat pukul 06.00 WIB
// dianggap milik hari sebelumnya.
func (r *Repo) HasDuplicate(
	ctx context.Context,
	key inputreqprotection.DuplicateKey,
	exceptNumber string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	location := key.Day.Location()
	except := normalize(exceptNumber)

	for _, p := range r.protections {
		if except != "" && normalize(p.Number) == except {
			continue
		}
		if normalize(p.PolicyNumber) != normalize(key.PolicyNumber) {
			continue
		}
		if strings.TrimSpace(p.Type) != strings.TrimSpace(key.Type) {
			continue
		}
		if !sameDay(p.InputDate, key.Day, location) {
			continue
		}
		return true, nil
	}
	return false, nil
}

// Create menyimpan proteksi baru beserta nomor yang terbit.
func (r *Repo) Create(
	ctx context.Context,
	draft inputreqprotection.Draft,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	if err := ctx.Err(); err != nil {
		return inputreqprotection.Protection{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Tahun diambil dari waktu yang DISERAHKAN pemanggil, bukan dari time.Now di sini —
	// supaya nomor yang terbit dapat diuji, dan supaya sumber waktunya satu.
	year := at.Year()
	r.counters[year]++

	p := inputreqprotection.Protection{
		Number:         inputreqprotection.FormatNumber(year, r.counters[year]),
		PolicyNumber:   draft.PolicyNumber,
		ClaimNumber:    draft.ClaimNumber,
		ClaimReference: draft.ClaimReference,
		Type:           draft.Type,
		InputDate:      at,
		Note:           draft.Note,
		AcceptStatus:   inputreqprotection.AcceptPending,
		CreatedBy:      by,
		CreatedAt:      at,
		ChangeDetail:   draft.ChangeDetail,
	}
	r.protections = append(r.protections, p)

	return p, nil
}

// Update menyunting proteksi yang belum tertaut klaim dan belum diakseptasi.
//
// Pemeriksaan keadaan DIULANG di sini meski usecase sudah melakukannya. Itu bukan
// pengulangan yang sia-sia: dua permintaan yang tiba bersamaan sama-sama lolos pemeriksaan
// di usecase, dan hanya pemeriksaan di dalam kunci penyimpanan yang menahan yang kedua.
func (r *Repo) Update(
	ctx context.Context,
	number string,
	draft inputreqprotection.Draft,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	if err := ctx.Err(); err != nil {
		return inputreqprotection.Protection{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	index := r.indexOf(number)
	if index < 0 {
		return inputreqprotection.Protection{}, inputreqprotection.ErrNotFound
	}

	existing := r.protections[index]
	if existing.Accepted() {
		return inputreqprotection.Protection{}, inputreqprotection.ErrAccepted
	}
	if !existing.Editable() {
		return inputreqprotection.Protection{}, inputreqprotection.ErrLocked
	}

	// Nomor, ID, waktu pembuatan, dan pembuatnya TIDAK berubah saat disunting. Keempatnya
	// menyatakan asal-usul baris; menggantinya akan membuat jejaknya menunjuk orang dan
	// waktu yang salah.
	//
	// Kolom akseptasi juga tidak disentuh — ia milik modul inboxacceptopenprotection.
	existing.PolicyNumber = draft.PolicyNumber
	existing.ClaimNumber = draft.ClaimNumber
	existing.ClaimReference = draft.ClaimReference
	existing.Type = draft.Type
	existing.Note = draft.Note
	existing.ChangeDetail = draft.ChangeDetail

	r.protections[index] = existing
	return existing, nil
}

// indexOf mencari proteksi menurut nomornya. Pemanggil WAJIB memegang kunci.
func (r *Repo) indexOf(number string) int {
	target := normalize(number)
	if target == "" {
		return -1
	}
	for i, p := range r.protections {
		if normalize(p.Number) == target {
			return i
		}
	}
	return -1
}

// matches menyatakan sebuah proteksi cocok dengan kata pencarian.
//
// Mencari pada tiga kolom sekaligus — No Proteksi, No Polis, No Klaim — meniru kotak
// pencarian gabungan yang dipakai layar inbox modul lain. Pencocokannya TIDAK peka
// besar-kecil, karena nomor polis warisan tidak diseragamkan.
func matches(p inputreqprotection.Protection, search string) bool {
	needle := normalize(search)
	if needle == "" {
		return true
	}
	return strings.Contains(normalize(p.Number), needle) ||
		strings.Contains(normalize(p.PolicyNumber), needle) ||
		strings.Contains(normalize(p.ClaimNumber), needle)
}

// sameDay membandingkan dua waktu sebagai TANGGAL KALENDER di zona yang diberikan.
func sameDay(a, b time.Time, location *time.Location) bool {
	if location == nil {
		location = time.UTC
	}
	ay, am, ad := a.In(location).Date()
	by, bm, bd := b.In(location).Date()
	return ay == by && am == bm && ad == bd
}

func normalize(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
