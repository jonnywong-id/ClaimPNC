import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { gunakanPortalTerpilih } from '@/app/portal'
import { gunakanSesi } from '@/app/sesi'

import { DaftarPortal, PemilihPortal } from './PemilihPortal'

// Enam entitas, sesuai isi POOLDATA.M_PORTAL_PNC yang diekspor Work Owner. Tiga di
// antaranya belum punya kredensial basis data.
const DAFTAR_PORTAL = {
  portal: [
    { id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true },
    { id: '202600102', nama: 'ASURANSI SIMAS INSURTECH', alias: 'ASI', siap: true },
    { id: '202600103', nama: 'SINARMAS ASURANSI SYARIAH', alias: 'SMAS', siap: false },
    { id: '202600104', nama: 'SINARMAS INSURANCE', alias: 'SMI', siap: false },
    { id: '202600105', nama: 'SINARMAS PENJAMINAN KREDIT', alias: 'SPK', siap: true },
    { id: '202600106', nama: 'SINARMAS PENJAMINAN KREDIT SYARIAH', alias: 'SPKS', siap: false },
  ],
  utama: 'ASM',
}

let jalurDipanggil: string[] = []
let headerDipakai: Record<string, string> = {}

function pasangFetch(badan: unknown = DAFTAR_PORTAL, status = 200) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    jalurDipanggil.push(url)
    headerDipakai = (init?.headers as Record<string, string>) ?? {}
    return Promise.resolve(
      new Response(JSON.stringify(badan), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function tampilkan(komponen: React.ReactNode) {
  const klien = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={klien}>{komponen}</QueryClientProvider>)
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
  jalurDipanggil = []
  headerDipakai = {}
  window.sessionStorage.clear()
  gunakanSesi.getState().bersihkan()
  gunakanPortalTerpilih.getState().bersihkan()
  masukkanSesi()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('pemilih portal', () => {
  it('menampilkan nama portal dari tabel, bukan daftar di kode', async () => {
    pasangFetch()
    tampilkan(<PemilihPortal />)

    const pilihan = await screen.findByLabelText('Portal')
    expect(within(pilihan).getByText('ASURANSI SINAR MAS')).toBeInTheDocument()
    expect(within(pilihan).getByText('ASURANSI SIMAS INSURTECH')).toBeInTheDocument()
    expect(within(pilihan).getByText(/SINARMAS PENJAMINAN KREDIT SYARIAH/)).toBeInTheDocument()
  })

  // Portal yang kredensial basis datanya belum diisi tetap terlihat — ditandai, bukan
  // disembunyikan — supaya pengguna tahu entitas itu direncanakan.
  it('menandai portal yang belum siap dan melarang memilihnya', async () => {
    pasangFetch()
    tampilkan(<PemilihPortal />)

    const pilihan = await screen.findByLabelText('Portal')
    const belumSiap = within(pilihan).getByText(/SINARMAS ASURANSI SYARIAH — belum tersedia/)
    expect(belumSiap).toBeInTheDocument()
    expect(belumSiap).toBeDisabled()

    const siap = within(pilihan).getByText('ASURANSI SIMAS INSURTECH')
    expect(siap).not.toBeDisabled()
  })

  // Portal awal mengikuti yang disebut server, bukan tebakan di frontend.
  it('memilih portal utama dari server sebagai pilihan awal', async () => {
    pasangFetch()
    tampilkan(<PemilihPortal />)

    await waitFor(() => expect(gunakanPortalTerpilih.getState().alias).toBe('ASM'))
  })

  it('menyimpan pilihan portal sehingga bertahan melewati muat ulang', async () => {
    pasangFetch()
    tampilkan(<PemilihPortal />)

    const pilihan = await screen.findByLabelText('Portal')
    await userEvent.setup().selectOptions(pilihan, 'SPK')

    expect(gunakanPortalTerpilih.getState().alias).toBe('SPK')
    expect(window.sessionStorage.getItem('claim-pnc.portal')).toBe('SPK')
  })

  it('mengirim token sesi saat meminta daftar portal', async () => {
    pasangFetch()
    tampilkan(<PemilihPortal />)

    await waitFor(() => expect(jalurDipanggil).toContain('/api/portal'))
    expect(headerDipakai['Authorization']).toBe('Bearer token-contoh')
  })

  it('menampilkan pesan ketika daftar portal gagal dimuat', async () => {
    pasangFetch({ kode: 'galat_internal', pesan: 'gagal' }, 500)
    tampilkan(<PemilihPortal />)

    expect(await screen.findByText('Daftar portal tidak dapat dimuat.')).toBeInTheDocument()
  })
})

describe('daftar portal', () => {
  it('menampilkan keenam entitas beserta status kesiapannya', async () => {
    pasangFetch()
    tampilkan(<DaftarPortal />)

    const baris = await screen.findAllByRole('row')
    // Satu baris kepala + enam entitas.
    expect(baris).toHaveLength(7)
    expect(screen.getAllByText('Tersedia')).toHaveLength(3)
    expect(screen.getAllByText('Menunggu kredensial basis data')).toHaveLength(3)
  })
})
