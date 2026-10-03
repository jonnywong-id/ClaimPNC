import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReportKPIPage } from './ReportKPIPage'
import type { MetadataResponse } from './types'

/**
 * Uji tambahan Report KPI dirender langsung: galat per bagian, ekspor ketiga tab, paginasi,
 * tab terhalang, dan cabang isian kosong. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/report-kpi'

const METADATA = {
  tab: [
    { kode: 'pic-teknik', judul: 'KPI PIC Teknik', grid: [], terhalang: false },
    {
      kode: 'adjuster',
      judul: 'KPI Adjuster',
      grid: [
        { kode: 'ringkasan', judul: 'Ringkasan Uji', kolom: [{ kunci: 'adjuster', judul: 'ADJUSTER' }] },
        {
          kode: 'rincian',
          judul: 'Rincian Uji',
          kolom: [
            { kunci: 'adjuster', judul: 'ADJUSTER' },
            { kunci: 'no_case', judul: 'NO CASE' },
            { kunci: 'tipe', judul: 'TIPE' },
            { kunci: 'tanggal', judul: 'TANGGAL' },
            { kunci: 'lain', judul: 'LAIN' },
          ],
        },
      ],
      terhalang: false,
    },
    {
      kode: 'admin',
      judul: 'KPI Admin',
      grid: [
        {
          kode: 'rincian-klaim',
          judul: 'Rincian Klaim',
          kolom: [
            { kunci: 'no_klaim', judul: 'No Klaim' },
            { kunci: 'aging_pembayaran_klaim', judul: 'Aging Pembayaran klaim', hanya_kelompok: 'PA' },
            { kunci: 'tgl_terima_lod', judul: 'Tgl Terima LOD', hanya_kelompok: 'PA' },
          ],
        },
      ],
      terhalang: false,
    },
    { kode: 'lain', judul: 'KPI Lain', grid: [], terhalang: true, alasan_terhalang: 'Menunggu kueri.' },
  ],
  tab_bawaan: 'adjuster',
  komponen: [
    { kode: 'nilai', judul: 'NILAI', kolom: 'NILAI' },
    { kode: 'ekstra', judul: 'EKSTRA', kolom: 'EKSTRA' },
  ],
  tipe_report: [{ kode: 'FINAL', judul: 'FINAL' }],
  selisih_terencana: [],
  kelompok_admin: [
    { kode: 'NONMBU', judul: 'NON MBU' },
    { kode: 'PA', judul: 'PA', keterangan: 'Kelompok PA mengukur dua tahap.' },
  ],
  selisih_terencana_admin: [],
  koordinator_di_kueri: '',
  lini_bisnis: [{ kode: 'PA', judul: 'PA' }],
  komponen_pic: [{ kode: 'analisa_klaim', judul: 'Analisa Klaim' }],
  selisih_terencana_pic: [],
  pic_dikecualikan_sla: [],
  tabel_sumber: '',
  tabel_tangga_nilai: '',
} as unknown as MetadataResponse

const FILTER = { tipe_report: 'FINAL', adjuster: '', dari: '2026-03-01', sampai: '2026-03-31' }

function detailRow(noCase: string, tanggal: string) {
  return { adjuster: 'PT CONTOH', no_case: noCase, tipe: 'FINAL', tanggal, nilai: { nilai: 2 } }
}

type Answer = Response | Promise<Response> | 'aneh' | 'gantung' | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function fallback(url: string): Response {
  if (url === `${PATH}/tab`) return json(200, METADATA)
  if (url.includes('/ekspor')) return new Response('a\n', { status: 200 })
  if (url.startsWith(`${PATH}/adjuster/pilihan`)) {
    return json(200, { adjuster: ['PT CONTOH', 'CV CONTOH'], penyaring: FILTER, portal: 'ASM' })
  }
  if (url.startsWith(`${PATH}/adjuster/ringkasan`)) {
    return json(200, { baris: [{ adjuster: 'PT CONTOH', tipe: 'FINAL', nilai: { nilai: 3 } }], penyaring: FILTER, portal: 'ASM' })
  }
  if (url.startsWith(`${PATH}/adjuster`)) {
    const halaman = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
    return json(200, {
      baris: [detailRow(`KASUS-${halaman}`, halaman === 1 ? '' : '2026-03-04')],
      paginasi: { halaman, ukuran: 50, total: 120, total_halaman: 3 },
      penyaring: FILTER,
      portal: 'ASM',
    })
  }
  if (url.startsWith(`${PATH}/admin/kartu-skor`)) {
    return json(200, {
      identitas: { kategori: 'KATEGORI UJI', nama_koordinator: 'Koordinator Uji', nik: '9', unit_kerja: 'UNIT' },
      tanggal_efektif: '2026-03-31',
      metrik: [{ kode: 'a', judul: 'Metrik Cacah', nilai: 7, bentuk: 'cacah' }],
      achievement: 'TERCAPAI TARGET',
      penyaring: {},
      portal: 'ASM',
    })
  }
  if (url.startsWith(`${PATH}/admin`)) {
    return json(200, {
      baris: [{ no_klaim: 'ADM-1', aging_pembayaran_klaim: 2.345, tgl_terima_lod: '' }],
      paginasi: { halaman: 1, ukuran: 50, total: 1, total_halaman: 1 },
      penyaring: {},
      portal: 'ASM',
    })
  }
  if (url.startsWith(`${PATH}/pic-teknik`)) {
    return json(200, {
      kartu_skor: [
        {
          pic: 'PICUJI',
          leader: false,
          nilai_berbobot: null,
          baris: [{ komponen: 'analisa_klaim', judul: 'Analisa Klaim', total: 1, tercapai: 1, persentase: 33.333, nilai: 4 }],
        },
      ],
      rekapitulasi: { pic: 'Leader', leader: true, nilai_berbobot: 7.456, baris: [] },
      penyaring: {},
      portal: 'ASM',
    })
  }
  return json(404, { kode: 'x', pesan: 'x' })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom === 'gantung') return new Promise<Response>(() => {})
    if (custom === 'aneh') {
      return Promise.resolve({
        get status(): number {
          throw 'bukan galat'
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    return Promise.resolve(fallback(url))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ReportKPIPage />
    </QueryClientProvider>,
  )
}

function stubDownload(): string[] {
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  const names: string[] = []
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
    names.push(this.download)
  })
  return names
}

const API_FAIL = (): Answer => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' })
// Pesan cadangan saat galat bukan Error.
const FALLBACK_TEXT = 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'

async function searchAdjuster(user: ReturnType<typeof userEvent.setup>) {
  await screen.findByRole('tab', { name: /KPI Adjuster/ })
  await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), 'FINAL')
  await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

async function searchAdmin(user: ReturnType<typeof userEvent.setup>, group = 'PA') {
  await user.click(await screen.findByRole('tab', { name: /KPI Admin/ }))
  await user.selectOptions(screen.getByLabelText('Pilih Data KPI'), group)
  await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

async function searchPIC(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('tab', { name: /KPI PIC Teknik/ }))
  await user.selectOptions(screen.getByLabelText('Lini Bisnis'), 'PA')
  await user.type(screen.getByLabelText('Periode — Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Periode — Sampai'), '2026-03-31')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSelectedPortal.getState().clear()
})

describe('kerangka layar', () => {
  it.each([
    { name: 'galat API', answer: API_FAIL, text: 'Basis data sibuk.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: FALLBACK_TEXT },
  ])('menampilkan galat keterangan layar: $name', async ({ answer, text }) => {
    installFetch((url) => (url === `${PATH}/tab` ? answer() : undefined))
    show()

    expect(await screen.findByText('Keterangan layar tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
    // Tanpa keterangan, bilah tab tidak digambar dan penyaring Adjuster tetap tampil.
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Pilih Tipe Report')).toBeInTheDocument()
  })

  it('tidak menyebut entitas bila portal belum dipilih', () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(screen.getByText(/tim admin registrasi\.$/)).toBeInTheDocument()
  })

  it('mematikan tab terhalang beserta alasannya', async () => {
    installFetch()
    show()

    const blocked = await screen.findByRole('tab', { name: /KPI Lain/ })
    expect(blocked).toBeDisabled()
    expect(blocked).toHaveAttribute('title', 'Menunggu kueri.')
    expect(within(blocked).getByText('belum tersedia')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /KPI Adjuster/ })).toHaveAttribute('aria-selected', 'true')
  })
})

describe('tab KPI Adjuster', () => {
  it('memuat daftar adjuster lalu mengirim adjuster yang dipilih', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await searchAdjuster(user)

    const select = screen.getByLabelText('Pilih Adjuster')
    await within(select).findByRole('option', { name: 'CV CONTOH' })
    await user.selectOptions(select, 'CV CONTOH')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    await waitFor(() =>
      expect(urls.some((u) => u.startsWith(`${PATH}/adjuster/ringkasan`) && u.includes('adjuster=CV+CONTOH'))).toBe(true),
    )
  })

  it('memakai judul grid dari server, mengisi sel kosong, dan berpindah halaman rincian', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await searchAdjuster(user)

    expect(await screen.findByText('Ringkasan Uji')).toBeInTheDocument()
    const table = await screen.findByRole('table', { name: /Rincian KPI per kasus survei/ })
    const row = (await within(table).findByText('KASUS-1')).closest('tr')!
    // Tanggal kosong, kolom tak dikenal, dan komponen yang tidak dikirim server bertanda hubung.
    expect(within(row).getAllByText('—')).toHaveLength(3)

    await user.click(screen.getAllByRole('button', { name: 'Halaman berikutnya' })[0]!)
    expect(await within(table).findByText('KASUS-2')).toBeInTheDocument()
    expect(within(table).getByText('2026-03-04')).toBeInTheDocument()
    expect(urls.some((u) => u.includes('&halaman=2'))).toBe(true)
  })

  it('memakai judul bawaan bila grid tidak dikirim server', async () => {
    installFetch((url) =>
      url === `${PATH}/tab`
        ? json(200, { ...METADATA, tab: [{ kode: 'adjuster', judul: 'KPI Adjuster', grid: [], terhalang: false }] })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdjuster(user)

    expect(await screen.findByText('Summary KPI Adjuster')).toBeInTheDocument()
    expect(screen.getByText('Detail KPI Adjuster')).toBeInTheDocument()
  })

  it('menampilkan galat ringkasan dan rincian', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/adjuster/ringkasan`) ? API_FAIL() : url.startsWith(`${PATH}/adjuster?`) ? 'aneh' : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdjuster(user)

    expect(await screen.findByText('Ringkasan tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()
    expect(await screen.findByText('Rincian tidak dapat diambil')).toBeInTheDocument()
  })

  it('mengunduh rincian dan menampilkan galat ekspor', async () => {
    const names = stubDownload()
    let fail = false
    installFetch((url) => (url.includes('/ekspor') && fail ? json(500, { kode: 'x', pesan: 'Ekspor ditolak.' }) : undefined))
    const user = userEvent.setup()
    show()
    await searchAdjuster(user)
    await screen.findByText('KASUS-1')

    await user.click(screen.getByRole('button', { name: 'Export Rincian' }))
    await waitFor(() => expect(names).toHaveLength(1))
    expect(urls.some((u) => u.includes('/adjuster/ekspor') && u.includes('grid=rincian'))).toBe(true)

    fail = true
    await user.click(screen.getByRole('button', { name: 'Export Ringkasan' }))
    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
  })
})

describe('tab KPI Admin', () => {
  it('menyebut keterangan kelompok, menggambar kartu tercapai, dan membulatkan sel angka', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    expect(screen.getByText('Kelompok PA mengukur dua tahap.')).toBeInTheDocument()
    expect(await screen.findByText('TERCAPAI TARGET')).toHaveClass('bg-blue-50')
    expect(screen.getByText('Metrik Cacah').parentElement).toHaveTextContent('7')
    // Koordinator di kueri kosong: catatan perbedaan nama tidak digambar.
    expect(screen.queryByText(/teks kueri Pega menyebut nama koordinator/)).not.toBeInTheDocument()

    const table = screen.getByRole('table', { name: /Rincian klaim yang ditangani tim admin/ })
    const row = (await within(table).findByText('ADM-1')).closest('tr')!
    expect(within(row).getByText('2.35')).toBeInTheDocument()
    expect(within(row).getByText('—')).toBeInTheDocument()
  })

  it('tidak menggambar kesimpulan bila server tidak mengirimnya', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/admin/kartu-skor`)
        ? json(200, {
            identitas: { kategori: 'KATEGORI TANPA HASIL', nama_koordinator: 'K', nik: '1', unit_kerja: 'U' },
            tanggal_efektif: '2026-03-31',
            metrik: [],
            achievement: '',
            penyaring: {},
            portal: 'ASM',
          })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    expect(await screen.findByText('KATEGORI TANPA HASIL')).toBeInTheDocument()
    expect(screen.queryByText(/TERCAPAI/)).not.toBeInTheDocument()
  })

  it('menampilkan keadaan menghitung kartu skor', async () => {
    installFetch((url) => (url.startsWith(`${PATH}/admin/kartu-skor`) ? 'gantung' : undefined))
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    expect(screen.getByText('Menghitung kartu skor…')).toBeInTheDocument()
  })

  it('menampilkan galat kartu skor dan rincian', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/admin/kartu-skor`) ? API_FAIL() : url.startsWith(`${PATH}/admin?`) ? 'aneh' : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    expect(await screen.findByText('Kartu skor tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()
    expect(await screen.findByText('Rincian tidak dapat diambil')).toBeInTheDocument()
  })

  it('menandai isian yang ditolak server', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/admin`)
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Permintaan belum benar.',
            detail: [{ field: 'kelompok', pesan: 'Kelompok wajib dipilih.' }],
          })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    expect(await screen.findByText('Kelompok wajib dipilih.')).toBeInTheDocument()
  })

  it('mengunduh rincian lalu menyebut galat ekspor tanpa pesan', async () => {
    const names = stubDownload()
    let fail = false
    installFetch((url) => (url.includes('/admin/ekspor') && fail ? new Response('rusak', { status: 500 }) : undefined))
    const user = userEvent.setup()
    show()
    await searchAdmin(user)
    await screen.findByText('ADM-1')

    await user.click(screen.getByRole('button', { name: 'Export Detail Data' }))
    await waitFor(() => expect(names).toHaveLength(1))
    expect(urls.some((u) => u.startsWith(`${PATH}/admin/ekspor?kelompok=PA`))).toBe(true)

    fail = true
    await user.click(screen.getByRole('button', { name: 'Export Detail Data' }))
    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
  })
})

describe('tab KPI PIC Teknik', () => {
  it('menggambar kartu bukan leader dan rekapitulasi berbobot', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    expect(await screen.findByText('PICUJI')).toBeInTheDocument()
    expect(screen.queryByText('Leader tim')).not.toBeInTheDocument()
    expect(screen.getByText('33.33%')).toBeInTheDocument()
    expect(screen.getByText('Nilai berbobot 7.46 dari 15')).toBeInTheDocument()
  })

  it('menampilkan keadaan menghitung penilaian', async () => {
    installFetch((url) => (url.startsWith(`${PATH}/pic-teknik`) ? 'gantung' : undefined))
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    expect(screen.getByText('Menghitung penilaian…')).toBeInTheDocument()
  })

  it.each([
    { name: 'galat API', answer: API_FAIL, text: 'Basis data sibuk.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: FALLBACK_TEXT },
  ])('menampilkan galat penilaian: $name', async ({ answer, text }) => {
    installFetch((url) => (url.startsWith(`${PATH}/pic-teknik`) ? answer() : undefined))
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    expect(await screen.findByText('Penilaian tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('menandai isian yang ditolak server', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/pic-teknik`)
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Permintaan belum benar.',
            detail: [{ field: 'lini_bisnis', pesan: 'Lini bisnis wajib dipilih.' }],
          })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    expect(await screen.findByText('Lini bisnis wajib dipilih.')).toBeInTheDocument()
  })

  it('mengunduh berkas lalu menampilkan galat ekspor', async () => {
    const names = stubDownload()
    let fail = false
    installFetch((url) => (url.includes('/pic-teknik/ekspor') && fail ? API_FAIL() : undefined))
    const user = userEvent.setup()
    show()
    await searchPIC(user)
    await screen.findByText('PICUJI')

    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    await waitFor(() => expect(names).toHaveLength(1))
    expect(urls.some((u) => u.startsWith(`${PATH}/pic-teknik/ekspor?lini_bisnis=PA`))).toBe(true)

    fail = true
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
  })

  it('tetap dapat dibuka tanpa daftar lini bisnis dan komponen', async () => {
    installFetch((url) =>
      url === `${PATH}/tab`
        ? json(200, { ...METADATA, lini_bisnis: undefined, komponen_pic: undefined })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await user.click(await screen.findByRole('tab', { name: /KPI PIC Teknik/ }))

    expect(within(screen.getByLabelText('Lini Bisnis')).getAllByRole('option')).toHaveLength(1)
  })
})
