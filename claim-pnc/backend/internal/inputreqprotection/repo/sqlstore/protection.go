package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inputreqprotection"
)

// Repo membaca dan menulis POOLDATA.T_CLAIM_OPENPROTECTION.
//
// # Kolom yang ditulisnya, dan yang TIDAK
//
// Ia menulis kolom PEMBUATAN saja: `ID`, `POLICY_NO`, `CLAIM_NO`, `ID_CLAIM`, `PROTECTION_TYPE`,
// `CREATE_DATE`, `CREATED_BY`, `NOTES`, `OLD_DATA`, `NEW_DATA`, `OBJECT_NAME`, `BRANCH_NAME`,
// `STATUS_ACTIVE`.
//
// Kolom AKSEPTASI — `APPROVAL_STATUS`, `RESOLVED_BY`, dan `RESOLVED_DATE_TIME` —
// tidak pernah disentuh di sini; ketiganya milik modul `inboxacceptopenprotection`.
// Pembagian itu yang menjaga `P-1` tetap berlaku meski dua modul menyentuh satu tabel.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// tanggalTeks adalah bentuk tanggal di dalam OLD_DATA/NEW_DATA.
//
// Lihat kepala protection.sql untuk alasan memilih bentuk ini, bukan format Pega.
const tanggalTeks = "2006-01-02"

// List membaca satu halaman permintaan yang belum diakseptasi.
func (r *Repo) List(ctx context.Context, f inputreqprotection.Filter) (inputreqprotection.Page, error) {
	f = f.Normalize()

	cari := searchArgs(f.Search)

	var total int
	if err := r.db.QueryRowContext(ctx, query("protection_count"), cari...).Scan(&total); err != nil {
		return inputreqprotection.Page{}, fmt.Errorf(
			"inputreqprotection/sqlstore: menghitung permintaan proteksi: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query("protection_list"),
		append(append([]any(nil), cari...), f.Offset, f.Limit)...)
	if err != nil {
		return inputreqprotection.Page{}, fmt.Errorf(
			"inputreqprotection/sqlstore: membaca permintaan proteksi: %w", err)
	}
	defer rows.Close()

	page := make([]inputreqprotection.Protection, 0, f.Limit)
	for rows.Next() {
		p, err := scanProtection(rows)
		if err != nil {
			return inputreqprotection.Page{}, err
		}
		page = append(page, p)
	}
	if err := rows.Err(); err != nil {
		return inputreqprotection.Page{}, fmt.Errorf(
			"inputreqprotection/sqlstore: membaca baris permintaan proteksi: %w", err)
	}

	return inputreqprotection.Page{Protections: page, Total: total}, nil
}

// Get membaca satu permintaan menurut nomornya.
func (r *Repo) Get(ctx context.Context, number string) (inputreqprotection.Protection, error) {
	// Kuncinya dibandingkan dengan `UPPER(TRIM(ID)) = :1`, sehingga nilainya pun harus
	// sudah dirapikan di sini — bukan dibungkus fungsi lagi di dalam SQL.
	row := r.db.QueryRowContext(ctx, query("protection_get"), kunci(number))

	p, err := scanProtection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inputreqprotection.Protection{}, inputreqprotection.ErrNotFound
	}
	if err != nil {
		return inputreqprotection.Protection{}, err
	}
	return p, nil
}

// HasDuplicate menyatakan sudah ada permintaan dengan polis dan tipe yang sama pada hari
// yang sama.
func (r *Repo) HasDuplicate(
	ctx context.Context,
	key inputreqprotection.DuplicateKey,
	exceptNumber string,
) (bool, error) {
	// Nomor yang dikecualikan muncul DUA KALI di dalam kueri, sehingga dikirim dua kali —
	// driver mengikat menurut urutan kemunculan penanda, bukan menurut nomornya.
	var except any
	if strings.TrimSpace(exceptNumber) != "" {
		except = kunci(exceptNumber)
	}

	var jumlah int
	err := r.db.QueryRowContext(ctx, query("protection_duplicate"),
		strings.ToUpper(strings.TrimSpace(key.PolicyNumber)),
		strings.TrimSpace(key.Type),
		except, except,
		key.Day).Scan(&jumlah)
	if err != nil {
		return false, fmt.Errorf("inputreqprotection/sqlstore: memeriksa proteksi ganda: %w", err)
	}
	return jumlah > 0, nil
}

// createAttempts adalah banyaknya percobaan penyisipan saat nomornya bentrok.
//
// Bentrok hanya mungkin terjadi bila dua permintaan menerbitkan nomor yang sama, dan itu
// dijawab dengan mengambil nomor berikutnya — bukan dengan menolak permintaan pengguna.
//
// Tiga sudah lebih dari cukup: setiap percobaan membaca ulang nomor tertinggi, sehingga
// dua permintaan bersamaan paling banyak menabrak sekali.
//
// CATATAN: percobaan ulang ini HANYA berfungsi bila ada constraint unik pada kolom ID.
// Constraint itu belum ada — lihat protection.sql, kueri protection_next_sequence.
const createAttempts = 3

// Create menyimpan permintaan baru beserta nomor yang terbit.
//
// Penerbitan nomor dan penyisipan barisnya berada DI DALAM SATU TRANSAKSI. Memisahkannya
// berarti nomor dapat terbit lalu barisnya gagal disimpan, meninggalkan lubang pada deret —
// dan pada deret yang dibaca manusia, lubang akan terus ditanyakan.
func (r *Repo) Create(
	ctx context.Context,
	draft inputreqprotection.Draft,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	var terakhir error

	for percobaan := 0; percobaan < createAttempts; percobaan++ {
		p, err := r.createOnce(ctx, draft, by, at)
		if err == nil {
			return p, nil
		}
		if !isUniqueViolation(err) {
			return inputreqprotection.Protection{}, err
		}
		terakhir = err
	}

	return inputreqprotection.Protection{}, fmt.Errorf(
		"inputreqprotection/sqlstore: nomor proteksi bentrok setelah %d percobaan: %w",
		createAttempts, terakhir)
}

func (r *Repo) createOnce(
	ctx context.Context,
	draft inputreqprotection.Draft,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: membuka transaksi: %w", err)
	}
	// Rollback pada jalur galat; pada jalur sukses ia tidak berpengaruh karena Commit
	// sudah berjalan lebih dulu.
	defer func() { _ = tx.Rollback() }()

	year := at.Year()
	pola := fmt.Sprintf("%s.%02d.%%", inputreqprotection.NumberPrefix, year%100)

	var urut int64
	if err := tx.QueryRowContext(ctx, query("protection_next_sequence"), pola).Scan(&urut); err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: menerbitkan nomor proteksi: %w", err)
	}

	nomor := inputreqprotection.FormatNumber(year, urut)
	lama, baru := encodeChangeDetail(draft.Type, draft.ChangeDetail)

	if _, err := tx.ExecContext(ctx, query("protection_insert"),
		nomor,
		nullIfEmpty(draft.PolicyNumber),
		nullIfEmpty(draft.ClaimNumber),
		nullIfEmpty(draft.ClaimReference),
		nullIfEmpty(draft.Type),
		at.UTC(),
		nullIfEmpty(by),
		nullIfEmpty(draft.Note),
		lama,
		baru,
		nullIfEmpty(draft.ChangeDetail.ObjectName),
		nullIfEmpty(draft.ChangeDetail.BranchName),
	); err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: menyimpan permintaan proteksi: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: menyelesaikan transaksi: %w", err)
	}

	return inputreqprotection.Protection{
		Number:         nomor,
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
	}, nil
}

// Update menyunting permintaan yang belum tertaut klaim dan belum diakseptasi.
//
// Bila tidak ada baris yang tersentuh, sebabnya DIBEDAKAN lewat pembacaan ulang: tidak ada,
// sudah tertaut klaim, atau sudah diakseptasi. Ketiganya menuntut pesan yang berbeda, dan
// mengembalikan satu galat umum akan membuat pengguna menunggu sesuatu yang tidak akan
// datang.
func (r *Repo) Update(
	ctx context.Context,
	number string,
	draft inputreqprotection.Draft,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	lama, baru := encodeChangeDetail(draft.Type, draft.ChangeDetail)

	hasil, err := r.db.ExecContext(ctx, query("protection_update"),
		nullIfEmpty(draft.PolicyNumber),
		nullIfEmpty(draft.ClaimNumber),
		nullIfEmpty(draft.ClaimReference),
		nullIfEmpty(draft.Type),
		nullIfEmpty(draft.Note),
		lama,
		baru,
		nullIfEmpty(draft.ChangeDetail.ObjectName),
		nullIfEmpty(draft.ChangeDetail.BranchName),
		kunci(number),
	)
	if err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: menyunting permintaan proteksi: %w", err)
	}

	tersentuh, err := hasil.RowsAffected()
	if err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: membaca jumlah baris tersentuh: %w", err)
	}

	if tersentuh == 0 {
		existing, getErr := r.Get(ctx, number)
		switch {
		case getErr != nil:
			return inputreqprotection.Protection{}, getErr
		case existing.Accepted():
			return inputreqprotection.Protection{}, inputreqprotection.ErrAccepted
		default:
			return inputreqprotection.Protection{}, inputreqprotection.ErrLocked
		}
	}

	return r.Get(ctx, number)
}

// ── Pemindaian dan penerjemahan ──────────────────────────────────────────────────

// pemindai menyatukan *sql.Row dan *sql.Rows supaya scanProtection melayani keduanya.
type pemindai interface {
	Scan(dest ...any) error
}

// scanProtection membaca satu baris menjadi Protection.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: tabelnya membolehkan NULL pada
// semuanya, dan memindainya ke string biasa akan gagal pada baris pertama yang kolomnya
// kosong.
func scanProtection(row pemindai) (inputreqprotection.Protection, error) {
	var (
		id                                string
		nopolis, noklaim, idpega, tipe    sql.NullString
		dibuatPada                        sql.NullTime
		notes, status, dibuatOleh         sql.NullString
		lama, baru, namaObjek, namaCabang sql.NullString
	)

	if err := row.Scan(&id, &nopolis, &noklaim, &idpega, &tipe,
		&dibuatPada, &notes, &status, &dibuatOleh,
		&lama, &baru, &namaObjek, &namaCabang); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inputreqprotection.Protection{}, err
		}
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: memindai permintaan proteksi: %w", err)
	}

	p := inputreqprotection.Protection{
		Number:         strings.TrimSpace(id),
		PolicyNumber:   teks(nopolis),
		ClaimNumber:    teks(noklaim),
		ClaimReference: teks(idpega),
		Type:           teks(tipe),
		Note:           teks(notes),
		AcceptStatus:   teks(status),
		CreatedBy:      teks(dibuatOleh),
	}
	if dibuatPada.Valid {
		p.InputDate = dibuatPada.Time
		p.CreatedAt = dibuatPada.Time
	}
	p.ChangeDetail = decodeChangeDetail(p.Type, teks(lama), teks(baru))
	p.ChangeDetail.ObjectName = teks(namaObjek)
	p.ChangeDetail.BranchName = teks(namaCabang)

	return p, nil
}

// encodeChangeDetail menyusun isi OLD_DATA dan NEW_DATA menurut tipe proteksi.
//
// Tipe selain '7' dan '8' TIDAK menyimpan apa pun di kedua kolom — layar tidak menampilkan
// panelnya, sehingga isian yang kebetulan terkirim tidak punya arti dan tidak boleh
// tersimpan diam-diam.
func encodeChangeDetail(protectionType string, d inputreqprotection.ChangeDetail) (any, any) {
	switch strings.TrimSpace(protectionType) {
	case inputreqprotection.TypeChangeLossDate:
		return nullIfEmpty(formatTanggal(d.LossDateBefore)), nullIfEmpty(formatTanggal(d.LossDateAfter))
	case inputreqprotection.TypeChangeCauseOfLoss:
		return nullIfEmpty(d.CauseOfLossID), nullIfEmpty(d.CauseOfLossMasterID)
	default:
		return nil, nil
	}
}

// decodeChangeDetail membaca OLD_DATA dan NEW_DATA menurut tipe proteksi.
//
// Tanggal yang tidak dapat dibaca menjadi nil, bukan galat: baris warisan dapat memuat
// bentuk lain, dan satu baris berformat asing tidak boleh mematikan seluruh daftar.
func decodeChangeDetail(protectionType, lama, baru string) inputreqprotection.ChangeDetail {
	switch strings.TrimSpace(protectionType) {
	case inputreqprotection.TypeChangeLossDate:
		return inputreqprotection.ChangeDetail{
			LossDateBefore: bacaTanggal(lama),
			LossDateAfter:  bacaTanggal(baru),
		}
	case inputreqprotection.TypeChangeCauseOfLoss:
		return inputreqprotection.ChangeDetail{
			CauseOfLossID:       lama,
			CauseOfLossMasterID: baru,
		}
	default:
		return inputreqprotection.ChangeDetail{}
	}
}

func formatTanggal(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(tanggalTeks)
}

func bacaTanggal(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parsed, err := time.Parse(tanggalTeks, s)
	if err != nil {
		return nil
	}
	return &parsed
}

// kunci merapikan nomor proteksi menjadi bentuk yang dibandingkan kueri.
//
// Kueri membandingkannya dengan `UPPER(TRIM(ID)) = :n`, sehingga perapiannya harus terjadi
// DI SINI juga. Membungkusnya lagi dengan fungsi di dalam SQL tidak salah, tetapi membuat
// dua tempat yang harus sepakat — dan yang tidak sepakat menghasilkan pencarian yang selalu
// gagal tanpa satu pun galat.
func kunci(number string) string {
	return strings.ToUpper(strings.TrimSpace(number))
}

// searchArgs menyusun keempat argumen kotak pencarian.
//
// # Kenapa empat, bukan satu
//
// Kata pencarian muncul EMPAT KALI di dalam kueri — sekali sebagai penentu apakah
// pencariannya aktif, tiga kali sebagai pola LIKE. Driver mengikat argumen menurut urutan
// KEMUNCULAN penanda, bukan menurut nomornya, sehingga mengirimnya sekali menghasilkan
// ORA-01008.
//
// # Kenapa polanya dibentuk di Go
//
// Merangkai `'%' || :1 || '%'` di dalam SQL membuat tanda persen dan garis bawah yang
// diketik pengguna menjadi WILDCARD tanpa disengaja — dan nomor polis memang memuat tanda
// baca. Di sini keduanya di-escape lebih dulu.
func searchArgs(search string) []any {
	search = strings.TrimSpace(search)
	if search == "" {
		// Keempatnya NULL: cabang `:1 IS NULL` yang menyala, dan ketiga LIKE tidak pernah
		// diuji.
		return []any{nil, nil, nil, nil}
	}

	pola := "%" + escapeLike(strings.ToUpper(search)) + "%"
	return []any{strings.ToUpper(search), pola, pola, pola}
}

// escapeLike menetralkan wildcard LIKE di dalam kata pencarian.
//
// Backslash TIDAK dipakai sebagai penanda escape di sini — kueri tidak memakai klausa
// ESCAPE — sehingga yang dilakukan adalah membuang karakternya, bukan meloloskannya.
// Konsekuensinya: mencari "50%" menemukan yang memuat "50". Itu perilaku yang masuk akal
// bagi kotak pencarian bebas, dan jauh lebih baik daripada wildcard yang tidak diminta.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "%", "")
	return strings.ReplaceAll(s, "_", "")
}

func teks(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}

// nullIfEmpty mengirim NULL alih-alih teks kosong.
//
// Oracle memperlakukan teks kosong SEBAGAI NULL pada kolom VARCHAR2, sehingga keduanya
// berakhir sama di sana. Yang dijaga di sini adalah niatnya tetap terbaca dari kode, dan
// adapter ini tetap benar bila kelak dijalankan terhadap PostgreSQL — yang MEMBEDAKAN
// keduanya (`D-20`, `D-24`).
func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

// isUniqueViolation menyatakan galat berasal dari pelanggaran constraint unik.
//
// Diperiksa lewat TEKS galatnya, bukan lewat tipe driver: `godror` tidak mengekspor tipe
// galat yang stabil untuk ini, dan memeriksa kode ORA di dalam pesannya adalah jalan yang
// sama yang ditempuh adapter modul lain.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "ORA-00001")
}
