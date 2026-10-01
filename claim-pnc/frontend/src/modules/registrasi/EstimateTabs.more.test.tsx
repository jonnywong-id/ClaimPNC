import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { formatDate } from '@/components/format'

import { DocumentTab, ProgressTab, SurveyTab } from './EstimateTabs'

/**
 * Uji tab pendamping Input Estimasi — Survey, Unggah Dokumen, dan Progress — dirender
 * langsung, tanpa layar klaim. Seluruh data KARANGAN (`D-69`).
 */

const BASE = '/api/registrasi/klaim/klaim-1'

type Answer = Response | Promise<Response> | 'putus' | undefined

let calls: { url: string; method: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (url: string, method: string) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method, body: init?.body })
    const custom = answer(url, method)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom) return Promise.resolve(custom)
    return Promise.resolve(json(404, { kode: 'tidak_ditemukan', pesan: 'x' }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('SurveyTab', () => {
  it('menampilkan memuat lalu baris survei dengan tanda hubung untuk isian kosong', async () => {
    installFetch((url) =>
      url === `${BASE}/survey`
        ? json(200, {
            survey: [
              {
                kasus_id: 'ASM-FW-GCNMFW-WORK SRV-1',
                tipe: '2',
                nama_surveyor: '',
                tanggal_survey: '2026-05-01',
                lokasi_survey: 'Lokasi Survei',
                nama_objek: '',
                lokasi_objek: '',
                urutan: '1',
                status: '',
                keterangan: '',
                tanggal_input: '',
              },
              {
                kasus_id: 'SRV-2',
                tipe: '9',
                nama_surveyor: 'BUDI',
                tanggal_survey: '2026-05-02',
                lokasi_survey: '',
                nama_objek: 'Gudang',
                lokasi_objek: 'Blok A',
                urutan: '2',
                status: 'Selesai',
                keterangan: 'Baik',
                tanggal_input: '',
              },
              {
                kasus_id: 'SRV-3',
                tipe: '',
                nama_surveyor: '',
                tanggal_survey: '',
                lokasi_survey: '',
                nama_objek: '',
                lokasi_objek: '',
                urutan: '3',
                status: '',
                keterangan: '',
                tanggal_input: '',
              },
            ],
          })
        : undefined,
    )
    wrap(<SurveyTab claimID="klaim-1" />)

    expect(screen.getByText('Memuat…')).toBeInTheDocument()
    const first = (await screen.findByText('SRV-1')).closest('tr')!
    expect(within(first).getByText('Loss Adjuster')).toBeInTheDocument()
    expect(within(first).getByText('Lokasi Survei')).toBeInTheDocument()
    expect(within(first).getAllByText('—')).toHaveLength(4)

    const second = screen.getByText('SRV-2').closest('tr')!
    // Kode jenis yang tidak dikenal ditampilkan apa adanya.
    expect(within(second).getByText('9')).toBeInTheDocument()
    expect(within(second).getByText('Blok A')).toBeInTheDocument()

    const third = screen.getByText('SRV-3').closest('tr')!
    expect(within(third).getAllByText('—').length).toBeGreaterThanOrEqual(5)
    expect(screen.getByRole('button', { name: 'Ajukan Survey' })).toBeDisabled()
  })

  it('menyebut tabel kosong', async () => {
    installFetch((url) => (url === `${BASE}/survey` ? json(200, { survey: [] }) : undefined))
    wrap(<SurveyTab claimID="klaim-1" />)

    expect(await screen.findByText('Data Tidak Ada')).toBeInTheDocument()
  })

  it.each([
    { name: 'galat API', answer: (): Answer => json(500, { kode: 'x', pesan: 'Survei rusak.' }), text: 'Survei rusak.' },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
  ])('menampilkan galat muat: $name', async ({ answer, text }) => {
    installFetch((url) => (url === `${BASE}/survey` ? answer() : undefined))
    wrap(<SurveyTab claimID="klaim-1" />)

    expect(await screen.findByText('Data tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })
})

const DOCUMENTS = {
  kategori: [
    {
      kode: 'K1',
      nama: 'Dokumen Register',
      dokumen: [
        { id: 'D1', jenis_id: 'J1', nama: 'Formulir Klaim', wajib: true, minimal: '', terunggah: 0 },
        { id: 'D2', jenis_id: 'J2', nama: 'Foto Kerugian', wajib: false, minimal: '2', terunggah: 1 },
      ],
    },
    { kode: 'K2', nama: 'Dokumen Kosong', dokumen: [] },
  ],
  berkas: [
    {
      id: 'B1',
      nama: '',
      jenis_berkas: '',
      catatan: '',
      kategori: '',
      sub_kategori: '',
      tersimpan: true,
      diunggah_oleh: '',
      diunggah_pada: '',
    },
    {
      id: 'B2',
      nama: 'foto.jpg',
      jenis_berkas: 'image/jpeg',
      catatan: 'Foto depan',
      kategori: '',
      sub_kategori: '',
      tersimpan: true,
      diunggah_oleh: 'ADMIN',
      diunggah_pada: '2026-05-01T09:30:00+07:00',
    },
    {
      id: 'B3',
      nama: 'lain.pdf',
      jenis_berkas: 'application/pdf',
      catatan: '',
      kategori: '',
      sub_kategori: '',
      tersimpan: true,
      diunggah_oleh: 'ADMIN',
      diunggah_pada: '2026-05-02',
    },
  ],
}

describe('DocumentTab', () => {
  it('menampilkan checklist, berkas, dan waktu unggah', async () => {
    installFetch((url) => (url === `${BASE}/dokumen` ? json(200, DOCUMENTS) : undefined))
    wrap(<DocumentTab claimID="klaim-1" />)

    const formulir = (await screen.findByText('Formulir Klaim')).closest('tr')!
    expect(within(formulir).getByText('Ya')).toBeInTheDocument()
    // Minimal kosong ditulis 0, berdampingan dengan jumlah terunggah yang juga 0.
    expect(within(formulir).getAllByText('0')).toHaveLength(2)
    expect(within(screen.getByText('Foto Kerugian').closest('tr')!).getByText('Tidak')).toBeInTheDocument()
    expect(screen.getByText('Data Tidak Ada')).toBeInTheDocument()

    const empty = screen.getByText('(tanpa nama)').closest('tr')!
    expect(within(empty).getAllByText('—')).toHaveLength(4)
    expect(within(screen.getByText('foto.jpg').closest('tr')!).getByText(`${formatDate('2026-05-01')} 09:30`)).toBeInTheDocument()
    expect(within(screen.getByText('lain.pdf').closest('tr')!).getByText(formatDate('2026-05-02'))).toBeInTheDocument()
  })

  it('mengunggah berkas dengan catatan lalu menutup panel', async () => {
    installFetch((url) => (url === `${BASE}/dokumen` ? json(200, DOCUMENTS) : undefined))
    wrap(<DocumentTab claimID="klaim-1" />)

    const row = (await screen.findByText('Formulir Klaim')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Unggah Dokumen' }))
    expect(screen.getByText('Unggah Dokumen: Formulir Klaim')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Unggah' })).toBeDisabled()

    const file = new File(['isi'], 'formulir.pdf', { type: 'application/pdf' })
    await userEvent.upload(screen.getByLabelText('Berkas'), file)
    await userEvent.type(screen.getByLabelText('Catatan'), '  asli  ')
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    await waitFor(() =>
      expect(screen.queryByText('Unggah Dokumen: Formulir Klaim')).not.toBeInTheDocument(),
    )
    const sent = calls.find((c) => c.method === 'POST')
    expect(sent?.url).toBe(`${BASE}/dokumen`)
    const form = sent?.body as FormData
    expect(form.get('jenis_dokumen')).toBe('D1')
    expect(form.get('catatan')).toBe('asli')
    expect((form.get('berkas') as File).name).toBe('formulir.pdf')
  })

  it('menampilkan galat unggah dan menutup lewat Batal atau tombol yang sama', async () => {
    installFetch((url, method) => {
      if (url !== `${BASE}/dokumen`) return undefined
      return method === 'POST' ? json(413, { kode: 'x', pesan: 'Berkas terlalu besar.' }) : json(200, DOCUMENTS)
    })
    wrap(<DocumentTab claimID="klaim-1" />)

    const row = (await screen.findByText('Foto Kerugian')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Unggah Dokumen' }))
    await userEvent.upload(screen.getByLabelText('Berkas'), new File(['x'], 'besar.jpg'))
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    expect(await screen.findByText('Dokumen tidak dapat diunggah')).toBeInTheDocument()
    expect(screen.getByText('Berkas terlalu besar.')).toBeInTheDocument()
    // Tanpa catatan, bagian catatan tidak dikirim.
    expect((calls.find((c) => c.method === 'POST')?.body as FormData).has('catatan')).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByText('Unggah Dokumen: Foto Kerugian')).not.toBeInTheDocument()

    await userEvent.click(within(row).getByRole('button', { name: 'Unggah Dokumen' }))
    expect(screen.getByText('Unggah Dokumen: Foto Kerugian')).toBeInTheDocument()
    await userEvent.click(within(row).getByRole('button', { name: 'Unggah Dokumen' }))
    expect(screen.queryByText('Unggah Dokumen: Foto Kerugian')).not.toBeInTheDocument()
  })

  it('menampilkan galat muat checklist', async () => {
    installFetch((url) => (url === `${BASE}/dokumen` ? json(500, { kode: 'x', pesan: 'Checklist rusak.' }) : undefined))
    wrap(<DocumentTab claimID="klaim-1" />)

    expect(await screen.findByText('Checklist rusak.')).toBeInTheDocument()
  })
})

describe('ProgressTab', () => {
  it('menampilkan progres dan komunikasi dengan nama, kode, atau tanda hubung', async () => {
    installFetch((url) =>
      url === `${BASE}/progres`
        ? json(200, {
            progres: [
              {
                urutan: 1,
                tanggal_input: '2026-05-01T08:00:00+07:00',
                status_1: '10',
                status_1_nama: 'Survey',
                status_2: '20',
                status_2_nama: '',
                keterangan: '',
                tindak_lanjut: '',
                diinput_oleh: '',
              },
              {
                urutan: 2,
                tanggal_input: '',
                status_1: '',
                status_1_nama: '',
                status_2: '',
                status_2_nama: '',
                keterangan: 'Catatan',
                tindak_lanjut: '2026-05-03',
                diinput_oleh: 'ADMIN',
              },
            ],
            komunikasi: [
              {
                kasus_id: 'ASM-FW-GCNMFW-WORK PNC-1',
                id: 'C1',
                tanggal: '',
                pengirim: '',
                pesan: '',
                balasan: '',
                penjawab: '',
                tanggal_balasan: '',
              },
              {
                kasus_id: 'PNC-2',
                id: 'C2',
                tanggal: '2026-05-04',
                pengirim: 'ADJ',
                pesan: 'Mohon dokumen',
                balasan: 'Sudah',
                penjawab: 'ADMIN',
                tanggal_balasan: '',
              },
            ],
          })
        : undefined,
    )
    wrap(<ProgressTab claimID="klaim-1" />)

    const first = (await screen.findByText('Survey')).closest('tr')!
    expect(within(first).getByText('20')).toBeInTheDocument()
    expect(within(first).getAllByText('—')).toHaveLength(3)
    const second = screen.getByText('Catatan').closest('tr')!
    expect(within(second).getAllByText('—')).toHaveLength(3)

    const c1 = screen.getByText('PNC-1').closest('tr')!
    expect(within(c1).getAllByText('—')).toHaveLength(5)
    expect(screen.getByText('Mohon dokumen')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Input Progress Claim' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Kirim Pesan' })).toBeDisabled()
  })

  it('menyebut kedua tabel kosong', async () => {
    installFetch((url) => (url === `${BASE}/progres` ? json(200, { progres: [], komunikasi: [] }) : undefined))
    wrap(<ProgressTab claimID="klaim-1" />)

    await waitFor(() => expect(screen.getAllByText('Data Tidak Ada')).toHaveLength(2))
  })

  it('menampilkan pesan umum untuk galat yang tidak membawa pesan', async () => {
    installFetch((url) => (url === `${BASE}/progres` ? new Promise<Response>((_, reject) => reject('x')) : undefined))
    wrap(<ProgressTab claimID="klaim-1" />)

    expect(await screen.findByText('Data tidak dapat dimuat')).toBeInTheDocument()
  })
})
