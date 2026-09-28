import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DokumenPenunjangPanel } from './DokumenPenunjangPanel'
import type { DokumenPenunjang } from './types'

/**
 * Sakelar fitur di-mock supaya KEDUA keadaan tetap diuji.
 *
 * Nilai sebenarnya `false` — modul GCS belum disiapkan. Uji di berkas ini menyalakannya
 * supaya perilaku yang sudah dibangun tidak kehilangan penjaganya; menghidupkannya kembali
 * kelak menjadi langkah yang TERUJI, bukan lompatan.
 *
 * Keadaan MATI — yang berlaku hari ini — diuji di `DokumenPenunjangPanel.mati.test.tsx`,
 * berkas tersendiri karena `vi.mock` berlaku untuk SATU berkas utuh dan tidak dapat
 * dibalik di tengahnya.
 */
vi.mock('./fitur', () => ({ FITUR_DOKUMEN_PENUNJANG_AKTIF: true }))

let calls: { url: string; init: RequestInit | undefined }[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function dokumen(over: Partial<DokumenPenunjang> = {}): DokumenPenunjang {
  return {
    id: 'IMG-1',
    nama_berkas: 'FotoKerugianpdf',
    jenis_dokumen: '',
    url: 'https://contoh.invalid/berkas/IMG-1',
    tanggal_unggah: '26/09/2026 10:30',
    berlaku_sampai: '31/12/2026 23:59',
    kedaluwarsa: false,
    ...over,
  }
}

function stubFetch(answer: (url: string, init: RequestInit | undefined) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    return Promise.resolve(answer(url, init))
  })
}

function renderPanel(nomorKlaim: string | null, readOnly = false) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <DokumenPenunjangPanel nomorKlaim={nomorKlaim} readOnly={readOnly} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-30T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('tanpa nomor klaim', () => {
  // Panel ini muncul pada form yang klaimnya BELUM dipilih. Menembak server tanpa nomor
  // hanya menghasilkan permintaan sia-sia per ketikan.
  it('menjelaskan bahwa klaim harus dipilih dulu, dan tidak menembak server', async () => {
    stubFetch(() => jsonResponse(200, { data: [] }))
    renderPanel(null)

    expect(screen.getByText(/Pilih No Klaim lebih dulu/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Pilih berkas' })).not.toBeInTheDocument()

    await waitFor(() => expect(calls).toHaveLength(0))
  })
})

describe('daftar dokumen', () => {
  it('meminta dokumen KLAIM, bukan dokumen proteksi', async () => {
    stubFetch(() => jsonResponse(200, { data: [dokumen()] }))
    renderPanel('PNC-1865')

    await screen.findByText('FotoKerugianpdf')

    // Jalurnya bersarang di bawah klaim — itu yang membuat dokumen terlihat dari KEDUA
    // layar proteksi.
    expect(calls[0]?.url).toContain('/api/klaim/PNC-1865/dokumen-penunjang')
  })

  it('menyandikan nomor klaim yang memuat karakter khusus', async () => {
    stubFetch(() => jsonResponse(200, { data: [] }))
    // `CLAIMID` warisan berbentuk `ASM-FW-GCNMFW-WORK PNC-1865`; spasinya harus disandikan
    // atau jalurnya pecah menjadi dua segmen.
    renderPanel('ASM-FW-GCNMFW-WORK PNC-1865')

    await waitFor(() => expect(calls).toHaveLength(1))
    expect(calls[0]?.url).toContain('ASM-FW-GCNMFW-WORK%20PNC-1865')
  })

  it('menyatakan kosong, bukan diam, ketika klaim belum punya dokumen', async () => {
    stubFetch(() => jsonResponse(200, { data: [] }))
    renderPanel('PNC-1865')

    expect(await screen.findByText(/Belum ada dokumen penunjang/)).toBeInTheDocument()
  })

  // Badan respons datang dari jaringan, dan `as T` pada callAPI tidak memeriksa apa pun
  // saat berjalan. Respons tanpa `data` tidak boleh menggagalkan seluruh layar.
  it('tidak galat ketika respons kehilangan medan data', async () => {
    stubFetch(() => jsonResponse(200, {}))
    renderPanel('PNC-1865')

    expect(await screen.findByText(/Belum ada dokumen penunjang/)).toBeInTheDocument()
  })

  it('memberi tautan pada dokumen yang masih berlaku', async () => {
    stubFetch(() => jsonResponse(200, { data: [dokumen()] }))
    renderPanel('PNC-1865')

    const tautan = await screen.findByRole('link', { name: 'FotoKerugianpdf' })
    expect(tautan).toHaveAttribute('href', 'https://contoh.invalid/berkas/IMG-1')
  })

  // Tautan mati yang tetap dapat diklik membuat pengguna mengira berkasnya hilang.
  it('TIDAK memberi tautan pada dokumen yang kedaluwarsa, dan menyebut sebabnya', async () => {
    stubFetch(() => jsonResponse(200, { data: [dokumen({ kedaluwarsa: true })] }))
    renderPanel('PNC-1865')

    await screen.findByText('FotoKerugianpdf')
    expect(screen.queryByRole('link', { name: 'FotoKerugianpdf' })).not.toBeInTheDocument()
    expect(screen.getByText(/tautan kedaluwarsa/)).toBeInTheDocument()
  })

  it('membedakan tautan yang BELUM siap dari yang kedaluwarsa', async () => {
    stubFetch(() => jsonResponse(200, { data: [dokumen({ url: '' })] }))
    renderPanel('PNC-1865')

    await screen.findByText('FotoKerugianpdf')
    expect(screen.getByText(/tautan belum siap/)).toBeInTheDocument()
    expect(screen.queryByText(/kedaluwarsa/)).not.toBeInTheDocument()
  })
})

describe('unggah', () => {
  it('mengirim berkas sebagai multipart pada bagian bernama "berkas"', async () => {
    stubFetch((_url, init) => {
      if (init?.method === 'POST') return jsonResponse(201, { data: dokumen() })
      return jsonResponse(200, { data: [] })
    })
    renderPanel('PNC-1865')
    await screen.findByText(/Belum ada dokumen penunjang/)

    const berkas = new File(['isi berkas'], 'Foto Kerugian.pdf', { type: 'application/pdf' })
    // Dicari lewat NAMA AKSESIBELNYA, bukan lewat selector: input ini disembunyikan dan
    // dipicu tombol, sehingga namanya satu-satunya yang membuatnya dapat dipakai pembaca
    // layar. Uji yang menemukannya lewat `querySelector` akan tetap lulus meski namanya
    // hilang.
    await userEvent.upload(screen.getByLabelText('Pilih berkas untuk diunggah'), berkas)

    await waitFor(() => {
      expect(calls.some((c) => c.init?.method === 'POST')).toBe(true)
    })

    const unggah = calls.find((c) => c.init?.method === 'POST')
    const badan = unggah?.init?.body
    expect(badan).toBeInstanceOf(FormData)

    // Nama bagiannya HARUS `berkas` — itu yang dibaca `r.FormFile("berkas")`. Nama lain
    // membuat server menjawab "berkas tidak ditemukan" meski berkasnya jelas terkirim.
    const terkirim = (badan as FormData).get('berkas')
    expect(terkirim).toBeInstanceOf(File)
    expect((terkirim as File).name).toBe('Foto Kerugian.pdf')

    // Content-Type TIDAK disetel tangan: peramban yang mengisinya beserta `boundary`.
    const header = unggah?.init?.headers as Record<string, string> | undefined
    expect(header?.['Content-Type']).toBeUndefined()
  })

  it('menyegarkan daftar dari server setelah unggah, bukan menyisipkan hasilnya', async () => {
    let sudahUnggah = false
    stubFetch((_url, init) => {
      if (init?.method === 'POST') {
        sudahUnggah = true
        // Respons unggah sengaja TANPA url — begitulah layanan sungguhan menjawab.
        return jsonResponse(201, { data: dokumen({ url: '' }) })
      }
      return jsonResponse(200, {
        data: sudahUnggah ? [dokumen({ url: 'https://contoh.invalid/berkas/IMG-1' })] : [],
      })
    })
    renderPanel('PNC-1865')
    await screen.findByText(/Belum ada dokumen penunjang/)

    await userEvent.upload(
      screen.getByLabelText('Pilih berkas untuk diunggah'),
      new File(['isi'], 'surat.pdf', { type: 'application/pdf' }),
    )

    // Yang tampil adalah hasil pembacaan ULANG, yang punya tautan — bukan respons unggah
    // yang tautannya kosong.
    const tautan = await screen.findByRole('link', { name: 'FotoKerugianpdf' })
    expect(tautan).toHaveAttribute('href', 'https://contoh.invalid/berkas/IMG-1')
  })

  // Pesannya menentukan apa yang pengguna lakukan berikutnya, dan pada kegagalan yang satu
  // ini yang benar adalah JANGAN mengulang.
  it('menyampaikan pesan server apa adanya saat unggah gagal', async () => {
    stubFetch((_url, init) => {
      if (init?.method === 'POST') {
        return jsonResponse(500, {
          kode: 'tersimpan_sebagian',
          pesan:
            'Berkas sudah terkirim ke penyimpanan tetapi catatannya gagal disimpan. JANGAN unggah ulang.',
        })
      }
      return jsonResponse(200, { data: [] })
    })
    renderPanel('PNC-1865')
    await screen.findByText(/Belum ada dokumen penunjang/)

    await userEvent.upload(
      screen.getByLabelText('Pilih berkas untuk diunggah'),
      new File(['isi'], 'surat.pdf', { type: 'application/pdf' }),
    )

    expect(await screen.findByText(/JANGAN unggah ulang/)).toBeInTheDocument()
  })

  it('menyembunyikan tombol unggah pada mode baca-saja, daftarnya tetap terbaca', async () => {
    stubFetch(() => jsonResponse(200, { data: [dokumen()] }))
    renderPanel('PNC-1865', true)

    await screen.findByText('FotoKerugianpdf')
    expect(screen.queryByRole('button', { name: 'Pilih berkas' })).not.toBeInTheDocument()
  })
})
