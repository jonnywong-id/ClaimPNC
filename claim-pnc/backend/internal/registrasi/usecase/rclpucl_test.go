package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Uji modal "Kirim ke RCL/PUCL" (`Section/SectionPUCL-sect.xml`). Seluruh data KARANGAN
// (`D-69`).

func isiLengkap(taskID string, track int) usecase.SendToRCLPUCLCommand {
	return usecase.SendToRCLPUCLCommand{
		TaskID:      taskID,
		Track:       track,
		AnalystNote: "  Dokumen medis belum lengkap.  ",
		Subject:     "Kelengkapan Data dan Dokumen Klaim",
		OpeningNote: "Dengan hormat,",
		BodyNote:    "Mohon melengkapi hasil laboratorium.",
		ClosingNote: "Atas perhatiannya kami ucapkan terima kasih.",
	}
}

// Jalur PUCL melompat ke Assignment6 lewat ticket `SendtoPUCL` — antrean bersama
// RCL/PUCL. Diuji dari Choose Surveyor, tahap yang keputusan sesudahnya justru
// MENGAKHIRI alur: lompatan lateral harus tetap mengirimnya, bukan menutup klaimnya.
func TestSendToRCLPUCLJalurPUCLMasukAntreanRCLPUCL(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageRCLPUCL, result.Claim.CurrentStage)
	require.Equal(t, registrasi.PUCLTrackPUCL, result.Claim.PUCLStatus)
	require.Equal(t, usecase.StatusClaimAnalyst, result.Claim.ClaimStatus)

	// Suratnya tersimpan, dan isinya sudah dirapikan spasinya.
	require.Len(t, l.pucl.Letter, 1)
	letter := l.pucl.Letter[0]
	require.Equal(t, result.Claim.ID, letter.ClaimID)
	require.Equal(t, registrasi.PUCLTrackPUCL, letter.Track)
	require.Equal(t, "Dokumen medis belum lengkap.", letter.AnalystNote)
	require.Equal(t, "Mohon melengkapi hasil laboratorium.", letter.BodyNote)
	require.Equal(t, testOperator, letter.Operator)

	require.False(t, letter.SentAt.IsZero())

	// Catatan analis terbaca di tab Progress Claim & Komunikasi.
	comms, err := l.records.Communications(ctx, result.Claim.Keys())
	require.NoError(t, err)
	require.Len(t, comms, 1)
	require.Equal(t, registrasi.ChannelSendToRCLPUCL, comms[0].Channel)
	require.Equal(t, "Dokumen medis belum lengkap.", comms[0].Message)

	trail := l.store.AuditTrail()
	require.Equal(t, "KIRIM_KE_RCLPUCL", trail[len(trail)-1].Event)
	require.Contains(t, trail[len(trail)-1].Note, "(PUCL)")

	// Tugas lama tertutup: mengirim ulang lewat tugas yang sama ditolak.
	_, err = l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.ErrorIs(t, err, registrasi.ErrTaskAlreadyDone)
}

// Jalur RCL melompat ke Assignment12 lewat ticket `RCLDokter`. Inilah sebabnya isian
// "Nama Dokter" hanya tampil pada jalur itu.
func TestSendToRCLPUCLJalurRCLJatuhKeRCLDokter(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackRCL), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageRCLDoctor, result.Claim.CurrentStage)
	require.Equal(t, registrasi.PUCLTrackRCL, result.Claim.PUCLStatus)
}

// Jalur RCL singgah di Inbox RCL lebih dulu; ia TIDAK ikut masuk antrean RCL/PUCL.
// Surat tetap tersimpan utuh — yang berbeda hanya antrean yang menerimanya.
func TestSendToRCLPUCLJalurRCLTidakMasukAntreanRCLPUCL(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	_, err := l.service.SendToRCLPUCL(context.Background(),
		isiLengkap(task.ID, registrasi.PUCLTrackRCL), l.caller)
	require.NoError(t, err)

	require.Len(t, l.pucl.Letter, 1, "isian surat tetap tersimpan, tidak dibuang")
	letter := l.pucl.Letter[0]
	require.Equal(t, "Mohon melengkapi hasil laboratorium.", letter.BodyNote)

	require.False(t, letter.InRCLPUCLQueue(), "STATUS_CASE dibiarkan kosong pada jalur RCL")
	require.True(t, letter.EntersRCLInbox())
}

// Jalur PUCL dan Notification tidak punya singgahan: keduanya langsung masuk antrean
// RCL/PUCL, dan tidak satu pun menyentuh kedua penyaring Inbox RCL.
func TestSendToRCLPUCLJalurSelainRCLLangsungKeAntreanRCLPUCL(t *testing.T) {
	for _, track := range []int{registrasi.PUCLTrackPUCL, registrasi.PUCLTrackNotification} {
		l := setup(t)
		task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

		_, err := l.service.SendToRCLPUCL(context.Background(), isiLengkap(task.ID, track), l.caller)
		require.NoError(t, err)

		letter := l.pucl.Letter[0]
		require.True(t, letter.InRCLPUCLQueue(), "jalur %d masuk antrean RCL/PUCL", track)
		require.False(t, letter.EntersRCLInbox(), "jalur %d tidak menyentuh Inbox RCL", track)
		require.Empty(t, letter.DoctorName, "jalur %d tidak menulis Nama Dokter", track)
	}
}

// Jalur RCL SELALU membawa penghuni `NAMADOKTERRCL_1`, bahkan ketika analis tidak
// mengetik apa pun — isian "Nama Dokter" hanya tampil pada lini PA, dan lini contoh di
// sini bukan PA.
//
// Tanpa ini klaim RCL lini non-PA tidak muncul di inbox siapa pun: layar Inbox RCL
// menyaring kolom itu terhadap identitas pemanggilnya.
func TestSendToRCLPUCLJalurRCLSelaluMengisiPenyaringInboxRCL(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToRCLPUCL(context.Background(),
		isiLengkap(task.ID, registrasi.PUCLTrackRCL), l.caller)
	require.NoError(t, err)

	letter := l.pucl.Letter[0]
	require.NotEmpty(t, letter.DoctorName)
	require.Equal(t, result.NextTask.Owner, letter.DoctorName,
		"penyaring dokter dan penyaring pemilik tugas harus menunjuk orang yang sama")
	require.False(t, letter.SentAt.IsZero(), "TANGGALANALYSTSENDRCL_1 ikut terisi")
}

// Ketiga isian wajib section ditolak SEKALIGUS, bukan satu per satu — form ini panjang.
func TestSendToRCLPUCLMenolakSeluruhIsianWajibSekaligus(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	_, err := l.service.SendToRCLPUCL(ctx, usecase.SendToRCLPUCLCommand{TaskID: task.ID}, l.caller)

	var broken *registrasi.ValidationError
	require.ErrorAs(t, err, &broken)
	code := map[registrasi.ViolationCode]bool{}
	for _, v := range broken.Violation {
		code[v.Code] = true
	}
	require.True(t, code[registrasi.ViolationPUCLTrackUnknown])
	require.True(t, code[registrasi.ViolationPUCLNoteEmpty])
	require.True(t, code[registrasi.ViolationPUCLBodyEmpty])

	// Tidak ada surat yang tersimpan dan klaim tidak berpindah.
	require.Empty(t, l.pucl.Letter)
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageChooseSurveyor, stored.CurrentStage)
}

// Jalur Notification melompat ke antrean RCL/PUCL, sama seperti PUCL — alur Register
// tidak punya tahap bernama Notification, dan seluruh penanganannya hidup di layar Inbox
// RCL/PUCL. Lihat catatan pada TicketSendToRCLPUCL.
func TestSendToRCLPUCLJalurNotificationMasukAntreanRCLPUCL(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToRCLPUCL(context.Background(),
		isiLengkap(task.ID, registrasi.PUCLTrackNotification), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageRCLPUCL, result.Claim.CurrentStage)
	require.Equal(t, registrasi.PUCLTrackNotification, result.Claim.PUCLStatus)
	require.Len(t, l.pucl.Letter, 1)
	require.Equal(t, registrasi.PUCLTrackNotification, l.pucl.Letter[0].Track)

	trail := l.store.AuditTrail()
	require.Contains(t, trail[len(trail)-1].Note, "(Notification)")
}

// Kode jalur di luar ketiga pilihan ditolak. `0` memang ada di data produksi (belum
// dipilih), tetapi ia bukan pilihan yang dapat dikirim dari modal ini.
func TestSendToRCLPUCLMenolakKodeJalurDiLuarTigaPilihan(t *testing.T) {
	for _, track := range []int{0, 4, 9} {
		l := setup(t)
		task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

		_, err := l.service.SendToRCLPUCL(context.Background(), isiLengkap(task.ID, track), l.caller)

		var broken *registrasi.ValidationError
		require.ErrorAs(t, err, &broken, "jalur %d seharusnya ditolak", track)
	}
}

// "Nama Dokter" hanya berlaku pada jalur RCL lini PA. Di luar itu isiannya DIBUANG,
// bukan ditolak — layar memang tidak menampilkannya.
func TestSendToRCLPUCLMembuangNamaDokterDiLuarJalurnya(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	isi := isiLengkap(task.ID, registrasi.PUCLTrackPUCL)
	isi.DoctorName = "dr. Contoh"
	_, err := l.service.SendToRCLPUCL(context.Background(), isi, l.caller)
	require.NoError(t, err)

	require.Len(t, l.pucl.Letter, 1)
	require.Empty(t, l.pucl.Letter[0].DoctorName, "lini polis contoh bukan PA")
}

// Dokter yang dipilih TERSIMPAN, tetapi BELUM memindahkan klaimnya — ditahan Work Owner.
//
// # Uji ini mengunci keputusan, bukan perilaku yang diinginkan
//
// Yang diinginkan adalah `RouterRCLDokter` apa adanya: `Param.AssignTo :=
// ClaimData.NamaDokterRCL`. Itu pernah dipasang dan lulus uji, lalu ditahan Work Owner
// (2026-10-06) karena satu mata rantai belum terverifikasi: Inbox RCL mencocokkan
// `ASSIGNED_OPERATOR_ID` dengan `M_LOGIN_PNC.LOGIN_ID` pemanggil, sedangkan yang kita
// tulis adalah `pyStandardValue` prompt list — ruang nama yang berbeda. Tidak cocok,
// klaimnya hilang dari inbox siapa pun tanpa satu pun galat.
//
// Jadi uji ini ada supaya keadaan tertahan itu **disengaja dan terlihat**, bukan diam-diam
// berlaku. Begitu pemetaannya terverifikasi, uji ini yang pertama harus dibalik.
func TestSendToRCLPUCLDokterTersimpanTetapiBelumMemindahkanKlaim(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	// Isian "Nama Dokter" hanya tampil pada lini PA.
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.Policy.Line = registrasi.LinePersonalAccident
	require.NoError(t, l.store.Save(ctx, claim))

	const dokter = "WAHYUKRISTANTI" // `pyPromptTableList` NamaDokterRCL, baris pertama
	isi := isiLengkap(task.ID, registrasi.PUCLTrackRCL)
	isi.DoctorName = dokter

	result, err := l.service.SendToRCLPUCL(ctx, isi, l.caller)
	require.NoError(t, err)

	require.Equal(t, registrasi.StageRCLDoctor, result.Claim.CurrentStage)

	// Pilihannya TERSIMPAN — analis tidak kehilangan apa yang ia isi.
	require.Equal(t, dokter, l.pucl.Letter[0].DoctorName)

	// Tetapi tugasnya BELUM berpindah ke dokter itu. Inilah yang ditahan.
	require.Equal(t, registrasi.OperatorUnassigned, result.NextTask.Owner,
		"penerapan RouterRCLDokter ditahan sampai pemetaan nama dokter ke LOGIN_ID terverifikasi")
}

// Tanpa dokter yang dipilih, tugasnya TETAP diparkir di `ServicePNC`.
//
// Pasangan dari uji di atas, dan ia yang menjaga cadangannya tidak ikut hilang: lini
// selain PA tidak menampilkan isian "Nama Dokter" sama sekali, sehingga klaimnya harus
// mendarat di antrean belum-ditugaskan — bukan di tangan orang terakhir yang kebetulan
// tersimpan di suatu tempat.
func TestSendToRCLPUCLTanpaDokterTetapDiParkir(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToRCLPUCL(context.Background(),
		isiLengkap(task.ID, registrasi.PUCLTrackRCL), l.caller)
	require.NoError(t, err)

	require.Equal(t, registrasi.OperatorUnassigned, result.NextTask.Owner, "lini contoh bukan PA")
	require.Equal(t, registrasi.OperatorUnassigned, l.pucl.Letter[0].DoctorName)
}

// "Nama Dokter" tampil bila `RCL_PUCL != 2 && IsPA` — jadi jalur Notification pun
// membawanya, bukan jalur RCL saja. Diuji pada tingkat aturan karena lini polis contoh
// bukan PA: yang dipastikan di sini adalah jalur PUCL selalu membuangnya.
func TestSendToRCLPUCLJalurPUCLSelaluMembuangNamaDokter(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	isi := isiLengkap(task.ID, registrasi.PUCLTrackPUCL)
	isi.DoctorName = "dr. Contoh"
	_, err := l.service.SendToRCLPUCL(context.Background(), isi, l.caller)
	require.NoError(t, err)
	require.Empty(t, l.pucl.Letter[0].DoctorName)
}

// Notification tidak punya kelompok Perihal di master, sehingga SELURUH pilihan
// ditawarkan — bukan daftar kosong yang membuat analis tidak dapat memilih apa pun.
func TestPilihanPerihalNotificationMenawarkanSeluruhnya(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	semua, err := l.service.PUCLSubjectOptions(ctx, registrasi.PUCLTrackNotification)
	require.NoError(t, err)

	pucl, err := l.service.PUCLSubjectOptions(ctx, registrasi.PUCLTrackPUCL)
	require.NoError(t, err)
	rcl, err := l.service.PUCLSubjectOptions(ctx, registrasi.PUCLTrackRCL)
	require.NoError(t, err)

	require.Len(t, semua, len(pucl)+len(rcl))
}

// Isian yang melebihi panjang kolomnya ditolak di sini, bukan oleh Oracle: ORA-12899
// tidak dapat ditunjukkan ke pengguna sebagai pesan per isian.
func TestSendToRCLPUCLMenolakIsianMelebihiPanjangKolom(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	isi := isiLengkap(task.ID, registrasi.PUCLTrackPUCL)
	isi.BodyNote = strings.Repeat("x", registrasi.MaxPUCLNote+1)

	_, err := l.service.SendToRCLPUCL(context.Background(), isi, l.caller)

	var broken *registrasi.ValidationError
	require.ErrorAs(t, err, &broken)
	require.Equal(t, registrasi.ViolationNoteTooLong, broken.Violation[0].Code)
	require.Equal(t, "keterangan_isi", broken.Violation[0].Field)
}

// Pilihan Perihal menyempit menurut jalur: "Tolakan…" untuk RCL, "Kelengkapan…" untuk
// PUCL. Penyaringnya penyimpulan dari isi master — lihat rclpucl.sql.
func TestPilihanPerihalMenyempitMenurutJalur(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	pucl, err := l.service.PUCLSubjectOptions(ctx, registrasi.PUCLTrackPUCL)
	require.NoError(t, err)
	require.NotEmpty(t, pucl)
	for _, o := range pucl {
		require.Equal(t, registrasi.PUCLTrackPUCL, o.Track)
	}

	rcl, err := l.service.PUCLSubjectOptions(ctx, registrasi.PUCLTrackRCL)
	require.NoError(t, err)
	require.NotEmpty(t, rcl)
	require.NotEqual(t, len(pucl), len(rcl), "kedua jalur tidak menawarkan daftar yang sama")
	for _, o := range rcl {
		require.Equal(t, registrasi.PUCLTrackRCL, o.Track)
	}
}

// Grid alasan dicari di sumbernya, bukan di layar — tabelnya 2.116 baris.
func TestAlasanPenolakanDicariDiSumbernya(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	semua, err := l.service.PUCLRejectReasons(ctx, "", 0)
	require.NoError(t, err)
	require.Len(t, semua, 2)

	cocok, err := l.service.PUCLRejectReasons(ctx, "masa tunggu", 0)
	require.NoError(t, err)
	require.Len(t, cocok, 1)
	require.Equal(t, "002", cocok[0].ID)
	require.NotEmpty(t, cocok[0].Description, "Pilih menyalin deskripsinya ke Keterangan Isi")
}

// Dropdown "Nama Dokter" adalah `pyPromptTableList` property `NamaDokterRCL`, apa adanya.
//
// Yang diperiksa bukan sekadar jumlahnya, melainkan ketiga hal yang pernah salah:
//
//  1. URUTANNYA urutan prompt list (`REPEATINGINDEX` 1 lalu 2), bukan abjad. Diurutkan
//     abjad, MARGARETHA naik ke atas dan petugas memilih baris yang salah karena hafal
//     posisinya.
//  2. Nilai simpan dan label DIPISAH, dan pada baris kedua memang berbeda. Tertukar,
//     yang tersimpan adalah nama berspasi yang tidak pernah cocok dengan penyaring
//     Inbox RCL — dan klaimnya hilang dari semua inbox tanpa satu pesan galat.
//  3. Daftarnya TIDAK KOSONG tanpa basis data. Inilah cacat yang dilaporkan: kueri lama
//     ke `T_ACCESS_GROUP_PNC` tidak mengembalikan satu baris pun, dan layar menggambar
//     dropdown berisi "----- PILIH -----" saja.
func TestPilihanNamaDokterMengikutiPromptListPega(t *testing.T) {
	l := setup(t)

	pilihan := l.service.RCLDoctorOptions()
	require.Equal(t, []registrasi.RCLDoctorOption{
		{ID: "WAHYUKRISTANTI", Label: "WAHYUKRISTANTI"},
		{ID: "MARGARETHAROSAGUNAWAN", Label: "MARGARETHA ROSA GUNAWAN"},
	}, pilihan)
}

// Klaim yang dikirim ke RCL/PUCL dua kali tetap SATU surat.
//
// Primary key `TC_PNC_PUCL_PK` adalah `CLAIMID` tunggal — diperiksa ke katalog Oracle
// 2026-10-04, setelah sempat dikira (CLAIMID, TGL_CREATE_PUCL). Menyisipkan baris kedua
// akan ditolak ORA-00001, sehingga penyimpanannya wajib upsert.
func TestSuratRCLPUCLSatuBarisPerKlaim(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.NoError(t, err)
	require.Len(t, l.pucl.Letter, 1)

	// Klaim yang sama dikirim lagi — di produksi terjadi setelah ia kembali dari
	// RCL/PUCL. Yang tersimpan tetap satu baris, berisi isian terbaru.
	ulang := registrasi.PUCLLetter{ClaimID: result.Claim.ID, Track: registrasi.PUCLTrackRCL, BodyNote: "isi kedua"}
	require.NoError(t, l.pucl.SaveLetter(ctx, ulang))

	require.Len(t, l.pucl.Letter, 1)
	require.Equal(t, "isi kedua", l.pucl.Letter[0].BodyNote)
	require.Equal(t, registrasi.PUCLTrackRCL, l.pucl.Letter[0].Track)
}

// Surat membawa penunjuk adjustment: ID objek, nomor urut jaminan, nomor urut adjustment.
//
// `PUCLPost` menerimanya sebagai `idObj`, `idCov`, `idAdj` dan memakainya mencari baris
// yang distempel. Tanpa ketiganya, baris surat tidak menunjuk apa pun — dan kolomnya
// memang kosong sampai 2026-10-05.
func TestSuratRCLPUCLMembawaPenunjukAdjustment(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)

	claim, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal,
		Propose:     registrasi.Rupiah(10_000_000),
		Submitted:   registrasi.Rupiah(12_000_000),
		RiskType:    registrasi.RiskOfClaim,
		RiskPercent: 100_000,
	}), l.caller)
	require.NoError(t, err)

	_, err = l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.NoError(t, err)

	require.Len(t, l.pucl.Letter, 1)
	letter := l.pucl.Letter[0]
	require.Equal(t, claim.InsuredItem[0].ID, letter.ObjectID,
		"ID objek diambil dari .ObjectID, seperti ValidationAdjustment")
	require.Equal(t, 1, letter.CoverageIndex, "nomor urut jaminan — .pxListSubscript di Pega")
	require.Equal(t, 1, letter.AdjustmentIndex, "adjustment terakhir — baris yang <LAST> tunjuk")
}

// Klaim TANPA adjustment tetap menunjuk objek dan jaminannya.
//
// Tombol ini ada sejak tahap Choose Surveyor, jauh sebelum adjustment pertama
// ditambahkan — jadi inilah kasus yang paling sering terjadi, dan surat yang tidak
// menunjuk apa pun tidak dapat ditelusuri kembali ke pokok yang dibicarakannya.
//
// Yang tetap kosong hanya nomor urut adjustment: menulis angka di sana akan menunjuk
// baris yang tidak ada.
func TestSuratRCLPUCLTanpaAdjustmentTetapMenunjukObjekDanJaminan(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	_, err := l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.NoError(t, err)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)

	// Yang dipakai objek dan jaminan TERAKHIR klaim — kunci yang sama dengan
	// T_CLAIM_ADJUSTMENT: ID objek, lalu nomor urut jaminan di dalam objek itu.
	last := claim.InsuredItem[len(claim.InsuredItem)-1]
	require.NotEmpty(t, last.Coverage)

	require.Len(t, l.pucl.Letter, 1)
	letter := l.pucl.Letter[0]
	require.Equal(t, last.ID, letter.ObjectID)
	require.Equal(t, len(last.Coverage), letter.CoverageIndex)
	require.Zero(t, letter.AdjustmentIndex, "belum ada baris adjustment yang dapat ditunjuk")
}
