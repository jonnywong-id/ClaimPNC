import { compareCodeUnits } from './sort'

describe('compareCodeUnits', () => {
  it('menghasilkan urutan yang sama persis dengan sort() tanpa pembanding', () => {
    const daftar = ['b', 'B', 'a', 'A', 'É', 'e', '10', '9', '', 'aa', 'a ']
    // sort() tanpa pembanding SENGAJA dipakai di sini: itulah urutan acuan yang harus ditiru.
    // eslint-disable-next-line sonarjs/no-alphabetical-sort
    expect([...daftar].sort(compareCodeUnits)).toEqual([...daftar].sort())
  })

  it('mengembalikan negatif, positif, dan nol', () => {
    expect(compareCodeUnits('a', 'b')).toBe(-1)
    expect(compareCodeUnits('b', 'a')).toBe(1)
    expect(compareCodeUnits('a', 'a')).toBe(0)
  })
})
