import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSession } from '@/app/session'
import { formatPercent, formatRupiah, formatDate, rupiahToCents } from '@/components/format'

import { ClaimPage } from './ClaimPage'

const CLAIM = {
  klaim: {
    id: 'klaim-1',
    nomor: '',
    portal: 'ASM',
    polis: {
      nomor: 'POL-FIRE-0001',
      lini: '006',
      nama_lini: 'Fire',
      jenis_bisnis: 'Fire',
      mulai_pertanggungan: '2026-01-01',
      akhir_pertanggungan: '2026-12-31',
      mata_uang: 'IDR',
      nama_tertanggung: 'PT Contoh Industri Nusantara',
      deklarasi: false,
      penjamin_kredit: false,
    },
    tanggal_kejadian: '2026-06-05',
    tanggal_lapor: '2026-06-06',
    tanggal_terima_dokumen: '2026-06-07',
    lokasi: 'Gudang A',
    kronologi: 'Kebakaran gudang.',
    pelapor: { nama: 'Pelapor Uji', telepon: '0800', email: '', alamat: '', hubungan: 1, hubungan_lainnya: '' },
    nilai_estimasi_sen: 1_000_000_000,
    mata_uang: 'IDR',
    nomor_slik: '',
    ex_gratia: false,
    user_teknis: 'TEKNIK01',
    rcv_id: '',
    objek: [
      {
        id: 'OBJ-1',
        nama: 'Gudang',
        lokasi: 'Gudang A',
        coverage: [
          {
            id: 'CVG-1',
            penyebab_kerugian: '11817',
            tsi_sen: 50_000_000_000,
            spreading: [
              { jenis_treaty: '10007', nama: 'OR', share: 600_000, dihapus: false, objek_fac_offer: '' },
              { jenis_treaty: '10008', nama: 'Treaty', share: 400_000, dihapus: false, objek_fac_offer: '' },
            ],
          },
        ],
      },
    ],
    status_pucl: 0,
    transfer_compliance: false,
    status_proses: 'BERJALAN',
    status_klaim: '',
    flag_klaim: '0',
    status_posisi_progres: 'On Progress',
    tahap_kini: 'input-register',
  },
  tugas: {
    id: 'tugas-1',
    klaim_id: 'klaim-1',
    nomor_klaim: '',
    tahap: 'input-register',
    nama_tahap: 'Input Register',
    antrean: 'WORKLIST',
    workbasket: '',
    pemilik: '90000001',
    dapat_diambil: false,
    tindakan_keluar: 'InputRegister',
    dibuat_pada: '2026-06-10T03:00:00Z',
  },
  jalur: ['input-register', 'estimasi-admin', 'pilih-surveyor'],
  jejak_keputusan: null,
  large_loss: false,
}

const ALUR = {
  nama: 'Register',
  mulai: 'view-polis',
  tahap: [
    { id: 'input-register', nama: 'Input Register', antrean: 'WORKLIST', workbasket: '', router: 'PNCAdminRouter', tindakan_keluar: 'InputRegister', pega_id: 'Assignment1' },
    { id: 'estimasi-admin', nama: 'Input Estimasi', antrean: 'WORKLIST', workbasket: '', router: 'PNCAdminRouter', tindakan_keluar: 'InputEstimasi', pega_id: 'Assignment7' },
    { id: 'pilih-surveyor', nama: 'Choose Surveyor', antrean: 'WORKLIST', workbasket: '', router: 'PNCTeknikRouter', tindakan_keluar: 'InputSurveyor', pega_id: 'Assignment3' },
  ],
}

let sentBody: unknown = null

function stubFetch(registerAnswer: () => { body: unknown; status: number }) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    let body: unknown = {}
    let status = 200

    if (url === '/api/registrasi/alur') body = ALUR
    else if (url.startsWith('/api/registrasi/klaim/')) body = CLAIM
    else if (url === '/api/registrasi/register') {
      sentBody = init?.body ? JSON.parse(init.body as string) : null
      const answer = registerAnswer()
      body = answer.body
      status = answer.status
    }

    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function mount(component: ReactNode) {
  const apiClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={apiClient}>
      <MemoryRouter initialEntries={['/registrasi/klaim/klaim-1']}>
        <Routes>
          <Route path="/registrasi/klaim/:claimID" element={component} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  sentBody = null
  window.sessionStorage.clear()
  useSession.getState().cleanup()
  useSession.getState().signIn({
    token: 'token-contoh',
    pengguna: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    expiresAt: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('layar kerja klaim', () => {
  it('menampilkan tahapan klaim beserta tahap yang sedang berjalan', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)

    const stages = await screen.findByRole('navigation', { name: 'Tahapan klaim' })
    expect(stages).toHaveTextContent('Input Register')
    expect(stages).toHaveTextContent('Choose Surveyor')
    // Tahap berjalan ditandai untuk pembaca layar, bukan hanya dengan warna.
    expect(screen.getByText('Input Register')).toHaveAttribute('aria-current', 'step')
  })

  it('mengirim uang sebagai sen dan share sebagai persen dikali sepuluh ribu', async () => {
    stubFetch(() => ({ body: { ...CLAIM, klaim: { ...CLAIM.klaim, nomor: 'PNCN.26.0001' } }, status: 200 }))
    mount(<ClaimPage />)

    const save = await screen.findByRole('button', { name: /Simpan dan terbitkan/ })
    await userEvent.setup().click(save)

    await waitFor(() => expect(sentBody).not.toBeNull())
    expect(sentBody).toMatchObject({
      tugas_id: 'tugas-1',
      nilai_estimasi_sen: 1_000_000_000,
      kembali: false,
    })

    const objek = (sentBody as { objek: { coverage: { spreading: { share: number }[] }[] }[] }).objek
    expect(objek[0]?.coverage[0]?.spreading[0]?.share).toBe(600_000)
  })

  // Sistem lama menampilkan satu pesan, lalu pesan berikutnya setelah disimpan ulang.
  // Di sini seluruhnya ditampilkan sekaligus, dan masing-masing menempel pada kolomnya.
  it('menempelkan setiap pelanggaran pada kolom yang harus diperbaiki', async () => {
    stubFetch(() => ({
      body: {
        kode: 'validasi_gagal',
        pesan: 'Nomor SLIK Harus Diisi',
        violations: [
          { kode: 'nomor_slik_kosong', field: 'nomor_slik', pesan: 'Nomor SLIK Harus Diisi' },
          {
            kode: 'tanggal_lapor_sebelum_kejadian',
            field: 'tanggal_lapor',
            pesan: 'Tanggal Lapor harus setelah Tanggal Kejadian.',
          },
        ],
      },
      status: 422,
    }))
    mount(<ClaimPage />)

    const save = await screen.findByRole('button', { name: /Simpan dan terbitkan/ })
    await userEvent.setup().click(save)

    const slik = await screen.findByLabelText('Nomor SLIK')
    await waitFor(() => expect(slik).toHaveAttribute('aria-invalid', 'true'))

    const report = screen.getByLabelText('Tanggal lapor')
    expect(report).toHaveAttribute('aria-invalid', 'true')

    expect(screen.getByText('2 ketentuan belum terpenuhi:')).toBeInTheDocument()
  })

  it('menandai tombol Back sehingga server tahu klaim tidak maju', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)

    const kembali = await screen.findByRole('button', { name: 'Kembali (Back)' })
    await userEvent.setup().click(kembali)

    await waitFor(() => expect(sentBody).not.toBeNull())
    expect(sentBody).toMatchObject({ kembali: true })
  })
})

describe('pemformatan bersama', () => {
  it('menuliskan uang dari sen dengan pemisah Indonesia', () => {
    expect(formatRupiah(100_000_000)).toBe('Rp 1.000.000,00')
    expect(formatRupiah(0)).toBe('Rp 0,00')
    expect(formatRupiah(123_456)).toBe('Rp 1.234,56')
  })

  it('menuliskan persentase empat desimal tanpa nol di belakang', () => {
    expect(formatPercent(1_000_000)).toBe('100%')
    expect(formatPercent(999_999)).toBe('99,9999%')
  })

  // `new Date('2026-06-05')` ditafsirkan peramban sebagai tengah malam UTC, sehingga
  // pengguna di WIB melihat 4 Juni. Kelas kesalahan itu tidak boleh lahir kembali.
  it('menuliskan tanggal tanpa pernah bergeser sehari', () => {
    expect(formatDate('2026-06-05')).toBe('5 Juni 2026')
    expect(formatDate('2026-01-01')).toBe('1 Januari 2026')
    expect(formatDate('')).toBe('—')
  })

  it('membaca rupiah yang diketik dengan pemisah Indonesia menjadi sen', () => {
    expect(rupiahToCents('1.000.000')).toBe(100_000_000)
    expect(rupiahToCents('1.234,56')).toBe(123_456)
    expect(rupiahToCents('')).toBe(0)
    expect(Number.isNaN(rupiahToCents('bukan angka'))).toBe(true)
  })
})
