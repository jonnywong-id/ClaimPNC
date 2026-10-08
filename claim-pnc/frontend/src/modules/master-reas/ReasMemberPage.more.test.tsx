import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReasMemberPage } from './ReasMemberPage'

/** Uji tambahan Master Reas: galat pemuatan, pencarian, dan Refresh. Nilai KARANGAN (`D-69`). */

const ROWS = [
  { kode_reas: 'RE-002', nama_reas: 'Reas Beta', login: 'beta', email: 'b@contoh.invalid', negara: 'Singapura', tipe: '2', cadangan: false },
  { kode_reas: 'RE-001', nama_reas: 'Reas Alfa', login: 'alfa', email: 'a@contoh.invalid', negara: 'Indonesia', tipe: '1', cadangan: true },
]

type Reply = { body?: unknown; status?: number; fail?: boolean }
let calls: string[] = []

function installFetch(map: () => Reply) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const reply = map()
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
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <ReasMemberPage />
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

it.each([
  [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
  [{ status: 400, body: { kode: 'portal_tidak_disebut', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
  [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar member reas tidak dapat dimuat'],
])('menjelaskan galat pemuatan %#', async (reply, title) => {
  installFetch(() => reply)
  show()
  expect(await screen.findByText(title)).toBeInTheDocument()
  expect(screen.queryByRole('table')).not.toBeInTheDocument()
})

/*
  Disesuaikan 2026-10-05 dengan kepala kolom Pega yang ditetapkan Work Owner:
  No · Nama Reinsurer · Login · Email · Tipe · Aksi.

  Dua pengurutan di uji ini semula menembak kolom yang kini tidak ada — "Negara" (dihapus;
  Pega tidak punya kolomnya) dan "Kode Reas" (isinya pindah ke kolom "No").

  Pencariannya pun ikut berubah, dan itu BUKAN sekadar penyesuaian uji: `DataTable` mencari
  ke kolom yang DIGAMBAR, sehingga hilangnya kolom Negara membuat "singapura" tidak lagi
  menemukan apa pun. Itu justru menyelaraskannya dengan penyaring `cari` di server, yang
  memang tidak pernah mencari ke `COUNTRY`.
*/
it('mencari ke kolom yang digambar dan mengurutkan menurut kepala kolomnya', async () => {
  installFetch(() => ({ body: { member_reas: ROWS, portal: 'ASM' } }))
  const user = userEvent.setup()
  show()

  const table = await screen.findByRole('table')

  // Kolom "No" TIDAK dapat diurutkan: isinya nomor urut tampilan, dan mengurutkannya hanya
  // akan menomori ulang barisnya. Pengurutan dilakukan lewat kolom yang isinya data.
  expect(within(table).queryByRole('button', { name: 'No' })).toBeNull()

  await user.click(within(table).getByRole('button', { name: 'Nama Reinsurer' }))
  expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Reas Alfa')

  // Kode reas TIDAK tergambar — Pega tidak menempelkannya di bawah nama (Work Owner,
  // 2026-10-05) — tetapi ia tetap DAPAT DICARI lewat `value` kolom itu.
  expect(within(table).getAllByRole('row')[1]).not.toHaveTextContent('RE-001')
  await user.type(screen.getByLabelText('Cari kode, nama, login, atau email'), 'RE-001')
  expect(screen.getByRole('table')).toHaveTextContent('Reas Alfa')
  expect(screen.getByRole('table')).not.toHaveTextContent('Reas Beta')
  await user.clear(screen.getByLabelText('Cari kode, nama, login, atau email'))

  await user.type(screen.getByLabelText('Cari kode, nama, login, atau email'), 'beta')
  expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2)
  expect(screen.getByRole('table')).toHaveTextContent('Reas Beta')
})

it('TIDAK mencari ke negara — kolomnya tidak digambar, dan server pun tidak mencarinya', async () => {
  installFetch(() => ({ body: { member_reas: ROWS, portal: 'ASM' } }))
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.type(screen.getByLabelText('Cari kode, nama, login, atau email'), 'singapura')

  // Tidak satu baris pun cocok. `DataTable` mengganti tabelnya dengan pesan kosong, jadi
  // yang diperiksa adalah hilangnya kedua baris — bukan jumlah baris pada tabel yang
  // memang tidak lagi digambar.
  expect(screen.queryByText('Reas Beta')).toBeNull()
  expect(screen.queryByText('Reas Alfa')).toBeNull()
})

it('memuat ulang daftar lewat Refresh', async () => {
  installFetch(() => ({ body: { member_reas: ROWS, portal: 'ASM' } }))
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  const before = calls.length
  await user.click(screen.getByRole('button', { name: 'Refresh' }))
  await waitFor(() => expect(calls.length).toBe(before + 1))
})
