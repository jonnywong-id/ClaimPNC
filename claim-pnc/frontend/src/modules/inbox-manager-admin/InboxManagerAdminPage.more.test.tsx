import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxManagerAdminPage } from './InboxManagerAdminPage'
import type { MetadataResponse, Tab, WorkItem } from './types'

/**
 * Uji tambahan Inbox Manager Admin dirender langsung: galat, paginasi, tombol Lihat Detail,
 * pengurutan, dan ekspor. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-manager-admin'

const TAB: Tab = {
  kode: '1',
  nama: 'Manajemen Admin - Non MBU',
  keterangan: 'Antrean registrasi klaim Non-MBU.',
  kolom: [
    { kunci: 'id', judul: 'ID' },
    { kunci: 'tanggal_pendaftaran', judul: 'Tanggal Pendaftaran' },
  ] as Tab['kolom'],
  unit_organisasi: 'AdminPNC',
  lini_bisnis: 'NONMBU',
}

const METADATA: MetadataResponse = {
  tab: [TAB],
  tab_bawaan: '1',
  semua_tab: [TAB],
  lini_bisnis_yang_diharapkan: ['NONMBU'],
  lini_bisnis_anda: 'NONMBU',
  selisih_terencana: [],
  portal: 'ASM',
}

function row(id: string, referensi: string): WorkItem {
  return { id, referensi, tanggal_pendaftaran: '2026-01-02' } as unknown as WorkItem
}

function list(rows: WorkItem[], halaman = 1, total = rows.length, totalHalaman = 1) {
  return {
    tab: TAB,
    baris: rows,
    paginasi: { halaman, ukuran: 50, total, total_halaman: totalHalaman },
    portal: 'ASM',
  }
}

type Answer = Response | Promise<Response> | 'putus' | 'kosong' | undefined

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
    // Galat yang dilempar tanpa pesan.
    if (custom === 'kosong') return Promise.reject(new Error(''))
    if (custom) return Promise.resolve(custom)
    if (url === `${PATH}/tab`) return Promise.resolve(json(200, METADATA))
    if (url.startsWith(`${PATH}/ekspor`)) return Promise.resolve(new Response('a\n', { status: 200 }))
    return Promise.resolve(json(200, list([row('PNCN.26.0002', 'REF-2'), row('PNC-1865', '')])))
  })
}

function ClaimTarget() {
  const { claimID } = useParams()
  return <p>Halaman klaim {claimID}</p>
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox']}>
        <Routes>
          <Route path="/inbox" element={<InboxManagerAdminPage />} />
          <Route path="/registrasi/klaim/:claimID" element={<ClaimTarget />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
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

describe('galat', () => {
  it.each([
    { name: 'galat API', answer: (): Answer => json(500, { kode: 'x', pesan: 'Keterangan rusak.' }), text: 'Keterangan rusak.' },
    { name: 'galat tanpa pesan', answer: (): Answer => 'kosong', text: 'Tidak dapat menghubungi server Claim PNC.' },
  ])('menutup layar bila keterangan gagal dimuat: $name', async ({ answer, text }) => {
    installFetch((url) => (url === `${PATH}/tab` ? answer() : undefined))
    show()

    expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('menampilkan galat antrean', async () => {
    installFetch((url) => (url.startsWith(`${PATH}?`) ? 'putus' : undefined))
    show()

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })

  it('tidak menyebut daftar antrean bila server tidak mengirimnya', async () => {
    installFetch((url) =>
      url === `${PATH}/tab` ? json(200, { ...METADATA, tab: [], tab_bawaan: '', semua_tab: [] }) : undefined,
    )
    show()

    expect(await screen.findByText('Tidak ada antrean yang menjadi hak lini bisnis Anda')).toBeInTheDocument()
    expect(screen.queryByText('Antrean yang ada di layar ini:')).not.toBeInTheDocument()
    expect(screen.getByText(/Bila Anda seharusnya termasuk salah satunya/)).toBeInTheDocument()
  })
})

describe('antrean', () => {
  it('membuka klaim PNCN di halaman klaim aplikasi ini, sama seperti My Inbox', async () => {
    // Tujuannya `/registrasi/klaim/:claimID`, sama dengan tautan nomor klaim pada My Inbox.
    // Sebelumnya ia `/view-claim/:referensi` — modul yang TIDAK DIPAKAI LAGI (Work Owner,
    // 2026-09-30), dan tidak satu pun inbox Pega yang membukanya.
    installFetch((url) =>
      url.startsWith(`${PATH}?`) ? json(200, list([row('PNCN.26.0002', 'REF-2')])) : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Lihat Detail' }))

    // NOMOR klaim yang dikirim, bukan `referensi`: `referensi` adalah pzInsKey, dan `D-22`
    // melarang kunci teknis Pega dipakai sebagai alamat.
    expect(await screen.findByText('Halaman klaim PNCN.26.0002')).toBeInTheDocument()
  })

  it('membuka klaim warisan Pega juga, bukan hanya PNCN', async () => {
    // Antrean ini DIDOMINASI klaim `PNC-xxxx`. Mematikannya — menyalin pembatasan My
    // Inbox — membuat tombolnya praktis tidak pernah dapat ditekan.
    //
    // Halaman tujuannya memang melayani klaim warisan: `klaim_ambil_per_nomor` membaca
    // `POOLDATA.T_CLAIM_PNC`, tabel klaim warisan. Dan di Pega, tombol layar ini membuka
    // formulir Register_Flow baris itu apa pun format nomornya.
    installFetch((url) =>
      url.startsWith(`${PATH}?`) ? json(200, list([row('PNC-2856', 'REF-9')])) : undefined,
    )
    show()

    const button = await screen.findByRole('button', { name: 'Lihat Detail' })
    expect(button).toBeEnabled()

    await userEvent.click(button)
    expect(await screen.findByText('Halaman klaim PNC-2856')).toBeInTheDocument()
  })

  it('mematikan tombol untuk klaim yang belum bernomor, beserta alasannya', async () => {
    // Nomor terbit di ujung tahap Input Register (`ADR-0009`), sehingga baris tanpa nomor
    // memang belum punya halaman — bukan baris yang rusak.
    installFetch((url) => (url.startsWith(`${PATH}?`) ? json(200, list([row('', 'REF-8')])) : undefined))
    show()

    const button = await screen.findByRole('button', { name: 'Lihat Detail' })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('title', expect.stringContaining('belum bernomor'))
  })

  it('memberi kolom tombol judul "Aksi"', async () => {
    // Ketetapan Work Owner 2026-10-03. Mengosongkannya sudah dicoba di layar lain dan
    // ditolak: kolom tanpa judul terbaca sebagai kolom yang TIDAK ADA.
    installFetch()
    show()

    expect(await screen.findByRole('columnheader', { name: 'Aksi' })).toBeInTheDocument()
  })

  it('mengurutkan menurut kolom dan berpindah halaman', async () => {
    installFetch((url) => {
      if (!(url === PATH || url.startsWith(`${PATH}?`))) return undefined
      const halaman = Number(new URL(url, 'https://x').searchParams.get('halaman') ?? '1')
      return json(200, list([row(`KLM-H${halaman}`, ''), row('KLM-A', '')], halaman, 120, 3))
    })
    show()
    const table = await screen.findByRole('table')

    const id = within(table).getByRole('columnheader', { name: /^ID/ })
    await userEvent.click(within(id).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('KLM-A')
    const date = within(table).getByRole('columnheader', { name: /^Tanggal Pendaftaran/ })
    await userEvent.click(within(date).getByRole('button'))
    expect(date).toHaveAttribute('aria-sort', 'ascending')

    await userEvent.click(screen.getByRole('button', { name: /halaman berikutnya|Berikutnya/i }))
    expect(await screen.findByText('KLM-H2')).toBeInTheDocument()
    expect(urls).toContain(`${PATH}?tab=1&halaman=2`)
  })
})

describe('ekspor', () => {
  it('mengunduh dengan nama berkas bawaan bila server tidak menyebutnya', async () => {
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
    const names: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      names.push(this.download)
    })
    installFetch()
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => expect(names).toEqual(['inbox-manager-admin.csv']))
    expect(urls).toContain(`${PATH}/ekspor?tab=1`)
  })

  it.each([
    { name: 'pesan dari server', answer: (): Answer => json(403, { pesan: 'Tab bukan hak Anda.' }), text: 'Tab bukan hak Anda.' },
    { name: 'badan bukan JSON', answer: (): Answer => new Response('rusak', { status: 500 }), text: 'Berkas ekspor tidak dapat diambil.' },
  ])('menampilkan galat ekspor: $name', async ({ answer, text }) => {
    installFetch((url) => (url.startsWith(`${PATH}/ekspor`) ? answer() : undefined))
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(text)
  })
})
