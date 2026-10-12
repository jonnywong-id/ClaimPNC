import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it } from 'vitest'

import { ColumnFilter, matchesFilters } from './ColumnFilter'

/** Uji filter kolom grid objek pertanggungan. Data KARANGAN (`D-69`). */

function Harness({ values }: { values: string[] }) {
  const [picked, setPicked] = useState<string[]>([])
  return (
    <div>
      <ColumnFilter column="Nama" values={values} picked={picked} onChange={setPicked} />
      <output>{picked.join(',')}</output>
    </div>
  )
}

describe('ColumnFilter', () => {
  it('mencentang nilai, menyaring dengan Search Text, dan Clear Filter', async () => {
    render(<Harness values={['BUDI', 'ANI', 'BUDI', 'CITRA']} />)
    await userEvent.click(screen.getByRole('button', { name: 'Filter Nama' }))
    // Nilai unik, urut.
    expect(screen.getAllByRole('checkbox')).toHaveLength(3)

    await userEvent.type(screen.getByLabelText('Search Text Nama'), 'bu')
    expect(screen.getAllByRole('checkbox')).toHaveLength(1)
    await userEvent.click(screen.getByRole('checkbox', { name: 'BUDI' }))
    expect(screen.getByRole('status')).toHaveTextContent('BUDI')

    await userEvent.click(screen.getByRole('button', { name: 'Clear Filter' }))
    expect(screen.getByRole('status')).toHaveTextContent('')
  })

  it('matchesFilters: kolom tanpa centang tidak menyaring', () => {
    expect(matchesFilters({ Nama: 'BUDI', 'Tanggal Lahir': '03/02/1996' }, { Nama: [], 'Tanggal Lahir': ['03/02/1996'] })).toBe(true)
    expect(matchesFilters({ Nama: 'BUDI' }, { Nama: ['ANI'] })).toBe(false)
    expect(matchesFilters({ Nama: 'BUDI' }, {})).toBe(true)
  })
})
