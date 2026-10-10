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

    // Keduanya SELALU kosong dari server hari ini — kolom asalnya belum ada di T_SURVEYORLIST.
    // Data uji meniru itu apa adanya; mengisinya akan membuat uji lulus atas keadaan yang
    // tidak pernah terjadi. Status ASM dan Appointment No TIDAK termasuk — keduanya terisi.
    appointment_no: 'SRV-0101',
    reference_no: '',

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
    status_asm: 'LEADER',
    jenis_surveyor: '2',
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
    { kunci: 'appointment_no', judul: 'Appointment No', tersedia: true },
    {
      kunci: 'reference_no',
      judul: 'Reference No',
      keterangan: 'BELUM TERSEDIA. Kolom REFNO sudah ada tetapi masih kosong.',
      tersedia: false,
    },
    { kunci: 'claim_no', judul: 'Claim No', tersedia: true },
    { kunci: 'policy_no', judul: 'Policy No', tersedia: true },
    { kunci: 'insured_name', judul: 'Insured Name', tersedia: true },
    { kunci: 'cob', judul: 'COB', tersedia: true },
    { kunci: 'cause_of_loss', judul: 'Cause Of Loss', tersedia: true },
    { kunci: 'location', judul: 'Location', tersedia: true },
    { kunci: 'pic_asm', judul: 'PIC ASM', tersedia: true },
    { kunci: 'pic_loss_adjuster', judul: 'PIC Loss Adjuster', tersedia: true },
    { kunci: 'date_of_loss', judul: 'Date of Loss', tersedia: true },
    {
      kunci: 'aging',
      judul: 'Aging',
      keterangan: 'Dihitung dari tanggal janji survei dicatat.',
      tersedia: true,
    },
    {
      kunci: 'status_asm',
      judul: 'Status ASM',
      keterangan: 'Peran koasuransi ASM: LEADER atau MEMBER.',
      tersedia: true,
    },
  ],

  // Keempat tab pertama BELUM TERSEDIA — kolom penggeraknya belum ada di T_SURVEYORLIST.
  // Data uji meniru itu apa adanya, karena di situlah cacat layar paling mungkin muncul.
  tab: [
    {
      kunci: 'outstanding',
      judul: 'Outstanding',
      tersedia: false,
      alasan_tak_tersedia: 'Kolom ADJUSTERACCEPT sudah ada tetapi seluruh barisnya masih kosong.',
    },
    {
      kunci: 'invoice',
      judul: 'Invoice',
      tersedia: false,
      alasan_tak_tersedia: 'Kolom ADJUSTERACCEPT sudah ada tetapi seluruh barisnya masih kosong.',
    },
    {
      kunci: 'close',
      judul: 'Close',
      tersedia: false,
      alasan_tak_tersedia: 'Kolom PYSTATUSWORK sudah ada tetapi seluruh barisnya masih kosong.',
    },
    {
      kunci: 'all',
      judul: 'ALL',
      tersedia: false,
      alasan_tak_tersedia: 'Kolom ADJUSTERACCEPT sudah ada tetapi seluruh barisnya masih kosong.',
    },
    { kunci: 'belum-dijawab', judul: 'Not answered communication', tersedia: true },
    { kunci: 'belum-dibalas-asm', judul: 'Not replied from ASM', tersedia: true },
    { kunci: 'sudah-dibalas-asm', judul: 'Replied from ASM', tersedia: true },
  ],
  kolom_kpi: [
    { kunci: 'penjadwalan_survey', judul: 'PENJADWALAN SURVEY', tersedia: true },
    { kunci: 'immediate_advice', judul: 'IMMEDIATE ADVICE', tersedia: true },
    { kunci: 'preliminary_advice', judul: 'PRELIMINARY ADVICE', tersedia: true },
    { kunci: 'interim_report', judul: 'INTERIM REPORT', tersedia: true },
    { kunci: 'update_progress', judul: 'UPDATE PROGRESS', tersedia: true },
    { kunci: 'tanggapan_komunikasi', judul: 'TANGGAPAN KOMUNIKASI', tersedia: true },
    { kunci: 'propose_adjustment', judul: 'PROPOSE ADJUSTMENT', tersedia: true },
    { kunci: 'final_report', judul: 'FINAL REPORT', tersedia: true },
    { kunci: 'nilai', judul: 'NILAI', tersedia: true },
  ],

  // Tab bawaan adalah tab TERSEDIA pertama, bukan Outstanding. Server yang memutuskannya
  // (`inboxsurvey.DefaultAvailableTab`), dan data uji meniru keputusannya.
  tab_bawaan: 'belum-dijawab',
  status_survei: ["ALL", "OUTSTANDING", "FINAL"],
  tipe_report: ["DATA SUMMARY", "DATA DETAIL"],
  kuartal: ["1", "2", "3", "4"],
  ukuran_halaman: 25,
  selisih_terencana: ['Tab Close memakai status adjuster Close Case.'],
  keterbatasan: ['Empat kueri tab layar lama hilang dari export.'],
}

const jumlahTabResponse: JumlahTabResponse = {
  portal: 'ASM',
  identitas: identitas(),
  // HANYA tab tersedia yang dihitung server. Keempat tab lain tidak muncul di sini sama
  // sekali — angka nol akan menyatakan "tab ini kosong", padahal yang benar adalah "tab ini
  // belum dapat dihitung".
  tab: [
    { kunci: 'belum-dijawab', total: 4 },
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
    tab: 'belum-dijawab',
    cari: '',
    ...partial,
  }
}

const kpiResponse: KPIResponse = {
  portal: 'ASM',
  identitas: identitas(),
  status_survei: 'FINAL',
  tipe_report: 'DATA SUMMARY',
  kuartal: '',
  tahun: '',
  bentuk: 'per-adjuster',
  kolom_awal: [{ kunci: 'kelompok', judul: 'ADJUSTER', tersedia: true }],
  data: [
    {
      kelompok: 'BUDI',
      status: '',
      kuartal: '',
      bulan: '',
      case_id: '',
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

/** Mengisi kedua dropdown wajib panel KPI. */
async function pilihKPI(status = 'FINAL', tipe = 'DATA SUMMARY') {
  await userEvent.selectOptions(screen.getByLabelText(/Status Survey/), status)
  await userEvent.selectOptions(screen.getByLabelText(/Tipe Report/), tipe)
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
    // Diperiksa SEBELUM /kpi — keduanya berawalan sama, dan membalik urutannya membuat
    // dropdown tahun menerima jawaban ringkasan KPI.
    if (url.includes('/kpi/tahun')) {
      return Promise.resolve(jsonResponse(200, { portal: 'ASM', tahun: ['2026', '2025'] }))
    }
    if (url.includes('/kpi')) {
      if (typeof kpi === 'function') return Promise.resolve(kpi())

      // BENTUK jawabannya mengikuti isian, sama seperti server: begitu kuartal diminta,
      // barisnya berarti TAHUN, bukan adjuster. Mengembalikan satu bentuk untuk setiap
      // permintaan akan membuat uji judul kolom lulus tanpa menguji apa pun.
      const berkuartal = url.includes('kuartal=')
      return Promise.resolve(
        jsonResponse(200, {
          ...kpi,
          bentuk: berkuartal ? 'per-tahun' : 'per-adjuster',
          kolom_awal: [
            berkuartal
              ? { kunci: 'kelompok', judul: 'TAHUN', tersedia: true }
              : { kunci: 'kelompok', judul: 'ADJUSTER', tersedia: true },
          ],
        }),
      )
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
    // BELUM TERSEDIA, dan judulnya menyatakan itu. Tanpa penanda, kolom yang SELALU kosong
    // tidak dapat dibedakan dari kolom yang kebetulan kosong pada halaman ini.
    'Reference No · belum tersedia',

    // Keduanya TERSEDIA — judulnya polos, tanpa penanda. Status ASM sejak 2026-09-29,
    // Appointment No sejak 2026-10-03 setelah terbukti diturunkan dari CASEID.
    'Status ASM',
    'Appointment No',

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
  ]) {
    // getAllByText, bukan getByText: `DataTable` menggambar judul DUA KALI — sebagai `<th>`
    // pada tampilan meja, dan sebagai label sel pada tampilan kartu di layar sempit.
    expect(screen.getAllByText(judul).length).toBeGreaterThan(0)
  }
})

/**
 * Ketujuh tab digambar, masing-masing membawa KEADAANNYA di dalam namanya.
 *
 * Angkanya menyatu dengan nama — `Not answered communication (4)` — bukan digambar sebagai
 * lencana tersendiri. Itu akibat memakai `TabBar` bersama, yang tidak menyediakan slot
 * lencana; menambahkannya berarti mengubah komponen yang sudah dipakai layar lain.
 *
 * Tab yang BELUM TERSEDIA memakai slot yang sama untuk menyatakan keadaannya. Tanpa penanda
 * itu, tab tersebut tampil polos dengan daftar kosong — dan daftar kosong terbaca sebagai
 * "tidak ada pekerjaan", yang tidak pernah dilaporkan siapa pun sebagai kerusakan.
 *
 * Yang diuji tetap: ketujuh tab ADA, dan angkanya datang dari rute `/jumlah-tab` yang
 * TERPISAH — bukan dari jawaban daftar.
 */
it('menggambar ketujuh tab beserta jumlah atau keadaannya', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  for (const nama of [
    'Outstanding (belum tersedia)',
    'Invoice (belum tersedia)',
    'Close (belum tersedia)',
    'ALL (belum tersedia)',
    'Not answered communication (4)',
    'Not replied from ASM (1)',
    'Replied from ASM (1)',
  ]) {
    expect(screen.getByRole('tab', { name: nama })).toBeInTheDocument()
  }
})

/**
 * Tab yang belum tersedia MENGGANTI tabelnya dengan sebabnya, bukan menampilkannya kosong.
 *
 * Ini perbedaan yang menentukan. Daftar kosong terbaca sebagai "tidak ada pekerjaan untuk
 * saya", dan pembacaan itu salah — yang benar adalah kolom penggeraknya belum ada di basis
 * data. Yang pertama tidak pernah dilaporkan siapa pun; yang kedua akan.
 *
 * Tabnya TETAP dapat dipilih, dan itu disengaja: tab yang dimatikan tanpa penjelasan sama
 * membingungkannya dengan tab kosong.
 */
it('menjelaskan sebabnya saat tab yang belum tersedia dibuka', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  await userEvent.click(screen.getByRole('tab', { name: /^Outstanding/ }))

  expect(await screen.findByText(/Tab Outstanding belum dapat ditampilkan/)).toBeInTheDocument()
  // Nama kolomnya WAJIB terbaca apa adanya — `ADJUSTERACCEPT`, bukan `ADJUSTERACCEPT_1`.
  // Akhiran `_1` milik tabel datar Pega, dan pernah membuat kolom yang SUDAH ditambahkan
  // terbaca sebagai belum ada oleh orang yang menambahkannya.
  expect(
    screen.getByText(/ADJUSTERACCEPT sudah ada tetapi seluruh barisnya masih kosong/),
  ).toBeInTheDocument()

  // Tabelnya TIDAK digambar — bukan digambar kosong.
  expect(screen.queryByText('PNCN.26.0101')).not.toBeInTheDocument()
})

/**
 * Kolom yang belum tersedia menyatakan keadaannya di SELNYA juga, bukan hanya di judulnya.
 *
 * Em dash pada kolom yang SELALU kosong tidak dapat dibedakan dari em dash pada baris yang
 * kebetulan kosong, dan sebab keduanya berbeda jauh: yang satu menunggu Tim Pega, yang lain
 * menunggu petugas mengisi.
 */
it('menandai sel kolom yang belum tersedia', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  // SATU kolom belum tersedia — Reference No — dan seluruh selnya bertanda.
  expect(screen.getAllByText('belum tersedia').length).toBeGreaterThanOrEqual(1)

  // Status ASM TIDAK bertanda — kolomnya terisi, dan nilainya digambar apa adanya.
  expect(screen.getAllByText('LEADER').length).toBeGreaterThan(0)

  // Appointment No juga TIDAK bertanda, dan nomornya benar-benar tergambar.
  expect(screen.getByText('SRV-0101')).toBeInTheDocument()
})

/**
 * Kegagalan penghitung tab HARUS terlihat pengguna.
 *
 * Ini saudara kembar dari cacat `/keterangan` di bawah, dan ia SEMPAT TERLEWAT: perbaikan
 * pertama hanya menutup satu dari dua. Tanpa pesan ini, tab tampil tanpa angka dan pengguna
 * tidak dapat membedakan "tab ini memang kosong" dari "angkanya gagal diambil".
 *
 * Bedanya dengan `/keterangan`: kegagalan di sini TIDAK menutup layar. Daftarnya tetap
 * terbaca, sehingga yang benar adalah pemberitahuan di dalam layar — bukan pengganti seluruh
 * layar.
 */
it('memberi tahu pengguna saat jumlah per tab gagal diambil', async () => {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    if (url.includes('/keterangan')) {
      return Promise.resolve(jsonResponse(200, keteranganResponse))
    }
    if (url.includes('/jumlah-tab')) {
      return Promise.resolve(jsonResponse(500, { kode: 'galat_internal', pesan: 'rusak' }))
    }
    return Promise.resolve(jsonResponse(200, daftarResponse()))
  })
  renderPage()

  expect(
    await screen.findByText(/Jumlah pekerjaan per tab tidak dapat diambil/),
  ).toBeInTheDocument()

  // Daftarnya sendiri TETAP terbaca — kegagalan penghitung tidak boleh menutup layar.
  //
  // `findByText`, bukan `getByText`: penghitung gagal lebih cepat daripada daftar selesai
  // dimuat, sehingga pemeriksaan serentak di sini akan menguji urutan kedatangan jaringan,
  // bukan perilaku layarnya.
  expect(await screen.findByText('PNCN.26.0101')).toBeInTheDocument()

  // Dan tabnya tetap dapat dipakai, hanya tanpa angka.
  expect(
    screen.getByRole('tab', { name: 'Not answered communication' }),
  ).toBeInTheDocument()
})

/**
 * Tab bawaan datang dari server, bukan ditulis tetap di layar.
 *
 * Yang menentukan tab mana yang terbuka pertama kali adalah keputusan yang tercatat di domain
 * (`inboxsurvey.DefaultAvailableTab`). Menuliskannya juga di layar akan membuat keduanya dapat
 * berbeda tanpa ada yang menyadarinya.
 *
 * Perhatikan tab bawaannya BUKAN Outstanding. Bawaan layar lama adalah Outstanding, dan
 * Outstanding termasuk yang belum dapat dihitung — membuka layar di sana akan membuat kesan
 * pertama setiap pengguna berupa layar tanpa isi, dan kesan itu bertahan meski tab lain
 * berisi. Server yang memutuskannya, layar hanya mengikutinya.
 */
it('membuka tab bawaan dari keterangan server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await waitFor(() => expect(urlDaftar()).toContain('tab=belum-dijawab'))
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
  await userEvent.type(kotak, 'PNCN.26.0101')
  await waitFor(() => expect(urlDaftar()).toContain('cari=PNCN'))

  // Berpindah ke tab TERSEDIA yang lain — tab yang belum tersedia tidak menjalankan
  // permintaan daftar sama sekali, sehingga tidak dapat membuktikan pencariannya dikosongkan.
  await userEvent.click(screen.getByRole('tab', { name: /^Replied from ASM/ }))

  await waitFor(() => expect(urlDaftar()).toContain('tab=sudah-dibalas-asm'))
  expect(urlDaftar()).not.toContain('cari=')
})

/**
 * Aging `null` dan `0` digambar BERBEDA, dan itu inti kolomnya.
 *
 * `null` berarti tanggal janji surveinya tidak ada, sehingga umurnya tidak dapat dihitung.
 * Menggambar keduanya sama akan menampilkan "0 hari" pada baris yang sebenarnya tidak punya
 * angka — angka yang terlihat sah dan salah.
 */
it('membedakan Aging yang tidak dapat dihitung dari nol hari', async () => {
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
  expect(screen.getByText('tidak dapat dihitung')).toBeInTheDocument()
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
it('tidak menembak KPI sampai tombol Cari ditekan', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  expect(calls.some((c) => c.url.includes('/kpi?'))).toBe(false)

  // Membuka tabnya saja TIDAK cukup. Layar lama tidak memuat apa pun sampai Cari ditekan,
  // dan memuat sendiri berarti menjalankan laporan yang belum diminta.
  // Membuka tabnya menembak /kpi/tahun — dropdown Tahun Kuartal harus terisi SEBELUM
  // pengguna memilih apa pun. Yang TIDAK boleh ditembak adalah ringkasannya sendiri.
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))
  await screen.findByLabelText(/Status Survey/)
  expect(calls.some((c) => c.url.includes('/kpi?'))).toBe(false)
  await waitFor(() => expect(calls.some((c) => c.url.includes('/kpi/tahun'))).toBe(true))

  await pilihKPI()
  await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

  await waitFor(() => expect(calls.some((c) => c.url.includes('/kpi?'))).toBe(true))
})

/**
 * Tombol Cari mati sampai KEDUA isian wajib terisi.
 *
 * Keduanya ditandai bintang merah di layar lama. Membiarkannya hidup akan membuat pengguna
 * menekan Cari lalu menerima penolakan — satu perjalanan yang sudah dapat dicegah di layar.
 */
it('mematikan Cari dan Export Data sampai kedua isian wajib terisi', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))

  expect(screen.getByRole('button', { name: 'Cari' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Export Data' })).toBeDisabled()

  await userEvent.selectOptions(screen.getByLabelText(/Status Survey/), 'FINAL')
  expect(screen.getByRole('button', { name: 'Cari' })).toBeDisabled()

  await userEvent.selectOptions(screen.getByLabelText(/Tipe Report/), 'DATA SUMMARY')
  expect(screen.getByRole('button', { name: 'Cari' })).toBeEnabled()
  expect(screen.getByRole('button', { name: 'Export Data' })).toBeEnabled()
})

/**
 * Kuartal dan Tahun Kuartal hanya muncul pada Status Survey ALL atau FINAL.
 *
 * Meniru `pyVisibleWhen` panel KPI apa adanya — itu sebabnya tangkapan layar Pega hanya
 * memperlihatkan DUA kendali ketika Status Survey masih `--Pilih--`.
 */
it('menampilkan Kuartal hanya pada Status Survey ALL atau FINAL', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))

  expect(screen.queryByLabelText('Kuartal')).not.toBeInTheDocument()

  await userEvent.selectOptions(screen.getByLabelText(/Status Survey/), 'OUTSTANDING')
  expect(screen.queryByLabelText('Kuartal')).not.toBeInTheDocument()

  await userEvent.selectOptions(screen.getByLabelText(/Status Survey/), 'FINAL')
  expect(screen.getByLabelText('Kuartal')).toBeInTheDocument()
  expect(screen.getByLabelText('Tahun Kuartal')).toBeInTheDocument()
})

it('menggambar kesembilan judul angka KPI dari keterangan server', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))
  await pilihKPI()
  await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

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
 * Kolom pertama tabel KPI berganti ARTI menurut ISIAN, bukan menurut tab.
 *
 * Begitu Kuartal atau Tahun Kuartal diisi, pengelompokan berpindah ke per TAHUN. Judul yang
 * tidak ikut berganti akan membuat kolom berisi "2026" terbaca sebagai nama orang yang
 * kebetulan berupa angka.
 */
it('mengganti judul kolom kelompok ketika dikelompokkan per tahun', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')
  await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))

  await pilihKPI()
  await userEvent.click(screen.getByRole('button', { name: 'Cari' }))
  await screen.findByText('85.00')
  expect(screen.getAllByText('ADJUSTER').length).toBeGreaterThan(0)

  await userEvent.selectOptions(screen.getByLabelText('Kuartal'), '3')
  await userEvent.click(screen.getByRole('button', { name: 'Cari' }))

  await waitFor(() => expect(screen.getAllByText('TAHUN').length).toBeGreaterThan(0))
})

/**
 * Catatan migrasi TIDAK ditampilkan di layar (Work Owner, 2026-10-07).
 *
 * Server masih mengirim `selisih_terencana` dan `keterbatasan` — bentuk jawabannya sengaja
 * dibiarkan sama dengan ~30 layar lain — dan uji ini memastikan keduanya tetap TIDAK dirender
 * meski datanya ada. Itu sebabnya respons tiruan di berkas ini tetap memuat keduanya: kalau
 * datanya ikut dikosongkan, uji ini akan lulus karena alasan yang salah.
 *
 * Surveyor membuka layar ini untuk mengerjakan survei, bukan untuk membaca apa yang berbeda
 * dari Pega. Isinya hidup di `docs/catatan-pengembangan.md`.
 */
it('tidak menampilkan catatan perbedaan terhadap Pega meski server mengirimnya', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  expect(screen.queryByText(/Tab Close memakai status adjuster/)).not.toBeInTheDocument()
  expect(screen.queryByText(/Empat kueri tab layar lama hilang/)).not.toBeInTheDocument()
  expect(screen.queryByText('Perbedaan yang disengaja terhadap layar lama')).not.toBeInTheDocument()
  expect(screen.queryByText('Yang perlu diketahui')).not.toBeInTheDocument()
})

/**
 * Keterangan OPERASIONAL tetap tampil, dan itu pembedanya.
 *
 * Yang dihapus adalah catatan MIGRASI. Sebab sebuah tab belum dapat dibuka bukan catatan
 * migrasi melainkan jawaban atas pertanyaan pemakai — tanpa itu, tab yang mati terbaca sebagai
 * kerusakan diam.
 */
it('tetap menyebut sebab sebuah tab belum dapat dibuka', async () => {
  stubFetch(daftarResponse())
  renderPage()

  await screen.findByText('PNCN.26.0101')

  expect(screen.getByRole('tab', { name: /Close \(belum tersedia\)/ })).toBeInTheDocument()
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
