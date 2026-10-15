import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InsuredDataSection } from './InsuredData'
import type { Claim } from './types'

/** Blok DATA TERTANGGUNG KLAIM (InputRegisterDetail, IsPA). Seluruh data KARANGAN (`D-69`). */

const CLAIM = {
  id: 'klaim-1',
  polis: {
    nomor: 'POL-CONTOH', lini: '002', nama_lini: 'Personal Accident', jenis_bisnis: 'PA',
    mulai_pertanggungan: '2023-02-15', akhir_pertanggungan: '2066-02-15', mata_uang: 'IDR',
    nama_tertanggung: 'TERTANGGUNG CONTOH', deklarasi: false, penjamin_kredit: false,
    nama_sumbis: 'SUMBIS CONTOH', nama_bisnis: 'PERSONAL ACCIDENT',
  },
} as unknown as Claim

const PROFILE = {
  no_ktp: '3171000000000001',
  alamat: [{
    jenis: '1', nama_jenis: 'Alamat Rumah', alamat: 'Jl. Contoh No. 1', kota: 'C1', nama_kota: 'KOTA CONTOH',
    kecamatan: 'D1', nama_kecamatan: 'KECAMATAN CONTOH', kelurahan: 'R1', nama_kelurahan: 'KELURAHAN CONTOH',
    kode_pos: '10000',
    telepon: [
      { jenis: '1', nama_jenis: 'Telepon Biasa', kode: '', nomor: '0800000000', ekstensi: '' },
      { jenis: '', nama_jenis: 'Email', kode: '', nomor: 'contoh@contoh.example', ekstensi: '' },
    ],
  }],
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function Harness() {
  const [ktp, setKTP] = useState('')
  const [hp, setHP] = useState('')
  const [email, setEmail] = useState('')
  return <InsuredDataSection klaim={CLAIM} idCard={ktp} onIDCard={setKTP} phone={hp} onPhone={setHP} email={email} onEmail={setEmail} />
}

function wrap() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}><Harness /></QueryClientProvider>)
}

beforeEach(() => {
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('InsuredDataSection', () => {
  it('menampilkan polis, alamat, dan telepon dari CIF, serta mengisi No KTP bawaan', async () => {
    vi.stubGlobal('fetch', (url: string) =>
      Promise.resolve(url === '/api/registrasi/klaim/klaim-1/tertanggung' ? json(200, PROFILE) : json(404, {})))
    wrap()

    const region = screen.getByRole('region', { name: 'DATA TERTANGGUNG KLAIM' })
    expect(within(region).getByText('POL-CONTOH')).toBeInTheDocument()
    expect(within(region).getByText('SUMBIS CONTOH')).toBeInTheDocument()
    expect(within(region).getByRole('button', { name: 'Cari Polis' })).toBeDisabled()

    expect(await screen.findByText('ALAMAT RUMAH')).toBeInTheDocument()
    expect(screen.getByText('Jl. Contoh No. 1')).toBeInTheDocument()
    expect(screen.getByText('KELURAHAN CONTOH')).toBeInTheDocument()
    const phones = screen.getByRole('table', { name: 'Telephone dan Email' })
    expect(within(phones).getByText('Telepon Biasa')).toBeInTheDocument()
    expect(within(phones).getByText('contoh@contoh.example')).toBeInTheDocument()

    await waitFor(() => expect(screen.getByLabelText(/No KTP/)).toHaveValue('3171000000000001'))
    expect(screen.getByLabelText('No. HP')).toHaveValue('')
  })

  it('tetap tampil bila CIF tidak dapat dimuat', async () => {
    vi.stubGlobal('fetch', () => Promise.resolve(json(500, { kode: 'x', pesan: 'gagal' })))
    wrap()
    expect(await screen.findByText('Data alamat tertanggung tidak dapat dimuat.')).toBeInTheDocument()
    expect(screen.getByText('Data Tidak Ada')).toBeInTheDocument()
  })
})
