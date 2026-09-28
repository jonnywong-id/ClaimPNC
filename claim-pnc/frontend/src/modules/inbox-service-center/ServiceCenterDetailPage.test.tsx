import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse } from './types'

const ID = 'SC-000101'
const PATH = `/api/inbox-service-center/${ID}`

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Petugas Service Center',
  jenis: 'KARYAWAN',
  login: 'picsc',
  email: 'contoh.picsc@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/** Seluruh isinya KARANGAN — `D-69` berlaku untuk data uji sama seperti untuk dokumen. */
const DETAIL: DetailResponse = {
  klaim: {
    id: ID,
    repair_id: '1000101',
    no_polis: '90-001-2026-00000101',
    nasabah: 'Contoh Nasabah Satu',
    customer_name: 'Contoh Nasabah Satu',
    tanggal_input: '2026-09-22',
    model: 'Contoh Model X',
    no_imei: 'IMEI-CONTOH-001',
    symptom_description: 'Layar tidak menyala setelah terjatuh',
    total_biaya: '2214500',
    charger_adaptor: 'Y',
    repair_status_label: 'Repair Submitted',
    status_approval_label: 'Belum diajukan',
    // Sengaja kosong — layar harus menggambarnya sebagai tanda pisah, bukan sel kosong.
    alasan_batal: '',
    detail_part_json: '[{"part":"LCD","harga":1800000}]',
  },
  riwayat_progres: [
    {
      tanggal: '2026-09-22',
      catatan: 'Unit diterima di gerai, kelengkapan dicek.',
      oleh: 'PICSERVICECENTER',
    },
  ],
  kelompok: [
    {
      kode: 'general',
      judul: 'General Information',
      isian: [
        { kunci: 'id', judul: 'ID' },
        { kunci: 'no_polis', judul: 'NO. POLIS' },
        { kunci: 'tanggal_input', judul: 'Input Date' },
      ],
    },
    {
      kode: 'unit',
      judul: 'Informasi Unit',
      isian: [
        { kunci: 'model', judul: 'MODEL' },
        { kunci: 'no_imei', judul: 'NO. IMEI' },
      ],
    },
    {
      kode: 'perbaikan',
      judul: 'Informasi Perbaikan',
      isian: [
        { kunci: 'symptom_description', judul: 'SYMPTOM DESCRIPTION' },
        { kunci: 'alasan_batal', judul: 'ALASAN BATAL' },
      ],
    },
  ],
  portal: 'ASM',
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

function renderDetail() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/inbox-service-center/${ID}`]}>
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
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('bentuk layar rincian', () => {
  it('menggambar kelompok yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubFetch((url) => (url === PATH ? jsonResponse(200, DETAIL) : jsonResponse(404, {})))
    renderDetail()

    for (const judul of ['General Information', 'Informasi Unit', 'Informasi Perbaikan']) {
      expect(await screen.findByText(judul)).toBeInTheDocument()
    }
  })

  it('memakai judul isian dari Pega apa adanya', async () => {
    stubFetch((url) => (url === PATH ? jsonResponse(200, DETAIL) : jsonResponse(404, {})))
    renderDetail()

    // Huruf besar semua, dan tidak diterjemahkan — `D-13`.
    expect(await screen.findByText('NO. POLIS')).toBeInTheDocument()
    expect(screen.getByText('SYMPTOM DESCRIPTION')).toBeInTheDocument()
  })

  it('memformat tanggal dan menandai isian kosong dengan tanda pisah', async () => {
    stubFetch((url) => (url === PATH ? jsonResponse(200, DETAIL) : jsonResponse(404, {})))
    renderDetail()

    // Muncul DUA kali dan itu memang benar: sekali sebagai isian "Input Date", sekali di
    // riwayat progres. Yang diuji adalah bentuknya sudah diformat, bukan jumlahnya.
    expect((await screen.findAllByText('22 September 2026')).length).toBeGreaterThanOrEqual(1)
    expect(screen.queryByText('2026-09-22')).not.toBeInTheDocument()

    // Sel kosong tidak dapat dibedakan dari isian yang gagal dimuat.
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(1)
  })

  it('menampilkan riwayat progres', async () => {
    stubFetch((url) => (url === PATH ? jsonResponse(200, DETAIL) : jsonResponse(404, {})))
    renderDetail()

    expect(await screen.findByText(/Unit diterima di gerai/)).toBeInTheDocument()
  })

  it('menampilkan Detail Part apa adanya, beserta alasannya', async () => {
    stubFetch((url) => (url === PATH ? jsonResponse(200, DETAIL) : jsonResponse(404, {})))
    renderDetail()

    expect(await screen.findByText(/"part":"LCD"/)).toBeInTheDocument()
    expect(screen.getByText(/belum diuraikan menjadi tabel/i)).toBeInTheDocument()
  })

  it('menyediakan jalan kembali ke daftar', async () => {
    stubFetch((url) => (url === PATH ? jsonResponse(200, DETAIL) : jsonResponse(404, {})))
    renderDetail()

    const kembali = await screen.findByRole('link', { name: /Kembali ke Inbox Service Center/i })
    expect(kembali).toHaveAttribute('href', '/inbox-service-center')
  })
})

describe('klaim yang tidak dapat dibuka', () => {
  it('menjelaskan 404 sebagai klaim tidak ditemukan, bukan sebagai gangguan sistem', async () => {
    stubFetch(() =>
      jsonResponse(404, {
        kode: 'klaim_tidak_ditemukan',
        pesan: 'Klaim tidak ditemukan, atau bukan klaim yang Anda tangani.',
      }),
    )
    renderDetail()

    expect(await screen.findByText('Klaim tidak ditemukan')).toBeInTheDocument()
  })
})
