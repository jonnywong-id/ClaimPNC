package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxanalystdoctor"
)

// Repo membaca antrean Analyst Doctor dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Seluruh tabel yang dibacanya milik sistem lama,
// dan selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// List mengambil satu halaman antrean milik seorang operator.
//
// # Satu perjalanan, bukan dua
//
// Jumlah seluruh baris ikut dibawa kueri yang sama lewat `COUNT(*) OVER ()`. Kueri kedua
// yang hanya menghitung akan membaca ulang gabungan yang sama — dan gabungan itulah bagian
// yang mahal, bukan pemotongan halamannya.
//
// Akibatnya: pada halaman KOSONG, jumlah total tidak ikut terbawa karena tidak ada baris yang
// membawanya. Itu ditangani di bawah, dan bukan sekadar detail — tanpa penanganannya, bilah
// halaman menghilang begitu pengguna membuka halaman terakhir yang kebetulan kosong.
func (r *Repo) List(
	ctx context.Context,
	operator string,
	f inboxanalystdoctor.Filter,
) (inboxanalystdoctor.Page, error) {
	clean := f.Normalize()

	result := inboxanalystdoctor.Page{
		Tasks: []inboxanalystdoctor.AnalystDoctorTask{},
	}

	rows, err := r.db.QueryContext(ctx, query("list_tasks"),
		inboxanalystdoctor.TransferAnalystDoctor,
		operator,
		inboxanalystdoctor.StatusKerjaSelesai,
		keyword(clean.Search),
		clean.Offset,
		clean.Limit,
	)
	if err != nil {
		return inboxanalystdoctor.Page{}, fmt.Errorf("menjalankan kueri list_tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		task, total, err := scanTask(rows)
		if err != nil {
			return inboxanalystdoctor.Page{}, fmt.Errorf("membaca baris kueri list_tasks: %w", err)
		}
		result.Tasks = append(result.Tasks, task)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxanalystdoctor.Page{}, fmt.Errorf("menelusuri hasil kueri list_tasks: %w", err)
	}

	return result, nil
}

// keyword menyiapkan kata kunci sebagai nilai yang boleh NULL.
//
// Kata kunci kosong dikirim sebagai NULL, dan kueri menjawabnya dengan `:4 IS NULL` yang
// mematikan seluruh saringan pencarian. Mengirimnya sebagai teks kosong akan membuat
// `LIKE '%%'` — yang kebetulan juga cocok dengan semuanya, tetapi memaksa basis data
// memindai setiap baris alih-alih melewati predikatnya.
func keyword(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// scanTask membaca satu baris hasil beserta jumlah seluruh baris yang menyertainya.
//
// Urutan pembacaan di sini WAJIB sama dengan urutan kolom pada kueri dan dengan taskColumns
// di query.go. Ketiganya dijaga query_test.go.
func scanTask(rows *sql.Rows) (inboxanalystdoctor.AnalystDoctorTask, int, error) {
	var (
		task inboxanalystdoctor.AnalystDoctorTask

		reference        sql.NullString
		caseID           sql.NullString
		policyNumber     sql.NullString
		insuredName      sql.NullString
		branchName       sql.NullString
		adminName        sql.NullString
		technicalPIC     sql.NullString
		technicalPICNote sql.NullString
		registeredAt     sql.NullTime
		processStatus    sql.NullString
		assignedOperator sql.NullString
		total            int
	)

	// SELURUH kolom teks dibaca sebagai NullString, termasuk yang "pasti terisi".
	//
	// Itu bukan kehati-hatian berlebihan. `docs/kolom-t-claimlist-admin.md` mencatat 63 dari
	// 186 kolom tabel ini TIDAK PERNAH diisi, dan kolom yang terisi pun tidak punya
	// constraint NOT NULL. Membaca langsung ke string akan menghasilkan galat pemindaian
	// pada satu baris warisan, dan galat itu menjatuhkan SELURUH halaman — bukan satu sel.
	if err := rows.Scan(
		&reference,
		&caseID,
		&policyNumber,
		&insuredName,
		&branchName,
		&adminName,
		&technicalPIC,
		&technicalPICNote,
		&registeredAt,
		&processStatus,
		&assignedOperator,
		&total,
	); err != nil {
		return inboxanalystdoctor.AnalystDoctorTask{}, 0, err
	}

	task.ClaimID = reference.String
	task.ClaimNumber = caseID.String
	task.PolicyNumber = policyNumber.String
	task.InsuredName = insuredName.String
	task.BranchName = branchName.String
	task.AdminName = adminName.String
	task.TechnicalPIC = technicalPIC.String
	task.TechnicalPICNote = technicalPICNote.String
	task.ProcessStatus = processStatus.String
	task.AssignedOperator = assignedOperator.String

	if registeredAt.Valid {
		// Waktu disimpan UTC (`08-TECHNICAL-STRATEGY.md` §4.4). Konversi ke WIB terjadi di
		// SATU tempat saja, yaitu lapisan transport — tidak di sini, dan tidak di layar.
		task.RegisteredAt = registeredAt.Time.UTC()
	}

	return task, total, nil
}

// CheckTables memastikan kedua tabel terbaca dari koneksi ini.
//
// Dipakai perintah `-periksa`, bukan jalur permintaan pengguna.
func (r *Repo) CheckTables(ctx context.Context) error {
	var probe int
	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&probe); err != nil {
		return fmt.Errorf("membaca tabel antrean Analyst Doctor: %w", err)
	}
	return nil
}

// CheckColumns memastikan kedua kolom yang BELUM terkonfirmasi DBA memang ada.
//
// # Kenapa ia terpisah dari CheckTables
//
// Karena keduanya gagal karena sebab yang sama sekali berbeda, dan galat yang menyebut sebab
// yang salah mengirim orang yang memperbaikinya ke arah yang keliru:
//
//	CheckTables gagal   -> hak baca, atau tabelnya memang tidak ada di koneksi itu
//	CheckColumns gagal  -> nama kolomnya salah; properti Pega-nya tidak terekspos
//
// Yang kedua adalah keadaan yang DIDUGA akan terjadi sampai DBA menjawab, dan pesan galatnya
// karena itu menyebut apa yang harus diminta — bukan sekadar menyatakan kegagalan.
func (r *Repo) CheckColumns(ctx context.Context) error {
	var transfer, note int
	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(&transfer, &note); err != nil {
		return fmt.Errorf(
			"kolom ISCOMPLIANCETRANSFER_1 / ANALYSTDOCTORREMAKS_1 pada "+
				"DATAPEGA.PC_ASM_FW_GCNMFW_WORK tidak dapat dibaca. Kedua properti Pega-nya "+
				"ditandai `unexposed`, sehingga nama kolomnya masih menunggu konfirmasi DBA "+
				"— lihat kepala inboxanalystdoctor.sql: %w", err)
	}
	return nil
}
