import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { DateField, isoToText, textToISO } from './DateField'

function Harness({ onChange }: { onChange: (v: string) => void }) {
  const [value, setValue] = useState('')
  return (
    <DateField
      id="tgl"
      label="Tgl"
      value={value}
      onChange={(v) => {
        setValue(v)
        onChange(v)
      }}
    />
  )
}

describe('DateField', () => {
  it('placeholder DD/MM/YYYY, garis miring otomatis, nilai keluar ISO', async () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    const input = screen.getByLabelText('Tgl')
    expect(input).toHaveAttribute('placeholder', 'DD/MM/YYYY')
    await userEvent.type(input, '01092026')
    expect(input).toHaveValue('01/09/2026')
    expect(onChange).toHaveBeenLastCalledWith('2026-09-01')
  })

  it('tanggal yang tidak ada ditandai, tidak dikirim', async () => {
    const onChange = vi.fn()
    render(<Harness onChange={onChange} />)
    await userEvent.type(screen.getByLabelText('Tgl'), '31/02/2026')
    expect(screen.getByText('Tanggal tidak sah.')).toBeInTheDocument()
    expect(onChange).not.toHaveBeenCalled()
  })

  it('konversi dua arah', () => {
    expect(textToISO('29/02/2024')).toBe('2024-02-29')
    expect(textToISO('29/02/2026')).toBe('')
    expect(isoToText('2026-09-01')).toBe('01/09/2026')
  })
})
