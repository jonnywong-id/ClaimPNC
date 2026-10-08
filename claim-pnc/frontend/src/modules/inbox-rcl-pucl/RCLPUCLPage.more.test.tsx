import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { RCLPUCLPage } from './RCLPUCLPage'
import { SendtoRCLPUCLPage } from './SendtoRCLPUCLPage'
import type { MetadataResponse, Tab, WorkItem } from './types'

/**
 * Uji tambahan Inbox RCL/PUCL dan layar kerjanya, dirender langsung: tab terhalang,
 * paginasi, galat, ekspor, dan kunci klaim yang hilang. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-rcl-pucl'

const COLUMNS = [
  { kunci: 'no_case', judul: 'Nomor Case' },
  { kunci: 'no_polis', judul: 'No Polis' },
] as Tab['kolom']

const TAB_GRID: Tab = {
  kode: '2',
  nama: 'Kelengkapan Dokumen',
  keterangan: 'Klaim yang suratnya sudah dicetak.',
  kolom: COLUMNS,
  punya_laporan_rentang_tanggal: false,
  terhalang: false,
}

const TAB_BLOCKED: Tab = {
  kode: '9',
  nama: 'Tab Terhalang',
  keterangan: 'Antrean yang belum dapat diisi.',
  kolom: COLUMNS,
  punya_laporan_rentang_tanggal: false,
  terhalang: true,
  alasan_terhalang: 'Kolom penandanya belum dipastikan.',
  pemilik_penghalang: 'DBA',
}

const TAB_BLOCKED_NO_OWNER: Tab = { ...TAB_BLOCKED, kode: '8', nama: 'Tab Tanpa Pemilik', pemilik_penghalang: '' }

const METADATA: MetadataResponse = {
  tab: [TAB_GRID, TAB_BLOCKED, TAB_BLOCKED_NO_OWNER],
  tab_bawaan: '2',
  kolom_laporan: [],
  portal: 'ASM',
}

function item(noCase: string, noPolis: string): WorkItem {
  return { referensi: `REF ${noCase || 'X'}`, no_case: noCase, no_polis: noPolis } as unknown as WorkItem
}

type Answer = Response | Promise<Response> | 'aneh' | 'putus' | 'gantung' | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function fallback(url: string): Response {
  if (url === `${PATH}/tab`) return json(200, METADATA)
  if (url.startsWith(`${PATH}/ekspor`)) return new Response('a\n', { status: 200 })
  const halaman = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
  return json(200, {
    tab: TAB_GRID,
    baris: [item(`PNC-H${halaman}`, 'POLIS-TEKS'), item('', '')],
    paginasi: { halaman, ukuran: 50, total: 120, total_halaman: 3 },
    portal: 'ASM',
  })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom === 'gantung') return new Promise<Response>(() => {})
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
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

function show(entry = '/inbox-rcl-pucl') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[entry]}>
        <Routes>
          <Route path="/inbox-rcl-pucl" element={<RCLPUCLPage />} />
          <Route path="/inbox-rcl-pucl/klaim/:referensi" element={<SendtoRCLPUCLPage />} />
          <Route path="/inbox-rcl-pucl/klaim" element={<SendtoRCLPUCLPage />} />
        </Routes>
      </MemoryRouter>
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

const FALLBACK_TEXT = 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'

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

describe('antrean', () => {
  it('menggambar tab terhalang beserta pemilik penghalangnya tanpa meminta isinya', async () => {
    installFetch()
    show('/inbox-rcl-pucl?tab=9')

    expect(await screen.findByText('Tab Terhalang belum tersedia')).toBeInTheDocument()
    expect(screen.getByText('Kolom penandanya belum dipastikan.')).toBeInTheDocument()
    expect(screen.getByText('DBA')).toBeInTheDocument()
    const tab = screen.getByRole('tab', { name: /Tab Terhalang/ })
    expect(tab).toHaveAttribute('title', 'Kolom penandanya belum dipastikan.')
    expect(within(tab).getByText('belum tersedia')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Export To Excel' })).toBeDisabled()
    expect(urls.some((u) => u.includes('tab=9'))).toBe(false)
  })

  it('tidak menyebut pemilik bila server tidak mengirimnya', async () => {
    installFetch()
    show('/inbox-rcl-pucl?tab=8')

    expect(await screen.findByText('Tab Tanpa Pemilik belum tersedia')).toBeInTheDocument()
    expect(screen.queryByText('Menunggu:')).not.toBeInTheDocument()
  })

  it('memperlakukan nomor halaman yang rusak sebagai halaman pertama lalu berpindah halaman', async () => {
    installFetch()
    const user = userEvent.setup()
    show('/inbox-rcl-pucl?tab=2&halaman=abc')

    const table = await screen.findByRole('table')
    expect(await within(table).findByText('PNC-H1')).toBeInTheDocument()
    expect(urls).toContain(`${PATH}?tab=2`)
    // Nomor polis yang bukan tanggal tampil apa adanya; baris tanpa nomor case bertanda hubung.
    expect(within(table).getByText('POLIS-TEKS')).toBeInTheDocument()
    expect(within(table).getAllByText('—').length).toBeGreaterThanOrEqual(2)

    await user.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
    expect(await within(table).findByText('PNC-H2')).toBeInTheDocument()
    expect(urls).toContain(`${PATH}?tab=2&halaman=2`)
  })

  it.each([
    { name: 'galat API', answer: (): Answer => json(500, { kode: 'x', pesan: 'Basis data sibuk.' }), text: 'Basis data sibuk.' },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: FALLBACK_TEXT },
  ])('menampilkan galat antrean: $name', async ({ answer, text }) => {
    installFetch((url) => (url.startsWith(`${PATH}?`) ? answer() : undefined))
    show()

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('menutup layar bila keterangan gagal dimuat', async () => {
    installFetch((url) => (url === `${PATH}/tab` ? 'aneh' : undefined))
    show()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText(FALLBACK_TEXT)).toBeInTheDocument()
  })
})

describe('ekspor', () => {
  it('menyebut keadaan menyiapkan berkas', async () => {
    installFetch((url) => (url.startsWith(`${PATH}/ekspor`) ? 'gantung' : undefined))
    const user = userEvent.setup()
    show()
    await screen.findByText('PNC-H1')

    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByRole('button', { name: 'Menyiapkan berkas…' })).toBeDisabled()
  })

  it('memakai nama berkas bawaan bila server tidak menyebutnya', async () => {
    const names = stubDownload()
    installFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('PNC-H1')

    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => expect(names).toEqual(['inbox-rcl-pucl.csv']))
    expect(urls).toContain(`${PATH}/ekspor?tab=2`)
  })

  it.each([
    { name: 'pesan umum dari server', answer: (): Answer => json(500, { pesan: 'Laporan sedang dikunci.' }), text: 'Laporan sedang dikunci.' },
    { name: 'badan bukan JSON', answer: (): Answer => new Response('rusak', { status: 500 }), text: 'Berkas ekspor tidak dapat diambil.' },
  ])('menampilkan galat ekspor: $name', async ({ answer, text }) => {
    installFetch((url) => (url.startsWith(`${PATH}/ekspor`) ? answer() : undefined))
    const user = userEvent.setup()
    show()
    await screen.findByText('PNC-H1')

    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(text)
  })
})

describe('layar kerja', () => {
  it('menyebut galat sambungan dengan kalimat umum', async () => {
    installFetch((url) => (url.startsWith(`${PATH}/klaim/`) ? 'putus' : undefined))
    show('/inbox-rcl-pucl/klaim/REF%20PNC-1')

    expect(await screen.findByText('Isi klaim tidak dapat diambil')).toBeInTheDocument()
    expect(screen.getByText('Sambungan ke peladen gagal. Coba lagi beberapa saat lagi.')).toBeInTheDocument()
  })

  it('tidak meminta apa pun bila kunci klaim tidak ada di alamat', () => {
    installFetch()
    show('/inbox-rcl-pucl/klaim')

    // Tanpa kunci, permintaan tidak pernah dijalankan dan layar tetap menunggu.
    expect(screen.getByText('Memuat isi layar kerja…')).toBeInTheDocument()
    expect(urls.some((u) => u.startsWith(`${PATH}/klaim`))).toBe(false)
  })
})
