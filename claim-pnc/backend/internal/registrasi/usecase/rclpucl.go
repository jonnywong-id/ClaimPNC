package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ActionSendToRCLPUCL adalah nama local action tombol "Kirim ke RCL/PUCL"
// (`Flow Action/KomentarRCLPUCL-FA.xml`). Dicatat sebagai tindakan penutup tugas.
const ActionSendToRCLPUCL = "KomentarRCLPUCL"

// sendToRCLPUCLStages adalah tahap yang layarnya `ClaimSurvey_sect` — tempat tombol
// "Kirim ke RCL/PUCL" berada. Sama persis dengan daftar "Kirim ke Inputor": keduanya
// tombol pada baris yang sama di section itu.
var sendToRCLPUCLStages = map[string]bool{
	registrasi.StageChooseSurveyor:     true,
	registrasi.StageSendToTechnicalPIC: true,
	registrasi.StageEstimatePA:         true,
	registrasi.StageInvestigator:       true,
	registrasi.StageSendToAnalyst:      true,
}

// SendToRCLPUCLCommand adalah isi modal `SectionPUCL`, dalam urutan isiannya di layar.
type SendToRCLPUCLCommand struct {
	TaskID string

	// Track adalah "Pilih RCL / PUCL" — radio tiga pilihan, `pyRequired=true`.
	Track int

	// AnalystNote adalah "Catatan untuk RCL/PUCL" — `pyRequired=true`.
	AnalystNote string

	// Subject adalah "Perihal" — `pyRequired=false`.
	Subject string

	// OpeningNote dan ClosingNote `pyRequired=false`; BodyNote ("Keterangan Isi")
	// `pyRequired=true`.
	OpeningNote string
	BodyNote    string
	ClosingNote string

	// DoctorName adalah "Nama Dokter" — isiannya hanya tampil bila
	// `.ClaimData.PUCLStatus.RCL_PUCL != 2 && IsPA`.
	DoctorName string
}

// TicketSendToRCLPUCL memetakan jalur yang dipilih ke Ticket rule tujuannya.
//
// # Kenapa lompatan lateral, bukan keputusan alur
//
// Tombol ini ada di LIMA tahap (`ClaimSurvey_sect`), dan hanya satu di antaranya —
// Send To Analis — yang keputusan sesudahnya memang memilih RCL/PUCL. Menutup tahap
// lewat jalur penyelesaian biasa karena itu SALAH di empat tahap lainnya: dari Choose
// Surveyor, misalnya, alur justru berakhir (`EndAfterSurveyor`) sehingga klaim tertutup
// alih-alih terkirim.
//
// Pega memakai `SetTicket` — lompatan lateral yang menempel pada shape tujuan, tidak
// peduli klaim sedang di mana. Itulah yang ditiru di sini, sama seperti "Kirim ke
// Inputor" (`setToRegister_ticket`).
//
// # Nama ticketnya DISIMPULKAN, dan begini dasarnya
//
// Parameter `call SetTicket` pada `Activity/PUCLPost-Act.xml:5373` ber-`pyParamArray`
// KOSONG, sehingga nama ticket yang dikirimnya tidak terbaca dari export. Yang terbaca
// adalah nama ticket yang MENEMPEL pada kedua tahap tujuan — `SendtoPUCL` pada
// Assignment6 dan `RCLDokter` pada Assignment12.
//
//	PUCL          -> SendtoPUCL   antrean bersama RCL/PUCL
//	RCL           -> RCLDokter    Assignment12
//	Notification  -> SendtoPUCL   antrean bersama RCL/PUCL
//
// **RCL** menuju RCLDokter karena tiga hal sejalan: cabang Otherwise `Decision7` memang
// Assignment12, nama ticket tahap itu persis "RCLDokter", dan isian "Nama Dokter" modal
// ini tersimpan di `T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1` — kolom yang dipakai MENYARING
// layar Inbox RCL (`MENU_ID 62`), inbox milik dokter itu.
//
// **Notification** menuju antrean RCL/PUCL, dan ini penyimpulan yang paling lemah di
// berkas ini. Dasarnya: alur Register tidak punya tahap bernama Notification, kode `3`
// tidak menyentuh dokter mana pun, dan seluruh penanganannya hidup DI DALAM layar Inbox
// RCL/PUCL — `inboxrclpucl` memodelkannya sebagai `TrackCodeNotification`, kuerinya
// tidak menyaring jalur sama sekali, dan lampiran suratnya diatur
// `Section/SectionLampiranSuratPUCL-Section.xml` dengan kondisi
// `RCL_PUCL = 3 && MSIG = 'MSIG'`.
//
// Yang menjadi ketegangannya: bila klaim sampai ke keputusan `Decision7` lewat tombol
// Submit biasa, kode `3` jatuh ke cabang Otherwise — RCLDokter, bukan RCL/PUCL. Kedua
// jalur itu memang berbeda di Pega, dan yang ditiru di sini adalah jalur tombolnya.
func TicketSendToRCLPUCL(track int) (string, bool) {
	switch track {
	case registrasi.PUCLTrackPUCL, registrasi.PUCLTrackNotification:
		return "SendtoPUCL", true
	case registrasi.PUCLTrackRCL:
		return "RCLDokter", true
	}
	return "", false
}

// SendToRCLPUCL menjalankan tombol **Kirim** pada modal "Kirim ke RCL/PUCL".
//
// Urutannya meniru `PUCLPost`: simpan isian surat, setel penanda dan Status Klaim, lalu
// lompat ke tahap tujuan.
//
//	jalur PUCL          ->  RCL/PUCL   (Assignment6, antrean bersama RCLPUCL)
//	jalur RCL           ->  RCLDokter  (Assignment12)
//	jalur Notification  ->  RCL/PUCL   (Assignment6)
//
// # Yang ditulis
//
//	POOLDATA.TC_PNC_PUCL   satu baris surat — isian modal apa adanya, untuk ketiga jalur
//	POOLDATA.T_CLAIMLIST_ADMIN  Nama Dokter dan tanggal kirim — HANYA jalur RCL
//	klaim                  PUCLStatus (RCLPUCL) dan Status Klaim 1151
//	M_KOMUNIKASI_PNC       catatan analis, supaya terbaca di tab Progress & Komunikasi
//
// # Antrean mana yang menerimanya
//
// Jalur RCL singgah di **Inbox RCL** lebih dulu; ia TIDAK ikut muncul di Inbox RCL/PUCL.
// Keduanya ditegakkan di tempat yang berbeda, dan keduanya perlu:
//
//	tahap tujuan   TicketSendToRCLPUCL       RCL -> RCLDokter, dua lainnya -> RCL/PUCL
//	Inbox RCL      PUCLLetter.EntersRCLInbox NAMADOKTERRCL_1 + TANGGALANALYSTSENDRCL_1
//	Inbox RCL/PUCL PUCLLetter.InRCLPUCLQueue STATUS_CASE dibiarkan kosong pada jalur RCL
//
// `TGL_CETAK_DOKUMEN_PUCL` sengaja dibiarkan kosong — ia milik tombol "Download
// Dokumen" pada layar Inbox RCL/PUCL, yang memindahkan baris ke tab berikutnya.
func (l *Service) SendToRCLPUCL(ctx context.Context, p SendToRCLPUCLCommand, by Caller) (CompleteResult, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID})
	if err != nil {
		return CompleteResult{}, err
	}
	if !sendToRCLPUCLStages[claim.CurrentStage] {
		return CompleteResult{}, fmt.Errorf("%w: Kirim ke RCL/PUCL tidak tersedia pada tahap %q",
			registrasi.ErrInvalidAction, claim.CurrentStage)
	}

	letter, err := buildPUCLLetter(p, claim)
	if err != nil {
		return CompleteResult{}, err
	}

	ticket, ok := TicketSendToRCLPUCL(p.Track)
	if !ok {
		// Tidak tercapai: buildPUCLLetter sudah menolak jalur yang tidak dikenal.
		return CompleteResult{}, fmt.Errorf("%w: jalur RCL/PUCL %d", registrasi.ErrInvalidAction, p.Track)
	}
	target, ok := l.flow.StageByTicket(ticket)
	if !ok {
		return CompleteResult{}, fmt.Errorf("%w: tujuan %q", registrasi.ErrUnknownStage, ticket)
	}

	// Penanda jalur dan Status Klaim `1151` Analyst — `PUCLPost` langkah 15,
	// berprekondisi `param.Status == 1`. Konstanta yang sama dipakai tombol
	// "Transfer ke Analyst" (analyst.go).
	claim.PUCLStatus = p.Track
	claim.ClaimStatus = StatusClaimAnalyst

	now := l.clock.Now().UTC()
	letter.SentAt = now
	letter.Operator = by.Identity
	if err := task.Complete(by.Identity, ActionSendToRCLPUCL, now); err != nil {
		return CompleteResult{}, err
	}
	recipients, err := l.assigner.Assign(ctx, target, claim, by.Identity)
	if err != nil {
		return CompleteResult{}, fmt.Errorf("registrasi/usecase: menentukan penerima tahap %q: %w", target.ID, err)
	}
	fresh := registrasi.NewTask(l.id.New(), claim, target, recipients, now)

	// Jalur RCL harus benar-benar TERLIHAT di Inbox RCL, dan layar itu menyaring dengan
	// `NAMADOKTERRCL_1`. Isian "Nama Dokter" sendiri hanya tampil pada lini PA, sehingga
	// pada lini lain ia selalu kosong — dan surat yang menulis kolom itu kosong akan
	// membuat klaimnya tidak muncul di inbox siapa pun.
	//
	// Yang diisi karena itu PEMILIK tugas RCLDokter, orang yang sama dengan yang ditulis
	// ke `PXASSIGNEDOPERATORID` oleh mirrorInbox. Dengan begitu kedua penyaring layar itu
	// menunjuk orang yang sama, bukan saling meniadakan.
	if letter.EntersRCLInbox() && letter.DoctorName == "" {
		letter.DoctorName = recipients.Operator
	}

	from := task.Stage
	claim.CurrentStage = target.ID
	claim.RequestReturn = false
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if err := l.task.Save(ctx, fresh); err != nil {
			return err
		}
		if err := l.puclLetters.SaveLetter(ctx, letter); err != nil {
			return fmt.Errorf("registrasi/usecase: menyimpan surat RCL/PUCL: %w", err)
		}
		if err := l.records.AddCommunication(ctx, registrasi.NewCommunication{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Sender:      by.Identity,
			SenderName:  by.Name,
			Message:     letter.AnalystNote,
			Recipient:   recipients.Operator,
			Channel:     registrasi.ChannelSendToRCLPUCL,
			Status:      registrasi.CommunicationStatusOpen,
			At:          now,
		}); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       "KIRIM_KE_RCLPUCL",
			Actor:       by.Identity,
			At:          now,
			Note:        from + " → " + target.ID + " (" + registrasi.PUCLTrackName(p.Track) + ")",
		})
	})
	if err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{Claim: claim, NextTask: &fresh}, nil
}

// PUCLSubjectOptions melayani pilihan "Perihal" modal, disaring menurut jalur.
func (l *Service) PUCLSubjectOptions(ctx context.Context, track int) ([]registrasi.PUCLSubjectOption, error) {
	return l.puclOptions.SubjectOptions(ctx, track)
}

// PUCLRejectReasons melayani grid alasan penolakan modal.
func (l *Service) PUCLRejectReasons(ctx context.Context, keyword string, limit int) ([]registrasi.PUCLRejectReason, error) {
	return l.puclOptions.RejectReasons(ctx, keyword, limit)
}

// RCLDoctorOptions melayani dropdown "Nama Dokter" modal.
//
// Isiannya hanya tampil pada jalur RCL atau Notification lini PA — penyaring itu dipegang
// layar, bukan di sini, karena yang menentukannya adalah Group Panel klaim yang sedang
// dibuka. Yang dijaga di sini adalah ISI daftarnya: ia selalu himpunan nilai yang dapat
// dicocokkan penyaring Inbox RCL, apa pun jalurnya.
func (l *Service) RCLDoctorOptions(ctx context.Context) ([]registrasi.RCLDoctorOption, error) {
	return l.puclOptions.RCLDoctors(ctx)
}

// jejak merangkai keputusan yang dilewati menjadi satu baris jejak audit. Tanpa ini,
// klaim yang berakhir di RCLDokter alih-alih RCL/PUCL tidak dapat dijelaskan kepada
// siapa pun — dan cabang pertama keputusan itu menguji PERAN, bukan data klaim.
func jejak(trace []string) string {
	if len(trace) == 0 {
		return ""
	}
	return " [" + strings.Join(trace, "; ") + "]"
}

// lastAdjustmentOf menunjuk adjustment TERAKHIR klaim: ID objeknya, nomor urut jaminannya,
// dan nomor urut adjustment itu sendiri. Ketiganya berbasis 1; klaim tanpa adjustment
// mengembalikan nilai kosong.
//
// # Kenapa yang TERAKHIR
//
// Karena itulah baris yang distempel `PUCLPost`: seluruh langkahnya menulis ke
// `AdjustmentList(<LAST>)`, bukan ke adjustment yang ditunjuk `idAdj`. Di Pega nilainya
// datang dari `ValidationAdjustment` — yang berjalan saat adjustment ditambahkan, sehingga
// yang tercatat memang adjustment yang baru saja dikerjakan petugas.
//
// Aplikasi ini tidak menyimpan "adjustment yang terakhir dikerjakan" di mana pun, dan
// `T_CLAIM_PNC` tidak punya kolom untuknya. Menurunkannya dari isi klaim pada saat surat
// dibuat memberi baris yang SAMA tanpa menuntut kolom baru — dan bila kelak penandanya
// benar-benar disimpan, yang berubah hanya fungsi ini.
//
// # Klaim tanpa adjustment tetap menunjuk objek dan jaminannya
//
// Tombol "Kirim ke RCL/PUCL" ada sejak tahap Choose Surveyor — jauh sebelum adjustment
// pertama ditambahkan. Berhenti pada `len(Settlement) == 0` membuat ketiga kolom `ID_*`
// kosong pada justru kasus yang paling sering terjadi, dan surat yang tidak menunjuk apa
// pun tidak dapat ditelusuri kembali ke objek yang dibicarakannya.
//
// Karena itu jaminan yang PUNYA adjustment selalu menang; bila tidak ada satu pun, yang
// dipakai objek dan jaminan terakhir klaim. `adjustment` tetap nol di kasus itu — menulis
// `1` akan menunjuk baris yang tidak ada.
func lastAdjustmentOf(claim registrasi.Claim) (objectID string, coverage, adjustment int) {
	for _, item := range claim.InsuredItem {
		for position, cover := range item.Coverage {
			if len(cover.Settlement) > 0 {
				objectID = item.ID
				coverage = position + 1
				adjustment = len(cover.Settlement)
				continue
			}
			// Cadangan, dan hanya selama belum ada jaminan ber-adjustment: yang sudah
			// ditemukan tidak boleh tergeser oleh jaminan kosong sesudahnya.
			if adjustment == 0 {
				objectID = item.ID
				coverage = position + 1
			}
		}
	}
	return objectID, coverage, adjustment
}

// buildPUCLLetter memvalidasi isian modal lalu menyusun barisnya.
//
// Kewajiban isian mengikuti `pyRequired` pada `Section/SectionPUCL-sect.xml` apa adanya:
// jalur, Catatan untuk RCL/PUCL, dan Keterangan Isi. Perihal serta kedua keterangan lain
// TIDAK diwajibkan di sana, dan tidak diwajibkan di sini.
//
// Seluruh pelanggaran dikumpulkan sekaligus, bukan dihentikan pada yang pertama — form
// ini panjang, dan menolaknya satu per satu akan menyiksa pengguna (Cross-Cutting §1.2).
func buildPUCLLetter(p SendToRCLPUCLCommand, claim registrasi.Claim) (registrasi.PUCLLetter, error) {
	letter := registrasi.PUCLLetter{
		ClaimID:      claim.ID,
		Track:        p.Track,
		AnalystNote:  strings.TrimSpace(p.AnalystNote),
		Subject:      strings.TrimSpace(p.Subject),
		OpeningNote:  strings.TrimSpace(p.OpeningNote),
		BodyNote:     strings.TrimSpace(p.BodyNote),
		ClosingNote:  strings.TrimSpace(p.ClosingNote),
		DoctorName:   strings.TrimSpace(p.DoctorName),
		PolicyNumber: claim.Policy.Number,
		InsuredName:  claim.Policy.InsuredName,
		BusinessName: claim.Policy.BusinessName,
		BranchName:   claim.Policy.BranchName,
		SourceName:   claim.Policy.SourceOfBusinessName,
		GroupPanel:   string(claim.Policy.Line),
		TechnicalPIC: claim.TechnicalPIC,
		ClaimStatus:  string(StatusClaimAnalyst),
		DateOfLoss:   claim.DateOfLoss,
	}
	letter.ObjectID, letter.CoverageIndex, letter.AdjustmentIndex = lastAdjustmentOf(claim)

	var broken []registrasi.Violation
	if _, known := TicketSendToRCLPUCL(p.Track); !known {
		broken = append(broken, registrasi.Violation{
			Code: registrasi.ViolationPUCLTrackUnknown, Field: "jalur",
			Message: "Pilih RCL, PUCL, atau Notification lebih dulu.",
		})
	}
	if letter.AnalystNote == "" {
		broken = append(broken, registrasi.Violation{
			Code: registrasi.ViolationPUCLNoteEmpty, Field: "catatan",
			Message: "Catatan untuk RCL/PUCL wajib diisi.",
		})
	}
	if letter.BodyNote == "" {
		broken = append(broken, registrasi.Violation{
			Code: registrasi.ViolationPUCLBodyEmpty, Field: "keterangan_isi",
			Message: "Keterangan Isi wajib diisi.",
		})
	}

	// Batas panjang kolom. Dinyatakan di sini supaya penolakannya berupa pesan per
	// isian, bukan ORA-12899 yang tidak dapat ditunjukkan ke pengguna.
	type limited struct {
		value string
		max   int
		field string
		label string
	}
	for _, it := range []limited{
		{letter.AnalystNote, registrasi.MaxPUCLAnalystNote, "catatan", "Catatan untuk RCL/PUCL"},
		{letter.Subject, registrasi.MaxPUCLSubject, "perihal", "Perihal"},
		{letter.OpeningNote, registrasi.MaxPUCLNote, "keterangan_pembuka", "Keterangan Pembuka"},
		{letter.BodyNote, registrasi.MaxPUCLNote, "keterangan_isi", "Keterangan Isi"},
		{letter.ClosingNote, registrasi.MaxPUCLNote, "keterangan_penutup", "Keterangan Penutup"},
		{letter.DoctorName, registrasi.MaxPUCLDoctorName, "nama_dokter", "Nama Dokter"},
	} {
		if len([]rune(it.value)) > it.max {
			broken = append(broken, registrasi.Violation{
				Code: registrasi.ViolationNoteTooLong, Field: it.field,
				Message: fmt.Sprintf("%s paling banyak %d karakter.", it.label, it.max),
			})
		}
	}

	if len(broken) > 0 {
		return registrasi.PUCLLetter{}, &registrasi.ValidationError{Violation: broken}
	}

	// "Nama Dokter" hanya tampil pada jalur RCL lini PA. Isian yang datang di luar itu
	// DIBUANG, bukan ditolak: layar tidak menampilkannya, sehingga nilainya tidak
	// pernah datang dari pengguna yang sadar mengisinya.
	if !(p.Track == registrasi.PUCLTrackRCL && claim.Policy.Line == registrasi.LinePersonalAccident) {
		letter.DoctorName = ""
	}
	return letter, nil
}
