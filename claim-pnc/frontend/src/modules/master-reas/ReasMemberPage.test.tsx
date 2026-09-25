import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReasMemberPage } from './ReasMemberPage'

/**
 * Seluruh kode, nama perusahaan, login, dan surel di berkas ini KARANGAN, dan domainnya
 * `contoh.invalid` yang RFC 2606 cadangkan supaya tidak pernah dapat diselesaikan DNS
 * (`D-69`).
 */

/** Baris CADANGAN — TYPE '1'. */
const CADANGAN = {
  kode_reas: 'RE-001',
  nama_reas: 'Reasuransi Nusantara Jaya',
  login: 'ReasuransiNusantaraJaya',
  email: 'pla.nusantara@contoh.invalid',
  negara: 'Indonesia',
  tipe: '1',
  cadangan: true,
}

/**
 * Baris KEDUA milik perusahaan yang SAMA, hanya berbeda TYPE.
 *
 * Inilah bentuk yang paling mudah disalahpahami di layar ini: nama yang sama muncul dua
 * kali dengan surel berbeda. Tanpa baris ini di dalam uji, kolom Tipe dan kunci baris tiga
 * kolom tidak terbukti berguna sama sekali.
 */
const TIPE_DUA = {
  kode_reas: 'RE-001',
  nama_reas: 'Reasuransi Nusantara Jaya',
  login: 'ReasuransiNusantaraJaya',
  email: 'dla.nusantara@contoh.invalid',
  negara: 'Indonesia',
  tipe: '2',
  cadangan: false,
}

/** Baris yang LOGIN, EMAIL, dan NEGARA-nya kosong — ketiganya keadaan yang mungkin. */
const TIDAK_LENGKAP = {
  kode_reas: 'RE-003',
  nama_reas: 'Bahtera Reinsurance Ltd',
  login: '',
  email: '',
  negara: '',
  tipe: '',
  cadangan: false,
}

type Call = {
  url: string
  method: string
  header: Record<string, string>
}

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
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

/** defaultReply melayani daftar dengan ketiga baris contoh. */
function defaultReply(): (call: Call) => Reply {
  return () => ({
    body: { member_reas: [CADANGAN, TIPE_DUA, TIDAK_LENGKAP], portal: 'ASM' },
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ReasMemberPage />
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

describe('daftar member reas', () => {
  it('menampilkan keenam kolom dari baris yang dijawab server', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const baris = within(table).getAllByRole('row')

    // Satu baris kepala + tiga baris isi.
    expect(baris).toHaveLength(4)

    expect(within(table).getAllByText('RE-001')).toHaveLength(2)
    expect(within(table).getByText('pla.nusantara@contoh.invalid')).toBeInTheDocument()
    expect(within(table).getByText('dla.nusantara@contoh.invalid')).toBeInTheDocument()
    expect(within(table).getAllByText('Indonesia')).toHaveLength(2)
  })

  it('menampilkan dua baris terpisah untuk satu perusahaan dengan TYPE berbeda', async () => {
    /*
      Kunci baris di layar adalah TIGA kolom — kode + nama + tipe. Bila ia memakai kode reas
      sendirian, React akan menganggap kedua baris ini satu baris dan hanya salah satunya
      yang tampil. Uji ini yang menangkapnya.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const nusantara = within(table)
      .getAllByRole('row')
      .filter((row) => row.textContent?.includes('Reasuransi Nusantara Jaya'))

    expect(nusantara).toHaveLength(2)
  })

  it('menandai baris cadangan, dan tidak menandai yang bukan', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const penanda = within(table).getAllByText('cadangan')

    // Hanya CADANGAN yang bertanda; TIPE_DUA dan TIDAK_LENGKAP tidak.
    expect(penanda).toHaveLength(1)
  })

  it('menyatakan LOGIN dan EMAIL yang kosong, bukan membiarkannya sebagai sel kosong', async () => {
    /*
      Keduanya gagal dalam DIAM di sistem lama: mitra tanpa login tidak akan pernah melihat
      klaimnya sendiri, dan baris tanpa surel membuat dokumen PLA/DLA terbit lalu tidak
      sampai ke siapa pun. Sel kosong tidak menunjukkan keduanya.
    */
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const bahtera = within(table)
      .getAllByRole('row')
      .find((row) => row.textContent?.includes('Bahtera Reinsurance Ltd'))

    expect(bahtera).toBeDefined()
    expect(within(bahtera as HTMLElement).getAllByText('belum ada')).toHaveLength(2)
  })

  it('menyebut portal entitas yang sedang dibuka', async () => {
    installFetch(defaultReply())
    show()

    // Data master dimiliki masing-masing entitas; "data siapa ini" tidak boleh hanya
    // diandaikan pengguna (ADR-0030, R-20).
    expect(await screen.findByText('ASM')).toBeInTheDocument()
  })

  it('mengirim header portal pada permintaan daftar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    const daftar = calls.find((c) => c.url.includes('/api/master/reas'))
    expect(daftar).toBeDefined()
    expect(daftar?.method).toBe('GET')
    expect(Object.values(daftar?.header ?? {})).toContain('ASM')
  })
})

describe('layar baca-saja', () => {
  /*
    Ketiga uji di bawah menjaga keputusan yang berdasar bukti: harness `DataMemberReas`
    memuat satu grid dan satu tombol Refresh, dan satu-satunya penulis T_REINSURER di
    sistem lama adalah alur PLA/DLA lewat `Database/UPDATEREAS.prc` — bukan layar ini.

    Bila kelak terbukti sebaliknya, ketiganya yang akan gagal lebih dulu. Itu memang yang
    diinginkan: penambahan jalur tulis harus menjadi keputusan yang disadari.
  */

  it('tidak punya tombol Tambah maupun Ubah', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    expect(screen.queryByRole('button', { name: 'Tambah' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Ubah' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Simpan' })).toBeNull()
  })

  it('tidak pernah mengirim permintaan yang mengubah data', async () => {
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => {
      expect(calls.length).toBeGreaterThan(1)
    })
    expect(calls.every((c) => c.method === 'GET')).toBe(true)
  })

  it('menjelaskan di kaki halaman dari mana datanya berasal', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText(/Layar ini hanya menampilkan/)).toBeInTheDocument()
  })
})

describe('penolakan dan kegagalan', () => {
  it('meminta portal dipilih lebih dulu, tanpa menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()

    // Menembak server hanya untuk menerima penolakan akan menampilkan pesan galat pada
    // layar yang sebenarnya belum siap dibuka.
    expect(calls.filter((c) => c.url.includes('/api/master/reas'))).toHaveLength(0)
  })

  it('menjelaskan basis data entitas yang belum tersedia, bukan menyuruh mengulang', async () => {
    installFetch(() => ({
      body: { kode: 'portal_belum_siap', pesan: 'Portal belum siap.' },
      status: 503,
    }))
    show()

    expect(
      await screen.findByText('Basis data entitas ini belum tersedia'),
    ).toBeInTheDocument()
  })

  it('tidak menampilkan daftar kosong ketika pemuatannya gagal', async () => {
    /*
      Daftar kosong dan kegagalan membaca terlihat sama bila galatnya ditelan, dan pengguna
      tidak punya cara membedakan "belum ada mitra" dari "tabelnya tidak terbaca".
    */
    installFetch(() => ({
      body: { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' },
      status: 500,
    }))
    show()

    expect(await screen.findByText(/tidak dapat dimuat|kesalahan pada sistem/)).toBeInTheDocument()
    expect(screen.queryByRole('table')).toBeNull()
  })
})
