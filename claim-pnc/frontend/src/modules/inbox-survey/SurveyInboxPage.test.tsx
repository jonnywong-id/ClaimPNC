import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SurveyInboxPage } from './SurveyInboxPage'
import type {
  DaftarResponse,
  IdentitasSurveyor,
  JumlahTabResponse,
  KPIResponse,
  KeteranganResponse,
  TugasSurvei,
} from './types'

/**
 * Data uji seluruhnya KARANGAN.
 *
 * `D-69` melarang data nasabah nyata ditulis di berkas yang di-commit, dan larangan itu
 * berlaku untuk data uji sama seperti untuk dokumen.
 */
function tugas(partial: Partial<TugasSurvei> = {}): TugasSurvei {
  return {
    survei_id: 'ASM-FW-GCNMFW-WORK SRV-0001',
    klaim_id: 'ASM-FW-GCNMFW-WORK PNCN.26.0101',
    index_survei: '1',
    appointment_no: 'APP-0001',
    reference_no: 'REF-0001',
    claim_no: 'PNCN.26.0101',
    policy_no: '26.004.2026.00101',
    insured_name: 'Rangga Contoh',
    cob: 'Marine Cargo',
    cause_of_loss: 'Kerusakan Muatan',
    location: 'Pelabuhan Tanjung Priok',
    pic_asm: 'PICTEKNIK1',
    pic_loss_adjuster: 'BUDI',
    date_of_loss: '2026-08-20',
    aging: 8,
    status_asm: 'Survey Scheduled',
    jenis_surveyor: '2',
    status_survei: '0',
    ...partial,
  }
}

function identitas(partial: Partial<IdentitasSurveyor> = {}): IdentitasSurveyor {
  return {
    login: 'ADJLEADER',
    nama: 'BUDI',
    leader: false,
    cakupan: ['BUDI'],
    jumlah_tim: 0,
    ...partial,
  }
}

/**
 * Keterangan layar meniru jawaban server apa adanya, termasuk urutan kolom dan tabnya.
 *
 * Ketiga belas judul kolom dan ketujuh judul tab diambil dari
 * `Section/InboxSurvey_section-Section.xml`.
 */
const keteranganResponse: KeteranganResponse = {
  portal: 'ASM',
  kolom: [
    {
      kunci: 'appointment_no',
      judul: 'Appointment No',
      keterangan: 'Pemetaannya belum dikonfirmasi DBA.',
    },
    { kunci: 'reference_no', judul: 'Reference No' },
    { kunci: 'claim_no', judul: 'Claim No' },
    { kunci: 'policy_no', judul: 'Policy No' },
    { kunci: 'insured_name', judul: 'Insured Name' },
    { kunci: 'cob', judul: 'COB' },
    { kunci: 'cause_of_loss', judul: 'Cause Of Loss' },
    { kunci: 'location', judul: 'Location' },
    { kunci: 'pic_asm', judul: 'PIC ASM' },
    { kunci: 'pic_loss_adjuster', judul: 'PIC Loss Adjuster' },
    { kunci: 'date_of_loss', judul: 'Date of Loss' },
    { kunci: 'aging', judul: 'Aging', keterangan: 'Dibaca, bukan dihitung.' },
    { kunci: 'status_asm', judul: 'Status ASM' },
  ],
  tab: [
    { kunci: 'outstanding', judul: 'Outstanding' },
    { kunci: 'invoice', judul: 'Invoice' },
    { kunci: 'close', judul: 'Close' },
    { kunci: 'all', judul: 'ALL' },
    { kunci: 'belum-dijawab', judul: 'Not answered communication' },
    { kunci: 'belum-dibalas-asm', judul: 'Not replied from ASM' },
    { kunci: 'sudah-dibalas-asm', judul: 'Replied from ASM' },
  ],
  kolom_kpi: [
    { kunci: 'penjadwalan_survey', judul: 'PENJADWALAN SURVEY' },
    { kunci: 'immediate_advice', judul: 'IMMEDIATE ADVICE' },
    { kunci: 'preliminary_advice', judul: 'PRELIMINARY ADVICE' },
    { kunci: 'interim_report', judul: 'INTERIM REPORT' },
    { kunci: 'update_progress', judul: 'UPDATE PROGRESS' },
    { kunci: 'tanggapan_komunikasi', judul: 'TANGGAPAN KOMUNIKASI' },
    { kunci: 'propose_adjustment', judul: 'PROPOSE ADJUSTMENT' },
    { kunci: 'final_report', judul: 'FINAL REPORT' },
    { kunci: 'nilai', judul: 'NILAI' },
  ],
  tab_bawaan: 'outstanding',
  jenis_kpi: ['outstanding', 'final', 'kuartal'],
  ukuran_halaman: 25,
  selisih_terencana: ['Tab Close memakai status adjuster Close Case.'],
  keterbatasan: ['Empat kueri tab layar lama hilang dari export.'],
}

const jumlahTabResponse: JumlahTabResponse = {
  portal: 'ASM',
  identitas: identitas(),
  tab: [
    { kunci: 'outstanding', total: 3 },
    { kunci: 'invoice', total: 1 },
    { kunci: 'close', total: 2 },
    { kunci: 'all', total: 6 },
    { kunci: 'belum-dijawab', total: 1 },
    { kunci: 'belum-dibalas-asm', total: 1 },
    { kunci: 'sudah-dibalas-asm', total: 1 },
  ],
}

function daftarResponse(partial: Partial<DaftarResponse> = {}): DaftarResponse {
  return {
    portal: 'ASM',
    identitas: identitas(),
    data: [tugas()],
    total: 1,
    lewati: 0,
    batas: 25,
    tab: 'outstanding',
    cari: '',
    ...partial,
  }
}

const kpiResponse: KPIResponse = {
  portal: 'ASM',
  identitas: identitas(),
  jenis: 'outstanding',
  kategori: '',
  tahun: '',
  data: [
    {
      kelompok: 'BUDI',
      penjadwalan_survey: 85,
      immediate_advice: 75,
      preliminary_advice: 80,
      interim_report: 70,
      update_progress: 65,
      tanggapan_komunikasi: 90,
      propose_adjustment: 83,
      final_report: 87,
      nilai: 81,
    },
  ],
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
 * Menjawab KEEMPAT GET layar ini sekaligus.
 *
 * Urutan pemeriksaannya penting: `/keterangan`, `/jumlah-tab`, dan `/kpi` diperiksa LEBIH
 * DULU, karena ketiganya berawalan jalur yang sama dengan daftar. Membalik urutannya akan
 * membuat seluruh permintaan dijawab dengan badan daftar — dan layarnya tetap terisi,
 * sehingga kekeliruannya tidak akan terlihat.
 */
function stubFetch(
  daftar: DaftarResponse | (() => Response),
  kpi: KPIResponse | (() => Response) = kpiResponse,
) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    if (url.includes('/keterangan')) {
      return Promise.resolve(jsonResponse(200, keteranganResponse))
    }
    if (url.includes('/jumlah-tab')) {
      return Promise.resolve(jsonResponse(200, jumlahTabResponse))
    }
    if (url.includes('/kpi')) {
      return Promise.resolve(typeof kpi === 'function' ? kpi() : jsonResponse(200, kpi))
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
        <SurveyInboxPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** urlDaftar mengambil permintaan daftar TERAKHIR yang dikirim layar. */
function urlDaftar(): string {
  const daftar = calls.filter(
    (c) =>
      !c.url.includes('/keterangan') &&
      !c.url.includes('/jumlah-tab') &&
      !c.url.includes('/kpi'),
  )
  return daftar[daftar.length - 1]?.url ?? ''
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-09-29T12:00:00Z',
  })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/**
 * Judul kolom mengikuti section apa adanya (`D-13`), termasuk bahasa Inggrisnya.
 *
 * Yang diuji bukan sekadar bahwa judulnya benar, melainkan bahwa judulnya datang dari
 * SERVER: bila layar mengetiknya sendiri, uji ini tetap lulus sekarang tetapi akan menyimpang
 * diam-diam begitu bacaan export dikoreksi di backend.
 */
it('menggambar ketiga belas judul kolom dari keterangan server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  for (const judul of [
    'Appointment No',
    'Reference No',
    'Claim No',
    'Policy No',
    'Insured Name',
    'COB',
    'Cause Of Loss',
    'Location',
    'PIC ASM',
    'PIC Loss Adjuster',
    'Date of Loss',
    'Aging',
    'Status ASM',
  ]) {
    // getAllByText, bukan getByText: `DataTable` menggambar judul DUA KALI — sebagai `<th>`
    // pada tampilan meja, dan sebagai label sel pada tampilan kartu di layar sempit.
    expect(screen.getAllByText(judul).length).toBeGreaterThan(0)
  }
})

it('menggambar ketujuh tab beserta jumlahnya', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  for (const judul of [
    'Outstanding',
    'Invoice',
    'Close',
    'ALL',
    'Not answered communication',
    'Not replied from ASM',
    'Replied from ASM',
  ]) {
    expect(screen.getAllByText(judul).length).toBeGreaterThan(0)
  }

  // Angka pada bilah tab datang dari rute TERSENDIRI, bukan dari jawaban daftar.
  expect(screen.getByText('6')).toBeInTheDocument()
})

/**
 * Tab bawaan datang dari server, bukan ditulis tetap di layar.
 *
 * Yang menentukan tab mana yang terbuka pertama kali adalah keputusan yang tercatat di domain
 * (`inboxsurvey.DefaultTab`). Menuliskannya juga di layar akan membuat keduanya dapat berbeda
 * tanpa ada yang menyadarinya.
 */
it('membuka tab bawaan dari keterangan server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await waitFor(() => expect(urlDaftar()).toContain('tab=outstanding'))
})

/**
 * Berpindah tab mengosongkan pencarian DAN kembali ke halaman pertama.
 *
 * Kata kunci yang cocok di satu tab hampir selalu tidak cocok di tab lain, dan hasil kosong
 * sesudah berpindah tab terbaca sebagai "tab itu memang kosong" — bukan sebagai "pencarian
 * Anda masih menyala".
 */
it('mengosongkan pencarian saat berpindah tab', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  const kotak = screen.getByLabelText(/Cari Claim No/i)
  await userEvent.type(kotak, 'REF-0001')
  await waitFor(() => expect(urlDaftar()).toContain('cari=REF-0001'))

  await userEvent.click(screen.getByRole('tab', { name: /^Close/ }))

  await waitFor(() => expect(urlDaftar()).toContain('tab=close'))
  expect(urlDaftar()).not.toContain('cari=')
})

/**
 * Aging `null` dan `0` digambar BERBEDA, dan itu inti kolomnya.
 *
 * Kolom `AGING` boleh kosong. Menggambar keduanya sama akan menampilkan "0 hari" pada baris
 * yang sebenarnya belum pernah dihitung — angka yang terlihat sah dan salah.
 */
it('membedakan Aging yang belum dihitung dari nol hari', async () => {
  stubFetch(
    daftarResponse({
      data: [
        tugas({ claim_no: 'PNCN.26.0101', aging: 0 }),
        tugas({
          survei_id: 'ASM-FW-GCNMFW-WORK SRV-0002',
          claim_no: 'PNCN.26.0102',
          aging: null,
        }),
      ],
      total: 2,
    }),
  )
  renderPage()

  await screen.findByText('PNCN.26.0102')

  expect(screen.getByText('0 hari')).toBeInTheDocument()
  expect(screen.getByText('belum dihitung')).toBeInTheDocument()
})

/**
 * Cakupan tim ditampilkan HANYA bila pemanggil seorang leader.
 *
 * Tanpa keterangan ini, seorang leader yang melihat baris atas nama orang lain tidak punya
 * cara menjelaskan kenapa — dan yang pertama kali terpikir adalah "layarnya bocor".
 */
it('menyebutkan cakupan tim bagi seorang leader', async () => {
  stubFetch(
    daftarResponse({
      identitas: identitas({
        leader: true,
        cakupan: ['BUDI', 'SITI RAHAYU'],
        jumlah_tim: 1,
      }),
    }),
  )
  renderPage()

  await screen.findByText('PNCN.26.0101')

  expect(screen.getByText(/termasuk pekerjaan 1 anggota tim Anda/)).toBeInTheDocument()
  expect(screen.getByText(/SITI RAHAYU/)).toBeInTheDocument()
})

it('tidak menyebutkan cakupan tim bagi surveyor tanpa anggota', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  expect(screen.queryByText(/anggota tim Anda/)).not.toBeInTheDocument()
})

/**
 * Tab KPI TIDAK ditembak sebelum dibuka.
 *
 * Ia membaca tabel LAIN yang diisi procedure terpisah, dan sebagian lingkungan belum
 * memilikinya. Menembaknya saat pengguna masih di tab INBOX berarti satu galat yang tidak
 * pernah dilihat siapa pun — tetapi tetap tercatat di log.
 */
it('tidak menembak KPI sebelum tabnya dibuka', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  expect(calls.some((c) => c.url.includes('/kpi'))).toBe(false)

  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))

  await waitFor(() => expect(calls.some((c) => c.url.includes('/kpi'))).toBe(true))
})

it('menggambar kesembilan judul angka KPI dari keterangan server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))

  await screen.findByText('85.00')

  for (const judul of [
    'PENJADWALAN SURVEY',
    'IMMEDIATE ADVICE',
    'PRELIMINARY ADVICE',
    'INTERIM REPORT',
    'UPDATE PROGRESS',
    'TANGGAPAN KOMUNIKASI',
    'PROPOSE ADJUSTMENT',
    'FINAL REPORT',
    'NILAI',
  ]) {
    expect(screen.getAllByText(judul).length).toBeGreaterThan(0)
  }
})

/**
 * Kolom pertama tabel KPI berganti ARTI menurut jenis ringkasannya.
 *
 * Nama adjuster pada dua yang pertama, TAHUN pada yang ketiga. Judul yang tidak ikut berganti
 * akan membuat kolom tahun terbaca sebagai nama orang yang kebetulan berupa angka.
 */
it('mengganti judul kolom kelompok pada ringkasan per tahun', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))

  await screen.findByText('85.00')
  expect(screen.getAllByText('ADJUSTER').length).toBeGreaterThan(0)

  await userEvent.click(screen.getByRole('tab', { name: 'Data Final per Tahun' }))

  await waitFor(() => expect(screen.getAllByText('TAHUN').length).toBeGreaterThan(0))
})

/**
 * Selisih terencana dan keterbatasan datang dari SERVER.
 *
 * Keduanya harus hilang dengan sendirinya begitu penghalangnya hilang — tanpa menyentuh satu
 * baris pun di layar.
 */
it('menampilkan selisih terencana dan keterbatasan dari server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  expect(screen.getByText(/Tab Close memakai status adjuster/)).toBeInTheDocument()
  expect(screen.getByText(/Empat kueri tab layar lama hilang/)).toBeInTheDocument()
})

/**
 * Galat 403 "bukan surveyor" ditampilkan apa adanya, bukan sebagai tabel kosong.
 *
 * Inilah perbedaan yang paling mudah hilang: daftar kosong terbaca sebagai "tidak ada
 * pekerjaan hari ini", dan tidak pernah dilaporkan siapa pun sebagai kerusakan.
 */
it('menampilkan alasan saat pemanggil bukan surveyor', async () => {
  stubFetch(() =>
    jsonResponse(403, {
      kode: 'bukan_surveyor',
      pesan:
        'Akun Anda belum terdaftar pada Master Login Surveyor, sehingga antrean survei ' +
        'belum dapat ditampilkan.',
    }),
  )
  renderPage()

  expect(
    await screen.findByText(/belum terdaftar pada Master Login Surveyor/),
  ).toBeInTheDocument()
})

/**
 * Tanpa portal, layar TIDAK menembak apa pun.
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia — dan pesan
 * yang dilihat pengguna menjadi pesan galat, bukan ajakan memilih entitas.
 */
it('meminta pengguna memilih entitas sebelum menembak apa pun', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(daftarResponse())
  renderPage()

  expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})

/**
 * Kegagalan keterangan layar HARUS terlihat pengguna.
 *
 * # Kenapa uji ini ada, dan apa yang ditemukannya
 *
 * Tanpa keterangan, `tab` tidak pernah terisi; tanpa `tab`, permintaan daftar tidak pernah
 * dijalankan; dan permintaan yang tidak pernah dijalankan berstatus "menunggu" SELAMANYA.
 * Akibatnya tabel menampilkan kerangka pemuatan yang tidak pernah selesai, tanpa satu pun
 * pesan.
 *
 * Cacat itu nyata dan ditemukan justru oleh uji ini — bukan oleh pembacaan kode. Rantai
 * sebabnya melewati tiga berkas (hook, efek tab bawaan, prop isLoading), dan tidak satu pun
 * di antaranya terlihat salah bila dibaca sendiri-sendiri.
 */
it('memberi tahu pengguna saat keterangan layar gagal dimuat', async () => {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    return Promise.resolve(
      jsonResponse(500, { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' }),
    )
  })
  renderPage()

  expect(await screen.findByText('Layar tidak dapat disiapkan')).toBeInTheDocument()
  expect(screen.getByText('Terjadi kesalahan pada sistem.')).toBeInTheDocument()
})
