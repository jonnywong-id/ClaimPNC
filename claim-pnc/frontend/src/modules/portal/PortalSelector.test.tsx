import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { PortalList, PortalSelector } from './PortalSelector'

// Enam entitas, sesuai isi POOLDATA.M_PORTAL_PNC yang diekspor Work Owner. Tiga di
// antaranya belum punya kredensial basis data.
const PORTAL_LIST = {
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

let calledPaths: string[] = []
let usedHeaders: Record<string, string> = {}

function stubFetch(body: unknown = PORTAL_LIST, status = 200) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calledPaths.push(url)
    usedHeaders = (init?.headers as Record<string, string>) ?? {}
    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function mount(component: React.ReactNode) {
  const apiClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={apiClient}>{component}</QueryClientProvider>)
}

function setSession() {
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
}

beforeEach(() => {
  calledPaths = []
  usedHeaders = {}
  window.sessionStorage.clear()
  useSession.getState().cleanup()
  useSelectedPortal.getState().cleanup()
  setSession()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('pemilih portal', () => {
  it('menampilkan nama portal dari tabel, bukan daftar di kode', async () => {
    stubFetch()
    mount(<PortalSelector />)

    const options = await screen.findByLabelText('Portal')
    expect(within(options).getByText('ASURANSI SINAR MAS')).toBeInTheDocument()
    expect(within(options).getByText('ASURANSI SIMAS INSURTECH')).toBeInTheDocument()
    expect(within(options).getByText(/SINARMAS PENJAMINAN KREDIT SYARIAH/)).toBeInTheDocument()
  })

  // Portal yang kredensial basis datanya belum diisi tetap terlihat — ditandai, bukan
  // disembunyikan — supaya pengguna tahu entitas itu direncanakan.
  it('menandai portal yang belum siap dan melarang memilihnya', async () => {
    stubFetch()
    mount(<PortalSelector />)

    const options = await screen.findByLabelText('Portal')
    const notReady = within(options).getByText(/SINARMAS ASURANSI SYARIAH — belum tersedia/)
    expect(notReady).toBeInTheDocument()
    expect(notReady).toBeDisabled()

    const siap = within(options).getByText('ASURANSI SIMAS INSURTECH')
    expect(siap).not.toBeDisabled()
  })

  // Portal awal mengikuti yang disebut server, bukan tebakan di frontend.
  it('memilih portal utama dari server sebagai pilihan awal', async () => {
    stubFetch()
    mount(<PortalSelector />)

    await waitFor(() => expect(useSelectedPortal.getState().alias).toBe('ASM'))
  })

  it('menyimpan pilihan portal sehingga bertahan melewati muat ulang', async () => {
    stubFetch()
    mount(<PortalSelector />)

    const options = await screen.findByLabelText('Portal')
    await userEvent.setup().selectOptions(options, 'SPK')

    expect(useSelectedPortal.getState().alias).toBe('SPK')
    expect(window.sessionStorage.getItem('claim-pnc.portal')).toBe('SPK')
  })

  it('mengirim token sesi saat meminta daftar portal', async () => {
    stubFetch()
    mount(<PortalSelector />)

    await waitFor(() => expect(calledPaths).toContain('/api/portal'))
    expect(usedHeaders['Authorization']).toBe('Bearer token-contoh')
  })

  it('menampilkan pesan ketika daftar portal gagal dimuat', async () => {
    stubFetch({ kode: 'galat_internal', pesan: 'gagal' }, 500)
    mount(<PortalSelector />)

    expect(await screen.findByText('Daftar portal tidak dapat dimuat.')).toBeInTheDocument()
  })
})

describe('daftar portal', () => {
  it('menampilkan keenam entitas beserta status kesiapannya', async () => {
    stubFetch()
    mount(<PortalList />)

    const row = await screen.findAllByRole('row')
    // Satu baris kepala + enam entitas.
    expect(row).toHaveLength(7)
    expect(screen.getAllByText('Tersedia')).toHaveLength(3)
    expect(screen.getAllByText('Menunggu kredensial basis data')).toHaveLength(3)
  })
})
