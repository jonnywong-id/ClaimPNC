import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { backToListPath, ClaimReportFormPage } from './ClaimReportFormPage'

const ISIAN_KOSONG = {
  tanggal_terima_dokumen: '',
  tanggal_kejadian: '',
  nama_pelapor: '',
  email_pelapor: '',
  telepon_pelapor: '',
  nama_kurir: '',
  nomor_polis: '',
  tertanggung: '',
  nama_bisnis: '',
  nomor_rujukan: '',
  estimasi_kerugian: 0,
  lokasi_kejadian: '',
  subjek_email: '',
  kronologis: '',
  rincian_kerusakan: '',
  alasan: '',
  keterangan_belum_registrasi: '',
  jumlah_dokumen: 0,
}

function berkas(over: Record<string, unknown> = {}) {
  return {
    portal: 'ASM',
    dapat_disunting: true,
    laporan: {
      id: 'RCVN.26.0001',
      nomor_klaim: '',
      nomor_polis: '',
      tertanggung: '',
      nama_bisnis: '',
      nomor_rujukan: '',
      tanggal_kejadian: '',
      tanggal_masuk: '2026-09-19',
      tanggal_aging: '2026-09-19',
      umur_hari: 0,
      pembuat: 'adminpnc',
      kode_cabang: '1001',
      nama_cabang: 'Cabang Contoh Jakarta 1',
      alasan: '',
      subjek_email: '',
      pesan_akhir: '',
      posisi: 'Not Transferred',
      asal: 'claimpnc',
      rujukan_pega: '',
    },
    isian: ISIAN_KOSONG,
    ...over,
  }
}

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

function installFetch(map: (call: Call) => { body: unknown; status?: number }) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
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

function show(id = 'RCVN.26.0001') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/inbox/laporan-klaim/${id}`]}>
        <Routes>
          <Route path="/inbox/laporan-klaim/:id" element={<ClaimReportFormPage />} />
          <Route path="/inbox/laporan-klaim" element={<div data-testid="daftar" />} />
          <Route path="/registrasi/klaim/:klaimID" element={<div data-testid="klaim" />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
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
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('form Input Receive Document', () => {
  it('memakai label isian persis seperti layar lama', async () => {
    installFetch(() => ({ body: berkas() }))
    show()

    // Label dibaca apa adanya dari Section/ViewInputReceiveDocument_sec. Menerjemahkannya
    // akan lolos kompilasi tanpa satu pun tanda bahwa layar berubah bagi penggunanya.
    for (const label of [
      'Tanggal Terima Dokumen',
      'Nama Pengirim / Pelapor Dokumen',
      'Email Pengirim',
      'No. HP Pengirim',
      'Nama Kurir ASM',
      'Nomor Polis',
      'Tanggal Kejadian',
      'Nama Bisnis',
      'No. Referensi/Placing Slip',
      'Estimasi Kerugian',
      'Lokasi Kejadian',
      'Kronologis Kejadian',
      'Rincian Kerusakan',
      'Keterangan Belum Transfer',
      'Keterangan Belum Registrasi',
      'Total Jumlah Dokumen',
    ]) {
      expect(await screen.findByLabelText(label)).toBeInTheDocument()
    }
  })

  it('mengirim seluruh isian sebagai PUT, dan uang dalam sen', async () => {
    installFetch(() => ({ body: berkas() }))
    show()

    await screen.findByLabelText('Nomor Polis')
    await userEvent.type(screen.getByLabelText('Nomor Polis'), 'POL-CONTOH-1')
    await userEvent.type(screen.getByLabelText('Estimasi Kerugian'), '2.500.000,00')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.some((c) => c.method === 'PUT')).toBe(true))

    const saved = calls.find((c) => c.method === 'PUT')?.body as Record<string, unknown>
    expect(saved['nomor_polis']).toBe('POL-CONTOH-1')
    // ADR-0016: uang dikirim sebagai bilangan bulat SEN, tidak pernah pecahan.
    expect(saved['estimasi_kerugian']).toBe(250_000_000)
  })

  it('menyorot setiap isian yang ditolak server, bukan satu pesan saja', async () => {
    installFetch((call) =>
      call.method === 'PUT'
        ? {
            body: {
              kode: 'validasi_gagal',
              pesan: 'Ada isian yang belum benar.',
              detail: [
                { kolom: 'nama_pelapor', pesan: 'Nama Pengirim / Pelapor Dokumen paling panjang 255 karakter.' },
                { kolom: 'nomor_polis', pesan: 'Nomor Polis paling panjang 64 karakter.' },
              ],
            },
            status: 422,
          }
        : { body: berkas() },
    )
    show()

    await screen.findByLabelText('Nomor Polis')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    // Kedua pesan muncul DI BAWAH isiannya masing-masing. Satu pesan di atas form akan
    // membuat pengguna menebak kolom mana yang dimaksud — form ini memuat tujuh belas.
    expect(
      await screen.findByText('Nama Pengirim / Pelapor Dokumen paling panjang 255 karakter.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Nomor Polis paling panjang 64 karakter.')).toBeInTheDocument()
  })

  it('berkas milik Pega dibuka tanpa tombol Simpan dan isiannya terkunci', async () => {
    // Selama masa paralel, tepat satu sistem yang menulis sebuah berkas (ADR-0004, P-1).
    installFetch(() => ({
      body: berkas({
        dapat_disunting: false,
        laporan: { ...berkas().laporan, id: 'RCV-0001', asal: 'pega' },
      }),
    }))
    show('RCV-0001')

    expect(await screen.findByText('Berkas ini hanya dapat dibaca')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Simpan' })).toBeNull()
    expect(screen.getByLabelText('Nomor Polis')).toBeDisabled()
  })

  it('kewenangan menyunting dibaca dari server, bukan disimpulkan dari kolom asal', async () => {
    // Aturan siapa yang boleh menulis milik server. Bila layar menyimpulkannya sendiri,
    // satu aturan hidup di dua tempat — dan yang di layar akan tertinggal saat yang di
    // server berubah.
    installFetch(() => ({
      body: berkas({
        dapat_disunting: false,
        laporan: { ...berkas().laporan, asal: 'claimpnc' },
      }),
    }))
    show()

    expect(await screen.findByText('Berkas ini hanya dapat dibaca')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Simpan' })).toBeNull()
  })

  it('isian yang sudah terisi digambar apa adanya saat form dibuka', async () => {
    installFetch(() => ({
      body: berkas({
        isian: { ...ISIAN_KOSONG, nama_pelapor: 'Pelapor Contoh', estimasi_kerugian: 250_000_000 },
      }),
    }))
    show()

    expect(await screen.findByLabelText('Nama Pengirim / Pelapor Dokumen')).toHaveValue(
      'Pelapor Contoh',
    )
    expect(screen.getByLabelText('Estimasi Kerugian')).toHaveValue('2500000,00')
  })

  it('kegagalan memuat berkas ditampilkan, bukan form kosong yang menyesatkan', async () => {
    installFetch(() => ({
      body: { kode: 'tidak_ditemukan', pesan: 'Laporan klaim yang dimaksud tidak ditemukan.' },
      status: 404,
    }))
    show()

    expect(
      await screen.findByText('Laporan klaim yang dimaksud tidak ditemukan.'),
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('Nomor Polis')).toBeNull()
  })

  it('tombol Kembali mengantar ke daftar', async () => {
    installFetch(() => ({ body: berkas() }))
    show()

    await screen.findByLabelText('Nomor Polis')
    await userEvent.click(screen.getByRole('button', { name: 'Kembali ke daftar' }))

    expect(await screen.findByTestId('daftar')).toBeInTheDocument()
  })
})

describe('kembali ke daftar', () => {
  // Berkas yang baru dibuat berposisi "Not Transferred", sedangkan daftar selalu
  // terbuka di Outstanding. Tanpa pemetaan ini petugas kembali ke tab yang TIDAK
  // memuat berkasnya, lalu menyimpulkan pembuatannya gagal — padahal barisnya
  // tersimpan (keluhan Work Owner, 2026-09-24).
  it('membuka tab tempat berkas itu benar-benar berada', () => {
    expect(backToListPath('Not Transferred')).toContain('kategori=belum-diserahkan')
    expect(backToListPath('Not Registered')).toContain('kategori=belum-registrasi')
    expect(backToListPath('Outstanding')).toContain('kategori=outstanding')
  })

  // Posisi yang tidak dikenali TIDAK ditebak. Menebak akan membuka tab yang salah,
  // dan itu persis kegagalan yang fungsi ini ada untuk menutupnya.
  it('jatuh ke tab bawaan bila posisinya tidak dikenali', () => {
    expect(backToListPath(undefined)).toBe('/inbox/laporan-klaim')
    expect(backToListPath('Entah Apa')).toBe('/inbox/laporan-klaim')
  })
})

describe('tombol Register Klaim', () => {
  // Tombol ini ADA di layar lama (Section/InputReceiveDocument_sect.xml, memanggil
  // CreateRegisterKlaimPNC, yang langkah pertamanya Call CreateInputKlaim). Sejak
  // 2026-09-24 ia bekerja: memanggil POST /api/registrasi/klaim.
  //
  // Yang diuji di sini adalah tautannya, bukan isi modul registrasi. Nomor laporan WAJIB
  // ikut terkirim — itulah yang mengisi RCVID pada klaim, dan tanpanya baris RCV-nya tidak
  // pernah berpindah keluar dari tab "Not Transferred".
  it('mengirim nomor polis beserta nomor laporan asalnya', async () => {
    installFetch((call) =>
      call.url.includes('/api/registrasi/klaim')
        ? { body: { klaim: { id: 'KLM-1', nomor: 'PNCN.26.0007' } } }
        : { body: berkas({ isian: { ...ISIAN_KOSONG, nomor_polis: '01.002.2026.00001' } }) },
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Register Klaim' }))

    await waitFor(() => {
      const kirim = calls.find((c) => c.url.includes('/api/registrasi/klaim'))
      expect(kirim?.method).toBe('POST')
      expect(kirim?.body).toEqual({
        nomor_polis: '01.002.2026.00001',
        nomor_laporan: 'RCVN.26.0001',
      })
    })
  })

  // Setelah klaim terbit, petugas dibawa ke klaimnya — bukan ditinggalkan di form laporan
  // tanpa tanda apa pun bahwa sesuatu terjadi.
  it('membuka klaim yang baru terbit', async () => {
    installFetch((call) =>
      call.url.includes('/api/registrasi/klaim')
        ? { body: { klaim: { id: 'KLM-1', nomor: 'PNCN.26.0007' } } }
        : { body: berkas({ isian: { ...ISIAN_KOSONG, nomor_polis: '01.002.2026.00001' } }) },
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Register Klaim' }))

    expect(await screen.findByTestId('klaim')).toBeInTheDocument()
  })

  // Berkas milik Pega hanya dapat dibaca (P-1). Mendaftarkan klaim MENULIS ke berkas itu —
  // NOKLAIM-nya terisi — sehingga tombol ini tunduk pada kewenangan yang sama dengan
  // Simpan. Kewenangannya datang dari server, bukan disimpulkan layar.
  it('mati pada berkas yang hanya dapat dibaca', async () => {
    installFetch(() => ({ body: berkas({ dapat_disunting: false }) }))
    show()

    expect(await screen.findByRole('button', { name: 'Register Klaim' })).toBeDisabled()
  })

  // Nomor polis kosong ditolak SEBELUM permintaan berangkat: registrasi mengambil snapshot
  // polis, dan snapshot tidak dapat diambil tanpa nomornya. Menolaknya di layar memberi
  // petugas kalimat yang dapat ditindaklanjuti, bukan galat server.
  it('menolak tanpa nomor polis, tanpa memanggil server', async () => {
    installFetch(() => ({ body: berkas() }))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Register Klaim' }))

    expect(await screen.findByText(/Nomor Polis harus diisi/)).toBeInTheDocument()
    expect(calls.some((c) => c.url.includes('/api/registrasi/klaim'))).toBe(false)
  })
})
