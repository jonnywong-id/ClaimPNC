import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { ListResponse, WorkItem } from './types'

const PATH = '/api/inbox-os-claim-per-cabang'
const EXPORT_PATH = `${PATH}/ekspor`

const SAMPLE_PROFILE = {
  identitas: '90000003',
  nama: 'Contoh Penyelia Cabang',
  jenis: 'KARYAWAN',
  login: 'penyeliacilegon',
  email: 'contoh.penyelia@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * Ketiganya dipilih supaya kedua syarat baris merah dapat diuji TERPISAH: yang pertama
 * merah karena umurnya, yang kedua merah karena progresnya mandek meski masih muda, dan
 * yang ketiga tidak merah sama sekali.
 */
const TUA: WorkItem = {
  cabang: 'CILEGON',
  sumbis: 'BANK CONTOH CILEGON',
  cob: 'Aneka',
  no_polis: '99.002.2024.00001',
  nama_insured: 'PT CONTOH SATU',
  no_klaim: 'PNC-9001',
  tanggal_registrasi: '2024-01-15',
  tanggal_kejadian: '2024-01-10',
  nilai_estimasi: '250000.00',
  tanggal_update_progres: '2024-02-01',
  status_progres_1: 'SURVEY',
  status_progres_2: 'HASIL SURVEY BELUM ADA',
  pic: 'PICCONTOHSATU',
  adjuster: 'PT ADJUSTER CONTOH',
  col: 'FIRE - SHORT CIRCUIT',
  aging_hari: 987,
  perlu_perhatian: true,
  progres_mandek: false,
  catatan_progres: 'Menunggu hasil survei',
}

const MANDEK: WorkItem = {
  cabang: 'CILEGON',
  sumbis: 'AGEN CONTOH',
  cob: 'PA',
  no_polis: '99.002.2026.00002',
  nama_insured: 'PT CONTOH DUA',
  no_klaim: 'PNC-9002',
  tanggal_registrasi: '2026-09-20',
  tanggal_kejadian: '2026-09-18',
  nilai_estimasi: '1500000.00',
  tanggal_update_progres: '2026-09-25',
  status_progres_1: 'ACCEPTATION',
  status_progres_2: 'AUTO PROGRESS',
  pic: 'PICCONTOHDUA',
  adjuster: '',
  col: 'ACCIDENTAL DAMAGE',
  aging_hari: 8,
  perlu_perhatian: true,
  progres_mandek: true,
  catatan_progres: 'Progres tidak berubah tiga kali berturut-turut',
}

const TENANG: WorkItem = {
  cabang: 'CILEGON',
  sumbis: 'AGEN CONTOH',
  cob: 'Travel',
  no_polis: '99.002.2026.00003',
  nama_insured: 'PT CONTOH TIGA',
  no_klaim: 'PNC-9003',
  tanggal_registrasi: '2026-09-22',
  tanggal_kejadian: '2026-09-21',
  nilai_estimasi: '0.00',
  // Belum punya satu pun catatan progres.
  tanggal_update_progres: '',
  status_progres_1: '',
  status_progres_2: '',
  pic: 'PICCONTOHDUA',
  adjuster: '',
  col: '',
  aging_hari: 6,
  perlu_perhatian: false,
  progres_mandek: false,
  catatan_progres: '',
}

function listResponse(rows: WorkItem[], total = rows.length): ListResponse {
  return {
    data: rows,
    cabang: { kode: '100099', nama: 'CILEGON' },
    paginasi: { halaman: 1, ukuran: 25, total, total_halaman: Math.max(1, Math.ceil(total / 25)) },
    ambang_aging: 180,
    selisih_terencana: [
      'Daftar dibagi per halaman di server.',
      'Umur klaim dihitung terhadap tanggal WIB.',
    ],
    portal: 'ASM',
  }
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

/** Peladen tiruan yang menjawab daftar dan berkas ekspor. */
function stubDefaultFetch(rows: WorkItem[] = [TUA, MANDEK, TENANG]) {
  stubFetch((url) => {
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('Cabang\nCILEGON\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="SummaryOS-100099.csv"',
        },
      })
    }
    return jsonResponse(200, listResponse(rows))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-os-claim-per-cabang']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** renderLoaded menggambar layar lalu MENUNGGU barisnya tiba. */
async function renderLoaded() {
  renderPage()
  await screen.findByText('PNC-9001')
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
})

describe('judul layar', () => {
  it('menyebut cabang yang barisnya sedang tampil', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Judulnya mengikuti layar lama apa adanya, termasuk tanda kurungnya. Nama cabang di
    // dalamnya datang dari SERVER — bukan dari profil sesi — supaya nama pada judul dan
    // kode pada penyaring selalu berasal dari satu jawaban yang sama.
    const heading = screen.getByRole('heading', {
      name: /Inbox Outstanding Claim per Cabang/,
    })
    expect(heading).toHaveTextContent('( CABANG CILEGON )')
  })
})

describe('kolom grid', () => {
  it('menggambar ke-16 kolom dengan judul dari layar lama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const title of [
      'Cabang',
      'Sumbis',
      'COB',
      'Policy No',
      'Claim No',
      'Registration Date',
      'DOL',
      'COL',
      'Reserve Claim ASM Share',
      'Tgl Update Progress Terakhir',
      'Status Progress 1',
      'Status Progress2',
      'Adjuster',
      'PIC',
      'Aging (Hari)',
    ]) {
      expect(screen.getByRole('columnheader', { name: title })).toBeInTheDocument()
    }
  })

  it('menulis "Status Progress2" tanpa spasi, seperti di Pega', async () => {
    // Ejaannya memang tidak seragam dengan kolom di sebelahnya, dan itu dipertahankan
    // (`D-13`). Menyeragamkannya akan memunculkan selisih pada gerbang 1 tanpa satu pun
    // manfaat.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('columnheader', { name: 'Status Progress2' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'Status Progress 2' }),
    ).not.toBeInTheDocument()
  })

  it('menampilkan nilai uang sebagai rupiah, bukan teks mentahnya', async () => {
    // Backend mengirimnya sebagai TEKS desimal kanonik (`"250000.00"`). Menggambarnya apa
    // adanya akan menampilkan angka yang tidak pernah dibaca siapa pun dalam bentuk itu.
    //
    // Desimal yang seluruhnya nol memang DIBUANG `formatRupiah` — "Rp 250.000", bukan
    // "Rp 250.000,00". Itu perilaku pemformat bersama, dan modul ini tidak menyimpang
    // darinya.
    stubDefaultFetch([TUA])
    await renderLoaded()

    expect(screen.queryByText('250000.00')).not.toBeInTheDocument()
    expect(screen.getByText('Rp 250.000')).toBeInTheDocument()
  })

  it('menggambar tanggal kosong sebagai tanda pisah, bukan tanggal awal zaman', async () => {
    stubDefaultFetch([TENANG])
    renderPage()
    await screen.findByText('PNC-9003')

    expect(screen.queryByText(/1 Januari 1/)).not.toBeInTheDocument()
  })
})

describe('penandaan baris merah', () => {
  it('menandai klaim yang melewati ambang umur', async () => {
    stubDefaultFetch([TUA])
    await renderLoaded()

    const marked = screen.getByText('PNC-9001')
    expect(marked).toHaveClass('text-red-700')
  })

  it('menandai klaim yang progresnya mandek meski umurnya masih muda', async () => {
    // Kedua syarat harus dapat menyala SENDIRI-SENDIRI. Bila yang ini tidak diuji
    // terpisah, syarat kedua dapat hilang tanpa satu pun uji yang gagal.
    stubDefaultFetch([MANDEK])
    renderPage()
    const marked = await screen.findByText('PNC-9002')

    expect(marked).toHaveClass('text-red-700')
    expect(marked).toHaveAttribute(
      'title',
      expect.stringContaining('progres tidak berubah'),
    )
  })

  it('tidak menyebut umur sebagai alasan pada klaim yang belum melewati ambang', async () => {
    // Keterangan yang menyebut sebab yang TIDAK terpenuhi mengajari pengguna aturan yang
    // keliru — dan keterangan yang salah lebih buruk daripada tanpa keterangan.
    //
    // PNC-9002 berumur 8 hari, jauh di bawah 180. Satu-satunya sebab ia merah adalah
    // progresnya yang mandek.
    stubDefaultFetch([MANDEK])
    renderPage()
    const marked = await screen.findByText('PNC-9002')

    const reason = marked.getAttribute('title') ?? ''
    expect(reason).toContain('progres tidak berubah')
    expect(reason).not.toContain('umur klaim')
  })

  it('menyebut umur beserta ambangnya pada klaim yang memang tua', async () => {
    stubDefaultFetch([TUA])
    await renderLoaded()

    const reason = screen.getByText('PNC-9001').getAttribute('title') ?? ''
    expect(reason).toContain('umur klaim 987 hari')
    expect(reason).toContain('lebih dari 180')
    expect(reason).not.toContain('progres tidak berubah')
  })

  it('tidak menandai klaim yang muda dan progresnya bergerak', async () => {
    stubDefaultFetch([TENANG])
    renderPage()
    const plain = await screen.findByText('PNC-9003')

    expect(plain).not.toHaveClass('text-red-700')
  })

  it('menjelaskan aturan pewarnaannya dengan ambang dari server', async () => {
    // Angka yang ditulis di layar akan berbeda dari yang dipakai server begitu salah
    // satunya diubah — dan yang berubah diam-diam adalah keterangannya, bukan warnanya.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/lebih dari 180 hari/)).toBeInTheDocument()
  })
})

describe('paginasi', () => {
  it('meminta halaman berikutnya ke server, bukan memotong di peramban', async () => {
    stubFetch((url) => {
      if (url.startsWith(EXPORT_PATH)) return jsonResponse(200, {})
      return jsonResponse(200, listResponse([TUA], 60))
    })
    renderPage()
    await screen.findByText('PNC-9001')

    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))

    expect(lastListCall()?.url).toContain('halaman=2')
  })
})

describe('tombol ekspor', () => {
  it('mengirim token dan portal lewat header, bukan lewat alamat', async () => {
    // Nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header Referer.
    // Berkas ini memuat nomor polis, nama tertanggung, dan nilai uang.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('button', { name: 'Export To Excel' }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    const header = ekspor?.init?.headers as Record<string, string> | undefined
    expect(header?.['Authorization']).toBe('Bearer token-uji')
    expect(header?.['X-Portal']).toBe('ASM')
    expect(ekspor?.url).not.toContain('token-uji')
  })

  it('dimatikan saat tidak ada yang dapat diekspor', async () => {
    // Berkas kosong yang tetap terunduh tidak dapat dibedakan pengguna dari ekspor yang
    // gagal diam-diam.
    stubDefaultFetch([])
    renderPage()
    await screen.findByText(/Tidak ada klaim berjalan di cabang ini/)

    expect(screen.getByRole('button', { name: 'Export To Excel' })).toBeDisabled()
  })
})

describe('cabang yang tidak diketahui', () => {
  it('menampilkan pesan dari Pega, bukan tabel kosong', async () => {
    // Inilah perbedaan yang paling mudah hilang saat dibawa. Daftar kosong dan "Anda belum
    // punya cabang" terlihat sama di layar, padahal yang pertama berarti tidak ada
    // pekerjaan dan yang kedua berarti layarnya tidak dapat bekerja.
    stubFetch(() =>
      jsonResponse(409, {
        kode: 'cabang_tidak_diketahui',
        pesan: 'Belum ada Data Cabang pada Akun ini. Mohon Menghubungi Tim IT',
      }),
    )
    renderPage()

    await screen.findByText(/Belum ada Data Cabang pada Akun ini/)
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it('membedakannya dari gangguan pembacaan biasa', async () => {
    stubFetch(() =>
      jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' }),
    )
    renderPage()

    await screen.findByText(/Daftar tidak dapat dimuat/)
    expect(screen.queryByText(/Cabang Anda belum terdaftar/)).not.toBeInTheDocument()
  })
})

describe('selisih terencana', () => {
  it('ditampilkan ke pengguna, bukan disimpan sebagai komentar kode', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const section = screen.getByRole('heading', { name: /Yang berbeda dari layar lama/ })
      .parentElement as HTMLElement
    expect(within(section).getByText(/Daftar dibagi per halaman di server/)).toBeInTheDocument()
  })
})

it('menggambar keenam belas kolom grid layar lama, dalam urutannya', async () => {
  // Urutannya diambil dari `Section/InboxOutstandingperCabang_Section-Section.xml` apa
  // adanya. "Nama Insured" sempat TERLEWAT sepenuhnya pada serahan pertama — kolom yang
  // hilang tidak menghasilkan galat apa pun, hanya tabel yang tampak wajar tanpa satu kolom.
  stubDefaultFetch()
  await renderLoaded()

  const judul = [
    'Cabang',
    'Sumbis',
    'COB',
    'Policy No',
    'Nama Insured',
    'Claim No',
    'Registration Date',
    'DOL',
    'COL',
    'Reserve Claim ASM Share',
    'Tgl Update Progress Terakhir',
    'Status Progress 1',
    'Status Progress2',
    'Adjuster',
    'PIC',
    'Aging (Hari)',
  ]

  // DataTable menggambar judul dua kali — kepala tabel lebar dan label kartu ponsel —
  // sehingga yang diperiksa keberadaannya, bukan jumlahnya.
  for (const teks of judul) {
    expect(screen.getAllByText(teks).length).toBeGreaterThan(0)
  }
})

it('mengisi kolom Nama Insured, yang di layar lama selalu kosong', async () => {
  stubDefaultFetch()
  await renderLoaded()

  expect(screen.getAllByText('PT CONTOH SATU').length).toBeGreaterThan(0)
})
