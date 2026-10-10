import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DetailDialog } from './DetailDialog'
import type { DetailResponse } from './types'

/**
 * Uji tambahan popup Detail Klaim, dirender langsung: isian kosong pada ringkasan dan
 * ketiga grid, keadaan memuat, dan galat umum. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-os-claim-per-cabang'

const EMPTY_OBJECT = {
  id: 'OBJ-KOSONG',
  nama: '',
  lokasi: '',
  pekerjaan: '',
  tanggal_lahir: '',
  ktp_paspor: '',
  status_peserta: '',
  coverage: [],
}

function detail(cob: string): DetailResponse {
  return {
    ringkasan: {
      no_klaim: 'PNC-1',
      cob,
      occupation: '',
      total_sum_insured: '0',
      kronologi: '',
      total_reserve: '0',
      aging_hari: 3,
      perlu_perhatian: false,
      dominant_factor: '',
      claim_recommendation: '',
      note_pic: '',
    },
    objek: [EMPTY_OBJECT],
    riwayat_progres: [
      {
        tanggal_input: '',
        no_klaim: '',
        status_progres_1: '',
        status_progres_2: '',
        user_input: '',
        tanggal_next_followup: '',
        status: '',
        keterangan: '',
      },
    ],
    komunikasi_adjuster: [
      { nama_user: '', tanggal_proses: '', pesan: '', tanggal_balas: '', jawaban: '', internal: false },
    ],
    cabang: { kode: '1', nama: 'CABANG' },
    selisih_terencana: [],
    portal: 'ASM',
  }
}

type Answer = () => Response | Promise<Response>

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: Answer) {
  vi.stubGlobal('fetch', (url: string) => {
    if (url === `${PATH}/PNC-1`) return Promise.resolve().then(answer)
    return Promise.resolve(json(404, { kode: 'x', pesan: 'x' }))
  })
}

function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <DetailDialog nomorKlaim="PNC-1" onTutup={() => {}} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('DetailDialog', () => {
  it.each([
    { cob: '', expected: 2 },
    { cob: 'PA', expected: 3 },
    { cob: 'Travel', expected: 4 },
  ])('mengisi tanda hubung pada objek kosong lini "$cob"', async ({ cob, expected }) => {
    installFetch(() => json(200, detail(cob)))
    show()

    expect(screen.getByText('Memuat detail klaim…')).toBeInTheDocument()
    const section = (await screen.findByRole('heading', { name: 'Objek Pertanggungan' })).parentElement!
    const row = within(section).getAllByRole('row')[1]!
    expect(within(row).getAllByText('—')).toHaveLength(expected)
  })

  it('menandai seluruh isian kosong pada ringkasan, progres, dan komunikasi', async () => {
    installFetch(() => json(200, detail('Aneka')))
    show()

    const aging = await screen.findByText('3 hari')
    expect(aging).not.toHaveAttribute('title')
    expect(aging).not.toHaveClass('text-red-700')
    const summary = aging.closest('dl')!
    // COB, Occupation, Dominant Factor, Kronologi, Recommendation, dan Note dari PIC.
    expect(within(summary).getAllByText('—')).toHaveLength(5)

    const progress = screen.getByRole('heading', { name: 'Riwayat Progress' }).parentElement!
    expect(within(within(progress).getAllByRole('row')[1]!).getAllByText('—')).toHaveLength(8)

    const messages = screen.getByRole('heading', { name: 'KOMUNIKASI DENGAN LOSS ADJUSTER' }).parentElement!
    const messageRow = within(messages).getAllByRole('row')[1]!
    expect(within(messageRow).getAllByText('—')).toHaveLength(5)
    expect(within(messageRow).queryByText('internal')).not.toBeInTheDocument()
    expect(screen.queryByText('Perbedaan yang disengaja terhadap layar lama')).not.toBeInTheDocument()
  })

  it('menyebut galat umum dengan nada gangguan', async () => {
    installFetch(() => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }))
    show()

    expect(await screen.findByText('Detail klaim tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()
  })

  it('menyebut server tidak terhubung pada galat jaringan', async () => {
    installFetch(() => Promise.reject('x'))
    show()

    expect(await screen.findByText('Detail klaim tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
  })
})
