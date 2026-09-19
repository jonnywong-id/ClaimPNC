import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { Rute } from '@/app/App'
import { gunakanSesi } from '@/app/sesi'

const MASTER_PATH = '/api/master/ambang-komite'
const INTEGRITY_PATH = '/api/master/ambang-komite/integritas'
const PENJENJANGAN_PATH = '/api/komite/penjenjangan'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Empat baris contoh, diturunkan dari `Database/emailkomite.csv`.
 *
 * Sengaja tidak seluruh 30 baris: yang diuji di sini adalah perilaku layar, bukan aturan
 * penjenjangannya — kebenaran aturan sudah dibuktikan di sisi Go, tempat perhitungannya
 * terjadi.
 *
 * Alamat surel TIDAK ada di sini, dan memang tidak pernah dikirim server. `D-67`
 * menetapkan alamat pribadi pada master lama tidak dibawa ke sistem baru.
 */
const AMBANG = [
  {
    id: '7',
    nama: 'ELLENSUPRIYATI',
    operator_id: 'ELLENSUPRIYATI',
    lini: 'NONMBU',
    jenis_komite: '1',
    batas_bawah: '0.00',
    batas_atas: '50000000.00',
    jenjang: 1,
    aktif: true,
    untuk_adjustment: true,
    untuk_registrasi: false,
    untuk_penolakan: true,
    sedang_absen: false,
    jenjang_persetujuan: true,
  },
  {
    id: '1',
    nama: 'INDRA',
    operator_id: 'INDRAGUNAWAN',
    lini: 'NONMBU',
    jenis_komite: '1',
    batas_bawah: '50000001.00',
    batas_atas: '100000000.00',
    jenjang: 1,
    aktif: true,
    untuk_adjustment: true,
    untuk_registrasi: false,
    untuk_penolakan: false,
    sedang_absen: false,
    jenjang_persetujuan: true,
  },
  // Baris ber-DEGREE 0 yang hanya penerima pemberitahuan registrasi. Ia dibawa supaya
  // layar terbukti menjelaskan KENAPA ia tidak ikut menyetujui.
  {
    id: '9',
    nama: 'KLAIMNONMBU',
    operator_id: '',
    lini: 'NONMBU',
    jenis_komite: '',
    batas_bawah: '0.00',
    batas_atas: '0.00',
    jenjang: 0,
    aktif: true,
    untuk_adjustment: false,
    untuk_registrasi: true,
    untuk_penolakan: false,
    sedang_absen: false,
    jenjang_persetujuan: false,
  },
  {
    id: '41',
    nama: 'Dr. Wahyu',
    operator_id: 'WAHYUKRISTANTI',
    lini: 'PA',
    jenis_komite: '2',
    batas_bawah: '0.00',
    batas_atas: '10000000.00',
    jenjang: 1,
    aktif: true,
    untuk_adjustment: true,
    untuk_registrasi: false,
    untuk_penolakan: true,
    sedang_absen: false,
    jenjang_persetujuan: true,
  },
]

const MASTER_RESPONSE = {
  ambang: AMBANG,
  total: AMBANG.length,
  total_jenjang: 3,
  lini: ['NONMBU', 'PA'],
  kebijakan_pita: [
    { lini: 'NONMBU', batas: '100000000.00', pita_bawah: '1', pita_atas: '2' },
  ],
  mode: 'kumulatif',
}

const INTEGRITY_RESPONSE = {
  temuan: [
    {
      tingkat: 'peringatan',
      jenis: 'atap_tangga',
      lini: 'PA',
      pesan:
        'Tangga berhenti di 200000000.00. Klaim di atas nilai itu tetap mendapat ' +
        'seluruh penyetuju karena aturannya kumulatif.',
      id_ambang: ['44'],
    },
  ],
  jumlah_cacat: 0,
  jumlah_peringatan: 1,
}

const PENJENJANGAN_RESPONSE = {
  nilai: '80000000.00',
  lini: 'NONMBU',
  mode: 'kumulatif',
  berpita_nilai: true,
  pita: '1',
  penyetuju: [
    {
      urutan: 1,
      jenjang: 1,
      nama: 'ELLENSUPRIYATI',
      operator_id: 'ELLENSUPRIYATI',
      batas_bawah: '0.00',
      sedang_absen: false,
      id_ambang: '7',
    },
    {
      urutan: 2,
      jenjang: 1,
      nama: 'INDRA',
      operator_id: 'INDRAGUNAWAN',
      batas_bawah: '50000001.00',
      sedang_absen: false,
      id_ambang: '1',
    },
  ],
  jumlah_jenjang: 2,
  tanpa_penyetuju: false,
  urutan_tidak_pasti: true,
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(extra?: (url: string) => Response | undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))

    const custom = extra?.(url)
    if (custom) return Promise.resolve(custom)

    if (url === INTEGRITY_PATH) return Promise.resolve(jsonResponse(200, INTEGRITY_RESPONSE))
    if (url === MASTER_PATH) return Promise.resolve(jsonResponse(200, MASTER_RESPONSE))
    if (url.startsWith(PENJENJANGAN_PATH)) {
      return Promise.resolve(jsonResponse(200, PENJENJANGAN_RESPONSE))
    }
    return Promise.resolve(jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'tidak ada' }))
  })
}

function renderAt(path: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Rute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  gunakanSesi.setState({
    token: 'token-uji',
    pengguna: SAMPLE_PROFILE,
    berlakuSampai: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  gunakanSesi.getState().bersihkan()
})

describe('layar Ambang Komite', () => {
  it('menampilkan tangga ambang beserta nilai yang sudah diformat', async () => {
    stubFetch()
    renderAt('/master/ambang-komite')

    expect(await screen.findByText('Dr. Wahyu')).toBeInTheDocument()

    // Nama dan Operator ID ditampilkan terpisah: yang pertama untuk dibaca manusia, yang
    // kedua untuk ditelusuri ke data klaim.
    expect(screen.getByText('INDRA')).toBeInTheDocument()
    expect(screen.getByText('INDRAGUNAWAN')).toBeInTheDocument()

    // Nilai datang sebagai teks kanonik "50000001.00" dan ditampilkan sebagai rupiah.
    expect(screen.getByText('Rp 50.000.001')).toBeInTheDocument()

    // Ringkasan datang dari server, bukan dihitung ulang di layar.
    expect(screen.getByText(/3 jenjang persetujuan aktif dari 4 baris master/)).toBeInTheDocument()
  })

  // Batas atas nol berarti "tanpa batas", bukan "Rp 0". Menampilkannya apa adanya akan
  // membuat baris Bonding terlihat seperti tangga yang berhenti di nol rupiah.
  it('menampilkan batas atas nol sebagai tanpa batas', async () => {
    stubFetch()
    renderAt('/master/ambang-komite')
    await screen.findByText('KLAIMNONMBU')

    expect(screen.getAllByText('tanpa batas').length).toBeGreaterThan(0)
  })

  // Baris yang tidak ikut menyetujui harus dapat dibedakan DAN dijelaskan sebabnya.
  it('menjelaskan kenapa sebuah baris tidak ikut menyetujui', async () => {
    stubFetch()
    renderAt('/master/ambang-komite')
    await screen.findByText('KLAIMNONMBU')

    expect(screen.getByText('Pemberitahuan registrasi')).toBeInTheDocument()
    expect(screen.getAllByText('Jenjang').length).toBeGreaterThan(0)
  })

  it('menampilkan temuan integritas beserta barisnya', async () => {
    stubFetch()
    renderAt('/master/ambang-komite')

    expect(await screen.findByText(/Yang perlu diperhatikan/)).toBeInTheDocument()
    expect(screen.getByText(/baris 44/)).toBeInTheDocument()
  })

  // Ketiadaan tombol Tambah dan Ubah DIJELASKAN, bukan dibiarkan terlihat seperti fitur
  // yang terlupa. Pengguna yang terbiasa dengan layar master lain akan mencarinya.
  it('menjelaskan bahwa layarnya dibaca saja dan tidak menyediakan tombol ubah', async () => {
    stubFetch()
    renderAt('/master/ambang-komite')
    await screen.findByText('Dr. Wahyu')

    expect(screen.getByText(/Layar ini dibaca saja/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /tambah/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /ubah/i })).not.toBeInTheDocument()
  })

  it('membawa token sesi di header, bukan di URL', async () => {
    stubFetch()
    renderAt('/master/ambang-komite')
    await screen.findByText('Dr. Wahyu')

    const request = calls.find((p) => p.url === MASTER_PATH)
    expect(request).toBeDefined()
    expect(new Headers(request?.init?.headers).get('Authorization')).toBe('Bearer token-uji')
    expect(request?.url).not.toContain('token')
  })
})

describe('layar Penjenjangan Komite', () => {
  it('menghitung hanya setelah tombol ditekan, bukan pada setiap ketikan', async () => {
    stubFetch()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Nilai klaim')

    await user.type(screen.getByLabelText('Nilai klaim'), '80.000.000')
    expect(calls.some((p) => p.url.startsWith(PENJENJANGAN_PATH))).toBe(false)

    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(await screen.findByText(/2 jenjang harus menyetujui/)).toBeInTheDocument()
  })

  // Pemisah ribuan diurai di LAYAR, bukan dikirim apa adanya: server menolaknya dengan
  // sengaja karena artinya berbeda antar bahasa.
  it('mengirim nilai dalam bentuk kanonik, bukan apa yang diketik pengguna', async () => {
    stubFetch()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Nilai klaim')

    await user.type(screen.getByLabelText('Nilai klaim'), 'Rp 80.000.000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.click(screen.getByRole('button', { name: /hitung/i }))
    await screen.findByText(/2 jenjang harus menyetujui/)

    const request = calls.find((p) => p.url.startsWith(PENJENJANGAN_PATH))
    expect(request?.url).toContain('nilai=80000000.00')
    expect(request?.url).toContain('lini=NONMBU')
  })

  it('menolak nilai yang bukan angka tanpa memanggil server', async () => {
    stubFetch()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Nilai klaim')

    await user.type(screen.getByLabelText('Nilai klaim'), 'delapan puluh juta')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(await screen.findByText(/Masukkan nilai klaim/)).toBeInTheDocument()
    expect(calls.some((p) => p.url.startsWith(PENJENJANGAN_PATH))).toBe(false)
  })

  // Setiap penyetuju ditampilkan bersama ALASANNYA. Tanpa itu layar hanya menyodorkan
  // daftar nama dan pengguna tidak punya cara memeriksa apakah hasilnya masuk akal.
  it('menampilkan urutan dan alasan setiap penyetuju', async () => {
    stubFetch()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Nilai klaim')
    await user.type(screen.getByLabelText('Nilai klaim'), '80000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    await screen.findByText(/2 jenjang harus menyetujui/)

    // Urutan menyetujui terlihat sebagai angka, bukan hanya tersirat dari susunan.
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()

    // Dan alasan setiap orang ikut disebutkan — ambang yang membuatnya masuk daftar.
    expect(screen.getAllByText(/ikut karena nilai klaim melampaui/)).toHaveLength(2)
    expect(screen.getByText(/Rp 50\.000\.001/)).toBeInTheDocument()
  })

  // Keraguan urutan dilaporkan di layar, bukan disembunyikan — supaya perbedaan urutan
  // terhadap Pega pada kasus seri tidak terbaca sebagai cacat.
  it('memberitahukan ketika ada dua penyetuju ber-jenjang sama', async () => {
    stubFetch()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Nilai klaim')
    await user.type(screen.getByLabelText('Nilai klaim'), '80000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(
      await screen.findByText(/nomor jenjang yang sama/, { exact: false }),
    ).toBeInTheDocument()
  })

  // Pilihan lini datang DARI MASTER, bukan dari daftar tetap di dalam kode (`D-15`).
  it('mengisi pilihan lini dari master', async () => {
    stubFetch()
    renderAt('/komite/penjenjangan')

    const select = await screen.findByLabelText('Lini bisnis')
    const options = within(select).getAllByRole('option')

    expect(options.map((o) => o.getAttribute('value'))).toEqual(['', 'NONMBU', 'PA'])
  })

  it('menampilkan peringatan mencolok ketika tidak ada yang menyetujui', async () => {
    stubFetch((url) => {
      if (!url.startsWith(PENJENJANGAN_PATH)) return undefined
      return jsonResponse(200, {
        ...PENJENJANGAN_RESPONSE,
        penyetuju: [],
        jumlah_jenjang: 0,
        tanpa_penyetuju: true,
        urutan_tidak_pasti: false,
      })
    })
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Nilai klaim')
    await user.type(screen.getByLabelText('Nilai klaim'), '1')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(await screen.findByText(/Tidak ada jenjang yang cocok/)).toBeInTheDocument()
  })
})

describe('portal bermode satu-penyetuju', () => {
  const MASTER_SIMASNET = {
    ...MASTER_RESPONSE,
    lini: ['SIMASNET'],
    kebijakan_pita: [],
    mode: 'satu-penyetuju',
  }

  const HASIL_SIMASNET = {
    nilai: '1000000.00',
    lini: 'SIMASNET',
    mode: 'satu-penyetuju',
    berpita_nilai: false,
    penyetuju: [
      {
        urutan: 1,
        jenjang: 1,
        nama: 'Petugas B',
        operator_id: 'USERB',
        batas_bawah: '0.00',
        sedang_absen: false,
        id_ambang: '102',
      },
    ],
    kandidat: [
      {
        urutan: 1,
        jenjang: 1,
        nama: 'Petugas B',
        operator_id: 'USERB',
        batas_bawah: '0.00',
        sedang_absen: false,
        id_ambang: '102',
      },
      {
        urutan: 2,
        jenjang: 1,
        nama: 'Petugas C',
        operator_id: 'USERC',
        batas_bawah: '0.00',
        sedang_absen: false,
        id_ambang: '103',
      },
    ],
    dikecualikan_penginput: 'USERA',
    jumlah_jenjang: 1,
    tanpa_penyetuju: false,
    urutan_tidak_pasti: false,
  }

  function stubFetchSimasnet() {
    stubFetch((url) => {
      if (url === MASTER_PATH) return jsonResponse(200, MASTER_SIMASNET)
      if (url.startsWith(PENJENJANGAN_PATH)) return jsonResponse(200, HASIL_SIMASNET)
      return undefined
    })
  }

  // Isian Operator ID pengaju muncul pada SEMUA mode sejak pengecualian penginput
  // diberlakukan ke seluruh entitas (Work Owner, 2026-09-18).
  it('menampilkan isian Operator ID pengaju pada kedua mode', async () => {
    stubFetchSimasnet()
    renderAt('/komite/penjenjangan')

    expect(await screen.findByLabelText('Operator ID pengaju')).toBeInTheDocument()
  })

  // Yang dapat diperiksa pada mode ini bukan siapa yang terpilih — itu acak — melainkan
  // apakah kumpulan calonnya sudah benar. Karena itu kandidat ditampilkan utuh.
  it('menampilkan seluruh calon dan siapa yang dikecualikan', async () => {
    stubFetchSimasnet()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Operator ID pengaju')

    await user.type(screen.getByLabelText('Nilai klaim'), '1000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'SIMASNET')
    await user.type(screen.getByLabelText('Operator ID pengaju'), 'USERA')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(await screen.findByText('Satu penyetuju dipilih')).toBeInTheDocument()
    expect(screen.getByText(/Dipilih acak dari 2 calon/)).toBeInTheDocument()
    expect(screen.getByText(/dikeluarkan dari daftar calon/)).toBeInTheDocument()

    // Dan dinyatakan tegas bahwa hasil yang berbeda bukan cacat — supaya tidak
    // dilaporkan sebagai bug saat uji kesetaraan.
    expect(screen.getByText(/Karena pemilihannya acak/)).toBeInTheDocument()

    // Yang terpilih ditandai di antara para calon.
    expect(screen.getByText(/Petugas B · terpilih/)).toBeInTheDocument()
  })

  it('mengirim penginput sebagai parameter kueri', async () => {
    stubFetchSimasnet()
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Operator ID pengaju')

    await user.type(screen.getByLabelText('Nilai klaim'), '1000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'SIMASNET')
    await user.type(screen.getByLabelText('Operator ID pengaju'), 'USERA')
    await user.click(screen.getByRole('button', { name: /hitung/i }))
    await screen.findByText('Satu penyetuju dipilih')

    const request = calls.find((p) => p.url.startsWith(PENJENJANGAN_PATH))
    expect(request?.url).toContain('penginput=USERA')
  })
})

describe('pengecualian penginput pada mode kumulatif', () => {
  const HASIL_TERSINGKIR = {
    ...PENJENJANGAN_RESPONSE,
    penyetuju: [PENJENJANGAN_RESPONSE.penyetuju[1]],
    jumlah_jenjang: 1,
    urutan_tidak_pasti: false,
    dikecualikan_penginput: 'ELLENSUPRIYATI',
    tersingkir: [PENJENJANGAN_RESPONSE.penyetuju[0]],
  }

  const HASIL_HABIS = {
    ...HASIL_TERSINGKIR,
    penyetuju: [],
    jumlah_jenjang: 0,
    tanpa_penyetuju: true,
  }

  // Pengurangan jumlah penyetuju dilaporkan, tidak terjadi diam-diam. Tanpa keterangan
  // ini, dua orang yang menghitung klaim yang sama akan mendapat angka berbeda dan tidak
  // punya cara mengetahui sebabnya.
  it('memberitahukan siapa yang tersingkir dan berapa berkurangnya', async () => {
    stubFetch((url) =>
      url.startsWith(PENJENJANGAN_PATH) ? jsonResponse(200, HASIL_TERSINGKIR) : undefined,
    )
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Operator ID pengaju')

    await user.type(screen.getByLabelText('Nilai klaim'), '80000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.type(screen.getByLabelText('Operator ID pengaju'), 'ELLENSUPRIYATI')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(await screen.findByText(/1 jenjang harus menyetujui/)).toBeInTheDocument()
    expect(screen.getByText(/mengajukan klaim ini sendiri/)).toBeInTheDocument()
    expect(screen.getByText(/berkurang 1/)).toBeInTheDocument()
  })

  // Keadaan yang paling perlu terlihat: seluruh penyetuju tersingkir. Sebabnya harus
  // dibedakan dari "master tidak menjangkau nilai ini" — keduanya menuntut tindakan yang
  // berbeda dari orang yang berbeda.
  it('membedakan seluruh penyetuju tersingkir dari master yang tidak menjangkau', async () => {
    stubFetch((url) =>
      url.startsWith(PENJENJANGAN_PATH) ? jsonResponse(200, HASIL_HABIS) : undefined,
    )
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Operator ID pengaju')

    await user.type(screen.getByLabelText('Nilai klaim'), '20000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.type(screen.getByLabelText('Operator ID pengaju'), 'ELLENSUPRIYATI')
    await user.click(screen.getByRole('button', { name: /hitung/i }))

    expect(await screen.findByText('Seluruh penyetuju tersingkir')).toBeInTheDocument()
    expect(screen.queryByText('Tidak ada jenjang yang cocok')).not.toBeInTheDocument()
  })

  it('mengirim penginput pada mode kumulatif juga', async () => {
    stubFetch((url) =>
      url.startsWith(PENJENJANGAN_PATH) ? jsonResponse(200, HASIL_TERSINGKIR) : undefined,
    )
    renderAt('/komite/penjenjangan')

    const user = userEvent.setup()
    await screen.findByLabelText('Operator ID pengaju')

    await user.type(screen.getByLabelText('Nilai klaim'), '80000000')
    await user.selectOptions(screen.getByLabelText('Lini bisnis'), 'NONMBU')
    await user.type(screen.getByLabelText('Operator ID pengaju'), 'ELLENSUPRIYATI')
    await user.click(screen.getByRole('button', { name: /hitung/i }))
    await screen.findByText(/1 jenjang harus menyetujui/)

    const request = calls.find((p) => p.url.startsWith(PENJENJANGAN_PATH))
    expect(request?.url).toContain('penginput=ELLENSUPRIYATI')
  })
})
