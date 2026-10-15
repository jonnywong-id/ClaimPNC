import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { PartTypePage } from './PartTypePage'

/**
 * Uji tambahan Master Tipe Sparepart: galat muat, galat simpan, dan pengurutan.
 * Seluruh nilai KARANGAN (`D-69`).
 *
 * TIDAK ada uji keputusan borongan: layar ini tidak punya tombol Approve/Reject sama
 * sekali, mengikuti ketiga tab Pega yang nol tombol (koreksi Work Owner 2026-10-04).
 */

const APPROVED = {
  id_tipe_sparepart: '1',
  nama_tipe_sparepart: 'FUEL FILTER',
  id_kategori_sparepart: '1',
  status: '1',
  status_label: 'Approve',
}
const APPROVED_B = { ...APPROVED, id_tipe_sparepart: '2', nama_tipe_sparepart: 'AIR FILTER', id_kategori_sparepart: '3' }
const PENDING = [
  { ...APPROVED, id_tipe_sparepart: '4', nama_tipe_sparepart: 'TRACK ROLLER', status: '0' },
  { ...APPROVED, id_tipe_sparepart: '5', nama_tipe_sparepart: 'IDLER', status: '0' },
]
const OPTIONS = { kategori: [{ kode: '1', nama: 'ENGINE' }], terpotong: false, portal: 'ASM' }

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
  if (call.url.startsWith('/api/master/tipe-sparepart/pilihan')) return { body: OPTIONS }
  if (call.method === 'GET') {
    const status = new URL(call.url, 'https://x').searchParams.get('status')
    return {
      body: {
        tipe_sparepart: status === '0' ? PENDING : [APPROVED, APPROVED_B],
        status,
        portal: 'ASM',
      },
    }
  }
  return { body: { tipe_sparepart: APPROVED, portal: 'ASM' } }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <PartTypePage />
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
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar tipe sparepart tidak dapat dimuat'],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch((call) => (call.url.startsWith('/api/master/tipe-sparepart?') ? reply : null))
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 422, body: { kode: 'validasi_gagal', pesan: 'Ditolak.' } }, 'Belum dapat disimpan'],
    [{ status: 404, body: { kode: 'tidak_ditemukan', pesan: 'x' } }, 'Baris ini sudah tidak ada'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
  ])('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((call) => (call.method === 'PUT' ? reply : null))
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    await user.click(within(within(table).getByRole('row', { name: /FUEL FILTER/ })).getByRole('button', { name: 'Ubah' }))
    await screen.findByRole('option', { name: 'ENGINE' })
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menutup form setelah perubahan tersimpan', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    await user.click(within(within(table).getByRole('row', { name: /FUEL FILTER/ })).getByRole('button', { name: 'Ubah' }))
    await screen.findByRole('option', { name: 'ENGINE' })
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.queryByLabelText('Nama Tipe Sparepart')).not.toBeInTheDocument())
    expect(calls.find((c) => c.method === 'PUT')?.url).toBe('/api/master/tipe-sparepart/1')
  })

  it('menyebut kategori yatim dengan kodenya pada dropdown', async () => {
    installFetch((call) =>
      call.method === 'GET' && call.url.startsWith('/api/master/tipe-sparepart?')
        ? { body: { tipe_sparepart: [{ ...APPROVED, id_kategori_sparepart: '99' }], portal: 'ASM' } }
        : null,
    )
    const user = userEvent.setup()
    show()

    await user.click(await screen.findByRole('button', { name: 'Ubah' }))
    expect(
      await screen.findByRole('option', { name: '99 — kategori tidak lagi tersedia' }),
    ).toBeInTheDocument()
  })
})

describe('daftar', () => {
  it('mengurutkan setiap kolom data dan memuat ulang', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(table).getByRole('button', { name: 'Nama Tipe Sparepart' }))
    expect(first()).toContain('AIR FILTER')
    await user.click(within(table).getByRole('button', { name: 'ID Kategori Sparepart' }))
    expect(first()).toContain('FUEL FILTER')
    await user.click(within(table).getByRole('button', { name: 'ID Tipe Sparepart' }))
    expect(first()).toContain('FUEL FILTER')

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))
  })
})
