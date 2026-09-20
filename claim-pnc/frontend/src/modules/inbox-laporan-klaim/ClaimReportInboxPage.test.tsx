import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimReportInboxPage } from './ClaimReportInboxPage'

const CATEGORY = [
  { kode: 'outstanding', judul: 'Outstanding Data', jumlah: 4, komunikasi: false },
  { kode: 'belum-registrasi', judul: 'Unregistered data', jumlah: 3, komunikasi: false },
  { kode: 'belum-diserahkan', judul: "Data hasn't been transferred", jumlah: 4, komunikasi: false },
  { kode: 'sudah-akseptasi', judul: 'Data has been accepted', jumlah: 2, komunikasi: false },
  // Tab ini TIDAK punya lencana di sistem lama — kueri pencacahnya justru mengecualikan
  // berkas yang ditolak. `jumlah: null` menyatakan "tidak dihitung", bukan "nol".
  { kode: 'ditolak', judul: 'Data rejected', jumlah: null, komunikasi: false },
  { kode: 'komunikasi-belum-dijawab', judul: 'Not answered communication', jumlah: 1, komunikasi: true },
  { kode: 'komunikasi-menunggu-asm', judul: 'Not replied from ASM', jumlah: 2, komunikasi: true },
  { kode: 'komunikasi-dijawab-asm', judul: 'Replied from ASM', jumlah: 0, komunikasi: true },
  { kode: 'semua', judul: 'All data', jumlah: 11, komunikasi: false },
]

const OPTIONS = {
  portal: 'ASM',
  kategori: CATEGORY.map((c) => ({ ...c, jumlah: null })),
  bisnis: [
    { kode: '', nama: 'Semua bisnis' },
    { kode: 'pa', nama: 'Personal Accident' },
    { kode: 'travel', nama: 'Travel' },
  ],
  kanwil: [
    { kode: '01', nama: 'Kanwil Jakarta' },
    { kode: '02', nama: 'Kanwil Jawa Barat' },
  ],
}

function report(over: Partial<Report> = {}): Report {
  return {
    id: 'RCV-0001',
    nomor_klaim: 'PNC-1801',
    nomor_polis: 'POL-PA-0001',
    tertanggung: 'Tertanggung Contoh A',
    nama_bisnis: 'Personal Accident',
    nomor_rujukan: 'REF-0001',
    tanggal_kejadian: '2026-09-07',
    tanggal_masuk: '2026-09-09',
    tanggal_aging: '2026-09-09',
    umur_hari: 10,
    pembuat: 'adminpnc',
    kode_cabang: '1001',
    nama_cabang: 'Cabang Contoh Jakarta 1',
    alasan: '',
    subjek_email: '',
    pesan_akhir: '',
    posisi: 'Outstanding',
    asal: 'pega',
    rujukan_pega: '',
    ...over,
  }
}

type Report = {
  id: string
  nomor_klaim: string
  nomor_polis: string
  tertanggung: string
  nama_bisnis: string
  nomor_rujukan: string
  tanggal_kejadian: string
  tanggal_masuk: string
  tanggal_aging: string
  umur_hari: number
  pembuat: string
  kode_cabang: string
  nama_cabang: string
  alasan: string
  subjek_email: string
  pesan_akhir: string
  posisi: string
  asal: string
  rujukan_pega: string
}

function listBody(over: { laporan?: Report[]; halaman?: Partial<Page> } = {}) {
  return {
    portal: 'ASM',
    kategori: CATEGORY,
    laporan: over.laporan ?? [report()],
    halaman: { halaman: 1, ukuran: 10, total: 4, total_halaman: 1, ...over.halaman },
  }
}

type Page = { halaman: number; ukuran: number; total: number; total_halaman: number }

type Call = { url: string; method: string; header: Record<string, string> }

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

/** defaultReply melayani pilihan dan daftar; sisanya dijawab pemanggil. */
function defaultReply(extra?: (call: Call) => Reply | undefined) {
  return (call: Call): Reply => {
    const custom = extra?.(call)
    if (custom) return custom
    if (call.url.includes('/pilihan')) return { body: OPTIONS }
    if (call.method === 'GET') return { body: listBody() }
    return { body: { laporan: report({ id: 'RCVN.26.0001' }), portal: 'ASM' }, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ClaimReportInboxPage />
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

describe('daftar Inbox Laporan Klaim', () => {
  it('menggambar kesembilan tab dengan judul persis seperti layar Pega', async () => {
    installFetch(defaultReply())
    show()

    const tabs = await screen.findByRole('navigation', { name: 'Kelompok laporan' })
    for (const judul of [
      'Outstanding Data',
      'Unregistered data',
      "Data hasn't been transferred",
      'Data has been accepted',
      'Data rejected',
      'Not answered communication',
      'Not replied from ASM',
      'Replied from ASM',
      'All data',
    ]) {
      expect(within(tabs).getByRole('button', { name: new RegExp(judul) })).toBeInTheDocument()
    }
  })

  it('menampilkan lencana nol tetapi TIDAK menampilkan lencana yang tidak dihitung', async () => {
    installFetch(defaultReply())
    show()

    const tabs = await screen.findByRole('navigation', { name: 'Kelompok laporan' })

    // "Replied from ASM" berjumlah 0 — lencananya ADA, dan menyatakan tidak ada berkas.
    // Namanya disebut lengkap supaya pembaca layar tidak membacakan angka telanjang.
    expect(
      within(tabs).getByRole('button', { name: 'Replied from ASM, 0 berkas' }),
    ).toBeInTheDocument()
    expect(
      within(tabs).getByRole('button', { name: 'Outstanding Data, 4 berkas' }),
    ).toBeInTheDocument()

    // "Data rejected" tidak dicacah sistem lama. Lencananya TIDAK digambar, karena
    // lencana bertuliskan 0 akan menyatakan hal yang tidak benar.
    expect(within(tabs).getByRole('button', { name: 'Data rejected' })).toBeInTheDocument()
  })

  it('memakai nama kolom layar lama, tanpa alias Pega yang menyesatkan', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    for (const nama of [
      'Case ID',
      'Case PNC',
      'Polis no',
      'Insured Name',
      'Business Name',
      'Date of loss',
      'Input Date',
      'Cabang Klaim',
      'Total Aging',
      'Position',
    ]) {
      expect(within(table).getByRole('columnheader', { name: nama })).toBeInTheDocument()
    }

    // Alias lama — "Kurir" untuk nama bisnis, "UserAdmin" untuk nama cabang — tidak
    // dibawa. Keduanya menyebut hal yang sama sekali lain dari isinya (D-19).
    expect(within(table).queryByRole('columnheader', { name: 'Kurir' })).toBeNull()
    expect(within(table).queryByRole('columnheader', { name: 'UserAdmin' })).toBeNull()
  })

  it('menggambar kolom Last message HANYA pada tab komunikasi', async () => {
    installFetch(defaultReply())
    show()

    let table = await screen.findByRole('table')
    expect(within(table).queryByRole('columnheader', { name: 'Last message' })).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: /Not replied from ASM/ }))

    await waitFor(async () => {
      table = await screen.findByRole('table')
      expect(within(table).getByRole('columnheader', { name: 'Last message' })).toBeInTheDocument()
    })
  })

  it('menandai asal setiap baris supaya berkas Pega dan berkas baru dapat dibedakan', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'GET' && !call.url.includes('/pilihan')
          ? {
              body: listBody({
                laporan: [report(), report({ id: 'RCVN.26.0001', asal: 'claimpnc' })],
              }),
            }
          : undefined,
      ),
    )
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('Pega')).toBeInTheDocument()
    expect(within(table).getByText('Baru')).toBeInTheDocument()
  })
})

describe('penyaring dan halaman', () => {
  it('mengirim portal aktif di setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(calls.length).toBeGreaterThan(0)
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
      expect(call.header['Authorization']).toBe('Bearer token-contoh')
    }
  })

  it('berpindah tab mengirim kategorinya dan kembali ke halaman pertama', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'GET' && !call.url.includes('/pilihan')
          ? { body: listBody({ halaman: { total: 40, total_halaman: 4 } }) }
          : undefined,
      ),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    await waitFor(() => expect(lastListCall()).toContain('halaman=2'))

    await userEvent.click(screen.getByRole('button', { name: /All data/ }))

    await waitFor(() => {
      const url = lastListCall()
      expect(url).toContain('kategori=semua')
      // Halaman kembali ke satu: tetap di halaman keempat setelah berpindah tab akan
      // menampilkan daftar kosong pada tab yang isinya lebih sedikit.
      expect(url).not.toContain('halaman=')
    })
  })

  it('memilih kanwil dan bisnis ikut terkirim sebagai penyaring', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    await userEvent.selectOptions(screen.getByLabelText('Pilih Kanwil'), '01')
    await waitFor(() => expect(lastListCall()).toContain('kanwil=01'))

    await userEvent.selectOptions(screen.getByLabelText('Bisnis'), 'pa')
    await waitFor(() => expect(lastListCall()).toContain('bisnis=pa'))
  })

  it('pencarian baru dikirim setelah ditekan, bukan pada setiap huruf', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    const before = calls.length

    await userEvent.type(screen.getByLabelText('Case ID'), 'RCV-0003')
    expect(calls.length).toBe(before)

    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))
    await waitFor(() => expect(lastListCall()).toContain('cari=RCV-0003'))
  })

  it('menyebut letak halaman, bukan hanya nomornya', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'GET' && !call.url.includes('/pilihan')
          ? { body: listBody({ halaman: { total: 57, total_halaman: 6 } }) }
          : undefined,
      ),
    )
    show()

    await screen.findByRole('table')
    expect(screen.getByText(/Menampilkan/)).toBeInTheDocument()
    expect(screen.getByText('57')).toBeInTheDocument()
  })

  it('tombol Sebelumnya mati di halaman pertama', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'GET' && !call.url.includes('/pilihan')
          ? { body: listBody({ halaman: { total: 40, total_halaman: 4 } }) }
          : undefined,
      ),
    )
    show()

    await screen.findByRole('table')
    expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Berikutnya' })).toBeEnabled()
  })
})

describe('tindakan', () => {
  it('Buat Baru mengirim POST tanpa badan dan mengumumkan nomornya', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: /Buat Baru/ }))

    await waitFor(() => {
      expect(calls.some((c) => c.method === 'POST')).toBe(true)
    })
    expect(await screen.findByText(/RCVN\.26\.0001/)).toBeInTheDocument()
  })

  it('kegagalan memuat daftar ditampilkan tanpa mengosongkan layar', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'GET' && !call.url.includes('/pilihan')
          ? { body: { kode: 'galat_internal', pesan: 'Basis data tidak dapat dihubungi.' }, status: 500 }
          : undefined,
      ),
    )
    show()

    expect(
      await screen.findByText('Basis data tidak dapat dihubungi.'),
    ).toBeInTheDocument()
    // Tab tetap tergambar dari daftar pilihan, sehingga pengguna dapat mencoba tab lain
    // alih-alih menghadapi halaman kosong.
    expect(
      await screen.findByRole('navigation', { name: 'Kelompok laporan' }),
    ).toBeInTheDocument()
  })

  it('menuntun memilih portal alih-alih menembak API tanpa entitas', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})

/** lastListCall mengembalikan alamat permintaan daftar terakhir. */
function lastListCall(): string {
  const list = calls.filter((c) => c.method === 'GET' && !c.url.includes('/pilihan'))
  return list[list.length - 1]?.url ?? ''
}
