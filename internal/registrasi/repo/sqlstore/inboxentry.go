package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// InboxEntryStore menulis baris daftar kerja My Inbox ke POOLDATA.T_CLAIMLIST_ADMIN.
type InboxEntryStore struct {
	db *sql.DB
}

// NewInboxEntryStore membentuk penulis daftar kerja di atas sebuah koneksi.
func NewInboxEntryStore(db *sql.DB) *InboxEntryStore { return &InboxEntryStore{db: db} }

// Mirror menulis ulang baris daftar kerja satu klaim — memperbarui bila sudah ada,
// menyisipkan bila belum. Ikut transaksi yang dibawa context.
func (s *InboxEntryStore) Mirror(ctx context.Context, e registrasi.InboxEntry) error {
	key := e.Key()
	if key == "" {
		return nil
	}
	k := e.Claim
	p := k.Policy

	// Urutannya mengikuti SET pada daftar_kerja_perbarui kolom demi kolom, dimulai PYID.
	// Kode bisnis muncul dua kali: sekali untuk BUSINESSCODE_1, sekali untuk pencarian
	// BUSINESSGROUPID-nya.
	values := []any{
		key, // PYID
		registrasi.WorkObjectClass,
		registrasi.WorkFlowName,
		e.WorkStatus(),
		emptyTextAsNil(e.AssignedOperator()),
		emptyTextAsNil(e.StageName),
		emptyTextAsNil(k.RCVID),
		emptyTextAsNil(p.BusinessCode),
		emptyTextAsNil(p.BusinessCode),
		emptyTextAsNil(p.BranchCode),
		emptyTextAsNil(p.BranchName),
		emptyTextAsNil(k.CreatedBy),
		timeOrNil(k.CreatedAt),
		registerMoment(k.CreatedAt),
		emptyTextAsNil(k.TechnicalPIC),
		emptyTextAsNil(string(p.Line)),
		emptyTextAsNil(p.SourceOfBusinessName),
		emptyTextAsNil(p.BusinessName),
		// Nilai yang sama dengan kolom QQNAME di T_CLAIM_PNC (saveHeader). Policy.QQName
		// tidak dipakai: ia hanya terisi saat polis dibaca ulang, sehingga baris akan
		// berganti isi antara simpan pertama dan simpan berikutnya.
		emptyTextAsNil(p.InsuredName),
		emptyTextAsNil(p.Number),
		calendarDateOrNil(k.DateOfLoss),
		calendarDateOrNil(k.ReportDate),
		calendarDateOrNil(p.CoverageStart),
		calendarDateOrNil(p.CoverageEnd),
		emptyTextAsNil(p.ProdKe),
		emptyTextAsNil(string(k.ClaimStatus)),
		e.Active(),
	}

	// UPDATE menutup dengan PZINSKEY di WHERE; INSERT membukanya sebagai kolom pertama.
	update := append(append([]any{}, values...), key)
	insert := append([]any{key}, values...)
	if err := upsert(ctx, executorFrom(ctx, s.db),
		"daftar_kerja_perbarui", update,
		"daftar_kerja_sisip", insert,
	); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis daftar kerja %s ke POOLDATA.T_CLAIMLIST_ADMIN: %w", key, err)
	}
	return nil
}

var _ registrasi.InboxMirror = (*InboxEntryStore)(nil)
