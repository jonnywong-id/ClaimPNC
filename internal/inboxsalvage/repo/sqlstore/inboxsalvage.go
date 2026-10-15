package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/inboxsalvage"
)

// Repo membaca dan menulis data salvage pada SATU basis data entitas.
//
// # Ia MENULIS, berbeda dari sqlstore modul inbox lain
//
// Dua tabel — `POOLDATA.PNC_SALVAGE` dan `POOLDATA.DETAIL_PNC_SALVAGE` — dimiliki modul
// ini selama masa paralel, sehingga `P-1` terpenuhi. Seluruh tabel lain hanya dibaca.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk penyimpanan di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// PegaWorkKeyPrefix adalah awalan kunci objek kerja Pega pada kolom `T_CLAIM_PNC.CLAIMID`.
//
// Nama kelas internal Pega tertanam di dalam kunci data bisnis — utang teknis §4.1 yang
// `D-22` dan `D-71` hapus untuk klaim baru. Ia diikat sebagai nilai, bukan ditulis di dalam
// SQL, supaya terbaca sebagai sesuatu yang kelak berubah.
//
// Perhatikan SPASI di ujungnya: ia bagian dari nilainya, dan menghilangkannya membuat
// gabungan tidak pernah cocok — daftar kosong tanpa satu pun galat.
const PegaWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

// List mengembalikan satu halaman baris yang cocok beserta jumlah seluruhnya.
func (r *Repo) List(
	ctx context.Context,
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	if q.Tab.Family == inboxsalvage.FamilySalvage {
		return r.listSalvage(ctx, q, page)
	}
	return r.listClaim(ctx, q, page)
}

// listClaim melayani keluarga A dan B.
func (r *Repo) listClaim(
	ctx context.Context,
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	clean := page.Normalize()
	searching := q.Search != ""

	name, args := claimPlan(q, clean, searching)

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return inboxsalvage.Page{}, fmt.Errorf("inboxsalvage/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	result := inboxsalvage.Page{Pagination: clean, Items: []inboxsalvage.Row{}}
	for rows.Next() {
		var (
			claimNo, pic, business, fourth sql.NullString
			total                          int
		)
		if err := rows.Scan(&claimNo, &pic, &business, &fourth, &total); err != nil {
			return inboxsalvage.Page{}, fmt.Errorf(
				"inboxsalvage/sqlstore: %s: memindai baris: %w", name, err)
		}

		row := inboxsalvage.Row{
			Reference:    claimNo.String,
			ClaimNo:      claimNo.String,
			PIC:          pic.String,
			BusinessName: business.String,
		}

		// Kolom keempat berarti hal yang BERBEDA menurut keluarganya, dan itu satu-satunya
		// tempat kedua kueri berbeda bentuknya.
		if q.Tab.Family == inboxsalvage.FamilyClaim {
			row.LossDate = fourth.String
		} else {
			row.ObjectName = fourth.String
		}

		result.Items = append(result.Items, row)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxsalvage.Page{}, fmt.Errorf(
			"inboxsalvage/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return result, nil
}

// claimPlan memilih kueri keluarga A/B beserta susunan bind-nya.
//
// # Kenapa nama kueri dan susunan bind dipilih BERSAMAAN
//
// Karena keduanya harus berpindah bersama. Memisahkannya — peta kode tab ke nama kueri di
// satu tempat, susunan bind di tempat lain — membuat penambahan satu bind pada satu kueri
// dapat lupa diikuti di tempat yang lain, dan akibatnya adalah galat bind yang menyebut
// nomor, bukan menyebut tab mana yang rusak.
func claimPlan(
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
	searching bool,
) (string, []any) {
	pattern := "%" + escapeLike(q.Search) + "%"

	switch {
	case q.Tab.BuybackFilter && searching:
		return "list_claim_buyback_search", []any{pattern, page.Offset(), page.Size}

	case q.Tab.BuybackFilter:
		return "list_claim_buyback", []any{page.Offset(), page.Size}

	case q.Tab.OutstandingSalvage && searching:
		return "list_claim_search", []any{
			inboxsalvage.ClosedWorkStatuses[0],
			inboxsalvage.ClosedWorkStatuses[1],
			pattern, page.Offset(), page.Size,
		}

	case q.Tab.OutstandingSalvage:
		return "list_claim", []any{
			inboxsalvage.ClosedWorkStatuses[0],
			inboxsalvage.ClosedWorkStatuses[1],
			page.Offset(), page.Size,
		}

	case searching:
		return "list_claim_object_search", []any{
			q.Tab.SalvageStatus, pattern, page.Offset(), page.Size,
		}

	default:
		return "list_claim_object", []any{
			q.Tab.SalvageStatus, page.Offset(), page.Size,
		}
	}
}

// listSalvage melayani keluarga C — ketujuh daftar, satu kueri.
func (r *Repo) listSalvage(
	ctx context.Context,
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	clean := page.Normalize()

	// Penyaring status transfer. Tab Histori Salvage tidak menyaringnya sama sekali.
	transfer := patternAll
	if q.Tab.TransferStatus != "" {
		transfer = escapeLike(q.Tab.TransferStatus)
	}

	// Penyaring pemilik. Hanya tab Request Balai Lelang memakainya.
	owner := patternAll
	if q.Tab.OwnedByCaller {
		owner = escapeLike(q.Caller.Login)
	}

	claimPattern, picPattern := searchPatterns(q)

	args := []any{
		PegaWorkKeyPrefix,
		transfer,
		owner,
		claimPattern,
		picPattern,
		clean.Offset(),
		clean.Size,
	}

	rows, err := r.db.QueryContext(ctx, query("list_salvage"), args...)
	if err != nil {
		return inboxsalvage.Page{}, fmt.Errorf(
			"inboxsalvage/sqlstore: list_salvage: %w", err)
	}
	defer rows.Close()

	today, err := r.today(ctx)
	if err != nil {
		return inboxsalvage.Page{}, err
	}

	result := inboxsalvage.Page{Pagination: clean, Items: []inboxsalvage.Row{}}
	for rows.Next() {
		var (
			salvageID, claimNo, inputDate, pic, salvageType sql.NullString
			location, quantity, estimate, email, remark     sql.NullString
			acceptanceNo, transferStatus, acceptedValue     sql.NullString
			requestValue, requestNote                       sql.NullString
			total                                           int
		)
		if err := rows.Scan(
			&salvageID, &claimNo, &inputDate, &pic, &salvageType,
			&location, &quantity, &estimate, &email, &remark,
			&acceptanceNo, &transferStatus, &acceptedValue,
			&requestValue, &requestNote,
			&total,
		); err != nil {
			return inboxsalvage.Page{}, fmt.Errorf(
				"inboxsalvage/sqlstore: list_salvage: memindai baris: %w", err)
		}

		input := dateOnly(inputDate.String)

		result.Items = append(result.Items, inboxsalvage.Row{
			Reference:       salvageID.String,
			ClaimNo:         claimNo.String,
			SalvageID:       salvageID.String,
			InputDate:       input,
			PIC:             pic.String,
			SalvageType:     salvageType.String,
			SalvageLocation: location.String,
			AuctionStatus:   inboxsalvage.AuctionStatusOf(acceptedValue.String),
			Quantity:        quantity.String,
			EstimateValue:   estimate.String,
			Email:           email.String,
			Remark:          remark.String,
			AcceptanceNo:    acceptanceNo.String,
			TransferStatus:  transferStatus.String,
			RequestValue:    requestValue.String,
			RequestNote:     requestNote.String,
			SubmissionType:  inboxsalvage.SubmissionTypeOf(requestNote.String),
			Aging:           inboxsalvage.AgingOf(input, today),

			// Kolom "Catatan" SELALU kosong — lihat inboxsalvage.Row.Note.
			Note: "",
		})
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxsalvage.Page{}, fmt.Errorf(
			"inboxsalvage/sqlstore: list_salvage: membaca hasil: %w", err)
	}

	return result, nil
}

// searchPatterns menyusun kedua pola pencarian keluarga C.
//
// Keduanya digabung dengan ATAU di dalam kueri, sehingga tab yang hanya mencari nomor klaim
// harus MEMATIKAN cabang PIC — bukan membiarkannya `%`, yang akan mencocokkan segalanya dan
// membuat pencarian tidak menyaring apa pun.
func searchPatterns(q inboxsalvage.Query) (string, string) {
	if q.Search == "" {
		return patternAll, patternNever
	}

	escaped := escapeLike(q.Search)

	if q.Tab.SearchExact {
		// Tanpa `%` di kedua ujung, `LIKE` berperilaku sama dengan `=` — dan itulah yang
		// dipakai kueri lama pada ketiga tab ini.
		if q.Tab.SearchByPIC {
			return escaped, escaped
		}
		return escaped, patternNever
	}

	pattern := "%" + escaped + "%"
	if q.Tab.SearchByPIC {
		return pattern, pattern
	}
	return pattern, patternNever
}

// Detail mengembalikan isi panel "Detail Salvage" untuk satu pengajuan.
//
// DUA kueri, bukan satu dengan gabungan: kepala dan daftar barangnya punya kardinalitas
// yang berbeda, dan menggabungkannya akan mengulang seluruh isian kepala pada setiap baris
// barang. Keduanya tetap satu perjalanan pulang-pergi masing-masing — bukan satu per baris
// seperti pada jalur checker di sistem lama.
func (r *Repo) Detail(ctx context.Context, salvageID string) (inboxsalvage.Detail, error) {
	clean := strings.TrimSpace(salvageID)
	if clean == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}

	detail, err := r.detailHeader(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	items, err := r.detailItems(ctx, detail.ClaimNo, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	detail.Items = items

	history, err := r.historyOfClaim(ctx, detail.ClaimNo)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	detail.History = history

	// Pilihan objek dan coverage dibaca DI SINI PULA, bukan hanya pada jalur nomor klaim.
	//
	// Sebabnya satu daftar membuka FORM, bukan panel baca: "Rejected Checker" berisi
	// pengajuan yang dikembalikan checker kepada PIC untuk diperbaiki, dan memperbaikinya
	// berarti kedua autocomplete itu harus terisi. Barisnya membawa ID PENGAJUAN, sehingga
	// jalur inilah yang dilaluinya.
	objects, coverages, err := r.claimChoices(ctx, detail.ClaimNo)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	detail.ObjectChoices = objects
	detail.CoverageChoices = coverages

	return detail, nil
}

// historyOfClaim membaca grid "Detail History Salvage" satu klaim.
func (r *Repo) historyOfClaim(
	ctx context.Context,
	claimNo string,
) ([]inboxsalvage.HistoryRow, error) {
	rows, err := r.db.QueryContext(ctx, query("salvage_history_of_claim"), claimNo)
	if err != nil {
		return nil, fmt.Errorf("membaca riwayat salvage klaim: %w", err)
	}
	defer rows.Close()

	history := []inboxsalvage.HistoryRow{}
	for rows.Next() {
		var (
			inputDate, no, pic, minimum sql.NullString
			transferStatus, hasAccNo    sql.NullString
			salvageID                   sql.NullString
		)

		if err := rows.Scan(&inputDate, &no, &pic, &minimum,
			&transferStatus, &hasAccNo, &salvageID); err != nil {
			return nil, fmt.Errorf("memindai riwayat salvage: %w", err)
		}

		history = append(history, inboxsalvage.HistoryRow{
			SalvageID:    strings.TrimSpace(salvageID.String),
			InputDate:    dateOnly(inputDate.String),
			ClaimNo:      strings.TrimSpace(no.String),
			PIC:          strings.TrimSpace(pic.String),
			MinimumValue: strings.TrimSpace(minimum.String),
			Position: inboxsalvage.HistoryPositionOf(
				transferStatus.String, flagIsSet(hasAccNo.String)),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca riwayat salvage: %w", err)
	}

	return history, nil
}

// DetailByClaim membaca rincian lewat nomor klaim.
//
// Tiga langkah, dan urutannya menentukan pesan yang dilihat pengguna:
//
//  1. Klaimnya dibaca. Tidak ada -> ErrRowNotFound; nomor klaim yang salah memang
//     kekeliruan, dan pada aplikasi ini penyebab tersering bukan salah ketik melainkan
//     klaim milik entitas lain (`R-20`).
//  2. Pengajuan TERAKHIR miliknya dicari. Tidak ada -> panel tetap dikembalikan, dengan
//     isian klaim terisi dan HasSubmission salah. Itu keadaan yang sah, bukan kegagalan:
//     daftar Salvage Outstanding justru berisi klaim yang salvage-nya belum diajukan.
//  3. Kepala panel dan barangnya dibaca dengan ID itu.
func (r *Repo) DetailByClaim(
	ctx context.Context,
	claimNo string,
) (inboxsalvage.Detail, error) {
	clean := strings.TrimSpace(claimNo)
	if clean == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}

	base, err := r.claimHeader(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	// Riwayat dibaca SEBELUM pengajuan terakhirnya, karena ia dibutuhkan pada KEDUA
	// cabang di bawah — termasuk cabang klaim yang belum punya pengajuan sama sekali,
	// tempat form "Menambahkan Data Salvage" menggambarnya.
	history, err := r.historyOfClaim(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	base.History = history

	// Pilihan objek dan coverage dibaca pada KEDUA cabang pula, dan justru cabang "belum
	// punya pengajuan" yang paling membutuhkannya: itulah cabang yang membuka form
	// "Menambahkan Data Salvage", tempat kedua autocomplete-nya digambar.
	//
	// Sebelum ini form itu tidak dapat disimpan sama sekali pada jalur tersebut — kedua
	// kolomnya terkunci dan kosong, sementara keduanya wajib diisi (lihat form.Validate).
	objects, coverages, err := r.claimChoices(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	base.ObjectChoices = objects
	base.CoverageChoices = coverages

	salvageID, err := r.latestSalvageOfClaim(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	if salvageID == "" {
		return base, nil
	}

	detail, err := r.detailHeader(ctx, salvageID)
	if err != nil {
		// Pengajuan yang tercatat di DETAIL_PNC_SALVAGE tetapi TIDAK ada di PNC_SALVAGE
		// adalah data yang tidak sinkron, bukan kekeliruan pengguna. Panel tetap
		// menampilkan klaimnya, dengan keterangan bahwa pengajuannya tidak terbaca —
		// alih-alih menolak membuka seluruh panel.
		if errors.Is(err, inboxsalvage.ErrRowNotFound) {
			return base, nil
		}
		return inboxsalvage.Detail{}, err
	}

	items, err := r.detailItems(ctx, detail.ClaimNo, salvageID)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	// Isian klaim dipertahankan: kepala panel pengajuan tidak memuat PIC maupun tanggal
	// kejadian, dan keduanya digambar pada panel yang dibuka dari daftar berbasis klaim.
	detail.PIC = base.PIC
	detail.LossDate = base.LossDate
	if detail.BusinessName == "" {
		detail.BusinessName = base.BusinessName
	}
	detail.Items = items
	detail.History = base.History
	detail.ObjectChoices = base.ObjectChoices
	detail.CoverageChoices = base.CoverageChoices

	return detail, nil
}

// claimChoices membaca isi kedua autocomplete form "Menambahkan Data Salvage".
//
// # Kenapa dua kueri, bukan satu gabungan
//
// Karena hubungannya satu-ke-banyak: satu objek punya beberapa coverage. Menggabungkannya
// akan mengulang baris objek sebanyak coverage-nya, lalu memaksa kode di sini membuang
// duplikatnya — pekerjaan yang tidak dibutuhkan untuk dua daftar yang masing-masing berisi
// segelintir baris.
//
// # Klaim tanpa objek BUKAN galat
//
// Keduanya mengembalikan senarai KOSONG, bukan nil dan bukan galat. Kedua kolomnya
// menerima ketikan bebas — `pyAllowFreeFormInput=true` di layar lama — sehingga form tetap
// dapat diisi, hanya tanpa bantuan daftar.
func (r *Repo) claimChoices(
	ctx context.Context,
	claimNo string,
) ([]inboxsalvage.ObjectChoice, []inboxsalvage.CoverageChoice, error) {
	objects, err := r.claimObjects(ctx, claimNo)
	if err != nil {
		return nil, nil, err
	}

	coverages, err := r.claimCoverages(ctx, claimNo)
	if err != nil {
		return nil, nil, err
	}

	return objects, coverages, nil
}

// AttachDocument menyimpan satu berkas lampiran beserta penautnya ke pengajuan salvage.
//
// # Keenam langkahnya SATU transaksi
//
// Di sistem lama tiap langkah menutup transaksinya sendiri — kedua procedure lampiran
// ber-`COMMIT` di dalam, dan `InsertToSalvageDoc_SQL` menambah satu lagi. Akibatnya
// kegagalan di langkah keempat meninggalkan isi berkas tanpa keterangan, dan tidak ada
// yang memulihkannya.
//
// Di sini keenamnya dibungkus satu transaksi (`D-68`). `COMMIT` di dalam teks kueri lama
// karena itu tidak dibawa — yang dibawa hanya pemanggilan procedure-nya.
//
// # Urutannya mengikuti `SaveFilePenunjangBySalvage`
//
//  1. `GenerateimageID`     — ID gambar, penaut isi dengan keterangannya
//  2. `GetNamaFile`         — awalan nama berkas
//  3. `CountSalvage`        — urutan berkas ke berapa pada klaim ini
//  4. `SaveAttachmentToDBTemp_Sql` — ISI berkas
//  5. `SaveAttachmentToDB_Sql`     — KETERANGAN berkas
//  6. `GetDocumentData` + `InsertToSalvageDoc_SQL` — penaut ke pengajuan salvage
//
// Langkah histori disisipkan di antara 5 dan 6, mengikuti `PNCSaveAttachmentToDB` yang
// menjalankan keduanya berurutan.
func (r *Repo) AttachDocument(
	ctx context.Context,
	doc inboxsalvage.DocumentUpload,
) (inboxsalvage.AttachedDocument, error) {
	clean := doc.Clean()
	if err := clean.Validate(); err != nil {
		return inboxsalvage.AttachedDocument{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: memulai transaksi lampiran: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// IMAGEID dibangkitkan di Go, bukan di basis data — lihat inboxsalvage.NewImageID.
	imageID := inboxsalvage.NewImageID(time.Now())

	storedName, err := r.storedNameOf(ctx, tx, clean)
	if err != nil {
		return inboxsalvage.AttachedDocument{}, err
	}

	pegaKey := PegaWorkKeyPrefix + clean.ClaimNo

	// Isi berkas lebih dulu, keterangannya kemudian — urutan `SaveFilePenunjangBySalvage`.
	//
	// DATAID yang dikembalikan penyimpan ISI sengaja DIBUANG: ia bukan yang dipakai
	// penaut salvage. Yang dipakai adalah DATAID milik keterangannya, dibaca kembali
	// lewat IMAGEID di langkah terakhir.
	var contentID, contentErr sql.NullString
	_, err = tx.ExecContext(ctx, query("attachment_content"),
		clean.MimeType,                 // :1  PHOTOMIME
		storedName,                     // :2  PHOTONOTE  — kueri lama mengisinya nama berkas
		storedName,                     // :3  PHOTONAME
		inboxsalvage.DocumentCommand,   // :4 COMAND
		clean.Operator,                 // :5  INPUTOPERATOR
		imageID,                        // :6  PHOTOID
		nullIfEmpty(clean.Category),    // :7 PHOTOCATEGORY
		nullIfEmpty(clean.SubCategory), // :8 PHOTOSUBCATEGORY
		pegaKey,                        // :9  IDPEGA
		clean.Content64,                // :10 PHOTO
		sql.Out{Dest: &contentID},
		sql.Out{Dest: &contentErr},
	)
	if err != nil {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: menyimpan isi lampiran: %w", err)
	}
	if message := strings.TrimSpace(contentErr.String); message != "" {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: menyimpan isi lampiran ditolak: %s", message)
	}

	var metaID, metaErr sql.NullString
	_, err = tx.ExecContext(ctx, query("attachment_meta"),
		clean.MimeType,
		storedName,
		storedName,
		inboxsalvage.DocumentCommand,
		clean.Operator,
		imageID,
		nullIfEmpty(clean.Category),
		nullIfEmpty(clean.SubCategory),
		pegaKey,
		sql.Out{Dest: &metaID},
		sql.Out{Dest: &metaErr},
	)
	if err != nil {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: menyimpan keterangan lampiran: %w", err)
	}
	if message := strings.TrimSpace(metaErr.String); message != "" {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: menyimpan keterangan lampiran ditolak: %s", message)
	}

	// DATAID keterangan dibaca ULANG lewat IMAGEID, bukan diambil dari parameter keluar.
	//
	// Begitulah `InsertToSalvageDocument` melakukannya — ia menjalankan `GetDocumentData`
	// lebih dulu. Membaca ulang juga membuktikan barisnya benar-benar ada sebelum
	// penautnya ditulis.
	dataID, attachName, err := r.attachmentByImage(ctx, tx, imageID)
	if err != nil {
		return inboxsalvage.AttachedDocument{}, err
	}

	_, err = tx.ExecContext(ctx, query("attachment_history"),
		dataID,                         // :1 DATAID
		clean.ClaimNo,                  // :2 NOKLAIM
		clean.Operator,                 // :3 INPUTOPERATOR
		attachName,                     // :4 ATTACHNAME dan ATTACHNOTE
		clean.MimeType,                 // :5 ATTACHMIMETYPE
		nullIfEmpty(metaID.String),     // :6 MESSAGE
		nullIfEmpty(clean.Category),    // :7 CATRGORY
		nullIfEmpty(clean.SubCategory), // :8 GCNMCATEGORY
		nullIfEmpty(clean.SubCategory), // :9 GCNMTYPE
	)
	if err != nil {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: menulis histori lampiran: %w", err)
	}

	result := inboxsalvage.AttachedDocument{
		DataID:     dataID,
		ImageID:    imageID,
		StoredName: storedName,
	}

	// Penaut salvage DILEWATI bila pengajuannya belum punya ID.
	//
	// Itu terjadi pada form pengajuan BARU: berkas diunggah sebelum Submit, sehingga
	// `IDSALVAGE` memang belum terbit. Lampirannya tetap menempel ke klaimnya lewat
	// `IDPEGA`, dan itu bukan kegagalan.
	if clean.SalvageID != "" {
		var linkResult sql.NullString
		_, err = tx.ExecContext(ctx, query("salvage_document_link"),
			dataID,
			clean.ClaimNo,
			clean.SalvageID,
			attachName,
			clean.Operator,
			nullIfEmpty(clean.Category),
			nullIfEmpty(clean.SubCategory),
			sql.Out{Dest: &linkResult},
		)
		if err != nil {
			return inboxsalvage.AttachedDocument{}, fmt.Errorf(
				"inboxsalvage/sqlstore: menautkan lampiran ke salvage: %w", err)
		}
		result.LinkedToSalvage = true
	}

	if err := tx.Commit(); err != nil {
		return inboxsalvage.AttachedDocument{}, fmt.Errorf(
			"inboxsalvage/sqlstore: menyimpan transaksi lampiran: %w", err)
	}

	return result, nil
}

// storedNameOf menyusun nama berkas yang BENAR-BENAR tersimpan.
//
// Bentuknya mengikuti `SetFileNameSalvage`:
//
//	<awalan>-<nomor klaim>-<urutan>.<ekstensi>
//
// # Bila awalannya tidak terbaca, nama ASLI yang dipakai
//
// Awalan datang dari `LST_TYPE_DOC_BUSINESS.NAMAFILE`, dicari dengan jenis dokumen — dan
// modal unggahan salvage TIDAK punya pemilih jenis dokumen sama sekali, sehingga
// pencariannya sering tidak menghasilkan apa-apa.
//
// Pada keadaan itu nama aslinya dipakai, dibersihkan dari karakter yang tidak aman.
// Alternatifnya — awalan kosong — menghasilkan nama berbentuk `-PNC-123-1.pdf` yang
// menyembunyikan asal berkasnya dari siapa pun yang membacanya di daftar lampiran.
func (r *Repo) storedNameOf(
	ctx context.Context,
	tx *sql.Tx,
	doc inboxsalvage.DocumentUpload,
) (string, error) {
	prefix := ""
	if doc.SubCategory != "" {
		var found sql.NullString
		row := tx.QueryRowContext(ctx, query("attachment_name_prefix"), doc.SubCategory)
		switch err := row.Scan(&found); {
		case errors.Is(err, sql.ErrNoRows):
			// Jenis dokumen tanpa baris master BUKAN galat — lihat catatan di atas.
		case err != nil:
			return "", fmt.Errorf("membaca awalan nama berkas: %w", err)
		default:
			prefix = strings.TrimSpace(found.String)
		}
	}

	if prefix == "" {
		prefix = inboxsalvage.SafeFileName(doc.FileName)
	}

	sequence, err := r.attachmentSequence(ctx, tx, doc)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s-%s-%d", prefix, doc.ClaimNo, sequence)
	if doc.MimeType != "" {
		name += "." + doc.MimeType
	}
	return name, nil
}

// attachmentSequence menghitung berkas ke berapa ini pada klaimnya.
//
// `SetFileNameSalvage` memulai dari `0` lalu menambah `1`, sehingga berkas pertama
// bernomor `1`.
func (r *Repo) attachmentSequence(
	ctx context.Context,
	tx *sql.Tx,
	doc inboxsalvage.DocumentUpload,
) (int, error) {
	var existing int
	row := tx.QueryRowContext(ctx, query("attachment_sequence_of_claim"),
		doc.ClaimNo, nullIfEmpty(doc.Category), nullIfEmpty(doc.SubCategory))
	if err := row.Scan(&existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 1, nil
		}
		return 0, fmt.Errorf("menghitung lampiran klaim %q: %w", doc.ClaimNo, err)
	}
	return existing + 1, nil
}

// attachmentByImage membaca keterangan lampiran lewat IMAGEID.
func (r *Repo) attachmentByImage(
	ctx context.Context,
	tx *sql.Tx,
	imageID string,
) (string, string, error) {
	var dataID, attachName sql.NullString

	row := tx.QueryRowContext(ctx, query("attachment_by_image"), imageID)
	if err := row.Scan(&dataID, &attachName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Keterangan baru saja disisipkan; tidak adanya berarti procedure menolak
			// tanpa mengisi pesan galatnya. Dinyatakan sebagai galat, bukan dilewati:
			// tanpa DATAID, penaut salvage akan menunjuk baris yang tidak ada.
			return "", "", fmt.Errorf(
				"inboxsalvage/sqlstore: keterangan lampiran %q tidak terbaca setelah disimpan",
				imageID)
		}
		return "", "", fmt.Errorf("membaca keterangan lampiran: %w", err)
	}

	return strings.TrimSpace(dataID.String), strings.TrimSpace(attachName.String), nil
}

// Currencies membaca pilihan dropdown "Mata Uang" dari `POOLDATA.CURRENCY`.
//
// Baris kembar DIBUANG: tabelnya menyimpan satu baris per NEGARA, sehingga mata uang yang
// dipakai beberapa negara muncul berulang kali. Yang dipilih pengguna adalah kodenya, dan
// kode yang sama tercantum dua kali hanya membingungkan.
func (r *Repo) Currencies(
	ctx context.Context,
) ([]inboxsalvage.CurrencyOption, error) {
	rows, err := r.db.QueryContext(ctx, query("currency_options"))
	if err != nil {
		return nil, fmt.Errorf("membaca POOLDATA.CURRENCY: %w", err)
	}
	defer rows.Close()

	options := []inboxsalvage.CurrencyOption{}
	seen := map[string]bool{}

	for rows.Next() {
		var code sql.NullString
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("memindai pilihan mata uang: %w", err)
		}

		clean := strings.ToUpper(strings.TrimSpace(code.String))
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true

		options = append(options, inboxsalvage.CurrencyOption{Code: clean, Label: clean})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca pilihan mata uang: %w", err)
	}

	return options, nil
}

// claimObjects membaca objek pertanggungan milik satu klaim.
func (r *Repo) claimObjects(
	ctx context.Context,
	claimNo string,
) ([]inboxsalvage.ObjectChoice, error) {
	rows, err := r.db.QueryContext(ctx, query("claim_objects"), PegaWorkKeyPrefix, claimNo)
	if err != nil {
		return nil, fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_OBJECTLIST untuk klaim %q: %w", claimNo, err)
	}
	defer rows.Close()

	choices := []inboxsalvage.ObjectChoice{}
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("memindai objek klaim %q: %w", claimNo, err)
		}

		choice := inboxsalvage.ObjectChoice{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		}

		// Baris tanpa nama DIBUANG. Yang dibaca pengguna hanyalah namanya; pilihan tanpa
		// nama tergambar sebagai baris kosong yang tidak dapat dibedakan satu sama lain,
		// dan memilihnya menyimpan nama kosong — yang justru ditolak validasi form.
		if choice.Name == "" {
			continue
		}

		choices = append(choices, choice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca objek klaim %q: %w", claimNo, err)
	}

	return choices, nil
}

// claimCoverages membaca coverage milik satu klaim.
func (r *Repo) claimCoverages(
	ctx context.Context,
	claimNo string,
) ([]inboxsalvage.CoverageChoice, error) {
	rows, err := r.db.QueryContext(ctx, query("claim_coverages"), PegaWorkKeyPrefix, claimNo)
	if err != nil {
		return nil, fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_OBJECTCOVERAGE untuk klaim %q: %w", claimNo, err)
	}
	defer rows.Close()

	choices := []inboxsalvage.CoverageChoice{}
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("memindai coverage klaim %q: %w", claimNo, err)
		}

		choice := inboxsalvage.CoverageChoice{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		}
		if choice.Name == "" {
			continue
		}

		choices = append(choices, choice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca coverage klaim %q: %w", claimNo, err)
	}

	return choices, nil
}

// claimHeader membaca isian yang berasal dari klaim, bukan dari pengajuan.
func (r *Repo) claimHeader(
	ctx context.Context,
	claimNo string,
) (inboxsalvage.Detail, error) {
	var no, pic, businessName, lossDate sql.NullString

	row := r.db.QueryRowContext(ctx, query("claim_header"), claimNo)
	if err := row.Scan(&no, &pic, &businessName, &lossDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
		}
		return inboxsalvage.Detail{}, fmt.Errorf("membaca klaim %q: %w", claimNo, err)
	}

	return inboxsalvage.Detail{
		ClaimNo:      strings.TrimSpace(no.String),
		PIC:          strings.TrimSpace(pic.String),
		BusinessName: strings.TrimSpace(businessName.String),
		LossDate:     dateOnly(lossDate.String),
		Items:        []inboxsalvage.DetailBarang{},
	}, nil
}

// latestSalvageOfClaim mencari ID pengajuan terakhir milik sebuah klaim.
//
// Kosong berarti klaim itu belum punya pengajuan sama sekali — bukan galat.
func (r *Repo) latestSalvageOfClaim(
	ctx context.Context,
	claimNo string,
) (string, error) {
	var id sql.NullString

	row := r.db.QueryRowContext(ctx, query("latest_salvage_of_claim"), claimNo)
	if err := row.Scan(&id); err != nil {
		// Agregat `MAX` selalu mengembalikan satu baris, bahkan ketika tidak ada baris
		// yang cocok — barisnya berisi NULL. ErrNoRows karena itu tidak diharapkan;
		// ditangani supaya perubahan kuerinya kelak tidak menjadi galat yang membingungkan.
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("mencari pengajuan terakhir klaim %q: %w", claimNo, err)
	}

	return strings.TrimSpace(id.String), nil
}

// detailHeader membaca kepala panel.
func (r *Repo) detailHeader(
	ctx context.Context,
	salvageID string,
) (inboxsalvage.Detail, error) {
	var (
		id, claimNo, inputDate, salvageType, quantity  sql.NullString
		estimate, location, transferGA, transferStatus sql.NullString
		acceptanceDate, acceptanceNo, remark, currency sql.NullString
		objectName, objectID, coverageName, coverageID sql.NullString
		acceptedValue, email, offerValue, winnerName   sql.NullString
		auctionDate, surveyorName, surveyorPhone       sql.NullString
		surveyorEmail, inJabodetabek, legacyFlag       sql.NullString
		businessName                                   sql.NullString
	)

	err := r.db.QueryRowContext(
		ctx, query("detail_header"), PegaWorkKeyPrefix, salvageID,
	).Scan(
		&id, &claimNo, &inputDate, &salvageType, &quantity,
		&estimate, &location, &transferGA, &transferStatus,
		&acceptanceDate, &acceptanceNo, &remark, &currency,
		&objectName, &objectID, &coverageName, &coverageID,
		&acceptedValue, &email, &offerValue, &winnerName,
		&auctionDate, &surveyorName, &surveyorPhone,
		&surveyorEmail, &inJabodetabek, &legacyFlag,
		&businessName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}
	if err != nil {
		return inboxsalvage.Detail{}, fmt.Errorf(
			"inboxsalvage/sqlstore: detail_header: %w", err)
	}

	return inboxsalvage.Detail{
		// Baris yang terbaca dari `PNC_SALVAGE` berarti pengajuannya memang ada.
		HasSubmission: true,

		SalvageID:            id.String,
		ClaimNo:              claimNo.String,
		BusinessName:         businessName.String,
		InputDate:            dateOnly(inputDate.String),
		SalvageType:          salvageType.String,
		Quantity:             quantity.String,
		EstimateValue:        estimate.String,
		Location:             location.String,
		TransferGADate:       dateOnly(transferGA.String),
		TransferStatus:       transferStatus.String,
		Position:             inboxsalvage.PositionLabelOf(transferStatus.String),
		AcceptanceDate:       dateOnly(acceptanceDate.String),
		AcceptanceNo:         acceptanceNo.String,
		Remark:               remark.String,
		Currency:             currency.String,
		ObjectID:             objectID.String,
		ObjectName:           objectName.String,
		CoverageID:           coverageID.String,
		CoverageName:         coverageName.String,
		AcceptedValue:        acceptedValue.String,
		Email:                email.String,
		OfferValue:           offerValue.String,
		WinnerName:           winnerName.String,
		AuctionDate:          dateOnly(auctionDate.String),
		SurveyorName:         surveyorName.String,
		SurveyorPhone:        surveyorPhone.String,
		SurveyorEmail:        surveyorEmail.String,
		InJabodetabek:        flagIsSet(inJabodetabek.String),
		LegacyBeforeJuly2023: flagIsSet(legacyFlag.String),
	}, nil
}

// detailItems membaca daftar barang pada satu pengajuan.
func (r *Repo) detailItems(
	ctx context.Context,
	claimNo, salvageID string,
) ([]inboxsalvage.DetailBarang, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_items"), claimNo, salvageID)
	if err != nil {
		return nil, fmt.Errorf("inboxsalvage/sqlstore: detail_items: %w", err)
	}
	defer rows.Close()

	items := []inboxsalvage.DetailBarang{}
	for rows.Next() {
		var (
			name, unit, total, soldStatus          sql.NullString
			winner, acceptanceNo, accepted, remark sql.NullString
			count                                  int
		)
		if err := rows.Scan(
			&name, &count, &unit, &total, &soldStatus,
			&winner, &acceptanceNo, &accepted, &remark,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxsalvage/sqlstore: detail_items: memindai baris: %w", err)
		}

		items = append(items, inboxsalvage.DetailBarang{
			Name:          name.String,
			Count:         count,
			Unit:          unit.String,
			TotalValue:    total.String,
			SoldStatus:    inboxsalvage.SoldStatusOf(soldStatus.String),
			WinnerName:    winner.String,
			AcceptanceNo:  acceptanceNo.String,
			AcceptedValue: accepted.String,
			Remark:        remark.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxsalvage/sqlstore: detail_items: membaca hasil: %w", err)
	}

	return items, nil
}

// flagIsSet membaca penanda biner yang disimpan sebagai teks.
//
// Hanya `"1"` yang berarti benar. Kolomnya dapat berisi `NULL`, `"0"`, atau `"1"` — dan
// memperlakukan apa pun yang tidak kosong sebagai benar akan membalik arti `"0"`.
func flagIsSet(value string) bool {
	return strings.TrimSpace(value) == "1"
}

// Counts menyusun tabel ringkas "Status Salvage / Jumlah".
//
// # Kenapa satu kueri per baris, bukan satu kueri berisi banyak SUM
//
// Karena baris-barisnya menghitung TABEL YANG BERBEDA — dua di antaranya `T_CLAIM_PNC`,
// sepuluh `PNC_SALVAGE`, satu `DETAIL_PNC_SALVAGE`. Sistem lama menempuhnya dengan lima
// rule terpisah, dan tiga di antaranya menggabung-silangkan dua tabel tanpa syarat gabungan
// di `WHERE` hanya supaya seluruh hitungan muat dalam satu pernyataan. Itu bukan pola yang
// layak ditiru.
func (r *Repo) Counts(
	ctx context.Context,
	caller inboxsalvage.Caller,
) ([]inboxsalvage.StatusCount, error) {
	definitions := inboxsalvage.CountRows()
	counts := make([]inboxsalvage.StatusCount, 0, len(definitions))

	for _, definition := range definitions {
		total, err := r.countOne(ctx, definition, caller)
		if err != nil {
			return nil, err
		}
		counts = append(counts, inboxsalvage.StatusCount{
			Label: definition.Label,
			Tab:   definition.Tab,
			Total: total,
		})
	}

	return counts, nil
}

// countOne menjalankan satu hitungan pencacah.
func (r *Repo) countOne(
	ctx context.Context,
	definition inboxsalvage.CountRow,
	caller inboxsalvage.Caller,
) (int, error) {
	var (
		name string
		args []any
	)

	switch definition.Source {
	case inboxsalvage.CountFromSalvage:
		owner := patternAll
		if definition.OwnedByCaller {
			owner = escapeLike(caller.Login)
		}
		name = "count_salvage_status"
		args = []any{
			definition.TransferStatuses[0],
			lastOrFirst(definition.TransferStatuses),
			owner,
		}

	case inboxsalvage.CountFromClaimBuyback:
		name = "count_claim_buyback"

	case inboxsalvage.CountFromSalvageDetail:
		name = "count_salvage_detail_unsold"
		args = []any{UnsoldDetailStatus}

	default:
		name = "count_claim_status"
		args = []any{
			definition.SalvageStatuses[0],
			lastOrFirst(definition.SalvageStatuses),
		}

		// Baris Outstanding mencacah POPULASI YANG SAMA dengan daftarnya: penanda
		// salvage yang sama, DITAMBAH penyaring status kerja. Lihat
		// inboxsalvage.CountRow.ExcludesClosedWork.
		if definition.ExcludesClosedWork {
			name = "count_claim_outstanding"
			args = append(args,
				inboxsalvage.ClosedWorkStatuses[0],
				inboxsalvage.ClosedWorkStatuses[1],
			)
		}
	}

	var total int
	if err := r.db.QueryRowContext(ctx, query(name), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("inboxsalvage/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// lastOrFirst mengembalikan nilai kedua sebuah pasangan, atau mengulang yang pertama bila
// hanya ada satu.
//
// Kueri pencacah memakai `IN (:1, :2)` supaya satu pernyataan melayani baris yang
// menghitung satu nilai maupun dua. Mengulang nilai yang sama tidak mengubah hasilnya.
func lastOrFirst(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

// Create menyimpan satu pengajuan salvage beserta detail itemnya.
//
// # SATU transaksi, bukan sembilan commit
//
// Sistem lama menempuh jalur ini lewat procedure yang meng-`COMMIT` sendiri
// (`Database/INSERT_SALVAGE.prc:26` dan `:47`), lalu procedure KEDUA yang meng-`COMMIT`
// lagi untuk setiap baris detail. Kegagalan di tengah meninggalkan pengajuan tanpa detail,
// atau detail sebagian.
//
// `D-68` memindahkan kepemilikan transaksi ke Go. Di sini seluruhnya — pengajuan dan
// seluruh baris detailnya — berada dalam SATU transaksi: gagal berarti tidak ada apa pun
// yang tertinggal.
//
// Konsekuensinya untuk uji kesetaraan disadari: keadaan akhir SAAT GAGAL berbeda dari
// sistem lama, dan itu selisih yang disengaja (`14-TESTING-STRATEGY.md` §6.4).
func (r *Repo) Create(ctx context.Context, form inboxsalvage.Form) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	salvageID := form.SalvageID
	if form.Mode != inboxsalvage.FormModeUpdate {
		salvageID, err = nextSalvageID(ctx, tx)
		if err != nil {
			return "", err
		}
	}

	args := insertArgs(form, salvageID)

	statement := "insert_salvage"
	if form.Mode == inboxsalvage.FormModeUpdate {
		statement = "update_salvage"
	}

	result, err := tx.ExecContext(ctx, query(statement), args...)
	if err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: %s: %w", statement, err)
	}

	// Pembaruan yang tidak menyentuh satu baris pun berarti pengajuannya tidak ada.
	// Membiarkannya lolos akan melaporkan "tersimpan" untuk penyimpanan yang tidak terjadi.
	if form.Mode == inboxsalvage.FormModeUpdate {
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return "", inboxsalvage.ErrRowNotFound
		}
	}

	// Menyunting MENGGANTI daftar barang, bukan menambahkannya.
	//
	// Itu mekanisme layar lama apa adanya — lihat delete_salvage_detail. Tanpa langkah
	// ini, menyunting pengajuan berbarang tiga lalu menekan Submit menghasilkan enam
	// baris, dan barisnya membawa nilai uang.
	if form.Mode == inboxsalvage.FormModeUpdate {
		_, err := tx.ExecContext(
			ctx, query("delete_salvage_detail"), form.ClaimNo, salvageID)
		if err != nil {
			return "", fmt.Errorf("inboxsalvage/sqlstore: delete_salvage_detail: %w", err)
		}
	}

	if err := insertDetails(ctx, tx, form, salvageID); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: menyimpan transaksi: %w", err)
	}
	return salvageID, nil
}

// nextSalvageID membaca nomor ID salvage berikutnya.
//
// # Cacat yang DIWARISI, dan sejauh mana ia ditutup
//
// `MAX(IDSALVAGE) + 1` dapat menghasilkan nomor yang sama bagi dua penyimpanan yang
// berjalan bersamaan. Procedure lama punya cacat yang sama persis.
//
// Yang ditutup dari sisi ini: pembacaannya berada DI DALAM transaksi yang sama dengan
// penyisipannya, sehingga benturan berakhir sebagai galat kunci ganda — bukan sebagai dua
// baris ber-ID sama yang keduanya tersimpan. Penyimpanan yang kalah menerima galat dan
// pengguna dapat mengulang.
//
// Yang TIDAK ditutup: benturannya sendiri. Menghapusnya menuntut sequence, dan memindahkan
// penomoran yang sudah berjalan di produksi ke sequence adalah keputusan tersendiri yang
// belum diambil.
func nextSalvageID(ctx context.Context, tx *sql.Tx) (string, error) {
	var next int64
	if err := tx.QueryRowContext(ctx, query("next_salvage_id")).Scan(&next); err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: next_salvage_id: %w", err)
	}
	return strconv.FormatInt(next, 10), nil
}

// insertArgs menyusun ke-22 bind pernyataan simpan.
//
// Urutannya mengikuti urutan bind di inboxsalvage.sql, dan kesesuaiannya dijaga
// query_test.go. Pemetaannya ke parameter procedure lama ada di kepala berkas .sql itu.
func insertArgs(form inboxsalvage.Form, salvageID string) []any {
	return []any{
		form.ClaimNo,                        // :1  NOKLAIM
		nullIfEmpty(form.InputDate),         // :2  TGLINPUT
		form.SalvageType,                    // :3  JENISSALVAGE
		numberOrNull(form.Quantity),         // :4  QUANTITYSALVAGE
		numberOrNull(form.MinimumValue),     // :5  ESTIMASINILAI
		nullIfEmpty(form.Location),          // :6  LOKASISALVAGE
		form.TransferStatus,                 // :7  STSTRANSFER
		nullIfEmpty(form.Remark),            // :8  REMARK
		form.Caller.Login,                   // :9  PIC
		salvageID,                           // :10 IDSALVAGE
		nullIfEmpty(form.ObjectID),          // :11 IDOBJECT
		form.ObjectName,                     // :12 OBJECTNAME
		nullIfEmpty(form.CoverageID),        // :13 IDCOVERAGE
		form.CoverageName,                   // :14 COVERAGENAME
		nullIfEmpty(form.Currency),          // :15 CURRENCY
		numberOrNull(form.InsuredShare),     // :16 NILAIAKSEP
		nullIfEmpty(form.Email),             // :17 EMAIL
		numberOrNull(form.OfferValue),       // :18 NILAIPENAWARAN
		nullIfEmpty(form.SurveyorName),      // :19 PICSURVEY
		nullIfEmpty(form.SurveyorEmail),     // :20 EMAILSURVEY
		nullIfEmpty(form.SurveyorPhone),     // :21 NOTELP
		jabodetabekFlag(form.InJabodetabek), // :22 ISJABODATABEK
	}
}

// Nilai kolom `DETAIL_PNC_SALVAGE` yang di procedure lama ditulis TETAP.
//
// Keduanya diikat, bukan ditanam di dalam SQL, karena keduanya nilai bisnis (`D-15`).
// Nilainya sendiri disalin apa adanya dari `Database/INSERT_SALVAGE_DETAILS.prc:26`.
const (
	// NewDetailStatus adalah `STATUSTERJUAL` baris detail yang baru disisipkan — `'3'`.
	//
	// Perhatikan `'3'` pula yang dipakai kueri agregat sebagai penyaring
	// (`statusterjual is null or statusterjual = '3'`), sehingga baris yang baru disimpan
	// langsung ikut terhitung pada kolom "Nilai Request Balai Lelang".
	NewDetailStatus = "3"

	// EmptyAcceptanceNo adalah `NOAKSEPTASI` baris detail yang belum berakseptasi — `'-'`.
	//
	// Tanda hubung, bukan NULL. Itu keadaan di procedure lama, dan mengubahnya menjadi
	// NULL akan membuat baris baru dan baris lama terbaca berbeda pada laporan yang
	// membandingkan kolom ini.
	EmptyAcceptanceNo = "-"

	// UnsoldDetailStatus adalah `STATUSTERJUAL` yang dihitung baris pencacah "Tidak
	// Terjual" — `'0'`, dari `Activity/GCNMCountSalvage_act-Act.xml` langkah 48.
	UnsoldDetailStatus = "0"
)

// insertDetails menyisipkan seluruh baris Detail Item Salvage.
//
// Urutan `IDDETAILSALVAGE` melanjutkan baris yang SUDAH ADA untuk pengajuan itu, sama
// seperti procedure lama yang menghitungnya lebih dulu. Hitungannya dilakukan SEKALI di
// awal, bukan sekali per baris: keduanya berada di dalam satu transaksi, sehingga tidak ada
// baris lain yang dapat menyisip di antaranya.
func insertDetails(
	ctx context.Context,
	tx *sql.Tx,
	form inboxsalvage.Form,
	salvageID string,
) error {
	if len(form.Items) == 0 {
		return nil
	}

	var existing int
	err := tx.QueryRowContext(
		ctx, query("count_salvage_detail_for"), salvageID, form.ClaimNo,
	).Scan(&existing)
	if err != nil {
		return fmt.Errorf("inboxsalvage/sqlstore: count_salvage_detail_for: %w", err)
	}

	statement := query("insert_salvage_detail")
	for index, item := range form.Items {
		sequence := existing + index + 1
		detailID := form.ClaimNo + "/" + salvageID + "/" + strconv.Itoa(sequence)

		_, err := tx.ExecContext(ctx, statement,
			salvageID,                   // :1  IDSALVAGE
			form.ClaimNo,                // :2  NOKLAIM
			detailID,                    // :3  IDDETAILSALVAGE
			item.Name,                   // :4  NAMABARANG
			nullIfEmpty(item.Unit),      // :5  SATUAN
			nullIfEmpty(item.Remarks),   // :6  NOTE_ITEM
			numberOrNull(item.Quantity), // :7  HARGAITEM
			strconv.Itoa(sequence),      // :8  IDOBJECT
			nullIfEmpty(item.Remarks),   // :9  REMARK
			NewDetailStatus,             // :10 STATUSTERJUAL
			EmptyAcceptanceNo,           // :11 NOAKSEPTASI
		)
		if err != nil {
			return fmt.Errorf(
				"inboxsalvage/sqlstore: insert_salvage_detail baris %d: %w",
				index+1, err)
		}
	}

	return nil
}

// jabodetabekFlag menerjemahkan penanda "Lokasi Salvage Di Jabodatabek".
//
// Kolomnya bertipe teks di basis data, dan nilainya `"1"` atau `"0"` — bentuk yang sama
// dipakai seluruh penanda biner di skema ini.
func jabodetabekFlag(in bool) string {
	if in {
		return "1"
	}
	return "0"
}

// nullIfEmpty mengirim NULL alih-alih teks kosong.
//
// Perbedaannya nyata di layar: kolom yang NULL tergambar kosong, sementara teks kosong
// tergambar kosong pula — tetapi keduanya berbeda pada kueri yang memakai `IS NULL`, dan
// kueri agregat modul ini memakainya.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// numberOrNull mengirim NULL alih-alih teks kosong untuk kolom bertipe angka.
//
// Isian kosong TIDAK boleh dikirim sebagai `""`, karena ekspresi berikut gagal di Oracle
// dengan `ORA-01722`:
//
//	TO_NUMBER('')
//
// dan galat itu sampai ke pengguna sebagai kegagalan mentah.
//
// Nilainya sendiri tetap dikirim sebagai TEKS, bukan sebagai `float64`. `D-51` menetapkan
// nilai uang disimpan presisi penuh; mengubahnya menjadi bilangan pecahan biner di sini
// berarti pembulatan terjadi sebelum nilainya sampai ke basis data.
func numberOrNull(value string) any {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	return strings.ReplaceAll(clean, ",", ".")
}

// dateOnly memotong bagian waktu dari nilai tanggal yang dibaca basis data.
//
// Driver mengembalikan `DATE` Oracle sebagai teks berbentuk `YYYY-MM-DD HH24:MI:SS` pada
// sebagian setelan. Yang digambar layar hanyalah tanggalnya, dan yang dihitung "Aging" pun
// hanya tanggalnya.
func dateOnly(value string) string {
	clean := strings.TrimSpace(value)
	if len(clean) > len(inboxsalvage.DateLayout) {
		return clean[:len(inboxsalvage.DateLayout)]
	}
	return clean
}

// today membaca tanggal hari ini menurut BASIS DATA, bukan menurut mesin aplikasi.
//
// # Kenapa basis data
//
// Karena tanggal input salvage pun berasal dari sana. Menghitung selisih antara tanggal
// basis data dan jam mesin aplikasi berarti membandingkan dua jam yang dapat berbeda —
// dan pada layar yang menghitung umur pekerjaan, selisih satu hari terlihat sebagai
// pekerjaan yang terlambat padahal tidak.
//
// `CURRENT_DATE` dipakai, bukan `SYSDATE`: yang pertama mengikuti zona waktu sesi, yang
// kedua zona waktu mesin basis data. Perbedaannya adalah persis kelas cacat yang `R-12`
// catat.
func (r *Repo) today(ctx context.Context) (string, error) {
	var value sql.NullString
	err := r.db.QueryRowContext(
		ctx, "SELECT TO_CHAR(CURRENT_DATE, 'YYYY-MM-DD') FROM DUAL",
	).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: membaca tanggal basis data: %w", err)
	}
	return value.String, nil
}

// Ready memeriksa kolom yang dibaca modul ini memang ada.
//
// Dipakai perintah `check`, bukan jalur permintaan. Ia menjawab pertanyaan yang paling
// sering muncul saat layar kosong: apakah kolomnya tidak ada, atau datanya yang tidak ada.
func (r *Repo) Ready(ctx context.Context) (int, error) {
	var found int
	err := r.db.QueryRowContext(ctx, query("check_salvage_columns")).Scan(&found)
	if err != nil {
		return 0, fmt.Errorf("inboxsalvage/sqlstore: check_salvage_columns: %w", err)
	}
	return found, nil
}

// MarkSentToAuction mencatat jawaban balai lelang pada baris pengajuan.
//
// # Nilai kosong dikirim sebagai NULL, bukan sebagai teks kosong
//
// `COALESCE(:2, IDSIMASBID)` pada kuerinya hanya bekerja bila yang sampai ke basis data
// benar-benar NULL. Mengirim string kosong akan menimpa nomor SimasBid yang sudah ada
// dengan teks kosong — persis kelas cacat `IDSALVAGE` yang sudah diperbaiki sekali di
// modul ini (`D-49` #9), dan tidak boleh lahir kembali di kolom sebelahnya.
func (r *Repo) MarkSentToAuction(
	ctx context.Context, salvageID string, receipt inboxsalvage.AuctionReceipt,
) error {
	id := strings.TrimSpace(salvageID)
	if id == "" {
		return inboxsalvage.ErrRowNotFound
	}

	var auctionID any
	if nomor := strings.TrimSpace(receipt.AuctionID); nomor != "" {
		auctionID = nomor
	}

	result, err := r.db.ExecContext(
		ctx, query("mark_sent_to_auction"), receipt.TransferStatus(), auctionID, id)
	if err != nil {
		return fmt.Errorf("inboxsalvage/sqlstore: mark_sent_to_auction: %w", err)
	}

	// Pembaruan yang tidak menyentuh satu baris pun berarti pengajuannya tidak ada.
	// Membiarkannya lolos akan melaporkan "terkirim dan tercatat" untuk pencatatan yang
	// tidak terjadi.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return inboxsalvage.ErrRowNotFound
	}
	return nil
}
