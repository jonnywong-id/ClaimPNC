import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Rute } from '@/app/App'
import { gunakanPortalTerpilih } from '@/app/portal'
import { gunakanSesi } from '@/app/sesi'

const PROFIL_CONTOH = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

// Daftar portal dipanggil beranda begitu sesi terbit; peladen tiruan harus
// menjawabnya, kalau tidak layar beranda berhenti di keadaan memuat.
const DAFTAR_PORTAL = {
  portal: [
    { id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true },
    { id: '202600103', nama: 'SINARMAS ASURANSI SYARIAH', alias: 'SMAS', siap: false },
  ],
  utama: 'ASM',
}

type Panggilan = { url: string; init: RequestInit | undefined }

let panggilan: Panggilan[] = []

// pasangFetch memasang peladen tiruan. Permintaan /api/portal dijawab otomatis supaya
// setiap uji tidak perlu mengulang daftar portal yang sama.
function pasangFetch(jawaban: (url: string) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    panggilan.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(responsJSON(200, DAFTAR_PORTAL))
    return Promise.resolve(jawaban(url))
  })
}

function responsJSON(status: number, badan: unknown): Response {
  return new Response(JSON.stringify(badan), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function tampilkan() {
  const klien = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={klien}>
      <MemoryRouter initialEntries={['/masuk']}>
        <Rute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function isiDanKirim(namaPengguna: string, kataSandi: string) {
  const pengguna = userEvent.setup()
  await pengguna.type(screen.getByLabelText('Nama pengguna'), namaPengguna)
  await pengguna.type(screen.getByLabelText('Kata sandi'), kataSandi)
  await pengguna.click(screen.getByRole('button', { name: 'Masuk' }))
}

beforeEach(() => {
  panggilan = []
  window.sessionStorage.clear()
  gunakanSesi.getState().bersihkan()
  gunakanPortalTerpilih.getState().bersihkan()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('layar masuk', () => {
  it('menampilkan beranda setelah kredensial diterima', async () => {
    pasangFetch(() =>
      responsJSON(200, {
        token: 'token-contoh',
        tipe_token: 'Bearer',
        berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        pengguna: PROFIL_CONTOH,
      }),
    )
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')

    await waitFor(() => expect(screen.getByText('Selamat datang, Contoh Administrator.')).toBeInTheDocument())
    expect(screen.getByText(PROFIL_CONTOH.nama)).toBeInTheDocument()
  })

  // Ketiga jenis galat menampilkan pesan BERBEDA dan dapat ditindaklanjuti
  // (TKT-U1-002, acceptance criteria pertama).
  it.each([
    {
      nama: 'kredensial salah',
      status: 401,
      kode: 'kredensial_salah',
      judul: 'Nama pengguna atau kata sandi salah',
    },
    {
      nama: 'pengguna tidak aktif',
      status: 403,
      kode: 'pengguna_tidak_aktif',
      judul: 'Akun Anda tidak aktif',
    },
    {
      nama: 'sistem identitas tidak dapat dihubungi',
      status: 503,
      kode: 'sistem_identitas_tidak_terhubung',
      judul: 'Sistem identitas sedang tidak dapat dihubungi',
    },
  ])('membedakan galat $nama', async ({ status, kode, judul }) => {
    pasangFetch(() => responsJSON(status, { kode, pesan: 'pesan dari server' }))
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')

    const peringatan = await screen.findByRole('alert')
    expect(peringatan).toHaveTextContent(judul)
  })

  // Pesan untuk pengguna yang tidak ada dan pengguna yang ada berkata sandi salah harus
  // SAMA — membedakannya membocorkan siapa yang punya akun.
  it('tidak membocorkan keberadaan akun', async () => {
    pasangFetch(() =>
      responsJSON(401, { kode: 'kredensial_salah', pesan: 'Nama pengguna atau kata sandi salah.' }),
    )

    const pertama = tampilkan()
    await isiDanKirim('tidakpernahada', 'apa saja')
    const pesanPertama = (await screen.findByRole('alert')).textContent
    pertama.unmount()

    tampilkan()
    await isiDanKirim('adminpnc', 'sandi salah')
    const pesanKedua = (await screen.findByRole('alert')).textContent

    expect(pesanKedua).toBe(pesanPertama)
  })

  it('menampilkan pesan tersendiri ketika server tidak dapat dihubungi', async () => {
    vi.stubGlobal('fetch', () => Promise.reject(new TypeError('network down')))
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')

    const peringatan = await screen.findByRole('alert')
    expect(peringatan).toHaveTextContent('Server Claim PNC tidak dapat dihubungi')
  })

  it('menolak kirim bila isian kosong, tanpa memanggil server', async () => {
    pasangFetch(() => responsJSON(200, {}))
    tampilkan()

    await userEvent.setup().click(screen.getByRole('button', { name: 'Masuk' }))

    expect(await screen.findByText('Nama pengguna wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Kata sandi wajib diisi.')).toBeInTheDocument()
    expect(panggilan).toHaveLength(0)
  })

  it('tidak pernah menaruh kata sandi maupun token di URL', async () => {
    pasangFetch(() =>
      responsJSON(200, {
        token: 'token-contoh',
        tipe_token: 'Bearer',
        berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        pengguna: PROFIL_CONTOH,
      }),
    )
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')

    await waitFor(() => expect(panggilan.length).toBeGreaterThan(0))
    for (const { url } of panggilan) {
      expect(url).not.toContain('rahasia123')
      expect(url).not.toContain('token-contoh')
    }
    expect(window.location.search).toBe('')
  })

  it('tidak pernah menyimpan kata sandi di peramban', async () => {
    pasangFetch(() =>
      responsJSON(200, {
        token: 'token-contoh',
        tipe_token: 'Bearer',
        berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        pengguna: PROFIL_CONTOH,
      }),
    )
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')

    await waitFor(() => expect(gunakanSesi.getState().token).toBe('token-contoh'))
    expect(JSON.stringify(window.sessionStorage)).not.toContain('rahasia123')
  })

  it('mengirim token sebagai Bearer di header, bukan di tempat lain', async () => {
    pasangFetch((url) =>
      url === '/api/masuk'
        ? responsJSON(200, {
            token: 'token-contoh',
            tipe_token: 'Bearer',
            berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
            pengguna: PROFIL_CONTOH,
          })
        : new Response(null, { status: 204 }),
    )
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')
    await waitFor(() => expect(screen.getByText('Selamat datang, Contoh Administrator.')).toBeInTheDocument())

    await userEvent.setup().click(screen.getByRole('button', { name: 'Keluar' }))

    await waitFor(() => {
      const keluar = panggilan.find((p) => p.url === '/api/keluar')
      expect(keluar).toBeDefined()
      const header = keluar?.init?.headers as Record<string, string>
      expect(header['Authorization']).toBe('Bearer token-contoh')
    })
  })

  it('membersihkan sesi di peramban setelah keluar', async () => {
    pasangFetch((url) =>
      url === '/api/masuk'
        ? responsJSON(200, {
            token: 'token-contoh',
            tipe_token: 'Bearer',
            berlaku_sampai: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
            pengguna: PROFIL_CONTOH,
          })
        : new Response(null, { status: 204 }),
    )
    tampilkan()
    await isiDanKirim('adminpnc', 'rahasia123')
    await waitFor(() => expect(screen.getByText('Selamat datang, Contoh Administrator.')).toBeInTheDocument())

    await userEvent.setup().click(screen.getByRole('button', { name: 'Keluar' }))

    await waitFor(() => expect(gunakanSesi.getState().token).toBeNull())
    expect(await screen.findByRole('button', { name: 'Masuk' })).toBeInTheDocument()
  })
})
