import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { gunakanPortalTerpilih } from '@/app/portal'
import { gunakanSesi } from '@/app/sesi'

import { HalamanStatusProgres1 } from './HalamanStatusProgres1'

const POSISI = {
  posisi: [
    { kode: '002', nama: 'REGISTER' },
    { kode: '004', nama: 'SURVEY' },
    { kode: '006', nama: 'KOMITE' },
    { kode: '007', nama: 'AKSEPTASI' },
  ],
}

const DAFTAR = {
  portal: 'ASM',
  status_progres: [
    { id: '01', nama: 'DOKUMEN DITERIMA', kode_posisi: '002', nama_posisi: 'REGISTER' },
    { id: '02', nama: 'MENUNGGU JADWAL SURVEI', kode_posisi: '004', nama_posisi: 'SURVEY' },
  ],
}

type Permintaan = {
  url: string
  metode: string
  badan: unknown
  header: Record<string, string>
}

let dicatat: Permintaan[] = []

/** jawaban memetakan jalur permintaan ke respons yang dikembalikan fetch tiruan. */
type Jawaban = { badan: unknown; status?: number }

function pasangFetch(peta: (p: Permintaan) => Jawaban) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const permintaan: Permintaan = {
      url,
      metode: init?.method ?? 'GET',
      badan: init?.body ? JSON.parse(init.body as string) : undefined,
      header: (init?.headers as Record<string, string>) ?? {},
    }
    dicatat.push(permintaan)

    const { badan, status = 200 } = peta(permintaan)
    return Promise.resolve(
      new Response(JSON.stringify(badan), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

/** jawabanBaku melayani daftar dan posisi; mutasi dijawab pemanggil lewat `mutasi`. */
function jawabanBaku(mutasi?: (p: Permintaan) => Jawaban) {
  return (p: Permintaan): Jawaban => {
    if (p.url.startsWith('/api/master/posisi-klaim')) return { badan: POSISI }
    if (p.metode === 'GET') return { badan: DAFTAR }
    if (mutasi) return mutasi(p)
    return { badan: { status_progres: DAFTAR.status_progres[0], portal: 'ASM' }, status: 201 }
  }
}

function tampilkan() {
  const klien = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={klien}>
      <MemoryRouter>
        <HalamanStatusProgres1 />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function masukkanSesi() {
  gunakanSesi.getState().masuk({
    token: 'token-contoh',
    pengguna: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    berlakuSampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
}

beforeEach(() => {
  dicatat = []
  window.sessionStorage.clear()
  gunakanSesi.getState().bersihkan()
  gunakanPortalTerpilih.getState().bersihkan()
  masukkanSesi()
  gunakanPortalTerpilih.getState().pilih('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('daftar master status progres 1', () => {
  it('menampilkan judul dan ketiga kolom seperti layar lama', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()

    expect(
      screen.getByRole('heading', { name: 'Master Status Progres 1' }),
    ).toBeInTheDocument()

    const tabel = await screen.findByRole('table')
    expect(within(tabel).getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(
      within(tabel).getByRole('columnheader', { name: 'Status Progres' }),
    ).toBeInTheDocument()
    expect(within(tabel).getByRole('columnheader', { name: 'Posisi' })).toBeInTheDocument()
  })

  it('menampilkan baris beserta label posisinya', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()

    expect(await screen.findByText('DOKUMEN DITERIMA')).toBeInTheDocument()
    expect(screen.getByText('MENUNGGU JADWAL SURVEI')).toBeInTheDocument()
    expect(screen.getByText('REGISTER')).toBeInTheDocument()
    expect(screen.getByText('SURVEY')).toBeInTheDocument()
  })

  // Portal WAJIB ikut di setiap permintaan yang menyentuh basis data entitas. Backend
  // menolak yang tidak menyebutkannya, dan itu memang yang diinginkan (R-20).
  it('mengirim header portal dan token pada permintaan daftar', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()

    await waitFor(() => expect(screen.getByText('DOKUMEN DITERIMA')).toBeInTheDocument())

    const permintaanDaftar = dicatat.find(
      (p) => p.url === '/api/master/status-progres-1' && p.metode === 'GET',
    )
    expect(permintaanDaftar).toBeDefined()
    expect(permintaanDaftar?.header['X-Portal']).toBe('ASM')
    expect(permintaanDaftar?.header['Authorization']).toBe('Bearer token-contoh')
  })

  // Daftar posisi TIDAK menyertakan portal: ia daftar milik aplikasi, bukan isi basis
  // data entitas mana pun.
  it('tidak mengirim header portal saat meminta daftar posisi', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()

    await waitFor(() =>
      expect(dicatat.some((p) => p.url === '/api/master/posisi-klaim')).toBe(true),
    )
    const permintaanPosisi = dicatat.find((p) => p.url === '/api/master/posisi-klaim')
    expect(permintaanPosisi?.header['X-Portal']).toBeUndefined()
  })

  it('menyebutkan entitas yang sedang dilihat', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()

    expect(await screen.findByText(/Portal entitas:/)).toBeInTheDocument()
  })

  // Layar tidak menembak API hanya untuk menerima penolakan; ia menuntun pengguna.
  it('meminta pengguna memilih portal lebih dulu bila belum ada yang dipilih', async () => {
    gunakanPortalTerpilih.getState().bersihkan()
    pasangFetch(jawabanBaku())
    tampilkan()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(
      dicatat.some((p) => p.url === '/api/master/status-progres-1'),
    ).toBe(false)
  })

  it('membedakan entitas yang belum punya kredensial basis data', async () => {
    pasangFetch((p) => {
      if (p.url.startsWith('/api/master/posisi-klaim')) return { badan: POSISI }
      return {
        badan: { kode: 'portal_belum_siap', pesan: 'belum tersedia' },
        status: 503,
      }
    })
    tampilkan()

    expect(
      await screen.findByText('Basis data entitas ini belum tersedia'),
    ).toBeInTheDocument()
  })

  it('menampilkan pesan ketika tabel entitas masih kosong', async () => {
    pasangFetch((p) => {
      if (p.url.startsWith('/api/master/posisi-klaim')) return { badan: POSISI }
      return { badan: { status_progres: [], portal: 'ASM' } }
    })
    tampilkan()

    expect(
      await screen.findByText('Belum ada status progres pada entitas ini.'),
    ).toBeInTheDocument()
  })
})

describe('penambahan', () => {
  it('mengirim isian ke server beserta portalnya, tanpa ID', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Tambah' }))

    await pengguna.type(screen.getByLabelText('Status Progres'), 'MENUNGGU BERKAS')
    await pengguna.selectOptions(screen.getByLabelText('Posisi'), '004')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(dicatat.some((p) => p.metode === 'POST')).toBe(true)
    })

    const kirim = dicatat.find((p) => p.metode === 'POST')
    expect(kirim?.url).toBe('/api/master/status-progres-1')
    expect(kirim?.header['X-Portal']).toBe('ASM')
    // ID tidak dikirim: ia diterbitkan server dari isi tabel.
    expect(kirim?.badan).toEqual({ nama: 'MENUNGGU BERKAS', kode_posisi: '004' })
  })

  // Validasi di layar menahan isian kosong sebelum permintaan dikirim.
  it('menolak nama kosong tanpa menembak server', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Tambah' }))
    await pengguna.selectOptions(screen.getByLabelText('Posisi'), '002')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama status progres wajib diisi.')).toBeInTheDocument()
    expect(dicatat.some((p) => p.metode === 'POST')).toBe(false)
  })

  it('menolak posisi yang belum dipilih', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Tambah' }))
    await pengguna.type(screen.getByLabelText('Status Progres'), 'APA SAJA')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Posisi klaim wajib dipilih.')).toBeInTheDocument()
    expect(dicatat.some((p) => p.metode === 'POST')).toBe(false)
  })

  // Server mengirim SELURUH pelanggaran sekaligus (P-5), dan layar menyorotnya pada
  // isiannya masing-masing — bukan meringkasnya menjadi satu pesan.
  it('menyorot setiap isian yang ditolak server', async () => {
    pasangFetch(
      jawabanBaku(() => ({
        status: 422,
        badan: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          detail: [
            { kolom: 'nama', pesan: 'Nama status progres wajib diisi.' },
            { kolom: 'kode_posisi', pesan: 'Posisi klaim tidak dikenal.' },
          ],
        },
      })),
    )
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Tambah' }))
    await pengguna.type(screen.getByLabelText('Status Progres'), 'X')
    await pengguna.selectOptions(screen.getByLabelText('Posisi'), '002')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Posisi klaim tidak dikenal.')).toBeInTheDocument()
    expect(screen.getByText('Nama status progres wajib diisi.')).toBeInTheDocument()
  })

  // Form tetap terbuka saat penyimpanan gagal: menutupnya akan membuang isian yang
  // baru diketik pengguna.
  it('mempertahankan form dan isiannya ketika penyimpanan gagal', async () => {
    pasangFetch(
      jawabanBaku(() => ({
        status: 500,
        badan: { kode: 'galat_internal', pesan: 'gagal' },
      })),
    )
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Tambah' }))
    await pengguna.type(screen.getByLabelText('Status Progres'), 'MENUNGGU BERKAS')
    await pengguna.selectOptions(screen.getByLabelText('Posisi'), '004')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Terjadi kesalahan pada sistem')).toBeInTheDocument()
    expect(screen.getByLabelText('Status Progres')).toHaveValue('MENUNGGU BERKAS')
  })

  it('menutup form setelah penyimpanan berhasil', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Tambah' }))
    await pengguna.type(screen.getByLabelText('Status Progres'), 'MENUNGGU BERKAS')
    await pengguna.selectOptions(screen.getByLabelText('Posisi'), '004')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() =>
      expect(screen.queryByLabelText('Status Progres')).not.toBeInTheDocument(),
    )
  })
})

describe('penyuntingan', () => {
  it('memuat isian baris yang dipilih dan menampilkan ID sebagai tidak dapat diubah', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Ubah DOKUMEN DITERIMA' }))

    expect(screen.getByLabelText('Status Progres')).toHaveValue('DOKUMEN DITERIMA')
    expect(screen.getByLabelText('Posisi')).toHaveValue('002')
    expect(screen.getByText('(tidak dapat diubah)')).toBeInTheDocument()
    // ID tidak muncul sebagai isian yang dapat disunting.
    expect(screen.queryByLabelText('ID')).not.toBeInTheDocument()
  })

  it('mengirim PUT ke jalur berisi ID, tanpa ID di badan permintaan', async () => {
    pasangFetch(
      jawabanBaku(() => ({
        badan: {
          status_progres: {
            id: '01',
            nama: 'DOKUMEN LENGKAP',
            kode_posisi: '006',
            nama_posisi: 'KOMITE',
          },
          portal: 'ASM',
        },
      })),
    )
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Ubah DOKUMEN DITERIMA' }))
    await pengguna.clear(screen.getByLabelText('Status Progres'))
    await pengguna.type(screen.getByLabelText('Status Progres'), 'DOKUMEN LENGKAP')
    await pengguna.selectOptions(screen.getByLabelText('Posisi'), '006')
    await pengguna.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(dicatat.some((p) => p.metode === 'PUT')).toBe(true))

    const kirim = dicatat.find((p) => p.metode === 'PUT')
    expect(kirim?.url).toBe('/api/master/status-progres-1/01')
    expect(kirim?.header['X-Portal']).toBe('ASM')
    expect(kirim?.badan).toEqual({ nama: 'DOKUMEN LENGKAP', kode_posisi: '006' })
  })

  it('menutup form ketika dibatalkan tanpa mengirim apa pun', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()
    const pengguna = userEvent.setup()

    await pengguna.click(await screen.findByRole('button', { name: 'Ubah DOKUMEN DITERIMA' }))
    await pengguna.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByLabelText('Status Progres')).not.toBeInTheDocument()
    expect(dicatat.some((p) => p.metode === 'PUT' || p.metode === 'POST')).toBe(false)
  })
})

// Layar ini tidak punya tombol hapus, dan itu disengaja: sistem lama tidak punya satu
// pun pernyataan DELETE terhadap POOLDATA.GCNM_MST_PROGRESS_KLAIM, dan tabelnya tidak
// punya kolom penanda terhapus yang dapat dipakai D-66.
describe('penghapusan', () => {
  it('tidak menyediakan tombol hapus', async () => {
    pasangFetch(jawabanBaku())
    tampilkan()

    await screen.findByText('DOKUMEN DITERIMA')
    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})
