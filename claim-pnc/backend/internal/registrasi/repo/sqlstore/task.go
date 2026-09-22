package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"claim-pnc/internal/registrasi"
)

// TaskStore menyimpan tugas alur.
type TaskStore struct {
	db *sql.DB
}

// NewTaskStore membentuk repo; db wajib sudah terhubung.
func NewTaskStore(db *sql.DB) *TaskStore { return &TaskStore{db: db} }

// Save menuliskan satu tugas.
//
// UPDATE-nya bersyarat: ia menolak menimpa tugas yang sudah selesai, dan menolak
// mengubah tugas yang sudah dimiliki orang lain. Bila tidak ada baris yang terpengaruh,
// ada dua kemungkinan — tugasnya belum pernah ada, atau tugasnya sudah diambil orang
// lain — dan keduanya dibedakan dengan membaca ulang barisnya. Membedakan itu penting:
// yang pertama harus disisipkan, yang kedua harus dilaporkan sebagai bentrokan.
func (r *TaskStore) Save(ctx context.Context, t registrasi.Task) error {
	exec := executorFrom(ctx, r.db)

	result, err := exec.ExecContext(ctx, loadQuery("tugas_perbarui"),
		emptyTextAsNil(t.ClaimNumber),
		emptyTextAsNil(t.Owner),
		timePtrOrNil(t.ClaimedAt),
		timePtrOrNil(t.CompletedAt),
		emptyTextAsNil(t.CompletionReason),
		t.ID,
		t.Owner,
	)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: memperbarui tugas: %w", err)
	}
	row, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca jumlah baris tugas: %w", err)
	}
	if row > 0 {
		return nil
	}

	old, err := r.Get(ctx, t.ID)
	switch {
	case errors.Is(err, registrasi.ErrTaskNotFound):
		// Belum pernah ada — inilah kelahirannya.
	case err != nil:
		return err
	case !old.Open():
		return registrasi.ErrTaskAlreadyDone
	case old.Owned() && old.Owner != t.Owner:
		return registrasi.ErrTaskAlreadyClaimed
	default:
		// Baris ada, terbuka, dan pemiliknya sama — UPDATE tidak mengubah apa pun
		// karena nilainya memang sudah sama. Itu bukan kegagalan.
		return nil
	}

	if _, err := exec.ExecContext(ctx, loadQuery("tugas_sisip"),
		emptyTextAsNil(t.ClaimNumber),
		emptyTextAsNil(t.Owner),
		timePtrOrNil(t.ClaimedAt),
		timePtrOrNil(t.CompletedAt),
		emptyTextAsNil(t.CompletionReason),
		t.ID,
		t.ClaimID,
		t.Stage,
		string(t.Queue),
		emptyTextAsNil(t.Workbasket),
		t.CreatedAt.UTC(),
	); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyisipkan tugas: %w", err)
	}
	return nil
}

// Get mengembalikan satu tugas.
func (r *TaskStore) Get(ctx context.Context, id string) (registrasi.Task, error) {
	exec := executorFrom(ctx, r.db)
	row := exec.QueryRowContext(ctx, loadQuery("tugas_ambil"), id)

	t, err := scanTaskRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.Task{}, registrasi.ErrTaskNotFound
	}
	if err != nil {
		return registrasi.Task{}, fmt.Errorf("registrasi/sqlstore: membaca tugas: %w", err)
	}
	return t, nil
}

// OpenTaskForClaim mengembalikan tugas yang masih menunggu untuk sebuah klaim.
func (r *TaskStore) OpenTaskForClaim(ctx context.Context, claimID string) (registrasi.Task, error) {
	exec := executorFrom(ctx, r.db)
	row := exec.QueryRowContext(ctx, loadQuery("tugas_terbuka_klaim"), claimID)

	t, err := scanTaskRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.Task{}, registrasi.ErrTaskNotFound
	}
	if err != nil {
		return registrasi.Task{}, fmt.Errorf("registrasi/sqlstore: membaca tugas terbuka klaim: %w", err)
	}
	return t, nil
}

// Inbox mengembalikan pekerjaan yang menunggu seorang pengguna.
//
// Dua kueri, bukan satu dengan `IN (...)`: daftar workbasket panjangnya berubah-ubah,
// dan menyusun `IN` sepanjang daftar berarti teks SQL yang berbeda pada setiap
// pemanggilan — rencana eksekusi yang tidak pernah dipakai ulang, dan satu langkah lebih
// dekat ke perangkaian nilai ke dalam teks SQL.
func (r *TaskStore) Inbox(ctx context.Context, operator string, workbasket []string) ([]registrasi.Task, error) {
	exec := executorFrom(ctx, r.db)

	result, err := r.collect(ctx, exec, "tugas_inbox_milik_saya", operator)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, t := range result {
		seen[t.ID] = true
	}

	for _, w := range workbasket {
		queue, err := r.collect(ctx, exec, "tugas_inbox_antrean", w)
		if err != nil {
			return nil, err
		}
		for _, t := range queue {
			if seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			result = append(result, t)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (r *TaskStore) collect(ctx context.Context, exec executor, queryName, value string) ([]registrasi.Task, error) {
	row, err := exec.QueryContext(ctx, loadQuery(queryName), value)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca inbox: %w", err)
	}
	defer func() { _ = row.Close() }()

	var result []registrasi.Task
	for row.Next() {
		t, err := scanTask(row)
		if err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris inbox: %w", err)
		}
		result = append(result, t)
	}
	if err := row.Err(); err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: menelusuri inbox: %w", err)
	}
	return result, nil
}

// scanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi antarmuka apa pun di pustaka standar.
type scanner interface {
	Scan(target ...any) error
}

func scanTaskRow(row *sql.Row) (registrasi.Task, error) { return scanTask(row) }

func scanTask(row scanner) (registrasi.Task, error) {
	var (
		t                       registrasi.Task
		claimNumber, workbasket sql.NullString
		owner, completionReason sql.NullString
		queue                   string
		claimedAt, completedAt  sql.NullTime
	)

	if err := row.Scan(
		&t.ID, &t.ClaimID, &claimNumber, &t.Stage, &queue, &workbasket, &owner,
		&t.CreatedAt, &claimedAt, &completedAt, &completionReason,
	); err != nil {
		return registrasi.Task{}, err
	}

	t.ClaimNumber = claimNumber.String
	t.Queue = registrasi.QueueKind(queue)
	t.Workbasket = workbasket.String
	t.Owner = owner.String
	t.CompletionReason = completionReason.String
	if claimedAt.Valid {
		clock := claimedAt.Time
		t.ClaimedAt = &clock
	}
	if completedAt.Valid {
		clock := completedAt.Time
		t.CompletedAt = &clock
	}
	return t, nil
}

var _ registrasi.TaskRepo = (*TaskStore)(nil)
