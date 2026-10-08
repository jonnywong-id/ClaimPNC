import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SupplierPage } from './SupplierPage'

/**
 * Uji tambahan Master Supplier: galat muat dan simpan, pemilih kota, pengurutan, dan
 * jalur tambah yang berhasil. Seluruh nilai KARANGAN (`D-69`).
 */

const CODES = {
  portal: 'ASM',
  status_rekanan: [{ nilai: '1', label: '1' }],
  status_supply: [{ nilai: '1', label: 'Heavy Equipment' }],
  jenis_supplier: [{ nilai: '1', label: '1' }],
  status_aktif: [{ nilai: '1', label: 'Aktif' }],
  status_autopayment: [{ nilai: '1', label: 'Ya' }],
}

const ROW = {
  id_supplier: '0100000000009',
  id_lama: 'SUP-LAMA-9',
  nama: 'Supplier Zeta',
  alamat: 'Jalan Z',
  kota: 'Jakarta Pusat',
  nama_cabang: 'Cabang Contoh',
  kode_pos: '10110',
  negara: 'Indonesia',
  telepon: '021-9',
  fax: '',
  email: '',
  npwp: '',
  contact_person: 'Narahubung',
  status_rekanan: '1',
  status_supply: '1',
  supplier_he: '1',
  term_of_payment: '30',
  term_of_delivery: '14',
  keterangan: '',
  bank: 'Bank Contoh',
  no_account: '1',
  account_name: '',
  bank_branch: '',
  jenis_supplier: '1',
  status_aktif: '1',
  status_aktif_berlaku: '',
  status_autopayment: '1',
  diubah_oleh: '',
  diubah_pada: '',
  heavy_equipment: true,
  aktif: true,
}

const ROW_B = { ...ROW, id_supplier: '0100000000001', nama: 'Supplier Alfa', alamat: 'Jalan A', telepon: '021-1', id_lama: '' }

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
  if (call.url.startsWith('/api/master/supplier/cabang')) {
    return { body: { portal: 'ASM', cabang: [{ id: '001', nama: 'Cabang Contoh' }] } }
  }
  if (call.url.startsWith('/api/master/supplier/negara')) {
    return { body: { portal: 'ASM', negara: [{ id: 'ID', nama: 'Indonesia' }] } }
  }
  if (call.url.startsWith('/api/master/supplier/bank')) {
    return { body: { portal: 'ASM', bank: [{ kode: '002', nama: 'Bank Contoh' }] } }
  }
  if (call.url.startsWith('/api/master/supplier/kota')) {
    return { body: { portal: 'ASM', kota: [{ id: '3171', nama: 'Jakarta Pusat' }] } }
  }
  if (call.url.startsWith('/api/master/supplier/sandi')) return { body: CODES }
  if (call.method === 'GET') return { body: { supplier: [ROW, ROW_B], portal: 'ASM' } }
  return { body: { supplier: ROW, portal: 'ASM' } }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <SupplierPage />
    </QueryClientProvider>,
  )
}

async function openEdit(user: ReturnType<typeof userEvent.setup>, name = 'Supplier Zeta') {
  const table = await screen.findByRole('table')
  const row = within(table).getByRole('row', { name: new RegExp(name) })
  await user.click(within(row).getByRole('button', { name: 'Ubah' }))
  await screen.findByLabelText('Nama')
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
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar supplier tidak dapat dimuat'],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch((call) => (call.url === '/api/master/supplier' ? reply : null))
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 409, body: { kode: 'nama_supplier_sudah_ada', pesan: 'x' } }, 'Nama supplier itu sudah dipakai'],
    [{ status: 409, body: { kode: 'nama_supplier_terkunci', pesan: 'x' } }, 'Nama supplier tidak dapat diubah'],
    [{ status: 422, body: { kode: 'validasi_gagal', pesan: 'Ditolak.' } }, 'Belum dapat disimpan'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
  ])('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((call) => (call.method === 'PUT' ? reply : null))
    const user = userEvent.setup()
    show()

    await openEdit(user)
    // Nomor lama ditampilkan pada keterangan ID.
    expect(screen.getByText(/Nomor lama: SUP-LAMA-9\./)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Simpan' }))
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menampilkan galat kota dari server di bawah kota yang sudah dipilih', async () => {
    installFetch((call) =>
      call.method === 'PUT'
        ? {
            status: 422,
            body: {
              kode: 'validasi_gagal',
              pesan: 'Ditolak.',
              detail: [{ field: 'kota', pesan: 'Kota tidak ada lagi di master.' }],
            },
          }
        : null,
    )
    const user = userEvent.setup()
    show()

    await openEdit(user)
    await user.click(screen.getByRole('button', { name: 'Simpan' }))
    expect(await screen.findByText('Kota tidak ada lagi di master.')).toHaveAttribute('role', 'alert')
    expect(screen.getByText('Jakarta Pusat', { selector: 'span' })).toBeInTheDocument()
  })
})

describe('pemilih kota', () => {
  it('mengganti kota yang sudah dipilih lewat pencarian', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await openEdit(user)
    await user.click(screen.getByRole('button', { name: 'Ganti' }))
    const box = screen.getByLabelText('Kota')
    await user.type(box, 'j')
    expect(screen.queryByText('Mencari…')).not.toBeInTheDocument()
    await user.type(box, 'a')
    await user.click(await screen.findByRole('button', { name: /Jakarta Pusat/ }))

    expect(calls.some((c) => c.url === '/api/master/supplier/kota?cari=ja')).toBe(true)
    expect(screen.getByRole('button', { name: 'Ganti' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Simpan' }))
    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))
    expect(calls.find((c) => c.method === 'PUT')?.body).toMatchObject({ kota: 'Jakarta Pusat' })
  })

  it('menyebut kosong, gagal, dan sedang mencari', async () => {
    let mode: 'empty' | 'fail' | 'hold' = 'empty'
    installFetch((call) => {
      if (!call.url.startsWith('/api/master/supplier/kota')) return null
      if (mode === 'empty') return { body: { portal: 'ASM', kota: [] } }
      if (mode === 'fail') return { status: 500, body: {} }
      return { hold: true }
    })
    const user = userEvent.setup()
    show()

    await openEdit(user)
    await user.click(screen.getByRole('button', { name: 'Ganti' }))
    const box = screen.getByLabelText('Kota')
    await user.type(box, 'zz')
    expect(await screen.findByText('Tidak ada kota yang cocok dengan “zz”.')).toBeInTheDocument()

    mode = 'fail'
    await user.type(box, 'q')
    expect(await screen.findByText('Pencarian kota gagal. Coba beberapa saat lagi.')).toBeInTheDocument()

    mode = 'hold'
    await user.type(box, 'w')
    expect(await screen.findByText('Mencari…')).toBeInTheDocument()
    // Isian dibuka kembali saat difokuskan.
    await user.click(box)
    expect(screen.getByText('Mencari…')).toBeInTheDocument()
  })
})

describe('daftar dan tambah', () => {
  it('mengurutkan setiap kolom data, menulis tanda pisah status, dan memuat ulang', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    for (const title of ['ID', 'Input Nama', 'Alamat', 'Telp', 'Jenis Supplier', 'Status Rekanan', 'Status Aktif']) {
      await user.click(within(table).getByRole('button', { name: title }))
      expect(first()).not.toBe('')
    }
    await user.click(within(table).getByRole('button', { name: 'Input Nama' }))
    expect(first()).toContain('Supplier Alfa')

    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBeGreaterThan(before))
  })

  it('menambah supplier lewat POST lalu menutup form', async () => {
    installFetch((call) =>
      call.method === 'POST' ? { status: 201, body: { supplier: ROW, portal: 'ASM' } } : null,
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'New Supplier' }))
    await screen.findByRole('option', { name: 'Cabang Contoh (001)' })

    for (const [label, value] of [
      ['Nama', 'Supplier Baru'],
      ['Alamat', 'Jalan Baru'],
      ['Telepon', '021-5'],
      ['Contact Person', 'Orang'],
      ['Term of Payment', '30'],
      ['Term of Delivery', '7'],
      ['No Account', '123'],
    ] as const) {
      await user.type(screen.getByLabelText(label), value)
    }
    await user.selectOptions(screen.getByLabelText('Cabang'), 'Cabang Contoh')
    await user.selectOptions(screen.getByLabelText('Negara'), 'Indonesia')
    await screen.findByRole('option', { name: 'Bank Contoh (002)' })
    await user.selectOptions(screen.getByLabelText('Bank'), 'Bank Contoh')
    for (const label of ['Status Rekanan', 'Status Supply', 'Jenis Supplier', 'Status Aktif']) {
      await user.selectOptions(screen.getByLabelText(label), '1')
    }
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'POST')).toBe(true))
    expect(calls.find((c) => c.method === 'POST')?.body).toMatchObject({
      nama: 'Supplier Baru',
      kota: '',
      bank: 'Bank Contoh',
      status_aktif: '1',
    })
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: 'Tambah Master Supplier' })).not.toBeInTheDocument(),
    )
  })
})
