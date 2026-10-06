package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// PUCLStore menulis surat RCL/PUCL ke POOLDATA.TC_PNC_PUCL dan melayani kedua daftar
// pilihan modalnya. Lihat rclpucl.sql untuk asal setiap kolom dan untuk satu penyimpulan
// yang dinyatakan di sana (penyaring STS_STATUS).
type PUCLStore struct {
	db *sql.DB
}

// NewPUCLStore membentuk penyimpan surat RCL/PUCL di atas sebuah koneksi.
func NewPUCLStore(db *sql.DB) *PUCLStore { return &PUCLStore{db: db} }

var (
	_ registrasi.PUCLLetterStore  = (*PUCLStore)(nil)
	_ registrasi.PUCLOptionSource = (*PUCLStore)(nil)
)

// SaveLetter menyimpan satu baris surat, lalu — hanya pada jalur RCL — memperbarui kedua
// kolom yang disaring layar Inbox RCL pada baris daftar kerja klaim.
//
// UPSERT, bukan INSERT: primary key `TC_PNC_PUCL_PK` adalah `CLAIMID` TUNGGAL, sehingga
// satu klaim hanya boleh punya satu baris. Klaim yang kembali dari RCL/PUCL lalu dikirim
// lagi memperbarui barisnya; menyisipkan baris kedua akan ditolak ORA-00001.
//
// Keduanya berjalan di dalam transaksi pemanggil (`executorFrom`), sehingga surat yang
// tersimpan tanpa nama dokternya tidak mungkin terjadi.
//
// # Jalur RCL ditulis, tetapi tidak dimasukkan ke antrean RCL/PUCL
//
// Lihat `registrasi.PUCLLetter.InRCLPUCLQueue`: suratnya tetap tersimpan utuh, hanya
// `STATUS_CASE` yang dibiarkan kosong sehingga tab "Cetak Surat" melewatkannya. Tidak ada
// kueri modul `inboxrclpucl` yang perlu disunting untuk itu.
func (s *PUCLStore) SaveLetter(ctx context.Context, letter registrasi.PUCLLetter) error {
	exec := executorFrom(ctx, s.db)

	// Penanda antrean RCL/PUCL. Kosong pada jalur RCL — klaimnya sedang di Inbox RCL,
	// dan baru masuk antrean ini setelah dokter RCL meneruskannya.
	statusCase := any(nil)
	if letter.InRCLPUCLQueue() {
		statusCase = registrasi.PUCLStatusCaseOpen
	}

	// Urutannya mengikuti SET pada surat_rclpucl_perbarui kolom demi kolom, dimulai
	// TGL_CREATE_PUCL. UPDATE menutup dengan CLAIMID di WHERE; INSERT membukanya
	// sebagai kolom pertama.
	values := []any{
		letter.SentAt,
		emptyTextAsNil(letter.Operator),
		emptyTextAsNil(letter.Operator),
		registrasi.WorkStatusNew,
		statusCase,
		strconv.Itoa(letter.Track),
		emptyTextAsNil(letter.AnalystNote),
		emptyTextAsNil(letter.Subject),
		emptyTextAsNil(letter.OpeningNote),
		emptyTextAsNil(letter.BodyNote),
		emptyTextAsNil(letter.ClosingNote),
		emptyTextAsNil(letter.PolicyNumber),
		emptyTextAsNil(letter.InsuredName),
		emptyTextAsNil(letter.BusinessName),
		emptyTextAsNil(letter.BranchName),
		emptyTextAsNil(letter.SourceName),
		emptyTextAsNil(letter.GroupPanel),
		emptyTextAsNil(letter.TechnicalPIC),
		emptyTextAsNil(letter.ClaimStatus),
		timeOrNil(letter.DateOfLoss),
		letter.SentAt,
		emptyTextAsNil(letter.ObjectID),
		ordinalOrNil(letter.CoverageIndex),
		ordinalOrNil(letter.AdjustmentIndex),
	}
	update := append(append([]any{}, values...), letter.ClaimID)
	insert := append([]any{letter.ClaimID}, values...)

	if err := upsert(ctx, exec,
		"surat_rclpucl_perbarui", update,
		"surat_rclpucl_sisip", insert,
	); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan surat RCL/PUCL klaim %s: %w", letter.ClaimID, err)
	}

	// Kedua penyaring Inbox RCL ditulis pada jalur RCL SAJA, dan pada jalur itu SELALU —
	// bukan hanya ketika isian "Nama Dokter" kebetulan terisi. Isian itu hanya tampil di
	// lini PA; menggantungkan penulisannya pada isian itu membuat klaim RCL lini lain
	// tidak pernah muncul di inbox mana pun. Nilai cadangannya ditetapkan
	// `usecase.SendToRCLPUCL`, yang tahu siapa pemilik tugas RCLDokter.
	if !letter.EntersRCLInbox() {
		return nil
	}
	if _, err := exec.ExecContext(ctx, loadQuery("surat_rclpucl_dokter"),
		emptyTextAsNil(strings.TrimSpace(letter.DoctorName)),
		emptyTextAsNil(letter.AnalystNote), letter.SentAt, letter.ClaimID,
	); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis Nama Dokter RCL klaim %s: %w", letter.ClaimID, err)
	}
	return nil
}

// ordinalOrNil mengirim nomor urut sebagai TEKS, atau NULL bila tidak ada.
//
// Teks karena ketiga kolom `ID_*` bertipe `VARCHAR2`, dan nol berarti "tidak ada
// adjustment" — bukan adjustment ke-0. Menulis "0" akan menunjuk baris yang tidak ada.
func ordinalOrNil(value int) any {
	if value <= 0 {
		return nil
	}
	return strconv.Itoa(value)
}

// subjectStatusFor memetakan jalur ke nilai STS_STATUS masternya.
//
// Pemetaannya PENYIMPULAN — rule penyaringnya tidak ada di export. Alasannya ditulis
// lengkap di rclpucl.sql.
//
// **Notification TIDAK menyaring apa pun**, dan itu disengaja: master `M_PERIHAL_RCLPUCL`
// hanya memuat `STS_STATUS` 1 dan 2, sehingga tidak ada kelompok Perihal yang menjadi
// miliknya. Menyaringnya ke salah satu dari keduanya berarti mengarang; mengembalikan
// daftar kosong berarti analis tidak dapat memilih Perihal sama sekali. Yang ditawarkan
// karena itu seluruh 12 barisnya.
//
// Jalur yang tidak dikenal diperlakukan sama, dengan alasan yang sama.
func subjectStatusFor(track int) (any, any) {
	switch track {
	case registrasi.PUCLTrackRCL:
		return 1, 1
	case registrasi.PUCLTrackPUCL:
		return 2, 2
	}
	return nil, nil
}

// SubjectOptions membaca pilihan Perihal untuk satu jalur.
func (s *PUCLStore) SubjectOptions(ctx context.Context, track int) ([]registrasi.PUCLSubjectOption, error) {
	flag, value := subjectStatusFor(track)
	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("perihal_rclpucl"), flag, value)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca pilihan Perihal: %w", err)
	}
	defer rows.Close()

	result := []registrasi.PUCLSubjectOption{}
	for rows.Next() {
		var option registrasi.PUCLSubjectOption
		var name sql.NullString
		if err := rows.Scan(&option.ID, &name); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: memindai pilihan Perihal: %w", err)
		}
		option.Name = strings.TrimSpace(name.String)
		option.Track = track
		result = append(result, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca pilihan Perihal: %w", err)
	}
	return result, nil
}

// maxRejectReasonRows membatasi grid alasan penolakan.
//
// Tabelnya 2.116 baris. Angka ini membatasi yang DIKIRIM, bukan yang ada: pencarian
// menyempitkannya di basis data, dan pengguna yang belum menemukan alasannya mengetik
// lebih banyak huruf alih-alih menggulir ribuan baris.
const maxRejectReasonRows = 200

// RejectReasons mencari alasan penolakan. keyword kosong mengembalikan halaman pertama.
func (s *PUCLStore) RejectReasons(ctx context.Context, keyword string, limit int) ([]registrasi.PUCLRejectReason, error) {
	if limit <= 0 || limit > maxRejectReasonRows {
		limit = maxRejectReasonRows
	}

	var flag any
	pattern := ""
	if trimmed := strings.TrimSpace(keyword); trimmed != "" {
		flag = 1
		pattern = "%" + escapeLikePattern(strings.ToUpper(trimmed)) + "%"
	}

	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("alasan_reject"),
		flag, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca alasan penolakan: %w", err)
	}
	defer rows.Close()

	result := []registrasi.PUCLRejectReason{}
	for rows.Next() {
		var id string
		var name, description sql.NullString
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: memindai alasan penolakan: %w", err)
		}
		result = append(result, registrasi.PUCLRejectReason{
			ID:          strings.TrimSpace(id),
			Name:        strings.TrimSpace(name.String),
			Description: strings.TrimSpace(description.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca alasan penolakan: %w", err)
	}
	return result, nil
}

// RCLDoctors membaca pilihan dropdown "Nama Dokter".
//
// Sumbernya tabel dan penyaring yang SAMA dengan yang dipakai Inbox RCL mencari identitas
// lama pemanggilnya, sehingga setiap pilihan di sini pasti dapat dicocokkan penyaring itu.
// Alasan lengkapnya ada di rclpucl.sql.
func (s *PUCLStore) RCLDoctors(ctx context.Context) ([]registrasi.RCLDoctorOption, error) {
	groups := registrasi.RCLDoctorAccessGroups
	if len(groups) != 3 {
		return nil, fmt.Errorf("registrasi/sqlstore: kueri dokter RCL mengikat 3 grup akses, daftarnya %d", len(groups))
	}

	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("dokter_rcl"),
		groups[0], groups[1], groups[2], registrasi.RCLDoctorExcludedAccessGroup)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca pilihan Nama Dokter: %w", err)
	}
	defer rows.Close()

	result := []registrasi.RCLDoctorOption{}
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: memindai pilihan Nama Dokter: %w", err)
		}
		if name := strings.TrimSpace(id.String); name != "" {
			result = append(result, registrasi.RCLDoctorOption{ID: name})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca pilihan Nama Dokter: %w", err)
	}
	return result, nil
}

// escapeLikePattern menetralkan ketiga aksara yang punya arti khusus di dalam LIKE,
// supaya pencarian "100%" mencari teks itu dan bukan mencocokkan segalanya. Aksara
// pelarian `\` disepakati klausa `ESCAPE '\'` pada kuerinya.
func escapeLikePattern(text string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(text)
}
