package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/inboxinvestigator"
)

// InvestigationRepo menyimpan hasil investigasi satu portal di dalam memori.
//
// Ia adapter kedua seam InvestigationRepo — yang membuat seam itu nyata, bukan hipotetis
// (`04-FUTURE-ARCHITECTURE.md` §3). Ia juga satu-satunya cara formulir investigasi dapat
// dikerjakan dan diuji **sebelum** `POOLDATA.TC_PNC_INVESTIGASI` dibuat DBA.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga PERILAKU SIMPANNYA: sisip bila belum
// ada, perbarui bila sudah, dan satu peristiwa menyimpan formulir sekaligus memindahkan
// klaimnya. Repo memori yang menyimpan tanpa memindahkan akan membuat uji perpindahan
// status lulus tanpa membuktikan apa pun.
type InvestigationRepo struct {
	mutex   sync.Mutex
	rows    map[string]inboxinvestigator.Investigation
	moves   map[string]inboxinvestigator.Transition
	failure error
}

// NewInvestigationRepo membentuk repo kosong.
func NewInvestigationRepo() *InvestigationRepo {
	return &InvestigationRepo{
		rows:  map[string]inboxinvestigator.Investigation{},
		moves: map[string]inboxinvestigator.Transition{},
	}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *InvestigationRepo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// Load mengembalikan hasil investigasi yang tersimpan.
func (r *InvestigationRepo) Load(
	_ context.Context,
	claimRef string,
) (inboxinvestigator.Investigation, bool, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return inboxinvestigator.Investigation{}, false, r.failure
	}
	one, found := r.rows[strings.TrimSpace(claimRef)]
	return one, found, nil
}

// Save menyimpan formulir dan mencatat perpindahan klaimnya.
//
// Keduanya disimpan BERSAMA, sama seperti adapter SQL yang membungkusnya dalam satu
// transaksi. Galat yang dipasang menggagalkan keduanya sekaligus — bukan menyimpan
// formulirnya lalu gagal memindahkan, yang tidak mungkin terjadi pada adapter SQL.
func (r *InvestigationRepo) Save(
	_ context.Context,
	one inboxinvestigator.Investigation,
	move inboxinvestigator.Transition,
	_ string,
) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.failure != nil {
		return r.failure
	}
	key := strings.TrimSpace(one.ClaimRef)
	r.rows[key] = one
	r.moves[key] = move
	return nil
}

// MoveOf mengembalikan perpindahan yang tercatat untuk satu pekerjaan.
//
// Dipakai uji; ia yang membuktikan formulir yang disimpan BENAR-BENAR memindahkan klaimnya,
// bukan hanya menyimpan isiannya.
func (r *InvestigationRepo) MoveOf(claimRef string) (inboxinvestigator.Transition, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	move, found := r.moves[strings.TrimSpace(claimRef)]
	return move, found
}

var _ inboxinvestigator.InvestigationRepo = (*InvestigationRepo)(nil)
