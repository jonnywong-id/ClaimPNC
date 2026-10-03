import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SurveyorLoginForm } from './SurveyorLoginForm'
import { SurveyorLoginPage } from './SurveyorLoginPage'

/**
 * Uji tambahan Master Login: galat pemuatan dan simpan, pengurutan, dan pergantian baris.
 * Seluruh nama dan surel KARANGAN (`D-69`).
 */

const ROWS = [
  { nama: 'Zaki Contoh', login: 'ZakiContoh', email: 'z@contoh.invalid', telp: '2', alamat: 'B', status_login: '', login_leader: '' },
  { nama: 'Ani Contoh', login: 'AniContoh', email: 'a@contoh.invalid', telp: '1', alamat: 'A', status_login: 'Leader', login_leader: 'ZakiContoh' },
]

type Reply = { body?: unknown; status?: number; fail?: boolean }
let calls: { url: string; method: string }[] = []

function installFetch(map: (method: string) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method })
    const reply = map(method)
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
      <SurveyorLoginPage />
    </QueryClientProvider>,
  )
}

const LIST_REPLY: Reply = { body: { login_surveyor: ROWS, portal: 'ASM' } }

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
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar login surveyor tidak dapat dimuat'],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch(() => reply)
    show()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })
})

describe('galat simpan', () => {
  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 422, body: { kode: 'validasi_gagal', pesan: 'Isian ditolak.' } }, 'Belum dapat disimpan'],
    [{ status: 409, body: { kode: 'nama_login_surveyor_terkunci', pesan: 'x' } }, 'Nama tidak dapat diubah'],
    [{ status: 404, body: { kode: 'tidak_ditemukan', pesan: 'x' } }, 'Baris ini sudah tidak ada'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Terjadi kesalahan pada sistem'],
  ])('menjelaskan galat ubah %#', async (reply, title) => {
    installFetch((method) => (method === 'GET' ? LIST_REPLY : reply))
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const row = within(table).getByRole('row', { name: /Ani Contoh/ })
    await user.click(within(row).getByRole('button', { name: 'Ubah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(title)).toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Ubah Login Surveyor' })).toBeInTheDocument()
  })

  it('menyorot pelanggaran per isian tanpa kotak pesan, dan mengabaikan isian tak dikenal', async () => {
    installFetch((method) =>
      method === 'GET'
        ? LIST_REPLY
        : {
            status: 422,
            body: {
              kode: 'validasi_gagal',
              pesan: 'Isian ditolak.',
              detail: [
                { field: 'telp', pesan: 'Telp tidak sah.' },
                { field: 'login', pesan: 'tidak digambar' },
              ],
            },
          },
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Nama'), 'Baru Contoh')
    await user.type(screen.getByLabelText('Email'), 'baru@contoh.invalid')
    await user.type(screen.getByLabelText('Telp'), '9')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Telp tidak sah.')).toBeInTheDocument()
    expect(screen.queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
    expect(screen.queryByText('tidak digambar')).not.toBeInTheDocument()
  })

  it('menutup form setelah tambahan berhasil disimpan', async () => {
    installFetch((method) =>
      method === 'GET' ? LIST_REPLY : { status: 201, body: { login_surveyor: ROWS[0], portal: 'ASM' } },
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Nama'), 'Baru Contoh')
    await user.type(screen.getByLabelText('Email'), 'baru@contoh.invalid')
    await user.type(screen.getByLabelText('Telp'), '9')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() =>
      expect(screen.queryByRole('form', { name: 'Tambah Login Surveyor' })).not.toBeInTheDocument(),
    )
    expect(calls.some((c) => c.method === 'POST')).toBe(true)
  })
})

describe('daftar', () => {
  it('mengurutkan setiap kolom data dan menyaring lewat kotak cari', async () => {
    installFetch(() => LIST_REPLY)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    for (const title of ['Nama', 'Login', 'Email', 'Telp', 'Alamat']) {
      await user.click(within(table).getByRole('button', { name: title }))
      expect(first()).toContain('Ani Contoh')
    }

    await user.type(screen.getByRole('searchbox'), 'zaki')
    expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2)
  })

  it('memuat ulang lewat Refresh dan mengganti baris yang disunting tanpa menutup form', async () => {
    installFetch(() => LIST_REPLY)
    const user = userEvent.setup()
    show()

    const table = await screen.findByRole('table')
    const before = calls.length
    await user.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.length).toBe(before + 1))

    await user.click(within(within(table).getByRole('row', { name: /Ani Contoh/ })).getByRole('button', { name: 'Ubah' }))
    expect(screen.getByLabelText('Nama')).toHaveValue('Ani Contoh')
    expect(screen.getByText('Leader')).toBeInTheDocument()

    await user.click(within(within(table).getByRole('row', { name: /Zaki Contoh/ })).getByRole('button', { name: 'Ubah' }))
    expect(screen.getByLabelText('Nama')).toHaveValue('Zaki Contoh')
    // Status Login kosong ditulis tanda pisah.
    expect(screen.getByText('tidak bertaut ke tim mana pun')).toBeInTheDocument()
  })
})

describe('SurveyorLoginForm', () => {
  it('memotong ketikan pada batas panjang isiannya', async () => {
    const onSave = vi.fn()
    const user = userEvent.setup()
    render(
      <SurveyorLoginForm editing={null} isSaving={false} error={null} onSave={onSave} onCancel={() => {}} />,
    )

    await user.click(screen.getByLabelText('Nama'))
    await user.paste('A'.repeat(101))
    await user.click(screen.getByLabelText('Email'))
    await user.paste('e'.repeat(101))
    await user.click(screen.getByLabelText('Telp'))
    await user.paste('1'.repeat(51))
    await user.click(screen.getByLabelText('Alamat'))
    await user.paste('x'.repeat(251))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    // maxLength memotong ketikan, sehingga yang terkirim tetap di dalam batas.
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
    expect(onSave.mock.calls[0]?.[0]).toMatchObject({ telp: '1'.repeat(50) })
  })

  it('mengunci tombol selama menyimpan dan mengabaikan galat yang bukan galat API', () => {
    render(
      <SurveyorLoginForm
        editing={null}
        isSaving
        error={new Error('lain')}
        onSave={() => {}}
        onCancel={() => {}}
      />,
    )

    expect(screen.getByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Batal' })).toBeDisabled()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('menampilkan login ganda sebagai kotak pesan sekaligus sorotan', () => {
    render(
      <SurveyorLoginForm
        editing={null}
        isSaving={false}
        error={
          new APIError('kunci_login_surveyor_sudah_ada', 'Login sudah ada.', 409, [
            { field: 'nama', pesan: 'Login BudiHartono sudah dipakai.' },
          ])
        }
        onSave={() => {}}
        onCancel={() => {}}
      />,
    )

    expect(screen.getByText('Login itu sudah terdaftar')).toBeInTheDocument()
    expect(screen.getByText('Login BudiHartono sudah dipakai.')).toBeInTheDocument()
  })
})
