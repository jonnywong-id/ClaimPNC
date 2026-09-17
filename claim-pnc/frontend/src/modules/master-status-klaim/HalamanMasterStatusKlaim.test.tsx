import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Rute } from '@/app/App'
import { gunakanSesi } from '@/app/sesi'

const JALUR = '/api/master/status-klaim'

const PROFIL_CONTOH = {
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
const CONTOH = [
  { kode: '1134', label: 'Abbreviated Report', kode_lama: '01' },
  { kode: '1142', label: 'Rejected Claim', kode_lama: '09' },
  { kode: '1147', label: 'Register', kode_lama: '' },
  { kode: '1163', label: 'Paid', kode_lama: '' },
  { kode: '1166', label: 'LOD Accepted', kode_lama: '' },
]

/**
 * Bilah atas memuat pemilih portal, sehingga SETIAP layar di balik sesi ikut memanggil
 * /api/portal. Peladen tiruan menjawabnya otomatis supaya tiap uji di berkas ini tidak
 * perlu mengulang daftar yang sama — pola yang sama dipakai HalamanMasuk.test.tsx.
 */
const DAFTAR_PORTAL = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

type Panggilan = { url: string; init: RequestInit | undefined }

let panggilan: Panggilan[] = []

function responsJSON(status: number, badan: unknown): Response {
  return new Response(JSON.stringify(badan), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function pasangFetch(jawaban: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    panggilan.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(responsJSON(200, DAFTAR_PORTAL))
    return Promise.resolve(jawaban(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar dan menerima simpan. */
function pasangFetchBaku() {
  pasangFetch((url, init) => {
    if (url === JALUR && init?.method === 'POST') {
      return responsJSON(201, {
        status_klaim: { kode: '1167', label: bacaLabel(init), kode_lama: '' },
      })
    }
    if (url.startsWith(`${JALUR}/`) && init?.method === 'PUT') {
      const kode = url.slice(JALUR.length + 1)
      return responsJSON(200, { status_klaim: { kode, label: bacaLabel(init), kode_lama: '' } })
    }
    return responsJSON(200, { status_klaim: CONTOH, total: CONTOH.length })
  })
}

function bacaLabel(init: RequestInit | undefined): string {
  return (JSON.parse(String(init?.body ?? '{}')) as { label?: string }).label ?? ''
}

function tampilkan() {
  const klien = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={klien}>
      <MemoryRouter initialEntries={['/master/status-klaim']}>
        <Rute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  panggilan = []
  // Layar berada di balik sesi. Tanpa ini PenjagaSesi melempar ke layar masuk.
  gunakanSesi.setState({
    token: 'token-uji',
    pengguna: PROFIL_CONTOH,
    berlakuSampai: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  gunakanSesi.getState().bersihkan()
})

describe('daftar', () => {
  it('menampilkan kode, status, dan kode lama dari server', async () => {
    pasangFetchBaku()
    tampilkan()

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
    pasangFetchBaku()
    tampilkan()
    await screen.findByText('Paid')

    const permintaan = panggilan.find((p) => p.url === JALUR)
    expect(permintaan).toBeDefined()
    expect(permintaan?.url).not.toContain('token')

    const header = permintaan?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
  })

  it('menyaring baris saat pengguna mengetik di kotak cari', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.type(screen.getByRole('searchbox'), 'reject')

    expect(screen.getByText('Rejected Claim')).toBeInTheDocument()
    expect(screen.queryByText('Paid')).not.toBeInTheDocument()
    expect(screen.getByText(/1 dari 5 baris cocok/)).toBeInTheDocument()
  })

  it('mencari juga pada kolom kode, bukan hanya nama status', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.type(screen.getByRole('searchbox'), '1147')

    expect(screen.getByText('Register')).toBeInTheDocument()
    expect(screen.queryByText('Paid')).not.toBeInTheDocument()
  })

  it('memberi tahu saat pencarian tidak menemukan apa pun', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.type(screen.getByRole('searchbox'), 'zzz')

    expect(screen.getByText(/Tidak ada baris yang cocok/)).toBeInTheDocument()
  })

  it('menampilkan pesan yang dapat ditindaklanjuti saat pemuatan gagal', async () => {
    pasangFetch(() => responsJSON(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' }))
    tampilkan()

    expect(await screen.findByRole('alert')).toHaveTextContent(/gagal dimuat/i)
  })
})

describe('tambah', () => {
  it('mengirim POST tanpa kode, lalu memuat ulang daftar', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    // Kode tidak dapat disunting: ia dibuat sistem, sama seperti di Pega.
    expect(within(form).getByText('Dibuat sistem')).toBeInTheDocument()

    await pengguna.type(within(form).getByLabelText('Status'), 'Status Percobaan')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const kirim = panggilan.find((p) => p.init?.method === 'POST')
      expect(kirim).toBeDefined()
      expect(kirim?.url).toBe(JALUR)
      expect(JSON.parse(String(kirim?.init?.body))).toEqual({ label: 'Status Percobaan' })
    })

    // Daftar dimuat ulang dari server: kode baru hanya diketahui server.
    await waitFor(() => {
      expect(panggilan.filter((p) => p.url === JALUR && p.init?.method === 'GET').length).toBe(2)
    })
  })

  it('menolak status kosong tanpa memanggil server', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Status wajib diisi.')).toBeInTheDocument()
    expect(panggilan.some((p) => p.init?.method === 'POST')).toBe(false)
  })

  it('menolak status yang hanya berisi spasi', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /tambah status klaim/i })
    await pengguna.type(within(form).getByLabelText('Status'), '   ')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Status wajib diisi.')).toBeInTheDocument()
    expect(panggilan.some((p) => p.init?.method === 'POST')).toBe(false)
  })

  // Nama ganda hanya dapat diketahui server; layar harus menampilkan penolakannya
  // dengan jelas, bukan gagal diam-diam.
  it('menampilkan penolakan nama ganda dari server', async () => {
    pasangFetch((_url, init) => {
      if (init?.method === 'POST') {
        return responsJSON(409, {
          kode: 'label_status_sudah_dipakai',
          pesan: 'Status dengan nama itu sudah ada. Pakai nama lain.',
        })
      }
      return responsJSON(200, { status_klaim: CONTOH, total: CONTOH.length })
    })

    const pengguna = userEvent.setup()
    tampilkan()
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
    pasangFetch((_url, init) => {
      if (init?.method === 'POST') {
        return responsJSON(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'label', pesan: 'Status paling panjang 100 karakter.' }],
        })
      }
      return responsJSON(200, { status_klaim: CONTOH, total: CONTOH.length })
    })

    const pengguna = userEvent.setup()
    tampilkan()
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
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
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
      const kirim = panggilan.find((p) => p.init?.method === 'PUT')
      expect(kirim).toBeDefined()
      expect(kirim?.url).toBe(`${JALUR}/1163`)
      expect(JSON.parse(String(kirim?.init?.body))).toEqual({ label: 'Sudah Dibayar' })
    })
  })

  it('menutup form setelah tersimpan', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Ubah status Paid' }))
    const form = screen.getByRole('form', { name: /ubah status klaim/i })
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(screen.queryByRole('form', { name: /ubah status klaim/i })).not.toBeInTheDocument()
    })
  })

  it('membatalkan tanpa memanggil server', async () => {
    pasangFetchBaku()
    const pengguna = userEvent.setup()
    tampilkan()
    await screen.findByText('Paid')

    await pengguna.click(screen.getByRole('button', { name: 'Ubah status Paid' }))
    await pengguna.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('form', { name: /ubah status klaim/i })).not.toBeInTheDocument()
    expect(panggilan.some((p) => p.init?.method === 'PUT')).toBe(false)
  })
})

describe('yang sengaja tidak ada', () => {
  // Layar Pega tidak punya tombol hapus, dan ADR-0012 melarang master dihapus permanen
  // karena klaim lama merujuknya. Uji ini mengunci ketiadaan itu.
  it('tidak menyediakan tombol hapus', async () => {
    pasangFetchBaku()
    tampilkan()
    await screen.findByText('Paid')

    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})
