import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { NumberField } from './NumberField'

/** Pembungkus terkendali — persis cara form memakainya. */
function Terkendali({ awal = 0, onValue }: { awal?: number; onValue?: (n: number) => void }) {
  const [value, setValue] = useState(awal)
  return (
    <>
      <NumberField
        id="kuota"
        label="Max Cari Data"
        value={value}
        onValueChange={(next) => {
          setValue(next)
          onValue?.(next)
        }}
      />
      <output data-testid="nilai">{value}</output>
    </>
  )
}

const kotak = () => screen.getByLabelText('Max Cari Data') as HTMLInputElement

describe('NumberField', () => {
  // Cacat yang dilaporkan petugas: kotak berisi 0, diketik 10, yang tampil 010.
  //
  // Sebabnya isian terkendali yang nilainya berupa ANGKA — React hanya menggambar ulang
  // saat angkanya berubah, sehingga teks yang tidak mengubah angka mengendap di DOM.
  it('mengetik 10 pada kotak bernilai nol menampilkan 10, bukan 010', async () => {
    const user = userEvent.setup()
    render(<Terkendali awal={0} />)

    await user.click(kotak())
    await user.keyboard('10')

    expect(kotak().value).toBe('10')
    expect(screen.getByTestId('nilai')).toHaveTextContent('10')
  })

  // Nol di depan dibuang SEKETIKA, bukan saat disimpan — supaya yang dibaca petugas tidak
  // pernah berbeda dari yang akan tersimpan.
  it('membuang nol di depan saat diketik', async () => {
    const user = userEvent.setup()
    render(<Terkendali awal={0} />)

    await user.click(kotak())
    await user.keyboard('007')

    expect(kotak().value).toBe('7')
    expect(screen.getByTestId('nilai')).toHaveTextContent('7')
  })

  // Nol TUNGGAL dipertahankan. Mengosongkannya kembali di bawah jari petugas membuat
  // isiannya terasa rusak, padahal nilainya benar.
  it('mempertahankan nol tunggal yang diketik', async () => {
    const user = userEvent.setup()
    render(<Terkendali awal={0} />)

    await user.click(kotak())
    await user.keyboard('0')

    expect(kotak().value).toBe('0')
  })

  // Nilai awal KOSONG, bukan 0 — tidak ada angka yang harus dihapus lebih dulu, dan dari
  // keharusan itulah 010 lahir. `placeholder` menyatakan angka yang akan tersimpan.
  it('mulai kosong dengan placeholder 0', () => {
    render(<Terkendali awal={0} />)

    expect(kotak().value).toBe('')
    expect(kotak()).toHaveAttribute('placeholder', '0')
  })

  it('menampilkan nilai bukan nol apa adanya', () => {
    render(<Terkendali awal={8} />)

    expect(kotak().value).toBe('8')
  })

  // Kotak dikosongkan berarti NOL, bukan "belum diisi" — sama dengan yang tersimpan bila
  // dibiarkan kosong sejak awal.
  it('mengosongkan kotak melaporkan nol', async () => {
    const user = userEvent.setup()
    const onValue = vi.fn()
    render(<Terkendali awal={8} onValue={onValue} />)

    await user.clear(kotak())

    expect(kotak().value).toBe('')
    expect(onValue).toHaveBeenLastCalledWith(0)
  })

  // `type="number"` masih meloloskan huruf dan tanda di sebagian peramban. Pada kolom
  // kuota tidak satu pun bermakna.
  it('menolak huruf dan tanda', async () => {
    const user = userEvent.setup()
    render(<Terkendali awal={0} />)

    await user.click(kotak())
    await user.keyboard('1e-2')

    expect(kotak().value).toBe('12')
  })

  // React Hook Form mengembalikan NaN saat isian bilangan dikosongkan, dan undefined
  // sebelum form terisi. Keduanya tidak boleh tergambar sebagai teks di dalam kotak.
  it.each([
    ['NaN', Number.NaN],
    ['undefined', undefined as unknown as number],
  ])('menampilkan kosong, bukan teks, untuk nilai %s', (_nama, nilai) => {
    render(<NumberFieldLuar value={nilai} />)

    expect(kotak().value).toBe('')
  })

  // Nilai yang berganti DARI LUAR — baris lain dipilih, form dimuat ulang — harus terbaca.
  it('mengikuti nilai yang berganti dari luar', () => {
    const { rerender } = render(<NumberFieldLuar value={0} />)
    expect(kotak().value).toBe('')

    rerender(<NumberFieldLuar value={25} />)
    expect(kotak().value).toBe('25')
  })
})

function NumberFieldLuar({ value }: { value: number }) {
  return <NumberField id="kuota" label="Max Cari Data" value={value} onValueChange={() => {}} />
}
