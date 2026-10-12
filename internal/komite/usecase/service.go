// Package usecase mengorkestrasi modul Komite: membaca master ambang, menghitung
// penjenjangan, dan melaporkan kesehatan master.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket komite dan tidak tahu
// apa pun soal HTTP maupun SQL.
//
// # Tiga aksi, dan banyak yang sengaja belum ada
//
// Yang ada: melihat tangga ambang, menghitung siapa yang harus menyetujui sebuah nilai,
// dan memeriksa integritas master.
//
// Yang TIDAK ada, beserta alasannya:
//
//   - Mengubah master ambang. Work Owner menetapkan 2026-09-17 bahwa aplikasi ini
//     MEMBACA SAJA POOLDATA.EMAILKOMITE selama masa paralel. `P-1` menetapkan satu tabel
//     hanya boleh ditulis satu sistem, dan tabel ini masih ditulis Pega serta dibaca 17
//     kueri di sana. Memindahkan kepemilikannya menuntut prosedur `D-63` — permintaan
//     tertulis, persetujuan Work Owner, pelaksanaan DBA — dan itu belum ditempuh.
//   - Mencatat keputusan komite. Itu `TKT-B07-002`, yang bergantung pada `B-5` dan
//     `B-6`; keduanya belum ada di aplikasi ini.
//   - Persetujuan otomatis. Itu `TKT-B07-003`, dan Work Owner memutuskan 2026-09-17
//     perilaku itu TIDAK dibawa.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// Service melayani pembacaan master ambang dan perhitungan penjenjangan.
type Service struct {
	repo       komite.Repo
	policy     komite.Policy
	randomizer komite.Randomizer
}

// Options adalah bahan pembentuk Service.
type Options struct {
	Repo komite.Repo

	// Policy boleh dikosongkan; bila kosong dipakai komite.DefaultPolicy.
	//
	// Ia dipasok dari luar karena isinya berbeda per PORTAL: batas pita Rp 100.000.000
	// pada entitas rupiah versus USD 7.000 pada entitas SMI, dan mode satu-penyetuju
	// pada entitas Simasnet versus kumulatif pada yang lain. Menanamkannya di dalam
	// aturan akan membuat satu portal diam-diam memakai aturan portal lain.
	Policy *komite.Policy

	// Randomizer dipakai kebijakan bermode satu-penyetuju. Bila kosong dipakai pemilih
	// tetap, sehingga hasilnya dapat diulang — diam-diam menjadi acak jauh lebih
	// berbahaya daripada diam-diam menjadi tetap.
	Randomizer komite.Randomizer
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.Repo == nil {
		return nil, errors.New("komite/usecase: seam penyimpanan wajib diisi")
	}

	policy := komite.DefaultPolicy()
	if o.Policy != nil {
		policy = *o.Policy
	}
	return &Service{repo: o.Repo, policy: policy, randomizer: o.Randomizer}, nil
}

// Policy mengembalikan aturan pita yang sedang berlaku.
//
// Dikirim ke layar supaya pengguna dapat melihat batas pita yang dipakai, bukan
// menghafalnya. Angka yang menentukan uang tidak boleh hanya hidup di dalam kode tanpa
// pernah terlihat siapa pun.
func (s *Service) Policy() komite.Policy { return s.policy }

// ListThresholds mengembalikan seluruh baris master ambang komite.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan angka: isinya 30 baris dan
// bertambah beberapa baris per tahun. Layar yang datanya besar — inbox dan laporan —
// tidak boleh mengikuti pola ini.
func (s *Service) ListThresholds(ctx context.Context) ([]komite.Threshold, error) {
	list, err := s.repo.ListThresholds(ctx)
	if err != nil {
		return nil, fmt.Errorf("komite/usecase: membaca master ambang: %w", err)
	}
	return list, nil
}

// ListBusinessLines mengembalikan lini bisnis yang punya jenjang persetujuan.
func (s *Service) ListBusinessLines(ctx context.Context) ([]komite.BusinessLine, error) {
	list, err := s.ListThresholds(ctx)
	if err != nil {
		return nil, err
	}
	return komite.ListBusinessLines(list), nil
}

// Tiering menghitung siapa saja yang harus menyetujui sebuah nilai klaim.
//
// Ia menggantikan rangkaian `SetEmailKomite` beserta tiga variannya, ditambah
// `SetListComiteeClaimPerObjAdj` yang menetapkan `KomiteLoop := pxResultCount`.
//
// # Nilai yang diterima adalah nilai yang SUDAH terkonversi
//
// Konversi kurs bukan urusan modul ini. `D-48` menetapkan konversi memakai kurs pada
// TANGGAL KEJADIAN, dan bila kurs untuk tanggal itu tidak tersedia klaimnya DITOLAK —
// tidak ada nilai bawaan. Itu lingkup `TKT-B05-002`, yang belum ada.
//
// Sampai modul itu tiba, nilai masuk dari pemanggil apa adanya. Yang dijaga di sini
// hanyalah bahwa perbandingannya memakai presisi penuh (`I-12`), bukan angka yang sudah
// dibulatkan lebih dulu.
func (s *Service) Tiering(
	ctx context.Context,
	value money.Money,
	line komite.BusinessLine,
) (komite.Tiering, error) {
	return s.TieringWith(ctx, value, line, "")
}

// TieringWith menghitung penyetuju dengan menyebutkan siapa yang mengajukan.
//
// Penginput dikecualikan dari calon penyetuju pada KEDUA mode (Work Owner, 2026-09-18)
// supaya tidak menyetujui pengajuannya sendiri.
func (s *Service) TieringWith(
	ctx context.Context,
	value money.Money,
	line komite.BusinessLine,
	applicant string,
) (komite.Tiering, error) {
	list, err := s.repo.ListThresholds(ctx)
	if err != nil {
		return komite.Tiering{}, fmt.Errorf("komite/usecase: membaca master ambang: %w", err)
	}

	result, err := komite.DetermineWith(value, line, list, s.policy, komite.Options{
		Applicant:  applicant,
		Randomizer: s.randomizer,
	})
	if err != nil {
		// Galat validasi dan lini tak dikenal diteruskan apa adanya supaya transport
		// dapat memetakannya ke status HTTP yang tepat. Membungkusnya di sini akan
		// membuat errors.Is di sana gagal mengenalinya.
		return komite.Tiering{}, err
	}
	return result, nil
}

// Integrity memeriksa kesehatan master ambang dan mengembalikan seluruh temuannya.
//
// Inilah satu-satunya pemakaian `LIMIT_TOP` yang dibenarkan `D-47`, dan ia langsung
// menjawab tiga hal yang selama ini hanya tercatat sebagai pertanyaan terbuka: sampai
// nilai berapa tangga tiap lini membedakan jenjang, apakah ada rentang yang berlubang
// atau menindih, dan di mana urutan menyetujui menjadi tidak pasti.
func (s *Service) Integrity(ctx context.Context) ([]komite.Finding, error) {
	list, err := s.repo.ListThresholds(ctx)
	if err != nil {
		return nil, fmt.Errorf("komite/usecase: membaca master ambang: %w", err)
	}
	return komite.CheckIntegrity(list, s.policy), nil
}
