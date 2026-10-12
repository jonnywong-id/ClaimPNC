package inboxbandinghargasalvage

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Berkas ini memuat tombol **"Lihat File"** pada kolom Action grid Request.
//
// # Apa yang digantikan
//
//	Section/ButtonApproveRejectedRequest-Section.xml     tombolnya (`pyLocalAction`)
//	Flow Action/DokumenBandingSalvage-FA.xml             dialognya
//	Activity/LihatDokRequestSalvage-Act.xml              pemasok datanya
//	Section/DokBandingHargaSalvage-Section.xml           grid di dalam dialognya
//
// # Alur di layar lama, dan dari mana tiap kolomnya datang
//
// Activity-nya menempuh DUA kueri, dan yang pertama tidak punya rule sendiri — teksnya
// dirangkai menjadi properti klipboard lalu dijalankan mentah:
//
//	TempClaimAttach.AlasanKlaim := "select iddoc as \"ClaimID\", tglins as \"DateOfLoss\"
//	                                  from POOLDATA.SALAVAGEDOCUMENT
//	                                 where noklaim = '" + Param.iddetailsalvage + "'
//	                                   and idsalvage = '" + Param.IDsalvage + "'
//	                                   and tipedocsalvage = 'Request Banding Harga Salvage'
//	                                   and idbalailelang is null"
//
// lalu untuk SETIAP baris hasilnya, satu kueri lagi ke tabel lampiran dengan `pyID = IDDOC`.
//
// Ketiga kolom dialognya karena itu berasal dari tempat yang berbeda-beda:
//
//	Kategori   KONSTANTA "BandingHarga" — bukan dari basis data sama sekali
//	Nama       DATA_ATTACHFILE.ATTACHNAME
//	Tanggal    SALAVAGEDOCUMENT.TGLINS — bukan INPUTDATE tabel lampiran
//
// Yang terakhir mudah salah: activity menyalinnya ke properti bernama `INPUTDATE`, padahal
// nilainya diambil dari `TGLINS` tabel dokumen. Nama properti klipboard sekali lagi tidak
// menyatakan asal nilainya.
//
// # Satu rule yang tetap tidak ada, dan kenapa ia tidak lagi menahan
//
// `GetAttachmentReqSalvage` — kueri kedua itu — tidak ada di export. Isinya tidak perlu
// ditebak: kelasnya `ASM-FW-GCNMFW-Int-DATA_ATTACHFILE`, kuncinya `pyID` yang diisi `IDDOC`,
// dan ketiga kolom yang dibaca hasilnya adalah `ATTACHNAME`, `ATTACHMIMETYPE`, dan
// `ATTACHFILE`. Kueri yang sama persis sudah terbukti berjalan di modul `inboxpladla`
// (`repo/sqlstore/detail.sql`), atas tabel yang sama dan kunci yang sama.

// DocumentCategory adalah isi kolom "Kategori" pada dialog ini.
//
// Ia KONSTANTA, bukan kolom basis data: activity lama menetapkannya
// `tempAttachment.pxResults(<LAST>).ATTACHNOTE := "BandingHarga"` untuk setiap baris. Nilai
// kolom `ATTACHNOTE` yang sesungguhnya di basis data tidak pernah dibaca.
//
// Ia tetap digambar karena layar lama menggambarnya (`D-13`), dan dinyatakan sebagai
// konstanta di sini supaya tidak ada yang mencarinya di tabel.
const DocumentCategory = "BandingHarga"

// documentType adalah nilai `SALAVAGEDOCUMENT.TIPEDOCSALVAGE` yang menandai dokumen banding.
//
// Ia tidak diekspor: ia penyaring kueri, bukan sesuatu yang dibaca layar.
const documentType = "Request Banding Harga Salvage"

// ErrDocumentNotFound berarti dokumen yang diminta tidak dapat diserahkan.
//
// Dua sebab, dan keduanya sengaja TIDAK dibedakan: dokumennya memang tidak ada, atau ia
// bukan milik banding yang ditangani komite ini. Membedakannya memberi tahu penanya bahwa
// sebuah `DATAID` nyata — dan id itu berurutan, sehingga menebaknya mudah.
var ErrDocumentNotFound = errors.New(
	"inboxbandinghargasalvage: dokumen tidak ditemukan atau bukan milik Anda")

// DocumentRow adalah satu baris pada dialog "Lihat File".
type DocumentRow struct {
	// ID adalah `SALAVAGEDOCUMENT.IDDOC`, yang sekaligus `DATA_ATTACHFILE.DATAID`.
	//
	// Kedua kolom itu memang bernilai sama — activity lama menyalinnya begitu saja
	// (`tempAttachment…DATAID := .ClaimID`, dan `.ClaimID` di sana adalah `IDDOC`).
	ID string

	// Name adalah `DATA_ATTACHFILE.ATTACHNAME` — nama berkas yang diunduh.
	Name string

	// UploadedAt adalah `SALAVAGEDOCUMENT.TGLINS`. Nil berarti kolomnya kosong.
	UploadedAt *time.Time
}

// Category menyerahkan isi kolom "Kategori" baris ini.
//
// Ia method, bukan field, supaya tidak ada satu pun adapter yang tergoda mengisinya dari
// basis data — nilainya konstanta, dan itu fakta tentang layar lama yang layak dijaga.
func (DocumentRow) Category() string { return DocumentCategory }

// DocumentContent adalah isi satu dokumen beserta keterangan penyerahannya.
type DocumentContent struct {
	Name     string
	MIMEType string
	Content  []byte
}

// DocumentQuery adalah permintaan daftar dokumen satu banding, sudah tervalidasi.
type DocumentQuery struct {
	// DetailObject adalah `IDDETAILSALVAGE`.
	//
	// Ia dibandingkan dengan kolom `SALAVAGEDOCUMENT.NOKLAIM` — dan itu BUKAN salah ketik
	// di sini. Kueri lama membandingkan keduanya, dan kueri itu kueri BACA: bila tidak
	// pernah mencocokkan apa pun, dialognya selalu kosong dan pengguna akan melaporkannya.
	// Kolom itu karena itu hampir pasti berisi id detail salvage meski namanya `NOKLAIM`.
	// Lihat catatan di PlannedDifferences.
	DetailObject string

	// SalvageID adalah `IDSALVAGE`.
	SalvageID string

	// Reviewer menyatakan atas nama siapa dokumennya dibaca.
	//
	// Ia TIDAK ada di kueri lama, dan penambahannya disengaja — lihat catatan penyaring
	// kepemilikan pada kueri `list_documents`.
	Reviewer Reviewer
}

// NewDocumentQuery membentuk permintaan daftar dokumen yang sah.
func NewDocumentQuery(
	detailObject, salvageID string,
	caller Caller,
) (DocumentQuery, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return DocumentQuery{}, ErrCallerUnknown
	}

	violations := []Violation{}

	detail := strings.TrimSpace(detailObject)
	if detail == "" {
		violations = append(violations, Violation{
			Field:   FieldDetailObject,
			Message: "Barang yang dokumennya dibuka tidak terbaca. Muat ulang daftarnya.",
		})
	}

	salvage := strings.TrimSpace(salvageID)
	if salvage == "" {
		violations = append(violations, Violation{
			Field:   FieldSalvageID,
			Message: "Pengajuan salvage tidak terbaca. Muat ulang daftarnya.",
		})
	}

	if len(violations) > 0 {
		return DocumentQuery{}, NewValidationError(violations)
	}

	return DocumentQuery{
		DetailObject: detail,
		SalvageID:    salvage,
		Reviewer:     ReviewerFor(cleanCaller),
	}, nil
}

// DocumentReader adalah seam ke dokumen banding harga.
//
// Ia TERPISAH dari Repo, dengan alasan yang sama seperti Writer: ia dipakai satu layar
// sekunder saja, dan menyatukannya berarti setiap pengisi Repo — termasuk yang dipakai uji
// daftar — ikut memikul pembacaan lampiran yang tidak dipakainya.
//
// Pemisahan itu punya satu manfaat tambahan di sini: pengisi yang TIDAK menyediakan
// pembacaan lampiran tetap sah, dan layarnya menyatakan tombol "Lihat File" tidak tersedia
// alih-alih gagal.
type DocumentReader interface {
	// ListDocuments menyerahkan dokumen banding satu barang.
	//
	// Daftar KOSONG bukan galat: sebuah banding boleh diajukan tanpa dokumen pendukung, dan
	// dokumen yang sudah ditandai ditolak sengaja tidak ikut — lihat kueri.
	ListDocuments(ctx context.Context, query DocumentQuery) ([]DocumentRow, error)

	// DocumentContent menyerahkan isi satu dokumen.
	//
	// Mengembalikan ErrDocumentNotFound bila dokumennya tidak ada ATAU bukan milik banding
	// yang ditangani komite pemanggil.
	DocumentContent(
		ctx context.Context, documentID string, query DocumentQuery,
	) (DocumentContent, error)
}

// DocumentReaderSelector memilih DocumentReader milik satu portal entitas.
//
// Alasannya sama dengan RepoSelector. Di sini taruhannya khas: yang diserahkan adalah ISI
// BERKAS, sehingga permintaan yang jatuh ke koneksi bawaan menyerahkan dokumen milik badan
// hukum lain — dan tidak seperti angka di grid, berkas yang sudah terunduh tidak dapat
// ditarik kembali (`R-20`).
type DocumentReaderSelector func(portalAlias string) (DocumentReader, error)
