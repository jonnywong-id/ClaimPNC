import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SupportingFilesButton } from './SupportingFiles'

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

const checklist = {
  kategori: [{ kode: '1', nama: 'Dokumen Klaim', dokumen: [{ id: '14904', nama: 'Laporan Survei' }] }],
  berkas: [],
}

let posted: { url: string; body: FormData } | null = null

function show() {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    if (init?.method === 'POST') {
      posted = { url, body: init.body as FormData }
      return Promise.resolve(json(201, checklist))
    }
    return Promise.resolve(json(200, checklist))
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <SupportingFilesButton claimID="klaim-1" address={{ tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3 }} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  posted = null
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('SupportingFilesButton', () => {
  it('mengunggah berkas beserta jenis dokumen, catatan, dan alamat adjustment', async () => {
    const user = userEvent.setup()
    show()

    await user.click(screen.getByRole('button', { name: 'Unggah File Penunjang' }))
    await screen.findByRole('option', { name: 'Dokumen Klaim — Laporan Survei' })

    const file = new File(['%PDF'], 'survei.pdf', { type: 'application/pdf' })
    await user.upload(screen.getByLabelText('Berkas 1'), file)
    await user.selectOptions(screen.getByLabelText('Jenis dokumen 1'), '14904')
    await user.type(screen.getByLabelText('Catatan 1'), 'laporan')
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByRole('status')).toHaveTextContent('1 berkas penunjang tersimpan.')
    expect(posted?.url).toContain('/api/registrasi/klaim/klaim-1/adjustment/file-penunjang')
    const body = posted!.body
    expect(body.get('tugas_id')).toBe('tugas-1')
    expect(body.get('objek')).toBe('1')
    expect(body.get('jaminan')).toBe('2')
    expect(body.get('adjustment')).toBe('3')
    expect(body.getAll('jenis_dokumen')).toEqual(['14904'])
    expect(body.getAll('catatan_berkas')).toEqual(['laporan'])
    expect((body.get('berkas') as File).name).toBe('survei.pdf')
  })

  it('menolak Submit tanpa berkas atau tanpa jenis dokumen', async () => {
    const user = userEvent.setup()
    show()

    await user.click(screen.getByRole('button', { name: 'Unggah File Penunjang' }))
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Pilih minimal satu berkas.')

    await user.upload(screen.getByLabelText('Berkas 1'), new File(['x'], 'x.pdf'))
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Pilih jenis dokumen untuk setiap berkas.')
    expect(posted).toBeNull()
  })
})
