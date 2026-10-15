package sqlstore

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// NAMA DOKTER BENAR-BENAR TERIKAT KE `TC_PNC_PUCL.NAMA_DOKTER_RCL`.
//
// # Kenapa diuji di sini, bukan cukup di usecase
//
// Uji usecase membuktikan nama dokter BERTAHAN di dalam surat. Ia tidak membuktikan nilai
// itu sampai ke kolomnya: di antara keduanya ada daftar argumen posisional, dan go-ora
// mengikat menurut urutan KEMUNCULAN penanda — bukan menurut nomornya. Satu argumen yang
// bergeser menaruh nama dokter di kolom lain, tanpa galat sintaks.
//
// Itulah persis bentuk gejala yang dilaporkan Work Owner: nilainya ada di surat, kolomnya
// tetap kosong. Uji ini menutup jarak terakhir antara keduanya.
func TestSuratRCLPUCLMengikatNamaDokterKeKolomnya(t *testing.T) {
	const dokter = "WAHYUKRISTANTI"

	surat := registrasi.PUCLLetter{
		ClaimID:      "PNCN.26.31",
		Track:        registrasi.PUCLTrackRCL,
		AnalystNote:  "Dokumen medis belum lengkap.",
		BodyNote:     "Mohon melengkapi hasil laboratorium.",
		DoctorName:   dokter,
		TechnicalPIC: "TEKNIK01",
		Operator:     "ANALIS01",
		SentAt:       time.Date(2026, 10, 7, 3, 0, 0, 0, time.UTC),
	}

	// Posisinya BERBEDA di kedua pernyataan: UPDATE menutup daftarnya dengan CLAIMID,
	// sedangkan INSERT membukanya. Keduanya diuji supaya kedua jalur upsert terbukti —
	// bukan hanya yang kebetulan dijalankan pada satu klaim contoh.
	t.Run("jalur UPDATE menaruhnya di argumen ke-25", func(t *testing.T) {
		db, mock := be4DB(t)

		mock.ExpectExec(be4Q("surat_rclpucl_perbarui")).
			WithArgs(argsDenganDokter(26, 25, dokter)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(be4Q("surat_rclpucl_dokter")).
			WillReturnResult(sqlmock.NewResult(0, 1))

		require.NoError(t, NewPUCLStore(db).SaveLetter(context.Background(), surat))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("jalur INSERT menaruhnya di argumen ke-26", func(t *testing.T) {
		db, mock := be4DB(t)

		// UPDATE tidak mengenai satu baris pun -> INSERT dijalankan.
		mock.ExpectExec(be4Q("surat_rclpucl_perbarui")).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(be4Q("surat_rclpucl_sisip")).
			WithArgs(argsDenganDokter(26, 26, dokter)...).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(be4Q("surat_rclpucl_dokter")).
			WillReturnResult(sqlmock.NewResult(0, 1))

		require.NoError(t, NewPUCLStore(db).SaveLetter(context.Background(), surat))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Jalur PUCL tidak menampilkan isian Nama Dokter, sehingga kolomnya ditulis NULL — bukan
// teks kosong, yang akan membuat baris lama dan baris baru terbaca berbeda oleh laporan.
func TestSuratRCLPUCLJalurPUCLMenulisNamaDokterNULL(t *testing.T) {
	db, mock := be4DB(t)

	mock.ExpectExec(be4Q("surat_rclpucl_perbarui")).
		WithArgs(argsDenganDokter(26, 25, nil)...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Jalur PUCL tidak menyentuh `surat_rclpucl_dokter` sama sekali.

	require.NoError(t, NewPUCLStore(db).SaveLetter(context.Background(), registrasi.PUCLLetter{
		ClaimID:  "PNCN.26.32",
		Track:    registrasi.PUCLTrackPUCL,
		Operator: "ANALIS01",
		SentAt:   time.Date(2026, 10, 7, 3, 0, 0, 0, time.UTC),
	}))
	require.NoError(t, mock.ExpectationsWereMet())
}

// argsDenganDokter menyusun `total` argumen yang seluruhnya bebas KECUALI posisi `pos`.
//
// Hanya satu posisi yang dipatok, dan itu disengaja: yang diuji di sini adalah LETAK nama
// dokter, bukan isi dua puluh lima argumen lain yang sudah dijaga uji lain. Memaku
// seluruhnya membuat uji ini gagal setiap kali kolom yang tidak berkaitan berubah — dan
// uji yang gagal karena alasan yang tidak berkaitan adalah uji yang akhirnya diabaikan.
func argsDenganDokter(total, pos int, nilai driver.Value) []driver.Value {
	args := make([]driver.Value, total)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	args[pos-1] = nilai
	return args
}
