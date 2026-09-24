import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AnalystDoctorPage } from './AnalystDoctorPage'
import type { DaftarResponse, KeteranganResponse, TugasAnalystDoctor } from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen. Pada layar ini ia lebih mengikat lagi:
 * antreannya berisi klaim Personal Accident, dan `FR-R2` memperlakukan data medis secara
 * khusus.
 */
function tugas(partial: Partial<TugasAnalystDoctor> = {}): TugasAnalystDoctor {
  return {
    klaim_id: 'ASM-FW-GCNMFW-WORK PNCN.26.0311',
    nomor_case: 'PNCN.26.0311',
    nomor_polis: '26.002.2026.00311',
    nama_tertanggung: 'Bayu Pratama Contoh',
    nama_cabang: 'JAKARTA PUSAT',
    nama_admin: 'ADMINREG1',
    komentar_pic_teknis: 'Mohon dinilai kelayakan biaya rawat inap.',
    pic_teknis: 'PICTEKNIKPA1',
    tanggal_pendaftaran: '2026-09-18',
    lama_hari: 5,
    status_proses: 'Open',
    operator_penerima: 'ADMINKLAIM',
    ...partial,
  }
}

/**
 * Keterangan layar meniru jawaban server apa adanya, termasuk urutan kolomnya.
 *
 * Kedelapan judulnya diambil dari rule `pyCaption …` pada
 * `Harness/inboxAnalystDoctor_Harness-Harness.xml`.
 */
const keteranganResponse: KeteranganResponse = {
  portal: 'ASM',
  kolom: [
    { kunci: 'nomor_case', judul: 'Nomor Case' },
    { kunci: 'nomor_polis', judul: 'No Polis' },
    { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
    { kunci: 'nama_cabang', judul: 'Nama Cabang' },
    { kunci: 'tanggal_pendaftaran', judul: 'Tanggal Pendaftaran' },
    { kunci: 'nama_admin', judul: 'Nama Admin' },
    {
      kunci: 'komentar_pic_teknis',
      judul: 'Komentar dari PIC Teknis',
      keterangan: 'Belum terbawa. Properti Pega-nya tidak terekspos.',
    },
    { kunci: 'lama_hari', judul: 'Lama Waktu Klaim' },
  ],
  selisih_terencana: ['Kolom "Lama Waktu Klaim" berisi umur tugas dalam hari.'],
  keterbatasan: ['Kolom "Komentar dari PIC Teknis" masih kosong.'],
  ukuran_halaman: 25,
}

function daftarResponse(partial: Partial<DaftarResponse> = {}): DaftarResponse {
  return {
    portal: 'ASM',
    data: [tugas()],
    total: 1,
    lewati: 0,
    batas: 25,
    cari: '',
    ...partial,
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

/**
 * Menjawab kedua GET layar ini sekaligus.
 *
 * Layar memanggil `/keterangan` dan daftar bersamaan; menjawab keduanya di satu tempat
 * membuat tiap uji hanya perlu menyatakan yang berbeda.
 */
function stubFetch(daftar: DaftarResponse | (() => Response)) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    if (url.includes('/keterangan')) {
      return Promise.resolve(jsonResponse(200, keteranganResponse))
    }
    return Promise.resolve(typeof daftar === 'function' ? daftar() : jsonResponse(200, daftar))
  })
}

function renderPage() {
  // Cache dimatikan supaya tiap uji berdiri sendiri; retry dimatikan supaya jalur galat
  // tidak menunggu percobaan ulang.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AnalystDoctorPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** urlDaftar mengambil permintaan daftar TERAKHIR yang dikirim layar. */
function urlDaftar(): string {
  const daftar = calls.filter((c) => !c.url.includes('/keterangan'))
  return daftar[daftar.length - 1]?.url ?? ''
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-09-24T12:00:00Z',
  })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/**
 * Judul kolom mengikuti harness apa adanya (`D-13`).
 *
 * Yang diuji di sini bukan sekadar bahwa judulnya benar, melainkan bahwa judulnya datang
 * dari SERVER: bila layar mengetiknya sendiri, uji ini tetap lulus sekarang tetapi akan
 * menyimpang diam-diam begitu bacaan export dikoreksi di backend.
 */
it('menggambar kedelapan judul kolom dari keterangan server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0311')

  for (const judul of [
    'Nomor Case',
    'No Polis',
    'Nama Tertanggung',
    'Nama Cabang',
    'Tanggal Pendaftaran',
    'Nama Admin',
    'Komentar dari PIC Teknis',
    'Lama Waktu Klaim',
  ]) {
    // getAllByText, bukan getByText: `DataTable` menggambar judul DUA KALI — sebagai `<th>`
    // pada tampilan meja, dan sebagai label sel pada tampilan kartu di layar sempit.
    // Keduanya memang harus ada, dan menuntut satu kemunculan akan menolak tabel yang benar.
    expect(screen.getAllByText(judul).length).toBeGreaterThan(0)
  }
})

it('menampilkan isi baris beserta umur tugas dalam hari', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0311')

  expect(screen.getByText('26.002.2026.00311')).toBeInTheDocument()
  expect(screen.getByText('Bayu Pratama Contoh')).toBeInTheDocument()
  expect(screen.getByText('2026-09-18')).toBeInTheDocument()
  expect(screen.getByText('5 hari')).toBeInTheDocument()
})

/**
 * Sel yang belum terbawa MENJELASKAN dirinya.
 *
 * Tanpa ini, sel kosong terbaca sebagai "PIC Teknis memang tidak menulis apa-apa" — dan
 * tidak ada seorang pun yang menanyakannya.
 */
it('menandai kolom komentar yang belum terbawa, bukan membiarkannya hampa', async () => {
  stubFetch(daftarResponse({ data: [tugas({ komentar_pic_teknis: '' })] }))
  renderPage()

  await screen.findByText('PNCN.26.0311')

  expect(screen.getByText('belum terbawa')).toBeInTheDocument()
})

it('menampilkan nama PIC Teknis sebagai keterangan komentarnya', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0311')

  expect(screen.getByText('— PICTEKNIKPA1')).toBeInTheDocument()
})

/**
 * Pencarian dikerjakan SERVER.
 *
 * Menyaring di peramban hanya menyentuh halaman yang sedang terbuka, sehingga hasilnya
 * BOHONG: pengguna diberi tahu sesuatu tidak ada padahal ia di halaman berikutnya.
 */
it('mengirim kata kunci ke server, bukan menyaring di peramban', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0311')

  await userEvent.type(screen.getByLabelText(/Cari Nomor Case/i), '0298')

  await waitFor(() => {
    expect(urlDaftar()).toContain('cari=0298')
  })
})

/**
 * Mengetik kata kunci mengembalikan ke halaman pertama.
 *
 * Tanpa itu, mencari saat berada di halaman lima menampilkan halaman lima dari hasil baru —
 * yang hampir selalu kosong, dan terbaca sebagai "tidak ditemukan".
 */
it('kembali ke halaman pertama saat kata kunci berubah', async () => {
  stubFetch(daftarResponse({ total: 120 }))
  renderPage()

  await screen.findByText('PNCN.26.0311')

  // Bilah halaman berpaginasi server berbentuk First/Previous/Next/Last, mengikuti
  // `Section/ButtonPagingInbox-Section.xml` — bukan deretan nomor. Tombolnya dikenali lewat
  // `aria-label`, karena teks yang terlihat disingkat pada layar sempit.
  await userEvent.click(screen.getByRole('button', { name: 'Halaman berikutnya' }))
  await waitFor(() => expect(urlDaftar()).toContain('lewati=25'))

  await userEvent.type(screen.getByLabelText(/Cari Nomor Case/i), '0298')

  await waitFor(() => {
    const url = urlDaftar()
    expect(url).toContain('cari=0298')
    expect(url).not.toContain('lewati=')
  })
})

it('mengirim header portal pada setiap permintaan', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0311')

  for (const call of calls) {
    const headers = new Headers(call.init?.headers)
    expect(headers.get('X-Portal')).toBe('ASM')
  }
})

/**
 * Layar tidak menembak server sebelum portal dipilih.
 *
 * Backend memang menolaknya (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk
 * menerima penolakan adalah perjalanan yang sia-sia — dan pesannya tidak menjelaskan apa yang
 * harus dilakukan pengguna.
 */
it('meminta pengguna memilih entitas lebih dulu, tanpa menembak server', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(daftarResponse())
  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})

it('menampilkan pesan galat saat antrean gagal dimuat', async () => {
  stubFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'Basis data tidak siap.' }))
  renderPage()

  expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
})

/**
 * Antrean kosong tidak boleh terbaca sebagai kerusakan — atau sebaliknya.
 *
 * Keduanya diuji karena pesannya BERBEDA, dan yang menghasilkannya pun berbeda: yang pertama
 * milik layar ini, yang kedua digantikan `DataTable` begitu `serverSearch` terisi.
 * Menyamakannya membuat pengguna yang lupa mengosongkan kotak cari mengira ia tidak punya
 * pekerjaan.
 */
it('menyatakan antrean memang kosong saat tidak ada pencarian', async () => {
  stubFetch(daftarResponse({ data: [], total: 0 }))
  renderPage()

  expect(
    await screen.findByText('Tidak ada tugas penilaian medis untuk Anda saat ini.'),
  ).toBeInTheDocument()
})

it('menyatakan pencarian tidak cocok saat kotak cari terisi', async () => {
  stubFetch(daftarResponse({ data: [], total: 0 }))
  renderPage()

  await userEvent.type(await screen.findByLabelText(/Cari Nomor Case/i), 'zzz')

  // Pesannya milik `DataTable`, bukan layar ini — dan menyebut kata kuncinya, sehingga
  // pengguna melihat APA yang dicari saat hasilnya nihil.
  expect(await screen.findByText(/Tidak ada baris yang cocok dengan/)).toBeInTheDocument()
  expect(screen.getByText(/zzz/)).toBeInTheDocument()
})

/**
 * Selisih terencana dan keterbatasan DITAMPILKAN, bukan disimpan sebagai catatan teknis.
 *
 * `D-54` menetapkan selisih di luar 13 butir `P-5` dinyatakan. Menyatakannya di layar itulah
 * yang membuat keputusannya terlihat oleh orang yang memakai layarnya.
 */
it('menampilkan selisih terencana dan keterbatasan dari server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0311')

  expect(screen.getByText('Perbedaan yang disengaja terhadap layar lama')).toBeInTheDocument()
  expect(
    screen.getByText('Kolom "Lama Waktu Klaim" berisi umur tugas dalam hari.'),
  ).toBeInTheDocument()
  expect(screen.getByText('Yang perlu diketahui')).toBeInTheDocument()
  expect(
    screen.getByText('Kolom "Komentar dari PIC Teknis" masih kosong.'),
  ).toBeInTheDocument()
})

/**
 * Klaim yang sama dengan DUA penugasan terbuka muncul dua kali.
 *
 * Itu perilaku `INNER JOIN` Pega yang sengaja dibawa (`P-5`). Kunci barisnya karena itu harus
 * menggabungkan klaim_id dengan nomor case — memakai klaim_id saja akan membuat React
 * menemukan kunci ganda pada baris yang memang seharusnya kembar.
 */
it('menggambar baris kembar tanpa kunci ganda', async () => {
  const warn = vi.spyOn(console, 'error').mockImplementation(() => {})

  stubFetch(
    daftarResponse({
      data: [tugas(), tugas({ nomor_case: 'PNCN.26.0312' })],
      total: 2,
    }),
  )
  renderPage()

  await screen.findByText('PNCN.26.0311')
  expect(screen.getByText('PNCN.26.0312')).toBeInTheDocument()

  const duplicateKeyWarning = warn.mock.calls.some((args) =>
    String(args[0]).includes('same key'),
  )
  expect(duplicateKeyWarning).toBe(false)

  warn.mockRestore()
})
