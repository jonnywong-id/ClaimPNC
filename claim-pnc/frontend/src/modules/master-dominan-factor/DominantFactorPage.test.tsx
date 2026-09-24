import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { HEADER_PORTAL } from '@/api/client'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/dominan-factor'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Lima baris contoh yang TEKS-nya dikarang.
 *
 * Isi `POOLDATA.M_DOMINAN_FACTOR` yang sebenarnya belum pernah diterima dari DBA, dan
 * tidak ada satu pun CSV master untuk tabel ini. Yang diuji di berkas ini adalah
 * PERILAKU layar, bukan isi masternya — jadi teks apa pun cukup, asalkan bentuknya benar.
 *
 * Dua hal dipilih dengan sengaja, dan keduanya menguji perilaku nyata:
 *
 *   - ID mencapai dua digit (`10`), supaya pengurutan numerik benar-benar teruji.
 *   - Satu baris ber-keterangan KOSONG, karena modul ini memang menerimanya (keputusan
 *     Work Owner 2026-09-20) sehingga baris seperti itu pasti akan muncul di daftar.
 */
const SAMPLE = [
  { id: '1', nama: 'Contoh Faktor A' },
  { id: '2', nama: 'Contoh Faktor B' },
  { id: '3', nama: '' },
  { id: '9', nama: 'Contoh Faktor I' },
  { id: '10', nama: 'Contoh Faktor J' },
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
      return jsonResponse(201, { dominan_factor: { id: '11', nama: readNama(init) }, portal: 'ASM' })
    }
    if (url.startsWith(`${ROUTES}/`) && init?.method === 'PUT') {
      const id = url.slice(ROUTES.length + 1)
      return jsonResponse(200, { dominan_factor: { id, nama: readNama(init) }, portal: 'ASM' })
    }
    return jsonResponse(200, {
      dominan_factor: SAMPLE,
      total: SAMPLE.length,
      portal: 'ASM',
    })
  })
}

function readNama(init: RequestInit | undefined): string {
  return (JSON.parse(String(init?.body ?? '{}')) as { nama?: string }).nama ?? ''
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/dominan-factor']}>
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
  it('menampilkan ID dan keterangan dari server', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Contoh Faktor A')).toBeInTheDocument()
    expect(screen.getByText('Contoh Faktor J')).toBeInTheDocument()

    // Total datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/5 faktor dominan terdaftar/)).toBeInTheDocument()
  })

  /*
    Keterangan kosong DITERIMA modul ini, sehingga baris seperti itu pasti muncul. Ia
    ditandai, bukan dibiarkan sebagai sel kosong yang terlihat seperti tabel rusak.
  */
  it('menandai baris yang keterangannya kosong', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Faktor A')

    expect(screen.getByText('(tanpa keterangan)')).toBeInTheDocument()
  })

  it('membawa token sesi di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Faktor A')

    const request = calls.find((p) => p.url === ROUTES)
    expect(request).toBeDefined()
    expect(request?.url).not.toContain('token')

    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
  })

  /*
    Portal entitas WAJIB ikut di setiap permintaan. Tanpa header ini backend menolak
    permintaannya — dan itu memang yang diinginkan: jatuh diam-diam ke portal utama
    berarti menampilkan faktor dominan badan hukum lain tanpa satu pun tanda di layar
    (`R-20`).
  */
  it('membawa portal entitas di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Faktor A')

    const request = calls.find((p) => p.url === ROUTES)
    expect(request?.url).not.toContain('ASM')

    const header = request?.init?.headers as Record<string, string>
    expect(header[HEADER_PORTAL]).toBe('ASM')
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Faktor A')

    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('menyaring baris saat pengguna mengetik di kotak cari', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.type(screen.getByRole('searchbox'), 'faktor j')

    expect(screen.getByText('Contoh Faktor J')).toBeInTheDocument()
    expect(screen.queryByText('Contoh Faktor A')).not.toBeInTheDocument()
  })

  it('mencari juga pada kolom ID, bukan hanya keterangan', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.type(screen.getByRole('searchbox'), '10')

    expect(screen.getByText('Contoh Faktor J')).toBeInTheDocument()
    expect(screen.queryByText('Contoh Faktor B')).not.toBeInTheDocument()
  })

  it('menampilkan pesan yang dapat ditindaklanjuti saat pemuatan gagal', async () => {
    installFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' }))
    show()

    expect(await screen.findByRole('alert')).toHaveTextContent(/gagal dimuat/i)
  })
})

describe('tambah', () => {
  it('mengirim POST tanpa ID, lalu memuat ulang daftar', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = screen.getByRole('form', { name: /menambah data/i })
    // ID tidak dapat disunting: ia dibuat sistem, sama seperti di Pega.
    expect(within(form).getByText('Dibuat sistem')).toBeInTheDocument()

    await pengguna.type(within(form).getByLabelText('Keterangan'), 'Faktor Percobaan')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'POST')
      expect(send).toBeDefined()
      expect(send?.url).toBe(ROUTES)
      expect(JSON.parse(String(send?.init?.body))).toEqual({ nama: 'Faktor Percobaan' })
    })

    // Daftar dimuat ulang dari server: nomor baru hanya diketahui server.
    await waitFor(() => {
      expect(calls.filter((p) => p.url === ROUTES && p.init?.method === 'GET').length).toBe(2)
    })
  })

  /*
    Ini kebalikan dari uji yang sama di Master Status Klaim, dan perbedaannya DISENGAJA.

    Layar Pega menerima keterangan kosong (`pyRequired=false`), dan Work Owner memutuskan
    perilaku itu dipertahankan pada 2026-09-20 sesuai `P-5`. Bila uji ini kelak gagal
    karena seseorang menambahkan `.min(1)` pada skemanya, yang perlu dibaca lebih dulu
    adalah keputusannya — bukan uji ini.
  */
  it('menerima keterangan kosong dan tetap mengirimnya ke server', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'POST')
      expect(send).toBeDefined()
      expect(JSON.parse(String(send?.init?.body))).toEqual({ nama: '' })
    })
  })

  /*
    Karena keterangan ganda diterima, pengguna tidak akan pernah ditolak sistem — jadi
    satu-satunya cara ia tahu adalah diberi tahu. Peringatan itu menggantikan pemeriksaan
    yang sengaja tidak ada.
  */
  it('memperingatkan bahwa keterangan ganda tidak ditolak sistem', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })

    expect(within(form).getByText(/boleh terdaftar lebih dari satu kali/i)).toBeInTheDocument()
  })

  it('menampilkan detail validasi per field dari server', async () => {
    installFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [{ field: 'nama', pesan: 'Keterangan paling panjang 100 karakter.' }],
        })
      }
      return jsonResponse(200, { dominan_factor: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })
    await pengguna.type(within(form).getByLabelText('Keterangan'), 'Apa pun')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(/paling panjang 100/i)
  })

  /*
    ID dibentuk `max+1`, sehingga dua penyimpanan yang benar-benar bersamaan dapat
    memperebutkan nomor yang sama. Pesannya harus menuntun pengguna mencoba lagi —
    bukan membuatnya mengira isiannya salah.
  */
  it('menuntun mencoba ulang saat nomor bentrok', async () => {
    installFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(409, {
          kode: 'id_dominan_factor_sudah_dipakai',
          pesan: 'Nomor yang dibuat sistem sudah dipakai.',
        })
      }
      return jsonResponse(200, { dominan_factor: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })

    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = screen.getByRole('form', { name: /menambah data/i })
    await pengguna.type(within(form).getByLabelText('Keterangan'), 'Faktor Baru')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(/simpan sekali lagi/i)
    // Form tetap terbuka supaya isian pengguna tidak hilang.
    expect(within(form).getByLabelText('Keterangan')).toHaveValue('Faktor Baru')
  })
})

describe('ubah', () => {
  it('mengirim PUT dengan ID di jalur URL, bukan di badan permintaan', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    await pengguna.click(screen.getByRole('button', { name: /Ubah faktor dominan Contoh Faktor A/i }))

    const form = screen.getByRole('form', { name: /memperbaharui data/i })
    // ID baris yang sedang diubah ditampilkan, dan tetap tidak dapat disunting.
    expect(within(form).getByText('1')).toBeInTheDocument()

    const isian = within(form).getByLabelText('Keterangan')
    expect(isian).toHaveValue('Contoh Faktor A')

    await pengguna.clear(isian)
    await pengguna.type(isian, 'Nama Diperbarui')
    await pengguna.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const send = calls.find((p) => p.init?.method === 'PUT')
      expect(send).toBeDefined()
      expect(send?.url).toBe(`${ROUTES}/1`)
      // ID TIDAK ikut di badan permintaan: menerimanya dari klien akan membuat satu
      // faktor dapat dipindahkan ke nomor lain, dan memutus klaim yang menyimpannya.
      expect(JSON.parse(String(send?.init?.body))).toEqual({ nama: 'Nama Diperbarui' })
    })
  })

  it('dapat mengubah baris yang keterangannya kosong', async () => {
    installDefaultFetch()
    const pengguna = userEvent.setup()
    show()
    await screen.findByText('Contoh Faktor A')

    // Baris tanpa keterangan tetap punya tombol Ubah; label aksesibilitasnya jatuh ke
    // ID supaya tombolnya tetap dapat disebut pengguna papan ketik dan pembaca layar.
    await pengguna.click(screen.getByRole('button', { name: /Ubah faktor dominan 3/i }))

    const form = screen.getByRole('form', { name: /memperbaharui data/i })
    expect(within(form).getByLabelText('Keterangan')).toHaveValue('')
  })
})

describe('yang sengaja tidak ada', () => {
  /*
    Tidak ada tombol Hapus, dan ketiadaannya diuji — bukan sekadar dicatat di komentar.

    Layar Pega tidak punya tombolnya, `Database/PEGA_M_DOMINAN_FACTOR.prc` hanya mengenal
    INSERT dan UPDATE, dan menghapus satu baris akan membuat setiap klaim yang menyimpan
    ID itu di `T_CLAIM_DOMINANFACTOR` kehilangan artinya (`ADR-0012`).
  */
  it('tidak menyediakan tombol hapus', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Faktor A')

    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})
