import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DocumentTypePage } from './DocumentTypePage'

/** Uji tambahan Daftar Tipe Dokumen: galat muat, galat simpan, dan pengurutan. Nilai KARANGAN. */

const LIST = {
  portal: 'ASM',
  total: 2,
  tipe_dokumen: [
    { id: '10002', tipe_dokumen: 'Dokumen Survey', status_proses: 'Survey' },
    { id: '10001', tipe_dokumen: 'Dokumen Registrasi', status_proses: 'Register' },
  ],
}

type Reply = { body?: unknown; status?: number; fail?: boolean }
let calls: { url: string; method: string }[] = []

function installFetch(map: (url: string, method: string) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method })
    const reply = map(url, method)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
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
      <DocumentTypePage />
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
    installFetch(() => reply)
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
  ])('menjelaskan galat tambah %#', async (reply, title) => {
    installFetch((_, method) => (method === 'GET' ? { body: LIST } : reply))
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(title)).toBeInTheDocument()
    // Form tetap terbuka supaya isiannya tidak hilang.
    expect(screen.getByLabelText('Jenis Dokumen')).toBeInTheDocument()
  })

  it('menutup form lewat Batal dan membuka kunci tombol Tambah', async () => {
    installFetch(() => ({ body: LIST }))
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByLabelText('Jenis Dokumen')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).not.toBeDisabled()
  })
})

describe('daftar', () => {
  it('mengurutkan menurut ID, Tipe Dokumen, dan Status Proses', async () => {
    installFetch(() => ({ body: LIST }))
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const firstRow = () => within(table).getAllByRole('row')[1]?.textContent ?? ''

    await user.click(within(table).getByRole('button', { name: 'ID' }))
    expect(firstRow()).toContain('10001')
    await user.click(within(table).getByRole('button', { name: 'Tipe Dokumen' }))
    expect(firstRow()).toContain('Dokumen Registrasi')
    await user.click(within(table).getByRole('button', { name: 'Status Proses' }))
    expect(firstRow()).toContain('Register')
  })

  it('memuat ulang daftar lewat Refresh', async () => {
    installFetch(() => ({ body: LIST }))
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBe(before + 1))
  })
})
