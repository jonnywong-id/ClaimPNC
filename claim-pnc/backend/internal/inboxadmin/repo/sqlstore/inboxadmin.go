package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxadmin"
)

// Repo membaca antrean kerja Inbox Admin dari SATU basis data entitas.
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

// plan menyebut kueri mana yang melayani sebuah tab dan bagaimana argumennya disusun.
//
// # Kenapa argumennya disusun per tab, bukan seragam
//
// Karena jumlah bind-nya memang berbeda: tab Status RCL/PUCL tidak menerima satu pun,
// sedangkan Unregistered RCV menerima sepuluh. Menyeragamkannya menuntut setiap kueri
// menyebut bind yang tidak dipakainya.
//
// # Kenapa satu nilai dikirim BERULANG
//
// go-ora mengikat parameter MENURUT POSISI, bukan menurut nama. Kueri yang memakai ulang
// penanda yang sama (`:1` tiga kali) menghasilkan ORA-01008 — atau, lebih buruk, tidak
// menghasilkan galat sama sekali tetapi mengikat nilai ke penanda yang salah. Keduanya
// terukur 2026-10-07: tab ALL, Unregistered RCV, dan Branch Claim gagal dengan ORA-01008,
// sedangkan All Case Admin diam-diam selalu kosong karena login pemanggil terikat ke kotak
// cari. Setiap kemunculan karena itu bernomor sendiri, dan argumennya disusun menurut
// urutan kemunculan — pola yang sama dengan inboxcloseclaim.
type plan struct {
	// name adalah nama kueri di berkas .sql.
	name string

	// args menyusun argumen bind sesuai urutan KEMUNCULAN penanda di kueri itu.
	args func(q inboxadmin.Query) []any
}

// repeat mengulang satu nilai n kali, untuk penanda yang muncul berkali-kali.
func repeat(value any, n int) []any {
	result := make([]any, n)
	for i := range result {
		result[i] = value
	}
	return result
}

// join menyambung beberapa kelompok argumen menjadi satu urutan.
func join(groups ...[]any) []any {
	var result []any
	for _, g := range groups {
		result = append(result, g...)
	}
	return result
}

// nilOrText mengirim teks kosong sebagai NULL, supaya `:n IS NULL` mematikan saringannya.
func nilOrText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// branchArgs adalah bind batas cabang: dua kemunculan, `:n IS NULL OR ... = :n+1`.
func branchArgs(q inboxadmin.Query) []any {
	return repeat(nilOrText(q.Scope.BranchCode), 2)
}

// regionArgs adalah bind kanwil: dua kemunculan, dengan bentuk yang sama.
func regionArgs(q inboxadmin.Query) []any {
	return repeat(nilOrText(q.Scope.RegionCode), 2)
}

// withBranch dan withBranchAndRegion menambahkan bind batas data di BELAKANG bind kueri —
// klausanya disisipkan tepat sebelum ORDER BY, sehingga kemunculannya selalu terakhir.
func withBranch(base func(inboxadmin.Query) []any) func(inboxadmin.Query) []any {
	return func(q inboxadmin.Query) []any { return join(base(q), branchArgs(q)) }
}

func withBranchAndRegion(base func(inboxadmin.Query) []any) func(inboxadmin.Query) []any {
	return func(q inboxadmin.Query) []any { return join(base(q), branchArgs(q), regionArgs(q)) }
}

// keywordThenBusiness adalah urutan bind tab ALL dan Branch Claim: kata kunci x3, lini
// bisnis x5.
func keywordThenBusiness(q inboxadmin.Query) []any {
	return join(repeat(keyword(q), 3), repeat(string(q.Business), 5))
}

// callerThenKeyword adalah urutan bind Request Survey dan All Case Admin: login, lalu kata
// kunci x3.
func callerThenKeyword(q inboxadmin.Query) []any {
	return join([]any{q.Caller.Login}, repeat(keyword(q), 3))
}

// plans memetakan kode tab ke kuerinya.
//
// Peta ini adalah satu-satunya tempat kode tab bertemu nama kueri. Tab yang tidak ada di
// sini menghasilkan galat yang menyebut kodenya — bukan kueri kosong yang mengembalikan nol
// baris dan terbaca seperti antrean yang memang kosong.
var plans = map[string]plan{
	inboxadmin.TabAll: {
		name: "list_all",
		args: withBranchAndRegion(keywordThenBusiness),
	},
	inboxadmin.TabUnregisteredRCV: {
		name: "list_unregistered",
		args: withBranchAndRegion(func(q inboxadmin.Query) []any {
			return join(repeat("NORMAL", 2), repeat(keyword(q), 3), repeat(string(q.Business), 5))
		}),
	},
	inboxadmin.TabRCVOnline: {
		name: "list_unregistered",
		args: withBranchAndRegion(func(q inboxadmin.Query) []any {
			return join(repeat("ONLINE", 2), repeat(keyword(q), 3), repeat(string(q.Business), 5))
		}),
	},
	inboxadmin.TabRequestSurvey: {
		name: "list_request_survey",
		args: withBranch(callerThenKeyword),
	},
	inboxadmin.TabRequestDocument: {
		name: "list_request_document",
		args: func(q inboxadmin.Query) []any {
			return []any{q.Caller.Login}
		},
	},
	inboxadmin.TabAllCaseAdmin: {
		name: "list_all_case_admin",
		args: withBranch(callerThenKeyword),
	},
	inboxadmin.TabBranchClaim: {
		name: "list_branch_claim",
		args: withBranchAndRegion(keywordThenBusiness),
	},
	inboxadmin.TabRCLPUCL: {
		name: "list_rcl_pucl",
		args: func(inboxadmin.Query) []any { return nil },
	},
}

// keyword menyiapkan kata kunci sebagai nilai yang boleh NULL.
//
// Kata kunci kosong dikirim sebagai NULL, dan kueri menjawabnya dengan `:1 IS NULL` yang
// mematikan seluruh saringan pencarian. Mengirimnya sebagai teks kosong akan membuat
// `LIKE '%%'` — yang kebetulan juga cocok dengan semuanya, tetapi memaksa basis data
// memindai setiap baris alih-alih melewati predikatnya.
func keyword(q inboxadmin.Query) any {
	if q.Keyword == "" {
		return nil
	}
	return q.Keyword
}

// List mengambil SELURUH baris satu tab, belum dipaginasi.
//
// Pemotongan halaman terjadi di aplikasi (inboxadmin.Slice) atas keputusan Work Owner
// 2026-09-20 — lihat catatan di kepala inboxadmin.sql.
//
// Klaim Pega dan klaim PNCN keduanya dibaca dari POOLDATA.T_CLAIMLIST_ADMIN, sehingga
// satu kueri melayani keduanya — lihat bagian SUMBER ANTREAN di inboxadmin.sql.
func (r *Repo) List(ctx context.Context, q inboxadmin.Query) ([]inboxadmin.WorkItem, error) {
	selected, known := plans[q.Tab.Code]
	if !known {
		return nil, fmt.Errorf("inboxadmin/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}
	return r.run(ctx, selected, q)
}

// run menjalankan satu kueri dan memindai seluruh barisnya.
func (r *Repo) run(ctx context.Context, selected plan, q inboxadmin.Query) ([]inboxadmin.WorkItem, error) {
	rows, err := r.db.QueryContext(ctx, query(selected.name), selected.args(q)...)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri %s: %w", selected.name, err)
	}
	defer rows.Close()

	items := []inboxadmin.WorkItem{}
	for rows.Next() {
		item, err := scanWorkItem(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca baris kueri %s: %w", selected.name, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri %s: %w", selected.name, err)
	}

	return items, nil
}

// CheckTable memastikan tabel inti modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int
	if err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored); err != nil {
		return fmt.Errorf("membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK: %w", err)
	}
	return nil
}

// scanner adalah bentuk minimal yang dibutuhkan scanWorkItem, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanWorkItem memindai satu baris menjadi WorkItem.
//
// Urutannya WAJIB sama dengan resultColumns dan dengan urutan kolom di inboxadmin.sql.
// Ketiganya dijaga query_test.go.
//
// Seluruh kolom dipindai lewat tipe yang mengizinkan NULL: setiap kueri mengisi hanya
// kolom yang berlaku bagi tabnya, dan sisanya memang NULL.
func scanWorkItem(row scanner) (inboxadmin.WorkItem, error) {
	var (
		caseID, reference, policyNumber, insuredName          sql.NullString
		businessName, businessSource, branchName, claimBranch sql.NullString
		creator                                               sql.NullString
		lossDate, reportDate, inputDate, lodDate              flexTime
		note, claimPosition, claimStatus, lodStatus           sql.NullString
		requestDate                                           flexTime
		policyBranch, surveyBranch, technicalPIC              sql.NullString
		surveyor, surveyNumber                                sql.NullString
		inboxDate                                             flexTime
		analystNote, rclPUCLStatus                            sql.NullString
		letterPrintDate                                       flexTime
		claimAge, expiryStatus                                sql.NullString
	)

	err := row.Scan(
		&caseID, &reference, &policyNumber, &insuredName,
		&businessName, &businessSource, &branchName, &claimBranch, &creator,
		&lossDate, &reportDate, &inputDate, &lodDate,
		&note, &claimPosition, &claimStatus, &lodStatus,
		&requestDate, &policyBranch, &surveyBranch, &technicalPIC,
		&surveyor, &surveyNumber,
		&inboxDate, &analystNote, &rclPUCLStatus, &letterPrintDate,
		&claimAge, &expiryStatus,
	)
	if err != nil {
		return inboxadmin.WorkItem{}, err
	}

	return inboxadmin.WorkItem{
		CaseID:          caseID.String,
		Reference:       reference.String,
		PolicyNumber:    policyNumber.String,
		InsuredName:     insuredName.String,
		BusinessName:    businessName.String,
		BusinessSource:  businessSource.String,
		BranchName:      branchName.String,
		ClaimBranch:     claimBranch.String,
		Creator:         creator.String,
		LossDate:        lossDate.value(),
		ReportDate:      reportDate.value(),
		InputDate:       inputDate.value(),
		LODDate:         lodDate.value(),
		Note:            note.String,
		ClaimPosition:   claimPosition.String,
		ClaimStatus:     claimStatus.String,
		LODStatus:       lodStatus.String,
		RequestDate:     requestDate.value(),
		PolicyBranch:    policyBranch.String,
		SurveyBranch:    surveyBranch.String,
		TechnicalPIC:    technicalPIC.String,
		Surveyor:        surveyor.String,
		SurveyNumber:    surveyNumber.String,
		InboxDate:       inboxDate.value(),
		AnalystNote:     analystNote.String,
		RCLPUCLStatus:   rclPUCLStatus.String,
		LetterPrintDate: letterPrintDate.value(),
		ClaimAge:        claimAge.String,
		ExpiryStatus:    expiryStatus.String,
	}, nil
}

// flexTime memindai kolom tanggal yang di sistem lama TIDAK selalu bertipe tanggal.
//
// `REPORTDATE_1` pada DATAPEGA.PC_ASM_FW_GCNMFW_WORK bertipe VARCHAR2 berisi teks DateTime
// Pega (`20200105T170000.000 GMT`), sementara kolom tanggal lain bertipe DATE atau
// TIMESTAMP. sql.NullTime menolak teks ("unsupported Scan"), dan galat itu menggagalkan
// seluruh tab — terukur 2026-10-07 pada tab ALL dan Branch Claim.
//
// Teks diurai sebagai waktu GMT, sama seperti nilai TIMESTAMP yang disimpan Pega. Teks
// yang tidak dapat diurai menjadi NULL, bukan galat: satu sel yang rusak tidak boleh
// mengosongkan seluruh antrean.
//
// Pointer pada value(), bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan
// dari "belum diisi" saat ditampilkan, dan kolom Aging yang dihitung darinya akan
// menghasilkan angka dalam ratusan ribu hari.
type flexTime struct {
	at    time.Time
	valid bool
}

// Scan memenuhi sql.Scanner.
func (f *flexTime) Scan(src any) error {
	f.at, f.valid = time.Time{}, false
	switch v := src.(type) {
	case nil:
	case time.Time:
		f.at, f.valid = v, true
	case string:
		f.at, f.valid = parsePegaTime(v)
	case []byte:
		f.at, f.valid = parsePegaTime(string(v))
	default:
		return fmt.Errorf("tipe tanggal %T tidak dikenali", src)
	}
	return nil
}

func (f flexTime) value() *time.Time {
	if !f.valid {
		return nil
	}
	at := f.at
	return &at
}

var pegaTimeLayouts = []string{
	"20060102T150405.000 MST",
	"20060102T150405 MST",
	"20060102T150405.000",
	"20060102",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parsePegaTime(text string) (time.Time, bool) {
	text = strings.TrimSpace(text)
	for _, layout := range pegaTimeLayouts {
		if at, err := time.Parse(layout, text); err == nil {
			return at.UTC(), true
		}
	}
	return time.Time{}, false
}
