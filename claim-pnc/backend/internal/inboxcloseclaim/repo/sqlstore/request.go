package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxcloseclaim"
)

// RequestRepo membaca dan menulis POOLDATA.CPNC_PERMINTAAN_KLAIM.
//
// Tabel itu MILIK APLIKASI INI sepenuhnya — dibuat migrasi `0006`, tidak dibaca dan tidak
// ditulis Pega. `P-1` terpenuhi tanpa negosiasi kepemilikan.
//
// Ia sengaja TERPISAH dari Repo meski keduanya berada di paket yang sama dan melayani satu
// layar. Pembelahannya mengikuti kepemilikan tabel, bukan ukuran berkas: yang satu tidak
// boleh menulis apa pun, yang lain menulis. Menyatukannya akan membuat aturan itu
// bergantung pada kehati-hatian orang yang menyuntingnya berikutnya.
type RequestRepo struct {
	db *sql.DB
}

// NewRequestRepo membentuk repo permintaan.
func NewRequestRepo(db *sql.DB) *RequestRepo { return &RequestRepo{db: db} }

// UniqueKeyName dipakai menerjemahkan galat bentrok menjadi galat domain.
//
// Ia konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam. Mengganti
// namanya di migrasi tanpa mengganti konstanta ini akan membuat permintaan ganda muncul
// sebagai galat `500` alih-alih pesan yang dapat dibaca.
const UniqueKeyName = "CPNC_PERMINTAAN_KLAIM_UK_PENDING"

// maxClaimsPerQuery membatasi banyaknya klaim yang keadaannya diminta sekaligus.
//
// Oracle membatasi daftar ekspresi `IN` pada 1.000 butir. Batas di sini jauh lebih kecil dan
// cukup: satu halaman paling banyak berisi inboxcloseclaim.MaxLimit baris, dan pemanggil
// yang meminta lebih banyak dari itu sedang melakukan sesuatu yang tidak dirancang.
const maxClaimsPerQuery = 500

// Record menyimpan satu permintaan.
//
// Ia tidak pernah menimpa apa pun: berkas kuerinya tidak memuat UPDATE maupun DELETE, dan
// akun aplikasi tidak diberi hak untuk keduanya.
func (r *RequestRepo) Record(ctx context.Context, request inboxcloseclaim.ClaimRequest) error {
	request = request.Normalize()

	_, err := r.db.ExecContext(ctx, query("request_insert"),
		request.ID,
		string(request.Kind),
		request.ClaimID,
		nilIfEmpty(request.ClaimNumber),
		nilIfEmpty(request.Reason),
		string(request.Status),
		nilIfEmpty(request.EffectWorkStatus),
		nilIfEmpty(request.EffectClaimStatus),
		nilIfEmpty(request.CopyScope),
		request.ActorLogin,
		nilIfEmpty(request.ActorName),
		request.RequestedAt.UTC(),
	)
	if err != nil {
		return translateRequestWriteError(err)
	}
	return nil
}

// PendingFor mengembalikan permintaan yang masih menunggu, dikunci ClaimID.
func (r *RequestRepo) PendingFor(
	ctx context.Context,
	claimIDs []string,
) (map[string][]inboxcloseclaim.ClaimRequest, error) {
	unique := uniqueNonEmpty(claimIDs)
	if len(unique) == 0 {
		// Tanpa satu pun klaim tidak ada yang perlu ditanyakan — dan `IN ()` bukan SQL yang
		// sah di dialek mana pun.
		return map[string][]inboxcloseclaim.ClaimRequest{}, nil
	}
	if len(unique) > maxClaimsPerQuery {
		return nil, fmt.Errorf(
			"inboxcloseclaim/sqlstore: %d klaim sekaligus melebihi batas %d",
			len(unique), maxClaimsPerQuery,
		)
	}

	statement, args := expandClaims(query("request_pending_for"), unique)

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("inboxcloseclaim/sqlstore: membaca permintaan tertunda: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := map[string][]inboxcloseclaim.ClaimRequest{}
	for rows.Next() {
		request, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("inboxcloseclaim/sqlstore: membaca baris permintaan: %w", err)
		}
		result[request.ClaimID] = append(result[request.ClaimID], request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxcloseclaim/sqlstore: menelusuri permintaan tertunda: %w", err)
	}
	return result, nil
}

// CheckTable memastikan tabel permintaan dapat dibaca akun aplikasi.
//
// Dipanggil perintah `-periksa`. Tanpa pemeriksaan ini, tabel yang belum dibuat DBA baru
// ketahuan saat pengguna pertama menekan tombol ReOpen — dan yang dilihatnya adalah galat
// 500, bukan keterangan bahwa migrasi `0006` belum dijalankan di portal itu.
func (r *RequestRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("request_check_table"))
	if err != nil {
		return fmt.Errorf("inboxcloseclaim/sqlstore: memeriksa tabel permintaan: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// expandClaims mengganti penanda /*CLAIMS*/ dengan daftar penanda parameter.
//
// # Yang disisipkan hanyalah PENANDA, tidak pernah nilainya
//
// Banyaknya klaim berbeda tiap halaman, sehingga daftar `IN` tidak dapat ditulis tetap di
// berkas `.sql`. Yang dirangkai di sini adalah `:1, :2, …` — seluruh nilai tetap dikirim
// terpisah lewat parameter binding.
//
// Perbedaannya dengan pola `{ASIS:…}` warisan itulah yang menentukan: di sana NILAI
// pengguna disisipkan ke dalam teks SQL; di sini tidak ada satu pun karakter yang berasal
// dari masukan pengguna yang masuk ke dalam pernyataan.
func expandClaims(statement string, claimIDs []string) (string, []any) {
	markers := make([]string, 0, len(claimIDs))
	args := make([]any, 0, len(claimIDs))

	for i, id := range claimIDs {
		markers = append(markers, ":"+strconv.Itoa(i+1))
		args = append(args, id)
	}
	return strings.Replace(statement, "/*CLAIMS*/", strings.Join(markers, ", "), 1), args
}

// uniqueNonEmpty merapikan daftar klaim: membuang yang kosong dan yang berulang.
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

func scanRequest(row scanner) (inboxcloseclaim.ClaimRequest, error) {
	var (
		id                string
		kind              string
		claimID           string
		claimNumber       sql.NullString
		reason            sql.NullString
		status            string
		effectWorkStatus  sql.NullString
		effectClaimStatus sql.NullString
		copyScope         sql.NullString
		actorLogin        sql.NullString
		actorName         sql.NullString
		requestedAt       sql.NullTime
	)

	if err := row.Scan(
		&id, &kind, &claimID, &claimNumber, &reason, &status,
		&effectWorkStatus, &effectClaimStatus, &copyScope,
		&actorLogin, &actorName, &requestedAt,
	); err != nil {
		return inboxcloseclaim.ClaimRequest{}, err
	}

	return inboxcloseclaim.ClaimRequest{
		ID:                id,
		Kind:              inboxcloseclaim.RequestKind(strings.TrimSpace(kind)),
		ClaimID:           strings.TrimSpace(claimID),
		ClaimNumber:       strings.TrimSpace(claimNumber.String),
		Reason:            reason.String,
		Status:            inboxcloseclaim.RequestStatus(strings.TrimSpace(status)),
		EffectWorkStatus:  strings.TrimSpace(effectWorkStatus.String),
		EffectClaimStatus: strings.TrimSpace(effectClaimStatus.String),
		CopyScope:         strings.TrimSpace(copyScope.String),
		ActorLogin:        inboxcloseclaim.OperatorKey(actorLogin.String),
		ActorName:         strings.TrimSpace(actorName.String),
		RequestedAt:       requestedAt.Time,
	}, nil
}

// translateRequestWriteError menerjemahkan bentrok kunci menjadi galat domain.
//
// Yang dicocokkan adalah NAMA CONSTRAINT, bukan nomor galat driver — nama itu milik kita
// dan tidak berubah saat driver atau basis datanya berganti.
//
// Bentrok di sini berarti permintaan sejenis atas klaim yang sama masih menunggu. Lapisan
// usecase sudah memeriksanya lebih dulu, tetapi pemeriksaan itu dan penyimpanannya BUKAN
// satu operasi atomik: dua permintaan yang tiba bersamaan — dua tab yang terbuka, keduanya
// ditekan — dapat lolos keduanya. Constraint inilah yang menutup celah itu, dan terjemahan
// ini yang membuat penutupannya terbaca sebagai pesan, bukan sebagai galat 500.
//
// Pada layar ini akibatnya nyata: dua permintaan salin yang lolos berarti DUA KLAIM BARU
// dari satu tombol yang ditekan dua kali.
func translateRequestWriteError(err error) error {
	if strings.Contains(strings.ToUpper(err.Error()), UniqueKeyName) {
		return inboxcloseclaim.ErrRequestPending
	}
	return fmt.Errorf("inboxcloseclaim/sqlstore: mencatat permintaan: %w", err)
}

var _ inboxcloseclaim.RequestRepo = (*RequestRepo)(nil)
