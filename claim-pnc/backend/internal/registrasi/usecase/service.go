// Package usecase mengorkestrasi modul Registrasi Klaim.
//
// Di sinilah urutan pekerjaan ditentukan: ambil polis, validasi, terbitkan nomor,
// pindahkan klaim ke tahap berikutnya, terbitkan pemberitahuan, rekam jejak audit. Tidak
// ada aturan bisnis baru di sini — aturannya milik paket registrasi; yang ada di sini
// adalah URUTAN dan BATAS TRANSAKSI.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// Caller adalah identitas orang yang sedang menekan tombol.
//
// Perannya ikut dibawa karena satu keputusan alur benar-benar bergantung padanya —
// lihat catatan pada `DecisionAfterAnalyst`.
type Caller struct {
	// Identity adalah pengenal operator: NIK untuk karyawan, LOGIN_ID untuk
	// non-karyawan (`ADR-0024`).
	Identity string
	Name     string

	Roles []string

	// Workbasket menyebut antrean bersama yang boleh dilihat pemanggil. Daftarnya
	// kelak datang dari tabel peran `TKT-F3-004`; sampai itu ada, ia diisi pemanggil.
	Workbasket []string
}

// Service adalah muka modul registrasi bagi lapisan transport.
type Service struct {
	flow registrasi.Definition

	claim       registrasi.ClaimRepo
	task        registrasi.TaskRepo
	policy      registrasi.PolicyRepo
	number      registrasi.NumberIssuer
	parameter   registrasi.Parameter
	rate        registrasi.ExchangeRateSource
	assigner    registrasi.Assigner
	notifier    registrasi.Notifier
	audit       registrasi.AuditRecorder
	reportLink  registrasi.ClaimReportLink
	area        registrasi.AreaDirectory
	items       registrasi.PolicyItemSource
	currency    registrasi.CurrencyDirectory
	options     registrasi.ItemOptionSource
	records     registrasi.ClaimRecordSource
	faceSheet   registrasi.FaceSheetSource
	renderer    registrasi.FaceSheetRenderer
	pla         registrasi.PLASource
	plaRenderer registrasi.PLARenderer
	groups      registrasi.GroupSource
	inbox       registrasi.InboxMirror
	accounts    registrasi.AccountDirectory
	tiering     registrasi.CommitteeTiering
	committees  registrasi.CommitteeStore
	id          registrasi.IDGenerator
	unit        registrasi.UnitOfWork
	clock       clock.Clock

	validateOnReturn bool
}

// Options adalah bahan pembentuk Layanan. Seluruhnya wajib.
type Options struct {
	ClaimRepo          registrasi.ClaimRepo
	TaskRepo           registrasi.TaskRepo
	PolicyRepo         registrasi.PolicyRepo
	NumberIssuer       registrasi.NumberIssuer
	Parameter          registrasi.Parameter
	ExchangeRateSource registrasi.ExchangeRateSource
	Assigner           registrasi.Assigner
	Notifier           registrasi.Notifier
	AuditRecorder      registrasi.AuditRecorder
	ClaimReportLink    registrasi.ClaimReportLink
	AreaDirectory      registrasi.AreaDirectory

	// PolicyItems membaca objek, coverage, dan spreading polis untuk klaim yang baru
	// dibuka (CallActivityInputRegister).
	PolicyItems registrasi.PolicyItemSource

	// CurrencyDirectory membaca pilihan Mata Uang tahap Input Estimasi.
	CurrencyDirectory registrasi.CurrencyDirectory

	// ItemOptions membaca pilihan Objek item estimasi dari polis.
	ItemOptions registrasi.ItemOptionSource

	// ClaimRecords membaca tab Survey, Unggah Dokumen, dan Progress Claim & Komunikasi.
	ClaimRecords registrasi.ClaimRecordSource

	// FaceSheet membaca data pendamping Claim Face Sheet dan menyimpan revisinya;
	// FaceSheetRenderer membentuk dokumennya.
	FaceSheet         registrasi.FaceSheetSource
	FaceSheetRenderer registrasi.FaceSheetRenderer

	// PLA menerbitkan dan membaca PLA koasuransi; PLARenderer membentuk dokumennya.
	PLA         registrasi.PLASource
	PLARenderer registrasi.PLARenderer

	// Groups membaca keanggotaan grup pengguna (POOLDATA.M_LOGIN_GROUP_PNC) — penentu
	// peran dan tahap yang boleh dikerjakannya (access.go).
	Groups registrasi.GroupSource

	// Inbox menulis baris daftar kerja My Inbox (POOLDATA.T_CLAIMLIST_ADMIN) setiap kali
	// klaim atau tugasnya berubah — lihat registrasi.InboxEntry.
	Inbox registrasi.InboxMirror

	// Accounts membaca Master Rekening untuk isian No Rekening penerima klaim.
	Accounts registrasi.AccountDirectory

	// CommitteeTiering menghitung penyetuju komite (modul komite); Committees menyimpan
	// kasus komite di POOLDATA.T_CLAIM_KOMITE_LIST.
	CommitteeTiering registrasi.CommitteeTiering
	Committees       registrasi.CommitteeStore

	IDGenerator registrasi.IDGenerator
	UnitOfWork  registrasi.UnitOfWork
	Clock       clock.Clock

	// ValidateOnReturn menentukan apakah tombol Back ikut melewati gerbang validasi.
	//
	// # Kenapa ini sebuah sakelar, dan kenapa bawaannya mati
	//
	// Pada `Activity/InputRegister_act-Act.xml` langkah 7 ada prasyarat
	// `.pyNote=="Back"` yang bertindak `exit activity`, dengan keterangan penulisnya
	// sendiri: *"exit activity when back and set local value"*. Bila prasyarat itu
	// berlaku, menekan Back melewati seluruh validasi.
	//
	// Tetapi `pyStepsPreCondition` pada langkah itu bernilai `false` — dan pembacaan
	// kami atas nilai itu adalah "prasyarat tertulis tetapi sedang DIMATIKAN". Bila
	// pembacaan itu benar, Pega yang berjalan hari ini MENJALANKAN seluruh validasi
	// saat Back, sehingga petugas tidak dapat mundur justru ketika datanya belum benar.
	//
	// Kami tidak menebak. Bawaannya `false` — mengikuti maksud yang tertulis di
	// keterangan langkahnya — dan perilaku sebaliknya tersedia dengan mengubah satu
	// nilai ini. Pertanyaannya diajukan ke Work Owner; lihat catatan pengembangan.
	ValidateOnReturn bool
}

// NewService membentuk layanan dan menolak bila ada seam yang belum terpasang.
//
// Penolakannya terjadi saat perakitan, bukan saat permintaan pertama datang: aplikasi
// yang setengah terpasang lebih berbahaya daripada aplikasi yang tidak start.
func NewService(o Options) (*Service, error) {
	missing := []string{}
	check := func(name string, mounted bool) {
		if !mounted {
			missing = append(missing, name)
		}
	}
	check("KlaimRepo", o.ClaimRepo != nil)
	check("TugasRepo", o.TaskRepo != nil)
	check("PolisRepo", o.PolicyRepo != nil)
	check("PenerbitNomor", o.NumberIssuer != nil)
	check("Parameter", o.Parameter != nil)
	check("SumberKurs", o.ExchangeRateSource != nil)
	check("Penugasan", o.Assigner != nil)
	check("Notifier", o.Notifier != nil)
	check("PerekamAudit", o.AuditRecorder != nil)
	check("TautanLaporan", o.ClaimReportLink != nil)
	check("DirektoriWilayah", o.AreaDirectory != nil)
	check("ObjekPolis", o.PolicyItems != nil)
	check("DirektoriMataUang", o.CurrencyDirectory != nil)
	check("PilihanItem", o.ItemOptions != nil)
	check("CatatanKlaim", o.ClaimRecords != nil)
	check("ClaimFaceSheet", o.FaceSheet != nil)
	check("PembentukFaceSheet", o.FaceSheetRenderer != nil)
	check("PLA", o.PLA != nil)
	check("PembentukPLA", o.PLARenderer != nil)
	check("GrupPengguna", o.Groups != nil)
	check("DaftarKerja", o.Inbox != nil)
	check("MasterRekening", o.Accounts != nil)
	check("PenjenjanganKomite", o.CommitteeTiering != nil)
	check("KasusKomite", o.Committees != nil)
	check("PembuatID", o.IDGenerator != nil)
	check("UnitKerja", o.UnitOfWork != nil)
	check("Jam", o.Clock != nil)

	if len(missing) > 0 {
		return nil, fmt.Errorf("registrasi/usecase: seam belum terpasang: %v", missing)
	}

	return &Service{
		flow:             registrasi.RegisterFlow(),
		claim:            o.ClaimRepo,
		task:             o.TaskRepo,
		policy:           o.PolicyRepo,
		number:           o.NumberIssuer,
		parameter:        o.Parameter,
		rate:             o.ExchangeRateSource,
		assigner:         o.Assigner,
		notifier:         o.Notifier,
		audit:            o.AuditRecorder,
		reportLink:       o.ClaimReportLink,
		area:             o.AreaDirectory,
		items:            o.PolicyItems,
		currency:         o.CurrencyDirectory,
		options:          o.ItemOptions,
		records:          o.ClaimRecords,
		faceSheet:        o.FaceSheet,
		renderer:         o.FaceSheetRenderer,
		pla:              o.PLA,
		plaRenderer:      o.PLARenderer,
		groups:           o.Groups,
		inbox:            o.Inbox,
		accounts:         o.Accounts,
		tiering:          o.CommitteeTiering,
		committees:       o.Committees,
		id:              o.IDGenerator,
		unit:             o.UnitOfWork,
		clock:            o.Clock,
		validateOnReturn: o.ValidateOnReturn,
	}, nil
}

// Flow mengembalikan definisi alur Register.
//
// Layar memakainya untuk menggambar jalur tahap; ia hanya dibaca, tidak pernah diubah.
func (l *Service) Flow() registrasi.Definition { return l.flow }

// advance menutup tugas berjalan dan melahirkan tugas tahap berikutnya.
//
// Ia mengembalikan tugas baru, atau nil bila klaim mencapai simpul akhir. Jejak
// keputusan yang dilewati ikut dikembalikan supaya dapat masuk ke jejak audit — tanpa
// itu, klaim yang berpindah ke tahap tak terduga tidak dapat dijelaskan kepada siapa
// pun.
func (l *Service) advance(
	fctx advanceContext,
) (*registrasi.Task, []string, error) {
	node, trace, err := l.flow.Next(fctx.claim.CurrentStage, registrasi.FlowContext{
		Claim:       *fctx.claim,
		CallerRoles: fctx.caller.Roles,
	})
	if err != nil {
		return nil, trace, err
	}

	if err := fctx.task.Complete(fctx.caller.Identity, fctx.action, fctx.now); err != nil {
		return nil, trace, err
	}

	if node.Kind == registrasi.NodeEnd {
		fctx.claim.ProcessStatus = node.End.ProcessStatus
		fctx.claim.ProgressPositionStatus = registrasi.PositionDone
		fctx.claim.CurrentStage = ""
		return nil, trace, nil
	}

	stage := node.Stage
	recipients, err := l.assigner.Assign(fctx.ctx, stage, *fctx.claim, fctx.caller.Identity)
	if err != nil {
		return nil, trace, fmt.Errorf("registrasi/usecase: menentukan penerima tahap %q: %w", stage.ID, err)
	}

	// PIC yang dipilih router ditulis ke klaim, supaya PICTEKNIK dan USERTEKNIS_1 berisi
	// pemegang tugasnya — lihat registrasi/technicalpic.go.
	registrasi.AdoptTechnicalPIC(fctx.claim, stage, recipients)

	fresh := registrasi.NewTask(l.id.New(), *fctx.claim, stage, recipients, fctx.now)
	fctx.claim.CurrentStage = stage.ID
	return &fresh, trace, nil
}

// loadOpenTask mengambil klaim beserta tugas terbukanya, dan memastikan pemanggil
// benar-benar berada di tahap yang ia klaim.
func (l *Service) loadOpenTask(fctx loadContext) (registrasi.Claim, registrasi.Task, error) {
	task, err := l.task.Get(fctx.ctx, fctx.taskID)
	if err != nil {
		return registrasi.Claim{}, registrasi.Task{}, err
	}
	if !task.Open() {
		return registrasi.Claim{}, registrasi.Task{}, registrasi.ErrTaskAlreadyDone
	}

	claim, err := l.claim.Get(fctx.ctx, task.ClaimID)
	if err != nil {
		return registrasi.Claim{}, registrasi.Task{}, err
	}
	if claim.CurrentStage != task.Stage {
		// Claim berpindah tahap sejak layar dimuat — lewat lompatan lateral, atau
		// lewat rekan kerja yang lebih dulu menyelesaikannya. Menolak lebih baik
		// daripada menuliskan hasil kerja ke tahap yang sudah lewat.
		return registrasi.Claim{}, registrasi.Task{}, registrasi.ErrStageMismatch
	}

	stage, ok := l.flow.Stage(task.Stage)
	if !ok {
		return registrasi.Claim{}, registrasi.Task{}, fmt.Errorf("%w: %q",
			registrasi.ErrUnknownStage, task.Stage)
	}
	if fctx.requiredStage != "" && stage.ID != fctx.requiredStage {
		return registrasi.Claim{}, registrasi.Task{}, registrasi.ErrStageMismatch
	}
	if fctx.action != "" && stage.ExitAction != fctx.action {
		return registrasi.Claim{}, registrasi.Task{}, fmt.Errorf("%w: %q pada tahap %q",
			registrasi.ErrInvalidAction, fctx.action, stage.ID)
	}

	return claim, task, nil
}

// advanceContext adalah bahan satu perpindahan tahap.
//
// Ia dikumpulkan menjadi satu struct alih-alih tujuh parameter berurutan, supaya
// pemanggilnya tidak dapat tertukar urutan — dua di antaranya sama-sama string.
type advanceContext struct {
	ctx    context.Context
	claim  *registrasi.Claim
	task   *registrasi.Task
	caller Caller
	action string
	now    time.Time
}

// loadContext adalah bahan pemuatan klaim beserta tugas berjalannya.
type loadContext struct {
	ctx    context.Context
	taskID string

	// requiredStage menolak permintaan yang ditujukan ke tahap lain. Kosong berarti tahap
	// apa pun diterima.
	requiredStage string

	// action menolak tindakan yang tidak menjadi penutup tahap itu. Kosong berarti
	// action tidak diperiksa.
	action string
}

// ResolveCaller melengkapi pemanggil dengan perannya dari POOLDATA.M_LOGIN_GROUP_PNC.
// Transport memanggilnya sekali per permintaan, sebelum layanan mana pun dipakai.
func (l *Service) ResolveCaller(ctx context.Context, by Caller) (Caller, error) {
	groups, err := l.groups.GroupsOf(ctx, by.Identity)
	if err != nil {
		return Caller{}, err
	}
	by.Roles = registrasi.RolesOfGroups(append(append([]string{}, by.Roles...), groups...))
	return by, nil
}

// canWork menyatakan pemanggil boleh mengerjakan tugas: pemiliknya, tugas belum bertuan,
// atau ia memegang grup tahap itu (`registrasi.CanWork`).
func (l *Service) canWork(task registrasi.Task, by Caller) bool {
	stage, _ := l.flow.Stage(task.Stage)
	return registrasi.CanWork(task, stage, by.Identity, by.Roles)
}

// mirrorInbox menulis ulang baris daftar kerja klaim dari keadaannya sekarang.
//
// Dipanggil di akhir setiap transaksi yang mengubah klaim atau tugasnya. Tugas terbuka
// dibaca ulang di dalam transaksi itu, bukan diteruskan pemanggil, supaya satu fungsi ini
// yang menentukan isi baris — pemanggil tidak dapat lupa menyertakan tugas yang baru lahir.
func (l *Service) mirrorInbox(ctx context.Context, claim registrasi.Claim) error {
	var open *registrasi.Task
	task, err := l.task.OpenTaskForClaim(ctx, claim.ID)
	switch {
	case err == nil:
		open = &task
	case errors.Is(err, registrasi.ErrTaskNotFound):
	default:
		return fmt.Errorf("registrasi/usecase: membaca tugas terbuka untuk daftar kerja: %w", err)
	}
	if err := l.inbox.Mirror(ctx, registrasi.NewInboxEntry(claim, open, l.flow)); err != nil {
		return fmt.Errorf("registrasi/usecase: menulis daftar kerja: %w", err)
	}
	return nil
}

// CanWork adalah canWork bagi transport — layar memakainya untuk mengunci isian tugas yang
// tidak boleh dikerjakan pemanggil.
func (l *Service) CanWork(task registrasi.Task, by Caller) bool { return l.canWork(task, by) }
