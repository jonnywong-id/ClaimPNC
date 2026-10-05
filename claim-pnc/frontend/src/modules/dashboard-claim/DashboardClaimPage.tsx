import { useEffect, useMemo, useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import {
  PAGE_SIZE,
  unduhTile,
  useIzinPermintaanKlaim,
  usePenyaringDashboard,
  useRingkasanDashboard,
  useTelusurDashboard,
} from './api'
import { DialogPermintaanKlaim } from './DialogPermintaanKlaim'
import { DialogTransfer } from './DialogTransfer'
import { KartuPenghitung } from './KartuPenghitung'
import { RingkasanDonut } from './RingkasanDonut'
import { TampunganPIC } from './TampunganPIC'
import type {
  BarisKlaim,
  BarisSurvei,
  JenisPermintaanKlaim,
  Kartu,
  TabDashboard,
  Tile,
} from './types'

/**
 * Layar Dashboard Claim.
 *
 * Menggantikan `Harness/DashboardClaim_Harness-Harness.xml`, yang terdaftar di
 * `Navigation/pyCaseWorkerNavigation-Navigation.xml` sebagai butir menu ber-`showHarness`.
 *
 * | Hal | Sumbernya |
 * |---|---|
 * | Empat kartu penghitung | `Section/DashboardClaim_Section-Section.xml` |
 * | Kolom telusur bertipe klaim | `Section/DashboardClaim_Section2-Section.xml` |
 * | Kolom telusur bertipe survei | `Section/DashboardClaim_Section1-Section.xml` |
 * | Keempat tile dan kuerinya | `Activity/SetDashboardClaim-Act.xml` |
 * | Kelima pilihan lini bisnis | activity yang sama, cabang `TempView2.Remark` |
 *
 * # Ini PANDANGAN MANAJERIAL, bukan inbox
 *
 * Tidak satu pun angkanya disaring menurut siapa yang membukanya. Perbedaan itu yang
 * membuatnya tidak memakai ulang layar My Inbox, meski keduanya sama-sama menampilkan klaim
 * berjalan — kuerinya di sana menyaring `PXASSIGNEDOPERATORID`, di sini tidak.
 *
 * # Tanpa grafik, dan itu keputusan
 *
 * `D-13` menetapkan tata letak mengikuti Pega supaya pengguna tidak perlu belajar ulang.
 * Layar lamanya memang empat angka dan satu tabel; menambahkan grafik akan menambah
 * permukaan yang harus diuji tanpa diminta siapa pun. Work Owner memilihnya demikian pada
 * 2026-09-26.
 */
export function DashboardClaimPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabDashboard>('dashboard')

  const [liniBisnis, setLiniBisnis] = useState('')
  const [cari, setCari] = useState('')
  const [tile, setTile] = useState<Tile | null>(null)
  const [halaman, setHalaman] = useState(1)

  // Keadaan tab Tampungan disimpan TERPISAH dari tab Dashboard.
  //
  // Keduanya mencari hal yang berbeda — yang satu No Klaim/No Polis pada telusur tile, yang
  // lain No Klaim di penampungan. Membagi satu keadaan membuat pencarian terbawa saat
  // berpindah tab, dan pengguna melihat daftar kosong tanpa sebab yang terlihat.
  // Baris yang sedang diajukan transfernya. Hanya tile Outstanding yang memilikinya —
  // layar lama hanya menggambar tombol Transfer di sana dan di tab Tampungan.
  const [transferKlaim, setTransferKlaim] = useState<BarisKlaim | null>(null)

  // "Transfer All Case By UserID" — tombol tingkat layar pada grid Outstanding, bukan per
  // baris. Ia memindahkan SELURUH pekerjaan satu operator sekaligus.
  const [transferMassal, setTransferMassal] = useState(false)

  const [cariTampungan, setCariTampungan] = useState('')
  const [halamanTampungan, setHalamanTampungan] = useState(1)

  function gantiTab(berikutnya: TabDashboard) {
    setTab(berikutnya)
  }

  function gantiCariTampungan(nilai: string) {
    setCariTampungan(nilai)
    setHalamanTampungan(1)
  }

  const penyaring = usePenyaringDashboard()
  const ringkasan = useRingkasanDashboard({ lini_bisnis: liniBisnis, cari })
  const telusur = useTelusurDashboard(tile, { lini_bisnis: liniBisnis, cari, halaman })

  /**
   * Mengubah penyaring mengembalikan telusur ke halaman pertama.
   *
   * Tanpa ini, pengguna yang sedang di halaman 7 lalu mempersempit penyaring akan melihat
   * tabel KOSONG — bukan karena tidak ada hasilnya, melainkan karena halaman 7 tidak ada
   * lagi. Kegagalan itu tampak persis seperti "tidak ada data".
   */
  function gantiLiniBisnis(nilai: string) {
    setLiniBisnis(nilai)
    setHalaman(1)
  }

  function gantiCari(nilai: string) {
    setCari(nilai)
    setHalaman(1)
  }

  function pilihTile(dipilih: Tile) {
    // Menekan kartu yang sama menutup telusurnya. Ia satu-satunya cara kembali ke tampilan
    // ringkasan saja tanpa memuat ulang halaman.
    setTile((sekarang) => (sekarang === dipilih ? null : dipilih))
    setHalaman(1)
  }

  const kartu: Kartu[] = ringkasan.data?.kartu ?? kartuKosong(penyaring.data?.tile)

  const pilihanLini = useMemo(
    () => (penyaring.data?.lini_bisnis ?? []).map((p) => ({ value: p.nilai, label: p.label })),
    [penyaring.data],
  )

  if (portal === null) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
        <ErrorMessage
          tone="penolakan"
          title="Portal entitas belum dipilih"
          description="Pilih entitas lebih dulu. Angka pada layar ini dibaca dari basis data entitas yang sedang dibuka, dan permintaan tanpa entitas ditolak."
        />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6">
        <nav aria-label="Jejak lokasi" className="mb-2 text-xs font-medium text-slate-500">
          <ol className="flex items-center gap-1.5">
            <li>Proses Produksi</li>
            <li aria-hidden="true" className="text-slate-300">
              /
            </li>
            <li className="text-slate-700">Dashboard Claim</li>
          </ol>
        </nav>

        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Dashboard Claim</h1>

        {/*
          Kalimat ini menyatakan hal yang tidak terlihat dari angkanya sendiri: layar ini
          mudah tertukar dengan My Inbox, dan tertukarnya tidak menghasilkan galat apa pun.

          Ajakan menelusurnya sengaja TIDAK diulang di sini — ia hidup di satu tempat, pada
          panel kosong di bawah, supaya tidak ada dua kalimat yang harus dijaga tetap sama.
        */}
        <p className="mt-2 max-w-3xl text-sm leading-relaxed text-slate-600">
          Ringkasan pekerjaan klaim pada entitas yang sedang dibuka. Angka di bawah menghitung
          pekerjaan <strong>seluruh organisasi</strong>, bukan pekerjaan Anda sendiri.
        </p>

        <p className="mt-3 text-xs text-slate-500">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">
            {ringkasan.data?.portal ?? portal ?? '—'}
          </span>
        </p>
      </header>

      {/*
        Kedua tab layar lama. Ia `tablist` sungguhan, bukan sepasang tombol yang diberi
        gaya: dengan peran yang benar, pembaca layar menyebutkan "tab 1 dari 2" dan panah
        kiri-kanan berpindah tab tanpa satu baris kode tambahan.
      */}
      <div role="tablist" aria-label="Tab Dashboard Claim" className="mb-6 flex gap-1 border-b border-slate-200">
        {([
          ['dashboard', 'Dashboard Claim'],
          ['tampungan', 'Inbox Tampungan PIC'],
        ] as const).map(([nilai, judul]) => (
          <button
            key={nilai}
            role="tab"
            type="button"
            aria-selected={tab === nilai}
            onClick={() => gantiTab(nilai)}
            className={[
              '-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition ease-halus',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
              'focus-visible:outline-blue-600',
              tab === nilai
                ? 'border-blue-600 text-blue-700'
                : 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700',
            ].join(' ')}
          >
            {judul}
          </button>
        ))}
      </div>

      {tab === 'tampungan' ? (
        <TampunganPIC
          cari={cariTampungan}
          onCari={gantiCariTampungan}
          halaman={halamanTampungan}
          onHalaman={setHalamanTampungan}
        />
      ) : (
        <>
      {/* Penyaring. Keduanya berlaku untuk keempat kartu SEKALIGUS, supaya angka-angkanya
          tidak pernah berasal dari populasi yang berbeda. */}
      <section
        aria-label="Penyaring"
        className="mb-6 rounded-kartu border border-slate-200 bg-white p-4 shadow-lembut"
      >
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <SelectField
            id="dashboard-lini-bisnis"
            label="Lini Bisnis"
            value={liniBisnis}
            options={pilihanLini}
            emptyText="Semua Lini Bisnis"
            onChange={(event) => gantiLiniBisnis(event.target.value)}
          />

          <div className="flex items-end">
            <Button
              tone="kedua"
              onClick={() => {
                ringkasan.refetch()
                if (tile !== null) telusur.refetch()
              }}
              disabled={ringkasan.isFetching}
            >
              <ReloadIcon className="h-4 w-4" />
              {ringkasan.isFetching ? 'Memuat…' : 'Muat ulang'}
            </Button>
          </div>

          {/*
            "Transfer All Case By UserID" — tombol tingkat layar pada layar lama, bukan per
            baris. Ia memindahkan SELURUH pekerjaan satu operator sekaligus, sehingga ia
            tidak bergantung pada kartu mana yang sedang dipilih dan tetap digambar di sini.
          */}
          <div className="flex items-end">
            <Button tone="kedua" onClick={() => setTransferMassal(true)}>
              Transfer All Case By UserID
            </Button>
          </div>
        </div>
      </section>

      {ringkasan.isError ? (
        <div className="mb-6">
          <ErrorMessage
            tone="gangguan"
            title="Ringkasan tidak dapat dibaca"
            description={pesanGalat(ringkasan.error)}
          />
        </div>
      ) : null}

      <section aria-label="Ringkasan" className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {kartu.map((item) => (
          <KartuPenghitung
            key={item.tile}
            kartu={item}
            terpilih={item.tile === tile}
            memuat={ringkasan.isPending}
            onPilih={pilihTile}
          />
        ))}
      </section>

      {/*
        Donut komposisi — padanan grafik pada layar lama, yang menggambarnya berdampingan
        dengan daftar "Status Register / Jumlah". Keduanya menampilkan angka yang sama.

        Digambar SETELAH kartunya, bukan sebelum: kartu membawa angka yang terbaca pembaca
        layar, dan donut menyembunyikan dirinya dari pembaca layar supaya angka yang sama
        tidak dibacakan dua kali.
      */}
      <RingkasanDonut kartu={kartu} terpilih={tile} onPilih={pilihTile} />

      {/*
        Kedua panel keterangan — `catatan_warisan` dan `selisih_terencana` — DIHAPUS dari
        layar atas permintaan Work Owner 2026-09-26.

        Kedua field-nya TIDAK dihapus dari respons `/ringkasan`: ia tetap dibaca penguji
        gerbang 1 saat membandingkan angka (`D-54`), dan membuangnya dari kontrak akan
        menghapus jejak keputusan yang mendasarinya. Yang berubah hanyalah ia tidak lagi
        digambar di layar.
      */}

      {tile === null ? (
        <p className="rounded-kartu border border-dashed border-slate-300 bg-slate-50 p-8 text-center text-sm text-slate-600">
          Pilih salah satu kartu di atas untuk menelusuri barisnya.
        </p>
      ) : (
        <Telusur
          tile={tile}
          judul={telusur.data?.judul ?? ''}
          cari={cari}
          onCari={gantiCari}
          halaman={halaman}
          onHalaman={setHalaman}
          state={telusur}
          liniBisnis={liniBisnis}
          onTransfer={setTransferKlaim}
        />
      )}
        </>
      )}

      {transferMassal ? (
        <DialogTransfer lingkup="massal" onTutup={() => setTransferMassal(false)} />
      ) : null}

      {transferKlaim !== null ? (
        <DialogTransfer
          lingkup="baris"
          nomorKlaim={transferKlaim.nomor_klaim}
          klaimID={transferKlaim.klaim_id}
          onTutup={() => setTransferKlaim(null)}
        />
      ) : null}
    </div>
  )
}

/**
 * Tabel telusur satu tile.
 *
 * Kolomnya berbeda menurut bentuk baris, dan pemilihannya dibaca dari respons — bukan
 * disimpulkan dari nama tile.
 */
function Telusur({
  tile,
  judul,
  cari,
  onCari,
  halaman,
  onHalaman,
  state,
  liniBisnis,
  onTransfer,
}: {
  tile: Tile
  judul: string
  cari: string
  onCari: (nilai: string) => void
  halaman: number
  onHalaman: (halaman: number) => void
  state: ReturnType<typeof useTelusurDashboard>
  liniBisnis: string
  onTransfer: (row: BarisKlaim) => void
}) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const [galatUnduh, setGalatUnduh] = useState<string | null>(null)

  // Baris yang dicentang pada kolom "Pilih" — hanya tile Close Claim yang memilikinya.
  //
  // Disimpan sebagai himpunan `klaim_id`, bukan daftar baris: barisnya diganti setiap kali
  // data dimuat ulang, dan menyimpan objeknya akan membuat centang menunjuk salinan lama.
  const [dipilih, setDipilih] = useState<Set<string>>(new Set())
  const [permintaan, setPermintaan] = useState<JenisPermintaanKlaim | null>(null)

  const izin = useIzinPermintaanKlaim(tile === 'close-claim')

  // Centang dibuang saat halaman, pencarian, atau penyaring berubah.
  //
  // Tanpa ini, pengguna mencentang tiga baris di halaman 1, berpindah ke halaman 2, lalu
  // menekan ReOpen — dan tiga klaim yang TIDAK terlihat di layarnya ikut terkirim.
  useEffect(() => {
    setDipilih(new Set())
  }, [tile, halaman, cari, liniBisnis])

  function gantiPilih(klaimID: string) {
    setDipilih((sebelumnya) => {
      const berikutnya = new Set(sebelumnya)
      if (berikutnya.has(klaimID)) berikutnya.delete(klaimID)
      else berikutnya.add(klaimID)
      return berikutnya
    })
  }

  async function unduh() {
    setGalatUnduh(null)
    try {
      await unduhTile(tile, { lini_bisnis: liniBisnis, cari }, token, portal)
    } catch (failure) {
      // Unduhan gagal TIDAK boleh diam: pengguna menekan tombol, tidak ada berkas yang
      // muncul, dan tanpa pesan ia tidak tahu apakah hasilnya kosong atau permintaannya
      // yang gagal.
      setGalatUnduh(pesanGalat(failure))
    }
  }

  const tombolUnduh = (
    <Button tone="kedua" onClick={() => void unduh()}>
      Export to Excel
    </Button>
  )

  const data = state.data
  const bentuk = data?.bentuk ?? (tile === 'loss-adjuster' || tile === 'internal-surveyor' ? 'survei' : 'klaim')

  const keterangan = data?.halaman
  const pagination = {
    page: keterangan?.halaman ?? halaman,
    size: keterangan?.ukuran ?? PAGE_SIZE,
    total: keterangan?.total ?? 0,
    totalPage: keterangan?.total_halaman ?? 1,
    onPageChange: onHalaman,
    isLoading: state.isFetching,
  }

  const galat = state.isError ? (
    <ErrorMessage
      tone="gangguan"
      title="Daftar tidak dapat dibaca"
      description={pesanGalat(state.error)}
    />
  ) : undefined

  const peringatanUnduh =
    galatUnduh !== null ? (
      <div className="mb-4">
        <ErrorMessage tone="gangguan" title="Unduhan gagal" description={galatUnduh} />
      </div>
    ) : null

  if (bentuk === 'survei') {
    return (
      <>
      {peringatanUnduh}
      <DataTable<BarisSurvei>
        title={judul || 'Telusur'}
        label={`Daftar survei ${judul}`}
        // Kedua tile survei punya kolom yang BERBEDA di layar lama, bukan sekadar isi yang
        // berbeda. Memilihnya di sini, bukan menggabungkan keduanya menjadi satu set
        // terbesar, supaya tidak ada kolom yang digambar tanpa data yang mengisinya.
        columns={tile === 'loss-adjuster' ? kolomLossAdjuster : kolomInternalSurveyor}
        rows={data?.survei ?? []}
        rowKey={(row) => row.survei_id || row.nomor_survei}
        isLoading={state.isPending}
        error={galat}
        serverSearch={{ value: cari, onChange: onCari }}
        searchLabel="Cari No Survey atau No Polis"
        emptyMessage="Tidak ada survei yang cocok dengan penyaring."
        showHeaderWhenEmpty
        pagination={pagination}
        actions={tombolUnduh}
      />
      </>
    )
  }

  const barisKlaim = data?.klaim ?? []
  const barisDipilih = barisKlaim.filter((row) => dipilih.has(row.klaim_id || row.nomor_klaim))

  // Kedua tombol hanya ada pada tile Close Claim, karena di layar lama keduanya digambar
  // oleh `InboxManagerReopen1_Sec` — section yang hanya menampung grid Close Claim.
  const tombolCloseClaim =
    tile === 'close-claim' ? (
      <>
        <Button
          tone="kedua"
          disabled={barisDipilih.length === 0 || izin.data?.boleh_mengajukan === false}
          onClick={() => setPermintaan('salin')}
        >
          Copy Klaim
        </Button>
        <Button
          tone="kedua"
          disabled={barisDipilih.length === 0 || izin.data?.boleh_mengajukan === false}
          onClick={() => setPermintaan('reopen')}
        >
          ReOpen
        </Button>
      </>
    ) : null

  return (
    <>
    {peringatanUnduh}

    {/*
      Sebab tombolnya mati dinyatakan, bukan dibiarkan ditebak.

      Tombol yang mati tanpa keterangan terbaca sebagai layar rusak. Alasannya datang dari
      DOMAIN modul Inbox Close Claim — di sistem lama penjaganya When rule
      `IsManagerPNC_CLOSE` — bukan ditulis di layar ini.
    */}
    {tile === 'close-claim' && izin.data?.boleh_mengajukan === false ? (
      <p className="mb-4 rounded-kartu border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
        {izin.data.alasan_tidak_boleh ??
          'Peran Anda tidak dapat mengajukan ReOpen maupun Copy Klaim.'}
      </p>
    ) : null}

    <DataTable<BarisKlaim>
      title={judul || 'Telusur'}
      label={`Daftar klaim ${judul}`}
      // Outstanding menggambar Report Date dan Claim status; Close Claim tidak. Perbedaan
      // itu ada di layar lama dan ditiru apa adanya.
      columns={
        tile === 'close-claim'
          ? kolomCloseClaim(dipilih, gantiPilih)
          : kolomOutstanding(onTransfer)
      }
      rows={barisKlaim}
      rowKey={(row) => row.klaim_id || row.nomor_klaim}
      isLoading={state.isPending}
      error={galat}
      serverSearch={{ value: cari, onChange: onCari }}
      searchLabel="Cari No Klaim atau No Polis"
      emptyMessage="Tidak ada klaim yang cocok dengan penyaring."
      showHeaderWhenEmpty
      pagination={pagination}
      actions={
        <>
          {tombolCloseClaim}
          {tombolUnduh}
        </>
      }
    />

    {permintaan !== null ? (
      <DialogPermintaanKlaim
        jenis={permintaan}
        baris={barisDipilih}
        onTutup={() => {
          setPermintaan(null)
          // Centang dibuang setelah dialog ditutup: permintaannya sudah tercatat, dan
          // membiarkan barisnya tercentang mengundang pengajuan kedua atas klaim yang sama
          // — yang lalu ditolak 409 dan terbaca sebagai kegagalan baru.
          setDipilih(new Set())
        }}
      />
    ) : null}
    </>
  )
}

/**
 * Umur dalam hari, digambar seperti layar lama.
 *
 * Pega menampilkannya sebagai `6y ago` / `5y ago`, dan bentuk itu ditiru apa adanya
 * (`D-13`) — bukan diterjemahkan. Yang dikirim server adalah ANGKA hari; pemformatannya
 * ada di sini supaya pengurutan kolom tetap pengurutan angka, bukan pengurutan teks.
 */
function umur(hari: number): string {
  if (hari <= 0) return '—'
  if (hari >= 365) return `${Math.floor(hari / 365)}y ago`
  if (hari >= 30) return `${Math.floor(hari / 30)}mo ago`
  return `${hari}d ago`
}

/** Kolom bersama kedua tile bertipe klaim, dalam urutan yang sama dengan layar lama. */
const kolomKlaimDasar: Column<BarisKlaim>[] = [
  { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim, width: '9rem' },
  { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
  { key: 'nama_bisnis', title: 'Nama Bisnis', value: (row) => row.nama_bisnis },
  { key: 'sumber_bisnis', title: 'Sumber Bisnis', value: (row) => row.sumber_bisnis },
  { key: 'nama_cabang', title: 'Nama Cabang', value: (row) => row.nama_cabang },
  {
    key: 'tanggal_pendaftaran',
    title: 'Tanggal Pendaftaran',
    value: (row) => row.tanggal_pendaftaran,
    render: (row) => formatDate(row.tanggal_pendaftaran),
    width: '10rem',
  },
]

const kolomLamaWaktu: Column<BarisKlaim> = {
  key: 'lama_hari',
  title: 'Lama Waktu Klaim',
  value: (row) => String(row.lama_hari).padStart(8, '0'),
  render: (row) => umur(row.lama_hari),
  width: '9rem',
}

const kolomPICAdmin: Column<BarisKlaim>[] = [
  { key: 'pic_teknik', title: 'PIC Teknik', value: (row) => row.pic_teknik },
  { key: 'admin_pnc', title: 'Admin PNC', value: (row) => row.admin_pnc },
]

/**
 * Kolom tile **Outstanding**, mengikuti grid layar lama apa adanya:
 *
 *	No Klaim · No Polis · Nama Tertanggung · Nama Bisnis · Sumber Bisnis · Nama Cabang ·
 *	Tanggal Pendaftaran · Report Date · Lama Waktu Klaim · PIC Teknik · Admin PNC · Claim status
 *
 * Kolom "Pilih" dan tombol "Transfer" TIDAK dibawa: keduanya menulis, dan modul ini hanya
 * membaca selama masa paralel (`P-1`).
 *
 * `klaim_id` tidak dijadikan kolom — ia kunci teknis Pega yang memuat nama kelas internal,
 * dan `D-22` menetapkannya tidak ditampilkan kepada pengguna.
 */
function kolomOutstanding(
  onTransfer: (row: BarisKlaim) => void,
): Column<BarisKlaim>[] {
  return [
  ...kolomKlaimDasar,
  {
    key: 'tanggal_lapor',
    title: 'Report Date',
    value: (row) => row.tanggal_lapor,
    render: (row) => (row.tanggal_lapor ? formatDate(row.tanggal_lapor) : '—'),
    width: '10rem',
  },
  kolomLamaWaktu,
  ...kolomPICAdmin,
  { key: 'status_klaim_label', title: 'Claim status', value: (row) => row.status_klaim_label },
  {
    key: 'aksi',
    title: 'Aksi',
    // Kolom aksi tidak diurutkan dan tidak dicari: isinya tombol, bukan data.
    noSort: true,
    value: () => '',
    width: '7rem',
    render: (row) => (
      <Button tone="kedua" onClick={() => onTransfer(row)}>
        Transfer
      </Button>
    ),
  },
  ]
}

/**
 * Kolom tile **Close Claim**, mengikuti grid layar lama apa adanya:
 *
 *	No Klaim · No Polis · Nama Tertanggung · Nama Bisnis · Sumber Bisnis · Nama Cabang ·
 *	Tanggal Pendaftaran · Lama Waktu Klaim · PIC Teknik · Admin PNC
 *
 * Ia sengaja BERBEDA dari Outstanding: layar lama tidak menggambar Report Date maupun Claim
 * status di sini. Menyamakan keduanya akan menampilkan kolom yang datanya memang tidak
 * dibaca — dan kolom kosong tidak dapat dibedakan dari data yang hilang.
 *
 * # Kolom "Pilih"
 *
 * Kolom pertamanya adalah kotak centang, persis `Section/InboxManagerReopen1_Sec-Section.xml`
 * yang menggambar caption **"Pilih"** dengan kontrol `pxCheckbox`. Centangnya melayani kedua
 * tombol di atas tabel — ReOpen dan Copy Klaim.
 *
 * **Tanpa "Pilih Semua".** Section lamanya memuat tepat SATU kontrol checkbox, yaitu yang
 * per baris; tidak ada kontrol di kepala kolomnya. Menambahkannya akan memudahkan pekerjaan
 * yang akibatnya tidak dapat dibatalkan — mencentang seluruh halaman lalu menekan ReOpen
 * mencatat 25 permintaan sekaligus.
 */
function kolomCloseClaim(
  dipilih: Set<string>,
  onPilih: (klaimID: string) => void,
): Column<BarisKlaim>[] {
  return [
  {
    key: 'pilih',
    title: 'Pilih',
    // Kolom centang tidak diurutkan dan tidak dicari: isinya kendali, bukan data.
    noSort: true,
    value: () => '',
    width: '4rem',
    render: (row) => {
      const id = row.klaim_id || row.nomor_klaim
      return (
        <input
          type="checkbox"
          checked={dipilih.has(id)}
          onChange={() => onPilih(id)}
          // Nomor klaimnya disebut, bukan "Pilih" saja: pembaca layar membacakan 25 kotak
          // centang berturut-turut, dan label yang sama persis membuat keduanya tidak
          // dapat dibedakan.
          aria-label={`Pilih klaim ${row.nomor_klaim}`}
          className="h-4 w-4 rounded border-slate-300 text-blue-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
        />
      )
    },
  },
  ...kolomKlaimDasar,
  kolomLamaWaktu,
  ...kolomPICAdmin,
  ]
}

/**
 * Kolom tile **Loss Adjuster**, mengikuti grid layar lama apa adanya:
 *
 *	Appointment No · Reference No · Claim No · Policy No · Adjuster · PIC Adjuster ·
 *	Insured Name · PIC ASM · Status · Aging
 *
 * Judulnya berbahasa Inggris karena memang begitu di layar lama (`D-13`) — berbeda dari
 * tile Internal Surveyor di bawah yang judulnya berbahasa Indonesia. Keduanya ditiru apa
 * adanya, bukan diseragamkan.
 *
 * Lokasi Survei dan Tanggal Survey TIDAK ada di sini: kueri loss adjuster memang tidak
 * mengambil tanggalnya, dan grid lamanya tidak menggambar lokasinya.
 */
const kolomLossAdjuster: Column<BarisSurvei>[] = [
  { key: 'nomor_survei', title: 'Appointment No', value: (row) => row.nomor_survei, width: '9rem' },
  { key: 'nomor_referensi', title: 'Reference No', value: (row) => row.nomor_referensi, width: '9rem' },
  { key: 'nomor_klaim', title: 'Claim No', value: (row) => row.nomor_klaim, width: '9rem' },
  { key: 'nomor_polis', title: 'Policy No', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_surveyor', title: 'Adjuster', value: (row) => row.nama_surveyor },
  { key: 'pic_adjuster', title: 'PIC Adjuster', value: (row) => row.pic_adjuster },
  { key: 'nama_tertanggung', title: 'Insured Name', value: (row) => row.nama_tertanggung },
  { key: 'pic_teknik', title: 'PIC ASM', value: (row) => row.pic_teknik },
  { key: 'status_survei', title: 'Status', value: (row) => row.status_survei },
  {
    key: 'lama_hari',
    title: 'Aging',
    value: (row) => String(row.lama_hari).padStart(8, '0'),
    render: (row) => umur(row.lama_hari),
    width: '8rem',
  },
]

/**
 * Kolom tile **Internal Surveyor**, mengikuti grid layar lama apa adanya:
 *
 *	Nomor Case · No Klaim · No Polis · Nama Tertanggung · Aging · Tanggal Survey ·
 *	Lokasi Survey · Status Survey · PIC ASM · Surveyor
 *
 * Judulnya berbahasa Indonesia, berbeda dari tile Loss Adjuster di atas. Keduanya memang
 * begitu di layar lama.
 */
const kolomInternalSurveyor: Column<BarisSurvei>[] = [
  { key: 'nomor_survei', title: 'Nomor Case', value: (row) => row.nomor_survei, width: '9rem' },
  { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim, width: '9rem' },
  { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
  {
    key: 'lama_hari',
    title: 'Aging',
    value: (row) => String(row.lama_hari).padStart(8, '0'),
    render: (row) => umur(row.lama_hari),
    width: '8rem',
  },
  {
    key: 'tanggal_survei',
    title: 'Tanggal Survey',
    value: (row) => row.tanggal_survei,
    render: (row) => (row.tanggal_survei ? formatDate(row.tanggal_survei) : '—'),
    width: '10rem',
  },
  { key: 'lokasi_survei', title: 'Lokasi Survey', value: (row) => row.lokasi_survei },
  { key: 'status_survei', title: 'Status Survey', value: (row) => row.status_survei },
  { key: 'pic_teknik', title: 'PIC ASM', value: (row) => row.pic_teknik },
  { key: 'nama_surveyor', title: 'Surveyor', value: (row) => row.nama_surveyor },
]

/**
 * Kartu kosong yang digambar SEBELUM angkanya datang.
 *
 * Kerangkanya dibentuk dari keterangan tile — yang dibaca dari bentuk layar dan bukan dari
 * data — sehingga keempat kartu sudah pada tempatnya saat angkanya menyusul. Tanpa ini,
 * layar melompat setiap kali dimuat.
 *
 * Bila bentuk layar pun belum datang, keempatnya tetap digambar dari daftar tetap di bawah.
 * Judulnya akan tertimpa begitu server menjawab.
 */
function kartuKosong(daftar?: { tile: Tile; judul: string; bentuk: 'klaim' | 'survei' }[]): Kartu[] {
  const bawaan: { tile: Tile; judul: string; bentuk: 'klaim' | 'survei' }[] = daftar ?? [
    { tile: 'outstanding', judul: 'Outstanding', bentuk: 'klaim' },
    { tile: 'close-claim', judul: 'Close Claim', bentuk: 'klaim' },
    { tile: 'loss-adjuster', judul: 'Loss Adjuster', bentuk: 'survei' },
    { tile: 'internal-surveyor', judul: 'Internal Surveyor', bentuk: 'survei' },
  ]
  return bawaan.map((item) => ({ ...item, jumlah: 0 }))
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}
