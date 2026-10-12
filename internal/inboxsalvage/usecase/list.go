// Package usecase mengorkestrasi modul Inbox Salvage.
//
// Empat operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List      mengambil isi satu daftar
//	Counts    mengambil tabel ringkas "Status Salvage / Jumlah"
//	Create    menyimpan satu pengajuan salvage beserta detail itemnya
//
// Ekspor TIDAK menjadi operasi kelima: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// # Yang MENULIS, dan kenapa itu berbeda dari modul inbox lain
//
// Create menulis ke `POOLDATA.PNC_SALVAGE` dan `POOLDATA.DETAIL_PNC_SALVAGE`. Keduanya
// dimiliki modul ini selama masa paralel — tidak ada layar Pega lain yang menulisinya —
// sehingga `P-1` terpenuhi.
//
// Empat langkah yang dijalankan `Activity/SetStsSalvagePNC_act-Act.xml` dan TIDAK dijalankan
// di sini, seluruhnya menembak sesuatu di luar basis data ini: unggah berkas ke penyimpanan
// eksternal, kirim ke balai lelang SimasBid, kirim email, dan sisipkan salinan JSON klaim.
// Ketiadaannya dinyatakan ke pengguna lewat PlannedDifferences, bukan disamarkan.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxsalvage"
)

// Service melayani modul Inbox Salvage.
type Service struct {
	repoSelector inboxsalvage.RepoSelector
	logger       *slog.Logger

	// notifier dan auction BOLEH nil, dan nil berarti "tidak dipasang" — bukan "gagal".
	//
	// Keduanya menyentuh sistem di luar aplikasi ini, dan keduanya dapat belum
	// terkonfigurasi di sebuah lingkungan tanpa itu berarti ada yang rusak. Yang TIDAK
	// boleh terjadi adalah menyamarkannya: Submit tetap berhasil, dan jawabannya menyebut
	// apa yang tidak terjadi.
	notifier inboxsalvage.Notifier
	auction  inboxsalvage.AuctionHouse
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxsalvage.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger

	// Notifier mengirim pemberitahuan "pengajuan salvage tersimpan".
	//
	// Boleh nil. Lihat Service.
	Notifier inboxsalvage.Notifier

	// Auction mengirim pengajuan ke balai lelang SimasBid.
	//
	// Boleh nil. Lihat Service.
	Auction inboxsalvage.AuctionHouse
}

// NewService membentuk layanan modul Inbox Salvage.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxsalvage/usecase: RepoSelector wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		logger:       o.Logger,
		notifier:     o.Notifier,
		auction:      o.Auction,
	}, nil
}
// Metadata adalah keterangan layar yang tidak bergantung isi daftar.
type Metadata struct {
	// Tabs adalah daftar yang DITAWARKAN layar beserta kolomnya — sembilan dari tiga
	// belas. Lihat inboxsalvage.HiddenTabs untuk keempat yang tidak, beserta alasannya.
	Tabs []inboxsalvage.Tab

	// DefaultTab adalah daftar yang terbuka pertama kali.
	DefaultTab string

	// StatusOptions adalah isi daftar pilihan "Status Salvage" pada form Tambah.
	StatusOptions []inboxsalvage.StatusOption

	// UploadColumns adalah judul kolom berkas "Upload Detail Salvage".
	//
	// Ia dikirim ke layar supaya keterangan di dekat tombol unggah menyebut kolom yang
	// BENAR-BENAR dibaca — bukan daftar yang ditulis ulang di layar dan kelak berbeda dari
	// yang dibaca pengurai.
	UploadColumns []string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string

	// CurrencyOptions adalah isi daftar pilihan "Mata Uang" pada form Tambah.
	//
	// Berbeda dari isian lain di struct ini, ia dibaca dari BASIS DATA — `POOLDATA.CURRENCY`,
	// tabel yang sama yang dibaca Report Definition `SelectCurrency_RD` di layar lama.
	//
	// KOSONG bila tabelnya tidak terbaca, dan itu bukan galat: lihat Metadata().
	CurrencyOptions []inboxsalvage.CurrencyOption
}

// Metadata menyerahkan keterangan layar.
//
// # Kenapa ia kini menerima ctx dan portal
//
// Karena satu isiannya BUKAN bentuk layar melainkan data entitas: pilihan "Mata Uang"
// dibaca dari `POOLDATA.CURRENCY`, dan tabel itu hidup di database portal yang sedang
// dipilih. Sisanya tetap konstanta.
//
// # Kenapa kegagalan membacanya TIDAK menggagalkan Metadata
//
// Karena kolom "Mata Uang" adalah satu isian dari tujuh belas, sementara Metadata adalah
// jawaban yang menentukan apakah layar ini dapat digambar sama sekali. Menjadikannya galat
// berarti satu master yang tidak terbaca menutup seluruh Inbox Salvage — kelas kegagalan
// yang sudah pernah terjadi di modul ini dan tidak boleh diulang.
//
// Yang terjadi sebagai gantinya: daftarnya kosong, kolomnya tetap digambar tanpa tanda
// wajib, dan sebabnya ditulis ke jejak log supaya tidak hilang diam-diam.
func (s *Service) Metadata(ctx context.Context, portalAlias string) Metadata {
	columns := make([]string, 0,
		len(inboxsalvage.RequiredUploadColumn)+len(inboxsalvage.OptionalUploadColumn))
	columns = append(columns, inboxsalvage.RequiredUploadColumn...)
	columns = append(columns, inboxsalvage.OptionalUploadColumn...)

	return Metadata{
		Tabs:               inboxsalvage.Tabs(),
		DefaultTab:         inboxsalvage.DefaultTab,
		StatusOptions:      inboxsalvage.StatusOptions(),
		UploadColumns:      columns,
		PlannedDifferences: inboxsalvage.PlannedDifferences,
		CurrencyOptions:    s.currencies(ctx, portalAlias),
	}
}

// currencies membaca pilihan mata uang, dan MENELAN kegagalannya ke dalam jejak log.
//
// Penelanan itu disengaja dan dibatasi di satu tempat ini saja — lihat alasannya pada
// Metadata. Ia satu-satunya tempat di modul ini yang memperlakukan galat basis data
// sebagai keadaan yang dapat dilanjutkan.
func (s *Service) currencies(
	ctx context.Context,
	portalAlias string,
) []inboxsalvage.CurrencyOption {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		s.log(ctx, "pilihan mata uang dilewati: portal tidak dapat dipilih",
			"portal", portalAlias, "galat", err.Error())
		return []inboxsalvage.CurrencyOption{}
	}

	options, err := repo.Currencies(ctx)
	if err != nil {
		s.log(ctx, "pilihan mata uang tidak terbaca",
			"portal", portalAlias, "galat", err.Error())
		return []inboxsalvage.CurrencyOption{}
	}

	return options
}

// log menulis satu baris jejak bila logger tersedia.
func (s *Service) log(ctx context.Context, message string, attrs ...any) {
	if s.logger == nil {
		return
	}
	s.logger.WarnContext(ctx, message, attrs...)
}

// Listed adalah isi satu daftar beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxsalvage.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim:
	// daftar yang diminta kosong menjadi daftar bawaan, dan layar harus tahu daftar mana
	// yang sebenarnya dijawab.
	Query inboxsalvage.Query
}

// List mengambil isi satu daftar.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
	input inboxsalvage.QueryInput,
	page inboxsalvage.Pagination,
) (Listed, error) {
	query, err := inboxsalvage.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi daftar %s: %w", query.Tab.Code, err)
	}

	// SETIAP pembukaan dicatat, bukan hanya yang mencurigakan.
	//
	// Seluruh baris layar ini memuat nomor klaim DAN nilai uang — nilai pengajuan PIC,
	// nilai request balai lelang, nilai penawaran. Selama pemeriksaan peran belum ada
	// (`TKT-F3-004`), jejak inilah satu-satunya hal yang menyatakan siapa yang membukanya,
	// dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru untuk
	// keadaan seperti ini.
	//
	// Kata kunci pencarian TIDAK ikut dicatat: ia dapat memuat nomor klaim, dan nomor
	// klaim adalah data nasabah (`D-69`). Yang dicatat adalah APAKAH pengguna mencari.
	if s.logger != nil {
		s.logger.Info(
			"daftar salvage dibuka",
			slog.String("modul", "inbox-salvage"),
			slog.String("daftar", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Bool("mencari", query.Search != ""),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}

// Counts mengambil tabel ringkas "Status Salvage / Jumlah".
//
// Ia terpisah dari List, dan itu disengaja: tabel ringkasnya TIDAK berubah saat pengguna
// berpindah daftar, sehingga layar dapat menyimpannya lebih lama daripada isi tabel. Satu
// pemanggilan yang mengembalikan keduanya sekaligus akan memaksa keempat belas hitungannya
// dijalankan ulang setiap kali pengguna membuka tab lain.
func (s *Service) Counts(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
) ([]inboxsalvage.StatusCount, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return nil, inboxsalvage.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	counts, err := repo.Counts(ctx, cleanCaller)
	if err != nil {
		return nil, fmt.Errorf("mengambil tabel ringkas salvage: %w", err)
	}
	return counts, nil
}

// Detail mengambil isi panel "Detail Salvage" untuk satu pengajuan.
//
// # Kenapa ia TIDAK memeriksa apakah pengajuannya ada di daftar pemanggil
//
// Karena panel dibuka dengan ID, bukan lewat daftar — begitu pula di Pega, tempat tombolnya
// mengirim `IDSALVAGE` dan tidak satu pun penyaring daftar ikut. Pengajuan yang sudah
// berpindah status sejak daftarnya dimuat tetap harus dapat dibuka; menolaknya akan membuat
// layar gagal justru pada keadaan yang paling sering terjadi — checker baru saja
// menyetujuinya.
//
// Batas yang tetap berlaku adalah PORTAL: ID-nya dicari di basis data entitas yang sedang
// dipilih, dan ID milik entitas lain menghasilkan "tidak ditemukan" — bukan diam-diam
// dilayani koneksi lain (`R-20`).
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
	key inboxsalvage.DetailKey,
	reference string,
) (inboxsalvage.Detail, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	var detail inboxsalvage.Detail
	switch key {
	case inboxsalvage.DetailKeyClaim:
		detail, err = repo.DetailByClaim(ctx, reference)
	default:
		detail, err = repo.Detail(ctx, reference)
	}

	if err != nil {
		if errors.Is(err, inboxsalvage.ErrRowNotFound) {
			return inboxsalvage.Detail{}, err
		}
		return inboxsalvage.Detail{}, fmt.Errorf("mengambil detail salvage: %w", err)
	}

	if s.logger != nil {
		// Nomor klaim TIDAK dicatat; ID pengajuan dicatat.
		//
		// Nomor klaim adalah data nasabah (`D-69`), sedangkan ID pengajuan adalah kunci
		// teknis yang tidak menyebutkan siapa pun. Yang dibutuhkan penelusuran adalah
		// "pengajuan mana yang dibuka", bukan "klaim siapa".
		s.logger.Info("detail salvage dibuka",
			slog.String("modul", "inbox-salvage"),
			slog.String("kunci", string(key)),
			slog.String("id_salvage", detail.SalvageID),
			slog.Bool("ada_pengajuan", detail.HasSubmission),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias),
			slog.Int("barang", len(detail.Items)))
	}

	return detail, nil
}

// Created adalah hasil penyimpanan satu pengajuan.
type Created struct {
	// SalvageID adalah ID yang terbit.
	//
	// Di sistem lama nilai ini keluar lewat parameter `ErrMsg OUT` yang SEKALIGUS membawa
	// pesan galat (`Database/INSERT_SALVAGE.prc:26`) — kontrak yang `D-68` nyatakan tidak
	// dibawa. Di sini ID ada di sini, dan kegagalan ada di galat.
	SalvageID string

	// ItemCount adalah jumlah baris Detail Item Salvage yang ikut tersimpan.
	ItemCount int

	// Auction menyatakan apa yang terjadi pada pengiriman ke balai lelang.
	Auction AuctionOutcome

	// Notification menyatakan apa yang terjadi pada pemberitahuan surel.
	Notification NotificationOutcome
}

// AuctionOutcome adalah hasil pengiriman satu pengajuan ke balai lelang.
//
// # Kenapa TIGA keadaan, bukan berhasil-gagal
//
// Karena "tidak dipasang" dan "dipasang lalu gagal" menuntut tindakan yang berbeda dari
// orang yang berbeda. Yang pertama urusan Tim Infra — alamatnya belum diisi di lingkungan
// ini. Yang kedua urusan petugas — pengajuannya perlu dikirim ulang. Menyatukan keduanya
// menjadi satu penanda "tidak terkirim" membuat petugas menunggu perbaikan yang bukan
// miliknya, atau sebaliknya.
type AuctionOutcome struct {
	// Attempted bernilai salah bila AuctionHouse memang tidak dipasang.
	Attempted bool

	// Accepted bernilai benar hanya bila balai lelang MENERIMA pengajuan ini.
	Accepted bool

	// AuctionID adalah nomor dari balai lelang, bila ada.
	AuctionID string

	// Message adalah jawaban balai lelang, atau sebab kegagalannya.
	//
	// Ia dibawa sampai ke layar, karena petugaslah yang memutuskan apakah pengajuan ini
	// perlu dikirim ulang — dan ia tidak dapat memutuskannya dari penanda biner.
	Message string
}

// NotificationOutcome adalah hasil pengiriman pemberitahuan surel.
type NotificationOutcome struct {
	// Attempted bernilai salah bila Notifier memang tidak dipasang.
	Attempted bool

	// Sent bernilai benar bila surelnya benar-benar terkirim.
	Sent bool
}

// auctionEntityFlag memetakan portal menjadi `Lelang.Flag` pada badan permintaan SimasBid.
//
// Nilainya diambil apa adanya dari `Activity/Insert_salvageToSimasBid-Act.xml`:
//
//	@if(TempGetApp.LSC_ID=="SIMASNET","ASI","ASM")
//
// CATATAN. Ini bentuk hardcode yang `ADR-0025` perintahkan menjadi master atau
// konfigurasi, sama seperti `portalsRegisteredWithCashier` di modul Master Rekening. Ia
// dibiarkan sebagai konstanta yang TERLIHAT dan bernama alih-alih tersebar di dalam
// percabangan, supaya saat master portal siap yang perlu diubah hanya satu tempat ini.
var auctionEntityFlag = map[string]string{
	"SIMASNET": "ASI",
}

// defaultAuctionEntityFlag berlaku bagi portal yang tidak disebut di atas.
const defaultAuctionEntityFlag = "ASM"

// portalWithCompositeAuctionID menyebut portal yang `Lelang.IDObject`-nya berbentuk
// `<nomor klaim>/<id salvage>`, bukan ID detail salvage.
//
// Diambil dari ekspresi yang sama:
//
//	@if(TempGetApp.LSC_ID=="SIMASNET", Param.NoKlaim+"/"+Param.IDSalvage,
//	    Param.IDDetailSalvage)
//
// Bentuk kedua — ID detail salvage — menunjuk SATU BARANG, bukan satu pengajuan. Lihat
// catatan pada Service.sendToAuction.
var portalWithCompositeAuctionID = map[string]bool{
	"SIMASNET": true,
}

// Create menyimpan satu pengajuan salvage beserta detail itemnya.
//
// # Urutannya: simpan, kirim ke balai lelang, kirim surel
//
// Sama seperti `Activity/SetStsSalvagePNC_act-Act.xml`, yang menjalankan
// `Insert_salvageToSimasBid` pada langkah 19 dan `UploadingFileUntukSendByEmail` pada
// langkah 23 — keduanya SESUDAH penyimpanan.
//
// # Kedua akibat sampingan TIDAK dapat menggagalkan penyimpanan
//
// Pengajuan yang sudah tersimpan tetap tersimpan meski balai lelang tidak dapat dihubungi
// dan meski server surel sedang mati. Pilihan ini berpihak pada pekerjaan pengguna:
// kehilangan tujuh belas isian yang sudah diketik karena jaringan ke sistem lain sedang
// bermasalah adalah kerugian yang jauh lebih besar daripada satu pengiriman yang harus
// diulang.
//
// Yang menggantikan kegagalan diam-diam adalah JAWABAN YANG MENYEBUTKANNYA. Layar
// menuliskan apa yang terjadi dan apa yang tidak, sehingga petugas tahu apakah ia masih
// perlu mengabari seseorang.
//
// # Keduanya di LUAR transaksi basis data
//
// `09-API-STRATEGY.md` §8.2: pemanggilan sistem eksternal tidak boleh berada di dalam
// transaksi — kegagalan jaringan tidak boleh menahan kunci baris. Repo.Create sudah
// menutup transaksinya sebelum baris mana pun di bawah ini berjalan.
func (s *Service) Create(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
	input inboxsalvage.FormInput,
) (Created, error) {
	form, err := inboxsalvage.NewForm(input, caller)
	if err != nil {
		return Created{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Created{}, err
	}

	salvageID, err := repo.Create(ctx, form)
	if err != nil {
		return Created{}, fmt.Errorf("menyimpan pengajuan salvage: %w", err)
	}

	// Penyimpanan dicatat SELALU, dan lebih rinci daripada pembacaan.
	//
	// Ia menulis nilai uang ke tabel yang dibaca laporan, dan `D-59` menetapkan tidak ada
	// pemisahan tugas formal — orang yang sama dapat membuat dan menyetujui. Jejak inilah
	// satu-satunya kontrol pengimbangnya.
	//
	// Nomor klaim TIDAK ikut dicatat: ia data nasabah (`D-69`). Yang dicatat adalah ID
	// salvage, yang menunjuk barisnya tanpa menyebut klaim siapa.
	if s.logger != nil {
		s.logger.Info(
			"pengajuan salvage disimpan",
			slog.String("modul", "inbox-salvage"),
			slog.String("id_salvage", salvageID),
			slog.String("mode", string(form.Mode)),
			slog.String("pemanggil", form.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Int("detail_item", len(form.Items)),
		)
	}

	created := Created{SalvageID: salvageID, ItemCount: len(form.Items)}
	created.Auction = s.sendToAuction(ctx, repo, portalAlias, form, salvageID)
	created.Notification = s.notifySubmission(ctx, repo, portalAlias, form, salvageID)
	return created, nil
}

// sendToAuction mengirim satu pengajuan ke balai lelang, lalu mencatat jawabannya.
//
// # Satu pengajuan, satu pengiriman
//
// Sistem lama mengirim per BARANG: `Lelang.IDObject` pada portal non-Insurtech berisi ID
// detail salvage, dan pemanggilnya memutari daftar barang. Di sini satu pengajuan dikirim
// sekali, dengan ID pengajuan sebagai kuncinya.
//
// Perbedaannya nyata dan dinyatakan di PlannedDifferences. Alasan memilihnya: pengiriman
// per barang membuat satu Submit menghasilkan banyak pengiriman yang sebagian dapat
// berhasil dan sebagian gagal, sementara `STSTRANSFER` hanya punya SATU nilai untuk
// seluruh pengajuan — sehingga keadaan setengah terkirim tidak dapat dinyatakan sama
// sekali. Yang pertama membuat cacat itu terlihat; yang kedua menyembunyikannya.
func (s *Service) sendToAuction(
	ctx context.Context,
	repo inboxsalvage.Repo,
	portalAlias string,
	form inboxsalvage.Form,
	salvageID string,
) AuctionOutcome {
	if s.auction == nil {
		return AuctionOutcome{Message: inboxsalvage.ErrAuctionNotAvailable.Error()}
	}

	flag := auctionEntityFlag[portalAlias]
	if flag == "" {
		flag = defaultAuctionEntityFlag
	}

	itemID := salvageID
	if portalWithCompositeAuctionID[portalAlias] {
		itemID = form.ClaimNo + "/" + salvageID
	}

	submission := inboxsalvage.AuctionSubmission{
		ItemID:          itemID,
		ClaimNo:         form.ClaimNo,
		SalvageID:       salvageID,
		ItemName:        form.ObjectName,
		ItemDescription: form.Remark,
		Location:        form.Location,
		InJabodetabek:   form.InJabodetabek,
		Price:           form.MinimumValue,
		EntityFlag:      flag,
	}

	outcome := AuctionOutcome{Attempted: true}

	receipt, err := s.auction.SendSalvage(ctx, submission)
	if err != nil {
		outcome.Message = err.Error()
		s.warn("pengiriman pengajuan salvage ke balai lelang gagal",
			salvageID, portalAlias, err.Error())
		return outcome
	}

	outcome.Accepted = receipt.Accepted()
	outcome.AuctionID = receipt.AuctionID
	outcome.Message = receipt.Message

	// Jawabannya DICATAT apa pun isinya, termasuk penolakan.
	//
	// Penolakan yang tidak dicatat membuat baris tetap bertanda "belum dikirim", dan
	// petugas mengirimnya berulang kali ke balai lelang yang sudah menolaknya.
	if err := repo.MarkSentToAuction(ctx, salvageID, receipt); err != nil {
		s.warn("jawaban balai lelang gagal dicatat pada pengajuan",
			salvageID, portalAlias, err.Error())
	}

	if s.logger != nil {
		s.logger.Info("pengajuan salvage dikirim ke balai lelang",
			slog.String("modul", "inbox-salvage"),
			slog.String("id_salvage", salvageID),
			slog.String("portal", portalAlias),
			slog.Bool("diterima", outcome.Accepted),
			slog.String("id_balai_lelang", outcome.AuctionID),
		)
	}
	return outcome
}

// notifySubmission mengirim pemberitahuan "pengajuan salvage tersimpan".
//
// # Kenapa rinciannya DIBACA ULANG, bukan disalin dari form
//
// Karena subjek surel memuat NAMA TERTANGGUNG, dan form Tambah tidak memuatnya — ia milik
// klaimnya, bukan milik pengajuannya. Sistem lama membacanya dari
// `pyWorkPage.Policy.QQName`, halaman yang sudah termuat di sana sepanjang alur; di sini
// halaman itu tidak ada, dan satu pembacaan adalah harga yang dibayar untuk subjek yang
// sama.
//
// Pembacaannya di LUAR transaksi dan hanya terjadi bila Notifier memang dipasang.
// Kegagalannya tidak menggagalkan apa pun — surel tetap dikirim, hanya tanpa ekor subjek.
func (s *Service) notifySubmission(
	ctx context.Context,
	repo inboxsalvage.Repo,
	portalAlias string,
	form inboxsalvage.Form,
	salvageID string,
) NotificationOutcome {
	if s.notifier == nil {
		return NotificationOutcome{}
	}

	notice := inboxsalvage.SubmissionNotice{
		SalvageID:    salvageID,
		ClaimNo:      form.ClaimNo,
		SalvageType:  form.SalvageType,
		Location:     form.Location,
		Currency:     form.Currency,
		MinimumValue: form.MinimumValue,
		OfferValue:   form.OfferValue,
		Remark:       form.Remark,
		Submitter:    form.Caller.Login,
		ItemCount:    len(form.Items),
	}
	if form.Email != "" {
		notice.ExtraRecipients = []string{form.Email}
	}

	if detail, err := repo.Detail(ctx, salvageID); err == nil {
		notice.InsuredName = detail.BusinessName
	}

	outcome := NotificationOutcome{Attempted: true}

	if err := s.notifier.NotifySalvageSubmitted(ctx, notice); err != nil {
		// WARN, bukan ERROR: penyimpanannya berhasil, dan yang gagal adalah akibat
		// sampingannya. Mencatatnya sebagai ERROR akan menyamakannya dengan kegagalan
		// yang membuat pengguna kehilangan pekerjaan (`11-CROSSCUTTING.md` §2.2).
		s.warn("pemberitahuan pengajuan salvage gagal dikirim",
			salvageID, portalAlias, err.Error())
		return outcome
	}

	outcome.Sent = true
	return outcome
}

// warn menulis satu baris peringatan dengan isian yang sama di setiap tempat.
//
// Nomor klaim TIDAK ikut, dengan alasan yang sama seperti pada pencatatan penyimpanan: ia
// data nasabah (`D-69`). Yang menunjuk barisnya adalah ID salvage.
func (s *Service) warn(message, salvageID, portalAlias, reason string) {
	if s.logger == nil {
		return
	}
	s.logger.Warn(message,
		slog.String("modul", "inbox-salvage"),
		slog.String("id_salvage", salvageID),
		slog.String("portal", portalAlias),
		slog.String("galat", reason),
	)
}
// AttachedDocuments adalah hasil satu permintaan unggah, satu baris per berkas.
type AttachedDocuments struct {
	Items []inboxsalvage.AttachedDocument
}

// AttachDocuments menyimpan berkas-berkas dari modal "UploadDocument_Salvage".
//
// # Kenapa satu per satu, bukan sekaligus
//
// Karena begitulah layar lama: `SaveFilePenunjangBySalvage` memutari
// `dragDropFileUpload.pxResults` dan menjalankan seluruh rantai simpan untuk SETIAP
// berkas. Nomor urut nama berkasnya pun dihitung ulang tiap putaran, sehingga menggabung
// keduanya menjadi satu transaksi akan mengubah penomorannya.
//
// # Berkas pertama yang gagal MENGHENTIKAN sisanya
//
// Yang sudah tersimpan tidak ditarik kembali — tiap berkas punya transaksinya sendiri,
// sama seperti di Pega. Pemanggil diberi tahu berapa yang berhasil, supaya pengguna tahu
// mana yang perlu diulang alih-alih mengunggah semuanya lagi.
func (s *Service) AttachDocuments(
	ctx context.Context,
	portalAlias string,
	caller inboxsalvage.Caller,
	docs []inboxsalvage.DocumentUpload,
) (AttachedDocuments, error) {
	if len(docs) == 0 {
		return AttachedDocuments{}, inboxsalvage.ErrNoDocument
	}
	if len(docs) > inboxsalvage.MaxDocumentPerUpload {
		return AttachedDocuments{}, inboxsalvage.ErrTooManyDocuments
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return AttachedDocuments{}, err
	}

	result := AttachedDocuments{Items: []inboxsalvage.AttachedDocument{}}

	for index, doc := range docs {
		doc.Operator = caller.Clean().Login

		saved, err := repo.AttachDocument(ctx, doc)
		if err != nil {
			return result, fmt.Errorf("menyimpan dokumen ke-%d: %w", index+1, err)
		}
		result.Items = append(result.Items, saved)

		// Setiap dokumen dicatat, dan alasannya sama dengan pencatatan penyimpanan
		// pengajuan: tidak ada pemisahan tugas formal (`D-59`), sehingga jejak adalah
		// satu-satunya kontrol pengimbang.
		//
		// Nama berkas dan nomor klaim TIDAK dicatat — keduanya dapat memuat nama
		// tertanggung (`D-69`). Yang dicatat adalah ID yang menunjuk barisnya.
		if s.logger != nil {
			s.logger.Info(
				"dokumen salvage disimpan",
				slog.String("modul", "inbox-salvage"),
				slog.String("data_id", saved.DataID),
				slog.String("image_id", saved.ImageID),
				slog.Bool("tertaut_salvage", saved.LinkedToSalvage),
				slog.String("pemanggil", caller.Login),
				slog.String("portal", portalAlias),
			)
		}
	}

	return result, nil
}
