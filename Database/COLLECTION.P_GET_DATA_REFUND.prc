CREATE OR REPLACE PROCEDURE COLLECTION.P_GET_DATA_REFUND( P_IDPEGA  IN    VARCHAR2,
                                                                 P_MSG          OUT   VARCHAR2,
                                                                 P_PENERIMA        OUT   VARCHAR2,
                                                                 P_NOREK      OUT   VARCHAR2,
                                                                 P_NILAI     OUT NUMBER) 
                                                     
IS
 ls_count_pol NUMBER(18,4);
 ls_count_pay NUMBER(18,4);
 ls_prod_ke VARCHAR2(100);
 ls_no_spak VARCHAR2(100);
 ls_norek   VARCHAR2(100);
 ls_jumlah  NUMBER(18,4);
 ls_penerima VARCHAR2(100);
 ls_count_norek VARCHAR2(100);

BEGIN

    P_MSG := 'OK';
    P_PENERIMA := '';
    P_NOREK := '';
    P_NILAI := 0;

    SELECT COUNT (1)
     INTO ls_count_pol
     FROM mst_det_sales
    WHERE id_pega = P_IDPEGA;

    IF ls_count_pol > 0 THEN
        SELECT mds_prod_ke, msp_no_spak
         INTO ls_prod_ke, ls_no_spak
         FROM mst_det_sales
        WHERE id_pega = P_IDPEGA;

        select count(1) into ls_count_pay
            from mst_cdnote, mst_cdpayment
        where mst_cdnote.cdn_id = mst_cdpayment.cdn_id
        and mst_cdnote.msp_no_spak = ls_no_spak
        and mst_cdnote.ldb_id = '2'
        and mst_cdnote.mds_prod_ke = ls_prod_ke
        and exists (select 1 from mst_ref_bank where mst_ref_bank.no_ref_bank = mst_cdpayment.no_ref_bank and kbn_no_voucher is not null);

        IF ls_count_pay > 0 THEN

            select count(1) into ls_count_norek
                from mst_cdnote, mst_cdpayment
            where mst_cdnote.cdn_id = mst_cdpayment.cdn_id
            and mst_cdnote.msp_no_spak = ls_no_spak
            and mst_cdnote.ldb_id = '2'
            and mst_cdnote.mds_prod_ke = ls_prod_ke
            and cdp_no_rek is not null
            and exists (select 1 from mst_ref_bank where mst_ref_bank.no_ref_bank = mst_cdpayment.no_ref_bank and kbn_no_voucher is not null);

            IF ls_count_norek > 0 THEN
                select trim(max(cdp_no_rek)), sum(cdp_jml_bayar)
                into ls_norek,ls_jumlah
                    from mst_cdnote, mst_cdpayment
                where mst_cdnote.cdn_id = mst_cdpayment.cdn_id
                and mst_cdnote.msp_no_spak = ls_no_spak
                and mst_cdnote.mds_prod_ke = ls_prod_ke
                and mst_cdnote.ldb_id = '2'
                and cdp_no_rek is not null
                and exists (select 1 from mst_ref_bank where mst_ref_bank.no_ref_bank = mst_cdpayment.no_ref_bank and kbn_no_voucher is not null);

                select account_name into ls_penerima from lst_account where trim(account_no) = trim(ls_norek) and rownum = 1; 

                P_PENERIMA := ls_penerima;
                P_NOREK := ls_norek;
                P_NILAI := ls_jumlah;
            ELSE
                select sum(cdp_jml_bayar)
                into ls_jumlah
                    from mst_cdnote, mst_cdpayment
                where mst_cdnote.cdn_id = mst_cdpayment.cdn_id
                and mst_cdnote.msp_no_spak = ls_no_spak
                and mst_cdnote.mds_prod_ke = ls_prod_ke
                and mst_cdnote.ldb_id = '2'
                and rownum = 1
                and exists (select 1 from mst_ref_bank where mst_ref_bank.no_ref_bank = mst_cdpayment.no_ref_bank and kbn_no_voucher is not null);

                P_PENERIMA := '';
                P_NOREK := 'NO REKENING TIDAK DITEMUKAN';
                P_NILAI := ls_jumlah;
            END IF;

        ELSE
            P_MSG := 'REFUND POLIS INI BELUM DIBAYAR / BELUM GL!';
        END IF;



    ELSE
        P_MSG := 'ID PEGA TIDAK DITEMUKAN!';
    END IF;


    EXCEPTION
            WHEN OTHERS THEN
          P_MSG := 'Error' || TO_CHAR(SQLCODE) || ' ' || SQLERRM || ' ' ||
                     DBMS_UTILITY.FORMAT_ERROR_BACKTRACE;

END P_GET_DATA_REFUND;
/
