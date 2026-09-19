import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoutes } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

// Daftar portal dipanggil beranda begitu sesi terbit; peladen tiruan harus
// menjawabnya, kalau tidak layar beranda berhenti di keadaan memuat.
const PORTAL_LIST = {
  portal: [
    { id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true },
    { id: '202600103', nama: 'SINARMAS ASURANSI SYARIAH', alias: 'SMAS', siap: false },
  ],
  utama: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

// stubFetch memasang peladen tiruan. Permintaan /api/portal dijawab otomatis supaya
// setiap uji tidak perlu mengulang daftar portal yang sama.
function stubFetch(answer: (url: string) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    return Promise.resolve(answer(url))
  })
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function mount() {
  const apiClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={apiClient}>
      <MemoryRouter initialEntries={['/masuk']}>
        <AppRoutes />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function fillAndSubmit(username: string, password: string) {
  const pengguna = userEvent.setup()
  await pengguna.type(screen.getByLabelText('Nama pengguna'), username)
  await pengguna.type(screen.getByLabelText('Kata sandi'), password)
  await pengguna.click(screen.getByRole('button', { name: 'Masuk' }))
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().cleanup()
  useSelectedPortal.getState().cleanup()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('layar masuk', () => {
  it('menampilkan beranda setelah kredensial diterima', async () => {
    stubFetch(() =>
      jsonResponse(200, {
        token: 'token-contoh',
        tipe_token: 'Bearer',
        berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        pengguna: SAMPLE_PROFILE,
      }),
    )
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')

    await waitFor(() => expect(screen.getByText('Selamat datang, Contoh Administrator.')).toBeInTheDocument())
    expect(screen.getByText(SAMPLE_PROFILE.nama)).toBeInTheDocument()
  })

  // Ketiga jenis galat menampilkan pesan BERBEDA dan dapat ditindaklanjuti
  // (TKT-U1-002, acceptance criteria pertama).
  it.each([
    {
      nama: 'kredensial salah',
      status: 401,
      kode: 'kredensial_salah',
      title: 'Nama pengguna atau kata sandi salah',
    },
    {
      nama: 'pengguna tidak aktif',
      status: 403,
      kode: 'pengguna_tidak_aktif',
      title: 'Akun Anda tidak aktif',
    },
    {
      nama: 'sistem identitas tidak dapat dihubungi',
      status: 503,
      kode: 'sistem_identitas_tidak_terhubung',
      title: 'Sistem identitas sedang tidak dapat dihubungi',
    },
  ])('membedakan galat $nama', async ({ status, kode, title }) => {
    stubFetch(() => jsonResponse(status, { kode, pesan: 'pesan dari server' }))
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')

    const warning = await screen.findByRole('alert')
    expect(warning).toHaveTextContent(title)
  })

  // Pesan untuk pengguna yang tidak ada dan pengguna yang ada berkata sandi salah harus
  // SAMA — membedakannya membocorkan siapa yang punya akun.
  it('tidak membocorkan keberadaan akun', async () => {
    stubFetch(() =>
      jsonResponse(401, { kode: 'kredensial_salah', pesan: 'Nama pengguna atau kata sandi salah.' }),
    )

    const first = mount()
    await fillAndSubmit('tidakpernahada', 'apa saja')
    const firstMessage = (await screen.findByRole('alert')).textContent
    first.unmount()

    mount()
    await fillAndSubmit('adminpnc', 'sandi salah')
    const secondMessage = (await screen.findByRole('alert')).textContent

    expect(secondMessage).toBe(firstMessage)
  })

  it('menampilkan pesan tersendiri ketika server tidak dapat dihubungi', async () => {
    vi.stubGlobal('fetch', () => Promise.reject(new TypeError('network down')))
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')

    const warning = await screen.findByRole('alert')
    expect(warning).toHaveTextContent('Server Claim PNC tidak dapat dihubungi')
  })

  it('menolak kirim bila isian kosong, tanpa memanggil server', async () => {
    stubFetch(() => jsonResponse(200, {}))
    mount()

    await userEvent.setup().click(screen.getByRole('button', { name: 'Masuk' }))

    expect(await screen.findByText('Nama pengguna wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Kata sandi wajib diisi.')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('tidak pernah menaruh kata sandi maupun token di URL', async () => {
    stubFetch(() =>
      jsonResponse(200, {
        token: 'token-contoh',
        tipe_token: 'Bearer',
        berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        pengguna: SAMPLE_PROFILE,
      }),
    )
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')

    await waitFor(() => expect(calls.length).toBeGreaterThan(0))
    for (const { url } of calls) {
      expect(url).not.toContain('rahasia123')
      expect(url).not.toContain('token-contoh')
    }
    expect(window.location.search).toBe('')
  })

  it('tidak pernah menyimpan kata sandi di peramban', async () => {
    stubFetch(() =>
      jsonResponse(200, {
        token: 'token-contoh',
        tipe_token: 'Bearer',
        berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        pengguna: SAMPLE_PROFILE,
      }),
    )
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')

    await waitFor(() => expect(useSession.getState().token).toBe('token-contoh'))
    expect(JSON.stringify(window.sessionStorage)).not.toContain('rahasia123')
  })

  it('mengirim token sebagai Bearer di header, bukan di tempat lain', async () => {
    stubFetch((url) =>
      url === '/api/masuk'
        ? jsonResponse(200, {
            token: 'token-contoh',
            tipe_token: 'Bearer',
            berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
            pengguna: SAMPLE_PROFILE,
          })
        : new Response(null, { status: 204 }),
    )
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')
    await waitFor(() => expect(screen.getByText('Selamat datang, Contoh Administrator.')).toBeInTheDocument())

    await userEvent.setup().click(screen.getByRole('button', { name: 'Keluar' }))

    await waitFor(() => {
      const logout = calls.find((p) => p.url === '/api/keluar')
      expect(logout).toBeDefined()
      const header = logout?.init?.headers as Record<string, string>
      expect(header['Authorization']).toBe('Bearer token-contoh')
    })
  })

  it('membersihkan sesi di peramban setelah keluar', async () => {
    stubFetch((url) =>
      url === '/api/masuk'
        ? jsonResponse(200, {
            token: 'token-contoh',
            tipe_token: 'Bearer',
            berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
            pengguna: SAMPLE_PROFILE,
          })
        : new Response(null, { status: 204 }),
    )
    mount()
    await fillAndSubmit('adminpnc', 'rahasia123')
    await waitFor(() => expect(screen.getByText('Selamat datang, Contoh Administrator.')).toBeInTheDocument())

    await userEvent.setup().click(screen.getByRole('button', { name: 'Keluar' }))

    await waitFor(() => expect(useSession.getState().token).toBeNull())
    expect(await screen.findByRole('button', { name: 'Masuk' })).toBeInTheDocument()
  })
})
