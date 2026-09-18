import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { gunakanPortalTerpilih } from '@/app/portal'
import { gunakanSesi } from '@/app/sesi'

import { Kerangka } from './Kerangka'
import { menuUtama } from './menu'

const DAFTAR_PORTAL = {
  portal: [
    { id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true },
    { id: '202600102', nama: 'ASURANSI SIMAS INSURTECH', alias: 'ASI', siap: true },
  ],
  utama: 'ASM',
}

function pasangFetch() {
  vi.stubGlobal('fetch', (url: string) =>
    Promise.resolve(
      new Response(JSON.stringify(url.startsWith('/api/portal') ? DAFTAR_PORTAL : {}), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    ),
  )
}

/** tampilkan merakit kerangka di dalam router, pada jalur yang diminta. */
function tampilkan(jalurAwal: string, aksiDiHalaman = false) {
  const klien = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={klien}>
      <MemoryRouter initialEntries={[jalurAwal]}>
        <Routes>
          <Route
            path="/"
            element={<Kerangka anak={<p>isi beranda</p>} aksiDiHalaman={aksiDiHalaman} />}
          />
          <Route
            path="/master/status-progres-1"
            element={<Kerangka anak={<p>isi layar master</p>} aksiDiHalaman={aksiDiHalaman} />}
          />
        </Routes>
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
  window.sessionStorage.clear()
  gunakanSesi.getState().bersihkan()
  gunakanPortalTerpilih.getState().bersihkan()
  masukkanSesi()
  pasangFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('menu utama', () => {
  it('menampilkan setiap butir yang terdaftar di peta menu', () => {
    tampilkan('/')

    const menu = screen.getByRole('navigation', { name: 'Menu utama' })
    for (const kelompok of menuUtama) {
      for (const butir of kelompok.butir) {
        expect(within(menu).getByRole('link', { name: butir.label })).toBeInTheDocument()
      }
    }
  })

  // Menambah modul harus cukup dengan menambah satu baris di app/menu.ts. Uji ini
  // membuktikan komponennya benar-benar membaca data itu, bukan menulis butirnya sendiri.
  it('memuat layar master sebagai butir menu', () => {
    tampilkan('/')

    const tautan = screen.getByRole('link', { name: 'Status Progres 1' })
    expect(tautan).toHaveAttribute('href', '/master/status-progres-1')
  })

  it('mengelompokkan butir master data di bawah judulnya', () => {
    tampilkan('/')
    expect(screen.getByRole('heading', { name: 'Master Data' })).toBeInTheDocument()
  })

  // Penanda aktif bukan hanya warna: pengguna pembaca layar perlu tahu ia sedang di mana.
  it('menandai butir yang jalurnya sedang dibuka dengan aria-current', () => {
    tampilkan('/master/status-progres-1')

    expect(screen.getByRole('link', { name: 'Status Progres 1' })).toHaveAttribute(
      'aria-current',
      'page',
    )
    expect(screen.getByRole('link', { name: 'Beranda' })).not.toHaveAttribute('aria-current')
  })

  // Butir Beranda memakai `end`; tanpa itu ia ikut aktif pada SETIAP jalur, karena semua
  // jalur dimulai dengan "/".
  it('tidak menandai Beranda aktif saat berada di layar lain', () => {
    tampilkan('/master/status-progres-1')
    expect(screen.getByRole('link', { name: 'Beranda' })).not.toHaveAttribute('aria-current')
  })

  it('menandai Beranda aktif saat berada di beranda', () => {
    tampilkan('/')
    expect(screen.getByRole('link', { name: 'Beranda' })).toHaveAttribute('aria-current', 'page')
  })

  // Menu BUKAN kendali akses. Keterangan ini ada supaya penguji bisnis tidak mengira
  // izin peran sudah ditegakkan di antarmuka — cacat sistem lama yang tidak boleh diulang.
  it('menyebutkan bahwa daftar menu belum disaring izin peran', () => {
    tampilkan('/')
    expect(screen.getByText(/belum disaring izin peran/i)).toBeInTheDocument()
  })

  it('berpindah layar saat butir menu diklik', async () => {
    tampilkan('/')
    expect(screen.getByText('isi beranda')).toBeInTheDocument()

    await userEvent.setup().click(screen.getByRole('link', { name: 'Status Progres 1' }))
    expect(await screen.findByText('isi layar master')).toBeInTheDocument()
  })
})

describe('aksi tingkat aplikasi', () => {
  // Tanpa ini, layar modul menjadi jalan buntu: pengguna tidak dapat berpindah entitas
  // maupun keluar tanpa kembali ke beranda lebih dulu.
  it('menampilkan pemilih portal dan tombol keluar di layar modul', async () => {
    tampilkan('/master/status-progres-1')

    expect(await screen.findByLabelText('Portal')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Keluar' })).toBeInTheDocument()
  })

  // Beranda memuat keduanya di header-nya sendiri; menampilkannya dua kali membuat tidak
  // jelas pemilih portal mana yang berlaku.
  it('tidak menampilkannya ketika halaman sudah menyediakannya sendiri', () => {
    tampilkan('/', true)

    expect(screen.queryByLabelText('Portal')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Keluar' })).not.toBeInTheDocument()
    // Menunya tetap tampil — yang disembunyikan hanya aksinya.
    expect(screen.getByRole('navigation', { name: 'Menu utama' })).toBeInTheDocument()
  })

  it('mencabut sesi dan pilihan portal saat keluar ditekan', async () => {
    tampilkan('/master/status-progres-1')
    gunakanPortalTerpilih.getState().pilih('ASM')

    await userEvent.setup().click(screen.getByRole('button', { name: 'Keluar' }))

    await waitFor(() => expect(gunakanSesi.getState().token).toBeNull())
    expect(gunakanPortalTerpilih.getState().alias).toBeNull()
  })
})

describe('isi halaman', () => {
  it('merender halaman yang dibungkusnya', () => {
    tampilkan('/master/status-progres-1')
    expect(screen.getByText('isi layar master')).toBeInTheDocument()
  })
})
