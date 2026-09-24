package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/komite"
)

// DecisionRepo membaca dan menulis POOLDATA.CPNC_KOMITE_KEPUTUSAN.
//
// Tabel itu MILIK APLIKASI INI sepenuhnya — dibuat migrasi `0004`, tidak dibaca dan tidak
// ditulis Pega. `P-1` terpenuhi tanpa negosiasi kepemilikan.
//
// Ia sengaja TERPISAH dari InboxRepo meski keduanya berada di paket yang sama dan
// melayani satu layar. Pembelahannya mengikuti kepemilikan tabel, bukan ukuran berkas:
// yang satu tidak boleh menulis apa pun, yang lain menulis. Menyatukannya akan membuat
// aturan itu bergantung pada kehati-hatian orang yang menyuntingnya berikutnya.
type DecisionRepo struct {
	db *sql.DB
}

// NewDecisionRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang
// dituju.
func NewDecisionRepo(db *sql.DB) *DecisionRepo { return &DecisionRepo{db: db} }

// UniqueKeyName dipakai menerjemahkan galat bentrok menjadi galat domain.
//
// Ia konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam. Mengganti
// namanya di migrasi tanpa mengganti konstanta ini akan membuat keputusan ganda muncul
// sebagai galat `500` alih-alih pesan yang dapat dibaca.
const UniqueKeyName = "CPNC_KOMITE_KEPUTUSAN_UK"

// maxCasesPerQuery membatasi banyaknya kasus yang keputusannya diminta sekaligus.
//
// Oracle membatasi daftar ekspresi `IN` pada 1.000 butir. Batas di sini jauh lebih kecil
// dan cukup: satu halaman inbox paling banyak berisi komite.MaxPageSize baris, dan
// pemanggil yang meminta lebih banyak dari itu sedang melakukan sesuatu yang tidak
// dirancang.
const maxCasesPerQuery = 500

// ListForCases mengembalikan keputusan seluruh kasus yang disebut, dikunci CaseID.
//
// Banyak kasus sekaligus, bukan satu per satu: meminta keputusan per baris tabel adalah
// kueri di dalam perulangan, yang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 sebut sebagai
// hambatan peringkat ketiga.
func (r *DecisionRepo) ListForCases(
	ctx context.Context,
	caseIDs []string,
) (map[string][]komite.Decision, error) {
	unique := uniqueNonEmpty(caseIDs)
	if len(unique) == 0 {
		// Tanpa satu pun kasus tidak ada yang perlu ditanyakan — dan `IN ()` bukan SQL
		// yang sah di dialek mana pun.
		return map[string][]komite.Decision{}, nil
	}
	if len(unique) > maxCasesPerQuery {
		return nil, fmt.Errorf(
			"komite/sqlstore: %d kasus sekaligus melebihi batas %d",
			len(unique), maxCasesPerQuery,
		)
	}

	statement, args := expandCases(query("decision_list_for_cases"), unique)

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca keputusan komite: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := map[string][]komite.Decision{}
	for rows.Next() {
		decision, err := scanDecision(rows)
		if err != nil {
			return nil, fmt.Errorf("komite/sqlstore: membaca baris keputusan: %w", err)
		}
		result[decision.CaseID] = append(result[decision.CaseID], decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri keputusan komite: %w", err)
	}
	return result, nil
}

// Record menyimpan satu keputusan.
//
// Ia tidak pernah menimpa apa pun: berkas kuerinya tidak memuat UPDATE maupun DELETE, dan
// akun aplikasi tidak diberi hak untuk keduanya.
func (r *DecisionRepo) Record(ctx context.Context, d komite.Decision) error {
	_, err := r.db.ExecContext(ctx, query("decision_insert"),
		d.ID,
		d.CaseID,
		nilIfEmpty(d.ClaimNumber),
		d.Tier,
		string(d.Kind),
		nilIfEmpty(d.Note),
		komite.OperatorKey(d.ActorLogin),
		nilIfEmpty(d.ActorName),
		d.DecidedAt.UTC(),
	)
	if err != nil {
		return translateDecisionWriteError(err)
	}
	return nil
}

// CheckTable memastikan tabel keputusan dapat dibaca akun aplikasi.
func (r *DecisionRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("decision_check_table"))
	if err != nil {
		return fmt.Errorf("komite/sqlstore: memeriksa tabel keputusan komite: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// expandCases mengganti penanda /*CASES*/ dengan daftar penanda parameter.
//
// # Yang disisipkan hanyalah PENANDA, tidak pernah nilainya
//
// Banyaknya kasus berbeda tiap halaman, sehingga daftar `IN` tidak dapat ditulis tetap di
// berkas `.sql`. Yang dirangkai di sini adalah `:1, :2, …` — seluruh nilai tetap dikirim
// terpisah lewat parameter binding.
//
// Perbedaannya dengan pola `{ASIS:...}` warisan itulah yang menentukan: di sana NILAI
// pengguna disisipkan ke dalam teks SQL (538 kemunculan, `K-29`); di sini tidak ada satu
// pun karakter yang berasal dari masukan pengguna yang masuk ke dalam pernyataan.
func expandCases(statement string, caseIDs []string) (string, []any) {
	markers := make([]string, 0, len(caseIDs))
	args := make([]any, 0, len(caseIDs))

	for i, id := range caseIDs {
		markers = append(markers, ":"+strconv.Itoa(i+1))
		args = append(args, id)
	}
	return strings.Replace(statement, "/*CASES*/", strings.Join(markers, ", "), 1), args
}

// uniqueNonEmpty merapikan daftar kasus: membuang yang kosong dan yang berulang.
//
// Yang berulang dibuang bukan demi kerapian melainkan demi batas: satu halaman inbox
// dapat memuat nomor yang sama dua kali bila datanya cacat, dan itu tidak boleh menambah
// panjang daftar `IN`.
func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))

	for _, v := range values {
		clean := strings.TrimSpace(v)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		result = append(result, clean)
	}
	return result
}

func scanDecision(row scanner) (komite.Decision, error) {
	var (
		id          string
		caseID      string
		claimNumber sql.NullString
		tier        sql.NullInt64
		kind        string
		note        sql.NullString
		actorLogin  sql.NullString
		actorName   sql.NullString
		decidedAt   sql.NullTime
	)

	if err := row.Scan(
		&id, &caseID, &claimNumber, &tier, &kind, &note, &actorLogin, &actorName, &decidedAt,
	); err != nil {
		return komite.Decision{}, err
	}

	return komite.Decision{
		ID:          id,
		CaseID:      strings.TrimSpace(caseID),
		ClaimNumber: strings.TrimSpace(claimNumber.String),
		Tier:        int(tier.Int64),
		Kind:        komite.DecisionKind(strings.TrimSpace(kind)),
		Note:        note.String,
		ActorLogin:  komite.OperatorKey(actorLogin.String),
		ActorName:   strings.TrimSpace(actorName.String),
		DecidedAt:   decidedAt.Time,
	}, nil
}

// translateDecisionWriteError menerjemahkan bentrok kunci menjadi galat domain.
//
// Yang dicocokkan adalah NAMA CONSTRAINT, bukan nomor galat driver — nama itu milik kita
// dan tidak berubah saat driver atau basis datanya berganti.
//
// Bentrok di sini berarti orang yang sama menekan tombol dua kali, biasanya dari dua tab
// yang terbuka bersamaan. Lapisan usecase sudah memeriksanya lebih dulu, tetapi
// pemeriksaan itu dan penyimpanannya BUKAN satu operasi atomik — dua permintaan yang tiba
// bersamaan dapat lolos keduanya. Constraint inilah yang menutup celah itu, dan
// terjemahan ini yang membuat penutupannya terbaca sebagai pesan, bukan sebagai galat 500.
func translateDecisionWriteError(err error) error {
	if strings.Contains(strings.ToUpper(err.Error()), UniqueKeyName) {
		return komite.ErrDecisionClosed
	}
	return fmt.Errorf("komite/sqlstore: mencatat keputusan komite: %w", err)
}

var _ komite.DecisionRepo = (*DecisionRepo)(nil)
