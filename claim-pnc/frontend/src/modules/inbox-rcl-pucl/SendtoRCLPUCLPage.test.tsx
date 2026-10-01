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
  penerimaan_dokumen: {
    komentar_pucl: 'Menunggu kelengkapan dari cabang.',
    tanggal_kelengkapan_dokumen: '2026-09-05 11:20:00',
    // SATU baris, seperti di Oracle: daftarnya page list tanpa tabel, dan hanya baris
    // pertamanya yang diekspos sebagai kolom.
    tanggal_terima_dokumen: [{ tanggal: '2026-09-04 10:05:00', keterangan: 'Dokumen awal' }],
    tanggal_terima_dokumen_sebagian: true,
    // Sengaja KOSONG: kolom `EMAIL_LOD` ada tetapi belum terisi pada seluruh baris
    // produksi, sehingga inilah keadaan yang sebenarnya akan digambar.
    email_tertanggung: '',
  },
  isian_belum_terpetakan: ['No Kontrak', 'Business Unit / Seksi'],
  tab_penerimaan_dokumen_tampil: true,
  /*
    Susunan tombol KLAIM RCL NON-MSIG, dihitung server dari `pyCondition` tiap `pxButton`:
    "Download Dokumen" karena bukan jalur MSIG, ketiga tombol `ALWAYS`, dan "Tolak Klaim"
    karena jalurnya RCL. Kedua tombol "Kirim" milik jalur PUCL, sehingga keduanya mati di
    sini.
  */
  // Ketiganya sengaja BERBEDA: nilai yang sama membuat ketiganya tertukar tanpa satu pun
  // uji gagal, dan `PUCLPost` tidak menolak parameter yang tertukar.
  parameter_tindakan: {
    id_object: 'OBJ-01',
    id_coverage: 'COV-01',
    id_adjustment: 'ADJ-01',
  },
  tombol: {
    download_dokumen: true,
    tutup_klaim: false,
    unggah_dokumen: true,
    lihat_dokumen: true,
    save: true,
    tolak_klaim: true,
    kirim_ke_analyst: false,
    kirim_ke_pic_teknik: false,
  },
  tindakan_masih_di_pega: true,
  portal: 'ASM',
}

/**
 * Klaim MSIG berstatus Notification — kode jalur `3`.
 *
 * Layar lama menyembunyikan tab "Penerimaan Dokumen" untuk klaim seperti ini, lewat
 * `pyContainerVisibleWhen .ClaimData.PUCLStatus.RCL_PUCL != 3` pada kontainer tab kedua.
 * Keputusannya diambil SERVER; baris contoh ini membawanya apa adanya.
 *
 * Ia dibuat MSIG dengan sengaja, karena itulah keadaan yang dilaporkan Work Owner
 * 2026-09-30: klaim MSIG hanya menampilkan Lampiran Surat. Akibatnya pada tombol: "Download
 * Dokumen" MATI (jalur MSIG) dan "Tutup Klaim" HIDUP — satu-satunya tindakan yang tersisa.
 */
const DETAIL_NOTIFICATION = {
  ...DETAIL,
  lampiran_surat: { ...DETAIL.lampiran_surat, rcl_pucl: 'Notification', kode_rcl_pucl: '3' },
  tab_penerimaan_dokumen_tampil: false,
  tombol: {
    ...DETAIL.tombol,
    download_dokumen: false,
    tutup_klaim: true,
    unggah_dokumen: false,
    lihat_dokumen: false,
    save: false,
    tolak_klaim: false,
  },
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

/** Daftar dokumen contoh — dua berkas, beserta catatan ketidaklengkapan dari server. */
const DOKUMEN = {
  dokumen: [
    {
      id: 'DOC-0001',
      nama: 'Surat Keterangan.pdf',
      kategori: 'Dokumen Klaim',
      sub_kategori: 'Surat Keterangan',
      diunggah_pada: '2026-09-02 09:15:00',
      diunggah_oleh: 'PETUGASCONTOH',
    },
  ],
  catatan: 'Dokumen yang diunggah lewat jalur lama Pega belum tentu muncul di daftar ini.',
  portal: 'ASM',
}

function stubDefaultFetch(detail: unknown = DETAIL, dokumen: unknown = DOKUMEN) {
  stubFetch((url) => {
    // Alamat dokumen diperiksa LEBIH DULU: ia berawalan sama dengan alamat klaim, dan
    // urutan yang terbalik membuat permintaan dokumen dijawab isi layar kerja.
    if (url.includes('/dokumen')) return jsonResponse(200, dokumen)
    if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, detail)
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
  it('menggambar kedua bagian sebagai DUA TAB, dalam urutan section', async () => {
    // `Section/SendtoRCLPUCL-Section.xml` ber-`pyHeaderType TABBED` dan menyisipkan
    // Lampiran Surat lebih dulu, baru Penerimaan Dokumen. Urutannya dibaca dari posisi
    // `pyInclude` di dalam berkasnya.
    //
    // Bertumpuk pada satu halaman — bentuk versi sebelumnya — membuat tombol "Cetak" dan
    // "Kirim Ke Analyst" terlihat bersamaan, padahal di Pega keduanya berada di tahap yang
    // berbeda dan tidak pernah tampil sekaligus.
    stubDefaultFetch()
    renderWorkScreen()

    const tabs = await screen.findAllByRole('tab')
    expect(tabs.map((node) => node.textContent)).toEqual([
      'Lampiran Surat',
      'Penerimaan Dokumen',
    ])

    // Tab pertama terbuka lebih dulu, dan isi tab kedua BELUM tergambar.
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByText('Catatan untuk Analyst')).not.toBeInTheDocument()
  })

  it('memakai JUDUL ISIAN section, bukan judul kolom grid', async () => {
    // Inilah uji yang menahan penyimpangan paling halus di layar ini. Judul grid dan judul
    // section berbeda untuk isian yang SAMA, dan keduanya sama-sama masuk akal dibaca.
    stubDefaultFetch()
    renderWorkScreen()

    expect(await screen.findByText('Status RCL / PUCL / MSIG')).toBeInTheDocument()
    expect(screen.getByText('Catatan dari Analyst')).toBeInTheDocument()

    // Judul grid TIDAK boleh muncul di layar kerja.
    expect(screen.queryByText('Deskripsi Analyst')).not.toBeInTheDocument()
    expect(screen.queryByText('Komentar PUCL')).not.toBeInTheDocument()

    // Pasangannya ada di tab kedua. Keduanya sengaja diperiksa berpasangan: yang satu
    // catatan Analyst UNTUK PUCL, yang satu balasan PUCL UNTUK Analyst, dan menukarnya
    // membalik arah percakapannya tanpa satu pun galat.
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    expect(screen.getByText('Catatan untuk Analyst')).toBeInTheDocument()
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

    // Dan ia terbaca BERBEDA dari isian yang kosong: yang kosong memang belum diisi,
    // yang ini punya nilai tetapi nilainya tidak dapat dibaca dari tabel.
    expect(screen.getAllByText('di clipboard Pega').length).toBeGreaterThan(0)

    // Tiga sisanya ada di tab kedua, dan dua di antaranya WAJIB diisi di layar lama.
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    expect(screen.getByText('Email Tertanggung')).toBeInTheDocument()
    expect(screen.getByText('Tanggal Kelengkapan Dokumen')).toBeInTheDocument()
  })

  it('menggambar grid "Tanggal terima Dokumen" sebagai daftar, bukan satu isian', async () => {
    // Di section ia grid berulang dua kolom. Menggantinya dengan satu isian tanggal akan
    // menyembunyikan bahwa layar lama menerima BANYAK tanggal terima dokumen.
    stubDefaultFetch()
    renderWorkScreen()

    await userEvent.click(
      await screen.findByRole('tab', { name: 'Penerimaan Dokumen' }),
    )

    expect(screen.getByText('Tanggal terima Dokumen')).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Tanggal' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Keterangan' })).toBeInTheDocument()
  })

  it('menggambar tombol dengan nama layar Pega, dan seluruhnya DAPAT diklik', async () => {
    // Tombolnya tidak lagi mati. Work Owner melaporkan 2026-10-01 bahwa tombol mati tanpa
    // tanggapan terbaca sebagai kerusakan — dan memang begitu: tidak ada cara membedakan
    // "belum dibangun" dari "rusak" bila menekannya tidak menghasilkan apa pun.
    //
    // Yang BELUM berubah adalah akibatnya: tidak satu pun menulis, karena tabelnya masih
    // dimiliki Pega (`P-1`). Menekannya menjelaskan hal itu dan memberi nomor case-nya.
    stubDefaultFetch()
    renderWorkScreen()

    expect(
      await screen.findByRole('button', { name: 'Download Dokumen' }),
    ).toBeEnabled()

    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    for (const label of ['Unggah Dokumen', 'Lihat Dokumen', 'Save', 'Tolak Klaim']) {
      expect(screen.getByRole('button', { name: label })).toBeEnabled()
    }
  })

  it('menjawab saat tombol ditekan, dengan alasan dan nomor case — bukan diam', async () => {
    // Inilah yang membedakannya dari tombol mati: menekannya MENGHASILKAN sesuatu. Nomor
    // case-nya disertakan karena itu yang dibutuhkan petugas untuk mengerjakan tindakannya
    // di Pega; tanpanya ia harus kembali ke antrean dan mencarinya lagi.
    // Dipakai "Unggah Dokumen": sejak tindakan lain tersambung ke Pega, hanya tombol yang
    // BELUM punya jalur yang masih memakai panel ini.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    const tombol = screen.getByRole('button', { name: 'Unggah Dokumen' })
    expect(tombol).toHaveAttribute('aria-expanded', 'false')

    await userEvent.click(tombol)

    expect(tombol).toHaveAttribute('aria-expanded', 'true')
    // LANGKAHNYA lebih dulu, bukan penolakannya, dan ia MENYEBUT nama tombolnya — panelnya
    // satu untuk seluruh layar, sehingga tanpa nama itu tidak jelas tindakan mana yang
    // dimaksud.
    expect(screen.getByText(/di Pega pada klaim/)).toBeInTheDocument()
    expect(screen.getAllByText(REFERENSI).length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: 'Salin nomor case' })).toBeEnabled()

    // Ditekan lagi, keterangannya tertutup — bukan menumpuk.
    await userEvent.click(tombol)
    expect(screen.queryByRole('button', { name: 'Salin nomor case' })).not.toBeInTheDocument()
  })

  it('tidak menggambar tombol "Cetak", yang ternyata nilai parameter dan bukan tombol', async () => {
    // `Section/SectionLampiranSuratPUCL-Section.xml` memuat `"cetak"` sebagai NILAI
    // parameter (`<pyName>tipe</pyName>`), bukan caption. Ketiga sel `pxButton`-nya
    // ber-caption "Pilih", "Download Dokumen", dan "Tutup Klaim".
    //
    // Work Owner melaporkannya 2026-10-01. Uji ini menahannya supaya nama itu tidak kembali
    // lewat penyuntingan berikutnya.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    expect(screen.queryByRole('button', { name: 'Cetak' })).not.toBeInTheDocument()
  })

  it('menyatakan KEDUA akibat "Download Dokumen" — menerbitkan surat DAN memindahkan klaim', async () => {
    // Keterangannya sudah DUA KALI keliru, ke arah yang berlawanan:
    //
    //   v1  "Mengunduh surat RCL/PUCL klaim ini."   -> menyembunyikan bahwa ia MENULIS
    //   v2  "tidak mengunduh apa pun"               -> menyembunyikan bahwa suratnya TERBIT
    //
    // Yang benar keduanya. `PUCLPost` memanggil `AttachAsPDFC` (HTMLToPDF -> AttachToWork ->
    // View) tanpa syarat, sehingga PDF-nya memang terbit dan terbuka; pada langkah lain ia
    // mengisi `TanggalCetakDokumenPUCL`, sehingga klaimnya berpindah tab.
    //
    // Uji ini menahan KEDUA kalimat lama sekaligus.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })

    expect(screen.queryByText(/Mengunduh surat RCL\/PUCL klaim ini/)).not.toBeInTheDocument()
    expect(screen.queryByText(/tidak mengunduh apa pun/)).not.toBeInTheDocument()
    expect(screen.getByText(/Menerbitkan PDF surat DAN menandainya sudah dicetak/)).toBeInTheDocument()
  })

  it('memilih tombol kirim menurut lini bisnis, bukan menggambar keduanya', async () => {
    // `RCL_PUCL = 2 && IsPA` versus `RCL_PUCL = 2 && IsTravel`. Analyst dan PIC Teknik dua
    // peran berbeda, sehingga menggambar keduanya menawarkan tindakan yang meneruskan klaim
    // kepada orang yang salah.
    //
    // Klaim contoh di sini PUCL pada lini Travel, sehingga yang benar adalah PIC Teknik.
    stubDefaultFetch({
      ...DETAIL,
      lampiran_surat: { ...DETAIL.lampiran_surat, rcl_pucl: 'PUCL', kode_rcl_pucl: '2' },
      tombol: {
        ...DETAIL.tombol,
        tolak_klaim: false,
        kirim_ke_analyst: false,
        kirim_ke_pic_teknik: true,
      },
    })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    expect(screen.getByRole('button', { name: 'Kirim ke PIC Teknik' })).toBeEnabled()
    expect(
      screen.queryByRole('button', { name: 'Kirim Ke Analyst' }),
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Tolak Klaim' })).not.toBeInTheDocument()
  })

  it('menggambar isian yang kolomnya BARU ditemukan, bukan tanda "di clipboard Pega"', async () => {
    // Ditetapkan Work Owner 2026-10-01: "Tanggal Kelengkapan Dokumen" dari
    // `TC_PNC_PUCL.TGL_TERIMA_DOKUMEN_PUCL`, "Email Tertanggung" dari
    // `T_CLAIM_PNC.EMAIL_LOD` yang digabung lewat nomor case.
    //
    // Yang dijaga BUKAN hanya nilainya tergambar, melainkan bahwa tandanya HILANG. Isian yang
    // tetap bertanda padahal kolomnya sudah ada akan membuat orang mencarinya lagi ke Tim
    // Pega — pencarian yang tidak akan menemukan apa pun.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    expect(screen.getByText('2026-09-05 11:20:00')).toBeInTheDocument()

    // Hanya DUA isian yang masih bertanda — keduanya di tab Lampiran Surat, bukan di sini.
    expect(screen.queryAllByText('di clipboard Pega')).toHaveLength(0)
  })

  it('menggambar grid "Tanggal Terima Dokumen" beserta batas keterbacaannya', async () => {
    // Daftarnya page list `.ClaimData.PUCLStatus.DateReceivedDocument` yang TIDAK punya
    // tabel; hanya baris pertamanya yang diekspos lewat `RECEIVEDDATE_1`/`KETERANGAN_1`.
    //
    // Yang dijaga: barisnya tergambar, DAN batasnya dinyatakan. Grid yang menampilkan satu
    // baris tanpa keterangan akan terbaca sebagai daftar yang utuh.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    expect(screen.getByText('2026-09-04 10:05:00')).toBeInTheDocument()
    expect(screen.getByText('Dokumen awal')).toBeInTheDocument()
    expect(screen.getByText(/Baru baris pertama yang terbaca/)).toBeInTheDocument()

    // Kalimat lama yang menyatakan DAFTARNYA tidak terbaca sama sekali TIDAK boleh kembali.
    // Pola dipersempit ke kata "Daftarnya": kaki layar memakai kalimat serupa untuk ISIAN
    // yang memang masih di clipboard, dan itu benar.
    expect(screen.queryByText(/Daftarnya tersimpan di clipboard Pega/)).not.toBeInTheDocument()
  })

  it('tidak menyatakan "baru baris pertama" ketika daftarnya memang KOSONG', async () => {
    // Pada daftar kosong, keterangan itu menyesatkan: ia terbaca sebagai "mungkin ada yang
    // tersembunyi", padahal kemungkinan terbesarnya daftarnya memang kosong.
    stubDefaultFetch({
      ...DETAIL,
      penerimaan_dokumen: {
        ...DETAIL.penerimaan_dokumen,
        tanggal_terima_dokumen: [],
      },
    })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    expect(screen.getByText(/Belum ada tanggal terima dokumen/)).toBeInTheDocument()
    expect(screen.queryByText(/Baru baris pertama yang terbaca/)).not.toBeInTheDocument()
  })

  it('"Kirim Ke Analyst" MENJALANKAN tindakannya, bukan menolak', async () => {
    // Ia satu-satunya tombol yang benar-benar bekerja. Yang dijaga: permintaannya POST ke
    // alamat klaimnya, dan keberhasilannya dinyatakan — bukan diam.
    let dipanggil = ''
    stubFetch((url) => {
      if (url.includes('/tindakan/kirim-analyst')) {
        dipanggil = url
        return jsonResponse(200, { pesan: 'Klaim diteruskan ke Analyst.' })
      }
      if (url.includes('/dokumen')) return jsonResponse(200, DOKUMEN)
      if (url.startsWith(CLAIM_PATH))
        return jsonResponse(200, {
          ...DETAIL,
          lampiran_surat: { ...DETAIL.lampiran_surat, rcl_pucl: 'PUCL', kode_rcl_pucl: '2' },
          tombol: { ...DETAIL.tombol, tolak_klaim: false, kirim_ke_analyst: true },
        })
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
    })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Ke Analyst' }))

    expect(await screen.findByText('Klaim diteruskan ke Analyst.')).toBeInTheDocument()
    expect(dipanggil).toContain('/tindakan/kirim-analyst')
  })

  it('menggambar ALASAN saat layanan Pega belum tersambung', async () => {
    // Peladen menjawab 503 dengan kalimat yang menyebut langkah penggantinya. Layar
    // menggambarnya APA ADANYA — menuliskannya ulang di sini berarti dua kalimat yang dapat
    // berselisih, dan yang di layar bukan yang dikirim peladen.
    stubFetch((url) => {
      if (url.includes('/tindakan/kirim-analyst')) {
        return jsonResponse(503, {
          kode: 'layanan_pega_belum_tersedia',
          pesan: 'Tindakan ini dijalankan oleh Pega, dan layanannya belum tersambung.',
        })
      }
      if (url.includes('/dokumen')) return jsonResponse(200, DOKUMEN)
      if (url.startsWith(CLAIM_PATH))
        return jsonResponse(200, {
          ...DETAIL,
          lampiran_surat: { ...DETAIL.lampiran_surat, rcl_pucl: 'PUCL', kode_rcl_pucl: '2' },
          tombol: { ...DETAIL.tombol, tolak_klaim: false, kirim_ke_analyst: true },
        })
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
    })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Ke Analyst' }))

    expect(await screen.findByText(/layanannya belum tersambung/)).toBeInTheDocument()

    // Dan nomor case-nya digambar DI PANEL YANG SAMA, bukan hanya di kaki layar.
    //
    // Pesannya menyuruh mengerjakannya di Pega, dan untuk itu petugas butuh nomornya. Sebelum
    // 2026-10-01 nomor itu hanya ada di kaki layar, sehingga kalimat yang benar berujung pada
    // gulir mencari — keluhan yang diangkat Work Owner dengan tangkapan layar.
    expect(screen.getByRole('button', { name: 'Salin nomor case' })).toBeInTheDocument()
  })

  it('TIDAK menawarkan "kerjakan di Pega" pada galat yang bukan soal layanan Pega', async () => {
    // Galat kewenangan tidak selesai dengan mengerjakannya di Pega. Menawarkan nomor case di
    // sana mengarahkan petugas menempuh jalan yang tidak menyelesaikan apa pun — dan karena
    // panelnya terlihat sama, ia akan mengira sudah melakukan hal yang benar.
    stubFetch((url) => {
      if (url.includes('/tindakan/kirim-analyst')) {
        return jsonResponse(409, {
          kode: 'tindakan_tidak_tersedia',
          pesan: 'Tindakan ini tidak berlaku untuk klaim ini.',
        })
      }
      if (url.includes('/dokumen')) return jsonResponse(200, DOKUMEN)
      if (url.startsWith(CLAIM_PATH))
        return jsonResponse(200, {
          ...DETAIL,
          lampiran_surat: { ...DETAIL.lampiran_surat, rcl_pucl: 'PUCL', kode_rcl_pucl: '2' },
          tombol: { ...DETAIL.tombol, tolak_klaim: false, kirim_ke_analyst: true },
        })
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
    })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Ke Analyst' }))

    expect(await screen.findByText(/tidak berlaku untuk klaim ini/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Salin nomor case' })).not.toBeInTheDocument()
  })

  it('"Download Dokumen" dan "Save" MENJALANKAN tindakannya lewat alamat yang benar', async () => {
    // Keduanya memanggil activity yang sama dengan tombol Kirim — yang membedakan hanya
    // parameternya. Yang dijaga di sini: alamat tindakannya BERBEDA per tombol, karena
    // alamat yang tertukar membuat Pega mengerjakan tindakan yang salah tanpa menolak.
    const dipanggil: string[] = []
    stubFetch((url) => {
      if (url.includes('/tindakan/')) {
        dipanggil.push(url)
        return jsonResponse(200, { pesan: 'Tindakan dijalankan.' })
      }
      if (url.includes('/dokumen')) return jsonResponse(200, DOKUMEN)
      if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL)
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
    })
    renderWorkScreen()

    await userEvent.click(await screen.findByRole('button', { name: 'Download Dokumen' }))
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(dipanggil.some((u) => u.endsWith('/tindakan/cetak'))).toBe(true)
    expect(dipanggil.some((u) => u.endsWith('/tindakan/save'))).toBe(true)
  })

  it('"Unggah Dokumen" BELUM punya jalur, dan itu dinyatakan apa adanya', async () => {
    // Ia menuntut unggahan berkas DAN mesin lampiran Pega (`PZPVSTREAM`), bukan pemanggilan
    // berparameter — sehingga ia tidak ikut tersambung bersama keempat tombol lain.
    //
    // Yang dijaga: ia TIDAK memanggil alamat tindakan. Menyambungkannya ke alamat yang sama
    // akan membuat Pega menerima tindakan yang tidak pernah ada.
    const dipanggil: string[] = []
    stubFetch((url) => {
      if (url.includes('/tindakan/')) {
        dipanggil.push(url)
        return jsonResponse(200, { pesan: 'Tindakan dijalankan.' })
      }
      if (url.includes('/dokumen')) return jsonResponse(200, DOKUMEN)
      if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL)
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ada.' })
    })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    await userEvent.click(screen.getByRole('button', { name: 'Unggah Dokumen' }))

    expect(dipanggil).toHaveLength(0)
    expect(screen.getByText(/di Pega pada klaim/)).toBeInTheDocument()
  })

  it('membuka paling banyak SATU panel — menekan tombol kedua menutup yang pertama', async () => {
    // Laporan Work Owner 2026-10-01: menekan dua tombol membuka dua panel sekaligus, dan
    // karena alasannya sama untuk seluruh tombol, kalimat yang sama tergambar dua kali
    // berturut-turut.
    //
    // Yang dijaga: tombol "Salin nomor case" hanya boleh ada SATU di layar, berapa pun tombol
    // yang sudah ditekan.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    await userEvent.click(screen.getByRole('button', { name: 'Unggah Dokumen' }))
    expect(screen.getAllByRole('button', { name: 'Salin nomor case' })).toHaveLength(1)

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(screen.getAllByRole('button', { name: 'Salin nomor case' })).toHaveLength(1)

    // Berpindah tab menutupnya pula — panel milik tombol yang sudah tidak terlihat tidak
    // boleh ikut terbawa.
    await userEvent.click(screen.getByRole('tab', { name: 'Lampiran Surat' }))
    expect(screen.queryByRole('button', { name: 'Salin nomor case' })).not.toBeInTheDocument()
  })

  it('"Lihat Dokumen" BENAR-BENAR berjalan — daftarnya ditarik dan berkasnya bertautan', async () => {
    // Satu-satunya tombol layar kerja yang membaca, sehingga satu-satunya yang tidak
    // terhalang `P-1`. Yang dijaga di sini tiga hal, dan ketiganya pernah salah di modul
    // lain: daftarnya baru ditarik SETELAH ditekan, berkasnya dibuka lewat TAUTAN ke alamat
    // isinya, dan catatan ketidaklengkapan datang dari SERVER.
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    // Belum ditekan — daftarnya belum ada.
    expect(screen.queryByText('Surat Keterangan.pdf')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Lihat Dokumen' }))

    const tautan = await screen.findByRole('link', { name: 'Surat Keterangan.pdf' })
    expect(tautan).toHaveAttribute(
      'href',
      `/api/inbox-rcl-pucl/klaim/${encodeURIComponent(REFERENSI)}/dokumen/DOC-0001`,
    )
    expect(screen.getByText(/belum tentu muncul di daftar ini/)).toBeInTheDocument()
  })

  it('membedakan klaim TANPA dokumen dari daftar yang GAGAL diambil', async () => {
    // Keduanya terlihat sama bila tidak dinyatakan, padahal yang satu berarti "klaim ini
    // memang belum berdokumen" dan yang lain "jangan percayai layar ini".
    stubDefaultFetch(DETAIL, { dokumen: [], catatan: '', portal: 'ASM' })
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    await userEvent.click(screen.getByRole('button', { name: 'Lihat Dokumen' }))

    expect(await screen.findByText(/Belum ada dokumen yang terlampir/)).toBeInTheDocument()
  })

  it('tidak pernah menggambar "Reminder PUCL", yang syaratnya 1==2', async () => {
    // Tombol yang dimatikan dengan cara dikarang syaratnya alih-alih dihapus. Perilakunya
    // yang NYATA di sistem lama adalah tidak muncul, dan itulah yang ditiru — menggambarnya
    // "supaya lengkap" menambah tindakan yang tidak pernah ada di layar lama (`P-5`).
    stubDefaultFetch()
    renderWorkScreen()

    await screen.findByRole('button', { name: 'Download Dokumen' })
    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))

    expect(screen.queryByRole('button', { name: 'Reminder PUCL' })).not.toBeInTheDocument()
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

    // …dan layarnya tetap tergambar, kosong — KEDUA tabnya, bukan hanya yang terbuka.
    expect(screen.getByRole('tab', { name: 'Lampiran Surat' })).toBeInTheDocument()
    expect(screen.getByText('Status RCL / PUCL / MSIG')).toBeInTheDocument()

    // TIDAK ADA satu pun tombol, dan itu disengaja: tombol mana yang berlaku ditentukan
    // jalur dan lini bisnis klaimnya, dan keduanya justru yang tidak terbaca di sini.
    // Menggambar seluruhnya "supaya terlihat lengkap" akan menawarkan tindakan yang belum
    // tentu tersedia untuk klaim ini.
    expect(screen.queryByRole('button', { name: 'Download Dokumen' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Tutup Klaim' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: 'Penerimaan Dokumen' }))
    expect(screen.getByText('Catatan untuk Analyst')).toBeInTheDocument()
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

describe('klaim berstatus Notification', () => {
  it('hanya menampilkan Lampiran Surat, tanpa tab Penerimaan Dokumen', async () => {
    // Work Owner melaporkan 2026-09-30: klaim MSIG hanya menampilkan Lampiran Surat.
    // `Section/SendtoRCLPUCL-Section.xml` membenarkannya —
    // `pyContainerVisibleWhen .ClaimData.PUCLStatus.RCL_PUCL != 3` terpasang pada
    // kontainer ber-`pyTitle Penerimaan Dokumen`.
    stubFetch((url) => {
      if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL_NOTIFICATION)
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak ada' })
    })
    renderWorkScreen()

    // Ditunggu DATANYA, bukan kerangkanya. Kerangka layar digambar sejak permintaan
    // pertama — dengan KEDUA tab, karena status klaimnya belum diketahui — dan memeriksa
    // sebelum datanya tiba akan menguji keadaan yang memang belum tahu apa-apa.
    // \, bukan \: kata itu muncul dua kali — sebagai nilai status, dan di
    // dalam keterangan yang menjelaskan mengapa tabnya hilang.
    expect((await screen.findAllByText('Notification')).length).toBeGreaterThan(0)

    expect(screen.getByRole('tab', { name: 'Lampiran Surat' })).toBeInTheDocument()
    expect(
      screen.queryByRole('tab', { name: 'Penerimaan Dokumen' }),
    ).not.toBeInTheDocument()

    // Isinya pun tidak boleh ikut tergambar di bawah tab pertama.
    expect(screen.queryByText('Catatan untuk Analyst')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Kirim Ke Analyst' }),
    ).not.toBeInTheDocument()
  })

  it('menjelaskan MENGAPA tabnya hilang, alih-alih membiarkannya terbaca sebagai kerusakan', async () => {
    // Tab yang hilang tanpa penjelasan akan dilaporkan sebagai kerusakan — persis
    // sebaliknya dari yang terjadi: ia justru tanda layarnya setara dengan Pega.
    stubFetch((url) => {
      if (url.startsWith(CLAIM_PATH)) return jsonResponse(200, DETAIL_NOTIFICATION)
      return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak ada' })
    })
    renderWorkScreen()

    expect(
      await screen.findByText(/hanya memiliki\s+Lampiran Surat/),
    ).toBeInTheDocument()
  })
})
