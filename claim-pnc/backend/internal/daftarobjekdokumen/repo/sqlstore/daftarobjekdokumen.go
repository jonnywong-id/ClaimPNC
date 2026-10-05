// Package sqlstore memenuhi seam daftarobjekdokumen.Repo dan BusinessRepo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (`ADR-0030` Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
//
// # Isi master ini tinggal di dokumen JSON
//
// Bukan di kolom. Alasannya, beserta buktinya dari katalog dan dari data, ada di kepala
// `daftarobjekdokumen.sql`. Yang perlu diketahui saat membaca berkas ini:
//
//   - keterangan dan pemetaan bisnis dibaca dari `JSON_DATA`
//   - penyimpanan menulis `JSON_DATA` **dan** kolom `KET_DOC_OBJ`
//   - dokumen JSON disusun di Go, bukan dengan fungsi JSON di SQL
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar `ListDocumentObject` adalah
// satu-satunya layar Pega yang menulis objek dokumen. Memindahkan layarnya karena itu
// memindahkan kepemilikan tabelnya secara utuh.
//
// POOLDATA.BUSINESS TIDAK termasuk: ia milik GISFW dan hanya dibaca (`D-03`).
package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/daftarobjekdokumen"
)

// sequenceDigits adalah lebar nomor urut pada ID.
//
// LIMA, dan angkanya dibaca dari procedure yang benar-benar ada di basis data —
// `POOLDATA.PEGA_LST_DOC_OBJ` membentuk ID dengan `lpad(to_char(LST_DOC_OBJ_SEQ.nextval),
// 5, '0')`. Ditegaskan data: ID yang terpakai hari ini `100766`..`100777`, enam karakter,
// dengan kode situs "1" di depannya.
//
// Versi pertama modul ini memasang EMPAT, disalin dari procedure tabel bersaudara
// `PEGA_LST_DOC_TYPE`. Lebar nomor urut berbeda per tabel dan harus dibaca dari procedure
// tabel itu sendiri, bukan dari tetangganya.
const sequenceDigits = 5

// Repo membaca dan menulis POOLDATA.LST_DOC_OBJ.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// Kunci di dalam dokumen JSON, persis seperti yang ditulis sistem lama
// (`POOLDATA.PROCESS_LST_DOC_OBJ`) dan seperti yang terbaca pada data hari ini.
const (
	keyID          = "ID"
	keyListLBUID   = "LIST_LBU_ID"
	keyDescription = "KET_DOC_OBJ"
)

// document adalah dokumen JSON satu baris, disimpan sebagai PETA kunci mentah.
//
// Peta, bukan struct bertipe, dengan satu alasan: kunci yang TIDAK dikenal modul ini pun
// ikut terbawa saat baris disimpan ulang. Dokumen pada data hari ini hanya memuat ketiga
// kunci di atas, tetapi menyusun ulang dengan struct bertipe berarti kunci keempat yang
// kelak ditambahkan siapa pun akan hilang diam-diam pada penyuntingan pertama.
//
// Yang TIDAK dijamin peta ini adalah urutan kuncinya — Go mengurutkan kunci peta saat
// menulis JSON. Itu tidak berakibat apa pun: baik `JSON_VALUE` maupun `JSON_TABLE` tidak
// peduli urutan, dan view POOLDATA.V_LST_DOC_OBJ_BISNIS membaca lewat jalur, bukan posisi.
type document map[string]json.RawMessage

type documentLBU struct {
	ID string `json:"ID"`
}

// List membaca seluruh objek dokumen, tanpa pemetaan bisnisnya.
func (r *Repo) List(ctx context.Context) ([]daftarobjekdokumen.DocumentObject, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("document_object_list"))
	if err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftarobjekdokumen.DocumentObject
	for rows.Next() {
		var id, description, oldID sql.NullString
		if err := rows.Scan(&id, &description, &oldID); err != nil {
			return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca baris: %w", err)
		}
		result = append(result, daftarobjekdokumen.DocumentObject{
			ID:          strings.TrimSpace(id.String),
			Description: strings.TrimSpace(description.String),
			OldID:       strings.TrimSpace(oldID.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu objek dokumen LENGKAP dengan pemetaan bisnisnya.
//
// Nama bisnisnya TIDAK diisi di sini — yang tersimpan hanyalah ID. Pelengkapannya ada di
// usecase, yang memang sudah membaca master bisnis.
func (r *Repo) Get(ctx context.Context, id string) (daftarobjekdokumen.DocumentObject, error) {
	row, _, err := r.read(ctx, id)
	return row, err
}

// read mengembalikan barisnya beserta dokumen JSON mentahnya.
//
// Dokumen mentah dibutuhkan jalur tulis: bagian dokumen yang TIDAK dikenal modul ini pun
// harus ikut terbawa saat disimpan ulang, bukan hilang diam-diam.
func (r *Repo) read(ctx context.Context, id string) (daftarobjekdokumen.DocumentObject, document, error) {
	var rawID, description, oldID, raw sql.NullString

	err := r.db.QueryRowContext(ctx, getQuery("document_object_get"), id).
		Scan(&rawID, &description, &oldID, &raw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftarobjekdokumen.DocumentObject{}, document{}, daftarobjekdokumen.ErrNotFound
	case err != nil:
		return daftarobjekdokumen.DocumentObject{}, document{}, fmt.Errorf(
			"daftarobjekdokumen/sqlstore: membaca %q: %w", id, err)
	}

	doc := parseDocument(raw.String)

	row := daftarobjekdokumen.DocumentObject{
		ID:          strings.TrimSpace(rawID.String),
		Description: strings.TrimSpace(description.String),
		OldID:       strings.TrimSpace(oldID.String),
		Businesses:  businessesOf(doc),
	}
	return row, doc, nil
}

// parseDocument membaca dokumen JSON, dan memaafkan isi yang tidak dapat dibaca.
//
// Dokumen yang rusak TIDAK menggagalkan pembacaan: barisnya tetap tampil, hanya pemetaan
// bisnisnya yang kosong. Menggagalkan seluruh baris karena dokumennya cacat berarti satu
// baris rusak membuat layar tidak dapat dibuka sama sekali.
func parseDocument(raw string) document {
	if strings.TrimSpace(raw) == "" {
		return document{}
	}
	var doc document
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return document{}
	}
	return doc
}

// businessesOf membongkar LIST_LBU_ID menjadi daftar bisnis — ID saja.
//
// Namanya TIDAK diisi di sini: dokumen ini memang tidak menyimpannya. Pelengkapannya ada
// di usecase, yang memang sudah membaca master bisnis.
func businessesOf(doc document) []daftarobjekdokumen.Business {
	result := make([]daftarobjekdokumen.Business, 0)

	raw, exists := doc[keyListLBUID]
	if !exists {
		return result
	}

	var list []documentLBU
	if err := json.Unmarshal(raw, &list); err != nil {
		// Senarai yang bentuknya tidak terduga diperlakukan sebagai kosong, dengan alasan
		// yang sama seperti dokumen yang rusak: barisnya tetap dapat dibuka.
		return result
	}

	for _, b := range list {
		if clean := strings.TrimSpace(b.ID); clean != "" {
			result = append(result, daftarobjekdokumen.Business{ID: clean})
		}
	}
	return result
}

// Insert menerbitkan ID baru lalu menyimpan barisnya beserta pemetaan bisnisnya.
//
// Seluruh langkahnya berada dalam SATU transaksi, mengikuti `D-68` yang memindahkan
// kepemilikan transaksi ke Go. Procedure lama menjalankan ROLLBACK-nya sendiri di dalam
// handler galat setelah sebagian pekerjaan dilakukan; di sini kegagalan di tengah tidak
// meninggalkan apa pun.
func (r *Repo) Insert(ctx context.Context, data daftarobjekdokumen.SaveData) (daftarobjekdokumen.DocumentObject, error) {
	saved, err := r.inTransaction(ctx, func(tx *sql.Tx) (daftarobjekdokumen.DocumentObject, error) {
		id, err := issueID(ctx, tx)
		if err != nil {
			return daftarobjekdokumen.DocumentObject{}, err
		}

		raw, err := buildDocument(document{}, id, data)
		if err != nil {
			return daftarobjekdokumen.DocumentObject{}, err
		}

		if _, err := tx.ExecContext(ctx, getQuery("document_object_insert"),
			id, data.Description, raw); err != nil {
			return daftarobjekdokumen.DocumentObject{}, fmt.Errorf(
				"daftarobjekdokumen/sqlstore: menyisipkan %q: %w", id, err)
		}

		return daftarobjekdokumen.DocumentObject{ID: id, Description: data.Description}, nil
	})
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang benar-benar tersimpan.
	return r.Get(ctx, saved.ID)
}

// Update menyimpan perubahan pada baris yang sudah ada beserta pemetaan bisnisnya.
//
// Dokumen JSON-nya disusun ulang dari dokumen LAMA, bukan dari nol: bagian yang tidak
// dikenal modul ini ikut terbawa. Menulis dokumen baru dari nol akan membuang apa pun yang
// pernah ditaruh sistem lama di sana — diam-diam, dan tanpa cara memulihkannya.
func (r *Repo) Update(ctx context.Context, id string, data daftarobjekdokumen.SaveData) (daftarobjekdokumen.DocumentObject, error) {
	_, previous, err := r.read(ctx, id)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}

	_, err = r.inTransaction(ctx, func(tx *sql.Tx) (daftarobjekdokumen.DocumentObject, error) {
		raw, err := buildDocument(previous, id, data)
		if err != nil {
			return daftarobjekdokumen.DocumentObject{}, err
		}

		result, err := tx.ExecContext(ctx, getQuery("document_object_update"),
			data.Description, raw, id)
		if err != nil {
			return daftarobjekdokumen.DocumentObject{}, fmt.Errorf(
				"daftarobjekdokumen/sqlstore: memperbarui %q: %w", id, err)
		}

		// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak
		// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
		// atas perubahan yang tidak pernah terjadi.
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return daftarobjekdokumen.DocumentObject{}, daftarobjekdokumen.ErrNotFound
		}
		return daftarobjekdokumen.DocumentObject{}, nil
	})
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	return r.Get(ctx, id)
}

// buildDocument menyusun dokumen JSON yang disimpan.
//
// ID selalu ikut di dalam dokumen, meniru bentuk yang ditulis sistem lama — di sana ID
// disisipkan dengan mengganti penanda 'UnknownID' setelah nomornya terbit.
func buildDocument(previous document, id string, data daftarobjekdokumen.SaveData) (string, error) {
	fresh := document{}
	for key, value := range previous {
		fresh[key] = value
	}

	list := make([]documentLBU, 0, len(data.Businesses))
	for _, b := range data.Businesses {
		if clean := strings.TrimSpace(b.ID); clean != "" {
			list = append(list, documentLBU{ID: clean})
		}
	}

	for key, value := range map[string]any{
		keyID:          id,
		keyDescription: data.Description,
		keyListLBUID:   list,
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("daftarobjekdokumen/sqlstore: menyusun %s pada dokumen %q: %w", key, id, err)
		}
		fresh[key] = encoded
	}

	raw, err := json.Marshal(fresh)
	if err != nil {
		return "", fmt.Errorf("daftarobjekdokumen/sqlstore: menyusun dokumen JSON %q: %w", id, err)
	}
	return string(raw), nil
}

// CheckTable memastikan kedua objek yang dipakai modul ini ada dan dapat dibaca akun
// aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("document_object_check_table"))
	if err != nil {
		return fmt.Errorf("daftarobjekdokumen/sqlstore: POOLDATA.LST_DOC_OBJ tidak dapat dibaca: %w", err)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return fmt.Errorf("daftarobjekdokumen/sqlstore: POOLDATA.LST_DOC_OBJ tidak dapat dibaca: %w", err)
	}
	return nil
}

// inTransaction menjalankan satu satuan kerja di dalam transaksi.
//
// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun.
// Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan
// kunci baris sampai koneksinya didaur ulang.
func (r *Repo) inTransaction(
	ctx context.Context,
	work func(tx *sql.Tx) (daftarobjekdokumen.DocumentObject, error),
) (daftarobjekdokumen.DocumentObject, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := work(tx)
	if err != nil {
		return daftarobjekdokumen.DocumentObject{}, err
	}
	if err := tx.Commit(); err != nil {
		return daftarobjekdokumen.DocumentObject{}, fmt.Errorf("daftarobjekdokumen/sqlstore: menutup transaksi: %w", err)
	}
	return result, nil
}

// issueID membentuk ID persis seperti `POOLDATA.PEGA_LST_DOC_OBJ`: kode situs disambung
// nomor urut lima digit dari POOLDATA.LST_DOC_OBJ_SEQ.
//
// Perangkaian dan pemformatannya dikerjakan di Go, bukan di SQL — LPAD dan TO_CHAR termasuk
// yang dilarang `09-DATABASE-STRATEGY.md` §4 karena keduanya mengikat kueri pada dialek
// Oracle.
func issueID(ctx context.Context, tx *sql.Tx) (string, error) {
	var site string
	if err := tx.QueryRowContext(ctx, getQuery("document_object_site")).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Tanpa baris situs, ID tidak dapat dibentuk sama sekali. Procedure lama
			// menjawab keadaan ini dengan kalimat di ErrMsg lalu berhenti seolah tidak
			// terjadi apa-apa; di sini ia menjadi galat yang benar-benar galat.
			return "", errors.New("daftarobjekdokumen/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
		}
		return "", fmt.Errorf("daftarobjekdokumen/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("document_object_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("daftarobjekdokumen/sqlstore: mengambil nomor urut: %w", err)
	}

	return strings.TrimSpace(site) + FiveDigits(sequence), nil
}

// FiveDigits meniru lpad(to_char(seq), 5, '0') pada procedure lama.
//
// # Batas yang nyata, bukan teoretis
//
// Bilangan di atas 99999 dikembalikan apa adanya, sama seperti LPAD Oracle — dan ID
// ke-100000 karena itu menjadi TUJUH karakter. Kolom ID berlebar CHAR(6), sehingga
// penyisipan berikutnya akan DITOLAK basis data (ORA-12899), bukan diterima dengan ID aneh.
//
// Urutannya ada di 778 hari ini, jadi batas itu masih sangat jauh. Perilakunya tetap
// dibiarkan apa adanya: memotongnya menjadi lima digit akan menghasilkan ID GANDA, yang
// jauh lebih buruk daripada penyisipan yang gagal dengan pesan jelas.
//
// Diekspor supaya perilaku ini dapat diuji.
func FiveDigits(n int64) string {
	digits := strconv.FormatInt(n, 10)
	for len(digits) < sequenceDigits {
		digits = "0" + digits
	}
	return digits
}

// BusinessRepo membaca POOLDATA.BUSINESS milik GISFW.
//
// Tipe terpisah, bukan method tambahan pada Repo, supaya "modul ini hanya membaca bisnis"
// terbaca dari bentuknya — dan supaya hak akses yang dibutuhkan keduanya dapat diminta
// terpisah ke DBA.
type BusinessRepo struct {
	db *sql.DB
}

// NewBusinessRepo membentuk repo master bisnis.
func NewBusinessRepo(db *sql.DB) *BusinessRepo { return &BusinessRepo{db: db} }

// List membaca seluruh bisnis, terurut menurut namanya.
func (r *BusinessRepo) List(ctx context.Context) ([]daftarobjekdokumen.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("business_list"))
	if err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca daftar bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]daftarobjekdokumen.Business, 0)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: membaca baris bisnis: %w", err)
		}
		result = append(result, daftarobjekdokumen.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftarobjekdokumen/sqlstore: menelusuri daftar bisnis: %w", err)
	}
	return result, nil
}

// CheckTable memastikan POOLDATA.BUSINESS dapat dibaca akun aplikasi.
func (r *BusinessRepo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("business_check_table"))
	if err != nil {
		return fmt.Errorf("daftarobjekdokumen/sqlstore: POOLDATA.BUSINESS tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

var (
	_ daftarobjekdokumen.Repo         = (*Repo)(nil)
	_ daftarobjekdokumen.BusinessRepo = (*BusinessRepo)(nil)
)
