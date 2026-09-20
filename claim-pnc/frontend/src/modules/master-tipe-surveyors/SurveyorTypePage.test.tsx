import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/tipe-surveyor'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Keempat tipe surveyor nyata, dibaca dari POOLDATA.V_M_SURVEYORS portal ASM.
 *
 * Kodenya tidak dikarang: tiga di antaranya dipatok langsung di kueri Pega, sehingga
 * memakai kode lain di sini akan membuat uji ini menguji sesuatu yang tidak pernah ada.
 */
const SAMPLE = [
  { kode: '1001', deskripsi: 'INTERNAL SURVEYOR', kode_lama: '' },
  { kode: '1002', deskripsi: 'LOSS ADJUSTER', kode_lama: '' },
  { kode: '1003', deskripsi: 'EXPERT', kode_lama: '' },
  { kode: '1004', deskripsi: 'SURVEY AGENT', kode_lama: '' },
]

/**
 * Bilah atas memuat pemilih portal, sehingga SETIAP layar di balik sesi ikut memanggil
 * /api/portal. Peladen tiruan menjawabnya otomatis supaya tiap uji di berkas ini tidak
 * perlu mengulang daftar yang sama.
 */
const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(reply: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    // Kerangka layar memuat menunya sendiri sejak menu dibaca dari basis data. Ia dijawab
    // di sini supaya uji layar ini menguji layarnya, bukan jalur galat menu.
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(reply(url, init))
  })
}

function readDescription(init: RequestInit | undefined): string {
  return (JSON.parse(String(init?.body ?? '{}')) as { deskripsi?: string }).deskripsi ?? ''
}

/** Peladen tiruan yang menjawab daftar dan menerima simpan. */
function installDefaultFetch() {
  installFetch((url, init) => {
    if (url === ROUTE && init?.method === 'POST') {
      return jsonResponse(201, {
        tipe_surveyor: { kode: '1005', deskripsi: readDescription(init), kode_lama: '' },
        portal: 'ASM',
      })
    }
    if (url.startsWith(`${ROUTE}/`) && init?.method === 'PUT') {
      const kode = url.slice(ROUTE.length + 1)
      return jsonResponse(200, {
        tipe_surveyor: { kode, deskripsi: readDescription(init), kode_lama: '' },
        portal: 'ASM',
      })
    }
    return jsonResponse(200, { tipe_surveyor: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/tipe-surveyor']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  // Layar berada di balik sesi. Tanpa ini SessionGuard melempar ke layar masuk.
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('daftar', () => {
  it('menampilkan kode dan nama tipe dari server', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('INTERNAL SURVEYOR')).toBeInTheDocument()
    expect(screen.getByText('LOSS ADJUSTER')).toBeInTheDocument()
    expect(screen.getByText('1003')).toBeInTheDocument()

    // Total datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/4 tipe surveyor terdaftar/)).toBeInTheDocument()
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('EXPERT')

    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('membawa token dan portal di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('EXPERT')

    const request = calls.find((p) => p.url === ROUTE)
    expect(request).toBeDefined()
    expect(request?.url).not.toContain('token')
    expect(request?.url).not.toContain('ASM')

    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
    expect(header[HEADER_PORTAL]).toBe('ASM')
  })

  /*
    Kolom "Kode lama" tidak ditampilkan bila seluruh barisnya kosong.

    Keempat baris portal ASM memang tidak punya penomoran lama, dan kolom yang selamanya
    berisi tanda hubung hanya menambah lebar tabel tanpa memberi tahu apa pun.
  */
  it('menyembunyikan kolom kode lama saat tidak ada yang mengisinya', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('EXPERT')

    expect(screen.queryByRole('columnheader', { name: /kode lama/i })).not.toBeInTheDocument()
  })

  it('menampilkan kolom kode lama begitu ada baris yang mengisinya', async () => {
    installFetch(() =>
      jsonResponse(200, {
        tipe_surveyor: [{ kode: '1001', deskripsi: 'INTERNAL SURVEYOR', kode_lama: '01' }],
        total: 1,
        portal: 'ASM',
      }),
    )
    show()
    await screen.findByText('INTERNAL SURVEYOR')

    expect(screen.getByRole('columnheader', { name: /kode lama/i })).toBeInTheDocument()
    expect(screen.getByText('01')).toBeInTheDocument()
  })

  it('menyaring baris saat pengguna mengetik di kotak cari', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.type(screen.getByRole('searchbox'), 'adjuster')

    expect(screen.getByText('LOSS ADJUSTER')).toBeInTheDocument()
    expect(screen.queryByText('EXPERT')).not.toBeInTheDocument()
  })

  it('meminta pengguna memilih portal lebih dulu bila belum ada yang dipilih', async () => {
    useSelectedPortal.getState().clear()
    installDefaultFetch()
    show()

    expect(await screen.findByText(/Portal entitas belum dipilih/)).toBeInTheDocument()
    // Dan tidak menembak API-nya sama sekali: layar yang menuntun lebih berguna daripada
    // pesan galat yang dipancing sendiri.
    expect(calls.some((p) => p.url === ROUTE)).toBe(false)
  })
})

describe('menambah', () => {
  it('mengirim deskripsi tanpa kode, lalu menutup form', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.click(screen.getByRole('button', { name: /tambah/i }))
    await pengguna.type(screen.getByLabelText('Tipe Surveyor'), 'ADJUSTER INDEPENDEN')
    await pengguna.click(screen.getByRole('button', { name: /^simpan$/i }))

    await waitFor(() => {
      const kirim = calls.find((p) => p.url === ROUTE && p.init?.method === 'POST')
      expect(kirim).toBeDefined()
      // KODE tidak pernah dikirim klien: pada penambahan ia dibuat server.
      expect(JSON.parse(String(kirim?.init?.body))).toEqual({ deskripsi: 'ADJUSTER INDEPENDEN' })
      expect((kirim?.init?.headers as Record<string, string>)[HEADER_PORTAL]).toBe('ASM')
    })

    await waitFor(() => {
      expect(screen.queryByRole('form', { name: /tambah tipe surveyor/i })).not.toBeInTheDocument()
    })
  })

  it('menolak isian kosong tanpa menembak server', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.click(screen.getByRole('button', { name: /tambah/i }))
    await pengguna.click(screen.getByRole('button', { name: /^simpan$/i }))

    expect(await screen.findByText(/wajib diisi/i)).toBeInTheDocument()
    expect(calls.some((p) => p.init?.method === 'POST')).toBe(false)
  })

  it('menampilkan pesan yang dapat ditindaklanjuti saat namanya sudah dipakai', async () => {
    installFetch((url, init) => {
      if (url === ROUTE && init?.method === 'POST') {
        return jsonResponse(409, {
          kode: 'tipe_surveyor_sudah_ada',
          pesan: 'Tipe surveyor dengan nama itu sudah ada. Pakai nama lain.',
        })
      }
      return jsonResponse(200, { tipe_surveyor: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.click(screen.getByRole('button', { name: /tambah/i }))
    await pengguna.type(screen.getByLabelText('Tipe Surveyor'), 'EXPERT')
    await pengguna.click(screen.getByRole('button', { name: /^simpan$/i }))

    expect(await screen.findByText(/sudah dipakai/i)).toBeInTheDocument()
    // Form TETAP terbuka beserta isiannya: menutupnya akan membuang yang baru diketik.
    expect(screen.getByRole('form', { name: /tambah tipe surveyor/i })).toBeInTheDocument()
  })

  /*
    Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas.

    Server mengirim SELURUH pelanggaran sekaligus (P-5), dan itu hanya berguna bila layar
    menyorotnya di tempat isiannya.
  */
  it('menyorot isian yang dilaporkan server sebagai tidak sah', async () => {
    installFetch((url, init) => {
      if (url === ROUTE && init?.method === 'POST') {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'deskripsi', pesan: 'Tipe surveyor paling panjang 100 karakter.' }],
        })
      }
      return jsonResponse(200, { tipe_surveyor: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.click(screen.getByRole('button', { name: /tambah/i }))
    await pengguna.type(screen.getByLabelText('Tipe Surveyor'), 'TIPE BARU')
    await pengguna.click(screen.getByRole('button', { name: /^simpan$/i }))

    expect(await screen.findByText(/paling panjang 100 karakter/i)).toBeInTheDocument()
  })
})

describe('mengubah', () => {
  it('mengirim PUT ke kode baris itu, dan kodenya tidak dapat disunting', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.click(screen.getByRole('button', { name: /ubah tipe surveyor EXPERT/i }))

    // Kode digambar sebagai kotak mati, bukan isian — ia dipatok kueri Pega.
    expect(screen.getByRole('form', { name: /ubah tipe surveyor/i })).toBeInTheDocument()
    expect(screen.queryByLabelText(/^kode$/i)).not.toBeInTheDocument()

    const isian = screen.getByLabelText('Tipe Surveyor')
    await pengguna.clear(isian)
    await pengguna.type(isian, 'TENAGA AHLI')
    await pengguna.click(screen.getByRole('button', { name: /^simpan$/i }))

    await waitFor(() => {
      const kirim = calls.find((p) => p.init?.method === 'PUT')
      expect(kirim).toBeDefined()
      expect(kirim?.url).toBe(`${ROUTE}/1003`)
      expect(JSON.parse(String(kirim?.init?.body))).toEqual({ deskripsi: 'TENAGA AHLI' })
    })
  })

  it('mengisi form dengan nama yang sedang berlaku', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('EXPERT')

    await pengguna.click(screen.getByRole('button', { name: /ubah tipe surveyor LOSS ADJUSTER/i }))

    expect(screen.getByLabelText('Tipe Surveyor')).toHaveValue('LOSS ADJUSTER')
  })
})

describe('kegagalan memuat', () => {
  it('membedakan entitas yang belum siap dari gangguan biasa', async () => {
    installFetch(() =>
      jsonResponse(503, { kode: 'portal_belum_siap', pesan: 'belum tersedia' }),
    )
    show()

    // Diperiksa DI DALAM kotak pesannya, bukan di seluruh halaman: sidebar pun memuat
    // kalimat "Hubungi administrator Claim PNC" ketika menunya kosong, dan pencarian
    // selebar halaman akan menemukan keduanya.
    const kotakPesan = await screen.findByRole('alert')
    expect(within(kotakPesan).getByText(/belum tersedia/i)).toBeInTheDocument()
    expect(within(kotakPesan).getByText(/administrator Claim PNC/i)).toBeInTheDocument()
  })
})
