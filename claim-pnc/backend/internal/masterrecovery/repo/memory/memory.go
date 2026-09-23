// Package memory adalah pengisi seam masterrecovery.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master
// Recovery dapat dicoba utuh — termasuk penerbitan nomor batch, pemilihan principal,
// unggahan bukti bayar, dan penyimpanannya.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/masterrecovery"
)

// Repo menyimpan batch recovery, master principal, dan lampiran di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu sync.RWMutex

	recovery  []masterrecovery.Recovery
	principal []masterrecovery.Principal
	policy    map[string]masterrecovery.PolicyReference
	document  map[string]masterrecovery.Document

	batch         int64
	documentOrder int64
	issues        error
}

// NewRepo membentuk repo berisi principal dan acuan polis yang diberikan.
func NewRepo(principal []masterrecovery.Principal, policy map[string]masterrecovery.PolicyReference) *Repo {
	clean := map[string]masterrecovery.PolicyReference{}
	for number, reference := range policy {
		clean[policyKey(number)] = reference
	}
	return &Repo{
		principal: append([]masterrecovery.Principal(nil), principal...),
		policy:    clean,
		document:  map[string]masterrecovery.Document{},
	}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issues = err
}

// Saved mengembalikan seluruh batch yang tersimpan.
//
// Ia TIDAK memenuhi bagian mana pun dari seam — masterrecovery.Repo tidak punya metode
// daftar, karena sistem lama pun tidak punya. Ia ada khusus untuk pengujian, supaya sebuah
// test dapat memastikan apa yang benar-benar tersimpan tanpa memaksa modul menumbuhkan
// kemampuan yang tidak diminta siapa pun.
func (r *Repo) Saved() []masterrecovery.Recovery {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]masterrecovery.Recovery(nil), r.recovery...)
}

// NextBatch mengembalikan nomor batch berikutnya.
func (r *Repo) NextBatch(context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return 0, r.issues
	}
	return r.batch + 1, nil
}

// Insert menyimpan satu batch dengan nomor yang diterbitkan di sini.
func (r *Repo) Insert(_ context.Context, recovery masterrecovery.Recovery) (masterrecovery.Recovery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterrecovery.Recovery{}, r.issues
	}

	r.batch++
	recovery.Batch = r.batch
	// Disalin supaya pemanggil yang menyunting senarainya setelah menyimpan tidak ikut
	// mengubah yang sudah tersimpan — kekeliruan yang mudah terjadi dan sulit dilacak.
	recovery.ClaimLine = append([]masterrecovery.ClaimLine(nil), recovery.ClaimLine...)
	r.recovery = append(r.recovery, recovery)
	return recovery, nil
}

// ListPrincipal mengembalikan seluruh principal, terurut menurut nama.
func (r *Repo) ListPrincipal(context.Context) ([]masterrecovery.Principal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return nil, r.issues
	}

	result := append([]masterrecovery.Principal(nil), r.principal...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// FindPrincipal mencari principal tanpa membedakan besar-kecil huruf, meniru pencocokan
// `upper(...)` pada procedure lama.
func (r *Repo) FindPrincipal(_ context.Context, clientID, name string) (masterrecovery.Principal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return masterrecovery.Principal{}, r.issues
	}

	wanted := masterrecovery.PrincipalKey(clientID, name)
	for _, p := range r.principal {
		if masterrecovery.PrincipalKey(p.ClientID, p.Name) == wanted {
			return p, nil
		}
	}
	return masterrecovery.Principal{}, masterrecovery.ErrPrincipalNotFound
}

// SavePrincipal mencatat principal beserta VA-nya.
func (r *Repo) SavePrincipal(_ context.Context, principal masterrecovery.Principal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return r.issues
	}
	r.principal = append(r.principal, principal)
	return nil
}

// LookupPolicy mencari identitas yang menempel pada sebuah nomor polis.
func (r *Repo) LookupPolicy(_ context.Context, policyNo string) (masterrecovery.PolicyReference, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return masterrecovery.PolicyReference{}, r.issues
	}

	reference, existing := r.policy[policyKey(policyNo)]
	if !existing {
		return masterrecovery.PolicyReference{}, masterrecovery.ErrPolicyNotFound
	}
	return reference, nil
}

// SaveDocument menyimpan lampiran dan mengembalikan penandanya.
//
// Bentuk penandanya meniru sqlstore — dua digit tahun disambung sepuluh angka urut —
// supaya layar yang menampilkannya berperilaku sama dengan dan tanpa Oracle.
func (r *Repo) SaveDocument(_ context.Context, document masterrecovery.Document) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return "", r.issues
	}

	r.documentOrder++
	id := SampleYear + tenDigits(r.documentOrder)
	r.document[id] = document
	return id, nil
}

// Document mengembalikan lampiran yang tersimpan, untuk pengujian.
func (r *Repo) Document(id string) (masterrecovery.Document, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	document, existing := r.document[id]
	return document, existing
}

// SampleYear adalah dua digit tahun yang dipakai penanda lampiran tanpa basis data.
//
// TETAP, bukan diambil dari jam sistem: penanda yang berubah setiap pergantian tahun akan
// membuat pengujian gagal tanpa ada yang menyentuh kode.
const SampleYear = "26"

func tenDigits(n int64) string {
	number := strconv.FormatInt(n, 10)
	if len(number) >= 10 {
		return number
	}
	return strings.Repeat("0", 10-len(number)) + number
}

// policyKey menyeragamkan nomor polis sebelum dicocokkan. Nomor polis warisan kerap
// membawa spasi tepi, dan membiarkannya membuat pencarian gagal tanpa sebab yang terlihat.
func policyKey(policyNo string) string {
	return strings.ToUpper(strings.TrimSpace(policyNo))
}

var _ masterrecovery.Repo = (*Repo)(nil)
