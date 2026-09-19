import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Rute } from '@/app/App'
import { gunakanSesi } from '@/app/sesi'
import type { ClaimReport } from '@/api/tipe'

const PATH = '/api/pelaporan-klaim'

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

/**
 * Laporan contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
function report(partial: Partial<ClaimReport>): ClaimReport {
  return {
    nomor: 'LPK.26.0001',
    nama_pelapor: 'Bagas Prasetya',
    email_pengirim: 'bagas@contoh.example',
    telepon_pengirim: '021-5550101',
    nama_kurir: 'JNE',
    subjek_email: 'Laporan kerugian',
    nomor_polis: 'CONTOH-PL-000117',
    nama_tertanggung: 'PT Harapan Sentosa',
    email_tertanggung: '',
    kode_bisnis: '006',
    group_panel: '006',
    nomor_referensi: '',
    tanggal_kejadian: '2026-09-09',
    lokasi_kejadian: 'Gudang Cakung',
    kronologi: 'Api muncul dari panel listrik.',
    rincian_kerusakan: 'Rak penyimpanan.',
    sim_pengendara: '',
    nilai_estimasi: '450000000',
    tipe_klaim: 'Fire',
    jumlah_dokumen: 6,
    tanggal_terima_dokumen: '2026-09-12',
    nomor_klaim: '',
    ditransfer: false,
    tanggal_transfer: '',
    tanggal_registrasi: '',
    alasan_belum_transfer: '',
    catatan_belum_registrasi: '',
    tahap: 'BELUM_TRANSFER',
    tahap_label: 'Belum ditransfer',
    dapat_ditransfer: true,
    dapat_diubah: true,
    kode_cabang: '100081',
    diinput_oleh: 'adminpnc',
    diinput_pada: '2026-09-10T02:00:00Z',
    diubah_pada: '2026-09-10T02:00:00Z',
    ...partial,
  }
}

const SAMPLES: ClaimReport[] = [
  report({}),
  report({
    nomor: 'LPK.26.0002',
    nama_pelapor: 'Nadia Rahmawati',
    nama_tertanggung: 'Ibu Saraswati',
    nomor_polis: 'CONTOH-PL-000204',
    tahap: 'BELUM_REGISTRASI',
    tahap_label: 'Belum diregistrasi',
    ditransfer: true,
    tanggal_transfer: '2026-09-11T02:00:00Z',
    dapat_ditransfer: false,
  }),
  report({
    nomor: 'LPK.26.0003',
    nama_pelapor: 'Rizal Abdurrahman',
    nama_tertanggung: 'CV Bina Karya',
    nomor_polis: 'CONTOH-PL-000318',
    tahap: 'SUDAH_REGISTRASI',
    tahap_label: 'Sudah diregistrasi',
    ditransfer: true,
    nomor_klaim: 'PNCN.26.0148',
    dapat_ditransfer: false,
    dapat_diubah: false,
  }),
]

const SUMMARY = [
  { tahap: 'BELUM_TRANSFER', label: 'Belum ditransfer', jumlah: 1 },
  { tahap: 'BELUM_REGISTRASI', label: 'Belum diregistrasi', jumlah: 1 },
  { tahap: 'SUDAH_REGISTRASI', label: 'Sudah diregistrasi', jumlah: 1 },
  { tahap: 'SUDAH_AKSEPTASI', label: 'Sudah diakseptasi', jumlah: 0 },
  { tahap: 'DITOLAK', label: 'Ditolak', jumlah: 0 },
]

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal, sehingga SETIAP layar di balik sesi ikut memanggil
    // /api/portal. Dijawab otomatis supaya tiap uji tidak perlu mengulang daftar yang sama.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    return Promise.resolve(answer(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar dan menerima seluruh aksi. */
function stubDefaultFetch() {
  stubFetch((url, init) => {
    if (url === PATH && init?.method === 'POST') {
      return jsonResponse(201, { laporan: report({ nomor: 'LPK.26.0009' }) })
    }
    if (url.endsWith('/transfer')) {
      return jsonResponse(200, {
        laporan: report({ ditransfer: true, tahap: 'BELUM_REGISTRASI', dapat_ditransfer: false }),
      })
    }
    if (url.endsWith('/klaim')) {
      return jsonResponse(200, {
        laporan: report({ nomor_klaim: 'PNCN.26.0148', tahap: 'SUDAH_REGISTRASI' }),
      })
    }
    if (init?.method === 'PUT') {
      return jsonResponse(200, { laporan: report({}) })
    }
    return jsonResponse(200, {
      laporan: SAMPLES,
      jumlah: SAMPLES.length,
      batas: 50,
      lewati: 0,
      ringkasan: SUMMARY,
    })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/pelaporan-klaim']}>
        <Rute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * Permintaan daftar terakhir yang dikirim layar.
 *
 * Metodenya dibandingkan dengan 'GET' dan bukan diperiksa "kosong": klien API selalu
 * menyetel `method`, termasuk untuk GET.
 */
function lastListCall(): Call | undefined {
  return [...calls]
    .reverse()
    .find((c) => c.url.startsWith(PATH) && (c.init?.method ?? 'GET') === 'GET')
}

beforeEach(() => {
  calls = []
  // Layar berada di balik sesi. Tanpa ini PenjagaSesi melempar ke layar masuk.
  gunakanSesi.setState({
    token: 'token-uji',
    pengguna: SAMPLE_PROFILE,
    berlakuSampai: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  gunakanSesi.getState().bersihkan()
})

describe('daftar', () => {
  it('menampilkan laporan beserta tahapnya dari server', async () => {
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText('Bagas Prasetya')).toBeInTheDocument()
    expect(screen.getByText('PT Harapan Sentosa')).toBeInTheDocument()
    expect(screen.getByText('PNCN.26.0148')).toBeInTheDocument()

    // Jumlah datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/3 laporan pada tahap ini/)).toBeInTheDocument()
  })

  it('membawa token sesi di header, bukan di URL', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    const request = lastListCall()
    expect(request).toBeDefined()
    expect(request?.url).not.toContain('token')

    const headers = request?.init?.headers as Record<string, string>
    expect(headers['Authorization']).toBe('Bearer token-uji')
  })

  it('menampilkan KELIMA tab termasuk yang jumlahnya nol', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    const tabBar = screen.getByRole('tablist', { name: 'Tahap laporan' })
    for (const label of [
      'Semua',
      'Belum ditransfer',
      'Belum diregistrasi',
      'Sudah diregistrasi',
      'Sudah diakseptasi',
      'Ditolak',
    ]) {
      expect(within(tabBar).getByRole('tab', { name: new RegExp(label) })).toBeInTheDocument()
    }
  })

  it('memindahkan tahap ke parameter kueri saat tab ditekan', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('tab', { name: /Ditolak/ }))

    await waitFor(() => {
      expect(lastListCall()?.url).toContain('tahap=DITOLAK')
    })
  })

  it('mengirim pencarian ke server, bukan menyaring di peramban', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.type(screen.getByRole('searchbox'), 'Harapan')

    await waitFor(() => {
      expect(lastListCall()?.url).toContain('cari=Harapan')
    })
  })
})

describe('aksi tahap', () => {
  it('menampilkan tombol Transfer hanya pada laporan yang belum ditransfer', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    // Satu-satunya contoh yang belum ditransfer adalah baris pertama.
    expect(screen.getAllByRole('button', { name: /Transfer laporan/ })).toHaveLength(1)
  })

  it('menonaktifkan Ubah pada laporan yang sudah menjadi klaim', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Rizal Abdurrahman')

    const button = screen.getByRole('button', { name: 'Ubah laporan LPK.26.0003' })
    expect(button).toBeDisabled()
  })

  it('mengirim permintaan transfer ke jalur aksinya sendiri', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: /Transfer laporan LPK.26.0001/ }))

    await waitFor(() => {
      const action = calls.find((c) => c.url.endsWith('/transfer'))
      expect(action).toBeDefined()
      expect(action?.init?.method).toBe('POST')
    })
  })

  it('menolak penautan klaim tanpa nomor klaim', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Nadia Rahmawati')

    await userEvent.click(screen.getByRole('button', { name: /Tautkan laporan LPK.26.0002/ }))

    const linkButton = await screen.findByRole('button', { name: 'Tautkan' })
    expect(linkButton).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Nomor klaim'), 'PNCN.26.0148')
    expect(linkButton).toBeEnabled()
  })
})

describe('form', () => {
  it('menolak simpan tanpa nama pelapor, dan tidak memanggil server', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: /Catat laporan/ }))
    await screen.findByRole('form', { name: 'Catat laporan klaim' })

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama pelapor wajib diisi.')).toBeInTheDocument()
    expect(calls.filter((c) => c.init?.method === 'POST')).toHaveLength(0)
  })

  it('menolak estimasi bertitik ribuan — tebakan artinya salah seratus kali lipat', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: /Catat laporan/ }))
    await screen.findByRole('form', { name: 'Catat laporan klaim' })

    await userEvent.type(screen.getByLabelText('Nama pelapor'), 'Bagas Prasetya')
    await userEvent.type(screen.getByLabelText('Estimasi kerugian'), '1.500.000')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(/tanpa titik ribuan/)).toBeInTheDocument()
    expect(calls.filter((c) => c.init?.method === 'POST')).toHaveLength(0)
  })

  it('mengirim isian ke server dan menutup form saat berhasil', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: /Catat laporan/ }))
    await screen.findByRole('form', { name: 'Catat laporan klaim' })

    await userEvent.type(screen.getByLabelText('Nama pelapor'), 'Sekar Ayu')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.init?.method === 'POST' && c.url === PATH)
      expect(sent).toBeDefined()
      const body = JSON.parse(String(sent?.init?.body)) as Record<string, unknown>
      expect(body['nama_pelapor']).toBe('Sekar Ayu')
      // Nomor TIDAK pernah dikirim klien: pada pencatatan ia dibuat server.
      expect(body).not.toHaveProperty('nomor')
      // Penanda tahap juga tidak: perpindahan tahap punya jalurnya sendiri.
      expect(body).not.toHaveProperty('ditransfer')
      expect(body).not.toHaveProperty('nomor_klaim')
    })

    await waitFor(() => {
      expect(screen.queryByRole('form', { name: 'Catat laporan klaim' })).not.toBeInTheDocument()
    })
  })

  it('mengirim jumlah dokumen sebagai ANGKA, bukan teks', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: /Catat laporan/ }))
    await screen.findByRole('form', { name: 'Catat laporan klaim' })

    await userEvent.type(screen.getByLabelText('Nama pelapor'), 'Sekar Ayu')
    const countField = screen.getByLabelText('Jumlah dokumen')
    await userEvent.clear(countField)
    await userEvent.type(countField, '7')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.init?.method === 'POST' && c.url === PATH)
      expect(sent).toBeDefined()
      const body = JSON.parse(String(sent?.init?.body)) as Record<string, unknown>
      expect(body['jumlah_dokumen']).toBe(7)
    })
  })

  it('mengisi form dengan isi laporan saat mengubah', async () => {
    stubDefaultFetch()
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: 'Ubah laporan LPK.26.0001' }))

    const form = await screen.findByRole('form', { name: 'Ubah laporan klaim' })
    expect(within(form).getByLabelText('Nama pelapor')).toHaveValue('Bagas Prasetya')
    expect(within(form).getByLabelText('Nomor polis')).toHaveValue('CONTOH-PL-000117')
    expect(within(form).getByLabelText('Tanggal kejadian')).toHaveValue('2026-09-09')
  })
})

describe('galat', () => {
  it('menampilkan pesan gangguan saat daftar gagal dimuat', async () => {
    stubFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' }))
    renderPage()

    expect(await screen.findByText('Daftar laporan gagal dimuat')).toBeInTheDocument()
  })

  it('membedakan konflik dari gangguan saat aksi ditolak', async () => {
    stubFetch((url, init) => {
      if (url.endsWith('/transfer')) {
        return jsonResponse(409, {
          kode: 'laporan_sudah_ditransfer',
          pesan: 'Laporan ini sudah ditransfer ke ASM pusat.',
        })
      }
      if (init?.method && init.method !== 'GET') {
        return jsonResponse(200, { laporan: report({}) })
      }
      return jsonResponse(200, {
        laporan: SAMPLES,
        jumlah: SAMPLES.length,
        batas: 50,
        lewati: 0,
        ringkasan: SUMMARY,
      })
    })
    renderPage()
    await screen.findByText('Bagas Prasetya')

    await userEvent.click(screen.getByRole('button', { name: /Transfer laporan LPK.26.0001/ }))

    expect(await screen.findByText('Laporan sudah berubah')).toBeInTheDocument()
  })
})
