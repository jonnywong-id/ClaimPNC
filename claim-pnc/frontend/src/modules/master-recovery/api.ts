import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { blobAPI, callAPI, downloadAPI, uploadAPI } from '@/api/client'
import type {
  RecoveryClaimLineResponse,
  RecoveryDocumentResponse,
  RecoveryFormResponse,
  RecoveryInput,
  RecoveryListResponse,
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
      client.invalidateQueries({ queryKey: keyOf('principal', portal, token) })
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
      client.invalidateQueries({ queryKey: keyOf('form', portal, token) })
      // Tab Outstanding ikut dimuat ulang supaya batch yang baru disimpan langsung
      // terlihat. Di layar lama, hal ini menuntut petugas menekan Refresh sendiri.
      client.invalidateQueries({ queryKey: keyOf('daftar', portal, token) })
    },
  })
}

/**
 * Hook tab **Outstanding** — daftar batch recovery yang sudah tercatat.
 *
 * Mengisi grid `Data_BACTH_RECOVERY.pxResults` pada
 * `Section/OutstandingMasterRecovery-Section.xml:17510`.
 *
 * # Kenapa hook ini ada, padahal sempat dinyatakan tidak perlu
 *
 * Kesimpulan sebelumnya — "layar lama tidak punya daftar" — KELIRU. Ia disimpulkan dari
 * tidak adanya kueri pembaca di export, padahal export itu sendiri tidak lengkap
 * (`R-16`): rule yang memuat halaman grid itu memang hilang, tetapi grid-nya ada dan
 * berisi data di Pega yang berjalan.
 *
 * # Paginasi dari server
 *
 * Sepuluh baris per halaman, sama dengan `pyRDLPageSize` grid lama. Pencarian principal
 * dan penyaringan tahun juga dikerjakan server — bukan di peramban — supaya jumlah baris
 * yang disebut layar selalu jumlah yang benar-benar ada, bukan jumlah yang kebetulan
 * sudah terunduh.
 */
export function useRecoveryList(filter: { cari?: string; tahun?: string; halaman?: number } = {}) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const cari = filter.cari?.trim() ?? ''
  const tahun = filter.tahun?.trim() ?? ''
  const halaman = Math.max(1, filter.halaman ?? 1)

  const query = new URLSearchParams()
  if (cari !== '') query.set('cari', cari)
  if (tahun !== '') query.set('tahun', tahun)
  query.set('limit', String(PAGE_SIZE))
  query.set('lewati', String((halaman - 1) * PAGE_SIZE))

  return useQuery({
    // Filter ikut ke kunci cache: dua pencarian berbeda adalah dua hasil berbeda, dan
    // menyatukannya akan menampilkan hasil pencarian sebelumnya sesaat setelah kata
    // kuncinya diubah.
    queryKey: [...keyOf('daftar', portal, token), cari, tahun, halaman] as const,
    queryFn: () => callAPI<RecoveryListResponse>(`${ROUTE}/?${query.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null,
    // Sependek bekal awal layar, dan karena alasan yang sama: petugas lain dapat menambah
    // batch kapan saja, sehingga daftar yang ditahan lama akan keliru tanpa terlihat.
    staleTime: 0,
  })
}

/** Jumlah baris per halaman — sama dengan `pyRDLPageSize` grid Outstanding. */
export const PAGE_SIZE = 10

/**
 * Hook tombol **View Document** pada grid dalam.
 *
 * Ia bukan `<a href>` biasa karena rutenya berada di balik sesi dan portal, dan peramban
 * tidak mengirim kedua header itu pada navigasi — persoalan yang sama dengan unduhan
 * berkas contoh.
 *
 * # Ditampilkan atau diunduh, ditentukan SERVER
 *
 * Versi pertama selalu membuka tab baru. Akibatnya lampiran yang tidak dapat digambar
 * peramban — XLSX pada portal ASM — muncul sebagai berhalaman-halaman karakter acak.
 *
 * Sekarang keputusannya dibaca dari `Content-Disposition`: `inline` dibuka di tab baru,
 * selebihnya diunduh dengan namanya. Yang menentukan adalah daftar jenis aman di server;
 * menebaknya lagi di sini akan membuat kedua sisi dapat berbeda pendapat.
 *
 * URL objeknya DICABUT setelah dipakai — tanpa itu isinya tertahan di memori peramban
 * selama tab hidup, dan di layar ini isinya dapat berupa pindaian bukti transfer nasabah.
 */
export function useViewDocument() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (id: string) => {
      const berkas = await blobAPI(`${ROUTE}/bukti-bayar/${encodeURIComponent(id)}`, {
        token,
        portal,
      })

      const alamat = URL.createObjectURL(berkas.isi)
      if (berkas.bolehDitampilkan) {
        window.open(alamat, '_blank', 'noopener,noreferrer')
        // Dicabut setelah peramban sempat membacanya. Mencabutnya seketika akan
        // membatalkan tab yang baru saja dibuka.
        globalThis.setTimeout(() => URL.revokeObjectURL(alamat), 60_000)
        return
      }

      const tautan = document.createElement('a')
      tautan.href = alamat
      tautan.download = berkas.nama || 'bukti-bayar'
      tautan.click()
      URL.revokeObjectURL(alamat)
    },
  })
}

/**
 * Memuat ulang tab Outstanding dari luar komponen yang memilikinya.
 *
 * Ada supaya tombol **Refresh** di kepala layar dapat menyegarkan daftar tanpa kuerinya
 * harus diangkat ke layar — daftar tetap dimiliki komponennya sendiri, dan yang dibagikan
 * hanyalah cara menandainya basi.
 */
export function useRefreshRecoveryList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return () => {
    client.invalidateQueries({ queryKey: keyOf('daftar', portal, token) })
  }
}

/**
 * Hook unduhan berkas contoh CSV.
 *
 * Menggantikan tautan **Format File** beserta rule `DownloadFileCSVFormaatter`. Isi
 * contohnya disusun backend dari sumber yang SAMA dengan pembacanya, sehingga keduanya
 * tidak dapat berbeda pendapat. Bentuknya SATU kolom, sama persis dengan berkas contoh
 * sistem lama; yang menyesuaikan adalah pembacanya, yang menerima kolom nilai klaim
 * sebagai opsional.
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
