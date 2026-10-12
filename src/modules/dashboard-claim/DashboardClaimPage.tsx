import { useEffect, useMemo, useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
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
import { DialogRincianKlaim } from './DialogRincianKlaim'
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
/**
 * Isi panel penyaring `FilterDashboardClaim`.
 *
 * Satu bentuk, bukan lima state lepas: draf dan terapan WAJIB punya bentuk yang sama, dan
 * menyalin yang satu ke yang lain harus satu baris. Lima pasang state lepas akan membuat
 * "terapkan" menjadi lima penyalinan yang salah satunya dapat terlupa.
 */
type PanelPenyaring = {
  nomorPolis: string
  nomorKlaim: string
  pic: string
  statusTransfer: string
  statusBayar: string
}

const PANEL_KOSONG: PanelPenyaring = {
  nomorPolis: '',
  nomorKlaim: '',
  pic: '',
  statusTransfer: '',
  statusBayar: '',
}

export function DashboardClaimPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const [tab, setTab] = useState<TabDashboard>('dashboard')

  const [liniBisnis, setLiniBisnis] = useState('')
  const [cari, setCari] = useState('')

  /*
    Kelima isian panel penyaring `Section/FilterDashboardClaim_sec-Section.xml`.

    Namanya mengikuti ISINYA. Di layar lama properti penyimpannya menyesatkan — Nopolis
    disimpan pada `TempInputFilter.CaseID`, PIC pada `TempInputFilter.ClaimID`, status
    pembayaran pada `.City`, dan status transfer pada `.CityID`.

    # Kenapa ada DUA salinan: draf dan terapan

    Layar lama TIDAK menyaring sambil mengetik. Panelnya punya tombol **Search Data**, dan
    penyaringnya baru berlaku saat tombol itu ditekan. Bentuk sebelumnya di sini menembak
    ulang pada setiap ketikan — lebih gesit, tetapi bukan perilaku yang direplikasi, dan
    pada tabel berpuluh juta baris ia juga mengirim satu permintaan per huruf.

    `draf` adalah apa yang sedang diketik; `terapan` adalah apa yang benar-benar menyaring.
    Keduanya dipisah supaya "yang terlihat di kotak" dan "yang sedang berlaku" tidak dapat
    menyimpang tanpa disadari.
  */
  const [draf, setDraf] = useState(PANEL_KOSONG)
  const [terapan, setTerapan] = useState(PANEL_KOSONG)
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
  /**
   * Baris yang dibawa ke Transfer All Case By UserID.
   *
   * `null` berarti dialognya tertutup; irisan kosong berarti terbuka tanpa satu pun baris
   * tercentang — dan keduanya memang keadaan yang berbeda.
   */
  /*
    Permintaan transfer massal yang sedang terbuka.

    `semuaCocok` dibawa bersama barisnya, bukan disimpan terpisah: dialog harus tahu APA yang
    dipindahkan — baris tercentang, atau seluruh hasil penyaring — dan dua state yang harus
    sepakat tentang hal yang sama adalah dua state yang akan menyimpang.
  */
  const [transferMassal, setTransferMassal] = useState<{
    baris: BarisKlaim[]
    semuaCocok: boolean
  } | null>(null)

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
  // Ringkasan menerima penyaring yang SAMA dengan telusur. Kartu yang disaring berbeda dari
  // daftarnya akan menampilkan angka yang tidak cocok dengan isi tabel di bawahnya, dan tidak
  // ada galat yang muncul.
  const penyaringAktif = {
    lini_bisnis: liniBisnis,
    cari,
    nomor_polis: terapan.nomorPolis,
    nomor_klaim: terapan.nomorKlaim,
    pic: terapan.pic,
    status_transfer: terapan.statusTransfer,
    status_bayar: terapan.statusBayar,
  }

  /**
   * Apakah ada yang bisa dikosongkan.
   *
   * Draf DAN terapan sama-sama diperiksa: kotak yang sudah dikosongkan tangan tetapi belum
   * ditekan Search Data meninggalkan penyaring yang masih berlaku, dan Clear Filter harus
   * tetap dapat ditekan untuk mencabutnya.
   */
  const panelTerisi =
    Object.values(draf).some((nilai) => nilai !== '') ||
    Object.values(terapan).some((nilai) => nilai !== '')

  /** Menerapkan isi panel — tombol "Search Data" pada layar lama. */
  function terapkanPanel() {
    setTerapan(draf)
    setHalaman(1)
  }

  /**
   * Mengosongkan panel — tombol "Clear Filter".
   *
   * Keduanya dikosongkan sekaligus. Mengosongkan draf saja akan menyisakan penyaring yang
   * MASIH berlaku sementara kotaknya sudah kosong — pengguna membaca daftar tersaring
   * tanpa satu pun petunjuk mengapa.
   */
  function kosongkanPanel() {
    setDraf(PANEL_KOSONG)
    setTerapan(PANEL_KOSONG)
    setHalaman(1)
  }

  const ringkasan = useRingkasanDashboard(penyaringAktif)
  const telusur = useTelusurDashboard(tile, { ...penyaringAktif, halaman })

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

  /**
   * Isi kedua dropdown panel, datang dari server bersama pilihan lini bisnis.
   *
   * Pilihan bernilai kosong DIBUANG di sini: `SelectField` sudah menggambar baris kosongnya
   * sendiri lewat `emptyText`, sehingga membiarkannya akan memberi dua baris "Semua …" yang
   * berbeda tulisan tetapi sama artinya.
   */
  const pilihanStatusTransfer = useMemo(
    () =>
      (penyaring.data?.status_transfer ?? [])
        .filter((p) => p.nilai !== '')
        .map((p) => ({ value: p.nilai, label: p.label })),
    [penyaring.data],
  )

  const pilihanStatusBayar = useMemo(
    () =>
      (penyaring.data?.status_bayar ?? [])
        .filter((p) => p.nilai !== '')
        .map((p) => ({ value: p.nilai, label: p.label })),
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

          {/*
            Kelima isian panel penyaring layar lama, dengan nama yang dilihat pengguna di
            sana: Nopolis, No Klaim, PIC, Status Transfer, Status Pembayaran.

            Kedua dropdown sempat TIDAK digambar, dengan alasan bahwa kolom yang disaringnya
            tidak ada pada tabel yang dibaca tile Outstanding. Alasan itu KELIRU: keduanya
            memang bukan kolom — `Activity/GCNMGetManagerCase_Act-Act.xml` menyaring Status
            Transfer lewat sub-kueri EXISTS atas tabel adjustment, dan Status Pembayaran
            lewat kode status klaim `1163` (Paid).
          */}
          <Field
            id="dashboard-nomor-polis"
            label="Nopolis"
            value={draf.nomorPolis}
            onChange={(event) =>
              setDraf((isi) => ({ ...isi, nomorPolis: event.target.value }))
            }
          />

          <Field
            id="dashboard-nomor-klaim"
            label="No Klaim"
            value={draf.nomorKlaim}
            onChange={(event) =>
              setDraf((isi) => ({ ...isi, nomorKlaim: event.target.value }))
            }
            hint="Contoh : PNC-1234"
          />

          <Field
            id="dashboard-pic"
            label="PIC"
            value={draf.pic}
            onChange={(event) =>
              setDraf((isi) => ({ ...isi, pic: event.target.value }))
            }
          />

          {/*
            Teks pilihan kosong kedua dropdown disalin APA ADANYA dari
            `Section/FilterDashboardClaim_sec-Section.xml`, elemen `pyNoSelectionText`
            (`D-13`). Section-nya diterima 2026-10-08; sebelum itu keduanya berbunyi
            "Semua Status Transfer" dan "Semua Status Pembayaran" — karangan kita sendiri.

            Huruf besar dan tanda hubungnya ikut ditiru. Ia terbaca aneh, dan memang begitu
            di layar lama — menyeragamkannya berarti mengubah teks yang sudah dikenal
            pengguna.
          */}
          <SelectField
            id="dashboard-status-transfer"
            label="Status Transfer"
            value={draf.statusTransfer}
            options={pilihanStatusTransfer}
            emptyText="---PILIH STATUS TRANSFER---"
            onChange={(event) =>
              setDraf((isi) => ({ ...isi, statusTransfer: event.target.value }))
            }
          />

          <SelectField
            id="dashboard-status-bayar"
            label="Status Pembayaran"
            value={draf.statusBayar}
            options={pilihanStatusBayar}
            emptyText="---PILIH STATUS PEMBAYARAN---"
            onChange={(event) =>
              setDraf((isi) => ({ ...isi, statusBayar: event.target.value }))
            }
          />

          {/*
            Tiga tombol panel layar lama: Search Data, Clear Filter, dan Muat ulang.

            Search Data ADA karena panelnya memang tidak menyaring sambil mengetik — lihat
            catatan pada `draf`. Clear Filter mengosongkan kotaknya DAN penyaring yang
            sedang berlaku sekaligus; mengosongkan kotaknya saja akan menyisakan daftar
            tersaring tanpa satu pun petunjuk mengapa.

            "Select All" pada layar lama TIDAK digambar di sini — lihat catatan di bawah.
          */}
          <div className="col-span-full flex flex-wrap items-end gap-2">
            <Button tone="utama" onClick={terapkanPanel} disabled={ringkasan.isFetching}>
              Search Data
            </Button>

            <Button tone="kedua" onClick={kosongkanPanel} disabled={!panelTerisi}>
              Clear Filter
            </Button>

            <Button
              tone="halus"
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
            "Transfer All Case By UserID" TIDAK lagi digambar di panel ini.

            Di layar lama ia milik `InboxOutstandingClaim_Section` — tombol di atas GRID,
            bersebelahan dengan "Select All" yang menyuapinya. Menaruhnya di panel penyaring
            memisahkannya dari centang yang menjadi isinya.
          */}
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
          kunciPenyaring={JSON.stringify(penyaringAktif)}
          onTransferMassal={(baris, semuaCocok) => setTransferMassal({ baris, semuaCocok })}
        />
      )}
        </>
      )}

      {transferMassal !== null ? (
        <DialogTransfer
          lingkup="massal"
          baris={transferMassal.baris}
          semuaCocok={transferMassal.semuaCocok}
          penyaringAktif={penyaringAktif}
          onTutup={() => setTransferMassal(null)}
        />
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
  kunciPenyaring,
  onTransfer,
  onTransferMassal,
}: {
  tile: Tile
  judul: string
  cari: string
  onCari: (nilai: string) => void
  halaman: number
  onHalaman: (halaman: number) => void
  state: ReturnType<typeof useTelusurDashboard>
  liniBisnis: string

  /**
   * Sidik jari penyaring yang sedang berlaku, untuk membuang mode "Select All".
   *
   * Dikirim sebagai satu teks, bukan objek: objek baru pada setiap penggambaran akan
   * membuat efeknya berjalan terus-menerus dan membuang mode itu sebelum sempat dipakai.
   */
  kunciPenyaring: string

  onTransfer: (row: BarisKlaim) => void
  /**
   * Membuka Transfer All Case By UserID dengan baris yang sedang tercentang.
   *
   * Barisnya dikirim ke ATAS, bukan state halaman yang diturunkan ke bawah: centang hidup
   * di sini bersama barisnya, dan menaikkannya ke halaman akan membuat dua tempat harus
   * sepakat tentang baris mana yang sedang tampil.
   */
  onTransferMassal: (baris: BarisKlaim[], semuaCocok: boolean) => void
}) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const [galatUnduh, setGalatUnduh] = useState<string | null>(null)

  /*
    Klaim yang rinciannya sedang dibuka.

    Disimpan BARISNYA, bukan hanya kuncinya: judul popup menyebut nomor klaim, dan barisnya
    sudah memuatnya. Mengambilnya ulang dari server hanya untuk judul adalah perjalanan
    jaringan yang tidak perlu.
  */
  const [rincian, setRincian] = useState<BarisKlaim | null>(null)

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

  /*
    Mode "seluruh hasil penyaring" — inilah arti "Select All".

    Ia TERPISAH dari centang per baris, dan pemisahannya disengaja. Centang per baris
    sengaja dibuang saat halaman berganti (lihat useEffect di atas): tanpa itu, pengguna
    mencentang tiga baris di halaman 1, berpindah ke halaman 2, lalu menekan ReOpen — dan
    tiga klaim yang TIDAK terlihat ikut terkirim.

    Alasan itu tetap berlaku. Yang tidak berlaku untuknya adalah "Select All", karena di sana
    pengguna memang menyatakan seluruhnya. Jadi mode ini bertahan melewati halaman, dan
    DIBUANG saat penyaringnya berubah — sebab "seluruhnya" lalu berarti himpunan yang lain.
  */
  const [semuaCocok, setSemuaCocok] = useState(false)

  useEffect(() => {
    setSemuaCocok(false)
  }, [tile, kunciPenyaring])

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

  /*
    Nomor baris dihitung dari posisi baris pada halaman INI ditambah offset halamannya.

    Peta dibangun sekali per penggambaran, bukan `indexOf` per sel: `indexOf` pada 25 baris
    dipanggil 25 kali menjadi pekerjaan kuadrat, dan pada grid yang isinya dapat tumbuh itu
    biaya yang tidak perlu dibayar.

    Kuncinya sama persis dengan `rowKey` tabel. Kunci yang berbeda akan menomori baris yang
    salah pada data yang nomor klaimnya kembar — dan data warisan memuat kasus seperti itu.
  */
  const awalHalaman = (pagination.page - 1) * pagination.size
  const petaNomor = new Map(
    barisKlaim.map((row, index) => [row.klaim_id || row.nomor_klaim, awalHalaman + index + 1]),
  )
  const nomorBaris = (row: BarisKlaim) => petaNomor.get(row.klaim_id || row.nomor_klaim) ?? 0

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

  /*
    Dua tombol tingkat grid pada Inbox Outstanding layar lama.

    "Select All" berarti SELURUH hasil penyaring — lintas halaman, bukan 25 baris pada halaman
    yang terbuka. Itu arti namanya, dan itu yang diminta Work Owner (2026-10-07).

    Centang per baris DIMATIKAN selagi mode itu hidup. Alternatifnya — membiarkan pengguna
    melepas satu baris dari "seluruhnya" — menuntut daftar pengecualian yang ikut dikirim ke
    server, dan penyaring bersisa-kecuali itu belum ada. Mematikannya membuat batasnya
    terlihat; membiarkannya membuat layar menjanjikan hal yang tidak dikerjakannya.
  */
  const totalCocok = pagination.total

  const tombolOutstanding =
    tile === 'outstanding' ? (
      <>
        <Button
          tone="kedua"
          disabled={totalCocok === 0}
          onClick={() => {
            // Keduanya diubah bersama: mode hidup berarti centang per baris tidak lagi
            // bermakna, dan meninggalkannya terisi akan membuat pembatalan mengembalikan
            // centang yang sudah tidak terlihat alasannya.
            setSemuaCocok((aktif) => !aktif)
            setDipilih(new Set())
          }}
        >
          {semuaCocok ? 'Batalkan Pilihan' : 'Select All'}
        </Button>
        <Button
          tone="kedua"
          onClick={() => onTransferMassal(semuaCocok ? [] : barisDipilih, semuaCocok)}
        >
          Transfer All Case By UserID
        </Button>
      </>
    ) : null

  /*
    Jumlahnya dinyatakan, bukan dibiarkan ditebak.

    "Select All" yang menyentuh 1.639 klaim dan yang menyentuh 3 klaim terlihat sama persis di
    layar bila angkanya tidak disebut — dan yang pertama tidak dapat dibatalkan dengan mudah.
  */
  const keteranganPilihan =
    tile === 'outstanding' && semuaCocok ? (
      <p className="mb-4 rounded-kartu border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-900">
        <strong>{totalCocok.toLocaleString('id-ID')}</strong> klaim terpilih — seluruh hasil
        penyaring, termasuk halaman lain. Centang per baris dimatikan selama pilihan ini aktif.
      </p>
    ) : null

  return (
    <>
    {peringatanUnduh}
    {keteranganPilihan}

    {rincian !== null ? (
      <DialogRincianKlaim
        klaimID={rincian.klaim_id || rincian.nomor_klaim}
        nomorKlaim={rincian.nomor_klaim}
        onTutup={() => setRincian(null)}
      />
    ) : null}

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
          ? kolomCloseClaim(dipilih, gantiPilih, nomorBaris, setRincian)
          : kolomOutstanding(onTransfer, dipilih, gantiPilih, nomorBaris, semuaCocok, setRincian)
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
          {tombolOutstanding}
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

/**
 * Menggambar stempel waktu `YYYY-MM-DD HH:MM:SS` sebagai tanggal beserta jamnya.
 *
 * Ada DI SINI, bukan di `components/format`, karena `formatDate` bersama dipakai belasan modul
 * yang sudah selesai — dan hanya grid ini yang menampilkan jam.
 *
 * Dua kolom memakainya: **Tanggal Pendaftaran** dan **Report Date**. Layar Pega menggambar
 * keduanya dengan jam (`24 Jan 20 14:54:24`); menampilkannya sebagai tanggal saja
 * menghilangkan isi, bukan sekadar perinciannya — dua klaim yang didaftarkan pada hari yang
 * sama menjadi tidak terbedakan urutannya.
 *
 * Jam `00:00:00` DIGAMBAR apa adanya, sama seperti Pega. Menyembunyikannya membuat dua
 * keadaan yang berbeda — tengah malam, dan baris yang memang tidak membawa jam — tampak sama.
 */
function tanggalJam(nilai: string): string {
  if (!nilai) return '—'

  // Bentuk lama (tanggal saja) tetap digambar benar: ini jalur yang dilalui data mana pun
  // yang belum membawa jam, dan membiarkannya jatuh ke cabang bawah akan menampilkan
  // "— 00:00:00" untuk tanggal yang sebenarnya sah.
  const [tanggal, jam] = nilai.split(' ')
  if (!tanggal) return '—'
  if (!jam) return formatDate(tanggal)

  return `${formatDate(tanggal)} ${jam}`
}

/**
 * Nomor klaim sebagai tautan — di Pega ia pranala, dan di sini pun.
 *
 * # Ke mana ia menuju, dan kenapa ke sana
 *
 * Preseden modul lain: tautkan ke modul yang benar-benar menggantikan Flow Action yang Pega
 * gambar untuk kelas objek kerja baris itu. `inbox-claim-treaty-non-prop` menempuh itu dan
 * berakhir di `input-acceptation`, karena `InputAcceptation` satu-satunya Flow Action yang
 * terdaftar pada kelasnya.
 *
 * Baris di sini berkelas **`ASM-FW-GCNMFW-Work-PNC`** — case klaim utama — dan layar yang
 * dibukanya adalah harness **`PNCViewClaim`**, yang **tidak ada di export** (`R-16`;
 * `catatan-pengembangan.md:1832`). Jadi bentuk layarnya pun belum diketahui, apalagi modul
 * penggantinya.
 *
 * Yang dipakai karena itu `/view-claim/:referensi` — rute yang sudah dicadangkan dan sudah
 * ditautkan **tujuh inbox lain**. Pengguna sampai di penampung yang menyatakan layarnya belum
 * dibangun; itu jujur, dan konsisten dengan seluruh aplikasi. Begitu modul View Claim ada,
 * satu rute berubah untuk delapan layar sekaligus — bukan delapan perubahan.
 *
 * # Yang dikirim NOMOR klaim, bukan kunci teknis Pega
 *
 * Alamatnya terbaca orang, dapat disalin ke percakapan, dan tidak membocorkan bentuk kunci
 * internal Pega ke bilah alamat (`D-22`). Kunci teknisnya tetap dikirim server pada setiap
 * baris, sehingga beralih memakainya kelak tidak menuntut perubahan kontrak.
 *
 * # Baris tanpa nomor klaim TIDAK menjadi tautan
 *
 * Tautan beralamat kosong tetap dapat diklik dan membawa pengguna ke layar yang pasti gagal.
 * Data warisan memuat baris seperti itu — uji adapter memori memakainya sebagai contoh.
 */
function TautanKlaim({
  row,
  onBuka,
}: {
  row: BarisKlaim
  onBuka: (row: BarisKlaim) => void
}) {
  if (!row.nomor_klaim) return <span className='text-slate-400'>—</span>

  return (
    <button
      type='button'
      onClick={() => onBuka(row)}
      className={[
        'font-medium text-blue-700 underline-offset-2 hover:underline',
        'focus:outline-none focus-visible:rounded-kontrol',
        'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
        'focus-visible:outline-blue-600',
      ].join(' ')}
    >
      {row.nomor_klaim}
    </button>
  )
}

/** Kolom bersama kedua tile bertipe klaim, dalam urutan yang sama dengan layar lama. */
function kolomKlaimDasar(onBuka: (row: BarisKlaim) => void): Column<BarisKlaim>[] {
  return [
  {
    key: 'nomor_klaim',
    title: 'No Klaim',
    value: (row) => row.nomor_klaim,
    width: '9rem',
    render: (row) => <TautanKlaim row={row} onBuka={onBuka} />,
  },
  { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
  { key: 'nama_bisnis', title: 'Nama Bisnis', value: (row) => row.nama_bisnis },
  { key: 'sumber_bisnis', title: 'Sumber Bisnis', value: (row) => row.sumber_bisnis },
  { key: 'nama_cabang', title: 'Nama Cabang', value: (row) => row.nama_cabang },
  {
    key: 'tanggal_pendaftaran',
    title: 'Tanggal Pendaftaran',
    value: (row) => row.tanggal_pendaftaran,
    render: (row) => tanggalJam(row.tanggal_pendaftaran),
    width: '12rem',
  },
  ]
}

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
  dipilih: Set<string>,
  onPilih: (klaimID: string) => void,
  nomor: (row: BarisKlaim) => number,
  semuaCocok: boolean,
  onBuka: (row: BarisKlaim) => void,
): Column<BarisKlaim>[] {
  return [
  kolomNomor(nomor),
  // Layar lama menggambar "Pilih" di grid INI juga, dan centangnya melayani
  // "Transfer All Case By UserID".
  kolomPilih(dipilih, onPilih, semuaCocok),
  ...kolomKlaimDasar(onBuka),
  {
    key: 'tanggal_lapor',
    title: 'Report Date',
    value: (row) => row.tanggal_lapor,
    render: (row) => tanggalJam(row.tanggal_lapor),
    width: '12rem',
  },
  kolomLamaWaktu,
  ...kolomPICAdmin,
  { key: 'status_klaim_label', title: 'Claim status', value: (row) => row.status_klaim_label },

  /*
    Posisi Klaim dan Progress Klaim DICABUT 2026-10-07.

    Keduanya dibangun atas pembacaan `InboxOutstandingClaim_Section:20167` dan `:20279`,
    tempat `.CloseClaimNote` dan `.ClaimNo` terikat sebagai nilai sel ber-`pyVisible=ALWAYS`.
    Pembacaan itu KELIRU: Work Owner menggulir grid Pega ke kanan, dan setelah "Claim status"
    langsung tombol Transfer.

    `pyVisible=ALWAYS` membuktikan satu elemen terlihat — BUKAN bahwa ia kolom pada grid yang
    sedang dicari. Ikatan di baris itu rupanya milik tata letak lain di section yang sama.

    Ikut dicabut sampai ke pangkalnya: `posisi.sql`, `posisi.go`, `attachPositions`,
    `ClaimRow.Position`/`.Progress`, dan kedua field DTO — seluruh rantainya hanya melayani
    kedua kolom ini, termasuk SATU KUERI TAMBAHAN per halaman.
  */
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
 * Kolom **"Pilih"** — kotak centang per baris.
 *
 * Dipakai DUA grid, dan keduanya menggambarnya di layar lama: `InboxManagerReopen1_Sec`
 * untuk Close Claim (melayani ReOpen dan Copy Klaim) dan `InboxOutstandingClaim_Section`
 * untuk Outstanding (melayani Transfer All Case By UserID).
 *
 * Satu definisi untuk keduanya, bukan dua salinan: bentuk dan label aksesibilitasnya wajib
 * sama, dan dua salinan akan menyimpang pada perubahan berikutnya.
 */
/**
 * Kolom nomor baris — kolom paling kiri pada grid Pega.
 *
 * PERHATIAN: penomorannya BERLANJUT antar halaman (halaman 2 mulai dari 26), bukan mengulang
 * dari 1. Itu satu-satunya hal di grid ini yang saya PILIH, bukan salin — tangkapan layar
 * Pega hanya memperlihatkan halaman pertama, sehingga perilakunya di halaman kedua tidak
 * terbaca dari sana.
 *
 * Dipilih berlanjut supaya tidak bertentangan dengan keterangan di bawah tabel, yang berbunyi
 * 'Menampilkan 26–50 dari 1.639 baris' pada halaman yang sama. Nomor yang mengulang dari 1 di
 * sebelah keterangan itu akan membuat keduanya saling membantah.
 */
function kolomNomor(nomor: (row: BarisKlaim) => number): Column<BarisKlaim> {
  return {
    key: 'nomor_baris',
    title: '',
    // Bukan data: tidak diurutkan dan tidak ikut dicari.
    noSort: true,
    value: () => '',
    width: '3rem',
    render: (row) => <span className='text-slate-400'>{nomor(row)}</span>,
  }
}

function kolomPilih(
  dipilih: Set<string>,
  onPilih: (klaimID: string) => void,
  semuaCocok: boolean,
): Column<BarisKlaim> {
  return {
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
          // Mode "seluruh hasil penyaring" menggambar SETIAP baris tercentang, di halaman
          // mana pun — itu yang membedakannya dari centang per halaman. Dimatikan karena
          // melepas satu baris darinya menuntut daftar pengecualian yang belum ada.
          checked={semuaCocok || dipilih.has(id)}
          disabled={semuaCocok}
          onChange={() => onPilih(id)}
          // Nomor klaimnya disebut, bukan "Pilih" saja: pembaca layar membacakan 25 kotak
          // centang berturut-turut, dan label yang sama persis membuat keduanya tidak
          // dapat dibedakan.
          aria-label={`Pilih klaim ${row.nomor_klaim}`}
          className="h-4 w-4 rounded border-slate-300 text-blue-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
        />
      )
    },
  }
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
 * # Kolom "Pilih" — urutannya BERBEDA dari Outstanding
 *
 * Kotak centangnya ada di kolom **kedua**, sesudah No Klaim. Itu bukan selera: baris header
 * `Section/InboxManagerReopen1_Sec-Section.xml` menyusunnya `No Klaim` lalu `Pilih`,
 * sedangkan `InboxOutstandingClaim_Section` menyusunnya terbalik — `Pilih` lalu `No Klaim`.
 * Kedua urutan ditiru apa adanya, bukan diseragamkan.
 *
 * Centangnya melayani kedua tombol di atas tabel — ReOpen dan Copy Klaim.
 *
 * **Tanpa "Select All" — berbeda dari Outstanding.** Section lamanya memuat tepat SATU
 * kontrol checkbox, yaitu yang per baris; tidak ada tombol pilih-semua. Outstanding punya
 * tombol itu dan Close Claim tidak, dan perbedaan itu ditiru apa adanya — menambahkannya di
 * sini akan memudahkan pekerjaan yang akibatnya tidak dapat dibatalkan: mencentang seluruh
 * halaman lalu menekan ReOpen mencatat 25 permintaan sekaligus.
 */
function kolomCloseClaim(
  dipilih: Set<string>,
  onPilih: (klaimID: string) => void,
  nomor: (row: BarisKlaim) => number,
  onBuka: (row: BarisKlaim) => void,
): Column<BarisKlaim>[] {
  // `No Klaim` mendahului `Pilih` di sini — lihat keterangan di atas.
  const [noKlaim, ...sisaKlaimDasar] = kolomKlaimDasar(onBuka)
  return [
  kolomNomor(nomor),
  ...(noKlaim ? [noKlaim] : []),
  kolomPilih(dipilih, onPilih, false),
  ...sisaKlaimDasar,
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
