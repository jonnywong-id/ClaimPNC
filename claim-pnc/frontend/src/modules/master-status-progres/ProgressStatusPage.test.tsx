import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ProgressStatusPage } from './ProgressStatusPage'

// Kesembilan baris persis seperti yang dijawab server — lihat
// `internal/masterstatusprogres/position.go`. Nilai simpanan dan labelnya memang sama;
// di Pega dropdown-nya mengikat keduanya ke satu properti yang sama. Dan "All" memang
// terulang di baris terakhir.
const POSITIONS = {
  posisi: [
    { kode: 'All', nama: 'All' },
    { kode: 'REGISTER', nama: 'REGISTER' },
    { kode: 'KOMITE', nama: 'KOMITE' },
    { kode: 'SURVEY', nama: 'SURVEY' },
    { kode: 'AKSEPTASI', nama: 'AKSEPTASI' },
    { kode: 'OUTSTANDING', nama: 'OUTSTANDING' },
    { kode: 'BENGKEL', nama: 'BENGKEL' },
    { kode: 'PROCUREMENT', nama: 'PROCUREMENT' },
    { kode: 'All', nama: 'All' },
  ],
}

const LIST = {
  portal: 'ASM',
  status_progres: [
    { id: '01', nama: 'DOKUMEN DITERIMA', kode_posisi: 'REGISTER', nama_posisi: 'REGISTER' },
    { id: '02', nama: 'MENUNGGU JADWAL SURVEI', kode_posisi: 'SURVEY', nama_posisi: 'SURVEY' },
  ],
}

type Call = {
  url: string
  method: string
  body: unknown
  header: Record<string, string>
}

let calls: Call[] = []

/** reply memetakan jalur permintaan ke respons yang dikembalikan fetch tiruan. */
type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
      header: (init?.headers as Record<string, string>) ?? {},
    }
    calls.push(call)

    const { body, status = 200 } = map(call)
    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

/** defaultReply melayani daftar dan posisi; mutasi dijawab pemanggil lewat `mutation`. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/posisi-klaim')) return { body: POSITIONS }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return { body: { status_progres: LIST.status_progres[0], portal: 'ASM' }, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProgressStatusPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function startSession() {
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
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
  startSession()
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('daftar master status progres 1', () => {
  it('menampilkan judul dan ketiga kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Status Progres 1' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(
      within(table).getByRole('columnheader', { name: 'Status Progres' }),
    ).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Posisi' })).toBeInTheDocument()
  })

  it('menampilkan baris beserta label posisinya', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('DOKUMEN DITERIMA')).toBeInTheDocument()
    expect(screen.getByText('MENUNGGU JADWAL SURVEI')).toBeInTheDocument()
    expect(screen.getByText('REGISTER')).toBeInTheDocument()
    expect(screen.getByText('SURVEY')).toBeInTheDocument()
  })

  // Portal WAJIB ikut di setiap permintaan yang menyentuh basis data entitas. Backend
  // menolak yang tidak menyebutkannya, dan itu memang yang diinginkan (R-20).
  it('mengirim header portal dan token pada permintaan daftar', async () => {
    installFetch(defaultReply())
    show()

    await waitFor(() => expect(screen.getByText('DOKUMEN DITERIMA')).toBeInTheDocument())

    const listCall = calls.find(
      (call) => call.url === '/api/master/status-progres-1' && call.method === 'GET',
    )
    expect(listCall).toBeDefined()
    expect(listCall?.header['X-Portal']).toBe('ASM')
    expect(listCall?.header['Authorization']).toBe('Bearer token-contoh')
  })

  // Daftar posisi TIDAK menyertakan portal: ia daftar milik aplikasi, bukan isi basis
  // data entitas mana pun.
  it('tidak mengirim header portal saat meminta daftar posisi', async () => {
    installFetch(defaultReply())
    show()

    await waitFor(() =>
      expect(calls.some((call) => call.url === '/api/master/posisi-klaim')).toBe(true),
    )
    const positionCall = calls.find((call) => call.url === '/api/master/posisi-klaim')
    expect(positionCall?.header['X-Portal']).toBeUndefined()
  })

  it('menyebutkan entitas yang sedang dilihat', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText(/Portal entitas:/)).toBeInTheDocument()
  })

  // Layar tidak menembak API hanya untuk menerima penolakan; ia menuntun pengguna.
  it('meminta pengguna memilih portal lebih dulu bila belum ada yang dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls.some((call) => call.url === '/api/master/status-progres-1')).toBe(false)
  })

  it('membedakan entitas yang belum punya kredensial basis data', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/posisi-klaim')) return { body: POSITIONS }
      return {
        body: { kode: 'portal_belum_siap', pesan: 'belum tersedia' },
        status: 503,
      }
    })
    show()

    expect(await screen.findByText('Basis data entitas ini belum tersedia')).toBeInTheDocument()
  })

  it('menampilkan pesan ketika tabel entitas masih kosong', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/posisi-klaim')) return { body: POSITIONS }
      return { body: { status_progres: [], portal: 'ASM' } }
    })
    show()

    expect(
      await screen.findByText('Belum ada status progres pada entitas ini.'),
    ).toBeInTheDocument()
  })
})

describe('penambahan', () => {
  it('mengirim isian ke server beserta portalnya, tanpa ID', async () => {
    installFetch(defaultReply())
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))

    await user.type(screen.getByLabelText('Status Progres'), 'MENUNGGU BERKAS')
    await user.selectOptions(screen.getByLabelText('Posisi'), 'SURVEY')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'POST')).toBe(true)
    })

    const sent = calls.find((call) => call.method === 'POST')
    expect(sent?.url).toBe('/api/master/status-progres-1')
    expect(sent?.header['X-Portal']).toBe('ASM')
    // ID tidak dikirim: ia diterbitkan server dari isi tabel.
    expect(sent?.body).toEqual({ nama: 'MENUNGGU BERKAS', kode_posisi: 'SURVEY' })
  })

  // Validasi di layar menahan isian kosong sebelum permintaan dikirim.
  it('menolak nama kosong tanpa menembak server', async () => {
    installFetch(defaultReply())
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    await user.selectOptions(screen.getByLabelText('Posisi'), 'REGISTER')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama status progres wajib diisi.')).toBeInTheDocument()
    expect(calls.some((call) => call.method === 'POST')).toBe(false)
  })

  it('menolak posisi yang belum dipilih', async () => {
    installFetch(defaultReply())
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Status Progres'), 'APA SAJA')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Posisi klaim wajib dipilih.')).toBeInTheDocument()
    expect(calls.some((call) => call.method === 'POST')).toBe(false)
  })

  // Server mengirim SELURUH pelanggaran sekaligus (P-5), dan layar menyorotnya pada
  // isiannya masing-masing — bukan meringkasnya menjadi satu pesan.
  it('menyorot setiap isian yang ditolak server', async () => {
    installFetch(
      defaultReply(() => ({
        status: 422,
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          detail: [
            { kolom: 'nama', pesan: 'Nama status progres wajib diisi.' },
            { kolom: 'kode_posisi', pesan: 'Posisi klaim tidak dikenal.' },
          ],
        },
      })),
    )
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Status Progres'), 'X')
    await user.selectOptions(screen.getByLabelText('Posisi'), 'REGISTER')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Posisi klaim tidak dikenal.')).toBeInTheDocument()
    expect(screen.getByText('Nama status progres wajib diisi.')).toBeInTheDocument()
  })

  // Form tetap terbuka saat penyimpanan gagal: menutupnya akan membuang isian yang
  // baru diketik pengguna.
  it('mempertahankan form dan isiannya ketika penyimpanan gagal', async () => {
    installFetch(
      defaultReply(() => ({
        status: 500,
        body: { kode: 'galat_internal', pesan: 'gagal' },
      })),
    )
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Status Progres'), 'MENUNGGU BERKAS')
    await user.selectOptions(screen.getByLabelText('Posisi'), 'SURVEY')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Terjadi kesalahan pada sistem')).toBeInTheDocument()
    expect(screen.getByLabelText('Status Progres')).toHaveValue('MENUNGGU BERKAS')
  })

  it('menutup form setelah penyimpanan berhasil', async () => {
    installFetch(defaultReply())
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Status Progres'), 'MENUNGGU BERKAS')
    await user.selectOptions(screen.getByLabelText('Posisi'), 'SURVEY')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.queryByLabelText('Status Progres')).not.toBeInTheDocument())
  })
})

describe('penyuntingan', () => {
  it('memuat isian baris yang dipilih dan menampilkan ID sebagai tidak dapat diubah', async () => {
    installFetch(defaultReply())
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Ubah DOKUMEN DITERIMA' }))

    expect(screen.getByLabelText('Status Progres')).toHaveValue('DOKUMEN DITERIMA')
    expect(screen.getByLabelText('Posisi')).toHaveValue('REGISTER')
    expect(screen.getByText('(tidak dapat diubah)')).toBeInTheDocument()
    // ID tidak muncul sebagai isian yang dapat disunting.
    expect(screen.queryByLabelText('ID')).not.toBeInTheDocument()
  })

  it('mengirim PUT ke jalur berisi ID, tanpa ID di badan permintaan', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          status_progres: {
            id: '01',
            nama: 'DOKUMEN LENGKAP',
            kode_posisi: 'KOMITE',
            nama_posisi: 'KOMITE',
          },
          portal: 'ASM',
        },
      })),
    )
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Ubah DOKUMEN DITERIMA' }))
    await user.clear(screen.getByLabelText('Status Progres'))
    await user.type(screen.getByLabelText('Status Progres'), 'DOKUMEN LENGKAP')
    await user.selectOptions(screen.getByLabelText('Posisi'), 'KOMITE')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((call) => call.method === 'PUT')).toBe(true))

    const sent = calls.find((call) => call.method === 'PUT')
    expect(sent?.url).toBe('/api/master/status-progres-1/01')
    expect(sent?.header['X-Portal']).toBe('ASM')
    expect(sent?.body).toEqual({ nama: 'DOKUMEN LENGKAP', kode_posisi: 'KOMITE' })
  })

  it('menutup form ketika dibatalkan tanpa mengirim apa pun', async () => {
    installFetch(defaultReply())
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Ubah DOKUMEN DITERIMA' }))
    await user.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByLabelText('Status Progres')).not.toBeInTheDocument()
    expect(calls.some((call) => call.method === 'PUT' || call.method === 'POST')).toBe(false)
  })
})

// Grid layar lama BERHALAMAN — `Section/BrowseStatusProgress-Section.xml` menyisipkan
// `pyGridPaginator` dengan `pyPageSize = Other` dan `pyPageSizeOther = 15`. Versi
// pertama layar ini menggambar seluruh baris sekaligus, dan Work Owner menemukannya pada
// 2026-09-20. Uji ini yang menjaga paginasinya tidak hilang lagi diam-diam.
describe('paginasi', () => {
  it('menyalakan paginasi tabel', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('DOKUMEN DITERIMA')
    // Daftar contohnya dua baris, sehingga nomor halaman memang tidak digambar —
    // ringkasan barisnya yang membuktikan paginasi menyala. Ukuran halamannya sendiri
    // dikunci di DataTable.test.tsx, bukan di sini.
    expect(screen.getByText(/dari 2 baris/)).toBeInTheDocument()
  })
})

// Layar ini tidak punya tombol hapus, dan itu disengaja: sistem lama tidak punya satu
// pun pernyataan DELETE terhadap POOLDATA.GCNM_MST_PROGRESS_KLAIM, dan tabelnya tidak
// punya kolom penanda terhapus yang dapat dipakai D-66.
describe('penghapusan', () => {
  it('tidak menyediakan tombol hapus', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('DOKUMEN DITERIMA')
    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})
