import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { Component, DetailRow, MetadataResponse, SummaryRow } from './types'

const PATH = '/api/report-kpi'
const TAB_PATH = `${PATH}/tab`
const SUMMARY_PATH = `${PATH}/adjuster/ringkasan`
const ADJUSTER_PATH = `${PATH}/adjuster/pilihan`
const EXPORT_PATH = `${PATH}/adjuster/ekspor`
const DETAIL_PATH = `${PATH}/adjuster`

const SAMPLE_PROFILE = {
  identitas: '90000009',
  nama: 'Contoh Penyelia KPI',
  jenis: 'KARYAWAN',
  login: 'penyeliakpicontoh',
  email: 'contoh.kpi@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kesembilan komponen, DALAM URUTAN yang dikirim server.
 *
 * Urutan itu ikut diuji, dan alasannya khas layar ini: kesembilannya berisi angka 1–5,
 * sehingga satu kolom yang tergeser tidak terlihat sebagai kerusakan — ia hanya terlihat
 * sebagai nilai yang berbeda.
 */
const KOMPONEN: Component[] = [
  { kode: 'penjadwalan_survey', judul: 'PENJADWALAN SURVEY', kolom: 'SURVEYLAP' },
  { kode: 'immediate_advice', judul: 'IMMEDIATE ADVICE', kolom: 'IMMEDIATEADVICE' },
  { kode: 'preliminary_advice', judul: 'PRELIMINARY ADVICE', kolom: 'PRELIMINARYADVICE' },
  { kode: 'interim_report', judul: 'INTERIM REPORT', kolom: 'INTERIM' },
  { kode: 'update_progress', judul: 'UPDATE PROGRESS', kolom: 'PROGRESS' },
  { kode: 'tanggapan_komunikasi', judul: 'TANGGAPAN KOMUNIKASI', kolom: 'KOMUNIKASI' },
  { kode: 'propose_adjustment', judul: 'PROPOSE ADJUSTMENT', kolom: 'PROPOSE' },
  { kode: 'final_report', judul: 'FINAL REPORT', kolom: 'FINALREPORT' },
  { kode: 'nilai', judul: 'NILAI', kolom: 'NILAI' },
]

const METADATA: MetadataResponse = {
  tab: [
    {
      kode: 'pic-teknik',
      judul: 'KPI PIC Teknik',
      grid: [
        {
          kode: 'kartu-skor-pic',
          judul: 'Data KPI PIC Teknik',
          kolom: [
            { kunci: 'pic', judul: 'PIC' },
            { kunci: 'kpi', judul: 'KPI' },
            { kunci: 'total', judul: 'Total' },
            { kunci: 'tercapai', judul: 'Tercapai' },
            { kunci: 'persentase', judul: 'Persentase' },
            { kunci: 'nilai', judul: 'Nilai' },
          ],
        },
      ],
      terhalang: false,
    },
    {
      kode: 'adjuster',
      judul: 'KPI Adjuster',
      grid: [
        {
          kode: 'ringkasan',
          judul: 'Summary KPI Adjuster',
          kolom: [
            { kunci: 'adjuster', judul: 'ADJUSTER' },
            // Hanya pada tipe ALL — begitulah Pega: hanya kueri penggabung
            // (`GetSummaryKPIAdjusterALL`) yang mengembalikan kolom tipe.
            { kunci: 'tipe', judul: 'TIPE', hanya_tipe_gabungan: true },
          ],
        },
        {
          kode: 'rincian',
          judul: 'Detail KPI Adjuster',
          kolom: [
            { kunci: 'adjuster', judul: 'ADJUSTER' },
            { kunci: 'no_case', judul: 'NO CASE' },
            { kunci: 'tipe', judul: 'TIPE' },
            { kunci: 'tanggal', judul: 'TANGGAL' },
          ],
        },
      ],
      terhalang: false,
    },
    {
      kode: 'admin',
      judul: 'KPI Admin',
      grid: [
        // Kartu skor BUKAN tabel — ia tidak punya kolom. Bentuknya datang dari metrik
        // yang dikirim server, bukan dari susunan kolom.
        { kode: 'kartu-skor', judul: 'Data KPI', kolom: [] },
        {
          kode: 'rincian-klaim',
          judul: 'Rincian Klaim',
          kolom: [
            { kunci: 'no_klaim', judul: 'No Klaim' },
            { kunci: 'no_polis', judul: 'No Polis' },
            { kunci: 'business', judul: 'BUSINESS', hanya_kelompok: 'NONMBU' },
            { kunci: 'tgl_regist_klaim', judul: 'Tgl Regist Klaim' },
            // Judulnya SAMA dengan baris di bawah, dan itu bukan salah salin: layar lama
            // menamai keduanya begitu meski kolom basis datanya berbeda (`D-13`).
            { kunci: 'tgl_terima_dokumen', judul: 'Tgl Terima Dokumen', hanya_kelompok: 'NONMBU' },
            { kunci: 'flag', judul: 'Flag', hanya_kelompok: 'NONMBU' },
            { kunci: 'aging_regist_klaim', judul: 'Aging Regist Klaim' },
            { kunci: 'tgl_terima_dokumen_pa', judul: 'Tgl Terima Dokumen', hanya_kelompok: 'PA' },
            { kunci: 'status_sla_regist', judul: 'Status SLA Regist Klaim', hanya_kelompok: 'PA' },
            { kunci: 'tgl_terima_lod', judul: 'Tgl Terima LOD', hanya_kelompok: 'PA' },
            { kunci: 'tgl_pembayaran', judul: 'Tgl Pembayaran', hanya_kelompok: 'PA' },
            { kunci: 'aging_pembayaran_klaim', judul: 'Aging Pembayaran klaim', hanya_kelompok: 'PA' },
            { kunci: 'status_sla_pembayaran', judul: 'Status SLA Pembayaran Klaim', hanya_kelompok: 'PA' },
          ],
        },
      ],
      terhalang: false,
    },
  ],
  tab_bawaan: 'adjuster',
  komponen: KOMPONEN,
  tipe_report: [
    { kode: 'OUTSTANDING', judul: 'OUTSTANDING', keterangan: 'Kasus yang masih berjalan.' },
    { kode: 'FINAL', judul: 'FINAL', keterangan: 'Kasus yang laporan akhirnya sudah masuk.' },
    {
      kode: 'ALL',
      judul: 'ALL',
      keterangan: 'Keduanya sekaligus. Satu adjuster tampil dua baris — dan itu bukan baris ganda.',
    },
  ],
  selisih_terencana: [
    'Layar ini MEMBACA saja. Di Pega, menekan Cari menghitung ulang lalu menyimpannya.',
    'Nilai komponen yang kosong ditampilkan sebagai tanda hubung, bukan sebagai 0.',
  ],
  kelompok_admin: [
    { kode: 'NONMBU', judul: 'NON MBU' },
    { kode: 'PA', judul: 'PA' },
  ],
  selisih_terencana_admin: [
    'Pembagian dijaga NULLIF: periode tanpa klaim menghasilkan nilai kosong, bukan galat.',
  ],
  koordinator_di_kueri: 'YUSMIARSIH DYAHPUSPITA S',

  lini_bisnis: [
    { kode: 'NONMBU', judul: 'NON MBU', keterangan: 'Group Panel 003, 004, dan 006.' },
    { kode: 'PA', judul: 'PA' },
    { kode: 'TRAVEL', judul: 'TRAVEL' },
    { kode: 'BONDING', judul: 'BONDING' },
  ],
  komponen_pic: [
    // Dua yang bertangga MENURUN diberi penandanya — itu yang diuji di bawah.
    {
      kode: 'update_progress',
      judul: 'Update Status Progress (max terlambat 25%)',
      tangga_menurun: true,
      berbobot: true,
    },
    { kode: 'analisa_klaim', judul: 'Analisa Klaim (max 10 hari)' },
    { kode: 'akseptasi_klaim', judul: 'Akseptasi Klaim ( 1 hari )' },
    { kode: 'sla_klaim', judul: 'SLA Klaim', tangga_menurun: true },
  ],
  selisih_terencana_pic: [
    'NILAI BERLAWANAN ARAH PADA DUA KOMPONEN — direplikasi dari Pega.',
  ],
  pic_dikecualikan_sla: ['BAMBANGSETIADJIGUNAWAN', 'DANIELLISWANDI'],

  tabel_sumber: 'POOLDATA.DETAIL_KPI_ADJUSTER',
  tabel_tangga_nilai: 'POOLDATA.M_KPI_PNC',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `interim_report` sengaja `null`: begitulah server mengirim komponen yang belum dinilai.
 * Uji di bawah memastikan layar menggambarnya sebagai tanda hubung, bukan sebagai 0.
 */
const RINGKASAN: SummaryRow = {
  adjuster: 'PT ADJUSTER NUSA CONTOH',
  tipe: 'FINAL',
  nilai: {
    penjadwalan_survey: 3.5,
    immediate_advice: 4,
    preliminary_advice: 3,
    interim_report: null,
    update_progress: 4.25,
    tanggapan_komunikasi: 3,
    propose_adjustment: 3,
    final_report: 2,
    nilai: 3.28,
  },
}

const RINCIAN: DetailRow = {
  adjuster: 'PT ADJUSTER NUSA CONTOH',
  no_case: 'CONTOH-KPI-0001',
  tipe: 'FINAL',
  tanggal: '2026-03-04',
  nilai: {
    penjadwalan_survey: 3,
    immediate_advice: 4,
    preliminary_advice: 3,
    interim_report: null,
    update_progress: 4,
    tanggapan_komunikasi: 3,
    propose_adjustment: 3,
    final_report: 2,
    nilai: 3.2,
  },
}

type Call = { url: string; init?: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

/**
 * penyaringOf membaca kembali penyaring dari alamat permintaan.
 *
 * Peladen nyata mengirim balik penyaring yang BENAR-BENAR dipakainya, dan layar memakai
 * nilai itu — bukan isian yang sedang diketik — untuk memutuskan kolom mana yang digambar.
 * Tiruan yang mengembalikan nilai tetap akan membuat uji perpindahan tipe report lulus
 * tanpa membuktikan apa pun.
 */
function penyaringOf(url: string) {
  const params = new URL(url, 'http://uji.invalid').searchParams
  return {
    tipe_report: params.get('tipe_report') ?? '',
    adjuster: params.get('adjuster') ?? '',
    dari: params.get('dari') ?? '',
    sampai: params.get('sampai') ?? '',
  }
}

/** Peladen tiruan yang menjawab bentuk layar, kedua grid, dan daftar pilihan. */
function stubDefaultFetch(options?: {
  ringkasan?: SummaryRow[]
  rincian?: DetailRow[]
}) {
  const ringkasan = options?.ringkasan ?? [RINGKASAN]
  const rincian = options?.rincian ?? [RINCIAN]

  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)

    if (url.startsWith(EXPORT_PATH)) {
      return new Response('ADJUSTER\nPT ADJUSTER NUSA CONTOH\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="ringkasan-kpi-adjuster.csv"',
        },
      })
    }

    if (url.startsWith(ADJUSTER_PATH)) {
      return jsonResponse(200, {
        adjuster: ['PT ADJUSTER NUSA CONTOH', 'CV SURVEI CONTOH SEJAHTERA'],
        penyaring: penyaringOf(url),
        portal: 'ASM',
      })
    }

    if (url.startsWith(SUMMARY_PATH)) {
      return jsonResponse(200, { baris: ringkasan, penyaring: penyaringOf(url), portal: 'ASM' })
    }

    // Diletakkan PALING BAWAH: `/adjuster` adalah awalan dari ketiga jalur di atasnya,
    // sehingga memeriksanya lebih dulu akan menelan permintaan ringkasan dan pilihan.
    if (url.startsWith(DETAIL_PATH)) {
      return jsonResponse(200, {
        baris: rincian,
        paginasi: {
          halaman: 1,
          ukuran: 50,
          total: rincian.length,
          total_halaman: 1,
        },
        penyaring: penyaringOf(url),
        portal: 'ASM',
      })
    }

    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak dikenali' })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/report-kpi']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada bilah tab, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara tab baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: /KPI Adjuster/ })
}

/** isiPenyaring mengisi ketiga isian wajib lalu menekan Cari. */
async function cari(user: ReturnType<typeof userEvent.setup>) {
  await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), 'FINAL')
  await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

function lastCallStartingWith(prefix: string): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(prefix))
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('bentuk layar', () => {
  it('menggambar ketiga tab', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /KPI PIC Teknik/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /KPI Adjuster/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /KPI Admin/ })).toBeInTheDocument()
  })

  // Tab yang belum dibangun TIDAK dapat diklik. Membiarkannya dapat dipilih lalu
  // menampilkan layar kosong akan terbaca sebagai kerusakan, bukan sebagai pekerjaan yang
  // belum selesai.
  // Ketiga tab kini aktif. Uji ini dipertahankan — bukan dihapus — supaya tab yang kelak
  // ditandai terhalang lagi tidak diam-diam lolos sebagai tab yang dapat diklik.
  it('mengaktifkan ketiga tab karena ketiganya sudah dibangun', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /KPI PIC Teknik/ })).toBeEnabled()
    expect(screen.getByRole('tab', { name: /KPI Adjuster/ })).toBeEnabled()
    expect(screen.getByRole('tab', { name: /KPI Admin/ })).toBeEnabled()

    expect(screen.queryByText(/belum tersedia/)).not.toBeInTheDocument()
  })

  // Sebelum Cari ditekan, layar TIDAK menampilkan tabel kosong.
  //
  // Tabel kosong tidak terbedakan dari "tidak ada data", dan itu kekeliruan yang mahal di
  // layar penilaian kinerja: penyelia menyimpulkan adjusternya tidak bekerja.
  it('belum menampilkan tabel sebelum Cari ditekan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByText('Summary KPI Adjuster')).not.toBeInTheDocument()
    expect(screen.queryByText('Detail KPI Adjuster')).not.toBeInTheDocument()
    expect(
      screen.getByText(/Layar lama pun menuntut keduanya sebelum menampilkan apa pun/),
    ).toBeInTheDocument()
  })
})

describe('penyaring', () => {
  it('mengirim ketiga penyaring ke kedua grid', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await cari(userEvent.setup())

    await screen.findByText('Summary KPI Adjuster')

    const summary = lastCallStartingWith(SUMMARY_PATH)
    expect(summary?.url).toContain('tipe_report=FINAL')
    expect(summary?.url).toContain('dari=2026-03-01')
    expect(summary?.url).toContain('sampai=2026-03-31')
  })

  // Penyaring yang dipakai ringkasan dan rincian WAJIB sama. Berkas dan tabel yang
  // penyaringnya berbeda tidak dapat dicocokkan pengguna dengan apa pun.
  it('memakai penyaring yang sama untuk ringkasan, rincian, dan ekspor', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()
    await cari(user)

    await screen.findByText('Detail KPI Adjuster')
    await user.click(screen.getByRole('button', { name: 'Export Ringkasan' }))

    const wanted = ['tipe_report=FINAL', 'dari=2026-03-01', 'sampai=2026-03-31']
    for (const part of wanted) {
      expect(lastCallStartingWith(SUMMARY_PATH)?.url).toContain(part)
      expect(lastCallStartingWith(DETAIL_PATH)?.url).toContain(part)
      expect(lastCallStartingWith(EXPORT_PATH)?.url).toContain(part)
    }
  })

  // Tombol ekspor mati sebelum Cari ditekan: berkasnya mengikuti penyaring yang SEDANG
  // ditampilkan, dan belum ada yang ditampilkan.
  it('mematikan tombol ekspor sebelum Cari ditekan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('button', { name: 'Export Ringkasan' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export Rincian' })).toBeDisabled()
  })

  // Keterangan tipe report digambar, bukan hanya sebagai atribut `title`: satu pilihan —
  // ALL — berperilaku berbeda dari dua lainnya, dan itu tidak terbaca dari namanya.
  it('menjelaskan tipe report yang sedang dipilih', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()

    await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), 'ALL')
    expect(screen.getByText(/dua baris/i)).toBeInTheDocument()
  })
})

describe('grid', () => {
  it('menggambar kesembilan kolom komponen sesuai urutan server', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await cari(userEvent.setup())

    await screen.findByText('Summary KPI Adjuster')

    for (const component of KOMPONEN) {
      expect(
        screen.getAllByRole('columnheader', { name: new RegExp(component.judul) }).length,
      ).toBeGreaterThan(0)
    }
  })

  // Kolom TIPE pada grid Summary HANYA muncul pada tipe report ALL.
  //
  // Meniru Pega apa adanya: hanya kueri penggabung (`GetSummaryKPIAdjusterALL`) yang
  // mengembalikan kolom itu — ditulis sebagai literal `'OUTSTANDING'` dan `'FINAL'`.
  // Menggambarnya pada tipe tunggal menambah kolom yang tidak ada di layar lama (`D-13`),
  // dan isinya pun tidak berarti apa-apa di sana.
  it('menyembunyikan kolom TIPE pada tipe report tunggal', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await cari(userEvent.setup())

    const table = await screen.findByRole('table', { name: /Ringkasan KPI per adjuster/ })
    expect(
      within(table).queryByRole('columnheader', { name: /TIPE/ }),
    ).not.toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: /ADJUSTER/ })).toBeInTheDocument()
  })

  it('menampilkan kolom TIPE pada tipe report ALL', async () => {
    stubDefaultFetch()
    const user = userEvent.setup()
    await renderLoaded()

    await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), 'ALL')
    await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
    await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    const table = await screen.findByRole('table', { name: /Ringkasan KPI per adjuster/ })
    expect(
      await within(table).findByRole('columnheader', { name: /TIPE/ }),
    ).toBeInTheDocument()
  })

  it('menampilkan baris ringkasan dan rincian', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await cari(userEvent.setup())

    await screen.findByText('Detail KPI Adjuster')

    expect(screen.getAllByText('PT ADJUSTER NUSA CONTOH').length).toBeGreaterThan(0)
    expect(screen.getByText('CONTOH-KPI-0001')).toBeInTheDocument()
  })

  // Nilai kosong DIGAMBAR sebagai tanda hubung, bukan sebagai 0.
  //
  // Keduanya berbeda artinya pada laporan penilaian kinerja: yang pertama berarti
  // komponennya belum dinilai, yang kedua berarti adjuster tidak mendapat poin — dan hanya
  // yang kedua yang menurunkan nilainya.
  it('menggambar komponen yang belum dinilai sebagai tanda hubung, bukan nol', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await cari(userEvent.setup())

    const table = await screen.findByRole('table', { name: /Ringkasan KPI per adjuster/ })
    const row = within(table).getByText('PT ADJUSTER NUSA CONTOH').closest('tr')

    expect(row).not.toBeNull()
    expect(within(row as HTMLElement).getByText('—')).toBeInTheDocument()
    expect(within(row as HTMLElement).queryByText('0')).not.toBeInTheDocument()
  })
})

describe('galat', () => {
  // Pelanggaran validasi ditandai PER ISIAN, bukan sebagai satu pesan di atas form.
  //
  // Server mengirim ketiganya sekaligus (`P-5`), dan layar harus menaruh masing-masing di
  // isiannya — pada form berisi empat isian, satu pesan umum memaksa pengguna menebak.
  it('menandai isian yang ditolak server', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      return jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Permintaan belum benar.',
        detail: [
          { field: 'tipe_report', pesan: 'Tipe report wajib dipilih.' },
          { field: 'dari', pesan: 'Periode "Dari" wajib diisi.' },
        ],
      })
    })

    await renderLoaded()

    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    expect(await screen.findByText('Tipe report wajib dipilih.')).toBeInTheDocument()
    expect(screen.getByText('Periode "Dari" wajib diisi.')).toBeInTheDocument()
  })
})

describe('selisih terencana', () => {
  // Panel selisih DIHAPUS dari layar atas keputusan Work Owner (2026-09-25): layarnya
  // mengikuti Pega apa adanya, sehingga panel yang menjelaskan selisih menjadi tambahan
  // yang tidak ada di sana.
  //
  // Uji ini menegaskan penghapusannya alih-alih menghapus ujinya. Menghapus ujinya akan
  // membuat panel yang kelak ditambahkan kembali lolos tanpa ada yang menyadarinya —
  // sedangkan keterangannya sendiri TIDAK hilang: ia tetap dikirim pada `/tab` dan tetap
  // tertulis di dokumen, tempat penguji gerbang 1 membacanya.
  it('tidak menggambar panel selisih terhadap layar Pega', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.queryByText(/Selisih terencana terhadap layar Pega/),
    ).not.toBeInTheDocument()
    expect(screen.queryByText('POOLDATA.DETAIL_KPI_ADJUSTER')).not.toBeInTheDocument()
  })

  // Keterangannya tetap SAMPAI ke layar lewat `/tab`, hanya tidak digambar.
  //
  // Itu yang membedakan "dihapus dari tampilan" dari "dihapus dari kontrak" — dan yang
  // kedua akan menghilangkan rekaman yang dituntut `D-54`.
  it('tetap menerima keterangan selisih dari server meski tidak menggambarnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const tab = lastCallStartingWith(TAB_PATH)
    expect(tab).toBeDefined()
    expect(METADATA.selisih_terencana.length).toBeGreaterThan(0)
  })
})

/* ══════════════════════════ Tab KPI Admin ══════════════════════════ */

const SCORECARD_PATH = `${PATH}/admin/kartu-skor`
const ADMIN_EXPORT_PATH = `${PATH}/admin/ekspor`
const ADMIN_DETAIL_PATH = `${PATH}/admin`

/**
 * Kartu skor contoh — seluruh angkanya KARANGAN, seperti baris adjuster di atas.
 *
 * `nilai` `null` pada satu metrik disengaja: begitulah server mengirim metrik yang TIDAK
 * dapat dihitung karena pembaginya nol. Uji di bawah memastikan layar membedakannya dari
 * nol, karena pada kartu penilaian kinerja keduanya berarti hal yang sangat berbeda.
 */
const KARTU_SKOR = {
  identitas: {
    kategori: 'ALL (NON HEALTH DAN NON MBU)',
    nama_koordinator: 'Contoh Koordinator Admin',
    nik: '90000001',
    unit_kerja: 'ALL (NON HEALTH DAN NON MBU)',
  },
  tanggal_efektif: '2026-03-31',
  metrik: [
    { kode: 'total_klaim_leader', judul: 'Total Klaim Leader', nilai: 4, bentuk: 'cacah' },
    { kode: 'rasio_leader', judul: 'Persentase Leader', nilai: 25, bentuk: 'persen' },
    { kode: 'skor_leader', judul: 'Nilai Leader', nilai: 0, bentuk: 'nilai' },
    { kode: 'rasio_member', judul: 'Persentase Member', nilai: null, bentuk: 'persen' },
    { kode: 'total_nilai', judul: 'Total Nilai', nilai: 40, bentuk: 'desimal' },
  ],
  achievement: 'TIDAK TERCAPAI TARGET',
  penyaring: { kelompok: 'NONMBU', dari: '2026-03-01', sampai: '2026-03-31' },
  portal: 'ASM',
}

const RINCIAN_ADMIN = {
  no_klaim: 'CONTOH-ADM-0001',
  no_polis: 'CONTOH-POL-0001',
  business: 'CONTOH ANEKA',
  tgl_regist_klaim: '2026-03-02',
  tgl_terima_dokumen: '2026-03-01',
  flag: 'LEADER',
  aging_regist_klaim: 1,
  tgl_terima_dokumen_pa: '',
  tgl_terima_lod: '',
  tgl_pembayaran: '',
  aging_pembayaran_klaim: null,
  status_sla_regist: '',
  status_sla_pembayaran: '',
}

/** Peladen tiruan yang menjawab bentuk layar beserta kedua permintaan tab KPI Admin. */
function stubAdminFetch(options?: { kartu?: unknown; rincian?: unknown[] }) {
  const kartu = options?.kartu ?? KARTU_SKOR
  const rincian = options?.rincian ?? [RINCIAN_ADMIN]

  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)

    if (url.startsWith(ADMIN_EXPORT_PATH)) {
      return new Response('No Klaim\nCONTOH-ADM-0001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="rincian-kpi-admin.csv"',
        },
      })
    }

    if (url.startsWith(SCORECARD_PATH)) return jsonResponse(200, kartu)

    // PALING BAWAH: `/admin` adalah awalan kedua jalur di atasnya, sehingga memeriksanya
    // lebih dulu akan menelan permintaan kartu skor dan ekspor.
    if (url.startsWith(ADMIN_DETAIL_PATH)) {
      return jsonResponse(200, {
        baris: rincian,
        paginasi: { halaman: 1, ukuran: 50, total: rincian.length, total_halaman: 1 },
        penyaring: { kelompok: 'NONMBU', dari: '2026-03-01', sampai: '2026-03-31' },
        portal: 'ASM',
      })
    }

    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak dikenali' })
  })
}

/** bukaAdmin menggambar layar lalu berpindah ke tab KPI Admin. */
async function bukaAdmin(user: ReturnType<typeof userEvent.setup>) {
  await renderLoaded()
  await user.click(screen.getByRole('tab', { name: /KPI Admin/ }))
}

/** cariAdmin mengisi ketiga isian wajib tab KPI Admin lalu menekan Cari. */
async function cariAdmin(user: ReturnType<typeof userEvent.setup>) {
  await user.selectOptions(screen.getByLabelText('Pilih Data KPI'), 'NONMBU')
  await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

describe('tab KPI Admin', () => {
  // Berpindah tab menukar SELURUH isi layar, bukan menambah bagian.
  //
  // Kedua tab punya isian bernama "Periode — Dari", dan bila keduanya tergambar bersamaan
  // pengisian akan mengenai yang salah tanpa ada yang menyadarinya.
  it('menukar isi layar saat tab berpindah', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)

    expect(screen.getByLabelText('Pilih Data KPI')).toBeInTheDocument()
    expect(screen.queryByLabelText('Pilih Tipe Report')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Pilih Adjuster')).not.toBeInTheDocument()
  })

  it('belum meminta apa pun sebelum Cari ditekan', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)

    expect(lastCallStartingWith(SCORECARD_PATH)).toBeUndefined()
    expect(
      screen.getByText(/Periode tanggal masih kosong/),
    ).toBeInTheDocument()
  })

  it('mengirim kelompok dan periode ke kartu skor dan rincian', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)
    await cariAdmin(user)

    await screen.findByText('Rincian Klaim')

    const kartu = lastCallStartingWith(SCORECARD_PATH)
    expect(kartu?.url).toContain('kelompok=NONMBU')
    expect(kartu?.url).toContain('dari=2026-03-01')
    expect(kartu?.url).toContain('sampai=2026-03-31')
  })

  // Identitas koordinator dan NIK di-hardcode di dalam kueri Pega, dan itu DIPERTAHANKAN
  // apa adanya (`P-5`). Yang diuji di sini: keduanya benar-benar sampai ke layar, bukan
  // diam-diam hilang di jalan.
  it('menggambar identitas kartu skor beserta kesimpulannya', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)
    await cariAdmin(user)

    expect(await screen.findByText(/Contoh Koordinator Admin/)).toBeInTheDocument()
    expect(screen.getByText(/NIK 90000001/)).toBeInTheDocument()
    expect(screen.getByText('TIDAK TERCAPAI TARGET')).toBeInTheDocument()
  })

  // Metrik yang tidak dapat dihitung digambar sebagai tanda hubung, BUKAN sebagai 0%.
  //
  // Periode tanpa satu pun klaim membuat pembaginya nol. Menggambarnya 0% menyatakan
  // kinerjanya nol, padahal yang benar adalah tidak ada yang dapat dinilai.
  it('membedakan metrik yang tidak dapat dihitung dari nilai nol', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)
    await cariAdmin(user)

    const kosong = await screen.findByText('Persentase Member')
    expect(kosong.parentElement).toHaveTextContent('—')

    // Pembanding: metrik yang memang bernilai nol tetap digambar sebagai angka.
    expect(screen.getByText('Nilai Leader').parentElement).toHaveTextContent('0')
  })

  // Kolom rincian BERBEDA antar kelompok, dan dua di antaranya berjudul SAMA
  // ("Tgl Terima Dokumen") meski kolom basis datanya berbeda. Karena itu penyaring
  // `hanya_kelompok` tidak boleh dilewati: bila dilewati, kedua kolom bernama sama akan
  // tergambar berdampingan dan tidak ada yang dapat membedakannya.
  it('menggambar hanya kolom milik kelompok yang dipilih', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)
    await cariAdmin(user)

    const tabel = await screen.findByRole('table', {
      name: /Rincian klaim yang ditangani tim admin/,
    })
    const kepala = within(tabel).getAllByRole('columnheader').map((h) => h.textContent)

    expect(kepala).toContain('BUSINESS')
    expect(kepala).toContain('Flag')
    expect(kepala).not.toContain('Status SLA Regist Klaim')
    expect(kepala).not.toContain('Tgl Terima LOD')

    // "Tgl Terima Dokumen" hanya boleh muncul SEKALI, bukan dua kali.
    expect(kepala.filter((teks) => teks === 'Tgl Terima Dokumen')).toHaveLength(1)
  })

  // Panel selisih dihapus dari tab ini pula — lihat catatan pada describe di atas.
  it('tidak menggambar panel selisih terencana', async () => {
    stubAdminFetch()
    const user = userEvent.setup()
    await bukaAdmin(user)

    expect(screen.queryByText(/NULLIF/)).not.toBeInTheDocument()
    expect(
      screen.queryByText(/Selisih terencana terhadap layar Pega/),
    ).not.toBeInTheDocument()
  })
})

/* ══════════════════════════ Tab KPI PIC Teknik ══════════════════════════ */

const PIC_PATH = `${PATH}/pic-teknik`
const PIC_EXPORT_PATH = `${PATH}/pic-teknik/ekspor`

/**
 * Kartu skor contoh — angkanya dipilih supaya TEMUAN utamanya terlihat.
 *
 * CONTOHPICSATU 90% tepat waktu bernilai 1; CONTOHPICDUA 20% bernilai 5. Tangga MENURUN
 * membuat yang rajin bernilai rendah, dan itu perilaku Pega yang direplikasi.
 */
const KARTU_PIC = {
  pic: 'CONTOHPICSATU',
  leader: true,
  nilai_berbobot: 13.5,
  baris: [
    {
      komponen: 'update_progress',
      judul: 'Update Status Progress (max terlambat 25%)',
      total: 10,
      tercapai: 9,
      persentase: 90,
      nilai: 1,
    },
    {
      komponen: 'analisa_klaim',
      judul: 'Analisa Klaim (max 10 hari)',
      total: 1,
      tercapai: 1,
      persentase: 100,
      nilai: 5,
    },
    {
      komponen: 'akseptasi_klaim',
      judul: 'Akseptasi Klaim ( 1 hari )',
      total: 1,
      tercapai: 1,
      persentase: 100,
      nilai: 5,
    },
    // Komponen yang TIDAK dapat dinilai — kosong, bukan nol.
    {
      komponen: 'sla_klaim',
      judul: 'SLA Klaim',
      total: 0,
      tercapai: 0,
      persentase: null,
      nilai: null,
    },
  ],
}

const REKAP_PIC = {
  pic: 'Leader',
  leader: true,
  nilai_berbobot: null,
  baris: [
    {
      komponen: 'update_progress',
      judul: 'Update Status Progress (max terlambat 25%)',
      total: 20,
      tercapai: 11,
      persentase: 55,
      nilai: 3,
    },
  ],
}

/** Peladen tiruan yang menjawab bentuk layar dan kartu skor tab KPI PIC Teknik. */
function stubPICFetch(options?: { kartu?: unknown[] }) {
  const kartu = options?.kartu ?? [KARTU_PIC]

  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)

    if (url.startsWith(PIC_EXPORT_PATH)) {
      return new Response('PIC\nCONTOHPICSATU\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="kpi-pic-teknik.csv"',
        },
      })
    }

    if (url.startsWith(PIC_PATH)) {
      return jsonResponse(200, {
        kartu_skor: kartu,
        rekapitulasi: REKAP_PIC,
        penyaring: { lini_bisnis: 'NONMBU', dari: '2026-03-01', sampai: '2026-03-31' },
        portal: 'ASM',
      })
    }

    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak dikenali' })
  })
}

/** bukaPIC menggambar layar lalu berpindah ke tab KPI PIC Teknik. */
async function bukaPIC(user: ReturnType<typeof userEvent.setup>) {
  await renderLoaded()
  await user.click(screen.getByRole('tab', { name: /KPI PIC Teknik/ }))
}

/** cariPIC mengisi ketiga isian wajib lalu menekan Cari. */
async function cariPIC(user: ReturnType<typeof userEvent.setup>) {
  await user.selectOptions(screen.getByLabelText('Lini Bisnis'), 'NONMBU')
  await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

describe('tab KPI PIC Teknik', () => {
  it('menukar isi layar saat tab berpindah', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)

    expect(screen.getByLabelText('Lini Bisnis')).toBeInTheDocument()
    expect(screen.queryByLabelText('Pilih Tipe Report')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Pilih Data KPI')).not.toBeInTheDocument()
  })

  it('belum meminta apa pun sebelum Cari ditekan', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)

    expect(lastCallStartingWith(PIC_PATH)).toBeUndefined()
    expect(screen.getByText(/Periode tanggal masih kosong/)).toBeInTheDocument()
  })

  it('mengirim lini bisnis dan periode', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    await screen.findByText('CONTOHPICSATU')

    const call = lastCallStartingWith(PIC_PATH)
    expect(call?.url).toContain('lini_bisnis=NONMBU')
    expect(call?.url).toContain('dari=2026-03-01')
    expect(call?.url).toContain('sampai=2026-03-31')
  })

  // Satu kartu per petugas, ditutup kartu rekapitulasi.
  it('menggambar satu kartu per petugas beserta rekapitulasinya', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    expect(await screen.findByText('CONTOHPICSATU')).toBeInTheDocument()
    expect(screen.getByText('Rekapitulasi seluruh PIC')).toBeInTheDocument()
    expect(screen.getByText('Leader tim')).toBeInTheDocument()
  })

  // INILAH uji yang paling penting di tab ini.
  //
  // Baris bertangga menurun WAJIB diberi penanda. Tanpa penanda, pembaca yang melihat
  // "90% → nilai 1" akan melaporkannya sebagai kerusakan — padahal itu perilaku Pega yang
  // direplikasi, dan menelusurinya kembali jauh lebih mahal daripada membacanya.
  it('menandai baris yang tangga nilainya menurun', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    // Dicari lewat isi SEL, bukan lewat teks biasa: penanda "↓" memecah teks sel menjadi
    // dua simpul, sehingga pencocok teks bawaan tidak menemukannya sebagai satu kesatuan.
    const kartu = (await screen.findByText('CONTOHPICSATU')).closest('section')
    expect(kartu).not.toBeNull()

    const sel = (awalan: string) =>
      within(kartu as HTMLElement).getByText(
        (_, element) =>
          element?.tagName === 'TD' &&
          element.textContent?.startsWith(awalan) === true,
      )

    expect(sel('Update Status Progress').textContent).toContain('↓')

    // Pembanding: komponen bertangga MENAIK tidak diberi penanda.
    expect(sel('Analisa Klaim').textContent).not.toContain('↓')
  })

  // Nilai yang tidak dapat dihitung digambar sebagai tanda hubung, BUKAN sebagai 0.
  it('membedakan nilai yang tidak dapat dihitung dari nol', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    const sla = (await screen.findByText('SLA Klaim')).closest('tr')
    expect(sla).not.toBeNull()
    expect(within(sla as HTMLElement).getAllByText('—')).toHaveLength(2)
  })

  it('menggambar nilai berbobot beserta skalanya', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    expect(await screen.findByText(/Nilai berbobot 13.5 dari 15/)).toBeInTheDocument()
  })

  // Panel selisih dihapus dari tab ini pula, dan di sinilah penghapusannya paling terasa:
  // tab ini yang punya tiga cacat direplikasi, termasuk nilai yang berlawanan arah.
  //
  // Keterangannya tidak hilang — ia tetap dikirim pada `/tab` dan tetap tertulis di
  // dokumen. Yang hilang hanyalah tampilannya, atas keputusan Work Owner (2026-09-25).
  it('tidak menggambar panel selisih terencana', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)

    expect(screen.queryByText(/Baca dulu/)).not.toBeInTheDocument()
    expect(screen.queryByText(/NILAI BERLAWANAN ARAH/)).not.toBeInTheDocument()
    expect(
      screen.queryByText(/BAMBANGSETIADJIGUNAWAN, DANIELLISWANDI/),
    ).not.toBeInTheDocument()
  })

  // Penanda "↓" pada baris bertangga menurun TETAP ADA meski panelnya hilang.
  //
  // Ia satu-satunya keterangan yang tersisa di layar untuk nilai yang berlawanan arah, dan
  // ia melekat pada barisnya sendiri — bukan panel terpisah.
  it('mempertahankan penanda tangga menurun setelah panel dihapus', async () => {
    stubPICFetch()
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    const kartu = (await screen.findByText('CONTOHPICSATU')).closest('section')
    const progres = within(kartu as HTMLElement).getByText(
      (_, element) =>
        element?.tagName === 'TD' &&
        element.textContent?.startsWith('Update Status Progress') === true,
    )
    expect(progres.textContent).toContain('↓')
  })

  it('memberitahu bila lini bisnis tidak punya petugas', async () => {
    stubPICFetch({ kartu: [] })
    const user = userEvent.setup()
    await bukaPIC(user)
    await cariPIC(user)

    expect(
      await screen.findByText(/Tidak ada petugas terdaftar pada lini bisnis ini/),
    ).toBeInTheDocument()
  })
})
