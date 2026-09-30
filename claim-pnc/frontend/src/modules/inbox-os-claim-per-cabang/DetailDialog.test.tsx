import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, ListResponse, WorkItem } from './types'

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
      nama: 'GUDANG A',
      lokasi: 'JL CONTOH NO 1',
      pekerjaan: '',
      tanggal_lahir: '',
      ktp_paspor: '',
      status_peserta: '',
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
      nama: 'BUDI CONTOH',
      lokasi: '',
      pekerjaan: 'TEKNISI',
      tanggal_lahir: '1990-03-17',
      ktp_paspor: '3200000000000001',
      status_peserta: 'KARYAWAN',
    },
  ],
  komunikasi_adjuster: [],
}

/** Lini Travel — varian kolom ketiga: Nama Peserta, Status, KTP/Paspor, Tanggal Lahir. */
const DETAIL_TRAVEL: DetailResponse = {
  ...DETAIL_PA,
  ringkasan: { ...DETAIL_PA.ringkasan, cob: 'Travel' },
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
