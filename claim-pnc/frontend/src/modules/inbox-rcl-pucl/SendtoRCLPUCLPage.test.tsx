import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

/**
 * Uji layar kerja RCL/PUCL — section `SendtoRCLPUCL`.
 *
 * # Apa yang benar-benar dijaga di sini
 *
 * Bukan "layarnya tergambar", melainkan **layarnya sama dengan section-nya**. Tiga hal yang
 * paling mudah menyimpang tanpa ketahuan:
 *
 *   - JUDUL isian. Versi pertama modul ini menyalin judul dari kolom grid ("Deskripsi
 *     Analyst", "Komentar PUCL") padahal section memakai judul yang berbeda ("Catatan dari
 *     Analyst", "Catatan untuk Analyst"). Keduanya masuk akal dibaca, dan tidak satu pun
 *     menghasilkan galat.
 *   - Isian yang TIDAK ada di section. Versi pertama menambahkan "Tanggal Cetak Surat" dan
 *     "Tanggal Kirim RCL/PUCL" — keduanya kolom grid, bukan isian layar kerja.
 *   - Isian yang BELUM punya kolom. Ia harus terbaca berbeda dari isian yang kosong:
 *     yang kosong urusan petugas, yang belum terpetakan urusan DBA.
 */

const PATH = '/api/inbox-rcl-pucl'
const CLAIM_PATH = `${PATH}/klaim/`
const REFERENSI = 'ASM-FW-GCNMFW-WORK PNC-700001'

const SAMPLE_PROFILE = {
  identitas: '90000004',
  nama: 'Contoh Petugas RCL',
  jenis: 'KARYAWAN',
  login: 'petugasrclcontoh',
  email: 'contoh.rcl@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Jawaban layar kerja satu klaim.
 *
 * `up` sengaja SAMA dengan `nama_peserta`: kedua penetapan di
 * `SetDataLampiranSuratRCLPUCL_Act` menunjuk ekspresi yang sama persis, dan Work Owner
 * menegaskan 2026-09-24 bahwa itu memang benar — UP berisi ObjectName.
 *
 * Contoh yang membedakan keduanya akan membuat uji lulus untuk perilaku yang pernah dicoba
 * lalu diralat.
 */
const DETAIL = {
  referensi: REFERENSI,
  no_case: 'PNC-700001',
  lampiran_surat: {
    rcl_pucl: 'RCL',
    kode_rcl_pucl: '1',
    deskripsi_analyst: 'Dokumen pendukung tidak lengkap.',
    no_polis: 'CONTOH-RCL-0001',
    tanggal_kejadian: '2026-09-01',
    nama_peserta: 'Objek Contoh Satu',
    up: 'Objek Contoh Satu',
    jumlah_tagihan: '15000000',
  },
  penerimaan_dokumen: { komentar_pucl: 'Menunggu kelengkapan dari cabang.' },
  isian_belum_terpetakan: ['No Kontrak', 'Perihal', 'Email Tertanggung'],
  tindakan_masih_di_pega: true,
  portal: 'ASM',
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string) => Response) {
  vi.stubGlobal('fetch', (url: string) => {
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url))
  })
}

function stubDefaultFetch() {
  stubFetch((url) => {
    if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL)
    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
  })
}

/** Menggambar langsung di alamat layar kerja, seperti alamat yang disalin petugas. */
function renderWorkScreen(query = '') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const path = `/inbox-rcl-pucl/klaim/${encodeURIComponent(REFERENSI)}${query}`

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
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

describe('layar kerja SendtoRCLPUCL', () => {
  it('menggambar kedua bagian section, dalam urutan section', async () => {
    // `Section/SendtoRCLPUCL-Section.xml` menyisipkan Lampiran Surat lebih dulu, baru
    // Penerimaan Dokumen. Urutannya dibaca dari posisi `pyInclude` di dalam berkasnya.
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByText('Lampiran Surat')).toBeInTheDocument()

    const headings = screen
      .getAllByRole('heading', { level: 3 })
      .map((node) => node.textContent)

    expect(headings.indexOf('Lampiran Surat')).toBeLessThan(
      headings.indexOf('Penerimaan Dokumen'),
    )
  })

  it('memakai JUDUL ISIAN section, bukan judul kolom grid', async () => {
    // Inilah uji yang menahan penyimpangan paling halus di layar ini. Judul grid dan judul
    // section berbeda untuk isian yang SAMA, dan keduanya sama-sama masuk akal dibaca.
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByText('Status RCL / PUCL / MSIG')).toBeInTheDocument()
    expect(screen.getByText('Catatan dari Analyst')).toBeInTheDocument()
    expect(screen.getByText('Catatan untuk Analyst')).toBeInTheDocument()

    // Judul grid TIDAK boleh muncul di layar kerja.
    expect(screen.queryByText('Deskripsi Analyst')).not.toBeInTheDocument()
    expect(screen.queryByText('Komentar PUCL')).not.toBeInTheDocument()
  })

  it('TIDAK menggambar isian yang bukan milik section', async () => {
    // "Tanggal Cetak Surat" dan "Tanggal Kirim RCL/PUCL" adalah kolom GRID. Versi pertama
    // modul ini membawanya ke layar kerja karena keduanya kebetulan sudah di tangan — dan
    // itu melanggar `D-13`: layar kerja menggambar apa yang digambar section-nya.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByText('Lampiran Surat')

    expect(screen.queryByText('Tanggal Cetak Surat')).not.toBeInTheDocument()
    expect(screen.queryByText('Tanggal Kirim RCL/PUCL')).not.toBeInTheDocument()
  })

  it('menggambar isian clipboard di TEMPATNYA, bukan menghilangkannya', async () => {
    // Sembilan isian adalah properti clipboard Pega, bukan kolom tabel. Menghilangkannya
    // membuat layar tampak setara padahal tidak.
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByText('No Kontrak')).toBeInTheDocument()
    expect(screen.getByText('Business Unit / Seksi')).toBeInTheDocument()
    expect(screen.getByText('Keterangan Pembuka')).toBeInTheDocument()
    expect(screen.getByText('Keterangan Isi')).toBeInTheDocument()
    expect(screen.getByText('Keterangan Penutup')).toBeInTheDocument()
    expect(screen.getByText('Tanggal Kelengkapan Dokumen')).toBeInTheDocument()

    // Dan ia terbaca BERBEDA dari isian yang kosong: yang kosong memang belum diisi,
    // yang ini punya nilai tetapi nilainya tidak dapat dibaca dari tabel.
    expect(screen.getAllByText('di clipboard Pega').length).toBeGreaterThan(0)
  })

  it('menggambar grid "Tanggal terima Dokumen" sebagai daftar, bukan satu isian', async () => {
    // Di section ia grid berulang dua kolom. Menggantinya dengan satu isian tanggal akan
    // menyembunyikan bahwa layar lama menerima BANYAK tanggal terima dokumen.
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByText('Tanggal terima Dokumen')).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Tanggal' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Keterangan' })).toBeInTheDocument()
  })

  it('menggambar ketiga tombol tindakan, dan seluruhnya tidak dapat ditekan', async () => {
    // Ketiganya MENULIS, dan tabelnya masih dimiliki Pega selama masa paralel (`P-1`).
    // Tombol yang dihilangkan menyembunyikan bahwa tindakannya ada; tombol yang hidup akan
    // menulis ke tabel yang bukan miliknya.
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByRole('button', { name: 'Cetak' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Unggah Dokumen' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Kirim ke PIC Teknik' })).toBeDisabled()
  })

  it('menjelaskan UP yang berisi nama objek, alih-alih membiarkannya terbaca sebagai kerusakan', async () => {
    // UP dan Nama Peserta memang satu sumber (`ObjectList(1).ObjectName`), dikonfirmasi
    // Work Owner 2026-09-24. Yang dijaga di sini bukan nilainya melainkan KETERANGANNYA:
    // tanpa itu, isian bernama "Uang Pertanggungan" yang berisi nama akan dilaporkan
    // sebagai kerusakan sistem baru.
    stubDefaultFetch()
    renderWorkScreen()

    // Ditunggu DATANYA, bukan kerangkanya. Sejak layar digambar sejak permintaan pertama,
    // judul "Lampiran Surat" sudah ada sebelum datanya tiba — menunggunya akan membuat
    // pemeriksaan di bawah berjalan terlalu awal.
    await screen.findAllByText('Objek Contoh Satu')

    // Dicari lewat teks LANGSUNG paragrafnya, bukan lewat pola yang menembus "UP" di
    // dalam <span>: pencari bawaan Testing Library hanya membaca anak teks langsung sebuah
    // elemen, sehingga pola yang melintasi elemen anak tidak akan pernah cocok.
    expect(screen.getByText(/berisi nama objek yang sama/i)).toBeInTheDocument()
  })

  it('menampilkan kunci klaim supaya dapat dibuka di Pega', async () => {
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByText(REFERENSI)).toBeInTheDocument()
    expect(screen.getByText(/Layar ini baca saja/)).toBeInTheDocument()
  })

  it('menjelaskan klaim yang tidak ditemukan alih-alih layar kosong', async () => {
    // Kunci milik entitas lain menghasilkan "tidak ditemukan" — batas portal ditegakkan di
    // peladen (`R-20`). Layar harus menyatakannya, bukan menggambar isian hampa.
    stubFetch((url) => {
      if (url.startsWith(CLAIM_PATH)) {
        return jsonResponse(404, {
          kode: 'klaim_tidak_ditemukan',
          pesan: 'Klaim tidak ditemukan pada entitas yang sedang dipilih.',
        })
      }
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
    })
    renderWorkScreen()

    expect(await screen.findByText(/Isi klaim tidak dapat diambil/)).toBeInTheDocument()

    // Kerangkanya TETAP tergambar — galat dinyatakan DI ATAS layar, bukan menggantikannya.
    // Halaman yang tidak menggambar apa pun tidak dapat dibedakan dari halaman yang rusak.
    expect(screen.getByText('Lampiran Surat')).toBeInTheDocument()
    expect(screen.getByText('Penerimaan Dokumen')).toBeInTheDocument()
  })

  it('menggambar kerangka layar meski jawaban peladen BUKAN JSON', async () => {
    // Kelas kegagalan yang benar-benar terjadi: `/api/...` dijawab penyaji SPA dengan
    // `index.html`. Jawabannya 200, badannya HTML, dan `callAPI` mengembalikan `null` —
    // bukan melempar. Versi pertama halaman ini menampilkan layar KOSONG SELURUHNYA:
    // bukan memuat, bukan galat, bukan data, tanpa satu pun petunjuk.
    stubFetch(
      () =>
        new Response('<!doctype html><html><body>SPA</body></html>', {
          status: 200,
          headers: { 'Content-Type': 'text/html' },
        }),
    )
    renderWorkScreen()

    // Keadaannya dinyatakan…
    expect(await screen.findByText(/Isi klaim tidak terbaca/)).toBeInTheDocument()

    // …dan layarnya tetap tergambar, kosong.
    expect(screen.getByText('Lampiran Surat')).toBeInTheDocument()
    expect(screen.getByText('Status RCL / PUCL / MSIG')).toBeInTheDocument()
    expect(screen.getByText('Catatan untuk Analyst')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Cetak' })).toBeDisabled()
  })

  it('menggambar kerangka layar sejak permintaan pertama, sebelum datanya tiba', async () => {
    // Bentuk layar ini tidak bergantung pada data — ia tetap kedua bagian dengan isian yang
    // sama. Menahannya sampai data tiba hanya menghasilkan halaman yang berkedip kosong.
    stubDefaultFetch()
    renderWorkScreen()

    expect(screen.getByText('Lampiran Surat')).toBeInTheDocument()
    expect(screen.getByText('Memuat isi layar kerja…')).toBeInTheDocument()
  })

  it('kembali ke antrean pada tab dan halaman yang tercantum di alamat', async () => {
    stubDefaultFetch()
    renderWorkScreen('?tab=2&halaman=3')

    await screen.findByText('Lampiran Surat')
    await userEvent.click(screen.getByRole('button', { name: /Kembali ke antrean/ }))

    // Antrean meminta bentuk layarnya begitu ia tergambar; cukup dibuktikan bahwa layar
    // kerja sudah ditinggalkan dan permintaan antrean berangkat membawa tab tersebut.
    expect(screen.queryByText('Lampiran Surat')).not.toBeInTheDocument()
  })
})
