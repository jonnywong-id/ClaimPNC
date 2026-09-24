import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { HEADER_PORTAL } from '@/api/client'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/status-klaim'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Lima baris contoh, disalin dari Database/v_sts_claim.csv.
 *
 * Sengaja tidak seluruh 33 baris: yang diuji di sini adalah perilaku layar, bukan isi
 * masternya — kelengkapan 33 kode sudah diuji di sisi Go, tempat datanya berasal.
 */
const SAMPLE = [
  { kode: '1134', label: 'Abbreviated Report', kode_lama: '01' },
  { kode: '1142', label: 'Rejected Claim', kode_lama: '09' },
  { kode: '1147', label: 'Register', kode_lama: '' },
  { kode: '1163', label: 'Paid', kode_lama: '' },
  { kode: '1166', label: 'LOD Accepted', kode_lama: '' },
]

/**
 * Bilah atas memuat pemilih portal, sehingga SETIAP layar di balik sesi ikut memanggil
 * /api/portal. Peladen tiruan menjawabnya otomatis supaya tiap uji di berkas ini tidak
 * perlu mengulang daftar yang sama — pola yang sama dipakai LoginPage.test.tsx.
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
    // Kerangka layar memuat menunya sendiri sejak menu dibaca dari basis data. Ia
    // dijawab di sini supaya uji layar ini menguji layarnya, bukan jalur galat menu.
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(reply(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar dan menerima simpan. */
function installDefaultFetch() {
  installFetch((url, init) => {
    if (url === ROUTES && init?.method === 'POST') {
      return jsonResponse(201, {
        status_klaim: { kode: '1167', label: readLabel(init), kode_lama: '' },
      })
    }
    if (url.startsWith(`${ROUTES}/`) && init?.method === 'PUT') {
      const kode = url.slice(ROUTES.length + 1)
      return jsonResponse(200, { status_klaim: { kode, label: readLabel(init), kode_lama: '' } })
    }
    return jsonResponse(200, { status_klaim: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function readLabel(init: RequestInit | undefined): string {
  return (JSON.parse(String(init?.body ?? '{}')) as { label?: string }).label ?? ''
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/status-klaim']}>
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
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('daftar', () => {
  it('menampilkan kode, status, dan kode lama dari server', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Abbreviated Report')).toBeInTheDocument()
    expect(screen.getByText('Paid')).toBeInTheDocument()

    // Total datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/5 status terdaftar/)).toBeInTheDocument()

    // Kode lama hanya melekat pada kode-kode awal; sisanya ditandai, bukan dikosongkan
    // begitu saja sehingga kolomnya terlihat rusak.
    expect(screen.getAllByText('01').length).toBeGreaterThan(0)
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
  })

  it('membawa token sesi di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Paid')

    const request = calls.find((p) => p.url === ROUTES)
    expect(request).toBeDefined()
    expect(request?.url).not.toContain('token')

    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
  })

  /*
    Portal entitas WAJIB ikut di setiap permintaan sejak penyelarasan 2026-09-19.

    Sebelumnya modul ini selalu membaca basis data portal utama. Tanpa header ini backend
    menolak permintaannya — dan itu memang yang diinginkan: jatuh diam-diam ke portal
    utama berarti menampilkan status klaim badan hukum lain tanpa satu pun tanda di layar
    (`R-20`).

    Portalnya tidak di-set uji ini secara langsung: pemilih portal di bilah atas memilih
    portal utama dari server, persis seperti yang dialami pengguna.
  */
  it('membawa portal entitas di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Paid')

    const request = calls.find((p) => p.url === ROUTES)
    expect(request?.url).not.toContain('ASM')

    const header = request?.init?.headers as Record<string, string>
    expect(header[HEADER_PORTAL]).toBe('ASM')
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Paid')

    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('menyaring baris saat pengguna mengetik di kotak cari', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.type(screen.getByRole('searchbox'), 'reject')

    expect(screen.getByText('Rejected Claim')).toBeInTheDocument()
    expect(screen.queryByText('Paid')).not.toBeInTheDocument()
    expect(screen.getByText(/1 dari 5 baris cocok/)).toBeInTheDocument()
  })

  it('mencari juga pada kolom kode, bukan hanya nama status', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.type(screen.getByRole('searchbox'), '1147')

    expect(screen.getByText('Register')).toBeInTheDocument()
    expect(screen.queryByText('Paid')).not.toBeInTheDocument()
  })

  it('memberi tahu saat pencarian tidak menemukan apa pun', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.type(screen.getByRole('searchbox'), 'zzz')

    expect(screen.getByText(/Tidak ada baris yang cocok/)).toBeInTheDocument()
  })

  it('menampilkan pesan yang dapat ditindaklanjuti saat pemuatan gagal', async () => {
    installFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' }))
    show()

    expect(await screen.findByRole('alert')).toHaveTextContent(/gagal dimuat/i)
  })
})

describe('tambah', () => {
  it('mengirim POST tanpa kode, lalu memuat ulang daftar', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    // Kode tidak dapat disunting: ia dibuat sistem, sama seperti di Pega.
    expect(within(form).getByText('Dibuat sistem')).toBeInTheDocument()

    await pengguna.type(within(form).getByLabelText('Status'), 'Status Percobaan')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'POST')
      expect(send).toBeDefined()
      expect(send?.url).toBe(ROUTES)
      expect(JSON.parse(String(send?.init?.body))).toEqual({ label: 'Status Percobaan' })
    })

    // Daftar dimuat ulang dari server: kode baru hanya diketahui server.
    await waitFor(() => {
      expect(calls.filter((p) => p.url === ROUTES && p.init?.method === 'GET').length).toBe(2)
    })
  })

  it('menolak status kosong tanpa memanggil server', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Status wajib diisi.')).toBeInTheDocument()
    expect(calls.some((p) => p.init?.method === 'POST')).toBe(false)
  })

  it('menolak status yang hanya berisi spasi', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    await pengguna.type(within(form).getByLabelText('Status'), '   ')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Status wajib diisi.')).toBeInTheDocument()
    expect(calls.some((p) => p.init?.method === 'POST')).toBe(false)
  })

  // Nama ganda hanya dapat diketahui server; layar harus menampilkan penolakannya
  // dengan jelas, bukan gagal diam-diam.
  it('menampilkan penolakan nama ganda dari server', async () => {
    installFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(409, {
          kode: 'label_status_sudah_dipakai',
          pesan: 'Status dengan nama itu sudah ada. Pakai nama lain.',
        })
      }
      return jsonResponse(200, { status_klaim: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    await pengguna.type(within(form).getByLabelText('Status'), 'Paid')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(/nama status sudah dipakai/i)
    // Form tetap terbuka supaya isian pengguna tidak hilang.
    expect(within(form).getByLabelText('Status')).toHaveValue('Paid')
  })

  it('menampilkan detail validasi per field dari server', async () => {
    installFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'label', pesan: 'Status paling panjang 100 karakter.' }],
        })
      }
      return jsonResponse(200, { status_klaim: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    await pengguna.type(within(form).getByLabelText('Status'), 'Status Percobaan')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(/paling panjang 100 karakter/i)
  })
})

describe('ubah', () => {
  it('mengisi form dengan nilai baris yang dipilih dan mengirim PUT ke kodenya', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Ubah status Paid' }))

    const form = screen.getByRole('form', { name: /ubah status klaim/i })
    expect(within(form).getByLabelText('Status')).toHaveValue('Paid')
    // Kode ditampilkan tetapi tidak dapat disunting.
    expect(within(form).getByText('1163')).toBeInTheDocument()
    expect(within(form).queryByLabelText('Kode')).not.toBeInTheDocument()

    await pengguna.clear(within(form).getByLabelText('Status'))
    await pengguna.type(within(form).getByLabelText('Status'), 'Sudah Dibayar')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'PUT')
      expect(send).toBeDefined()
      expect(send?.url).toBe(`${ROUTES}/1163`)
      expect(JSON.parse(String(send?.init?.body))).toEqual({ label: 'Sudah Dibayar' })
    })
  })

  it('menutup form setelah tersimpan', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Ubah status Paid' }))
    const form = screen.getByRole('form', { name: /ubah status klaim/i })
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(screen.queryByRole('form', { name: /ubah status klaim/i })).not.toBeInTheDocument()
    })
  })

  it('membatalkan tanpa memanggil server', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Ubah status Paid' }))
    await pengguna.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('form', { name: /ubah status klaim/i })).not.toBeInTheDocument()
    expect(calls.some((p) => p.init?.method === 'PUT')).toBe(false)
  })
})

describe('yang sengaja tidak ada', () => {
  // Layar Pega tidak punya tombol hapus, dan ADR-0012 melarang master dihapus permanen
  // karena klaim lama merujuknya. Uji ini mengunci ketiadaan itu.
  it('tidak menyediakan tombol hapus', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Paid')

    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})
