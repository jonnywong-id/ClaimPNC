import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/pic-teknik'
const DIRECTORY = `${ROUTE}/direktori`

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Contoh yang dipakai peladen tiruan.
 *
 * Isinya dikarang — isi MST_USER_TEKNIK tidak ada di export — dan memakai domain
 * `example.invalid` yang memang dicadangkan supaya tidak mungkin tertukar dengan pegawai
 * sungguhan.
 *
 * Seluruhnya AKTIF, karena itulah yang dikirim server: penyaring `STS_AKTIF = '1'` ada di
 * kueri, bukan di layar.
 */
const SAMPLE = [
  {
    id_operator: 'PICTEKNIK01',
    nama: 'Contoh Kepala Teknik',
    email: 'contoh.kepalateknik@example.invalid',
    lini_bisnis: 'NONMBU',
    grup: 'TEKNIK JAKARTA',
    atasan: '',
    kuota: 20,
    kuota_luar: 0,
    beban_kerja: 6,
    grup_panel: '',
    aktif: true,
  },
  {
    id_operator: 'PICTEKNIK03',
    nama: 'Contoh Petugas Teknik',
    email: 'contoh.petugas@example.invalid',
    lini_bisnis: 'NONMBU',
    grup: 'TEKNIK SURABAYA',
    atasan: 'PICTEKNIK01',
    kuota: 10,
    kuota_luar: 0,
    beban_kerja: 10,
    grup_panel: '',
    aktif: true,
  },
]

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

function body(init: RequestInit | undefined): Record<string, unknown> {
  return JSON.parse(String(init?.body ?? '{}')) as Record<string, unknown>
}

/** Peladen tiruan yang menjawab daftar, pencarian direktori, dan simpan. */
function installDefaultFetch() {
  installFetch((url, init) => {
    if (url.startsWith(DIRECTORY)) {
      return jsonResponse(200, {
        pegawai: {
          id_operator: 'PICTEKNIK05',
          nama: 'Contoh Petugas Baru',
          email: 'contoh.baru@example.invalid',
          atasan: 'PICTEKNIK01',
          nama_atasan: 'Contoh Kepala Teknik',
        },
        portal: 'ASM',
      })
    }
    if (url === ROUTE && init?.method === 'POST') {
      const sent = body(init)
      return jsonResponse(201, {
        pic_teknik: { ...SAMPLE[0], ...sent, nama: 'Contoh Petugas Baru', beban_kerja: 0 },
        portal: 'ASM',
      })
    }
    if (url.startsWith(`${ROUTE}/`) && init?.method === 'PUT') {
      const sent = body(init)
      return jsonResponse(200, {
        pic_teknik: { ...SAMPLE[0], ...sent },
        portal: 'ASM',
      })
    }
    return jsonResponse(200, { pic_teknik: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/pic-teknik']}>
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
  it('menampilkan petugas dari server', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Contoh Kepala Teknik')).toBeInTheDocument()
    // getAllBy, bukan getBy: PICTEKNIK01 muncul dua kali dengan sengaja — sebagai ID baris
    // pertama, dan sebagai ATASAN baris kedua. Itulah bentuk kolom atasan yang memang
    // menyimpan ID, bukan nama.
    expect(screen.getAllByText('PICTEKNIK01').length).toBeGreaterThan(0)
    expect(screen.getByText('contoh.petugas@example.invalid')).toBeInTheDocument()

    // Total datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/2 petugas aktif/)).toBeInTheDocument()
  })

  // Keputusan "daftar hanya menampilkan yang aktif" harus TERBACA pengguna, bukan hanya
  // berlaku diam-diam — kalau tidak, petugas yang dinonaktifkan akan dikira terhapus.
  it('menjelaskan bahwa petugas nonaktif tidak ditampilkan', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    expect(screen.getByText(/Petugas nonaktif tidak ditampilkan/)).toBeInTheDocument()
  })

  it('menandai petugas yang bebannya sudah mencapai kuota', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Petugas Teknik')

    // Ditandai teks, bukan warna saja — warna sendiri tidak terbaca semua orang.
    expect(screen.getByText('penuh')).toBeInTheDocument()
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('membawa token dan portal di header, bukan di URL', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    const request = calls.find((p) => p.url === ROUTE)
    expect(request).toBeDefined()
    expect(request?.url).not.toContain('token')
    expect(request?.url).not.toContain('ASM')

    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
    expect(header[HEADER_PORTAL]).toBe('ASM')
  })

  /*
    Kolom "Kuota sistem lain" tidak ditampilkan bila seluruh barisnya nol — kolom yang
    selamanya kosong hanya menambah lebar tabel tanpa memberi tahu apa pun.
  */
  it('menyembunyikan kolom kuota sistem lain saat seluruhnya nol', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    expect(screen.queryAllByText('Kuota sistem lain')).toHaveLength(0)
  })

  it('menampilkan kolom kuota sistem lain saat ada yang mengisinya', async () => {
    installFetch(() =>
      jsonResponse(200, {
        pic_teknik: [{ ...SAMPLE[0], kuota_luar: 3 }],
        total: 1,
        portal: 'ASM',
      }),
    )
    show()
    await screen.findByText('Contoh Kepala Teknik')

    // getAllBy: DataTable menggambar judul kolom dua kali — sekali untuk tabel, sekali
    // untuk tata letak kartu pada layar sempit (`D-12`).
    expect(screen.getAllByText('Kuota sistem lain').length).toBeGreaterThan(0)
  })

  // Tidak ada Hapus — layar Pega pun tidak punya, dan menghapus petugas memutus rujukan
  // penugasan pada klaim lama.
  it('tidak menyediakan tombol hapus', async () => {
    installDefaultFetch()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menuntun memilih portal sebelum menembak server', async () => {
    installDefaultFetch()
    useSelectedPortal.getState().clear()
    show()

    expect(await screen.findByText(/Portal entitas belum dipilih/)).toBeInTheDocument()
    // Permintaan daftar TIDAK dikirim: layar yang menuntun lebih berguna daripada pesan
    // galat dari server.
    expect(calls.find((p) => p.url === ROUTE)).toBeUndefined()
  })
})

describe('menambah', () => {
  it('mengisi nama dan atasan dari hasil pencarian direktori', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('ID Operator'), 'PICTEKNIK05')
    await user.click(screen.getByRole('button', { name: /Cari pegawai di direktori/i }))

    // Nama datang dari direktori dan tidak dapat diketik.
    expect(await screen.findByText('Contoh Petugas Baru')).toBeInTheDocument()
    // Atasan diusulkan dari blok EmpLeader respons direktori.
    await waitFor(() => {
      expect(screen.getByLabelText('Atasan')).toHaveValue('PICTEKNIK01')
    })
    expect(screen.getByText(/Menurut direktori:/)).toBeInTheDocument()
  })

  it('tidak pernah mengirim nama ke server', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('ID Operator'), 'PICTEKNIK05')
    await user.click(screen.getByRole('button', { name: /Cari pegawai di direktori/i }))
    await screen.findByText('Contoh Petugas Baru')

    // Dikosongkan lebih dulu: pencarian sudah mengusulkan surel dari direktori, dan
    // mengetik di atasnya akan merangkai dua alamat menjadi satu.
    const email = screen.getByLabelText('Email')
    await user.clear(email)
    await user.type(email, 'baru@example.invalid')
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    await waitFor(() => {
      expect(calls.find((p) => p.url === ROUTE && p.init?.method === 'POST')).toBeDefined()
    })

    const sent = body(calls.find((p) => p.url === ROUTE && p.init?.method === 'POST')?.init)
    // Ketiganya dimiliki server, bukan layar. Server bahkan menolak badan yang memuatnya.
    expect(sent).not.toHaveProperty('nama')
    expect(sent).not.toHaveProperty('grup_panel')
    expect(sent).not.toHaveProperty('beban_kerja')
    expect(sent.id_operator).toBe('PICTEKNIK05')
  })

  // Surel diusulkan dari direktori supaya petugas tidak mengetik ulang, tetapi ia tetap
  // MILIK master ini — di sistem lama pun surel diketik, bukan diturunkan.
  it('mengusulkan surel dari direktori saat isiannya masih kosong', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('ID Operator'), 'PICTEKNIK05')
    await user.click(screen.getByRole('button', { name: /Cari pegawai di direktori/i }))

    await waitFor(() => {
      expect(screen.getByLabelText('Email')).toHaveValue('contoh.baru@example.invalid')
    })
  })

  it('menolak email yang kosong sebelum menembak server', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('ID Operator'), 'PICTEKNIK05')
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    expect(await screen.findByText('Email wajib diisi.')).toBeInTheDocument()
    expect(calls.find((p) => p.init?.method === 'POST')).toBeUndefined()
  })

  // Katalog yang belum diisi dibedakan dari jaringan yang putus: yang ini tidak akan pulih
  // sendiri, dan pesannya harus mengatakan itu alih-alih menyuruh mencoba lagi.
  it('mengatakan mengulang tidak menolong saat layanan direktori belum terdaftar', async () => {
    installFetch((url) => {
      if (url.startsWith(DIRECTORY)) {
        return jsonResponse(503, {
          kode: 'direktori_pegawai_belum_terdaftar',
          pesan: 'Layanan pencarian pegawai belum terdaftar untuk entitas ini.',
        })
      }
      return jsonResponse(200, { pic_teknik: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('ID Operator'), 'PICTEKNIK05')
    await user.click(screen.getByRole('button', { name: /Cari pegawai di direktori/i }))

    expect(
      await screen.findByText(/Layanan pencarian pegawai belum terdaftar/),
    ).toBeInTheDocument()
    expect(screen.getByText(/Mengulang tidak akan menolong/)).toBeInTheDocument()
  })

  it('menyorot kolom ID saat pegawai tidak terdaftar di direktori', async () => {
    installFetch((url) => {
      if (url.startsWith(DIRECTORY)) {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [
            { field: 'id_operator', pesan: 'ID operator tidak terdaftar di direktori pegawai.' },
          ],
        })
      }
      return jsonResponse(200, { pic_teknik: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('ID Operator'), 'TIDAKTERDAFTAR')
    await user.click(screen.getByRole('button', { name: /Cari pegawai di direktori/i }))

    expect(
      await screen.findByText('ID operator tidak terdaftar di direktori pegawai.'),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('ID Operator')).toHaveAttribute('aria-invalid', 'true')
  })
})

describe('mengubah', () => {
  it('mengunci ID operator dan mengirim perubahan lewat PUT', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getAllByRole('button', { name: /^Ubah PIC teknik/i })[0]!)

    // ID digambar sebagai kotak mati; ia tidak lagi berupa isian yang dapat diketik.
    expect(screen.queryByLabelText('ID Operator')).not.toBeInTheDocument()
    expect(screen.getByText(/Tidak dapat diubah/)).toBeInTheDocument()

    const group = screen.getByLabelText('Grup')
    await user.clear(group)
    await user.type(group, 'TEKNIK BANDUNG')
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    await waitFor(() => {
      expect(calls.find((p) => p.init?.method === 'PUT')).toBeDefined()
    })

    const request = calls.find((p) => p.init?.method === 'PUT')
    expect(request?.url).toBe(`${ROUTE}/PICTEKNIK01`)
    expect(body(request?.init).grup).toBe('TEKNIK BANDUNG')
  })

  // Peringatan diberikan SEBELUM disimpan. Tanpa itu, petugas yang dinonaktifkan tampak
  // seperti terhapus dan pengguna akan melaporkannya sebagai kehilangan data.
  it('memperingatkan bahwa menonaktifkan menghilangkan baris dari daftar', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getAllByRole('button', { name: /^Ubah PIC teknik/i })[0]!)

    expect(screen.getByText(/hilang dari daftar/)).toBeInTheDocument()
    expect(screen.getByText(/Datanya tidak dihapus/)).toBeInTheDocument()
  })

  it('menampilkan beban kerja sebagai keterangan, bukan isian', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findByText('Contoh Kepala Teknik')

    await user.click(screen.getAllByRole('button', { name: /^Ubah PIC teknik/i })[0]!)

    expect(screen.getByText('6 pekerjaan berjalan')).toBeInTheDocument()
    expect(screen.queryByLabelText('Beban Kerja')).not.toBeInTheDocument()
  })
})
