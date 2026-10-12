import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CommitteeTransferDialog, prefillPA } from './CommitteeTransferDialog'
import type { Claim, CommitteeNote } from './types'

/** Uji modal "Transfer Claim ke Komite" (ClaimComitee_OC). Seluruh data KARANGAN (`D-69`). */

let calls: { url: string; body: unknown }[] = []

const empty: CommitteeNote = {
  kronologi_kejadian: '', jumlah_kerugian: '', polis_liability: '', remarks: '', remarks_investigasi: '',
  diagnosa: '', kode_diagnosa: '', desc_diagnosa: '', penerima_klaim: '',
}

function line(extra: Record<string, unknown> = {}) {
  return { nilai_propose_sen: 150_000_000, nilai_pengajuan_sen: 200_000_000, status_akseptasi: '', ...extra }
}

function claim(lini: string, lines: unknown[], extra: Record<string, unknown> = {}): Claim {
  return {
    id: 'klaim-1',
    polis: { lini, jenis_bisnis: lini === '002' ? 'PersonalAccident' : 'Fire' },
    objek: [{
      id: 'OBJ1', nama: 'Peserta Contoh',
      coverage: [{
        id: '10001', nama: 'Meninggal Dunia', penyebab_kerugian: 'Kecelakaan', adjustment: lines,
        item: [{ estimasi: [{ sudah_cfs: true }] }],
      }],
    }],
    ...extra,
  } as unknown as Claim
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : undefined })
    return Promise.resolve(
      new Response(JSON.stringify({ klaim: { id: 'klaim-1' } }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

function open(k: Claim, props: { analyst?: boolean; adjustment?: number; stage?: string }, onClose = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  render(
    <QueryClientProvider client={client}>
      <CommitteeTransferDialog
        klaim={k}
        taskID="tugas-1"
        stage={props.stage ?? 'kirim-analis'}
        object={1}
        coverage={1}
        adjustment={props.adjustment}
        receivers={[
          { id: '1', nama: 'Budi', alamat: '', nama_bank: '', nomor_rekening: '' },
          { id: '2', nama: 'Ani', alamat: '', nama_bank: 'BCA', nomor_rekening: '123' },
        ]}
        analyst={props.analyst ?? false}
        onClose={onClose}
      />
    </QueryClientProvider>,
  )
  return onClose
}

describe('CommitteeTransferDialog', () => {
  it('Aksep (analis) menyimpan isian lalu mentransfer adjustment TERAKHIR jaminan', async () => {
    const onClose = open(claim('002', [line({ komite_id: 'KMTN.26.1', status_akseptasi: '1' }), line()]), { analyst: true })
    expect(screen.queryByRole('button', { name: 'Kirim Komite' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Aksep' }))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(calls.map((c) => c.url)).toEqual([
      '/api/registrasi/klaim/klaim-1/jaminan/isian-komite',
      '/api/registrasi/klaim/klaim-1/adjustment/komite',
    ])
    expect(calls[1]?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 1, adjustment: 2 })
  })

  it('tanpa analis dan tanpa baris yang dapat ditransfer: tidak ada Aksep maupun Kirim Komite', () => {
    open(claim('006', [line({ komite_id: 'KMTN.26.1', status_akseptasi: '0' })]), {})
    expect(screen.queryByRole('button', { name: 'Aksep' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim Komite' })).not.toBeInTheDocument()
    expect(screen.getByLabelText('Kronologi Kejadian')).toBeInTheDocument()
  })

  it('selain Travel dan PA: hanya Batal dan Kirim Komite, juga bagi analis', () => {
    open(claim('006', [line()]), { adjustment: 1, analyst: true })
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual(['Batal', 'Kirim Komite'])
  })

  it('Travel: Kirim Komite menunggu penerima berdata bank lengkap, lalu mengirim penerimanya', async () => {
    const onClose = open(claim('005', [line()]), { adjustment: 1 })
    expect(screen.queryByLabelText('Kronologi Kejadian')).not.toBeInTheDocument()
    const send = screen.getByRole('button', { name: 'Kirim Komite' })
    expect(send).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Simpan' })).toBeEnabled()

    await userEvent.selectOptions(screen.getByLabelText(/Penerima Klaim/), '1')
    expect(screen.getByText('Receiver Claim harus di isi')).toBeInTheDocument()
    expect(send).toBeDisabled()

    await userEvent.selectOptions(screen.getByLabelText(/Penerima Klaim/), '2')
    expect(send).toBeEnabled()
    await userEvent.click(send)
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(calls[1]?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 1, adjustment: 1, penerima_klaim: '2' })
  })
})

describe('ValidationProposeValue (PA PHK)', () => {
  function phk(lines: unknown[], cfs: boolean): Claim {
    const k = claim('002', lines) as unknown as { objek: { coverage: { id: string; item: unknown[] }[] }[] }
    k.objek[0]!.coverage[0]!.id = '10010'
    k.objek[0]!.coverage[0]!.item = [{ estimasi: [{ sudah_cfs: cfs }] }]
    return k as unknown as Claim
  }

  it('menampilkan Total Klaim kosong dan CFS belum diunduh, dan menonaktifkan Kirim Komite bila nilainya 0', () => {
    open(phk([line({ nilai_propose_sen: 0, nilai_pengajuan_sen: 0 })], false), {})
    expect(screen.getByText(/Nilai TOTAL KLAIM klaim tidak boleh kosong/)).toBeInTheDocument()
    expect(screen.getByText(/Download CFS sebelum TOTAL KLAIM di ISI/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kirim Komite' })).toBeDisabled()
  })

  it('tanpa pesan dan Kirim Komite aktif bila Total Klaim terisi dan CFS sudah ada', () => {
    open(phk([line()], true), {})
    expect(screen.queryByText(/Nilai TOTAL KLAIM/)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kirim Komite' })).toBeEnabled()
  })
})

describe('Detail Diagnosa (PA)', () => {
  it('Cari mencari kode diagnosa, Pilih mengisi Kode dan Desc Diagnose yang ikut tersimpan', async () => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : undefined })
      const body = url.startsWith('/api/registrasi/diagnosa')
        ? { pilihan: [{ kode: 'S92.0', deskripsi: 'FRACTURE OF CALCANEUS' }] }
        : { klaim: { id: 'klaim-1' } }
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    open(claim('002', [line()]), {})

    await userEvent.type(screen.getByLabelText('Cari Kode / Desc Diagnose'), 'fracture')
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))
    expect(await screen.findByText('FRACTURE OF CALCANEUS')).toBeInTheDocument()
    expect(calls[0]?.url).toBe('/api/registrasi/diagnosa?cari=fracture')

    await userEvent.click(screen.getByRole('button', { name: 'Pilih' }))
    expect(screen.getByLabelText('Kode Diagnose')).toHaveValue('S92.0')
    expect(screen.getByLabelText('Desc Diagnose')).toHaveValue('FRACTURE OF CALCANEUS')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))
    await waitFor(() => expect(calls.some((c) => c.url.endsWith('/jaminan/isian-komite'))).toBe(true))
    expect(calls.find((c) => c.url.endsWith('/jaminan/isian-komite'))?.body).toMatchObject({
      kode_diagnosa: 'S92.0', desc_diagnosa: 'FRACTURE OF CALCANEUS',
    })
  })
})

describe('prefillPA', () => {
  it('mengisi Jumlah Kerugian dan Remaks dari adjustment terakhir bila masih kosong', () => {
    const n = prefillPA(claim('002', [line()], { sudah_transfer_analis: true }), 1, 1, empty)
    expect(n.jumlah_kerugian).toBe('Tertanggung atas nama Peserta Contoh mendapatkan santunan kecelakaan sebesar Rp 1.500.000,-')
    expect(n.remarks).toBe('Klaim diusulkan dibayar sebesar Rp 1.500.000,- sesuai dengan santunan Meninggal Dunia (Kecelakaan).')
  })

  it('memakai Nilai Pengajuan bila Total Klaim 0, tidak menimpa isian, dan diam di luar PA', () => {
    const zero = prefillPA(claim('002', [line({ nilai_propose_sen: 0 })]), 1, 1, empty)
    expect(zero.jumlah_kerugian).toContain('Rp 2.000.000,-')
    expect(zero.remarks).toBe('') // belum ditransfer ke Analyst

    const kept = prefillPA(claim('002', [line()]), 1, 1, { ...empty, jumlah_kerugian: 'Sudah diisi' })
    expect(kept.jumlah_kerugian).toBe('Sudah diisi')

    expect(prefillPA(claim('006', [line()]), 1, 1, empty)).toEqual(empty)
  })
})
