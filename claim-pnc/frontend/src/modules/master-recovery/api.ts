import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, downloadAPI, uploadAPI } from '@/api/client'
import type {
  RecoveryClaimLineResponse,
  RecoveryDocumentResponse,
  RecoveryFormResponse,
  RecoveryInput,
  RecoveryPolicyReference,
  RecoveryPrincipalListResponse,
  RecoverySaveResponse,
  VirtualAccountResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/recovery'

/**
 * Kunci cache selalu menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (`R-20`).
 *
 * Di modul ini akibatnya paling berat: yang di-cache memuat NOMOR REKENING VIRTUAL, dan
 * nomor milik entitas lain yang tampil di layar dapat mengarahkan dana ke rekening yang
 * keliru.
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 */
function keyOf(part: string, portal: string | null, token: string | null) {
  return ['master-recovery', part, portal, token] as const
}

/**
 * Hook bekal awal layar: nomor batch perkiraan dan pilihan tahun.
 *
 * Menggantikan `Activity/GetIDMasterRecoveryKlaim-Act.xml` beserta pengisian dropdown
 * Tahun, yang di sistem lama dijalankan saat harness dibuka.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useRecoveryForm() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keyOf('form', portal, token),
    queryFn: () => callAPI<RecoveryFormResponse>(`${ROUTE}/form`, { token, portal }),
    enabled: token !== null && portal !== null,
    // TIDAK di-cache lama, berbeda dari layar master lain.
    //
    // Nomor batch adalah perkiraan yang berubah setiap kali petugas mana pun menyimpan.
    // Menahannya lima menit akan membuat angka yang ditampilkan semakin jauh dari
    // kenyataan tanpa ada yang menyadarinya.
    staleTime: 0,
  })
}

/**
 * Hook daftar principal untuk isian "Nama Principal".
 *
 * Menggantikan pra-aktivitas `getDataAllMSTVA` yang mengisi autocomplete pada layar lama.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi: master Virtual Account berisi dua baris
 * pada portal ASM, dan ia daftar yang bertambah beberapa baris per tahun. Layar yang
 * datanya besar — inbox dan laporan — tidak boleh mengikuti pola ini.
 */
export function useRecoveryPrincipals() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keyOf('principal', portal, token),
    queryFn: () => callAPI<RecoveryPrincipalListResponse>(`${ROUTE}/principal`, { token, portal }),
    enabled: token !== null && portal !== null,
    // Master principal nyaris tidak berubah dalam satu sesi kerja. Penerbitan VA baru
    // membatalkan cache ini secara eksplisit di bawah, sehingga principal yang baru
    // didaftarkan langsung muncul sebagai pilihan.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook pencarian identitas polis.
 *
 * Menggantikan `RDB List/GetRecoveryClaimData-SQL.xml`. Di sistem lama pencarian ini
 * berjalan sebagai bagian dari aksi simpan, sehingga nomor polis yang salah ketik baru
 * ketahuan SETELAH batch tersimpan dengan keempat identitas kosong. Di sini ia dipanggil
 * saat petugas menekan Cari, supaya hasilnya terlihat sebelum menyimpan.
 *
 * Ia mutation, bukan query, meski hanya membaca: yang memicunya adalah tindakan petugas,
 * bukan pembukaan layar. Query akan berjalan sendiri pada setiap perubahan isian — dan
 * kueri ini menyeberang DB Link ke basis data lain.
 */
export function useLookupPolicy() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (policyNo: string) =>
      callAPI<RecoveryPolicyReference>(`${ROUTE}/polis/${encodeURIComponent(policyNo)}`, {
        token,
        portal,
      }),
  })
}

/**
 * Hook penerbitan rekening virtual.
 *
 * Menggantikan tombol **Generated VA** beserta `Activity/GeneratedVAClaimRecovery-Act.xml`.
 *
 * Daftar principal dimuat ulang setelah berhasil, sehingga principal yang baru didaftarkan
 * langsung dapat dipilih pada form di bawahnya tanpa memuat ulang halaman.
 */
export function useIssueVirtualAccount() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: { client_id: string; nama_principal: string; email_inputor_va: string }) =>
      callAPI<VirtualAccountResponse>(`${ROUTE}/virtual-account`, {
        metode: 'POST',
        body,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keyOf('principal', portal, token) })
    },
  })
}

/**
 * Hook unggahan Bukti Bayar.
 *
 * Menggantikan tombol **Upload Document** beserta `Call PNCSaveAttachmentToDB`. Yang
 * dikembalikan adalah DATAID pada POOLDATA.DATA_ATTACHFILE, yang kemudian dikirim kembali
 * bersama permintaan simpan.
 */
export function useUploadPaymentProof() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: ({ berkas, keterangan }: { berkas: File; keterangan?: string }) =>
      uploadAPI<RecoveryDocumentResponse>(`${ROUTE}/bukti-bayar`, {
        berkas,
        ...(keterangan ? { keterangan } : {}),
        token,
        portal,
      }),
  })
}

/**
 * Hook pembacaan berkas CSV daftar klaim.
 *
 * Menggantikan tombol **Upload Data Klaim**. Ia TIDAK menyimpan apa pun — barisnya
 * dikembalikan untuk ditampilkan, lalu dikirim kembali bersama permintaan simpan, persis
 * seperti sistem lama menyusunnya di klipboard sebelum menyimpan.
 */
export function useReadClaimLine() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (berkas: File) =>
      uploadAPI<RecoveryClaimLineResponse>(`${ROUTE}/baris-klaim`, { berkas, token, portal }),
  })
}

/**
 * Hook Transfer Recovery — menyimpan satu batch.
 *
 * Menggantikan tombol **Transfer Recovery** beserta
 * `Activity/Insert_mst_recoveryKlaimASM-Act.xml`.
 *
 * Bekal awal dimuat ulang setelah berhasil supaya nomor batch perkiraan untuk entri
 * berikutnya mencerminkan yang baru saja tersimpan.
 */
export function useSaveRecovery() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: RecoveryInput) =>
      callAPI<RecoverySaveResponse>(`${ROUTE}/`, { metode: 'POST', body, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keyOf('form', portal, token) })
    },
  })
}

/**
 * Hook unduhan berkas contoh CSV.
 *
 * Menggantikan tautan **Format File** beserta rule `DownloadFileCSVFormaatter`, yang tidak
 * ada di export — isi contohnya karena itu disusun backend dari sumber yang SAMA dengan
 * pembacanya, sehingga keduanya tidak dapat berbeda pendapat.
 *
 * Ia bukan tautan `<a href>` biasa: rutenya berada di balik sesi dan portal, dan peramban
 * tidak mengirim kedua header itu pada navigasi.
 */
export function useDownloadTemplate() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: () =>
      downloadAPI(`${ROUTE}/format-unggahan`, 'format-data-klaim-recovery.csv', {
        token,
        portal,
      }),
  })
}
