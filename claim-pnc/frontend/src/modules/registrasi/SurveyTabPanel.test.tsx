import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SurveyTabPanel } from './SurveyTabPanel'
import type { Claim, SurveyObject, SurveyTabResponse } from './types'

/** Uji grid Tambah Survey (InputSurvey). Seluruh data KARANGAN (`D-69`). */

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function object(extra: Partial<SurveyObject>): SurveyObject {
  return {
    objek_id: '1', urutan: 1, nama_objek: 'GUDANG A', lokasi_objek: 'JAKARTA', pilih: false, lokasi_survey: '',
    tipe_surveyor: '', nama_surveyor: '', login_surveyor: '', alamat_surveyor: '', email_surveyor: '',
    kode_cabang: '', nama_cabang: '', nama_surveyor_marine: '', login_surveyor_marine: '', status: '',
    ...extra,
  }
}

let tab: SurveyTabResponse
let posted: { url: string; body: unknown }[]

beforeEach(() => {
  posted = []
  tab = { objek: [object({})], tambah_tampil: true, anggota_koasuransi: false, marine_hull: false, pa: false }
  useSession.setState({ token: 'token' } as never)
  useSelectedPortal.setState({ alias: 'ASM' } as never)
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    if (url.endsWith('/survey/tab')) return Promise.resolve(json(200, tab))
    if (url.startsWith('/api/registrasi/surveyor')) {
      return Promise.resolve(json(200, [{ id: 'S1', nama: 'PT ADJUSTER CONTOH', login: 'ADJ1', alamat: 'JL CONTOH', cabang: '', nama_cabang: '', email: '', kontak: '' }]))
    }
    if (url.endsWith('/survey')) return Promise.resolve(json(200, { survey: [] }))
    if (init?.method === 'POST') {
      posted.push({ url, body: JSON.parse(String(init.body)) })
      return Promise.resolve(json(200, { ...tab, survey_terbit: ['SRVN.26.1'], komite_terbit: [], otomatis: false }))
    }
    return Promise.resolve(json(404, { kode: 'x', pesan: 'x' }))
  })
})

afterEach(() => vi.unstubAllGlobals())

function wrap(node: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<QueryClientProvider client={client}>{node}</QueryClientProvider>)
}

const klaim = { id: 'PNCN.26.1', user_teknis: 'PIC1', pelapor: { nama: 'PELAPOR', telepon: '0800', email: '', alamat: '', hubungan: 0, hubungan_lainnya: '' } } as unknown as Claim

describe('Tambah Survey', () => {
  it('tipe surveyor pada objek yang tidak dipilih menolak "Harus Pilih Object" dan kembali ke Internal', async () => {
    wrap(<SurveyTabPanel klaim={klaim} taskID="T1" />)
    const type = await screen.findByRole('combobox', { name: 'Tipe Surveyor GUDANG A' })
    await userEvent.selectOptions(type, '2')
    expect(screen.getByText('Harus Pilih Object')).toBeInTheDocument()
    expect(type).toHaveValue('1')
    expect(screen.getByRole('button', { name: 'Transfer Survei' })).toBeInTheDocument()
  })

  it('Loss Adjuster pada objek terpilih menampilkan Transfer Komite; Internal menampilkan Transfer Survei', async () => {
    wrap(<SurveyTabPanel klaim={klaim} taskID="T1" />)
    await userEvent.click(await screen.findByRole('checkbox', { name: 'Pilih GUDANG A' }))
    const type = screen.getByRole('combobox', { name: 'Tipe Surveyor GUDANG A' })
    await userEvent.selectOptions(type, '2')
    expect(screen.getByRole('button', { name: 'Transfer Komite' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Transfer Survei' })).not.toBeInTheDocument()

    await userEvent.selectOptions(type, '1')
    await userEvent.click(screen.getByRole('button', { name: 'Transfer Survei' }))
    expect(await screen.findByText(/Survey terbit: SRVN.26.1/)).toBeInTheDocument()
    expect(posted[0]!.url).toContain('/survey/transfer')
    expect(posted[0]!.body).toMatchObject({ tugas_id: 'T1', tipe_surveyor_kasus: '1' })
  })

  it('anggota koasuransi memakai Transfer Survei untuk Loss Adjuster', async () => {
    tab = { ...tab, anggota_koasuransi: true }
    wrap(<SurveyTabPanel klaim={klaim} taskID="T1" />)
    await userEvent.click(await screen.findByRole('checkbox', { name: 'Pilih GUDANG A' }))
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Tipe Surveyor GUDANG A' }), '2')
    expect(screen.getByRole('button', { name: 'Transfer Survei' })).toBeInTheDocument()
  })

  it('Batal Survei hanya pada status Sedang Proses dan meminta konfirmasi', async () => {
    tab = { ...tab, objek: [object({ status: '3', id_survey: 'SRVN.26.7', tipe_surveyor: '1' })] }
    wrap(<SurveyTabPanel klaim={klaim} taskID="T1" />)
    await userEvent.click(await screen.findByRole('button', { name: 'Batal Survei' }))
    expect(screen.getByText('Apakah anda yakin ingin membatalkan survey ini?')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Ya' }))
    expect(posted[0]!.url).toContain('/survey/batal')
    expect(posted[0]!.body).toEqual({ tugas_id: 'T1', objek_id: '1' })
  })

  it('sub-tab Tambah Survey tidak tampil untuk PncAdmin', async () => {
    tab = { ...tab, tambah_tampil: false }
    wrap(<SurveyTabPanel klaim={klaim} taskID="T1" />)
    // Setelah tab dimuat, sub-tab pertama yang tampil adalah Permintaan Survey.
    expect(await screen.findByText('Survey Atas Permintaan')).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Tambah Survey' })).not.toBeInTheDocument()
  })
})
