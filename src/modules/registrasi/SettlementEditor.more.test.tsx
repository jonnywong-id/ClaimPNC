import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SettlementDetail, SettlementEditor, finishMoney, groupMoney, inpatientDays } from './SettlementEditor'
import type { Settlement } from './types'

/**
 * Uji isian Adjustment baru dan tampilan bacanya, dirender langsung. Hitungan nilai
 * dikerjakan server; layar hanya mengirim isian dan menampilkan jawabannya. Data KARANGAN.
 */

const BASE = '/api/registrasi/klaim/klaim-1/adjustment'

type Answer = Response | Promise<Response> | 'putus' | undefined

let calls: { url: string; body: Record<string, unknown> }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function line(extra: Partial<Settlement> = {}): Settlement {
  return {
    tipe_pembayaran: '1',
    nama_tipe_pembayaran: 'Final',
    mata_uang: 'IDR',
    kurs_e4: 152_000_000,
    nilai_propose_sen: 100_000,
    nilai_pengajuan_sen: 90_000,
    loc: 0,
    nilai_salvage_sen: 0,
    nilai_salvage_b_sen: 0,
    nilai_interim_sen: 0,
    nilai_estimasi_sen: 0,
    tipe_resiko: '9',
    persen_resiko: 0,
    nilai_resiko_sen: 1_000,
    nilai_gross_sen: 80_000,
    share_asm: 500_000,
    nilai_asm_sen: 40_000,
    nilai_akseptasi_sen: 0,
    kronologi: '',
    catatan: '',
    status_akseptasi: '',
    nomor_akseptasi: '',
    ...extra,
  }
}

function installFetch(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : {} })
    const custom = answer(url)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom) return Promise.resolve(custom)
    return Promise.resolve(json(200, { adjustment: line(), spreading: [] }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

function editor(
  props: {
    travel?: boolean
    nonMBU?: boolean
    exGratia?: boolean
    pa?: boolean
    analyst?: boolean
    analystTransfer?: boolean
    existing?: { index: number; line: Settlement }
    onCreated?: (index: number) => void
    onClose?: () => void
  } = {},
) {
  return (
    <SettlementEditor
      claimID="klaim-1"
      taskID="tugas-1"
      object={1}
      coverage={1}
      policyCurrency="IDR"
      currencies={[
        { id: 'IDR', nama: 'Rupiah' },
        { id: 'USD', nama: 'Dolar' },
      ]}
      travel={props.travel ?? false}
      nonMBU={props.nonMBU ?? false}
      exGratia={props.exGratia ?? false}
      pa={props.pa ?? false}
      analyst={props.analyst ?? false}
      analystTransfer={props.analystTransfer ?? false}
      {...(props.existing ? { existing: props.existing } : {})}
      {...(props.onCreated ? { onCreated: props.onCreated } : {})}
      onClose={props.onClose ?? (() => {})}
    />
  )
}

const previews = () => calls.filter((c) => c.url === `${BASE}/hitung`)

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('SettlementEditor', () => {
  // InputAdjustment_sect: LOC dan Salvage A/B di kontainer !IsPATRAVEL; Lama hari rawat inap dan Status Aksep
  // Analysator IsPA; tabel treaty hanya di kontainer .ExGratia = 1.
  // Work Owner 2026-10-09: isian uang memasang pemisah ribuan saat diketik, dan dua desimal saat ditinggalkan.
  it('memformat isian uang dengan pemisah ribuan dan dua desimal', () => {
    expect(groupMoney('3000000')).toBe('3.000.000')
    expect(groupMoney('3.000.000,5')).toBe('3.000.000,5')
    expect(groupMoney('3000000,456')).toBe('3.000.000,45')
    expect(groupMoney('0012')).toBe('12')
    expect(groupMoney(',5')).toBe('0,5')
    expect(groupMoney('abc')).toBe('')
    expect(finishMoney('3000000')).toBe('3.000.000,00')
    expect(finishMoney('3.000.000,5')).toBe('3.000.000,50')
    expect(finishMoney('')).toBe('')
  })

  it('isian Nilai Pengajuan menampilkan pemisah saat diketik', async () => {
    installFetch(() => undefined)
    wrap(editor({ pa: true }))
    const user = userEvent.setup()
    const field = screen.getByLabelText('Nilai Pengajuan')
    await user.type(field, '3000000')
    expect(field).toHaveValue('3.000.000')
    await user.tab()
    expect(field).toHaveValue('3.000.000,00')

    // Nilai Pengajuan Tertanggung PA terkirim untuk PROPOSE_VALUE_TERTANGGUNG.
    const insured = screen.getByLabelText('Nilai Pengajuan Tertanggung')
    await user.type(insured, '2500000')
    await user.tab()
    expect(insured).toHaveValue('2.500.000,00')
    await waitFor(() =>
      expect(previews().at(-1)?.body).toMatchObject({
        nilai_pengajuan_sen: 300_000_000,
        nilai_pengajuan_tertanggung_sen: 250_000_000,
      }),
    )
  })

  // ValidationAdjustment langkah 95: selisih hari Tanggal Masuk sampai Tanggal Keluar Rawat Inap.
  it('menghitung Lama Hari Rawat Inap dari tanggal masuk dan keluar', () => {
    expect(inpatientDays('2026-08-10', '2026-08-12')).toBe(2)
    expect(inpatientDays('2026-08-10', '2026-08-10')).toBe(0)
    expect(inpatientDays('2026-02-27', '2026-03-02')).toBe(3)
    expect(inpatientDays('2026-08-10', '')).toBeNull()
    expect(inpatientDays('2026-08-10', '2026-08-09')).toBeNull()
  })

  it('lini PA: tanpa LOC dan Salvage, dengan Lama Hari Rawat Inap; tabel treaty hanya Ex Gratia', async () => {
    installFetch((url) =>
      url === `${BASE}/hitung`
        ? json(200, {
            adjustment: line({ tipe_pembayaran: '2' }),
            spreading: [{ jenis_treaty: '10007', nama: 'OR', share: 1_000_000, dihapus: false, objek_fac_offer: '' }],
          })
        : undefined,
    )
    wrap(editor({ pa: true }))

    // Baris PA baru bertipe Interim, baca saja bagi selain Analyst.
    expect(screen.queryByRole('combobox', { name: 'Tipe Pembayaran' })).not.toBeInTheDocument()
    expect(screen.getByText('Interim')).toBeInTheDocument()
    // Kontainer IsPA: Nilai Pengajuan Tertanggung dan Nilai Pengajuan; Total Klaim (.ProposeAdjustmentValue)
    // hanya pada baris isAnalistorTransfer.
    expect(screen.getByLabelText('Nilai Pengajuan Tertanggung')).toBeInTheDocument()
    expect(screen.getByLabelText('Nilai Pengajuan')).toBeEnabled()
    expect(screen.queryByLabelText('Total Klaim')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Lack Of Document (%)')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Nilai Salvage A')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Nilai Salvage B')).not.toBeInTheDocument()
    expect(screen.getByText('Lama Hari Rawat Inap')).toBeInTheDocument()
    expect(screen.getByText('Status Aksep Analysator')).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'Spreading jaminan' })).not.toBeInTheDocument()
  })

  // isAnalistorTransfer (jaminan PHK) memunculkan Nilai Propose Adjustment; IsAnalisator membuka Tipe Pembayaran
  // dan menonaktifkan Nilai Pengajuan.
  it('lini PA: Total Klaim untuk baris isAnalistorTransfer, Tipe Pembayaran terbuka bagi Analyst', () => {
    installFetch(() => undefined)
    const { unmount } = wrap(editor({ pa: true, analystTransfer: true }))
    expect(screen.getByLabelText('Total Klaim')).toBeInTheDocument()
    unmount()

    wrap(editor({ pa: true, analyst: true }))
    expect(screen.getByRole('combobox', { name: 'Tipe Pembayaran' })).toHaveValue('2')
    expect(screen.getByLabelText('Nilai Pengajuan')).toBeDisabled()
  })

  it('menghitung Adjuster Fee dari Professional Fee, Survey Expenses, dan VAT', async () => {
    installFetch((url) =>
      url === `${BASE}/hitung`
        ? json(200, {
            adjustment: line({ tipe_pembayaran: '4', nilai_asm_sen: 55_500 }),
            spreading: [{ jenis_treaty: '10007', nama: '', share: 1_000_000, dihapus: false, objek_fac_offer: '' }],
          })
        : undefined,
    )
    wrap(editor({ nonMBU: true, exGratia: true }))

    await userEvent.selectOptions(screen.getByLabelText('Mata Uang'), 'USD')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pembayaran'), '4')
    await userEvent.type(screen.getByLabelText('Professional Fee'), '500')
    await userEvent.type(screen.getByLabelText('Survey Expenses'), '50')
    await userEvent.selectOptions(screen.getByLabelText('Tipe VAT'), '2')
    await userEvent.type(screen.getByLabelText('VAT (%)'), '11')

    expect(await screen.findByText('555,00')).toBeInTheDocument()
    expect(screen.getByText('Tanggal Transfer LOD')).toBeInTheDocument()
    expect(screen.getByText('Status Lunas')).toBeInTheDocument()
    // Spreading tanpa nama ditampilkan dengan kode treatynya.
    const spreading = screen.getByRole('table', { name: 'Spreading jaminan' })
    expect(within(spreading).getByText('10007')).toBeInTheDocument()

    await waitFor(() =>
      expect(previews().at(-1)?.body).toMatchObject({
        tipe_pembayaran: '4',
        mata_uang: 'USD',
        professional_fee_sen: 50_000,
        survey_expenses_sen: 5_000,
        vat: 110_000,
        tipe_vat: '2',
        tipe_resiko: '',
        nilai_propose_sen: 0,
      }),
    )
  })

  it('mengirim risiko persen dan nilai manual sesuai tipe resiko', async () => {
    installFetch(() => undefined)
    wrap(editor())

    await userEvent.selectOptions(screen.getByLabelText('Tipe Pembayaran'), '2')
    await userEvent.type(screen.getByLabelText('Total Klaim'), '1.000')
    await userEvent.type(screen.getByLabelText('Nilai Pengajuan Tertanggung'), '900')
    await userEvent.type(screen.getByLabelText('Lack Of Document (%)'), '5')
    await userEvent.type(screen.getByLabelText('Nilai Salvage A'), '10')
    await userEvent.type(screen.getByLabelText('Nilai Salvage B'), '20')

    await userEvent.selectOptions(screen.getByLabelText('Tipe Resiko Sendiri'), '2')
    await userEvent.type(screen.getByLabelText('Persen Resiko Sendiri (%)'), '2,5')
    await waitFor(() =>
      expect(previews().at(-1)?.body).toMatchObject({
        tipe_pembayaran: '2',
        nilai_propose_sen: 100_000,
        nilai_pengajuan_sen: 90_000,
        loc: 50_000,
        nilai_salvage_sen: 1_000,
        nilai_salvage_b_sen: 2_000,
        tipe_resiko: '2',
        persen_resiko: 25_000,
        nilai_resiko_sen: 0,
      }),
    )

    await userEvent.selectOptions(screen.getByLabelText('Tipe Resiko Sendiri'), '3')
    expect(screen.queryByLabelText('Persen Resiko Sendiri (%)')).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Nilai Resiko Sendiri'), '7')
    await waitFor(() =>
      expect(previews().at(-1)?.body).toMatchObject({ tipe_resiko: '3', persen_resiko: 0, nilai_resiko_sen: 700 }),
    )
  })

  it('pada Travel mengunci mata uang dan tidak mengirim LOC maupun salvage', async () => {
    installFetch(() => undefined)
    wrap(editor({ travel: true }))

    expect(screen.getByLabelText('Mata Uang')).toBeDisabled()
    expect(screen.queryByText('Status Persetujuan LOD')).not.toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pembayaran'), '5')
    expect(screen.queryByLabelText('Lack Of Document (%)')).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Total Klaim'), '100')
    await waitFor(() =>
      expect(previews().at(-1)?.body).toMatchObject({ tipe_pembayaran: '5', loc: 0, nilai_salvage_sen: 0 }),
    )
  })

  it('menolak angka yang tidak valid tanpa menghitung', async () => {
    installFetch(() => undefined)
    wrap(editor())

    await userEvent.selectOptions(screen.getByLabelText('Tipe Pembayaran'), '1')
    await userEvent.selectOptions(screen.getByLabelText('Tipe Resiko Sendiri'), '1')
    await waitFor(() => expect(previews().length).toBeGreaterThan(0))
    const before = previews().length
    const saves = calls.filter((c) => c.url === BASE).length
    await userEvent.type(screen.getByLabelText('Persen Resiko Sendiri (%)'), 'abc')
    await userEvent.tab()

    expect(await screen.findByText('Angka tidak valid.')).toBeInTheDocument()
    // Isian rusak tidak disimpan.
    expect(calls.filter((c) => c.url === BASE).length).toBe(saves)
    // Hitungan terakhir yang terkirim tetap hitungan sebelum isian rusak.
    expect(previews().length).toBe(before)
  })

  it.each([
    {
      name: 'pelanggaran validasi saat menyimpan',
      path: BASE,
      answer: (): Answer =>
        json(422, {
          kode: 'validasi_gagal',
          pesan: 'x',
          detail: [{ kode: 'a', field: '', pesan: 'Nilai melebihi sisa TSI.' }],
        }),
      title: 'Adjustment belum tersimpan',
      text: 'Nilai melebihi sisa TSI.',
    },
    {
      name: 'galat hitung',
      path: `${BASE}/hitung`,
      answer: (): Answer => json(500, { kode: 'x', pesan: 'Kurs tidak ditemukan.' }),
      title: 'Nilai belum dapat dihitung',
      text: 'Kurs tidak ditemukan.',
    },
    {
      name: 'jaringan putus saat menyimpan',
      path: BASE,
      answer: (): Answer => 'putus',
      title: 'Adjustment belum tersimpan',
      text: 'Tidak dapat menghubungi server Claim PNC.',
    },
  ])('menampilkan galat: $name', async ({ path, answer, title, text }) => {
    installFetch((url) => (url === path ? answer() : undefined))
    wrap(editor())

    // Pilihan berubah: dihitung dan langsung disimpan (SetNilaiResikoSendiri).
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pembayaran'), '1')

    expect(await screen.findByText(title)).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  // Pega tidak punya Simpan/Batal: perubahan isian langsung disimpan; Hapus menutup baris yang belum tersimpan.
  it('menyimpan baris baru begitu isian berubah, tanpa Simpan dan Batal', async () => {
    installFetch((url) => (url === BASE ? json(200, { klaim: { objek: [{ coverage: [{ adjustment: [line()] }] }] } }) : undefined))
    const onCreated = vi.fn()
    const onClose = vi.fn()
    wrap(editor({ onCreated, onClose }))

    expect(screen.queryByRole('button', { name: 'Simpan' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Batal' })).not.toBeInTheDocument()
    await userEvent.selectOptions(screen.getByLabelText('Tipe Pembayaran'), '1')
    await waitFor(() => expect(onCreated).toHaveBeenCalledWith(0))
    expect(calls.find((c) => c.url === BASE)?.body).toMatchObject({ tugas_id: 'tugas-1', objek: 1, jaminan: 1, tipe_pembayaran: '1' })

    await userEvent.click(screen.getByRole('button', { name: 'Hapus' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('baris tersimpan: perubahan disimpan ulang lewat rute ubah; Hapus menunggu kolom DIHAPUS_PADA', async () => {
    installFetch(() => undefined)
    wrap(editor({ existing: { index: 0, line: line({ tipe_resiko: '3' }) } }))

    expect(screen.getByLabelText('Total Klaim')).toHaveValue('1000,00')
    expect(screen.getByRole('button', { name: 'Hapus' })).toBeDisabled()
    await userEvent.clear(screen.getByLabelText('Total Klaim'))
    await userEvent.type(screen.getByLabelText('Total Klaim'), '2000')
    await userEvent.tab()
    await waitFor(() => expect(calls.find((c) => c.url === `${BASE}/ubah`)?.body).toMatchObject({ adjustment: 1, nilai_propose_sen: 200_000 }))
  })
})

describe('SettlementDetail', () => {
  it('menampilkan Adjuster Fee dan tanda kosong pada lini Travel', () => {
    render(
      <SettlementDetail
        line={line({ tipe_pembayaran: '4', nama_tipe_pembayaran: 'Adjuster Fee' })}
        currencyName="Rupiah"
        estimation={0}
        spreading={[]}
        travel
        nonMBU={false}
      />,
    )

    const group = screen.getByRole('group', { name: 'Detail adjustment Adjuster Fee' })
    expect(within(group).getByText('Nilai Adjuster Fee')).toBeInTheDocument()
    expect(within(group).queryByText('Total Klaim')).not.toBeInTheDocument()
    expect(within(group).queryByText('Status Persetujuan LOD')).not.toBeInTheDocument()
    expect(within(group).queryByRole('table')).not.toBeInTheDocument()
  })

  it('menampilkan nilai opsional, persen, dan nilai estimasi jaminan pada Non-MBU', () => {
    render(
      <SettlementDetail
        line={line({ tipe_resiko: '1', persen_resiko: 25_000, loc: 50_000, nilai_salvage_sen: 3_000, nilai_interim_sen: 4_000 })}
        currencyName="Rupiah"
        estimation={123_400}
        spreading={[{ jenis_treaty: '10008', nama: 'Treaty', share: 400_000, dihapus: false, objek_fac_offer: '' }]}
        travel={false}
        nonMBU
        exGratia
      />,
    )

    const group = screen.getByRole('group', { name: 'Detail adjustment Final' })
    expect(within(group).getByText('% dari Nilai Klaim')).toBeInTheDocument()
    expect(within(group).getByText('2,5000')).toBeInTheDocument()
    expect(within(group).getByText('5,0000')).toBeInTheDocument()
    expect(within(group).getByText('30,00')).toBeInTheDocument()
    expect(within(group).getByText('40,00')).toBeInTheDocument()
    expect(within(group).getByText('1.234,00')).toBeInTheDocument()
    expect(within(group).getByText('Tanggal Bayar Kasir')).toBeInTheDocument()
    expect(within(group).getByRole('table', { name: 'Spreading jaminan' })).toHaveTextContent('Treaty')
  })
})
