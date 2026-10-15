package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/dashboardclaim"
)

// AssignmentWriter memindahkan PIC Teknik di dalam memori.
//
// Adapter KEDUA di balik seam `dashboardclaim.AssignmentWriter`, dan pada seam ini nilainya
// lebih besar daripada sekadar membuat seam-nya nyata: sisi SQL-nya menulis tabel milik Pega
// dan menuntut hak basis data yang belum diberikan, sehingga tanpa adapter ini tombol Assign
// tidak dapat dicoba sama sekali di lingkungan pengembangan.
type AssignmentWriter struct {
	mu    sync.Mutex
	repo  *Repo
	moves []dashboardclaim.PICMove
}

// NewAssignmentWriter membungkus Repo yang memegang klaimnya.
//
// Repo-nya DIBUTUHKAN, bukan opsional: pemindahan yang tidak mengubah daftar akan terlihat
// berhasil lalu tidak berbekas, dan itu persis kelas kekeliruan yang membuat adapter memori
// ini ada.
func NewAssignmentWriter(repo *Repo) *AssignmentWriter {
	return &AssignmentWriter{repo: repo}
}

// MovePIC mengubah PIC Teknik klaim pada simpanan memori.
func (w *AssignmentWriter) MovePIC(
	ctx context.Context,
	move dashboardclaim.PICMove,
) (dashboardclaim.PICMoveResult, error) {
	_ = ctx

	claimID := strings.TrimSpace(move.ClaimID)
	toOperator := strings.TrimSpace(move.ToOperator)
	if claimID == "" || toOperator == "" {
		return dashboardclaim.PICMoveResult{}, dashboardclaim.ErrClaimNotFound
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	from, found := w.repo.movePIC(claimID, toOperator)
	if !found {
		return dashboardclaim.PICMoveResult{}, dashboardclaim.ErrClaimNotFound
	}

	w.moves = append(w.moves, move)
	return dashboardclaim.PICMoveResult{FromOperator: from}, nil
}

// Moves mengembalikan pemindahan yang tercatat, untuk diperiksa uji.
func (w *AssignmentWriter) Moves() []dashboardclaim.PICMove {
	w.mu.Lock()
	defer w.mu.Unlock()

	result := make([]dashboardclaim.PICMove, len(w.moves))
	copy(result, w.moves)
	return result
}

// movePIC mengubah PIC Teknik satu klaim dan mengembalikan PIC sebelumnya.
//
// Berada di Repo, bukan di AssignmentWriter, karena yang diubah adalah klaim — dan klaimnya
// dimiliki Repo. Mengubahnya dari luar akan membuat dua tempat memegang kebenaran yang sama.
func (r *Repo) movePIC(claimID, toOperator string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.claims {
		row := &r.claims[i].Row
		if row.ClaimID != claimID {
			continue
		}
		from := row.TechnicalPIC
		row.TechnicalPIC = toOperator
		return from, true
	}
	return "", false
}

// MoveAllForOperator memindahkan seluruh klaim berjalan milik satu petugas.
//
// Sisi SQL-nya menyaring klaim yang masih berjalan; di sini penyaring itu tidak ada, karena
// simpanan memori memang hanya memuat klaim berjalan — `SampleOutstanding()` dan
// `SampleClosed()` disimpan terpisah.
func (w *AssignmentWriter) MoveAllForOperator(ctx context.Context, from, to string) (int, error) {
	_ = ctx

	fromOperator := strings.TrimSpace(from)
	toOperator := strings.TrimSpace(to)
	if fromOperator == "" || toOperator == "" || fromOperator == toOperator {
		return 0, nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.repo.moveAllForOperator(fromOperator, toOperator), nil
}

// moveAllForOperator mengubah PIC pada setiap klaim milik petugas itu.
func (r *Repo) moveAllForOperator(from, to string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	moved := 0
	for i := range r.claims {
		row := &r.claims[i].Row
		if row.TechnicalPIC != from {
			continue
		}
		row.TechnicalPIC = to
		moved++
	}
	return moved
}

// MoveAllMatching memindahkan seluruh klaim yang cocok dengan penyaring, di dalam memori.
//
// Penyaring yang DITERAPKAN di sini sengaja hanya kotak cari dan lini bisnis — persis yang
// dapat dijawab simpanan memori. Penyaring panel (Nopolis, No Klaim, PIC, status transfer,
// status bayar) tidak dapat ditiru tanpa menyalin seluruh logika SQL-nya, dan fake yang
// menebak-nebak penyaring akan melaporkan jumlah yang BERBEDA dari Oracle.
//
// Batas itu dinyatakan, bukan disembunyikan: adapter ini ada supaya tombolnya dapat dicoba
// tanpa Oracle, bukan supaya hasilnya dapat dipercaya sama dengan produksi.
func (w *AssignmentWriter) MoveAllMatching(
	ctx context.Context,
	filter dashboardclaim.Filter,
	to string,
) (int, error) {
	_ = ctx

	toOperator := strings.TrimSpace(to)
	if toOperator == "" {
		return 0, nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.repo.moveAllMatching(filter.Normalize(), toOperator), nil
}

// moveAllMatching mengubah PIC pada setiap klaim yang cocok.
func (r *Repo) moveAllMatching(filter dashboardclaim.Filter, to string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	moved := 0
	for i := range r.claims {
		row := &r.claims[i].Row
		if !cocokPenyaringSederhana(*row, filter) {
			continue
		}
		if row.TechnicalPIC == to {
			continue
		}
		row.TechnicalPIC = to
		moved++
	}
	return moved
}

// cocokPenyaringSederhana menjawab kotak cari saja — lihat catatan pada MoveAllMatching.
func cocokPenyaringSederhana(row dashboardclaim.ClaimRow, filter dashboardclaim.Filter) bool {
	if filter.Search == "" {
		return true
	}

	cari := strings.ToUpper(filter.Search)
	return strings.Contains(strings.ToUpper(row.ClaimNumber), cari) ||
		strings.Contains(strings.ToUpper(row.PolicyNumber), cari)
}
