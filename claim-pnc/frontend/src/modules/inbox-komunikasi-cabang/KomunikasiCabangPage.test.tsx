import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  BranchScope,
  Conversation,
  MetadataResponse,
  Tab,
} from './types'

const PATH = '/api/inbox-komunikasi-cabang'
const TAB_PATH = `${PATH}/tab`
const EXPORT_PATH = `${PATH}/ekspor`
const DETAIL_PATH = `${PATH}/komunikasi/`

const SAMPLE_PROFILE = {
  identitas: '90000007',
  nama: 'Contoh Petugas Cabang',
  jenis: 'KARYAWAN',
  login: 'petugascabangcontoh',
  email: 'contoh.cabang@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kedua kolom TOMBOL, identik di kedua tab dan berjudul "Button" apa adanya.
 *
 * Keduanya ada di KEDUA grid layar lama, sebagai kolom DI DALAM grid — bukan di bilah aksi.
 *
 * Kedua tab tetap berbeda jumlah kolom ISIAN-nya (tiga dan lima), karena pada tab "Belum
 * Dijawab" kolom balasan dan penjawab dijamin kosong oleh penyaring `REPLYMESSAGE IS NULL`.
 */
const KOLOM_TOMBOL: Tab['kolom'] = [
  { kunci: 'aksi_detail', judul: 'Button' },
  { kunci: 'aksi_selesai', judul: 'Button' },
]

const TAB_BELUM: Tab = {
  kode: '1',
  nama: 'Belum Dijawab',
  keterangan: 'Percakapan yang pesannya belum dibalas sama sekali.',
  kolom: [
    { kunci: 'tanggal', judul: 'Tanggal' },
    { kunci: 'pengirim', judul: 'Pengirim(Dari)' },
    { kunci: 'pesan', judul: 'Pesan' },
    ...KOLOM_TOMBOL,
  ],
}

const TAB_SUDAH: Tab = {
  kode: '2',
  nama: 'Sudah Dijawab',
  keterangan: 'Percakapan yang sudah dibalas. Diurutkan dari balasan TERBARU.',
  kolom: [
    { kunci: 'tanggal', judul: 'Tanggal' },
    { kunci: 'pengirim', judul: 'Pengirim(Dari)' },
    { kunci: 'pesan', judul: 'Pesan' },
    { kunci: 'jawaban_terakhir', judul: 'Jawaban Terakhir' },
    { kunci: 'penjawab', judul: 'Penjawab(Dari)' },
    ...KOLOM_TOMBOL,
  ],
  catatan: 'Tab ini menampilkan BALASAN TERAKHIR pada setiap percakapan.',
}

const METADATA: MetadataResponse = {
  tab: [TAB_BELUM, TAB_SUDAH],
  tab_bawaan: '1',
  kolom_ekspor: [
    { kunci: 'tanggal', judul: 'Tanggal' },
    { kunci: 'tujuan', judul: 'Tujuan' },
    { kunci: 'status_register', judul: 'Status Register' },
  ],
  selisih_terencana: [
    'Kedua daftar dipisahkan menjadi TAB. Layar lama menggambar keduanya bertumpuk.',
    'Urutan kedua tab BERLAWANAN, dan itu perilaku layar lama apa adanya.',
  ],
  portal: 'ASM',
}

/** Batas data petugas cabang yang cabangnya TERBACA. */
const CABANG_TERBACA: BranchScope = {
  kode: '1001',
  kantor_pusat: false,
  terbaca: true,
  keterangan: 'Menampilkan percakapan cabang 1001 saja — percakapan cabang lain tidak tampil.',
}

/**
 * Batas data petugas yang cabangnya TIDAK dapat diturunkan.
 *
 * Keputusan Work Owner 2026-09-24 (`P-5`): ia dilayani sebagai KANTOR PUSAT, sama seperti
 * sistem lama. Uji di bawah memastikan keadaan itu DINYATAKAN di layar alih-alih tampak
 * sama dengan petugas pusat yang sah — pelebaran batas data yang tidak menghasilkan satu
 * pun galat.
 */
const CABANG_TIDAK_TERBACA: BranchScope = {
  kode: '1',
  kantor_pusat: true,
  terbaca: false,
  keterangan:
    'Kode cabang Anda tidak dapat dibaca dari data kepegawaian, sehingga daftar ini ' +
    'menampilkan percakapan KANTOR PUSAT.',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `jawaban_terakhir` dan `penjawab` sengaja KOSONG: begitulah yang selalu dikirim server
 * pada tab "Belum Dijawab".
 */
const BARIS: Conversation = {
  komunikasi: 'KOM-9001',
  tanggal: '2026-09-10 09:30',
  pengirim: 'PUSAT (adminpnccontoh)',
  asal: 'PUSAT',
  operator_pengirim: 'adminpnccontoh',
  pesan: 'Mohon lengkapi berita acara kerugian.',
  jawaban_terakhir: '',
  penjawab: '',
  tujuan: 'CABANG',
  status_register: '0',
  tanggal_jawaban: '',
}

const BARIS_DIJAWAB: Conversation = {
  ...BARIS,
  komunikasi: 'KOM-9002',
  jawaban_terakhir: 'Berita acara sudah diunggah hari ini.',
  penjawab: 'Petugas Cabang Contoh (CABANG)',
  tanggal_jawaban: '2026-09-12 16:20',
}

/** Jawaban layar Detail Komunikasi. */
const DETAIL = {
  komunikasi: 'KOM-9001',
  asal: 'PUSAT',
  pesan: [
    {
      tanggal: '2026-09-10 09:30',
      pengirim: 'PUSAT (adminpnccontoh)',
      asal: 'PUSAT',
      operator_pengirim: 'adminpnccontoh',
      pesan: 'Mohon lengkapi berita acara kerugian.',
      jawaban: 'Berita acara sudah diunggah hari ini.',
      penjawab: 'Petugas Cabang Contoh',
      tanggal_jawaban: '2026-09-12 16:20',
    },
  ],
  lampiran: [
    {
      id_dokumen: 'DOC-9001',
      jenis_dokumen: 'DOKUMEN KLAIM',
      rincian_dokumen: 'Laporan Survei',
      catatan: 'Revisi kedua',
      tanggal_unggah: '2026-09-12 16:25',
      sudah_diunggah: true,
    },
    {
      id_dokumen: 'DOC-9002',
      jenis_dokumen: 'DOKUMEN KLAIM',
      rincian_dokumen: 'Foto Objek',
      catatan: '',
      tanggal_unggah: '',
      sudah_diunggah: false,
    },
  ],
  tindakan_masih_di_pega: true,
  batas_cabang: CABANG_TERBACA,
  portal: 'ASM',
}

type Call = { url: string; init?: RequestInit | undefined }

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

/** Peladen tiruan yang menjawab bentuk layar, isi kedua tab, dan layar detailnya. */
function stubDefaultFetch(branch: BranchScope = CABANG_TERBACA) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    if (url.startsWith(DETAIL_PATH)) return jsonResponse(200, DETAIL)
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('Tanggal\n2026-09-10\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition':
            'attachment; filename="komunikasi-cabang-cabang-1001-belum-dijawab.csv"',
        },
      })
    }

    // Tab yang diminta menentukan bentuk jawabannya. Menjawab tab yang sama untuk setiap
    // permintaan akan membuat uji perpindahan tab lulus tanpa membuktikan apa pun.
    const wanted = new URL(url, 'http://uji.invalid').searchParams.get('tab')
    const answered = wanted === TAB_SUDAH.kode ? TAB_SUDAH : TAB_BELUM
    const baris = answered.kode === TAB_SUDAH.kode ? [BARIS_DIJAWAB] : [BARIS]

    return jsonResponse(200, {
      tab: answered,
      baris,
      paginasi: { halaman: 1, ukuran: 20, total: baris.length, total_halaman: 1 },
      // Kedua angka SENGAJA tidak sama dengan jumlah baris tabel, dan totalnya sengaja
      // tidak sama dengan jumlah keduanya — begitulah perilaku pencacah sistem lama.
      ringkasan: { belum_dijawab: 4, sudah_dijawab: 2, total: 6 },
      batas_cabang: branch,
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
      <MemoryRouter initialEntries={['/inbox-komunikasi-cabang']}>
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
async function renderLoaded(branch: BranchScope = CABANG_TERBACA) {
  stubDefaultFetch(branch)
  renderPage()
  await screen.findByRole('tab', { name: /Belum Dijawab/ })
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
  it('menggambar kedua tab', async () => {
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Belum Dijawab/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Sudah Dijawab/ })).toBeInTheDocument()
  })

  it('membuka tab "Belum Dijawab" lebih dulu, karena itulah pekerjaan yang menunggu', async () => {
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Belum Dijawab/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar TIGA kolom pada tab "Belum Dijawab", tanpa kolom balasan', async () => {
    await renderLoaded()

    // Ditunggu, bukan dibaca serentak: bilah tab tiba bersama jawaban `/tab`, sedangkan
    // tabelnya baru tiba bersama jawaban daftar.
    expect(await screen.findByRole('columnheader', { name: 'Tanggal' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Pengirim(Dari)' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Pesan' })).toBeInTheDocument()

    // Kolom balasan dan penjawab TIDAK digambar di sini — penyaring tabnya menjamin
    // keduanya kosong, dan section lama memang tidak menggambarnya.
    expect(
      screen.queryByRole('columnheader', { name: 'Jawaban Terakhir' }),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('columnheader', { name: 'Penjawab(Dari)' }),
    ).not.toBeInTheDocument()
  })

  it('menggambar LIMA kolom setelah berpindah ke tab "Sudah Dijawab"', async () => {
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Sudah Dijawab/ }))

    expect(
      await screen.findByRole('columnheader', { name: 'Jawaban Terakhir' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Penjawab(Dari)' })).toBeInTheDocument()
  })

  it('menggambar lencana pencacah pada kedua tab', async () => {
    // Layar lama menggambar kedua angka BERSAMAAN di atas kedua grid sebagai diagram
    // lingkaran. Begitu keduanya menjadi tab, lencana inilah satu-satunya cara melihat
    // berapa yang menunggu di tab sebelah TANPA berpindah ke sana.
    await renderLoaded()

    // Angkanya tiba bersama jawaban DAFTAR, bukan bersama bentuk layar — lencananya sengaja
    // tidak digambar selama pencacahnya belum diketahui, supaya "0" sementara tidak
    // menyatakan tidak ada pekerjaan pada tab yang mungkin penuh.
    expect(await screen.findByRole('tab', { name: /Belum Dijawab.*4/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Sudah Dijawab.*2/ })).toBeInTheDocument()
  })
})

describe('batas cabang', () => {
  it('menyatakan cabang mana yang sedang dilihat', async () => {
    // Petugas yang tidak tahu daftarnya sedang disaring akan menyimpulkan tidak ada
    // percakapan, padahal yang benar adalah tidak ada percakapan DI CABANGNYA.
    await renderLoaded()

    expect(await screen.findByText(/percakapan cabang 1001 saja/i)).toBeInTheDocument()
  })

  it('memperingatkan bila kode cabang pemanggil tidak terbaca', async () => {
    // Keadaan ini tampak PERSIS seperti petugas kantor pusat yang sah — daftarnya sama,
    // dan tidak ada satu pun galat. Peringatan inilah satu-satunya yang membedakannya.
    await renderLoaded(CABANG_TIDAK_TERBACA)

    const notice = await screen.findByRole('alert')
    expect(notice).toHaveTextContent(/tidak dapat dibaca/i)
    expect(notice).toHaveTextContent(/KANTOR PUSAT/)
  })

  it('tidak menandai peringatan bila cabangnya terbaca', async () => {
    await renderLoaded()

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

describe('perpindahan tab', () => {
  it('mengirim kode tab yang diminta ke peladen', async () => {
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Sudah Dijawab/ }))
    await screen.findByRole('columnheader', { name: 'Jawaban Terakhir' })

    const url = new URL(lastListCall()?.url ?? '', 'http://uji.invalid')
    expect(url.searchParams.get('tab')).toBe(TAB_SUDAH.kode)
  })

  it('menggambar catatan tab bila ada', async () => {
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: /Sudah Dijawab/ }))

    expect(await screen.findByText(/BALASAN TERAKHIR/)).toBeInTheDocument()
  })
})

describe('kolom tombol', () => {
  it('menggambar kedua tombol pada setiap baris, di KEDUA tab', async () => {
    // Keduanya kolom sungguhan di dalam grid layar lama, dua kali — sekali untuk tiap grid.
    // Versi pertama modul ini melewatkan keduanya.
    await renderLoaded()

    expect(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /Selesai Komunikasi percakapan KOM-9001/ }),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: /Sudah Dijawab/ }))

    expect(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9002/ }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: /Selesai Komunikasi percakapan KOM-9002/ }),
    ).toBeInTheDocument()
  })

  it('mempertahankan judul kolom "Button" apa adanya seperti layar lama', async () => {
    // Dua kolom berjudul sama memang tidak membantu, tetapi `D-13` menetapkan teks layar
    // mengikuti Pega. Yang ditambahkan adalah nama yang dibaca pembaca layar, bukan judulnya.
    await renderLoaded()

    expect(await screen.findAllByRole('columnheader', { name: 'Button' })).toHaveLength(2)
  })

  it('sel Pesan TIDAK lagi menjadi tautan pembuka', async () => {
    // Layar lama tidak punya tautan pada sel Pesan sama sekali. Dua cara berbeda membuka
    // baris membuat pengguna mengira keduanya melakukan hal yang berbeda.
    await renderLoaded()

    await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ })
    expect(
      screen.queryByRole('button', { name: /^Mohon lengkapi berita acara/ }),
    ).not.toBeInTheDocument()
  })

  it('menjawab "Selesai Komunikasi" dengan alasan dari peladen, bukan dengan diam', async () => {
    // Tindakan ini MENULIS — ia mengubah kanal percakapan sehingga barisnya hilang dari
    // kedua tab — dan tabelnya milik Pega selama masa paralel. Tombol yang diam saat ditekan
    // tidak terbedakan dari tombol yang rusak.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url.startsWith(`${PATH}/tindakan`)) {
        return jsonResponse(501, {
          kode: 'belum_tersedia',
          pesan:
            'Mengirim pesan, membalas, menutup percakapan, dan menambah percakapan belum ' +
            'tersedia di sistem baru. Kerjakan lewat Pega.',
        })
      }
      return jsonResponse(200, {
        tab: TAB_BELUM,
        baris: [BARIS],
        paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
        ringkasan: { belum_dijawab: 4, sudah_dijawab: 2, total: 6 },
        batas_cabang: CABANG_TERBACA,
        portal: 'ASM',
      })
    })
    renderPage()
    await screen.findByRole('tab', { name: /Belum Dijawab/ })

    await userEvent.click(
      await screen.findByRole('button', { name: /Selesai Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByRole('alert')).toHaveTextContent(/Kerjakan lewat Pega/)
  })

  it('mengirim nama tindakan ke peladen supaya pemakaiannya tercatat', async () => {
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Selesai Komunikasi percakapan KOM-9001/ }),
    )

    const call = [...calls].reverse().find((c) => c.url.startsWith(`${PATH}/tindakan`))
    expect(call).toBeDefined()

    const url = new URL(call?.url ?? '', 'http://uji.invalid')
    expect(url.searchParams.get('tindakan')).toBe('selesai-komunikasi')
    expect(url.searchParams.get('komunikasi')).toBe('KOM-9001')
    expect(call?.init?.method).toBe('POST')
  })
})

describe('tombol bilah atas', () => {
  it('menggambar Refresh dan Tambah', async () => {
    await renderLoaded()

    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeInTheDocument()
  })

  it('Refresh benar-benar mengambil ulang daftar, bukan ditolak', async () => {
    // Satu-satunya tombol layar lama yang bekerja di sini: ia tidak menulis apa pun,
    // sehingga tidak ada alasan menahannya.
    await renderLoaded()

    const before = calls.filter((c) => c.url.startsWith(PATH) && !c.url.includes('/tab')).length

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await screen.findByRole('columnheader', { name: 'Tanggal' })
    const after = calls.filter((c) => c.url.startsWith(PATH) && !c.url.includes('/tab')).length

    expect(after).toBeGreaterThan(before)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})

describe('detail komunikasi', () => {
  it('membuka panel utas saat sel Pesan diklik', async () => {
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByRole('region', { name: /Detail komunikasi/ })).toBeInTheDocument()
  })

  it('menggambar balasan sebagai baris tersendiri di bawah pesannya', async () => {
    // Section lama hanya menggambar tiga kolom dan tidak menampilkan balasan sama sekali,
    // padahal satu baris tabel menyimpan pesan DAN balasannya — sehingga utasnya terbaca
    // separuh. Yang ditambahkan adalah barisnya, bukan kolomnya.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByText(/Berita acara sudah diunggah/)).toBeInTheDocument()
  })

  it('membedakan lampiran yang sudah diunggah dari yang belum', async () => {
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByText(/Sudah Upload/)).toBeInTheDocument()
    expect(screen.getByText('Belum Upload')).toBeInTheDocument()
  })

  it('menyatakan bahwa membalas belum tersedia alih-alih menggambar kotak isian', async () => {
    // Kotak isian yang tampak dapat diketik tetapi menolak saat dikirim lebih buruk
    // daripada tidak ada: pengguna sudah mengetik kalimatnya, dan kalimat itu hilang.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByText(/Membalas belum tersedia di sini/)).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: /balasan/i })).not.toBeInTheDocument()
  })
})

describe('keterangan yang wajib terlihat', () => {
  it('menyebut keempat tindakan yang masih dikerjakan lewat Pega', async () => {
    // Layar yang kehilangan tombolnya tanpa penjelasan akan dilaporkan sebagai kerusakan,
    // dan penggunanya tidak akan tahu ia masih harus mengerjakannya lewat Pega.
    //
    // Pencariannya DIBATASI pada panel itu sendiri: sejak tombolnya benar-benar digambar,
    // teks "Selesai Komunikasi" dan "Tambah" muncul pula sebagai tombol — dan pencarian
    // seluruh halaman akan menemukan lebih dari satu.
    await renderLoaded()

    const heading = await screen.findByText(/Yang masih dikerjakan lewat Pega/)
    const panel = heading.closest('section')
    expect(panel).not.toBeNull()

    const within = panel as HTMLElement
    for (const label of ['Kirim Pesan', 'Balas', 'Selesai Komunikasi', 'Tambah']) {
      expect(within.textContent).toContain(label)
    }
  })

  it('menggambar selisih terencana dari peladen', async () => {
    await renderLoaded()

    expect(
      await screen.findByText(/Kedua daftar dipisahkan menjadi TAB/),
    ).toBeInTheDocument()
  })

  it('menyebutkan kolom berkas unduhan sebelum diunduh', async () => {
    // Berkasnya memuat tiga kolom yang TIDAK ada di tabel mana pun. Perbedaan itu tidak
    // boleh baru ketahuan setelah berkasnya dibuka.
    await renderLoaded()

    expect(await screen.findByText(/Berisi:.*Status Register/)).toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menolak menggambar daftar sebelum entitas dipilih', async () => {
    useSelectedPortal.getState().clear()
    stubDefaultFetch()
    renderPage()

    expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
  })
})
