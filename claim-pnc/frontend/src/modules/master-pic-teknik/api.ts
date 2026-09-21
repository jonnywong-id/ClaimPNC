import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  EmployeeResponse,
  TechnicianInput,
  TechnicianListResponse,
  TechnicianResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/pic-teknik'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (ADR-0030). Tanpa itu, berpindah entitas akan menampilkan daftar entitas sebelumnya
 * dari cache — pengguna melihat daftar petugas yang masuk akal, dan tidak ada apa pun di
 * layar yang menandakan orang-orang itu milik badan hukum lain (R-20).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-pic-teknik', portal, token] as const
}

/**
 * Hook daftar Master PIC Teknik.
 *
 * Menggantikan Report Definition `BrowseVMstUserTeknis_RD` yang mengisi grid layar
 * `UserTeknisInbox`.
 *
 * # Daftarnya hanya memuat petugas AKTIF
 *
 * Itu bukan pilihan layar melainkan perilaku server, dan server meniru Report Definition
 * lama yang mematok `STS_AKTIF = '1'` sebagai penyaring tetap. Petugas nonaktif tetap
 * dapat dibuka lewat ID — lihat useTechnician.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server: isinya puluhan baris, dan Report
 * Definition lama pun memuat seluruhnya dengan batas `pyMaxRecords=500`. Layar yang
 * datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useTechnicianList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<TechnicianListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
    // Master nyaris tidak pernah berubah dalam satu sesi kerja. Lima menit menahan
    // pemuatan ulang yang tidak perlu, sementara tombol Refresh tetap tersedia bagi
    // pengguna yang tahu datanya baru saja diubah orang lain.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook membuka satu petugas berdasarkan ID.
 *
 * Ia menjangkau petugas NONAKTIF yang tidak muncul di daftar — itulah satu-satunya jalan
 * mengaktifkannya kembali. Dipakai layar saat ID diketik langsung, bukan dipilih dari
 * baris daftar.
 *
 * Dinonaktifkan saat `operatorID` kosong supaya membuka form tambah tidak menembak server.
 */
export function useTechnician(operatorID: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['master-pic-teknik', 'satu', portal, token, operatorID] as const,
    queryFn: () =>
      callAPI<TechnicianResponse>(`${ROUTE}/${encodeURIComponent(operatorID ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && (operatorID ?? '') !== '',
  })
}

/**
 * Hook pencarian pegawai di direktori.
 *
 * Menggantikan `SetMstUserTeknisMstUser_act`, yang di layar Pega berjalan begitu ID
 * operator diisi dan mengisi `TempDcol.MCL_NAME`.
 *
 * # Kenapa mutation, bukan query
 *
 * Pencariannya dipicu TINDAKAN pengguna — menekan Cari atau meninggalkan kolom ID — bukan
 * oleh layar yang terbuka. Sebagai query ia akan menembak direktori pada setiap ketukan
 * huruf, dan direktori itu API luar yang setiap panggilannya berbiaya.
 *
 * Hasilnya dipakai untuk MENGISI form. Server tetap mencarinya ulang saat menyimpan —
 * nama yang dikirim klien tidak pernah dipercaya.
 */
export function useLookupEmployee() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (operatorID: string) =>
      callAPI<EmployeeResponse>(`${ROUTE}/direktori/${encodeURIComponent(operatorID)}`, {
        token,
        portal,
      }),
  })
}

type SaveFields = TechnicianInput & {
  /** Kosong berarti menambah; terisi berarti mengubah petugas dengan ID itu. */
  ubah?: boolean
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempDcol` untuk keduanya. Perbedaannya hanya pada metode dan jalur, dan itu satu baris.
 *
 * Pada pengubahan, `id_operator` tetap ikut di badan permintaan tetapi SERVER
 * MENGABAIKANNYA — yang dipakai adalah yang di jalur URL. Itu disengaja: dua sumber untuk
 * satu nilai berarti keduanya dapat berbeda, dan yang menang menjadi soal urutan baca.
 */
export function useSaveTechnician() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ ubah, ...input }: SaveFields) =>
      callAPI<TechnicianResponse>(
        ubah ? `${ROUTE}/${encodeURIComponent(input.id_operator)}` : ROUTE,
        {
          metode: ubah ? 'PUT' : 'POST',
          body: input,
          token,
          portal,
        },
      ),
    onSuccess: () => {
      // Daftar dimuat ulang dari server, BUKAN disunting di cache. Nama petugas hanya
      // diketahui server — ia dicari ke direktori saat menyimpan — dan menebaknya di klien
      // akan menampilkan nama yang salah sampai muat ulang berikutnya.
      //
      // Menonaktifkan petugas juga MENGHILANGKAN barisnya dari daftar; cache yang disunting
      // sendiri tidak akan mencerminkan itu.
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
      void client.invalidateQueries({ queryKey: ['master-pic-teknik', 'satu'] })
    },
  })
}
