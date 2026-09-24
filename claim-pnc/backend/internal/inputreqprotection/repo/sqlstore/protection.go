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
// Ia menulis kolom PEMBUATAN saja: `OPEN_PROTECTION_ID`, `POLICY_NO`, `CLAIM_NO`,
// `ID_CLAIM`, `PROTECTION_TYPE_ID`, `CREATE_DATE`, `CREATED_BY`, `NOTES`, `OLD_DATA`,
// `NEW_DATA`, `OBJECT_NAME`, `BRANCH_NAME`, `STATUS_ACTIVE`.
//
// Kolom AKSEPTASI — `APPROVAL_STATUS`, `RESOLVED_BY`, dan `RESOLVED_DATETIME` —
// tidak pernah disentuh di sini; ketiganya milik modul `inboxacceptopenprotection`.
// Pembagian itu yang menjaga `P-1` tetap berlaku meski dua modul menyentuh satu tabel.
//
// Master `POOLDATA.M_CLAIM_PROTECTION_TYPE` dibaca lewat LEFT JOIN untuk nama tipenya, dan
// TIDAK PERNAH ditulis: modul ini memakai masternya, bukan mengelolanya.
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

// Create menyimpan permintaan baru beserta nomor yang terbit.
//
// # Kenapa TIDAK ADA percobaan ulang di sini
//
// Versi sebelumnya menurunkan nomor dari `MAX(...)+1` dan mencoba ulang sampai tiga kali
// saat kena `ORA-00001`, karena dua permintaan bersamaan dapat membaca nilai yang sama.
//
// Sejak `POOLDATA.CLAIM_PROTECTION_SEQ` dibuat (Work Owner, 2026-09-24), setiap pemanggil
// `NEXTVAL` menerima nilai yang berbeda TANPA membaca isi tabel. Bentroknya tidak lagi
// mungkin, sehingga percobaan ulangnya dihapus — bukan disederhanakan.
//
// Percobaan ulang yang dipertahankan setelah sebabnya hilang justru berbahaya: ia
// menyembunyikan bentrok yang sebenarnya menandakan hal lain, misalnya nomor yang disisipkan
// tangan ke tabel.
//
// # Kenapa tetap satu transaksi
//
// Penerbitan nomor dan penyisipan barisnya tetap berada DI DALAM SATU TRANSAKSI. Alasannya
// berubah: bukan lagi untuk mempersempit balapan, melainkan supaya kegagalan penyisipan
// tidak menyisakan pekerjaan setengah jalan.
//
// Perlu dicatat apa yang TIDAK dijamin transaksi itu: nomor yang sudah diambil dari sequence
// TIDAK kembali saat rollback — itu sifat sequence, di Oracle maupun PostgreSQL. Deret nomor
// proteksi karena itu dapat berlubang, dan lubang itu bukan tanda kerusakan.
func (r *Repo) Create(
	ctx context.Context,
	draft inputreqprotection.Draft,
	claim inputreqprotection.Claim,
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

	var urut int64
	if err := tx.QueryRowContext(ctx, query("protection_next_sequence")).Scan(&urut); err != nil {
		return inputreqprotection.Protection{}, fmt.Errorf(
			"inputreqprotection/sqlstore: menerbitkan nomor proteksi: %w", err)
	}

	nomor := inputreqprotection.FormatNumber(at.Year(), urut)
	detail := deriveChangeDetail(draft, claim)
	lama, baru := encodeChangeDetail(draft.Type, detail)

	if _, err := tx.ExecContext(ctx, query("protection_insert"),
		nomor,
		// Nomor polis DITURUNKAN dari klaim, bukan diterima dari form — persis yang
		// dilakukan `Activity/OpenProtection-Act.xml` saat klaim dicari.
		nullIfEmpty(claim.PolicyNumber),
		nullIfEmpty(draft.ClaimNumber),
		// ID_CLAIM DITURUNKAN dari nomor klaim, bukan diterima dari form.
		//
		// Work Owner menegaskan 2026-09-24 bahwa ClaimNo dan ClaimID berisi nilai yang
		// sama. Menanyakannya dua kali akan membuat keduanya berbeda cepat atau lambat —
		// tanpa galat, hanya proteksi yang menunjuk dua klaim berbeda.
		nullIfEmpty(draft.ClaimNumber),
		nullIfEmpty(draft.Type),
		at.UTC(),
		nullIfEmpty(by),
		nullIfEmpty(draft.Note),
		lama,
		baru,
		nullIfEmpty(detail.ObjectName),
		nullIfEmpty(detail.BranchName),
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
		PolicyNumber:   claim.PolicyNumber,
		ClaimNumber:    draft.ClaimNumber,
		ClaimReference: draft.ClaimNumber,
		Type:           draft.Type,
		InputDate:      at,
		Note:           draft.Note,
		AcceptStatus:   inputreqprotection.AcceptPending,
		CreatedBy:      by,
		CreatedAt:      at,
		ChangeDetail:   detail,
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
	claim inputreqprotection.Claim,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	detail := deriveChangeDetail(draft, claim)
	lama, baru := encodeChangeDetail(draft.Type, detail)

	hasil, err := r.db.ExecContext(ctx, query("protection_update"),
		// Nomor polis DITURUNKAN dari klaim, bukan diterima dari form — persis yang
		// dilakukan `Activity/OpenProtection-Act.xml` saat klaim dicari.
		nullIfEmpty(claim.PolicyNumber),
		nullIfEmpty(draft.ClaimNumber),
		// ID_CLAIM DITURUNKAN dari nomor klaim, bukan diterima dari form.
		//
		// Work Owner menegaskan 2026-09-24 bahwa ClaimNo dan ClaimID berisi nilai yang
		// sama. Menanyakannya dua kali akan membuat keduanya berbeda cepat atau lambat —
		// tanpa galat, hanya proteksi yang menunjuk dua klaim berbeda.
		nullIfEmpty(draft.ClaimNumber),
		nullIfEmpty(draft.Type),
		nullIfEmpty(draft.Note),
		lama,
		baru,
		nullIfEmpty(detail.ObjectName),
		nullIfEmpty(detail.BranchName),
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
		namaTipe                          sql.NullString
		dibuatPada                        sql.NullTime
		notes, status, dibuatOleh         sql.NullString
		lama, baru, namaObjek, namaCabang sql.NullString
	)

	// namaTipe datang dari LEFT JOIN ke master, sehingga ia NULL untuk dua keadaan yang
	// berbeda: kode tipenya kosong, atau kodenya ada tetapi tidak terdaftar di master.
	// Keduanya diperlakukan sama di sini — yang membedakannya adalah lapisan tampilan,
	// yang menampilkan kode apa adanya saat namanya tidak ada.
	if err := row.Scan(&id, &nopolis, &noklaim, &idpega, &tipe, &namaTipe,
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
		TypeName:       teks(namaTipe),
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

// TypeRepo membaca master tipe proteksi.
//
// Terpisah dari Repo dengan sengaja — lihat alasannya pada deklarasi seam-nya di
// `inputreqprotection.TypeRepo`. Keduanya memakai koneksi yang SAMA, sehingga proteksi dan
// nama tipenya tidak pernah datang dari portal yang berbeda.
type TypeRepo struct {
	db *sql.DB
}

// NewTypeRepo membentuk repo master; db wajib koneksi portal yang sama dengan Repo.
func NewTypeRepo(db *sql.DB) *TypeRepo { return &TypeRepo{db: db} }

// ListTypes membaca seluruh tipe proteksi beserta namanya.
//
// Baris ber-kode kosong DILEWATI. Ia tidak dapat dipilih pengguna — menyimpannya akan
// menghasilkan proteksi tanpa tipe, yang kemudian lenyap dari kedua antrean akseptasi
// karena penyaringnya membandingkan kode.
func (r *TypeRepo) ListTypes(ctx context.Context) ([]inputreqprotection.ProtectionType, error) {
	rows, err := r.db.QueryContext(ctx, query("protection_type_list"))
	if err != nil {
		return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca master tipe proteksi: %w", err)
	}
	defer rows.Close()

	daftar := make([]inputreqprotection.ProtectionType, 0, 16)
	for rows.Next() {
		var kode, nama sql.NullString
		if err := rows.Scan(&kode, &nama); err != nil {
			return nil, fmt.Errorf(
				"inputreqprotection/sqlstore: memindai master tipe proteksi: %w", err)
		}
		id := teks(kode)
		if id == "" {
			continue
		}
		daftar = append(daftar, inputreqprotection.ProtectionType{ID: id, Name: teks(nama)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inputreqprotection/sqlstore: membaca baris master tipe proteksi: %w", err)
	}
	return daftar, nil
}

// ── Pencarian klaim ──────────────────────────────────────────────────────────────

// ClaimRepo mencari klaim yang hendak ditaut.
//
// # Ia HANYA membaca
//
// Ketiga tabel yang dibacanya dimiliki Pega selama masa paralel. `P-1` melarang dua sistem
// MENULIS satu tabel; membaca tidak dilarang. Tidak ada satu pun method di sini yang
// menulis, dan itu bukan kebetulan melainkan batas yang dijaga.
type ClaimRepo struct {
	db *sql.DB
}

// NewClaimRepo membentuk repo pencarian klaim; db wajib koneksi portal yang dituju.
func NewClaimRepo(db *sql.DB) *ClaimRepo { return &ClaimRepo{db: db} }

// FindClaim mencari klaim menurut nomornya.
func (r *ClaimRepo) FindClaim(
	ctx context.Context,
	number string,
) (inputreqprotection.Claim, error) {
	var (
		id                 string
		polis, tertanggung sql.NullString
		dol                sql.NullTime
		penyebab           sql.NullString
		cabang, namaObjek  sql.NullString
	)

	err := r.db.QueryRowContext(ctx, query("claim_find"), kunci(number)).
		Scan(&id, &polis, &tertanggung, &dol, &penyebab, &cabang, &namaObjek)
	if errors.Is(err, sql.ErrNoRows) {
		return inputreqprotection.Claim{}, inputreqprotection.ErrClaimNotFound
	}
	if err != nil {
		return inputreqprotection.Claim{}, fmt.Errorf(
			"inputreqprotection/sqlstore: mencari klaim: %w", err)
	}

	klaim := inputreqprotection.Claim{
		Number:       strings.TrimSpace(id),
		PolicyNumber: teks(polis),
		InsuredName:  teks(tertanggung),
		CauseOfLoss:  teks(penyebab),
		BranchName:   teks(cabang),
		ObjectName:   teks(namaObjek),
	}
	if dol.Valid {
		waktu := dol.Time
		klaim.LossDate = &waktu
	}
	return klaim, nil
}

// deriveChangeDetail menyusun isi panel Detail Perubahan dari DUA sumber.
//
// # Kenapa dua sumber, bukan satu
//
// Panel di Pega punya dua sisi yang asalnya berbeda, dan itulah inti cacat yang diperbaiki
// pada 2026-09-24:
//
//	Current Date Of Loss   dari KLAIM     (`.ClaimDataProtect.BeforeDateOfLoss`)
//	Next Date Of Loss      dari PENGGUNA
//	Cause Of Loss Dipilih  dari KLAIM
//	Next Cause Of Loss     dari PENGGUNA
//	Object Name            dari KLAIM
//	Branch Name            dari KLAIM
//
// Implementasi pertama menerima KEENAMNYA dari form. Akibatnya seseorang dapat menyimpan
// permintaan "ubah DOL" yang menyebut DOL sebelum yang tidak pernah menjadi DOL klaim itu —
// dan petugas akseptasi menyetujuinya tanpa cara mengetahuinya.
//
// Fungsi ini dipanggil di adapter, bukan di lapisan atas, supaya TIDAK ADA jalur penyimpanan
// yang dapat melewatinya.
func deriveChangeDetail(
	draft inputreqprotection.Draft,
	claim inputreqprotection.Claim,
) inputreqprotection.ChangeDetail {
	return inputreqprotection.ChangeDetail{
		// Sisi KLAIM.
		LossDateBefore: claim.LossDate,
		CauseOfLossID:  claim.CauseOfLoss,
		ObjectName:     claim.ObjectName,
		BranchName:     claim.BranchName,

		// Sisi PENGGUNA.
		LossDateAfter:       draft.Change.LossDateAfter,
		CauseOfLossMasterID: draft.Change.CauseOfLossAfter,
	}
}
