package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// ClaimReportLink menautkan klaim ke berkas Receive Document pada
// POOLDATA.T_CLAIM_RECIVEDCLAIM.
//
// Tabel itu dimiliki bersama dengan Pega selama masa paralel. Yang memisahkan kepemilikan
// adalah awalan kunci: baris terbitan aplikasi ini ber-CLAIMID `RCVN.%`, dan pengisi ini
// MENOLAK menulis ke baris lain — lihat catatan pagar P-1 pada reportlink.sql.
type ClaimReportLink struct {
	db *sql.DB
}

// NewClaimReportLink membentuk penaut di atas sebuah koneksi.
func NewClaimReportLink(db *sql.DB) *ClaimReportLink { return &ClaimReportLink{db: db} }

// MarkHandedOver mengisi TRANSFERASM, memindahkan berkas dari Not Transferred ke
// Not Registered.
func (p *ClaimReportLink) MarkHandedOver(ctx context.Context, reportID string, at time.Time) error {
	id := strings.TrimSpace(reportID)
	if id == "" {
		return fmt.Errorf("registrasi/sqlstore: nomor laporan kosong")
	}

	exec := executorFrom(ctx, p.db)
	result, err := exec.ExecContext(ctx, loadQuery("laporan_tandai_diserahkan"), at.UTC(), id)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menandai laporan %s diserahkan: %w", id, err)
	}
	row, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca jumlah baris laporan: %w", err)
	}
	if row > 0 {
		return nil
	}

	// Nol baris punya TIGA sebab yang berbeda akibatnya, dan menyamakan ketiganya akan
	// menyembunyikan pelanggaran kepemilikan di balik pesan "tidak ditemukan".
	return p.explain(ctx, id, true)
}

// AttachClaimNumber mengisi NOKLAIM, memindahkan berkas ke Outstanding.
func (p *ClaimReportLink) AttachClaimNumber(ctx context.Context, reportID, claimNumber string) error {
	id := strings.TrimSpace(reportID)
	number := strings.TrimSpace(claimNumber)
	if id == "" {
		return fmt.Errorf("registrasi/sqlstore: nomor laporan kosong")
	}
	if number == "" {
		// Menuliskan string kosong BUKAN hal yang sama dengan membiarkan NULL: kueri
		// posisi membedakan keduanya, dan berkas ber-NOKLAIM kosong akan terbaca sebagai
		// sudah bernomor lalu lenyap dari seluruh tab.
		return fmt.Errorf("registrasi/sqlstore: nomor klaim kosong untuk laporan %s", id)
	}

	exec := executorFrom(ctx, p.db)
	result, err := exec.ExecContext(ctx, loadQuery("laporan_pasang_nomor_klaim"), number, id)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: memasang nomor klaim pada laporan %s: %w", id, err)
	}
	row, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca jumlah baris laporan: %w", err)
	}
	if row > 0 {
		return nil
	}
	return p.explain(ctx, id, false)
}

// explain mengubah "nol baris terpengaruh" menjadi galat yang menyebutkan sebabnya.
func (p *ClaimReportLink) explain(ctx context.Context, id string, handingOver bool) error {
	exec := executorFrom(ctx, p.db)

	var milikKita, sudahDiserahkan, sudahBernomor int
	err := exec.QueryRowContext(ctx, loadQuery("laporan_keadaan"), id).
		Scan(&milikKita, &sudahDiserahkan, &sudahBernomor)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("registrasi/sqlstore: laporan %s tidak ditemukan", id)
	case err != nil:
		return fmt.Errorf("registrasi/sqlstore: membaca keadaan laporan %s: %w", id, err)
	}

	if milikKita == 0 {
		return fmt.Errorf(
			"registrasi/sqlstore: laporan %s dimiliki sistem lama dan hanya dapat dibaca (P-1)", id)
	}
	if handingOver && sudahDiserahkan == 1 {
		// Bukan kegagalan: berkas sudah pernah diserahkan, dan tanggalnya sengaja tidak
		// digeser. Pendaftaran boleh lanjut.
		return nil
	}
	if !handingOver && sudahDiserahkan == 0 {
		return fmt.Errorf(
			"registrasi/sqlstore: laporan %s belum ditandai diserahkan; "+
				"memasang nomor klaim sekarang akan membuatnya hilang dari seluruh tab", id)
	}
	return fmt.Errorf("registrasi/sqlstore: laporan %s tidak dapat diperbarui", id)
}

var _ registrasi.ClaimReportLink = (*ClaimReportLink)(nil)
