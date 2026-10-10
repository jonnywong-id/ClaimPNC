import { describe, expect, it } from 'vitest'

import { ambil, ambilDaftar } from './dokumen'

describe('ambil', () => {
  const dokumen = {
    ClaimData: {
      DateOfLoss: '2026-07-04',
      ReporterTelp: '',
      ClaimEstimate: 1500000,
      ExGratia: false,
      TanggalSelesaiRawatInap: null,
    },
  }

  /*
    Pembedaan inilah alasan pembaca ini ada.

    Sel kosong yang berarti "isiannya memang belum diisi" dan sel kosong yang berarti
    "jalurnya salah" tampak SAMA PERSIS di layar — dan yang kedua adalah cacat yang tidak
    akan pernah dilaporkan siapa pun, karena tidak ada yang tampak rusak.
  */
  it('membedakan jalur yang tidak ada dari jalur yang ada tetapi kosong', () => {
    expect(ambil(dokumen, 'ClaimData.TidakAda')).toEqual({ ada: false })
    expect(ambil(dokumen, 'ClaimData.ReporterTelp')).toEqual({ ada: true, nilai: '' })
  })

  // `null` berarti jalurnya ADA, hanya nilainya yang belum diisi.
  it('memperlakukan null sebagai ada tetapi kosong', () => {
    expect(ambil(dokumen, 'ClaimData.TanggalSelesaiRawatInap')).toEqual({ ada: true, nilai: '' })
  })

  // Angka dan boolean memang tersimpan begitu di dokumennya; mengosongkannya akan
  // menyembunyikan isian yang sebenarnya terisi.
  it('membawa angka dan boolean apa adanya', () => {
    expect(ambil(dokumen, 'ClaimData.ClaimEstimate')).toEqual({ ada: true, nilai: '1500000' })
    expect(ambil(dokumen, 'ClaimData.ExGratia')).toEqual({ ada: true, nilai: 'false' })
  })

  it('tidak menembus nilai yang bukan objek', () => {
    expect(ambil(dokumen, 'ClaimData.DateOfLoss.Lagi')).toEqual({ ada: false })
  })
})

describe('ambilDaftar', () => {
  // Pega menulis satu baris sebagai OBJEK, bukan larik berisi satu. Layar yang hanya menerima
  // larik akan menampilkan grid kosong untuk klaim berobjek tunggal.
  it('menerima satu baris yang ditulis sebagai objek', () => {
    const satu = { ClaimData: { ObjectList: { ObjectName: 'Rumah' } } }
    expect(ambilDaftar(satu, 'ClaimData.ObjectList')).toEqual([{ ObjectName: 'Rumah' }])
  })

  it('membuang isi larik yang bukan objek', () => {
    const campur = { ClaimData: { ObjectList: [{ ObjectName: 'A' }, null, 'bukan objek'] } }
    expect(ambilDaftar(campur, 'ClaimData.ObjectList')).toEqual([{ ObjectName: 'A' }])
  })

  it('mengembalikan daftar kosong bila jalurnya tidak ada', () => {
    expect(ambilDaftar({}, 'ClaimData.ObjectList')).toEqual([])
  })
})
