import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse } from './types'

const CLAIM = 'CLMNP-1001'
const PATH = `/api/input-acceptation/${CLAIM}`

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminnonprop1',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Jawaban contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * Bentuknya sengaja memuat ketiga keadaan yang membedakan layar ini: isian yang hanya
 * ditampilkan, isian yang dapat diubah, dan isian yang TERHALANG.
 */
const DETAIL: DetailResponse = {
  no_klaim: CLAIM,
  status_kerja: 'Pending-Acceptation',
  operator_pengubah: 'ADMINNONPROP1',
  kelompok: [
    {
      kode: 'treaty',
      judul: 'Treaty Information',
      isian: [
        {
          kunci: 'treaty_id',
          judul: 'Treaty ID',
          nilai: 'TNP-2026-01',
          dapat_diubah: false,
          terhalang: false,
        },
        {
          kunci: 'ceding_name',
          judul: 'Ceding Name',
          nilai: '',
          dapat_diubah: false,
          terhalang: true,
          alasan_terhalang:
            'Isian ini berada di halaman TreatyInMaster berkelas ' +
            'ASM-FW-GISFW-Int-TREATY_IN dan tidak diisi rule mana pun di export.',
          pemilik_penghalang: 'Tim Pega',
        },
      ],
      tabel: [],
    },
    {
      kode: 'klaim',
      judul: 'Claim Information',
      isian: [
        {
          kunci: 'dla_no_ceding',
          judul: 'DLA No Ceding',
          nilai: 'DLA/2026/0001',
          dapat_diubah: true,
          terhalang: false,
        },
      ],
      tabel: [],
    },
    {
      kode: 'akseptasi',
      judul: 'Acceptation',
      isian: [],
      tabel: [
        {
          kode: 'adjustment_list',
          judul: 'Acceptation',
          kolom: [
            { kunci: 'type', judul: 'Type', dapat_diubah: true },
            { kunci: 'acceptation_no', judul: 'Acceptation No', dapat_diubah: false },
            { kunci: 'status', judul: 'Status', dapat_diubah: false },
          ],
          baris: [
            { type: 'Final', acceptation_no: 'AKS/TNP/2026/0001', status: '1' },
          ],
          terhalang: false,
        },
      ],
    },
  ],
  selisih_terencana: [
    'Submit belum menulis ke basis data selama tabelnya masih dimiliki Pega.',
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

/** Peladen tiruan yang menjawab rincian dan menolak Submit, seperti peladen sungguhan. */
function stubDefaultFetch(submitAnswer?: Response) {
  stubFetch((url, init) => {
    if (url === PATH && (init?.method ?? 'GET') === 'GET') {
      return jsonResponse(200, DETAIL)
    }
    if (url === PATH) {
      return (
        submitAnswer ??
        jsonResponse(409, {
          kode: 'belum_dapat_disimpan',
          pesan:
            'Akseptasi belum dapat disimpan dari sistem baru. Tabel itu masih dimiliki ' +
            'Pega. Perubahan Anda TIDAK tersimpan.',
        })
      )
    }
    return jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ditemukan.' })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/input-acceptation/${CLAIM}`]}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function renderLoaded() {
  renderPage()
  await screen.findByRole('heading', { name: 'Acceptation Claim' })
  await screen.findByRole('heading', { name: 'Treaty Information' })
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

describe('bentuk layar', () => {
  it('memakai judul kontainer Pega, bukan nama menu', async () => {
    // "Acceptation Claim" adalah `<pyValue>` kontainer di section.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('heading', { name: 'Acceptation Claim' }),
    ).toBeInTheDocument()
  })

  it('menggambar kelompok dan isian yang ditetapkan server', async () => {
    // Bentuk layar datang dari katalog di backend, bukan daftar tetap di frontend.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('heading', { name: 'Claim Information' })).toBeInTheDocument()

    // Tingkat judulnya disebut karena kelompok "Acceptation" memuat grid yang di section
    // berjudul sama: kelompok menjadi h2, gridnya h3. Keduanya memang tampil bersamaan.
    expect(
      screen.getByRole('heading', { level: 2, name: 'Acceptation' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { level: 3, name: 'Acceptation' }),
    ).toBeInTheDocument()

    expect(screen.getByText('TNP-2026-01')).toBeInTheDocument()
  })

  it('menyediakan tautan kembali ke antrean asalnya', async () => {
    // Layar ini tidak punya butir menu; satu-satunya pintunya adalah inbox.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('link', { name: /Kembali ke Claims In Progress/ }),
    ).toHaveAttribute('href', '/inbox-claim-treaty-non-prop')
  })
})

describe('isian yang terhalang', () => {
  it('digambar dengan alasannya, bukan disembunyikan', async () => {
    // Menyembunyikannya membuat pengguna yang membandingkan layar ini dengan Pega mengira
    // isiannya hilang.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText('Ceding Name')).toBeInTheDocument()
    expect(screen.getByText('belum tersedia')).toBeInTheDocument()
    expect(screen.getByText(/ASM-FW-GISFW-Int-TREATY_IN/)).toBeInTheDocument()
  })

  it('tidak dapat diisi', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Isian terhalang tidak punya kotak isian sama sekali — bukan kotak yang dimatikan.
    // Kotak yang dimatikan tetap terbaca sebagai "nanti bisa diisi".
    expect(screen.queryByLabelText('Ceding Name')).not.toBeInTheDocument()
  })
})

describe('isian yang dapat diubah', () => {
  it('digambar sebagai kotak isian, yang hanya ditampilkan sebagai teks', async () => {
    // Pembedaannya datang dari server (`dapat_diubah`), bukan ditebak layar. Yang ditebak
    // akan berselisih dengan yang divalidasi server.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByLabelText('DLA No Ceding')).toHaveValue('DLA/2026/0001')
    expect(screen.queryByLabelText('Treaty ID')).not.toBeInTheDocument()
  })
})

describe('tombol Submit', () => {
  it('mati selama tidak ada yang berubah', async () => {
    // Submit yang tidak mengubah apa pun tetap ditolak server, dan penolakan atas
    // permintaan kosong hanya membingungkan.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled()
  })

  it('hidup begitu ada isian yang diubah', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('DLA No Ceding'), 'X')

    expect(screen.getByRole('button', { name: 'Submit' })).toBeEnabled()
  })

  it('mengirim HANYA isian yang diubah, bukan seluruh layar', async () => {
    // Mengirim seluruh isian berarti mengirim balik nilai read-only yang dihitung server,
    // dan server akan menolaknya — padahal pengguna tidak menyentuhnya.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.clear(screen.getByLabelText('DLA No Ceding'))
    await userEvent.type(screen.getByLabelText('DLA No Ceding'), 'DLA/2026/0002')
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    const submit = [...calls]
      .reverse()
      .find((c) => c.url === PATH && c.init?.method === 'POST')
    const body = JSON.parse(String(submit?.init?.body)) as {
      isian: Record<string, string>
      tabel: Record<string, unknown>
    }

    expect(body.isian).toEqual({ dla_no_ceding: 'DLA/2026/0002' })
    expect(body.isian).not.toHaveProperty('treaty_id')
  })

  it('mengirim HANYA kolom grid yang dapat diubah', async () => {
    // Kolom read-only pada grid adalah hasil hitungan server. Mengirimnya balik membuka
    // kemungkinan klien menimpanya.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('DLA No Ceding'), 'X')
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    const submit = [...calls]
      .reverse()
      .find((c) => c.url === PATH && c.init?.method === 'POST')
    const body = JSON.parse(String(submit?.init?.body)) as {
      tabel: Record<string, Array<Record<string, string>>>
    }

    expect(body.tabel['adjustment_list']).toEqual([{ type: 'Final' }])
  })

  it('menampilkan penolakan kepemilikan tabel apa adanya', async () => {
    // Penolakan ini BUKAN cacat: tabelnya masih ditulis Pega (`P-1`). Pesannya menyebut
    // sebab dan jalan keluarnya, dan yang terpenting menyatakan perubahannya TIDAK
    // tersimpan — pengguna yang mengira sudah tersimpan tidak akan mengulanginya di Pega.
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('DLA No Ceding'), 'X')
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByText(/TIDAK tersimpan/)).toBeInTheDocument()
  })

  it('menandai isian yang ditolak server di tempatnya', async () => {
    // Satu pesan di atas form akan memaksa pengguna menebak isian mana yang ditolak.
    stubDefaultFetch(
      jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Permintaan belum benar.',
        detail: [
          { field: 'dla_no_ceding', pesan: 'Nomor DLA tidak dikenali.' },
        ],
      }),
    )
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('DLA No Ceding'), 'X')
    await userEvent.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByText('Nomor DLA tidak dikenali.')).toBeInTheDocument()
    expect(screen.getByLabelText('DLA No Ceding')).toHaveAttribute('aria-invalid', 'true')
  })
})

describe('tabel', () => {
  it('menggambar kolom dan baris yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const table = screen.getByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'Type' })).toBeInTheDocument()
    expect(
      within(table).getByRole('columnheader', { name: 'Acceptation No' }),
    ).toBeInTheDocument()
    expect(within(table).getByText('AKS/TNP/2026/0001')).toBeInTheDocument()
  })
})

describe('selisih terencana', () => {
  it('ditampilkan apa adanya dari server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      await screen.findByText(/Submit belum menulis ke basis data/),
    ).toBeInTheDocument()
  })
})
