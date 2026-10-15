import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { usePrintLOD } from './api'

/**
 * Print LOD menulis PDFTYPE dan PRINTLOD_DATE baris adjustment. Tanpa memuat ulang klaim,
 * kolom Adjustment grid dan form AcceptationLOD tetap menampilkan Tipe PDF dan Tanggal Cetak
 * LOD yang kosong sampai halaman dibuka ulang.
 */
describe('usePrintLOD', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('memuat ulang klaim sesudah LOD berhasil dicetak', async () => {
    // Badannya STRING, bukan `new Blob([...])`.
    //
    // Response milik undici menolak Blob bentukan jsdom, dan penolakannya terjadi DI DALAM
    // `try` pembungkus fetch di `unduhBerkas` — sehingga ia tertelan dan muncul sebagai
    // NetworkError, seolah tiruannya tidak terpasang. Seluruh uji unduhan di
    // `src/api/client.test.ts` memakai badan string karena sebab yang sama.
    vi.stubGlobal('fetch', () =>
      Promise.resolve(
        new Response('%PDF', {
          status: 200,
          headers: { 'Content-Type': 'application/pdf', 'Content-Disposition': 'attachment; filename="LOD.pdf"' },
        }),
      ),
    )
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )
    const { result } = renderHook(() => usePrintLOD('klaim-1'), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ tugas_id: 't', objek: 1, jaminan: 1, adjustment: 1, tipe_pdf: '3' })
    })

    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['registrasi', 'klaim', 'klaim-1'] })
  })
})
