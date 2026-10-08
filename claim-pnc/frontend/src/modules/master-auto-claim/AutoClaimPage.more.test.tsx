import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AutoClaimPage } from './AutoClaimPage'

/**
 * Uji tambahan Master Auto Claim: galat muat dan simpan, pemilih Sumber Bisnis dan Client,
 * jalur tambah, dan pengurutan. Seluruh nilai KARANGAN (`D-69`).
 */

const ROW = {
  inisial: 'AGN010',
  nama_penerima: 'MITRA ZETA',
  nama_bank: 'BANK CONTOH NIAGA',
  no_rekening: '2000000002',
  pct_max: '90',
  pic_lapor: 'PIC Zeta',
  email_lapor: 'zeta@contoh.example',
  alamat_penerima: 'Jalan Z',
  id_client: 'CLI010',
  nama_client: 'TERTANGGUNG ZETA',
  claim_allowed: '1',
  komite: 'ADMINPNC',
  status: '1',
  status_label: 'Approve',
  dapat_dipakai: true,
}
const ROW_B = { ...ROW, inisial: 'AGN001', nama_penerima: 'MITRA ALFA', pct_max: '10', komite: 'KOMITEA' }

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
  if (call.url.startsWith('/api/master/auto-claim/bank')) {
    return { body: { portal: 'ASM', bank: [{ kode: '001', nama: 'BANK CONTOH NIAGA' }] } }
  }
  if (call.url.startsWith('/api/master/auto-claim/sumber-bisnis')) {
    return { body: { portal: 'ASM', sumber_bisnis: [{ id: 'AGN099', nama: 'PT SUMBER BARU' }] } }
  }
  if (call.url.startsWith('/api/master/auto-claim/client')) {
    return { body: { portal: 'ASM', client: [{ id: 'CLI099', nama: 'PT CLIENT BARU' }] } }
  }
  if (call.method === 'GET') return { body: { auto_claim: [ROW, ROW_B], portal: 'ASM' } }
  return { body: { auto_claim: ROW, portal: 'ASM' } }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <AutoClaimPage />
    </QueryClientProvider>,
  )
}

async function openEditForm(user: ReturnType<typeof userEvent.setup>) {
  const table = await screen.findByRole('table')
  const row = within(table).getByRole('row', { name: /MITRA ZETA/ })
  await user.click(within(row).getByRole('button', { name: 'Ubah' }))
  return screen.findByRole('form', { name: 'Ubah Master Auto Claim' })
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
    installFetch((call) => (call.url.startsWith('/api/master/auto-claim?') ? reply : null))
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 409, body: { kode: 'sumber_bisnis_sudah_ada', pesan: 'x' } }, 'Sumber bisnis ini sudah ada di Master Auto Claim'],
    [{ status: 422, body: { kode: 'validasi_gagal', pesan: 'Ditolak.' } }, 'Belum dapat disimpan'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'status_tidak_dikenal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
  ])('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((call) => (call.method === 'PUT' ? reply : null))
    const user = userEvent.setup()
    show()

    const form = await openEditForm(user)
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('menyorot galat client di bawah client yang sudah dipilih', async () => {
    installFetch((call) =>
      call.method === 'PUT'
        ? {
            status: 422,
            body: {
              kode: 'validasi_gagal',
              pesan: 'Ditolak.',
              detail: [
                { field: 'id_client', pesan: 'Client sudah tidak ada.' },
                { field: 'no_rekening', pesan: 'Rekening tidak sah.' },
              ],
            },
          }
        : null,
    )
    const user = userEvent.setup()
    show()

    const form = await openEditForm(user)
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText('Client sudah tidak ada.')).toHaveAttribute('role', 'alert')
    expect(within(form).getByText('Rekening tidak sah.')).toBeInTheDocument()
    expect(within(form).queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
  })

  it('menyimpan perubahan dengan client dikosongkan lalu menutup form', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const form = await openEditForm(user)
    const clientBox = within(form).getByText('TERTANGGUNG ZETA').closest('div') as HTMLElement
    await user.click(within(clientBox).getByRole('button', { name: 'Ganti' }))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    expect(calls.find((c) => c.method === 'PUT')?.body).toMatchObject({
      id_client: '',
      nama_client: '',
      status: '0',
    })
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Ubah Master Auto Claim' })).not.toBeInTheDocument(),
    )
  })
})

describe('tambah', () => {
  it('menambah sumber bisnis beserta client lewat POST', async () => {
    installFetch((call) =>
      call.method === 'POST' ? { status: 201, body: { auto_claim: ROW, portal: 'ASM' } } : null,
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Master Auto Claim' })

    await user.type(within(form).getByRole('searchbox', { name: 'SUMBER BISNIS' }), 'su')
    await user.click(await within(form).findByRole('button', { name: /PT SUMBER BARU/ }))
    await user.type(within(form).getByRole('searchbox', { name: 'CARI CLIENT' }), 'cl')
    await user.click(await within(form).findByRole('button', { name: /PT CLIENT BARU/ }))

    await within(form).findByRole('option', { name: /BANK CONTOH NIAGA/ })
    await user.selectOptions(within(form).getByLabelText('BANK PENERIMA'), 'BANK CONTOH NIAGA')
    await user.type(within(form).getByLabelText('NO REKENING'), '123')
    await user.type(within(form).getByLabelText('PCT_MAX'), '50')
    await user.type(within(form).getByLabelText('PIC'), 'PIC Baru')
    await user.type(within(form).getByLabelText('EMAIL LAPOR'), 'baru@contoh.example')
    await user.type(within(form).getByLabelText('ALAMAT PENERIMA'), 'Jalan Baru')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'POST')).toBe(true))
    expect(calls.find((c) => c.method === 'POST')?.body).toMatchObject({
      inisial: 'AGN099',
      nama_penerima: 'PT SUMBER BARU',
      id_client: 'CLI099',
      nama_client: 'PT CLIENT BARU',
      pct_max: '50',
    })
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Tambah Master Auto Claim' })).not.toBeInTheDocument(),
    )
  })

  it('pemilih sumber bisnis menyebut kosong, gagal, sedang mencari, dan dapat diganti', async () => {
    let mode: 'empty' | 'fail' | 'hold' | 'ok' = 'empty'
    installFetch((call) => {
      if (!call.url.startsWith('/api/master/auto-claim/sumber-bisnis')) return null
      if (mode === 'empty') return { body: { portal: 'ASM', sumber_bisnis: [] } }
      if (mode === 'fail') return { status: 500, body: {} }
      if (mode === 'hold') return { hold: true }
      return null
    })
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Master Auto Claim' })
    const box = within(form).getByRole('searchbox', { name: 'SUMBER BISNIS' })

    await user.type(box, 'z')
    expect(within(form).queryByText('Mencari…')).not.toBeInTheDocument()
    await user.type(box, 'z')
    expect(await within(form).findByText('Tidak ada yang cocok dengan “zz”.')).toBeInTheDocument()

    mode = 'fail'
    await user.type(box, 'q')
    expect(await within(form).findByText('Pencarian gagal. Coba beberapa saat lagi.')).toBeInTheDocument()

    mode = 'hold'
    await user.type(box, 'w')
    expect(await within(form).findByText('Mencari…')).toBeInTheDocument()

    mode = 'ok'
    await user.clear(box)
    await user.type(box, 'ok')
    await user.click(await within(form).findByRole('button', { name: /PT SUMBER BARU/ }))
    await user.click(within(form).getAllByRole('button', { name: 'Ganti' })[0]!)
    expect(within(form).getByRole('searchbox', { name: 'SUMBER BISNIS' })).toHaveValue('')
    expect(within(form).getByRole('button', { name: 'Simpan' })).toBeDisabled()
  })
})

describe('daftar', () => {
  it('mengurutkan setiap kolom data dan memuat ulang', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    for (const title of [
      'INISIAL',
      'NAMA PENERIMA',
      'BANK PENERIMA',
      'NO REKENING',
      'ALAMAT PENERIMA',
      'EMAIL LAPOR',
      'PCT MAX',
      'PIC',
      'KOMITE',
    ]) {
      await user.click(within(table).getByRole('button', { name: title }))
      expect(first()).not.toBe('')
    }
    await user.click(within(table).getByRole('button', { name: 'INISIAL' }))
    expect(first()).toContain('AGN001')

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))
  })
})
