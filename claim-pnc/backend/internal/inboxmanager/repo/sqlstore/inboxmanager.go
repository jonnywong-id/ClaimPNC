package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxmanager"
)

// Repo membaca dan menuliskan antrean Inbox Manager pada SATU basis data entitas.
//
// Berbeda dari seluruh repo modul inbox lain, repo ini punya operasi yang MENULIS. Yang
// ditulisnya adalah kolom persetujuan pada tabel POOLDATA — tidak ada satu pun pernyataan
// yang menyentuh skema DATAPEGA, yang selama masa paralel dimiliki Pega (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// lineFlags mengubah lini bisnis petugas menjadi lima bendera yang dikirim ke kueri.
//
// # Kenapa bendera, bukan satu nilai yang dibandingkan berkali-kali
//
// Karena driver mengikat argumen menurut urutan kemunculan penanda, bukan menurut nomornya.
// Satu nilai yang dibandingkan lima kali menuntut lima argumen yang sama, dan bentuk itu
// mudah salah hitung — kesalahan yang hanya muncul di Oracle sebagai ORA-01008.
//
// # Lini bisnis yang TIDAK dikenali membuka SELURUH lini, dan itu selisih terencana
//
// Di Pega, petugas yang `pyPosition`-nya bukan salah satu dari keempat nilai tidak memicu satu
// pun Property-Set penyaring, sehingga `tempQuery.MCL_NAME` tetap berisi nilai yang disusun
// untuk TABEL LAIN — potongan SQL yang menyebut `group_panel`, `groupbisnisid`, dan `pic`,
// kolom yang tidak ada di tabel kerja. Dashboard Outstanding-nya karena itu GAGAL, bukan
// menampilkan apa pun.
//
// Perilaku itu tidak direplikasi: yang direplikasi adalah perilaku, bukan cacat yang
// menghasilkan galat. Petugas tanpa lini bisnis yang dikenali melihat seluruh lini pada
// entitasnya — arah yang sama dengan modul Inbox Outstanding, dan aman karena pemisahan
// antarentitas terjadi di tingkat KONEKSI (`ADR-0030`).
func lineFlags(line string) []any {
	clean := strings.ToUpper(strings.TrimSpace(line))

	all, nonMBU, bonding, pa, travel := 0, 0, 0, 0, 0
	switch clean {
	case inboxmanager.LineNonMBU:
		nonMBU = 1
	case inboxmanager.LineBonding:
		bonding = 1
	case inboxmanager.LinePA:
		pa = 1
	case inboxmanager.LineTravel:
		travel = 1
	default:
		all = 1
	}

	return []any{all, nonMBU, bonding, pa, travel}
}

// counterSpec memetakan sebuah tab ke kueri pencacahnya.
//
// Hanya SEPULUH tab yang punya pencacah, dan itu bentuk sistem lama:
// `Activity/CountDashbroardManager` menjalankan tepat sepuluh kueri `Count*`. Tab
// Produktivitas Klaim, Klaim, dan Approval Master tidak punya satu pun — activity itu menulis
// angka tetap `"1"` untuk ketiganya, yakni penanda, bukan hitungan.
var counterSpecs = []struct {
	tab   string
	query string
}{
	{inboxmanager.TabOutstanding, "count_outstanding"},
	{inboxmanager.TabMasterBengkel, "count_bengkel"},
	{inboxmanager.TabMasterPanel, "count_panel"},
	{inboxmanager.TabNomorRangka, "count_nomor_rangka"},
	{inboxmanager.TabMasterSparepart, "count_sparepart"},
	{inboxmanager.TabKategoriSparepart, "count_kategori_sparepart"},
	{inboxmanager.TabTipeSparepart, "count_tipe_sparepart"},
	{inboxmanager.TabGroupingSparepart, "count_grouping_sparepart"},
	{inboxmanager.TabPaymentAkseptasi, "count_payment_akseptasi"},
	{inboxmanager.TabPenolakanKlaim, "count_penolakan_klaim"},
}

// Counters menghitung kesepuluh pencacah di kepala layar.
//
// # Satu sumber yang rusak TIDAK menghilangkan sembilan lainnya
//
// Kegagalan sebuah pencacah dikembalikan sebagai Counter ber-Unavailable, bukan sebagai galat
// yang membatalkan seluruh permintaan. Ini bukan kehati-hatian teoretis:
// `POOLDATA.SPAREPART_HE` adalah view berstatus INVALID saat diperiksa 2026-09-28, sehingga
// tanpa perlakuan ini kepala layar kosong seluruhnya karena satu view yang rusak.
//
// Angka nol yang sesungguhnya berarti "tidak terbaca" tidak dipakai: ia akan membuat penyelia
// mengira antreannya kosong, dan itu kebohongan yang tidak menghasilkan satu pun galat.
func (r *Repo) Counters(
	ctx context.Context,
	caller inboxmanager.Caller,
) ([]inboxmanager.Counter, error) {
	result := []inboxmanager.Counter{}

	for _, spec := range counterSpecs {
		tab, known := inboxmanager.FindTab(spec.tab)
		if !known {
			continue
		}

		counter := inboxmanager.Counter{
			TabCode: tab.Code,
			Label:   tab.Name,
		}
		if tab.Kind == inboxmanager.KindQueue {
			counter.Parent = inboxmanager.TabApprovalMaster
		}

		var args []any
		if spec.query == "count_outstanding" {
			args = lineFlags(caller.LineBusiness)
		}

		var total int
		err := r.db.QueryRowContext(ctx, query(spec.query), args...).Scan(&total)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			counter.Count = 0
		case err != nil:
			counter.Unavailable = unavailableReason(spec.tab, err)
		default:
			counter.Count = total
		}

		result = append(result, counter)
	}

	return inboxmanager.SortCounters(result), nil
}

// unavailableReason menyusun keterangan yang dibaca penyelia saat sebuah sumber tidak dapat
// dibaca.
//
// Ia menyebut tabel atau view-nya, karena yang dapat memperbaikinya adalah DBA — dan laporan
// yang tidak menyebut objeknya akan menempuh dua putaran tanya-jawab sebelum sampai ke orang
// yang tepat.
func unavailableReason(tabCode string, err error) string {
	if tabCode == inboxmanager.TabMasterSparepart && isInvalidObject(err) {
		return "Sumbernya, view POOLDATA.SPAREPART_HE, sedang tidak dapat dibaca basis data. " +
			"View itu membaca kolom JSONDATA dari POOLDATA.M_SPAREPART_HE, dan kolom itu " +
			"sudah tidak ada lagi di sana. Perbaikannya ada di sisi DBA; tidak ada yang " +
			"perlu diubah di aplikasi."
	}
	return "Sumber antrean ini sedang tidak dapat dibaca basis data."
}

// Kode galat Oracle yang dikenali modul ini.
const (
	// oracleInvalidObject — objek yang dirujuk ada tetapi tidak sah, misalnya view yang
	// definisinya menyebut kolom yang sudah dihapus.
	oracleInvalidObject = "ORA-04063"

	// oracleMissingColumn — pengenal yang tidak sah, yang pada kueri modul ini selalu
	// berarti kolomnya belum ada.
	oracleMissingColumn = "ORA-00904"

	// oracleUnboundVariable — argumen bind kurang dari jumlah penanda yang muncul.
	//
	// Ia TIDAK pernah disebabkan pengguna: ia selalu cacat pemrograman. Ia dikenali supaya
	// pesannya di log menyebut sebabnya alih-alih hanya kode Oracle-nya — kelas cacat ini
	// sudah pernah menggigit modul Inbox RCL dan tidak tertangkap satu pun uji memori.
	oracleUnboundVariable = "ORA-01008"
)

func isInvalidObject(err error) bool {
	return err != nil && strings.Contains(err.Error(), oracleInvalidObject)
}

func isMissingColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), oracleMissingColumn)
}

func isUnboundVariable(err error) bool {
	return err != nil && strings.Contains(err.Error(), oracleUnboundVariable)
}

// wrapQueryError menerjemahkan kegagalan kueri menjadi galat yang dapat ditindaklanjuti.
func wrapQueryError(what string, err error) error {
	switch {
	case isInvalidObject(err), isMissingColumn(err):
		return fmt.Errorf("%s (%w): %v", what, inboxmanager.ErrSourceUnavailable, err)
	case isUnboundVariable(err):
		return fmt.Errorf(
			"%s: jumlah argumen bind tidak sama dengan jumlah penanda yang muncul di teks "+
				"kueri — driver mengikat menurut urutan kemunculan, bukan menurut nomor "+
				"penanda: %w", what, err)
	default:
		return fmt.Errorf("%s: %w", what, err)
	}
}

// Dashboard mengambil kedua panel sebuah tab dashboard.
func (r *Repo) Dashboard(
	ctx context.Context,
	q inboxmanager.Query,
) (inboxmanager.DashboardView, error) {
	switch q.Tab.Code {
	case inboxmanager.TabOutstanding:
		return r.dashboardOutstanding(ctx, q)
	case inboxmanager.TabProduktivitas:
		return r.dashboardProduktivitas(ctx, q)
	case inboxmanager.TabKlaim:
		return r.dashboardKlaim(ctx, q)
	default:
		return inboxmanager.DashboardView{}, fmt.Errorf(
			"tab %s bukan dashboard", q.Tab.Code)
	}
}

// dashboardOutstanding mengisi kedua grid tab Outstanding.
//
// Ia TIDAK melaporkan waktu penyegaran: sumbernya `T_CLAIMLIST_ADMIN` dibaca langsung, bukan
// cuplikan berkala, sehingga tidak ada penyegaran untuk dilaporkan.
func (r *Repo) dashboardOutstanding(
	ctx context.Context,
	q inboxmanager.Query,
) (inboxmanager.DashboardView, error) {
	flags := lineFlags(q.LineBusiness)

	// Kueri "per PIC" memuat penyaring lini bisnis DUA KALI — bagian ALL dan bagian
	// per-petugas — sehingga benderanya dikirim dua kali pula.
	picArgs := append(append([]any{}, flags...), flags...)

	pic, err := r.countByDimension(ctx, "dashboard_os_pic", picArgs)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	group, err := r.countByDimension(ctx, "dashboard_os_business_group", flags)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	panels := clonePanels(q.Tab.Panels)
	panels[0].Rows = pic
	panels[1].Rows = group

	return inboxmanager.DashboardView{Panels: panels}, nil
}

// countByDimension membaca grid berbentuk (dimensi, jumlah).
func (r *Repo) countByDimension(
	ctx context.Context,
	name string,
	args []any,
) ([]inboxmanager.DashboardRow, error) {
	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, wrapQueryError("menjalankan kueri "+name, err)
	}
	defer rows.Close()

	result := []inboxmanager.DashboardRow{}
	for rows.Next() {
		var dimension sql.NullString
		var total int
		if err := rows.Scan(&dimension, &total); err != nil {
			return nil, fmt.Errorf("membaca baris %s: %w", name, err)
		}

		key := inboxmanager.FieldPIC
		if name == "dashboard_os_business_group" {
			key = inboxmanager.FieldGrupBisnis
		}

		result = append(result, inboxmanager.DashboardRow{
			Cells: map[string]inboxmanager.DashboardCell{
				key:                           {Text: dimension.String},
				inboxmanager.FieldJumlahKlaim: {Count: total},
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil %s: %w", name, err)
	}

	return result, nil
}

// produktivitasPeriodArgs menyusun keenam belas batas periode kueri Produktivitas.
//
// Urutannya mengikuti urutan penanda di berkas .sql: pasangan (dari, sampai) delapan kali,
// bergantian antara periode berjalan dan periode tahun lalu, seurutan dengan kedelapan
// pencacahnya.
func produktivitasPeriodArgs(p inboxmanager.Period) []any {
	now := []any{p.From, p.Until}
	prior := []any{p.PriorFrom, p.PriorUntil}

	args := []any{}
	for i := 0; i < 4; i++ {
		args = append(args, now...)
		args = append(args, prior...)
	}
	return args
}

// dashboardProduktivitas mengisi kedua grid tab Produktivitas Klaim.
func (r *Repo) dashboardProduktivitas(
	ctx context.Context,
	q inboxmanager.Query,
) (inboxmanager.DashboardView, error) {
	flags := lineFlags(q.LineBusiness)
	periods := produktivitasPeriodArgs(q.Period)

	businessArgs := append(append([]any{}, periods...), flags...)

	// Kueri "per PIC" dua bagian, sehingga periode dan bendera dikirim dua kali.
	picArgs := append(append([]any{}, businessArgs...), businessArgs...)

	business, err := r.comparisonRows(ctx, "dashboard_produktivitas_business", businessArgs)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	pic, err := r.comparisonRows(ctx, "dashboard_produktivitas_pic", picArgs)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	panels := clonePanels(q.Tab.Panels)
	panels[0].Rows = business
	panels[1].Rows = pic

	refreshed, err := r.refreshedAt(ctx)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	return inboxmanager.DashboardView{Panels: panels, RefreshedAt: refreshed}, nil
}

// comparisonRows membaca grid Produktivitas — satu dimensi dan delapan pencacah.
func (r *Repo) comparisonRows(
	ctx context.Context,
	name string,
	args []any,
) ([]inboxmanager.DashboardRow, error) {
	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, wrapQueryError("menjalankan kueri "+name, err)
	}
	defer rows.Close()

	order := []string{
		inboxmanager.FieldTotalPeriodeIni, inboxmanager.FieldTotalPeriodeLTY,
		inboxmanager.FieldAksepPeriodeIni, inboxmanager.FieldAksepPeriodeLTY,
		inboxmanager.FieldTolakPeriodeIni, inboxmanager.FieldTolakPeriodeLTY,
		inboxmanager.FieldOSPeriodeIni, inboxmanager.FieldOSPeriodeLTY,
	}

	result := []inboxmanager.DashboardRow{}
	for rows.Next() {
		var dimension sql.NullString
		counts := make([]int, len(order))

		targets := make([]any, 0, len(order)+1)
		targets = append(targets, &dimension)
		for i := range counts {
			targets = append(targets, &counts[i])
		}

		if err := rows.Scan(targets...); err != nil {
			return nil, fmt.Errorf("membaca baris %s: %w", name, err)
		}

		cells := map[string]inboxmanager.DashboardCell{
			inboxmanager.FieldDimensi: {Text: dimension.String},
		}
		for i, key := range order {
			cells[key] = inboxmanager.DashboardCell{Count: counts[i]}
		}

		result = append(result, inboxmanager.DashboardRow{Cells: cells})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil %s: %w", name, err)
	}

	return result, nil
}

// dashboardKlaim mengisi kedua grid tab Klaim.
func (r *Repo) dashboardKlaim(
	ctx context.Context,
	q inboxmanager.Query,
) (inboxmanager.DashboardView, error) {
	flags := lineFlags(q.LineBusiness)

	// Periode boleh kosong di tab ini — lihat catatan pada kuerinya. Bendera pertama yang
	// menyatakannya, dan kedua batas tetap dikirim supaya jumlah argumen tidak berubah-ubah.
	allPeriods := 1
	from, until := time.Time{}, time.Time{}
	if !q.Period.Empty() {
		allPeriods = 0
		from, until = q.Period.From, q.Period.Until
	}

	args := append([]any{allPeriods, from, until}, flags...)

	business, err := r.claimRows(ctx, "dashboard_klaim_business", args, false)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	cause, err := r.claimRows(ctx, "dashboard_klaim_cause", args, true)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	panels := clonePanels(q.Tab.Panels)
	panels[0].Rows = business
	panels[1].Rows = cause

	refreshed, err := r.refreshedAt(ctx)
	if err != nil {
		return inboxmanager.DashboardView{}, err
	}

	return inboxmanager.DashboardView{Panels: panels, RefreshedAt: refreshed}, nil
}

// claimRows membaca grid Dashboard Klaim — empat pencacah dan tiga nilai uang.
//
// Nilai uang dipindai sebagai TEKS, bukan sebagai float64. Membacanya sebagai float berarti
// jumlah rupiah melewati bilangan pecahan biner, yang `09-DATABASE-STRATEGY.md` §5 dan
// invarian `I-12` larang — dan yang akibatnya baru terlihat pada digit terakhir sebuah
// laporan.
func (r *Repo) claimRows(
	ctx context.Context,
	name string,
	args []any,
	withCause bool,
) ([]inboxmanager.DashboardRow, error) {
	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, wrapQueryError("menjalankan kueri "+name, err)
	}
	defer rows.Close()

	result := []inboxmanager.DashboardRow{}
	for rows.Next() {
		var dimension, cause sql.NullString
		var total, accepted, rejected, outstanding int
		var acceptedAmount, rejectedAmount, outstandingAmount sql.NullString

		targets := []any{&dimension}
		if withCause {
			targets = append(targets, &cause)
		}
		targets = append(targets,
			&total, &accepted, &rejected, &outstanding,
			&acceptedAmount, &rejectedAmount, &outstandingAmount)

		if err := rows.Scan(targets...); err != nil {
			return nil, fmt.Errorf("membaca baris %s: %w", name, err)
		}

		cells := map[string]inboxmanager.DashboardCell{
			inboxmanager.FieldNamaBisnisDK: {Text: dimension.String},
			inboxmanager.FieldTotalKlaim:   {Count: total},
			inboxmanager.FieldJumlahAksep:  {Count: accepted},
			inboxmanager.FieldJumlahTolak:  {Count: rejected},
			inboxmanager.FieldJumlahOS:     {Count: outstanding},
			inboxmanager.FieldNilaiAksep:   {Amount: acceptedAmount.String},
			inboxmanager.FieldNilaiTolak:   {Amount: rejectedAmount.String},
			inboxmanager.FieldNilaiOS:      {Amount: outstandingAmount.String},
		}
		if withCause {
			cells[inboxmanager.FieldPenyebab] = inboxmanager.DashboardCell{Text: cause.String}
		}

		result = append(result, inboxmanager.DashboardRow{Cells: cells})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil %s: %w", name, err)
	}

	return result, nil
}

// refreshedAt membaca waktu cuplikan dashboard terakhir disegarkan.
//
// Tabel tanpa baris BUKAN galat: ia berarti penyegaran belum pernah tercatat, dan layar
// menyatakannya sebagai "belum diketahui" alih-alih gagal memuat seluruh dashboard karena
// satu keterangan tambahan.
func (r *Repo) refreshedAt(ctx context.Context) (*time.Time, error) {
	var at sql.NullTime
	err := r.db.QueryRowContext(ctx, query("dashboard_refreshed_at")).Scan(&at)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, wrapQueryError("membaca waktu penyegaran dashboard", err)
	}
	if !at.Valid {
		return nil, nil
	}
	value := at.Time
	return &value, nil
}

// clonePanels menyalin bentuk panel dari definisi tab supaya barisnya dapat diisi tanpa
// menulisi senarai global.
func clonePanels(panels []inboxmanager.Panel) []inboxmanager.Panel {
	result := make([]inboxmanager.Panel, len(panels))
	copy(result, panels)
	return result
}

// queueSpec memetakan sebuah tab antrean ke kueri daftarnya.
var queueQueries = map[string]string{
	inboxmanager.TabMasterBengkel:     "queue_bengkel",
	inboxmanager.TabMasterPanel:       "queue_panel",
	inboxmanager.TabNomorRangka:       "queue_nomor_rangka",
	inboxmanager.TabMasterSparepart:   "queue_sparepart",
	inboxmanager.TabKategoriSparepart: "queue_kategori_sparepart",
	inboxmanager.TabTipeSparepart:     "queue_tipe_sparepart",
	inboxmanager.TabGroupingSparepart: "queue_grouping_sparepart",
	inboxmanager.TabPaymentAkseptasi:  "queue_payment_akseptasi",
	inboxmanager.TabPenolakanKlaim:    "queue_penolakan_klaim",
}

// Queue mengambil SELURUH baris sebuah antrean, belum dipaginasi.
//
// Setiap kueri antrean mengembalikan `ROW_KEY` lebih dulu, lalu satu kolom per kolom yang
// diumumkan Tab.Columns, DALAM URUTAN YANG SAMA. Kesesuaian itu dijaga
// `TestJumlahKolomKueriSamaDenganJumlahKolomTab`.
func (r *Repo) Queue(
	ctx context.Context,
	q inboxmanager.Query,
) ([]inboxmanager.QueueRow, error) {
	name, known := queueQueries[q.Tab.Code]
	if !known {
		return nil, fmt.Errorf("tab %s bukan antrean", q.Tab.Code)
	}

	rows, err := r.db.QueryContext(ctx, query(name))
	if err != nil {
		return nil, wrapQueryError("menjalankan kueri "+name, err)
	}
	defer rows.Close()

	result := []inboxmanager.QueueRow{}
	for rows.Next() {
		row, err := scanQueueRow(rows, q.Tab.Columns)
		if err != nil {
			return nil, fmt.Errorf("membaca baris %s: %w", name, err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil %s: %w", name, err)
	}

	return result, nil
}

// scanner adalah bentuk minimal yang dibutuhkan scanQueueRow, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanQueueRow memindai satu baris antrean menjadi QueueRow.
//
// Seluruh sel dipindai lewat `any` lalu diubah menjadi teks. Itu bukan kemalasan: kesembilan
// antrean berada di atas tabel yang tidak berhubungan, dan kolomnya bercampur teks, angka,
// dan tanggal. Pemindaian bertipe tetap akan memaksa sembilan bentuk pemindai yang hampir
// sama.
func scanQueueRow(row scanner, columns []inboxmanager.Column) (inboxmanager.QueueRow, error) {
	var key sql.NullString
	values := make([]any, len(columns))

	targets := make([]any, 0, len(columns)+1)
	targets = append(targets, &key)
	for i := range values {
		targets = append(targets, &values[i])
	}

	if err := row.Scan(targets...); err != nil {
		return inboxmanager.QueueRow{}, err
	}

	cells := map[string]string{}
	for i, column := range columns {
		cells[column.Key] = asText(values[i])
	}

	return inboxmanager.QueueRow{Key: strings.TrimSpace(key.String), Cells: cells}, nil
}

// asText mengubah satu sel apa pun menjadi teks siap tampil.
//
// Tanggal diformat `dd/mm/yyyy`, mengikuti `TO_CHAR(tglinput,'dd/mm/yyyy')` yang dipakai
// kueri lama. Pemformatannya dilakukan DI SINI, bukan di SQL: `TO_CHAR` dilarang demi
// portabilitas, dan mengembalikan tanggal sebagai teks dari basis data membuat
// pengurutannya menjadi pengurutan teks (`09-DATABASE-STRATEGY.md` §3.2).
func asText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	case time.Time:
		return typed.Format("02/01/2006")
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

// decideArgs menyusun argumen pernyataan keputusan sebuah antrean.
//
// Ia mengembalikan nama kueri dan argumennya, atau galat bila kuncinya tidak berbentuk
// sebagaimana mestinya. Setiap antrean punya bentuk kuncinya sendiri, dan satu di antaranya
// gabungan empat kolom.
func decideArgs(d inboxmanager.Decision, key string) (string, []any, error) {
	status := d.Verdict.Value()

	switch d.Tab.Code {
	case inboxmanager.TabMasterBengkel:
		return "decide_bengkel", []any{status, nullIfEmpty(d.Reason), key}, nil

	case inboxmanager.TabMasterPanel:
		return "decide_panel", []any{status, nullIfEmpty(d.Reason), key}, nil

	case inboxmanager.TabNomorRangka:
		parts := strings.Split(key, "|")
		if len(parts) != 4 {
			return "", nil, fmt.Errorf(
				"kunci antrean nomor rangka harus gabungan empat kolom, bukan %q", key)
		}
		return "decide_nomor_rangka", []any{
			status, parts[0], parts[1], parts[2], parts[3],
		}, nil

	case inboxmanager.TabMasterSparepart:
		return "decide_sparepart", []any{status, d.Caller.Login, key}, nil

	case inboxmanager.TabKategoriSparepart:
		return "decide_kategori_sparepart", []any{status, key}, nil

	case inboxmanager.TabTipeSparepart:
		return "decide_tipe_sparepart", []any{status, key}, nil

	case inboxmanager.TabGroupingSparepart:
		return "decide_grouping_sparepart", []any{status, key}, nil

	case inboxmanager.TabPaymentAkseptasi:
		return "decide_payment_akseptasi", []any{status, nullIfEmpty(d.Reason), key}, nil

	case inboxmanager.TabPenolakanKlaim:
		return "decide_penolakan_klaim", []any{
			status, d.Caller.Login, nullIfEmpty(d.Reason), key,
		}, nil

	default:
		return "", nil, fmt.Errorf("tab %s tidak dapat diputuskan", d.Tab.Code)
	}
}

// nullIfEmpty mengirim NULL alih-alih teks kosong.
//
// Alasannya bukan kerapian: kolom alasan yang berisi teks kosong tidak dapat dibedakan dari
// alasan yang memang belum pernah diisi, sedangkan NULL dapat. Pada persetujuan — yang memang
// tidak menuntut alasan — itulah keadaan yang benar.
func nullIfEmpty(value string) any {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	return clean
}

// Decide menuliskan keputusan atas sejumlah baris, seluruhnya di dalam SATU transaksi.
//
// # Kenapa satu baris per pernyataan
//
// Karena daftar penanda bind yang panjangnya berubah-ubah membuat teks SQL tidak lagi tetap,
// dan bentuk itulah yang `08-TECHNICAL-STRATEGY.md` §4.3 larang. Biayanya satu perjalanan per
// baris pada operasi yang jarang dan berbaris sedikit; yang diperoleh adalah teks kueri yang
// dapat dibaca utuh di berkas .sql dan diuji tanpa basis data.
//
// # Kenapa SATU transaksi
//
// Supaya sekumpulan keputusan tidak dapat setengah jadi. Penyelia yang menyetujui sepuluh
// baris lalu gagal pada baris keenam tidak akan meninggalkan lima baris yang sudah berubah
// tanpa ia sadari.
//
// # Jumlah yang dikembalikan boleh lebih kecil daripada jumlah kunci
//
// Setiap pernyataan ikut menyaring status menunggu, sehingga baris yang sudah diputuskan
// orang lain TIDAK berubah. Selisihnya bukan kegagalan melainkan keadaan yang wajib
// disampaikan — lihat inboxmanager.DecisionResult.Stale.
func (r *Repo) Decide(ctx context.Context, d inboxmanager.Decision) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("memulai transaksi keputusan %s: %w", d.Tab.Code, err)
	}
	defer func() { _ = tx.Rollback() }()

	changed := 0
	for _, key := range d.Keys {
		name, args, err := decideArgs(d, key)
		if err != nil {
			return 0, err
		}

		result, err := tx.ExecContext(ctx, query(name), args...)
		if err != nil {
			return 0, wrapQueryError("menjalankan "+name, err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("membaca jumlah baris terubah %s: %w", name, err)
		}
		changed += int(affected)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("menyimpan keputusan %s: %w", d.Tab.Code, err)
	}

	return changed, nil
}

// LineBusinessFor membaca lini bisnis seorang petugas dari M_LOGIN_PNC.
//
// Petugas tanpa baris, atau yang kolomnya kosong, mengembalikan teks kosong TANPA galat. Di
// Pega `pyPosition` yang tidak cocok satu pun sekadar tidak memicu penyaring mana pun — ia
// tidak menggagalkan layarnya.
func (r *Repo) LineBusinessFor(ctx context.Context, loginID string) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(loginID))
	if id == "" {
		return "", nil
	}

	var line sql.NullString
	err := r.db.QueryRowContext(ctx, query("line_business_for"), id).Scan(&line)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("membaca lini bisnis petugas %s: %w", id, err)
	}

	return strings.TrimSpace(line.String), nil
}

// CheckTable memastikan ketiga tabel inti modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun.
//
// # Kenapa TIGA pemeriksaan, bukan satu
//
// Karena ketiganya menyentuh tabel yang berbeda dan gagal dengan sebab yang berbeda.
// Memisahkannya membuat pesan gagalnya menyebut satu hal saja.
//
// # Yang TIDAK diperiksa di sini, dan kenapa
//
// Kesembilan tabel antrean persetujuan. Kegagalan membacanya sudah dilaporkan per antrean
// lewat Counter.Unavailable, yang menyebut objeknya dan sampai ke layar — bentuk laporan yang
// lebih berguna daripada satu perintah pemeriksa yang berhenti pada objek pertama yang rusak.
//
// Dan hak TULIS atas kesembilan tabel itu tidak dapat diperiksa tanpa benar-benar menulis;
// perintah pemeriksa tidak boleh meninggalkan jejak di basis data mana pun.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"membaca POOLDATA.T_CLAIMLIST_ADMIN — sumber dashboard Outstanding sejak "+
				"sumbernya dipindahkan dari skema DATAPEGA: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, query("check_dashboard_snapshot")).Scan(&ignored); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"membaca POOLDATA.PEGA_DASHBOARDPNC — sumber dashboard Produktivitas Klaim "+
				"dan dashboard Klaim: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, query("check_line_business")).Scan(&ignored); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf(
			"membaca kolom LINE_BUSINESS pada POOLDATA.M_LOGIN_PNC — kolom ini yang "+
				"menentukan lini bisnis penyaring dashboard. Perhatikan GARIS BAWAH pada "+
				"namanya: migrasi yang menulisnya `LINEBUSINESS` sudah dicabut justru "+
				"karena salah nama: %w", err)
	}

	return nil
}
