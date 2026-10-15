import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimStatusPage } from './ClaimStatusPage'

/** Uji tambahan Master Status Klaim: galat muat dan simpan, Refresh, dan tombol bekerja. */

const LIST = {
  status_klaim: [
    { kode: '1147', label: 'Register', kode_lama: '' },
    { kode: '1163', label: 'Paid', kode_lama: '' },
  ],
  total: 2,
  portal: 'ASM',
}

type Reply = { body?: unknown; status?: number; fail?: boolean; hold?: boolean }
let calls: { url: string; method: string }[] = []

function installFetch(map: (method: string) => Reply | null) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method })
    const reply = map(method) ?? { body: LIST }
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

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ClaimStatusPage />
    </QueryClientProvider>,
  )
}

async function openEditPaid(user: ReturnType<typeof userEvent.setup>) {
  await screen.findAllByText('Paid')
  await user.click(screen.getByRole('button', { name: 'Ubah status Paid' }))
  return screen.getByRole('form', { name: 'Ubah status klaim' })
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
    [{ fail: true }, 'Tidak dapat menghubungi server'],
    [{ status: 400, body: { kode: 'portal_tidak_disebut', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'Tabel status hilang.' } }, 'Daftar status gagal dimuat'],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch(() => reply)
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('memuat ulang daftar lewat Refresh', async () => {
    installFetch(() => null)
    const user = userEvent.setup()
    show()

    await screen.findAllByText('Paid')
    const before = calls.length
    await user.click(screen.getByRole('button', { name: /Refresh/ }))
    await waitFor(() => expect(calls.length).toBe(before + 1))
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Tidak dapat menghubungi server'],
    [{ status: 404, body: { kode: 'status_klaim_tidak_ditemukan', pesan: 'x' } }, 'Status tidak ditemukan'],
    [{ status: 409, body: { kode: 'kode_status_sudah_dipakai', pesan: 'x' } }, 'Kode bentrok'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'Penyimpanan rusak.' } }, 'Gagal menyimpan'],
  ])('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((method) => (method === 'PUT' ? reply : null))
    const user = userEvent.setup()
    show()

    const form = await openEditPaid(user)
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('memakai pesan server saat galat validasi tidak membawa detail', async () => {
    installFetch((method) =>
      method === 'PUT' ? { status: 422, body: { kode: 'validasi_gagal', pesan: 'Status tidak sah.' } } : null,
    )
    const user = userEvent.setup()
    show()

    const form = await openEditPaid(user)
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByText('Isian belum benar')).toBeInTheDocument()
    expect(within(form).getByText('Status tidak sah.')).toBeInTheDocument()
  })

  it('mengunci tombol dan menulis Menyimpan selama permintaan berjalan', async () => {
    installFetch((method) => (method === 'PUT' ? { hold: true } : null))
    const user = userEvent.setup()
    show()

    const form = await openEditPaid(user)
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))
    expect(await within(form).findByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
    expect(within(form).getByRole('button', { name: 'Batal' })).toBeDisabled()
  })
})
