import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  BusinessChoiceListResponse,
  BusinessDocumentRuleBatchInput,
  BusinessDocumentRuleCreateResponse,
  BusinessDocumentRuleInput,
  BusinessDocumentRuleListResponse,
  BusinessDocumentRuleResponse,
  MasterChoiceListResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/tipe-dokumen-bisnis'
const ROUTE_BUSINESS = '/api/master/bisnis-pilihan'
const ROUTE_DOCUMENT_TYPE = '/api/master/tipe-dokumen-pilihan'
const ROUTE_DETAIL_DOCUMENT = '/api/master/detail-dokumen-pilihan'
const ROUTE_OBJECT_DOCUMENT = '/api/master/objek-dokumen-pilihan'

/**
 * Seluruh kunci cache menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * daftar itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 */
function businessKey(portal: string | null, token: string | null) {
  return ['daftar-tipe-dokumen-bisnis', portal, token] as const
}

function ruleListKey(businessID: string, portal: string | null, token: string | null) {
  return ['daftar-tipe-dokumen-bisnis', 'bisnis', businessID, portal, token] as const
}

function ruleKey(id: string, portal: string | null, token: string | null) {
  return ['daftar-tipe-dokumen-bisnis', 'aturan', id, portal, token] as const
}

function choiceKey(name: string, portal: string | null, token: string | null) {
  return ['daftar-tipe-dokumen-bisnis', 'pilihan', name, portal, token] as const
}

/**
 * Hook grid tingkat pertama: lini bisnis yang SUDAH punya aturan dokumen.
 *
 * Menggantikan `RDB List/BrowseLSTDetailDocument_sql`.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useBusinessList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: businessKey(portal, token),
    queryFn: () => callAPI<BusinessChoiceListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook grid tingkat kedua: aturan dokumen milik satu lini bisnis.
 *
 * Menggantikan `RDB List/Select_TYPE_DOCUMENT`.
 *
 * `businessID` null berarti belum ada bisnis yang dipilih — hook-nya diam.
 */
export function useBusinessDocumentRuleList(businessID: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ruleListKey(businessID ?? '', portal, token),
    queryFn: () =>
      callAPI<BusinessDocumentRuleListResponse>(
        `${ROUTE}/bisnis/${encodeURIComponent(businessID ?? '')}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && businessID !== null,
  })
}

/**
 * Hook satu baris LENGKAP dengan daftar jaminannya.
 *
 * Dimuat terpisah, bukan diambil dari hasil daftar, karena daftar memang tidak
 * membawanya. Memakai baris dari daftar akan membuat layar mengira setiap baris tidak
 * punya jaminan — dan itu perbedaan antara dokumen yang wajib dan yang tidak.
 */
export function useBusinessDocumentRule(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ruleKey(id ?? '', portal, token),
    queryFn: () =>
      callAPI<BusinessDocumentRuleResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && id !== null,
  })
}

/**
 * Keempat hook daftar pilihan.
 *
 * Seluruhnya MENUNTUT portal: keempat masternya hidup di basis data setiap entitas.
 *
 * Daftarnya jarang berubah, sehingga tidak dimuat ulang setiap kali form dibuka.
 * Kegagalannya TIDAK menghalangi penyimpanan — isiannya tetap dapat diketik sendiri,
 * sehingga yang hilang hanya kenyamanan memilih.
 */
function useChoiceList(name: string, route: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: choiceKey(name, portal, token),
    queryFn: () => callAPI<MasterChoiceListResponse>(route, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/** Daftar SELURUH lini bisnis — termasuk yang belum punya satu aturan pun. */
export function useBusinessChoiceList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: choiceKey('bisnis', portal, token),
    queryFn: () => callAPI<BusinessChoiceListResponse>(ROUTE_BUSINESS, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/** Daftar tahap dokumen — REGISTER, SURVEY, COMMITEE, PAYMENT, SALVAGE, COLLECTING DOCUMENT. */
export function useDocumentTypeChoiceList() {
  return useChoiceList('tipe-dokumen', ROUTE_DOCUMENT_TYPE)
}

/**
 * Daftar rincian dokumen, beserta tahap pemiliknya di `id_induk`.
 *
 * Layar menyaringnya menurut Tipe Dokumen yang sudah dipilih pada baris yang sama —
 * penyaringan yang di Pega dikerjakan server lewat parameter `idDocument`.
 */
export function useDetailDocumentChoiceList() {
  return useChoiceList('detail-dokumen', ROUTE_DETAIL_DOCUMENT)
}

/** Daftar objek dokumen. */
export function useObjectDocumentChoiceList() {
  return useChoiceList('objek-dokumen', ROUTE_OBJECT_DOCUMENT)
}

/**
 * Hook penambahan: beberapa bisnis dikali beberapa baris dokumen.
 *
 * Seluruh daftar dibuang dari cache setelahnya, bukan hanya daftar bisnisnya: baris baru
 * dapat lahir pada bisnis mana pun yang dipilih, dan menebak yang mana berarti menyusun
 * ulang di peramban apa yang sudah diketahui server.
 */
export function useCreateBusinessDocumentRule() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: BusinessDocumentRuleBatchInput) =>
      callAPI<BusinessDocumentRuleCreateResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['daftar-tipe-dokumen-bisnis'] })
    },
  })
}

/** Hook penyuntingan satu baris aturan. */
export function useUpdateBusinessDocumentRule() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: BusinessDocumentRuleInput }) =>
      callAPI<BusinessDocumentRuleResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    // Daftar DAN baris yang disunting keduanya dibuang dari cache. Tanpa yang kedua,
    // membuka kembali baris yang sama akan menampilkan isi sebelum perubahan — dan
    // pengguna akan mengira penyimpanannya gagal.
    onSuccess: (_result, variables) => {
      void client.invalidateQueries({ queryKey: ['daftar-tipe-dokumen-bisnis'] })
      void client.invalidateQueries({ queryKey: ruleKey(variables.id, portal, token) })
    },
  })
}

/**
 * Hook penambahan satu jaminan.
 *
 * MENAMBAH, tidak mengganti daftarnya — dan tidak ada hook pasangannya untuk membuang,
 * karena tidak ada jalur hapus terhadap tabel itu di seluruh sistem lama. Menambahkannya
 * di sini berarti mengarang kemampuan yang dapat mengubah dokumen wajib menjadi tidak
 * wajib pada klaim yang sedang berjalan.
 */
export function useAddBusinessDocumentRuleCoverage() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, coverageID }: { id: string; coverageID: string }) =>
      callAPI<BusinessDocumentRuleResponse>(`${ROUTE}/${encodeURIComponent(id)}/jenis-klaim`, {
        metode: 'POST',
        body: { id_jenis_klaim: coverageID },
        token,
        portal,
      }),
    onSuccess: (_result, variables) => {
      void client.invalidateQueries({ queryKey: ruleKey(variables.id, portal, token) })
    },
  })
}
