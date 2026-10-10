import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppRoute } from "@/app/App";
import { useSelectedPortal } from "@/app/portal";
import { useSession } from "@/app/session";

import type {
  CheckerResponse,
  Decision,
  DocumentChecklistRow,
  SurveyResult,
  WorkItem,
} from "./types";

/**
 * Form Compliance Checker.
 *
 * Yang diuji di sini adalah janji formnya, bukan tata letaknya: keempat pilihan tergambar,
 * isian tersemai dari keputusan yang sudah tersimpan, dan apa yang BENAR-BENAR dikirim saat
 * Simpan ditekan.
 *
 * Tata letaknya sendiri sengaja tidak diuji ketat, karena section aslinya — `CompliancePNC`
 * — HILANG dari export (`R-16`). Menguntuhkan urutan isian berarti mengunci tebakan.
 */

const PATH = "/api/inbox-compliance";
const CLAIM_KEY = "ASM-FW-GCNMFW-WORK PNC-9001";
const CHECKER_PATH = `${PATH}/${encodeURIComponent(CLAIM_KEY)}`;

const SAMPLE_PROFILE = {
  identitas: "90000001",
  nama: "Contoh Administrator",
  jenis: "KARYAWAN",
  login: "adminpnc",
  email: "contoh.admin@example.invalid",
  perusahaan: "ASM",
};

const PORTAL_LIST = {
  portal: [
    { id: "202600101", nama: "ASURANSI SINAR MAS", alias: "ASM", siap: true },
  ],
  utama: "ASM",
};

/**
 * Klaim contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const CLAIM: WorkItem = {
  referensi: CLAIM_KEY,
  nomor_case: "PNC-9001",
  no_klaim: "",
  no_polis: "POL-CONTOH-0001",
  nama_tertanggung: "PT SUMBER CONTOH SENTOSA",
  nama_bisnis: "Aneka",
  nama_cabang: "Cabang Contoh Pusat",
  nama_admin: "ADMINCONTOH1",
  tanggal_kirim_compliance: "2026-09-21",
  tanggal_kirim_post_audit: null,
  catatan_compliance: "",
  aging: "2 days 3 hours ago",
  aging_jam: 51,
  outstanding: "",
};

const CHOICES = [
  { nilai: "0", label: "Fraud / Tolak" },
  { nilai: "1", label: "Bayar / Valid" },
  { nilai: "2", label: "Bayar / PostAudit" },
  { nilai: "3", label: "Lain-Lain" },
];

/**
 * Jawaban POST `/keputusan` yang BERHASIL.
 *
 * Wajib dipakai oleh setiap uji yang benar-benar menekan Simpan atau Kirim. Menjawabnya
 * dengan badan form — yang `keputusan`-nya `null` — membuat layar melempar
 * `Cannot read properties of null`, dan Vitest memperingatkan bahwa galat tak tertangani
 * dapat membuat uji LAIN lulus secara palsu.
 */
const SIMPAN_BERHASIL = {
  keputusan: {
    pilihan: "1",
    pilihan_label: "Bayar / Valid",
    note: "",
    komentar: [],
    catatan_investigator: "",
    diputuskan_oleh: "ADMINCONTOH1",
    diputuskan_pada: "2026-09-22 10:00",
    tanggal_valid: "2026-09-22 10:00",
    tanggal_kirim_post_audit: null,
  },
  post_audit: null,
  keterbatasan: "",
  portal: "ASM",
};

const LIMITATION =
  "Keputusan tersimpan di aplikasi baru. Status klaim di sistem lama belum ikut " +
  "berubah dan klaimnya masih menunggu di antrean Compliance Pega.";

/**
 * Tombol untuk lini selain PA dan Travel, yakni keadaan umum.
 *
 * Ketiga tombol umum selalu tampil. Kedua tombol pengiriman sengaja `false`: masing-masing
 * menuntut `IsPA` dan `IsTravel`, dan klaim contoh di berkas ini bukan keduanya. Uji
 * khusus di bawah yang menyalakannya satu per satu.
 */
const ACTIONS = {
  unggah_dokumen: true,
  unduh_dokumen_reject: true,
  simpan: true,
  kirim_ke_analyst: false,
  kirim_ke_pic_teknik: false,
};

/**
 * Badan jawaban form.
 *
 * `survei` sengaja punya nilai bawaan yang MEMATIKAN blok Hasil Investigasi: klaim contoh
 * di berkas ini bukan PA. Uji khusus yang menyalakannya.
 *
 * Bentuknya dipasang pada `CheckerResponse` lewat `satisfies` supaya isian baru pada
 * kontrak menggagalkan berkas ini saat `tsc` — bukan lolos diam-diam sebagai objek bebas,
 * yang sudah dua kali membuat uji hijau atas layar yang sebenarnya rusak.
 */
function checkerBody(
  keputusan: Decision | null = null,
  tombol = ACTIONS,
  survei: {
    tampilkan_hasil_investigasi: boolean;
    hasil_investigasi: SurveyResult[];
    hasil_investigasi_gagal_dibaca: boolean;
  } = {
    tampilkan_hasil_investigasi: false,
    hasil_investigasi: [],
    hasil_investigasi_gagal_dibaca: false,
  },

  // Tab Dokumen — padam secara bawaan, sama seperti Hasil Investigasi. Klaim contoh
  // bukan Travel, dan tab ini hanya ada pada Travel.
  dokumen: {
    tampilkan_daftar_dokumen: boolean;
    daftar_dokumen: DocumentChecklistRow[];
    daftar_dokumen_gagal_dibaca: boolean;
  } = {
    tampilkan_daftar_dokumen: false,
    daftar_dokumen: [],
    daftar_dokumen_gagal_dibaca: false,
  },
) {
  return {
    klaim: CLAIM,
    pilihan: CHOICES,
    keputusan,
    tombol,
    keterbatasan: LIMITATION,
    ...survei,
    ...dokumen,

    // Pra-isi form Surat Penolakan. Seluruhnya KARANGAN (`D-69`).
    //
    // `tanggal_kejadian` sengaja `null` di sini, bukan teks: itu bentuk yang dikirim
    // server ketika kolomnya kosong, dan dialognya harus menanganinya tanpa menggambar
    // "null" sebagai isian.
    pra_isi_surat_penolakan: {
      nama_pasien: "PESERTA CONTOH",
      tempat_kejadian: "RS CONTOH",
      tanggal_kejadian: null,
    },

    portal: "ASM",
  } satisfies CheckerResponse;
}

// `init` WAJIB ada tetapi boleh `undefined`, bukan opsional. Dengan
// `exactOptionalPropertyTypes` keduanya tidak sama, dan bentuk opsional menolak
// `{ url, init }` ketika `init` memang tidak terisi — persis yang terjadi pada permintaan
// GET. Bentuknya disamakan dengan InboxCompliancePage.test.tsx.
type Call = { url: string; init: RequestInit | undefined };
let calls: Call[] = [];

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal("fetch", (url: string, init?: RequestInit) => {
    calls.push({ url, init });
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === "/api/portal")
      return Promise.resolve(jsonResponse(200, PORTAL_LIST));
    if (url === "/api/menu")
      return Promise.resolve(jsonResponse(200, { menu: [] }));
    return Promise.resolve(answer(url, init));
  });
}

function renderForm() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter
        initialEntries={[`/inbox-compliance/${encodeURIComponent(CLAIM_KEY)}`]}
      >
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/**
 * Menggambar form lalu MENUNGGU isinya tiba, bukan sekadar judulnya.
 *
 * Hasil `render` diteruskan supaya uji yang perlu menggambar dua keadaan berturut-turut
 * dapat melepas yang pertama — tanpa itu keduanya hidup bersamaan dan `getByRole`
 * menemukan dua tab bernama sama.
 */
async function renderLoaded() {
  const hasil = renderForm();
  await screen.findByRole("radio", { name: "Bayar / Valid" });
  return hasil;
}

function lastSubmit(): Call | undefined {
  return [...calls]
    .reverse()
    .find((c) => c.url.endsWith("/keputusan") && c.init?.method === "POST");
}

beforeEach(() => {
  calls = [];
  useSession.setState({
    token: "token-uji",
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  });
  useSelectedPortal.getState().select("ASM");
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("Form Compliance Checker", () => {
  /**
   * Alamat yang diminta membawa kunci teknis Pega dalam bentuk TERKODE.
   *
   * Kuncinya memuat spasi. Mengirimnya mentah menghasilkan alamat yang tidak sah, dan
   * gagalnya berupa 404 yang tidak menyebut sebabnya — jadi yang diuji adalah alamat
   * permintaannya, bukan sekadar bahwa layarnya tergambar.
   */
  it("meminta formnya dengan kunci teknis Pega yang terkode", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    expect(calls.some((c) => c.url === CHECKER_PATH)).toBe(true);
    expect(
      calls.some((c) => c.url.includes("WORK PNC-9001")),
      "spasi pada kunci teknis harus dikodekan",
    ).toBe(false);
  });

  /**
   * Keempat pilihan tergambar, dan urutannya mengikuti server.
   *
   * Urutannya `0,1,2,3` sesuai `pyStandardValue` pada
   * `Property/PilihanCompliance_property.xml`, dan layar Pega menggambarnya dalam urutan
   * yang sama. Mengurutkannya ulang di layar berarti menyimpang tanpa alasan.
   */
  it("menggambar keempat Pilihan Compliance dalam urutan dari server", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    const radios = screen.getAllByRole("radio");
    expect(radios.map((r) => (r as HTMLInputElement).value)).toEqual([
      "0",
      "1",
      "2",
      "3",
    ]);

    for (const choice of CHOICES) {
      expect(
        screen.getByRole("radio", { name: choice.label }),
      ).toBeInTheDocument();
    }
  });

  /**
   * Kotak keterbatasan TIDAK digambar — layar Pega tidak punya kotak seperti itu.
   *
   * # Kenapa uji ini dibalik, bukan dihapus
   *
   * Versi sebelumnya menuntut kotak itu ADA, dengan alasan ia satu-satunya tempat petugas
   * diberi tahu bahwa keputusannya belum mengubah klaim di Pega. Alasan itu **sudah tidak
   * benar lagi**: sejak `ApplyDecisionToClaim`, `AppendHistory`, dan `MoveAssignment`
   * berjalan, keputusan memang mengubah klaim dan memindahkan penugasannya.
   *
   * Work Owner kemudian meminta layar ini sama persis dengan Pega — "jangan ada yang
   * dilebihin dan dikurangin". Kotak itu tambahan kita, maka ia dicabut.
   *
   * Uji ini dibalik, bukan dihapus, supaya pencabutannya **terkunci**. Dihapus begitu saja,
   * tidak ada yang mencegah kotak itu kembali diam-diam pada perubahan berikutnya.
   *
   * Yang masih benar dari kalimat lama — baris antrean milik Pega sendiri tetap menunggu di
   * sana, karena `P-1` melarang kita menulis ke tabel penugasan Pega — pindah ke catatan
   * lingkup penguji, bukan ke layar petugas.
   */
  it("TIDAK menggambar kotak keterbatasan, karena Pega tidak punya", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    expect(screen.queryByText(LIMITATION)).not.toBeInTheDocument();
  });

  /**
   * Deretan tombol mengikuti apa yang SERVER bolehkan, bukan disimpulkan di layar.
   *
   * Syaratnya dibaca dari sel 76–80 layout S14 `Section/CompliancePNC-Section.xml`.
   */
  it("menggambar tombol sesuai yang dibolehkan server", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    expect(
      screen.getByRole("button", { name: "Unggah Dokumen" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Simpan Data" }),
    ).toBeInTheDocument();

    // Klaim contoh bukan PA, sehingga tombol ini TIDAK boleh ada.
    expect(
      screen.queryByRole("button", { name: "Kirim ke Analyst" }),
    ).not.toBeInTheDocument();

    // Tombol kelima Pega tidak pernah dibangun — syaratnya saling meniadakan.
    expect(
      screen.queryByRole("button", { name: /PIC Teknik/ }),
    ).not.toBeInTheDocument();
  });

  /** Pada klaim PA, "Kirim ke Analyst" ikut tergambar. */
  it("menggambar Kirim ke Analyst pada klaim PA", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, { ...ACTIONS, kirim_ke_analyst: true }),
      ),
    );
    await renderLoaded();

    expect(
      screen.getByRole("button", { name: "Kirim ke Analyst" }),
    ).toBeInTheDocument();
  });

  /**
   * Klaim Travel mendapat ketiga tombol umum DITAMBAH "Kirim ke PIC Teknik".
   *
   * # Kenapa uji ini dibalik
   *
   * Versi sebelumnya menuntut klaim Travel tidak mendapat satu tombol pun, termasuk
   * Simpan, dan menuntut layar menjelaskan sebabnya. Premisnya salah: ketiga tombol umum
   * ber-`pyVisible=ALWAYS`, dan yang `!IsTravel` adalah WADAH salah satu bilah tombol.
   * Bilah pasangannya — `ComplianceChecker` S4 — justru ber-`IsTravel`.
   *
   * Dibalik supaya perilaku yang benar terkunci, dan supaya kotak penjelas yang sudah
   * dicabut tidak kembali bersama premisnya.
   */
  it("menggambar tombol Travel, termasuk Kirim ke PIC Teknik", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, {
          unggah_dokumen: true,
          unduh_dokumen_reject: true,
          simpan: true,
          kirim_ke_analyst: false,
          kirim_ke_pic_teknik: true,
        }),
      ),
    );
    await renderLoaded();

    expect(
      screen.getByRole("button", { name: "Simpan Data" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Unggah Dokumen" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Kirim ke PIC Teknik" }),
    ).toBeInTheDocument();

    // Travel bukan PA, jadi tombol ini TIDAK boleh ikut tampil.
    expect(
      screen.queryByRole("button", { name: "Kirim ke Analyst" }),
    ).not.toBeInTheDocument();
  });

  /** Ringkasan klaim tergambar, termasuk kolom yang disembunyikan pada grid. */
  it("menampilkan ringkasan klaim yang sedang diperiksa", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    expect(screen.getByText("PNC-9001")).toBeInTheDocument();
    expect(screen.getByText("POL-CONTOH-0001")).toBeInTheDocument();
    expect(screen.getByText("PT SUMBER CONTOH SENTOSA")).toBeInTheDocument();

    // Nama Bisnis dan Nama Cabang disembunyikan di GRID karena layar Pega hanya
    // menampilkan satu kolom di sana. Alasan itu tidak berlaku pada form.
    expect(screen.getByText("Aneka")).toBeInTheDocument();
    expect(screen.getByText("Cabang Contoh Pusat")).toBeInTheDocument();
  });

  /**
   * Isian tersemai dari keputusan yang SUDAH tersimpan.
   *
   * Petugas yang membuka form kedua kalinya harus melihat pilihan yang sudah dibuatnya,
   * bukan kotak kosong yang membuatnya mengira keputusannya hilang.
   */
  it("menyemai isian dari keputusan yang sudah tersimpan", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody({
          pilihan: "2",
          pilihan_label: "Bayar / PostAudit",
          note: "Perlu diperiksa ulang",
          komentar: [
            {
              urutan: 1,
              tanggal: "2026-09-22 10:00",
              komentar: "to compilance",
            },
          ],
          catatan_investigator: "Temuan Investigator.",
          diputuskan_oleh: "ADMINCONTOH1",
          diputuskan_pada: "2026-09-22 10:00",
          tanggal_valid: null,
          tanggal_kirim_post_audit: "2026-09-22 10:00",
        }),
      ),
    );
    await renderLoaded();

    expect(
      (
        screen.getByRole("radio", {
          name: "Bayar / PostAudit",
        }) as HTMLInputElement
      ).checked,
    ).toBe(true);
    // Komentar yang tersimpan tersemai ke gridnya — inilah isian petugas.
    expect(screen.getByDisplayValue("to compilance")).toBeInTheDocument();

    // Keterangan Investigator TAMPIL, tetapi tidak sebagai isian: ia read-only di Pega.
    expect(screen.getByText("Temuan Investigator.")).toBeInTheDocument();

    // "Note Lainya" TIDAK tampil pada Bayar/PostAudit — syaratnya `pilihan==3`.
    expect(screen.queryByLabelText("Note Lainya")).not.toBeInTheDocument();
    expect(
      screen.queryByDisplayValue("Perlu diperiksa ulang"),
    ).not.toBeInTheDocument();
  });

  /**
   * Bayar/PostAudit memperingatkan bahwa ia MENERBITKAN baris baru setiap kali disimpan.
   *
   * Ia satu-satunya pilihan yang punya akibat di luar keputusan itu sendiri —
   * `SetComplianceResult` langkah 10 — dan tidak ada kunci idempotensi yang menahan
   * penekanan kedua.
   */
  it("memperingatkan hanya pada pilihan Bayar/PostAudit", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    const warning = /menerbitkan satu baris Post Audit baru/i;
    expect(screen.queryByText(warning)).not.toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("radio", { name: "Bayar / PostAudit" }),
    );
    expect(screen.getByText(warning)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("radio", { name: "Bayar / Valid" }));
    expect(screen.queryByText(warning)).not.toBeInTheDocument();
  });

  /**
   * Yang dikirim adalah pilihan yang DIPILIH beserta kedua catatan — tidak lebih.
   *
   * Kunci klaimnya TIDAK ikut di badan: ia sudah ada di jalur, dan badan yang menyebut
   * klaim berbeda dari jalurnya akan diam-diam memutuskan klaim yang salah.
   */
  it("mengirim pilihan dan kedua catatan, tanpa mengulang kunci klaim di badan", async () => {
    stubFetch((_url, init) => {
      if (init?.method === "POST") {
        return jsonResponse(200, {
          keputusan: {
            pilihan: "3",
            pilihan_label: "Lain-Lain",
            note: "Indikasi dokumen palsu",
            komentar: [
              {
                urutan: 1,
                tanggal: "2026-09-23 09:00",
                komentar: "Diteruskan ke investigasi",
              },
            ],
            catatan_investigator: "",
            diputuskan_oleh: "adminpnc",
            diputuskan_pada: "2026-09-23 09:00",
            tanggal_valid: null,
            tanggal_kirim_post_audit: null,
          },
          post_audit: null,
          keterbatasan: LIMITATION,
          portal: "ASM",
        });
      }
      return jsonResponse(200, checkerBody());
    });
    await renderLoaded();

    // Lain-Lain dipilih, bukan Fraud/Tolak: "Note Lainya" hanya muncul pada pilihan ini
    // — syarat `.ClaimData.PilihanCompliance==3` pada selnya.
    await userEvent.click(screen.getByRole("radio", { name: "Lain-Lain" }));
    await userEvent.type(
      screen.getByLabelText("Note Lainya"),
      "Indikasi dokumen palsu",
    );
    // Grid kini mulai KOSONG — "Data Tidak Ada", meniru Pega — sehingga barisnya
    // ditambahkan dulu. Versi sebelumnya memaksa satu baris kosong selalu ada.
    await userEvent.click(screen.getByRole("button", { name: "✚ Tambah" }));
    await userEvent.type(
      screen.getByLabelText("Komentar 1"),
      "Diteruskan ke investigasi",
    );
    await userEvent.click(screen.getByRole("button", { name: "Simpan Data" }));

    await waitFor(() => expect(lastSubmit()).toBeDefined());

    const submit = lastSubmit();
    expect(submit?.url).toBe(`${CHECKER_PATH}/keputusan`);
    expect(JSON.parse(String(submit?.init?.body))).toEqual({
      pilihan: "3",

      // Tombol yang ditekan adalah "Simpan Data", sehingga aksinya `simpan` — bukan
      // `kirim`. Keduanya berbeda akibat: `kirim` memindahkan klaim keluar dari antrean.
      aksi: "simpan",

      note: "Indikasi dokumen palsu",
      // Tanggal dikirim KOSONG: selnya di Pega bernilai bawaan waktu sekarang, dan
      // server yang mengisinya — bukan layar (`F-5`).
      komentar: [{ tanggal: "", komentar: "Diteruskan ke investigasi" }],
    });
  });

  /**
   * Nomor Post Audit yang terbit DITAMPILKAN, bukan ditelan.
   *
   * Tanpa nomornya, petugas tidak punya cara menunjuk pemeriksaan yang baru dibuatnya —
   * dan mencarinya di tab Post Audit berarti menebak baris mana yang barusan terbit.
   */
  it("menampilkan nomor Post Audit yang terbit", async () => {
    stubFetch((_url, init) => {
      if (init?.method === "POST") {
        return jsonResponse(200, {
          keputusan: {
            pilihan: "2",
            pilihan_label: "Bayar / PostAudit",
            note: "",
            catatan: "",
            diputuskan_oleh: "adminpnc",
            diputuskan_pada: "2026-09-23 09:00",
            tanggal_valid: null,
            tanggal_kirim_post_audit: "2026-09-23 09:00",
          },
          post_audit: {
            nomor_case: "CPL-100001",
            no_klaim: CLAIM_KEY,
            nama_tertanggung: CLAIM.nama_tertanggung,
            no_polis: CLAIM.no_polis,
            catatan: "",
            tanggal_kirim_post_audit: "2026-09-23 09:00",
            portal: "ASM",
          },
          keterbatasan: LIMITATION,
          portal: "ASM",
        });
      }
      return jsonResponse(200, checkerBody());
    });
    await renderLoaded();

    await userEvent.click(
      screen.getByRole("radio", { name: "Bayar / PostAudit" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Simpan Data" }));

    expect(await screen.findByText(/CPL-100001/)).toBeInTheDocument();
  });

  /**
   * Pelanggaran 422 ditandai DI ISIANNYA, bukan sebagai satu pesan di atas form.
   *
   * Server mengirim seluruh pelanggaran sekaligus (`P-5`), dan menampilkan satu saja
   * membuat petugas memperbaiki satu isian lalu menemukan isian berikutnya.
   */
  it("menandai isian yang dilanggar, bukan satu pesan umum", async () => {
    stubFetch((_url, init) => {
      if (init?.method === "POST") {
        return jsonResponse(422, {
          kode: "VALIDASI",
          pesan: "Isian tidak sah.",
          detail: [
            { field: "pilihan", pesan: "Pilihan Compliance belum dipilih." },
          ],
        });
      }
      return jsonResponse(200, checkerBody());
    });
    await renderLoaded();

    await userEvent.click(screen.getByRole("button", { name: "Simpan Data" }));

    expect(
      await screen.findByText("Pilihan Compliance belum dipilih."),
    ).toBeInTheDocument();
  });

  /**
   * Klaim yang sudah keluar dari antrean dijawab pesan yang MENYURUH menyegarkan, bukan
   * pesan "tidak ditemukan" yang menyuruh mencari.
   */
  it("membedakan klaim di luar antrean dari gangguan teknis", async () => {
    stubFetch(() =>
      jsonResponse(409, {
        kode: "KLAIM_TIDAK_DI_ANTREAN",
        pesan: "Klaim tidak sedang menunggu di antrean Compliance.",
      }),
    );
    renderForm();

    expect(
      await screen.findByText("Klaim tidak ada di antrean Compliance"),
    ).toBeInTheDocument();
  });

  /**
   * Grid komentar dapat BERTAMBAH baris, dan seluruh barisnya terkirim.
   *
   * Inilah satu-satunya tempat petugas Compliance menulis — `.ClaimData.ComplianceList`,
   * bukan `ComplianceRemark` yang read-only di Pega. Tanpa tombol tambah, grid itu hanya
   * menampung satu komentar, sedangkan di Pega ia dapat banyak.
   */
  it("menambah baris komentar dan mengirim seluruhnya", async () => {
    stubFetch((_url, init) => {
      if (init?.method === "POST") {
        return jsonResponse(200, {
          keputusan: {
            pilihan: "1",
            pilihan_label: "Bayar / Valid",
            note: "",
            komentar: [],
            catatan_investigator: "",
            diputuskan_oleh: "adminpnc",
            diputuskan_pada: "2026-09-23 09:00",
            tanggal_valid: "2026-09-23 09:00",
            tanggal_kirim_post_audit: null,
          },
          post_audit: null,
          keterbatasan: LIMITATION,
          portal: "ASM",
        });
      }
      return jsonResponse(200, checkerBody());
    });
    await renderLoaded();

    await userEvent.click(screen.getByRole("radio", { name: "Bayar / Valid" }));
    await userEvent.click(screen.getByRole("button", { name: "✚ Tambah" }));
    await userEvent.type(screen.getByLabelText("Komentar 1"), "Baris pertama");

    await userEvent.click(screen.getByRole("button", { name: "✚ Tambah" }));
    await userEvent.type(screen.getByLabelText("Komentar 2"), "Baris kedua");

    await userEvent.click(screen.getByRole("button", { name: "Simpan Data" }));
    await waitFor(() => expect(lastSubmit()).toBeDefined());

    expect(JSON.parse(String(lastSubmit()?.init?.body))).toEqual({
      pilihan: "1",
      aksi: "simpan",

      // Note tetap dikirim meski tidak tampil: syaratnya di Pega sisi klien, sehingga
      // isian yang tersembunyi tetap ikut tersimpan saat Obj-Save.
      note: "",

      komentar: [
        { tanggal: "", komentar: "Baris pertama" },
        { tanggal: "", komentar: "Baris kedua" },
      ],
    });
  });

  /**
   * Blok "Hasil Investigasi" TIDAK digambar ketika servernya tidak menyalakannya.
   *
   * Syaratnya `IsPA`, dan klaim contoh berkas ini bukan PA.
   */
  it("tidak menggambar blok Hasil Investigasi pada lini selain PA", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    expect(screen.queryByText("Hasil Investigasi")).not.toBeInTheDocument();
  });

  /** Keempat kolom dan isinya tergambar pada lini PA. */
  it("menggambar keempat kolom Hasil Investigasi pada lini PA", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, {
          tampilkan_hasil_investigasi: true,
          hasil_investigasi: [
            {
              tanggal_investigasi: "2026-09-30 11:00",
              nama_peserta: "Peserta Contoh",
              lokasi_objek: "Cabang Contoh",
              status: "Final Report",
            },
          ],
          hasil_investigasi_gagal_dibaca: false,
        }),
      ),
    );
    await renderLoaded();

    // Isinya di balik TAB — sama seperti Pega. Tab Compliance yang terbuka lebih dulu,
    // sehingga tabelnya belum tergambar sampai tabnya ditekan.
    expect(screen.queryByText("Tanggal Investigasi")).not.toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("tab", { name: "Hasil Investigasi" }),
    );

    // Judul keempat kolom, apa adanya seperti di Pega.
    expect(screen.getByText("Tanggal Investigasi")).toBeInTheDocument();
    expect(screen.getByText("Nama Peserta")).toBeInTheDocument();
    expect(screen.getByText("Lokasi Objek")).toBeInTheDocument();

    // Isi barisnya benar-benar sampai ke layar, bukan hanya rangkanya.
    expect(screen.getByText("Peserta Contoh")).toBeInTheDocument();
    expect(screen.getByText("Cabang Contoh")).toBeInTheDocument();
    expect(screen.getByText("Final Report")).toBeInTheDocument();
  });

  /**
   * Klaim PA yang belum pernah disurvei tetap menggambar bloknya — tabel kosong.
   *
   * Inilah sebab blok ini digerbangi `tampilkan_hasil_investigasi`, bukan oleh panjang
   * senarainya: keduanya nol baris, tetapi hanya yang ini boleh tergambar.
   */
  it("menggambar blok kosong untuk klaim PA yang belum disurvei", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, {
          tampilkan_hasil_investigasi: true,
          hasil_investigasi: [],
          hasil_investigasi_gagal_dibaca: false,
        }),
      ),
    );
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("tab", { name: "Hasil Investigasi" }),
    );

    // Judul kolomnya ada — membuktikan TABELNYA tergambar, bukan sekadar tabnya.
    expect(screen.getByText("Nama Peserta")).toBeInTheDocument();
    expect(screen.getByText("Data Tidak Ada")).toBeInTheDocument();
  });

  /**
   * Kegagalan baca DINYATAKAN, tidak menyaru sebagai tabel kosong.
   *
   * Tanpa ini, petugas memutuskan klaim PA sambil mengira tidak ada hasil investigasi —
   * padahal ada, dan hanya sedang tidak terbaca.
   */
  it("menyatakan ketika hasil investigasi gagal dibaca", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, {
          tampilkan_hasil_investigasi: true,
          hasil_investigasi: [],
          hasil_investigasi_gagal_dibaca: true,
        }),
      ),
    );
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("tab", { name: "Hasil Investigasi" }),
    );

    expect(
      screen.getByText("Hasil investigasi tidak dapat dibaca"),
    ).toBeInTheDocument();
  });

  /**
   * Bilah tab mengikuti Pega: Compliance selalu ada, Hasil Investigasi hanya pada PA.
   *
   * `pyNewTabGroup = true` pada layout S1 `Section/ComplianceChecker-Section.xml`.
   * Pembacaan pertama menyimpulkan keempat layout itu blok bertumpuk — salah, dan layar
   * Pega yang berjalan membuktikannya sebagai tab.
   */
  it("menggambar tab Hasil Investigasi hanya pada lini PA", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    const { unmount } = await renderLoaded();

    expect(screen.getByRole("tab", { name: "Compliance" })).toBeInTheDocument();
    expect(
      screen.queryByRole("tab", { name: "Hasil Investigasi" }),
    ).not.toBeInTheDocument();

    unmount();

    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, {
          tampilkan_hasil_investigasi: true,
          hasil_investigasi: [],
          hasil_investigasi_gagal_dibaca: false,
        }),
      ),
    );
    await renderLoaded();

    expect(
      screen.getByRole("tab", { name: "Hasil Investigasi" }),
    ).toBeInTheDocument();
  });

  /**
   * Berpindah tab TIDAK menyimpan keputusan.
   *
   * Tombol tanpa `type` di dalam `<form>` bertipe `submit`. Pada form ini akibatnya bukan
   * gangguan kecil: menyimpan memindahkan klaim keluar dari antrean.
   */
  it("berpindah tab tidak mengirim permintaan simpan", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, {
          tampilkan_hasil_investigasi: true,
          hasil_investigasi: [],
          hasil_investigasi_gagal_dibaca: false,
        }),
      ),
    );
    await renderLoaded();

    const sebelum = calls.length;
    await userEvent.click(
      screen.getByRole("tab", { name: "Hasil Investigasi" }),
    );
    await userEvent.click(screen.getByRole("tab", { name: "Compliance" }));

    expect(calls.length).toBe(sebelum);
  });

  /**
   * "Kirim ke Analyst" mengirim `aksi: "kirim"`, bukan `"simpan"`.
   *
   * Keduanya berakibat sangat berbeda: `simpan` meninggalkan klaim di antrean Compliance,
   * `kirim` menjalankan `SetComplianceResult` yang memindahkannya keluar — dan sesudah
   * itu form ini tidak dapat dibuka lagi.
   */
  it("Kirim ke Analyst mengirim aksi kirim", async () => {
    stubFetch((url, init) =>
      String(url).endsWith("/keputusan") && init?.method === "POST"
        ? jsonResponse(200, SIMPAN_BERHASIL)
        : jsonResponse(
            200,
            checkerBody(null, { ...ACTIONS, kirim_ke_analyst: true }),
          ),
    );
    await renderLoaded();

    await userEvent.click(screen.getByRole("radio", { name: "Bayar / Valid" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Kirim ke Analyst" }),
    );

    await waitFor(() => expect(lastSubmit()).toBeDefined());
    expect(JSON.parse(String(lastSubmit()?.init?.body)).aksi).toBe("kirim");
  });

  /** "Kirim ke PIC Teknik" mengirim nilai yang SAMA — tujuan ditentukan lini bisnis. */
  it("Kirim ke PIC Teknik mengirim aksi kirim yang sama", async () => {
    stubFetch((url, init) =>
      String(url).endsWith("/keputusan") && init?.method === "POST"
        ? jsonResponse(200, SIMPAN_BERHASIL)
        : jsonResponse(
            200,
            checkerBody(null, { ...ACTIONS, kirim_ke_pic_teknik: true }),
          ),
    );
    await renderLoaded();

    await userEvent.click(screen.getByRole("radio", { name: "Bayar / Valid" }));
    await userEvent.click(
      screen.getByRole("button", { name: "Kirim ke PIC Teknik" }),
    );

    await waitFor(() => expect(lastSubmit()).toBeDefined());
    expect(JSON.parse(String(lastSubmit()?.init?.body)).aksi).toBe("kirim");
  });
  /*
    Lima uji grid dokumen DICABUT 2026-10-08 bersama gridnya:

      tidak menggambar grid dokumen ketika tidak ada berkas
      menggambar ketiga kolom dokumen beserta tombol Lihat
      menekan Lihat meminta tautan baru lalu membukanya
      menyatakan ketika dokumen gagal dibaca
      Hapus dibatalkan / Hapus yang dikonfirmasi

    Kelimanya mengunci perilaku yang TIDAK ada di Pega: form di sana tidak pernah memuat
    lampiran dari basis data saat dibuka. Lihat catatan pada halaman dan pada
    usecase.CheckerOpened.

    Uji "Unggah Dokumen" DIPERTAHANKAN di bawah — tombolnya memang ada di Pega.
  */

  /** Satu baris dokumen contoh, dipakai uji unggah dan hapus. */
  const DOKUMEN = [
    {
      id: "4411",
      nama_file: "Surat Keterangan.pdf",
      tipe_file: "application/pdf",
      kategori: "Dokumen Pendukung",
      id_penyimpanan: "IMG-7781",
      tanggal_unggah: null,
    },
  ];

  /**
   * "Unggah Dokumen" mengirim berkas sebagai `FormData`, bukan JSON ber-Base64.
   *
   * Base64 membengkakkan badan sepertiga, dan pembengkakan itu sudah ditanggung sekali
   * lagi di server saat adapter menyiapkannya untuk layanan penyimpanan.
   */
  it("Unggah Dokumen mengirim berkas sebagai FormData", async () => {
    stubFetch((url, init) => {
      if (String(url).endsWith("/dokumen") && init?.method === "POST") {
        return jsonResponse(201, DOKUMEN[0]);
      }
      return jsonResponse(200, checkerBody());
    });

    await renderLoaded();

    const berkas = new File(["isi"], "Surat Keterangan.pdf", {
      type: "application/pdf",
    });
    const pemilih =
      document.querySelector<HTMLInputElement>('input[type="file"]');
    expect(pemilih).not.toBeNull();

    await userEvent.upload(pemilih as HTMLInputElement, berkas);

    await waitFor(() =>
      expect(calls.some((c) => c.url.endsWith("/dokumen"))).toBe(true),
    );

    const kirim = calls.find(
      (c) => c.url.endsWith("/dokumen") && c.init?.method === "POST",
    );
    expect(kirim?.init?.body).toBeInstanceOf(FormData);
    expect((kirim?.init?.body as FormData).get("berkas")).toBe(berkas);
  });

  // Kedua uji "Hapus" dicabut bersama grid dokumennya — tombolnya hidup di grid itu.
  // Endpoint DELETE-nya tetap ada dan tetap teruji di backend; yang hilang hanya
  // permukaan layarnya.
});

/**
 * Form "Download Dokumen Reject".
 *
 * Tombolnya tidak lagi mati: ia membuka dialog berisi delapan isian dan grid Alasan,
 * lalu Generate PDF melampirkan suratnya ke klaim — bukan mengunduhnya.
 */
describe("ComplianceCheckerPage — Surat Penolakan", () => {
  const SURAT_TERBIT = {
    dokumen: {
      id: "9001",
      nama_file: "Surat_Penolakan_dan_Penarikan_Dana.pdf",
      tipe_file: "pdf",
      kategori: "RejectLetter",
      id_penyimpanan: "IMG-9001",
      tanggal_unggah: "2026-10-08 09:00",
    },
    mengganti_surat_sebelumnya: false,
    portal: "ASM",
  };

  function suratTerbit(url: string, init?: RequestInit) {
    if (url.endsWith("/surat-penolakan") && init?.method === "POST") {
      return jsonResponse(200, SURAT_TERBIT);
    }
    return jsonResponse(200, checkerBody());
  }

  function badanSurat(): Record<string, unknown> {
    const kirim = [...calls]
      .reverse()
      .find((c) => c.url.endsWith("/surat-penolakan") && c.init?.method === "POST");
    return JSON.parse(String(kirim?.init?.body)) as Record<string, unknown>;
  }

  /** Tombolnya membuka dialog, bukan diam. */
  it("tombol Download Dokumen Reject membuka formnya", async () => {
    stubFetch(suratTerbit);
    await renderLoaded();

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    );

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Generate PDF" }),
    ).toBeInTheDocument();
  });

  /**
   * Tiga isian terbawa dari klaim; yang keempat terbuka KOSONG.
   *
   * "Tanggal Keluar Rawat Inap" tidak punya kolom di `T_CLAIM_PNC`, sehingga tidak ada
   * yang dapat dipra-isi — dan `tanggal_kejadian` yang `null` tidak boleh tergambar
   * sebagai teks "null".
   */
  it("mengisi tiga isian dari klaim dan membiarkan sisanya kosong", async () => {
    stubFetch(suratTerbit);
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    );

    expect(screen.getByLabelText("Nama Pasien / No.Reg")).toHaveValue(
      "PESERTA CONTOH",
    );
    expect(screen.getByLabelText("Tempat Kejadian / Perawatan")).toHaveValue(
      "RS CONTOH",
    );
    expect(
      screen.getByLabelText("Tanggal Kejadian / Tanggal Masuk Rawat Inap"),
    ).toHaveValue("");
    expect(screen.getByLabelText("Tanggal Keluar Rawat Inap")).toHaveValue("");
  });

  /** Generate PDF mengirim seluruh isian, dan menutup dialognya saat berhasil. */
  it("Generate PDF mengirim isian lalu menutup dialog", async () => {
    stubFetch(suratTerbit);
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    );

    await userEvent.type(screen.getByLabelText("Up"), "Bagian Umum");
    await userEvent.type(screen.getByLabelText("Alasan 1"), "Dokumen tidak lengkap");
    await userEvent.click(screen.getByRole("button", { name: "Generate PDF" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );

    const badan = badanSurat();
    expect(badan["up"]).toBe("Bagian Umum");
    expect(badan["alasan"]).toEqual(["Dokumen tidak lengkap"]);
  });

  /** Tambah Alasan menambah satu baris, dan barisnya ikut terkirim. */
  it("Tambah Alasan menambah baris yang ikut terkirim", async () => {
    stubFetch(suratTerbit);
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    );

    await userEvent.type(screen.getByLabelText("Alasan 1"), "Pertama");
    await userEvent.click(screen.getByRole("button", { name: "Tambah Alasan" }));
    await userEvent.type(screen.getByLabelText("Alasan 2"), "Kedua");
    await userEvent.click(screen.getByRole("button", { name: "Generate PDF" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(badanSurat()["alasan"]).toEqual(["Pertama", "Kedua"]);
  });

  /**
   * Gagal -> dialog TETAP terbuka beserta isiannya.
   *
   * Isian form ini tidak tersimpan di mana pun, sehingga menutup dialog pada kegagalan
   * berarti petugas mengetik ulang delapan isian dan seluruh baris alasan.
   */
  it("kegagalan tidak menutup dialog dan tidak menghapus isian", async () => {
    stubFetch((url, init) => {
      if (url.endsWith("/surat-penolakan") && init?.method === "POST") {
        return jsonResponse(500, {
          kode: "galat_internal",
          pesan: "Terjadi kesalahan pada sistem.",
        });
      }
      return jsonResponse(200, checkerBody());
    });
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    );

    await userEvent.type(screen.getByLabelText("Up"), "Bagian Umum");
    await userEvent.click(screen.getByRole("button", { name: "Generate PDF" }));

    await screen.findByText("Surat penolakan gagal dibuat");
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByLabelText("Up")).toHaveValue("Bagian Umum");
  });

  /** Setelah berhasil, layar menyebut suratnya DILAMPIRKAN — bukan diunduh. */
  it("mengabarkan bahwa suratnya tersimpan sebagai dokumen klaim", async () => {
    stubFetch(suratTerbit);
    await renderLoaded();
    await userEvent.click(
      screen.getByRole("button", { name: "Download Dokumen Reject" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Generate PDF" }));

    await screen.findByText("Surat penolakan dibuat");
    expect(
      screen.getByText(/tersimpan sebagai dokumen klaim/),
    ).toBeInTheDocument();
  });
});

/**
 * Tab **Dokumen** — layout S3, bersyarat `IsTravel`.
 *
 * Ditemukan 2026-10-08 ketika Work Owner membuka layar Pega untuk klaim Travel. Uji di
 * sini mengunci dua hal sekaligus: tabnya hanya ada pada Travel, dan isinya daftar
 * periksa kelengkapan — bukan daftar berkas.
 */
describe("ComplianceCheckerPage — tab Dokumen", () => {
  const DAFTAR = [
    {
      id_kategori: "10064",
      kategori: "Akte/Surat Keterangan Kematian (Copy)",
      wajib_unggah: "Ya",
      minimal_unggah: 1,
      total_sudah_diunggah: 0,
    },
    {
      id_kategori: "10065",
      kategori: "Foto Penguburan (jika ada)",
      // Sengaja KOSONG — itu yang benar-benar terlihat di layar Pega.
      wajib_unggah: "",
      minimal_unggah: 1,
      total_sudah_diunggah: 2,
    },
  ];

  const TRAVEL = {
    tampilkan_daftar_dokumen: true,
    daftar_dokumen: DAFTAR,
    daftar_dokumen_gagal_dibaca: false,
  };

  /** Tabnya TIDAK digambar di luar Travel — syaratnya lini, bukan ada-tidaknya data. */
  it("tidak menggambar tab Dokumen di luar lini Travel", async () => {
    stubFetch(() => jsonResponse(200, checkerBody()));
    await renderLoaded();

    expect(
      screen.queryByRole("tab", { name: "Dokumen" }),
    ).not.toBeInTheDocument();
  });

  /** Pada Travel tabnya ada, dan isinya keempat kolom daftar periksa. */
  it("menggambar keempat kolom daftar periksa pada klaim Travel", async () => {
    stubFetch(() =>
      jsonResponse(200, checkerBody(null, ACTIONS, undefined, TRAVEL)),
    );
    await renderLoaded();

    await userEvent.click(screen.getByRole("tab", { name: "Dokumen" }));

    expect(screen.getByText("Kategori")).toBeInTheDocument();
    expect(screen.getByText("Wajib Unggah")).toBeInTheDocument();
    expect(screen.getByText("Minimal Unggah")).toBeInTheDocument();
    expect(screen.getByText("Total Sudah Diunggah")).toBeInTheDocument();

    expect(
      screen.getByText("Akte/Surat Keterangan Kematian (Copy)"),
    ).toBeInTheDocument();
  });

  /**
   * Kategori yang BELUM punya berkas tetap tampil.
   *
   * Inilah yang membedakannya dari daftar berkas: baris bernilai 0 justru yang paling
   * berguna, karena ia yang menunjukkan apa yang masih kurang.
   */
  it("menampilkan kategori yang belum punya berkas", async () => {
    stubFetch(() =>
      jsonResponse(200, checkerBody(null, ACTIONS, undefined, TRAVEL)),
    );
    await renderLoaded();
    await userEvent.click(screen.getByRole("tab", { name: "Dokumen" }));

    const baris = screen
      .getByText("Akte/Surat Keterangan Kematian (Copy)")
      .closest("tr");
    expect(baris).not.toBeNull();
    expect(baris).toHaveTextContent("0");
  });

  /** Travel tanpa master menggambar tabel kosong, bukan galat. */
  it("menggambar Data Tidak Ada ketika masternya kosong", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, undefined, {
          tampilkan_daftar_dokumen: true,
          daftar_dokumen: [],
          daftar_dokumen_gagal_dibaca: false,
        }),
      ),
    );
    await renderLoaded();
    await userEvent.click(screen.getByRole("tab", { name: "Dokumen" }));

    expect(screen.getByText("Data Tidak Ada")).toBeInTheDocument();
  });

  /**
   * Gagal baca DINYATAKAN — tabel kosong tidak boleh terbaca sebagai "tidak ada yang
   * wajib", karena petugas dapat meneruskan klaim yang dokumennya belum lengkap.
   */
  it("menyatakan ketika daftar dokumen gagal dibaca", async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        checkerBody(null, ACTIONS, undefined, {
          tampilkan_daftar_dokumen: true,
          daftar_dokumen: [],
          daftar_dokumen_gagal_dibaca: true,
        }),
      ),
    );
    await renderLoaded();
    await userEvent.click(screen.getByRole("tab", { name: "Dokumen" }));

    expect(
      screen.getByText("Daftar dokumen tidak dapat dibaca"),
    ).toBeInTheDocument();
  });
});

/**
 * Ketiga tombol pada baris daftar periksa — Unggah, Lihat, Ubah Kategori.
 *
 * Yang diuji di sini BUKAN tampilannya melainkan **ke mana permintaannya pergi**: tombol
 * yang mengirim ke kategori yang salah tidak terlihat salah di layar, dan baru ketahuan
 * ketika berkas nasabah mendarat di kategori orang lain.
 */
describe("ComplianceCheckerPage — tombol baris Dokumen", () => {
  const BARIS = {
    id_kategori: "10064",
    kategori: "Akte/Surat Keterangan Kematian (Copy)",
    wajib_unggah: "Ya",
    minimal_unggah: 1,
    total_sudah_diunggah: 1,
  };

  const TRAVEL = {
    tampilkan_daftar_dokumen: true,
    daftar_dokumen: [BARIS],
    daftar_dokumen_gagal_dibaca: false,
  };

  const DOKUMEN_KATEGORI = {
    dokumen: [
      {
        id: "4411",
        nama_file: "akte.pdf",
        tipe_file: "pdf",
        kategori: "10064",
        id_penyimpanan: "IMG-1",
        tanggal_unggah: null,
      },
    ],
    portal: "ASM",
  };

  function jawab(url: string, init?: RequestInit) {
    const u = String(url);
    if (u.includes("/dokumen?kategori=") || u.endsWith("/dokumen")) {
      if (init?.method === undefined || init.method === "GET") {
        return jsonResponse(200, DOKUMEN_KATEGORI);
      }
    }
    if (u.includes("/kategori") && init?.method === "POST") {
      return new Response(null, { status: 204 });
    }
    return jsonResponse(200, checkerBody(null, ACTIONS, undefined, TRAVEL));
  }

  async function bukaTabDokumen() {
    await renderLoaded();
    await userEvent.click(screen.getByRole("tab", { name: "Dokumen" }));
  }

  /** Unggah dari baris mengirim ke endpoint dokumen klaim ini. */
  it("tombol Unggah pada baris mengirim berkas", async () => {
    stubFetch(jawab);
    await bukaTabDokumen();

    const berkas = new File(["isi"], "akte.pdf", { type: "application/pdf" });
    const pemilih = document.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement;
    await userEvent.upload(pemilih, berkas);

    await waitFor(() =>
      expect(
        calls.some(
          (c) => c.url.endsWith("/dokumen") && c.init?.method === "POST",
        ),
      ).toBe(true),
    );
  });

  /**
   * "Lihat" meminta dokumen pada KATEGORI BARIS ITU, bukan seluruh dokumen klaim.
   *
   * Inilah yang membedakannya dari daftar dokumen biasa — dan kalau penyaringnya hilang,
   * petugas melihat dokumen kategori lain seolah milik kategori ini.
   */
  it("Lihat meminta dokumen pada kategori barisnya", async () => {
    stubFetch(jawab);
    await bukaTabDokumen();

    await userEvent.click(
      screen.getByRole("button", {
        name: "Lihat dokumen Akte/Surat Keterangan Kematian (Copy)",
      }),
    );

    await waitFor(() =>
      expect(calls.some((c) => c.url.includes("kategori=10064"))).toBe(true),
    );
    expect(await screen.findByText("akte.pdf")).toBeInTheDocument();
  });

  /**
   * "Ubah Kategori" meminta SELURUH dokumen klaim, bukan kategori barisnya.
   *
   * Dokumen yang hendak dipindahkan menurut definisi ada di kategori LAIN — menyaringnya
   * ke kategori baris akan membuat daftarnya selalu kosong kecuali yang sudah di sana.
   */
  it("Ubah Kategori meminta seluruh dokumen klaim", async () => {
    stubFetch(jawab);
    await bukaTabDokumen();

    await userEvent.click(
      screen.getByRole("button", {
        name: "Ubah Kategori Dok Akte/Surat Keterangan Kematian (Copy)",
      }),
    );

    await waitFor(() =>
      expect(
        calls.some(
          (c) =>
            c.url.endsWith("/dokumen") &&
            (c.init?.method === undefined || c.init.method === "GET"),
        ),
      ).toBe(true),
    );
  });

  /** Dokumen yang SUDAH di kategori ini tidak dapat dipindahkan ke sini. */
  it("dokumen yang sudah di kategori ini tombolnya mati", async () => {
    stubFetch(jawab);
    await bukaTabDokumen();

    await userEvent.click(
      screen.getByRole("button", {
        name: "Ubah Kategori Dok Akte/Surat Keterangan Kematian (Copy)",
      }),
    );

    const tombol = await screen.findByRole("button", {
      name: "Pindahkan akte.pdf ke Akte/Surat Keterangan Kematian (Copy)",
    });
    expect(tombol).toBeDisabled();
    expect(tombol).toHaveTextContent("Sudah di sini");
  });
});
