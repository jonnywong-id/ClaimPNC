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

// NumberIssuer menerbitkan nomor klaim berformat `PNCN.YY.xxxx` (`D-71`, `ADR-0009`).
//
// # Kenapa pencacah tabel, bukan SEQUENCE Oracle
//
// Sequence Oracle tidak dapat direset per tahun tanpa DDL, dan `ADR-0005` menetapkan
// perpindahan ke PostgreSQL akan datang — sequence adalah salah satu tempat kedua basis
// data paling berbeda. Satu baris per tahun yang dikunci `SELECT … FOR UPDATE` berjalan
// sama di keduanya.
//
// # Kenapa ia WAJIB dipanggil di dalam transaksi
//
// Kuncinya adalah kunci baris, dan kunci baris hanya bertahan selama transaksi. Dipanggil
// di luar transaksi, dua pendaftaran bersamaan akan membaca angka yang sama. Usecase
// memanggilnya dari dalam `UnitOfWork.Jalankan`; pemeriksaannya ada di bawah supaya
// pelanggaran terhadap aturan ini menjadi galat yang terlihat, bukan nomor ganda yang
// baru ketahuan berbulan-bulan kemudian.
type NumberIssuer struct {
	db     *sql.DB
	prefix string
}

// NewNumberIssuer membentuk penerbit di atas sebuah koneksi.
func NewNumberIssuer(db *sql.DB) *NumberIssuer {
	return &NumberIssuer{db: db, prefix: "PNCN"}
}

// Issue mengembalikan nomor klaim berikutnya untuk tahun yang berlaku.
//
// Tahun diambil menurut WIB: klaim yang didaftarkan pukul 06.00 WIB tanggal 1 Januari
// masih 31 Desember di UTC, dan nomornya harus membawa tahun yang dilihat petugas.
func (p *NumberIssuer) Issue(ctx context.Context, at time.Time) (string, error) {
	tx, ok := txFrom(ctx)
	if !ok {
		return "", errors.New("registrasi/sqlstore: nomor klaim hanya boleh diterbitkan di dalam transaksi")
	}

	year := clock.DateWIB(at).Year()

	var last int64
	err := tx.QueryRowContext(ctx, loadQuery("nomor_kunci_tahun"), year).Scan(&last)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, loadQuery("nomor_mulai_tahun"), year); err != nil {
			return "", fmt.Errorf("registrasi/sqlstore: memulai pencacah tahun %d: %w", year, err)
		}
		last = 1
	case err != nil:
		return "", fmt.Errorf("registrasi/sqlstore: mengunci pencacah tahun %d: %w", year, err)
	default:
		if _, err := tx.ExecContext(ctx, loadQuery("nomor_naikkan"), year); err != nil {
			return "", fmt.Errorf("registrasi/sqlstore: menaikkan pencacah tahun %d: %w", year, err)
		}
		last++
	}

	// Lebar minimum empat digit, dan TUMBUH bila terlampaui. Memotong pada empat digit
	// membuat nomor ke-10.001 menabrak nomor yang sudah terbit. Apakah lebarnya memang
	// dibuat tetap masih pertanyaan terbuka `TKT-F2-006`; yang dipilih di sini adalah
	// tafsir yang tidak dapat menghasilkan tabrakan.
	return fmt.Sprintf("%s.%02d.%04d", p.prefix, year%100, last), nil
}

// AuditRecorder menuliskan jejak audit.
type AuditRecorder struct {
	db *sql.DB
	id registrasi.IDGenerator
}

// NewAuditRecorder membentuk perekam di atas sebuah koneksi.
func NewAuditRecorder(db *sql.DB, id registrasi.IDGenerator) *AuditRecorder {
	return &AuditRecorder{db: db, id: id}
}

// Record menambahkan satu baris jejak audit.
func (p *AuditRecorder) Record(ctx context.Context, j registrasi.AuditTrail) error {
	exec := executorFrom(ctx, p.db)
	_, err := exec.ExecContext(ctx, loadQuery("audit_sisip"),
		p.id.New(),
		emptyTextAsNil(j.ClaimID),
		emptyTextAsNil(j.ClaimNumber),
		j.Event,
		j.Actor,
		j.At.UTC(),
		trim(j.Note, 2000),
	)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: merekam jejak audit: %w", err)
	}
	return nil
}

// Notifier mencatat peristiwa pemberitahuan ke kotak keluar.
//
// Ia MENCATAT, bukan mengirim — lihat kontrak seam `registrasi.Notifier`. Pengirimannya
// milik `S-3`, yang membaca CPNC_NOTIFIKASI di luar transaksi.
type Notifier struct {
	db *sql.DB
	id registrasi.IDGenerator
}

// NewNotifier membentuk notifier di atas sebuah koneksi.
func NewNotifier(db *sql.DB, id registrasi.IDGenerator) *Notifier {
	return &Notifier{db: db, id: id}
}

// Send mencatat satu peristiwa pemberitahuan.
func (n *Notifier) Send(ctx context.Context, m registrasi.Notification) error {
	exec := executorFrom(ctx, n.db)
	_, err := exec.ExecContext(ctx, loadQuery("notifikasi_sisip"),
		n.id.New(),
		string(m.Kind),
		emptyTextAsNil(m.ClaimNumber),
		emptyTextAsNil(m.PolicyNumber),
		trim(strings.Join(m.Recipients, ", "), 2000),
		int64(m.RupiahValue),
		m.At.UTC(),
	)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: mencatat pemberitahuan: %w", err)
	}
	return nil
}

// trim memangkas teks agar muat di kolomnya.
//
// Kolom yang meluap ditolak Oracle dengan galat yang tidak berbicara apa pun kepada
// pengguna, dan yang meluap di sini adalah keterangan — bukan data bisnis. Memangkasnya
// lebih baik daripada menggagalkan penyimpanan klaim karena jejaknya terlalu panjang.
func trim(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit]
}

var (
	_ registrasi.NumberIssuer  = (*NumberIssuer)(nil)
	_ registrasi.AuditRecorder = (*AuditRecorder)(nil)
	_ registrasi.Notifier      = (*Notifier)(nil)
)
