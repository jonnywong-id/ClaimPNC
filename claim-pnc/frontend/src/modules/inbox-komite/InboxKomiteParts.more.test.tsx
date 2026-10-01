import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, render, renderHook, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MemoryRouter, Route, Routes } from 'react-router-dom'

import { APIError, NetworkError } from '@/api/client'
import type { KomiteAdjustmentLine, KomiteCase, KomiteTransferDetail } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AttachmentTab, ClaimSheet, PolicyTab } from './ClaimSheet'
import { CommitteeBottom } from './CommitteeBottom'
import { DecisionPanel } from './DecisionPanel'
import { InboxKomitePage } from './InboxKomitePage'
import { KmtnDecisionForm } from './KmtnDecisionForm'
import { KomiteCasePage } from './KomiteCasePage'
import { useDecide } from './api'
import { formatPersen, formatTanggal, formatWaktu } from './format'

/**
 * Uji tambahan untuk bagian-bagian Inbox Komite yang dirender langsung, tanpa merakit
 * seluruh aplikasi. Seluruh nilai KARANGAN (`D-69`).
 */

const KASUS: KomiteCase = {
  nomor_case: 'K-3001',
  nomor_klaim: 'PNCN.26.0300',
  nomor_polis: 'CONTOH-PL-0300',
  nama_tertanggung: 'PT Kasus Contoh',
  nama_bisnis: 'Property All Risk',
  sumber_bisnis: 'Direct',
  cabang: 'Bandung',
  aging_komite: 2,
  penjenjangan: {
    kesimpulan: 'menunggu',
    jumlah_jenjang: 2,
    jenjang_kini: 1,
    jenjang_disetujui: 0,
    jumlah_jenjang_belum_diketahui: false,
    selesai: false,
    sudah_saya_putuskan: false,
    keputusan: [],
  },
}

/** Kasus yang seluruh kolom kepalanya kosong — setiap sel jatuh ke tanda pisah. */
const KASUS_KOSONG: KomiteCase = {
  ...KASUS,
  nomor_klaim: '',
  nomor_polis: '',
  nama_tertanggung: '',
  nama_bisnis: '',
  sumber_bisnis: '',
  cabang: '',
}

function line(over: Partial<KomiteAdjustmentLine> = {}): KomiteAdjustmentLine {
  return {
    id_objek: 'OBJ-1',
    id_coverage: 'COV-1',
    nilai_gross: '0',
    nilai_usulan: '0',
    nilai_akseptasi: '0',
    nilai_salvage: '0',
    nilai_asm_share: '0',
    nilai_risiko_sendiri: '0',
    ex_gratia: false,
    ...over,
  } as KomiteAdjustmentLine
}

function transfer(over: Partial<KomiteTransferDetail> = {}): KomiteTransferDetail {
  return {
    judul: 'CLAIM COMMITTEE',
    he_dapat_dinilai: false,
    baris: [],
    komite: null,
    klaim: null,
    coverage: [],
    kosong: false,
    nilai_uang_kosong: false,
    ...over,
  }
}

// ── format ─────────────────────────────────────────────────────────────────────────

describe('format', () => {
  it('menulis tanda pisah untuk tanggal kosong maupun yang tidak terbaca', () => {
    expect(formatTanggal(undefined)).toBe('—')
    expect(formatTanggal('bukan-tanggal')).toBe('—')
    expect(formatWaktu('')).toBe('—')
    expect(formatWaktu('bukan-tanggal')).toBe('—')
  })

  it('menampilkan tanggal dalam zona WIB', () => {
    // 20:00 UTC adalah 03:00 WIB keesokan harinya.
    expect(formatTanggal('2026-09-11T20:00:00Z')).toBe(
      new Date('2026-09-11T20:00:00Z').toLocaleDateString('id-ID', {
        timeZone: 'Asia/Jakarta',
        day: '2-digit',
        month: 'short',
        year: 'numeric',
      }),
    )
    expect(formatTanggal('2026-09-11T20:00:00Z')).toContain('12')
    expect(formatWaktu('2026-09-11T20:00:00Z')).toContain('03')
  })

  it('menulis persen apa adanya atau tanda pisah saat kosong', () => {
    expect(formatPersen(' 12.5 ')).toBe('12.5 %')
    expect(formatPersen('  ')).toBe('—')
    expect(formatPersen(undefined)).toBe('—')
  })
})

// ── ClaimSheet ─────────────────────────────────────────────────────────────────────

describe('ClaimSheet', () => {
  /** Mengambil isi sel `<dd>` yang berlabel tertentu. */
  function valueOf(label: string, scope: HTMLElement = document.body) {
    const dt = within(scope).getAllByText(label)[0]!
    return dt.nextElementSibling?.textContent ?? ''
  }

  it('jatuh ke data kasus lalu tanda pisah ketika rincian transfer belum ada', () => {
    render(<ClaimSheet kasus={KASUS_KOSONG} />)

    expect(valueOf('INSURED')).toBe('—')
    expect(valueOf('CLASS OF INSURANCE')).toBe('—')
    expect(valueOf('PERIOD OF INSURANCE')).toBe('—')
    expect(valueOf('NATURE OF LOSS')).toBe('—')
    expect(valueOf('RESERVES')).toBe('—')
    expect(valueOf('LEADER')).toBe('—')
    expect(screen.getByText(/belum punya barisnya/)).toBeInTheDocument()
    expect(screen.queryByText('EX GRATIA')).not.toBeInTheDocument()
  })

  it('menggambar travel tanpa NATURE OF LOSS dan coverage, dengan label CURRENCY', () => {
    render(
      <ClaimSheet
        kasus={KASUS}
        transfer={transfer({
          klaim: { group_panel: '005', kode_mata_uang: 'USD', ex_gratia: 'YA' },
          coverage: [{ nilai_tsi: '100', analisis_terisi: false, nama_coverage: 'Bagasi' }],
        })}
      />,
    )

    expect(screen.getByText('DATE OF EVENT')).toBeInTheDocument()
    expect(screen.queryByText('DATE OF ACCIDENT')).not.toBeInTheDocument()
    expect(screen.queryByText('NATURE OF LOSS')).not.toBeInTheDocument()
    expect(screen.queryByText('Bagasi')).not.toBeInTheDocument()
    expect(valueOf('CURRENCY')).toBe('USD')
    expect(valueOf('EX GRATIA')).toBe('YA')
  })

  it('merangkai sebab kerugian tanpa pengulangan dan menamai coverage tanpa nama', () => {
    render(
      <ClaimSheet
        kasus={KASUS}
        transfer={transfer({
          klaim: { group_panel: '006', kelas_asuransi: 'Fire', tertanggung: 'PT Dari Klaim' },
          polis: { mulai: '2026-01-01T00:00:00Z', berakhir: '2026-12-31T00:00:00Z' },
          coverage: [
            { id_objek: 'O1', id_coverage: 'C1', nilai_tsi: '1000', sebab_kerugian: ' Banjir ', analisis_terisi: true, nama_coverage: 'Banjir' },
            { id_objek: 'O1', id_coverage: 'C2', nilai_tsi: '2000', sebab_kerugian: 'Banjir', analisis_terisi: true },
            { id_objek: 'O2', id_coverage: 'C3', nilai_tsi: '3000', sebab_kerugian: 'Kebakaran', analisis_terisi: true, nama_coverage: 'Api' },
            { id_objek: 'O3', id_coverage: 'C4', nilai_tsi: '0', analisis_terisi: false, nama_coverage: 'Tanpa sebab' },
          ],
        })}
      />,
    )

    expect(valueOf('NATURE OF LOSS')).toBe('Banjir, Kebakaran')
    expect(screen.getByText('Coverage tanpa nama')).toBeInTheDocument()
    // Kelas asuransi dari klaim menggantikan nama bisnis kasus, di kedua tempatnya.
    expect(screen.getAllByText('Fire')).toHaveLength(2)
    expect(valueOf('INSURED')).toBe('PT Dari Klaim')
    expect(valueOf('PERIOD OF INSURANCE')).toContain('S/D')
  })

  it('menggambar CO MEMBER, spreading empat kolom, dan Fac-Out per baris berlabel', () => {
    render(
      <ClaimSheet
        kasus={KASUS}
        transfer={transfer({
          leader: { nama: 'PT Leader Contoh', persen: '60' },
          spreading_lengkap: true,
          fac_out: [{ reasuradur: 'Reas Fac Satu', persen_polis: '15' }, { reasuradur: 'Reas Fac Dua' }],
          faktor_dominan: ['Cuaca ekstrem', 'Kelalaian'],
          baris: [
            line({
              kode_mata_uang: 'IDR',
              nilai_komite: '1000000',
              co_member: [{ nama: 'PT Anggota', mata_uang: 'IDR', persen: '40', nilai: '400000' }],
              spreading: [
                { jenis_treaty: 'OR', nama_treaty: 'Own Retention', mata_uang: 'IDR', persen: '70', nilai: '700000' },
                { jenis_treaty: 'QS', persen: '30' },
              ],
            }),
            line({ id_objek: '', id_coverage: '' }),
          ],
        })}
      />,
    )

    expect(valueOf('LEADER')).toMatch(/PT Leader Contoh\s+60 %/)
    // Dua baris adjustment diberi label objek/coverage masing-masing.
    expect(screen.getByText(/Objek OBJ-1 · Coverage COV-1/)).toBeInTheDocument()
    expect(screen.getByText('Objek — · Coverage —')).toBeInTheDocument()
    expect(screen.getByText('PT Anggota')).toBeInTheDocument()
    expect(screen.getByText('Own Retention')).toBeInTheDocument()
    // Nama treaty kosong jatuh ke jenisnya; nilai kosong ditulis tanda pisah.
    expect(screen.getByText('QS')).toBeInTheDocument()
    expect(screen.getAllByText('Reas Fac Satu')).toHaveLength(2)
    expect(screen.getAllByText('IDR').length).toBeGreaterThan(0)
    expect(screen.getByText('Cuaca ekstrem')).toBeInTheDocument()
    expect(screen.getAllByText(/nilai komite —/).length).toBe(1)
  })

  it('memakai spreading dua kolom tanpa Fac-Out bila bukan bentuk lengkap', () => {
    render(
      <ClaimSheet
        kasus={KASUS}
        transfer={transfer({ baris: [line({ spreading: [{ jenis_treaty: 'OR', persen: '100' }] })] })}
      />,
    )

    // Hanya tabel spreading yang tergambar, dan bentuk pendeknya berkolom dua.
    expect(screen.getAllByRole('table')).toHaveLength(1)
    expect(screen.getAllByRole('columnheader').map((h) => h.textContent)).toEqual([
      'Spreading',
      'Share (%)',
    ])
    expect(screen.queryByText('List Reas Fac-Out')).not.toBeInTheDocument()
    expect(screen.queryByText(/Objek OBJ-1/)).not.toBeInTheDocument()
  })

  it('PolicyTab dan AttachmentTab jatuh ke tanda pisah dan kalimat kosong', () => {
    render(
      <>
        <PolicyTab kasus={KASUS_KOSONG} />
        <AttachmentTab />
      </>,
    )

    const policy = screen.getByRole('region', { name: 'Policy Detail' })
    expect(valueOf('GROUP PANEL', policy)).toBe('—')
    expect(valueOf('SHARE ASM', policy)).toBe('—')
    expect(valueOf('LEADER', policy)).toBe('—')
    expect(screen.getByText('Belum ada lampiran untuk klaim ini.')).toBeInTheDocument()
  })

  it('PolicyTab memakai data klaim dan polis bila ada', () => {
    render(
      <PolicyTab
        kasus={KASUS}
        transfer={transfer({
          klaim: { group_panel: '006', peran_koasuransi: 'LEADER', persen_asm_share: '25', cabang: 'Medan' },
          leader: { nama: 'PT L', persen: '25' },
          polis: { mulai: '2026-01-01T00:00:00Z' },
        })}
      />,
    )

    expect(valueOf('GROUP PANEL')).toBe('006')
    expect(valueOf('POSISI KOASURANSI')).toBe('LEADER')
    expect(valueOf('SHARE ASM')).toBe('25 %')
    expect(valueOf('BRANCH')).toBe('Medan')
    expect(valueOf('PERIOD OF INSURANCE')).toMatch(/S\/D —$/)
  })

  it('AttachmentTab menulis lampiran beserta pengunggahnya', () => {
    render(
      <AttachmentTab
        transfer={transfer({
          lampiran: [{ id: 'A1', nama: 'foto.jpg', catatan: 'kerusakan', kategori: 'Foto', diunggah_oleh: 'PETUGAS' }],
        })}
      />,
    )

    expect(screen.getByText('foto.jpg')).toBeInTheDocument()
    expect(screen.getByText('PETUGAS')).toBeInTheDocument()
  })
})

// ── CommitteeBottom ────────────────────────────────────────────────────────────────

describe('CommitteeBottom', () => {
  it('menebalkan baris total dan menulis keterangan estimasi alih-alih angkanya', () => {
    render(
      <CommitteeBottom
        transfer={transfer({
          baris: [
            line({
              rincian: {
                judul: 'Claim Adjustment',
                judul_nilai: 'ADJUSTMENT',
                tersedia: true,
                baris: [
                  { deskripsi: 'Gross Loss', mata_uang_estimasi: 'IDR', estimasi: '1000', persen: '100', mata_uang: 'IDR', nilai: '900' },
                  { deskripsi: 'Deductible', keterangan_estimasi: 'per kejadian' },
                  { deskripsi: 'Net', total: true, persen: '90', mata_uang: 'IDR', nilai: '810' },
                ],
              },
            }),
          ],
        })}
      />,
    )

    expect(screen.getByRole('columnheader', { name: 'ADJUSTMENT' })).toBeInTheDocument()
    expect(screen.getByText('per kejadian')).toBeInTheDocument()
    expect(screen.getByText('Net').tagName).toBe('STRONG')
    expect(screen.getByText('90').tagName).toBe('STRONG')
    expect(screen.getByText('Gross Loss').tagName).not.toBe('STRONG')
    // Satu baris saja: tidak ada label objek/coverage.
    expect(screen.queryByText(/^Objek /)).not.toBeInTheDocument()
  })

  it('menyebut tabel yang belum dibangun dan memberi label pada beberapa baris', () => {
    render(
      <CommitteeBottom
        transfer={transfer({
          baris: [
            line({ rincian: { judul: 'Salvage', judul_nilai: 'NILAI', tersedia: false, baris: [] } }),
            line({
              id_objek: '',
              id_coverage: '',
              rincian: { judul: 'Claim Accepted', judul_nilai: 'CLAIM ACCEPTED', tersedia: true, baris: [{ deskripsi: 'Total' }] },
            }),
            // Baris tanpa rincian tidak menggambar apa pun.
            line({ id_objek: 'X' }),
          ],
        })}
      />,
    )

    expect(
      screen.getByText('Salvage: tabel nilai untuk jenis pembayaran ini belum dibangun.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Objek — · Coverage —')).toBeInTheDocument()
  })

  it('menulis kategori dari catatan, lalu kategori, dan isi kedua daftar komite', () => {
    render(
      <CommitteeBottom
        transfer={transfer({
          lampiran: [
            { id: '1', nama: 'a.pdf', catatan: 'Surat jalan', kategori: 'Lain' },
            { id: '2', nama: 'b.pdf', kategori: 'Foto' },
            { id: '3' },
          ],
          riwayat_komite: [{ nomor_case: 'K-OLD', nama_komite: 'Komite Lama', status: 'Approved', jenjang: 1 }],
          daftar_komite: [{ nomor_case: 'K-3001', nama_komite: 'Komite Kini', status: 'Waiting', catatan: 'cek ulang' }],
        })}
      />,
    )

    expect(screen.getByText('Surat jalan')).toBeInTheDocument()
    expect(screen.getByText('Foto')).toBeInTheDocument()
    expect(screen.queryByText('Lain')).not.toBeInTheDocument()
    expect(screen.getByText('K-OLD')).toBeInTheDocument()
    expect(screen.getByText('Komite Kini')).toBeInTheDocument()
    expect(screen.getByText('cek ulang')).toBeInTheDocument()
  })

  it('menulis kalimat kosong saat tidak ada lampiran maupun komite', () => {
    render(<CommitteeBottom transfer={transfer()} />)

    expect(screen.getAllByText('Data Tidak Ada')).toHaveLength(2)
    expect(
      screen.getByText('Belum ada komite adjustment sebelumnya untuk klaim ini.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Case ini belum punya anggota komite.')).toBeInTheDocument()
  })
})

// ── DecisionPanel (tidak terpasang, tetapi jalurnya dipertahankan) ────────────────

describe('DecisionPanel', () => {
  function show(props: Partial<Parameters<typeof DecisionPanel>[0]> = {}) {
    const onSubmit = vi.fn()
    const onClose = vi.fn()
    const view = render(
      <DecisionPanel
        item={KASUS}
        working={false}
        error={null}
        onClose={onClose}
        onSubmit={onSubmit}
        {...props}
      />,
    )
    return { onSubmit, onClose, view }
  }

  it('memindahkan fokus ke judul dan menulis tanda pisah untuk data kosong', () => {
    show({ item: KASUS_KOSONG })

    expect(screen.getByRole('heading', { name: 'Keputusan komite — K-3001' })).toHaveFocus()
    expect(screen.getByText('Klaim — · tanpa nama tertanggung')).toBeInTheDocument()
    expect(screen.getAllByText('—')).toHaveLength(3)
  })

  it('menuntut pilihan sebelum menyimpan', async () => {
    const user = userEvent.setup()
    const { onSubmit } = show()

    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    expect(screen.getByText('Pilih salah satu keputusan lebih dulu.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('menyimpan persetujuan seketika, dengan catatan yang dirapikan', async () => {
    const user = userEvent.setup()
    const { onSubmit } = show()

    await user.click(screen.getByRole('radio', { name: /^Setuju/ }))
    expect(screen.getByLabelText('Catatan')).toBeInTheDocument()
    expect(screen.getByText('Boleh dikosongkan pada persetujuan.')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Catatan'), '  oke  ')
    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    expect(onSubmit).toHaveBeenCalledWith('setuju', 'oke')
  })

  it('mewajibkan catatan dan meminta konfirmasi pada penolakan', async () => {
    const user = userEvent.setup()
    const { onSubmit } = show()

    await user.click(screen.getByRole('radio', { name: /^Tolak/ }))
    expect(screen.getByLabelText('Catatan (wajib)')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))
    expect(
      screen.getByText('Catatan wajib diisi supaya alasannya dapat ditindaklanjuti.'),
    ).toBeInTheDocument()

    await user.type(screen.getByLabelText('Catatan (wajib)'), 'dokumen kurang')
    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    const alert = screen.getByRole('alert')
    expect(alert).toHaveTextContent('Kasus K-3001 akan ditolak')
    expect(onSubmit).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Ya, lanjutkan' }))
    expect(onSubmit).toHaveBeenCalledWith('tolak', 'dokumen kurang')
  })

  it('Batal membatalkan konfirmasi pengembalian tanpa menutup panel', async () => {
    const user = userEvent.setup()
    const { onSubmit, onClose } = show()

    await user.click(screen.getByRole('radio', { name: /^Kembalikan/ }))
    await user.type(screen.getByLabelText('Catatan (wajib)'), 'perbaiki')
    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))
    expect(screen.getByRole('alert')).toHaveTextContent('akan dikembalikan')

    await user.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
    expect(onSubmit).not.toHaveBeenCalled()

    // Tanpa konfirmasi berjalan, Batal menutup panel.
    await user.click(screen.getByRole('button', { name: 'Batal' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('mengganti pilihan membatalkan konfirmasi yang sedang menunggu', async () => {
    const user = userEvent.setup()
    show()

    await user.click(screen.getByRole('radio', { name: /^Tolak/ }))
    await user.type(screen.getByLabelText('Catatan (wajib)'), 'alasan')
    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))
    expect(screen.getByRole('button', { name: 'Ya, lanjutkan' })).toBeInTheDocument()

    await user.click(screen.getByRole('radio', { name: /^Setuju/ }))
    expect(screen.getByRole('button', { name: 'Simpan keputusan' })).toBeInTheDocument()
  })

  it('menolak catatan yang melampaui seribu karakter', async () => {
    const user = userEvent.setup()
    const { onSubmit } = show()

    await user.click(screen.getByRole('radio', { name: /^Setuju/ }))
    await user.click(screen.getByLabelText('Catatan'))
    await user.paste('x'.repeat(1001))
    await user.click(screen.getByRole('button', { name: 'Simpan keputusan' }))

    expect(screen.getByText('Catatan paling panjang 1000 karakter.')).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('mengosongkan isian saat berpindah kasus', async () => {
    const user = userEvent.setup()
    const { view, onSubmit, onClose } = show()

    await user.click(screen.getByRole('radio', { name: /^Tolak/ }))
    await user.type(screen.getByLabelText('Catatan (wajib)'), 'milik kasus pertama')

    view.rerender(
      <DecisionPanel
        item={{ ...KASUS, nomor_case: 'K-3002' }}
        working={false}
        error={null}
        onClose={onClose}
        onSubmit={onSubmit}
      />,
    )

    expect(screen.getByRole('heading', { name: 'Keputusan komite — K-3002' })).toBeInTheDocument()
    await waitFor(() => expect(screen.getByLabelText('Catatan')).toBeInTheDocument())
    expect((screen.getByLabelText('Catatan') as HTMLTextAreaElement).value).toBe('')
    expect(screen.getAllByRole('radio').map((r) => (r as HTMLInputElement).checked)).toEqual([false, false, false])
  })

  it('mematikan tombol selama menyimpan', () => {
    show({ working: true })
    expect(screen.getByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Batal' })).toBeDisabled()
  })

  it('menutup panel lewat Tutup', async () => {
    const user = userEvent.setup()
    const { onClose } = show()
    await user.click(screen.getByRole('button', { name: 'Tutup' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it.each([
    [new NetworkError(), 'Tidak dapat menghubungi server', /Keputusan belum tersimpan/],
    [
      new APIError('jejak_keputusan_belum_siap', 'Tabel keputusan belum dimigrasi.', 503),
      'Keputusan belum dapat dicatat',
      'Tabel keputusan belum dimigrasi.',
    ],
    [
      new APIError('komite_sudah_selesai', 'Sudah diputuskan di tab lain.', 409),
      'Keputusan sudah tercatat',
      'Sudah diputuskan di tab lain.',
    ],
    [new APIError('galat_internal', 'Server rusak.', 500), 'Keputusan gagal disimpan', 'Server rusak.'],
    ['bukan galat', 'Keputusan gagal disimpan', 'Terjadi kesalahan pada sistem.'],
  ])('menjelaskan galat simpan %#', (error, judul, isi) => {
    show({ error })
    expect(screen.getByText(judul)).toBeInTheDocument()
    expect(screen.getByText(isi)).toBeInTheDocument()
  })
})

// ── useDecide ─────────────────────────────────────────────────────────────────────

describe('useDecide', () => {
  beforeEach(() => {
    useSession.setState({
      token: 'token-uji',
      user: null,
      validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    useSession.getState().clear()
  })

  it('mengirim keputusan tanpa jenjang lalu memuat ulang seluruh daftar inbox', async () => {
    const calls: { url: string; init: RequestInit | undefined }[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      return Promise.resolve(
        new Response(JSON.stringify({ kasus: KASUS, sekarang: '2026-09-20T00:00:00Z' }), {
          status: 200,
        }),
      )
    })
    const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    )

    const { result } = renderHook(() => useDecide(), { wrapper })
    act(() => {
      result.current.mutate({ caseID: 'K 1/2', decision: 'tolak', note: 'alasan' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(calls[0]?.url).toBe('/api/komite/inbox/K%201%2F2/keputusan')
    expect(calls[0]?.init?.method).toBe('POST')
    expect(JSON.parse(String(calls[0]?.init?.body))).toEqual({
      keputusan: 'tolak',
      catatan: 'alasan',
    })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['inbox-komite', 'token-uji'] })
  })
})

// ── Halaman Inbox, rincian, dan form KMTN ─────────────────────────────────────────

type Answer = { status?: number; body?: unknown; fail?: boolean; hold?: boolean }

let pageCalls: { url: string; init: RequestInit | undefined }[] = []

function stubPages(answer: (url: string, init?: RequestInit) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    pageCalls.push({ url, init })
    const reply = answer(url, init)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    if (reply.hold) return new Promise(() => {})
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function listBody(rows: KomiteCase[], extra: Record<string, unknown> = {}) {
  return {
    kasus: rows,
    total: rows.length,
    lewati: 0,
    batas: 25,
    ringkasan: { outstanding: rows.length, diterima: 0, ditolak: 0 },
    kotak: 'outstanding',
    operator: 'ELLENSUPRIYATI',
    sekarang: '2026-09-20T02:00:00Z',
    jejak_keputusan_tersedia: true,
    penyaring_pemilik_aktif: true,
    ...extra,
  }
}

function renderAt(path: string) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/komite/inbox" element={<InboxKomitePage />} />
          <Route path="/komite/inbox/:nomor" element={<KomiteCasePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function startPageSession(login = 'ELLENSUPRIYATI') {
  useSession.setState({
    token: 'token-uji',
    user: { identitas: '1', nama: 'Uji', jenis: 'KARYAWAN', login, email: '', perusahaan: 'ASM' },
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
}

describe('InboxKomitePage', () => {
  beforeEach(() => {
    pageCalls = []
    startPageSession()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    useSession.getState().clear()
  })

  it('menggambar sel kosong, lencana umur, dan keadaan berjenjang', async () => {
    stubPages(() => ({
      body: listBody([
        { ...KASUS_KOSONG, nomor_case: 'K-1', aging_komite: 1, tanggal_komite: 'bukan-tanggal' },
        {
          ...KASUS,
          nomor_case: 'K-2',
          sumber_bisnis: '',
          aging_komite: 4,
          keputusan_pega: 'disetujui',
          penjenjangan: { ...KASUS.penjenjangan, kesimpulan: 'disetujui', jenjang_kini: 0, jumlah_jenjang: 3 },
        },
        {
          ...KASUS,
          nomor_case: 'K-3',
          aging_komite: 8,
          penjenjangan: {
            ...KASUS.penjenjangan,
            kesimpulan: 'lain' as KomiteCase['penjenjangan']['kesimpulan'],
            jumlah_jenjang_belum_diketahui: true,
            jenjang_kini: 2,
          },
        },
      ]),
    }))
    const user = userEvent.setup()
    renderAt('/komite/inbox')

    expect(await screen.findByText('1 hari')).toBeInTheDocument()
    expect(screen.getByText('1 hari').className).toContain('bg-slate-100')
    expect(screen.getByText('4 hari').className).toContain('bg-amber-50')
    expect(screen.getByText('8 hari').className).toContain('bg-red-50')
    expect(screen.getByText('Jenjang 3 dari 3')).toBeInTheDocument()
    expect(screen.getByText('Jenjang ke-2')).toBeInTheDocument()
    expect(screen.getByText('Pega: disetujui')).toBeInTheDocument()
    expect(screen.getByText('lain').className).toContain('bg-slate-100')
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(5)

    const before = pageCalls.length
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    await waitFor(() => expect(pageCalls.length).toBeGreaterThan(before))
  })

  it('menjelaskan kosong per kotak dan galat pemuatan', async () => {
    let mode: 'empty' | 'fail' | 'api' | 'odd' = 'empty'
    stubPages((url) => {
      if (mode === 'fail') return { fail: true }
      if (mode === 'api') return { status: 500, body: { kode: 'galat_internal', pesan: 'Tabel komite rusak.' } }
      if (mode === 'odd') return { status: 500, body: {} }
      return { body: listBody([], { kotak: new URL(url, 'https://x').searchParams.get('kotak') }) }
    })
    const user = userEvent.setup()
    renderAt('/komite/inbox')

    expect(await screen.findByText('Tidak ada kasus yang menunggu keputusan Anda.')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: /Diterima/ }))
    expect(await screen.findByText('Belum ada kasus yang Anda setujui.')).toBeInTheDocument()
    await user.click(screen.getByRole('tab', { name: /Ditolak/ }))
    expect(await screen.findByText('Belum ada kasus yang Anda tolak atau kembalikan.')).toBeInTheDocument()

    mode = 'fail'
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    expect(await screen.findByText('Tidak dapat menghubungi server')).toBeInTheDocument()

    mode = 'api'
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    expect(await screen.findByText('Tabel komite rusak.')).toBeInTheDocument()

    mode = 'odd'
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    expect(await screen.findByText('Terjadi kesalahan pada sistem.')).toBeInTheDocument()
  })

  it('berpindah halaman dan membersihkan rentang tanggal', async () => {
    stubPages((url) => {
      const lewati = Number(new URL(url, 'https://x').searchParams.get('lewati') ?? '0')
      return { body: { ...listBody([KASUS]), total: 30, lewati } }
    })
    const user = userEvent.setup()
    renderAt('/komite/inbox')

    expect(await screen.findByText('Menampilkan 1–1 dari 30 kasus.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(await screen.findByText('Menampilkan 26–26 dari 30 kasus.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    expect(await screen.findByText('Menampilkan 1–1 dari 30 kasus.')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Tgl input dari'), '2026-09-01')
    await user.click(screen.getByRole('button', { name: 'Bersihkan' }))
    expect(screen.getByLabelText('Tgl input dari')).toHaveValue('')
    expect(screen.queryByRole('button', { name: 'Bersihkan' })).not.toBeInTheDocument()
    expect(pageCalls.some((c) => c.url.includes('dari=2026-09-01'))).toBe(true)
  })
})

describe('KomiteCasePage', () => {
  beforeEach(() => {
    startPageSession()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    useSession.getState().clear()
  })

  it.each([
    ['disetujui', 'Diterima'],
    ['ditolak', 'Ditolak'],
    ['dikembalikan', 'Dikembalikan'],
    ['lain', 'Menunggu'],
  ])('menulis keputusan Pega %s sebagai %s', async (outcome, label) => {
    stubPages(() => ({
      body: { kasus: { ...KASUS, keputusan_pega: outcome }, sekarang: '2026-09-20T02:00:00Z' },
    }))
    renderAt('/komite/inbox/K-3001')

    expect(await screen.findByText(label)).toBeInTheDocument()
  })

  it('menjelaskan galat jaringan dan galat lain, lalu memuat ulang', async () => {
    let mode: 'fail' | 'api' | 'odd' = 'fail'
    stubPages(() =>
      mode === 'fail'
        ? { fail: true }
        : mode === 'api'
          ? { status: 500, body: { kode: 'galat_internal', pesan: 'Rincian rusak.' } }
          : { status: 500, body: {} },
    )
    const user = userEvent.setup()
    renderAt('/komite/inbox/K-3001')

    expect(await screen.findByText('Tidak dapat menghubungi server')).toBeInTheDocument()
    mode = 'api'
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    expect(await screen.findByText('Rincian rusak.')).toBeInTheDocument()
    mode = 'odd'
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    expect(await screen.findByText('Terjadi kesalahan pada sistem.')).toBeInTheDocument()
  })

  it('menggambar kartu adjustment, catatan komite Pega, dan analisis coverage', async () => {
    stubPages(() => ({
      body: {
        kasus: KASUS,
        transfer: transfer({
          komite: {
            tipe_komite: '',
            nama_komite: '',
            jenjang: 0,
            nilai_klaim: '0',
            persen_asm_share: '',
            catatan: 'Catatan Pega.',
          } as KomiteTransferDetail['komite'],
          baris: [
            line({
              nomor_klaim: 'PNCN.26.0300',
              id_objek: '',
              id_coverage: '',
              ex_gratia: true,
              nomor_akseptasi: 'AKS-1',
              tanggal_akseptasi: '2026-09-01T00:00:00Z',
              sebab_kerugian: 'Kebakaran gudang',
              catatan: 'Catatan baris.',
              persen_asm_share: '40',
            }),
          ],
          coverage: [
            {
              id_objek: '',
              id_coverage: '',
              nilai_tsi: '1000',
              analisis_terisi: true,
              keadaan_kerugian: 'Terbakar habis',
              catatan: '  ',
            },
          ],
          klaim: {
            status_klaim: '',
            persen_asm_share: '',
            koasuransi: '',
            kronologi: 'Api menjalar.',
            rekomendasi: 'Bayar.',
          },
        }),
        sekarang: '2026-09-20T02:00:00Z',
      },
    }))
    renderAt('/komite/inbox/K-3001')

    expect(await screen.findByText('Ex-Gratia')).toBeInTheDocument()
    expect(screen.getByText('Catatan Pega.')).toBeInTheDocument()
    expect(screen.getByText(/Akseptasi AKS-1 ·/)).toBeInTheDocument()
    expect(screen.getByText('Kebakaran gudang')).toBeInTheDocument()
    expect(screen.getByText('Catatan baris.')).toBeInTheDocument()
    expect(screen.getByText('Share ASM (40%)')).toBeInTheDocument()
    expect(screen.getByText('Terbakar habis')).toBeInTheDocument()
    expect(screen.getByText('Api menjalar.')).toBeInTheDocument()
    expect(screen.getByText('Bayar.')).toBeInTheDocument()
    expect(screen.getAllByText('Objek — · Coverage —').length).toBeGreaterThan(0)
  })
})

describe('KmtnDecisionForm', () => {
  function show(caseID = 'KMTN.26.9') {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    })
    return render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <KmtnDecisionForm caseID={caseID} />
        </MemoryRouter>
      </QueryClientProvider>,
    )
  }

  beforeEach(() => {
    // Login berhuruf kecil membuktikan pencocokan anggota yang ditunggu tidak peka huruf.
    startPageSession('ellensupriyati')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    useSession.getState().clear()
  })

  it('menyebut status sedang dimuat, gagal dimuat, dan sudah ditutup', async () => {
    let mode: 'hold' | 'fail' | 'approved' | 'rejected' = 'hold'
    stubPages(() =>
      mode === 'hold'
        ? { hold: true }
        : mode === 'fail'
          ? { status: 500, body: {} }
          : {
              body: {
                id: 'KMTN.26.9',
                nomor_klaim: 'X',
                status: mode === 'approved' ? 'disetujui' : 'ditolak',
              },
            },
    )
    const view = show()
    expect(screen.getByText('Loading committee status…')).toBeInTheDocument()
    view.unmount()

    mode = 'fail'
    const failed = show()
    expect(await screen.findByText(/could not be loaded/)).toBeInTheDocument()
    failed.unmount()

    mode = 'approved'
    const approved = show()
    expect(await screen.findByText('This committee is closed — approved.')).toBeInTheDocument()
    approved.unmount()

    mode = 'rejected'
    show()
    expect(await screen.findByText('This committee is closed — rejected.')).toBeInTheDocument()
  })

  it('menyebut anggota berikutnya tanpa nama bila server tidak menyebutnya', async () => {
    stubPages(() => ({ body: { id: 'KMTN.26.9', nomor_klaim: 'X', status: 'berjalan' } }))
    show()
    expect(
      await screen.findByText('Waiting for the decision of the next committee member.'),
    ).toBeInTheDocument()
  })

  it('mengabarkan isian yang kurang satu per satu dan galat simpan', async () => {
    let fail: Answer = { status: 409, body: { kode: 'bukan_giliran', pesan: 'Bukan giliran Anda.' } }
    stubPages((_, init) =>
      init?.method === 'POST'
        ? fail
        : { body: { id: 'KMTN.26.9', nomor_klaim: 'X', status: 'berjalan', menunggu: 'ELLENSUPRIYATI' } },
    )
    const user = userEvent.setup()
    show()

    await user.click(await screen.findByLabelText('Rejected'))
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    expect(screen.getByText('Catatan Komite is required.')).toBeInTheDocument()

    await user.type(screen.getByLabelText(/Catatan Komite/), 'tolak')
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    expect(await screen.findByText('Bukan giliran Anda.')).toBeInTheDocument()

    // Galat jaringan bukan APIError, sehingga pesannya pesan umum.
    fail = { fail: true }
    await user.click(screen.getByRole('button', { name: 'Submit' }))
    expect(await screen.findByText('The decision could not be saved. Try again.')).toBeInTheDocument()
  })

  it('tidak menggambar apa pun pada case yang bukan KMTN', () => {
    pageCalls = []
    stubPages(() => ({ body: {} }))
    const { container } = show('K-3001')
    expect(container).toBeEmptyDOMElement()
    expect(pageCalls).toHaveLength(0)
  })
})
