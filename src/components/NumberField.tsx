import { useEffect, useState, type InputHTMLAttributes } from 'react'

import { Field } from './Field'

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> & {
  id: string
  label: string
  /** Nilai sekarang. Kotak kosong berarti nol — lihat catatan di bawah. */
  value: number
  onValueChange: (value: number) => void
  error?: string | undefined
  hint?: string | undefined
}

/**
 * NumberField adalah isian bilangan bulat tak bertanda yang TERKENDALI.
 *
 * # Cacat yang ia tutup
 *
 * Isian `<input type="number">` terkendali yang nilainya berupa ANGKA punya cacat yang
 * tidak kelihatan sampai dipakai: React hanya menggambar ulang ketika ANGKAnya berubah.
 * Mengetik `0` di kotak yang sudah berisi `0` tidak mengubah angkanya, sehingga teks yang
 * baru diketik mengendap di DOM tanpa pernah dirapikan.
 *
 * Akibatnya terlihat di Master Masking: kotak berisi `0`, petugas mengetik `10`, dan yang
 * tampil `010`. Nilai yang tersimpan memang 10 — tetapi yang DIBACA petugas 010, dan pada
 * kolom berisi kuota itu cukup untuk membuat orang ragu apakah isiannya benar.
 *
 * Komponen ini memegang TEKS selama diketik dan melaporkan ANGKA ke pemanggil, sehingga
 * yang tampil selalu sama dengan yang diketik, dan yang tersimpan selalu angkanya.
 *
 * # Kotak kosong berarti nol
 *
 * Nilai awalnya KOSONG, bukan `0`. Kotak yang sudah berisi `0` memaksa petugas menghapusnya
 * lebih dulu sebelum mengetik — dan justru dari situ `010` lahir.
 *
 * Supaya kosong tidak berarti "belum diputuskan", `placeholder` menampilkan `0`: angka yang
 * benar-benar akan tersimpan bila kotaknya dibiarkan kosong. Keduanya menyimpan nilai yang
 * sama, jadi tidak ada arti yang hilang — `0` memang nilai yang sah pada kuota masking.
 *
 * # Nol di depan dibuang saat diketik, bukan saat disimpan
 *
 * `01` menjadi `1` seketika. Merapikannya saat simpan berarti petugas sempat membaca angka
 * yang berbeda dari yang akan tersimpan.
 *
 * # `inputMode="numeric"`, BUKAN `type="number"`
 *
 * Sama seperti Master Recovery, Master Sparepart, Inbox Salvage, dan Daftar Detail Dokumen
 * Travel, yang keempatnya sudah menolak `type="number"` dengan alasannya masing-masing.
 *
 * Alasannya di sini: `type="number"` mengembalikan nilai KOSONG selama ketikan belum sah.
 * Mengetik `1e` membuat peramban melaporkan `""`, sehingga `1` yang sudah diketik ikut
 * terhapus — digit hilang karena karakter yang bahkan tidak kita terima. Dengan `text`,
 * yang menentukan apa yang boleh masuk adalah `sanitize` di bawah, bukan peramban.
 *
 * Ia sekaligus menghilangkan tombol naik-turun. Kuota diketik, bukan dinaikkan satu per
 * satu sampai 100.000, dan layar lama pun hanya menyediakan kotak isian biasa.
 */
export function NumberField({
  value,
  onValueChange,
  max,
  disabled,
  ...rest
}: Props) {
  const [text, setText] = useState(() => toText(value))

  // Nilai dapat berganti dari luar — baris lain dipilih, form dimuat ulang, isian direset.
  // Disamakan hanya bila ANGKAnya memang berbeda; menyamakan teksnya akan menghapus apa
  // yang sedang diketik petugas pada setiap gambar ulang.
  useEffect(() => {
    setText((current) => (toNumber(current) === value ? current : toText(value)))
  }, [value])

  return (
    <Field
      {...rest}
      type="text"
      inputMode="numeric"
      autoComplete="off"
      placeholder="0"
      // Batas panjang, bukan batas nilai. Nilainya diperiksa server, yang menolak beserta
      // nama isiannya; yang dicegah di sini hanya ketikan sepanjang tiga puluh digit.
      maxLength={typeof max === 'number' ? String(max).length : undefined}
      disabled={disabled}
      value={text}
      onChange={(event) => {
        const next = sanitize(event.target.value)
        setText(next)
        onValueChange(toNumber(next))
      }}
    />
  )
}

/**
 * Kosong untuk nol, supaya tidak ada `0` yang harus dihapus lebih dulu.
 *
 * Nilai yang bukan bilangan juga menjadi kosong. React Hook Form mengembalikan `NaN` saat
 * isian bilangan dikosongkan dan `undefined` sebelum form terisi; tanpa penjagaan ini,
 * keduanya tergambar apa adanya sebagai teks `NaN` di dalam kotak kuota.
 */
function toText(value: number): string {
  return Number.isFinite(value) && value !== 0 ? String(value) : ''
}

function toNumber(text: string): number {
  if (text === '') return 0
  const parsed = Number(text)
  return Number.isFinite(parsed) ? parsed : 0
}

/**
 * Menyisakan angka saja, dan membuang nol di depan.
 *
 * `type="number"` tetap meloloskan `e`, `+`, `-`, dan `.` di sebagian peramban, dan pada
 * isian kuota tidak satu pun bermakna. Nol tunggal dipertahankan — petugas yang mengetik
 * `0` harus melihat `0`, bukan kotak yang kembali kosong di bawah jarinya.
 */
function sanitize(raw: string): string {
  const digit = raw.replace(/\D/g, '')
  if (digit === '') return ''
  const trimmed = digit.replace(/^0+/, '')
  return trimmed === '' ? '0' : trimmed
}
