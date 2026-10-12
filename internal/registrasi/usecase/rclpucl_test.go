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
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	_, err := l.service.SendToRCLPUCL(context.Background(),
		isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.NoError(t, err)

	letter := l.pucl.Letter[0]
	require.True(t, letter.InRCLPUCLQueue(), "jalur PUCL masuk antrean RCL/PUCL")
	require.False(t, letter.EntersRCLInbox(), "jalur PUCL tidak menyentuh Inbox RCL")
	require.Empty(t, letter.DoctorName, "jalur PUCL tidak menulis Nama Dokter")
}

// Notification SINGGAH DI INBOX RCL lebih dulu, sama seperti RCL — dikoreksi 2026-10-07.
//
// # Uji ini mengunci koreksi, dan bentuk lamanya mengunci cacatnya
//
// Sebelumnya Notification diperlakukan seperti PUCL: langsung ke antrean RCL/PUCL. Tiga
// artefak export mematahkannya, dan ketiganya memisahkan **PUCL sendirian**:
//
//	Section/RCLDokter-Section.xml   layar dokter digambar untuk RCL_PUCL 1 DAN 3
//	Section/SectionPUCL-sect.xml    isian "Nama Dokter" tampil bila RCL_PUCL != 2
//	InboxRCLDokter_RD penyaring D   .ClaimData.NamaDokterRCL = Param.assign
//
// Tujuan AKHIRNYA tidak berubah: sesudah dokter menekan Submit, Notification tetap sampai
// ke antrean RCL/PUCL. Yang berubah hanya singgahannya.
func TestSendToRCLPUCLJalurNotificationSinggahDiInboxRCL(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	pic := claim.TechnicalPIC
	require.NotEmpty(t, pic)

	result, err := l.service.SendToRCLPUCL(ctx,
		isiLengkap(task.ID, registrasi.PUCLTrackNotification), l.caller)
	require.NoError(t, err)

	require.Equal(t, registrasi.StageRCLDoctor, result.Claim.CurrentStage,
		"Notification singgah di dokter, bukan langsung ke antrean RCL/PUCL")

	letter := l.pucl.Letter[0]
	require.False(t, letter.InRCLPUCLQueue(), "STATUS_CASE dibiarkan kosong — belum di antrean")
	require.True(t, letter.EntersRCLInbox(), "penyaring Inbox RCL ditulis")
	require.Equal(t, pic, letter.AssignedOperator())
	require.Equal(t, pic, result.NextTask.Owner)
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
	require.Equal(t, registrasi.StageRCLDoctor, result.Claim.CurrentStage)
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

// Dokter yang dipilih TERSIMPAN sebagai keterangan, tetapi yang MEMINDAHKAN klaim adalah
// PIC Teknik — keputusan Work Owner 2026-10-07.
//
// # Apa yang dikunci uji ini
//
// `RouterRCLDokter` di Pega berbunyi `Param.AssignTo := ClaimData.NamaDokterRCL`, dan nama
// itu tidak pernah dapat dicocokkan ke `M_LOGIN_PNC.LOGIN_ID` — penyaring satu-satunya
// Inbox RCL. Selama pertanyaan itu terbuka, tugasnya diparkir di `ServicePNC` dan
// `ASSIGNED_OPERATOR_ID` diisi analis, sehingga klaim RCL mendarat di Inbox RCL analis
// sendiri. Work Owner menutupnya dengan menetapkan **user teknis** sebagai pemiliknya.
//
// Pilihan dokternya tidak dibuang: ia tetap tersimpan sebagai keterangan.
func TestSendToRCLPUCLDipegangPICTeknikMeskiDokterDipilih(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	// Isian "Nama Dokter" hanya tampil pada lini PA.
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.Policy.Line = registrasi.LinePersonalAccident
	require.NoError(t, l.store.Save(ctx, claim))
	pic := claim.TechnicalPIC
	require.NotEmpty(t, pic, "klaim contoh sudah melewati tahap teknis")

	const dokter = "WAHYUKRISTANTI" // `pyPromptTableList` NamaDokterRCL, baris pertama
	isi := isiLengkap(task.ID, registrasi.PUCLTrackRCL)
	isi.DoctorName = dokter

	result, err := l.service.SendToRCLPUCL(ctx, isi, l.caller)
	require.NoError(t, err)

	require.Equal(t, registrasi.StageRCLDoctor, result.Claim.CurrentStage)

	// Pilihannya TERSIMPAN — analis tidak kehilangan apa yang ia isi.
	require.Equal(t, dokter, l.pucl.Letter[0].DoctorName)

	// Yang memegang klaimnya PIC Teknik, bukan analis dan bukan ServicePNC.
	require.Equal(t, pic, result.NextTask.Owner)
	require.Equal(t, pic, l.pucl.Letter[0].AssignedOperator(),
		"ASSIGNED_OPERATOR_ID dan pemilik tugas wajib menunjuk orang yang sama")
}

// Tanpa dokter yang dipilih — lini selain PA tidak menampilkan isiannya sama sekali —
// klaimnya tetap mendarat di PIC Teknik, dan kolom nama dokter menunjuk orang yang sama.
func TestSendToRCLPUCLTanpaDokterTetapKePICTeknik(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	pic := claim.TechnicalPIC
	require.NotEmpty(t, pic)

	result, err := l.service.SendToRCLPUCL(ctx,
		isiLengkap(task.ID, registrasi.PUCLTrackRCL), l.caller)
	require.NoError(t, err)

	require.Equal(t, pic, result.NextTask.Owner, "lini contoh bukan PA")
	require.Equal(t, pic, l.pucl.Letter[0].DoctorName)
	require.Equal(t, pic, l.pucl.Letter[0].AssignedOperator())
}

// Klaim yang belum punya PIC Teknik tidak boleh berakhir tanpa pemilik: `ASSIGNED_OPERATOR_ID`
// kosong membuatnya hilang dari Inbox RCL siapa pun, tanpa satu pun galat.
func TestSendToRCLPUCLTanpaPICTeknikJatuhKeAnalis(t *testing.T) {
	letter := registrasi.PUCLLetter{Operator: "ANALIS01", Track: registrasi.PUCLTrackRCL}
	require.Equal(t, "ANALIS01", letter.AssignedOperator())

	letter.TechnicalPIC = "  TEKNIK01  "
	require.Equal(t, "TEKNIK01", letter.AssignedOperator(), "PIC Teknik menang dan dipangkas")
}

// PEMILIK `ASSIGNED_OPERATOR_ID` BERGANTUNG PADA JALUR (Work Owner, 2026-10-07).
//
//	PUCL                 admin klaim — langsung masuk antrean RCL/PUCL
//	RCL dan Notification PIC Teknik  — singgah di Inbox RCL lebih dulu
//
// Ketiga identitas contoh sengaja BERBEDA satu sama lain. Nilai yang kembar akan membuat
// uji ini lulus pada aturan mana pun, sehingga ia tidak menguji apa pun.
func TestPemilikPenugasanSuratBergantungPadaJalur(t *testing.T) {
	dasar := registrasi.PUCLLetter{
		Operator: "ANALIS01", TechnicalPIC: "TEKNIK01", ClaimAdmin: "ADMINKLAIM01",
	}

	for _, uji := range []struct {
		nama  string
		track int
		mau   string
	}{
		{"PUCL memakai admin klaim", registrasi.PUCLTrackPUCL, "ADMINKLAIM01"},
		{"RCL memakai PIC Teknik", registrasi.PUCLTrackRCL, "TEKNIK01"},
		{"Notification memakai PIC Teknik", registrasi.PUCLTrackNotification, "TEKNIK01"},
	} {
		t.Run(uji.nama, func(t *testing.T) {
			letter := dasar
			letter.Track = uji.track
			require.Equal(t, uji.mau, letter.AssignedOperator())
		})
	}
}

// Rantai cadangan jalur PUCL: admin klaim, lalu PIC Teknik, lalu analis.
//
// Tidak satu pun boleh mengembalikan kosong — baris ber-`ASSIGNED_OPERATOR_ID` kosong
// tidak terbaca penyaring kepemilikan mana pun, dan klaimnya hilang tanpa satu pun galat.
func TestPemilikJalurPUCLPunyaRantaiCadangan(t *testing.T) {
	letter := registrasi.PUCLLetter{Track: registrasi.PUCLTrackPUCL, Operator: "ANALIS01"}
	require.Equal(t, "ANALIS01", letter.AssignedOperator(), "tanpa admin dan tanpa PIC")

	letter.TechnicalPIC = "TEKNIK01"
	require.Equal(t, "TEKNIK01", letter.AssignedOperator(), "tanpa admin, PIC yang dipakai")

	letter.ClaimAdmin = "  ADMINKLAIM01  "
	require.Equal(t, "ADMINKLAIM01", letter.AssignedOperator(), "admin menang dan dipangkas")
}

// NAMA DOKTER BERTAHAN PADA JALUR RCL **DAN** NOTIFICATION, bukan RCL saja.
//
// Isiannya tampil bila `RCL_PUCL != 2 && IsPA` (`SectionPUCL-sect.xml:2149`), dan layar
// memang menggambarnya pada kedua jalur (`showDoctorName`). Syarat di `buildPUCLLetter`
// sempat berbunyi `Track == RCL`, sehingga apa yang diisi analis pada jalur Notification
// DIBUANG diam-diam — separuh gejala yang dilaporkan Work Owner 2026-10-07.
func TestNamaDokterBertahanPadaJalurRCLDanNotification(t *testing.T) {
	for _, track := range []int{registrasi.PUCLTrackRCL, registrasi.PUCLTrackNotification} {
		l := setup(t)
		ctx := context.Background()
		task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

		claim, err := l.store.Get(ctx, task.ClaimID)
		require.NoError(t, err)
		claim.Policy.Line = registrasi.LinePersonalAccident
		require.NoError(t, l.store.Save(ctx, claim))

		isi := isiLengkap(task.ID, track)
		isi.DoctorName = "WAHYUKRISTANTI"

		_, err = l.service.SendToRCLPUCL(ctx, isi, l.caller)
		require.NoError(t, err)

		require.Equalf(t, "WAHYUKRISTANTI", l.pucl.Letter[0].DoctorName,
			"jalur %d menampilkan isian Nama Dokter, jadi isiannya tidak boleh dibuang", track)
	}
}

// Jalur PUCL tetap MEMBUANGNYA — layar tidak menampilkan isiannya di sana.
func TestNamaDokterDibuangPadaJalurPUCLWalauLiniPA(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.Policy.Line = registrasi.LinePersonalAccident
	require.NoError(t, l.store.Save(ctx, claim))

	isi := isiLengkap(task.ID, registrasi.PUCLTrackPUCL)
	isi.DoctorName = "WAHYUKRISTANTI"

	_, err = l.service.SendToRCLPUCL(ctx, isi, l.caller)
	require.NoError(t, err)
	require.Empty(t, l.pucl.Letter[0].DoctorName)
}

// Jalur PUCL dari ujung ke ujung: surat yang tersimpan membawa admin klaim.
func TestSendToRCLPUCLJalurPUCLMenugaskanKeAdminKlaim(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	admin := claim.CreatedBy
	require.NotEmpty(t, admin, "klaim contoh wajib punya admin, kalau tidak uji ini tidak membuktikan apa pun")

	_, err = l.service.SendToRCLPUCL(ctx, isiLengkap(task.ID, registrasi.PUCLTrackPUCL), l.caller)
	require.NoError(t, err)

	letter := l.pucl.Letter[0]
	require.Equal(t, admin, letter.AssignedOperator(),
		"jalur PUCL menugaskan ke admin klaim, bukan ke analis maupun PIC Teknik")
	require.Equal(t, l.caller.Identity, letter.Operator,
		"OPERATOR_ID tetap mencatat analis yang menekan Kirim")
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

// BATAS "NAMA DOKTER" DIJAGA DALAM DUA SATUAN — karakter dan byte.
//
// Nilainya ditulis ke DUA kolom yang satuannya berbeda:
//
//	T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1   VARCHAR2(128 CHAR)
//	TC_PNC_PUCL.NAMA_DOKTER_RCL         VARCHAR2(255 BYTE)
//
// Memeriksa satu saja meninggalkan celah yang hanya terpicu pada nama tertentu: 128 aksara
// non-ASCII memenuhi batas karakter tetapi menjadi 384 byte dan menembus kolom kedua.
// Keduanya karena itu diuji terpisah, dan keduanya pada jalur RCL lini PA — satu-satunya
// jalur yang tidak mengosongkan isian ini.
func TestNamaDokterDibatasiKarakterDanByte(t *testing.T) {
	jalankan := func(t *testing.T, dokter string) *registrasi.ValidationError {
		t.Helper()
		l := setup(t)
		ctx := context.Background()
		task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

		claim, err := l.store.Get(ctx, task.ClaimID)
		require.NoError(t, err)
		claim.Policy.Line = registrasi.LinePersonalAccident
		require.NoError(t, l.store.Save(ctx, claim))

		isi := isiLengkap(task.ID, registrasi.PUCLTrackRCL)
		isi.DoctorName = dokter

		_, err = l.service.SendToRCLPUCL(ctx, isi, l.caller)
		var broken *registrasi.ValidationError
		require.ErrorAs(t, err, &broken)
		return broken
	}

	t.Run("melebihi batas karakter", func(t *testing.T) {
		broken := jalankan(t, strings.Repeat("x", registrasi.MaxPUCLDoctorName+1))
		require.Equal(t, "nama_dokter", broken.Violation[0].Field)
		require.Equal(t, registrasi.ViolationNoteTooLong, broken.Violation[0].Code)
	})

	// 100 aksara — di bawah batas 128 karakter — tetapi 300 byte pada UTF-8, sehingga
	// menembus kolom 255 byte. Inilah kasus yang lolos bila hanya karakternya diperiksa.
	//
	// Dipakai aksara 3-byte, bukan 2-byte: 100 aksara beraksen hanya 200 byte dan TIDAK
	// menembus apa pun — fixture pertama saya persis begitu, dan kedua `require` di bawah
	// yang menangkapnya. Contoh yang tidak memenuhi premisnya sendiri akan "lulus" dengan
	// alasan yang salah.
	t.Run("memenuhi batas karakter tetapi menembus batas byte", func(t *testing.T) {
		nama := strings.Repeat("あ", 100) // 3 byte per aksara -> 300 byte
		require.LessOrEqual(t, len([]rune(nama)), registrasi.MaxPUCLDoctorName)
		require.Greater(t, len(nama), registrasi.MaxPUCLDoctorNameBytes)

		broken := jalankan(t, nama)
		require.Equal(t, "nama_dokter", broken.Violation[0].Field)
	})
}

// Batas aplikasi tidak boleh LEBIH LONGGAR daripada kolom yang menampungnya.
//
// Patokan 512 pernah berlaku di sini — lebih longgar daripada kedua kolomnya — sehingga
// isian yang lolos validasi ditolak Oracle dengan ORA-12899, galat yang tidak dapat
// ditunjukkan kepada pengguna sebagai pesan per isian.
func TestBatasNamaDokterTidakLebihLonggarDariKolomnya(t *testing.T) {
	require.LessOrEqual(t, registrasi.MaxPUCLDoctorName, 128,
		"T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1 adalah VARCHAR2(128 CHAR)")
	require.LessOrEqual(t, registrasi.MaxPUCLDoctorNameBytes, 255,
		"TC_PNC_PUCL.NAMA_DOKTER_RCL adalah VARCHAR2(255 BYTE)")
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
