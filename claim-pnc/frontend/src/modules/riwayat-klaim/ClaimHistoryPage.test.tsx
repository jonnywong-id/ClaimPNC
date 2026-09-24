import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { ClaimHistory, OpenResponse, SearchType } from './types'

const PATH = '/api/riwayat-klaim'
const OPEN_PATH = `${PATH}/buka`

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Tipe pencarian contoh — cukup mewakili keempat bentuk isian yang mungkin. */
const SEARCH_TYPES: SearchType[] = [
  {
    kode: '2',
    label: 'Nama Customer',
    pakai_teks: true,
    pakai_tanggal_pencarian: false,
    pakai_tanggal_lahir: false,
    kolom_tambahan: [],
    tersedia: true,
  },
  {
    kode: '6',
    label: 'Tgl Kejadian',
    pakai_teks: false,
    pakai_tanggal_pencarian: true,
    pakai_tanggal_lahir: false,
    kolom_tambahan: [],
    tersedia: true,
  },
  {
    kode: '9',
    label: 'Tanggal Lahir',
    pakai_teks: false,
    pakai_tanggal_pencarian: false,
    pakai_tanggal_lahir: true,
    kolom_tambahan: ['nama_objek', 'tanggal_lahir'],
    tersedia: true,
  },
  {
    kode: '12',
    label: 'No Rekening',
    pakai_teks: true,
    pakai_tanggal_pencarian: true,
    pakai_tanggal_lahir: false,
    kolom_tambahan: [],
    tersedia: false,
    alasan_belum_tersedia: 'Menunggu API pengganti DB Link ke basis data pembayaran (R-03).',
  },
]

const OPENED: OpenResponse = {
  tipe_pencarian: SEARCH_TYPES,
  proteksi: { jatah_total: 50, jatah_terpakai: 1, jatah_sisa: 49 },
  portal: 'ASM',
}

/**
 * Klaim contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const CLAIMS: ClaimHistory[] = [
  {
    referensi: 'ASM-FW-GCNMFW-WORK PNC-9001',
    nomor_klaim: 'PNC-9001',
    nomor_polis: 'POL-CONTOH-0001',
    nama_tertanggung: 'PT SUMBER CONTOH SENTOSA',
    tanggal_kejadian: '2026-03-12',
    bisnis: 'Fire / Property',
    cabang: 'Cabang Contoh Pusat',
    status: 'Resolved-Completed',
    posisi_klaim: 'Paid',
    tanggal_close: '2026-05-02',
    catatan_close: 'Klaim selesai dibayar penuh.',
    pic_teknis: 'PIC Teknik Contoh A',
    nomor_akseptasi: '',
    nomor_balai_lelang: '',
    nama_objek: '',
    tanggal_lahir: null,
  },
]

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

/** Peladen tiruan yang meluluskan gerbang dan menjawab satu baris hasil. */
function stubDefaultFetch() {
  stubFetch((url) => {
    if (url === OPEN_PATH) return jsonResponse(200, OPENED)
    return jsonResponse(200, {
      klaim: CLAIMS,
      halaman: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      proteksi: OPENED.proteksi,
      portal: 'ASM',
    })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/riwayat-klaim']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderOpened menggambar layar lalu MENUNGGU gerbang selesai.
 *
 * Penantiannya pada salah satu pilihan dropdown, bukan pada labelnya: label "Tipe
 * Pencarian" sudah ada sejak penggambaran pertama, sementara isinya baru tiba bersama
 * jawaban gerbang. Menunggu label saja membuat uji menekan dropdown yang masih kosong.
 */
async function renderOpened(): Promise<HTMLElement> {
  renderPage()
  await screen.findByRole('option', { name: 'Nama Customer' })
  return screen.getByLabelText('Tipe Pencarian')
}

function openCalls(): Call[] {
  return calls.filter((c) => c.url === OPEN_PATH)
}

function lastSearchCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url.startsWith(`${PATH}?`))
}

beforeEach(() => {
  calls = []
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

describe('gerbang proteksi data', () => {
  it('menjalankan gerbang sekali saat layar dibuka, dan menyebut sisa jatah', async () => {
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText(/Sisa jatah pencarian Anda/)).toBeInTheDocument()
    expect(screen.getByText('49')).toBeInTheDocument()

    // SATU kali, bukan dua. Membuka layar memakai jatah, dan `StrictMode` menjalankan
    // efek dua kali di pengembangan — itulah sebabnya pembukaan memakai useQuery yang
    // hasilnya di-cache, bukan useMutation di dalam useEffect.
    expect(openCalls()).toHaveLength(1)
    expect(openCalls()[0]?.init?.method).toBe('POST')
  })

  it('menutup seluruh layar bila pengguna belum terdaftar', async () => {
    stubFetch((url) => {
      if (url === OPEN_PATH) {
        return jsonResponse(403, {
          kode: 'proteksi_belum_terdaftar',
          pesan: 'belum terdaftar',
        })
      }
      return jsonResponse(200, {})
    })
    renderPage()

    expect(
      await screen.findByText('Anda belum terdaftar di Master Proteksi Data'),
    ).toBeInTheDocument()
    // Formulir pencarian tidak digambar sama sekali: gerbang menutup layar, bukan
    // sekadar tombolnya.
    expect(screen.queryByLabelText('Tipe Pencarian')).not.toBeInTheDocument()
  })

  it('membedakan jatah habis dari belum terdaftar', async () => {
    stubFetch((url) => {
      if (url === OPEN_PATH) {
        return jsonResponse(409, { kode: 'jatah_pencarian_habis', pesan: 'habis' })
      }
      return jsonResponse(200, {})
    })
    renderPage()

    expect(
      await screen.findByText('Jatah pencarian data Anda sudah habis'),
    ).toBeInTheDocument()
  })

  it('meminta pengguna memilih portal lebih dulu bila belum ada yang dipilih', async () => {
    useSelectedPortal.getState().clear()
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    // Jatah tidak boleh terpakai oleh permintaan yang sudah pasti ditolak server.
    expect(openCalls()).toHaveLength(0)
  })
})

describe('formulir pencarian', () => {
  it('menggambar isian mengikuti tipe pencarian yang dipilih', async () => {
    stubDefaultFetch()
    const type = await renderOpened()

    // Tipe teks: satu isian teks, tanpa isian tanggal.
    await userEvent.selectOptions(type, '2')
    expect(screen.getByLabelText('Nama Pencarian')).toBeInTheDocument()
    expect(screen.queryByLabelText('Tanggal Pencarian')).not.toBeInTheDocument()

    // Tipe tanggal kejadian: isian "Tanggal Pencarian", bukan yang teks.
    await userEvent.selectOptions(type, '6')
    expect(screen.getByLabelText('Tanggal Pencarian')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nama Pencarian')).not.toBeInTheDocument()

    // Tipe tanggal lahir: isian "Tanggal Lahir" — isian yang BERBEDA dari "Tanggal
    // Pencarian", dan perbedaan itulah yang melahirkan cacat sistem lama.
    await userEvent.selectOptions(type, '9')
    expect(screen.getByLabelText('Tanggal Lahir')).toBeInTheDocument()
    expect(screen.queryByLabelText('Tanggal Pencarian')).not.toBeInTheDocument()
  })

  it('menampilkan DUA isian sekaligus pada tipe No Rekening, seperti layar lama', async () => {
    stubDefaultFetch()
    const type = await renderOpened()
    await userEvent.selectOptions(type, '12')

    expect(screen.getByLabelText('Nama Pencarian')).toBeInTheDocument()
    expect(screen.getByLabelText('Tanggal Pencarian')).toBeInTheDocument()
  })

  it('menolak menjalankan tipe yang belum tersedia, beserta alasannya', async () => {
    stubDefaultFetch()
    const type = await renderOpened()
    await userEvent.selectOptions(type, '12')

    expect(screen.getByText(/Menunggu API pengganti DB Link/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Cari/ })).toBeDisabled()
  })

  it('membersihkan isian saat tipe pencarian diganti', async () => {
    stubDefaultFetch()
    const type = await renderOpened()
    await userEvent.selectOptions(type, '2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'CONTOH')

    await userEvent.selectOptions(type, '12')
    // Isian teks tipe sebelumnya tidak boleh terbawa: nilai sisa yang ikut terkirim
    // adalah kelas cacat yang justru sedang direplikasi di tempat lain, dan ia tidak
    // boleh lahir kembali di tempat yang tidak mewarisinya.
    expect(screen.getByLabelText('Nama Pencarian')).toHaveValue('')
  })
})

describe('hasil pencarian', () => {
  it('tidak menembak server sebelum tombol Cari ditekan', async () => {
    stubDefaultFetch()
    // Ditunggu sampai gerbang SELESAI, bukan sampai labelnya muncul: menunggu label saja
    // membuat uji ini lulus hanya karena ia memeriksa terlalu dini.
    await renderOpened()

    expect(lastSearchCall()).toBeUndefined()
  })

  it('menampilkan hasil beserta kolom grid sistem lama', async () => {
    stubDefaultFetch()
    const type = await renderOpened()
    await userEvent.selectOptions(type, '2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'CONTOH')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(await screen.findByText('PT SUMBER CONTOH SENTOSA')).toBeInTheDocument()
    expect(screen.getByText('PNC-9001')).toBeInTheDocument()
    expect(screen.getByText('Paid')).toBeInTheDocument()

    // Nama kolom TIDAK diterjemahkan — `D-13` menetapkan tampilan meniru Pega.
    expect(screen.getByRole('columnheader', { name: /No Klaim/ })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /Posisi Klaim/ })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: /PIC Teknis/ })).toBeInTheDocument()
  })

  it('mengirim tipe, nilai, token, dan portal pada permintaan pencarian', async () => {
    stubDefaultFetch()
    const type = await renderOpened()
    await userEvent.selectOptions(type, '2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'CONTOH')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    await screen.findByText('PT SUMBER CONTOH SENTOSA')

    const call = lastSearchCall()
    expect(call?.url).toContain('tipe=2')
    expect(call?.url).toContain('nilai=CONTOH')

    const header = new Headers(call?.init?.headers)
    // Token di header, TIDAK pernah di URL: nilai di URL ikut tercatat di log peramban,
    // log proxy, dan header Referer.
    expect(header.get('Authorization')).toBe('Bearer token-uji')
    expect(header.get('X-Portal')).toBe('ASM')
    expect(call?.url).not.toContain('token-uji')
  })

  it('menandai isian yang ditolak server sebagai galat validasi', async () => {
    stubFetch((url) => {
      if (url === OPEN_PATH) return jsonResponse(200, OPENED)
      return jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Isian belum benar.',
        detail: [{ field: 'nilai_pencarian', pesan: 'Isi Nama Customer yang dicari.' }],
      })
    })
    const type = await renderOpened()
    await userEvent.selectOptions(type, '2')
    await userEvent.type(screen.getByLabelText('Nama Pencarian'), 'X')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    expect(await screen.findByText('Isi Nama Customer yang dicari.')).toBeInTheDocument()
  })

  it('menyatakan keterbatasan pencarian Tanggal Lahir di layar', async () => {
    stubFetch((url) => {
      if (url === OPEN_PATH) return jsonResponse(200, OPENED)
      return jsonResponse(200, {
        klaim: [],
        halaman: { halaman: 1, ukuran: 20, total: 0, total_halaman: 1 },
        proteksi: OPENED.proteksi,
        portal: 'ASM',
      })
    })
    const type = await renderOpened()
    await userEvent.selectOptions(type, '9')
    await userEvent.type(screen.getByLabelText('Tanggal Lahir'), '1990-07-17')
    await userEvent.click(screen.getByRole('button', { name: /Cari/ }))

    // Cacat yang direplikasi dinyatakan di layar. Tanpa ini, hasil yang selalu kosong
    // akan dilaporkan berulang kali sebagai kerusakan modul.
    expect(
      await screen.findByText(/hasilnya karena itu selalu kosong/),
    ).toBeInTheDocument()
  })
})
