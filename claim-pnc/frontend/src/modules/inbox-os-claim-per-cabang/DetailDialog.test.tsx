import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, ListResponse, WorkItem, SummaryResponse } from './types'

const PATH = '/api/inbox-os-claim-per-cabang'

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
 * Dua baris contoh, dan keduanya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, termasuk pada data uji.
 *
 * Yang pertama berlini Aneka, yang kedua PA. Perbedaan itu yang membuktikan popup memilih
 * VARIAN KOLOM grid objek, bukan selalu menggambar kolom yang sama.
 */
const ANEKA: WorkItem = {
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

const PA: WorkItem = {
  ...ANEKA,
  cob: 'PA',
  no_klaim: 'PNC-9002',
  aging_hari: 6,
  perlu_perhatian: false,
}

function listResponse(rows: WorkItem[]): ListResponse {
  return {
    data: rows,
    cabang: { kode: '100099', nama: 'CILEGON' },
    paginasi: { halaman: 1, ukuran: 25, total: rows.length, total_halaman: 1 },
    ambang_aging: 180,
    selisih_terencana: ['Daftar dibagi per halaman di server.'],
    portal: 'ASM',
  }
}

const DETAIL_ANEKA: DetailResponse = {
  ringkasan: {
    no_klaim: 'PNC-9001',
    cob: 'Aneka',
    occupation: 'PERKANTORAN',
    total_sum_insured: '5000000.00',
    kronologi: 'Terjadi kebakaran pada malam hari.',
    total_reserve: '250000.00',
    aging_hari: 987,
    perlu_perhatian: true,
    dominant_factor: 'Kelalaian Tertanggung, Dokumen Tidak Lengkap',
    claim_recommendation: 'Disarankan survei ulang.',
    note_pic: 'Menunggu hasil survei',
  },
  objek: [
    {
      id: 'OBJ-1',
      nama: 'GUDANG A',
      lokasi: 'JL CONTOH NO 1',
      pekerjaan: '',
      tanggal_lahir: '',
      ktp_paspor: '',
      status_peserta: '',
      coverage: [
        {
          id: 'CVG-1',
          coverage: 'FLEXAS',
          mata_uang: 'IDR',
          tsi: '3000000.00',
          object_item: [
            {
              id: '1',
              object_item: 'BUILDINGS',
              deskripsi_item: 'Bangunan gudang',
              estimasi: [
                {
                  estimasi_ke: '1',
                  tanggal_estimasi: '2024-01-20 09:00',
                  tipe_estimasi: 'Claim',
                  mata_uang: 'IDR',
                  nilai_kurs: '1.00',
                  nilai_estimasi: '3000000.00',
                },
                {
                  // Nilai NEGATIF — koreksi yang saling meniadakan, ada di data nyata.
                  estimasi_ke: '2',
                  tanggal_estimasi: '2024-01-21 10:00',
                  tipe_estimasi: 'Claim',
                  mata_uang: 'IDR',
                  nilai_kurs: '1.00',
                  nilai_estimasi: '-500000.00',
                },
              ],
            },
          ],
          list_spreading: [
            {
              tipe_treaty: 'FAC-OUT',
              currency: 'IDR',
              estimasi_value: '2500000.00',
              pembagian_persentase: '100.0000',
              result_value: '2500000.00',
            },
          ],
          // CO MEMBER sengaja KOSONG: kedua contoh dari Work Owner memperlihatkan
          // "Data Tidak Ada", dan tabelnya TETAP harus tergambar.
          co_member: [],
        },
      ],
    },
    {
      // Objek KEDUA sengaja tanpa coverage: baris seperti ini tidak boleh dapat dibuka,
      // supaya tidak ada yang mengundang diklik lalu membuka panel kosong.
      id: 'OBJ-2',
      nama: 'GUDANG B',
      lokasi: 'JL CONTOH NO 2',
      pekerjaan: '',
      tanggal_lahir: '',
      ktp_paspor: '',
      status_peserta: '',
      coverage: [],
    },
  ],
  riwayat_progres: [
    {
      tanggal_input: '2024-02-01 16:00',
      no_klaim: 'PNC-9001',
      status_progres_1: 'SURVEY',
      status_progres_2: 'HASIL SURVEY BELUM ADA',
      user_input: 'PICCONTOHSATU',
      tanggal_next_followup: '2024-02-08',
      status: 'OPEN',
      keterangan: 'Menunggu hasil survei',
    },
  ],
  komunikasi_adjuster: [
    {
      nama_user: 'PICCONTOHSATU',
      tanggal_proses: '2024-02-02 15:00',
      pesan: 'Mohon kirimkan laporan survei.',
      tanggal_balas: '2024-02-03 18:00',
      jawaban: 'Laporan sedang disusun.',
      internal: true,
    },
  ],
  cabang: { kode: '100099', nama: 'CILEGON' },
  selisih_terencana: [
    'Nilai "Total Sum Insured" adalah TSI objek terakhir pada klaim, bukan jumlah seluruh objek.',
  ],
  portal: 'ASM',
}

const DETAIL_PA: DetailResponse = {
  ...DETAIL_ANEKA,
  ringkasan: { ...DETAIL_ANEKA.ringkasan, no_klaim: 'PNC-9002', cob: 'PA', occupation: '' },
  objek: [
    {
      id: 'OBJ-PA-1',
      nama: 'BUDI CONTOH',
      lokasi: '',
      pekerjaan: 'TEKNISI',
      tanggal_lahir: '1990-03-17',
      ktp_paspor: '3200000000000001',
      status_peserta: 'KARYAWAN',
      coverage: [],
    },
  ],
  komunikasi_adjuster: [],
}

/** Lini Travel — varian kolom ketiga: Nama Peserta, Status, KTP/Paspor, Tanggal Lahir. */
const DETAIL_TRAVEL: DetailResponse = {
  ...DETAIL_PA,
  ringkasan: { ...DETAIL_PA.ringkasan, cob: 'Travel' },
}

/**
 * Jawaban panel ringkasan. Angkanya KARANGAN, dan sengaja konsisten dengan ketiga baris
 * contoh: 3 berkas, dan jumlah nilainya sama dengan jumlah kolom Reserve Claim ASM Share.
 *
 * Konsistensi itu yang diuji — panel yang angkanya tidak cocok dengan grid di bawahnya
 * adalah kelas cacat yang tidak menghasilkan galat apa pun.
 */
function summaryResponse(): SummaryResponse {
  return {
    posisi: '2026-09-28',
    total_berkas: 3,
    total_estimasi: '2000000.00',
    total_reserve_or: '0.00',
    total_reserve_or_terbaca: true,
    umur_di_atas_2_tahun: 1,
    sebaran_umur: [
      { label: 'Sampai 6 bulan', berkas: 2, nilai: '1750000.00' },
      { label: '6–12 bulan', berkas: 0, nilai: '0.00' },
      { label: '1–2 tahun', berkas: 0, nilai: '0.00' },
      { label: 'Di atas 2 tahun', berkas: 1, nilai: '250000.00' },
    ],
    per_cob: [
      { nama: 'PA', berkas: 1, nilai: '1500000.00', umur_di_atas_2_tahun: 0 },
      { nama: 'Aneka', berkas: 1, nilai: '250000.00', umur_di_atas_2_tahun: 1 },
    ],
    per_sumber_bisnis: [
      { nama: 'BANK CONTOH CILEGON', berkas: 1, nilai: '250000.00', umur_di_atas_2_tahun: 1 },
      { nama: '(tanpa keterangan)', berkas: 1, nilai: '0.00', umur_di_atas_2_tahun: 0 },
    ],
    cabang: { kode: '100099', nama: 'CILEGON' },
    portal: 'ASM',
  }
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/** stubServer menjawab daftar, lalu popup menurut nomor klaimnya. */
function stubServer(details: Record<string, Response | (() => Response)> = {}) {
  vi.stubGlobal('fetch', (url: string) => {
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    if (url.startsWith(`${PATH}/ringkasan`))
      return Promise.resolve(jsonResponse(200, summaryResponse()))

    for (const [nomor, answer] of Object.entries(details)) {
      if (url === `${PATH}/${nomor}`) {
        return Promise.resolve(typeof answer === 'function' ? answer() : answer)
      }
    }

    return Promise.resolve(jsonResponse(200, listResponse([ANEKA, PA])))
  })
}

async function openDetail(nomor: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-os-claim-per-cabang']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )

  await screen.findByText(nomor)
  await userEvent.click(
    screen.getByRole('button', { name: `Lihat detail klaim ${nomor}` }),
  )
  return screen.findByRole('dialog')
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
})

it('menggambar keempat bagian popup dengan label layar lama', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  const box = within(dialog)

  // Label ditulis PERSIS seperti di Pega, termasuk spasi sebelum titik dua.
  await box.findByText('Occupation :')
  box.getByText('Total Sum Insured :')
  box.getByText('Total Reserve :')
  box.getByText('Kronologi :')
  box.getByText('Dominant Factor :')
  box.getByText('Claim Recommendation :')
  box.getByText('Note dari PIC :')
  box.getByText('Aging :')

  box.getByText('Objek Pertanggungan')
  box.getByText('Riwayat Progress')
  box.getByText('KOMUNIKASI DENGAN LOSS ADJUSTER')
})

it('menampilkan nilai uang dengan format rupiah, bukan teks mentah', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  const box = within(dialog)

  await box.findByText('Rp 5.000.000')
  box.getByText('Rp 250.000')

  // Teks desimal kanonik TIDAK boleh bocor ke layar.
  expect(dialog.textContent).not.toContain('5000000.00')
})

it('memakai kolom objek bervarian lokasi untuk lini selain PA dan Travel', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  const box = within(dialog)

  // DataTable menggambar judul kolom DUA KALI — sekali sebagai kepala tabel lebar, dan
  // sekali sebagai label kartu pada lebar ponsel. Keduanya sah, sehingga yang diperiksa
  // keberadaannya, bukan jumlahnya.
  await box.findAllByText('Nama Objek')
  expect(box.getAllByText('Lokasi Object').length).toBeGreaterThan(0)
  expect(box.queryAllByText('KTP/Paspor')).toHaveLength(0)
  expect(box.queryAllByText('Nama Peserta')).toHaveLength(0)
})

it('memakai kolom objek bervarian pekerjaan untuk lini PA', async () => {
  // `IsPA` berpasangan dengan susunan berkolom "Pekerjaan" — BUKAN dengan susunan peserta.
  // Keduanya sempat tertukar, dan tertukarnya tidak menghasilkan galat apa pun: grid tetap
  // terisi, hanya kolomnya yang milik lini lain.
  stubServer({ 'PNC-9002': jsonResponse(200, DETAIL_PA) })

  const dialog = await openDetail('PNC-9002')
  const box = within(dialog)

  await box.findAllByText('Nama Objek')
  expect(box.getAllByText('Pekerjaan').length).toBeGreaterThan(0)
  expect(box.getAllByText('Tanggal lahir').length).toBeGreaterThan(0)

  // Kolom milik varian Travel TIDAK boleh muncul di PA.
  expect(box.queryAllByText('Nama Peserta')).toHaveLength(0)
  expect(box.queryAllByText('KTP/Paspor')).toHaveLength(0)
  expect(box.queryAllByText('Lokasi Object')).toHaveLength(0)
})

it('memakai kolom objek bervarian peserta untuk lini Travel', async () => {
  // Varian ketiga. Tanpa uji ini, dua dari tiga varian dapat runtuh menjadi satu tanpa
  // satu pun tanda — persis yang pernah terjadi.
  stubServer({ 'PNC-9002': jsonResponse(200, DETAIL_TRAVEL) })

  const dialog = await openDetail('PNC-9002')
  const box = within(dialog)

  await box.findAllByText('Nama Peserta')
  expect(box.getAllByText('Status').length).toBeGreaterThan(0)
  expect(box.getAllByText('KTP/Paspor').length).toBeGreaterThan(0)
  expect(box.getAllByText('Tanggal Lahir').length).toBeGreaterThan(0)

  expect(box.queryAllByText('Nama Objek')).toHaveLength(0)
  expect(box.queryAllByText('Lokasi Object')).toHaveLength(0)
})

it('menandai pesan dari petugas internal', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  await within(dialog).findByText('internal')
})

it('menampilkan pesan kosong yang menjelaskan dirinya saat tidak ada komunikasi', async () => {
  // Grid kosong harus terbaca sebagai "belum ada komunikasi", bukan sebagai gagal memuat.
  stubServer({ 'PNC-9002': jsonResponse(200, DETAIL_PA) })

  const dialog = await openDetail('PNC-9002')
  await within(dialog).findByText('Belum ada komunikasi dengan loss adjuster.')
})

it('hanya mengirim NOMOR klaim, tidak ada nilai uang maupun umur', async () => {
  // Mengirim nilai cadangan dari peramban berarti angka uang di popup ditentukan pengirim
  // permintaan, dan itu dapat diubah lewat alat pengembang biasa.
  const seen: string[] = []
  vi.stubGlobal('fetch', (url: string) => {
    seen.push(url)
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    if (url.startsWith(`${PATH}/ringkasan`))
      return Promise.resolve(jsonResponse(200, summaryResponse()))
    if (url === `${PATH}/PNC-9001`) return Promise.resolve(jsonResponse(200, DETAIL_ANEKA))
    return Promise.resolve(jsonResponse(200, listResponse([ANEKA, PA])))
  })

  await openDetail('PNC-9001')

  const detailCall = seen.find((url) => url.startsWith(`${PATH}/PNC-9001`))
  expect(detailCall).toBe(`${PATH}/PNC-9001`)
  expect(detailCall).not.toContain('reserve')
  expect(detailCall).not.toContain('aging')
})

it('menjelaskan klaim yang tidak ditemukan sebagai penolakan, bukan gangguan', async () => {
  stubServer({
    'PNC-9001': jsonResponse(404, {
      kode: 'klaim_tidak_ditemukan',
      pesan: 'Klaim tidak ditemukan pada cabang Anda. Muat ulang daftar, lalu buka kembali dari barisnya.',
    }),
  })

  const dialog = await openDetail('PNC-9001')
  const box = within(dialog)

  await box.findByText('Klaim tidak ditemukan')
  box.getByText(/Muat ulang daftar/)
})

it('menutup popup lewat tombol Tutup', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  await within(dialog).findByText('Occupation :')

  await userEvent.click(within(dialog).getByRole('button', { name: 'Tutup' }))
  expect(screen.queryByRole('dialog')).toBeNull()
})

it('menutup popup lewat tombol Escape', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  await within(dialog).findByText('Occupation :')

  await userEvent.keyboard('{Escape}')
  expect(screen.queryByRole('dialog')).toBeNull()
})

it('menyatakan selisih terencana milik POPUP, bukan milik daftar', async () => {
  stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

  const dialog = await openDetail('PNC-9001')
  const box = within(dialog)

  const catatan = await box.findByText('Perbedaan yang disengaja terhadap layar lama')

  // Dicari DI DALAM blok catatannya, bukan di seluruh dialog: "Total Sum Insured" juga
  // muncul sebagai label ringkasan di atas, dan pencarian global akan cocok dengan yang itu
  // meski daftar selisihnya kosong.
  const blok = catatan.closest('details')
  expect(blok).not.toBeNull()
  expect(blok?.textContent).toContain('TSI objek terakhir')

  // Selisih milik DAFTAR tidak boleh ikut tampil di popup.
  expect(dialog.textContent).not.toContain('Daftar dibagi per halaman di server.')
})

describe('baris objek dapat dibuka', () => {
  // Layar lama MEMANG punya kemampuan ini, dan pembacaan pertama kami keliru menyimpulkan
  // sebaliknya. Mekanismenya bukan `pyExpandable` melainkan master-detail:
  //
  //   Section/DetailKlaimCabang_Sect-Section.xml
  //     pyRowEditing = masterDetail    (:5616, :7193, :9798)
  //     pyEditAction = ViewObjectItem  (:5625, :7179, :9815)
  //
  // Karena kekeliruan itu sempat dinyatakan sebagai fakta, perilakunya dijaga uji — bukan
  // hanya diperbaiki sekali lalu dipercaya bertahan.

  it('membuka coverage milik objek yang diklik', async () => {
    stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

    const dialog = await openDetail('PNC-9001')
    const box = within(dialog)

    // Sebelum dibuka, isi coverage belum tergambar sama sekali.
    expect(box.queryByText('FLEXAS')).toBeNull()

    await userEvent.click(box.getByText('GUDANG A'))

    expect(await box.findByText('FLEXAS')).toBeInTheDocument()

    // "IDR" muncul di DUA tingkat — coverage dan estimasi — sehingga pencarian global
    // menjadi ambigu. Yang diperiksa barisnya sendiri.
    const barisCoverage = box.getByText('FLEXAS').closest('tr') as HTMLElement
    expect(barisCoverage).not.toBeNull()
    expect(within(barisCoverage).getByText('IDR')).toBeInTheDocument()
    // TANPA 'Rp' — layar lama menggambar angkanya polos, dan mata uangnya kolom tersendiri.
    expect(within(barisCoverage).getByText('3.000.000')).toBeInTheDocument()
    expect(box.queryByText('Rp 3.000.000')).toBeNull()
  })

  it('membuka Object Item dan Estimasi hanya setelah COVERAGE diklik', async () => {
    // Baris coverage dapat dibuka, sama seperti baris objek di atasnya — `ViewObjectCoverage`
    // pun `pyRowEditing = masterDetail` dengan aksi `ViewObjectCoverageObjectItem`.
    //
    // Serahan sebelumnya menggambar seluruh isinya sekaligus begitu objek dibuka. Pada klaim
    // bercoverage banyak itu menumpahkan semuanya tanpa diminta, dan Work Owner menangkapnya.
    stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

    const dialog = await openDetail('PNC-9001')
    const box = within(dialog)
    await userEvent.click(box.getByText('GUDANG A'))

    // Coverage sudah tampak, tetapi ISINYA belum.
    expect(await box.findByText('FLEXAS')).toBeInTheDocument()
    expect(box.queryByText('BUILDINGS')).toBeNull()
    expect(box.queryByText('Estimasi Ke')).toBeNull()

    await userEvent.click(box.getByText('FLEXAS'))

    expect(await box.findByText('BUILDINGS')).toBeInTheDocument()
    expect(box.getByText('Bangunan gudang')).toBeInTheDocument()

    for (const judul of [
      'Object Item',
      'Deskripsi Item',
      'Estimasi Ke',
      'Tanggal Estimasi',
      'Tipe Estimasi',
      'Nilai Kurs (IDR)',
      'Nilai Estimasi',
    ]) {
      expect(box.getByText(judul)).toBeInTheDocument()
    }

    // Nilai NEGATIF digambar apa adanya. Contoh Fire dari layar lama memuat pasangan yang
    // saling meniadakan; menyaringnya akan membuat jumlahnya tidak pernah cocok.
    expect(box.getByText('-500.000')).toBeInTheDocument()

    // Tanggal estimasi WAJIB tergambar, tidak boleh "—". Serahan sebelumnya mengambilnya
    // dari `ESTIMATIONDATE`, yang terisi pada 32 dari 51.535 baris saja — sehingga kolomnya
    // kosong di layar untuk hampir setiap klaim. Sumbernya kini `INSERTDATE`.
    //
    // Dicari DI DALAM barisnya: tanggal lain bertahun sama ada di riwayat progres dan di
    // komunikasi adjuster, sehingga pencarian global cocok dengan baris yang salah.
    const barisEstimasi = box.getByText('BUILDINGS').closest('table') as HTMLElement
    expect(within(barisEstimasi).getByText('2024-01-20 09:00')).toBeInTheDocument()
  })

  it('menggambar List Spreading dan CO MEMBER, termasuk yang kosong', async () => {
    // Tabel yang kosong TETAP digambar dengan "Data Tidak Ada" — begitulah layar lama
    // menyatakannya, dan kedua contoh dari Work Owner memperlihatkannya persis demikian.
    // Menghilangkan tabelnya akan membuat pengguna mengira bagian itu tidak ada.
    stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

    const dialog = await openDetail('PNC-9001')
    const box = within(dialog)
    await userEvent.click(box.getByText('GUDANG A'))
    await userEvent.click(await box.findByText('FLEXAS'))

    expect(await box.findByText('List Spreading')).toBeInTheDocument()
    expect(box.getByText('CO MEMBER')).toBeInTheDocument()

    expect(box.getByText('FAC-OUT')).toBeInTheDocument()
    // Persentase berdesimal EMPAT, mengikuti layar lama.
    expect(box.getByText('100.0000%')).toBeInTheDocument()

    expect(box.getByText('Data Tidak Ada')).toBeInTheDocument()
    expect(box.queryByText('3000000.00')).toBeNull()
  })

  it('TIDAK dapat dibuka pada objek yang tidak punya coverage', async () => {
    // Baris yang terbuka menjadi panel kosong lebih buruk daripada baris yang tidak dapat
    // dibuka: yang pertama membuat pengguna mengira datanya hilang.
    stubServer({ 'PNC-9001': jsonResponse(200, DETAIL_ANEKA) })

    const dialog = await openDetail('PNC-9001')
    const box = within(dialog)

    const barisKosong = box.getByText('GUDANG B').closest('tr')
    expect(barisKosong).not.toBeNull()
    expect(barisKosong).not.toHaveAttribute('aria-expanded')

    const barisBerisi = box.getByText('GUDANG A').closest('tr')
    expect(barisBerisi).toHaveAttribute('aria-expanded', 'false')
  })
})
