import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DocumentResponse } from './types'

const PATH = '/api/inbox-manager-receive-pucl'
const DOC_PATH = `${PATH}/dokumen`

const SAMPLE_PROFILE = {
  identitas: '90000003',
  nama: 'Contoh Penyelia',
  jenis: 'KARYAWAN',
  login: 'penyeliacontoh',
  email: 'contoh.penyelia@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Kunci teknis Pega. Ia memuat SPASI, dan itulah yang membuat pengkodean alamat wajib. */
const REFERENSI = 'ASM-FW-GCNMFW-WORK RCV-900001'

/**
 * Isi layar kerja contoh — seluruhnya KARANGAN (`D-69`).
 *
 * Bentuknya meniru jawaban server apa adanya: `kelompok` menyatakan isian mana yang
 * digambar, `nilai` menyatakan isinya, dan isian bertanda `terhalang` sengaja TIDAK punya
 * entri di `nilai`. Pemisahan itulah yang diuji di bawah.
 */
const DOKUMEN: DocumentResponse = {
  referensi: REFERENSI,
  no_case: 'RCV-900001',
  no_klaim_pnc: 'PNCN.26.0001',
  jenis_klaim: 'PA',
  status_kerja: 'Open',

  kelompok: [
    {
      judul: 'Penerimaan Dokumen',
      isian: [
        { kunci: 'tanggal_terima_dokumen', judul: 'Tanggal Terima Dokumen' },
        { kunci: 'nama_pengirim', judul: 'Nama Pengirim / Pelapor Dokumen' },
      ],
    },
    {
      judul: 'Data Polis',
      isian: [
        { kunci: 'nomor_polis', judul: 'Nomor Polis' },
        {
          kunci: 'polis_leader',
          judul: 'Polis Leader',
          terhalang: true,
          alasan_terhalang:
            'Isian ini hidup di dalam blob objek kerja Pega dan tidak punya kolom basis ' +
            'data.',
          pemilik_penghalang: 'Tim Pega + DBA',
        },
      ],
    },
    {
      judul: 'Kejadian',
      isian: [
        { kunci: 'kronologis_kejadian', judul: 'Kronologis Kejadian', bertingkat: true },
      ],
    },
    {
      judul: 'Transfer dan Registrasi',
      isian: [{ kunci: 'subjek_email', judul: 'Subjek Email' }],
    },
  ],

  nilai: {
    tanggal_terima_dokumen: '03/09/2026',
    nama_pengirim: 'Pengirim Contoh Satu',
    nomor_polis: 'CONTOH-PA-0001',
    kronologis_kejadian: 'Kronologi contoh baris pertama.',

    // Sumbernya ADA, isinya kosong. Ia harus tergambar sebagai tanda pisah — bukan sebagai
    // "belum terbawa", yang berarti hal yang berbeda sama sekali.
    subjek_email: '',
  },

  tindakan: [
    {
      kode: 'register-klaim',
      label: 'Register Klaim',
      activity_pega: 'CreateRegisterKlaimPNC',
      pemilik: '`B-2` Registrasi Klaim',
    },
    {
      kode: 'simpan',
      label: 'Simpan',
      activity_pega: '(flow action save)',
      pemilik: 'modul ini',
    },
  ],

  portal: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

/** Menggambar layar kerja langsung lewat alamatnya, tanpa melewati antrean. */
function renderWorkScreen() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter
        initialEntries={[
          `/inbox-manager-receive-pucl/dokumen/${encodeURIComponent(REFERENSI)}?tab=1`,
        ]}
      >
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
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

describe('bentuk layar kerja', () => {
  beforeEach(() => {
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) return jsonResponse(200, DOKUMEN)
      return jsonResponse(404, { kode: 'tidak_dikenal', pesan: 'Tidak dilayani uji ini.' })
    })
  })

  it('meminta berkasnya dengan kunci yang TERKODEKAN', async () => {
    // `pzInsKey` memuat spasi. Spasi mentah di dalam alamat bukan alamat yang sah, dan
    // kegagalannya tidak selalu terlihat sebagai galat — sebagian peladen mengubahnya
    // diam-diam.
    renderWorkScreen()
    await screen.findByText(/Berkas RCV-900001/)

    const dokumen = calls.find((c) => c.url.startsWith(DOC_PATH))
    expect(dokumen?.url).toBe(`${DOC_PATH}/${encodeURIComponent(REFERENSI)}`)
    expect(dokumen?.url).not.toContain(' ')
  })

  it('menggambar judul isian dan kelompok yang ditetapkan server', async () => {
    // Ke-36 judul dibaca backend dari `Section/InputReceiveDocument_sect.xml`. Menyalinnya
    // ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal
    // saat yang lain diperbaiki.
    renderWorkScreen()

    expect(await screen.findByText('Nama Pengirim / Pelapor Dokumen')).toBeInTheDocument()
    expect(screen.getByText('Kronologis Kejadian')).toBeInTheDocument()
    expect(screen.getByText('Penerimaan Dokumen')).toBeInTheDocument()
    expect(screen.getByText('Data Polis')).toBeInTheDocument()
  })

  it('menggambar isi isian yang punya nilai', async () => {
    renderWorkScreen()

    expect(await screen.findByText('Pengirim Contoh Satu')).toBeInTheDocument()
    expect(screen.getByText('CONTOH-PA-0001')).toBeInTheDocument()
  })

  it('menyatakan nomor klaim PNC yang kosong sebagai belum diregistrasi', async () => {
    // Kosong berarti berkasnya belum menjadi klaim — keadaan yang SAH, dan justru itulah
    // yang dikerjakan tombol "Register Klaim". Tanda pisah akan terbaca sebagai data hilang.
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) {
        return jsonResponse(200, { ...DOKUMEN, no_klaim_pnc: '' })
      }
      return jsonResponse(404, {})
    })
    renderWorkScreen()

    expect(await screen.findByText(/belum diregistrasi/)).toBeInTheDocument()
  })
})

describe('isian yang belum terbawa', () => {
  beforeEach(() => {
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) return jsonResponse(200, DOKUMEN)
      return jsonResponse(404, {})
    })
  })

  it('MENANDAI isian yang terhalang, bukan menyembunyikannya', async () => {
    // Menyembunyikannya membuat pengguna yang membandingkan layar ini dengan Pega mengira
    // isiannya hilang. Menggambarnya dengan tanda membuat ia tahu isiannya ada dan kenapa
    // kosong — itu pula yang dituntut uji kesetaraan gerbang 1.
    renderWorkScreen()

    expect(await screen.findByText('Polis Leader')).toBeInTheDocument()

    // `getAllByText`, bukan `getByText`: frasa yang sama muncul dua kali dengan sengaja —
    // sekali sebagai TANDA pada isiannya, sekali lagi di keterangan bawah yang menjelaskan
    // arti tanda itu. Keduanya memang harus ada.
    expect(screen.getAllByText('belum terbawa').length).toBeGreaterThan(0)
  })

  it('membedakan isian yang belum terbawa dari isian yang sumbernya kosong', async () => {
    // Keduanya terlihat sama bila tidak dibedakan, padahal tindak lanjutnya berbeda sama
    // sekali: yang satu menuntut kolomnya dibuka di Pega, yang lain menuntut petugas
    // mengisinya.
    renderWorkScreen()

    const kosong = await screen.findByText('Subjek Email')
    expect(kosong.parentElement?.textContent).toContain('—')
    expect(kosong.parentElement?.textContent).not.toContain('belum terbawa')
  })

  it('menyebut jumlah dan pemilik penghalangnya di satu tempat', async () => {
    // Tanda per isian menyatakan isian MANA; keterangan ini menyatakan BERAPA dan KENAPA.
    // Tanpa yang kedua, pengguna yang melihat tanda tersebar akan mengira layarnya rusak
    // alih-alih belum lengkap. Penghalang tanpa alamat tidak pernah hilang (`D-36`).
    renderWorkScreen()

    expect(await screen.findByText(/1 isian belum terbawa dari Pega/)).toBeInTheDocument()
    expect(screen.getByText(/Tim Pega \+ DBA/)).toBeInTheDocument()
  })
})

describe('tombol tulis', () => {
  it('menggambar tombol yang dikirim server', async () => {
    // Tombolnya tetap digambar supaya keberadaannya terlihat: layar yang kehilangan
    // tombolnya tanpa penjelasan akan dilaporkan sebagai kerusakan.
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) return jsonResponse(200, DOKUMEN)
      return jsonResponse(404, {})
    })
    renderWorkScreen()

    expect(await screen.findByRole('button', { name: 'Register Klaim' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Simpan' })).toBeInTheDocument()
  })

  it('MENGIRIM penekanannya ke peladen dan menampilkan alasannya', async () => {
    // Alasan mengapa sebuah tombol belum bekerja adalah keadaan SISTEM, bukan keadaan
    // layar. Menuliskannya di layar berarti alasan yang sama hidup di dua tempat, dan yang
    // di layar akan tertinggal pada hari tombolnya mulai bekerja.
    //
    // Penekanannya pun DICATAT di sisi peladen — jejak itulah satu-satunya tanda seberapa
    // sering tombol ini benar-benar dibutuhkan selama masa paralel.
    stubFetch((url) => {
      if (url.startsWith(`${PATH}/tindakan`)) {
        return jsonResponse(501, {
          kode: 'belum_tersedia',
          pesan: 'Tindakan atas berkas ini belum tersedia di sistem baru.',
        })
      }
      if (url.startsWith(DOC_PATH)) return jsonResponse(200, DOKUMEN)
      return jsonResponse(404, {})
    })
    renderWorkScreen()

    await userEvent.click(await screen.findByRole('button', { name: 'Register Klaim' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/belum tersedia/i)

    const tindakan = calls.find((c) => c.url.startsWith(`${PATH}/tindakan`))
    expect(tindakan?.url).toContain('tindakan=register-klaim')
    expect(tindakan?.init?.method).toBe('POST')
  })

  it('menampilkan kunci berkas supaya tidak perlu dicari saat dikerjakan di Pega', async () => {
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) return jsonResponse(200, DOKUMEN)
      return jsonResponse(404, {})
    })
    renderWorkScreen()

    expect(await screen.findByText(REFERENSI)).toBeInTheDocument()
  })
})

describe('keadaan yang tidak wajar', () => {
  it('menyatakan berkas yang tidak ditemukan alih-alih menggambar layar kosong', async () => {
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) {
        return jsonResponse(404, {
          kode: 'berkas_tidak_ditemukan',
          pesan: 'Berkas penerimaan dokumen ini tidak ditemukan.',
        })
      }
      return jsonResponse(404, {})
    })
    renderWorkScreen()

    expect(await screen.findByText(/Berkas tidak dapat dimuat/)).toBeInTheDocument()
    expect(screen.getByText(/tidak ditemukan/)).toBeInTheDocument()
  })

  it('menyatakan jawaban 200 yang badannya bukan JSON, bukan halaman kosong', async () => {
    // `callAPI` mengembalikan `null` — bukan melempar — untuk keadaan ini, misalnya saat
    // alamat `/api/...` dijawab penyaji SPA dengan `index.html`. Tanpa penanganan, halaman
    // yang dihasilkan benar-benar kosong: bukan memuat, bukan galat, bukan data. Itu kelas
    // kegagalan yang paling sulit dilaporkan pengguna.
    stubFetch((url) => {
      if (url.startsWith(DOC_PATH)) {
        return new Response('<!doctype html><html></html>', {
          status: 200,
          headers: { 'Content-Type': 'text/html' },
        })
      }
      return jsonResponse(404, {})
    })
    renderWorkScreen()

    expect(await screen.findByText(/Berkas tidak terbaca/)).toBeInTheDocument()
  })
})
