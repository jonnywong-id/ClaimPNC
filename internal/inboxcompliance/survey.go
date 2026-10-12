package inboxcompliance

import "time"

// SurveyResult adalah satu baris pada blok "Hasil Investigasi" form Compliance Checker.
//
// # Dari mana bentuk ini berasal
//
// `Section/ComplianceChecker-Section.xml` layout S2 berjudul **"Hasil Investigasi"**,
// bersyarat `IsPA`, menyisipkan `Section/ViewHasilSurvey-Section.xml`.
//
// Di dalam section itu ada DUA bentuk yang saling meniadakan:
//
//	S13/S14/S15  `!isPA_PNC`  delapan isian kepala + grid lima kolom
//	S20/S21      `isPA_PNC`   grid EMPAT kolom saja, tanpa isian kepala
//
// `isPA_PNC` = `.Policy.Quotation.GroupPanel = "002"`, dan `IsPA` =
// `pyWorkPage.Policy.Quotation.GroupPanel = "002"` — syarat yang sama atas properti yang
// sama. Karena blok ini HANYA muncul ketika `IsPA`, di dalamnya `isPA_PNC` selalu benar.
//
// Artinya pada form Compliance Checker yang tergambar **hanya S20/S21**: satu tabel
// baca-saja berisi empat kolom. Kedelapan isian kepala milik lini selain PA, dan tidak
// pernah tampil di sini.
//
// # Kolomnya, dan sumbernya di basis data
//
//	"Tanggal Investigasi"  .SurveyDate       POOLDATA.T_SURVEYORLIST.SURVEYDATE
//	"Nama Peserta"         .ObjectName       POOLDATA.T_SURVEYORLIST.OBJECT_NAME
//	"Lokasi Objek"         .ObjectLocation   POOLDATA.T_SURVEYORLIST.LOCATION_OBJECT
//	"Status"               .SurveyStatus     POOLDATA.T_SURVEYORLIST.STS_SURVEY
//
// Judul kolom diambil dari `pyValue` sel header 149–152; nama properti dari sel 154–157;
// nama kolom basis data dari `Database/INSERT_SURVEYORLIST.prc:31-32`, satu-satunya
// tempat seluruh kolom tabel itu disebut berurutan.
//
// Perhatikan "Nama Peserta" dipetakan ke `OBJECT_NAME`, bukan ke kolom bernama peserta.
// Itu memang begitu di Pega: pada lini PA objek pertanggungannya adalah **orang**,
// sehingga nama objek dan nama peserta adalah hal yang sama.
//
// # Seluruhnya baca-saja
//
// Kelima sel ber-`pyReadOnly = true`. Blok ini menampilkan hasil kerja Investigator
// kepada petugas Compliance; ia bukan tempat Compliance menulis.
type SurveyResult struct {
	// SurveyedAt adalah `SURVEYDATE`.
	//
	// Pointer karena kolomnya dapat kosong, dan tanggal kosong harus dapat dibedakan
	// dari 1 Januari tahun nol. Layar menggambarnya sebagai sel kosong, bukan "01/01/01".
	SurveyedAt *time.Time

	// ObjectName adalah `OBJECT_NAME` — "Nama Peserta" pada lini PA.
	ObjectName string

	// ObjectLocation adalah `LOCATION_OBJECT`.
	ObjectLocation string

	// Status adalah `STS_SURVEY`.
	//
	// Teks apa adanya, TIDAK dipetakan ke label lain. Kolom itu menyimpan status yang
	// sudah terbaca manusia — "Final Report", "Invoice Fee", "Close Case", dan belasan
	// lainnya — sebagaimana sudah ditetapkan modul `inboxsurvey`
	// (`repo/sqlstore/inboxsurvey.sql`, bagian "STS_SURVEY adalah ADJUSTERSTATUS_1").
	//
	// Pega menggambarnya sebagai `pxDropdown` yang READ-ONLY, yakni teksnya saja.
	Status string
}
