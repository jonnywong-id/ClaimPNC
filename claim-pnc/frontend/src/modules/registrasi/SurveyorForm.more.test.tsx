import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SurveyorForm } from './SurveyorForm'
import type { Claim, Settlement, Task } from './types'

/**
 * Uji layar InputSurveyor dirender langsung: tombol per lini bisnis, tab Survey, tampilan
 * Input Register, dan grid Adjustment. Seluruh data KARANGAN (`D-69`).
 */

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

let estimateAnswer: () => Response = () => json(500, { kode: 'x', pesan: 'x' })

function installFetch() {
  vi.stubGlobal('fetch', (url: string) => {
    if (url.startsWith('/api/registrasi/estimasi')) return Promise.resolve(estimateAnswer())
    if (url === '/api/registrasi/mata-uang') {
      return Promise.resolve(json(200, { pilihan: [{ id: 'IDR', nama: 'Rupiah' }] }))
    }
    if (url.endsWith('/survey')) return Promise.resolve(json(200, { survey: [] }))
    if (url.endsWith('/dokumen')) return Promise.resolve(json(200, { kategori: [], berkas: [] }))
    if (url.endsWith('/progres')) return Promise.resolve(json(200, { progres: [], komunikasi: [] }))
    if (url.includes('/pilihan-item')) return Promise.resolve(json(200, { pilihan: [] }))
    return Promise.resolve(json(404, { kode: 'x', pesan: 'x' }))
  })
}

function settlement(extra: Partial<Settlement>): Settlement {
  return {
    tipe_pembayaran: '1',
    nama_tipe_pembayaran: 'Final',
    mata_uang: 'IDR',
    kurs_e4: 10000,
    nilai_propose_sen: 0,
    nilai_pengajuan_sen: 0,
    loc: 0,
    nilai_salvage_sen: 0,
    nilai_salvage_b_sen: 0,
    nilai_interim_sen: 0,
    nilai_estimasi_sen: 0,
    tipe_resiko: '',
    persen_resiko: 0,
    nilai_resiko_sen: 0,
    nilai_gross_sen: 0,
    share_asm: 0,
    nilai_asm_sen: 0,
    nilai_akseptasi_sen: 0,
    kronologi: '',
    catatan: '',
    status_akseptasi: '',
    nomor_akseptasi: '',
    ...extra,
  }
}

function claim(extra: Partial<Claim> = {}, polis: Partial<Claim['polis']> = {}): Claim {
  return {
    id: 'klaim-1',
    nomor: 'PNCN.26.0001',
    portal: 'ASM',
    polis: {
      nomor: 'POL-1',
      lini: '006',
      nama_lini: 'Fire',
      jenis_bisnis: 'Lain',
      mulai_pertanggungan: '2026-01-01',
      akhir_pertanggungan: '2026-12-31',
      mata_uang: 'IDR',
      nama_tertanggung: '',
      deklarasi: false,
      penjamin_kredit: false,
      ...polis,
    },
    tanggal_kejadian: '2026-06-05',
    tanggal_lapor: '2026-06-06',
    tanggal_terima_dokumen: '2026-06-07',
    lokasi: '',
    kronologi: '',
    pelapor: { nama: '', telepon: '', email: '', alamat: '', hubungan: 1, hubungan_lainnya: '' },
    wilayah: {} as Claim['wilayah'],
    prinsip_mengenal_nasabah: '1',
    komentar_suspicious: '',
    nilai_estimasi_sen: 0,
    mata_uang: 'IDR',
    nomor_slik: '',
    ex_gratia: true,
    user_teknis: '',
    rcv_id: '',
    objek: [
      {
        id: 'OBJ-1',
        nama: '',
        lokasi: '',
        coverage: [
          {
            id: 'CVG-1',
            nama: '',
            penyebab_kerugian: '',
            tsi_sen: 100_000,
            spreading: [
              { jenis_treaty: '10007', nama: '', share: 600_000, dihapus: false, objek_fac_offer: '' },
              { jenis_treaty: '10008', nama: 'Dihapus', share: 400_000, dihapus: true, objek_fac_offer: '' },
            ],
            item: [
              {
                estimasi: [
                  { tipe: '1', mata_uang: 'IDR', tanggal: '', nilai_sen: 5_000 },
                  { tipe: '2', mata_uang: 'IDR', tanggal: '', nilai_sen: 7_000 },
                ],
              } as never,
            ],
            adjustment: [
              settlement({ status_akseptasi: '1', nilai_akseptasi_sen: 30_000, tipe_pdf_lod: '15', nama_tipe_pdf_lod: 'LOD Final' }),
              settlement({ tipe_pembayaran: '4', nama_tipe_pembayaran: 'Adjuster Fee', status_akseptasi: '1', nilai_akseptasi_sen: 12_000 }),
              settlement({ status_akseptasi: '', nilai_akseptasi_sen: 99_900 }),
            ],
          },
          { id: 'CVG-2', nama: 'Kosong', penyebab_kerugian: '', tsi_sen: 0, spreading: [] },
        ],
      },
    ],
    status_pucl: 0,
    transfer_compliance: false,
    status_proses: 'BERJALAN',
    status_klaim: '',
    flag_klaim: '0',
    status_posisi_progres: 'On Progress',
    tahap_kini: 'pilih-surveyor',
    ...extra,
  }
}

function task(extra: Partial<Task> = {}): Task {
  return {
    id: 'tugas-1',
    klaim_id: 'klaim-1',
    nomor_klaim: 'PNCN.26.0001',
    tahap: 'pilih-surveyor',
    nama_tahap: 'Choose Surveyor',
    antrean: 'WORKLIST',
    workbasket: '',
    pemilik: '90000001',
    dapat_diambil: false,
    tindakan_keluar: '',
    dibuat_pada: '2026-06-10T03:00:00Z',
    ...extra,
  }
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

beforeEach(() => {
  installFetch()
  useSession.setState({
    token: 'token-uji',
    user: { identitas: '90000001', nama: 'Contoh', jenis: 'KARYAWAN', login: 'x', email: '', perusahaan: 'ASM' },
    validUntil: '2026-12-31T00:00:00Z',
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('bagian atas', () => {
  it('lini Travel: Kirim ke Marketing tampil, Kirim ke Admin dan tab Survey tidak', () => {
    wrap(<SurveyorForm klaim={claim({ status_klaim: '1147' }, { lini: '005', jenis_bisnis: 'Travel' })} tugas={task()} />)

    expect(screen.getByRole('heading', { name: 'InputSurveyor' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kirim ke Marketing' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Kirim ke Admin' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Survey' })).not.toBeInTheDocument()
    // Status Klaim tidak tampil untuk Travel (layout !IsTravel).
    expect(screen.queryByLabelText('Status Klaim')).not.toBeInTheDocument()
  })

  it('lini lain menampilkan kode Status Klaim bila namanya tidak ada', () => {
    wrap(<SurveyorForm klaim={claim({ status_klaim: '1147' })} tugas={task()} />)
    expect(screen.getByRole('option', { name: '1147' })).toBeInTheDocument()
  })

  it('lini PA tanpa status: tidak ada tombol kirim ke Admin maupun Marketing', () => {
    wrap(<SurveyorForm klaim={claim({}, { lini: '002', jenis_bisnis: 'PA' })} tugas={task({ tindakan_keluar: 'InputSurveyorPA' })} />)

    expect(screen.getByRole('heading', { name: 'InputSurveyorPA' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim ke Admin' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim ke Marketing' })).not.toBeInTheDocument()
    // Status Klaim tidak tampil untuk PA (sel !IsPA).
    expect(screen.queryByLabelText('Status Klaim')).not.toBeInTheDocument()
  })

  // Kirim ke RCL/PUCL — ClaimSurvey_sect sel 6, `isAnalystPA_PNC || isAnalisatorTravel`.
  //
  // KEDUANYA menguji data klaim, bukan peran penekan tombol: `isAnalisatorTravel`
  // berkelas `ASM-FW-GCNMFW-Work-PNC`, sedangkan rule peran (`IsAnalisator`) berkelas
  // `Data-Admin-Operator-ID`. Lihat catatan pada showSendToRCLPUCL.
  it.each([
    {
      name: 'PA yang sudah ditransfer ke Analyst',
      klaim: claim({ sudah_transfer_analis: true }, { lini: '002', jenis_bisnis: 'PA' }),
      tugas: task(),
    },
    {
      name: 'Travel yang sudah ditransfer ke Analyst',
      klaim: claim({ sudah_transfer_analis: true }, { lini: '005', jenis_bisnis: 'Travel' }),
      tugas: task(),
    },
    {
      name: 'Travel sudah transfer, penekan BUKAN anggota grup Analyst',
      klaim: claim({ sudah_transfer_analis: true }, { lini: '005', jenis_bisnis: 'Travel' }),
      tugas: task({ analis: false }),
    },
  ])('menampilkan Kirim ke RCL/PUCL untuk $name', ({ klaim, tugas }) => {
    wrap(<SurveyorForm klaim={klaim} tugas={tugas} />)
    // Tugasnya dapat dikerjakan pemanggil, sehingga tombolnya membuka modal.
    expect(screen.getByRole('button', { name: 'Kirim ke RCL/PUCL' })).toBeEnabled()
  })

  it('membuka modal KomentarRCLPUCL saat Kirim ke RCL/PUCL ditekan', async () => {
    const user = userEvent.setup()
    wrap(
      <SurveyorForm
        klaim={claim({ sudah_transfer_analis: true }, { lini: '002', jenis_bisnis: 'PA' })}
        tugas={task()}
      />,
    )
    await user.click(screen.getByRole('button', { name: 'Kirim ke RCL/PUCL' }))
    expect(screen.getByRole('dialog', { name: 'Kirim ke RCL/PUCL' })).toBeInTheDocument()
  })

  it('mematikan Kirim ke RCL/PUCL bila tugasnya bukan milik pemanggil', () => {
    wrap(
      <SurveyorForm
        klaim={claim({ sudah_transfer_analis: true }, { lini: '002', jenis_bisnis: 'PA' })}
        tugas={task({ dapat_dikerjakan: false, pemilik: 'ORANG LAIN' })}
      />,
    )
    expect(screen.getByRole('button', { name: 'Kirim ke RCL/PUCL' })).toBeDisabled()
  })

  it.each([
    {
      name: 'PA yang belum ditransfer ke Analyst',
      klaim: claim({}, { lini: '002', jenis_bisnis: 'PA' }),
      tugas: task(),
    },
    {
      name: 'Travel yang belum ditransfer ke Analyst',
      klaim: claim({}, { lini: '005', jenis_bisnis: 'Travel' }),
      tugas: task({ analis: true }),
    },
    {
      name: 'lini lain meski sudah ditransfer ke Analyst',
      klaim: claim({ sudah_transfer_analis: true }),
      tugas: task({ analis: true }),
    },
  ])('tidak menampilkan Kirim ke RCL/PUCL untuk $name', ({ klaim, tugas }) => {
    wrap(<SurveyorForm klaim={klaim} tugas={tugas} />)
    expect(screen.queryByRole('button', { name: 'Kirim ke RCL/PUCL' })).not.toBeInTheDocument()
  })

  it.each([
    { name: 'Marine Cargo menurut jenis bisnis', polis: { lini: '999', jenis_bisnis: 'MarineCargo' } },
    { name: 'Fire menurut jenis bisnis', polis: { lini: '999', jenis_bisnis: 'Fire' } },
    { name: 'Aneka menurut kode bisnis', polis: { lini: '999', jenis_bisnis: 'Lain', kode_bisnis: '10140' } },
  ])('menampilkan tab Survey untuk $name', ({ polis }) => {
    wrap(<SurveyorForm klaim={claim({}, polis)} tugas={task()} />)
    expect(screen.getByRole('tab', { name: 'Survey' })).toBeInTheDocument()
  })

  it('tidak menampilkan pemberitahuan pemilik bila tugas dapat dikerjakan grup', () => {
    wrap(<SurveyorForm klaim={claim()} tugas={task({ pemilik: 'ORANG LAIN', dapat_dikerjakan: true })} />)
    expect(screen.queryByText(/Tugas ini milik/)).not.toBeInTheDocument()
    for (const add of screen.getAllByRole('button', { name: 'Tambah' })) expect(add).toBeEnabled()
  })

  it('mengunci layar bila server menyatakan tugas tidak dapat dikerjakan', () => {
    wrap(<SurveyorForm klaim={claim()} tugas={task({ dapat_dikerjakan: false })} />)
    expect(screen.getByText(/Tugas ini milik 90000001\./, { selector: 'p' })).toBeInTheDocument()
    for (const add of screen.getAllByRole('button', { name: 'Tambah' })) expect(add).toBeDisabled()
  })
})

describe('tab', () => {
  it('menampilkan Input Register dengan tanda hubung dan spreading yang tidak dihapus', async () => {
    wrap(<SurveyorForm klaim={claim()} tugas={task()} />)

    await userEvent.click(screen.getByRole('tab', { name: 'Input Register' }))
    expect(screen.getByText('YES')).toBeInTheDocument()
    expect(screen.getByText('Fire (006)')).toBeInTheDocument()
    const table = screen.getByRole('table', { name: 'Objek, jaminan, dan spreading' })
    const rows = within(table).getAllByRole('row')
    expect(rows[1]).toHaveTextContent('OBJ-1')
    expect(rows[1]).toHaveTextContent('CVG-1')
    expect(rows[1]).toHaveTextContent('10007 60')
    expect(rows[1]).not.toHaveTextContent('Dihapus')
    // Jaminan kedua tidak mengulang nama objek, dan spreading kosong ditulis tanda hubung.
    expect(rows[2]).toHaveTextContent('Kosong')
    expect(rows[2]).not.toHaveTextContent('OBJ-1')
    expect(within(rows[2]!).getByText('—')).toBeInTheDocument()
  })

  it('membuka tab Survey, Unggah Dokumen, dan Progress', async () => {
    wrap(<SurveyorForm klaim={claim()} tugas={task()} />)

    await userEvent.click(screen.getByRole('tab', { name: 'Survey' }))
    expect(await screen.findByRole('button', { name: 'Ajukan Survey' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('tab', { name: 'Unggah Dokumen' }))
    expect(await screen.findByText('Berkas yang sudah diunggah')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('tab', { name: 'Progress Claim & Komunikasi' }))
    expect(await screen.findByRole('button', { name: 'Input Progress Claim' })).toBeInTheDocument()
  })

  it('menyebut klaim tanpa objek pada Input Register dan Adjustment', async () => {
    wrap(<SurveyorForm klaim={claim({ objek: [] })} tugas={task()} />)

    expect(screen.getByText('Klaim ini belum punya objek.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('tab', { name: 'Input Register' }))
    expect(screen.getByText('Klaim ini belum punya objek.')).toBeInTheDocument()
  })
})

describe('grid Adjustment', () => {
  // ShowObjectAdj IsPA: Pekerjaan dan Tanggal Lahir (T_PERSONLIST), Nilai Estimasi, tanpa Lokasi dan Adjuster.
  it('lini PA memakai kolom grid objek PA', () => {
    const base = claim()
    const pa = claim({ objek: base.objek.map((o) => ({ ...o, pekerjaan: 'KARYAWAN', tanggal_lahir: '1989-05-24' })) }, { lini: '002', jenis_bisnis: 'PA' })
    wrap(<SurveyorForm klaim={pa} tugas={task({ tindakan_keluar: 'InputSurveyorPA' })} />)

    const summary = screen.getByRole('table', { name: 'Adjustment dan akseptasi' })
    const header = within(summary).getAllByRole('columnheader').slice(0, 7).map((h) => h.textContent)
    expect(header).toEqual(['', 'Nama Objek', 'Pekerjaan', 'Tanggal Lahir', 'Currency', 'Nilai Estimasi', 'Nilai Akseptasi Klaim'])
    const row = within(summary).getAllByRole('row')[1]!
    expect(row).toHaveTextContent('KARYAWAN')
    expect(row).toHaveTextContent('24/05/89')
    // ObjectCoverageAdj IsPA: baris coverage memuat Penyebab Kerugian.
    const coverage = screen.getAllByRole('table').find((t) => t.querySelector('caption')?.textContent?.startsWith('Jaminan'))!
    expect(within(coverage).getAllByRole('columnheader').slice(0, 5).map((h) => h.textContent)).toEqual(['', 'Nama Coverage', 'Penyebab Kerugian', 'Mata Uang', 'TSI'])
  })

  it('lini PA hanya menampilkan objek yang punya coverage', () => {
    const base = claim()
    const covered = base.objek[0]!
    const pa = claim(
      { objek: [{ ...covered, nama: 'PESERTA TANPA COVERAGE', coverage: [] }, { ...covered, nama: 'PESERTA BERCOVERAGE' }] },
      { lini: '002', jenis_bisnis: 'PA' },
    )
    wrap(<SurveyorForm klaim={pa} tugas={task({ tindakan_keluar: 'InputSurveyorPA' })} />)

    const summary = screen.getByRole('table', { name: 'Adjustment dan akseptasi' })
    expect(summary).not.toHaveTextContent('PESERTA TANPA COVERAGE')
    expect(summary).toHaveTextContent('PESERTA BERCOVERAGE')
  })

  // TrfKomiteButton (IsPA): Transfer ke Analyst pada setiap jaminan yang belum ditandai, tahap Estimation PA.
  it('lini PA tahap Estimation menampilkan Transfer ke Analyst pada jaminan yang belum ditandai', () => {
    const base = claim({ tahap_kini: 'estimasi-pa' }, { lini: '002', jenis_bisnis: 'PA' })
    const pa = {
      ...base,
      objek: base.objek.map((o) => ({
        ...o,
        coverage: o.coverage.map((c, i) => ({ ...c, id: i === 0 ? '10003' : '10009', sudah_transfer_analis: i === 0 })),
      })),
    }
    wrap(<SurveyorForm klaim={pa} tugas={task({ tahap: 'estimasi-pa', nama_tahap: 'Estimation', tindakan_keluar: 'InputSurveyor' })} />)

    const buttons = screen.getAllByRole('button', { name: 'Transfer ke Analyst' })
    expect(buttons).toHaveLength(1)
    // Sel terakhir baris jaminan itu sendiri (sejajar Nama Coverage), di atas grid adjustment-nya.
    expect(buttons[0]!.closest('tr')).toHaveTextContent('Kosong')
  })

  // ValidationAdjustment step 15: Tambah pada jaminan PA tanpa adjustment memanggil NewEstimationPA lebih dulu.
  it('lini PA: Tambah memanggil rute tambah sebelum membuka baris baru', async () => {
    const seen: string[] = []
    vi.stubGlobal('fetch', (url: string) => {
      seen.push(url)
      if (url === '/api/registrasi/mata-uang') return Promise.resolve(json(200, { pilihan: [{ id: 'IDR', nama: 'Rupiah' }] }))
      if (url.endsWith('/adjustment/tambah')) return Promise.resolve(json(200, { klaim: { id: 'klaim-1' } }))
      return Promise.resolve(json(200, { pilihan: [], adjustment: settlement({}), spreading: [] }))
    })
    const base = claim({ tahap_kini: 'estimasi-pa' }, { lini: '002', jenis_bisnis: 'PA' })
    const pa = { ...base, objek: base.objek.map((o) => ({ ...o, coverage: [{ ...o.coverage[1]!, id: '10003' }] })) }
    wrap(<SurveyorForm klaim={pa} tugas={task({ tahap: 'estimasi-pa', nama_tahap: 'Estimation', tindakan_keluar: 'InputSurveyor' })} />)

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(await screen.findByRole('group', { name: 'Adjustment baru' })).toBeInTheDocument()
    expect(seen).toContain('/api/registrasi/klaim/klaim-1/adjustment/tambah')
  })

  it('menjumlahkan nilai akseptasi klaim dan adjuster, lalu membuka editor Tambah', async () => {
    wrap(<SurveyorForm klaim={claim()} tugas={task()} />)

    const summary = screen.getByRole('table', { name: 'Adjustment dan akseptasi' })
    const objectRow = within(summary).getAllByRole('row')[1]!
    // Hanya baris berakseptasi yang dijumlahkan: 300,00 klaim dan 120,00 adjuster fee.
    expect(objectRow).toHaveTextContent('300,00')
    expect(objectRow).toHaveTextContent('120,00')
    expect(await within(objectRow).findByText('Rupiah')).toBeInTheDocument()
    // Tipe PDF (.PDFType) kini dropdown Tipe LOD; pilihan tercetak tetap tampil.
    expect(screen.getAllByRole('option', { name: 'LOD Final' }).length).toBeGreaterThan(0)

    const empty = screen.getByRole('table', { name: 'Adjustment Kosong' })
    expect(within(empty).getByText('Data Tidak Ada')).toBeInTheDocument()

    const [addFirst] = screen.getAllByRole('button', { name: 'Tambah' })
    await userEvent.click(addFirst!)
    // Editor terbuka; seluruh tombol Tambah pada jaminan itu dimatikan selama editor terbuka.
    expect(screen.getAllByRole('button', { name: 'Tambah' })[0]).toBeDisabled()
  })
})

describe('sub-tab Estimasi Pembayaran', () => {
  it('menyebut estimasi tersimpan', async () => {
    estimateAnswer = () => json(200, { klaim: claim(), tugas: task(), jalur: [], jejak_keputusan: null, large_loss: false })
    wrap(<SurveyorForm klaim={claim()} tugas={task()} />)

    await userEvent.click(screen.getByRole('tab', { name: 'Estimasi Pembayaran' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Estimasi disimpan.')).toBeInTheDocument()
  })

  it('menampilkan galat simpan estimasi', async () => {
    estimateAnswer = () => json(500, { kode: 'galat_internal', pesan: 'Estimasi ditolak server.' })
    wrap(<SurveyorForm klaim={claim()} tugas={task()} />)

    await userEvent.click(screen.getByRole('tab', { name: 'Estimasi Pembayaran' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Estimasi belum dapat disimpan')).toBeInTheDocument()
    expect(screen.getByText('Estimasi ditolak server.')).toBeInTheDocument()
  })

  it('mengunci Save bila tugas milik orang lain', async () => {
    wrap(<SurveyorForm klaim={claim()} tugas={task({ pemilik: 'ORANG LAIN' })} />)

    await userEvent.click(screen.getByRole('tab', { name: 'Estimasi Pembayaran' }))
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })
})
