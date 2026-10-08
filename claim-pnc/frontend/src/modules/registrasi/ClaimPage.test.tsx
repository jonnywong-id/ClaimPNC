import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSession } from '@/app/session'
import { formatPercent, formatRupiah, formatDate, rupiahToCents } from '@/components/format'

import { ClaimPage } from './ClaimPage'

const CLAIM = {
  klaim: {
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
      mata_uang: 'IDR',
      nama_tertanggung: 'PT Contoh Industri Nusantara',
      deklarasi: false,
      penjamin_kredit: false,
    },
    tanggal_kejadian: '2026-06-05',
    tanggal_lapor: '2026-06-06',
    tanggal_terima_dokumen: '2026-06-07',
    lokasi: 'Gudang A',
    kronologi: 'Kebakaran gudang.',
    pelapor: { nama: 'Pelapor Uji', telepon: '0800', email: '', alamat: '', hubungan: 1, hubungan_lainnya: '' },
    nilai_estimasi_sen: 1_000_000_000,
    mata_uang: 'IDR',
    nomor_slik: '',
    ex_gratia: false,
    user_teknis: 'TEKNIK01',
    rcv_id: '',
    objek: [
      {
        id: 'OBJ-1',
        nama: 'Gudang',
        lokasi: 'Gudang A',
        coverage: [
          {
            id: 'CVG-1',
            nama: 'All Risk',
            penyebab_kerugian: '11817',
            tsi_sen: 50_000_000_000,
            spreading: [
              { jenis_treaty: '10007', nama: 'OR', share: 600_000, dihapus: false, objek_fac_offer: '' },
              { jenis_treaty: '10008', nama: 'Treaty', share: 400_000, dihapus: false, objek_fac_offer: '' },
            ],
          },
        ],
      },
    ],
    status_pucl: 0,
    transfer_compliance: false,
    status_proses: 'BERJALAN',
    status_klaim: '',
    flag_klaim: '0',
    status_posisi_progres: 'On Progress',
    tahap_kini: 'input-register',
  },
  tugas: {
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
  },
  jalur: ['input-register', 'estimasi-admin', 'pilih-surveyor'],
  jejak_keputusan: null,
  large_loss: false,
}

const ALUR = {
  nama: 'Register',
  mulai: 'view-polis',
  tahap: [
    { id: 'input-register', nama: 'Input Register', antrean: 'WORKLIST', workbasket: '', router: 'PNCAdminRouter', tindakan_keluar: 'InputRegister', pega_id: 'Assignment1' },
    { id: 'estimasi-admin', nama: 'Input Estimasi', antrean: 'WORKLIST', workbasket: '', router: 'PNCAdminRouter', tindakan_keluar: 'InputEstimasi', pega_id: 'Assignment7' },
    { id: 'pilih-surveyor', nama: 'Choose Surveyor', antrean: 'WORKLIST', workbasket: '', router: 'PNCTeknikRouter', tindakan_keluar: 'InputSurveyor', pega_id: 'Assignment3' },
  ],
}

let sentBody: unknown = null
let draftBody: unknown = null

/** Master wilayah palsu: satu rangkaian nyata dari layar Pega. */
const WILAYAH: Record<string, { id: string; nama: string; kode_pos?: string }[]> = {
  'negara|': [
    { id: '100009', nama: 'INDONESIA' },
    { id: '100001', nama: 'TIMOR LESTE' },
  ],
  'provinsi|INDONESIA': [{ id: '10012', nama: 'DI YOGYAKARTA' }],
  'kota|10012': [{ id: '10259', nama: 'KAB. SLEMAN' }],
  'kabupaten|10259': [{ id: '10000925', nama: 'KEC. DEPOK' }],
  'kelurahan|10000925': [{ id: '10004326', nama: 'KEL. CATURTUNGGAL', kode_pos: '55281' }],
}

function stubFetch(registerAnswer: () => { body: unknown; status: number }) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    let body: unknown = {}
    let status = 200

    if (url === '/api/registrasi/alur') body = ALUR
    else if (url.startsWith('/api/registrasi/wilayah/')) {
      const [path, query] = url.split('?')
      const level = path!.split('/').pop() ?? ''
      const parent = new URLSearchParams(query ?? '').get('induk') ?? ''
      body = { pilihan: WILAYAH[`${level}|${parent}`] ?? [] }
    } else if (url === '/api/registrasi/register/simpan') {
      draftBody = init?.body ? JSON.parse(init.body as string) : null
      body = CLAIM
    }
    else if (url.startsWith('/api/registrasi/klaim/')) body = CLAIM
    else if (url === '/api/registrasi/register') {
      sentBody = init?.body ? JSON.parse(init.body as string) : null
      const answer = registerAnswer()
      body = answer.body
      status = answer.status
    }

    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function mount(component: ReactNode) {
  const apiClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={apiClient}>
      <MemoryRouter initialEntries={['/registrasi/klaim/klaim-1']}>
        <Routes>
          <Route path="/registrasi/klaim/:claimID" element={component} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  sentBody = null
  draftBody = null
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSession.getState().login({
    token: 'token-contoh',
    user: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    validUntil: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('layar kerja klaim', () => {
  it('menampilkan tahapan klaim beserta tahap yang sedang berjalan', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)

    const stages = await screen.findByRole('navigation', { name: 'Tahapan klaim' })
    expect(stages).toHaveTextContent('Input Register')
    expect(stages).toHaveTextContent('Choose Surveyor')
    // Tahap berjalan ditandai untuk pembaca layar, bukan hanya dengan warna.
    expect(within(stages).getByText('Input Register')).toHaveAttribute('aria-current', 'step')
  })

  // InputRegister-sect.xml: Register · Kuisioner (!IsTravel) · Unggah Dokumen (!IsTravel) · Progress,
  // untuk semua lini (Work Owner, 2026-10-03). Tab Register memuat formulir InputRegisterDetail2
  // sampai section InputRegisterDetail diterima.
  it('memakai bingkai InputRegister dengan isian Input Register untuk lini selain PA', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)

    const tabs = await screen.findByRole('tablist', { name: 'Tab InputRegister' })
    expect(within(tabs).getAllByRole('tab').map((t) => t.textContent)).toEqual(
      ['Register', 'Kuisioner', 'Unggah Dokumen', 'Progress Claim & Komunikasi'])
    expect(within(tabs).getByRole('tab', { name: 'Register' })).toHaveAttribute('aria-selected', 'true')
    expect(within(tabs).queryByRole('tab', { name: 'Estimasi & Adjustment' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim ke Inputor' })).not.toBeInTheDocument()

    // InputRegisterDetail: DATA TERTANGGUNG KLAIM dan Email Tertanggung hanya IsPA; isian ClaimSurvey
    // (InputRegisterDetail2: Remarks Recommendation, Subject Email, Status Salvage) tidak tampil di sini.
    expect(screen.queryByRole('region', { name: 'DATA TERTANGGUNG KLAIM' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Email Pelapor')).toBeInTheDocument()
    expect(screen.getByLabelText('Status Pelapor')).toBeInTheDocument()
    expect(screen.queryByLabelText('Email Tertanggung')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Remarks Recommendation')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Status Salvage')).not.toBeInTheDocument()
    expect(screen.queryByText('Detail Ekspedisi')).not.toBeInTheDocument()

    // Tab Kuisioner = Section/QuestionnaireClaim.
    await userEvent.setup().click(within(tabs).getByRole('tab', { name: 'Kuisioner' }))
    expect(screen.getByLabelText('Nama Kepentingan')).toBeVisible()
    expect(screen.getByLabelText('Hilangnya Perlindungan')).toBeVisible()
    expect(screen.queryByLabelText('Catatan Ke Analyst')).not.toBeInTheDocument()
    // Ex Gratia (isPA_PNC) dan Tanggal Terima Dokumen (IsTravelPA) tidak tampil untuk Fire.
    expect(screen.queryByText('Ex Gratia')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Tanggal Terima Dokumen')).not.toBeInTheDocument()
  })

  it('menyembunyikan Kuisioner dan Unggah Dokumen untuk Travel (!IsTravel)', async () => {
    const travel = { ...CLAIM, klaim: { ...CLAIM.klaim, polis: { ...CLAIM.klaim.polis, lini: '005', nama_lini: 'Travel' } } }
    stubFetch(() => ({ body: travel, status: 200 }))
    const base = globalThis.fetch
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) =>
      url === '/api/registrasi/klaim/klaim-1'
        ? Promise.resolve(new Response(JSON.stringify(travel), { status: 200, headers: { 'Content-Type': 'application/json' } }))
        : base(url, init))
    mount(<ClaimPage />)

    const tabs = await screen.findByRole('tablist', { name: 'Tab InputRegister' })
    expect(within(tabs).getAllByRole('tab').map((t) => t.textContent)).toEqual(['Register', 'Progress Claim & Komunikasi'])
  })

  it('menampilkan isian khusus PA menurut InputRegisterDetail', async () => {
    const pa = { ...CLAIM, klaim: { ...CLAIM.klaim, polis: { ...CLAIM.klaim.polis, lini: '002', nama_lini: 'Personal Accident' } } }
    stubFetch(() => ({ body: pa, status: 200 }))
    const base = globalThis.fetch
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) =>
      url === '/api/registrasi/klaim/klaim-1'
        ? Promise.resolve(new Response(JSON.stringify(pa), { status: 200, headers: { 'Content-Type': 'application/json' } }))
        : base(url, init))
    mount(<ClaimPage />)

    const tabs = await screen.findByRole('tablist', { name: 'Tab InputRegister' })
    expect(within(tabs).getByRole('tab', { name: 'Kuisioner' })).toBeInTheDocument()
    expect(within(tabs).queryByRole('tab', { name: 'Investigasi' })).not.toBeInTheDocument()

    expect(screen.getByRole('region', { name: 'DATA TERTANGGUNG KLAIM' })).toBeInTheDocument()
    expect(screen.getByLabelText(/No KTP/)).toBeInTheDocument()
    expect(screen.getByText('Detail Ekspedisi')).toBeInTheDocument()
    expect(screen.getByLabelText('Email Tertanggung')).toBeInTheDocument()
    expect(screen.getByLabelText('Catatan Ke Analyst')).toBeInTheDocument()
    expect(screen.getByLabelText('Tanggal Terima Dokumen')).toBeInTheDocument()
    expect(screen.getByText('Ex Gratia')).toBeInTheDocument()
    expect(screen.queryByLabelText('Remarks Recommendation')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Subject Email')).not.toBeInTheDocument()

    // Tanggal Keluar Rawat Inap hanya setelah Rawat Inap dicentang.
    expect(screen.queryByLabelText('Tanggal Keluar Rawat Inap')).not.toBeInTheDocument()
    await userEvent.setup().click(screen.getByLabelText('Apakah Melakukan Rawat Inap ?'))
    expect(screen.getByLabelText('Tanggal Keluar Rawat Inap')).toBeInTheDocument()
  })

  it('mengirim uang sebagai sen dan share sebagai persen dikali sepuluh ribu', async () => {
    stubFetch(() => ({ body: { ...CLAIM, klaim: { ...CLAIM.klaim, nomor: 'PNCN.26.0001' } }, status: 200 }))
    mount(<ClaimPage />)

    const save = await screen.findByRole('button', { name: 'Next' })
    await userEvent.setup().click(save)

    await waitFor(() => expect(sentBody).not.toBeNull())
    expect(sentBody).toMatchObject({
      tugas_id: 'tugas-1',
      nilai_estimasi_sen: 1_000_000_000,
      kembali: false,
    })

    const objek = (sentBody as { objek: { coverage: { nama: string; spreading: { share: number }[] }[] }[] }).objek
    expect(objek[0]?.coverage[0]?.spreading[0]?.share).toBe(600_000)
    // Objek dan coverage terisi dari polis saat klaim dibuka; namanya ikut kembali supaya
    // tidak hilang saat disimpan.
    expect(objek[0]?.coverage[0]?.nama).toBe('All Risk')

    // Jaminan tersembunyi sampai baris objeknya dibuka; nilainya tetap ikut terkirim.
    await userEvent.setup().click(screen.getAllByRole('button', { name: /^Buka jaminan objek/ })[0]!)
    expect(screen.getByLabelText('Nama coverage jaminan 1 objek 1')).toHaveValue('All Risk')
  })

  // Sistem lama menampilkan satu pesan, lalu pesan berikutnya setelah disimpan ulang.
  // Di sini seluruhnya ditampilkan sekaligus, dan masing-masing menempel pada kolomnya.
  it('menempelkan setiap pelanggaran pada kolom yang harus diperbaiki', async () => {
    stubFetch(() => ({
      body: {
        kode: 'validasi_gagal',
        pesan: 'Nomor SLIK Harus Diisi',
        detail: [
          { kode: 'nomor_slik_kosong', field: 'nomor_slik', pesan: 'Nomor SLIK Harus Diisi' },
          {
            kode: 'tanggal_lapor_sebelum_kejadian',
            field: 'tanggal_lapor',
            pesan: 'Tanggal Lapor harus setelah Tanggal Kejadian.',
          },
        ],
      },
      status: 422,
    }))
    mount(<ClaimPage />)

    const save = await screen.findByRole('button', { name: 'Next' })
    await userEvent.setup().click(save)

    const slik = await screen.findByLabelText('Nomor SLIK')
    await waitFor(() => expect(slik).toHaveAttribute('aria-invalid', 'true'))

    const report = screen.getByLabelText('Tanggal Lapor')
    expect(report).toHaveAttribute('aria-invalid', 'true')

    expect(screen.getByText('2 ketentuan belum terpenuhi:')).toBeInTheDocument()
  })

  it('menandai tombol Back sehingga server tahu klaim tidak maju', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)

    const kembali = await screen.findByRole('button', { name: 'Back' })
    await userEvent.setup().click(kembali)

    await waitFor(() => expect(sentBody).not.toBeNull())
    expect(sentBody).toMatchObject({ kembali: true })
  })

  // Rangkaian dari tangkapan layar Pega. Memilih kelurahan mengisi Kode Pos dari master,
  // dan Kota sampai Kode Pos baru tampil setelah Negara INDONESIA dipilih.
  it('memilih wilayah bertingkat dan mengisi kode pos dari kelurahan', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)
    const user = userEvent.setup()

    const negara = await screen.findByLabelText('Negara')
    expect(screen.queryByLabelText('Kota')).not.toBeInTheDocument()

    await waitFor(() => expect(screen.getByRole('option', { name: 'INDONESIA' })).toBeInTheDocument())
    await user.selectOptions(negara, '100009')

    const provinsi = screen.getByLabelText('Provinsi')
    await waitFor(() => expect(screen.getByRole('option', { name: 'DI YOGYAKARTA' })).toBeInTheDocument())
    await user.selectOptions(provinsi, '10012')

    await waitFor(() => expect(screen.getByRole('option', { name: 'KAB. SLEMAN' })).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText('Kota'), '10259')

    await waitFor(() => expect(screen.getByRole('option', { name: 'KEC. DEPOK' })).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText('Kabupaten'), '10000925')

    await waitFor(() => expect(screen.getByRole('option', { name: 'KEL. CATURTUNGGAL' })).toBeInTheDocument())
    await user.selectOptions(screen.getByLabelText('Kelurahan'), '10004326')

    expect(screen.getByLabelText('Kode Pos')).toHaveValue('55281')

    await user.click(screen.getByLabelText('SUSPICIOUS'))
    await user.type(screen.getByLabelText('Komentar Suspicious'), 'Dokumen janggal')

    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(sentBody).not.toBeNull())
    expect(sentBody).toMatchObject({
      wilayah: {
        negara: 'INDONESIA', negara_id: '100009',
        provinsi: 'DI YOGYAKARTA', provinsi_id: '10012',
        kota: 'KAB. SLEMAN', kota_id: '10259',
        kabupaten: 'KEC. DEPOK', kabupaten_id: '10000925',
        kelurahan: 'KEL. CATURTUNGGAL', kelurahan_id: '10004326',
        kode_pos: '55281',
      },
      prinsip_mengenal_nasabah: '2',
      komentar_suspicious: 'Dokumen janggal',
    })
  })

  // Save menyimpan TANPA menutup tahap: ia memanggil jalur simpan, bukan jalur register.
  it('tombol Save menyimpan tanpa menutup tahap Input Register', async () => {
    stubFetch(() => ({ body: CLAIM, status: 200 }))
    mount(<ClaimPage />)

    await userEvent.setup().click(await screen.findByRole('button', { name: 'Save' }))

    await waitFor(() => expect(draftBody).not.toBeNull())
    expect(sentBody).toBeNull()
    expect(draftBody).toMatchObject({ tugas_id: 'tugas-1', prinsip_mengenal_nasabah: '1', ex_gratia: false })
    expect(await screen.findByText(/Klaim tetap di tahap Input Register/)).toBeInTheDocument()
  })
})

describe('pemformatan bersama', () => {
  it('menuliskan uang dari sen dengan pemisah Indonesia', () => {
    expect(formatRupiah(100_000_000)).toBe('Rp 1.000.000,00')
    expect(formatRupiah(0)).toBe('Rp 0,00')
    expect(formatRupiah(123_456)).toBe('Rp 1.234,56')
  })

  it('menuliskan persentase empat desimal tanpa nol di belakang', () => {
    expect(formatPercent(1_000_000)).toBe('100%')
    expect(formatPercent(999_999)).toBe('99,9999%')
  })

  // `new Date('2026-06-05')` ditafsirkan peramban sebagai tengah malam UTC, sehingga
  // pengguna di WIB melihat 4 Juni. Kelas kesalahan itu tidak boleh lahir kembali.
  it('menuliskan tanggal tanpa pernah bergeser sehari', () => {
    expect(formatDate('2026-06-05')).toBe('5 Juni 2026')
    expect(formatDate('2026-01-01')).toBe('1 Januari 2026')
    expect(formatDate('')).toBe('—')
  })

  it('membaca rupiah yang diketik dengan pemisah Indonesia menjadi sen', () => {
    expect(rupiahToCents('1.000.000')).toBe(100_000_000)
    expect(rupiahToCents('1.234,56')).toBe(123_456)
    expect(rupiahToCents('')).toBe(0)
    expect(Number.isNaN(rupiahToCents('bukan angka'))).toBe(true)
  })
})

// Tahap Input Estimasi: klaim Non-MBU yang lolos Input Register sampai di sini, dan layar
// yang muncul adalah isian estimasi — bukan tombol "Selesaikan" umum.
describe('tahap Input Estimasi', () => {
  const AT_ESTIMATE = {
    ...CLAIM,
    klaim: { ...CLAIM.klaim, nomor: 'PNCN.26.0012', tahap_kini: 'estimasi-admin' },
    tugas: { ...CLAIM.tugas, tahap: 'estimasi-admin', nama_tahap: 'Input Estimasi', tindakan_keluar: 'InputEstimasi' },
    jalur: ['estimasi-admin', 'pilih-surveyor'],
  }

  const RECORDS = {
    survey: {
      survey: [
        {
          kasus_id: 'ASM-FW-GCNMFW-WORK SRV-9', tipe: '2', nama_surveyor: 'PT ADJUSTER CONTOH', tanggal_survey: '2026-09-20',
          lokasi_survey: '', nama_objek: 'Gudang', lokasi_objek: 'Jakarta', urutan: '1', status: 'Preliminary Advice',
          keterangan: '', tanggal_input: '2026-09-21',
        },
      ],
    },
    dokumen: {
      kategori: [
        { kode: 'REGISTER', nama: 'PENDAFTARAN', dokumen: [{ id: '14901', jenis_id: '10064', nama: 'PELAPORAN KLAIM', wajib: true, minimal: '1', terunggah: 2 }] },
        { kode: 'SURVEY', nama: 'SURVEI', dokumen: [] },
        { kode: 'SALVAGE', nama: 'SALVAGE', dokumen: [] },
      ],
      berkas: [
        {
          id: '1', nama: 'laporan.pdf', jenis_berkas: 'pdf', catatan: '', kategori: '10064', sub_kategori: '14901',
          tersimpan: true, diunggah_oleh: 'ADMIN01', diunggah_pada: '2026-09-24T15:58:53+07:00',
        },
      ],
    },
    progres: {
      progres: [
        {
          urutan: 1, tanggal_input: '2026-09-25T09:09:09+07:00', status_1: '002', status_1_nama: 'REGISTRASI', status_2: '2',
          status_2_nama: 'POLIS BELUM ADA', keterangan: 'Auto Create Register', tindak_lanjut: '2026-10-02T09:09:09+07:00', diinput_oleh: 'ADMIN01',
        },
      ],
      komunikasi: [
        {
          kasus_id: 'ASM-FW-GCNMFW-WORK SRV-9', id: '22', tanggal: '2026-09-26T10:00:00+07:00', pengirim: 'ADMIN SATU',
          pesan: 'Mohon laporan survey', balasan: 'Sudah dikirim', penjawab: 'SURVEYOR', tanggal_balasan: '',
        },
      ],
    },
  }

  let estimateBody: { url: string; body: unknown } | null = null
  let uploads: FormData[] = []
  let itemOptions: { nama: string; kelompok: string; tsi_sen: number }[] = []

  function stubEstimate(claim: unknown = AT_ESTIMATE) {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      let body: unknown = {}
      if (url === '/api/registrasi/alur') body = ALUR
      else if (url.includes('/pilihan-item')) body = { pilihan: itemOptions }
      else if (url.endsWith('/survey')) body = RECORDS.survey
      else if (url.endsWith('/dokumen')) {
        if (init?.method === 'POST') uploads.push(init.body as FormData)
        body = RECORDS.dokumen
      }
      else if (url.endsWith('/progres')) body = RECORDS.progres
      else if (url.startsWith('/api/registrasi/klaim/')) body = claim
      else if (url === '/api/registrasi/mata-uang') body = { pilihan: [{ id: 'IDR', nama: 'IDR' }, { id: '10001', nama: 'USD' }] }
      else if (url.startsWith('/api/registrasi/estimasi')) {
        estimateBody = { url, body: init?.body ? JSON.parse(init.body as string) : null }
        body = AT_ESTIMATE
      }
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
  }

  beforeEach(() => {
    estimateBody = null
    uploads = []
    itemOptions = []
  })

  // Section InputEstimasiAdmin tidak punya tombol Next: tahap ditutup Kirim PIC Teknik
  // (finishAssignment), yang mati selama belum ada Claim Face Sheet (!isCFS).
  it('Kirim PIC Teknik mati sebelum CFS, dan tidak ada tombol Next', async () => {
    stubEstimate()
    mount(<ClaimPage />)
    const user = userEvent.setup()

    expect(await screen.findByRole('region', { name: 'Input Estimasi' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Selesaikan/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument()

    await user.type(screen.getByLabelText('Objek'), 'Gudang')
    await user.click(screen.getByRole('button', { name: 'Tambah estimasi' }))
    await user.type(screen.getByLabelText('Nilai Estimasi'), '8.000.000')
    expect(screen.getByRole('button', { name: 'Kirim PIC Teknik' })).toBeDisabled()
  })

  it('Kirim PIC Teknik setelah CFS menutup tahap Input Estimasi', async () => {
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim,
        objek: [
          {
            ...AT_ESTIMATE.klaim.objek[0]!,
            coverage: [
              {
                ...AT_ESTIMATE.klaim.objek[0]!.coverage[0]!,
                item: [
                  {
                    nama: 'Gudang', deskripsi: '', kelompok: '',
                    estimasi: [{ tipe: '1', mata_uang: 'IDR', tanggal: '2026-09-27', nilai_sen: 800_000_000, kurs_e4: 10_000, nilai_idr_sen: 800_000_000, sudah_cfs: true }],
                  },
                ],
              },
            ],
          },
        ],
      },
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    const send = await screen.findByRole('button', { name: 'Kirim PIC Teknik' })
    await waitFor(() => expect(send).toBeEnabled())
    await user.click(send)

    await waitFor(() => expect(estimateBody).not.toBeNull())
    expect(estimateBody?.url).toBe('/api/registrasi/estimasi')
    expect(estimateBody?.body).toMatchObject({
      tugas_id: 'tugas-1',
      kembali: false,
      objek: [{ coverage: [{ item: [{ nama: 'Gudang', estimasi: [{ tipe: '1', mata_uang: 'IDR', nilai_sen: 800_000_000 }] }] }] }],
    })
  })

  // Sesudah Kirim PIC Teknik, klaim Non-MBU berada di Choose Surveyor: layar InputSurveyor
  // (ClaimSurvey_sect), bukan tombol "Selesaikan" umum.
  it('Choose Surveyor menampilkan layar InputSurveyor', async () => {
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: { ...AT_ESTIMATE.klaim, tahap_kini: 'pilih-surveyor' },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor' },
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    expect(await screen.findByRole('region', { name: 'InputSurveyor' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Selesaikan/ })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kirim ke Admin' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Kirim ke Inputor' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim ke Marketing' })).not.toBeInTheDocument()

    expect(screen.getByRole('tab', { name: 'Adjustment & Akseptasi' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('columnheader', { name: 'Nilai Akseptasi Klaim' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeEnabled()

    await user.click(screen.getByRole('tab', { name: 'Penerima Klaim' }))
    expect(screen.getByRole('columnheader', { name: 'Alamat' })).toBeInTheDocument()

    // Kirim ke Inputor (local action AnalystRemarks) membuka modal catatan.
    await user.click(screen.getByRole('button', { name: 'Kirim ke Inputor' }))
    expect(screen.getByRole('dialog', { name: 'Kirim ke Inputor' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(screen.queryByRole('dialog', { name: 'Kirim ke Inputor' })).not.toBeInTheDocument()
  })

  // Tahap Estimation PA (Assignment4) ditutup flow action InputSurveyor — layar ClaimSurvey_sect
  // dengan perbedaan PA: tanpa Status Klaim (!IsPA), tanpa Survey dan Kirim ke Admin, dengan tab
  // Investigasi, serta Submit/Back untuk memajukan tahap.
  // Register_Flow: Estimation PA → Investigator (Assignment11, workbasket InvestigatorPNC) →
  // Send To Analis. Investigator ditutup flow action InputInvestigator, dengan bingkai
  // ClaimSurvey_sect yang sama dan tab Investigasi terbuka lebih dulu.
  it('Investigator PA memakai layar ClaimSurvey dan menutup tahap dengan InputInvestigator', async () => {
    let completed: { url: string; body: unknown } | null = null
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim, tahap_kini: 'investigator',
        polis: { ...AT_ESTIMATE.klaim.polis, lini: '002', nama_lini: 'Personal Accident', jenis_bisnis: 'PersonalAccident' },
      },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'investigator', nama_tahap: 'Investigator', tindakan_keluar: 'InputInvestigator' },
    })
    const base = globalThis.fetch
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      if (url.endsWith('/selesai')) {
        completed = { url, body: init?.body ? JSON.parse(init.body as string) : null }
        return Promise.resolve(new Response(JSON.stringify(AT_ESTIMATE), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return base(url, init)
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    expect(await screen.findByRole('region', { name: 'InputInvestigator' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Investigasi' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.queryByLabelText('Status Klaim')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Submit' }))
    await waitFor(() => expect(completed).not.toBeNull())
    expect(completed!.body).toMatchObject({ action: 'InputInvestigator', kembali: false })
  })

  it('tugas Investigator di antrean harus diambil sebelum Submit', async () => {
    let took = ''
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim, tahap_kini: 'investigator',
        polis: { ...AT_ESTIMATE.klaim.polis, lini: '002', nama_lini: 'Personal Accident', jenis_bisnis: 'PersonalAccident' },
      },
      tugas: {
        ...AT_ESTIMATE.tugas, tahap: 'investigator', nama_tahap: 'Investigator', tindakan_keluar: 'InputInvestigator',
        antrean: 'InvestigatorPNC', pemilik: '', dapat_diambil: true,
      },
    })
    const base = globalThis.fetch
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      if (url.endsWith('/ambil')) {
        took = url
        return Promise.resolve(new Response(JSON.stringify({ ...AT_ESTIMATE.tugas, klaim_id: 'klaim-1' }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return base(url, init)
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    expect(await screen.findByText(/masih di antrean InvestigatorPNC/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Ambil' }))
    await waitFor(() => expect(took.endsWith('/ambil')).toBe(true))
  })

  it('Estimation PA menampilkan layar InputSurveyor versi PA', async () => {
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim, tahap_kini: 'estimasi-pa',
        polis: { ...AT_ESTIMATE.klaim.polis, lini: '002', nama_lini: 'Personal Accident', jenis_bisnis: 'PersonalAccident' },
      },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'estimasi-pa', nama_tahap: 'Estimation', tindakan_keluar: 'InputSurveyor' },
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    expect(await screen.findByRole('region', { name: 'InputSurveyor' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Status Klaim')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim ke Admin' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Survey' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Submit' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Back' })).toBeEnabled()

    await user.click(screen.getByRole('tab', { name: 'Investigasi' }))
    // TabInvestigasi-sect.xml: hanya "Hasil investigasi" (Input Investigasi bersyarat 1=2).
    expect(await screen.findByRole('tab', { name: 'Hasil investigasi' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Input Investigasi' })).not.toBeInTheDocument()
    expect(screen.getByText('Tanggal Investigasi')).toBeInTheDocument()

    // InputEstimasi-sect.xml: PA tanpa Estimasi Pembayaran, dengan Catatan Untuk Analyst.
    await user.click(screen.getByRole('tab', { name: 'Estimasi & Adjustment' }))
    expect(screen.queryByRole('tab', { name: 'Estimasi Pembayaran' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: 'Catatan Untuk Analyst' }))
    expect(screen.getByText('Catatan Dari Compliance')).toBeInTheDocument()
    expect(screen.getByText('Komentar Dari Analyst Doctor')).toBeInTheDocument()
  })

  // Sub-tab Estimasi Pembayaran InputSurveyor adalah section InputEstimasiDetail yang sama
  // dengan Input Estimasi (ClaimSurvey_sect menanamnya): estimasi dapat ditambah dan disimpan
  // tanpa memindahkan tahap — Kirim PIC Teknik tidak ada di sini.
  it('Choose Surveyor: Estimasi Pembayaran dapat ditambah dan disimpan', async () => {
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: { ...AT_ESTIMATE.klaim, tahap_kini: 'pilih-surveyor' },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor' },
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    await screen.findByRole('region', { name: 'InputSurveyor' })
    await user.click(screen.getByRole('tab', { name: 'Estimasi Pembayaran' }))
    expect(screen.getByRole('button', { name: 'Download Claim Face Sheet' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim PIC Teknik' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(estimateBody).not.toBeNull())
    expect(estimateBody?.url).toBe('/api/registrasi/estimasi/simpan')
    expect(estimateBody?.body).toMatchObject({ tugas_id: 'tugas-1', kembali: false })
  })

  // Choose Surveyor dirutekan ke PIC Teknik; bila tugas milik orang lain, Tambah dikunci
  // dengan keterangan alih-alih ditolak server setelah diisi. Kolom "Button" tidak ada.
  it('Choose Surveyor milik PIC Teknik lain: Tambah dikunci', async () => {
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: { ...AT_ESTIMATE.klaim, tahap_kini: 'pilih-surveyor' },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor', pemilik: 'TEKNIK01' },
    })
    mount(<ClaimPage />)

    expect(await screen.findByText(/Tugas ini milik TEKNIK01/, { selector: 'p' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()
    expect(screen.queryByRole('columnheader', { name: 'Button' })).not.toBeInTheDocument()
  })

  // Baris adjustment tersimpan dapat diklik untuk dilihat kembali; tab Penerima Klaim
  // menampilkan penerima (ViewShowReceiver: Nama, Alamat).
  it('adjustment tersimpan dapat dibuka, dan Penerima Klaim menampilkan penerimanya', async () => {
    const base = AT_ESTIMATE.klaim.objek[0]!
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim,
        tahap_kini: 'pilih-surveyor',
        penerima_klaim: [{ id: '1', nama: 'TERTANGGUNG UJI', alamat: 'JL. UJI NO. 1', nama_bank: '', nomor_rekening: '' }],
        objek: [
          {
            ...base,
            coverage: [
              {
                ...base.coverage[0]!,
                adjustment: [
                  {
                    tipe_pembayaran: '2', nama_tipe_pembayaran: 'Interim', mata_uang: 'IDR', kurs_e4: 10_000,
                    nilai_propose_sen: 250_000_000, nilai_pengajuan_sen: 250_000_000, loc: 0, nilai_salvage_sen: 0,
                    nilai_salvage_b_sen: 0, nilai_interim_sen: 0, nilai_estimasi_sen: 0, tipe_resiko: '3',
                    persen_resiko: 0, nilai_resiko_sen: 0, nilai_gross_sen: 250_000_000, share_asm: 650_000,
                    nilai_asm_sen: 162_500_000, nilai_akseptasi_sen: 250_000_000, kronologi: '', catatan: '',
                    status_akseptasi: '', nomor_akseptasi: '',
                  },
                ],
              },
            ],
          },
        ],
      },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor' },
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    const row = await screen.findByRole('button', { name: /Adjustment 1/ })
    expect(row).toHaveAttribute('aria-expanded', 'false')
    await user.click(row)
    // Baris yang belum diakseptasi tetap dapat diubah (InputAdjustment: nonaktif hanya bila AcceptanceStatus != '').
    const detail = screen.getByRole('group', { name: 'Ubah adjustment 1' })
    expect(within(detail).getByLabelText('Tipe Resiko Sendiri')).toHaveValue('3')
    expect(within(detail).getByRole('button', { name: 'Hapus' })).toBeDisabled()
    await user.click(row)
    expect(screen.queryByRole('group', { name: 'Ubah adjustment 1' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'Penerima Klaim' }))
    expect(screen.getByRole('cell', { name: 'TERTANGGUNG UJI' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'JL. UJI NO. 1' })).toBeInTheDocument()
  })

  // Baris Penerima Klaim dibuka menjadi panel InputReceiver: No Rekening membaca Master
  // Rekening saat ditinggalkan, Email/Telepon terisi dari master, lalu Simpan mengirim penerima.
  it('baris Penerima Klaim dibuka, No Rekening mengisi dari master, dan Simpan mengirim penerima', async () => {
    const atSurveyor = {
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim,
        tahap_kini: 'pilih-surveyor',
        penerima_klaim: [{ id: '1', nama: 'TERTANGGUNG UJI', alamat: 'JL. UJI NO. 1', nama_bank: '', nomor_rekening: '' }],
      },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor' },
    }
    const account = {
      nomor_rekening: '1234567890', nama: 'PT CONTOH PENERIMA', nama_bank: 'BANK CONTOH', nama_cabang_bank: 'JAKARTA',
      alamat: 'JL. CONTOH NO. 1', id_bank: '001', email: 'penerima@contoh.internal', telepon: '',
      tanggal_approve_kasir: '', tanggal_approve_komite: '2026-07-03',
    }
    const sent: { url: string; body: unknown }[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      let body: unknown = {}
      if (url === '/api/registrasi/rekening/1234567890') body = account
      else if (url.endsWith('/penerima')) {
        sent.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
        body = atSurveyor
      } else if (url === '/api/registrasi/alur') body = ALUR
      else if (url === '/api/registrasi/mata-uang') body = { pilihan: [{ id: 'IDR', nama: 'IDR' }] }
      else if (url.startsWith('/api/registrasi/klaim/')) body = atSurveyor
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    await user.click(await screen.findByRole('tab', { name: 'Penerima Klaim' }))
    const row = screen.getByRole('button', { name: 'TERTANGGUNG UJI' })
    expect(row).toHaveAttribute('aria-expanded', 'false')
    await user.click(row)
    const pane = screen.getByRole('group', { name: 'InputReceiver' })

    await user.type(within(pane).getByLabelText('No Rekening'), '1234567890')
    await user.tab()
    expect(await within(pane).findByText('BANK CONTOH')).toBeInTheDocument()
    expect(within(pane).getByText('JAKARTA')).toBeInTheDocument()
    expect(within(pane).getByText('PT CONTOH PENERIMA')).toBeInTheDocument()
    expect(within(pane).getByText(formatDate('2026-07-03'))).toBeInTheDocument()
    await waitFor(() => expect(within(pane).getByLabelText(/Email/)).toHaveValue('penerima@contoh.internal'))

    await user.type(within(pane).getByLabelText('Telepon'), '0812')
    await user.click(within(pane).getByRole('button', { name: 'Simpan' }))
    await waitFor(() => expect(sent).toHaveLength(1))
    expect(sent[0]?.url).toBe('/api/registrasi/klaim/klaim-1/penerima')
    expect(sent[0]?.body).toEqual({
      tugas_id: 'tugas-1', id: '1', nomor_rekening: '1234567890', email: 'penerima@contoh.internal', telepon: '0812',
    })
  })

  // Transfer Komite mengirim alamat baris adjustment; baris yang sudah ditransfer menampilkan
  // status komite per jenjang dan tombolnya mati (IsKomiteTransfer).
  it('Transfer Komite mengirim baris adjustment, dan baris tertransfer menampilkan status komite', async () => {
    const base = AT_ESTIMATE.klaim.objek[0]!
    const line = {
      tipe_pembayaran: '1', nama_tipe_pembayaran: 'Final', mata_uang: 'IDR', kurs_e4: 10_000,
      nilai_propose_sen: 1_000_000_000, nilai_pengajuan_sen: 1_200_000_000, loc: 0, nilai_salvage_sen: 0,
      nilai_salvage_b_sen: 0, nilai_interim_sen: 0, nilai_estimasi_sen: 0, tipe_resiko: '1',
      persen_resiko: 100_000, nilai_resiko_sen: 100_000_000, nilai_gross_sen: 900_000_000, share_asm: 1_000_000,
      nilai_asm_sen: 900_000_000, nilai_akseptasi_sen: 900_000_000, kronologi: '', catatan: '',
      status_akseptasi: '', nomor_akseptasi: '',
    }
    const atSurveyor = (lines: unknown[]) => ({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim,
        tahap_kini: 'pilih-surveyor',
        objek: [{ ...base, coverage: [{ ...base.coverage[0]!, adjustment: lines }] }],
      },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor' },
    })
    const sent: { url: string; body: unknown }[] = []
    let current = atSurveyor([line, { ...line, komite_id: 'KMTN-00001', status_akseptasi: '0' }])
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      let body: unknown = {}
      if (url.endsWith('/adjustment/komite')) {
        sent.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
        body = { klaim: current.klaim, komite: { id: 'KMTN-00002', nomor_klaim: 'PNCN.26.0001', status: 'berjalan', anggota: [] } }
      } else if (url === '/api/registrasi/komite/KMTN-00001') {
        body = {
          id: 'KMTN-00001', nomor_klaim: 'PNCN.26.0001', status: 'berjalan', menunggu: 'KOMITE01',
          anggota: [
            { jenjang: 1, komite: 'KOMITE01', keputusan: '0', catatan: '' },
            { jenjang: 2, komite: 'KOMITE02', keputusan: '0', catatan: '' },
          ],
        }
      } else if (url === '/api/registrasi/alur') body = ALUR
      else if (url === '/api/registrasi/mata-uang') body = { pilihan: [{ id: 'IDR', nama: 'IDR' }] }
      else if (url.startsWith('/api/registrasi/klaim/')) body = current
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    const buttons = await screen.findAllByRole('button', { name: 'Transfer Komite' })
    expect(buttons).toHaveLength(2)
    expect(buttons[0]).toBeEnabled()
    expect(buttons[1]).toBeDisabled()
    expect(await screen.findByText(/Komite KMTN-00001 · jenjang 1\/2 menunggu KOMITE01/)).toBeInTheDocument()
    expect(screen.getByText('Belum ditransfer')).toBeInTheDocument()

    current = atSurveyor([line, line])
    await user.click(buttons[0]!)
    await waitFor(() => expect(sent).toHaveLength(1))
    expect(sent[0]?.url).toBe('/api/registrasi/klaim/klaim-1/adjustment/komite')
    expect(sent[0]?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 1, adjustment: 1 })
  })

  // Anggota grup PIC Teknik (M_LOGIN_GROUP_PNC) boleh mengerjakan tugas milik PIC lain:
  // server menyatakannya lewat dapat_dikerjakan.
  it('Choose Surveyor milik PIC lain tetapi dapat dikerjakan grup: Tambah aktif', async () => {
    stubEstimate({
      ...AT_ESTIMATE,
      klaim: { ...AT_ESTIMATE.klaim, tahap_kini: 'pilih-surveyor' },
      tugas: {
        ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor',
        pemilik: 'TEKNIK01', dapat_dikerjakan: true,
      },
    })
    mount(<ClaimPage />)

    expect(await screen.findByRole('button', { name: 'Tambah' })).toBeEnabled()
    expect(screen.queryByText(/Tugas ini milik/)).not.toBeInTheDocument()
  })

  // Tombol Tambah membuka baris isian di dalam grid Adjustment dan mengirimnya ke rute adjustment.
  it('Tambah pada grid Adjustment mengirim isian adjustment', async () => {
    // Klaim Ex Gratia: tabel treaty adjustment hanya tampil di kontainer `.ExGratia = 1`.
    const atSurveyor = {
      ...AT_ESTIMATE,
      klaim: { ...AT_ESTIMATE.klaim, tahap_kini: 'pilih-surveyor', ex_gratia: true },
      tugas: { ...AT_ESTIMATE.tugas, tahap: 'pilih-surveyor', nama_tahap: 'Choose Surveyor', tindakan_keluar: 'InputSurveyor' },
    }
    let sent: { url: string; body: unknown } | null = null
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      let body: unknown = {}
      if (url.endsWith('/adjustment/hitung')) {
        body = {
          adjustment: {
            tipe_pembayaran: '1', nama_tipe_pembayaran: 'Final', mata_uang: 'IDR', kurs_e4: 10_000,
            nilai_propose_sen: 1_000_000_000, nilai_pengajuan_sen: 1_200_000_000, loc: 0, nilai_salvage_sen: 0,
            nilai_salvage_b_sen: 0, nilai_interim_sen: 0, nilai_estimasi_sen: 5_000_000_000, tipe_resiko: '1',
            persen_resiko: 100_000, nilai_resiko_sen: 100_000_000, nilai_gross_sen: 900_000_000, share_asm: 345_000,
            nilai_asm_sen: 310_500_000, nilai_akseptasi_sen: 900_000_000, kronologi: '', catatan: '',
            status_akseptasi: '', nomor_akseptasi: '',
          },
          spreading: [{ jenis_treaty: '10007', nama: 'QS', share: 72_460, dihapus: false, objek_fac_offer: '' }],
        }
      } else if (url.endsWith('/adjustment')) {
        sent = { url, body: init?.body ? JSON.parse(init.body as string) : null }
        // Isian belum lengkap ditolak (seperti SetNilaiResikoSendiri); lengkap tersimpan sebagai baris 1.
        const complete = (sent.body as { persen_resiko?: number } | null)?.persen_resiko
        if (!complete) {
          return Promise.resolve(
            new Response(JSON.stringify({ kode: 'validasi_gagal', pesan: 'x', detail: [{ kode: 'a', field: '', pesan: 'Tipe Resiko Harus Diisi' }] }), {
              status: 422,
              headers: { 'Content-Type': 'application/json' },
            }),
          )
        }
        const cov = atSurveyor.klaim.objek[0]!
        body = {
          ...atSurveyor,
          klaim: {
            ...atSurveyor.klaim,
            objek: [{ ...cov, coverage: [{ ...cov.coverage[0]!, adjustment: [{ tipe_pembayaran: '1', nama_tipe_pembayaran: 'Final', status_akseptasi: '' }] }] }],
          },
        }
      } else if (url === '/api/registrasi/alur') body = ALUR
      else if (url === '/api/registrasi/mata-uang') body = { pilihan: [{ id: 'IDR', nama: 'IDR' }] }
      else if (url.startsWith('/api/registrasi/klaim/')) body = atSurveyor
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Tambah' }))
    const row = screen.getByRole('group', { name: 'Adjustment baru' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await user.selectOptions(within(row).getByLabelText('Tipe Pembayaran'), '1')
    await user.type(within(row).getByLabelText('Total Klaim'), '10.000.000')
    await user.type(within(row).getByLabelText('Nilai Pengajuan Tertanggung'), '12.000.000')
    await user.selectOptions(within(row).getByLabelText('Tipe Resiko Sendiri'), '1')
    await user.type(within(row).getByLabelText('Persen Resiko Sendiri (%)'), '10')
    // Nilai tampilan dihitung server (pratinjau), bukan oleh layar.
    expect(await within(row).findByText('9.000.000,00')).toBeInTheDocument()
    expect(within(row).getByText('34,5000')).toBeInTheDocument()
    expect(within(row).getByRole('cell', { name: 'QS' })).toBeInTheDocument()
    // Tanpa Simpan: isian yang ditinggalkan langsung disimpan.
    await user.tab()

    await waitFor(() => expect(sent).not.toBeNull())
    expect(sent!.body).toMatchObject({
      tugas_id: 'tugas-1', objek: 1, jaminan: 1, tipe_pembayaran: '1',
      nilai_propose_sen: 1_000_000_000, nilai_pengajuan_sen: 1_200_000_000, tipe_resiko: '1', persen_resiko: 100_000,
    })
    await waitFor(() => expect(screen.queryByRole('group', { name: 'Adjustment baru' })).not.toBeInTheDocument())
  })

  it('Save memakai jalur simpan, Back mengirim kembali', async () => {
    stubEstimate()
    mount(<ClaimPage />)
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: 'Save' }))
    await waitFor(() => expect(estimateBody?.url).toBe('/api/registrasi/estimasi/simpan'))

    await user.click(screen.getByRole('button', { name: 'Back' }))
    await waitFor(() => expect(estimateBody?.url).toBe('/api/registrasi/estimasi'))
    expect(estimateBody?.body).toMatchObject({ kembali: true })
  })

  // Download Claim Face Sheet: isian disimpan dulu, lalu PDF diminta untuk objek dan
  // jaminan baris itu. Sesudahnya estimasi terkunci dan estimasi baru harus menunggu CFS.
  it('Download Claim Face Sheet menyimpan, mengunduh, lalu mengunci estimasi', async () => {
    const saved = {
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim,
        objek: [
          {
            ...AT_ESTIMATE.klaim.objek[0]!,
            coverage: [
              {
                ...AT_ESTIMATE.klaim.objek[0]!.coverage[0]!,
                item: [
                  {
                    nama: 'Gudang', deskripsi: '', kelompok: '',
                    estimasi: [{ tipe: '1', mata_uang: 'IDR', tanggal: '2026-09-27', nilai_sen: 800_000_000, kurs_e4: 10_000, nilai_idr_sen: 800_000_000, sudah_cfs: false }],
                  },
                ],
              },
            ],
          },
        ],
      },
    }
    const calls: { url: string; method: string; body: unknown }[] = []
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:cfs', revokeObjectURL: () => undefined }))
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, method: init?.method ?? 'GET', body: init?.body ? JSON.parse(init.body as string) : null })
      if (url.endsWith('/cfs')) {
        return Promise.resolve(
          new Response('%PDF-1.3', {
            status: 200,
            headers: { 'Content-Type': 'application/pdf', 'Content-Disposition': 'attachment; filename="Revisi0.pdf"' },
          }),
        )
      }
      let body: unknown = {}
      if (url === '/api/registrasi/alur') body = ALUR
      else if (url.includes('/pilihan-item')) body = { pilihan: [] }
      else if (url === '/api/registrasi/mata-uang') body = { pilihan: [{ id: 'IDR', nama: 'IDR' }] }
      else if (url.startsWith('/api/registrasi/estimasi')) body = saved
      else if (url.startsWith('/api/registrasi/klaim/')) body = AT_ESTIMATE
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    mount(<ClaimPage />)
    const user = userEvent.setup()

    const button = await screen.findByRole('button', { name: 'Download Claim Face Sheet' })
    expect(button).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'Tambah estimasi' }))
    await user.type(screen.getByLabelText('Nilai Estimasi'), '8.000.000')
    expect(screen.getByRole('button', { name: 'Tambah estimasi' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Download Claim Face Sheet' }))

    expect(await screen.findByText(/Claim Face Sheet diunduh/)).toBeInTheDocument()
    const post = calls.filter((c) => c.method === 'POST').map((c) => c.url)
    expect(post).toEqual(['/api/registrasi/estimasi/simpan', '/api/registrasi/klaim/klaim-1/cfs'])
    expect(calls.find((c) => c.url.endsWith('/cfs'))?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 1 })
    expect(screen.getByText('Sudah CFS')).toBeInTheDocument()
    expect(screen.getByLabelText('Nilai Estimasi')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Tambah estimasi' })).toBeEnabled()
  })

  // Print PLA: aktif hanya untuk polis berkoasuransi yang jaminannya sudah dibuatkan CFS
  // (IsNoCoins || !isCFS), lalu meminta dokumen untuk objek dan jaminan baris itu.
  it('Print PLA aktif setelah CFS pada polis berkoasuransi dan mengunduh dokumennya', async () => {
    const withCoins = (jenis: string) => ({
      ...AT_ESTIMATE,
      klaim: {
        ...AT_ESTIMATE.klaim,
        polis: { ...AT_ESTIMATE.klaim.polis, jenis_koasuransi: jenis, peran_koasuransi: 'LEADER' },
        objek: [
          {
            ...AT_ESTIMATE.klaim.objek[0]!,
            coverage: [
              {
                ...AT_ESTIMATE.klaim.objek[0]!.coverage[0]!,
                item: [
                  {
                    nama: 'Gudang', deskripsi: '', kelompok: '',
                    estimasi: [{ tipe: '1', mata_uang: 'IDR', tanggal: '2026-09-27', nilai_sen: 800_000_000, kurs_e4: 10_000, nilai_idr_sen: 800_000_000, sudah_cfs: true }],
                  },
                ],
              },
            ],
          },
        ],
      },
    })
    let claim = withCoins('0')
    const calls: { url: string; body: unknown }[] = []
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:pla', revokeObjectURL: () => undefined }))
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      if (url.endsWith('/pla/daftar')) {
        calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
        const list = { revisi_cfs: 0, baru_terbit: 1, pla: [{ nomor: 'J261000000000000001', penerima: 'ANGGOTA SATU', tipe: 'COINS', catatan: 'Estimation only.', email: '', tanggal: '2026-09-27' }] }
        return Promise.resolve(new Response(JSON.stringify(list), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      if (url.endsWith('/pla')) {
        calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
        return Promise.resolve(
          new Response('%PDF-1.3', { status: 200, headers: { 'Content-Type': 'application/pdf', 'Content-Disposition': 'attachment; filename="PLACOINSJ26.pdf"' } }),
        )
      }
      let body: unknown = {}
      if (url === '/api/registrasi/alur') body = ALUR
      else if (url.includes('/pilihan-item')) body = { pilihan: [] }
      else if (url === '/api/registrasi/mata-uang') body = { pilihan: [{ id: 'IDR', nama: 'IDR' }] }
      else if (url.startsWith('/api/registrasi/klaim/')) body = claim
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    const first = mount(<ClaimPage />)
    expect(await screen.findByRole('button', { name: 'Print PLA' })).toBeDisabled()
    first.unmount()

    claim = withCoins('2')
    mount(<ClaimPage />)
    const user = userEvent.setup()
    const button = await screen.findByRole('button', { name: 'Print PLA' })
    await waitFor(() => expect(button).toBeEnabled())
    await user.click(button)

    expect(await screen.findByRole('dialog', { name: 'Print PLA' })).toBeInTheDocument()
    expect(await screen.findByText('ANGGOTA SATU')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Print All PLA' }))
    expect(await screen.findByText('PLA diunduh.')).toBeInTheDocument()
    expect(calls).toEqual([
      { url: '/api/registrasi/klaim/klaim-1/pla/daftar', body: { tugas_id: 'tugas-1', objek: 1, jaminan: 1 } },
      { url: '/api/registrasi/klaim/klaim-1/pla', body: { tugas_id: 'tugas-1', objek: 1, jaminan: 1 } },
    ])
  })

  // Bagian atas dari InputEstimasiAdmin_SECT: Status Klaim bernama, PIC Teknis, Log Transfer
  // Klaim, dan keempat tab dengan Estimasi Pembayaran terbuka.
  it('menampilkan bagian atas dan tab seperti layar Pega', async () => {
    stubEstimate()
    mount(<ClaimPage />)

    await screen.findByRole('region', { name: 'Input Estimasi' })
    expect(screen.getByText('Catatan ke PIC Teknis')).toBeInTheDocument()
    expect(screen.getByText('TEKNIK01')).toBeInTheDocument()
    expect(screen.getByText('Data Tidak Ada')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Estimasi Pembayaran' })).toHaveAttribute('aria-selected', 'true')
    for (const label of ['Nama Objek', 'Lokasi Object', 'Nilai Klaim', 'Nilai Adjuster', 'Jaminan', 'Deskripsi Item']) {
      expect(screen.getByText(label)).toBeInTheDocument()
    }
  })

  it('membaca tab Survey, Unggah Dokumen, dan Progress Claim & Komunikasi', async () => {
    stubEstimate()
    mount(<ClaimPage />)
    const user = userEvent.setup()

    await screen.findByRole('region', { name: 'Input Estimasi' })
    await user.click(screen.getByRole('tab', { name: 'Survey' }))
    expect(await screen.findByText('PT ADJUSTER CONTOH')).toBeInTheDocument()
    expect(screen.getByText('SRV-9')).toBeInTheDocument()
    expect(screen.getByText('Loss Adjuster')).toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'Unggah Dokumen' }))
    expect(await screen.findByText('PELAPORAN KLAIM')).toBeInTheDocument()
    expect(screen.getByText('Ya')).toBeInTheDocument()
    expect(screen.getByText('laporan.pdf')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Unggah Dokumen' })).toBeEnabled()
    // Sub-tab InputRegister-sect.xml: SALVAGE tampil selain Travel dan PA.
    const docTabs = screen.getByRole('tablist', { name: 'Kategori dokumen' })
    expect(within(docTabs).getByRole('tab', { name: 'PENDAFTARAN' })).toHaveAttribute('aria-selected', 'true')
    expect(within(docTabs).getByRole('tab', { name: 'SALVAGE' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Lihat dokumen (2)' }))
    expect(screen.getAllByText(/laporan\.pdf/).length).toBeGreaterThan(1)

    await user.click(screen.getByRole('tab', { name: 'Progress Claim & Komunikasi' }))
    expect(await screen.findByText('Auto Create Register')).toBeInTheDocument()
    expect(screen.getByText('POLIS BELUM ADA')).toBeInTheDocument()
    expect(screen.getByText('Mohon laporan survey')).toBeInTheDocument()
    expect(screen.getByText('Sudah dikirim')).toBeInTheDocument()
  })

  it('sub-tab Unggah Dokumen mengikuti lini: PA tanpa SALVAGE, Travel hanya DOKUMEN TRAVEL', async () => {
    const cases: [string, string[], string[]][] = [
      ['002', ['PENDAFTARAN', 'SURVEI'], ['SALVAGE', 'DOKUMEN TRAVEL']],
      ['005', ['DOKUMEN TRAVEL'], ['PENDAFTARAN', 'SALVAGE']],
    ]
    for (const [lini, expectTabs, absent] of cases) {
      stubEstimate({ ...AT_ESTIMATE, klaim: { ...AT_ESTIMATE.klaim, polis: { ...AT_ESTIMATE.klaim.polis, lini } } })
      const view = mount(<ClaimPage />)
      const user = userEvent.setup()
      await user.click(await screen.findByRole('tab', { name: 'Unggah Dokumen' }))
      const docTabs = await screen.findByRole('tablist', { name: 'Kategori dokumen' })
      for (const t of expectTabs) expect(within(docTabs).getByRole('tab', { name: t })).toBeInTheDocument()
      for (const t of absent) expect(within(docTabs).queryByRole('tab', { name: t })).not.toBeInTheDocument()
      view.unmount()
    }
  })

  // Tombol Unggah Dokumen: berkas, jenis dokumen baris itu, dan catatan dikirim multipart.
  it('mengunggah dokumen dari baris checklist', async () => {
    stubEstimate()
    mount(<ClaimPage />)
    const user = userEvent.setup()

    await screen.findByRole('region', { name: 'Input Estimasi' })
    await user.click(screen.getByRole('tab', { name: 'Unggah Dokumen' }))
    await user.click(await screen.findByRole('button', { name: 'Unggah Dokumen' }))

    const unggah = screen.getByRole('button', { name: 'Unggah' })
    // Selalu dapat ditekan, seperti Pega — tanpa berkas ia membuka pemilih berkas.
    expect(unggah).toBeEnabled()
    await user.upload(screen.getByLabelText('Berkas'), new File(['%PDF'], 'lapor.pdf', { type: 'application/pdf' }))
    await user.type(screen.getByLabelText('Catatan'), 'asli')
    await user.click(unggah)

    await waitFor(() => expect(uploads).toHaveLength(1))
    const sent = uploads[0]!
    expect((sent.get('berkas') as File).name).toBe('lapor.pdf')
    expect(sent.get('jenis_dokumen')).toBe(RECORDS.dokumen.kategori[0]!.dokumen[0]!.id)
    expect(sent.get('catatan')).toBe('asli')
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Unggah' })).toBeNull())
  })

  // Lini Fire: Objek dipilih dari item properti polis, dan kelompoknya ikut terkirim.
  // Objek adalah DROPDOWN item polis (Work Owner 2026-10-08): tidak dapat diketik bebas.
  // Memilih item ikut mengisi kelompoknya.
  it('memilih Objek dari item properti polis Fire', async () => {
    itemOptions = [
      { nama: 'BUILDING', kelompok: 'BUILDING(S)', tsi_sen: 0 },
      { nama: 'CONTENTS', kelompok: 'OTHERS', tsi_sen: 0 },
    ]
    stubEstimate()
    mount(<ClaimPage />)
    const user = userEvent.setup()

    const field = await screen.findByRole('combobox', { name: 'Objek' })
    await waitFor(() => expect(screen.getByRole('option', { name: 'CONTENTS' })).toBeInTheDocument())
    expect(field.tagName).toBe('SELECT')
    await user.selectOptions(field, 'CONTENTS')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(estimateBody).not.toBeNull())
    expect(estimateBody?.body).toMatchObject({
      objek: [{ coverage: [{ item: [{ nama: 'CONTENTS', kelompok: 'OTHERS' }] }] }],
    })
  })

  it('Objek tidak dapat diisi bila polis tidak memiliki daftar item', async () => {
    itemOptions = []
    stubEstimate()
    mount(<ClaimPage />)

    const field = await screen.findByRole('combobox', { name: 'Objek' })
    expect(field).toBeDisabled()
    expect(screen.getByRole('option', { name: '— tidak ada pilihan —' })).toBeInTheDocument()
    expect(screen.getByText(/Polis tidak memiliki daftar item/)).toBeInTheDocument()
  })
})
