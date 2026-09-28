package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/repo/memory"
)

// Uji di berkas ini menjaga BATAS KEPEMILIKAN layar rincian.
//
// Ia satu-satunya tempat batas itu dapat diuji sama sekali: di penyimpanan SQL ia hidup di
// dalam teks kueri, sebagai `IN (SELECT … WHERE LOGIN = …)`, sehingga tidak ada uji yang
// dapat membuktikannya tanpa Oracle.
//
// Taruhannya bukan kerapian. Layar ini dibaca PIHAK LUAR, dan satu batas yang hilang tidak
// menghasilkan galat — hanya nilai uang, dokumen, dan isi percakapan milik mitra lain yang
// terbuka kepada mitra yang salah.

func detailStore() *memory.Store { return memory.NewSampleStore() }

func scopeOfSample(claimNo string) inboxpladla.DetailScope {
	return inboxpladla.DetailScope{
		ClaimKey:       "ASM-FW-GCNMFW-WORK " + claimNo,
		Login:          memory.SampleReinsurerLogin,
		ReinsurerCodes: []string{"R100"},
	}
}

// Klaim yang pemberitahuannya dikirimkan kepada pemanggil DAPAT dibuka.
func TestTheHeaderOpensForAClaimAdvisedToTheCaller(t *testing.T) {
	header, err := detailStore().ClaimHeader(context.Background(), scopeOfSample("PNC-2001"))

	require.NoError(t, err)
	require.Equal(t, "PNC-2001", header.ClaimNo)
	require.Equal(t, "Register", header.StatusLabel,
		"arti kode status ikut dikirim, bukan angkanya saja")
}

// Klaim yang TIDAK menyangkut pemanggil dijawab "tidak ditemukan".
//
// Jawabannya sengaja sama dengan klaim yang memang tidak ada. Membedakan keduanya memberi
// tahu penanya bahwa klaimnya ADA — keterangan yang tidak berhak ia terima.
func TestAClaimBelongingToAnotherPartnerIsReportedAsNotFound(t *testing.T) {
	// PNC-2008 pemberitahuannya dikirim ke kode `R900`, milik login kedua.
	_, err := detailStore().ClaimHeader(context.Background(), scopeOfSample("PNC-2008"))
	require.ErrorIs(t, err, inboxpladla.ErrRowNotFound)

	_, err = detailStore().ClaimHeader(context.Background(), scopeOfSample("PNC-9999"))
	require.ErrorIs(t, err, inboxpladla.ErrRowNotFound,
		"klaim yang tidak ada dan klaim yang bukan milik pemanggil dijawab SAMA")
}

// Klaim TANPA pemberitahuan tetap dapat dibuka bila ada PERCAKAPAN yang menyangkutnya.
//
// Tanpa jalur ini, ketiga daftar komunikasi akan menggambar tombol rincian yang selalu
// ditolak: klaimnya memang tidak punya satu pun PLA maupun DLA.
func TestAClaimWithOnlyAConversationCanStillBeOpened(t *testing.T) {
	header, err := detailStore().ClaimHeader(context.Background(), scopeOfSample("PNC-2009"))

	require.NoError(t, err)
	require.Equal(t, "PNC-2009", header.ClaimNo)
}

// Grid pemberitahuan hanya menampilkan MILIK PEMANGGIL.
//
// Di Pega ia memuat seluruh mitra beserta nilai masing-masing, karena gridnya dimuat dari
// objek kerja klaim. Itu kebocoran antar mitra, dan ia tidak dibawa.
func TestTheAdviceGridHidesOtherPartnersAdvices(t *testing.T) {
	rows, err := detailStore().Advices(
		context.Background(), scopeOfSample("PNC-2001"), inboxpladla.AdviceKindPLA)

	require.NoError(t, err)
	require.Len(t, rows, 2)

	for _, row := range rows {
		require.NotEqual(t, "PLA/2026/2001-LAIN", row.No,
			"pemberitahuan milik mitra lain tidak boleh terlihat")
	}
}

// Grid pemberitahuan hanya menampilkan yang SUDAH terkirim.
func TestTheAdviceGridHidesAdvicesThatWereNeverSent(t *testing.T) {
	rows, err := detailStore().Advices(
		context.Background(), scopeOfSample("PNC-2002"), inboxpladla.AdviceKindDLA)

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "DLA/2026/2002", rows[0].No)
	require.Equal(t, "AKS/2026/2002", rows[0].AcceptanceNo,
		"nomor akseptasi hanya ada pada DLA")
}

// Dokumen milik mitra LAIN pada nomor pemberitahuan yang SAMA tidak terlihat.
//
// Inilah yang dijaga penyaring `T_DOC_REAS.LOGIN` yang diputuskan Work Owner —
// `GetDokumenReas` tidak memakainya.
func TestDocumentsOfAnotherPartnerAreHiddenEvenOnTheSameAdvice(t *testing.T) {
	rows, err := detailStore().Documents(
		context.Background(), scopeOfSample("PNC-2001"),
		"PLA/2026/2001-R1", inboxpladla.AdviceKindPLA)

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "DOK-01", rows[0].ID)
}

// Dokumen pada pemberitahuan yang BELUM terkirim tidak terlihat.
func TestDocumentsOfAnUnsentAdviceAreHidden(t *testing.T) {
	rows, err := detailStore().Documents(
		context.Background(), scopeOfSample("PNC-2002"),
		"DLA/2026/2002-DRAF", inboxpladla.AdviceKindDLA)

	require.NoError(t, err)
	require.Empty(t, rows)
}

// Isi dokumen DAPAT diambil bila seluruh rantai kepemilikannya sah.
func TestDocumentContentIsServedWhenTheWholeChainHolds(t *testing.T) {
	content, err := detailStore().DocumentContent(
		context.Background(), scopeOfSample("PNC-2001"), "DOK-01")

	require.NoError(t, err)
	require.Equal(t, "laporan-kerugian.pdf", content.Name)
	require.NotEmpty(t, content.Content)
}

// Isi dokumen milik mitra lain DITOLAK meski id-nya benar.
//
// `DATAID` adalah angka, dan angka dapat ditebak. Rantainya diperiksa ulang seluruhnya,
// bukan hanya id-nya.
func TestDocumentContentIsRefusedForAnotherPartnersDocument(t *testing.T) {
	_, err := detailStore().DocumentContent(
		context.Background(), scopeOfSample("PNC-2001"), "DOK-03")

	require.ErrorIs(t, err, inboxpladla.ErrDocumentNotFound)
}

// Riwayat komunikasi memuat KEDUA sisi percakapan yang menyangkut pemanggil.
func TestTheConversationGridShowsBothSidesThatInvolveTheCaller(t *testing.T) {
	rows, err := detailStore().Conversations(
		context.Background(), scopeOfSample("PNC-2002"))

	require.NoError(t, err)
	require.Len(t, rows, 2, "satu dikirim pemanggil, satu lagi sudah dijawab")

	for _, row := range rows {
		require.NotEqual(t, "KOM-04", row.ID)
	}
}

// Percakapan yang TIDAK menyangkut pemanggil tidak terlihat.
func TestAConversationBetweenOtherPartiesIsHidden(t *testing.T) {
	rows, err := detailStore().Conversations(
		context.Background(), scopeOfSample("PNC-2001"))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "KOM-01", rows[0].ID)
	require.True(t, rows[0].CanReply)
}

// Percakapan yang sudah ada balasannya TIDAK boleh dibalas.
func TestAnAnsweredConversationIsMarkedAsNotReplyable(t *testing.T) {
	rows, err := detailStore().Conversations(
		context.Background(), scopeOfSample("PNC-2002"))
	require.NoError(t, err)

	for _, row := range rows {
		if row.ID != "KOM-03" {
			continue
		}
		require.True(t, row.Answered)
		require.False(t, row.CanReply)
		require.Equal(t, "Siti Aminah", row.ReplierName)
		return
	}
	t.Fatal("percakapan KOM-03 tidak ditemukan")
}

// Balasan TERSIMPAN, dan percakapannya berpindah menjadi sudah dijawab.
func TestAReplyIsStoredAndMovesTheConversation(t *testing.T) {
	store := detailStore()
	scope := scopeOfSample("PNC-2001")

	command, err := inboxpladla.NewReplyCommand(
		scope.ClaimKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "Nilai disetujui."},
		inboxpladla.Caller{Login: memory.SampleReinsurerLogin, Name: "Mitra Contoh"},
		time.Date(2026, time.February, 8, 9, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	require.NoError(t, store.Reply(context.Background(), scope, command))

	rows, err := store.Conversations(context.Background(), scope)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	require.Equal(t, "Nilai disetujui.", rows[0].Reply)
	require.Equal(t, "Mitra Contoh", rows[0].ReplierName)
	require.True(t, rows[0].Answered)
	require.False(t, rows[0].CanReply)
}

// Balasan KEDUA ditolak — ia akan menimpa balasan pertama tanpa dapat dipulihkan.
//
// `M_KOMUNIKASI_PNC` menyimpan SATU balasan per percakapan; `REPLYMESSAGE` adalah kolom
// tunggal, bukan tabel anak. `ReplyKomunikasi` tidak memagarinya sama sekali.
func TestASecondReplyIsRefusedInsteadOfOverwritingTheFirst(t *testing.T) {
	store := detailStore()
	scope := scopeOfSample("PNC-2002")

	command, err := inboxpladla.NewReplyCommand(
		scope.ClaimKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-03", Message: "Menimpa?"},
		inboxpladla.Caller{Login: memory.SampleReinsurerLogin},
		time.Now(),
	)
	require.NoError(t, err)

	require.ErrorIs(t,
		store.Reply(context.Background(), scope, command),
		inboxpladla.ErrConversationAlreadyAnswered)

	rows, err := store.Conversations(context.Background(), scope)
	require.NoError(t, err)
	for _, row := range rows {
		if row.ID == "KOM-03" {
			require.Equal(t, "Sudah lengkap, terima kasih.", row.Reply,
				"balasan pertama harus tetap utuh")
		}
	}
}

// Percakapan milik pihak lain TIDAK dapat dibalas, meski nomornya benar.
func TestAConversationBetweenOtherPartiesCannotBeReplied(t *testing.T) {
	store := detailStore()
	scope := scopeOfSample("PNC-2001")

	command, err := inboxpladla.NewReplyCommand(
		scope.ClaimKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-04", Message: "Bukan urusan saya."},
		inboxpladla.Caller{Login: memory.SampleReinsurerLogin},
		time.Now(),
	)
	require.NoError(t, err)

	require.ErrorIs(t,
		store.Reply(context.Background(), scope, command),
		inboxpladla.ErrConversationNotFound)
}

// Nomor percakapan milik KLAIM LAIN tidak dapat dibalas lewat alamat klaim ini.
//
// `ReplyKomunikasi` hanya menyaring `komunikasiid`. Di Pega itu tidak terasa karena tombol
// balas hanya dapat dicapai dari layar yang sudah menyaringnya; di sini alamatnya dapat
// dipanggil langsung.
func TestAConversationOfAnotherClaimCannotBeRepliedThroughThisClaim(t *testing.T) {
	store := detailStore()

	command, err := inboxpladla.NewReplyCommand(
		scopeOfSample("PNC-2001").ClaimKey,
		// KOM-02 milik PNC-2002.
		inboxpladla.ReplyInput{ConversationID: "KOM-02", Message: "Salah klaim."},
		inboxpladla.Caller{Login: memory.SampleReinsurerLogin},
		time.Now(),
	)
	require.NoError(t, err)

	require.ErrorIs(t,
		store.Reply(context.Background(), scopeOfSample("PNC-2001"), command),
		inboxpladla.ErrConversationNotFound)
}

// Login pengembangan yang ditentukan MENGGANTIKAN mitra pertama seluruhnya.
//
// # Kenapa uji ini ada
//
// Penolakan layar ini terhadap login yang tidak terdaftar adalah perilaku yang BENAR, dan
// justru karena itu layar ini tidak dapat dilihat sama sekali di lingkungan pengembangan:
// login pengembang adalah pegawai internal.
//
// `NewSampleStoreFor` menjawabnya dengan mengganti login mitra pada DATA CONTOH — bukan
// dengan melonggarkan penyaringnya. Uji ini yang menjaga bedanya: login yang diberikan
// mendapat SELURUH data mitra pertama, dan login lain tetap ditolak.
func TestTheDevelopmentLoginReplacesTheFirstPartnerEntirely(t *testing.T) {
	const dev = "JONNY"

	store := memory.NewSampleStoreFor(dev)
	ctx := context.Background()

	codes, err := store.ReinsurerCodes(ctx, dev)
	require.NoError(t, err)
	require.Equal(t, []string{"R100"}, codes,
		"login pengembangan harus mewarisi kode mitra pertama, bukan kode baru")

	// Mitra bawaan sudah TIDAK terdaftar lagi — ia digantikan, bukan ditemani.
	//
	// Bila ia masih terdaftar, dua login akan berbagi kode yang sama dan tidak ada yang
	// dapat memastikan data contoh milik siapa.
	digantikan, err := store.ReinsurerCodes(ctx, memory.SampleReinsurerLogin)
	require.NoError(t, err)
	require.Empty(t, digantikan)

	scope := inboxpladla.DetailScope{
		ClaimKey:       "ASM-FW-GCNMFW-WORK PNC-2001",
		Login:          dev,
		ReinsurerCodes: codes,
	}

	// Ketiganya harus ikut berpindah. Mendaftarkan login saja — tanpa memindahkan
	// datanya — menghasilkan layar yang menjawab "belum ada pekerjaan", dan itu terbaca
	// sebagai perbaikan yang gagal.
	header, err := store.ClaimHeader(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, "PNC-2001", header.ClaimNo)

	advices, err := store.Advices(ctx, scope, inboxpladla.AdviceKindPLA)
	require.NoError(t, err)
	require.NotEmpty(t, advices)

	conversations, err := store.Conversations(ctx, scope)
	require.NoError(t, err)
	require.NotEmpty(t, conversations, "percakapan harus ikut berpindah ke login itu")

	documents, err := store.Documents(
		ctx, scope, "PLA/2026/2001-R1", inboxpladla.AdviceKindPLA)
	require.NoError(t, err)
	require.NotEmpty(t, documents, "dokumen harus ikut berpindah ke login itu")
}

// Login KEDUA tidak ikut berpindah.
//
// Ia yang membuktikan perbedaan antara daftar yang mencocokkan SELURUH kode reasuradur dan
// daftar yang hanya mencocokkan kode tertinggi — dan perbedaan itu menuntut dua login yang
// benar-benar berbeda.
func TestTheSecondSampleLoginIsLeftUntouched(t *testing.T) {
	store := memory.NewSampleStoreFor("JONNY")

	codes, err := store.ReinsurerCodes(context.Background(), memory.SampleSecondLogin)
	require.NoError(t, err)
	require.Equal(t, []string{"R901", "R900"}, codes)
}

// Login kosong jatuh ke bawaan, bukan menghasilkan data contoh tanpa mitra sama sekali.
func TestAnEmptyDevelopmentLoginFallsBackToTheDefault(t *testing.T) {
	store := memory.NewSampleStoreFor("   ")

	codes, err := store.ReinsurerCodes(
		context.Background(), memory.SampleReinsurerLogin)
	require.NoError(t, err)
	require.Equal(t, []string{"R100"}, codes)
}
