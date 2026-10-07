import { useMemo, useState } from 'react'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'
import { ReloadIcon } from '@/components/Icon'
import { SelectField } from '@/components/SelectField'

import { PAGE_SIZE, usePenyaringDashboard, useRingkasanDashboard, useTelusurDashboard } from './api'
import { KartuPenghitung } from './KartuPenghitung'
import type { BarisKlaim, BarisSurvei, Kartu, Tile } from './types'

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

  const [liniBisnis, setLiniBisnis] = useState('')
  const [cari, setCari] = useState('')
  const [tile, setTile] = useState<Tile | null>(null)
  const [halaman, setHalaman] = useState(1)

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
        />
      )}
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
}: Readonly<{
  tile: Tile
  judul: string
  cari: string
  onCari: (nilai: string) => void
  halaman: number
  onHalaman: (halaman: number) => void
  state: ReturnType<typeof useTelusurDashboard>
}>) {
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

  if (bentuk === 'survei') {
    return (
      <DataTable<BarisSurvei>
        title={judul || 'Telusur'}
        label={`Daftar survei ${judul}`}
        columns={kolomSurvei}
        rows={data?.survei ?? []}
        rowKey={(row) => row.survei_id || row.nomor_survei}
        isLoading={state.isPending}
        error={galat}
        serverSearch={{ value: cari, onChange: onCari }}
        searchLabel="Cari No Survey atau No Polis"
        emptyMessage="Tidak ada survei yang cocok dengan penyaring."
        showHeaderWhenEmpty
        pagination={pagination}
      />
    )
  }

  return (
    <DataTable<BarisKlaim>
      title={judul || 'Telusur'}
      label={`Daftar klaim ${judul}`}
      columns={kolomKlaim}
      rows={data?.klaim ?? []}
      rowKey={(row) => row.klaim_id || row.nomor_klaim}
      isLoading={state.isPending}
      error={galat}
      serverSearch={{ value: cari, onChange: onCari }}
      searchLabel="Cari No Klaim atau No Polis"
      emptyMessage="Tidak ada klaim yang cocok dengan penyaring."
      showHeaderWhenEmpty
      pagination={pagination}
    />
  )
}

/**
 * Kolom telusur bertipe klaim.
 *
 * Keenam kolom pertama mengikuti `Section/DashboardClaim_Section2-Section.xml` apa adanya —
 * No Klaim, No Polis, Nama Tertanggung, Nama Bisnis, Sumber Bisnis, Nama Cabang.
 *
 * `klaim_id` TIDAK dijadikan kolom: ia kunci teknis Pega yang memuat nama kelas internal,
 * dan `D-22` menetapkannya tidak ditampilkan kepada pengguna.
 */
const kolomKlaim: Column<BarisKlaim>[] = [
  { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim, width: '10rem' },
  { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
  { key: 'nama_bisnis', title: 'Nama Bisnis', value: (row) => row.nama_bisnis },
  { key: 'sumber_bisnis', title: 'Sumber Bisnis', value: (row) => row.sumber_bisnis },
  { key: 'nama_cabang', title: 'Nama Cabang', value: (row) => row.nama_cabang },
  { key: 'pic_teknik', title: 'PIC Teknik', value: (row) => row.pic_teknik },
  {
    key: 'tanggal_register',
    title: 'Tanggal Register',
    value: (row) => row.tanggal_register,
    render: (row) => formatDate(row.tanggal_register),
    width: '10rem',
  },
]

/**
 * Kolom telusur bertipe survei.
 *
 * Mengikuti `Section/DashboardClaim_Section1-Section.xml` — No Klaim, No Polis, Nama
 * Tertanggung, Surveyor, Tanggal Survey, Status Survey, Status Register.
 *
 * Tanggal Survey hanya terisi pada tile Internal Surveyor; kueri loss adjuster memang tidak
 * mengambilnya, dan kolomnya dibiarkan kosong alih-alih diisi tanggal lain yang kebetulan
 * ada.
 */
const kolomSurvei: Column<BarisSurvei>[] = [
  { key: 'nomor_survei', title: 'No Survey', value: (row) => row.nomor_survei, width: '10rem' },
  { key: 'nomor_klaim', title: 'No Klaim', value: (row) => row.nomor_klaim, width: '10rem' },
  { key: 'nomor_polis', title: 'No Polis', value: (row) => row.nomor_polis, width: '11rem' },
  { key: 'nama_tertanggung', title: 'Nama Tertanggung', value: (row) => row.nama_tertanggung },
  { key: 'nama_surveyor', title: 'Surveyor', value: (row) => row.nama_surveyor },
  { key: 'pic_teknik', title: 'PIC Teknik', value: (row) => row.pic_teknik },
  { key: 'lokasi_survei', title: 'Lokasi Survei', value: (row) => row.lokasi_survei },
  {
    key: 'tanggal_survei',
    title: 'Tanggal Survey',
    value: (row) => row.tanggal_survei,
    render: (row) => (row.tanggal_survei ? formatDate(row.tanggal_survei) : '—'),
    width: '10rem',
  },
  { key: 'status_survei', title: 'Status Survey', value: (row) => row.status_survei },
  { key: 'status_proses', title: 'Status Register', value: (row) => row.status_proses },
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
