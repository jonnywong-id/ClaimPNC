import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { OutstandingPage } from './OutstandingPage'
import type {
  OutstandingClaim,
  OutstandingListResponse,
  OutstandingSummaryResponse,
} from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
function claim(partial: Partial<OutstandingClaim> = {}): OutstandingClaim {
  return {
    klaim_id: 'KLM-0001',
    nomor_klaim: 'PNCN.26.0001',
    nomor_polis: 'POL-FIRE-0001',
    nama_tertanggung: 'PT Bumi Contoh Sentosa',
    nama_bisnis: 'Fire',
    sumber_bisnis: 'Broker',
    nama_cabang: 'JKT',
    group_panel: '006',
    tanggal_pendaftaran: '2026-09-16',
    tanggal_kejadian: '2026-09-08',
    tanggal_lapor: '2026-09-10',
    // Nilai `PYSTATUSWORK` sebagaimana benar-benar tersimpan — bukan 'BERJALAN', yang
    // berasal dari tabel yang sempat salah dipakai.
    status_proses: 'New',
    status_tampil: 'On Progress',
    status_klaim: '1147',
    pic_teknik: 'BUDISANTOSO',
    admin_pnc: 'ADMINPNC',
    umur_hari: 4,
    aging_hari: 2,
    posisi_klaim: 'On Progress',
    tahap_kini: 'Input Estimasi',
    pemegang_tugas: 'BUDISANTOSO',
    ...partial,
  }
}

function listResponse(partial: Partial<OutstandingListResponse> = {}): OutstandingListResponse {
  return {
    klaim: [claim()],
    total: 1,
    pemilik: 'ADMINPNC',
    ...partial,
  }
}

/**
 * Ringkasan status dokumen — sumber donut.
 *
 * Uji di berkas ini menstub SATU fungsi fetch untuk seluruh URL, sehingga tanpa ini
 * `/ringkasan` ikut menerima bentuk respons daftar. Itu pernah terjadi dan menjatuhkan
 * seluruh layar, bukan hanya panelnya.
 */
function summaryResponse(
  partial: Partial<OutstandingSummaryResponse> = {},
): OutstandingSummaryResponse {
  return {
    // Kesembilan tab dalam urutan layar Pega — enam di antaranya tanpa jumlah, persis
    // seperti yang dikirim backend. Uji yang hanya memuat dua tab yang terhitung akan
    // lulus tanpa pernah menyentuh perilaku "tab tanpa lencana".
    status: [
      { kode: 'lengkap', judul: 'Complete documents', jumlah: 0, dapat_dipilih: true },
      { kode: 'belum-lengkap', judul: 'Documents not complete', jumlah: 1, dapat_dipilih: true },
      { kode: 'temporary-close', judul: 'Temporary Close', jumlah: null, dapat_dipilih: false },
      {
        kode: 'deadline-temporary-close',
        judul: 'Deadline To Temporary Close',
        jumlah: null,
        dapat_dipilih: false,
      },
      { kode: 'loss-adjuster', judul: 'Loss Adjuster', jumlah: null, dapat_dipilih: false },
      {
        kode: 'internal-surveyor',
        judul: 'Internal Surveyor',
        jumlah: null,
        dapat_dipilih: false,
      },
      { kode: 'semua', judul: 'ALL Case', jumlah: 1, dapat_dipilih: true },
      { kode: 'komunikasi', judul: 'Communication', jumlah: null, dapat_dipilih: false },
      { kode: 'tka', judul: 'TKA', jumlah: null, dapat_dipilih: false },
    ],
    total: 1,
    pemilik: 'ADMINPNC',
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

function stubFetch(answer: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    // `/ringkasan` dijawab di sini kecuali uji yang bersangkutan menjawabnya sendiri.
    //
    // Tanpa cabang ini, setiap uji harus mengingat untuk menjawab endpoint yang tidak
    // sedang diujinya — dan yang lupa tidak gagal dengan jelas, melainkan menjatuhkan
    // seluruh layar.
    if (url.includes('/ringkasan')) {
      return Promise.resolve(jsonResponse(200, summaryResponse()))
    }
    return Promise.resolve(answer(url, init))
  })
}

function renderPage() {
  // Cache dimatikan supaya tiap uji berdiri sendiri; retry dimatikan supaya jalur galat
  // tidak menunggu percobaan ulang.
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <OutstandingPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-09-20T12:00:00Z',
  })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

// Judul kolom mengikuti `Section/InboxRegister_Section-Section.xml` apa adanya (`D-13`).
//
// Uji ini menjaga kesetaraan itu: mengganti judul menjadi bahasa Indonesia akan lulus uji
// lain tetapi membuat layar berbeda dari yang dikenal pengguna.
it('memakai judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')

  for (const title of [
    'Claim no',
    'Policy no',
    'Insured name',
    'Business Name',
    'Business source',
    'Branch name',
    'Admin name',
    'Register Date',
    'Date of loss',
    'Total Aging',
    'Aging',
    'Claim status',
    'Status ASM',
    'ASM PIC',
  ]) {
    // Nama dicocokkan PERSIS, bukan sebagian: "Aging" dan "Total Aging" adalah dua kolom
    // berbeda, dan pencocokan sebagian akan menemukan keduanya sekaligus.
    expect(screen.getByRole('columnheader', { name: title })).toBeInTheDocument()
  }
})

// Kolom "Aging" yang belum terisi tampil sebagai tanda hubung, bukan "0".
//
// Keduanya berbeda: nol hari adalah angka yang diketahui, belum terisi bukan.
it('membedakan Aging yang belum terisi dari nol hari', async () => {
  stubFetch(() =>
    jsonResponse(
      200,
      listResponse({
        klaim: [claim({ klaim_id: 'A', aging_hari: 0 }), claim({ klaim_id: 'B', aging_hari: null })],
        total: 2,
      }),
    ),
  )
  renderPage()

  expect(await screen.findByText('0 hari')).toBeInTheDocument()
})

it('menampilkan isi kolom klaim', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('PNCN.26.0001')).toBeInTheDocument()
  expect(screen.getByText('PT Bumi Contoh Sentosa')).toBeInTheDocument()
  expect(screen.getByText('POL-FIRE-0001')).toBeInTheDocument()
  // ADMINPNC muncul DUA KALI: di kolom "Admin name" dan di keterangan pemilik daftar.
  // getByText akan gagal pada kecocokan ganda, dan itu bukan cacat yang diuji di sini.
  expect(screen.getAllByText('ADMINPNC').length).toBeGreaterThan(0)
  expect(screen.getByText('BUDISANTOSO')).toBeInTheDocument()
  expect(screen.getByText('1147')).toBeInTheDocument()
})

// "Status ASM" yang kosong adalah keadaan NYATA, bukan kasus tepi yang dikarang: baris
// produksi yang diserahkan Work Owner tidak menyertakan kolom `STATUSLOCK_1` sama sekali.
//
// Sel yang benar-benar kosong membuat baris tampak rusak; ia dinyatakan dengan tanda hubung.
it('menandai Status ASM yang tidak terisi, bukan membiarkan selnya kosong', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ klaim: [claim({ status_klaim: '' })] })))
  renderPage()

  await screen.findByText('PNCN.26.0001')
  expect(screen.getByRole('cell', { name: '—' })).toBeInTheDocument()
})

it('menampilkan sumber bisnis', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('Broker')).toBeInTheDocument()
})

it('mengirim portal aktif pada setiap permintaan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')

  const request = calls.find((c) => c.url.startsWith('/api/inbox-outstanding'))
  expect(request).toBeDefined()

  const header = new Headers(request?.init?.headers)
  expect(header.get('X-Portal')).toBe('ASM')
})

// Klaim yang belum lolos Input Register belum bernomor. Menampilkannya sebagai sel kosong
// membuat layar tampak rusak; ia dinyatakan dengan kata-kata.
it('menyatakan klaim yang belum bernomor, bukan membiarkannya kosong', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ klaim: [claim({ nomor_klaim: '' })] })))
  renderPage()

  expect(await screen.findByText('belum bernomor')).toBeInTheDocument()
})

// Pemegang tugas dan tahap TETAP diterima dari server, tetapi TIDAK ditampilkan sebagai
// kolom — `Section/InboxRegister_Section-Section.xml` tidak punya kolomnya.
//
// Uji ini menjaga keputusan itu tetap disengaja. Tanpa uji, kolomnya mudah ditambahkan
// kembali karena datanya memang ada di tangan.
it('tidak menampilkan tahap maupun pemegang tugas sebagai kolom', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')

  expect(screen.queryByRole('columnheader', { name: /Tahap/ })).not.toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: /[Pp]emegang/ })).not.toBeInTheDocument()
  expect(screen.queryByText('Input Estimasi')).not.toBeInTheDocument()
})

// Inilah uji yang menjaga janji terpenting layar ini.
//
// Daftar kosong pada layar bernama "My Inbox" punya dua sebab yang tampak persis sama —
// memang tidak ada pekerjaan, atau penyaringnya salah orang. Menyebut pemiliknya di layar
// membedakan keduanya.
it("menyatakan pekerjaan siapa yang ditampilkan", async () => {
  stubFetch(() => jsonResponse(200, listResponse({ pemilik: "DEWILESTARI" })))
  renderPage()

  expect(await screen.findByText(/Menampilkan pekerjaan milik/)).toBeInTheDocument()
  expect(screen.getByText("DEWILESTARI")).toBeInTheDocument()
})


// Pencarian dikerjakan SERVER. Bila ia dikerjakan di peramban, hasilnya hanya menyentuh
// halaman yang sedang terbuka — dan pengguna diberi tahu bahwa klaimnya tidak ada padahal
// ia ada di halaman lain.
it('mengirim kata pencarian ke server, tidak menyaring di peramban', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('PNCN.26.0001')
  calls = []

  await userEvent.type(screen.getByRole('searchbox'), 'POL-FIRE')

  await waitFor(() => {
    const request = calls.find((c) => c.url.includes('cari='))
    expect(request).toBeDefined()
  })
})

/**
 * Unduhan TIDAK membawa penyaring layar.
 *
 * Nama uji ini sempat berbunyi "meminta unduhan dengan penyaring yang sedang berlaku",
 * padahal isinya tidak pernah memeriksa satu pun penyaring — ia hanya memeriksa `/unduh`
 * dan header portal. Namanya menjanjikan hal yang tidak diujinya, dan hal itu sekarang
 * justru tidak lagi benar: backend mengabaikan penyaring layar pada unduhan.
 */
it('meminta unduhan tanpa membawa penyaring layar', async () => {
  stubFetch((url) => {
    if (url.includes('/unduh')) {
      return new Response('No Klaim\nPNCN.26.0001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="inbox-outstanding-20260920.csv"',
        },
      })
    }
    return jsonResponse(200, listResponse())
  })

  // jsdom tidak menyediakan keduanya; unduhan memakai objek URL untuk menyerahkan berkas
  // ke peramban.
  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:uji'),
    revokeObjectURL: vi.fn(),
  })

  renderPage()
  await screen.findByText('PNCN.26.0001')

  await userEvent.click(screen.getByRole('button', { name: /Unduh CSV/ }))

  await waitFor(() => {
    const request = calls.find((c) => c.url.includes('/unduh'))
    expect(request).toBeDefined()
    expect(new Headers(request?.init?.headers).get('X-Portal')).toBe('ASM')
    expect(request?.url).not.toContain('cari=')
    expect(request?.url).not.toContain('tahap=')
    expect(request?.url).not.toContain('cabang=')
  })
})

/**
 * Tombol unduh tetap dapat ditekan meski daftarnya kosong.
 *
 * Ini sisi LAYAR dari cacat yang ditemukan Work Owner. Tombolnya dulu dimatikan lewat
 * `disabled={… || total === 0}`, sehingga petugas yang tidak sedang memegang satu pun tugas
 * tidak dapat menekannya sama sekali — padahal berkasnya tetap berisi.
 *
 * Terbukti pada data ASM: seorang petugas berlini NONMBU dengan 0 pekerjaan di inbox tetap
 * mengunduh 354 baris.
 */
it('tetap mengizinkan unduhan ketika daftar kosong', async () => {
  stubFetch((url) => {
    if (url.includes('/unduh')) {
      return new Response('No Klaim\nPNCN.26.0009\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="inbox-outstanding-20260924.csv"',
        },
      })
    }
    return jsonResponse(200, { klaim: [], total: 0, pemilik: 'ADMINPNC' })
  })

  vi.stubGlobal('URL', {
    ...URL,
    createObjectURL: vi.fn(() => 'blob:uji'),
    revokeObjectURL: vi.fn(),
  })

  renderPage()

  const tombol = await screen.findByRole('button', { name: /Unduh CSV/ })
  expect(tombol).toBeEnabled()

  await userEvent.click(tombol)
  await waitFor(() => {
    expect(calls.find((c) => c.url.includes('/unduh'))).toBeDefined()
  })
})

it('menuntun pengguna memilih entitas ketika portal belum dipilih', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(() => jsonResponse(200, listResponse()))

  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()

  // Tidak ada permintaan yang ditembakkan: menembaknya hanya untuk menerima penolakan
  // adalah perjalanan jaringan yang sia-sia.
  expect(calls.filter((c) => c.url.startsWith('/api/inbox-outstanding'))).toHaveLength(0)
})

it('menampilkan pesan ketika daftar gagal dimuat', async () => {
  stubFetch(() =>
    jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' }),
  )
  renderPage()

  expect(await screen.findByText('Daftar klaim tidak dapat dimuat')).toBeInTheDocument()
})

it('menyembunyikan paginasi ketika tidak ada klaim', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ klaim: [], total: 0 })))
  renderPage()

  expect(await screen.findByText('Tidak ada klaim yang masih berjalan.')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Berikutnya' })).not.toBeInTheDocument()
})

/**
 * Panel ringkasan menampilkan jumlah per status dokumen.
 *
 * Donutnya sendiri `aria-hidden` — deret tab di atasnya yang menjadi sumber resminya, dan
 * itulah yang diperiksa di sini. Memeriksa SVG-nya akan menguji Recharts, bukan modul ini.
 */
it('menampilkan kesembilan tab status dokumen dalam urutan layar Pega', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  const tabs = await screen.findByRole('navigation', { name: 'Status dokumen' })
  const judul = within(tabs)
    .getAllByRole('button')
    .map((b) => b.textContent?.replace(/\d+$/, '').trim())

  // Urutannya dibaca dari label tebal `Section/InboxRegister_Section-Section.xml`.
  // Menata ulangnya akan memindahkan tab yang sudah dihafal petugas.
  expect(judul).toEqual([
    'Complete documents',
    'Documents not complete',
    'Temporary Close',
    'Deadline To Temporary Close',
    'Loss Adjuster',
    'Internal Surveyor',
    'ALL Case',
    'Communication',
    'TKA',
  ])
})

/**
 * Tab yang sumber datanya belum dimigrasikan TIDAK diberi lencana, dan tidak dapat
 * ditekan.
 *
 * BUKAN lencana bertuliskan nol: nol menyatakan "tidak ada satu pun", padahal yang benar
 * "belum dihitung". Konvensi yang sama dipakai tab "Data rejected" pada Inbox Laporan
 * Klaim, dan dengan alasan yang sama.
 */
it('tidak memberi lencana pada tab yang belum dapat dihitung', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  const tabs = await screen.findByRole('navigation', { name: 'Status dokumen' })
  const utils = within(tabs)

  const lossAdjuster = utils.getByRole('button', { name: 'Loss Adjuster, belum tersedia' })
  expect(lossAdjuster).toBeDisabled()
  // Tidak ada angka menempel pada judulnya.
  expect(lossAdjuster.textContent).toBe('Loss Adjuster')

  // Yang terhitung justru punya angkanya.
  expect(
    utils.getByRole('button', { name: 'Documents not complete, 1 klaim' }),
  ).toBeInTheDocument()
})

/**
 * Panel ringkasan TETAP TAMPIL meski inbox kosong.
 *
 * Versi pertama mengembalikan `null` saat total nol, sehingga petugas yang tidak sedang
 * memegang satu pun tugas tidak melihat panelnya sama sekali — dan tidak ada cara
 * membedakan "saya tidak punya pekerjaan" dari "fiturnya tidak ada". Persis kegagalan yang
 * sama dengan tombol Unduh yang dulu dimatikan saat daftar kosong.
 */
it('tetap menampilkan panel ringkasan ketika inbox kosong', async () => {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url.includes('/ringkasan')) {
      return Promise.resolve(
        jsonResponse(
          200,
          summaryResponse({
            status: [
              { kode: 'lengkap', judul: 'Complete documents', jumlah: 0, dapat_dipilih: true },
              {
                kode: 'belum-lengkap',
                judul: 'Documents not complete',
                jumlah: 0,
                dapat_dipilih: true,
              },
              { kode: 'semua', judul: 'ALL Case', jumlah: 0, dapat_dipilih: true },
            ],
            total: 0,
          }),
        ),
      )
    }
    return Promise.resolve(jsonResponse(200, listResponse({ klaim: [], total: 0 })))
  })

  renderPage()

  const tabs = await screen.findByRole('navigation', { name: 'Status dokumen' })
  expect(within(tabs).getByRole('button', { name: 'ALL Case, 0 klaim' })).toBeInTheDocument()
  expect(screen.getByText('Tidak ada klaim berjalan untuk diringkas.')).toBeInTheDocument()
})

/** Mengeklik satu status menyaring grid lewat parameter `status_dokumen`. */
it('menyaring daftar saat satu status dokumen dipilih', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  const tabs = await screen.findByRole('navigation', { name: 'Status dokumen' })
  await userEvent.click(
    within(tabs).getByRole('button', { name: 'Documents not complete, 1 klaim' }),
  )

  await waitFor(() => {
    expect(calls.find((c) => c.url.includes('status_dokumen=belum-lengkap'))).toBeDefined()
  })

  // Ringkasan TIDAK ikut disaring — seluruh irisan harus tetap terlihat supaya pilihannya
  // dapat dibatalkan. Tanpa ini, donut menyusut menjadi satu irisan begitu diklik dan
  // tidak ada jalan kembali selain memuat ulang halaman.
  const ringkasanTersaring = calls.find(
    (c) => c.url.includes('/ringkasan') && c.url.includes('status_dokumen'),
  )
  expect(ringkasanTersaring).toBeUndefined()
})
