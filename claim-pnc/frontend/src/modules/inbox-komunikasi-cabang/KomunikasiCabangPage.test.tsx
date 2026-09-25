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
const BRANCH_PATH = `${PATH}/cabang`
const MESSAGE_PATH = `${PATH}/pesan`

/**
 * Daftar cabang untuk pemilih tujuan.
 *
 * Isinya KARANGAN — `D-69` melarang data perusahaan ditulis di berkas yang di-commit, dan
 * larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const BRANCH_LIST = {
  cabang: [
    { kode: '1001', nama: 'CABANG CONTOH SATU' },
    { kode: '1002', nama: 'CABANG CONTOH DUA' },
  ],
  tujuan: ['PUSAT', 'CABANG'],
  portal: 'ASM',
}

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
/**
 * Jawaban layar Detail Komunikasi.
 *
 * Utasnya DUA ucapan, masing-masing tiga isian — bentuk yang dibaca dari tabel riwayat.
 * Sampai 2026-09-24 ia satu ucapan berisi pesan DAN balasannya, karena utasnya dibaca dari
 * tabel percakapan.
 */
const DETAIL = {
  komunikasi: 'KOM-9001',
  asal: 'PUSAT',
  pesan: [
    {
      tanggal: '2026-09-10 09:30',
      pengirim: 'adminpnccontoh',
      pesan: 'Mohon lengkapi berita acara kerugian.',
    },
    {
      tanggal: '2026-09-12 16:20',
      pengirim: 'petugascabangcontoh',
      pesan: 'Berita acara sudah diunggah hari ini.',
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
  balas_tersedia: true,
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
    if (url === BRANCH_PATH) return jsonResponse(200, BRANCH_LIST)
    if (url === MESSAGE_PATH) {
      return jsonResponse(201, {
        komunikasi: 'KOM-9100',
        pesan: 'Pesan terkirim. Ia muncul di tab "Belum Dijawab" milik kantor pusat.',
        portal: 'ASM',
      })
    }
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

  it('meminta penegasan lebih dulu, tidak langsung menutup percakapan', async () => {
    // Menutup percakapan TIDAK DAPAT DIBATALKAN — barisnya hilang dari kedua tab, dan
    // sistem lama tidak punya satu pun tindakan yang membukanya kembali. Tombolnya pun
    // berada di dalam baris tabel, tempat satu klik meleset mengenai baris tetangga.
    //
    // Penegasan ini SELISIH YANG DISENGAJA terhadap Pega, yang menutupnya seketika.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Selesai Komunikasi percakapan KOM-9001/ }),
    )

    expect(
      await screen.findByRole('alertdialog', { name: /Tutup percakapan KOM-9001/ }),
    ).toBeInTheDocument()

    // Belum satu pun permintaan tulis terkirim. Penegasan yang sudah mengirim lebih dulu
    // bukan penegasan.
    expect(calls.some((c) => c.url.includes('/selesai'))).toBe(false)
  })

  it('membatalkan penegasan tidak menutup apa pun', async () => {
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Selesai Komunikasi percakapan KOM-9001/ }),
    )
    await userEvent.click(await screen.findByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(calls.some((c) => c.url.includes('/selesai'))).toBe(false)
  })

  it('menutup percakapan lewat alamatnya sendiri setelah ditegaskan', async () => {
    // Alamatnya bersarang di bawah nomor percakapan, BUKAN satu endpoint "tindakan"
    // bersama: yang satu dapat diulang dan yang satu tidak dapat dibatalkan, sehingga
    // keduanya tidak boleh berbagi satu bentuk permintaan dan satu baris log akses.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Selesai Komunikasi percakapan KOM-9001/ }),
    )
    await userEvent.click(
      await screen.findByRole('button', { name: /Ya, tutup percakapan/ }),
    )

    const call = [...calls].reverse().find((c) => c.url.includes('/selesai'))
    expect(call).toBeDefined()
    expect(call?.url).toBe(`${PATH}/komunikasi/KOM-9001/selesai`)
    expect(call?.init?.method).toBe('POST')
  })

  it('menyatakan alasan dari peladen bila penutupan ditolak', async () => {
    // Dua orang dapat menekan tombol yang sama pada layar yang sama-sama usang. Yang kedua
    // harus tahu bahwa BUKAN dia yang menutupnya.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url.includes('/selesai')) {
        return jsonResponse(404, {
          kode: 'komunikasi_tidak_ditemukan',
          pesan:
            'Percakapan tidak ditemukan. Ia mungkin sudah ditutup orang lain — segarkan ' +
            'daftar, lalu periksa pilihan portal di bilah atas.',
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
    await userEvent.click(
      await screen.findByRole('button', { name: /Ya, tutup percakapan/ }),
    )

    expect(await screen.findByRole('alert')).toHaveTextContent(/sudah ditutup orang lain/)
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

  it('menggambar setiap ucapan sebagai barisnya sendiri, termasuk balasannya', async () => {
    // Sampai 2026-09-24 utas dibaca dari tabel percakapan, sehingga layar detail selalu
    // menampilkan tepat SATU ucapan — betapapun panjang percakapannya — dengan balasannya
    // digambar sebagai blok di dalamnya.
    //
    // Keterangan Work Owner mengoreksinya: utas dibaca dari tabel RIWAYAT, tempat setiap
    // pesan dan setiap balasan menempati barisnya sendiri.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    // Pencarian DIBATASI pada panel detail: kalimat yang sama muncul pula di sel "Pesan"
    // pada grid di atasnya, dan pencarian seluruh halaman akan menemukan dua.
    const panel = await screen.findByRole('region', { name: /Detail komunikasi/ })

    expect(panel.textContent).toContain('Mohon lengkapi berita acara')
    expect(panel.textContent).toContain('Berita acara sudah diunggah')

    // Dua ucapan = dua baris daftar, bukan satu baris berisi keduanya.
    expect(panel.querySelectorAll('ol > li')).toHaveLength(2)
  })

  it('menyatakan keadaan percakapan yang belum punya satu pun ucapan', async () => {
    // Percakapan yang dibuat lewat layar lain punya kepala tanpa utas. Ia harus terbuka
    // dengan keterangan, BUKAN dijawab "tidak ditemukan".
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url.startsWith(DETAIL_PATH)) {
        return jsonResponse(200, { ...DETAIL, pesan: [], lampiran: [] })
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
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByText(/belum memuat satu pun ucapan/)).toBeInTheDocument()
  })

  it('membedakan lampiran yang sudah diunggah dari yang belum', async () => {
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByText(/Sudah Upload/)).toBeInTheDocument()
    expect(screen.getByText('Belum Upload')).toBeInTheDocument()
  })

  it('menggambar kotak "Masukkan Balasan" yang benar-benar dapat dipakai', async () => {
    // Sampai 2026-09-24 yang digambar di sini hanyalah KETERANGAN bahwa membalas belum
    // tersedia — karena activity di baliknya tidak ada di export mana pun. Ia diterima, dan
    // kotaknya menjadi isian sungguhan.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(
      await screen.findByRole('textbox', { name: /Masukkan Balasan/ }),
    ).toBeInTheDocument()
    expect(screen.queryByText(/Membalas belum tersedia di sini/)).not.toBeInTheDocument()
  })

  it('menonaktifkan tombol Balas selama isiannya masih kosong', async () => {
    // Balasan kosong yang tersimpan menyetel status menjadi "sudah dijawab", sehingga
    // percakapannya berpindah tab — terbaca sudah dijawab padahal tidak ada jawabannya.
    // Peladen menolaknya; layar tidak perlu membiarkannya terkirim lebih dulu.
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    expect(await screen.findByRole('button', { name: 'Balas' })).toBeDisabled()

    await userEvent.type(
      await screen.findByRole('textbox', { name: /Masukkan Balasan/ }),
      'Sudah kami tindak lanjuti.',
    )
    expect(screen.getByRole('button', { name: 'Balas' })).toBeEnabled()
  })

  it('mengirim balasan ke alamat percakapannya sendiri', async () => {
    await renderLoaded()

    await userEvent.click(
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )
    await userEvent.type(
      await screen.findByRole('textbox', { name: /Masukkan Balasan/ }),
      'Sudah kami tindak lanjuti.',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Balas' }))

    const call = [...calls].reverse().find((c) => c.url.includes('/balas'))
    expect(call).toBeDefined()
    expect(call?.url).toBe(`${PATH}/komunikasi/KOM-9001/balas`)
    expect(call?.init?.method).toBe('POST')

    // Hanya isi balasannya. Penjawabnya diambil dari sesi di peladen — jejak yang isinya
    // ditentukan pengirim permintaan bukan jejak.
    expect(JSON.parse(String(call?.init?.body))).toEqual({
      pesan: 'Sudah kami tindak lanjuti.',
    })
  })

  it('menahan kalimat yang sudah diketik bila balasannya ditolak', async () => {
    // Kalimat yang sudah diketik adalah pekerjaan penggunanya. Mengosongkannya saat
    // permintaan ditolak membuang pekerjaan itu tanpa ia sempat menyalinnya.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url.includes('/balas')) {
        return jsonResponse(503, {
          kode: 'sumber_cabang_tidak_terbaca',
          pesan: 'Sumber data cabang sedang tidak dapat dibaca.',
        })
      }
      if (url.startsWith(DETAIL_PATH)) return jsonResponse(200, DETAIL)
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
      await screen.findByRole('button', { name: /Detail Komunikasi percakapan KOM-9001/ }),
    )

    const box = await screen.findByRole('textbox', { name: /Masukkan Balasan/ })
    await userEvent.type(box, 'Kalimat yang tidak boleh hilang.')
    await userEvent.click(screen.getByRole('button', { name: 'Balas' }))

    expect(
      await screen.findByText(/Sumber data cabang sedang tidak dapat dibaca/),
    ).toBeInTheDocument()
    expect(box).toHaveValue('Kalimat yang tidak boleh hilang.')
  })
})

describe('keterangan yang wajib terlihat', () => {
  it('TIDAK lagi menggambar panel "Yang masih dikerjakan lewat Pega"', async () => {
    // Panel itu menyebutkan tindakan yang masih harus dikerjakan lewat Pega. Sejak KEEMPAT
    // tindakan tulis layar lama bekerja di sini, ia tidak punya satu pun isi yang benar —
    // dan panel yang menyatakan ada sesuatu yang belum tersedia akan membuat pembacanya
    // mencari apa.
    await renderLoaded()

    expect(screen.queryByText(/Yang masih dikerjakan lewat Pega/)).not.toBeInTheDocument()
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

describe('kirim pesan', () => {
  it('menampilkan form saat "Tambah" ditekan, bukan menolak dengan alasan', async () => {
    // Sampai 2026-09-24 tombol ini dijawab 501: formnya dikira tidak ada di export. Ia ada —
    // tersembunyi sebagai blok bersyarat di dalam section daftar, bukan section tersendiri.
    await renderLoaded()

    expect(screen.queryByRole('form', { name: /Kirim pesan baru/ })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      await screen.findByRole('form', { name: /Kirim pesan baru/ }),
    ).toBeInTheDocument()
  })

  it('tidak menyentuh peladen hanya untuk membuka formnya', async () => {
    // Di Pega "Tambah" menjalankan data transform lalu me-refresh section — ia menyetel
    // penanda di server tanpa menulis apa pun. Di sini keadaan layar tinggal di layar.
    await renderLoaded()

    const before = calls.length
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    // Yang boleh bertambah hanyalah pengambilan daftar cabang.
    const baru = calls.slice(before).filter((c) => !c.url.startsWith(BRANCH_PATH))
    expect(baru).toHaveLength(0)
  })

  it('menyembunyikan pemilih cabang selama tujuannya PUSAT', async () => {
    // Isian wajib yang tampak tetapi tidak berlaku adalah sumber kebingungan yang lebih
    // besar daripada isian yang muncul saat dibutuhkan.
    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    expect(screen.queryByLabelText(/Cabang/)).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Tujuan'), 'CABANG')
    expect(await screen.findByLabelText(/Cabang/)).toBeInTheDocument()
  })

  it('menahan tombol kirim sampai cabang dipilih bila tujuannya CABANG', async () => {
    // Peladen menolaknya juga, dengan kalimat yang dibawa apa adanya dari activity lama.
    // Layar tidak perlu membiarkannya terkirim lebih dulu untuk memberi tahu apa yang sudah
    // terlihat kosong di layarnya sendiri.
    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    await userEvent.selectOptions(screen.getByLabelText('Tujuan'), 'CABANG')
    await userEvent.type(screen.getByLabelText('Pesan'), 'Mohon konfirmasi.')

    expect(screen.getByRole('button', { name: 'Kirim Pesan' })).toBeDisabled()

    await userEvent.selectOptions(await screen.findByLabelText(/Cabang/), '1002')
    expect(screen.getByRole('button', { name: 'Kirim Pesan' })).toBeEnabled()
  })

  it('mengirim ketiga isian ke alamat pembuatan percakapan', async () => {
    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    await userEvent.selectOptions(screen.getByLabelText('Tujuan'), 'CABANG')
    await userEvent.selectOptions(await screen.findByLabelText(/Cabang/), '1002')
    await userEvent.type(screen.getByLabelText('Pesan'), 'Mohon konfirmasi.')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Pesan' }))

    const call = [...calls].reverse().find((c) => c.url === MESSAGE_PATH)
    expect(call).toBeDefined()
    expect(call?.init?.method).toBe('POST')
    expect(JSON.parse(String(call?.init?.body))).toEqual({
      tujuan: 'CABANG',
      cabang: '1002',
      pesan: 'Mohon konfirmasi.',
    })
  })

  it('mengosongkan cabang yang tertinggal saat tujuannya kembali ke PUSAT', async () => {
    // Pilihan yang tertinggal tak terlihat lalu ikut terkirim akan mengirim pesan ke cabang
    // yang sudah tidak dimaksudkan penggunanya.
    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    await userEvent.selectOptions(screen.getByLabelText('Tujuan'), 'CABANG')
    await userEvent.selectOptions(await screen.findByLabelText(/Cabang/), '1002')
    await userEvent.selectOptions(screen.getByLabelText('Tujuan'), 'PUSAT')

    await userEvent.type(screen.getByLabelText('Pesan'), 'Mohon konfirmasi.')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Pesan' }))

    const call = [...calls].reverse().find((c) => c.url === MESSAGE_PATH)
    expect(JSON.parse(String(call?.init?.body))).toEqual({
      tujuan: 'PUSAT',
      cabang: '',
      pesan: 'Mohon konfirmasi.',
    })
  })

  it('menutup form dan menyatakan hasilnya setelah pesannya terkirim', async () => {
    // Perilaku yang sama dengan sistem lama, yang mengosongkan `pyLabel` pada langkah
    // terakhir activity-nya sehingga bloknya tertutup sendiri.
    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    await userEvent.type(screen.getByLabelText('Pesan'), 'Mohon konfirmasi.')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Pesan' }))

    expect(await screen.findByText(/Pesan terkirim/)).toBeInTheDocument()
    expect(screen.queryByRole('form', { name: /Kirim pesan baru/ })).not.toBeInTheDocument()
  })

  it('membatalkan form tidak mengirim apa pun', async () => {
    await renderLoaded()
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    await userEvent.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByRole('form', { name: /Kirim pesan baru/ })).not.toBeInTheDocument()
    expect(calls.some((c) => c.url === MESSAGE_PATH)).toBe(false)
  })

  it('menahan isian yang sudah diketik bila pengirimannya ditolak', async () => {
    // Kalimat yang sudah diketik adalah pekerjaan penggunanya. Menutup form saat permintaan
    // ditolak membuang pekerjaan itu tanpa ia sempat menyalinnya.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url === BRANCH_PATH) return jsonResponse(200, BRANCH_LIST)
      if (url === MESSAGE_PATH) {
        return jsonResponse(503, {
          kode: 'sumber_cabang_tidak_terbaca',
          pesan: 'Sumber data cabang sedang tidak dapat dibaca.',
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

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByRole('form', { name: /Kirim pesan baru/ })

    const box = screen.getByLabelText('Pesan')
    await userEvent.type(box, 'Kalimat yang tidak boleh hilang.')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Pesan' }))

    expect(
      await screen.findByText(/Sumber data cabang sedang tidak dapat dibaca/),
    ).toBeInTheDocument()
    expect(box).toHaveValue('Kalimat yang tidak boleh hilang.')
  })

  it('menyatakan keadaan ketika tidak ada satu pun cabang yang dapat dipilih', async () => {
    // Nyata di produksi bila `V_D_SURVEYORS` kosong atau kueri yang menyusun daftarnya salah
    // sasaran — dan kuerinya memang DITEBAK: yang asli tidak ada di export mana pun.
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url === BRANCH_PATH) {
        return jsonResponse(200, { cabang: [], tujuan: ['PUSAT', 'CABANG'], portal: 'ASM' })
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

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.selectOptions(await screen.findByLabelText('Tujuan'), 'CABANG')

    expect(await screen.findByText(/Tidak ada cabang yang dapat dipilih/)).toBeInTheDocument()
  })
})
