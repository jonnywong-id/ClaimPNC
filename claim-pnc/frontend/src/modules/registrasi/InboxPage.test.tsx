import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSession } from '@/app/session'

import { InboxPage } from './InboxPage'

// Dua kelompok isi Inbox, sesuai definisi `D-79`: tugas Worklist yang sudah menjadi
// milik pengguna, dan tugas Workbasket yang belum bertuan.
const INBOX = {
  tugas: [
    {
      id: 'tugas-1',
      klaim_id: 'klaim-1',
      nomor_klaim: 'PNCN.26.0001',
      tahap: 'input-register',
      nama_tahap: 'Input Register',
      antrean: 'WORKLIST',
      workbasket: '',
      pemilik: '90000001',
      dapat_diambil: false,
      tindakan_keluar: 'InputRegister',
      dibuat_pada: '2026-06-10T03:00:00Z',
    },
    {
      id: 'tugas-2',
      klaim_id: 'klaim-2',
      nomor_klaim: 'PNCN.26.0002',
      tahap: 'investigator',
      nama_tahap: 'Investigator',
      antrean: 'WORKBASKET',
      workbasket: 'InvestigatorPNC',
      pemilik: '',
      dapat_diambil: true,
      tindakan_keluar: 'InputInvestigator',
      dibuat_pada: '2026-06-10T04:00:00Z',
    },
  ],
}

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

function stubFetch(answer: (url: string) => { body: unknown; status: number }) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : null,
    })
    const { body, status } = answer(url)
    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function mount(component: ReactNode) {
  const apiClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={apiClient}>
      <MemoryRouter>{component}</MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSession.getState().login({
    token: 'token-contoh',
    user: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    validUntil: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('inbox registrasi', () => {
  it('menampilkan kedua kelompok pekerjaan beserta antreannya', async () => {
    stubFetch(() => ({ body: INBOX, status: 200 }))
    mount(<InboxPage />)

    expect(await screen.findByText('PNCN.26.0001')).toBeInTheDocument()
    expect(screen.getByText('PNCN.26.0002')).toBeInTheDocument()
    expect(screen.getByText('Worklist')).toBeInTheDocument()
    expect(screen.getByText(/Workbasket · InvestigatorPNC/)).toBeInTheDocument()
  })

  // Tugas antrean bersama BELUM menjadi pekerjaan siapa pun. Tombolnya harus berbunyi
  // "Ambil", bukan "Kerjakan" — perbedaan yang menentukan apakah petugas tahu ia sedang
  // mengklaim pekerjaan orang lain atau tidak.
  it('membedakan tugas yang harus diambil dari tugas yang sudah menjadi milik pengguna', async () => {
    stubFetch(() => ({ body: INBOX, status: 200 }))
    mount(<InboxPage />)

    const row = await screen.findAllByRole('row')
    // Satu baris kepala + dua tugas.
    expect(row).toHaveLength(3)

    const [, worklistTask, workbasketTask] = row
    expect(worklistTask).toBeDefined()
    expect(workbasketTask).toBeDefined()

    expect(within(worklistTask as HTMLElement).getByRole('link', { name: 'Kerjakan' })).toBeInTheDocument()
    expect(within(workbasketTask as HTMLElement).getByRole('button', { name: 'Ambil' })).toBeInTheDocument()
  })

  it('mengirim token sesi saat memuat inbox', async () => {
    stubFetch(() => ({ body: INBOX, status: 200 }))
    mount(<InboxPage />)

    await waitFor(() => expect(calls.some((p) => p.url === '/api/registrasi/inbox')).toBe(true))
  })

  it('mengambil tugas antrean lewat endpoint yang benar', async () => {
    stubFetch((url) =>
      url.includes('/ambil')
        ? { body: { ...INBOX.tugas[1], pemilik: '90000001', dapat_diambil: false }, status: 200 }
        : { body: INBOX, status: 200 },
    )
    mount(<InboxPage />)

    const button = await screen.findByRole('button', { name: 'Ambil' })
    await userEvent.setup().click(button)

    await waitFor(() =>
      expect(
        calls.some((p) => p.url === '/api/registrasi/tugas/tugas-2/ambil' && p.method === 'POST'),
      ).toBe(true),
    )
  })

  // Dua orang yang menekan "Ambil" pada tugas yang sama adalah kejadian biasa pada
  // antrean bersama. Yang kedua harus diberi tahu, bukan diam-diam kehilangan
  // pekerjaannya.
  it('menjelaskan ketika tugas sudah diambil pengguna lain', async () => {
    stubFetch((url) =>
      url.includes('/ambil')
        ? {
            body: { kode: 'tugas_sudah_diambil', pesan: 'Tugas ini sudah diambil pengguna lain.' },
            status: 409,
          }
        : { body: INBOX, status: 200 },
    )
    mount(<InboxPage />)

    const button = await screen.findByRole('button', { name: 'Ambil' })
    await userEvent.setup().click(button)

    expect(await screen.findByText('Tugas ini sudah diambil pengguna lain.')).toBeInTheDocument()
  })

  it('menampilkan pesan ketika inbox kosong', async () => {
    stubFetch(() => ({ body: { tugas: [] }, status: 200 }))
    mount(<InboxPage />)

    expect(
      await screen.findByText(/Tidak ada pekerjaan yang menunggu/),
    ).toBeInTheDocument()
  })

  it('membuka klaim baru dari nomor polis', async () => {
    stubFetch((url) =>
      url === '/api/registrasi/klaim'
        ? {
            body: {
              klaim: { id: 'klaim-baru', nomor: '', tahap_kini: 'view-polis' },
              tugas: null,
              jalur: null,
              jejak_keputusan: null,
              large_loss: false,
            },
            status: 201,
          }
        : { body: { tugas: [] }, status: 200 },
    )
    mount(<InboxPage />)

    const pengguna = userEvent.setup()
    await pengguna.type(await screen.findByLabelText('Nomor polis'), 'POL-FIRE-0001')
    await pengguna.click(screen.getByRole('button', { name: 'Buka klaim' }))

    await waitFor(() => {
      const request = calls.find((p) => p.url === '/api/registrasi/klaim')
      expect(request).toBeDefined()
      expect(request?.body).toMatchObject({ nomor_polis: 'POL-FIRE-0001' })
    })
  })
})
