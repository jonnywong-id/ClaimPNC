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
	selected inputreqprotection.CoverageRow,
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
	detail := deriveChangeDetail(draft, claim, selected)
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
		// sama. Itu benar untuk klaim SISTEM BARU dan tidak berlaku untuk klaim Pega, yang
		// menyimpan IDPEGA (Work Owner, 2026-09-26) — aturan lengkapnya beserta alasannya
		// ada di inputreqprotection.ClaimReferenceOf.
		//
		// Tetap DITURUNKAN, bukan ditanyakan ke form: dua isian yang wajib bersesuaian
		// tetapi diketik terpisah akan berbeda cepat atau lambat, tanpa galat.
		nullIfEmpty(inputreqprotection.ClaimReferenceOf(claim)),
		nullIfEmpty(draft.Type),
		at.UTC(),
		nullIfEmpty(by),
		nullIfEmpty(draft.Note),
		lama,
		baru,
		nullIfEmpty(detail.ObjectName),
		nullIfEmpty(detail.BranchName),
		// Sasaran perubahan Cause of Loss. NULL untuk tipe lain — deriveChangeDetail yang
		// memastikannya, supaya aturan "tipe mana yang punya sasaran" hidup di satu tempat.
		nullIfEmpty(detail.ObjectID),
		nullIfEmpty(detail.ObjectCoverageID),
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
	selected inputreqprotection.CoverageRow,
	by string,
	at time.Time,
) (inputreqprotection.Protection, error) {
	detail := deriveChangeDetail(draft, claim, selected)
	lama, baru := encodeChangeDetail(draft.Type, detail)

	hasil, err := r.db.ExecContext(ctx, query("protection_update"),
		// Nomor polis DITURUNKAN dari klaim, bukan diterima dari form — persis yang
		// dilakukan `Activity/OpenProtection-Act.xml` saat klaim dicari.
		nullIfEmpty(claim.PolicyNumber),
		nullIfEmpty(draft.ClaimNumber),
		// ID_CLAIM DITURUNKAN dari nomor klaim, bukan diterima dari form.
		//
		// Work Owner menegaskan 2026-09-24 bahwa ClaimNo dan ClaimID berisi nilai yang
		// sama. Itu benar untuk klaim SISTEM BARU dan tidak berlaku untuk klaim Pega, yang
		// menyimpan IDPEGA (Work Owner, 2026-09-26) — lihat
		// inputreqprotection.ClaimReferenceOf.
		//
		// Pada PEMBARUAN, nilainya ikut dihitung ulang: pemohon dapat mengganti nomor klaim
		// yang ditaut, dan ID_CLAIM yang tertinggal pada klaim lama akan menunjuk klaim
		// yang bukan lagi miliknya.
		nullIfEmpty(inputreqprotection.ClaimReferenceOf(claim)),
		nullIfEmpty(draft.Type),
		nullIfEmpty(draft.Note),
		lama,
		baru,
		nullIfEmpty(detail.ObjectName),
		nullIfEmpty(detail.BranchName),
		// Ikut ditulis ulang — termasuk menjadi NULL ketika tipenya berubah dari '8'.
		nullIfEmpty(detail.ObjectID),
		nullIfEmpty(detail.ObjectCoverageID),
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
		idObjek, idCoverage               sql.NullString
	)

	// namaTipe datang dari LEFT JOIN ke master, sehingga ia NULL untuk dua keadaan yang
	// berbeda: kode tipenya kosong, atau kodenya ada tetapi tidak terdaftar di master.
	// Keduanya diperlakukan sama di sini — yang membedakannya adalah lapisan tampilan,
	// yang menampilkan kode apa adanya saat namanya tidak ada.
	if err := row.Scan(&id, &nopolis, &noklaim, &idpega, &tipe, &namaTipe,
		&dibuatPada, &notes, &status, &dibuatOleh,
		&lama, &baru, &namaObjek, &namaCabang,
		&idObjek, &idCoverage); err != nil {
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

	// Sasaran perubahan Cause of Loss. Dibaca APA ADANYA — termasuk untuk tipe selain '8',
	// yang seharusnya kosong. Mengosongkannya di sini akan menyembunyikan baris yang
	// terlanjur salah terisi, dan baris semacam itu justru yang perlu terlihat.
	p.ChangeDetail.ObjectID = teks(idObjek)
	p.ChangeDetail.ObjectCoverageID = teks(idCoverage)

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

// prefixKunciKlaimPega adalah awalan kunci instance Pega pada `T_CLAIM_PNC.CLAIMID`.
//
// # Kenapa ia konstanta di Go, bukan literal di dalam SQL
//
// Nama kelas internal Pega yang tertanam di kunci data bisnis adalah utang teknis yang
// `D-22` hapus untuk klaim baru — tetapi klaim warisan terlanjur membawanya, dan tidak
// dinomori ulang. Ia karena itu tidak dapat dihindari selama masa paralel; yang dapat
// dilakukan adalah menaruhnya di SATU tempat bernama, bukan menyebarkannya ke berkas SQL.
//
// Bentuknya terukur dan hanya dua (hitungan 2026-09-26 atas 2.176 baris):
//
//	ASM-FW-GCNMFW-WORK PNC-1865   2.167 baris   klaim warisan Pega
//	PNCN.26.0007                      9 baris   klaim sistem baru, tanpa awalan
//
// Nol bentuk ketiga, nol spasi ganda. Saat klaim warisan habis, konstanta ini dan cabang
// keduanya hilang bersamanya — dan itu terlihat sebagai satu penghapusan, bukan sebagai
// perburuan literal di banyak kueri.
const prefixKunciKlaimPega = "ASM-FW-GCNMFW-WORK "

// ClaimRepo mencari klaim yang hendak ditaut.
//
// # Sumbernya T_CLAIM_PNC
//
// Work Owner menetapkan 2026-09-26: *"cari noklaim di input req nya ke t_claim_pnc"*, dan
// *"jangan gunakan t_claimlist_admin sama sekali"*. Tabel kerja Pega
// `PC_ASM_FW_GCNMFW_WORK` tidak lagi dibaca; `T_CLAIMLIST_ADMIN` tidak pernah dibaca.
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

// FindClaim mencari klaim menurut nomornya di `POOLDATA.T_CLAIM_PNC`.
//
// Nomor yang diketik pengguna dicocokkan ke `CLAIMID` dalam DUA bentuk sekaligus — apa
// adanya untuk klaim sistem baru, dan berawalan untuk klaim warisan Pega. Keduanya dikirim
// sebagai argumen terpisah karena driver mengikat menurut urutan KEMUNCULAN penanda, bukan
// menurut nomornya; `:1` yang dipakai berulang menghasilkan ORA-01008.
func (r *ClaimRepo) FindClaim(
	ctx context.Context,
	number string,
) (inputreqprotection.Claim, error) {
	var (
		claimID            string
		polis, tertanggung sql.NullString
		dol                sql.NullTime
		penyebab           sql.NullString
		cabang, namaObjek  sql.NullString
		kodeBisnis         sql.NullString
	)

	nomor := kunci(number)
	err := r.db.QueryRowContext(ctx, query("claim_find"), nomor, prefixKunciKlaimPega+nomor).
		Scan(&claimID, &polis, &tertanggung, &dol, &penyebab, &cabang, &namaObjek, &kodeBisnis)
	if errors.Is(err, sql.ErrNoRows) {
		return inputreqprotection.Claim{}, inputreqprotection.ErrClaimNotFound
	}
	if err != nil {
		return inputreqprotection.Claim{}, fmt.Errorf(
			"inputreqprotection/sqlstore: mencari klaim: %w", err)
	}

	klaim := inputreqprotection.Claim{
		// Number diturunkan dari CLAIMID, BUKAN dari kolom CLAIMNO. CLAIMNO kosong pada 479
		// baris, berulang pada satu pasang, dan berbeda isi pada 11 baris — memakainya
		// membuat nomor yang ditampilkan berbeda dari nomor yang dicari.
		Number: nomorKlaimDari(claimID),
		// PegaID adalah CLAIMID UTUH — nilai inilah yang disimpan sebagai ID_CLAIM dan
		// menjadi kunci UPDATE saat proteksinya disetujui.
		PegaID:       strings.TrimSpace(claimID),
		PolicyNumber: teks(polis),
		InsuredName:  teks(tertanggung),
		CauseOfLoss:  teks(penyebab),
		BranchName:   teks(cabang),
		ObjectName:   teks(namaObjek),
		BusinessCode: teks(kodeBisnis),
	}
	if dol.Valid {
		waktu := dol.Time
		klaim.LossDate = &waktu
	}
	return klaim, nil
}

// ListCoverages membaca seluruh coverage klaim beserta kunci barisnya.
//
// Kedua penanda kunci dikirim sama seperti FindClaim — nomor apa adanya dan nomor
// berawalan kunci Pega — supaya klaim warisan dan klaim sistem baru sama-sama ketemu.
//
// # Daftar kosong dikembalikan sebagai daftar kosong
//
// Bukan sebagai galat. Klaim yang belum punya coverage adalah keadaan biasa, dan panel yang
// menyatakan "tidak ada coverage" jauh lebih jujur daripada galat yang menyuruh pengguna
// mencari sebab di tempat yang salah.
func (r *ClaimRepo) ListCoverages(
	ctx context.Context,
	number string,
) ([]inputreqprotection.CoverageRow, error) {
	nomor := kunci(number)
	rows, err := r.db.QueryContext(ctx, query("claim_coverages"), nomor, prefixKunciKlaimPega+nomor)
	if err != nil {
		return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca coverage klaim: %w", err)
	}
	defer rows.Close()

	var hasil []inputreqprotection.CoverageRow
	for rows.Next() {
		var (
			objectID, coverageID   sql.NullString
			namaObjek, namaCover   sql.NullString
			penyebab, penyebabKode sql.NullString
		)
		if err := rows.Scan(
			&objectID, &coverageID, &namaObjek, &namaCover, &penyebab, &penyebabKode,
		); err != nil {
			return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca baris coverage: %w", err)
		}
		hasil = append(hasil, inputreqprotection.CoverageRow{
			ObjectID:         teks(objectID),
			ObjectCoverageID: teks(coverageID),
			ObjectName:       teks(namaObjek),
			CoverageName:     teks(namaCover),
			CauseOfLoss:      teks(penyebab),
			CauseOfLossID:    teks(penyebabKode),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca coverage klaim: %w", err)
	}
	return hasil, nil
}

// nomorKlaimDari memotong awalan kunci Pega, bila ada.
//
// Perbandingannya TIDAK peka huruf besar-kecil: `CLAIMID` pada data produksi tertulis
// `ASM-FW-GCNMFW-WORK`, sementara nama kelas yang sama muncul juga sebagai
// `ASM-FW-GCNMFW-Work-PNC` di tempat lain. Mencocokkan apa adanya akan membuat satu ejaan
// lolos dan ejaan lain tidak — tanpa gejala, karena keduanya tetap menghasilkan teks.
func nomorKlaimDari(claimID string) string {
	rapi := strings.TrimSpace(claimID)
	if len(rapi) > len(prefixKunciKlaimPega) &&
		strings.EqualFold(rapi[:len(prefixKunciKlaimPega)], prefixKunciKlaimPega) {
		return strings.TrimSpace(rapi[len(prefixKunciKlaimPega):])
	}
	return rapi
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
	selected inputreqprotection.CoverageRow,
) inputreqprotection.ChangeDetail {
	// Nama objek diambil dari BARIS YANG DIPILIH bila ada, baru jatuh ke klaim.
	//
	// Klaim menyimpan nama objek PERTAMA; pada klaim bercoverage banyak itu belum tentu objek
	// yang diubah. Nilai dari baris pilihan lebih tepat, dan hanya dipakai bila terisi supaya
	// tipe '7' — yang tidak punya pilihan — tetap memakai nama dari klaim seperti sebelumnya.
	namaObjek := claim.ObjectName
	if strings.TrimSpace(selected.ObjectName) != "" {
		namaObjek = selected.ObjectName
	}

	return inputreqprotection.ChangeDetail{
		// Sisi KLAIM.
		LossDateBefore: claim.LossDate,
		// CauseOfLossID adalah isi OLD_DATA, dan sejak 2026-10-05 ia KODE penyebab kerugian —
		// bukan deskripsinya (keputusan Work Owner: "old data new data simpan idcol aja").
		//
		// Diambil dari BARIS YANG DIPILIH, bukan dari klaim. Klaim tidak punya penyebab
		// kerugian tunggal: kueri terhadap `PNC-1452` mengembalikan empat baris dengan empat
		// penyebab berbeda, dan `Claim.CauseOfLoss` hanyalah yang pertama di antaranya.
		//
		// Menyimpan kode, bukan deskripsi, membuat OLD_DATA dan NEW_DATA sebentuk — keduanya
		// `D_COL_ID` — sehingga perbandingan "dari apa menjadi apa" tidak pernah bergantung
		// pada teks yang dapat berubah di master.
		CauseOfLossID: selected.CauseOfLossID,
		ObjectName:    namaObjek,
		BranchName:    claim.BranchName,

		// Sisi PENGGUNA.
		LossDateAfter:       draft.Change.LossDateAfter,
		CauseOfLossMasterID: draft.Change.CauseOfLossAfter,

		// Sasaran perubahan Cause of Loss — baris coverage yang DIPILIH pemohon.
		//
		// Datang dari pengguna, bukan diturunkan dari klaim: klaim punya banyak coverage, dan
		// yang menentukan mana di antaranya adalah tombol Pilih yang ditekan pemohon.
		// Validasi sudah menolak permintaan tipe '8' yang tidak menyebutnya.
		ObjectID:         draft.Change.ObjectID,
		ObjectCoverageID: draft.Change.ObjectCoverageID,
	}
}

// CauseRepo membaca master penyebab kerugian dari `POOLDATA.D_CAUSE_OF_LOSS`.
//
// KOLOM biasa — bukan `JSONDATA`, dan bukan view. Lini bisnisnya dari tabel anak
// `POOLDATA.D_CAUSE_OF_LOSS_BUSINESS`. Ditetapkan Work Owner 2026-10-05; alasannya beserta
// akibatnya ada di kepala kueri `cause_of_loss_options`.
type CauseRepo struct{ db *sql.DB }

// NewCauseRepo membentuk pembaca master penyebab kerugian.
func NewCauseRepo(db *sql.DB) *CauseRepo { return &CauseRepo{db: db} }

// ListCauseOfLoss mengembalikan pilihan dropdown "Next Cause Of Loss".
//
// Kode bisnis dikirim DUA KALI karena penandanya muncul dua kali di dalam kueri — sekali
// pada pemeriksaan NULL, sekali pada `JSON_EXISTS`. Driver mengikat argumen menurut urutan
// kemunculan, bukan menurut nomornya; mengirimnya sekali menghasilkan
// **ORA-01008: not all variables bound**.
func (r *CauseRepo) ListCauseOfLoss(
	ctx context.Context,
	businessCode string,
) ([]inputreqprotection.CauseOfLossOption, error) {
	kode := nullIfEmpty(strings.TrimSpace(businessCode))

	rows, err := r.db.QueryContext(ctx, query("cause_of_loss_options"), kode, kode)
	if err != nil {
		return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca master penyebab kerugian: %w", err)
	}
	defer rows.Close()

	var hasil []inputreqprotection.CauseOfLossOption
	for rows.Next() {
		var id, deskripsi, kodeRugi sql.NullString
		if err := rows.Scan(&id, &deskripsi, &kodeRugi); err != nil {
			return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca baris penyebab kerugian: %w", err)
		}
		// Baris tanpa D_COL_ID dilewati: ia tidak dapat disimpan sebagai pilihan, dan
		// menampilkannya berarti menawarkan sesuatu yang gagal saat dipilih.
		if strings.TrimSpace(teks(id)) == "" {
			continue
		}
		hasil = append(hasil, inputreqprotection.CauseOfLossOption{
			ID:          teks(id),
			Description: teks(deskripsi),
			LossCode:    teks(kodeRugi),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inputreqprotection/sqlstore: membaca master penyebab kerugian: %w", err)
	}
	return hasil, nil
}

var _ inputreqprotection.CauseOfLossRepo = (*CauseRepo)(nil)
