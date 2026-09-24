package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// InboxRepo membaca kasus komite dari tabel WARISAN.
//
// # Ia tidak pernah menulis
//
// Keempat tabel yang dibacanya masih ditulis Pega, dan `P-1` menetapkan satu tabel hanya
// boleh ditulis satu sistem. Keputusan komite ditulis ke tabel milik aplikasi ini
// sendiri — lihat DecisionRepo pada decision.go.
//
// # POOLDATA dan DATAPEGA yang mana
//
// Setiap server punya POOLDATA-nya sendiri, dan aplikasi lama membaca POOLDATA milik
// server tempat ia berjalan (Work Owner, 2026-09-19). Kueri ini karena itu WAJIB
// dijalankan pada koneksi portal yang sedang AKTIF, bukan pada koneksi utama. Selama itu
// belum terpasang (`TKT-F6-002`), portal mana pun yang dipilih akan membaca antrean
// komite milik portal utama — pekerjaan milik badan hukum lain, tanpa satu pun tanda.
// Catatannya ada di cmd/claimpnc/main.go.
type InboxRepo struct {
	db *sql.DB
}

// NewInboxRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewInboxRepo(db *sql.DB) *InboxRepo { return &InboxRepo{db: db} }

// ListCases membaca satu halaman inbox beserta jumlah seluruh yang cocok.
//
// Jumlahnya dibaca LEBIH DULU, dalam kueri terpisah. Menghitungnya dari panjang hasil
// halaman akan selalu salah kecuali seluruh baris muat dalam satu halaman — dan layar
// memakainya untuk memberi tahu bahwa masih ada yang belum tampak.
func (r *InboxRepo) ListCases(ctx context.Context, f komite.InboxFilter) (komite.InboxPage, error) {
	f = f.Normalize()

	// Operator kosong berarti NOL BARIS, bukan semua baris. Dijawab tanpa menyentuh basis
	// data sama sekali: kueri yang penyaring pemiliknya kosong tidak boleh pernah
	// dikirim, supaya satu salah ketik pada klausa WHERE tidak dapat membuatnya
	// mengembalikan seluruh antrean komite perusahaan.
	if f.Operator == "" {
		return komite.InboxPage{Cases: []komite.CommitteeCase{}}, nil
	}

	var total int
	if err := r.db.QueryRowContext(ctx, query("inbox_count"), countArgs(f)...).Scan(&total); err != nil {
		return komite.InboxPage{}, fmt.Errorf("komite/sqlstore: menghitung inbox: %w", err)
	}

	args := append(countArgs(f), f.Offset, f.Limit)
	rows, err := r.db.QueryContext(ctx, query("inbox_list"), args...)
	if err != nil {
		return komite.InboxPage{}, fmt.Errorf("komite/sqlstore: membaca inbox: %w", err)
	}
	defer func() { _ = rows.Close() }()

	cases := make([]komite.CommitteeCase, 0, f.Limit)
	for rows.Next() {
		found, err := scanCase(rows)
		if err != nil {
			return komite.InboxPage{}, fmt.Errorf("komite/sqlstore: membaca baris inbox: %w", err)
		}
		cases = append(cases, found)
	}
	if err := rows.Err(); err != nil {
		return komite.InboxPage{}, fmt.Errorf("komite/sqlstore: menelusuri inbox: %w", err)
	}

	return komite.InboxPage{Cases: cases, Total: total}, nil
}

// Summarize menghitung isi ketiga kotak dalam satu perjalanan ke basis data.
func (r *InboxRepo) Summarize(ctx context.Context, f komite.InboxFilter) (komite.InboxSummary, error) {
	f = f.Normalize()
	if f.Operator == "" {
		return komite.InboxSummary{}, nil
	}

	var outstanding, accepted, rejected sql.NullInt64
	err := r.db.QueryRowContext(ctx, query("inbox_summary"), summaryArgs(f)...).
		Scan(&outstanding, &accepted, &rejected)
	if err != nil {
		return komite.InboxSummary{}, fmt.Errorf("komite/sqlstore: menghitung isi kotak: %w", err)
	}

	// SUM atas nol baris mengembalikan NULL, bukan 0. Membacanya sebagai int biasa akan
	// GAGAL untuk pengguna yang inbox-nya memang kosong — yaitu setiap orang pada hari
	// pertama modul ini menyala.
	return komite.InboxSummary{
		Outstanding: int(outstanding.Int64),
		Accepted:    int(accepted.Int64),
		Rejected:    int(rejected.Int64),
	}, nil
}

// FindCase mengambil satu kasus tanpa memandang pemiliknya.
func (r *InboxRepo) FindCase(ctx context.Context, caseID string) (komite.CommitteeCase, error) {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		return komite.CommitteeCase{}, komite.ErrCaseNotFound
	}

	row := r.db.QueryRowContext(ctx, query("inbox_get"), caseID)
	found, err := scanCase(row)
	if errors.Is(err, sql.ErrNoRows) {
		return komite.CommitteeCase{}, komite.ErrCaseNotFound
	}
	if err != nil {
		return komite.CommitteeCase{}, fmt.Errorf("komite/sqlstore: membaca kasus komite: %w", err)
	}
	return found, nil
}

// CheckTables memastikan seluruh tabel warisan dapat dibaca akun aplikasi.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip di layar:
// tabelnya tidak ada versus tidak punya hak baca.
func (r *InboxRepo) CheckTables(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("inbox_check_table"))
	if err != nil {
		return fmt.Errorf("komite/sqlstore: memeriksa tabel inbox komite: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// countArgs menyusun argumen penyaring sesuai urutan penanda pada inbox_count.
//
// Urutannya WAJIB sama dengan inbox_list sampai penanda terakhir sebelum paginasi —
// itulah sebabnya inbox_list memakai daftar yang sama lalu menambahkan offset dan limit
// di belakangnya. Menyusun dua daftar terpisah akan membuat keduanya berbeda diam-diam.
func countArgs(f komite.InboxFilter) []any {
	kind := string(f.Kind)
	search := nilIfEmpty(searchPattern(f.Search))
	from, to := dateBounds(f)

	return []any{
		f.Operator, f.Operator, f.Operator,
		kind, kind, kind,
		search, search, search,
		from, from,
		to, to,
	}
}

// summaryArgs menyusun argumen inbox_summary, yang TIDAK memakai penyaring kotak.
func summaryArgs(f komite.InboxFilter) []any {
	search := nilIfEmpty(searchPattern(f.Search))
	from, to := dateBounds(f)

	return []any{
		f.Operator, f.Operator, f.Operator,
		search, search, search,
		from, from,
		to, to,
	}
}

// dateBounds mengubah rentang tanggal WIB menjadi batas yang dapat dibandingkan SQL.
//
// # Ujung atas digeser satu hari, dan perbandingannya `<`
//
// Rentang "Tgl Input Sampai" inklusif bagi pengguna: mencari sampai tanggal 20 harus ikut
// menampilkan yang masuk pukul 23:59 tanggal 20. Membandingkan `<= tanggal 20` terhadap
// kolom bertipe waktu akan memotongnya di tengah malam dan membuang sehari penuh — satu
// kesalahan paling umum pada penyaring tanggal, dan akibatnya pekerjaan hari itu tampak
// tidak ada.
//
// # Yang BELUM pasti, dan harus diverifikasi terhadap data sungguhan
//
// Batasnya disusun sebagai tengah malam WIB, sementara `PXCREATEDATETIME` disimpan Pega
// dalam GMT lalu digeser tujuh jam secara MANUAL di setiap tempat yang membutuhkannya —
// 118 titik di 36 activity (`R-12`, `F-5`). Bila sebagian data tersimpan SUDAH tergeser
// dan sebagian belum, batas rentang akan meleset tujuh jam pada sebagian baris.
//
// Ini tidak dapat dipastikan dari export; ia menuntut pembandingan terhadap data nyata di
// staging (`D-53`). Dicatat sebagai pertanyaan terbuka, bukan diselesaikan dengan tebakan.
func dateBounds(f komite.InboxFilter) (any, any) {
	var from, to any
	if !f.DateFrom.IsZero() {
		from = f.DateFrom
	}
	if !f.DateTo.IsZero() {
		to = f.DateTo.AddDate(0, 0, 1)
	}
	return from, to
}

// scanCase memetakan satu baris hasil menjadi kasus komite.
//
// Antarmuka `scanner` yang dipakainya dideklarasikan di threshold.go — satu untuk seluruh
// paket, sehingga *sql.Rows dan *sql.Row dilayani fungsi pemindai yang sama.
//
// # Di sinilah nama kolom warisan berhenti
//
// Pemetaan dari nama kolom Pega ke nama domain terjadi HANYA di fungsi ini. Lapisan di
// atasnya tidak pernah melihat `IBNR`, `pyScore`, maupun `DraftWordingID` — tiga alias
// yang di sistem lama sama sekali tidak mencerminkan isinya.
// # Seluruh kolom dipindai ke `any`, bukan ke tipe yang tepat
//
// Alasannya sama dengan scanRow pada threshold.go, dan di sini lebih kuat lagi: DDL tabel
// warisan BELUM PERNAH KITA LIHAT (`R-08` masih terbuka), dan isi master yang sudah
// diserahkan membuktikan kolom yang tampak angka dapat tersimpan sebagai TEKS.
//
// Memindai ke tipe yang ditebak akan membuat SELURUH inbox seseorang gagal dengan galat
// konversi yang tidak menyebut kolom mana yang bermasalah — pada layar yang berisi
// pekerjaannya, bukan pada layar acuan yang dapat ditunda.
//
// Kolom waktu adalah pengecualian: `sql.NullTime` tidak menebak apa pun, ia hanya
// membedakan NULL dari nilai yang ada.
func scanCase(row scanner) (komite.CommitteeCase, error) {
	var (
		caseID           any
		claimNumber      any
		policyNumber     any
		insuredName      any
		businessName     any
		sourceOfBusiness any
		branchName       any
		groupPanel       any
		claimPIC         any
		assignedOperator any
		committeeDate    sql.NullTime
		createdAt        sql.NullTime
		workStatus       any
		typeKomite       any
		paymentType      any
		claimValue       any
		asmShareValue    any
		orValue          any
		committeeNote    any
		legacyApprove    any
		legacyTier       any
		aiResult         any
		aiNoteAccepted   any
		aiNoteRejected   any
		aiAssessedAt     sql.NullTime
		aiPresent        any
	)

	if err := row.Scan(
		&caseID, &claimNumber, &policyNumber, &insuredName, &businessName,
		&sourceOfBusiness, &branchName, &groupPanel, &claimPIC, &assignedOperator,
		&committeeDate, &createdAt, &workStatus,
		&typeKomite, &paymentType,
		&claimValue, &asmShareValue, &orValue,
		&committeeNote, &legacyApprove, &legacyTier,
		&aiResult, &aiNoteAccepted, &aiNoteRejected, &aiAssessedAt, &aiPresent,
	); err != nil {
		return komite.CommitteeCase{}, err
	}

	id := toText(caseID)

	// Nilai uang TIDAK boleh gagal diam-diam. Berbeda dari kolom keterangan, ia yang
	// dibaca anggota komite saat memutuskan — dan nol yang seharusnya empat puluh lima
	// juta adalah kesalahan yang tidak terlihat keliru.
	claim, err := moneyOrZero(claimValue)
	if err != nil {
		return komite.CommitteeCase{}, fmt.Errorf("nilai klaim kasus %s: %w", id, err)
	}
	share, err := moneyOrZero(asmShareValue)
	if err != nil {
		return komite.CommitteeCase{}, fmt.Errorf("nilai ASM share kasus %s: %w", id, err)
	}
	orShare, err := moneyOrZero(orValue)
	if err != nil {
		return komite.CommitteeCase{}, fmt.Errorf("nilai OR ASM kasus %s: %w", id, err)
	}

	// Jenjang yang tidak terbaca TIDAK menggagalkan barisnya. Ia keterangan pelengkap —
	// jenjang yang tercatat di Pega — dan menahan seluruh pekerjaan seseorang karena satu
	// kolom pelengkap bermasalah adalah pertukaran yang jelas keliru.
	tier, tierErr := toInt(legacyTier)
	if tierErr != nil {
		tier = 0
	}

	found := komite.CommitteeCase{
		CaseID:           id,
		ClaimNumber:      toText(claimNumber),
		PolicyNumber:     toText(policyNumber),
		InsuredName:      toText(insuredName),
		BusinessName:     toText(businessName),
		SourceOfBusiness: toText(sourceOfBusiness),
		BranchName:       toText(branchName),
		GroupPanel:       toText(groupPanel),
		ClaimPIC:         toText(claimPIC),
		AssignedOperator: toText(assignedOperator),
		CommitteeDate:    committeeDate.Time,
		CreatedAt:        createdAt.Time,
		WorkStatus:       toText(workStatus),

		// Tipe Komite diturunkan di GO, bukan di SQL. Bentuk aslinya menjalankan ENAM
		// subkueri berkorelasi ke tabel yang sama untuk satu baris; di sini ia satu tabel
		// keputusan yang dapat diuji tanpa basis data.
		CommitteeKind: komite.CommitteeKindOf(toText(typeKomite), toText(paymentType)),

		ClaimValue:    claim,
		ASMShareValue: share,
		ORValue:       orShare,

		CommitteeNote: toText(committeeNote),
		LegacyOutcome: legacyOutcome(toText(legacyApprove)),
		LegacyTier:    tier,

		AIResult:        toText(aiResult),
		AINoteAccepted:  toText(aiNoteAccepted),
		AINoteRejected:  toText(aiNoteRejected),
		AIAssessedAt:    aiAssessedAt.Time,
		HasAIAssessment: toFlag(aiPresent),
	}
	return found.Normalized(), nil
}

// legacyOutcome menurunkan keputusan yang tercatat di Pega dari kolom STATUSAPPROVE.
//
// # Cabang `else` rule lama TIDAK ditiru, dan itu disengaja
//
// `GetKomitePAditerima` menurunkannya begitu saja:
//
//	case when b.STATUSAPPROVE = '1' then 'DITERIMA' else 'DITOLAK' end
//
// Kolom yang KOSONG bukan '1', sehingga kasus yang belum diputuskan ikut terbaca
// "DITOLAK" di sana. Menirunya akan membuat setiap klaim yang masih menunggu tampak sudah
// ditolak — bukan kesetaraan perilaku, melainkan penyalinan cacat ke tempat yang lebih
// terlihat.
//
// Di sini ketiadaan nilai menjadi OutcomePending, dan hanya nilai yang benar-benar ada
// yang menjadi diterima atau ditolak.
func legacyOutcome(approve string) komite.Outcome {
	switch strings.TrimSpace(approve) {
	case "":
		return komite.OutcomePending
	case "1":
		return komite.OutcomeApproved
	default:
		return komite.OutcomeRejected
	}
}

// moneyOrZero membaca nilai uang dari basis data, memperlakukan NULL sebagai nol.
//
// NULL berarti kasus itu belum punya baris di `T_CLAIM_KOMITE_LIST` — keadaan yang nyata
// pada kasus yang baru masuk komite. Ia bukan galat, dan menolaknya akan membuat kasus
// yang paling perlu dikerjakan justru tidak dapat ditampilkan.
func moneyOrZero(value any) (money.Money, error) {
	if value == nil {
		return money.Zero, nil
	}
	return money.FromSQLValue(value)
}

// nilIfEmpty mengubah teks kosong menjadi NULL.
//
// Itulah yang membuat satu kueri melayani seluruh gabungan penyaring tanpa merangkai teks
// SQL: penyaring yang tidak diisi menjadi NULL, dan `:n IS NULL OR …` membuatnya tidak
// mempersempit apa pun.
func nilIfEmpty(s string) any {
	if trimmed := strings.TrimSpace(s); trimmed != "" {
		return trimmed
	}
	return nil
}

// searchPattern melindungi wildcard LIKE yang diketik pengguna.
//
// Tanpa ini, mencari "100%" akan cocok dengan SEMUA baris. Pada layar yang menampilkan
// nilai klaim dan nama tertanggung, penyaring yang diam-diam melebar berarti anggota
// komite melihat pekerjaan yang tidak ia cari.
//
// Karakter pelariannya backslash, sama dengan klausa ESCAPE pada kuerinya. Backslash itu
// sendiri dilarikan LEBIH DULU, supaya pelariannya tidak dilarikan dua kali.
func searchPattern(search string) string {
	search = strings.TrimSpace(search)
	if search == "" {
		return ""
	}
	search = strings.ReplaceAll(search, `\`, `\\`)
	search = strings.ReplaceAll(search, "%", `\%`)
	search = strings.ReplaceAll(search, "_", `\_`)
	return search
}

var _ komite.InboxRepo = (*InboxRepo)(nil)
