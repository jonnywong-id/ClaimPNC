import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-claim-treaty-non-prop'
const TAB_PATH = `${PATH}/tab`
const EXPORT_PATH = `${PATH}/ekspor`

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminnonprop1',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Tab antrean milik pemanggil — satu-satunya yang mengenal KEDUA checkbox. */
const TAB_ADMIN: Tab = {
  kode: '1',
  nama: 'Work Treatyin Non Propotional Admin',
  keterangan: 'Klaim treaty non-proporsional yang ditugaskan kepada Anda.',
  kolom: [
    { kunci: 'no_klaim', judul: 'No Klaim' },
    { kunci: 'master_id', judul: 'MasterID' },
    { kunci: 'id_master', judul: 'ID Master' },
    { kunci: 'no_polis', judul: 'Policy No' },
    { kunci: 'tanggal_kejadian', judul: 'Date of Loss' },
    { kunci: 'nama_bisnis', judul: 'Business Name' },
    { kunci: 'ceding_co', judul: 'Ceding Co Name' },
    { kunci: 'nama_tertanggung', judul: 'Insured Name' },
    { kunci: 'status', judul: 'Status' },
    { kunci: 'aging', judul: 'Aging' },
    { kunci: 'operator_pembuat', judul: 'Create Operator' },
  ],
  hanya_milik_saya: true,
  pakai_lihat_semua: true,
  pakai_lihat_tba: true,
  terhalang: false,
}

/** Tab antrean bersama — tidak mengenal satu pun checkbox, dan tanpa kolom "ID Master". */
const TAB_TEKNIK: Tab = {
  kode: '2',
  nama: 'Work Treatyin Non Propotional Teknik',
  keterangan: 'Antrean bersama PIC Teknik treaty — belum diambil siapa pun.',
  kolom: [
    { kunci: 'no_klaim', judul: 'No Klaim' },
    { kunci: 'nama_bisnis', judul: 'Class of Business' },
    { kunci: 'status', judul: 'Status' },
  ],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  pakai_lihat_tba: false,
  terhalang: false,
}

/** Tab yang digambar tetapi belum dapat diisi. */
const TAB_KOMITE: Tab = {
  kode: '3',
  nama: 'Work List Treatyin Non Proportional Komite',
  keterangan: 'Antrean persetujuan komite klaim treaty non-proporsional.',
  kolom: [],
  hanya_milik_saya: false,
  pakai_lihat_semua: false,
  pakai_lihat_tba: false,
  terhalang: true,
  alasan_terhalang:
    'Antrean komite treaty non-proporsional belum dapat dibaca sistem baru. Kuerinya ' +
    'dipanggil GetWorkCNP_Act tetapi tidak ada di export rule.',
  pemilik_penghalang: 'Tim Pega — dibutuhkan export rule KmtGetInboxListCNP_SQL.',
}

const METADATA: MetadataResponse = {
  tab: [TAB_ADMIN, TAB_TEKNIK, TAB_KOMITE],
  tab_bawaan: '1',
  selisih_terencana: [
    'Checkbox "See TBA Claim" kini BERDIRI SENDIRI.',
    'Kolom "Status" menyatakan ANTREAN, bukan status klaim.',
  ],
  portal: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const ROW: WorkItem = {
  referensi: 'ASSIGN-WORKLIST CLMNP-1001!FLOW',
  no_klaim: 'CLMNP-1001',
  master_id: 'TNP-2026-01',
  id_master: 'TNP-2026-01',
  no_polis: '99.002.2026.00001',
  tanggal_kejadian: '2026-08-14',
  nama_bisnis: 'Property All Risk',
  sumber_bisnis: 'Treaty Inward',
  ceding_co: 'Asuransi Contoh Pertama',
  nama_tertanggung: 'PT Contoh Sejahtera',
  status: 'Estimation',
  aging: 0,
  operator_pembuat: 'ADMINNONPROP1',
  operator_pengubah: 'ADMINNONPROP1',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

/** Peladen tiruan yang menjawab bentuk layar dan satu baris antrean. */
function stubDefaultFetch(rows: WorkItem[] = [ROW], tab: Tab = TAB_ADMIN) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('No Klaim\nCLMNP-1001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="klaim-treaty-non-prop-tab1.csv"',
        },
      })
    }
    return jsonResponse(200, {
      tab,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 25, total: rows.length, total_halaman: 1 },
      penyaring: { lihat_semua: false, lihat_tba: false },
      portal: 'ASM',
    })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-claim-treaty-non-prop']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada salah satu tab, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara bilah tab baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: /Non Propotional Admin/ })
}

function lastListCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url === PATH || c.url.startsWith(`${PATH}?`))
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('bentuk layar', () => {
  it('membuka tab bawaan yang ditetapkan server, yaitu antrean milik pemanggil', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Non Propotional Admin/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('mempertahankan kedua ejaan judul tab seperti di Pega (D-13)', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Section yang SAMA memakai dua ejaan: "Propotional" pada dua tab pertama dan
    // "Proportional" pada tab komite. Keduanya dibawa apa adanya — menyeragamkannya
    // berarti memilih salah satu yang benar tanpa dasar.
    expect(screen.getByRole('tab', { name: /Non Propotional Admin/ })).toBeInTheDocument()
    expect(
      screen.getByRole('tab', { name: /Non Proportional Komite/ }),
    ).toBeInTheDocument()
  })

  it('menggambar kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const column of TAB_ADMIN.kolom) {
      expect(
        await screen.findByRole('columnheader', { name: column.judul }),
      ).toBeInTheDocument()
    }
  })

  it('menggambar KEDUA kolom master id, karena keduanya dari sumber berbeda', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // "MasterID" dari kolom objek kerja, "ID Master" dari blob JSON. Apakah isinya selalu
    // sama belum pernah diperiksa (`R-08`), sehingga keduanya dibawa apa adanya.
    expect(await screen.findByRole('columnheader', { name: 'MasterID' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'ID Master' })).toBeInTheDocument()
  })
})

describe('kedua checkbox', () => {
  it('menggambar keduanya hanya pada tab yang mengenalnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByLabelText('See All Claim')).toBeInTheDocument()
    expect(screen.getByLabelText('See TBA Claim')).toBeInTheDocument()
  })

  it('mengirim lihat_tba TANPA lihat_semua — selisih terencana P-5', async () => {
    // Inilah perbaikan yang disetujui Work Owner 2026-09-22. Di layar Pega, mencentang
    // "See TBA Claim" sendirian tidak mengubah apa pun: kueri TBA-nya dijaga dua
    // prakondisi ber-AND. Di sini ia berdiri sendiri.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See TBA Claim'))

    const call = lastListCall()
    expect(call?.url).toContain('lihat_tba=1')
    expect(call?.url).not.toContain('lihat_semua')
  })

  it('mengirim keduanya saat dicentang bersama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See All Claim'))
    await userEvent.click(screen.getByLabelText('See TBA Claim'))

    const call = lastListCall()
    expect(call?.url).toContain('lihat_semua=1')
    expect(call?.url).toContain('lihat_tba=1')
  })

  it('membersihkan kedua centang saat berpindah tab', async () => {
    // Membawa centang ke tab yang tidak mengenalnya akan membuat centang yang tampak
    // aktif padahal tidak mengubah apa pun.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See All Claim'))
    await userEvent.click(screen.getByRole('tab', { name: /Non Propotional Teknik/ }))

    const call = lastListCall()
    expect(call?.url).not.toContain('lihat_semua')
    expect(call?.url).not.toContain('lihat_tba')
  })

  it('tidak menggambar satu pun checkbox pada tab yang tidak mengenalnya', async () => {
    stubDefaultFetch([ROW], TAB_TEKNIK)
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Non Propotional Teknik/ }))

    expect(screen.queryByLabelText('See All Claim')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('See TBA Claim')).not.toBeInTheDocument()
  })
})

describe('tab yang terhalang', () => {
  it('menampilkan alasan dan pemiliknya, bukan tabel kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Non Proportional Komite/ }))

    expect(await screen.findByText(/tidak ada di export rule/)).toBeInTheDocument()
    expect(screen.getByText(/KmtGetInboxListCNP_SQL/)).toBeInTheDocument()
  })

  it('TIDAK meminta isinya ke server', async () => {
    // Permintaannya pasti ditolak 422, dan galat yang sudah diketahui pasti terjadi bukan
    // galat yang layak ditampilkan.
    stubDefaultFetch()
    await renderLoaded()

    calls = []
    await userEvent.click(screen.getByRole('tab', { name: /Non Proportional Komite/ }))
    await screen.findByText(/tidak ada di export rule/)

    expect(lastListCall()).toBeUndefined()
  })
})

describe('tombol ekspor', () => {
  it('mengirim penyaring yang SAMA dengan yang sedang tampil', async () => {
    // Ekspor yang mengabaikan penyaring akan mengeluarkan berkas yang isinya tidak dapat
    // dicocokkan dengan apa pun di layar.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByLabelText('See TBA Claim'))
    await userEvent.click(screen.getByRole('button', { name: 'Export Data' }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    expect(ekspor?.url).toContain('lihat_tba=1')
  })

  it('membawa token dan portal sebagai header, bukan di dalam alamat', async () => {
    // `<a href>` tidak membawa header, dan menaruh token di URL akan meninggalkan
    // jejaknya di riwayat peramban, log proxy, dan header Referer.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(await screen.findByRole('button', { name: 'Export Data' }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    const header = ekspor?.init?.headers as Record<string, string> | undefined

    expect(header?.['Authorization']).toBe('Bearer token-uji')
    expect(ekspor?.url).not.toContain('token-uji')
  })

  it('dimatikan saat antreannya kosong', async () => {
    // Berkas kosong yang tetap terunduh tidak dapat dibedakan pengguna dari ekspor yang
    // gagal diam-diam.
    stubDefaultFetch([])
    await renderLoaded()

    expect(await screen.findByRole('button', { name: 'Export Data' })).toBeDisabled()
  })
})

describe('penggambaran sel', () => {
  it('menampilkan aging nol sebagai angka, bukan sebagai tanda pisah', async () => {
    // Nol hari adalah nilai yang SAH dan berarti "masuk hari ini". Mengubahnya menjadi
    // "—" akan menyamakannya dengan kolom yang gagal dimuat.
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('0 hari')).toBeInTheDocument()
  })

  it('menampilkan selisih terencana yang dikirim server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText(/See TBA Claim" kini BERDIRI SENDIRI/)).toBeInTheDocument()
  })
})

describe('tombol buat klaim', () => {
  it('menolak dengan alasan, bukan dengan "halaman tidak ditemukan"', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url === `${PATH}/klaim`) {
        return jsonResponse(501, {
          kode: 'belum_tersedia',
          pesan:
            'Pembuatan klaim treaty non-prop belum tersedia di sistem baru. Buat klaim ' +
            'treaty non-prop baru lewat Pega.',
        })
      }
      return jsonResponse(200, {
        tab: TAB_ADMIN,
        baris: [ROW],
        paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
        penyaring: { lihat_semua: false, lihat_tba: false },
        portal: 'ASM',
      })
    })
    await renderLoaded()

    await userEvent.click(
      screen.getByRole('button', { name: 'Create Claim Treaty Non Prop' }),
    )

    expect(await screen.findByText(/lewat Pega/)).toBeInTheDocument()
  })
})
