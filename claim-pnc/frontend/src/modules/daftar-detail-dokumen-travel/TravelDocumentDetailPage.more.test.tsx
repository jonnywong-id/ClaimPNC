import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { TravelDocumentDetailPage } from './TravelDocumentDetailPage'

/**
 * Uji tambahan Daftar Detail Dokumen Travel: galat muat dan simpan, jalur tambah, dan
 * pengurutan. Seluruh nilai KARANGAN (`D-69`).
 */

const LIST = {
  portal: 'ASM',
  total: 2,
  detail_dokumen_travel: [
    { id: '00002', id_dokumen: '100009', nama_dokumen: 'Tiket', status_wajib: false, minimal_unggah: 3, jaminan: [] },
    { id: '00001', id_dokumen: '100001', nama_dokumen: '', status_wajib: true, minimal_unggah: 1, jaminan: [] },
  ],
}
const PLANS = {
  portal: 'ASM',
  plan: [{ id: 'TP01', nama: 'Travel Plan Silver' }],
  jaminan: [{ id: 'TC02', nama: 'Kehilangan Bagasi', id_plan: 'TP01' }],
}

type Call = { url: string; method: string; body: unknown }
type Reply = { body?: unknown; status?: number; fail?: boolean; hold?: boolean }
let calls: Call[] = []

function installFetch(map: (call: Call) => Reply | null) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
    }
    calls.push(call)
    const reply = map(call) ?? defaults(call)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    if (reply.hold) return new Promise(() => {})
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function defaults(call: Call): Reply {
  if (call.url.startsWith('/api/master/dokumen-travel-pilihan')) {
    return { body: { portal: 'ASM', dokumen: [{ id: '100001', nama: 'Paspor' }] } }
  }
  if (call.url.startsWith('/api/master/plan-travel')) return { body: PLANS }
  if (call.method === 'GET' && call.url.includes('/daftar-detail-dokumen-travel/')) {
    return { body: { portal: 'ASM', detail_dokumen_travel: { ...LIST.detail_dokumen_travel[0], jaminan: [] } } }
  }
  if (call.method === 'GET') return { body: LIST }
  return { status: 201, body: { portal: 'ASM', detail_dokumen_travel: LIST.detail_dokumen_travel[0] } }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <TravelDocumentDetailPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('galat pemuatan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 400, body: { kode: 'portal_tidak_disebut', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar tidak dapat dimuat'],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch((call) => (call.url === '/api/master/daftar-detail-dokumen-travel' ? reply : null))
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 404, body: { kode: 'detail_dokumen_travel_tidak_ditemukan', pesan: 'x' } }, 'Baris ini sudah tidak ada'],
    [{ status: 404, body: { kode: 'tidak_ditemukan', pesan: 'x' } }, 'Baris ini sudah tidak ada'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
  ])('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((call) => (call.method === 'PUT' ? reply : null))
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah Tiket' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })
    await waitFor(() => expect(within(form).getByRole('button', { name: 'Ubah' })).not.toBeDisabled())
    await user.click(within(form).getByRole('button', { name: 'Ubah' }))

    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('menutup form setelah perubahan tersimpan, dan baris tanpa nama memakai ID-nya', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    expect(screen.getByRole('button', { name: 'Ubah 00001' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Ubah Tiket' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })
    await waitFor(() => expect(within(form).getByRole('button', { name: 'Ubah' })).not.toBeDisabled())
    await user.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => expect(screen.queryByRole('form', { name: 'Detail Dokumen Travel' })).not.toBeInTheDocument())
    expect(calls.find((c) => c.method === 'PUT')?.url).toBe('/api/master/daftar-detail-dokumen-travel/00002')
  })

  it('menyebut daftar dokumen pilihan yang gagal dimuat', async () => {
    installFetch((call) =>
      call.url.startsWith('/api/master/dokumen-travel-pilihan') ? { status: 500, body: {} } : null,
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(await screen.findByText('Daftar dokumen travel tidak dapat dimuat')).toBeInTheDocument()
  })
})

describe('tambah', () => {
  it('mengirim plan dan jaminan yang kodenya dicarikan dari nama, lalu menutup form', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })

    await user.type(within(form).getByLabelText('ID Dokumen'), '100001')
    await user.type(within(form).getByLabelText('Nama Dokumen'), 'Paspor')
    await user.selectOptions(within(form).getByLabelText('Status Wajib'), 'Ya')
    await user.click(within(form).getByRole('button', { name: 'Tambah Plan' }))
    await user.click(within(form).getByRole('button', { name: 'Tambah Plan' }))
    await user.type(within(form).getByLabelText('Nama Plan baris 1'), 'Travel Plan Silver')
    await user.type(within(form).getByLabelText('Nama Jaminan baris 1'), 'Kehilangan Bagasi')
    await user.type(within(form).getByLabelText('Nama Jaminan baris 2'), 'Jaminan bebas')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'POST')).toBe(true))
    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      id_dokumen: '100001',
      nama_dokumen: 'Paspor',
      status_wajib: true,
      minimal_unggah: 0,
      jaminan: [
        { id_plan: 'TP01', nama_plan: 'Travel Plan Silver', id_jaminan: 'TC02', nama_jaminan: 'Kehilangan Bagasi' },
        { id_plan: '', nama_plan: '', id_jaminan: '', nama_jaminan: 'Jaminan bebas' },
      ],
    })
    await waitFor(() => expect(screen.queryByRole('form', { name: 'Detail Dokumen Travel' })).not.toBeInTheDocument())
  })

  it('menghapus baris pembatasan sebelum menyimpan', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })
    await user.click(within(form).getByRole('button', { name: 'Tambah Plan' }))
    await user.click(within(form).getByRole('button', { name: 'Hapus pembatasan baris 1' }))
    expect(within(form).getByText(/Belum ada pembatasan/)).toBeInTheDocument()
  })
})

describe('daftar', () => {
  it('mengurutkan setiap kolom data dan memuat ulang', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(table).getByRole('button', { name: 'ID' }))
    expect(first()).toContain('00001')
    await user.click(within(table).getByRole('button', { name: 'ID Dokumen' }))
    expect(first()).toContain('100001')
    await user.click(within(table).getByRole('button', { name: 'Nama Dokumen' }))
    expect(first()).toContain('00001')
    await user.click(within(table).getByRole('button', { name: 'Status Wajib' }))
    expect(first()).toContain('Tidak')
    await user.click(within(table).getByRole('button', { name: 'Minimal Unggah' }))
    expect(first()).toContain('00001')

    await user.type(screen.getByRole('searchbox'), 'tiket')
    expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2)

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))
  })
})
