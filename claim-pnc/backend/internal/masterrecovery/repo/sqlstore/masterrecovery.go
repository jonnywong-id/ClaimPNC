package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/masterrecovery"
)

// Repo membaca dan menulis objek Master Recovery pada satu basis data entitas.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master Recovery adalah
// SATU-SATUNYA penulis MST_RECOVERY_ASM_PENJAMINAN di sistem lama, dan itu diperiksa ke
// seluruh export, bukan diandaikan:
//
//	Database/INSERTMASTERRECOVERYKLAIM.prc          satu-satunya berkas yang memuat INSERT
//	RDB List/InsertMasterRecoveryKlaimASM-SQL.xml   satu-satunya pemanggil procedure itu
//	Activity/Insert_mst_recoveryKlaimASM-Act.xml    satu-satunya pemanggil rule itu
//
// Memindahkan layar itu ke sini karena itu memindahkan kepemilikan tabelnya secara utuh.
//
// Dua tabel lain yang ikut ditulis — MST_VIRTUAL_ACCOUNT_PNC dan DATA_ATTACHFILE — masih
// DIBAGI dengan alur Pega lain (lampiran klaim menulis ke tabel yang sama). Keduanya
// hanya disisipi, tidak pernah diubah maupun dihapus, sehingga penulisan bersama itu
// tidak dapat saling menimpa.
type Repo struct {
	db *sql.DB

	// now dapat diganti pada pengujian: dua digit tahun ikut membentuk DATAID lampiran,
	// dan pengujian yang bergantung pada tahun berjalan akan gagal sendiri setiap
	// pergantian tahun.
	now func() time.Time
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db, now: time.Now} }

// NextBatch mengembalikan nomor batch berikutnya untuk ditampilkan di layar.
//
// Nilainya PERKIRAAN dan disebut demikian di seluruh modul: dua petugas yang membuka
// layar bersamaan memperoleh angka yang sama. Yang mengikat adalah nomor yang diterbitkan
// Insert di dalam transaksinya sendiri.
func (r *Repo) NextBatch(ctx context.Context) (int64, error) {
	var batch int64
	if err := r.db.QueryRowContext(ctx, getQuery("recovery_next_batch")).Scan(&batch); err != nil {
		return 0, fmt.Errorf("masterrecovery/sqlstore: membaca nomor batch berikutnya: %w", err)
	}
	return batch, nil
}

// Insert menyimpan satu batch recovery.
//
// # Tiga langkah dalam SATU transaksi
//
//  1. Kunci tabel — supaya penerbitan nomor tidak dapat berlomba.
//  2. Terbitkan nomor batch.
//  3. Sisipkan barisnya.
//
// Ini memperbaiki dua cacat nyata sistem lama sekaligus. Pertama, nomor batch di sana
// dibaca layar (`GetIDMasterRecoveryKlaim`) jauh sebelum disimpan, sehingga dua petugas
// yang membuka layar bersamaan pasti memperoleh nomor yang sama. Kedua — dan ini yang
// lebih berbahaya — `INSERTMASTERRECOVERYKLAIM.prc` menjawab keadaan itu dengan MELEWATI
// penyisipan tanpa satu pun pesan, sehingga petugas kedua melihat "berhasil" atas batch
// yang tidak pernah tersimpan.
//
// Di sini nomornya diterbitkan di dalam kunci, dan bentrok yang tersisa — bila ada —
// menjadi ErrBatchTaken yang terlihat, bukan kehilangan data yang senyap.
func (r *Repo) Insert(ctx context.Context, recovery masterrecovery.Recovery) (masterrecovery.Recovery, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterrecovery.Recovery{}, fmt.Errorf("masterrecovery/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, getQuery("recovery_lock_table")); err != nil {
		return masterrecovery.Recovery{}, fmt.Errorf("masterrecovery/sqlstore: mengunci tabel recovery: %w", err)
	}

	var batch int64
	if err := tx.QueryRowContext(ctx, getQuery("recovery_next_batch")).Scan(&batch); err != nil {
		return masterrecovery.Recovery{}, fmt.Errorf("masterrecovery/sqlstore: menerbitkan nomor batch: %w", err)
	}
	recovery.Batch = batch

	if _, err := tx.ExecContext(ctx, getQuery("recovery_insert"),
		recovery.Batch,
		nullable(recovery.PrincipalName),
		nullable(recovery.Year),
		int64(recovery.ClaimAmount),
		int64(recovery.PreviousPayment),
		int64(recovery.Payment),
		int64(recovery.Remainder),
		nullable(recovery.Remark),
		nullable(recovery.CasePosition),
		nullable(recovery.DocumentID),
		nullable(recovery.VirtualAccountNumber),
		nullable(recovery.InputBy),
		nullable(recovery.ClientID),
		nullable(recovery.ServiceLogID),
		nullable(recovery.PolicyNo),
		nullable(recovery.BusinessID),
		nullable(recovery.BranchID),
		nullable(recovery.AgentID),
		nullable(recovery.MarketingID),
	); err != nil {
		return masterrecovery.Recovery{}, translateWriteError(err, "menyisipkan batch recovery")
	}

	// Baris klaim disisipkan DI DALAM transaksi yang sama.
	//
	// Itu yang membuat batch tanpa rincian polis tidak pernah ada: bila salah satu baris
	// gagal, kepala batch-nya ikut batal. Di sistem lama keduanya menyatu dalam satu kolom
	// sehingga persoalan ini tidak muncul — memindahkannya ke tabel tersendiri
	// memunculkannya, dan transaksi inilah jawabannya.
	for index, line := range recovery.ClaimLine {
		if _, err := tx.ExecContext(ctx, getQuery("recovery_claim_line_insert"),
			recovery.Batch,
			// Urutan dimulai dari 1, sama dengan nomor baris yang dilaporkan pembaca CSV
			// kepada petugas — supaya "baris ke-3" berarti hal yang sama di layar dan di
			// basis data.
			index+1,
			line.PolicyNo,
			int64(line.ClaimAmount),
		); err != nil {
			return masterrecovery.Recovery{}, translateWriteError(err, "menyisipkan baris klaim recovery")
		}
	}

	if err := tx.Commit(); err != nil {
		return masterrecovery.Recovery{}, fmt.Errorf("masterrecovery/sqlstore: menyimpan batch recovery: %w", err)
	}
	return recovery, nil
}

// List membaca seluruh batch milik principal yang masuk halaman ini.
//
// Angka kedua yang dikembalikan adalah jumlah PRINCIPAL yang cocok, bukan jumlah baris —
// lihat masterrecovery.Repo.
//
// Jumlah itu dihitung dengan kueri TERPISAH, bukan dengan window function di kueri yang
// sama. Alasannya portabilitas sekaligus keterbacaan: `COUNT(*) OVER ()` berlaku di Oracle
// dan PostgreSQL, tetapi menggabungkannya membuat kolom hitung terbawa ke setiap baris di
// dalam kueri yang sudah bersubkueri. Dua kueri kecil terhadap tabel sekecil ini lebih
// murah daripada satu kueri yang sulit dikoreksi.
func (r *Repo) List(ctx context.Context, f masterrecovery.ListFilter) ([]masterrecovery.Recovery, int, error) {
	name := nullable(f.PrincipalName)
	year := nullable(f.Year)

	var total int
	if err := r.db.QueryRowContext(ctx, getQuery("recovery_count"),
		name, name, year, year,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("masterrecovery/sqlstore: menghitung batch recovery: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, getQuery("recovery_list"),
		name, name, year, year, f.Offset, f.Limit,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("masterrecovery/sqlstore: membaca daftar batch recovery: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterrecovery.Recovery
	for rows.Next() {
		recovery, err := scanRecovery(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("masterrecovery/sqlstore: membaca baris batch recovery: %w", err)
		}
		result = append(result, recovery)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("masterrecovery/sqlstore: menelusuri daftar batch recovery: %w", err)
	}
	return result, total, nil
}

// scanRecovery membaca satu baris daftar.
//
// Seluruh kolom teks diperlakukan NULLABLE. Itu bukan kehati-hatian berlebihan: ketiga
// baris yang ada di portal ASM punya NULL pada sebagian kolomnya, dan tabel ini tidak
// punya satu pun NOT NULL selain kunci utamanya.
//
// Kolom uang dibaca sebagai sql.NullInt64, sehingga NULL menjadi nol — bukan galat.
// Sistem lama menyimpan batch tanpa nilai, dan menolak membacanya berarti satu baris
// warisan membuat seluruh halaman gagal tampil.
func scanRecovery(rows rowScanner) (masterrecovery.Recovery, error) {
	var (
		batch           int64
		principalName   sql.NullString
		year            sql.NullString
		claimAmount     sql.NullInt64
		previousPayment sql.NullInt64
		payment         sql.NullInt64
		remainder       sql.NullInt64
		remark          sql.NullString
		casePosition    sql.NullString
		documentID      sql.NullString
		virtualAccount  sql.NullString
		clientID        sql.NullString
		policyNo        sql.NullString
		serviceLogID    sql.NullString
		inputDate       sql.NullTime

		attachName     sql.NullString
		attachOperator sql.NullString
		attachDate     sql.NullTime
	)
	if err := rows.Scan(
		&batch, &principalName, &year,
		&claimAmount, &previousPayment, &payment, &remainder,
		&remark, &casePosition, &documentID, &virtualAccount, &clientID, &policyNo,
		&serviceLogID, &inputDate,
		&attachName, &attachOperator, &attachDate,
	); err != nil {
		return masterrecovery.Recovery{}, err
	}

	// Lampiran hanya dianggap ADA bila barisnya benar-benar ketemu di DATA_ATTACHFILE.
	// DOKUMENID yang terisi tetapi menunjuk baris yang sudah tidak ada menghasilkan
	// seluruh kolom LEFT JOIN bernilai NULL — dan di layar itu harus terbaca "belum ada
	// bukti bayar", bukan tombol yang menjanjikan berkas yang tidak dapat dibuka.
	var attachment *masterrecovery.AttachmentInfo
	if id := strings.TrimSpace(documentID.String); id != "" && (attachName.Valid || attachOperator.Valid || attachDate.Valid) {
		attachment = &masterrecovery.AttachmentInfo{
			ID:         id,
			Name:       strings.TrimSpace(attachName.String),
			UploadedBy: strings.TrimSpace(attachOperator.String),
			UploadedAt: attachDate.Time,
		}
	}

	return masterrecovery.Recovery{
		Attachment:           attachment,
		Batch:                batch,
		PrincipalName:        strings.TrimSpace(principalName.String),
		Year:                 strings.TrimSpace(year.String),
		ClaimAmount:          masterrecovery.Amount(claimAmount.Int64),
		PreviousPayment:      masterrecovery.Amount(previousPayment.Int64),
		Payment:              masterrecovery.Amount(payment.Int64),
		Remainder:            masterrecovery.Amount(remainder.Int64),
		Remark:               strings.TrimSpace(remark.String),
		CasePosition:         strings.TrimSpace(casePosition.String),
		DocumentID:           strings.TrimSpace(documentID.String),
		VirtualAccountNumber: strings.TrimSpace(virtualAccount.String),
		ClientID:             strings.TrimSpace(clientID.String),
		PolicyNo:             strings.TrimSpace(policyNo.String),
		ServiceLogID:         strings.TrimSpace(serviceLogID.String),
		// NULL menjadi waktu nol, bukan galat. Baris warisan boleh tidak punya INSERTDATE,
		// dan menolak membacanya berarti satu baris lama membuat seluruh halaman gagal.
		InputDate: inputDate.Time,
	}, nil
}

// FindDocument membaca satu Bukti Bayar beserta isinya.
func (r *Repo) FindDocument(ctx context.Context, id string) (masterrecovery.Document, error) {
	var (
		content  []byte
		name     sql.NullString
		mimeType sql.NullString
		note     sql.NullString
		imageID  sql.NullString
	)

	err := r.db.QueryRowContext(ctx, getQuery("attachment_read"), id).
		Scan(&content, &name, &mimeType, &note, &imageID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterrecovery.Document{}, masterrecovery.ErrDocumentNotFound
	case err != nil:
		return masterrecovery.Document{}, fmt.Errorf("masterrecovery/sqlstore: membaca bukti bayar: %w", err)
	}

	// Baris ADA tetapi kolom BLOB-nya kosong. Itu bukan salah tautan: sistem lama
	// menyimpan sebagian berkas di penyimpanan luar dan hanya menaruh penunjuknya di
	// IMAGEID. Dibedakan supaya layar dapat mengatakan yang sebenarnya.
	if len(content) == 0 {
		return masterrecovery.Document{}, masterrecovery.ErrDocumentElsewhere
	}

	return masterrecovery.Document{
		Name:     strings.TrimSpace(name.String),
		MimeType: strings.TrimSpace(mimeType.String),
		Note:     strings.TrimSpace(note.String),
		Content:  content,
	}, nil
}

// ListPrincipal membaca seluruh principal di master Virtual Account.
func (r *Repo) ListPrincipal(ctx context.Context) ([]masterrecovery.Principal, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("principal_list"))
	if err != nil {
		return nil, fmt.Errorf("masterrecovery/sqlstore: membaca daftar principal: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterrecovery.Principal
	for rows.Next() {
		principal, err := scanPrincipal(rows)
		if err != nil {
			return nil, fmt.Errorf("masterrecovery/sqlstore: membaca baris principal: %w", err)
		}
		result = append(result, principal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterrecovery/sqlstore: menelusuri daftar principal: %w", err)
	}
	return result, nil
}

// FindPrincipal mencari satu principal menurut Client ID dan nama.
func (r *Repo) FindPrincipal(ctx context.Context, clientID, name string) (masterrecovery.Principal, error) {
	row := r.db.QueryRowContext(ctx, getQuery("principal_find"), clientID, name)

	principal, err := scanPrincipal(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterrecovery.Principal{}, masterrecovery.ErrPrincipalNotFound
	case err != nil:
		return masterrecovery.Principal{}, fmt.Errorf("masterrecovery/sqlstore: mencari principal: %w", err)
	}
	return principal, nil
}

// SavePrincipal mencatat principal beserta VA yang baru diterbitkan.
func (r *Repo) SavePrincipal(ctx context.Context, principal masterrecovery.Principal) error {
	if _, err := r.db.ExecContext(ctx, getQuery("principal_insert"),
		nullable(principal.ClientID),
		nullable(principal.Name),
		nullable(principal.Status),
		nullable(principal.Message),
		nullable(principal.VirtualAccountNumber),
		nullable(principal.Email),
	); err != nil {
		return fmt.Errorf("masterrecovery/sqlstore: mencatat principal: %w", err)
	}
	return nil
}

// LookupPolicy mencari identitas yang menempel pada sebuah nomor polis.
//
// # Kueri ini menyeberang DB Link, dan itu disadari
//
// MST_DET_SALES berada di basis data lain dan dibaca lewat `@ASMD`. `D-25` menetapkan
// seluruh pemakaian DB Link diganti pemanggilan API, dan penggantinya — "API Master
// Sales" — belum ada (`R-03`).
//
// Selama masa paralel link-nya masih hidup dan masih dibaca Pega, sehingga memakainya di
// sini meneruskan ketergantungan yang sudah ada alih-alih menambah yang baru. Ia
// diisolasi pada satu kueri bernama supaya penggantinya kelak menyentuh satu tempat saja.
func (r *Repo) LookupPolicy(ctx context.Context, policyNo string) (masterrecovery.PolicyReference, error) {
	var (
		businessID  sql.NullString
		branchID    sql.NullString
		agentID     sql.NullString
		marketingID sql.NullString
	)
	err := r.db.QueryRowContext(ctx, getQuery("policy_reference"), policyNo).
		Scan(&businessID, &branchID, &agentID, &marketingID)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterrecovery.PolicyReference{}, masterrecovery.ErrPolicyNotFound
	case err != nil:
		return masterrecovery.PolicyReference{}, fmt.Errorf("masterrecovery/sqlstore: membaca identitas polis: %w", err)
	}

	return masterrecovery.PolicyReference{
		BusinessID:  strings.TrimSpace(businessID.String),
		BranchID:    strings.TrimSpace(branchID.String),
		AgentID:     strings.TrimSpace(agentID.String),
		MarketingID: strings.TrimSpace(marketingID.String),
	}, nil
}

// Kategori yang dituliskan pada baris lampiran.
//
// Keduanya konstanta karena mereka MENANDAI asal lampiran: tanpa penanda, bukti bayar
// recovery tidak dapat dibedakan dari sembilan ribu lampiran lain di tabel yang sama.
// Sistem lama mengirimkannya sebagai parameter dari layar; di sini ia tetap, karena hanya
// satu layar yang menulis lewat jalur ini.
const (
	DocumentCategory    = "RECOVERY"
	DocumentSubCategory = "BUKTI BAYAR"
)

// SaveDocument menyimpan Bukti Bayar dan mengembalikan DATAID-nya.
//
// # Bentuk DATAID, dan buktinya
//
// Meniru `Database/SET_ATTACHMENT_64BIT.prc`: dua digit tahun disambung nomor urut yang
// dipadatkan menjadi sepuluh angka. Bentuknya diverifikasi ke data nyata pada 2026-09-19
// — DATAID terbesar di portal ASM berupa dua angka tahun diikuti sepuluh angka urut, dan
// urutan ATTACHFILE_SEQ berada pada angka yang bersesuaian.
//
// Perangkaian dan pemadatannya dikerjakan di Go, bukan dengan LPAD dan TO_CHAR: keduanya
// termasuk yang dilarang `09-DATABASE-STRATEGY.md` §4 karena mengikat kueri pada dialek
// Oracle.
//
// # Satu transaksi, tanpa COMMIT di tengah
//
// Procedure lama menjalankan COMMIT sendiri, dan satu-satunya ROLLBACK-nya berada SESUDAH
// commit itu sehingga tidak memulihkan apa pun (`D-68`). Di sini ketiga langkahnya — urut,
// catatan penerbitan, dan barisnya — berada dalam satu transaksi: bila yang terakhir
// gagal, tidak ada nomor yang telanjur tercatat sebagai terpakai.
func (r *Repo) SaveDocument(ctx context.Context, document masterrecovery.Document) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("masterrecovery/sqlstore: memulai transaksi lampiran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var order int64
	if err := tx.QueryRowContext(ctx, getQuery("attachment_next_sequence")).Scan(&order); err != nil {
		return "", fmt.Errorf("masterrecovery/sqlstore: mengambil nomor urut lampiran: %w", err)
	}

	year := TwoDigitYear(r.now())
	dataID := year + TenDigits(order)

	key, err := newCounterKey()
	if err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, getQuery("attachment_counter_insert"), key, year, order); err != nil {
		return "", fmt.Errorf("masterrecovery/sqlstore: mencatat penerbitan nomor lampiran: %w", err)
	}

	if _, err := tx.ExecContext(ctx, getQuery("attachment_insert"),
		dataID,
		document.Content,
		nullable(document.UploadedBy),
		nullable(document.Name),
		nullable(document.Note),
		nullable(document.MimeType),
		DocumentCategory,
		DocumentSubCategory,
		// IDPEGA menunjuk kasus Pega yang memiliki lampiran ini. Batch recovery bukan kasus
		// Pega dan tidak punya kunci semacam itu, sehingga kolomnya dikosongkan alih-alih
		// diisi nilai karangan yang akan menyesatkan setiap pembaca kelak.
		nil,
	); err != nil {
		return "", translateWriteError(err, "menyimpan bukti bayar")
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("masterrecovery/sqlstore: menyimpan bukti bayar: %w", err)
	}
	return dataID, nil
}

// TwoDigitYear meniru `to_char(sysdate,'yy')`.
//
// Diekspor supaya perilakunya dapat diuji tanpa basis data.
func TwoDigitYear(at time.Time) string {
	year := at.Year() % 100
	if year < 10 {
		return "0" + strconv.Itoa(year)
	}
	return strconv.Itoa(year)
}

// TenDigits meniru `lpad(to_char(runno), 10, '0')`.
//
// Bilangan yang sudah lebih dari sepuluh angka dikembalikan APA ADANYA, sama seperti LPAD
// Oracle — tidak dipotong. Memotongnya akan menghasilkan DATAID GANDA, yang jauh lebih
// buruk daripada DATAID yang kepanjangan: kolomnya VARCHAR2(100), jadi yang panjang tetap
// tersimpan sementara yang ganda menimpa lampiran milik orang lain.
//
// Batasnya jauh: urutan berada di sekitar 1,9 juta pada 2026-09-19, sementara sepuluh
// angka menampung sepuluh miliar.
func TenDigits(n int64) string {
	number := strconv.FormatInt(n, 10)
	if len(number) >= 10 {
		return number
	}
	return strings.Repeat("0", 10-len(number)) + number
}

// newCounterKey membentuk nilai KEY pada C_COUNTER_ATTACHMENT.
//
// Procedure lama memakai `new_uuid` milik basis data. Yang dibutuhkan hanyalah nilai yang
// tidak pernah berulang, dan itu dapat dibuat di mana saja — membuatnya di sini
// melepaskan kueri dari satu fungsi khas Oracle lagi.
//
// Panjangnya 32 huruf heksadesimal, aman di dalam VARCHAR2(100).
func newCounterKey() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("masterrecovery/sqlstore: membentuk kunci penerbitan lampiran: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

// nullable mengirim NULL alih-alih teks kosong.
//
// Perbedaannya nyata di Oracle untuk kolom teks — keduanya memang sama di sana — tetapi ia
// tetap dipakai supaya baris yang ditulis modul ini tidak berbeda bentuk dari baris yang
// ditulis procedure lama, dan supaya perilakunya tidak berubah saat basis datanya kelak
// berpindah ke PostgreSQL, tempat ” dan NULL BERBEDA.
func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

func scanPrincipal(rows rowScanner) (masterrecovery.Principal, error) {
	var (
		clientID      sql.NullString
		name          sql.NullString
		accountNumber sql.NullString
		email         sql.NullString
		status        sql.NullString
		message       sql.NullString
	)
	if err := rows.Scan(&clientID, &name, &accountNumber, &email, &status, &message); err != nil {
		return masterrecovery.Principal{}, err
	}

	// Seluruhnya dirapikan di satu tempat: keenam kolom NULLABLE, dan dua di antaranya —
	// STATUS dan MESSAGE — memang NULL pada kedua baris yang ada di portal ASM.
	return masterrecovery.Principal{
		ClientID:             strings.TrimSpace(clientID.String),
		Name:                 strings.TrimSpace(name.String),
		VirtualAccountNumber: strings.TrimSpace(accountNumber.String),
		Email:                strings.TrimSpace(email.String),
		Status:               strings.TrimSpace(status.String),
		Message:              strings.TrimSpace(message.String),
	}, nil
}

// PrimaryKeyName adalah nama kunci utama MST_RECOVERY_ASM_PENJAMINAN.
//
// Diverifikasi langsung dari ALL_CONSTRAINTS pada 2026-09-19. Ia konstanta supaya kode
// dan basis data tidak dapat berbeda pendapat diam-diam.
const PrimaryKeyName = "MST_RECOVERY_ASM_PENJAMINAN_PK"

// translateWriteError mengubah pelanggaran kunci utama menjadi galat domain.
//
// Yang dicocokkan adalah NAMA CONSTRAINT, bukan nomor galat driver — nama itu tidak
// berubah saat driver atau basis datanya berganti.
func translateWriteError(err error, activity string) error {
	if strings.Contains(strings.ToUpper(err.Error()), PrimaryKeyName) {
		return masterrecovery.ErrBatchTaken
	}
	return fmt.Errorf("masterrecovery/sqlstore: %s: %w", activity, err)
}

var _ masterrecovery.Repo = (*Repo)(nil)

// CheckTable menguji ketiga tabel yang dipakai modul ini dapat dibaca akun aplikasi,
// tanpa mengambil satu baris pun.
//
// Ketiganya diperiksa TERPISAH karena ketiganya dapat gagal sendiri-sendiri, dan tindak
// lanjutnya berbeda: tabel recovery yang tidak ada berarti entitas itu belum memakai
// modul ini sama sekali, sedangkan tabel lampiran yang tidak dapat dibaca berarti hak
// akses akun aplikasi yang kurang.
func (r *Repo) CheckTable(ctx context.Context) error {
	for _, check := range []struct {
		name  string
		query string
	}{
		{"POOLDATA.MST_RECOVERY_ASM_PENJAMINAN", "recovery_check_table"},
		{"POOLDATA.MST_VIRTUAL_ACCOUNT_PNC", "principal_check_table"},
		{"POOLDATA.DATA_ATTACHFILE", "attachment_check_table"},
	} {
		rows, err := r.db.QueryContext(ctx, getQuery(check.query))
		if err != nil {
			return fmt.Errorf("masterrecovery/sqlstore: %s tidak dapat dibaca: %w", check.name, err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("masterrecovery/sqlstore: menutup pemeriksaan %s: %w", check.name, err)
		}
	}
	return nil
}

// CheckClaimLineTable menguji tabel baris klaim ada dan dapat dibaca akun aplikasi.
//
// Dipisahkan dari CheckTable dengan sengaja: ia SATU-SATUNYA objek modul ini yang dibuat
// migrasi kita sendiri (`0013`), sehingga kegagalannya punya tindak lanjut yang khas —
// migrasinya belum dijalankan DBA — bukan soal hak akses atas tabel warisan.
//
// Selama tabel ini belum ada, batch tetap dapat dicatat TETAPI daftar polisnya akan gagal
// tersimpan, dan karena keduanya satu transaksi, seluruh penyimpanan ikut batal. Itulah
// sebabnya ia layak dilaporkan terpisah dan tegas.
func (r *Repo) CheckClaimLineTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("recovery_claim_line_check_table"))
	if err != nil {
		return fmt.Errorf("masterrecovery/sqlstore: POOLDATA.CPNC_RECOVERY_BARIS_KLAIM tidak dapat dibaca: %w", err)
	}
	return rows.Close()
}

// CheckPolicyLink menguji DB Link ke MST_DET_SALES dapat ditembak.
//
// Dipisahkan dari CheckTable dengan sengaja: ia SATU-SATUNYA bagian modul ini yang
// menyeberang ke basis data lain, dan kegagalannya tidak menghalangi pencatatan batch —
// hanya membuat keempat kolom identitas polis kosong. Membedakannya membuat laporan
// pemeriksaan menyebutkan hal yang benar, alih-alih menyatakan seluruh modul tidak siap.
func (r *Repo) CheckPolicyLink(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("policy_reference"), "")
	if err != nil {
		return fmt.Errorf("masterrecovery/sqlstore: DB Link MST_DET_SALES@ASMD tidak dapat ditembak: %w", err)
	}
	return rows.Close()
}
