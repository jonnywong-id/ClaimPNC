package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	"claim-pnc/internal/reportkpi"
)

// Berkas ini memenuhi bagian tab **KPI Admin** dari seam reportkpi.Repo tanpa basis data.
//
// # Kenapa ia MENGHITUNG, bukan mengembalikan angka tetap
//
// Karena kartu skornya adalah hasil hitungan, dan hitungan itulah yang paling mudah salah.
// Penyimpanan yang mengembalikan angka tetap akan membuat layar terlihat benar sementara
// tangga nilai, bobot, dan rasio pencapaiannya tidak pernah teruji sekali pun.
//
// Empat aturan ditiru dengan sengaja, dan ketiganya adalah tempat pengisi memori paling
// mudah menyimpang:
//
//   - Tangga nilai 1–5 memakai ambang yang SAMA dengan `CASE` di kueri, termasuk cabang
//     `= 1` yang tidak pernah tercapai karena `BETWEEN 0.5 AND 1` menangkapnya lebih dulu.
//   - Pembagi nol menghasilkan nilai KOSONG, bukan nol dan bukan panik.
//   - Ambang "melewati SLA" berbeda antar kelompok: `> 1` pada NON-MBU, `> 0` pada PA.
//   - Rentang tanggal SETENGAH TERBUKA, sama dengan kuerinya.

// AdminRow adalah satu klaim yang ditangani tim admin.
//
// Ia sengaja menyimpan bahan MENTAH — umur dalam hari kerja, penanda leader, kelompok —
// bukan hasil hitungan, supaya kartu skor benar-benar dihitung di sini seperti di basis
// data.
type AdminRow struct {
	Group reportkpi.AdminGroup

	ClaimNumber  string
	PolicyNumber string
	BusinessName string
	TeamFlag     string

	// Leader menyatakan baris ini milik kelompok leader (`reinsurer = '1'` di basis data).
	// Hanya berarti pada NON-MBU.
	Leader bool

	RegisterDate   string // `YYYY-MM-DD` — yang disaring periode
	TransferDate   string
	ReceiveDate    string
	LODReceiveDate string
	AcceptanceDate string

	// RegisterAging dan PaymentAging adalah TAT dalam hari kerja, hasil
	// `get_working_hours(...)/28800` di basis data.
	RegisterAging float64
	PaymentAging  float64

	// HasPayment menyatakan baris ini punya akseptasi — `NOAKSEPTASI IS NOT NULL`.
	// Hanya baris seperti ini yang ikut pencacah pembayaran.
	HasPayment bool

	AdminName   string
	ClaimStatus string

	// InLegacyPaymentWindow menyatakan baris ini jatuh di dalam rentang 2023 yang
	// tertanam pada pembagi `total_pembayaran_pa`. Lihat reportkpi_admin.sql.
	InLegacyPaymentWindow bool
}

// AdminTotals menghitung kartu skor dari baris contoh.
func (s *Store) AdminTotals(
	_ context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if q.Group == reportkpi.AdminGroupPA {
		return s.adminTotalsPA(q), nil
	}
	return s.adminTotalsNonMBU(q), nil
}

func (s *Store) adminTotalsNonMBU(q reportkpi.AdminQuery) reportkpi.AdminTotals {
	var leaderOver, leaderTotal, memberOver, memberTotal float64

	for _, row := range s.adminRows {
		if !adminMatches(row, q) {
			continue
		}
		if row.Leader {
			leaderTotal++
			if row.RegisterAging > 1 {
				leaderOver++
			}
			continue
		}
		memberTotal++
		if row.RegisterAging > 1 {
			memberOver++
		}
	}

	leaderPercent := ratioPercent(leaderOver, leaderTotal)
	memberPercent := ratioPercent(memberOver, memberTotal)
	leaderScore := slaScore(leaderPercent)
	memberScore := slaScore(memberPercent)

	totals := reportkpi.AdminTotals{
		LeaderOverSLA: reportkpi.NewScore(leaderOver),
		LeaderTotal:   reportkpi.NewScore(leaderTotal),
		LeaderPercent: leaderPercent,
		LeaderScore:   leaderScore,
		MemberOverSLA: reportkpi.NewScore(memberOver),
		MemberTotal:   reportkpi.NewScore(memberTotal),
		MemberPercent: memberPercent,
		MemberScore:   memberScore,
	}

	// Subtotal dan seterusnya hanya berarti bila kedua nilainya ada. Menghitungnya dari
	// nilai yang tidak ada akan menghasilkan angka yang tampak sah — dan itu justru
	// kegagalan yang paling sulit terlihat pada kartu skor.
	if leaderScore.Present && memberScore.Present {
		leaderSubtotal := (leaderScore.Value / 5) * reportkpi.AdminLeaderWeight * 100
		memberSubtotal := (memberScore.Value / 5) * reportkpi.AdminMemberWeight * 100
		total := leaderSubtotal + memberSubtotal

		totals.LeaderSubtotal = reportkpi.NewScore(leaderSubtotal)
		totals.MemberSubtotal = reportkpi.NewScore(memberSubtotal)
		totals.QuantitativeTotal = reportkpi.NewScore(total)
		totals.AchievementRatio = reportkpi.NewScore(
			roundTo2(total / ((3.0 / 5.0) * 90)))
	}

	return totals
}

func (s *Store) adminTotalsPA(q reportkpi.AdminQuery) reportkpi.AdminTotals {
	var registerOver, claimTotal, paymentOver, paymentTotal float64

	for _, row := range s.adminRows {
		if !adminMatches(row, q) {
			continue
		}
		claimTotal++
		// Ambangnya `> 0`, BUKAN `> 1` seperti NON-MBU. Ditiru apa adanya.
		if row.RegisterAging > 0 {
			registerOver++
		}
		if row.HasPayment && row.PaymentAging > 0 {
			paymentOver++
		}
	}

	// Pembagi `total_pembayaran_pa` TIDAK menyaring periode yang dipilih pengguna; ia
	// terkunci pada rentang 2023 di dalam kueri. Ditiru di sini pula — kalau tidak, layar
	// pengembangan akan menampilkan angka yang berbeda dari produksi tanpa sebab yang
	// terlihat.
	for _, row := range s.adminRows {
		if row.Group != reportkpi.AdminGroupPA || !row.InLegacyPaymentWindow {
			continue
		}
		if row.HasPayment && row.PaymentAging > 0 {
			paymentTotal++
		}
	}

	return reportkpi.AdminTotals{
		RegisterOverSLA: reportkpi.NewScore(registerOver),
		RegisterTotal:   reportkpi.NewScore(claimTotal),
		RegisterScore:   slaScore(ratioPercent(registerOver, claimTotal)),
		PaymentOverSLA:  reportkpi.NewScore(paymentOver),
		PaymentTotal:    reportkpi.NewScore(paymentTotal),
		PaymentScore:    slaScore(ratioPercent(paymentOver, claimTotal)),
	}
}

// AdminDetail mengambil satu halaman grid rincian.
func (s *Store) AdminDetail(
	_ context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := []AdminRow{}
	for _, row := range s.adminRows {
		if adminMatches(row, q) {
			matched = append(matched, row)
		}
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].RegisterDate != matched[j].RegisterDate {
			return matched[i].RegisterDate > matched[j].RegisterDate
		}
		return matched[i].ClaimNumber < matched[j].ClaimNumber
	})

	clean := page.Normalize()
	total := len(matched)

	from := clean.Offset()
	if from > total {
		from = total
	}
	to := from + clean.Size
	if to > total {
		to = total
	}

	rows := make([]reportkpi.AdminDetailRow, 0, to-from)
	for _, row := range matched[from:to] {
		rows = append(rows, reportkpi.AdminDetailRow{
			ClaimNumber:    row.ClaimNumber,
			PolicyNumber:   row.PolicyNumber,
			BusinessName:   row.BusinessName,
			RegisterDate:   row.RegisterDate,
			TransferDate:   row.TransferDate,
			TeamFlag:       row.TeamFlag,
			RegisterAging:  reportkpi.NewScore(row.RegisterAging),
			ReceiveDate:    row.ReceiveDate,
			LODReceiveDate: row.LODReceiveDate,
			AcceptanceDate: row.AcceptanceDate,
			PaymentAging:   reportkpi.NewScore(row.PaymentAging),
			RegisterSLA:    slaLabel(row.RegisterAging),
			PaymentSLA:     slaLabel(row.PaymentAging),
			AdminName:      row.AdminName,
			ClaimStatus:    row.ClaimStatus,
		})
	}

	return reportkpi.AdminDetailPage{Rows: rows, Total: total}, nil
}

// adminMatches menyatakan satu baris lolos kelompok dan periode permintaan.
func adminMatches(row AdminRow, q reportkpi.AdminQuery) bool {
	if row.Group != q.Group {
		return false
	}
	return withinRange(row.RegisterDate, q.Range)
}

// ratioPercent menghitung persentase, atau menyatakan ia tidak dapat dihitung.
//
// Pembagi nol menghasilkan nilai KOSONG. Di basis data, pembagian itu justru menggagalkan
// kueri dengan `ORA-01476`; di sini ia ditahan dan hasilnya dinyatakan tidak ada, sehingga
// layar dapat menggambarkan keadaannya alih-alih menampilkan kegagalan mentah.
func ratioPercent(over, total float64) reportkpi.Score {
	if total == 0 {
		return reportkpi.EmptyScore()
	}
	return reportkpi.NewScore((over / total) * 100)
}

// slaScore menerjemahkan persentase menjadi nilai 1–5.
//
// Tangga ambangnya ditiru kata demi kata dari `CASE` di kueri, TERMASUK keanehannya:
// cabang `= 1` tidak pernah tercapai, karena `BETWEEN 0.5 AND 1` di atasnya sudah
// menangkap nilai 1. Memperbaikinya di sini akan membuat pengisi memori menjawab berbeda
// dari basis data pada satu nilai tepat — dan selisih satu titik itulah yang paling sulit
// ditelusuri.
func slaScore(percent reportkpi.Score) reportkpi.Score {
	if !percent.Present {
		return reportkpi.EmptyScore()
	}

	p := percent.Value
	switch {
	case p < 0.5:
		return reportkpi.NewScore(5)
	case p >= 0.5 && p <= 1:
		return reportkpi.NewScore(4)
	case p > 1 && p <= 1.5:
		return reportkpi.NewScore(2)
	case p > 1.5 && p <= 2:
		return reportkpi.NewScore(1)
	default:
		return reportkpi.NewScore(0)
	}
}

// slaLabel menerjemahkan umur menjadi penanda SLA, seperti `CASE` pada kueri rincian PA.
func slaLabel(aging float64) string {
	if aging > 1 {
		return "TIDAK SLA"
	}
	return "SLA"
}

// adminWithinYear menyatakan sebuah tanggal berada di dalam satu tahun tertentu.
//
// Dipakai contoh untuk menandai baris yang jatuh di rentang 2023 yang tertanam.
func adminWithinYear(date string, year int) bool {
	moment, err := time.Parse("2006-01-02", strings.TrimSpace(date))
	if err != nil {
		return false
	}
	return moment.Year() == year
}
