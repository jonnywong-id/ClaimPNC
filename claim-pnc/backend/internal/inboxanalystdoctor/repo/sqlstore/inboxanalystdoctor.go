package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
		inboxanalystdoctor.TaskLabelAnalystDoctor,
		operator,
		inboxanalystdoctor.StatusKerjaSelesai,
		keyword(clean.Search),     // :4 penyaring pencarian aktif?
		likePattern(clean.Search), // :5 pola untuk Nomor Case
		likePattern(clean.Search), // :6 pola untuk No Polis — nilainya sama
		clean.Offset,              // :7
		clean.Limit,               // :8
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

// likePattern membentuk pola LIKE untuk kedua kolom yang dicari.
//
// # Kenapa polanya dibentuk di Go, bukan dirangkai di SQL
//
// Versi sebelumnya menulis `LIKE '%' || UPPER(:4) || '%'` dan memakai `:4` yang sama pada
// TIGA tempat. Driver go-ora menghitung setiap kemunculan `:n` sebagai satu variabel yang
// harus diikat, sehingga kueri menuntut delapan ikatan sementara pemanggil mengirim enam —
// dan Oracle menjawab ORA-01008. Membentuk polanya di sini membuat tiap penanda muncul
// tepat sekali.
//
// # Kenapa huruf besar
//
// Karena sisi SQL memakai `UPPER(...)`. Perbandingan yang hanya satu sisinya diseragamkan
// tidak pernah cocok, dan gagalnya DIAM: pengguna mengetik huruf kecil lalu diberi tahu
// klaimnya tidak ada.
//
// # Kenapa di-escape
//
// Supaya pencarian "100%" tidak berubah menjadi pola yang mencocokkan apa saja. ESCAPE-nya
// dinyatakan di sisi SQL. Pola ini sama dengan `inboxcloseclaim.likePattern`.
func likePattern(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
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

// CheckColumns memastikan SETIAP kolom yang dibaca kueri daftar memang ada.
//
// # Kenapa ia terpisah dari CheckTables
//
// Karena keduanya gagal karena sebab yang sama sekali berbeda, dan galat yang menyebut sebab
// yang salah mengirim orang yang memperbaikinya ke arah yang keliru:
//
//	CheckTables gagal   -> hak baca, atau tabelnya memang tidak ada di koneksi itu
//	CheckColumns gagal  -> ada nama kolom yang tidak ada di tabelnya
//
// # Kenapa ia memeriksa SELURUH kolom, bukan yang paling meragukan saja
//
// Versi sebelumnya hanya memeriksa dua kolom tebakan — `ISCOMPLIANCETRANSFER_1` dan
// `ANALYSTDOCTORREMAKS_1` — lalu berhenti di situ. Keduanya ternyata memang tidak ada
// (terverifikasi ke katalog Oracle 2026-10-09), dan karena pemeriksaannya berhenti pada
// temuan pertama, kolom lain tidak pernah sempat terperiksa sama sekali.
//
// Pemeriksaan yang menyerah pada temuan pertama menyembunyikan temuan kedua. Kini seluruh
// kolom yang benar-benar dipakai ikut di-parse Oracle dalam satu kueri.
func (r *Repo) CheckColumns(ctx context.Context) error {
	var probe [10]int
	targets := []any{
		&probe[0], &probe[1], &probe[2], &probe[3], &probe[4],
		&probe[5], &probe[6], &probe[7], &probe[8], &probe[9],
	}
	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(targets...); err != nil {
		return fmt.Errorf(
			"kolom antrean Analyst Doctor tidak lengkap pada "+
				"DATAPEGA.PC_ASM_FW_GCNMFW_WORK atau DATAPEGA.PC_ASSIGN_WORKLIST. "+
				"Galat Oracle menyebut nama kolom yang salah — lihat kepala "+
				"inboxanalystdoctor.sql: %w", err)
	}
	return nil
}
