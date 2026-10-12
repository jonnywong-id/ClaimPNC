import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReportKlaimPage } from './ReportKlaimPage'

/**
 * Katalog contoh — bentuknya sama persis dengan yang dikirim backend: DATAR dan sudah
 * berurutan menurut urutan layar Pega.
 *
 * Isinya mewakili lima keadaan yang membedakan satu baris dari yang lain:
 *
 *	tat               memakai rentang tanggal DAN lini bisnis
 *	adjuster          TERHALANG
 *	komite            bertombol DUA
 *	klaim-per-bisnis  satu-satunya yang punya autocomplete Bisnis di barisnya
 *	akseptasi         satu-satunya yang punya kotak centang "Treaty" di barisnya
 */
const CATALOG = {
  judul: 'Report Claim',
  laporan: [
    {
      kode: 'tat',
      judul: 'REPORT TAT',
      tombol: [{ kode: '', label: 'Export Data TAT' }],
      penyaring: {
        rentang_tanggal: true,
        lini_bisnis: true,
        status_compliance: false,
        bisnis: false,
        rincian: false,
      },
      tersedia: true,
      sumber: { activity: 'PNCTATReport1_Act' },
    },
    {
      kode: 'adjuster',
      judul: 'REPORT ADJUSTER',
      tombol: [{ kode: '', label: 'Export Data Adjuster' }],
      penyaring: {
        rentang_tanggal: true,
        lini_bisnis: false,
        status_compliance: false,
        bisnis: false,
        rincian: false,
      },
      tersedia: false,
      alasan: 'Report Definition-nya tidak ada di export Pega.',
      penghalang: 'R-16',
      sumber: { activity: 'PNCAdjusterReport_Act' },
    },
    {
      kode: 'klaim-per-bisnis',
      judul: 'REPORT KLAIM PER BISNIS',
      tombol: [{ kode: '', label: 'Export Data' }],
      penyaring: {
        rentang_tanggal: false,
        lini_bisnis: false,
        status_compliance: false,
        bisnis: true,
        rincian: false,
      },
      tersedia: true,
      sumber: { activity: 'ExportDataClaimBusiness' },
    },
    {
      kode: 'akseptasi',
      judul: 'REPORT AKSEPTASI',
      tombol: [{ kode: '', label: 'Export Data Akseptasi' }],
      penyaring: {
        rentang_tanggal: true,
        lini_bisnis: false,
        status_compliance: false,
        bisnis: false,
        rincian: true,
      },
      tersedia: true,
      sumber: { activity: 'ReportAkseptasiNonMBU' },
    },
    {
      kode: 'komite',
      judul: 'REPORT DATA KOMITE',
      tombol: [
        { kode: 'approve', label: 'Export Data Approve' },
        { kode: 'rejected', label: 'Export Data Rejected' },
      ],
      penyaring: {
        rentang_tanggal: true,
        lini_bisnis: true,
        status_compliance: false,
        bisnis: false,
        rincian: false,
      },
      tersedia: true,
      sumber: { activity: 'PNCReportDataKomites_act' },
    },
  ],
  lini_bisnis: [
    { nilai: '', nama: '----- Pilih -----' },
    { nilai: '002', nama: 'Personal Accident' },
    { nilai: '346', nama: 'Non-MBU' },
  ],
  status_compliance: [
    { nilai: '0', nama: 'Fraud / ditolak' },
    { nilai: '1', nama: 'Valid / Bayar' },
    { nilai: '2', nama: 'Post Audit / Bayar' },
  ],
}

const BISNIS = {
  bisnis: [{ kode: '10076', nama: 'Contoh Bisnis Aneka' }],
}

type Call = { url: string; header: Record<string, string> }

let calls: Call[] = []

function installFetch() {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, header: (init?.headers as Record<string, string>) ?? {} })

    if (url.includes('/pilihan-bisnis')) {
      return Promise.resolve(
        new Response(JSON.stringify(BISNIS), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }
    if (url.includes('/ekspor')) {
      return Promise.resolve(
        new Response('No Klaim\nPNCN.26.0001\n', {
          status: 200,
          headers: {
            'Content-Type': 'text/csv; charset=utf-8',
            'Content-Disposition': 'attachment; filename="Laporan Data PLA.csv"',
          },
        }),
      )
    }
    return Promise.resolve(
      new Response(JSON.stringify(CATALOG), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/report-klaim']}>
        <ReportKlaimPage />
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

  // simpanBerkas membuat tautan lalu menekannya; jsdom tidak dapat bernavigasi.
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('layar Report Klaim', () => {
  it('menggambar judul dan daftar laporan', async () => {
    installFetch()
    show()

    expect(await screen.findByRole('heading', { name: 'REPORT TAT', level: 2 })).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Report Claim', level: 1 })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Export Data TAT' })).toBeEnabled()
  })

  // Daftarnya DATAR dan berurutan — tidak ada judul kelompok, dan urutannya tidak diubah.
  // Itu susunan yang dihafal pengguna di layar lama.
  it('tidak mengelompokkan laporan dan tidak mengurutkannya ulang', async () => {
    installFetch()
    show()

    await screen.findByRole('heading', { name: 'REPORT TAT', level: 2 })

    expect(screen.queryByRole('heading', { name: 'Klaim', level: 2 })).toBeNull()
    expect(screen.queryByRole('heading', { name: 'Akseptasi & Penyelesaian' })).toBeNull()

    const judul = screen
      .getAllByRole('heading', { level: 2 })
      .map((h) => h.textContent)
    expect(judul).toEqual([
      'REPORT TAT',
      'REPORT ADJUSTER',
      'REPORT KLAIM PER BISNIS',
      'REPORT AKSEPTASI',
      'REPORT DATA KOMITE',
    ])
  })

  // Dropdown lini bisnis berlabel "Bisnis", bukan "Treaty". Yang berlabel "Treaty" adalah
  // kotak centang pada panel Akseptasi.
  it('memberi label Bisnis pada dropdown lini bisnis', async () => {
    installFetch()
    show()

    expect(await screen.findByLabelText('Bisnis')).toBeVisible()
    expect(screen.queryByLabelText('Treaty')).toBeNull()
  })

  // Status Compliance adalah TIGA radio, bukan dropdown — `pxRadioButtons` di harness.
  it('menggambar Status Compliance sebagai tiga radio', async () => {
    installFetch()
    show()

    await screen.findByRole('heading', { name: 'REPORT TAT', level: 2 })

    const radio = screen.getAllByRole('radio')
    expect(radio).toHaveLength(3)
    expect(screen.getByLabelText('Fraud / ditolak')).toBeVisible()
    expect(screen.getByLabelText('Valid / Bayar')).toBeVisible()
    expect(screen.getByLabelText('Post Audit / Bayar')).toBeVisible()
  })

  it('mengirim kode Status Compliance, bukan labelnya', async () => {
    installFetch()
    show()

    await screen.findByRole('button', { name: 'Export Data TAT' })
    await userEvent.click(screen.getByLabelText('Post Audit / Bayar'))

    expect((screen.getByLabelText('Post Audit / Bayar') as HTMLInputElement).value).toBe('2')
  })

  // Kotak centang "Treaty" berada DI BARIS panel Akseptasi, bukan di bilah penyaring.
  it('menempatkan kotak centang Treaty pada baris Akseptasi saja', async () => {
    installFetch()
    show()

    const akseptasi = (
      await screen.findByRole('heading', { name: 'REPORT AKSEPTASI', level: 2 })
    ).closest('article') as HTMLElement
    expect(within(akseptasi).getByLabelText('Treaty')).toBeVisible()

    const tat = screen
      .getByRole('heading', { name: 'REPORT TAT', level: 2 })
      .closest('article') as HTMLElement
    expect(within(tat).queryByLabelText('Treaty')).toBeNull()
  })

  // Autocomplete bisnis berada DI BARIS panel Klaim Per Bisnis, bukan di bilah penyaring.
  it('menempatkan autocomplete Bisnis pada baris Klaim Per Bisnis saja', async () => {
    installFetch()
    show()

    const perBisnis = (
      await screen.findByRole('heading', { name: 'REPORT KLAIM PER BISNIS', level: 2 })
    ).closest('article') as HTMLElement
    expect(within(perBisnis).getByLabelText('Bisnis untuk Report Klaim Per Bisnis')).toBeVisible()

    // Bilah atas tetap satu-satunya yang berlabel "Bisnis" — kontrol baris ini tidak
    // berlabel di layar lama, dan tidak diberi label terlihat di sini.
    expect(screen.getAllByLabelText('Bisnis')).toHaveLength(1)
  })

  // Panel bertombol dua adalah SATU baris, bukan dua baris berjudul sama.
  it('menggambar dua tombol pada panel Data Komite', async () => {
    installFetch()
    show()

    expect(await screen.findByRole('button', { name: 'Export Data Approve' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Export Data Rejected' })).toBeVisible()
  })

  // Panel terhalang TETAP tampil bertanda sebabnya. Menghilangkannya membuat pengguna
  // melaporkan laporan yang "hilang".
  it('menampilkan panel terhalang beserta sebab dan penghalangnya', async () => {
    installFetch()
    show()

    expect(await screen.findByRole('heading', { name: 'REPORT ADJUSTER', level: 2 })).toBeVisible()
    expect(screen.getByText(/Report Definition-nya tidak ada di export/)).toBeVisible()
    expect(screen.getByText('R-16')).toBeVisible()
    expect(screen.getByText('Belum tersedia')).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Export Data Adjuster' })).toBeNull()
  })

  it('mengirim penyaring yang diisi ke alamat ekspor', async () => {
    installFetch()
    show()

    const tombol = await screen.findByRole('button', { name: 'Export Data TAT' })
    await userEvent.type(screen.getByLabelText('Dari'), '2026-09-01')
    await userEvent.type(screen.getByLabelText('Sampai'), '2026-09-30')
    await userEvent.selectOptions(screen.getByLabelText('Bisnis'), '002')
    await userEvent.click(tombol)

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('/report-klaim/tat/ekspor'))).toBe(true)
    })
    const ekspor = calls.find((c) => c.url.includes('/ekspor'))
    expect(ekspor?.url).toContain('dari=2026-09-01')
    expect(ekspor?.url).toContain('sampai=2026-09-30')
    expect(ekspor?.url).toContain('lini=002')
    expect(ekspor?.header['X-Portal']).toBe('ASM')
    expect(ekspor?.header['Authorization']).toBe('Bearer token-contoh')
  })

  // Kode tombol IKUT dikirim pada panel bertombol dua; tanpanya backend menolak, karena
  // memilihkan salah satunya berarti mengunduh laporan yang berbeda isi.
  it('mengirim kode tombol pada panel bertombol dua', async () => {
    installFetch()
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Export Data Rejected' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('/ekspor'))).toBe(true)
    })
    expect(calls.find((c) => c.url.includes('/ekspor'))?.url).toContain('aksi=rejected')
  })

  // Penyaring yang TIDAK berlaku pada sebuah laporan tidak ikut dikirim, meski isiannya
  // sempat terisi untuk laporan lain.
  it('tidak mengirim rentang tanggal untuk laporan yang tidak memakainya', async () => {
    installFetch()
    show()

    await screen.findByRole('button', { name: 'Export Data TAT' })
    await userEvent.type(screen.getByLabelText('Dari'), '2026-09-01')
    await userEvent.click(screen.getByRole('button', { name: 'Export Data' }))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('klaim-per-bisnis/ekspor'))).toBe(true)
    })
    const ekspor = calls.find((c) => c.url.includes('klaim-per-bisnis/ekspor'))
    expect(ekspor?.url).not.toContain('dari=')
  })

  // Daftar bisnis hanya ditarik ketika autocomplete-nya disentuh — ia dapat berisi ratusan
  // baris, dan 27 dari 28 panel tidak memakainya.
  it('menarik pilihan bisnis hanya saat dibutuhkan', async () => {
    installFetch()
    show()

    await screen.findByRole('button', { name: 'Export Data TAT' })
    expect(calls.some((c) => c.url.includes('/pilihan-bisnis'))).toBe(false)

    const perBisnis = screen
      .getByRole('heading', { name: 'REPORT KLAIM PER BISNIS', level: 2 })
      .closest('article') as HTMLElement
    await userEvent.click(within(perBisnis).getByLabelText('Bisnis untuk Report Klaim Per Bisnis'))

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('/pilihan-bisnis'))).toBe(true)
    })
  })

  // Tanpa portal, layar tidak dibuka sama sekali: berkasnya memuat data satu badan hukum
  // dan aplikasi ini melayani empat (`R-20`).
  it('menolak membuka layar sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(screen.getByText('Pilih entitas lebih dulu')).toBeVisible()
    expect(calls).toHaveLength(0)
  })
})
