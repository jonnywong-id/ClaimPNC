package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
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
// sedangkan Unregistered RCV menerima tiga. Menyeragamkannya menuntut setiap kueri
// menyebut bind yang tidak dipakainya — dan bind yang hanya ada supaya jumlahnya genap
// adalah bind yang akan membingungkan orang berikutnya yang membaca SQL-nya.
type plan struct {
	// name adalah nama kueri di berkas .sql.
	name string

	// args menyusun argumen bind sesuai urutan `:1`, `:2`, `:3` di kueri itu.
	args func(q inboxadmin.Query) []any
}

// plans memetakan kode tab ke kuerinya.
//
// Peta ini adalah satu-satunya tempat kode tab bertemu nama kueri. Tab yang tidak ada di
// sini menghasilkan galat yang menyebut kodenya — bukan kueri kosong yang mengembalikan nol
// baris dan terbaca seperti antrean yang memang kosong.
var plans = map[string]plan{
	inboxadmin.TabAll: {
		name: "list_all",
		args: func(q inboxadmin.Query) []any {
			return []any{keyword(q), string(q.Business)}
		},
	},
	inboxadmin.TabUnregisteredRCV: {
		name: "list_unregistered",
		args: func(q inboxadmin.Query) []any {
			return []any{keyword(q), string(q.Business), "NORMAL"}
		},
	},
	inboxadmin.TabRCVOnline: {
		name: "list_unregistered",
		args: func(q inboxadmin.Query) []any {
			return []any{keyword(q), string(q.Business), "ONLINE"}
		},
	},
	inboxadmin.TabRequestSurvey: {
		name: "list_request_survey",
		args: func(q inboxadmin.Query) []any {
			return []any{keyword(q), q.Caller.Login}
		},
	},
	inboxadmin.TabRequestDocument: {
		name: "list_request_document",
		args: func(q inboxadmin.Query) []any {
			return []any{q.Caller.Login}
		},
	},
	inboxadmin.TabAllCaseAdmin: {
		name: "list_all_case_admin",
		args: func(q inboxadmin.Query) []any {
			return []any{keyword(q), q.Caller.Login}
		},
	},
	inboxadmin.TabBranchClaim: {
		name: "list_branch_claim",
		args: func(q inboxadmin.Query) []any {
			return []any{keyword(q), string(q.Business)}
		},
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
func (r *Repo) List(ctx context.Context, q inboxadmin.Query) ([]inboxadmin.WorkItem, error) {
	selected, known := plans[q.Tab.Code]
	if !known {
		return nil, fmt.Errorf("inboxadmin/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}

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
		lossDate, reportDate, inputDate, lodDate              sql.NullTime
		note, claimPosition, claimStatus, lodStatus           sql.NullString
		requestDate                                           sql.NullTime
		policyBranch, surveyBranch, technicalPIC              sql.NullString
		surveyor, surveyNumber                                sql.NullString
		inboxDate                                             sql.NullTime
		analystNote, rclPUCLStatus                            sql.NullString
		letterPrintDate                                       sql.NullTime
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
		LossDate:        timeOrNil(lossDate),
		ReportDate:      timeOrNil(reportDate),
		InputDate:       timeOrNil(inputDate),
		LODDate:         timeOrNil(lodDate),
		Note:            note.String,
		ClaimPosition:   claimPosition.String,
		ClaimStatus:     claimStatus.String,
		LODStatus:       lodStatus.String,
		RequestDate:     timeOrNil(requestDate),
		PolicyBranch:    policyBranch.String,
		SurveyBranch:    surveyBranch.String,
		TechnicalPIC:    technicalPIC.String,
		Surveyor:        surveyor.String,
		SurveyNumber:    surveyNumber.String,
		InboxDate:       timeOrNil(inboxDate),
		AnalystNote:     analystNote.String,
		RCLPUCLStatus:   rclPUCLStatus.String,
		LetterPrintDate: timeOrNil(letterPrintDate),
		ClaimAge:        claimAge.String,
		ExpiryStatus:    expiryStatus.String,
	}, nil
}

// timeOrNil mengubah kolom tanggal yang boleh NULL menjadi pointer.
//
// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari "belum
// diisi" saat ditampilkan, dan kolom Aging yang dihitung darinya akan menghasilkan angka
// dalam ratusan ribu hari.
func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	at := value.Time
	return &at
}
