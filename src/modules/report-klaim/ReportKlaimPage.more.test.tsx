import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReportKlaimPage } from './ReportKlaimPage'

/**
 * Uji tambahan Report Klaim: penyaring status compliance dan rincian, keadaan katalog, dan
 * nada galat ekspor. Seluruh data KARANGAN (`D-69`).
 */

const SEMUA = {
  rentang_tanggal: true,
  lini_bisnis: true,
  status_compliance: true,
  bisnis: false,
  rincian: true,
  rincian_awal: false,
}

/** Kartu ber-Treaty yang di Pega mulai TERCENTANG — dua dari tiga. */
const SEMUA_TERCENTANG = { ...SEMUA, rincian_awal: true }

const CATALOG = {
  judul: '',
  laporan: [
    {
      kode: 'compliance',
      judul: 'REPORT COMPLIANCE',
      tombol: [{ kode: '', label: 'Export Compliance' }],
      penyaring: SEMUA,
      tersedia: true,
      sumber: { activity: 'X' },
    },
    {
      kode: 'akseptasi',
      judul: 'REPORT AKSEPTASI',
      tombol: [{ kode: '', label: 'Export Data Akseptasi' }],
      penyaring: SEMUA_TERCENTANG,
      tersedia: true,
      sumber: { activity: 'Z' },
    },
    {
      kode: 'mati',
      judul: 'REPORT MATI',
      tombol: [],
      penyaring: SEMUA,
      tersedia: false,
      alasan: 'Sumbernya belum ada.',
      sumber: { activity: 'Y' },
    },
  ],
  lini_bisnis: [{ nilai: '002', nama: 'Personal Accident' }],

  // Ketiganya saja, mengikuti ComplianceStatusOptions di peladen. Nilai "3"
  // ("Lain-Lain") ADA di data tetapi sengaja TIDAK ditawarkan penyaring ini, sehingga
  // fixture yang memuatnya akan menguji alamat yang tidak mungkin dihasilkan layar.
  status_compliance: [
    { nilai: '0', nama: 'Fraud / ditolak' },
    { nilai: '1', nama: 'Valid / Bayar' },
    { nilai: '2', nama: 'Post Audit / Bayar' },
  ],
}

type Answer = Response | Promise<Response> | 'putus' | 'aneh' | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    // Jawaban yang melempar nilai bukan Error saat headernya dibaca.
    if (custom === 'aneh') return Promise.resolve({ ok: true, get headers() { throw 'bukan galat' } } as unknown as Response)
    if (custom) return Promise.resolve(custom)
    if (url.includes('/ekspor')) {
      return Promise.resolve(
        new Response('a\n', {
          status: 200,
          headers: { 'Content-Disposition': 'attachment; filename="lap.csv"' },
        }),
      )
    }
    return Promise.resolve(json(200, CATALOG))
  })
}


/**
 * Kotak Treaty ke-n.
 *
 * Ditulis sebagai fungsi, bukan indeks langsung, karena TypeScript mode ketat
 * menganggap hasil indeks bisa undefined — dan melemparkan galat bernama di sini jauh
 * lebih mudah dibaca daripada tanda seru yang menyembunyikannya.
 */
function kotakTreaty(n: number): HTMLElement {
  const semua = screen.getAllByRole('checkbox', { name: 'Treaty' })
  const kotak = semua[n]
  if (kotak === undefined) {
    throw new Error(`kotak Treaty ke-${n} tidak ada; yang ada ${semua.length}`)
  }
  return kotak
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ReportKlaimPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSelectedPortal.getState().clear()
})

describe('katalog', () => {
  it('memakai judul bawaan, menyebut pemuatan, dan kartu mati tanpa penghalang', async () => {
    installFetch()
    show()

    expect(screen.getByText('Memuat daftar laporan…')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Report Claim' })).toBeInTheDocument()
    expect(await screen.findByText('Sumbernya belum ada.')).toBeInTheDocument()
    expect(screen.getByText('Belum tersedia')).toBeInTheDocument()
  })

  it('menampilkan galat katalog', async () => {
    installFetch((url) => (url.includes('/ekspor') ? undefined : json(500, { kode: 'x', pesan: 'Katalog rusak.' })))
    show()

    expect(await screen.findByText('Daftar laporan tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Katalog rusak.')).toBeInTheDocument()
  })
})

describe('ekspor', () => {
  // Status Compliance kini TIGA RADIO (pxRadioButtons di harness), dan kotak centangnya
  // berlabel "Treaty" di baris panelnya sendiri — bukan kotak centang bersama di bilah atas.
  it('mengirim status compliance dan rincian bila diisi', async () => {
    installFetch()
    show()

    const tombol = await screen.findByRole('button', { name: 'Export Compliance' })
    await userEvent.click(screen.getByLabelText('Valid / Bayar'))
    // Kotak Treaty milik kartu Compliance — yang PERTAMA. Kartu Akseptasi punya kotak
    // sendiri, dan sejak keadaannya dipisah per panel keduanya tidak lagi saling ikut.
    await userEvent.click(kotakTreaty(0))
    await userEvent.click(tombol)

    await waitFor(() =>
      expect(urls).toContain('/api/report-klaim/compliance/ekspor?status_compliance=1&rincian=1'),
    )
  })

  it.each([
    {
      name: 'ditolak server (4xx)',
      answer: (): Answer => json(422, { kode: 'validasi_gagal', pesan: 'Rentang terlalu panjang.' }),
      text: 'Rentang terlalu panjang.',
    },
    {
      name: 'galat server (5xx)',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }),
      text: 'Basis data sibuk.',
    },
    {
      name: 'jaringan putus',
      answer: (): Answer => 'putus',
      text: 'Tidak dapat menghubungi server Claim PNC.',
    },
  ])('menampilkan galat ekspor: $name', async ({ answer, text }) => {
    installFetch((url) => (url.includes('/ekspor') ? answer() : undefined))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Export Compliance' }))

    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('menampilkan pesan umum bila yang dilempar bukan Error', async () => {
    installFetch((url) => (url.includes('/ekspor') ? 'aneh' : undefined))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Export Compliance' }))

    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
    expect(screen.getByText('Terjadi kesalahan pada sistem.')).toBeInTheDocument()
  })
})

describe('kotak centang Treaty', () => {
  /*
   * Ketiga hal berikut dilaporkan Work Owner 2026-10-13 setelah membandingkan layar ini
   * dengan Pega produksi berdampingan. Ketiganya tidak terlihat dari harness XML.
   */

  it('mulai TERCENTANG bila peladen menyatakan begitu', async () => {
    installFetch()
    show()

    // Akseptasi mulai tercentang; Compliance mulai kosong. Keduanya di layar yang sama.
    await screen.findByRole('button', { name: 'Export Data Akseptasi' })
    const kotak = screen.getAllByRole('checkbox', { name: 'Treaty' })

    expect(kotak).toHaveLength(2)
    expect(kotak[0]).not.toBeChecked() // compliance — rincian_awal: false
    expect(kotak[1]).toBeChecked() //     akseptasi  — rincian_awal: true
  })

  // Keadaannya PER PANEL. Sebelumnya satu nilai bersama, sehingga mencentang di satu
  // kartu ikut mencentang di kartu lain — dan karena kotak ini memilih susunan kolom
  // berkas, akibatnya berkas kartu lain ikut berubah tanpa disentuh.
  it('mencentang satu panel tidak menyentuh panel lain', async () => {
    installFetch()
    show()

    await screen.findByRole('button', { name: 'Export Data Akseptasi' })
    const kotak = screen.getAllByRole('checkbox', { name: 'Treaty' })

    await userEvent.click(kotakTreaty(0))

    expect(kotak[0]).toBeChecked()
    expect(kotak[1]).toBeChecked() // tetap seperti semula, tidak ikut berubah
  })

  it('mengirim keadaan milik panel yang tombolnya ditekan', async () => {
    installFetch()
    show()

    // Akseptasi tercentang sejak awal, TANPA pengguna menyentuhnya.
    await userEvent.click(await screen.findByRole('button', { name: 'Export Data Akseptasi' }))

    await waitFor(() => expect(urls.some((u) => u.includes('/akseptasi/ekspor'))).toBe(true))
    expect(urls.find((u) => u.includes('/akseptasi/ekspor'))).toContain('rincian=1')
  })
})

describe('Refresh List', () => {
  // Tombol ini ada di Pega dan selama ini terlewat. Yang dimuat ulang DAFTAR panelnya —
  // ketersediaan sebuah panel dapat berubah tanpa layar di-reload.
  it('memuat ulang daftar panelnya', async () => {
    installFetch()
    show()

    await screen.findByRole('button', { name: 'Export Compliance' })
    const sebelum = urls.filter((u) => !u.includes('/ekspor')).length

    await userEvent.click(screen.getByRole('button', { name: 'Refresh List' }))

    await waitFor(() =>
      expect(urls.filter((u) => !u.includes('/ekspor')).length).toBeGreaterThan(sebelum),
    )
  })
})
