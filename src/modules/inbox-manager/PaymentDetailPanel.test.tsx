import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useSelectedPortal } from "@/app/portal";
import { useSession } from "@/app/session";

import { QueuePanel } from "./QueuePanel";
import type { PaymentDetail, Tab } from "./types";

/** Panel detail Payment Klaim Akseptasi — DETAILS + form, seperti Pega. Data KARANGAN (`D-69`). */

const DETAIL_PATH = "/api/inbox-manager/payment-akseptasi/AKS-001";

const TAB_PAYMENT: Tab = {
  kode: "12",
  nama: "Payment Klaim Akseptasi",
  keterangan: "Nomor akseptasi yang menunggu persetujuan atasan.",
  jenis: "antrean",
  kolom: [
    { kunci: "no_klaim", judul: "No Klaim" },
    { kunci: "no_akseptasi", judul: "No Akseptasi" },
  ],
  keputusan: {
    dapat_diputuskan: false,
    dapat_massal: false,
    alasan_setuju_ditahan: "Alasan penahanan dari server.",
    alasan_wajib_saat_menolak: true,
    label_alasan: "Remark Approval",
  },
  punya_penyaring_periode: false,
  punya_detail_akseptasi: true,
};

const DETAIL: PaymentDetail = {
  no_akseptasi: "AKS-001",
  no_klaim: "KLM-001",
  pic: "PIC-CONTOH",
  nama_tertanggung: "Tertanggung Contoh",
  penerima_klaim: "Penerima Contoh",
  kode_tipe_pembayaran: "2",
  tipe_pembayaran: "Interim",
  gross_value: "1234567.5",
  nilai_klaim: "1000000",
  mata_uang: "IDR",
  share_asm: "80",
  remark_akseptasi: "Catatan PIC contoh",
  adjustment_ditemukan: true,
  pilihan_status: [
    { nilai: "0", label: "Sedang Proses Komite" },
    { nilai: "1", label: "Setuju" },
    { nilai: "2", label: "Tidak Setuju" },
  ],
  riwayat: [
    {
      no_klaim: "KLM-001",
      no_akseptasi: "AKS-000",
      tanggal: "2026-10-01T02:00:00Z",
      note_manager: "Catatan atasan contoh",
      note_pic: "Catatan PIC lama",
      kode_status: "2",
      status: "Tidak Setuju",
    },
  ],
  dokumen: [
    {
      kunci: "1",
      nama: "lampiran-contoh.pdf",
      jenis: "application/pdf",
      catatan: "",
      tanggal_input: null,
      tertaut: true,
      dapat_dibuka: true,
    },
  ],
};

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
    return Promise.resolve(answer(url, init));
  });
}

function renderQueue() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <QueuePanel
        tab={TAB_PAYMENT}
        rows={[
          {
            kunci: "AKS-001",
            sel: { no_klaim: "KLM-001", no_akseptasi: "AKS-001" },
          },
        ]}
        isLoading={false}
        page={1}
        onPageChange={() => {}}
        onDecide={() => {}}
        isDeciding={false}
        onExport={() => {}}
        isExporting={false}
        onRefresh={() => {}}
      />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  calls = [];
  useSession.setState({
    token: "token-uji",
    user: {
      identitas: "90000058",
      nama: "Contoh Penyelia",
      jenis: "KARYAWAN",
      login: "penyelia.manager",
      email: "contoh.penyelia@example.invalid",
      perusahaan: "ASM",
    },
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  });
  useSelectedPortal.getState().select("ASM");
});

afterEach(() => {
  vi.unstubAllGlobals();
  useSession.getState().clear();
  useSelectedPortal.getState().clear();
});

describe("antrean Payment Klaim Akseptasi", () => {
  it("tanpa kotak pilih; setiap baris punya DETAILS, dan ada Refresh", () => {
    stubFetch(() => jsonResponse(200, DETAIL));
    renderQueue();

    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "DETAILS AKS-001" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Refresh/ })).toBeInTheDocument();
    expect(
      screen.queryByText("Detail Penerima Akseptasi"),
    ).not.toBeInTheDocument();
  });

  it("DETAILS membuka detail, riwayat, dan dokumen dengan label Pega", async () => {
    stubFetch(() => jsonResponse(200, DETAIL));
    renderQueue();

    await userEvent.click(
      screen.getByRole("button", { name: "DETAILS AKS-001" }),
    );

    const detail = await screen.findByRole("region", {
      name: "Detail Penerima Akseptasi",
    });
    expect(within(detail).getByText("Tertanggung Contoh")).toBeInTheDocument();
    expect(within(detail).getByText("Interim")).toBeInTheDocument();
    // Nilai uang dari teks presisi penuh, tanpa pembulatan.
    expect(within(detail).getByText("1.234.567,5")).toBeInTheDocument();
    expect(within(detail).getByText("Share ASM (%)")).toBeInTheDocument();

    expect(
      screen.getByRole("region", { name: "Persetujuan Akseptasi" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("History Payment Akseptasi Manager"),
    ).toBeInTheDocument();
    expect(screen.getByText("Catatan atasan contoh")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "lampiran-contoh.pdf" }),
    ).toBeInTheDocument();

    // Setuju ditahan server: pilihannya nonaktif, alasannya tertulis.
    expect(screen.getByRole("option", { name: "Setuju" })).toBeDisabled();
    expect(
      screen.getByText("Alasan penahanan dari server."),
    ).toBeInTheDocument();
  });

  it("Tidak Setuju menuntut Remark Approval sebelum SIMPAN", async () => {
    stubFetch((url) =>
      url.endsWith("/simpan")
        ? jsonResponse(200, {
            berubah: 1,
            penanda_adjustment: true,
            surel_terkirim: true,
          })
        : jsonResponse(200, DETAIL),
    );
    renderQueue();

    await userEvent.click(
      screen.getByRole("button", { name: "DETAILS AKS-001" }),
    );
    const form = await screen.findByRole("region", {
      name: "Persetujuan Akseptasi",
    });

    const save = within(form).getByRole("button", { name: "SIMPAN" });
    expect(save).toBeDisabled();

    await userEvent.selectOptions(
      within(form).getByLabelText("Status Approval"),
      "2",
    );
    expect(save).toBeDisabled();

    await userEvent.type(
      within(form).getByLabelText("Remark Approval"),
      "nilai belum sesuai",
    );
    expect(save).toBeEnabled();

    await userEvent.click(save);

    await waitFor(() => {
      const post = calls.find((call) => call.url === `${DETAIL_PATH}/simpan`);
      expect(post?.init?.method).toBe("POST");
      expect(JSON.parse(String(post?.init?.body))).toEqual({
        status: "2",
        catatan: "nilai belum sesuai",
      });
    });

    // Nomornya keluar dari antrean: panelnya ditutup.
    await waitFor(() =>
      expect(
        screen.queryByRole("region", { name: "Persetujuan Akseptasi" }),
      ).not.toBeInTheDocument(),
    );
  });

  it("Dokumen tertaut meminta alamat segar ke server lalu membukanya", async () => {
    const open = vi.fn();
    vi.stubGlobal("open", open);
    stubFetch((url) =>
      url.endsWith("/tautan")
        ? jsonResponse(200, {
            url: "https://penyimpanan.example.invalid/berkas",
          })
        : jsonResponse(200, DETAIL),
    );
    renderQueue();

    await userEvent.click(
      screen.getByRole("button", { name: "DETAILS AKS-001" }),
    );
    await userEvent.click(
      await screen.findByRole("button", { name: "lampiran-contoh.pdf" }),
    );

    await waitFor(() =>
      expect(open).toHaveBeenCalledWith(
        "https://penyimpanan.example.invalid/berkas",
        "_blank",
        "noopener,noreferrer",
      ),
    );
    expect(
      calls.some((call) => call.url === `${DETAIL_PATH}/dokumen/1/tautan`),
    ).toBe(true);
  });
});
