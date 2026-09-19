package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterrekening"
)

// defaultLimit membatasi jumlah baris yang dibaca bila pemanggil tidak menyebut batasnya.
//
// Ia ada supaya tidak ada jalan untuk membaca seluruh tabel tanpa sengaja. Sistem lama
// memotong hasilnya di 500 baris lewat pyMaxRecords pada 54 dari 56 laporan
// (09-DATABASE-STRATEGY §6.3); angka itu dipakai kembali di sini supaya halaman
// pertama berisi jumlah baris yang sama dengan yang biasa dilihat pengguna.
const defaultLimit = 500

// maxLimit menahan permintaan batas yang tidak masuk akal dari klien.
const maxLimit = 2000

// Repo membaca dan menulis POOLDATA.LST_ACCOUNT.
//
// PENULIS TUNGGAL (ADR-0004, P-1). Selama masa paralel, tabel ini masih ditulis Pega.
// Memindahkan Master Rekening ke aplikasi ini berarti kepemilikan tulis ikut berpindah
// — layar lamanya wajib dimatikan pada saat yang sama, bukan sesudahnya. Itu bagian
// dari rencana rollout, bukan detail yang dapat diurus belakangan.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca rekening yang cocok dengan filter beserta jumlah seluruh baris yang
// cocok sebelum dipotong paginasi.
func (r *Repo) List(ctx context.Context, f masterrekening.Filter) ([]masterrekening.Account, int, error) {
	saring := filterInput(f)

	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("account_count"), saring...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("masterrekening/sqlstore: menghitung rekening: %w", err)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	skip := f.Offset
	if skip < 0 {
		skip = 0
	}

	argumen := append(append([]any(nil), saring...), skip, limit)
	rows, err := r.db.QueryContext(ctx, getQuery("account_list"), argumen...)
	if err != nil {
		return nil, 0, fmt.Errorf("masterrekening/sqlstore: membaca daftar rekening: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]masterrekening.Account, 0, limit)
	for rows.Next() {
		acct, err := scanAccount(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("masterrekening/sqlstore: menelusuri daftar rekening: %w", err)
	}
	return result, total, nil
}

// filterInput menyusun sepuluh argumen saringan dalam urutan yang dituntut kedua
// kueri.
//
// Setiap saringan muncul dua kali di dalam SQL — sekali pada pemeriksaan IS NULL,
// sekali pada perbandingannya — sehingga nilainya dikirim dua kali pula. Nomor bind
// sengaja dibedakan, bukan diulang, supaya tidak bergantung pada tafsir driver
// terhadap bind bernomor sama.
func filterInput(f masterrekening.Filter) []any {
	status := emptyToNil(string(f.Status))
	nomor := emptyToNil(f.Number)
	owner := emptyToNil(f.OwnerName)
	bank := emptyToNil(f.BankName)

	var committee any
	if f.MyCommitteeOnly {
		committee = emptyToNil(f.CommitteeIdentity)
	}

	return []any{
		status, status,
		nomor, nomor,
		owner, owner,
		bank, bank,
		committee, committee,
	}
}

// emptyToNil mengubah teks kosong menjadi NULL.
//
// Itulah yang membuat satu kueri melayani seluruh gabungan saringan tanpa merangkai
// teks SQL: saringan yang tidak diisi menjadi NULL, dan `:n IS NULL OR …` membuatnya
// tidak mempersempit apa pun.
func emptyToNil(s string) any {
	if trimmed := strings.TrimSpace(s); trimmed != "" {
		return trimmed
	}
	return nil
}

// Get membaca satu rekening.
func (r *Repo) Get(ctx context.Context, k masterrekening.Key) (masterrekening.Account, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("account_get"), k.Number, k.BankCode)
	acct, err := scanAccount(rows)
	if errors.Is(err, sql.ErrNoRows) {
		return masterrekening.Account{}, masterrekening.ErrNotFound
	}
	if err != nil {
		return masterrekening.Account{}, err
	}
	return acct, nil
}

// FindByNumber membaca seluruh baris dengan nomor rekening tertentu, tanpa peduli banknya.
func (r *Repo) FindByNumber(ctx context.Context, nomor string) ([]masterrekening.Account, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("account_find_by_number"), strings.TrimSpace(nomor))
	if err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: mencari nomor rekening: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterrekening.Account
	for rows.Next() {
		acct, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, acct)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: menelusuri hasil pencarian nomor: %w", err)
	}
	return result, nil
}

// Save menyisipkan rekening baru.
func (r *Repo) Save(ctx context.Context, acct masterrekening.Account) error {
	_, err := r.db.ExecContext(ctx, getQuery("account_insert"),
		acct.Number,
		acct.OwnerName,
		acct.BankName,
		acct.BankBranch,
		acct.BankAddress,
		acct.BankCode,
		acct.AccountType,
		activeFlag(acct.Active),
		string(acct.Status),
		acct.CommitteeApproval,
		acct.Email,
		acct.SubmitterEmail,
		acct.Phone,
		acct.NIK,
		acct.DocumentID,
		acct.Note,
		acct.CreatedBy,
		acct.CreatedAt.UTC(),
		acct.UpdatedBy,
		acct.ServiceStatus,
		acct.PreviousBankCode,
		acct.PreviousNumber,
		acct.PreviousOwnerName,
	)
	if err != nil {
		return fmt.Errorf("masterrekening/sqlstore: menyisipkan rekening: %w", err)
	}
	return nil
}

// Update menulis ulang rekening yang sudah ada.
func (r *Repo) Update(ctx context.Context, acct masterrekening.Account) error {
	var decided any
	if acct.DecidedAt != nil {
		decided = acct.DecidedAt.UTC()
	}

	result, err := r.db.ExecContext(ctx, getQuery("account_update"),
		acct.OwnerName,
		acct.BankName,
		acct.BankBranch,
		acct.BankAddress,
		acct.AccountType,
		activeFlag(acct.Active),
		string(acct.Status),
		acct.CommitteeApproval,
		decided,
		acct.Email,
		acct.SubmitterEmail,
		acct.Phone,
		acct.NIK,
		acct.DocumentID,
		acct.Note,
		acct.UpdatedBy,
		acct.ServiceStatus,
		acct.CashierAccountID,
		acct.CashierResponse,
		acct.PreviousBankCode,
		acct.PreviousNumber,
		acct.PreviousOwnerName,
		acct.Number,
		acct.BankCode,
	)
	if err != nil {
		return fmt.Errorf("masterrekening/sqlstore: memperbarui rekening: %w", err)
	}
	return ensureRowTouched(result, masterrekening.ErrNotFound)
}

// ClearRejected membuang baris bekas penolakan komite.
//
// Syarat APPROVAL = '2' ditegakkan di dalam kueri. Bila tidak ada baris yang tersentuh,
// artinya barisnya tidak ada ATAU statusnya bukan ditolak; keduanya dijawab
// ErrAlreadyDecided karena keduanya berarti hal yang sama bagi pemanggil: baris ini
// tidak boleh dibuang.
func (r *Repo) ClearRejected(ctx context.Context, k masterrekening.Key) error {
	result, err := r.db.ExecContext(ctx, getQuery("account_clear_rejected"), k.Number, k.BankCode)
	if err != nil {
		return fmt.Errorf("masterrekening/sqlstore: membuang pengajuan yang ditolak: %w", err)
	}
	return ensureRowTouched(result, masterrekening.ErrAlreadyDecided)
}

// CheckTable menyatakan apakah POOLDATA.LST_ACCOUNT dapat dijangkau akun aplikasi.
//
// Dipakai mode -periksa pada binary, yang menguji kesiapan tanpa menulis apa pun.
func (r *Repo) CheckTable(ctx context.Context) (bool, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("account_check_table")).Scan(&total); err != nil {
		return false, fmt.Errorf("masterrekening/sqlstore: memeriksa tabel LST_ACCOUNT: %w", err)
	}
	return total > 0, nil
}

func ensureRowTouched(result sql.Result, bila error) error {
	// Sebagian driver tidak melaporkan jumlah baris yang tersentuh. Bila begitu,
	// ketiadaan angka BUKAN bukti bahwa tidak ada yang berubah — memperlakukannya
	// sebagai galat akan menolak penulisan yang sebenarnya berhasil.
	total, err := result.RowsAffected()
	if err != nil {
		return nil
	}
	if total == 0 {
		return bila
	}
	return nil
}

// activeFlag memetakan status aktif ke sandi kolom STS_AKTIF.
//
// Sandinya "Ya" dan "Tidak", BUKAN "1" dan "0". Nilainya diambil dari activity
// SetTipeRekening, yang mengisi daftar pilihan layar lama:
//
//	TempTipeBank.pxResults(<APPEND>).NomorKontrak = "Ya"    → dipakai TempBank.CaseID
//	TempTipeBank.pxResults(<APPEND>).NomorKontrak = "Tidak"     (alias CaseID = STS_AKTIF)
//
// Ia tidak dapat dipilih bebas selama Pega masih membaca kolom yang sama.
func activeFlag(active bool) string {
	if active {
		return "Ya"
	}
	return "Tidak"
}

// readActive menafsirkan isi kolom STS_AKTIF.
//
// Perbandingannya tanpa peduli besar-kecil huruf, dan "1" ikut diterima: kolom ini
// sudah dipakai bertahun-tahun oleh beberapa rule, dan menolak mengenali baris lama
// hanya karena ejaannya berbeda akan menampilkan rekening aktif sebagai nonaktif —
// yang berarti petugas mengira rekening itu tidak dapat dipakai membayar klaim.
func readActive(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "YA", "1", "Y", "AKTIF":
		return true
	default:
		return false
	}
}

// rowScanner menyatukan *sql.Row dan *sql.Rows sehingga satu fungsi pemindaian melayani
// keduanya. Tanpa ini, 27 kolom harus ditulis dua kali dan kedua salinannya harus
// diingat untuk diubah bersama-sama.
type rowScanner interface {
	Scan(to ...any) error
}

func scanAccount(p rowScanner) (masterrekening.Account, error) {
	var (
		acct    masterrekening.Account
		active  sql.NullString
		status  sql.NullString
		decided sql.NullTime
		diinput sql.NullTime
		respons sql.NullString

		name, bank, branch, address, bankCode    sql.NullString
		tipe, committee, email, emailInput, telp sql.NullString
		nik, document, note, olehInput, oleh     sql.NullString
		service, cashierID, flag                 sql.NullString
		previousBank, nomorLama, previousOwner   sql.NullString
	)

	err := p.Scan(
		&acct.Number,
		&name,
		&bank,
		&branch,
		&address,
		&bankCode,
		&tipe,
		&active,
		&status,
		&committee,
		&decided,
		&email,
		&emailInput,
		&telp,
		&nik,
		&document,
		&note,
		&olehInput,
		&diinput,
		&oleh,
		&service,
		&cashierID,
		&respons,
		&flag,
		&previousBank,
		&nomorLama,
		&previousOwner,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return masterrekening.Account{}, err
		}
		return masterrekening.Account{}, fmt.Errorf("masterrekening/sqlstore: membaca baris rekening: %w", err)
	}

	acct.Number = strings.TrimSpace(acct.Number)
	acct.OwnerName = text(name)
	acct.BankName = text(bank)
	acct.BankBranch = text(branch)
	acct.BankAddress = text(address)
	acct.BankCode = text(bankCode)
	acct.AccountType = text(tipe)
	acct.Active = readActive(text(active))
	acct.Status = masterrekening.ApprovalStatus(text(status))
	acct.CommitteeApproval = text(committee)
	acct.Email = text(email)
	acct.SubmitterEmail = text(emailInput)
	acct.Phone = text(telp)
	acct.NIK = text(nik)
	acct.DocumentID = text(document)
	acct.Note = text(note)
	acct.CreatedBy = text(olehInput)
	acct.UpdatedBy = text(oleh)
	acct.ServiceStatus = text(service)
	acct.CashierAccountID = text(cashierID)
	acct.ChangeFlag = text(flag)
	acct.PreviousBankCode = text(previousBank)
	acct.PreviousNumber = text(nomorLama)
	acct.PreviousOwnerName = text(previousOwner)

	// Pemangkasan yang dulu dilakukan SUBSTR/INSTR di dalam SQL kini terjadi di sini.
	acct.CashierResponse = masterrekening.TrimCashierResponse(text(respons))

	if diinput.Valid {
		acct.CreatedAt = diinput.Time.UTC()
	}
	if decided.Valid {
		t := decided.Time.UTC()
		acct.DecidedAt = &t
	}
	return acct, nil
}

func text(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return strings.TrimSpace(n.String)
}

var _ masterrekening.Repo = (*Repo)(nil)
