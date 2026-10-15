import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { callAPI } from "@/api/client";
import { useSelectedPortal } from "@/app/portal";
import { useSession } from "@/app/session";

import type {
  CheckerResponse,
  ClaimDocument,
  ListDocumentsResponse,
  ListResponse,
  MetadataResponse,
  OpenDocumentResponse,
  RejectLetterRequest,
  RejectLetterResponse,
  SendPostAuditRequest,
  SendPostAuditResponse,
  SubmitDecisionRequest,
  SubmitDecisionResponse,
} from "./types";

const PATH = "/api/inbox-compliance";

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kepatuhan satu badan
 * hukum bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ["inbox-compliance", "tab", portal, token] as const,

  list: (
    portal: string | null,
    token: string | null,
    tab: string,
    page: number,
  ) => ["inbox-compliance", "daftar", portal, token, tab, page] as const,

  checker: (portal: string | null, token: string | null, reference: string) =>
    ["inbox-compliance", "checker", portal, token, reference] as const,
};

/**
 * Hook keterangan layar — daftar tab beserta kolomnya.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxcompliance/tab.go`). Menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Hal yang sama berlaku untuk penanda `tersedia` dan kalimat `penghalang` pada tab Post
 * Audit: begitu daftar kolom `POOLDATA.T_CLAIM_COMPLIANCE_H` tiba dan kuerinya ditulis,
 * tab itu hidup tanpa satu baris pun di layar ini disunting.
 */
export function useInboxComplianceMetadata() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan jaringan
    // tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  });
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` yang berisi puluhan juta baris
 * (`D-10`). Berbeda dari layar Inbox Admin — yang atas keputusan Work Owner menarik seluruh
 * baris lalu memotongnya di server — layar ini memotong halamannya di basis data, sehingga
 * permintaan pertama tidak menjadi lebih lambat pada antrean yang panjang.
 *
 * `enabled` dipakai menahan permintaan pada tab yang belum dapat dilayani. Memanggilnya
 * tetap aman — server menjawab 503 dengan penjelasan — tetapi memanggil sesuatu yang sudah
 * pasti gagal hanya menambah galat di log tanpa menambah keterangan apa pun.
 */
export function useInboxComplianceList(
  tab: string,
  page: number,
  enabled: boolean,
) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: keys.list(portal, token, tab, page),
    queryFn: () =>
      callAPI<ListResponse>(buildPath(tab, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean kerja berubah saat petugas lain menyelesaikan pekerjaannya, jadi cache-nya
    // pendek — sama dengan layar antrean kerja lain.
    staleTime: 15 * 1000,
  });
}

/**
 * buildPath menyusun parameter permintaan.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan pada `tab` yang kedua akan menjadi tab bawaan
 * sementara yang pertama memang itu yang diinginkan.
 */
function buildPath(tab: string, page: number): string {
  const params = new URLSearchParams();

  if (tab) params.set("tab", tab);
  if (page > 1) params.set("halaman", String(page));

  const query = params.toString();
  return query ? `${PATH}?${query}` : PATH;
}

/**
 * Hook pengiriman satu klaim dari antrean Compliance ke Post Audit.
 *
 * # Kenapa dua daftar sekaligus yang disegarkan
 *
 * Karena satu pengiriman mengubah keduanya: barisnya MUNCUL di tab Post Audit, dan di Pega
 * klaimnya berhenti menunggu di antrean Compliance. Menyegarkan satu saja membuat layar
 * menampilkan keadaan yang tidak pernah ada.
 *
 * Kuncinya disegarkan menurut awalannya, bukan satu per satu, supaya halaman keberapa pun
 * yang sedang terbuka ikut termuat ulang.
 */
export function useSendToPostAudit() {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const client = useQueryClient();

  return useMutation({
    mutationFn: (body: SendPostAuditRequest) =>
      callAPI<SendPostAuditResponse>(`${PATH}/post-audit`, {
        token,
        portal,
        metode: "POST",
        body,
      }),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["inbox-compliance", "daftar"] });
    },
  });
}

/**
 * Hook pembukaan form Compliance Checker atas satu klaim.
 *
 * # Kenapa `retry: false`
 *
 * Karena galat yang paling mungkin terjadi di sini BUKAN gangguan jaringan melainkan
 * `ErrClaimNotInQueue` — klaimnya sudah diputuskan petugas lain, atau sudah selesai.
 * Mengulanginya tiga kali tidak akan mengubah jawabannya; yang ia lakukan hanyalah menunda
 * pesan yang perlu segera dibaca petugas.
 *
 * # Kenapa `staleTime` nol
 *
 * Berbeda dari daftarnya, form ini dibuka untuk DIKERJAKAN. Menampilkan keputusan yang
 * tersimpan beberapa detik lalu oleh petugas lain — lalu menimpanya — adalah kelas
 * kesalahan yang tidak sepadan dengan satu perjalanan jaringan yang dihemat.
 */
export function useComplianceChecker(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: keys.checker(portal, token, reference),
    queryFn: () =>
      callAPI<CheckerResponse>(`${PATH}/${encodeURIComponent(reference)}`, {
        token,
        portal,
      }),
    enabled: reference !== "" && token !== null && portal !== null,
    retry: false,
    staleTime: 0,
  });
}

/**
 * Hook penyimpanan keputusan Compliance — padanan tombol "Simpan Data".
 *
 * # Tiga cache yang disegarkan, dan kenapa ketiganya
 *
 *   - `checker`  — form itu sendiri, supaya keputusan yang baru tersimpan terbaca saat
 *                  form dibuka ulang.
 *   - `daftar`   — kedua tab. Pada pilihan Bayar/PostAudit barisnya MUNCUL di tab Post
 *                  Audit, dan menyegarkan satu saja membuat layar menampilkan keadaan
 *                  yang tidak pernah ada.
 *
 * Yang TIDAK berubah adalah antrean Compliance di Pega: klaimnya tetap menunggu di sana,
 * karena tabel klaim masih dimiliki Pega (`P-1`). Jadi baris yang baru diputuskan akan
 * tetap tampil di tab Compliance setelah disegarkan — itu BUKAN cache yang basi,
 * melainkan keterbatasan yang dinyatakan di `keterbatasan`.
 */
export function useSubmitComplianceDecision(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const client = useQueryClient();

  return useMutation({
    mutationFn: (body: SubmitDecisionRequest) =>
      callAPI<SubmitDecisionResponse>(
        `${PATH}/${encodeURIComponent(reference)}/keputusan`,
        { token, portal, metode: "POST", body },
      ),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["inbox-compliance", "checker"] });
      client.invalidateQueries({ queryKey: ["inbox-compliance", "daftar"] });
    },
  });
}

/**
 * Menerbitkan tautan baru untuk satu dokumen klaim, lalu membukanya.
 *
 * # Kenapa tautannya diminta setiap kali, bukan disimpan
 *
 * Ia berumur terbatas — `Durasi`, bawaan 3600 detik. Menyimpannya lalu memakainya ulang
 * menghasilkan tombol yang bekerja satu jam pertama lalu diam-diam berhenti.
 *
 * Karena itu ia `useMutation`, bukan `useQuery`: hasilnya TIDAK boleh di-cache.
 */
export function useOpenComplianceDocument(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useMutation({
    mutationFn: (documentID: string) =>
      callAPI<OpenDocumentResponse>(
        `${PATH}/${encodeURIComponent(reference)}/dokumen/` +
          `${encodeURIComponent(documentID)}/tautan`,
        { token, portal, metode: "POST" },
      ),
  });
}

/**
 * Mengunggah satu lampiran klaim.
 *
 * `FormData`, bukan JSON ber-Base64: Base64 membengkakkan badan sepertiga, dan
 * pembengkakan itu sudah ditanggung sekali lagi di server saat adapter menyiapkannya
 * untuk layanan penyimpanan.
 */
export function useUploadComplianceDocument(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const client = useQueryClient();

  return useMutation({
    mutationFn: (berkas: File) => {
      const body = new FormData();
      body.append("berkas", berkas);
      return callAPI<ClaimDocument>(
        `${PATH}/${encodeURIComponent(reference)}/dokumen`,
        { token, portal, metode: "POST", body },
      );
    },

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["inbox-compliance", "checker"] });
    },
  });
}

/**
 * Menghapus satu lampiran klaim.
 *
 * Penghapusannya FISIK — barisnya benar-benar hilang, dan berkasnya ikut dihapus dari
 * penyimpanan. Server menulis satu baris riwayat lebih dulu, dan itulah satu-satunya
 * jejak yang tersisa.
 */
export function useDeleteComplianceDocument(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const client = useQueryClient();

  return useMutation({
    mutationFn: (documentID: string) =>
      callAPI<void>(
        `${PATH}/${encodeURIComponent(reference)}/dokumen/` +
          `${encodeURIComponent(documentID)}`,
        { token, portal, metode: "DELETE" },
      ),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["inbox-compliance", "checker"] });
    },
  });
}

/**
 * Menerbitkan Surat Penolakan — tombol "Generate PDF".
 *
 * # Ia tidak mengunduh apa pun
 *
 * Namanya di Pega "Download Dokumen Reject", tetapi yang terjadi adalah surat
 * DILAMPIRKAN ke klaim (`DownloadPDFReject` langkah 13-25). Karena itu hook ini
 * menyegarkan form setelah berhasil — surat barunya muncul di grid Dokumen, dan dari
 * sanalah ia dibuka, lewat jalur tautan yang sama dengan dokumen lain.
 *
 * Menekan tombol dua kali MENGGANTI suratnya, tidak menumpuk. Jawabannya menyatakan mana
 * yang terjadi lewat `mengganti_surat_sebelumnya`.
 */
export function useGenerateRejectLetter(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const client = useQueryClient();

  return useMutation({
    mutationFn: (isian: RejectLetterRequest) =>
      callAPI<RejectLetterResponse>(
        `${PATH}/${encodeURIComponent(reference)}/surat-penolakan`,
        { token, portal, metode: "POST", body: isian },
      ),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["inbox-compliance", "checker"] });
    },
  });
}

/**
 * Mengambil lampiran klaim pada satu kategori — tombol "Lihat dokumen".
 *
 * Dibaca SAAT DITEKAN, bukan ikut pada pembukaan form: sebagian besar kategori kosong,
 * dan memuat seluruhnya setiap kali form dibuka berarti membaca banyak yang tidak pernah
 * dilihat. `enabled` karena itu mengikuti kategori yang sedang dibuka.
 *
 * Kategori kosong berarti SELURUH lampiran klaim — dipakai dialog Ubah Kategori.
 */
export function useComplianceDocumentsInCategory(
  reference: string,
  category: string | null,
) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);

  return useQuery({
    queryKey: ["inbox-compliance", "dokumen", portal, reference, category],
    enabled: category !== null,
    queryFn: () => {
      const jalur = `${PATH}/${encodeURIComponent(reference)}/dokumen`;
      const tanya = category ? `?kategori=${encodeURIComponent(category)}` : "";
      return callAPI<ListDocumentsResponse>(jalur + tanya, { token, portal });
    },
  });
}

/**
 * Memindahkan satu lampiran ke kategori lain — tombol "Ubah Kategori Dok".
 *
 * Menyegarkan DUA hal setelah berhasil: form (pencacah "Total Sudah Diunggah" berubah
 * pada kategori asal DAN tujuan) serta daftar dokumen per kategori.
 */
export function useChangeComplianceDocumentCategory(reference: string) {
  const token = useSession((state) => state.token);
  const portal = useSelectedPortal((state) => state.alias);
  const client = useQueryClient();

  return useMutation({
    mutationFn: ({
      dokumen,
      kategori,
    }: {
      dokumen: string;
      kategori: string;
    }) =>
      callAPI<void>(
        `${PATH}/${encodeURIComponent(reference)}/dokumen/` +
          `${encodeURIComponent(dokumen)}/kategori`,
        { token, portal, metode: "POST", body: { kategori } },
      ),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ["inbox-compliance", "checker"] });
      client.invalidateQueries({ queryKey: ["inbox-compliance", "dokumen"] });
    },
  });
}
