import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactElement } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { Button } from './Button'
import { ComboField } from './ComboField'
import { ErrorMessage } from './ErrorMessage'
import { Field } from './Field'
import { FormField } from './FormField'
import * as Icons from './Icon'
import { SelectField } from './SelectField'
import { TabBar } from './TabBar'
import { TextAreaField } from './TextAreaField'
import {
  centsToRupiah,
  formatDate,
  formatDateTimeWIB,
  formatPegaDateTime,
  formatPegaFormDate,
  formatPercent,
  formatRupiah,
  rupiahToCents,
  todayWIB,
} from './format'

/** Uji komponen bersama yang dipakai seluruh layar. */

afterEach(() => {
  vi.useRealTimers()
})

describe('Button', () => {
  it('bernada kedua dan bertipe button secara bawaan, lalu menggabungkan kelas tambahan', () => {
    render(
      <>
        <Button>Bawaan</Button>
        <Button tone="utama" type="submit" className="ekstra">
          Utama
        </Button>
        <Button tone="halus">Halus</Button>
      </>,
    )

    const bawaan = screen.getByRole('button', { name: 'Bawaan' })
    expect(bawaan).toHaveAttribute('type', 'button')
    expect(bawaan.className).toContain('border-slate-300')

    const utama = screen.getByRole('button', { name: 'Utama' })
    expect(utama).toHaveAttribute('type', 'submit')
    expect(utama.className).toContain('bg-blue-600')
    expect(utama.className.endsWith(' ekstra')).toBe(true)
    expect(screen.getByRole('button', { name: 'Halus' }).className).toContain('text-slate-600')
  })
})

describe('ErrorMessage', () => {
  it('membedakan nada penolakan dan gangguan', () => {
    render(
      <>
        <ErrorMessage title="Ditolak" description="Isian salah." tone="penolakan" />
        <ErrorMessage title="Gangguan" description="Server mati." tone="gangguan" />
      </>,
    )

    const [penolakan, gangguan] = screen.getAllByRole('alert')
    expect(penolakan).toHaveTextContent('DitolakIsian salah.')
    expect(penolakan?.className).toContain('border-red-200')
    expect(gangguan?.className).toContain('border-amber-200')
    expect(screen.getByText('Gangguan').className).toContain('text-amber-900')
  })
})

describe('Field', () => {
  it('menautkan petunjuk tanpa ikon dan meneruskan kelas tambahan', () => {
    render(<Field id="f" label="Nama" hint="Isi nama." className="ekstra" />)

    const input = screen.getByLabelText('Nama')
    expect(input).toHaveAttribute('aria-invalid', 'false')
    expect(input).toHaveAttribute('aria-describedby', 'f-petunjuk')
    expect(input.className).toContain('px-3')
    expect(input.className).toContain('ekstra')
    expect(screen.getByText('Isi nama.')).toHaveAttribute('id', 'f-petunjuk')
  })

  it('mengutamakan galat atas petunjuk dan mewarnai ikonnya', () => {
    const { container } = render(
      <Field id="g" label="Kode" hint="tidak tampil" error="Kode wajib." icon={<b>i</b>} />,
    )

    const input = screen.getByLabelText('Kode')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAttribute('aria-describedby', 'g-galat')
    expect(input.className).toContain('pl-10')
    expect(screen.queryByText('tidak tampil')).not.toBeInTheDocument()
    expect(screen.getByText('Kode wajib.')).toBeInTheDocument()
    expect(container.querySelector('span[aria-hidden="true"]')?.className).toContain('text-red-400')
  })

  it('tidak menulis apa pun di bawah isian tanpa galat maupun petunjuk', () => {
    const { container } = render(<Field id="h" label="Kosong" icon={<b>i</b>} disabled />)

    expect(screen.getByLabelText('Kosong')).toBeDisabled()
    expect(screen.getByLabelText('Kosong')).not.toHaveAttribute('aria-describedby')
    expect(container.querySelector('span[aria-hidden="true"]')?.className).toContain('text-slate-400')
    expect(container.querySelectorAll('p')).toHaveLength(0)
  })
})

describe('FormField', () => {
  it('menandai kegagalan dan meneruskan kelas tambahan', () => {
    render(
      <>
        <FormField id="a" label="Biasa" />
        <FormField id="b" label="Salah" failure="Wajib diisi." className="ekstra" />
      </>,
    )

    expect(screen.getByLabelText('Biasa')).toHaveAttribute('aria-invalid', 'false')
    expect(screen.getByLabelText('Biasa').className).toContain('border-slate-300')
    const salah = screen.getByLabelText('Salah')
    expect(salah).toHaveAttribute('aria-invalid', 'true')
    expect(salah).toHaveAttribute('aria-describedby', 'b-failure')
    expect(salah.className).toContain('border-red-400')
    expect(salah.className).toContain('ekstra')
    expect(screen.getByText('Wajib diisi.')).toBeInTheDocument()
  })
})

describe('SelectField', () => {
  it('menulis pilihan kosong bawaan dan menampung nilai kembar', () => {
    render(
      <SelectField
        id="s"
        label="Posisi"
        options={[
          { value: 'All', label: 'All' },
          { value: 'All', label: 'All' },
        ]}
      />,
    )

    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([
      '— pilih —',
      'All',
      'All',
    ])
    expect(screen.getByLabelText('Posisi')).toHaveAttribute('aria-invalid', 'false')
  })

  it('menulis galat, teks kosong sendiri, dan kelas tambahan', () => {
    render(
      <SelectField
        id="t"
        label="Status"
        options={[]}
        emptyText="--Pilih--"
        error="Pilih status."
        className="ekstra"
      />,
    )

    const select = screen.getByLabelText('Status')
    expect(select).toHaveAttribute('aria-describedby', 't-error')
    expect(select.className).toContain('border-red-400')
    expect(select.className).toContain('ekstra')
    expect(screen.getByRole('option', { name: '--Pilih--' })).toBeInTheDocument()
    expect(screen.getByText('Pilih status.')).toBeInTheDocument()
  })
})

describe('TextAreaField', () => {
  it('menautkan petunjuk dan memakai tiga baris secara bawaan', () => {
    render(<TextAreaField id="r" label="Remark" hint="Opsional." />)

    const area = screen.getByLabelText('Remark')
    expect(area).toHaveAttribute('rows', '3')
    expect(area).toHaveAttribute('aria-describedby', 'r-petunjuk')
    expect(area).toHaveAttribute('aria-invalid', 'false')
    expect(screen.getByText('Opsional.')).toBeInTheDocument()
  })

  it('mengutamakan galat dan menggabungkan kelas tambahan', () => {
    render(
      <TextAreaField id="q" label="Alasan" rows={5} hint="x" error="Wajib." className="ekstra" />,
    )

    const area = screen.getByLabelText('Alasan')
    expect(area).toHaveAttribute('rows', '5')
    expect(area).toHaveAttribute('aria-invalid', 'true')
    expect(area).toHaveAttribute('aria-describedby', 'q-galat')
    expect(area.className).toContain('ekstra')
    expect(screen.getByText('Wajib.')).toBeInTheDocument()
    expect(screen.queryByText('x')).not.toBeInTheDocument()
  })

  it('tanpa galat maupun petunjuk tidak menautkan apa pun', () => {
    render(<TextAreaField id="p" label="Bebas" />)
    expect(screen.getByLabelText('Bebas')).not.toHaveAttribute('aria-describedby')
  })
})

describe('ComboField', () => {
  it('menautkan daftar pilihan dan petunjuk', () => {
    const { container } = render(
      <ComboField id="c" label="Kota" options={['Jakarta', 'Bandung']} hint="Pilih atau ketik." />,
    )

    const input = screen.getByLabelText('Kota')
    const listID = input.getAttribute('list')
    expect(listID).toBeTruthy()
    const options = container.querySelectorAll(`datalist option`)
    expect([...options].map((o) => o.getAttribute('value'))).toEqual(['Jakarta', 'Bandung'])
    expect(container.querySelector('datalist')?.id).toBe(listID)
    expect(input).toHaveAttribute('aria-describedby', 'c-petunjuk')
    expect(input).not.toHaveAttribute('aria-invalid')
    expect(input.className).toContain('border-slate-300')
  })

  it('mengutamakan galat dan menggabungkan kelas tambahan', () => {
    render(<ComboField id="d" label="Kota" options={[]} error="Kota wajib." hint="x" className="ekstra" />)

    const input = screen.getByLabelText('Kota')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAttribute('aria-describedby', 'd-galat')
    expect(input.className).toContain('ekstra')
    expect(screen.getByRole('alert')).toHaveTextContent('Kota wajib.')
  })

  it('tanpa galat maupun petunjuk tidak menautkan apa pun', () => {
    render(<ComboField id="e" label="Kota" options={[]} />)
    expect(screen.getByLabelText('Kota')).not.toHaveAttribute('aria-describedby')
  })
})

describe('TabBar', () => {
  it('menandai tab aktif dan melaporkan tab yang dipilih', async () => {
    const onSelect = vi.fn()
    const user = userEvent.setup()
    render(
      <TabBar
        label="Daftar uji"
        active="b"
        onSelect={onSelect}
        tabs={[
          { kode: 'a', nama: 'Satu', keterangan: 'tab satu' },
          { kode: 'b', nama: 'Dua' },
        ]}
      />,
    )

    expect(screen.getByRole('tablist', { name: 'Daftar uji' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Dua' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Satu' })).toHaveAttribute('aria-selected', 'false')
    expect(screen.getByRole('tab', { name: 'Satu' })).toHaveAttribute('title', 'tab satu')

    await user.click(screen.getByRole('tab', { name: 'Satu' }))
    expect(onSelect).toHaveBeenCalledWith('a')
  })
})

describe('Icon', () => {
  it('menggambar setiap ikon sebagai SVG tersembunyi yang meneruskan atribut', () => {
    const entries = Object.entries(Icons).filter(([, value]) => typeof value === 'function')
    expect(entries.length).toBeGreaterThanOrEqual(24)

    for (const [name, Icon] of entries) {
      const Component = Icon as (props: { className?: string; 'data-name'?: string }) => ReactElement
      const { container, unmount } = render(<Component className="h-4" data-name={name} />)
      const svg = container.querySelector('svg')
      expect(svg, name).not.toBeNull()
      expect(svg).toHaveAttribute('aria-hidden', 'true')
      expect(svg).toHaveAttribute('class', 'h-4')
      expect(svg).toHaveAttribute('data-name', name)
      expect(svg?.childElementCount, name).toBeGreaterThan(0)
      unmount()
    }
  })
})

describe('format', () => {
  it('menulis sen sebagai rupiah dan persen empat desimal', () => {
    expect(formatRupiah(100000050)).toBe(`Rp ${(1000000.5).toLocaleString('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`)
    expect(formatPercent(999999)).toBe('99,9999%')
    expect(formatPercent(1000000)).toBe('100%')
    expect(formatPercent(125000)).toBe('12,5%')
  })

  it('membaca tanggal ISO tanpa menggeser zona waktu, dan mengembalikan yang tidak terbaca', () => {
    expect(formatDate('2026-06-01')).toBe('1 Juni 2026')
    expect(formatDate('')).toBe('—')
    expect(formatDate('2026/06/01')).toBe('2026/06/01')
    expect(formatDate('2026-13-01')).toBe('2026-13-01')
    expect(formatDate('2026-00-01')).toBe('2026-00-01')
    expect(formatDate('abcd-01-01')).toBe('abcd-01-01')
  })

  it('menulis waktu RFC3339 dalam WIB', () => {
    expect(formatDateTimeWIB('2026-09-30T00:59:00Z')).toBe('30/09/2026 7:59')
    expect(formatDateTimeWIB('2026-09-30T20:05:00Z')).toBe('01/10/2026 3:05')
    expect(formatDateTimeWIB(undefined)).toBe('')
    expect(formatDateTimeWIB('bukan waktu')).toBe('bukan waktu')
  })

  it('menulis waktu grid Pega sebagai dd/MM/yy HH:mm', () => {
    // Bentuk sel "Tanggal" pada grid Pega: `28/09/26 16:58` — tahun DUA digit dan jam
    // ber-nol di depan, berbeda dari isian baca saja pada form di atas.
    expect(formatPegaDateTime('2026-09-28T16:58:56+07:00')).toBe('28/09/26 16:58')
    expect(formatPegaDateTime('2026-09-30T00:59:00Z')).toBe('30/09/26 07:59')
    expect(formatPegaDateTime('2026-09-30T20:05:00Z')).toBe('01/10/26 03:05')
  })

  it('TIDAK menggeser waktu yang tidak menyebutkan zonanya', () => {
    // Nilai tanpa zona sudah berupa waktu dinding. Menggesernya tujuh jam adalah kelas
    // kesalahan yang melahirkan ratusan penyesuaian manual di sistem lama.
    expect(formatPegaDateTime('2026-09-28 16:58')).toBe('28/09/26 16:58')
    expect(formatPegaDateTime('2026-09-28 16:58:56')).toBe('28/09/26 16:58')
  })

  it('tidak mengarang jam pada nilai yang hanya memuat tanggal', () => {
    expect(formatPegaDateTime('2026-09-28')).toBe('28/09/26')
  })

  it('mengembalikan teks yang tidak terbaca apa adanya', () => {
    // Nilai mentah yang terbaca aneh masih dapat ditelusuri; tanda pisah menghapus jejaknya.
    expect(formatPegaDateTime('bukan waktu')).toBe('bukan waktu')
    expect(formatPegaDateTime('')).toBe('')
    expect(formatPegaDateTime('   ')).toBe('')
  })

  it('menulis isian tanggal pada FORM Pega bertahun empat digit', () => {
    // Bentuk isian "Tanggal Terima Dokumen" di layar kerja: `29/01/2020 11:58`. Berbeda
    // dari sel grid di atas, yang tahunnya dua digit.
    expect(formatPegaFormDate('2020-01-29T11:58:00+07:00')).toBe('29/01/2020 11:58')
    expect(formatPegaFormDate('2026-09-30T00:59:00Z')).toBe('30/09/2026 07:59')
  })

  it('tidak mengarang jam pada isian form yang hanya memuat tanggal', () => {
    // "Tanggal Input Dokumen" ditulis Pega tanpa jam sama sekali. Menambahkan "0:00"
    // mengarang ketelitian yang tidak ada di sumbernya.
    expect(formatPegaFormDate('2020-01-29')).toBe('29/01/2020')
  })

  it('meloloskan nilai form yang memang sudah berupa teks tanggal', () => {
    // Satu kolom penerimaan dokumen disimpan sebagai TEKS `dd/MM/yyyy` oleh procedure yang
    // mengisinya. Ia harus lolos tanpa disentuh, bukan berubah menjadi teks yang salah.
    expect(formatPegaFormDate('09/10/2026')).toBe('09/10/2026')
    expect(formatPegaFormDate('ANYWHERE IN INDONESIA')).toBe('ANYWHERE IN INDONESIA')
    expect(formatPegaFormDate('')).toBe('')
  })

  it('TIDAK menggeser isian form yang tidak menyebutkan zonanya', () => {
    expect(formatPegaFormDate('2020-01-29 11:58')).toBe('29/01/2020 11:58')
  })

  it('menghitung tanggal hari ini menurut WIB', () => {
    vi.useFakeTimers()
    // 18:00 UTC sudah lewat tengah malam di WIB.
    vi.setSystemTime(new Date('2026-12-31T18:00:00Z'))
    expect(todayWIB()).toBe('2027-01-01')
    vi.setSystemTime(new Date('2026-03-04T01:00:00Z'))
    expect(todayWIB()).toBe('2026-03-04')
  })

  it('membaca ketikan rupiah menjadi sen dan sebaliknya', () => {
    expect(rupiahToCents('1.234,56')).toBe(123456)
    expect(rupiahToCents('  ')).toBe(0)
    expect(rupiahToCents('abc')).toBeNaN()
    expect(centsToRupiah(123456)).toBe('1234,56')
    expect(centsToRupiah(0)).toBe('')
  })
})
