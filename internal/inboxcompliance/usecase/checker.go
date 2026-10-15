package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxcompliance"
)

// Form Compliance Checker — membuka satu klaim dari antrean, dan menyimpan keputusannya.
//
// # Kenapa kedua operasi ini ada di modul Inbox Compliance, bukan di modul tersendiri
//
// Karena di Pega pun begitu. Form ini satu-satunya flow action pada assignment yang
// menunggu di workbasket `CompliancePNC`, dan assignment itulah yang menjadi baris tab
// Compliance. Memisahkannya menjadi modul sendiri akan memutus satu alur menjadi dua, lalu
// menuntut keduanya sepakat tentang klaim mana yang sedang diperiksa.

// CheckerOpened adalah form yang terbuka beserta bahan menggambarnya.
type CheckerOpened struct {
	// Case adalah klaimnya beserta keputusan yang sudah pernah disimpan, bila ada.
	Case inboxcompliance.CheckerCase

	// Choices adalah keempat Pilihan Compliance beserta labelnya.
	//
	// Ia datang dari server, bukan disalin ke layar, dengan alasan yang sama seperti
	// daftar tab: keempat nilainya adalah HASIL PEMBACAAN
	// `Property/PilihanCompliance_property.xml`, dan tempat pembacaan itu tercatat adalah
	// backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat.
	Choices []inboxcompliance.Choice

	// SurveyResults mengisi blok "Hasil Investigasi".
	//
	// Selalu KOSONG pada lini selain PA — bukan karena klaimnya tidak punya survei,
	// melainkan karena blok itu memang tidak digambar di sana (`IsPA`). Layar memakai
	// `tombol`/lini bisnis untuk memutuskan menggambarnya, bukan panjang senarai ini.
	SurveyResults []inboxcompliance.SurveyResult

	// Documents dan DocumentsError DICABUT 2026-10-08.
	//
	// Keduanya mengisi grid dokumen pada tab Compliance. Gridnya memang ada di
	// `Section/CompliancePNC-Section.xml`, tetapi sumbernya `TempDocumentAttach` —
	// berkelas `Code-Pega-List`, yakni halaman klipboard, dan satu-satunya yang mengisinya
	// berjalan saat MENYIMPAN (`SaveAllDataattachfilecompilance`).
	//
	// Pega karena itu tidak pernah memuatnya dari basis data saat form dibuka. Kami
	// memuatnya dari `DATA_ATTACHFILE` setiap kali, sehingga menampilkan berkas yang di
	// layar Pega tidak pernah terlihat — terbukti pada satu klaim yang punya tiga lampiran
	// di basis data tetapi kosong di Pega.
	//
	// `repo.FindDocuments` TETAP ADA dan tetap dipakai: penggantian surat penolakan
	// mencarinya berdasarkan nama, dan DeleteDocument memeriksa kepemilikannya. Yang
	// dicabut hanya pembacaan saat form dibuka.

	// SurveyResultsError terisi ketika hasil investigasi GAGAL dibaca, sementara form
	// tetap dibuka.
	//
	// Dibedakan dari senarai kosong dengan sengaja: keduanya menggambar tabel tanpa
	// baris, tetapi yang satu berarti "belum pernah disurvei" dan yang lain berarti
	// "tidak terbaca". Menyamakannya akan menyembunyikan gangguan basis data sebagai
	// keadaan normal.
	SurveyResultsError error

	// DocumentChecklist mengisi tab "Dokumen".
	//
	// Selalu KOSONG pada lini selain Travel — bukan karena klaimnya tidak punya kategori
	// dokumen, melainkan karena tabnya memang tidak digambar di sana
	// (`pyContainerVisibleWhen = IsTravel` pada layout S3). Layar memakai lini bisnisnya
	// untuk memutuskan menggambar, bukan panjang senarai ini.
	DocumentChecklist []inboxcompliance.DocumentChecklistItem

	// DocumentChecklistError terisi ketika daftar periksa GAGAL dibaca, sementara form
	// tetap dibuka.
	//
	// Dibedakan dari senarai kosong, alasan yang sama seperti SurveyResultsError: tabel
	// kosong berarti "lini ini belum punya master jenis dokumen", dan itu berbeda dari
	// "tidak terbaca". Menyamakannya akan menyembunyikan gangguan basis data sebagai
	// keadaan normal — dan di sini akibatnya lebih jauh, karena petugas dapat menyimpulkan
	// tidak ada dokumen yang wajib.
	DocumentChecklistError error

	// RejectPrefill mengisi tiga isian form Surat Penolakan saat dialognya dibuka.
	//
	// # Dibaca saat FORM dibuka, bukan saat dialognya dibuka
	//
	// Pega membacanya lewat `AutoFillFormReject_Pre`, yakni pre-processing local action
	// `ShowFormRejectClaim` — jadi saat DIALOGNYA dibuka, bukan sebelumnya.
	//
	// Di sini ia ikut pada pembukaan form, dan itu penyimpangan yang disengaja: nilainya
	// tiga teks pendek dari satu kueri terindeks, form ini sudah membaca dokumen dan
	// hasil investigasi, dan dengan begini dialognya terbuka tanpa menunggu. Jumlah
	// kuerinya bertambah satu untuk petugas yang tidak pernah membuka dialog itu —
	// harga yang diterima.
	//
	// Kegagalannya TIDAK dibawa ke layar sebagai galat tersendiri: pra-isi yang kosong
	// hanya berarti petugas mengetik sendiri, dan itu keadaan yang memang mungkin terjadi
	// — isian keempatnya pun selalu kosong karena tidak punya kolom.
	RejectPrefill inboxcompliance.RejectPrefill
}

// OpenChecker membuka form Compliance Checker atas satu klaim.
//
// # Kenapa klaimnya dicari DI ANTREAN, bukan di seluruh tabel klaim
//
// Meniru Pega: form ini adalah flow action di atas sebuah assignment. Tanpa assignment yang
// menunggu, tidak ada yang dapat dibuka — dan itu bukan rincian teknis melainkan kewenangan.
// Klaim yang sudah selesai diperiksa, atau yang sedang menunggu di antrean lain, bukan
// pekerjaan petugas Compliance.
//
// Akibatnya galatnya ErrClaimNotInQueue, bukan "tidak ditemukan". Pembedaan itu yang membuat
// petugas menyegarkan daftarnya alih-alih mencari klaim yang sebenarnya ada.
func (s *Service) OpenChecker(
	ctx context.Context, portalAlias string, reference string,
) (CheckerOpened, error) {
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return CheckerOpened{}, err
	}

	if reference == "" {
		return CheckerOpened{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim yang dibuka tidak disebutkan.",
			}},
		)
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return CheckerOpened{}, err
	}

	claim, found, err := repo.FindInQueue(ctx, query, reference)
	if err != nil {
		return CheckerOpened{}, fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return CheckerOpened{}, inboxcompliance.ErrClaimNotInQueue
	}

	// Aging diisi di sini, bukan di repo, dengan alasan yang sama seperti pada daftar:
	// hitungan waktu tunggu adalah aturan bisnis yang `D-50` tetapkan ditulis ulang di Go.
	claim = claim.WithElapsed(s.clock.Now())

	opened := CheckerOpened{
		Case:    inboxcompliance.CheckerCase{Claim: claim},
		Choices: inboxcompliance.Choices(),
	}

	decision, decided, err := repo.FindDecision(ctx, reference)
	if err != nil {
		return CheckerOpened{}, fmt.Errorf("membaca keputusan Compliance: %w", err)
	}
	if decided {
		opened.Case.Decision = &decision
	}

	// Pembacaan dokumen klaim DICABUT di sini — lihat catatan pada CheckerOpened.
	//
	// Satu kueri per pembukaan form ikut hilang bersamanya.

	// Blok "Hasil Investigasi" — HANYA pada lini PA.
	//
	// Syaratnya `IsPA` pada layout S2 `Section/ComplianceChecker-Section.xml`. Membacanya
	// untuk lini lain berarti satu kueri setiap kali form dibuka yang hasilnya tidak
	// pernah digambar.
	if claim.GroupPanel == inboxcompliance.GroupPanelPersonalAccident {
		results, err := repo.FindSurveyResults(ctx, reference)
		if err != nil {
			// Kegagalannya TIDAK menutup form, dan itu keputusan sadar.
			//
			// Blok ini menampilkan hasil kerja orang lain; ia tidak menentukan apa yang
			// boleh petugas Compliance putuskan. Menggagalkan seluruh form karena satu
			// tabel bacaan tidak terbaca akan menghentikan pekerjaan yang sebenarnya
			// masih dapat berjalan.
			//
			// Yang TIDAK boleh terjadi adalah gagal diam-diam: blok kosong tanpa
			// keterangan akan terbaca sebagai "belum pernah disurvei", padahal artinya
			// "tidak terbaca". Karena itu galatnya dibawa ke layar lewat
			// SurveyResultsError, dan layar menggambarnya sebagai gangguan.
			opened.SurveyResultsError = err
		} else {
			opened.SurveyResults = results
		}
	}

	// Tab "Dokumen" — HANYA pada lini Travel.
	//
	// Syaratnya `IsTravel` pada layout S3 `Section/ComplianceChecker-Section.xml`, dan
	// `IsTravel` = `Policy.Quotation.GroupPanel = "005"`. Membacanya untuk lini lain
	// berarti satu kueri setiap kali form dibuka yang hasilnya tidak pernah digambar.
	//
	// Bentuknya sengaja dibuat kembar dengan blok Hasil Investigasi di atas: keduanya
	// tab bersyarat lini, keduanya tidak boleh menutup form saat gagal, dan keduanya
	// membedakan "kosong" dari "tidak terbaca". Dua hal yang sama sebaiknya terbaca sama.
	if claim.GroupPanel == inboxcompliance.GroupPanelTravel {
		items, err := repo.FindDocumentChecklist(ctx, reference)
		if err != nil {
			opened.DocumentChecklistError = err
		} else {
			opened.DocumentChecklist = items
		}
	}

	// Pra-isi form Surat Penolakan.
	//
	// Galatnya SENGAJA diabaikan, tidak seperti kedua pembacaan di atas: yang hilang
	// hanyalah kenyamanan mengetik, dan dialognya tetap terbuka dengan isian kosong yang
	// dapat diketik sendiri. Membawanya ke layar sebagai gangguan akan menakuti petugas
	// atas sesuatu yang tidak menghalangi pekerjaannya.
	//
	// Yang TIDAK diabaikan adalah jejaknya: galatnya dicatat supaya gangguan basis data
	// yang berulang tetap terlihat di log.
	if prefill, err := repo.FindRejectPrefill(ctx, reference); err != nil {
		if s.logger != nil {
			s.logger.Warn("pra-isi surat penolakan tidak terbaca",
				slog.String("klaim", reference),
				slog.String("galat", err.Error()),
				slog.String("akibat", "dialog terbuka dengan isian kosong"))
		}
	} else {
		opened.RejectPrefill = prefill
	}

	return opened, nil
}

// Decided adalah hasil penyimpanan satu keputusan Compliance.
type Decided struct {
	// Decision adalah keputusan yang tersimpan.
	Decision inboxcompliance.Decision

	// Claim adalah klaim yang diputuskan, sebagaimana terbaca dari antrean.
	Claim inboxcompliance.WorkItem

	// PostAudit adalah baris Post Audit yang ikut terbit, HANYA pada ChoicePostAudit.
	//
	// Pointer supaya layar dapat membedakan "Post Audit terbit, ini nomornya" dari tiga
	// pilihan lain yang tidak menerbitkan apa pun.
	PostAudit *inboxcompliance.PostAuditEntry
}

// SubmitDecision menyimpan keputusan Compliance atas satu klaim — padanan tombol
// "Simpan Data" pada `Section/ComplianceChecker-Section.xml`.
//
// # Urutan yang ditiru dari SetComplianceResult
//
//	langkah 1      baca PilihanCompliance
//	langkah 2      CPLValidDate := now           bila pilihan 1
//	langkah 5      TanggalKirimPostAudit := now  bila pilihan 2
//	langkah 6–8    InsertHistoryClaimPNC dengan keterangan per pilihan
//	langkah 9      Obj-Save
//	langkah 10     pxAddChildWork kelas Work-Compliance  bila pilihan 2
//
// Langkah 10 itulah yang di sini menjadi CreatePostAudit: child case `Work-Compliance`
// adalah baris Post Audit, dan awalan `CPL-` pada nomornya memang singkatan kelas itu.
//
// # Akibat pada klaim: tiga dari sembilan
//
// `SetComplianceResult` menulis sembilan hal. **Tiga** dikerjakan di sini lewat
// `ApplyDecisionToClaim` — `STATUSCLAIM`, `CPLVALID_DATE`, `POSTAUDIT_TF_ANALYSTDATE`.
//
// Catatan versi sebelumnya menyatakan tidak satu pun dikerjakan, dengan alasan `P-1`.
// Itu **tidak lagi berlaku**: Work Owner menetapkan `T_CLAIM_PNC` memang dipakai Go
// (2026-10-06), dan delapan modul lain sudah menulisnya.
//
// Enam sisanya BELUM, dan sebabnya berbeda-beda — bukan satu alasan seperti dulu:
//
//	ComplianceStatus pada Objek      tidak punya kolom di mana pun
//	ComplianceStatus pada Adjustment tidak punya kolom di mana pun
//	isComplianceTransfer             tidak punya kolom di mana pun
//	UserBusinessPA                   tidak punya kolom di mana pun
//	baris riwayat (InsertHistoryClaimPNC)  belum dibangun
//	Ticket SendtoAnalysator               belum dibangun
//
// Keempat yang pertama diperiksa langsung ke `Database/PEGA_CONVERT_JSONKLAIM_PNC.prc`,
// procedure yang meratakan klaim ke tabel: nol kemunculan. Ia menuntut Tim Pega
// mengeksposnya ke kolom lebih dulu — permintaan yang sama bentuknya dengan
// `ComplianceRemark`.
//
// Konsekuensinya tetap HARUS dinyatakan: klaim berpindah status dan tanggalnya tercatat,
// tetapi **belum keluar dari antrean Pega**, karena Ticket-nya belum dibangun.
func (s *Service) SubmitDecision(
	ctx context.Context,
	portalAlias string,
	caller Caller,
	input inboxcompliance.DecisionInput,
) (Decided, error) {
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return Decided{}, err
	}

	if input.Reference == "" {
		return Decided{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim yang diputuskan tidak disebutkan.",
			}},
		)
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Decided{}, err
	}

	claim, found, err := repo.FindInQueue(ctx, query, input.Reference)
	if err != nil {
		return Decided{}, fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return Decided{}, inboxcompliance.ErrClaimNotInQueue
	}

	// Di sini DULU ada penolakan khusus klaim Travel, atas premis bahwa form Travel tidak
	// punya tombol "Simpan Data".
	//
	// Premisnya salah dan penolakannya dicabut. Tombol Simpan ber-`pyVisible=ALWAYS` di
	// KEDUA bilah tombol Pega; yang `!IsTravel` adalah salah satu wadahnya, dan wadah
	// pasangannya (`ComplianceChecker` S4, `IsTravel`) melayani Travel. Lihat FormActions.
	//
	// Penolakan itu karena itu melarang apa yang Pega izinkan — persis yang `P-5` cegah.
	// Jangan dihidupkan kembali tanpa bukti baru dari export.

	decision, err := inboxcompliance.NewDecision(input, claim, caller.Login, s.clock.Now())
	if err != nil {
		return Decided{}, err
	}

	if err := repo.SaveDecision(ctx, decision); err != nil {
		return Decided{}, fmt.Errorf("menyimpan keputusan Compliance: %w", err)
	}

	// Semua yang di bawah ini adalah akibat `SetComplianceResult` — dan activity itu
	// HANYA dipanggil tombol "Kirim ke Analyst" (sel 79) dan "Kirim ke PIC Teknik"
	// (sel 80). Tombol "Simpan Data" (sel 78) tidak memanggilnya.
	//
	// Jadi menekan Simpan menyimpan keputusannya saja: klaim tidak berubah status, tidak
	// mendapat baris riwayat pilihan, tidak menerbitkan Post Audit, dan TETAP menunggu di
	// antrean Compliance.
	//
	// Versi sebelumnya menjalankan seluruhnya pada setiap simpan. Akibatnya menekan
	// "Simpan Data" memindahkan klaim keluar dari antrean — dan itu tidak dapat
	// dibatalkan petugas, karena formnya tidak dapat dibuka lagi setelah klaim berpindah.
	if input.Action != inboxcompliance.ActionSend {
		return Decided{Decision: decision, Claim: claim}, nil
	}

	// Akibat pada KLAIMNYA — padanan tiga langkah pertama `SetComplianceResult`.
	//
	// Urutannya disengaja dan sama alasannya dengan pasangan keputusan→Post Audit di
	// bawah: bila yang kedua gagal, yang tersimpan adalah keputusan tanpa klaim berpindah
	// status — terlihat petugas dan dapat diulang karena form masih terbuka. Kebalikannya
	// lebih buruk: klaim berpindah status tanpa jejak siapa yang memutuskan.
	if err := repo.ApplyDecisionToClaim(
		ctx, inboxcompliance.NewClaimEffect(decision),
	); err != nil {
		return Decided{}, fmt.Errorf("menulis akibat keputusan ke klaim: %w", err)
	}

	// Baris riwayat — padanan `Call InsertHistoryClaimPNC`, langkah 16-18.
	//
	// Ia TIDAK selalu ditulis: hanya pada pilihan 0/1/2 dan hanya pada lini PA, meniru
	// syarat ketiga langkah itu. Lihat NewHistoryEntry.
	//
	// Kegagalannya MENGGAGALKAN penyimpanan, berbeda dari akibat pada klaim di atas.
	// Alasannya `D-59`: tanpa pemisahan tugas, baris ini satu-satunya jejak keputusan
	// yang menyangkut uang. Keputusan tersimpan tanpa jejaknya lebih buruk daripada
	// keputusan yang gagal tersimpan dan dapat diulang.
	if entry, perlu := inboxcompliance.NewHistoryEntry(decision, claim); perlu {
		if err := repo.AppendHistory(ctx, entry); err != nil {
			return Decided{}, fmt.Errorf("menulis baris riwayat keputusan: %w", err)
		}
	}

	// Perpindahan penugasan — padanan Ticket `SendtoAnalysator`, langkah 22.
	//
	// Ia dikerjakan TERAKHIR di antara ketiga akibat, dan urutannya disengaja: setelah
	// baris ini berhasil, klaim HILANG dari antrean Compliance. Menjalankannya lebih dulu
	// akan membuat kegagalan berikutnya meninggalkan klaim yang sudah pindah tetapi
	// keputusannya belum tersimpan — tidak terlihat siapa pun, dan tidak dapat diulang
	// karena formnya pun tidak dapat dibuka lagi.
	//
	// Kegagalannya MENGGAGALKAN penyimpanan, sama seperti riwayat: keputusan yang
	// tersimpan tetapi klaimnya tidak berpindah akan membuat petugas memutuskannya lagi,
	// dan itu menerbitkan baris Post Audit kedua.
	if move, pindah := inboxcompliance.NewAssignmentMove(decision, claim); pindah {
		if err := repo.MoveAssignment(ctx, move); err != nil {
			return Decided{}, fmt.Errorf("memindahkan penugasan klaim: %w", err)
		}
	}

	result := Decided{Decision: decision, Claim: claim}

	// Hanya ChoicePostAudit yang menerbitkan baris Post Audit — meniru langkah 10
	// `SetComplianceResult`, yang prasyaratnya `local.pilihan == 2`.
	//
	// Catatan yang dibawa ke barisnya adalah Remarks, bukan Note. Itu bukan tebakan:
	// langkah 5 menyalin `.ClaimData.ComplianceRemark` ke
	// `childPageCompliance.ComplianceRemarks`, dan kolom `REMARKS` pada
	// `T_CLAIM_COMPLIANCE_H` itulah yang tampil sebagai "Catatan" di tab Post Audit.
	if decision.Choice == inboxcompliance.ChoicePostAudit {
		entry, err := inboxcompliance.NewPostAuditEntry(
			inboxcompliance.PostAuditInput{
				Reference: input.Reference,
				Remarks:   decision.Remarks,
			},
			claim,
			decision.DecidedAt,
		)
		if err != nil {
			return Decided{}, err
		}

		// Urutannya SENGAJA: keputusan disimpan lebih dulu, baris Post Audit menyusul.
		//
		// Keduanya TIDAK berada dalam satu transaksi, dan itu kekurangan yang disadari.
		// Bila yang kedua gagal, keputusan tersimpan tanpa baris Post Audit — keadaan
		// yang dapat dilihat petugas dan dapat diulang, karena form masih terbuka.
		// Kebalikannya jauh lebih buruk: baris Post Audit terbit tanpa jejak siapa yang
		// memutuskan.
		//
		// Membungkus keduanya menuntut kepemilikan transaksi di lapisan aplikasi
		// (`08-TECHNICAL-STRATEGY.md` §4.5), yang seam Repo modul ini belum punya.
		saved, err := repo.CreatePostAudit(ctx, entry)
		if err != nil {
			return Decided{}, fmt.Errorf("menerbitkan baris Post Audit: %w", err)
		}
		result.PostAudit = &saved
	}

	if s.logger != nil {
		// Satu-satunya jejak yang menyebut pelakunya sampai `S-5` ada. Keputusan ini
		// menyangkut uang — `ChoiceFraud` menolak klaim — sehingga ketiadaan jejak audit
		// yang sebenarnya lebih berat akibatnya di sini daripada di jalur mana pun lain
		// di modul ini (`D-59`).
		label := decision.Choice
		if choice, known := inboxcompliance.FindChoice(decision.Choice); known {
			label = choice.Label
		}

		s.logger.Info(
			"keputusan Compliance disimpan",
			slog.String("portal", portalAlias),
			slog.String("oleh", caller.Login),
			slog.String("klaim", decision.Reference),
			slog.String("pilihan", decision.Choice),
			slog.String("pilihan_label", label),
			slog.Bool("post_audit_terbit", result.PostAudit != nil),
		)
	}

	return result, nil
}
