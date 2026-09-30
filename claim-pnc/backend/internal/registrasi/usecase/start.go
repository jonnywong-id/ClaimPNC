package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// StartCommand membuka klaim baru dari sebuah polis.
type StartCommand struct {
	PolicyNumber string
	Portal       string

	// RCVID menautkan klaim ke berkas laporan asalnya.
	//
	// Di Pega, tombol Register Klaim pada form Input Receive Document memanggil
	// CreateRegisterKlaimPNC, yang membuat case klaim DARI berkas RCV yang sedang
	// dibuka. Tautan itu bukan hiasan: kolom NOKLAIM pada baris RCV diisi dari sini,
	// dan itulah yang memindahkan berkasnya keluar dari tab Not Transferred.
	//
	// Kosong berarti klaim dimulai langsung dari layar registrasi, tanpa berkas RCV.
	RCVID string
}

// StartResult adalah klaim yang baru dibuka beserta tugas pertamanya.
type StartResult struct {
	Claim registrasi.Claim
	Task  registrasi.Task
}

// Start membuka klaim baru pada tahap pertama alur Register, LENGKAP dengan nomornya.
//
// # Nomor terbit di sini, bukan di ujung Input Register
//
// Versi pertama menunda penerbitan nomor sampai tahap Input Register ditutup, dengan
// alasan bahwa polis yang dibuka sekadar untuk dilihat tidak perlu memakan satu nomor.
// Alasan itu masuk akal, tetapi **tidak sesuai sistem lama** — dan kesetaraan perilaku
// yang menentukan (`P-5`), bukan penghematan nomor.
//
// Buktinya: `Activity/CreateRegisterKlaimPNC_act.xml` memanggil `CreateInputKlaim`, yang
// langkah 2-nya `Call svcAddWorkObject`. Di dalamnya, langkah 9 `Call addWork` adalah
// pembuatan work object Pega — dan Pega memberi `pyID` DI SITU, sebelum assignment
// pertama. Langkah 4 `CreateInputKlaim` sudah menyusun
// `"ASSIGN-WORKLIST " + curWorkPage.pzInsKey + "!Register_Flow"`, yang hanya mungkin bila
// kuncinya — beserta nomornya — sudah ada.
//
// Akibatnya bagi pengguna terlihat langsung: menekan "Register Klaim" membuka klaim yang
// SUDAH bernomor, bukan layar berjudul "Klaim belum bernomor".
//
// # Harga yang diterima
//
// Klaim yang dibuka lalu ditinggalkan tetap memakan satu nomor, dan nomor tidak dapat
// ditarik kembali (`ADR-0009`). Deret nomor karena itu akan berlubang. Itu persis yang
// terjadi di sistem lama, dan menyamainya lebih penting daripada deret yang rapat.
func (l *Service) Start(ctx context.Context, p StartCommand, by Caller) (StartResult, error) {
	policyNumber := strings.TrimSpace(p.PolicyNumber)
	if policyNumber == "" {
		return StartResult{}, fmt.Errorf("registrasi/usecase: nomor polis wajib diisi")
	}

	policy, err := l.policy.Get(ctx, policyNumber)
	if err != nil {
		return StartResult{}, err
	}

	now := l.clock.Now().UTC()

	// Klaim yang lahir dari berkas RCV MELOMPAT ke Input Register, tidak melewati
	// View Polis.
	//
	// `Activity/CreateRegisterKlaimPNC_act.xml` langkah 27 memanggil
	// `SetTicket("setToRegister_ticket")`, dan di `Flow/Register_Flow.xml` tiket itu
	// (`Ticket8`) menempel pada `Assignment1` — assignment bernama "Input Register",
	// `pyUseCaseName` InputRegister. Itulah layar yang muncul di Pega tepat setelah
	// tombol Register Klaim ditekan.
	//
	// View Polis tetap menjadi tahap awal alur (`Start1`) untuk klaim yang dibuka
	// langsung dari layar registrasi, tanpa berkas RCV. Keduanya jalur yang berbeda,
	// dan RCVID-lah yang membedakannya: ia terisi HANYA lewat tombol Register Klaim.
	awal := l.flow.Start
	if strings.TrimSpace(p.RCVID) != "" {
		awal = registrasi.StageInputRegister
	}

	firstStage, ok := l.flow.Stage(awal)
	if !ok {
		return StartResult{}, fmt.Errorf("%w: %q", registrasi.ErrUnknownStage, awal)
	}

	claim := registrasi.Claim{
		ID:     l.id.New(),
		Portal: p.Portal,
		Policy: policy,

		Currency: policy.Currency,

		// Tautan ke berkas laporan asalnya; kosong bila klaim dimulai tanpa RCV.
		RCVID: strings.TrimSpace(p.RCVID),

		ProcessStatus:          registrasi.ProcessRunning,
		ClaimStatus:            "",
		ClaimFlag:              registrasi.FlagUnset,
		ProgressPositionStatus: registrasi.PositionInProgress,

		CurrentStage: firstStage.ID,
		CreatedBy:    by.Identity,
		CreatedAt:    now,
		UpdatedBy:    by.Identity,
		UpdatedAt:    now,
	}

	// Isi berkas RCV dibawa ke klaim.
	//
	// `Activity/CreateRegisterKlaimPNC_act.xml` langkah 14 menyalin sembilan nilai dari
	// berkas ke klaim sebelum menyimpannya. Tanpa penyalinan itu, petugas membuka layar
	// Register yang kosong dan mengetik ulang seluruh isi berkas yang baru saja diisinya.
	//
	// Berkas yang isinya masih kosong TIDAK menggagalkan pembuatan klaim: berkas RCV lahir
	// kosong dan boleh diregistrasi sebelum lengkap. Yang kosong tetap kosong di klaimnya,
	// dan gerbang validasi Input Register yang akan menuntutnya.
	if claim.RCVID != "" {
		isi, err := l.reportLink.Snapshot(ctx, claim.RCVID)
		if err != nil {
			return StartResult{}, fmt.Errorf("registrasi/usecase: membaca isi laporan: %w", err)
		}
		// Berkas yang sudah bernomor klaim tidak diregistrasi lagi: tanpa penolakan ini
		// terbit PNCN kedua dan NOKLAIM berkasnya tertimpa (Work Owner, 2026-09-29).
		if isi.ClaimNumber != "" {
			return StartResult{}, &registrasi.ReportAlreadyRegisteredError{
				ReportID: claim.RCVID, ClaimNumber: isi.ClaimNumber}
		}
		applyReport(&claim, isi)
	}

	// Objek, coverage, dan spreading diisi dari polis, seperti
	// `Activity/CallActivityInputRegister-Act.xml` langkah 16–20 dan 41. Aturannya ada di
	// registrasi.BuildInsuredItems; di sini hanya pengambilannya.
	//
	// Kegagalan membaca tabel polis MENGGAGALKAN pembukaan klaim, bukan diabaikan: klaim
	// tanpa objek tampak seperti polis tanpa objek, dan petugas akan mengetiknya ulang.
	source, err := l.items.Items(ctx, policy)
	if err != nil {
		return StartResult{}, fmt.Errorf("registrasi/usecase: membaca objek polis: %w", err)
	}
	claim.InsuredItem = registrasi.BuildInsuredItems(policy, claim.DateOfLoss, source)

	recipients, err := l.assigner.Assign(ctx, firstStage, claim, by.Identity)
	if err != nil {
		return StartResult{}, fmt.Errorf("registrasi/usecase: menentukan penerima tahap awal: %w", err)
	}
	task := registrasi.NewTask(l.id.New(), claim, firstStage, recipients, now)

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		// Nomor terbit DI DALAM transaksi, bersama klaimnya.
		//
		// Menerbitkannya di luar berarti transaksi yang gagal meninggalkan satu nomor yang
		// sudah terpakai dan tidak pernah menjadi klaim — lubang di deret yang muncul di
		// surat ke tertanggung dan di PLA/DLA ke reasuransi, dan tidak dapat ditambal
		// karena nomor tidak dapat ditarik kembali (`ADR-0009`).
		number, err := l.number.Issue(ctx, now)
		if err != nil {
			return fmt.Errorf("registrasi/usecase: menerbitkan nomor klaim: %w", err)
		}
		claim.Number = number
		task.ClaimNumber = number

		// Pengenal klaim = nomornya (Work Owner, 2026-09-26).
		//
		// Baris berkas RCVN di T_CLAIM_PNC ber-CLAIMID nomornya sendiri, sehingga Work
		// Owner mencari PNCN dengan cara yang sama — dan tidak menemukannya, karena CLAIMID
		// baris klaim berisi pengenal acak. Sejak nomor terbit saat klaim dibuka (keputusan
		// 43), pengenal acak itu tidak lagi punya alasan: nomornya sudah unik dan sudah ada
		// sebelum baris pertama ditulis.
		//
		// Dilakukan DI SINI, sebelum satu pun baris ditulis, supaya kepala klaim, pohon
		// anaknya, tugas, dan jejak audit semuanya memakai pengenal yang sama. Klaim yang
		// dibuka sebelum perubahan ini tetap memakai pengenal acaknya — tidak ada data lama
		// yang diubah.
		claim.ID = number
		task.ClaimID = number

		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		if err := l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID,
			Event:   "KLAIM_DIBUKA",
			Actor:   by.Identity,
			At:      now,
			Note:    "Polis " + policy.Number + " dibuka pada tahap " + firstStage.Name,
		}); err != nil {
			return err
		}

		// Berkas laporan asalnya ditandai diserahkan LALU dipasangi nomor klaim,
		// sehingga ia berpindah dari "Not Transferred" langsung ke "Outstanding".
		//
		// Urutannya mengikat: NOKLAIM yang terisi sementara TRANSFERASM masih kosong
		// menghasilkan kombinasi yang TIDAK punya tab, sehingga berkasnya lenyap dari
		// seluruh daftar. Lihat reportlink.sql.
		//
		// Keduanya di DALAM transaksi yang sama dengan pembuatan klaim: klaim yang lahir
		// tanpa berkasnya ikut berpindah adalah keadaan yang tampak seperti tombol tidak
		// bekerja, dan petugas akan menekannya lagi.
		if claim.RCVID == "" {
			return nil
		}
		if err := l.reportLink.MarkHandedOver(ctx, claim.RCVID, now); err != nil {
			return err
		}
		return l.reportLink.AttachClaimNumber(ctx, claim.RCVID, claim.Number)
	})
	if err != nil {
		return StartResult{}, err
	}

	return StartResult{Claim: claim, Task: task}, nil
}

// applyReport menyalin isi berkas laporan ke klaim yang baru dibuka.
//
// Pemetaannya mengikuti `Activity/CreateRegisterKlaimPNC_act.xml` langkah 14 satu per
// satu. Yang TIDAK disalin disebutkan di bawah, supaya ketiadaannya terbaca sebagai
// pilihan — bukan kelalaian.
//
// # Yang hanya DITAMBAHKAN, tidak menimpa
//
// Seluruh nilai hanya diisi bila berkasnya punya. Berkas yang separuh kosong tidak boleh
// mengosongkan nilai yang sudah ada di klaim — misalnya mata uang yang datang dari
// snapshot polis.
//
// # Hubungan pelapor
//
// Langkah 14 mengisi `InsuredRelationship := "7"` (lain-lain) dan
// `InsuredRelationshipOthers := Sender`. Itu bukan tebakan: nama pelapor pada berkas RCV
// adalah petugas atau pihak yang mengirim dokumen, bukan tertanggung sendiri.
// `RelationOther` menuntut keterangan, dan keterangan itulah yang diisi namanya.
//
// # Tanggal Lapor, bukan Tanggal Terima Dokumen
//
// Langkah 14 memetakan `ClaimData.ReportDate ← @toDateTime(ReceiveDocument.ReceivedDate)`
// dan `ClaimData.DateReceived ← ReceiveDocument.DateOfSentDocument`. Properti kedua tidak
// punya kolom pada tabel berkas, sehingga **Tanggal Terima Dokumen tidak dapat dibawa** dan
// tetap kosong.
//
// Membalik keduanya — mengisi `DateReceived` dari kolom yang bernama
// `TANGGALTERIMADOKUMEN` — adalah godaan yang saya sempat ikuti. Ia salah dua kali: kolom
// itu menyimpan `ReceivedDate`, dan `ReportDate` yang tertinggal kosong justru medan yang
// dipakai aturan "Tanggal Lapor ≤ DOL + 7 hari".
//
// # Yang TIDAK disalin, dan kenapa
//
//   - `ClaimData.DateReceived` — properti asalnya tidak punya kolom, lihat di atas.
//   - `Policy.PolicyNo` — klaim mengambil polis dari snapshot GISFW (`D-04`), bukan dari
//     teks di berkas. Nomor pada berkas dipakai memeriksa, lihat di bawah.
//   - `SubjectEmail`, `RincianKerusakan`, `NoKTP`, `SIM`, `ClaimNoExt` — belum ada
//     medannya di domain klaim ini.
//   - `DocumentList` — milik modul dokumen `S-1` yang belum ada.
//   - `KodeCabang` — datang dari snapshot polis, bukan dari berkas.
func applyReport(k *registrasi.Claim, isi registrasi.ClaimReportSnapshot) {
	if !isi.DateOfLoss.IsZero() {
		k.DateOfLoss = isi.DateOfLoss.UTC()
	}
	if !isi.ReportDate.IsZero() {
		k.ReportDate = isi.ReportDate.UTC()
	}

	if isi.ReporterName != "" {
		k.Reporter.Name = isi.ReporterName
		k.Reporter.Relation = registrasi.RelationOther
		k.Reporter.OtherRelation = isi.ReporterName
	}
	if isi.ReporterPhone != "" {
		k.Reporter.Phone = isi.ReporterPhone
	}
	if isi.ReporterEmail != "" {
		k.Reporter.Email = isi.ReporterEmail
	}

	if isi.Location != "" {
		k.Location = isi.Location
	}
	if isi.Chronology != "" {
		k.Chronology = isi.Chronology
	}
	if isi.EstimateValue > 0 {
		k.EstimateValue = isi.EstimateValue
	}
}
