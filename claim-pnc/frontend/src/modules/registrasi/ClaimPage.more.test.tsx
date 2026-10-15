import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimPage } from './ClaimPage'
import { CustomerPrinciple } from './types'

/**
 * Uji tambahan layar klaim: keadaan klaim tanpa tugas, tahap tanpa formulir, cabang galat,
 * dan isian Input Register yang tidak tersentuh uji utama. Seluruh data KARANGAN (`D-69`).
 */

const KLAIM = {
  id: 'klaim-1',
  nomor: '',
  portal: 'ASM',
  polis: {
    nomor: 'POL-FIRE-0001',
    lini: '006',
    nama_lini: 'Fire',
    jenis_bisnis: 'Fire',
    mulai_pertanggungan: '2026-01-01',
    akhir_pertanggungan: '2026-12-31',
    mata_uang: 'USD',
    nama_tertanggung: '',
    deklarasi: false,
    penjamin_kredit: false,
  },
  tanggal_kejadian: '2026-06-05',
  tanggal_lapor: '2026-06-06',
  tanggal_terima_dokumen: '2026-06-07',
  lokasi: 'Gudang A',
  kronologi: '',
  pelapor: { nama: '', telepon: '', email: '', alamat: '', hubungan: 0, hubungan_lainnya: '' },
  prinsip_mengenal_nasabah: '',
  nilai_estimasi_sen: 0,
  mata_uang: '',
  nomor_slik: '',
  ex_gratia: false,
  user_teknis: '',
  rcv_id: '',
  objek: [],
  status_pucl: 0,
  transfer_compliance: false,
  status_proses: 'BERJALAN',
  status_kerja_pega: '',
  status_klaim: '',
  flag_klaim: '0',
  status_posisi_progres: 'On Progress',
  tahap_kini: 'input-register',
}

const TUGAS = {
  id: 'tugas-1',
  klaim_id: 'klaim-1',
  nomor_klaim: '',
  tahap: 'input-register',
  nama_tahap: 'Input Register',
  antrean: 'WORKLIST',
  workbasket: '',
  pemilik: '90000001',
  dapat_diambil: false,
  tindakan_keluar: 'InputRegister',
  dibuat_pada: '2026-06-10T03:00:00Z',
}

type Answer = Response | Promise<Response> | 'putus' | 'aneh' | undefined

let calls: { url: string; method: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function response(klaim: Record<string, unknown> = {}, tugas: Record<string, unknown> | null = {}) {
  return {
    klaim: { ...KLAIM, ...klaim },
    tugas: tugas === null ? null : { ...TUGAS, ...tugas },
    jalur: [],
    jejak_keputusan: null,
    large_loss: false,
  }
}

function installFetch(claim: unknown, answer: (url: string, method: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method, body: init?.body ? JSON.parse(init.body as string) : undefined })
    const custom = answer(url, method)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'aneh') {
      return Promise.resolve({
        get status(): number {
          throw 'bukan galat'
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url === '/api/registrasi/alur') return Promise.resolve(json(200, { nama: 'Register', mulai: '', tahap: [] }))
    if (url.startsWith('/api/registrasi/wilayah/')) {
      const level = url.split('?')[0]!.split('/').pop()
      const pilihan =
        level === 'negara'
          ? [
              { id: '1', nama: 'INDONESIA' },
              { id: '2', nama: 'SINGAPURA' },
            ]
          : []
      return Promise.resolve(json(200, { pilihan }))
    }
    if (url === '/api/registrasi/klaim/klaim-1') return Promise.resolve(json(200, claim))
    return Promise.resolve(json(200, claim))
  })
}

function show(alamat = '/registrasi/klaim/klaim-1') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[alamat]}>
        <Routes>
          <Route path="/registrasi/klaim/:claimID" element={<ClaimPage />} />
          <Route path="/registrasi" element={<p>Halaman inbox</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: { identitas: '90000001', nama: 'Contoh', jenis: 'KARYAWAN', login: 'x', email: '', perusahaan: 'ASM' },
    validUntil: '2026-12-31T00:00:00Z',
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('keadaan klaim', () => {
  it.each([
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: 'Terjadi kesalahan pada sistem.' },
  ])('menampilkan galat muat klaim: $name', async ({ answer, text }) => {
    installFetch(response(), (url) => (url === '/api/registrasi/klaim/klaim-1' ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat klaim…')).toBeInTheDocument()
    expect(await screen.findByText('Klaim tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('menyebut klaim yang sudah selesai pada alur Register', async () => {
    // `status_proses` SELESAI — tanpa itu klaimnya bukan selesai, melainkan milik Pega.
    installFetch(response({ tahap_kini: '', status_proses: 'SELESAI' }, null))
    show()

    expect(await screen.findByText(/Klaim ini sudah selesai pada alur Register\./)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Klaim belum bernomor' })).toBeInTheDocument()
    expect(screen.queryByText(/tidak ada tugas terbuka/)).not.toBeInTheDocument()
  })

  /**
   * Tahap kosong punya DUA sebab, dan keduanya tidak boleh disebut "selesai".
   *
   * `tahap_kini` diturunkan dari tugas terbuka di `CPNC_TUGAS` — tabel milik aplikasi ini, yang
   * hanya terisi bila klaimnya didaftarkan DI SINI. Diukur 2026-10-12: dari 368 barisnya, 359
   * milik klaim `PNCN.` dan hanya 2 milik klaim warisan.
   *
   * Klaim warisan karena itu SELALU bertahap kosong. Menyebutnya "sudah selesai" adalah
   * pernyataan yang salah, dan `PNC-11` membuktikannya: `STATUSWORK` = `New`.
   */
  it('membedakan klaim milik Pega dari klaim yang sudah selesai', async () => {
    // Status Pega apa adanya — di luar BERJALAN/SELESAI/DITOLAK milik aplikasi ini.
    installFetch(response({ tahap_kini: '', status_proses: 'New' }, null))
    show()

    expect(await screen.findByText(/dikerjakan di Pega, bukan di aplikasi ini/)).toBeInTheDocument()
    expect(screen.queryByText(/sudah selesai pada alur Register/)).not.toBeInTheDocument()
  })

  /**
   * Status PEGA menang atas status tabel bisnis.
   *
   * `PNC-6` di DEV: `T_CLAIM_PNC.STATUSWORK` = `Open`, tetapi
   * `T_CLAIMLIST_ADMIN.PYSTATUSWORK` = `Resolved-Rejected`. Keduanya tidak sepakat pada 157
   * dari 1.360 klaim warisan (diukur 2026-10-12).
   *
   * Sebelum perbaikan ini, halaman membaca `Open` lalu menulis "dikerjakan di Pega" pada klaim
   * yang sebenarnya SUDAH SELESAI. Dilaporkan Work Owner pada klaim Close Claim.
   */
  it('memakai status Pega saat ia berbeda dari status tabel bisnis', async () => {
    installFetch(
      response({ tahap_kini: '', status_proses: 'Open', status_kerja_pega: 'Resolved-Rejected' }, null),
    )
    show()

    expect(await screen.findByText(/sudah selesai pada alur Register/)).toBeInTheDocument()
    expect(screen.queryByText(/dikerjakan di Pega, bukan di aplikasi ini/)).not.toBeInTheDocument()
  })

  /**
   * Tanpa status Pega, pemeriksaan jatuh ke status aplikasi ini — persis seperti sebelumnya.
   *
   * Kosong berarti klaimnya tidak punya baris di tabel daftar Pega, BUKAN berarti selesai.
   * Menganggapnya selesai akan menyatakan tuntas klaim yang belum tentu tuntas.
   */
  it('tidak menganggap selesai saat status Pega kosong', async () => {
    installFetch(response({ tahap_kini: '', status_proses: 'New', status_kerja_pega: '' }, null))
    show()

    expect(await screen.findByText(/dikerjakan di Pega, bukan di aplikasi ini/)).toBeInTheDocument()
  })

  /**
   * Klaim tanpa tugas memakai BINGKAI YANG SAMA, dibuka baca-saja.
   *
   * Seluruh isi halaman ini digambar oleh bingkai tahap, dan bingkai itu menuntut sebuah
   * tugas. Akibatnya klaim yang alurnya sudah selesai — dan seluruh klaim warisan yang alurnya
   * masih di Pega (`P-3`) — hanya menampilkan kepala dan satu kalimat. Dilaporkan Work Owner
   * 2026-10-12 pada `PNC-11`.
   *
   * Yang dijaga di sini: bingkainya **yang sama**, bukan bagian buatan sendiri yang menyerupai.
   * Bagian buatan sendiri sudah dicoba dan ditolak, dan uji ini yang mencegahnya kembali.
   */
  it('memakai bingkai yang sama, dibuka baca-saja, saat tidak ada tugas terbuka', async () => {
    installFetch(
      response(
        {
          tahap_kini: '',
          objek: [
            {
              id: 'objek-1',
              nama: 'Bangunan Gudang',
              lokasi: 'Cakung',
              coverage: [
                {
                  id: 'cov-1',
                  nama: 'RESIKO A',
                  penyebab_kerugian: 'ACCIDENT',
                  tsi_sen: 1500000000,
                  spreading: [],
                },
              ],
            },
          ],
        },
        null,
      ),
    )
    show()

    // Bingkainya bernama "Rincian Klaim" — BUKAN "InputSurveyor", yang akan menyatakan klaim
    // berada di tahap yang sudah dilewatinya.
    expect(await screen.findByRole('heading', { name: 'Rincian Klaim' })).toBeInTheDocument()

    // Deret tab yang sama dengan klaim yang punya tugas.
    for (const judul of ['Input Register', 'Estimasi & Adjustment', 'Unggah Dokumen']) {
      expect(screen.getByRole('tab', { name: judul })).toBeInTheDocument()
    }

    // Isinya tergambar — objek dan jaminannya, bukan layar kosong.
    expect(screen.getByText('Bangunan Gudang')).toBeInTheDocument()
    expect(screen.getByText('RESIKO A')).toBeInTheDocument()

    // Terkunci: sebabnya dinyatakan SEKALI, oleh kotak di tingkat halaman.
    //
    // Bingkai tahap sempat memasang peringatannya sendiri di bawah kotak itu, dan keduanya
    // mengatakan hal yang sama dengan kalimat berbeda. Dicabut 2026-10-12 setelah Work Owner
    // melaporkannya pada klaim Close Claim.
    expect(screen.getByText(/dikerjakan di Pega|sudah selesai pada alur Register/)).toBeInTheDocument()
    expect(screen.queryByText(/Tombol dan isiannya dinonaktifkan/)).not.toBeInTheDocument()
  })

  it('menyebut tahap yang tugasnya bukan milik pengguna', async () => {
    installFetch(response({ tahap_kini: 'investigator', nomor: 'PNCN.26.0005' }, null))
    show()

    expect(await screen.findByText(/Klaim berada di tahap investigator/)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'PNCN.26.0005' })).toBeInTheDocument()
  })
})

describe('tahap tanpa formulir', () => {
  // Investigator kini memakai formulir Surveyor (InputInvestigator); Compliance belum punya formulir.
  const AT_COMPLIANCE = response(
    { tahap_kini: 'compliance' },
    { tahap: 'compliance', nama_tahap: 'Compliance', tindakan_keluar: 'InputCompliance' },
  )

  it('menutup tahap lalu menampilkan percabangan yang dilewati', async () => {
    installFetch(AT_COMPLIANCE, (url) =>
      url === '/api/registrasi/tugas/tugas-1/selesai'
        ? json(200, { ...AT_COMPLIANCE, jejak_keputusan: ['Bukan PA', 'Bukan Travel'] })
        : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Selesaikan (InputCompliance)' }))
    expect(await screen.findByText('Percabangan yang dilewati')).toBeInTheDocument()
    expect(screen.getByText('Bukan Travel')).toBeInTheDocument()
    expect(calls.find((c) => c.url.endsWith('/selesai'))?.body).toEqual({ action: 'InputCompliance', kembali: false })
  })

  it('mengirim Back dan menampilkan galat penutupan tahap', async () => {
    installFetch(AT_COMPLIANCE, (url) =>
      url === '/api/registrasi/tugas/tugas-1/selesai'
        ? json(409, { kode: 'tugas_sudah_selesai', pesan: 'Tugas sudah selesai.' })
        : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Kembali (Back)' }))
    expect(await screen.findByText('Tahap tidak dapat ditutup')).toBeInTheDocument()
    expect(screen.getByText('Tugas sudah selesai.')).toBeInTheDocument()
    expect(calls.find((c) => c.url.endsWith('/selesai'))?.body).toMatchObject({ kembali: true })
  })

  it('tidak menampilkan percabangan bila server tidak mengirimnya', async () => {
    installFetch(AT_COMPLIANCE, (url) =>
      url === '/api/registrasi/tugas/tugas-1/selesai' ? json(200, { ...AT_COMPLIANCE, jejak_keputusan: [] }) : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Selesaikan (InputCompliance)' }))
    await waitFor(() => expect(calls.some((c) => c.url.endsWith('/selesai'))).toBe(true))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Selesaikan (InputCompliance)' })).toBeEnabled())
    expect(screen.queryByText('Percabangan yang dilewati')).not.toBeInTheDocument()
  })
})

describe('Input Register', () => {
  it('memakai mata uang polis dan nilai bawaan bila klaim belum mengisinya', async () => {
    installFetch(response(), (url) =>
      url === '/api/registrasi/register/simpan' ? json(500, { kode: 'x', pesan: 'Draf gagal.' }) : undefined,
    )
    show()

    expect(await screen.findByLabelText('Status Pelapor')).toHaveValue('')
    expect(screen.getByLabelText('Status RCL/PUCL')).toHaveValue('0')
    // Bagian Estimasi tidak tampil di Input Register (Work Owner 2026-10-08), tetapi nilai
    // bawaannya — mata uang polis dan Prinsip Mengenal Nasabah NORMAL — tetap dikirim.
    expect(screen.queryByLabelText('Mata Uang')).not.toBeInTheDocument()
    expect(screen.queryByRole('radio', { name: 'NORMAL' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Isian belum tersimpan')).toBeInTheDocument()
    expect(screen.getByText('Draf gagal.')).toBeInTheDocument()
    expect(calls.find((c) => c.url === '/api/registrasi/register/simpan')?.body).toMatchObject({
      mata_uang: 'USD',
      prinsip_mengenal_nasabah: CustomerPrinciple.Normal,
    })
  })

  it('meminta keterangan hubungan lain-lain, lalu mengirimnya', async () => {
    installFetch(response(), (url) =>
      url === '/api/registrasi/register' ? json(200, { ...response(), large_loss: true }) : undefined,
    )
    show()

    const hubungan = await screen.findByLabelText('Status Pelapor')
    expect(screen.queryByLabelText('Sebutkan...')).not.toBeInTheDocument()
    expect(within(hubungan).getAllByRole('option').map((o) => o.textContent)).toEqual([
      '— pilih —', 'Tertanggung', 'Suami/Istri', 'Anak', 'Orang Tua', 'Famili', 'Teman', 'Lainnya',
    ])
    await userEvent.selectOptions(hubungan, '7')
    await userEvent.type(screen.getByLabelText('Sebutkan...'), 'Kerabat')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Transfer Compliance' }))
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByText('Notice of Large Losses diterbitkan.')).toBeInTheDocument()
    expect(calls.find((c) => c.url === '/api/registrasi/register')?.body).toMatchObject({
      pelapor: { hubungan: 7, hubungan_lainnya: 'Kerabat' },
      transfer_compliance: true,
      kembali: false,
      mata_uang: 'USD',
    })
  })

  // SetTertanggungRegister: Tertanggung mengisi Nama/Telepon/Alamat Pelapor dari polis dan CIF;
  // status lain mengosongkannya.
  it('Status Pelapor Tertanggung mengisi data pelapor dari tertanggung, status lain mengosongkannya', async () => {
    installFetch(response({ polis: { ...response().klaim.polis, nama_tertanggung: 'PT TERTANGGUNG CONTOH' } }), (url) =>
      url.endsWith('/tertanggung')
        ? json(200, {
            no_ktp: '',
            alamat: [{ alamat: 'Jl. Contoh 1', telepon: [{ jenis: '1', nama_jenis: 'HP', kode: '', nomor: '0811111', ekstensi: '' }] }],
          })
        : undefined,
    )
    show()
    const hubungan = await screen.findByLabelText('Status Pelapor')
    await waitFor(() => expect(calls.some((c) => c.url.endsWith('/tertanggung'))).toBe(true))
    await userEvent.selectOptions(hubungan, '1')
    expect(screen.getByLabelText('Nama Pelapor')).toHaveValue('PT TERTANGGUNG CONTOH')
    expect(screen.getByLabelText('No. Telepon Pelapor')).toHaveValue('0811111')
    expect(screen.getByLabelText('Alamat Pelapor')).toHaveValue('Jl. Contoh 1')

    await userEvent.selectOptions(hubungan, '3')
    expect(screen.getByLabelText('Nama Pelapor')).toHaveValue('')
    expect(screen.getByLabelText('No. Telepon Pelapor')).toHaveValue('')
    expect(screen.getByLabelText('Alamat Pelapor')).toHaveValue('')
  })

  it('Next ditahan bila Lokasi Kerugian/Kejadian kosong; Save tetap menyimpan', async () => {
    installFetch(response({ lokasi: '' }))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Lokasi Kerugian/Kejadian is required.')).toBeInTheDocument()
    expect(calls.some((c) => c.url === '/api/registrasi/register')).toBe(false)
    expect(screen.getByLabelText('Lokasi Kerugian/Kejadian *')).toHaveFocus()

    // Spasi saja tetap dianggap kosong.
    await userEvent.type(screen.getByLabelText('Lokasi Kerugian/Kejadian *'), '   ')
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(calls.some((c) => c.url === '/api/registrasi/register')).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(calls.some((c) => c.url === '/api/registrasi/register/simpan')).toBe(true))
  })

  it('menampilkan galat simpan tanpa pelanggaran dan daftar banyak pelanggaran', async () => {
    let answer: () => Answer = () => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' })
    installFetch(response(), (url) => (url === '/api/registrasi/register' ? answer() : undefined))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Penyimpanan gagal')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()

    answer = () =>
      json(422, {
        kode: 'validasi_gagal',
        pesan: 'Klaim belum lengkap.',
        detail: [
          { kode: 'a', field: 'nomor_polis', pesan: 'Polis deklarasi tidak dapat diklaim.' },
          { kode: 'b', field: 'objek', pesan: 'Objek wajib diisi.' },
          { kode: 'c', field: 'penyebab_kerugian', pesan: 'Penyebab kerugian wajib.' },
          { kode: 'd', field: 'spreading', pesan: 'Total spreading harus 100%.' },
        ],
      })
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Klaim belum dapat disimpan')).toBeInTheDocument()
    expect(screen.getByText('4 ketentuan belum terpenuhi:')).toBeInTheDocument()
    expect(screen.getAllByText('Polis deklarasi tidak dapat diklaim.')).toHaveLength(2)
    expect(screen.getAllByText('Total spreading harus 100%.')).toHaveLength(2)
  })

  // Grid objek Section/InputRegisterDetail-sect.xml. Kolomnya berbeda menurut lini, dan
  // kedua uji di bawah menjaga perbedaannya tetap ada — menyeragamkannya akan membuat
  // lini PA kehilangan Pekerjaan dan Tanggal Lahir tanpa ada yang menyadarinya.

  // Empat grid objek Section/InputRegisterDetail-sect.xml, dipilih menurut lini polis.
  // Kolom polis read-only seperti di XML; "Type Object" (tampil 1==2) tidak digambar.
  const headerOf = async () => {
    const grid = await screen.findByRole('table', { name: 'Objek pertanggungan beserta jaminannya' })
    return { grid, header: within(grid).getAllByRole('columnheader').map((h) => h.textContent) }
  }

  it('grid objek lini PA berkolom Nama, Pekerjaan, dan Tanggal Lahir', async () => {
    installFetch(
      response({
        polis: { ...KLAIM.polis, lini: '002', nama_lini: 'Personal Accident', jenis_bisnis: 'PA' },
        objek: [
          {
            id: 'OBJ-1',
            nama: 'FRANSISCO AMADEUS J',
            lokasi: '',
            pekerjaan: 'KARYAWAN',
            tanggal_lahir: '1996-02-03',
            coverage: [],
          },
        ],
      }),
    )
    show()

    const { grid, header } = await headerOf()
    expect(header).toEqual(['', '#', 'Nama', 'Pekerjaan', 'Tanggal Lahir', 'Aksi'])
    // Seluruhnya milik polis dan digambar sebagai teks, bukan isian.
    expect(within(grid).getByText('FRANSISCO AMADEUS J')).toBeInTheDocument()
    expect(within(grid).getByText('KARYAWAN')).toBeInTheDocument()
    expect(within(grid).getByText('03/02/1996')).toBeInTheDocument()
    expect(within(grid).queryByRole('textbox')).toBeNull()
  })

  it('grid objek lini Travel berkolom Nama Peserta, Status, KTP/Paspor, dan Tanggal Lahir', async () => {
    installFetch(
      response({
        polis: { ...KLAIM.polis, lini: '005', nama_lini: 'Travel', jenis_bisnis: 'Travel' },
        objek: [
          { id: '1', nama: 'PESERTA SATU', lokasi: '', ktp_paspor: 'X123', status_peserta: 'Tertanggung',
            tanggal_lahir: '1990-01-02', coverage: [] },
        ],
      }),
    )
    show()

    const { grid, header } = await headerOf()
    expect(header).toEqual(['', '#', 'Nama Peserta', 'Status', 'KTP/Paspor', 'Tanggal Lahir', 'Aksi'])
    expect(within(grid).getByText('X123')).toBeInTheDocument()
    expect(within(grid).getByText('Tertanggung')).toBeInTheDocument()
    expect(within(grid).getByText('02/01/1990')).toBeInTheDocument()
  })

  it('grid objek lini HE berkolom Object, Model, Merk, Nama Tipe, Nomor Chasis, dan Location', async () => {
    installFetch(
      response({
        polis: { ...KLAIM.polis, lini: '003', jenis_bisnis: 'HE' },
        objek: [
          { id: '2', nama: 'DUMP TRUCK', lokasi: 'MOROWALI', model: 'DUMP TRUCK', merk: 'SANY',
            nama_tipe: 'SYZ324C8W', nomor_chasis: 'CH-9', coverage: [] },
        ],
      }),
    )
    show()

    const { grid, header } = await headerOf()
    expect(header).toEqual(['', '#', 'Object', 'Model', 'Merk', 'Nama Tipe', 'Nomor Chasis', 'Location', 'Aksi'])
    for (const value of ['SANY', 'SYZ324C8W', 'CH-9', 'MOROWALI']) {
      expect(within(grid).getByText(value)).toBeInTheDocument()
    }
    expect(within(grid).queryByRole('textbox')).toBeNull()
    expect(within(grid).queryByText('Type Object')).toBeNull()
  })

  // When IsHE = BusinessType "HE" OR "ContractorsPM" (When/IsHE-When.xml).
  it('grid objek HE juga dipakai lini ContractorsPM', async () => {
    installFetch(
      response({
        polis: { ...KLAIM.polis, lini: '003', jenis_bisnis: 'ContractorsPM' },
        objek: [
          { id: '1', nama: 'EXCAVATOR', lokasi: 'INDONESIA', model: 'EXCAVATOR', merk: 'HITACHI',
            nama_tipe: 'ZX210F-5G', nomor_chasis: 'CH-1', coverage: [] },
        ],
      }),
    )
    show()

    const { grid, header } = await headerOf()
    expect(header).toEqual(['', '#', 'Object', 'Model', 'Merk', 'Nama Tipe', 'Nomor Chasis', 'Location', 'Aksi'])
    expect(within(grid).getByText('HITACHI')).toBeInTheDocument()
    expect(within(grid).getByText('CH-1')).toBeInTheDocument()
  })

  it('grid objek lini lain berkolom Object dan Location; objek tambahan tetap dapat diisi', async () => {
    installFetch(
      response({
        objek: [{ id: '9', nama: 'Gudang Utama', lokasi: 'Jakarta', coverage: [] }],
      }),
    )
    show()

    const { grid, header } = await headerOf()
    expect(header).toEqual(['', '#', 'Object', 'Location', 'Aksi'])
    expect(within(grid).getByText('Gudang Utama')).toBeInTheDocument()
    expect(within(grid).getByText('Jakarta')).toBeInTheDocument()
    expect(within(grid).queryByText('Model')).toBeNull()

    // Objek tambahan tidak punya pasangan di polis: nama dan lokasinya diisi petugas, dan
    // kodenya diberikan otomatis karena grid Pega tidak punya kolom kode objek.
    await userEvent.click(screen.getByRole('button', { name: 'Tambah objek' }))
    expect(within(grid).getByLabelText('Nama objek baris 2')).toHaveValue('')
    expect(within(grid).getByLabelText('Lokasi objek baris 2')).toHaveValue('')
  })

  // Jaminan dan spreading tersembunyi sampai barisnya dibuka. Satu klaim Fire dapat
  // memuat belasan objek yang masing-masing berjaminan dan ber-spreading; menggambar
  // seluruhnya sekaligus menghapus perjajaran kolom yang menjadi alasan grid ini ada.

  it('jaminan tersembunyi sampai baris objeknya dibuka', async () => {
    installFetch(
      response({
        objek: [
          {
            id: 'OBJ-9',
            nama: 'Gudang Utama',
            lokasi: 'Jakarta',
            coverage: [{ id: 'C1', nama: 'All Risk', penyebab_kerugian: '', tsi_sen: 0, spreading: [] }],
          },
        ],
      }),
    )
    show()

    const toggle = await screen.findByRole('button', { name: 'Buka jaminan objek Gudang Utama' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByLabelText('Nama coverage jaminan 1 objek 1')).toBeNull()

    await userEvent.click(toggle)
    expect(screen.getByLabelText('Nama coverage jaminan 1 objek 1')).toHaveValue('All Risk')
    // Mata Uang mengikat mata uang POLIS, dan digambar sebagai teks — bukan isian.
    expect(screen.getByRole('table', { name: 'Jaminan objek Gudang Utama' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tutup jaminan objek Gudang Utama' }))
    expect(screen.queryByLabelText('Nama coverage jaminan 1 objek 1')).toBeNull()
  })

  it('spreading tersembunyi sampai baris jaminannya dibuka', async () => {
    installFetch(
      response({
        objek: [
          {
            id: 'OBJ-9',
            nama: 'Gudang Utama',
            lokasi: 'Jakarta',
            coverage: [
              {
                id: 'C1',
                nama: 'All Risk',
                penyebab_kerugian: '',
                tsi_sen: 0,
                spreading: [{ jenis_treaty: '10001', nama: 'OR', share: 1_000_000, objek_fac_offer: '', dihapus: false }],
              },
            ],
          },
        ],
      }),
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Buka jaminan objek Gudang Utama' }))
    expect(screen.queryByLabelText('Share persen')).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: 'Buka spreading jaminan 1 objek 1' }))
    expect(screen.getByLabelText('Share persen')).toHaveValue('100')
  })

  it('objek yang baru ditambahkan langsung terbuka', async () => {
    installFetch(response())
    show()

    // Menambah objek lalu mendapati tidak ada yang terjadi adalah tombol yang tampak
    // rusak. Baris baru karena itu digambar sudah terbuka.
    await userEvent.click(await screen.findByRole('button', { name: 'Tambah objek' }))
    expect(screen.getByRole('button', { name: 'Tambah coverage' })).toBeInTheDocument()
    expect(
      screen.getByText('Objek ini belum punya jaminan. Registrasi menolak objek tanpa jaminan.'),
    ).toBeInTheDocument()
  })

  it('objek tanpa jaminan dinyatakan, bukan dibiarkan tampak kosong', async () => {
    installFetch(
      response({ objek: [{ id: 'OBJ-9', nama: 'Gudang Utama', lokasi: 'Jakarta', coverage: [] }] }),
    )
    show()

    // Jaminan tersembunyi sampai barisnya dibuka — itulah sebab adanya tombol ini.
    await userEvent.click(
      await screen.findByRole('button', { name: 'Buka jaminan objek Gudang Utama' }),
    )

    // Gerbang validasi menolak objek tanpa jaminan. Menyatakannya di layar lebih murah
    // daripada membiarkan petugas menekan Next untuk mengetahuinya.
    expect(
      screen.getByText('Objek ini belum punya jaminan. Registrasi menolak objek tanpa jaminan.'),
    ).toBeInTheDocument()
  })

  it('Tambah coverage memilih coverage polis objek itu dan mengisi nama, TSI, serta spreading', async () => {
    installFetch(
      response({ objek: [{ id: 'OBJ-9', nama: 'Gudang Utama', lokasi: 'Jakarta', coverage: [] }] }),
      (url) =>
        url.includes('/pilihan-coverage?objek=OBJ-9')
          ? json(200, {
              pilihan: [
                {
                  id: 'C7',
                  nama: 'FLEXAS',
                  penyebab_kerugian: '',
                  tsi_sen: 100_000_000,
                  spreading: [{ jenis_treaty: '10001', nama: 'OR', share: 1_000_000, objek_fac_offer: '', dihapus: false }],
                },
              ],
            })
          : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Buka jaminan objek Gudang Utama' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah coverage' }))

    const code = await screen.findByRole('combobox', { name: 'Kode coverage jaminan 1 objek 1' })
    await screen.findByRole('option', { name: 'C7 — FLEXAS' })
    await userEvent.selectOptions(code, 'C7')

    expect(screen.getByLabelText('Nama coverage jaminan 1 objek 1')).toHaveValue('FLEXAS')
    expect(screen.getByLabelText('TSI jaminan 1 objek 1')).not.toHaveValue('')
    expect(screen.getByLabelText('Share persen')).toHaveValue('100')
    expect(calls.some((c) => c.url.includes('/klaim/klaim-1/pilihan-coverage?objek=OBJ-9'))).toBe(true)
  })

  it('Jenis Laporan adalah dropdown pilihan ReportType dengan bawaan Direct', async () => {
    installFetch(response())
    show()

    const field = await screen.findByRole('combobox', { name: 'Jenis Laporan' })
    expect(field).toHaveValue('1')
    expect(within(field).getAllByRole('option').map((o) => o.textContent)).toEqual([
      '— pilih —', 'Direct', 'Via Email', 'Via Fax', 'Via Pos / Kurir', 'Via Telephone', 'Via Portal',
    ])

    // Tersimpan ke T_CLAIM_PNC.REPORTTYPE lewat Save.
    await userEvent.selectOptions(field, '5')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(calls.some((c) => c.url === '/api/registrasi/register/simpan')).toBe(true))
    expect(calls.find((c) => c.url === '/api/registrasi/register/simpan')?.body).toMatchObject({ jenis_laporan: '5' })
  })

  it('Jenis Laporan tersimpan dimuat kembali, termasuk kode di luar daftar', async () => {
    installFetch(response({ jenis_laporan: '7' }))
    show()
    const field = await screen.findByRole('combobox', { name: 'Jenis Laporan' })
    expect(field).toHaveValue('7')
  })

  it('Mata Uang tersembunyi tetap dikirim sebagai kode; simbol lama IDR diganti kodenya', async () => {
    installFetch(response({ mata_uang: 'IDR' }), (url) =>
      url === '/api/registrasi/mata-uang'
        ? json(200, { pilihan: [{ id: '10001', nama: 'USD' }, { id: '10026', nama: 'IDR' }] })
        : undefined,
    )
    show()

    await screen.findByLabelText('Status Pelapor')
    // Simbol diterjemahkan begitu daftar mata uang termuat; Save lalu mengirim kodenya.
    await vi.waitFor(
      async () => {
        await userEvent.click(screen.getByRole('button', { name: 'Save' }))
        const sent = calls.filter((c) => c.url === '/api/registrasi/register/simpan').at(-1)?.body
        expect(sent).toMatchObject({ mata_uang: '10026' })
      },
      { timeout: 5000 },
    )
  })

  it('Tambah dan Hapus hanya mengubah layar — tabel baru berubah saat Save/Next/Back', async () => {
    installFetch(response())
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Tambah objek' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah coverage' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah spreading' }))
    await userEvent.click(screen.getByRole('button', { name: 'Hapus' }))
    await userEvent.click(screen.getByRole('button', { name: 'Hapus jaminan 1 objek 1' }))
    await userEvent.click(screen.getByRole('button', { name: 'Hapus objek baris 1' }))

    expect(calls.filter((c) => c.method !== 'GET')).toEqual([])
  })

  it('menambah dan membuang objek, coverage, serta spreading, dengan ringkasan share', async () => {
    installFetch(response())
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Tambah objek' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah coverage' }))
    await userEvent.type(screen.getByLabelText('TSI jaminan 1 objek 1'), '1.000')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah spreading' }))
    await userEvent.type(screen.getByLabelText('Share persen'), '60,5')
    // Objek Fac Offer tidak digunakan (Work Owner 2026-10-08): kolomnya tidak ada.
    expect(screen.queryByLabelText('Objek Fac Offer')).not.toBeInTheDocument()

    const summary = screen.getByRole('heading', { name: 'Ringkasan' }).parentElement!
    expect(within(summary).getByText(/belum 100%/)).toBeInTheDocument()
    expect(within(summary).getByText('1')).toBeInTheDocument()

    await userEvent.clear(screen.getByLabelText('Share persen'))
    await userEvent.type(screen.getByLabelText('Share persen'), '100')
    expect(within(summary).queryByText(/belum 100%/)).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus' }))
    expect(screen.queryByLabelText('Share persen')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus jaminan 1 objek 1' }))
    expect(screen.queryByLabelText('TSI jaminan 1 objek 1')).not.toBeInTheDocument()
    // Tombol buang objek bernama menurut barisnya: satu layar memuat belasan tombol
    // "Hapus", dan tiga di antaranya membuang hal yang berbeda.
    await userEvent.click(screen.getByRole('button', { name: 'Hapus objek baris 1' }))
    expect(screen.queryByRole('heading', { name: 'Ringkasan' })).not.toBeInTheDocument()
  })

  it('menyembunyikan isian wilayah Indonesia untuk negara lain dan mempertahankan nama tersimpan', async () => {
    installFetch(
      response({
        wilayah: {
          negara: 'TIMOR LESTE',
          negara_id: '99',
          provinsi: '',
          provinsi_id: '',
          kota: '',
          kota_id: '',
          kabupaten: '',
          kabupaten_id: '',
          kelurahan: '',
          kelurahan_id: '',
          kode_pos: '',
        },
      }),
    )
    show()

    const negara = await screen.findByLabelText('Negara')
    await within(negara).findByRole('option', { name: 'SINGAPURA' })
    // Negara tersimpan yang tidak ada di master tetap tampil sebagai pilihan terpilih.
    expect(negara).toHaveValue('99')
    expect(within(negara).getByRole('option', { name: 'TIMOR LESTE' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Kota')).not.toBeInTheDocument()

    await userEvent.selectOptions(negara, '2')
    expect(negara).toHaveValue('2')
    expect(screen.queryByLabelText('Kota')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Provinsi')).toBeEnabled()
  })

  it('kembali ke inbox lewat Cancel', async () => {
    installFetch(response())
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))
    expect(await screen.findByText('Halaman inbox')).toBeInTheDocument()
  })
})

/**
 * Tautan kembali memulihkan DAFTAR yang tadi dibuka, bukan sekadar membuka dashboard.
 *
 * Work Owner memintanya setelah mencari dengan lini bisnis lalu membuka satu klaim: kembali
 * ke daftar kosong berarti mengulang pencarian dari awal.
 */
describe('tautan kembali', () => {
  it('membawa seluruh penyaring kembali ke Dashboard Claim', async () => {
    installFetch(response({}, null))
    show(
      '/registrasi/klaim/klaim-1?asal=dashboard-claim&tile=outstanding&lini=003&pic=BUDI&hal=3',
    )

    const kembali = await screen.findByRole('link', { name: '← Kembali ke Dashboard Claim' })
    const tujuan = kembali.getAttribute('href') ?? ''
    expect(tujuan.startsWith('/dashboard-claim?')).toBe(true)

    const p = new URLSearchParams(tujuan.split('?')[1])
    expect(p.get('tile')).toBe('outstanding')
    expect(p.get('lini')).toBe('003')
    expect(p.get('pic')).toBe('BUDI')
    expect(p.get('hal')).toBe('3')

    // `asal` adalah penanda ARAH, bukan keadaan layar tujuan — ia tidak ikut pulang.
    expect(p.get('asal')).toBeNull()
  })

  it('tanpa asal, tujuannya tetap inbox seperti sebelumnya', async () => {
    installFetch(response({}, null))
    show()

    const kembali = await screen.findByRole('link', { name: '← Kembali ke inbox' })
    expect(kembali).toHaveAttribute('href', '/registrasi')
  })
})
