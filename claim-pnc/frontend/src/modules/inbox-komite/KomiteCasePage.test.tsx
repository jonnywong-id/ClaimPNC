import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSession } from '@/app/session'
import type { KomiteCase } from '@/api/types'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Ellen Supriyati',
  jenis: 'KARYAWAN',
  login: 'ELLENSUPRIYATI',
  email: 'contoh.komite@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Seluruh isinya KARANGAN — `D-69` melarang data nasabah di berkas yang di-commit. */
const KASUS: KomiteCase = {
  nomor_case: 'K-2601',
  nomor_klaim: 'PNCN.26.0101',
  nomor_polis: 'CONTOH-PL-000117',
  nama_tertanggung: 'PT Harapan Sentosa',
  nama_bisnis: 'Property All Risk',
  sumber_bisnis: 'Direct',
  cabang: 'Jakarta Pusat',
  tanggal_komite: '2026-09-11T02:00:00Z',
  tanggal_input: '2026-09-11T02:00:00Z',
  aging_komite: 9,
  status_kerja: 'Open',
  penjenjangan: {
    kesimpulan: 'menunggu',
    jumlah_jenjang: 0,
    jenjang_kini: 1,
    jenjang_disetujui: 0,
    jumlah_jenjang_belum_diketahui: true,
    selesai: false,
    sudah_saya_putuskan: false,
    keputusan: [],
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

function renderDetail(nomor = 'K-2601') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/komite/inbox/${nomor}`]}>
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
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('rincian kasus komite', () => {
  function stubBerhasil() {
    stubFetch(() =>
      jsonResponse(200, { kasus: KASUS, sekarang: '2026-09-20T02:00:00Z' }),
    )
  }

  it('meminta rincian kasus yang disebut di alamatnya', async () => {
    const dipanggil: string[] = []
    vi.stubGlobal('fetch', (url: string) => {
      dipanggil.push(url)
      if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
      if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
      return Promise.resolve(
        jsonResponse(200, { kasus: KASUS, sekarang: '2026-09-20T02:00:00Z' }),
      )
    })
    renderDetail()

    await screen.findByText('PT Harapan Sentosa')
    expect(dipanggil).toContain('/api/komite/inbox/K-2601')
  })

  // Judulnya diambil dari `Section/ShowTransfer` apa adanya, supaya anggota komite yang
  // berpindah dari layar lama mengenali tempatnya.
  //
  // Tanpa rincian, yang tergambar judul telanjang — sama seperti Pega ketika tidak satu pun
  // dari enam syarat akhirannya terpenuhi.
  it('memakai judul yang sama dengan ShowTransfer', async () => {
    stubBerhasil()
    renderDetail()

    expect(await screen.findByRole('heading', { name: 'CLAIM COMMITTEE' })).toBeInTheDocument()

    // Datanya ditunggu LEBIH DULU. Judul dan nomor case tergambar sebelum permintaan
    // selesai — dan sebelum itu, "Claim No." masih menampilkan nomor case dari alamatnya
    // sebagai cadangan. Memeriksa terlalu awal akan menguji keadaan sementara itu.
    await screen.findByText('PT Harapan Sentosa')

    // Teksnya terpecah dua simpul di JSX, jadi yang dicocokkan isi elemennya setelah
    // spasinya diratakan.
    const rapat = (el: Element | null) => (el?.textContent ?? '').replace(/\s+/g, ' ').trim()
    expect(
      screen.getAllByText((_, el) => rapat(el) === 'Claim No. PNCN.26.0101').length,
    ).toBeGreaterThan(0)
  })

  it('menampilkan data kasus yang kita punya', async () => {
    stubBerhasil()
    renderDetail()

    expect(await screen.findByText('PT Harapan Sentosa')).toBeInTheDocument()
    expect(screen.getByText('CONTOH-PL-000117')).toBeInTheDocument()
    expect(screen.getByText('Property All Risk')).toBeInTheDocument()
    expect(screen.getByText('Jakarta Pusat')).toBeInTheDocument()
    expect(screen.getByText('9 hari')).toBeInTheDocument()
  })

  // Halaman yang terlihat utuh padahal isinya belum ada adalah cara paling cepat membuat
  // orang mengira modulnya selesai. Ketiadaan itu WAJIB terbaca.
  it('menyebut bagian ShowTransfer yang belum dibangun, satu per satu', async () => {
    stubBerhasil()
    renderDetail()

    await screen.findByText('PT Harapan Sentosa')
    expect(screen.getByText('ShowTransferDetailHE')).toBeInTheDocument()
    expect(screen.getByText('UploadDocumentKomite')).toBeInTheDocument()
    expect(screen.getByText('ViewPolicyDetail')).toBeInTheDocument()

    // ShowTransferDetail SUDAH dibangun, jadi ia tidak boleh lagi disebut sebagai yang
    // belum ada. Daftar yang tidak ikut menyusut akan berhenti dipercaya orang.
    expect(screen.queryByText('ShowTransferDetail')).not.toBeInTheDocument()
  })

  it('tidak menawarkan tombol keputusan', async () => {
    stubBerhasil()
    renderDetail()

    await screen.findByText('PT Harapan Sentosa')
    expect(screen.queryByText('Putuskan')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Setuju|Tolak|Kembalikan/ })).not.toBeInTheDocument()
  })

  // `404` menyatukan "tidak ada" dan "bukan milik Anda" DENGAN SENGAJA. Layar tidak boleh
  // membedakannya kembali — itu akan mengubah alamat halaman ini menjadi alat menebak
  // nomor case.
  it('menjelaskan 404 tanpa membedakan tidak-ada dari bukan-milik-Anda', async () => {
    stubFetch(() =>
      jsonResponse(404, {
        kode: 'kasus_tidak_ditemukan',
        pesan: 'Kasus komite itu tidak ada di inbox Anda.',
      }),
    )
    renderDetail('K-9999')

    expect(await screen.findByText('Kasus itu tidak ada di inbox Anda')).toBeInTheDocument()
    expect(screen.getAllByText(/K-9999/).length).toBeGreaterThan(0)
  })

  it('menyediakan jalan kembali ke inbox', async () => {
    stubBerhasil()
    renderDetail()

    await screen.findByText('PT Harapan Sentosa')
    expect(screen.getByRole('link', { name: 'Kembali ke inbox' })).toHaveAttribute(
      'href',
      '/komite/inbox',
    )
  })
})

/**
 * Rincian transfer — pengganti `Section/ShowTransferDetail`.
 */
describe('detail transfer', () => {
  const TRANSFER = {
    judul: 'CLAIM COMMITTEE - ADJUSTMENT',
    he_dapat_dinilai: false,
    baris: [
      {
        nomor_klaim: 'PNC-3621',
        id_objek: '2',
        id_coverage: '1',
        mata_uang: '10026',
        jenis_pembayaran: '2',
        nilai_gross: '322979460.00',
        nilai_usulan: '0.00',
        nilai_akseptasi: '64595892.00',
        nilai_salvage: '0.00',
        nilai_asm_share: '64595892.00',
        nilai_risiko_sendiri: '0.00',
        persen_asm_share: '20',
        ex_gratia: false,
        sebab_kerugian: 'Keterangan sebab kerugian contoh.',
      },
    ],
    komite: {
      nama_komite: 'CONTOHKOMITE',
      jenjang: 1,
      tipe_komite: 'Interim',
      catatan: 'setuju',
      nilai_klaim: '500000.00',
      persen_asm_share: '100',
      tanggal_komite: '2026-09-11T02:00:00Z',
      kesimpulan: 'disetujui',
    },
    klaim: {
      tanggal_kejadian: '2026-09-01T02:00:00Z',
      tanggal_register: '2026-09-03T02:00:00Z',
      lokasi: 'Gudang contoh, Jakarta',
      kronologi: 'Kronologi contoh yang ditulis untuk uji.',
      status_klaim: '1149',
      persen_asm_share: '20',
      koasuransi: 'Contoh Koasuransi',
    },
    coverage: [
      {
        id_objek: '1',
        id_coverage: '1',
        nama_coverage: 'FLEXAS',
        sebab_kerugian: 'FIRE - OPEN FLAME',
        nilai_tsi: '1500000000.00',
        luas_kerugian: 'Luas kerugian contoh untuk uji.',
        analisis_terisi: true,
      },
    ],
    kosong: false,
    nilai_uang_kosong: false,
  }

  it('menampilkan nilai uang dari T_CLAIM_ADJUSTMENT', async () => {
    stubFetch(() =>
      jsonResponse(200, {
        kasus: KASUS,
        sekarang: '2026-09-20T02:00:00Z',
        transfer: TRANSFER,
      }),
    )
    renderDetail()

    await screen.findByText('Detail transfer')
    expect(screen.getByText('Rp 322.979.460')).toBeInTheDocument()
    // Muncul dua kali — sebagai Akseptasi dan sebagai Share ASM, karena share-nya 20%
    // dari gross kebetulan sama dengan nilai akseptasinya pada data contoh ini.
    expect(screen.getAllByText('Rp 64.595.892').length).toBe(2)
    expect(screen.getByText('Share ASM (20%)')).toBeInTheDocument()
    expect(screen.getByText('Keterangan sebab kerugian contoh.')).toBeInTheDocument()
  })

  it('menampilkan keputusan komite Pega beserta asalnya', async () => {
    stubFetch(() =>
      jsonResponse(200, {
        kasus: KASUS,
        sekarang: '2026-09-20T02:00:00Z',
        transfer: TRANSFER,
      }),
    )
    renderDetail()

    expect(await screen.findByText('Keputusan komite di Pega')).toBeInTheDocument()
    expect(screen.getByText('CONTOHKOMITE')).toBeInTheDocument()
    expect(screen.getByText('Interim')).toBeInTheDocument()
  })

  // Nol yang tidak dapat dibedakan dari "belum ada" terbaca sebagai angka yang sudah
  // diputuskan. Dari 189 case, 148 menempuh jalur ini — ia yang paling sering terlihat.
  it('menyebut ketiadaan sebagai keterangan, bukan sebagai Rp 0', async () => {
    stubFetch(() =>
      jsonResponse(200, {
        kasus: KASUS,
        sekarang: '2026-09-20T02:00:00Z',
        transfer: {
          judul: 'CLAIM COMMITTEE',
          he_dapat_dinilai: false,
          baris: [],
          komite: null,
          klaim: null,
          coverage: [],
          kosong: true,
          nilai_uang_kosong: true,
        },
      }),
    )
    renderDetail()

    await screen.findByText('Detail transfer')
    expect(screen.getByText(/bukan angka nol, melainkan belum ada angkanya/)).toBeInTheDocument()
    expect(screen.queryByText('Rp 0')).not.toBeInTheDocument()
  })
})

/**
 * Blok klaim dan analisis komite — bagian `ShowTransferDetail` yang selama ini KOSONG.
 *
 * Sebabnya satu: `T_CLAIM_PNC.CLAIMID` dan `T_CLAIM_OBJECTCOVERAGE.CLAIMID` menyimpan
 * nomor klaim BER-PREFIX `ASM-FW-GCNMFW-WORK `, dan kuncinya dipangkas lebih dulu — join
 * menghasilkan nol baris pada seluruh 189 case, tanpa satu pun galat.
 */
describe('klaim dan analisis komite', () => {
  const LENGKAP = {
    judul: 'CLAIM COMMITTEE',
    he_dapat_dinilai: false,
    baris: [],
    komite: null,
    klaim: {
      tanggal_kejadian: '2026-09-01T02:00:00Z',
      tanggal_register: '2026-09-03T02:00:00Z',
      lokasi: 'Gudang contoh, Jakarta',
      kronologi: 'Kronologi contoh yang ditulis untuk uji.',
      status_klaim: '1149',
      persen_asm_share: '20',
      koasuransi: 'Contoh Koasuransi',
      rekomendasi: 'Rekomendasi contoh.',
    },
    coverage: [
      {
        id_objek: '1',
        id_coverage: '1',
        nama_coverage: 'FLEXAS',
        sebab_kerugian: 'FIRE - OPEN FLAME',
        nilai_tsi: '1500000000.00',
        luas_kerugian: 'Luas kerugian contoh untuk uji.',
        analisis_terisi: true,
      },
    ],
    kosong: false,
    // Inilah keadaan mayoritas: klaim lengkap, nilai uang belum ada.
    nilai_uang_kosong: true,
  }

  function stubLengkap(transfer: unknown = LENGKAP) {
    stubFetch(() =>
      jsonResponse(200, {
        kasus: KASUS,
        sekarang: '2026-09-20T02:00:00Z',
        transfer,
      }),
    )
  }

  it('menampilkan klaim yang dikomitekan', async () => {
    stubLengkap()
    renderDetail()

    expect(await screen.findByText('Klaim yang dikomitekan')).toBeInTheDocument()
    expect(screen.getByText('Gudang contoh, Jakarta')).toBeInTheDocument()
    expect(screen.getByText('Kronologi contoh yang ditulis untuk uji.')).toBeInTheDocument()
    expect(screen.getByText('Rekomendasi contoh.')).toBeInTheDocument()
  })

  it('menampilkan objek, sebab kerugian, dan TSI-nya', async () => {
    stubLengkap()
    renderDetail()

    expect(await screen.findByText('Objek dan analisis komite')).toBeInTheDocument()
    expect(screen.getByText('FLEXAS')).toBeInTheDocument()
    expect(screen.getByText('FIRE - OPEN FLAME')).toBeInTheDocument()
    expect(screen.getByText('Rp 1.500.000.000')).toBeInTheDocument()
    expect(screen.getByText('Luas kerugian contoh untuk uji.')).toBeInTheDocument()
  })

  // Inti perbaikannya: nilai uang yang belum ada TIDAK BOLEH menyembunyikan data klaim.
  // 148 dari 189 case berada di keadaan ini, dan sebelumnya seluruhnya tampil kosong.
  it('tetap menampilkan klaim meski nilai uangnya belum ada', async () => {
    stubLengkap()
    renderDetail()

    await screen.findByText('Klaim yang dikomitekan')
    expect(screen.getByText(/bukan angka nol, melainkan belum ada angkanya/)).toBeInTheDocument()
    expect(screen.getByText('Objek dan analisis komite')).toBeInTheDocument()
  })

  // Judul blok yang di bawahnya kosong lebih buruk daripada tidak ada blok. 53 dari 79
  // baris coverage tidak punya satu pun keterangan analisis.
  it('tidak menggambar blok analisis ketika keterangannya kosong', async () => {
    stubLengkap({
      ...LENGKAP,
      coverage: [
        {
          id_objek: '1',
          id_coverage: '1',
          nama_coverage: 'FLEXAS',
          sebab_kerugian: 'FIRE - OPEN FLAME',
          nilai_tsi: '1500000000.00',
          analisis_terisi: false,
        },
      ],
    })
    renderDetail()

    await screen.findByText('Objek dan analisis komite')
    expect(screen.getByText('FLEXAS')).toBeInTheDocument()
    expect(screen.queryByText('Luas kerugian')).not.toBeInTheDocument()
    expect(screen.queryByText('Keadaan kerugian')).not.toBeInTheDocument()
  })

  it('melewati kedua blok ketika klaimnya memang tidak terbaca', async () => {
    stubLengkap({ ...LENGKAP, klaim: null, coverage: [] })
    renderDetail()

    await screen.findByText('PT Harapan Sentosa')
    expect(screen.queryByText('Klaim yang dikomitekan')).not.toBeInTheDocument()
    expect(screen.queryByText('Objek dan analisis komite')).not.toBeInTheDocument()
  })
})

/**
 * Judul bersyarat — tujuh sel label `Section/ShowTransfer`.
 *
 * Aturannya hidup di server; yang diuji di sini hanyalah bahwa layar MENAMPILKANNYA apa
 * adanya dan tidak menyusun ulang. Menyusun ulang di sini berarti aturan yang sama hidup
 * di dua tempat, dan yang satu akan tertinggal.
 */
describe('judul bersyarat ShowTransfer', () => {
  function stubJudul(judul: string, heDapatDinilai = false) {
    stubFetch(() =>
      jsonResponse(200, {
        kasus: KASUS,
        sekarang: '2026-09-20T02:00:00Z',
        transfer: {
          judul,
          he_dapat_dinilai: heDapatDinilai,
          baris: [],
          komite: null,
          klaim: null,
          coverage: [],
          kosong: true,
          nilai_uang_kosong: true,
        },
      }),
    )
  }

  it.each([
    'CLAIM COMMITTEE',
    'CLAIM COMMITTEE - SURVEY',
    'CLAIM COMMITTEE - ADJUSTMENT',
    'CLAIM COMMITTEE - TOLAK KLAIM',
    'CLAIM COMMITTEE - REJECT - TOLAK KLAIM',
  ])('memakai judul dari server apa adanya: %s', async (judul) => {
    stubJudul(judul)
    renderDetail()

    expect(await screen.findByRole('heading', { name: judul })).toBeInTheDocument()
  })

  // Perbedaan yang menentukan: "case ini bukan HE" dan "tidak diketahui apakah ia HE"
  // adalah dua pernyataan berbeda, dan hanya satu yang jujur ketika kolomnya kosong.
  it('menyatakan cabang HE TIDAK DAPAT dinilai selama BUSINESSTYPE kosong', async () => {
    stubJudul('CLAIM COMMITTEE', false)
    renderDetail()

    await screen.findByText('Bagian yang belum dibangun')
    expect(screen.getByText(/belum dapat dinilai sama sekali/)).toBeInTheDocument()
  })

  it('berhenti menyebut ketidaktahuan itu begitu BUSINESSTYPE terisi', async () => {
    stubJudul('CLAIM COMMITTEE', true)
    renderDetail()

    await screen.findByText('Bagian yang belum dibangun')
    expect(screen.queryByText(/belum dapat dinilai sama sekali/)).not.toBeInTheDocument()
    expect(screen.getByText(/lini HE belum ditangani modul ini/)).toBeInTheDocument()
  })

  // Blok surveyor ada di ShowTransfer dan TIDAK dibangun. Sebabnya disebut supaya yang
  // membacanya kelak tahu bahwa kolomnya sudah dicari, bukan belum dicari.
  it('menyebut blok surveyor beserta sebab ia tidak dibangun', async () => {
    stubJudul('CLAIM COMMITTEE - SURVEY')
    renderDetail()

    await screen.findByText('Bagian yang belum dibangun')
    expect(screen.getByText('Blok surveyor')).toBeInTheDocument()
    expect(screen.getByText(/terisi 0 dari 610 case komite/)).toBeInTheDocument()
  })
})
