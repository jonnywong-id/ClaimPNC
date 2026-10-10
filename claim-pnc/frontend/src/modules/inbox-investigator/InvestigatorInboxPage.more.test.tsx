import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { InvestigatorTask } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InvestigatorInboxPage } from './InvestigatorInboxPage'

/** Uji tambahan Inbox Investigator: cabang galat, tanggal rusak, dan pengurutan. Data KARANGAN. */

function task(partial: Partial<InvestigatorTask>): InvestigatorTask {
  return {
    referensi: 'REF-1',
    nomor_case: 'PNC-000001',
    nomor_polis: '00.000.2026.00001',
    nama_tertanggung: 'PT Contoh Satu',
    nama_peserta: 'Peserta Satu',
    nama_bisnis: 'Aneka',
    nama_cabang: 'Cabang Satu',
    nama_admin: 'ADMIN1',
    tanggal_pendaftaran: '2026-09-21T02:15:00Z',
    tanggal_survey: '2026-09-22T01:00:00Z',
    lini_bisnis: '002',
    ...partial,
  }
}

type Answer = () => Response | Promise<Response>

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stub(answer: Answer) {
  vi.stubGlobal('fetch', () => Promise.resolve().then(answer))
}

function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // MemoryRouter diperlukan sejak Nomor Case menjadi tautan: layar memakai useNavigate,
  // dan hook itu menuntut Router di atasnya.
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InvestigatorInboxPage />
      </MemoryRouter>
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

describe('galat muat', () => {
  it.each([
    {
      name: 'jaringan putus',
      answer: (() => Promise.reject(new TypeError('putus'))) as Answer,
      title: 'Server Claim PNC tidak dapat dihubungi',
    },
    {
      name: 'portal tidak disebut',
      answer: () => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal tidak dikenal',
      answer: () => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal belum siap',
      answer: () => json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      title: 'Basis data entitas ini belum tersedia',
    },
    {
      name: 'galat API lain',
      answer: () => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }),
      title: 'Antrean investigator tidak dapat dimuat',
      description: 'Basis data sibuk.',
    },
    {
      name: 'galat bukan API',
      answer: () =>
        ({
          get status(): number {
            throw new TypeError('rusak')
          },
        }) as unknown as Response,
      title: 'Terjadi kesalahan pada sistem',
    },
  ])('menampilkan pesan: $name', async ({ answer, title, description }) => {
    stub(answer)
    show()

    expect(screen.getByText('Memuat antrean investigator…')).toBeInTheDocument()
    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })
})

describe('tabel', () => {
  it('menampilkan tanda hubung untuk tanggal yang tidak dapat dibaca', async () => {
    stub(() =>
      json(200, {
        tugas: [task({ tanggal_pendaftaran: 'bukan-tanggal', tanggal_survey: '' })],
 lini_bisnis: '002',
        terpotong: false,
        batas_baris: 500,
        portal: 'ASM',
      }),
    )
    show()

    const row = (await screen.findByText('PNC-000001')).closest('tr')!
    expect(within(row).getAllByText('—')).toHaveLength(2)
  })

  it('mengurutkan menurut setiap kolom', async () => {
    stub(() =>
      json(200, {
        tugas: [
          task({ referensi: 'R1', nomor_case: 'PNC-000002', nama_peserta: 'Zeta' }),
          task({ referensi: 'R2', nomor_case: 'PNC-000001', nama_peserta: 'Alfa' }),
        ],
        terpotong: false,
        batas_baris: 500,
        portal: 'ASM',
      }),
    )
    show()
    const table = await screen.findByRole('table')

    for (const title of [
      'Nomor Case',
      'No Polis',
      'Nama Tertanggung',
      'Nama Peserta',
      'Nama Bisnis',
      'Nama Cabang',
      'Nama Admin',
      'Tanggal Pendaftaran',
      'Lama Masuk Inbox',
    ]) {
      const column = within(table).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(column).getByRole('button'))
      expect(column).toHaveAttribute('aria-sort', 'ascending')
    }
    const peserta = within(table).getByRole('columnheader', { name: /^Nama Peserta/ })
    await userEvent.click(within(peserta).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Alfa')
  })
})
