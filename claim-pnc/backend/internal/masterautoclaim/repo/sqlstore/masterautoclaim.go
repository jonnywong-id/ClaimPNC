// Package sqlstore memenuhi seam masterautoclaim.Store dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterautoclaim"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca dan menulis POOLDATA.M_AUTO_CLAIM_PNC, serta MEMBACA empat tabel acuan.
//
// Satu struct memenuhi kedua seam — masterautoclaim.Repo dan masterautoclaim.LookupRepo
// — karena keduanya selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah
// interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring komite memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel.
// Alasannya ada pada berkas .sql: yang ditempel di sistem lama adalah teks SQL berisi
// operator ID, dan itu pola `{ASIS:...}` yang dilarang tanpa perkecualian.
func (r *Repo) List(ctx context.Context, filter masterautoclaim.Filter) ([]masterautoclaim.AutoClaim, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if filter.CommitteeOnly {
		rows, err = r.db.QueryContext(ctx, getQuery("auto_claim_list_by_committee"),
			string(filter.Status), strings.TrimSpace(filter.CommitteeID))
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("auto_claim_list"), string(filter.Status))
	}
	if err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterautoclaim.AutoClaim
	for rows.Next() {
		ac, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterautoclaim/sqlstore: membaca baris daftar: %w", err)
		}
		result = append(result, ac)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu baris berdasarkan INISIALID-nya.
func (r *Repo) Get(ctx context.Context, initial string) (masterautoclaim.AutoClaim, error) {
	row := r.db.QueryRowContext(ctx, getQuery("auto_claim_get"), strings.TrimSpace(initial))

	ac, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterautoclaim.AutoClaim{}, masterautoclaim.ErrNotFound
	}
	if err != nil {
		return masterautoclaim.AutoClaim{}, fmt.Errorf("masterautoclaim/sqlstore: membaca %q: %w", initial, err)
	}
	return ac, nil
}

// Insert menolak INISIALID yang sudah dipakai, lalu menyisipkan barisnya.
//
// Keduanya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi:
// pemeriksaan dan penyisipan di sini bukan dua perkara melainkan satu, dan memisahkannya
// membuka kembali lubang balapan yang justru sedang dipersempit.
//
// SEBERAPA JAUH LUBANG ITU TERTUTUP — dinyatakan supaya tidak dikira selesai. FOR UPDATE
// tidak dapat mengunci baris yang belum ada: bila dua penambahan atas kode yang sama
// sama-sama menemukan nol baris, keduanya lolos. Yang benar-benar menutupnya adalah
// constraint unik pada INISIALID, dan itu menunggu DDL (R-08) serta prosedur perubahan
// skema (D-63).
func (r *Repo) Insert(ctx context.Context, ac masterautoclaim.AutoClaim) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("masterautoclaim/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
	// kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	taken, err := initialTaken(ctx, tx, ac.Initial)
	if err != nil {
		return err
	}
	if taken {
		return fmt.Errorf("%w: %q", masterautoclaim.ErrInitialTaken, ac.Initial)
	}

	if _, err := tx.ExecContext(ctx, getQuery("auto_claim_insert"),
		ac.Initial,
		ac.ReceiverName,
		ac.BankName,
		ac.AccountNumber,
		ac.MaxPercent,
		ac.ReporterPIC,
		ac.ReporterEmail,
		ac.ClaimAllowed,
		ac.ReceiverAddress,
		ac.SubmittedBy,
		ac.Committee,
		string(ac.Status),
		ac.ClientID,
		ac.ClientName,
	); err != nil {
		return fmt.Errorf("masterautoclaim/sqlstore: menyisipkan %q: %w", ac.Initial, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("masterautoclaim/sqlstore: menutup transaksi sisip: %w", err)
	}
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// NAMA_PENERIMA tidak ikut — lihat auto_claim_update pada berkas .sql.
func (r *Repo) Update(ctx context.Context, ac masterautoclaim.AutoClaim) error {
	result, err := r.db.ExecContext(ctx, getQuery("auto_claim_update"),
		ac.BankName,
		ac.AccountNumber,
		ac.MaxPercent,
		ac.ReporterPIC,
		ac.ReporterEmail,
		ac.ClaimAllowed,
		ac.ReceiverAddress,
		string(ac.Status),
		ac.SubmittedBy,
		ac.Committee,
		ac.ClientID,
		ac.ClientName,
		strings.TrimSpace(ac.Initial),
	)
	if err != nil {
		return fmt.Errorf("masterautoclaim/sqlstore: memperbarui %q: %w", ac.Initial, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh
	// satu baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan
	// mengatakan "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return masterautoclaim.ErrNotFound
	}
	return nil
}

// CheckTable memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("auto_claim_check_table"))
	if err != nil {
		return fmt.Errorf("masterautoclaim/sqlstore: POOLDATA.M_AUTO_CLAIM_PNC tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// SearchBusinessSources mencari Sumber Bisnis pada POOLDATA.AGENT.
func (r *Repo) SearchBusinessSources(ctx context.Context, keyword string) ([]masterautoclaim.BusinessSource, error) {
	pattern, exact := lookupArguments(keyword)

	rows, err := r.db.QueryContext(ctx, getQuery("auto_claim_business_source_search"), pattern, exact)
	if err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: mencari sumber bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterautoclaim.BusinessSource
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("masterautoclaim/sqlstore: membaca sumber bisnis: %w", err)
		}
		source := masterautoclaim.BusinessSource{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		}
		// Baris tanpa ID tidak dapat dipilih pengguna dan hanya akan menghasilkan master
		// auto claim tanpa kunci. Ia dilewati di sini, bukan dibiarkan muncul di layar.
		if source.ID == "" {
			continue
		}
		result = append(result, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: menelusuri sumber bisnis: %w", err)
	}
	return result, nil
}

// SearchClients mencari tertanggung pada POOLDATA.CLIENT.
func (r *Repo) SearchClients(ctx context.Context, keyword string) ([]masterautoclaim.Client, error) {
	pattern, exact := lookupArguments(keyword)

	rows, err := r.db.QueryContext(ctx, getQuery("auto_claim_client_search"), pattern, exact)
	if err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: mencari client: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterautoclaim.Client
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("masterautoclaim/sqlstore: membaca client: %w", err)
		}
		client := masterautoclaim.Client{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		}
		if client.ID == "" {
			continue
		}
		result = append(result, client)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: menelusuri client: %w", err)
	}
	return result, nil
}

// ListBanks membaca seluruh bank pada GENERAL.LST_BANK_GROUP.
func (r *Repo) ListBanks(ctx context.Context) ([]masterautoclaim.Bank, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("auto_claim_bank_list"))
	if err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: membaca daftar bank: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterautoclaim.Bank
	for rows.Next() {
		bank, err := scanBank(rows)
		if err != nil {
			return nil, fmt.Errorf("masterautoclaim/sqlstore: membaca baris bank: %w", err)
		}
		// Bank tanpa nama tidak dapat disimpan ke BANK_PENERIMA, dan bank tanpa kode
		// tidak akan pernah lolos pencocokan FindBankByName. Keduanya dilewati.
		if bank.Name == "" || bank.Code == "" {
			continue
		}
		result = append(result, bank)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterautoclaim/sqlstore: menelusuri daftar bank: %w", err)
	}
	return result, nil
}

// FindBankByName mencari satu bank menurut namanya.
func (r *Repo) FindBankByName(ctx context.Context, name string) (masterautoclaim.Bank, error) {
	clean := strings.ToUpper(strings.TrimSpace(name))
	if clean == "" {
		return masterautoclaim.Bank{}, masterautoclaim.ErrBankNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery("auto_claim_bank_by_name"), clean)

	bank, err := scanBank(row)
	if errors.Is(err, sql.ErrNoRows) {
		return masterautoclaim.Bank{}, masterautoclaim.ErrBankNotFound
	}
	if err != nil {
		return masterautoclaim.Bank{}, fmt.Errorf("masterautoclaim/sqlstore: membaca bank %q: %w", name, err)
	}
	return bank, nil
}

// Committee membaca penyetuju komite untuk baris baru.
//
// Tidak ada baris BUKAN galat: sistem lama pun menyimpan KOMITE kosong dalam keadaan
// itu. Yang mencatatnya sebagai peringatan adalah lapisan aplikasi, yang tahu akibatnya
// bagi pengguna.
func (r *Repo) Committee(ctx context.Context) (string, error) {
	var operator sql.NullString

	err := r.db.QueryRowContext(ctx, getQuery("auto_claim_committee")).Scan(&operator)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("masterautoclaim/sqlstore: membaca penyetuju komite: %w", err)
	}
	return strings.TrimSpace(operator.String), nil
}

// lookupArguments menyusun kedua parameter kueri lookup dari satu kata kunci.
//
// Perlakuannya mengikuti Activity/GetClientName step 3 apa adanya:
//
//	@toUpperCase(@replaceAll(Param.name, ".", ""))
//
// Tanda persen dan garis bawah pada kata kunci DILOLOSKAN sebelum dipakai LIKE. Tanpa
// itu, pengguna yang mengetik "%" menarik seluruh tabel, dan yang mengetik "_" mencocoki
// karakter apa pun — bukan celah keamanan, tetapi hasil yang tidak dapat dijelaskan
// kepada yang mengetiknya. `ESCAPE '\'` disebut eksplisit di kueri karena Oracle tidak
// punya karakter pelolos bawaan pada LIKE.
func lookupArguments(keyword string) (pattern, exact string) {
	clean := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(keyword), ".", ""))

	escaped := clean
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%", clean
}

type scanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris master auto claim.
//
// Seluruh kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang
// diketahui (R-08).
//
// Urutan kolomnya mengikuti berkas .sql, dan ketiga belas kolom dibaca pada urutan yang
// sama oleh auto_claim_list, auto_claim_list_by_committee, dan auto_claim_get. Itu yang
// membuat satu fungsi cukup untuk ketiganya — dan yang membuat uji urutan kolom pada
// query_test.go layak ada.
func scanRow(p scanner) (masterautoclaim.AutoClaim, error) {
	var (
		initial, receiverName, bankName, accountNumber, maxPercent sql.NullString
		reporterPIC, reporterEmail, claimAllowed, address          sql.NullString
		committee, status, clientID, clientName                    sql.NullString
	)
	if err := p.Scan(
		&initial, &receiverName, &bankName, &accountNumber, &maxPercent,
		&reporterPIC, &reporterEmail, &claimAllowed, &address,
		&committee, &status, &clientID, &clientName,
	); err != nil {
		return masterautoclaim.AutoClaim{}, err
	}
	return masterautoclaim.AutoClaim{
		Initial:         strings.TrimSpace(initial.String),
		ReceiverName:    strings.TrimSpace(receiverName.String),
		BankName:        strings.TrimSpace(bankName.String),
		AccountNumber:   strings.TrimSpace(accountNumber.String),
		MaxPercent:      strings.TrimSpace(maxPercent.String),
		ReporterPIC:     strings.TrimSpace(reporterPIC.String),
		ReporterEmail:   strings.TrimSpace(reporterEmail.String),
		ClaimAllowed:    strings.TrimSpace(claimAllowed.String),
		ReceiverAddress: strings.TrimSpace(address.String),
		Committee:       strings.TrimSpace(committee.String),
		Status:          masterautoclaim.ApprovalStatus(strings.TrimSpace(status.String)),
		ClientID:        strings.TrimSpace(clientID.String),
		ClientName:      strings.TrimSpace(clientName.String),
	}, nil
}

// scanBank membaca satu baris bank.
//
// USERINPUT sengaja TIDAK ikut dibaca pada scanRow maupun di sini: tidak ada satu pun
// layar yang menampilkannya, dan membacanya berarti mengirim nama operator ke peramban
// tanpa ada yang memakainya.
func scanBank(p scanner) (masterautoclaim.Bank, error) {
	var code, name sql.NullString
	if err := p.Scan(&code, &name); err != nil {
		return masterautoclaim.Bank{}, err
	}
	return masterautoclaim.Bank{
		Code: strings.TrimSpace(code.String),
		Name: strings.TrimSpace(name.String),
	}, nil
}

// initialTaken memeriksa INISIALID sudah dipakai, di dalam transaksi yang berjalan.
func initialTaken(ctx context.Context, tx *sql.Tx, initial string) (bool, error) {
	rows, err := tx.QueryContext(ctx, getQuery("auto_claim_list_initial_locked"), strings.TrimSpace(initial))
	if err != nil {
		return false, fmt.Errorf("masterautoclaim/sqlstore: memeriksa %q: %w", initial, err)
	}
	defer func() { _ = rows.Close() }()

	found := rows.Next()
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("masterautoclaim/sqlstore: menelusuri pemeriksaan %q: %w", initial, err)
	}
	return found, nil
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("masterautoclaim/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterautoclaim/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterautoclaim/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterautoclaim/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, rows := range body {
			if strings.HasPrefix(strings.TrimSpace(rows), "--") {
				continue
			}
			statement = append(statement, rows)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, rows := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(rows); strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}

var _ masterautoclaim.Store = (*Repo)(nil)
