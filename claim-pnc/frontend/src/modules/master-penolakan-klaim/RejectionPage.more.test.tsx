import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { RejectionPage } from './RejectionPage'

/**
 * Uji tambahan Master Penolakan Klaim: galat muat dan simpan pada kedua tab, validasi induk
 * baru, sorotan pelanggaran, dan pengurutan. Seluruh nilai KARANGAN (`D-69`).
 */

const PARENTS = { portal: 'ASM', status_1: [{ id: '1', nama: 'POLIS TIDAK BERLAKU' }] }
const ROW = {
  id: '3',
  nama: 'ALASAN ZETA',
  id_status_1: '1',
  nama_status_1: 'POLIS TIDAK BERLAKU',
  status: '2',
  status_label: 'REJECTED',
  diajukan_oleh: 'adminpnc',
  diajukan_pada: '',
  disetujui_oleh: 'manageradmin',
  disetujui_pada: 'bukan-tanggal',
  catatan_persetujuan: '',
}
const ROW_B = { ...ROW, id: '4', nama: 'ALASAN ALFA', status: '0', status_label: 'MENUNGGU', disetujui_oleh: '' }
const KOMITE = { portal: 'ASM', penolakan_komite: [{ id: '112', catatan: 'ZETA' }, { id: '111', catatan: 'ALFA' }] }

type Call = { url: string; method: string; body: unknown }
type Reply = { body?: unknown; status?: number; fail?: boolean }
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
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function defaults(call: Call): Reply {
  if (call.url.startsWith('/api/master/penolakan-klaim/status-1')) return { body: PARENTS }
  if (call.method === 'GET' && call.url.startsWith('/api/master/penolakan-komite')) return { body: KOMITE }
  if (call.method === 'GET') return { body: { portal: 'ASM', penolakan_klaim: [ROW, ROW_B] } }
  if (call.url.startsWith('/api/master/penolakan-komite')) {
    return { status: 201, body: { penolakan_komite: KOMITE.penolakan_komite[0], portal: 'ASM' } }
  }
  return { status: 201, body: { penolakan_klaim: ROW, portal: 'ASM' } }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <RejectionPage />
    </QueryClientProvider>,
  )
}

const isList = (call: Call) => call.method === 'GET' && !call.url.includes('status-1')
const isMutation = (call: Call) => call.method === 'POST' || call.method === 'PUT'

const LOAD_FAILURES: [Reply, string][] = [
  [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
  [{ status: 400, body: { kode: 'portal_tidak_disebut', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
  [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar tidak dapat dimuat'],
]

const SAVE_FAILURES: [Reply, string][] = [
  [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
  [{ status: 422, body: { kode: 'validasi_gagal', pesan: 'Ditolak.' } }, 'Belum dapat disimpan'],
  [{ status: 404, body: { kode: 'tidak_ditemukan', pesan: 'x' } }, 'Baris ini sudah tidak ada'],
  [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
  [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
]

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

describe('tab Penolakan Klaim', () => {
  it.each(LOAD_FAILURES)('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch((call) => (isList(call) ? reply : null))
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it.each(SAVE_FAILURES)('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((call) => (isMutation(call) ? reply : null))
    const user = userEvent.setup()
    show()

    await user.click(await screen.findByRole('button', { name: 'Ubah ALASAN ZETA' }))
    const form = screen.getByRole('form', { name: 'Ubah Penolakan Klaim' })
    expect(within(form).getByText('Menyimpan akan membatalkan persetujuan')).toBeInTheDocument()
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('menolak induk baru yang kosong atau terlalu panjang', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: 'Tambah Penolakan Klaim' })
    await screen.findByRole('option', { name: '1 — POLIS TIDAK BERLAKU' })
    await user.selectOptions(within(form).getByLabelText('Status Penolakan 1'), '+ Status Penolakan 1 baru…')
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'ALASAN BARU')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText('Status Penolakan 1 baru wajib diisi.')).toBeInTheDocument()

    // maxLength isian mencegah ketikan berlebih; tempelan yang dipotong tetap lolos skema.
    const fresh = within(form).getByLabelText('Nama Status Penolakan 1 baru')
    await user.click(fresh)
    await user.paste('B'.repeat(10))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    await waitFor(() => expect(calls.some(isMutation)).toBe(true))
    expect(calls.find(isMutation)?.body).toEqual({ nama: 'ALASAN BARU', id_status_1: '', nama_status_1: 'B'.repeat(10) })
  })

  it('menyorot pelanggaran induk dari server pada isian yang benar', async () => {
    installFetch((call) =>
      isMutation(call)
        ? {
            status: 422,
            body: {
              kode: 'validasi_gagal',
              pesan: 'Ditolak.',
              detail: [
                { field: 'id_status_1', pesan: 'Induk tidak dikenal.' },
                { field: 'nama_status_1', pesan: 'Nama induk bentrok.' },
              ],
            },
          }
        : null,
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: 'Tambah Penolakan Klaim' })
    await screen.findByRole('option', { name: '1 — POLIS TIDAK BERLAKU' })
    await user.selectOptions(within(form).getByLabelText('Status Penolakan 1'), '+ Status Penolakan 1 baru…')
    await user.type(within(form).getByLabelText('Nama Status Penolakan 1 baru'), 'INDUK')
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'ANAK')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Induk tidak dikenal.')).toBeInTheDocument()
    expect(within(form).getByText('Nama induk bentrok.')).toBeInTheDocument()
    expect(within(form).queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
  })

  it('menulis lencana menurut status, tanggal tak terbaca sebagai tanda pisah, dan mengurutkan', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('REJECTED').className).toContain('bg-red-50')
    expect(within(table).getByText('MENUNGGU').className).toContain('bg-slate-100')
    expect(within(table).getByText('manageradmin · —')).toBeInTheDocument()

    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(table).getByRole('button', { name: 'Status Penolakan 2' }))
    expect(first()).toContain('ALASAN ALFA')
    await user.click(within(table).getByRole('button', { name: 'Status Aproval' }))
    expect(first()).toContain('MENUNGGU')
    await user.click(within(table).getByRole('button', { name: 'Note Approval' }))
    await user.click(within(table).getByRole('button', { name: 'Status Penolakan 1' }))
    expect(first()).not.toBe('')

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))
  })
})

describe('tab Penolakan Komite', () => {
  async function openCommitteeTab(user: ReturnType<typeof userEvent.setup>) {
    await screen.findByRole('table')
    await user.click(screen.getByRole('tab', { name: 'Input Master Penolakan Komite' }))
    return screen.findByRole('columnheader', { name: /ID Master/ })
  }

  it.each(LOAD_FAILURES)('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch((call) =>
      call.method === 'GET' && call.url.startsWith('/api/master/penolakan-komite') ? reply : null,
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('tab', { name: 'Input Master Penolakan Komite' }))
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it.each(SAVE_FAILURES)('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((call) => (isMutation(call) ? reply : null))
    const user = userEvent.setup()
    show()

    await openCommitteeTab(user)
    await user.click(screen.getByRole('button', { name: 'Ubah ZETA' }))
    const form = screen.getByRole('form', { name: 'Ubah Penolakan Komite' })
    expect(within(form).getByText('112')).toBeInTheDocument()
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('menyorot catatan yang ditolak server lalu menutup form setelah berhasil', async () => {
    let attempt = 0
    installFetch((call) => {
      if (!isMutation(call)) return null
      attempt++
      return attempt === 1
        ? {
            status: 422,
            body: { kode: 'validasi_gagal', pesan: 'x', detail: [{ field: 'catatan', pesan: 'Catatan ganda.' }] },
          }
        : null
    })
    const user = userEvent.setup()
    show()

    await openCommitteeTab(user)
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: 'Tambah Penolakan Komite' })
    await user.type(within(form).getByLabelText('Note Komite Reject'), 'BARU')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText('Catatan ganda.')).toBeInTheDocument()

    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    await waitFor(() => expect(screen.queryByRole('form', { name: 'Tambah Penolakan Komite' })).not.toBeInTheDocument())
  })

  it('mengurutkan, menutup form lewat Batal, dan memuat ulang', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await openCommitteeTab(user)
    const table = screen.getByRole('table')
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(table).getByRole('button', { name: 'ID Master' }))
    expect(first()).toContain('111')
    await user.click(within(table).getByRole('button', { name: 'Note Komite Reject' }))
    expect(first()).toContain('ALFA')

    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form', { name: 'Tambah Penolakan Komite' })).not.toBeInTheDocument()

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))
  })
})
