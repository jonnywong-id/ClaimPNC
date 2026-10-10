package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	"claim-pnc/internal/inboxcompliance/usecase"
)

// Membuka form atas klaim yang sedang menunggu di antrean.
//
// Ia memeriksa tiga hal sekaligus karena ketiganya adalah satu janji: form yang terbuka
// membawa klaimnya, membawa keempat pilihan, dan Aging-nya SUDAH terisi — bukan dibiarkan
// kosong untuk diisi layar.
func TestOpenChecker(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)

	require.Equal(t, claimKey, opened.Case.Claim.Reference)
	require.Equal(t, "PNC-2114", opened.Case.Claim.CaseID)

	// Belum pernah diputuskan — dan itu keadaan normal, bukan galat.
	require.Nil(t, opened.Case.Decision)

	// Keempat pilihan datang dari server, bukan dari layar. Nilainya dibaca dari
	// `Property/PilihanCompliance_property.xml`.
	require.Len(t, opened.Choices, 4)
	require.Equal(t, inboxcompliance.ChoiceFraud, opened.Choices[0].Value)
	require.Equal(t, "Fraud / Tolak", opened.Choices[0].Label)
	require.Equal(t, inboxcompliance.ChoiceOther, opened.Choices[3].Value)
	require.Equal(t, "Lain-Lain", opened.Choices[3].Label)
}

// Klaim yang TIDAK di antrean tidak dapat dibuka, dan galatnya dibedakan.
//
// Ketiga kasus di bawah sama-sama menolak, dan ketiganya harus menolak dengan galat yang
// SAMA — `ErrClaimNotInQueue`, bukan "tidak ditemukan". Klaimnya boleh jadi ada dan sehat;
// yang tidak ada adalah assignment-nya.
func TestOpenCheckerMenolakKlaimDiLuarAntrean(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	cases := map[string]string{
		"sudah selesai":         "ASM-FW-GCNMFW-WORK PNC-SELESAI",
		"antrean lain":          "ASM-FW-GCNMFW-WORK PNC-LAIN",
		"tidak ada sama sekali": "ASM-FW-GCNMFW-WORK PNC-TIDAK-ADA",
	}

	for name, reference := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := service.OpenChecker(context.Background(), portal, reference)
			require.ErrorIs(t, err, inboxcompliance.ErrClaimNotInQueue)
		})
	}
}

// Keputusan tersimpan dan terbaca kembali saat form dibuka ulang.
//
// Ini yang membedakan form dari tombol sekali pakai: petugas yang membukanya lagi harus
// melihat pilihan yang sudah dibuatnya, bukan kotak kosong yang membuatnya mengira
// keputusannya hilang.
func TestSubmitDecisionTersimpanDanTerbacaUlang(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	decided, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
			Note:      "Dokumen lengkap.",
			Comments: []inboxcompliance.CommentInput{
				{Text: "Tidak ada temuan."},
			},
		},
	)
	require.NoError(t, err)

	require.Equal(t, inboxcompliance.ChoiceValid, decided.Decision.Choice)
	require.Equal(t, "ADMINCONTOH1", decided.Decision.DecidedBy)
	require.Equal(t, now, decided.Decision.DecidedAt)

	// Bayar/Valid mengisi CPLValidDate dan TIDAK menerbitkan baris Post Audit —
	// `SetComplianceResult` langkah 2 versus langkah 10.
	require.NotNil(t, decided.Decision.ValidatedAt)
	require.Equal(t, now, *decided.Decision.ValidatedAt)
	require.Nil(t, decided.Decision.SentToPostAuditAt)
	require.Nil(t, decided.PostAudit)

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)
	require.NotNil(t, opened.Case.Decision)
	require.Equal(t, inboxcompliance.ChoiceValid, opened.Case.Decision.Choice)
	require.Equal(t, "Dokumen lengkap.", opened.Case.Decision.Note)

	// Komentarnya ikut terbaca, bernomor mulai dari 1.
	//
	// Inilah tempat petugas menulis — BUKAN ComplianceRemark, yang pada
	// `Section/CompliancePNC-Section.xml` bertanda `pyEditOptions=Read-only` dan isinya
	// catatan Investigator.
	require.Len(t, opened.Case.Decision.Comments, 1)
	require.Equal(t, 1, opened.Case.Decision.Comments[0].Index)
	require.Equal(t, "Tidak ada temuan.", opened.Case.Decision.Comments[0].Text)
	require.Equal(t, now, opened.Case.Decision.Comments[0].Date,
		"tanggal yang tidak dikirim layar memakai waktu keputusan, padanan nilai bawaan "+
			"CurrentDateTime() pada sel Tanggal Komentar")
}

// Bayar/PostAudit — SATU-SATUNYA pilihan yang menerbitkan baris Post Audit.
//
// Padanan `SetComplianceResult` langkah 10, `pxAddChildWork` kelas `Work-Compliance`, yang
// prasyaratnya `local.pilihan == 2`.
//
// Yang dibawa ke barisnya adalah Catatan, bukan Note — langkah 5 menyalin
// `.ClaimData.ComplianceRemark` ke `childPageCompliance.ComplianceRemarks`.
func TestSubmitDecisionPostAuditMenerbitkanBaris(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	decided, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoicePostAudit,
			Note:      "Perlu diperiksa ulang.",
		},
	)
	require.NoError(t, err)

	require.NotNil(t, decided.PostAudit)
	require.Equal(t, "CPL.26.1", decided.PostAudit.CaseID)
	// Catatan yang dibawa ke baris Post Audit datang dari KLAIM, bukan dari layar:
	// `SetComplianceResult` langkah 5 menyalin `.ClaimData.ComplianceRemark`, yang pada
	// form ini read-only.
	//
	// Ia kosong di sini, dan itu BUKAN cacat uji melainkan keadaan yang sebenarnya:
	// kueri tab Compliance belum membawa kolom itu. `COMPLIANCE_REMARKS` yang ada pada
	// WorkItem hanya terisi di tab Post Audit, dari `h.REMARKS`. Kolom klaimnya menunggu
	// Report Definition `InboxRegisterCompliance_RD` dari Tim Pega.
	require.Empty(t, decided.PostAudit.Remarks,
		"catatan Investigator belum dibawa kueri tab Compliance")
	require.Equal(t, claimKey, decided.PostAudit.ClaimNumber,
		"kolom NO_KLAIM berisi kunci teknis Pega, bukan nomor klaim")

	require.NotNil(t, decided.Decision.SentToPostAuditAt)
	require.Nil(t, decided.Decision.ValidatedAt)

	// Barisnya benar-benar muncul di tab Post Audit, bukan hanya dikembalikan.
	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{Tab: inboxcompliance.TabPostAudit},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	found := false
	for _, item := range listed.Page.Items {
		if item.CaseID == "CPL.26.1" {
			found = true
		}
	}
	require.True(t, found, "baris Post Audit harus tampil di tabnya")
}

// Tiga pilihan selain Bayar/PostAudit TIDAK menerbitkan baris apa pun.
//
// Diuji bersama, karena yang penting bukan satu pilihan melainkan BATASnya: hanya satu
// dari empat yang punya akibat di luar keputusan itu sendiri.
func TestSubmitDecisionSelainPostAuditTidakMenerbitkanBaris(t *testing.T) {
	t.Parallel()

	for _, choice := range []string{
		inboxcompliance.ChoiceFraud,
		inboxcompliance.ChoiceValid,
		inboxcompliance.ChoiceOther,
	} {
		t.Run(choice, func(t *testing.T) {
			service := newService(t, queueStore())

			decided, err := service.SubmitDecision(
				context.Background(), portal, usecaseCaller(),
				inboxcompliance.DecisionInput{
					Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: choice},
			)
			require.NoError(t, err)
			require.Nil(t, decided.PostAudit)
		})
	}
}

// Menyimpan ulang MENIMPA keputusan sebelumnya, tidak menambah keputusan kedua.
//
// `.ClaimData.PilihanCompliance` adalah satu properti pada klaimnya, bukan daftar.
func TestSubmitDecisionMenimpaKeputusanSebelumnya(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())
	ctx := context.Background()

	_, err := service.SubmitDecision(ctx, portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: inboxcompliance.ChoiceValid})
	require.NoError(t, err)

	_, err = service.SubmitDecision(ctx, portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: inboxcompliance.ChoiceFraud})
	require.NoError(t, err)

	opened, err := service.OpenChecker(ctx, portal, claimKey)
	require.NoError(t, err)
	require.NotNil(t, opened.Case.Decision)
	require.Equal(t, inboxcompliance.ChoiceFraud, opened.Case.Decision.Choice)

	// Stempel waktu pilihan SEBELUMNYA tidak boleh tertinggal: keputusan yang berlaku
	// sekarang Fraud/Tolak, dan ia tidak punya CPLValidDate.
	require.Nil(t, opened.Case.Decision.ValidatedAt)
}

// Pilihan yang kosong maupun yang tidak dikenal DITOLAK, dan keduanya menunjuk isiannya.
func TestSubmitDecisionMenolakPilihanTidakSah(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	for name, choice := range map[string]string{"kosong": "", "tidak dikenal": "9"} {
		t.Run(name, func(t *testing.T) {
			_, err := service.SubmitDecision(
				context.Background(), portal, usecaseCaller(),
				inboxcompliance.DecisionInput{
					Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: choice},
			)

			var validation *inboxcompliance.ValidationError
			require.ErrorAs(t, err, &validation)
			require.Len(t, validation.Violations, 1)
			require.Equal(t, inboxcompliance.FieldChoice, validation.Violations[0].Field)
		})
	}
}

// SELURUH pelanggaran dikembalikan sekaligus, bukan yang pertama saja (`P-5`).
//
// Tanpa ini petugas memperbaiki satu isian, menekan Simpan, lalu menemukan isian
// berikutnya — persis perilaku yang `P-5` ada untuk mencegahnya.
func TestSubmitDecisionMengumpulkanSeluruhPelanggaran(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())
	panjang := strings.Repeat("a", inboxcompliance.NoteMaxLength+1)

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    "", // pelanggaran 1
			Note:      panjang,
			Comments: []inboxcompliance.CommentInput{
				{Text: strings.Repeat("b", inboxcompliance.CommentMaxLength+1)},
			},
		},
	)

	var validation *inboxcompliance.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 3)

	fields := make([]string, 0, 3)
	for _, v := range validation.Violations {
		fields = append(fields, v.Field)
	}
	require.ElementsMatch(t, []string{
		inboxcompliance.FieldChoice,
		inboxcompliance.FieldNote,
		inboxcompliance.FieldComments,
	}, fields)
}

// Klaim di luar antrean tidak dapat diputuskan, sama seperti tidak dapat dibuka.
//
// Penyaringnya WAJIB sama dengan OpenChecker. Bila berbeda, akan ada klaim yang formnya
// terbuka tetapi keputusannya ditolak — atau sebaliknya, yang jauh lebih buruk.
func TestSubmitDecisionMenolakKlaimDiLuarAntrean(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: "ASM-FW-GCNMFW-WORK PNC-SELESAI",
			Choice:    inboxcompliance.ChoiceValid,
		},
	)
	require.ErrorIs(t, err, inboxcompliance.ErrClaimNotInQueue)
}

// lineStore membentuk antrean berisi satu klaim pada lini bisnis tertentu.
//
// Ia dipakai uji-uji visibilitas tombol di bawah, yang seluruhnya bergantung pada
// GROUPPANEL dan hanya pada itu.
func lineStore(groupPanel string) *memory.Store {
	return memory.NewStore(memory.Row{
		Workbasket: inboxcompliance.WorkbasketCompliance,
		CreatedAt:  now,
		Item: inboxcompliance.WorkItem{
			CaseID:     "PNC-2114",
			Reference:  claimKey,
			GroupPanel: groupPanel,
		},
	})
}

// Tombol pada form mengikuti syarat Pega, dan lini bisnis satu-satunya penentunya.
//
// Syaratnya dibaca dari DUA bilah tombol yang saling melengkapi — `CompliancePNC` S14
// (`!IsTravel`) dan `ComplianceChecker` S4 (`IsTravel`):
//
//	Unggah Dokumen            ALWAYS
//	Download Dokumen Reject   ALWAYS
//	Simpan Data               ALWAYS
//	Kirim ke Analyst          IsPA
//	Kirim ke PIC Teknik       IsTravel
//
// Baris Travel adalah yang paling penting: versi pertama uji ini menuntut Travel TIDAK
// mendapat tombol apa pun, dan itu mengunci pembacaan yang salah selama beberapa hari.
func TestTombolFormMengikutiLiniBisnis(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		groupPanel string
		want       inboxcompliance.FormActions
	}{
		{
			// Travel mendapat ketiga tombol umum DARI BILAH KEDUA — `ComplianceChecker`
			// S4, yang wadahnya justru `IsTravel`. Ia juga satu-satunya lini yang
			// mendapat "Kirim ke PIC Teknik".
			name:       "Travel mendapat tiga tombol umum dan Kirim ke PIC Teknik",
			groupPanel: inboxcompliance.GroupPanelTravel,
			want: inboxcompliance.FormActions{
				UploadDocument: true, DownloadRejectLetter: true, Save: true,
				SendToTechnician: true,
			},
		},
		{
			name:       "PA mendapat tiga tombol umum dan Kirim ke Analyst",
			groupPanel: inboxcompliance.GroupPanelPersonalAccident,
			want: inboxcompliance.FormActions{
				UploadDocument: true, DownloadRejectLetter: true,
				Save: true, SendToAnalyst: true,
			},
		},
		{
			name:       "lini lain mendapat tiga tombol umum, tanpa Kirim ke Analyst",
			groupPanel: "006",
			want: inboxcompliance.FormActions{
				UploadDocument: true, DownloadRejectLetter: true, Save: true,
			},
		},
		{
			// LEFT JOIN membuat lini bisnis dapat kosong. Pega memperlakukannya sama:
			// `compareTwoValues("", "=", "005")` bernilai salah, sehingga klaimnya bukan
			// Travel dan bukan PA.
			name:       "lini bisnis kosong diperlakukan bukan Travel dan bukan PA",
			groupPanel: "",
			want: inboxcompliance.FormActions{
				UploadDocument: true, DownloadRejectLetter: true, Save: true,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			service := newService(t, lineStore(c.groupPanel))

			opened, err := service.OpenChecker(context.Background(), portal, claimKey)
			require.NoError(t, err)
			require.Equal(t, c.want, opened.Case.Actions())
		})
	}
}

// Klaim Travel DAPAT disimpan — uji ini dibalik, bukan dihapus.
//
// Versi sebelumnya menuntut klaim Travel DITOLAK, atas premis bahwa form Travel tidak
// punya tombol "Simpan Data". Premisnya salah: `pyVisible=ALWAYS` di kedua bilah tombol
// Pega, dan yang `!IsTravel` adalah salah satu wadahnya — lihat FormActions.
//
// Dibalik supaya pencabutannya terkunci: dihapus begitu saja, penolakan itu dapat kembali
// diam-diam dan melarang lagi apa yang Pega izinkan (`P-5`).
func TestSubmitDecisionMenerimaKlaimTravel(t *testing.T) {
	t.Parallel()

	service := newService(t, lineStore(inboxcompliance.GroupPanelTravel))

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
		},
	)
	require.NoError(t, err, "klaim Travel punya tombol Simpan di Pega, jadi harus dapat disimpan")
}

// Lini selain Travel tetap dapat disimpan — penjagaan di atas tidak boleh kebablasan.
func TestSubmitDecisionMenerimaKlaimSelainTravel(t *testing.T) {
	t.Parallel()

	service := newService(t, lineStore(inboxcompliance.GroupPanelPersonalAccident))

	decided, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
		},
	)
	require.NoError(t, err)
	require.Equal(t, inboxcompliance.ChoiceValid, decided.Decision.Choice)
}

// Portal yang tidak dikenal DITOLAK pada kedua operasi, tidak jatuh ke portal utama.
//
// Jatuh ke koneksi bawaan berarti menampilkan — dan memutuskan — pekerjaan satu badan
// hukum lewat portal badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestCheckerMenolakPortalTidakDikenal(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())
	ctx := context.Background()

	_, err := service.OpenChecker(ctx, "entitas-lain", claimKey)
	require.Error(t, err)

	_, err = service.SubmitDecision(ctx, "entitas-lain", usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: inboxcompliance.ChoiceValid})
	require.Error(t, err)
}

// Kunci klaim yang kosong ditolak sebagai pelanggaran validasi, bukan sebagai galat teknis.
func TestCheckerMenolakReferensiKosong(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.OpenChecker(context.Background(), portal, "")

	var validation *inboxcompliance.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxcompliance.FieldReference, validation.Violations[0].Field)
}

// Galat repo DIBUNGKUS dengan konteks, tidak dilewatkan apa adanya.
//
// Tanpa pembungkus, pesan yang sampai ke log hanya menyebut kegagalan basis data tanpa
// menyebut operasi mana yang gagal — dan pada modul dengan dua jalur tulis, itu perbedaan
// yang menentukan saat menelusuri masalah.
func TestSubmitDecisionMembungkusGalatPenyimpanan(t *testing.T) {
	t.Parallel()

	broken := &failingDecisionStore{Store: queueStore()}

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxcompliance.Repo, error) { return broken, nil },
		Clock:        fixedClock{at: now},
	})
	require.NoError(t, err)

	_, err = service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action: inboxcompliance.ActionSend, Reference: claimKey, Choice: inboxcompliance.ChoiceValid},
	)
	require.ErrorIs(t, err, errSaveFailed)
	require.Contains(t, err.Error(), "menyimpan keputusan Compliance")
}

var errSaveFailed = errors.New("basis data menolak")

// failingDecisionStore berperilaku sama dengan penyimpanan di memori, kecuali SaveDecision
// yang selalu gagal.
//
// Penyematan, bukan tiruan utuh: yang diuji adalah pembungkusan galat, dan menulis ulang
// keempat operasi lain hanya akan membuat uji ini gagal setiap kali seam-nya bertambah.
type failingDecisionStore struct {
	*memory.Store
}

func (s *failingDecisionStore) SaveDecision(
	context.Context, inboxcompliance.Decision,
) error {
	return errSaveFailed
}

// Baris komentar kosong DIBUANG, bukan diadukan.
//
// Grid Pega menambah baris setiap kali petugas menekan Enter, sehingga baris kosong di
// ujung adalah kejadian normal. Menolaknya akan membuat form gagal disimpan karena hal
// yang bukan kesalahan siapa pun.
func TestSubmitDecisionMembuangKomentarKosong(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	decided, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
			Comments: []inboxcompliance.CommentInput{
				{Text: "Terisi."},
				{Text: "   "}, // spasi saja — tetap dianggap kosong
				{Text: ""},
			},
		},
	)
	require.NoError(t, err)

	require.Len(t, decided.Decision.Comments, 1)

	// Penomorannya mengikuti urutan SETELAH penyaringan, sehingga tidak berlubang —
	// sama seperti nomor baris yang dilihat petugas pada grid Pega.
	require.Equal(t, 1, decided.Decision.Comments[0].Index)
	require.Equal(t, "Terisi.", decided.Decision.Comments[0].Text)
}

// Menyimpan LEBIH SEDIKIT komentar MENGHAPUS baris yang tidak ikut dikirim.
//
// # Uji ini kebalikan dari versi sebelumnya, dan kedua alasan lamanya sudah gugur
//
//   - *"`MERGE` per baris pada adapter SQL"* — tabel komentar terpisah sudah TIDAK ADA
//     sejak 2026-10-07. Komentar kini satu kolom JSON yang ditulis utuh setiap kali,
//     sehingga baris yang tidak dikirim memang hilang.
//   - *"grid Pega-nya tidak punya tombol hapus"* — **SALAH**. Perbandingan layar
//     berdampingan memperlihatkan tautan **Hapus** tepat di sebelah **Tambah**. Kesimpulan
//     lama itu diambil dari `pyEditBehaviors`, yang ternyata bukan tempat tombol itu
//     didefinisikan.
//
// Work Owner menetapkan Pega ditiru apa adanya, tanpa ditambah maupun dikurangi.
//
// Uji ini sekaligus menjaga fake tidak kembali menggabungkan — penggabungan akan membuat
// tombol Hapus diam-diam tidak berpengaruh, dan tidak ada galat yang memberitahukannya.
func TestSubmitDecisionMenghapusKomentarYangTidakDikirim(t *testing.T) {
	t.Parallel()

	service := newService(t, queueStore())

	_, err := service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
			Comments: []inboxcompliance.CommentInput{
				{Text: "Pertama."}, {Text: "Kedua."}, {Text: "Ketiga."},
			},
		},
	)
	require.NoError(t, err)

	// Simpan ulang dengan DUA baris saja.
	_, err = service.SubmitDecision(
		context.Background(), portal, usecaseCaller(),
		inboxcompliance.DecisionInput{
			Action:    inboxcompliance.ActionSend,
			Reference: claimKey,
			Choice:    inboxcompliance.ChoiceValid,
			Comments: []inboxcompliance.CommentInput{
				{Text: "Pertama diubah."}, {Text: "Kedua."},
			},
		},
	)
	require.NoError(t, err)

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)
	require.NotNil(t, opened.Case.Decision)

	require.Len(t, opened.Case.Decision.Comments, 2,
		"baris ketiga seharusnya TERHAPUS, bukan tertinggal — grid Pega punya tombol Hapus")
	require.Equal(t, "Pertama diubah.", opened.Case.Decision.Comments[0].Text)
	require.Equal(t, "Kedua.", opened.Case.Decision.Comments[1].Text)
}

// Blok "Hasil Investigasi" terisi pada lini PA, dan KOSONG pada lini lain.
//
// Syaratnya `IsPA` pada layout S2 `Section/ComplianceChecker-Section.xml`. Uji ini menyemai
// baris survei untuk KEDUA lini, lalu menuntut hanya PA yang membacanya — sehingga ia gagal
// baik ketika syaratnya hilang maupun ketika syaratnya terbalik.
func TestHasilInvestigasiHanyaUntukPA(t *testing.T) {
	t.Parallel()

	disurvei := time.Date(2026, 9, 30, 4, 0, 0, 0, time.UTC)
	baris := inboxcompliance.SurveyResult{
		SurveyedAt:     &disurvei,
		ObjectName:     "Peserta Contoh",
		ObjectLocation: "Cabang Contoh",
		Status:         "Final Report",
	}

	cases := []struct {
		name       string
		groupPanel string
		wantRows   int
	}{
		{"PA membaca hasil investigasi", inboxcompliance.GroupPanelPersonalAccident, 1},
		{"Travel tidak membacanya", inboxcompliance.GroupPanelTravel, 0},
		{"lini lain tidak membacanya", "006", 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			store := lineStore(c.groupPanel)
			store.SeedSurveyResults(claimKey, baris)

			service := newService(t, store)

			opened, err := service.OpenChecker(context.Background(), portal, claimKey)
			require.NoError(t, err)
			require.NoError(t, opened.SurveyResultsError)
			require.Len(t, opened.SurveyResults, c.wantRows)
		})
	}
}

// Keempat isian baris hasil investigasi sampai utuh ke pemanggil.
//
// Terpisah dari uji di atas supaya kegagalan "barisnya ada tetapi isinya kosong" dapat
// dibedakan dari "barisnya tidak terbaca sama sekali".
func TestHasilInvestigasiMembawaKeempatIsiannya(t *testing.T) {
	t.Parallel()

	disurvei := time.Date(2026, 9, 30, 4, 0, 0, 0, time.UTC)

	store := lineStore(inboxcompliance.GroupPanelPersonalAccident)
	store.SeedSurveyResults(claimKey, inboxcompliance.SurveyResult{
		SurveyedAt:     &disurvei,
		ObjectName:     "Peserta Contoh",
		ObjectLocation: "Cabang Contoh",
		Status:         "Final Report",
	})

	service := newService(t, store)

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)
	require.Len(t, opened.SurveyResults, 1)

	row := opened.SurveyResults[0]
	require.NotNil(t, row.SurveyedAt)
	require.Equal(t, disurvei, *row.SurveyedAt)
	require.Equal(t, "Peserta Contoh", row.ObjectName)
	require.Equal(t, "Cabang Contoh", row.ObjectLocation)
	require.Equal(t, "Final Report", row.Status)
}

// Klaim PA yang BELUM pernah disurvei membuka form dengan blok kosong — bukan galat.
//
// Ini keadaan normal, dan membedakannya dari kegagalan baca adalah inti SurveyResultsError.
func TestHasilInvestigasiKosongBukanGalat(t *testing.T) {
	t.Parallel()

	service := newService(t, lineStore(inboxcompliance.GroupPanelPersonalAccident))

	opened, err := service.OpenChecker(context.Background(), portal, claimKey)
	require.NoError(t, err)
	require.NoError(t, opened.SurveyResultsError)
	require.Empty(t, opened.SurveyResults)
}
