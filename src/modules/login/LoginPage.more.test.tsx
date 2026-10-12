import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, renderHook, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { LoginPage } from './LoginPage'
import { fetchMe, useExtendSession } from './api'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function newClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function show() {
  return render(
    <QueryClientProvider client={newClient()}>
      <MemoryRouter initialEntries={['/masuk']}>
        <LoginPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function fillAndSubmit() {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Nama pengguna'), 'adminpnc')
  await user.type(screen.getByLabelText('Kata sandi'), 'rahasia123')
  await user.click(screen.getByRole('button', { name: 'Masuk' }))
}

beforeEach(() => {
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('layar masuk — cabang galat tambahan', () => {
  // Kode galat yang tidak dikenali layar jatuh ke pesan umum yang menyebut administrator.
  it('menampilkan pesan umum untuk kode galat API yang tidak dikenal', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(jsonResponse(500, { kode: 'galat_internal', pesan: 'x' })),
    )
    show()
    await fillAndSubmit()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Terjadi kesalahan pada sistem')
    expect(alert).toHaveTextContent('Bila berulang, hubungi administrator Claim PNC.')
  })

  // Galat yang bukan APIError maupun NetworkError (mis. jawaban yang rusak) tetap
  // menghasilkan pesan umum, tanpa menyebut administrator.
  it('menampilkan pesan umum untuk galat yang bukan galat API', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve({
        get status(): number {
          throw new TypeError('jawaban rusak')
        },
      }),
    )
    show()
    await fillAndSubmit()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Terjadi kesalahan pada sistem')
    expect(alert).toHaveTextContent('Coba beberapa saat lagi.')
    expect(alert).not.toHaveTextContent('administrator')
  })

  it('menampilkan pemutar dan menonaktifkan tombol selama permintaan berjalan', async () => {
    let finish: (r: Response) => void = () => {}
    vi.stubGlobal(
      'fetch',
      () =>
        new Promise<Response>((resolve) => {
          finish = resolve
        }),
    )
    show()
    await fillAndSubmit()

    const button = await screen.findByRole('button', { name: /Memeriksa/ })
    expect(button).toBeDisabled()
    expect(button.querySelector('svg.animate-spin')).not.toBeNull()

    // Permintaan diselesaikan supaya tidak ada pekerjaan async yang tertinggal.
    act(() => finish(jsonResponse(401, { kode: 'kredensial_salah', pesan: 'x' })))
    expect(await screen.findByRole('alert')).toHaveTextContent('Nama pengguna atau kata sandi salah')
  })
})

describe('useExtendSession', () => {
  it('memperbarui batas berlaku sesi dari jawaban server', async () => {
    useSession.getState().login({
      token: 'token-lama',
      user: SAMPLE_PROFILE,
      validUntil: '2026-01-01T00:00:00Z',
    })
    const calls: { url: string; init: RequestInit | undefined }[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      return Promise.resolve(jsonResponse(200, { berlaku_sampai: '2026-12-31T00:00:00Z' }))
    })
    const client = newClient()
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )

    const { result } = renderHook(() => useExtendSession(), { wrapper })
    await act(async () => {
      await result.current.mutateAsync()
    })

    expect(calls[0]?.url).toBe('/api/sesi/perpanjang')
    expect(calls[0]?.init?.method).toBe('POST')
    expect((calls[0]?.init?.headers as Record<string, string>)['Authorization']).toBe(
      'Bearer token-lama',
    )
    expect(useSession.getState().validUntil).toBe('2026-12-31T00:00:00Z')
  })
})

describe('fetchMe', () => {
  it('memanggil /api/saya dengan token Bearer dan mengembalikan isinya', async () => {
    const calls: { url: string; init: RequestInit | undefined }[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      return Promise.resolve(jsonResponse(200, { pengguna: SAMPLE_PROFILE }))
    })

    const result = await fetchMe('token-saya')

    expect(result).toEqual({ pengguna: SAMPLE_PROFILE })
    expect(calls[0]?.url).toBe('/api/saya')
    expect((calls[0]?.init?.headers as Record<string, string>)['Authorization']).toBe(
      'Bearer token-saya',
    )
  })
})
