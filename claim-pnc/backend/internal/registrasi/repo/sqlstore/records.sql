-- Kueri tab pendamping tahap Input Estimasi: Survey, Unggah Dokumen, Progress Claim &
-- Komunikasi. SELURUHNYA membaca; tabel-tabel ini milik sistem lama dan tidak ditulis
-- modul registrasi (`P-1`).
--
-- Kunci klaim pada tabel warisan tidak seragam. Setiap kueri karena itu menerima TIGA
-- bind kunci berurutan: nomor klaim, pengenal internal, dan nomor berawalan kelas Pega.
-- Bentuknya disusun di satu tempat, `Claim.Keys` pada paket registrasi.

-- name: survey_daftar
--
-- Grid hasil survey `Section/ViewHasilSurvey-Section.xml`: SurveyDate, SurveyorName,
-- ObjectName, ObjectLocation, SurveyStatus. Kolomnya dibaca dari T_SURVEYORLIST, tabel
-- yang diisi `Database/INSERT_SURVEYORLIST.prc`.
--
-- Bind: :1 :2 :3 kunci klaim.
SELECT s.CASEID,
       s.SURVEYTYPE,
       s.SURVEYOR_NAME,
       s.SURVEYDATE,
       s.LOCATION_SURVEY,
       s.OBJECT_NAME,
       s.LOCATION_OBJECT,
       s.INDEX_SURVEY,
       s.STS_SURVEY,
       s.KETERANGAN,
       s.TGLINPUT
  FROM POOLDATA.T_SURVEYORLIST s
 WHERE s.PNCCASEID IN (:1, :2, :3)
 ORDER BY s.SURVEYDATE, s.CASEID, s.INDEX_SURVEY

-- name: dokumen_jenis_daftar
--
-- Jenis dokumen satu lini bisnis — gabungan `BrowseRegister_upload`,
-- `BrowseSurvey_upload`, `BrowseCommitee_upload`, `BrowsePayment_upload`, dan
-- `BrowseCollectingDoc_upload`, yang hanya berbeda pada nama jenis induknya.
--
-- # Nama jenis induk dibaca dari JSON_DATA, bukan dari view
--
-- Kueri lama menyaring lewat `v_lst_doc_type.type_document`. Pada basis data yang
-- diperiksa (2026-09-27) kolom itu KOSONG di keenam baris view, sementara nilainya
-- tersimpan di LST_DOC_TYPE.JSON_DATA. COALESCE memakai kolomnya bila kelak terisi,
-- dan JSON_DATA selama belum.
--
-- STS_WAJIB dikembalikan mentah; aturan wajibnya dihitung di Go karena bergantung pada
-- kelompok item klaim (lihat `registrasi.DocumentChecklist`).
--
-- Bind: :1 kode bisnis polis.
SELECT COALESCE(t.TYPE_DOCUMENT, JSON_VALUE(t.JSON_DATA, '$.TYPE_DOCUMENT')) AS CATEGORY,
       b.DOCUMENT_TYPE_ID,
       b.DOC_TYPE_DT_ID,
       b.DETAIL_DOKUMEN,
       b.STS_WAJIB,
       b.OBJECT_DOC_ID,
       b.MIN_DOC
  FROM POOLDATA.LST_TYPE_DOC_BUSINESS b
       INNER JOIN POOLDATA.LST_DOC_TYPE t
               ON t.ID = b.DOCUMENT_TYPE_ID
 WHERE b.BUSINESSID = :1
   AND b.DETAIL_DOKUMEN <> '-'
 ORDER BY b.DETAIL_DOKUMEN

-- name: dokumen_coverage_daftar
--
-- Coverage yang mewajibkan sebuah jenis dokumen pada lini Personal Accident —
-- subkueri COVERAGE_DOC_BUSINESS pada `BrowseRegisterCvg`.
--
-- Bind: :1 kode bisnis polis.
SELECT c.ID, c.COVERAGEID
  FROM POOLDATA.COVERAGE_DOC_BUSINESS c
 WHERE c.BUSINESSID = :1

-- name: lampiran_daftar
--
-- Berkas yang sudah diunggah untuk klaim. Isi berkasnya (ATTACHFILE) sengaja tidak
-- dibaca: ia BLOB, dan kueri lama mengubahnya ke base64 lewat fungsi basis data
-- `pooldata.base64encode` — yang tidak dibawa (`D-02`).
--
-- Bind: :1 :2 :3 kunci klaim.
SELECT a.DATAID,
       a.ATTACHNAME,
       a.ATTACHMIMETYPE,
       a.ATTACHNOTE,
       a.CATEGORY,
       a.SUB_CATEGORY,
       a.IMAGEID,
       a.INPUTOPERATOR,
       a.INPUTDATE
  FROM POOLDATA.DATA_ATTACHFILE a
 WHERE a.IDPEGA IN (:1, :2, :3)
 ORDER BY a.INPUTDATE, a.DATAID

-- name: progres_daftar
--
-- Riwayat progres klaim beserta nama kedua statusnya. Pasangan gabungannya sama dengan
-- kueri posisi modul Inbox Progress Claim: STATUS_PROGRESS1 ke GCNM_MST_PROGRESS_KLAIM,
-- STATUS_PROGRESS2 ke GCNM_MST_PROGRESS.
--
-- Bind: :1 :2 :3 kunci klaim.
SELECT g.ID_UPDATE,
       g.TGL_INPUT,
       g.STATUS_PROGRESS1,
       m1.STS_PROGRESS1,
       g.STATUS_PROGRESS2,
       m2.STS_PROGRESS2,
       g.KETERANGAN,
       g.NEXT_FOLLOWUP,
       g.USER_INPUT,
       g.POSISIID
  FROM POOLDATA.GCNM_PROGRESS_CLAIM g
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS_KLAIM m1
              ON m1.ID_PROGRESS = g.STATUS_PROGRESS1
       LEFT JOIN POOLDATA.GCNM_MST_PROGRESS m2
              ON m2.ID_MST = g.STATUS_PROGRESS2
 WHERE g.PNCCASEID IN (:1, :2, :3)
 ORDER BY g.TGL_INPUT DESC, g.ID_UPDATE DESC

-- name: komunikasi_daftar
--
-- Percakapan klaim — `RDB List/GetInboxKomunikasi-SQL.xml`. Kueri lama menyisipkan
-- daftar nomor kasus survey sebagai potongan SQL (`{ASIS:getkomunikasi.D_SURVEY_ID}`);
-- di sini daftarnya diambil subkueri atas T_SURVEYORLIST, sehingga tidak ada nilai yang
-- dirangkai. CASECLAIM ikut dicocokkan untuk percakapan yang langsung menunjuk klaim.
--
-- Bind: :1 :2 :3 kunci klaim, lalu :4 :5 :6 kunci yang sama.
SELECT k.CASEID,
       k.KOMUNIKASIID,
       k.CREATEDDATE,
       k.SENDER,
       k.SENDERNAME,
       k.MESSAGE,
       k.REPLYMESSAGE,
       k.REPLYFROMNAME,
       k.CREATEDATEREPLY,
       k.KOMUNIKASISTATUS,
       k.COMMUNICATE_FROM
  FROM POOLDATA.M_KOMUNIKASI_PNC k
 WHERE k.CASEID IN (SELECT s.CASEID
                      FROM POOLDATA.T_SURVEYORLIST s
                     WHERE s.PNCCASEID IN (:1, :2, :3))
    OR k.CASECLAIM IN (:4, :5, :6)
 ORDER BY k.CREATEDDATE DESC, k.KOMUNIKASIID DESC

-- name: komunikasi_sisip
--
-- Catatan tombol "Kirim ke Inputor" (`Section/AnalystRemarks_sect`). Percakapan tingkat klaim
-- menyimpan kunci klaim di CASEID dan CASECLAIM sekaligus. KOMUNIKASIID (`KOMUNIKASI_SEQ`) dan
-- CREATEDDATE (`sysdate`) diisi default kolom — tidak disebut di sini, sama seperti
-- `inboxkomunikasicabang` `message_insert`.
--
-- Bind: :1 CASEID · :2 CASECLAIM · :3 pengirim (login) · :4 nama pengirim · :5 isi pesan
--       :6 status · :7 tujuan (login Inputor) · :8 kanal (COMMUNICATE_FROM)
INSERT INTO POOLDATA.M_KOMUNIKASI_PNC
       (CASEID, CASECLAIM, SENDER, SENDERNAME, MESSAGE, KOMUNIKASISTATUS,
        COMMUNICATE_TO, COMMUNICATE_FROM)
VALUES (:1, :2, :3, :4, :5, :6, :7, :8)
