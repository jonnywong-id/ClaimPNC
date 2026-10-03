package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxrclpucl"
)

// Repo membaca antrean RCL/PUCL dari SATU basis data entitas.
//
// # Satu operasi menulis, dan hanya satu
//
// ReturnToAnalyst menandai klaim selesai dikerjakan PUCL pada `POOLDATA.TC_PNC_PUCL` — tabel
// milik APLIKASI INI. Seluruh method lain hanya membaca.
//
// Tabel `DATAPEGA` tidak pernah disentuh untuk menulis, dan itu bukan kebetulan: selama masa
// paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`), dan tabel engine Pega milik
// Pega.
//
// Catatan yang pernah berdiri di sini — bahwa `TC_PNC_PUCL` pun tidak ditulis karena diisi
// PROSES PENGISI — berlaku sampai 2026-10-01 dan kini dicabut untuk TIGA kolom:
// `PUCL_APPROVE`, `STATUS_CLAIM`, dan `TGL_CETAK_DOKUMEN_PUCL`. Ketiganya penanda
// PERPINDAHAN, bukan data yang dimuat pengisi, dan ketiganya ditulis `PUCLPost` di Pega —
// lihat kueri `return_to_analyst` untuk nomor langkahnya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// plan menyebut kueri mana yang melayani sebuah permintaan dan bagaimana argumennya
// disusun.
//
// Argumen paginasi disusun di sini pula, bukan ditambahkan pemanggil, supaya urutan bind
// setiap kueri hidup di satu tempat bersama namanya. Di modul ini bedanya nyata: dua kueri
// memakai enam bind, satu memakai tujuh.
type plan struct {
	// name adalah nama kueri di berkas .sql.
	name string

	// args menyusun argumen bind sesuai urutan `:1`, `:2`, … di kueri itu.
	args func(p inboxrclpucl.Pagination) []any
}

// planFor memilih kueri yang melayani sebuah permintaan.
//
// # Kenapa pemilihannya fungsi, bukan peta dari kode tab
//
// Karena yang dipilih bukan hanya NAMA kuerinya melainkan juga susunan bind-nya, dan
// keduanya harus berpindah bersama. Peta dari kode tab ke nama kueri akan menyimpan
// separuhnya di satu tempat dan separuh lagi di tempat lain.
//
// # Kenapa nilai penyaringnya diikat, bukan ditulis di dalam SQL
//
// Ketiga nilai — akun antrean bersama, status kerja yang dikecualikan, dan penanda
// persetujuan — adalah NILAI BISNIS, dan `D-15` melarangnya tertanam di dalam kode. Di sini
// alasannya lebih tajam daripada kerapian: satu nilai yang salah mengosongkan seluruh layar
// tanpa satu pun galat, dan tidak ada apa pun di antarmuka yang menandakannya.
//
// # Kenapa kelas objek kerja TIDAK lagi diikat
//
// `POOLDATA.TC_PNC_PUCL` tidak punya `PXOBJCLASS`. Pemisahan klaim dari berkas penerimaan
// dokumen — yang di tabel Pega ditegakkan penyaring itu — berpindah menjadi syarat PROSES
// PENGISI, yang hanya boleh memuat baris `ASM-FW-GCNMFW-Work-PNC`.
//
// `inboxrclpucl.WorkClassClaim` karena itu tidak lagi muncul di sini. Ia TETAP dipakai
// DailyReport, yang masih membaca tabel Pega — lihat catatan pada kueri `daily_report`.
func planFor(q inboxrclpucl.Query) (plan, error) {
	switch q.Tab.Code {
	case inboxrclpucl.TabCetakSurat:
		return plan{
			name: "list_cetak_surat",
			args: func(p inboxrclpucl.Pagination) []any {
				return []any{
					inboxrclpucl.WorkStatusCompleted,
					inboxrclpucl.ExpiryStatusActive,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	case inboxrclpucl.TabKelengkapanDokumen:
		return plan{
			name: "list_kelengkapan_dokumen",
			args: func(p inboxrclpucl.Pagination) []any {
				return []any{
					inboxrclpucl.WorkStatusCompleted,
					inboxrclpucl.PUCLReturnedToAnalyst,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	case inboxrclpucl.TabKlaimMSIG:
		return plan{
			name: "list_klaim_msig",
			args: func(p inboxrclpucl.Pagination) []any {
				// Satu bind LEBIH BANYAK daripada kueri di atasnya, dan itulah satu-satunya
				// perbedaannya: penanda jalur MSIG. Urutannya disisipkan SEBELUM paginasi,
				// mengikuti urutan `:n` di berkas .sql.
				return []any{
					inboxrclpucl.WorkStatusCompleted,
					inboxrclpucl.PUCLReturnedToAnalyst,
					inboxrclpucl.MSIGMarker,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	default:
		// Tab yang tidak dikenal seharusnya sudah ditolak NewQuery. Kalau ia sampai ke
		// sini, yang salah adalah kode — bukan permintaan pengguna — dan galatnya menyebut
		// kodenya alih-alih mengembalikan nol baris yang terbaca seperti antrean kosong.
		return plan{}, fmt.Errorf(
			"inboxrclpucl/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}
}

// List mengambil satu halaman baris beserta jumlah seluruh baris yang cocok.
//
// Paginasi dipotong BASIS DATA, bukan di aplikasi — lihat catatan paginasi di kepala
// inboxrclpucl.sql. Jumlah seluruhnya datang dari kolom TOTAL_ROWS pada baris mana pun; ia
// sama di seluruh baris karena dihitung `COUNT(*) OVER ()`.
func (r *Repo) List(
	ctx context.Context,
	q inboxrclpucl.Query,
	page inboxrclpucl.Pagination,
) (inboxrclpucl.Page, error) {
	selected, err := planFor(q)
	if err != nil {
		return inboxrclpucl.Page{}, err
	}

	clean := page.Normalize()
	result := inboxrclpucl.Page{
		Items:      []inboxrclpucl.WorkItem{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query(selected.name), selected.args(clean)...)
	if err != nil {
		return inboxrclpucl.Page{},
			fmt.Errorf("menjalankan kueri %s: %w", selected.name, err)
	}
	defer rows.Close()

	for rows.Next() {
		item, total, err := scanWorkItem(rows)
		if err != nil {
			return inboxrclpucl.Page{},
				fmt.Errorf("membaca baris kueri %s: %w", selected.name, err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxrclpucl.Page{},
			fmt.Errorf("menelusuri hasil kueri %s: %w", selected.name, err)
	}

	// Halaman kosong menyisakan Total nol, dan itu BENAR untuk halaman pertama yang memang
	// tidak punya baris. Ia TIDAK benar untuk halaman kelima dari antrean berisi tiga
	// baris — tetapi keadaan itu hanya tercapai lewat parameter yang diketik sendiri, dan
	// layar tidak pernah memintanya. Menambah satu kueri penghitung hanya untuk itu berarti
	// satu perjalanan tambahan pada setiap permintaan yang normal.

	return result, nil
}

// DailyReport mengambil satu halaman LAPORAN HARIAN RCL/PUCL.
//
// Ia terpisah dari List karena kuerinya memang berbeda — bukan hanya penyaringnya melainkan
// kolomnya, gabungannya, dan jumlah tabel yang dibacanya. Lihat catatan pada
// inboxrclpucl.DailyReportRow dan pada kueri `daily_report`.
//
// Rentang tanggal DIIKAT DUA KALI karena kueri ber-`UNION` dan kedua cabangnya menyaring
// rentang yang sama. Menulis penanda bind yang sama dua kali akan bergantung pada cara
// driver menafsirkan penanda berulang — perbedaan yang tidak terlihat saat membaca kueri,
// dan yang akibatnya adalah rentang tanggal yang salah pada salah satu cabang.
func (r *Repo) DailyReport(
	ctx context.Context,
	rng inboxrclpucl.DateRange,
	page inboxrclpucl.Pagination,
) ([]inboxrclpucl.DailyReportRow, int, error) {
	clean := page.Normalize()
	rows := []inboxrclpucl.DailyReportRow{}
	total := 0

	cursor, err := r.db.QueryContext(ctx, query("daily_report"),
		// Cabang antrean bersama.
		inboxrclpucl.WorkClassClaim,
		inboxrclpucl.RCLPUCLWorkbasket,
		rng.From,
		rng.To,
		// Cabang Personal Accident — TANPA gabungan antrean bersama, mengikuti kueri lama.
		inboxrclpucl.WorkClassClaim,
		inboxrclpucl.GroupPanelPA,
		rng.From,
		rng.To,
		// Paginasi.
		clean.Offset(),
		clean.Size,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("menjalankan kueri daily_report: %w", err)
	}
	defer cursor.Close()

	for cursor.Next() {
		row, rowTotal, err := scanReportRow(cursor)
		if err != nil {
			return nil, 0, fmt.Errorf("membaca baris kueri daily_report: %w", err)
		}
		rows = append(rows, row)
		total = rowTotal
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("menelusuri hasil kueri daily_report: %w", err)
	}

	return rows, total, nil
}

// Detail mengambil isi layar kerja RCL/PUCL untuk satu klaim.
//
// # Kuncinya kini NOMOR CASE, bukan kunci teknis Pega
//
// `POOLDATA.TC_PNC_PUCL` dikunci `CLAIMID`, yang berisi nomor case (`PNC-1865`) — bukan
// `PZINSKEY` berbentuk `ASM-FW-GCNMFW-WORK PNC-1865`. Nilai yang sampai ke sini adalah
// `WorkItem.Reference`, yang dipasok ketiga kueri daftar dari kolom yang sama persis,
// sehingga keduanya tidak dapat menyimpang.
//
// Penyaring kelas objek kerja hilang bersama kolomnya; perlindungan yang dulu diberikannya
// berpindah ke proses pengisi. Lihat catatan pada planFor.
//
// # Kenapa "tidak ditemukan" dibedakan dari "kosong"
//
// Klaim yang tidak ada dan klaim yang seluruh isiannya kosong terlihat SAMA di layar, dan
// hanya yang pertama yang merupakan kekeliruan. Dua hal yang paling mungkin menyebabkannya:
// kunci yang benar dibuka pada PORTAL YANG SALAH (`R-20`), dan klaim yang belum disalin
// proses pengisi ke tabel datar. Keduanya keterangan yang harus sampai ke pengguna, bukan
// layar kosong tanpa sebab.
func (r *Repo) Detail(
	ctx context.Context,
	reference string,
) (inboxrclpucl.ClaimDetail, error) {
	row := r.db.QueryRowContext(ctx, query("detail"), reference)

	detail, err := scanDetail(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxrclpucl.ClaimDetail{}, inboxrclpucl.ErrClaimNotFound
	}
	if err != nil {
		return inboxrclpucl.ClaimDetail{}, fmt.Errorf("membaca layar kerja klaim: %w", err)
	}
	return detail, nil
}

// Documents mengembalikan dokumen yang terlampir pada satu klaim.
func (r *Repo) Documents(
	ctx context.Context,
	caseNumber string,
) ([]inboxrclpucl.Document, error) {
	rows, err := r.db.QueryContext(ctx, query("documents"), caseNumber)
	if err != nil {
		return nil, fmt.Errorf("membaca daftar dokumen klaim: %w", err)
	}
	defer rows.Close()

	// Dikembalikan sebagai senarai KOSONG, bukan nil, supaya JSON-nya `[]` dan bukan `null`.
	// Layar membedakan "tidak ada dokumen" dari "daftarnya gagal dibaca", dan `null` membuat
	// keduanya terlihat sama.
	documents := []inboxrclpucl.Document{}
	for rows.Next() {
		var (
			id, name, mime         sql.NullString
			category, subCategory  sql.NullString
			uploadedAt, uploadedBy sql.NullString
			pegaVisible            sql.NullInt64
		)
		if err := rows.Scan(
			&id, &name, &mime, &category, &subCategory, &uploadedAt, &uploadedBy,
			&pegaVisible,
		); err != nil {
			return nil, fmt.Errorf("membaca baris dokumen: %w", err)
		}
		documents = append(documents, inboxrclpucl.Document{
			ID:          id.String,
			Name:        name.String,
			MimeType:    mime.String,
			Category:    category.String,
			SubCategory: subCategory.String,

			// Dibentuk dengan penggambar yang sama seperti seluruh tanggal modul ini —
			// `INPUTDATE` pun `TIMESTAMP(6)`. Membiarkannya mentah di sini akan membuat satu
			// layar menggambar dua bentuk tanggal berdampingan.
			UploadedAt: inboxrclpucl.DisplayTimeText(uploadedAt.String),
			UploadedBy: uploadedBy.String,

			PegaVisible: pegaVisible.Int64 == 1,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menutup daftar dokumen: %w", err)
	}
	return documents, nil
}

// DocumentContent mengembalikan isi satu dokumen.
func (r *Repo) DocumentContent(
	ctx context.Context,
	caseNumber, documentID string,
) (inboxrclpucl.DocumentContent, error) {
	var (
		name, mime sql.NullString
		content    []byte
	)
	err := r.db.QueryRowContext(ctx, query("document_content"), documentID, caseNumber).
		Scan(&name, &mime, &content)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxrclpucl.DocumentContent{}, inboxrclpucl.ErrDocumentNotFound
	}
	if err != nil {
		return inboxrclpucl.DocumentContent{}, fmt.Errorf("membaca isi dokumen: %w", err)
	}
	return inboxrclpucl.DocumentContent{
		Name:     name.String,
		MimeType: mime.String,
		Content:  content,
	}, nil
}

// ReturnToAnalyst menandai klaim sudah selesai dikerjakan PUCL.
//
// Yang ditulisnya HANYA `POOLDATA.TC_PNC_PUCL` — tabel milik aplikasi ini.
//
// # Ia pernah memindahkan penugasan di Pega, dan itu MERUSAK satu klaim
//
// Pada 2026-10-02 fungsi ini sempat menyisipkan penugasan `Send To Analis` ke
// `DATAPEGA.PC_ASSIGN_WORKLIST` lalu membuang penugasan workbasket RCL/PUCL — meniru alur
// Pega. Akibatnya klaim `PNC-2067` tidak dapat dibuka lagi di Pega:
//
//	Unable to open an instance using the given inputs:
//	ASSIGN-WORKBASKET ASM-FW-GCNMFW-WORK PNC-2067!REGISTER_FLOW
//
// Sebabnya baris yang kami tulis tidak memuat `PZPVSTREAM`. Kolom itu nullable di katalog,
// dan dari situ disimpulkan ia boleh dikosongkan — padahal dari **105.616** baris penugasan
// Pega, **nol** yang berbentuk begitu. Nullable berarti basis data mengizinkannya, bukan
// berarti Pega menerimanya.
//
// Klaimnya sudah dipulihkan, dan seluruh kode penulisnya DIHAPUS — bukan sekadar dimatikan.
// Kode mati yang menulis tabel Pega adalah jebakan bagi sesi berikutnya. Analisis alurnya
// tersimpan di `keputusan-implementasi.md` §161–§163, dan penjaganya ada di
// TestNoQueryWritesToPegaTables.
//
// Menghidupkannya kembali menuntut satu hal yang belum ada: cara membentuk `PZPVSTREAM` yang
// Pega terima. Perpindahan tahap yang sebenarnya menunggu `ActionClaimPUCL` dari Tim Pega.
//
// `caller` TIDAK ditulis ke tabel: `TC_PNC_PUCL` tidak punya kolom pelaku, dan menambah kolom
// menempuh `D-63` (permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA). Pelakunya
// tetap tercatat — di jejak log usecase, bersama nomor klaim dan portalnya.
func (r *Repo) ReturnToAnalyst(ctx context.Context, reference, caller string) error {
	// Urutan argumen mengikuti URUTAN KEMUNCULAN penanda di pernyataannya — nilai lebih dulu
	// (`SET`), baru nomor klaim (`WHERE`). Lihat catatan panjang pada kueri `return_to_analyst`:
	// menukarnya menghasilkan kegagalan yang SENYAP, bukan galat.
	res, err := r.db.ExecContext(ctx, query("return_to_analyst"),
		inboxrclpucl.PUCLReturnedToAnalyst,
		inboxrclpucl.StatusClaimAnalyst,
		strings.TrimSpace(reference),
	)
	if err != nil {
		return fmt.Errorf("menandai klaim %s selesai di PUCL: %w", reference, err)
	}

	// Jumlah baris DIPERIKSA, dan nol dinyatakan sebagai kegagalan.
	//
	// Bentuk sebelumnya mengabaikannya dengan alasan yang terdengar masuk akal — "nol berarti
	// sudah ditandai, dan itu keberhasilan". Alasan itu KELIRU dan berbiaya: ia membuat
	// pernyataan yang tidak mengenai satu baris pun tetap dilaporkan berhasil, sehingga cacat
	// urutan bind di atas hidup sampai Work Owner menemukannya dari layar.
	//
	// Pernyataannya kini tidak lagi menyaring nilai saat ini, sehingga nol baris berarti SATU
	// hal: klaimnya tidak ada. Itu memang ErrClaimNotFound.
	terpengaruh, err := res.RowsAffected()
	if err != nil {
		// Penggerak yang tidak dapat melaporkan jumlah baris bukan alasan menyatakan gagal —
		// pernyataannya sudah dijalankan tanpa galat. Yang hilang hanya kemampuan menilainya.
		return nil
	}
	if terpengaruh == 0 {
		return inboxrclpucl.ErrClaimNotFound
	}

	// Penugasan di Pega TIDAK disentuh — lihat catatan di kepala fungsi ini.
	return nil
}

// SaveReceipt menyimpan kedua isian Penerimaan Dokumen yang diketik petugas.
//
// Ia TIDAK menyentuh `PUCL_APPROVE`: "Save" di layar lama tidak memanggil `PUCLPost` sama
// sekali, sehingga menyimpan bukan memindahkan — klaimnya tetap menjadi pekerjaan PUCL.
//
// `caller` tidak ditulis ke tabel; `TC_PNC_PUCL` tidak punya kolom pelaku. Pelakunya tercatat
// di jejak log usecase, bersama nomor klaim dan portalnya.
func (r *Repo) SaveReceipt(
	ctx context.Context,
	reference string,
	in inboxrclpucl.ReceiptInput,
	caller string,
) error {
	_ = caller

	at, err := in.Validate()
	if err != nil {
		return err
	}

	res, err := r.db.ExecContext(ctx, query("save_receipt"),
		strings.TrimSpace(in.Note),
		at,
		strings.TrimSpace(reference),
	)
	if err != nil {
		return fmt.Errorf("menyimpan isian penerimaan dokumen klaim %s: %w", reference, err)
	}

	// Nol baris berarti klaimnya tidak ada — lihat alasan lengkapnya pada ReturnToAnalyst.
	terpengaruh, err := res.RowsAffected()
	if err != nil {
		return nil
	}
	if terpengaruh == 0 {
		return inboxrclpucl.ErrClaimNotFound
	}
	return nil
}

// MarkLetterPrinted menandai surat RCL/PUCL sudah diterbitkan.
//
// Ia TIDAK menyentuh `PUCL_APPROVE`: "Download Dokumen" mengirim `Status` kosong, sehingga
// kedua langkah yang menulisnya di `PUCLPost` terlewati. Klaimnya tetap pekerjaan PUCL — yang
// berpindah hanyalah TAB-nya.
func (r *Repo) MarkLetterPrinted(ctx context.Context, reference, caller string) error {
	_ = caller

	res, err := r.db.ExecContext(ctx, query("mark_letter_printed"),
		inboxrclpucl.StatusCasePrinted,
		inboxrclpucl.StatusClaimWaitingDocument,
		strings.TrimSpace(reference),
	)
	if err != nil {
		return fmt.Errorf("menandai surat klaim %s sudah dicetak: %w", reference, err)
	}

	terpengaruh, err := res.RowsAffected()
	if err != nil {
		return nil
	}
	if terpengaruh == 0 {
		return inboxrclpucl.ErrClaimNotFound
	}
	return nil
}

// AddDocument melampirkan satu berkas ke klaim — tombol "Unggah Dokumen".
//
// # Satu transaksi, tiga pernyataan
//
// Nomor urut diambil, baris pencacah disisipkan, lalu lampirannya. Ketiganya dibungkus satu
// transaksi supaya kegagalan di tengah tidak meninggalkan pencacah yang bertambah tanpa
// lampiran — tepat jenis selisih yang `D-68` lepaskan dari procedure ber-sembilan-`COMMIT`.
//
// # Kenapa kunci objek kerja dibaca lebih dulu
//
// `DATA_ATTACHFILE.IDPEGA` menyimpan `PZINSKEY` objek kerja, dan kueri `documents`
// menggabungkannya kembali lewat kolom itu. Baris yang `IDPEGA`-nya salah tersimpan dengan
// baik dan TIDAK PERNAH muncul di daftar dokumen klaimnya — kegagalan yang tidak menghasilkan
// satu pun galat.
func (r *Repo) AddDocument(
	ctx context.Context,
	reference string,
	upload inboxrclpucl.UploadedDocument,
	caller string,
) (inboxrclpucl.Document, error) {
	if err := upload.Validate(); err != nil {
		return inboxrclpucl.Document{}, err
	}

	key := strings.TrimSpace(reference)

	var workKey string
	err := r.db.QueryRowContext(ctx, query("work_object_key"), key).Scan(&workKey)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxrclpucl.Document{}, inboxrclpucl.ErrClaimNotFound
	}
	if err != nil {
		return inboxrclpucl.Document{}, fmt.Errorf(
			"membaca kunci objek kerja klaim %s: %w", reference, err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxrclpucl.Document{}, fmt.Errorf("memulai transaksi unggahan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var year string
	var runNo int64
	if err := tx.QueryRowContext(ctx, query("next_attachment_number")).
		Scan(&year, &runNo); err != nil {
		return inboxrclpucl.Document{}, fmt.Errorf("mengambil nomor lampiran: %w", err)
	}

	// Bentuk `DATAID` ditiru PERSIS: dua digit tahun diikuti nomor urut berlebar sepuluh
	// dengan nol di depan — `year || lpad(runno,10,'0')` pada procedure aslinya.
	dataID := fmt.Sprintf("%s%010d", year, runNo)

	// Kunci baris pencacah. Procedure aslinya memakai `new_uuid` dari basis data; di sini
	// dibentuk di Go, karena nilainya tidak pernah dibaca lagi — ia hanya membedakan satu
	// baris pencacah dari yang lain.
	counterKey := fmt.Sprintf("%s-%d", dataID, time.Now().UnixNano())

	if _, err := tx.ExecContext(ctx, query("insert_attachment_counter"),
		counterKey, year, runNo); err != nil {
		return inboxrclpucl.Document{}, fmt.Errorf("menulis pencacah lampiran: %w", err)
	}

	// Kategori disimpan apa adanya sebagai NAMA, bukan diterjemahkan menjadi kode. Itulah
	// bentuk yang dipakai Pega pada `PC_LINK_ATTACHMENT.PYCATEGORY`, dan kolom ini pun sudah
	// memuat nilai berupa teks pada baris yang ditulis jalur lain.
	//
	// `nil`, bukan string kosong, ketika petugas tidak memilih: kolom yang KOSONG dan kolom
	// yang BERISI teks nol-panjang tidak dapat dibedakan lagi sesudah tersimpan.
	// Kategori KOSONG disimpan sebagai `File`, bukan dibiarkan kosong.
	//
	// `File` adalah kategori lampiran bawaan Pega, dan dialognya pun menggambarnya terpilih
	// pada baris yang belum disentuh. Membiarkannya kosong punya akibat yang tidak terduga
	// sejak daftar dokumen menyaring menurut `GCNMGetAllAttachments`: baris tanpa kategori
	// tidak cocok dengan satu pun nama kategori lampiran, sehingga berkas yang baru saja
	// diunggah petugas LANGSUNG HILANG dari daftarnya.
	kategori := strings.TrimSpace(upload.Category)
	if kategori == "" {
		kategori = inboxrclpucl.DefaultAttachmentCategory
	}

	if _, err := tx.ExecContext(ctx, query("insert_attachment"),
		dataID,
		strings.TrimSpace(caller),
		strings.TrimSpace(upload.Name),
		strings.TrimSpace(upload.Note),
		strings.TrimSpace(upload.MimeType),
		upload.Content,
		workKey,
		kategori,
	); err != nil {
		return inboxrclpucl.Document{}, fmt.Errorf(
			"menyimpan lampiran klaim %s: %w", reference, err)
	}

	if err := tx.Commit(); err != nil {
		return inboxrclpucl.Document{}, fmt.Errorf("menutup transaksi unggahan: %w", err)
	}

	return inboxrclpucl.Document{
		ID:         dataID,
		Name:       strings.TrimSpace(upload.Name),
		MimeType:   strings.TrimSpace(upload.MimeType),
		UploadedBy: strings.TrimSpace(caller),
	}, nil
}

// DocumentCategories mengembalikan pilihan kolom "Category" pada dialog unggah.
func (r *Repo) DocumentCategories(
	ctx context.Context,
) ([]inboxrclpucl.DocumentCategory, error) {
	rows, err := r.db.QueryContext(ctx, query("document_categories"))
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri document_categories: %w", err)
	}
	defer rows.Close()

	// Senarai KOSONG, bukan nil: ia diserahkan apa adanya ke JSON, dan nil tergambar `null`
	// sementara layar mengharapkan larik.
	result := []inboxrclpucl.DocumentCategory{}
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("membaca baris document_categories: %w", err)
		}
		bersih := strings.TrimSpace(name.String)
		result = append(result, inboxrclpucl.DocumentCategory{
			Value: bersih,
			Label: bersih,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil document_categories: %w", err)
	}
	return result, nil
}

// CheckTable memastikan tabel DAN kolom yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca, keberadaan tabelnya, dan keberadaan kelima kolom penyaringnya.
//
// Pemeriksaan kolom terpisah dari pemeriksaan tabel dengan sengaja — lihat catatan pada
// kueri `check_columns`. Kolom yang tidak ada dan kolom yang ada tetapi kosong menghasilkan
// layar yang sama-sama kosong, dan hanya yang pertama yang merupakan kerusakan.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_rclpucl")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.TC_PNC_PUCL — tabel datar RCL/PUCL; bila ia belum dibuat, "+
				"jalankan Database/CREATE_TABLE_3.SQL lebih dulu: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca kolom penyaring POOLDATA.TC_PNC_PUCL (TGL_CETAK_DOKUMEN_PUCL, "+
				"STATUS_CASE, PUCL_APPROVE, MSIG, TGL_KIRIM_PUCL): %w", err)
	}

	// Ketiga tabel anak diperiksa TERPISAH: tanpa ketiganya, layar kerja tetap terbuka
	// tetapi "Nama Peserta", "UP", dan "Jumlah Tagihan" diam-diam kosong — dan kosong
	// adalah keadaan yang sah bagi klaim tanpa objek, sehingga tidak dapat dibedakan dari
	// kerusakan.
	//
	// `T_CLAIM_PNC` ikut karena ia JEMBATAN dari nomor case ke kunci teknis; tanpanya kedua
	// isian turunan tidak dapat dicapai sama sekali.
	if err := r.db.QueryRowContext(ctx, query("check_detail")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_PNC, POOLDATA.T_CLAIM_OBJECTLIST, atau "+
				"POOLDATA.T_CLAIM_ADJUSTMENT: %w", err)
	}

	// Kedua tabel Pega diperiksa PALING AKHIR dan terpisah, karena kegagalannya paling
	// SEMPIT akibatnya: sejak ketiga tab pindah ke tabel datar, keduanya hanya dipakai
	// laporan harian. Yang gagal karenanya hanyalah tombol unduh tab "Cetak Surat" — bukan
	// layarnya.
	if err := r.db.QueryRowContext(ctx, query("check_laporan")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK atau "+
				"DATAPEGA.PC_ASSIGN_WORKBASKET — keduanya hanya dipakai laporan harian "+
				"tab Cetak Surat: %w", err)
	}
	return nil
}

// EmptyDiagnosis menjelaskan mengapa ketiga tab kosong, dengan angka.
//
// Setiap isian menghitung baris yang LOLOS satu penyaring, berdiri sendiri. Penyaring yang
// menghasilkan NOL sementara Total tidak nol adalah yang mengosongkan layar.
type EmptyDiagnosis struct {
	Total           int
	NotCompleted    int
	WithoutLetter   int
	CaseStatusMatch int
	WithLetter      int
	StillWithPUCL   int
	NotMSIG         int
	MSIG            int
}

// DiagnoseEmpty menghitung berapa baris yang lolos tiap penyaring, satu per satu.
//
// Dipanggil `-periksa` HANYA saat ketiga tab kosong. Ketiga tab yang kosong terbaca persis
// sama dengan antrean yang memang sepi, dan kedua keadaan itu menuntut tindakan yang
// berbeda — yang satu menunggu pekerjaan, yang satu menunggu perbaikan proses pengisi.
func (r *Repo) DiagnoseEmpty(ctx context.Context) (EmptyDiagnosis, error) {
	var d EmptyDiagnosis

	err := r.db.QueryRowContext(ctx, query("diagnose_empty"),
		inboxrclpucl.WorkStatusCompleted,
		inboxrclpucl.ExpiryStatusActive,
		inboxrclpucl.PUCLReturnedToAnalyst,
		inboxrclpucl.MSIGMarker,
	).Scan(
		&d.Total, &d.NotCompleted, &d.WithoutLetter, &d.CaseStatusMatch,
		&d.WithLetter, &d.StillWithPUCL, &d.NotMSIG, &d.MSIG,
	)
	if err != nil {
		return EmptyDiagnosis{}, fmt.Errorf("menjalankan kueri diagnose_empty: %w", err)
	}
	return d, nil
}

// scanner adalah bentuk minimal yang dibutuhkan pemindai, sehingga keduanya dapat diuji
// tanpa basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanWorkItem memindai satu baris grid menjadi WorkItem beserta jumlah seluruh baris.
//
// Urutannya WAJIB sama dengan listColumns dan dengan urutan kolom di inboxrclpucl.sql.
// Ketiganya dijaga query_test.go.
//
// # Kenapa SELURUH kolom dipindai lewat tipe yang mengizinkan NULL
//
// Bukan kehati-hatian berlebih. `LETTER_PRINTED_AT` memang SELALU NULL pada tab Cetak Surat
// — penyaringnya `IS NULL`. `TRACK_CODE` kosong pada klaim yang jalurnya belum ditetapkan.
// Dan kelima kolom PUCL lain hanya punya 2–86 nilai berbeda di produksi, yang berarti
// sebagian besar barisnya kosong.
//
// # Kenapa waktu ikut dipindai sebagai teks, dan DI MANA ia dibentuk
//
// Bentuk yang dikembalikan driver bergantung pada tipe kolomnya. Memindainya sebagai
// `sql.NullString` membuat nilainya sampai ke sini apa adanya alih-alih gagal dipindai pada
// baris pertama di produksi — dan itu tetap berlaku sesudah pindah ke tabel datar, yang
// kolom waktunya pun `TIMESTAMP(6)`.
//
// Pemformatannya dikerjakan `inboxrclpucl.DisplayTimeText`, di sini — bukan di layar, dan
// bukan dengan `TO_CHAR` di dalam SQL.
//
//   - Bukan di layar, karena berkas ekspor CSV mengambil nilai domain LANGSUNG tanpa
//     melewati DTO (`http/export.go`). Memformatnya di layar akan membuat berkas dan tabel
//     menggambar isian yang sama dengan dua bentuk yang berbeda.
//   - Bukan `TO_CHAR`, karena pemformatan tampilan dilarang di SQL
//     (`09-DATABASE-STRATEGY.md` §4) dan tidak portabel ke PostgreSQL.
//   - Di sini dan bukan di lapisan atas, mengikuti preseden `TrackOf` beberapa baris di
//     bawah: penyimpanan SQL dan penyimpanan memori WAJIB menghasilkan teks yang sama
//     persis, dan itu hanya terjamin bila keduanya memakai penggambar yang sama.
//
// Terverifikasi terhadap Oracle 2026-09-30 pada tabel Pega: `TANGGALKIRIMPUCL_1`,
// `LAMAKLAIM_1`, `TANGGALCETAKDOKUMENPUCL_1`, dan `PXCREATEDATETIME` seluruhnya
// `TIMESTAMP(6)`, dan driver mengembalikannya sebagai teks ISO ber-offset
// (`2025-06-13T14:41:01.532+07:00`) — bentuk yang tidak pernah muncul di layar Pega.
// Keempat padanannya di `TC_PNC_PUCL` — `TGL_KIRIM_PUCL`, `LAMA_KLAIM`,
// `TGL_CETAK_DOKUMEN_PUCL`, `TGL_CREATE_PUCL` — bertipe sama persis.
func scanWorkItem(row scanner) (inboxrclpucl.WorkItem, int, error) {
	var (
		reference, caseID, policyNumber sql.NullString
		insuredName, inboxEntryAt       sql.NullString
		analystNote, trackCode          sql.NullString
		letterPrintedAt, claimAge       sql.NullString
		expiryStatus, createdAt         sql.NullString
		total                           sql.NullInt64
	)

	err := row.Scan(
		&reference, &caseID, &policyNumber, &insuredName, &inboxEntryAt,
		&analystNote, &trackCode, &letterPrintedAt, &claimAge, &expiryStatus,
		&createdAt, &total,
	)
	if err != nil {
		return inboxrclpucl.WorkItem{}, 0, err
	}

	return inboxrclpucl.WorkItem{
		Reference:    reference.String,
		CaseID:       caseID.String,
		PolicyNumber: policyNumber.String,
		InsuredName:  insuredName.String,
		InboxEntryAt: inboxrclpucl.DisplayTimeText(inboxEntryAt.String),
		AnalystNote:  analystNote.String,

		// Jalur DITERJEMAHKAN di sini, bukan di dalam kueri.
		//
		// Penerjemahannya milik domain (`TrackOf`), sehingga penyimpanan SQL dan
		// penyimpanan memori menghasilkan teks yang sama persis. Menuliskannya sebagai
		// `CASE` di dalam SQL akan membuat kedua pengisi seam punya dua penerjemah yang
		// dapat menyimpang tanpa ketahuan.
		//
		// Ketiga kodenya punya teks — RCL, PUCL, Notification; yang di luar ketiganya
		// menghasilkan teks kosong.
		Track: inboxrclpucl.TrackOf(trackCode.String),

		LetterPrintedAt: inboxrclpucl.DisplayTimeText(letterPrintedAt.String),

		// "Lama Klaim" digambar sebagai TANGGAL, karena isinya memang tanggal.
		//
		// Judulnya menyebut durasi dan tetap dibawa apa adanya (`D-13`); yang dibentuk di
		// sini isinya, bukan judulnya. Terverifikasi: kolomnya `TIMESTAMP(6)`, dan Work
		// Owner menjelaskan 2026-09-30 bahwa isinya tanggal kirim untuk proses PUCL.
		ClaimAge: inboxrclpucl.DisplayTimeText(claimAge.String),

		ExpiryStatus: expiryStatus.String,
		CreatedAt:    inboxrclpucl.DisplayTimeText(createdAt.String),
	}, int(total.Int64), nil
}

// scanDetail memindai satu baris layar kerja.
//
// Urutannya WAJIB sama dengan detailColumns dan dengan urutan kolom kueri `detail`.
//
// Kedua isian TURUNAN dipindai sebagai teks yang mengizinkan NULL, dan keduanya memang
// sering kosong: klaim tanpa objek, atau objek tanpa adjustment, menghasilkan subkueri yang
// tidak mengembalikan baris. Itu keadaan yang sah — bukan kegagalan.
func scanDetail(row scanner) (inboxrclpucl.ClaimDetail, error) {
	var (
		reference, insuredParty          sql.NullString
		claimNumber, trackCode           sql.NullString
		analystNote, policyNumber        sql.NullString
		lossDate, puclNote               sql.NullString
		subject, openingNote             sql.NullString
		bodyNote, closingNote            sql.NullString
		msigFlag, groupPanel             sql.NullString
		documentCompleteAt, insuredEmail sql.NullString
		idObject, idCoverage, idAdj      sql.NullString
		receivedDateFirst, receivedNote  sql.NullString
		firstObjectName, firstPropose    sql.NullString
	)

	err := row.Scan(
		&reference, &insuredParty,
		&claimNumber, &trackCode, &analystNote, &policyNumber,
		&lossDate, &puclNote,
		&subject, &openingNote, &bodyNote, &closingNote,
		&msigFlag, &groupPanel,
		&documentCompleteAt,
		&idObject, &idCoverage, &idAdj,
		&insuredEmail,
		&receivedDateFirst, &receivedNote,
		&firstObjectName, &firstPropose,
	)
	if err != nil {
		return inboxrclpucl.ClaimDetail{}, err
	}

	return inboxrclpucl.ClaimDetail{
		Reference:   reference.String,
		ClaimNumber: claimNumber.String,

		// Keduanya tidak digambar sebagai isian; keduanya memilih TOMBOL. Lihat
		// ClaimDetail.Buttons.
		MSIG:       msigFlag.String,
		GroupPanel: groupPanel.String,

		// Dibaca dari kolomnya sejak 2026-10-01; nilai penampung `"1"` dicabut.
		ActionParameters: inboxrclpucl.ActionParameters{
			IDObject:     idObject.String,
			IDCoverage:   idCoverage.String,
			IDAdjustment: idAdj.String,
		},

		Letter: inboxrclpucl.LetterDraft{
			Track:        inboxrclpucl.TrackOf(trackCode.String),
			TrackCode:    trackCode.String,
			AnalystNote:  analystNote.String,
			PolicyNumber: policyNumber.String,

			// `DATE_OF_LOSS` pun `TIMESTAMP(6)`, sehingga ia dibentuk dengan penggambar yang
			// sama. Membiarkannya mentah di sini sementara keempat isian tanggal grid
			// dibentuk akan membuat satu layar menggambar dua bentuk tanggal berdampingan.
			LossDate: inboxrclpucl.DisplayTimeText(lossDate.String),

			// Kedua isian diisi dari SATU sumber, dan itu memang benar.
			//
			// `SetDataLampiranSuratRCLPUCL_Act` menetapkan `.UP` dan `.NamaPeserta` dari
			// ekspresi yang sama persis, dan Work Owner menegaskan 2026-09-24 bahwa kolom
			// "UP" pada surat RCL/PUCL memang berisi NAMA OBJEK — bukan nilai
			// pertanggungan.
			//
			// Ditulis berdampingan dengan sengaja: keduanya terbaca sebagai salin-tempel
			// yang keliru, dan pernah "diperbaiki" atas dasar itu. Lihat
			// `inboxrclpucl.LetterDraft.SumInsured`.
			InsuredName: firstObjectName.String,
			SumInsured:  firstObjectName.String,

			BillAmount: firstPropose.String,

			// Keempatnya dibaca APA ADANYA dari tabel datar — tidak diturunkan, tidak
			// diterjemahkan. Lihat catatan pada `inboxrclpucl.LetterDraft.Subject`.
			Subject:     subject.String,
			OpeningNote: openingNote.String,
			BodyNote:    bodyNote.String,
			ClosingNote: closingNote.String,

			// Penerima surat. Hanya dipakai saat mencetak, dan sengaja TIDAK digambar di
			// layar kerja: section-nya tidak memuat isian itu (`D-13`).
			InsuredParty: strings.TrimSpace(insuredParty.String),
		},

		DocumentReceipt: inboxrclpucl.DocumentReceipt{
			PUCLNote: puclNote.String,

			// Keduanya sebelumnya digambar bertanda "di clipboard Pega". Kolomnya ternyata
			// ADA — `TGL_TERIMA_DOKUMEN_PUCL` di tabel datar, `EMAIL_LOD` di `T_CLAIM_PNC`.
			CompleteAt:   inboxrclpucl.DisplayTimeText(documentCompleteAt.String),
			InsuredEmail: insuredEmail.String,

			// Baris pertama grid "Tanggal Terima Dokumen". Baris kosong TIDAK dibentuk:
			// grid tanpa baris berarti daftarnya memang kosong, dan satu baris kosong
			// terbaca sebagai entri yang ada tetapi tak terisi.
			ReceivedDates: inboxrclpucl.ReceivedDatesOf(
				inboxrclpucl.DisplayTimeText(receivedDateFirst.String),
				receivedNote.String,
			),
		},
	}, nil
}

// scanReportRow memindai satu baris laporan harian beserta jumlah seluruh baris.
//
// Urutannya WAJIB sama dengan reportColumns dan dengan urutan kolom kueri `daily_report`.
// Ia TERPISAH dari scanWorkItem karena kolomnya memang berbeda — menyatukan keduanya akan
// menuntut satu pemindai yang separuh kolomnya selalu kosong, dan itu menyembunyikan
// perbedaan yang justru harus terlihat.
func scanReportRow(row scanner) (inboxrclpucl.DailyReportRow, int, error) {
	var (
		reference, caseID, policyNumber sql.NullString
		insuredName, sentAt             sql.NullString
		analystNote, letterPrintedAt    sql.NullString
		trackCode, claimStatus          sql.NullString
		total                           sql.NullInt64
	)

	err := row.Scan(
		&reference, &caseID, &policyNumber, &insuredName, &sentAt,
		&analystNote, &letterPrintedAt, &trackCode, &claimStatus,
		&total,
	)
	if err != nil {
		return inboxrclpucl.DailyReportRow{}, 0, err
	}

	return inboxrclpucl.DailyReportRow{
		Reference:       reference.String,
		CaseID:          caseID.String,
		PolicyNumber:    policyNumber.String,
		InsuredName:     insuredName.String,
		SentAt:          inboxrclpucl.DisplayTimeText(sentAt.String),
		AnalystNote:     analystNote.String,
		LetterPrintedAt: inboxrclpucl.DisplayTimeText(letterPrintedAt.String),

		// Kueri lama menuliskan kode mentah `1`/`2` ke dalam berkas. Di sini ia
		// diterjemahkan supaya berkas dan layar menyebut hal yang sama dengan kata yang
		// sama — selisih terencana, dinyatakan lewat PlannedDifferences.
		Track: inboxrclpucl.TrackOf(trackCode.String),

		ClaimStatus: claimStatus.String,
	}, int(total.Int64), nil
}

// RecordHistory menulis satu baris riwayat klaim.
//
// Ia membaca `PZINSKEY` lebih dulu karena kolom `CASEID` menyimpan kunci objek kerja, bukan
// nomor klaim — lihat catatan pada kueri `insert_history`.
func (r *Repo) RecordHistory(ctx context.Context, reference, statusNote, caller string) error {
	note := strings.TrimRight(statusNote, "\x00")
	if strings.TrimSpace(note) == "" {
		// Tindakan yang memang tidak menulis riwayat — "Tolak Klaim" dan "Save". Bukan galat.
		return nil
	}

	key := strings.TrimSpace(reference)

	var workKey string
	err := r.db.QueryRowContext(ctx, query("work_object_key"), key).Scan(&workKey)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxrclpucl.ErrClaimNotFound
	}
	if err != nil {
		return fmt.Errorf("membaca kunci objek kerja klaim %s: %w", reference, err)
	}

	// Spasi di ujung teks riwayat TIDAK dipangkas — ia ada di Pega dan ikut tersimpan.
	if _, err := r.db.ExecContext(ctx, query("insert_history"),
		workKey, note, strings.TrimSpace(caller),
	); err != nil {
		return fmt.Errorf("menulis riwayat klaim %s: %w", reference, err)
	}
	return nil
}

// MoveToSendToAnalyst memindahkan klaim ke tahap Send To Analis.
//
// Ketiga pernyataannya berada dalam SATU transaksi, dan itu bukan kehati-hatian berlebihan:
// tugas lama yang tertutup tanpa tugas baru terbuka membuat klaim hilang dari setiap inbox —
// tidak lagi pekerjaan PUCL, dan belum menjadi pekerjaan siapa pun. Itu kegagalan yang jauh
// lebih buruk daripada tombol yang menolak.
func (r *Repo) MoveToSendToAnalyst(ctx context.Context, reference, caller string) error {
	key := strings.TrimSpace(reference)

	var workKey, pic string
	var picNull sql.NullString
	err := r.db.QueryRowContext(ctx, query("technical_pic"), key).Scan(&workKey, &picNull)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxrclpucl.ErrClaimNotFound
	}
	if err != nil {
		return fmt.Errorf("membaca PIC Teknik klaim %s: %w", reference, err)
	}
	pic = strings.TrimSpace(picNull.String)
	if pic == "" {
		// Tugas Worklist WAJIB bertuan sejak lahir (`D-26`). Tugas tanpa pemilik pada
		// antrean yang bukan antrean bersama tidak akan muncul di inbox siapa pun — klaim
		// hilang tanpa galat. Menolak di sini membuat sebabnya terbaca.
		return inboxrclpucl.ErrTechnicalPICUnknown
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("memulai transaksi perpindahan tahap: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()

	// Nol baris BUKAN galat — klaim yang dimulai di Pega belum pernah punya tugas di sini.
	if _, err := tx.ExecContext(ctx, query("close_open_tasks"),
		now, inboxrclpucl.TicketSendToAnalyst, key,
	); err != nil {
		return fmt.Errorf("menutup tugas terbuka klaim %s: %w", reference, err)
	}

	if _, err := tx.ExecContext(ctx, query("open_task"),
		newTaskID(), workKey, key,
		inboxrclpucl.StageSendToAnalyst, inboxrclpucl.QueueWorklist, pic,
		now, now,
	); err != nil {
		return fmt.Errorf("membuka tugas Send To Analis klaim %s: %w", reference, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("menyimpan perpindahan tahap klaim %s: %w", reference, err)
	}
	return nil
}

// newTaskID membangkitkan pengenal tugas 128 bit dalam heksadesimal.
//
// Bentuknya sama dengan pembangkit pengenal modul lain, supaya baris yang ditulis modul ini
// tidak dapat dibedakan dari baris yang ditulis modul alur — keduanya mengisi tabel yang sama.
//
// Acak, bukan berurut: pengenal tugas tidak punya makna bisnis, dan nomor berurut akan
// membocorkan berapa banyak tugas yang sudah dibuat.
func newTaskID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand tidak gagal pada sistem yang sehat. Melanjutkan dengan pengenal yang
		// dapat ditebak lebih berbahaya daripada berhenti.
		panic("inboxrclpucl/sqlstore: sumber acak tidak tersedia: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
