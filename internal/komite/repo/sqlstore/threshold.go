package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// Repo membaca POOLDATA.EMAILKOMITE.
//
// # Kenapa modul ini TIDAK menulis
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel.
// Berbeda dari M_STS_CLAIM — yang layarnya berpindah utuh sehingga kepemilikannya ikut
// berpindah — tabel ini masih ditulis Pega dan dibaca 17 kueri di sana. Work Owner
// menetapkan 2026-09-17 bahwa aplikasi ini MEMBACA SAJA.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// ListThresholds membaca seluruh baris master ambang komite.
func (r *Repo) ListThresholds(ctx context.Context) ([]komite.Threshold, error) {
	rows, err := r.db.QueryContext(ctx, query("ambang_komite_daftar"))
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca master ambang: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []komite.Threshold
	for rows.Next() {
		threshold, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("komite/sqlstore: membaca baris ambang: %w", err)
		}
		result = append(result, threshold)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri master ambang: %w", err)
	}
	return result, nil
}

// CheckTable memastikan tabel dan seluruh kolomnya dapat dibaca akun aplikasi, tanpa
// mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip: tabelnya
// tidak ada versus akun aplikasi tidak punya hak baca. Tanpa pembedaan itu, keduanya
// muncul sebagai galat yang sama dan mengirim orang menelusuri arah yang salah.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("ambang_komite_periksa_tabel"))
	if err != nil {
		return fmt.Errorf("komite/sqlstore: memeriksa tabel ambang: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// scanner adalah bagian *sql.Rows dan *sql.Row yang dipakai di sini, supaya satu fungsi
// pemindai melayani keduanya.
type scanner interface {
	Scan(dest ...any) error
}

// scanRow mengubah satu baris EMAILKOMITE menjadi Threshold.
//
// # Kenapa seluruh kolom dipindai ke `any`, bukan ke tipe yang tepat
//
// Isi tabel ini tidak seragam tipenya. Pada berkas master yang diserahkan Work Owner,
// DEGREE, STS_AKTIF, dan TYPE_KOMITE tersimpan sebagai TEKS bertanda kutip ("1"), bukan
// angka — sementara LIMIT_BOTTOM dan LIMIT_TOP memang angka. Ditambah lagi, DDL tabel ini
// belum pernah kita lihat (`R-08` masih terbuka), sehingga tipe kolom yang sebenarnya
// baru akan diketahui saat kueri pertama benar-benar dijalankan.
//
// Memindai ke tipe yang ditebak akan membuat modul ini gagal dengan galat konversi yang
// tidak menjelaskan apa pun. Memindai ke `any` lalu menafsirkannya secara eksplisit
// membuat modul ini tahan terhadap kedua kemungkinan, dan kegagalannya menyebut kolom
// mana yang bermasalah.
func scanRow(row scanner) (komite.Threshold, error) {
	var (
		id, name, operator          any
		line, committeeType         any
		lowerBound, upperBound, deg any
		active, adj, reg, rej, abs  any
	)

	if err := row.Scan(
		&id, &name, &operator,
		&line, &committeeType,
		&lowerBound, &upperBound, &deg,
		&active, &adj, &reg, &rej, &abs,
	); err != nil {
		return komite.Threshold{}, err
	}

	lower, err := money.FromSQLValue(lowerBound)
	if err != nil {
		return komite.Threshold{}, fmt.Errorf("kolom LIMIT_BOTTOM: %w", err)
	}
	upper, err := money.FromSQLValue(upperBound)
	if err != nil {
		return komite.Threshold{}, fmt.Errorf("kolom LIMIT_TOP: %w", err)
	}

	// DEGREE yang tidak dapat dibaca DIABAIKAN menjadi nol, bukan menggagalkan seluruh
	// pembacaan. Alasannya: baris ber-DEGREE kosong memang ada di master — ID 9 dan ID
	// 10 — dan keduanya bukan jenjang persetujuan, sehingga tidak pernah ikut dihitung.
	// Menggagalkan pembacaan karena baris yang toh akan disaring akan membuat seluruh
	// layar kosong karena data yang tidak relevan.
	tier, _ := toInt(deg)

	return komite.Threshold{
		ID:              toText(id),
		Name:            toText(name),
		OperatorID:      toText(operator),
		BusinessLine:    komite.BusinessLine(toText(line)),
		CommitteeType:   toText(committeeType),
		LowerBound:      lower,
		UpperBound:      upper,
		Tier:            tier,
		Active:          toFlag(active),
		ForAdjustment:   toFlag(adj),
		ForRegistration: toFlag(reg),
		ForRejection:    toFlag(rej),
		Absent:          toFlag(abs),
	}.Normalized(), nil
}

// toText mengubah nilai kolom menjadi teks, apa pun bentuknya di driver.
//
// Spasi tepi dibuang di sini DAN sekali lagi oleh Threshold.Normalized. Pengulangan itu
// disengaja: kolom CHAR Oracle mengembalikan padding — masalah yang sama sudah terbukti
// pada OLD_LSC_ID di modul Master Status Klaim — dan baris ID 4 pada master ini bahkan
// menyimpan OPERATOR_ID yang diakhiri BARIS BARU.
func toText(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// 'f' dengan presisi -1, bukan 'g': 'g' menghasilkan notasi ilmiah untuk angka
		// besar, sehingga ID 100000000 akan menjadi "1e+08".
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

// toInt membaca kolom yang berisi bilangan bulat, baik tersimpan sebagai angka
// maupun sebagai teks.
func toInt(value any) (int, error) {
	switch v := value.(type) {
	case nil:
		return 0, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		text := toText(value)
		if text == "" {
			return 0, nil
		}
		return strconv.Atoi(text)
	}
}

// toFlag membaca kolom penanda.
//
// Yang dianggap menyala HANYA "1" dan "Y", sesuai cara seluruh kueri lama membacanya
// (`STS_AKTIF = '1'`). Kosong, NULL, dan "0" berarti padam.
//
// Ini bukan tempat untuk bermurah hati: penanda STS_ADJ yang salah dibaca sebagai
// menyala akan memasukkan baris pemberitahuan registrasi ke dalam daftar penyetuju uang.
func toFlag(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int64:
		return v == 1
	case float64:
		return v == 1
	default:
		text := strings.ToUpper(toText(value))
		return text == "1" || text == "Y"
	}
}
