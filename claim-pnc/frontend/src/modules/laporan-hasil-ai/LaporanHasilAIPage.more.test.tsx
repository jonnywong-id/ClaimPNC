import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, renderHook, screen, waitFor, within } from '@testing-library/react'
import type { ReactNode } from 'react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { LaporanHasilAIPage } from './LaporanHasilAIPage'
import { useExportLaporanHasilAI } from './api'

/**
 * Uji tambahan Laporan Hasil Data AI: galat muat, galat ekspor, pengurutan kolom tanggal,
 * dan judul tanpa entitas. Seluruh data KARANGAN (`D-69`).
 */

const ROUTE = '/api/laporan-hasil-ai'

function row(id: string, tanggalKomite: string | null, tanggalAI: string | null) {
  return {
    id,
    no_klaim: id,
    nama_object: '',
    komite_status: 'DITERIMA',
    kode_komite_status: '1',
    tanggal_komite: tanggalKomite,
    ai_status: 'DITERIMA',
    tanggal_ai: tanggalAI,
    note_ai_terima: '',
    note_ai_tolak: '',
    coverage_final: '',
    kategori_kronologi: '',
  }
}

const REPORT = {
  ringkasan: [],
  baris: [row('KLM-B', '2026-09-05', null), row('KLM-A', null, '2026-09-01')],
  paginasi: { halaman: 1, ukuran: 50, total: 2, total_halaman: 1 },
  filter: { dari: '2026-09-01', sampai: '2026-09-30' },
  portal: 'ASM',
}

type Answer = Response | 'aneh' | 'putus' | undefined

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
    if (custom === 'aneh') {
      return Promise.resolve({
        get status(): number {
          throw 'bukan galat'
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    return Promise.resolve(json(200, REPORT))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <LaporanHasilAIPage />
    </QueryClientProvider>,
  )
}

async function search(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Tgl Input Dari'), '2026-09-01')
  await user.type(screen.getByLabelText('Tgl Input Sampai'), '2026-09-30')
  await user.click(screen.getByRole('button', { name: 'Cari Data' }))
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('LaporanHasilAIPage', () => {
  it('tidak menyebut entitas bila portal belum dipilih', () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(screen.getByText(/disandingkan dengan keputusan komitenya\.$/)).toBeInTheDocument()
  })

  it.each([
    {
      name: 'galat API',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }),
      text: 'Basis data sibuk.',
    },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
    {
      name: 'galat bukan Error',
      answer: (): Answer => 'aneh',
      text: 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.',
    },
  ])('menampilkan galat pada kedua grid: $name', async ({ answer, text }) => {
    installFetch((url) => (url.startsWith(`${ROUTE}?`) ? answer() : undefined))
    const user = userEvent.setup()
    show()
    await search(user)

    expect(await screen.findByText('Ringkasan tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText('Rincian tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getAllByText(text)).toHaveLength(2)
  })

  it('menampilkan galat ekspor dari server', async () => {
    installFetch((url) =>
      url.startsWith(`${ROUTE}/ekspor`) ? json(500, { kode: 'x', pesan: 'Ekspor ditolak.' }) : undefined,
    )
    const user = userEvent.setup()
    show()
    await search(user)
    await screen.findByText('KLM-A')

    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
    expect(screen.getByText('Ekspor ditolak.')).toBeInTheDocument()
  })

  it('mengurutkan menurut kedua kolom tanggal, dengan tanggal kosong lebih dulu', async () => {
    installFetch()
    const user = userEvent.setup()
    show()
    await search(user)
    const table = await screen.findByRole('table', { name: /Penilaian AI beserta keputusan komitenya/ })
    await within(table).findByText('KLM-A')

    const komite = within(table).getByRole('columnheader', { name: /^Tanggal Komite/ })
    await user.click(within(komite).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('KLM-A')

    const ai = within(table).getByRole('columnheader', { name: /^Tanggal AI/ })
    await user.click(within(ai).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('KLM-B')
  })
})

describe('useExportLaporanHasilAI', () => {
  it('tidak mengirim parameter tanggal yang kosong', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    installFetch(() => new Response('a\n', { status: 200 }))
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => useExportLaporanHasilAI(), { wrapper })

    result.current.mutate({ dari: '', sampai: '' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(urls).toEqual([`${ROUTE}/ekspor?`])
    vi.restoreAllMocks()
  })
})
