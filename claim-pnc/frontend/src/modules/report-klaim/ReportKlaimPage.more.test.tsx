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
}

const CATALOG = {
  judul: '',
  kelompok: [
    {
      kode: 'klaim',
      judul: 'Klaim',
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
          kode: 'mati',
          judul: 'REPORT MATI',
          tombol: [],
          penyaring: SEMUA,
          tersedia: false,
          alasan: 'Sumbernya belum ada.',
          sumber: { activity: 'Y' },
        },
      ],
    },
  ],
  lini_bisnis: [{ nilai: '002', nama: 'Personal Accident' }],
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
  it('mengirim status compliance dan rincian bila diisi', async () => {
    installFetch()
    show()

    const tombol = await screen.findByRole('button', { name: 'Export Compliance' })
    await userEvent.type(screen.getByLabelText('Status Compliance'), '3')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Tampilkan kolom rincian' }))
    await userEvent.click(tombol)

    await waitFor(() =>
      expect(urls).toContain('/api/report-klaim/compliance/ekspor?status_compliance=3&rincian=1'),
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
