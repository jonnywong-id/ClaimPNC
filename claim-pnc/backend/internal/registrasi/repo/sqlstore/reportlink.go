package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// ClaimReportLink menautkan klaim ke berkas Receive Document pada
// POOLDATA.T_CLAIM_RECIVEDCLAIM.
//
// Tabel itu dimiliki bersama dengan Pega selama masa paralel. Yang memisahkan kepemilikan
// adalah awalan kunci: baris terbitan aplikasi ini ber-CLAIMID `RCVN%`, dan pengisi ini
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

// Snapshot membaca isi berkas laporan yang dibawa ke klaim.
//
// Berkas milik Pega BOLEH dibaca; yang dilarang `P-1` adalah menulisinya. Karena itu
// method ini tidak memagari `RCVN%` seperti kedua UPDATE di atas.
func (p *ClaimReportLink) Snapshot(
	ctx context.Context,
	reportID string,
) (registrasi.ClaimReportSnapshot, error) {
	id := strings.TrimSpace(reportID)
	if id == "" {
		return registrasi.ClaimReportSnapshot{}, fmt.Errorf("registrasi/sqlstore: nomor laporan kosong")
	}

	var (
		lossDate                       sql.NullTime
		reportDateText                 sql.NullString
		name, phone, email             sql.NullString
		location, chronology, policyNo sql.NullString
		claimNumber                    sql.NullString
		estimateRupiah                 sql.NullFloat64
	)

	exec := executorFrom(ctx, p.db)
	err := exec.QueryRowContext(ctx, loadQuery("laporan_isi"), id).Scan(
		&lossDate, &reportDateText, &name, &phone, &email,
		&location, &chronology, &estimateRupiah, &policyNo, &claimNumber,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return registrasi.ClaimReportSnapshot{}, fmt.Errorf(
			"registrasi/sqlstore: laporan %s tidak ditemukan", id)
	case err != nil:
		return registrasi.ClaimReportSnapshot{}, fmt.Errorf(
			"registrasi/sqlstore: membaca isi laporan %s: %w", id, err)
	}

	return registrasi.ClaimReportSnapshot{
		DateOfLoss: lossDate.Time,
		ReportDate: parseReportDate(reportDateText.String),

		ReporterName:  strings.TrimSpace(name.String),
		ReporterPhone: strings.TrimSpace(phone.String),
		ReporterEmail: strings.TrimSpace(email.String),

		Location:   strings.TrimSpace(location.String),
		Chronology: strings.TrimSpace(chronology.String),

		// Kolom warisan menyimpan RUPIAH; domain memakai SEN (`ADR-0016`).
		EstimateValue: registrasi.Money(int64(estimateRupiah.Float64 * 100)),

		PolicyNumber: strings.TrimSpace(policyNo.String),
		ClaimNumber:  strings.TrimSpace(claimNumber.String),
	}, nil
}

// parseReportDate menafsirkan kolom TANGGALTERIMADOKUMEN, yang bertipe teks.
//
// Isinya berbeda menurut siapa yang menulis barisnya:
//
//	baris aplikasi ini  ISO `YYYY-MM-DD`   — selalu, lihat pernyataan UPDATE modul laporan
//	baris Pega          ReferenceId        — bukan tanggal sama sekali
//
// Yang bukan tanggal diperlakukan sebagai TIDAK ADA, bukan sebagai galat: berkas warisan
// boleh diregistrasi, dan menggagalkan pembuatan klaim karena satu kolom yang di sistem
// lama pun tidak berisi tanggal adalah menolak pekerjaan yang sah. Medannya tetap kosong,
// dan gerbang validasi Input Register yang akan menuntutnya.
//
// # Ditafsirkan sebagai WIB, bukan UTC
//
// Teksnya adalah tanggal kalender yang dipilih petugas di layar, dan layar itu WIB.
// Menafsirkannya sebagai UTC menghasilkan instan yang tujuh jam lebih awal — tanggal WIB-nya
// kebetulan masih sama, karena WIB mendahului UTC, tetapi ia menjadi satu-satunya tanggal di
// klaim yang tidak bertengah-malam WIB seperti kolom `DATE` lainnya. Perbedaan diam-diam
// semacam itulah yang menjadi `R-12`, dan `F-5` menuntut satu penafsiran saja.
func parseReportDate(raw string) time.Time {
	teks := strings.TrimSpace(raw)
	if teks == "" {
		return time.Time{}
	}
	for _, bentuk := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(bentuk, teks, clock.ZoneWIB); err == nil {
			return t
		}
	}
	return time.Time{}
}
