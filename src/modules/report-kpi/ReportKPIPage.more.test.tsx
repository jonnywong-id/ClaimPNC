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
    {
      kode: 'pic-teknik',
      judul: 'KPI PIC Teknik',
      grid: [
        {
          kode: 'kartu-skor-pic',
          judul: 'Data KPI PIC Teknik',
          kolom: [
            { kunci: 'pic', judul: 'PIC' },
            { kunci: 'kpi', judul: 'KATEGORI' },
            { kunci: 'total', judul: 'TOTAL DATA' },
            { kunci: 'tercapai', judul: 'JUMLAH TERCAPAI' },
            { kunci: 'persentase', judul: 'TERCAPAI (%)' },
            { kunci: 'nilai', judul: 'NILAI' },
          ],
          kolom_akhir: [],
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
          judul: 'Ringkasan Uji',
          kolom: [{ kunci: 'adjuster', judul: 'ADJUSTER' }],
          kolom_akhir: [{ kunci: 'kategori', judul: 'KATEGORI' }],
        },
        {
          kode: 'rincian',
          judul: 'Rincian Uji',
          kolom: [
            { kunci: 'adjuster', judul: 'ADJUSTER' },
            { kunci: 'no_case', judul: 'NO CASE' },
            { kunci: 'no_klaim', judul: 'NO KLAIM' },
            { kunci: 'status_survey', judul: 'STATUS SURVEY' },
            { kunci: 'lain', judul: 'LAIN' },
          ],
          kolom_akhir: [{ kunci: 'kategori', judul: 'KATEGORI' }],
        },
      ],
      terhalang: false,
    },
    {
      kode: 'admin',
      judul: 'KPI Admin',
      grid: [
        {
          kode: 'kartu-skor',
          judul: 'Data KPI',
          kolom: [
            { kunci: 'unit_kerja', judul: 'UNIT KERJA' },
            { kunci: 'nama_koordinator', judul: 'NAMA KOORDINATOR' },
            { kunci: 'business', judul: 'BUSINESS' },
            { kunci: 'nik', judul: 'NIK' },
            { kunci: 'tanggal_efektif', judul: 'TANGGAL EFEKTIF' },
          ],
          kolom_akhir: [{ kunci: 'note', judul: 'NOTE' }],
        },
        {
          kode: 'rincian-klaim',
          judul: 'Rincian Klaim',
          kolom: [
            { kunci: 'no_klaim', judul: 'No Klaim' },
            // Dibatasi ke NONMBU, bukan PA: sejak 2026-10-09 hanya blok itu yang
            // digambar, sehingga kolom ber-`hanya_kelompok: 'PA'` tidak akan pernah
            // muncul dan uji pembulatan di bawah kehilangan objeknya.
            { kunci: 'aging_pembayaran_klaim', judul: 'Aging Pembayaran klaim', hanya_kelompok: 'NONMBU' },
            { kunci: 'tgl_terima_lod', judul: 'Tgl Terima LOD', hanya_kelompok: 'NONMBU' },
          ],
          kolom_akhir: [],
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
  kelompok_admin: [
    { kode: 'NONMBU', judul: 'NON MBU' },
    { kode: 'PA', judul: 'PA', keterangan: 'Kelompok PA mengukur dua tahap.' },
  ],
  lini_bisnis: [{ kode: 'PA', judul: 'PA' }],
  data_kpi: [
    { kode: '1', judul: 'Export KPI Progress' },
    { kode: '2', judul: 'Export KPI SLA' },
    { kode: '3', judul: 'Export KPI Akseptasi' },
    { kode: '4', judul: 'Export KPI Analisis' },
  ],
  komponen_pic: [{ kode: 'analisa_klaim', judul: 'Analisa Klaim' }],
  pic_dikecualikan_sla: [],
  periode_kpi: [
    { kode: '202602', judul: '202602' },
    // Maret 2026 dipilih helper: rentangnya 2026-03-01..2026-03-31, sama dengan
    // tanggal yang diketik blok NON MBU — sehingga kedua blok dapat dibandingkan.
    { kode: '202603', judul: '202603' },
    { kode: '202604', judul: '202604' },
  ],
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

// varian memilih grid mana yang digambar — di Pega satu grid saja, ditentukan
// isian "Pilih Tipe Report". Bawaannya DATA SUMMARY.
async function searchAdjuster(user: ReturnType<typeof userEvent.setup>, varian = 'DATA SUMMARY') {
  await screen.findByRole('tab', { name: /KPI Adjuster/ })
  await user.selectOptions(screen.getByLabelText('Pilih Status Survey'), 'FINAL')
  await user.type(screen.getByLabelText('Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Sampai'), '2026-03-31')
  // Pega menuntut isian ini pula — tanpanya tombol Cari ditolak.
  await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), varian)
  await user.click(screen.getByRole('button', { name: 'Cari' }))
}

/**
 * searchAdmin mencari pada KEDUA blok tab KPI Admin.
 *
 * Sejak 2026-10-09 tiap blok punya bilah penyaringnya sendiri, dan isiannya berbeda:
 * NON MBU dua tanggal, PA satu dropdown bulan. Keduanya dicari supaya uji yang menyoroti
 * salah satu tetap menemukan isinya.
 *
 * Bulan yang dipilih — `202603` — rentangnya sama persis dengan tanggal yang diketik di
 * blok NON MBU, sehingga keduanya dapat dibandingkan apa adanya.
 */
/**
 * searchAdmin mengisi penyaring tab KPI Admin lalu menekan Cari.
 *
 * SATU blok sejak 2026-10-09: Work Owner memastikan Pega hanya menampilkan bilah
 * `Dari`/`Sampai`, dan bilah PA dicabut.
 */
async function searchAdmin(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('tab', { name: /KPI Admin/ }))

  const blok = within(screen.getByRole('region', { name: 'NON MBU' }))
  await user.type(blok.getByLabelText('Dari'), '2026-03-01')
  await user.type(blok.getByLabelText('Sampai'), '2026-03-31')
  await user.click(blok.getByRole('button', { name: 'Cari' }))
}

async function searchPIC(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('tab', { name: /KPI PIC Teknik/ }))
  await user.type(screen.getByLabelText('Dari'), '2026-03-01')
  await user.type(screen.getByLabelText('Sampai'), '2026-03-31')
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

    // Judul grid Ringkasan diperiksa lebih dulu, lalu layar dipindah ke DATA DETAIL —
    // hanya satu grid digambar sekali jalan, sama seperti Pega.
    expect(await screen.findByText('Ringkasan Uji')).toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), 'DATA DETAIL')
    await user.click(screen.getByRole('button', { name: 'Cari' }))
    const table = await screen.findByRole('table', { name: /Rincian KPI per kasus survei/ })
    const row = (await within(table).findByText('KASUS-1')).closest('tr')!
    // Tanda hubung HANYA untuk komponen yang tidak dinilai — di sini satu, yaitu `ekstra`
    // yang tidak dikirim peladen. Di situ tanda hubung bermakna: ia membedakan "belum
    // dinilai" dari "bernilai nol".
    expect(within(row).getAllByText('—')).toHaveLength(1)

    // Kolom yang memang TIDAK PUNYA SUMBER digambar KOSONG, bukan bertanda hubung —
    // seperti Pega. `lain` tidak dikenali penggambar sel, `no_klaim` dan `status_survey`
    // tidak ada di tabelnya.
    const kepalaRincian = within(table).getAllByRole('columnheader').map((h) => h.textContent)
    const selRincian = within(row).getAllByRole('cell').map((c) => c.textContent)
    for (const judul of ['NO KLAIM', 'STATUS SURVEY', 'LAIN']) {
      expect(selRincian[kepalaRincian.indexOf(judul)]).toBe(judul)
    }

    await user.click(screen.getAllByRole('button', { name: 'Halaman berikutnya' })[0]!)
    expect(await within(table).findByText('KASUS-2')).toBeInTheDocument()
    // Kolom TANGGAL DICABUT: grid Detail Pega tidak memuatnya.
    expect(within(table).queryByText('2026-03-04')).not.toBeInTheDocument()
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
    // Grid Rincian TIDAK ikut digambar — ia muncul hanya pada DATA DETAIL.
    expect(screen.queryByText('Detail KPI Adjuster')).not.toBeInTheDocument()
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

    // Galat grid Rincian diperiksa setelah layar dipindah ke DATA DETAIL.
    await user.selectOptions(screen.getByLabelText('Pilih Tipe Report'), 'DATA DETAIL')
    await user.click(screen.getByRole('button', { name: 'Cari' }))
    expect(await screen.findByText('Rincian tidak dapat diambil')).toBeInTheDocument()
  })

  it('mengunduh rincian dan menampilkan galat ekspor', async () => {
    const names = stubDownload()
    let fail = false
    installFetch((url) => (url.includes('/ekspor') && fail ? json(500, { kode: 'x', pesan: 'Ekspor ditolak.' }) : undefined))
    const user = userEvent.setup()
    show()
    await searchAdjuster(user, 'DATA DETAIL')
    await screen.findByText('KASUS-1')

    // SATU tombol; yang diunduh ditentukan Pilih Tipe Report yang sedang terpilih.
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    await waitFor(() => expect(names).toHaveLength(1))
    expect(urls.some((u) => u.includes('/adjuster/ekspor') && u.includes('grid=rincian'))).toBe(true)

    fail = true
    await user.click(screen.getByRole('button', { name: 'Export Data' }))
    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
  })
})

describe('tab KPI Admin', () => {
  it('menggambar kartu tercapai dan membulatkan sel angka, tanpa keterangan kelompok', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    // Keterangan kelompok TIDAK digambar sejak 2026-10-09: bagian Admin pada section Pega
    // hanya memuat tiga label isian, tanpa judul blok maupun kalimat penjelas.
    expect(screen.queryByText('Kelompok PA mengukur dua tahap.')).not.toBeInTheDocument()

    const blok = within(await screen.findByRole('region', { name: 'NON MBU' }))

    // Kesimpulannya kini SEL kolom NOTE, bukan lencana berwarna.
    expect(await blok.findAllByText('TERCAPAI TARGET')).not.toHaveLength(0)

    const tabel = within(blok.getByRole('table'))
    const kepala = tabel.getAllByRole('columnheader').map((h) => h.textContent)
    const sel = tabel.getAllByRole('cell').map((c) => c.textContent)
    expect(sel[kepala.indexOf('Metrik Cacah')]).toMatch(/7$/)

    // Catatan perbedaan nama koordinator DICABUT — lihat `D-13` di ReportKPIAdmin.tsx.
    expect(screen.queryByText(/teks kueri Pega menyebut nama koordinator/)).not.toBeInTheDocument()

    // Grid rincian tidak digambar; rinciannya hanya lewat Export Detail Data.
    expect(screen.queryByText('Rincian Klaim')).not.toBeInTheDocument()
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

    const blok = within(await screen.findByRole('region', { name: 'NON MBU' }))
    expect(await blok.findAllByText('KATEGORI TANPA HASIL')).not.toHaveLength(0)
    expect(blok.queryByText(/TERCAPAI/)).not.toBeInTheDocument()
  })

  it('menampilkan keadaan menghitung kartu skor', async () => {
    installFetch((url) => (url.startsWith(`${PATH}/admin/kartu-skor`) ? 'gantung' : undefined))
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    // Keadaan memuat kini digambar DataTable, bukan kartu skor tersendiri.
    expect(
      within(screen.getByRole('region', { name: 'NON MBU' })).getByText(/Memuat/),
    ).toBeInTheDocument()
  })

  it('menampilkan galat kartu skor dan rincian', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/admin/kartu-skor`) ? API_FAIL() : url.startsWith(`${PATH}/admin?`) ? 'aneh' : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    const blok = within(await screen.findByRole('region', { name: 'NON MBU' }))
    expect(blok.getByText('Data KPI tidak dapat diambil')).toBeInTheDocument()
    expect(blok.getByText('Basis data sibuk.')).toBeInTheDocument()
  })

  it('menandai isian yang ditolak server', async () => {
    installFetch((url) =>
      url.startsWith(`${PATH}/admin`)
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'Permintaan belum benar.',
            detail: [{ field: 'dari', pesan: 'Tanggal Dari wajib diisi.' }],
          })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchAdmin(user)

    // Ditandai di KEDUA blok, masing-masing pada isiannya sendiri.
    //
    // Keduanya memanggil peladen sendiri-sendiri, jadi keduanya menerima penolakan yang
    // sama. Menuntut satu saja akan gagal dengan "found multiple" — dan itu justru
    // menyembunyikan bahwa tiap blok memang melaporkan penolakannya sendiri.
    const nonMBU = within(await screen.findByRole('region', { name: 'NON MBU' }))
    expect(await nonMBU.findByText('Tanggal Dari wajib diisi.')).toBeInTheDocument()

    const pa = within(screen.getByRole('region', { name: 'NON MBU' }))
    expect(await pa.findByText('Tanggal Dari wajib diisi.')).toBeInTheDocument()
  })

  it('mengunduh rincian lalu menyebut galat ekspor tanpa pesan', async () => {
    const names = stubDownload()
    let fail = false
    installFetch((url) => (url.includes('/admin/ekspor') && fail ? new Response('rusak', { status: 500 }) : undefined))
    const user = userEvent.setup()
    show()
    await searchAdmin(user)
    await screen.findAllByText('Data KPI')

    // Tombol milik bloknya sendiri, dan kelompoknya ikut terkirim.
    const blok = within(screen.getByRole('region', { name: 'NON MBU' }))
    await user.click(blok.getByRole('button', { name: 'Export Detail Data' }))
    await waitFor(() => expect(names).toHaveLength(1))
    expect(urls.some((u) => u.startsWith(`${PATH}/admin/ekspor?kelompok=NONMBU`))).toBe(true)

    fail = true
    await user.click(blok.getByRole('button', { name: 'Export Detail Data' }))
    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
  })
})

describe('tab KPI PIC Teknik', () => {
  it('menggambar kartu bukan leader dan rekapitulasi berbobot', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    expect(await screen.findAllByText('PICUJI')).not.toHaveLength(0)

    // Persentase digambar sebagai ANGKA saja — judul kolomnya sudah menyebut "(%)".
    expect(screen.getAllByText('33.33').length).toBeGreaterThan(0)
  })

  it('menampilkan keadaan menghitung penilaian', async () => {
    installFetch((url) => (url.startsWith(`${PATH}/pic-teknik`) ? 'gantung' : undefined))
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    // Keadaan memuat kini digambar DataTable.
    expect(screen.getByText(/Memuat/)).toBeInTheDocument()
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
            detail: [{ field: 'dari', pesan: 'Tanggal Dari wajib diisi.' }],
          })
        : undefined,
    )
    const user = userEvent.setup()
    show()
    await searchPIC(user)

    // Ditandai pada isian yang masih ada di layar — Lini Bisnis sudah dicabut.
    expect(await screen.findByText('Tanggal Dari wajib diisi.')).toBeInTheDocument()
  })

  it('mengunduh berkas lalu menampilkan galat ekspor', async () => {
    const names = stubDownload()
    let fail = false
    installFetch((url) => (url.includes('/pic-teknik/ekspor') && fail ? API_FAIL() : undefined))
    const user = userEvent.setup()
    show()
    await searchPIC(user)
    await screen.findAllByText('PICUJI')

    // DUA tombol berlabel sama, seperti Pega: yang pertama mengunduh penilaiannya, yang
    // kedua data klaim mentah menurut "Pilih Data KPI". Yang diuji di sini yang KEDUA.
    const tombol = () => screen.getAllByRole('button', { name: 'Export Data KPI' })
    expect(tombol()).toHaveLength(2)

    await user.click(tombol()[1]!)
    await waitFor(() => expect(names).toHaveLength(1))
    // Lininya kini TETAP NONMBU — isiannya dicabut mengikuti layar Pega.
    expect(urls.some((u) => u.startsWith(`${PATH}/pic-teknik/ekspor?lini_bisnis=NONMBU`))).toBe(true)

    fail = true
    await user.click(tombol()[1]!)
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

    // Dropdown Lini Bisnis sudah dicabut; yang masih harus benar adalah layarnya
    // TETAP terbuka walau keterangan lini dan komponen tidak dikirim server.
    expect(screen.getByLabelText('Dari')).toBeInTheDocument()
    expect(screen.queryByLabelText('Lini Bisnis')).not.toBeInTheDocument()
  })
})
