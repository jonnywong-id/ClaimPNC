// Package usecase mengorkestrasi modul Inbox PLA DLA — layar milik reasuradur.
//
// Layar induk — empat operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List      mengambil isi satu daftar
//	Counts    mengambil tabel ringkas "Status / Jumlah"
//
// Layar rincian ("Detail Claim") — empat lagi, di usecase/detail.go:
//
//	Detail           kepala klaim, grid PLA, grid DLA, dan riwayat komunikasi
//	Documents        dokumen satu nomor pemberitahuan
//	DocumentContent  isi satu dokumen
//	Reply            MENULIS balasan atas satu percakapan
//
// Ekspor TIDAK menjadi operasi kelima: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Perakitannya ada di
// http/export.go.
//
// # SATU LANGKAH MENDAHULUI SETIAP OPERASI, DAN IA MENENTUKAN SEGALANYA
//
// Login pemanggil diterjemahkan lebih dulu menjadi kode reasuradur lewat
// `POOLDATA.T_REINSURER.LOGIN`. Login yang tidak ada di sana menghasilkan
// ErrCallerNotAReinsurer — bukan daftar kosong.
//
// Perbedaan itu bukan kehalusan. Di Pega keduanya terlihat sama: layar kosong tanpa
// keterangan. Petugas internal yang tersesat ke menu ini menyimpulkan sistemnya rusak, dan
// reasuradur yang loginnya belum didaftarkan menunggu pekerjaan yang tidak akan pernah
// muncul. Keduanya menempuh tindakan yang berbeda, dan hanya pesan yang membedakannya yang
// dapat menuntun ke sana.
//
// # SATU operasi MENULIS, dan pelakunya PIHAK LUAR
//
// Layar induknya baca-saja, di Pega maupun di sini. Yang menulis adalah balasan komunikasi
// pada layar rincian — dan itu penulisan pertama di seluruh aplikasi ini yang pelakunya
// bukan pegawai Asuransi Sinar Mas melainkan mitra reasuransi.
//
// Konsekuensinya diambil di usecase/detail.go: setiap balasan DICATAT beserta pelakunya,
// dan pemagaran kepemilikannya berada di dalam pernyataan SQL-nya sendiri — bukan di
// lapisan mana pun di atasnya.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxpladla"
)

// Service melayani modul Inbox PLA DLA.
type Service struct {
	repoSelector inboxpladla.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxpladla.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox PLA DLA.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxpladla/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi daftar.
type Metadata struct {
	Tabs       []inboxpladla.Tab
	DefaultTab string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali: daftar tab dan kolomnya sama di seluruh
// entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:               inboxpladla.Tabs(),
		DefaultTab:         inboxpladla.DefaultTab,
		PlannedDifferences: inboxpladla.PlannedDifferences,
	}
}

// Listed adalah isi satu daftar beserta permintaan yang benar-benar dipakai.
type Listed struct {
	Page  inboxpladla.Page
	Query inboxpladla.Query
}

// List mengambil isi satu daftar.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	input inboxpladla.QueryInput,
	page inboxpladla.Pagination,
) (Listed, error) {
	repo, query, err := s.prepare(ctx, portalAlias, caller, input)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf(
			"mengambil isi daftar %s: %w", query.Tab.Code, err)
	}

	// SETIAP pembukaan dicatat.
	//
	// Barisnya memuat nama tertanggung dan nomor polis milik tertanggung Asuransi Sinar
	// Mas, dan yang membacanya adalah PIHAK LUAR — mitra reasuransi. Di seluruh modul
	// yang sudah dibangun, inilah satu-satunya layar yang datanya keluar dari dinding
	// perusahaan, dan jejak inilah satu-satunya yang mencatat siapa mengambil apa.
	//
	// Kata kunci pencarian TIDAK ikut dicatat: ia dapat memuat nomor klaim, dan nomor
	// klaim adalah data nasabah (`D-69`). Kode reasuradurnya DICATAT — ia bukan data
	// nasabah, dan tanpanya jejaknya tidak dapat menjawab "data milik mitra mana yang
	// diambil".
	if s.logger != nil {
		s.logger.Info(
			"daftar PLA/DLA reasuradur dibuka",
			slog.String("modul", "inbox-pla-dla"),
			slog.String("daftar", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.Any("kode_reasuradur", query.EffectiveReinsurerCodes()),
			slog.String("portal", portalAlias),
			slog.Bool("mencari", query.Search != ""),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}

// Counts mengambil tabel ringkas "Status / Jumlah".
//
// Ia terpisah dari List, dan itu disengaja: isinya tidak berubah saat pengguna berpindah
// HALAMAN, sehingga layar dapat menyimpannya lebih lama daripada isi tabel. Ia memang
// berubah saat pengguna berpindah TAB atau mengubah pencarian — karena itu penyaringnya
// ikut, bukan diabaikan.
func (s *Service) Counts(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	input inboxpladla.QueryInput,
) ([]inboxpladla.StatusCount, error) {
	repo, query, err := s.prepare(ctx, portalAlias, caller, input)
	if err != nil {
		return nil, err
	}

	counts, err := repo.Counts(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"mengambil tabel ringkas %s: %w", query.Tab.Code, err)
	}
	return counts, nil
}

// ListCounts mengambil tabel **"Status / Jumlah"** — satu baris per DAFTAR.
//
// # Kenapa ia memanggil List, bukan kueri hitung tersendiri
//
// Karena angkanya WAJIB sama dengan isi daftarnya, dan satu-satunya cara menjamin itu
// adalah memakai pernyataan yang sama persis. Keenam kueri daftar sudah membawa
// `COUNT(*) OVER ()`, sehingga meminta satu baris saja sudah mengembalikan jumlah
// seluruhnya — tanpa satu pun SQL baru yang harus dijaga tetap sejalan.
//
// Kueri hitung tersendiri pernah ditempuh untuk StatusCount, dan berkas .sql-nya sendiri
// memperingatkan bahayanya: *"Penyaringnya WAJIB sama persis dengan list_pla. Angka yang
// tidak cocok dengan tabel di bawahnya adalah hal pertama yang dilaporkan pengguna
// sebagai kerusakan."* Peringatan itu dipatuhi di sini dengan tidak membuat kuerinya.
//
// # Biayanya: enam perjalanan ke basis data
//
// Diterima secara sadar. Pega pun memuat SELURUH baris keenam daftar ke klipboard untuk
// menghitung tabel yang sama, sehingga ini tidak lebih mahal daripada layar lama — dan
// profil bebannya memang data besar dengan konkurensi rendah (`D-10`).
//
// # Tampilan XOL tidak ikut dicacah
//
// Ia bukan daftar klaim, dan di Pega pun tidak punya baris di tabel ini: tabelnya berisi
// enam baris, sementara layarnya punya tujuh tampilan.
func (s *Service) ListCounts(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	input inboxpladla.QueryInput,
) ([]inboxpladla.ListCount, error) {
	repo, codes, err := s.reinsurerOf(ctx, portalAlias, caller)
	if err != nil {
		return nil, err
	}

	counts := []inboxpladla.ListCount{}
	for _, tab := range inboxpladla.Tabs() {
		// Kata kunci pencarian DIBAWA, kode tabnya diganti.
		//
		// Tanpa membawa pencariannya, angka di samping daftar akan menyatakan populasi
		// yang berbeda dari tabel yang sedang dilihat pengguna — persis selisih yang
		// paling cepat dilaporkan sebagai kerusakan.
		query, err := inboxpladla.NewQuery(
			inboxpladla.QueryInput{Tab: tab.Code, Search: input.Search}, caller, codes)
		if err != nil {
			return nil, err
		}

		// Satu baris sudah cukup: jumlahnya datang dari `COUNT(*) OVER ()`, bukan dari
		// banyaknya baris yang terkirim.
		page, err := repo.List(ctx, query, inboxpladla.Pagination{Page: 1, Size: 1})
		if err != nil {
			return nil, fmt.Errorf("menghitung isi daftar %s: %w", tab.Code, err)
		}

		counts = append(counts, inboxpladla.ListCount{
			Code:  tab.Code,
			Name:  tab.Name,
			Total: page.Total,
		})
	}

	return counts, nil
}

// prepare menerjemahkan login menjadi kode reasuradur lalu menyusun permintaannya.
//
// Ia dipakai List dan Counts supaya keduanya menyaring dengan cara yang SAMA PERSIS.
// Menyusun keduanya terpisah akan memungkinkan tabel ringkas menghitung populasi yang
// berbeda dari daftarnya — dan angka yang tidak cocok dengan tabel di bawahnya adalah hal
// pertama yang dilaporkan pengguna sebagai kerusakan.
func (s *Service) prepare(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
	input inboxpladla.QueryInput,
) (inboxpladla.Repo, inboxpladla.Query, error) {
	repo, codes, err := s.reinsurerOf(ctx, portalAlias, caller)
	if err != nil {
		return nil, inboxpladla.Query{}, err
	}

	query, err := inboxpladla.NewQuery(input, caller, codes)
	if err != nil {
		return nil, inboxpladla.Query{}, err
	}

	return repo, query, nil
}

// reinsurerOf menerjemahkan login pemanggil menjadi kode reasuradurnya.
//
// Login yang tidak terdaftar menghasilkan ErrCallerNotAReinsurer — bukan senarai kosong
// yang kelak berubah menjadi daftar kosong tanpa keterangan.
func (s *Service) reinsurerOf(
	ctx context.Context,
	portalAlias string,
	caller inboxpladla.Caller,
) (inboxpladla.Repo, []string, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return nil, nil, inboxpladla.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, nil, err
	}

	codes, err := repo.ReinsurerCodes(ctx, cleanCaller.Login)
	if err != nil {
		return nil, nil, fmt.Errorf("mencari kode reasuradur pemanggil: %w", err)
	}

	if len(codes) == 0 {
		// Dicatat, dan ini satu-satunya penolakan di modul ini yang layak dicatat
		// tersendiri: ia menandakan seseorang membuka menu yang bukan untuknya, atau
		// seorang mitra yang pendaftarannya belum lengkap. Keduanya menuntut tindakan
		// administrator, dan tanpa jejak tidak ada yang tahu perlu bertindak.
		if s.logger != nil {
			s.logger.Info(
				"pemanggil bukan reasuradur terdaftar",
				slog.String("modul", "inbox-pla-dla"),
				slog.String("pemanggil", cleanCaller.Login),
				slog.String("portal", portalAlias),
			)
		}
		return nil, nil, inboxpladla.ErrCallerNotAReinsurer
	}

	return repo, codes, nil
}
