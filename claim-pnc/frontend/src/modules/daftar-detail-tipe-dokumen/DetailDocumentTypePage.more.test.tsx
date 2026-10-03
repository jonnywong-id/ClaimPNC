import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DetailDocumentTypePage } from './DetailDocumentTypePage'

/** Uji tambahan Detail Tipe Dokumen: cabang galat muat/simpan, pengurutan, dan galat pilihan. */

const ROUTE = '/api/master/detail-tipe-dokumen'

const REFERENCES = {
  portal: 'ASM',
  tipe_dokumen: [{ id: '10001', nama: 'Dokumen Registrasi' }],
  penyebab_kerugian: [],
  objek_dokumen: [],
  bisnis: [],
  tidak_tersedia: [],
}

function row(id: string, detail: string, extra: Record<string, unknown> = {}) {
  return {
    id,
    id_tipe_dokumen: '10001',
    nama_tipe_dokumen: 'Dokumen Registrasi',
    detail_dokumen: detail,
    status_tertanggung: '',
    id_penyebab_kerugian: '',
    keterangan_penyebab_kerugian: '',
    id_objek_dokumen: '',
    keterangan_objek_dokumen: '',
    resiko: '0',
    bisnis: [],
    ...extra,
  }
}

const LIST = {
  portal: 'ASM',
  total: 2,
  detail_tipe_dokumen: [row('100002', 'Zeta'), row('100001', '', { nama_tipe_dokumen: '' })],
}

type Call = { url: string; method: string }

let calls: Call[] = []

type Answer = Response | Promise<Response> | 'putus' | 'rusak' | undefined

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (call: Call) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = { url, method: init?.method ?? 'GET' }
    calls.push(call)
    const custom = answer(call)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'rusak') {
      return Promise.resolve({
        get status(): number {
          throw new TypeError('rusak')
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url.startsWith(`${ROUTE}/pilihan`)) return Promise.resolve(json(200, REFERENCES))
    if (call.method === 'GET' && url.startsWith(`${ROUTE}/`)) {
      return Promise.resolve(json(200, { portal: 'ASM', detail_tipe_dokumen: row('100002', 'Zeta') }))
    }
    if (call.method === 'GET') return Promise.resolve(json(200, LIST))
    return Promise.resolve(json(200, { portal: 'ASM', detail_tipe_dokumen: row('100002', 'Zeta') }))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <DetailDocumentTypePage />
    </QueryClientProvider>,
  )
}

const isList = (call: Call) => call.method === 'GET' && call.url === ROUTE

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('daftar', () => {
  it.each([
    { name: 'jaringan putus', answer: (): Answer => 'putus', title: 'Server Claim PNC tidak dapat dihubungi' },
    {
      name: 'portal tidak disebut',
      answer: (): Answer => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal tidak dikenal',
      answer: (): Answer => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'galat API lain',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'x' }),
      title: 'Daftar tidak dapat dimuat',
      description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
    },
    {
      name: 'galat bukan API',
      answer: (): Answer => 'rusak',
      title: 'Daftar tidak dapat dimuat',
      description: 'Coba beberapa saat lagi.',
    },
  ])('menampilkan galat muat: $name', async ({ answer, title, description }) => {
    installFetch((call) => (isList(call) ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat daftar detail tipe dokumen…')).toBeInTheDocument()
    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })

  it('mengurutkan menurut setiap kolom, menandai nama kosong, dan memuat ulang', async () => {
    installFetch()
    show()
    const table = await screen.findByRole('table')

    const emptyRow = within(table).getByRole('button', { name: 'Ubah 100001' }).closest('tr')!
    expect(within(emptyRow).getAllByText('—').length).toBeGreaterThan(0)
    // Baris tanpa detail dokumen memakai ID-nya pada label tombol Ubah.
    expect(within(table).getByRole('button', { name: 'Ubah 100001' })).toBeInTheDocument()

    for (const title of ['ID', 'Tipe Dokumen', 'Detail Dokumen', 'Objek Dokumen', 'Penyebab Kerugian']) {
      const header = within(table).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(header).getByRole('button'))
      expect(header).toHaveAttribute('aria-sort', 'ascending')
    }
    const id = within(table).getByRole('columnheader', { name: /^ID/ })
    await userEvent.click(within(id).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('100001')

    const before = calls.filter(isList).length
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.filter(isList).length).toBe(before + 1))
  })

  it('menyebut seluruh daftar pilihan tidak tersedia saat pilihan gagal dimuat', async () => {
    installFetch((call) => (call.url.startsWith(`${ROUTE}/pilihan`) ? json(500, { kode: 'x', pesan: 'x' }) : undefined))
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      await screen.findByText(/Saran untuk isian Tipe Dokumen, Penyebab Kerugian, Objek Dokumen, Bisnis tidak tersedia/),
    ).toBeInTheDocument()
  })
})

describe('simpan', () => {
  async function openEdit() {
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah Zeta' }))
    const form = await screen.findByRole('form')
    await waitFor(() => expect(within(form).getByRole('button', { name: 'Ubah' })).toBeEnabled())
    return form
  }

  it.each([
    {
      name: 'baris sudah tidak ada',
      answer: (): Answer => json(404, { kode: 'tidak_ditemukan', pesan: 'x' }),
      title: 'Baris ini sudah tidak ada',
    },
    {
      name: 'portal tidak disebut',
      answer: (): Answer => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal tidak dikenal',
      answer: (): Answer => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal belum siap',
      answer: (): Answer => json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      title: 'Basis data entitas ini belum tersedia',
    },
    {
      name: 'galat API lain',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'x' }),
      title: 'Terjadi kesalahan pada sistem',
    },
    {
      name: 'jaringan putus',
      answer: (): Answer => 'putus',
      title: 'Server Claim PNC tidak dapat dihubungi',
    },
  ])('menampilkan galat simpan: $name', async ({ answer, title }) => {
    installFetch((call) => (call.method === 'PUT' ? answer() : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('tidak menampilkan kotak pesan untuk galat bukan API lalu menutup lewat Batal', async () => {
    installFetch((call) => (call.method === 'PUT' ? 'rusak' : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    await waitFor(() => expect(within(form).getByRole('button', { name: 'Ubah' })).toBeEnabled())
    expect(within(form).queryByText('Terjadi kesalahan pada sistem')).not.toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form')).not.toBeInTheDocument()
  })
})
