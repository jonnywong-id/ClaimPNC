CREATE OR REPLACE PROCEDURE          CONVERTJSONPRODUCTION(vvIDPEGA VARCHAR2,ResponseMsg OUT CLOB, Errmsg OUT VARCHAR2)
IS
    Additional_FlagOldData  VARCHAR2(1);
    Additional_FlagEditData VARCHAR2(1);
    Address_ASMAddress      VARCHAR2(200);
    Address_ASMZipCode      CHAR(5);
    Address_ASMAddressType  CHAR(1);
    agingAmount             NUMBER;
    Coverage_BeginDate      DATE;
    Coverage_EndDate        DATE;
    Coverage_PremiYearly    NUMBER;
    Coverage_SisaPremiYearly NUMBER;
    Coverage_PremiTotal     NUMBER;
    Coverage_DiscountYearly NUMBER;
    Coverage_SisaDiscountYearly NUMBER;
    Coverage_DiscountTotal  NUMBER;
    Coverage_OutgoYearly    NUMBER;
    Coverage_SisaOutgoYearly NUMBER;
    Coverage_OutgoTotal     NUMBER;
    Coverage_CommYearly     NUMBER;
    Coverage_SisaCommYearly NUMBER;
    Coverage_CommTotal      NUMBER;
    Coverage_OCYearly       NUMBER;
    Coverage_SisaOCYearly   NUMBER;
    Coverage_OCTotal        NUMBER;
    Coverage_FlagOldData    VARCHAR2(1);
    Coverage_FlagEditData   VARCHAR2(1);
    Customer_ASMClientID    VARCHAR2(100);
    Customer_ASMClientIDPEGA VARCHAR2(100);
    Customer_ASMIDCard      VARCHAR2(100);
    Customer_ASMNPWP        VARCHAR2(100);
    Customer_Name           VARCHAR2(100);
    Customer_PyTitle        VARCHAR2(50);
    Customer_AsmPosition    VARCHAR2(50);
    Customer_ASMClientIDPEGAOLD VARCHAR2(200);
    cekAgent        NUMBER;
    cekAging        NUMBER;
    cekAsuradur     VARCHAR2(10);
    cekExistsOCID   VARCHAR2(20);
    cekMO           NUMBER;
    cekNPWP         CHAR(1);
    cekLeaderCoas   NUMBER;
    cekStsSyr       VARCHAR2(10);
    clientoldid     VARCHAR2(13);
    cnt             NUMBER;
    cntAgenProgresif NUMBER;
    cntClient       NUMBER;
    cntCoins        NUMBER;
    cntComm         NUMBER;
    cntConverted    NUMBER;
    cntDays         INTEGER;
    cntDaysYear     INTEGER;
    cntDetSales     NUMBER;
    cntDigitPolis   NUMBER;
    cntExist        NUMBER;
    cntJsonPolis    NUMBER;
    cntMaterai      NUMBER;
    cntObjList      NUMBER;
    cntOC           NUMBER;
    cntOldPolicy    NUMBER;
    cntOutgo        INTEGER;
    cntOutgoRow     NUMBER;
    cntPctPPH       NUMBER;
    cntProdke       NUMBER;
    cntTGeneral     NUMBER;
    cntTypeOutgo    NUMBER;
    cntVA           NUMBER;
    conversionNo    NUMBER;
    COMMKWIID       VARCHAR2(13);
    Coins_CoinsID       VARCHAR2(8);
    Coins_Leader        VARCHAR2(10);
    Coins_CoinsName     VARCHAR2(100);    
    Coins_PercentShare  NUMBER;
    Coins_TSIShare      NUMBER;
    Coins_PercentBrokerage NUMBER;
    Coins_PremiShare    NUMBER;
    Coins_Brokerage     NUMBER;
    Coins_BrokerageDPP  NUMBER;
    Coins_HandingFee    NUMBER;
    Coins_PercentHandlingFee NUMBER;
    Coins_PercentPPN    NUMBER;
    Coins_PPN           NUMBER;
    Coins_PercentPPH    NUMBER;
    Coins_PPH           NUMBER;
    Coins_DiscountShare NUMBER;
    Coins_LeaderPolicyNo VARCHAR2(50);
    Coins_FlagDelete    CHAR(1);
    Coins_FlagOldData   VARCHAR2(1);
    Develivery_ASMAddressType   CHAR(1);
    Develivery_ASMContactInfo   VARCHAR2(200);
    Develivery_ASMAddress       VARCHAR2(500);
    Develivery_ASMZipCode       CHAR(5);
    Develivery_pyPhoneNumber    VARCHAR2(50);
    DIGITCHECKSUM       VARCHAR2(30);
    DIGITCHECKSUM2      VARCHAR2(30);
    FacIn_AmountRIComm      NUMBER;
    FacIn_FacInGrossPremiumASM NUMBER;
    FacIn_PercentShare      NUMBER;
    FacIn_NoPolisLeader     VARCHAR2(20);
    Facout_StartPeriod      DATE;
    Facout_EndPeriod        DATE;
    Facout_ReinsurerID      VARCHAR2(10);
    Facout_ReinsurerName    VARCHAR2(30);
    Facout_IsNonMaterial    VARCHAR2(1);
    Facout_SyariahStatus    VARCHAR2(1);
    Facout_RISlipNo         VARCHAR2(20);
    Facout_EDMRINo          VARCHAR2(30);
    Facout_PrintDate        DATE;
    Facout_TotalTotalRIPremium NUMBER;
    Facout_RIPremium        NUMBER;
    Facout_LessComm         NUMBER;
    Facout_UjrahRe          NUMBER;
    flagDeleteOutgo         INTEGER;
    Installment_AdminFee    NUMBER;
    Installment_Commission  NUMBER;
    Installment_Discount    NUMBER;
    Installment_DueDate     DATE;
    Installment_InstallmentNo NUMBER;
    Installment_InstallmentPercentage NUMBER;
    Installment_OC          NUMBER;
    Installment_PaymentTotal NUMBER;
    Installment_Premium     NUMBER;
    Installment_Stamp       NUMBER;
    idKonvert       NUMBER;
    indexLocation   NUMBER;
    indexOccupation NUMBER;
    inputDateJson   DATE; -- tglprint
    INVOICENO       VARCHAR2(20);
    jenisCoas       CHAR(1);
    jmlObjectJson   NUMBER;
    jsonDataclob        CLOB;
    jsonObjectdataclob  CLOB;
    lkuId VARCHAR2(2);
    lppId VARCHAR2(20);
    l_jsonObject JSON_OBJECT_T;
    l_jsonObjectObject JSON_OBJECT_T;
    l_Object_arr JSON_ARRAY_T;
    l_Object_Obj JSON_OBJECT_T;
    l_Location_arr JSON_ARRAY_T;
    l_Location_Obj JSON_OBJECT_T;
    l_Occupation_arr JSON_ARRAY_T;
    l_Occupation_Obj JSON_OBJECT_T;
    indexobject NUMBER;
    l_coverage_arr JSON_ARRAY_T;
    l_coverage_Obj JSON_OBJECT_T;
    indexCoverage NUMBER;
    l_addCoverage_arr JSON_ARRAY_T;
    l_addCoverage_Obj JSON_OBJECT_T;
    indexAddCoverage NUMBER;
    l_CIF_Obj JSON_OBJECT_T;
    l_Payment_Obj JSON_OBJECT_T;
    l_Installment_arr JSON_ARRAY_T;
    l_Installment_Obj JSON_OBJECT_T;
    indexInstallment NUMBER;
    l_Outgo_arr JSON_ARRAY_T;
    l_Outgo_Obj JSON_OBJECT_T;
    indexOutgo NUMBER;
    l_Delivery_arr JSON_ARRAY_T;
    l_Delivery_Obj JSON_OBJECT_T;
    indexDelivery NUMBER;
    l_Address_arr JSON_ARRAY_T;
    l_Address_Obj JSON_OBJECT_T;
    indexAddress NUMBER;
    l_Telfax_arr JSON_ARRAY_T;
    l_Telfax_Obj JSON_OBJECT_T;
    indexTelfax NUMBER;
    l_Coins_arr JSON_ARRAY_T;
    l_Coins_Obj JSON_OBJECT_T;
    indexCoins NUMBER;    
    l_Facin_arr JSON_ARRAY_T;
    l_Facin_Obj JSON_OBJECT_T;
    indexFacin NUMBER;
    l_Customer_Obj JSON_OBJECT_T;
    l_FacOffer_arr JSON_ARRAY_T;
    l_FacOffer_obj JSON_OBJECT_T;
    indexFacOffer NUMBER;
    l_PrintRI_arr JSON_ARRAY_T;
    l_PrintRI_Obj JSON_OBJECT_T;
    indexPrintRI NUMBER;
    l_PrintRIObject_arr JSON_ARRAY_T;
    l_PrintRIObject_Obj JSON_OBJECT_T;
    indexPrintRIObject NUMBER;
    l_OldCoins_arr JSON_ARRAY_T;--OldCoins
    l_OldCoins_Obj JSON_OBJECT_T;
    indexOldCoins NUMBER;
    l_OldOutgo_arr JSON_ARRAY_T; --OldOutgo
    l_OldOutgo_Obj JSON_OBJECT_T;
    indexOldOutgo NUMBER;
    l_Oldcoverage_arr JSON_ARRAY_T; --OldCoverage
    l_Oldcoverage_Obj JSON_OBJECT_T;
    indexOldCoverage NUMBER;
    l_AnekaGeneral_Obj JSON_OBJECT_T;
    l_Obligee_Obj JSON_OBJECT_T;
    matKurang       NUMBER;
    matNoIjin       VARCHAR2(50);
    mclid           VARCHAR2(13);
    MSPNOSPAK       VARCHAR2(12);
    Obligee_KBGDate     DATE;
    Obligee_BankName    VARCHAR2(100);
    Obligee_KBGNo       VARCHAR2(100);
    Obligee_BankGuaranteeNo VARCHAR2(100);
    OldCoins_CoinsID VARCHAR2(8);
    OldCoins_TSIShare NUMBER;
    OldCoins_PremiShare NUMBER;
    OldCoins_Brokerage NUMBER;
    OldCoins_HandingFee NUMBER;
    OldCoins_PPN NUMBER;
    OldCoins_PPH NUMBER;
    OldCoins_DiscountShare NUMBER;
    OldCoverage_BeginDate DATE;
    OldCoverage_EndDate DATE;
    OldCoverage_PremiYearly NUMBER;
    OldCoverage_SisaPremiYearly NUMBER;
    OldCoverage_PremiTotal NUMBER;
    OldCoverage_DiscountYearly NUMBER;
    OldCoverage_SisaDiscountYearly NUMBER;
    OldCoverage_DiscountTotal NUMBER;
    OldCoverage_OutgoYearly NUMBER;
    OldCoverage_SisaOutgoYearly NUMBER;
    OldCoverage_OutgoTotal NUMBER;
    OldCoverage_CommYearly NUMBER;
    OldCoverage_SisaCommYearly NUMBER;
    OldCoverage_CommTotal NUMBER;
    OldCoverage_OCYearly NUMBER;
    OldCoverage_SisaOCYearly NUMBER;
    OldCoverage_OCTotal NUMBER;
    OldOutgo_OutgoAmount NUMBER;
    outgoAmount_ttl NUMBER;
    Outgo_OCID          VARCHAR2(20);
    Outgo_TypeOfOutgo   VARCHAR2(10);
    Outgo_ReceiverOutgo VARCHAR2(100);
    Outgo_SourceBizName VARCHAR2(300);
    Outgo_SourceOfBusiness VARCHAR2(100);
    Outgo_FlagDelete    VARCHAR2(1);
    Outgo_FlagOldData   VARCHAR2(1);
    Outgo_FlagEditData  VARCHAR2(1);
    Outgo_OutgoAmount   NUMBER;
    Outgo_OutgoAmountv  NUMBER;
    Outgo_PercentOutgo  NUMBER;
    Outgo_PPHAmount     NUMBER;
    Outgo_PPNAmount     NUMBER;
    Outgo_PPHType       VARCHAR2(10);
    Outgo_TotalComm     NUMBER;
    Outgo_GrossCommision    NUMBER;
    Outgo_GrossCommisionNew NUMBER;
    Outgo_CommisionOri      NUMBER;
    Outgo_CommisionDPP      NUMBER;
    Outgo_FormulaPercentage NUMBER;
    Outgo_PPNNote       VARCHAR2(500);
    Outgo_PPHNote       VARCHAR2(500);
    Payment_AdminFee    NUMBER;
    Payment_Commision   NUMBER;
    Payment_Diskon      NUMBER;
    Payment_DueDate     DATE;
    Payment_OC          NUMBER;
    Payment_Premium     NUMBER;
    Payment_StampPolicy NUMBER;
    Payment_StampReceipts NUMBER;
    Payment_SumPaymentTotal NUMBER;
    Payment_FlagMaterai VARCHAR2(5);
    Payment_PaymentMethod VARCHAR2(1);
    Payment_PremiTotal  NUMBER;
    Payment_TotalStamp  NUMBER;
    Policy_B2BBatchNo   VARCHAR2(100);
    Policy_CaseID       VARCHAR2(50);
    Policy_StartDateTime DATE;
    Policy_EndDateTime  DATE;
    Policy_ProdDateTime DATE;
    Yearly_ProdDateTime DATE;
    Policy_Currency     VARCHAR2(100);
    Policy_CustomerType VARCHAR2(100);
    Policy_TypeOfPolicy VARCHAR2(100);
    Policy_SyariahStatus VARCHAR2(100);
    Policy_TypeOfPacket VARCHAR2(100);
    Policy_QQName Varchar2(2500);
    Policy_Prodke VARCHAR2(100);
    Policy_PolicyNo VARCHAR2(14);
    Policy_EdmNo VARCHAR2(100);
    Policy_SumOfTSI NUMBER;
    Policy_CurrencyValue NUMBER;
    Policy_DiffDate         VARCHAR2(100);
    Policy_TypeOfCoins      VARCHAR2(100);
    Policy_IsDeclaration    VARCHAR2(10);
    Policy_RefNo            VARCHAR2(200);
    Policy_IdTransaction    VARCHAR2(200);
    Policy_CedingCo         VARCHAR2(100);
    Policy_IsAnekaPerYear   VARCHAR2(20);
    pctpph          NUMBER;
    pctppn          NUMBER;
    pctpphcoas      NUMBER;
    pctppncoas      NUMBER;
    pctrateprogresif NUMBER;
    pphnote         VARCHAR2(200);
    pphnotecoas     VARCHAR2(100);
    prodDateJson    VARCHAR2(500); -- kolom PRODDATETIME di json_polis
    prodkeDetSales  VARCHAR2(5);
    Quotation_BranchCode    VARCHAR2(100);
    Quotation_BranchName    VARCHAR2(200);
    Quotation_BusinessCode  VARCHAR2(100);
    Quotation_BusinessName  VARCHAR2(200);
    Quotation_EdmNote       VARCHAR2(400);
    Quotation_EdmType       VARCHAR2(100);
    Quotation_DetailEDM     VARCHAR2(100);
    Quotation_MarketingCode VARCHAR2(100);
    Quotation_MarketingName VARCHAR2(200);
    Quotation_SobLeader0    VARCHAR2(100);
    Quotation_SobLeader1    VARCHAR2(100);
    Quotation_SobLeader2    VARCHAR2(100);
    Quotation_SourceOfBusiness VARCHAR2(100);
    Quotation_AccessCode    VARCHAR2(100);
    Quotation_GroupPanel    VARCHAR2(100);
    Quotation_OldPolicyNo   VARCHAR2(14);
    Quotation_OldLeader0    VARCHAR2(100);
    Quotation_OldLeader1    VARCHAR2(100);
    responseMessage     CLOB;
    RECEIPTNO           VARCHAR2(13);
    SLIPNO              VARCHAR2(13);
    sourceBiz           VARCHAR2(10);
    sppaNo              VARCHAR2(12);
    stsConvert          NUMBER;
    stsOldPolicy        NUMBER;
    SumInstallment_Premium  NUMBER;
    SumInstallment_OC       NUMBER;
    SumInstallment_Commission NUMBER;
    SumInstallment_Discount NUMBER;
    SumInstallment_AdminFee NUMBER;
    SumInstallment_Stamp    NUMBER;
    sumPaymentTotal     NUMBER;
    Telfax_TelfaxCode   VARCHAR2(20);
    Telfax_TelfaxNumber CHAR(50);
    Telfax_TelfaxType   CHAR(1);
    tempInstallmentNo   VARCHAR2(2);
    tempOCID            VARCHAR2(20);
    TotalCommOC         NUMBER;
    TotalInstallmentPercentage NUMBER;
    TotalBrokerageCoins NUMBER;
    TotalDiscountCoins  NUMBER;
    TotalPercentCoins   NUMBER;
    TotalPremiumCoins   NUMBER;
    TotalTSICoins       NUMBER;
    typeOfCoins         CHAR(1);
    VIRTUALACCOUNTNO    VARCHAR2(16);
    VIRTUALACCOUNTUSDNO VARCHAR2(16);
    VIRTUALACCOUNTNAME  VARCHAR2(400);
    vIdPegaCek          VARCHAR2(100);
    vIdPegaExists       VARCHAR2(100);
    vIsAro              VARCHAR2(100);
    vIsB2B              VARCHAR2(100);
    vIsService          VARCHAR2(100);
    vJumlahObject       NUMBER;
    vLeader0            POOLDATA.T_AGENT.LEADER0%TYPE;
    vLeader1            POOLDATA.T_AGENT.LEADER0%TYPE;
    vOldLeader0         POOLDATA.T_AGENT.OLDID%TYPE;
    vOldLeader1         POOLDATA.T_AGENT.OLDID%TYPE;
    vNPWP               POOLDATA.T_MCLIENT.NPWP%TYPE;
    vPolicynoCek        VARCHAR2(40);
    vTotalYear          NUMBER;
    vPPNSTATUS          VARCHAR2(10);
    CNTOUTGO2           NUMBER;
    
    queue_options       DBMS_AQ.ENQUEUE_OPTIONS_T;
    message_properties  DBMS_AQ.MESSAGE_PROPERTIES_T;
    message_id          RAW(16);
    aro_message         POOLDATA.queue_aro_type;
    b2b_message         POOLDATA.queue_b2b_type;
    my_message          POOLDATA.queue_polis_type;
    service_message     POOLDATA.queue_service_type;
    
    
    CURSOR curr_pool_outgo(p_caseid in VARCHAR2, p_policyNo in VARCHAR2 )
      IS
        SELECT ocidold,ocid, sourcebizname,ppnpercent, pphpercent, 
               pphtype, typeofoutgo, receiveroutgo, percentoutgo, outgoamount, installmentno, pphnote, ppnnote
        from POOLDATA.t_pool_outgo WHERE CaseID = p_caseid  And PolicyNo= p_policyNo;
        
    CURSOR curr_pool_installment(p_caseid in VARCHAR2, p_policyNo in VARCHAR2 )
      IS
        SELECT installmentno
        from POOLDATA.t_pool_installment WHERE CaseID = TRIM(p_caseid)  And PolicyNo= TRIM(p_policyNo);
        
    CURSOR curr_person_obj(p_caseid in VARCHAR2)
    IS 
        Select idpega, nopolis, indexobject, to_clob(objectdata) objectdata, to_clob(coveragedata) coveragedata
        from POOLDATA.t_personlist
        WHERE idpega = p_caseid;
        
    CURSOR curr_vehicle_obj(p_caseid in VARCHAR2)
    IS
        SELECT idpega, nopolis, indexobject, to_clob(vehiclelist) vehiclelist, to_clob(coveragelist) coveragelist
        from POOLDATA.t_vehiclelist
        WHERE idpega = p_caseid;
        
    CURSOR curr_cargo_obj (p_caseid in VARCHAR2)
    IS
        Select idpega, nopolis, indexobject, to_clob(objectdata) objectdata, to_clob(coveragedata) coveragedata
        from POOLDATA.t_cargolist
        WHERE idpega = p_caseid;
      
    CURSOR curr_property_obj (p_caseid in VARCHAR2)
    IS
        Select idpega, nopolis, indexobject, to_clob(coveragelist) coveragelist, to_clob(propertylist) propertylist
        from POOLDATA.T_PROPERTYLIST
        WHERE idpega = p_caseid;
        
    CURSOR curr_aneka_obj (p_caseid in VARCHAR2)
    IS
        Select idpega, nopolis, indexobject, to_clob(coveragelist) coveragelist, to_clob(anekalist) anekalist
        from POOLDATA.T_ANEKALIST
        WHERE idpega = p_caseid;
    
BEGIN
    --DBMS_OUTPUT.PUT_LINE('Waktu Process Mulai= ' || CURRENT_TIMESTAMP );
    cnt := 1;
--    BEGIN
--        SELECT COUNT(1) into cnt from POOLDATA.tempprocess WHERE caseid = vvIDPEGA;
--        while cnt >= 1 loop
--            SELECT COUNT(1) into cnt from POOLDATA.tempprocess WHERE caseid = vvIDPEGA;
--            sys.dbms_lock.SLEEP(1);
--        end loop;
--        INSERT INTO POOLDATA.tempprocess(caseid) values (vvIDPEGA);
--        COMMIT;
--    END;
    BEGIN
        BEGIN
            SELECT max(conversion_no)+1 into ConversionNo from POOLDATA.LOG_KONVERSI_PROD WHERE ID_PEGA = vvIDPEGA;
            if ConversionNo is null then
                ConversionNo := 1;
            END IF;
        exception when no_data_found then
            ConversionNo := 1;
        END;
        INSERT INTO POOLDATA.LOG_KONVERSI_PROD (TGL_INPUT, ID_PEGA, WAKTU_MULAI, CONVERSION_NO, FLAG_KONVERSI)
            VALUES (CURRENT_TIMESTAMP, vvIDPEGA, CURRENT_TIMESTAMP, ConversionNo, '1');
        COMMIT;
        
        BEGIN
            SELECT COUNT(1) into cntJsonPolis FROM POOLDATA.JSON_POLIS WHERE idpega = vvIDPEGA;
            SELECT COUNT(1) into cntTGeneral FROM POOLDATA.T_GENERAL WHERE idpega = vvIDPEGA;
            if cntJsonPolis > 1 then
                Errmsg := 'Gagal proses data, IDPEGA : ' || vvIDPEGA || ' Terdapat lebih dari 1 row di JSON_POLIS';
                ROLLBACK;
                UPDATE POOLDATA.LOG_KONVERSI_PROD SET conversion_note = Errmsg, waktu_selesai = CURRENT_TIMESTAMP
                    WHERE id_pega = vvIDPEGA and conversion_no = ConversionNo;
                UPDATE POOLDATA.JSON_POLIS SET STS_KONVERSI = 99, ERR_NOTE = Errmsg
                    WHERE idpega = vvIDPEGA;
                --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
                COMMIT;
                RETURN;
            END IF;
            if cntJsonPolis = 0 then
                Errmsg := 'IDPEGA : ' || vvIDPEGA || ' Tidak ditemukan di JSON_POLIS';
                ROLLBACK;
                UPDATE POOLDATA.LOG_KONVERSI_PROD SET conversion_note = Errmsg, waktu_selesai = CURRENT_TIMESTAMP
                    WHERE id_pega = vvIDPEGA and conversion_no = ConversionNo;
                UPDATE POOLDATA.JSON_POLIS SET STS_KONVERSI = 99, ERR_NOTE = Errmsg
                    WHERE idpega = vvIDPEGA;
                --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
                COMMIT;
                RETURN;
            END IF;
            if cntTGeneral > 1 then
                Errmsg := 'ID Pega : ' || vvIDPEGA || ' Terdapat lebih dari 1 row di T_GENERAL';
                ROLLBACK;
                UPDATE POOLDATA.LOG_KONVERSI_PROD SET conversion_note = Errmsg, waktu_selesai = CURRENT_TIMESTAMP
                    WHERE id_pega = vvIDPEGA and conversion_no = ConversionNo;
                UPDATE POOLDATA.JSON_POLIS SET STS_KONVERSI = 99, ERR_NOTE = Errmsg
                    WHERE idpega = vvIDPEGA;
                --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
                COMMIT;
                RETURN;
            END IF;
            if cntTGeneral = 0 then
                Errmsg := 'IDPEGA : ' || vvIDPEGA || ' Tidak ditemukan di T_GENERAL';
                ROLLBACK;
                UPDATE POOLDATA.LOG_KONVERSI_PROD SET conversion_note = Errmsg, waktu_selesai = CURRENT_TIMESTAMP
                    WHERE id_pega = vvIDPEGA and conversion_no = ConversionNo;
                UPDATE POOLDATA.JSON_POLIS SET STS_KONVERSI = 99, ERR_NOTE = Errmsg
                    WHERE idpega = vvIDPEGA;
                --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
                COMMIT;
                RETURN;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            Errmsg := 'Error cek json_polis & t_general : ' || SQLERRM;
            ROLLBACK;
            UPDATE POOLDATA.LOG_KONVERSI_PROD SET conversion_note = Errmsg, waktu_selesai = CURRENT_TIMESTAMP
                WHERE id_pega = vvIDPEGA and conversion_no = ConversionNo;
            UPDATE POOLDATA.JSON_POLIS SET STS_KONVERSI = 99, ERR_NOTE = Errmsg
                WHERE idpega = vvIDPEGA;
            --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
            COMMIT;
            RETURN;
        END;

        BEGIN 
            SELECT a.sts_konversi,a.nopolis,a.Idpega,to_clob(a.data_jsonblob),a.tgl_Input,isb2b,isservice,isaro  
            into stsConvert,vpolicynocek,vidpegacek,jsondataclob,InputDateJson ,vISB2B,visservice,visaro
                from POOLDATA.json_polis a , POOLDATA.t_general b WHERE a.idpega = vvIDPEGA and a.idpega=b.idpega;
                
            if stsConvert = 1 then
                SELECT COUNT(1) into cntConverted from POOLDATA.t_pool_policy WHERE CaseID = vvIDPEGA;
--                SELECT COUNT(1) into cntDetSales from mst_det_sales@asmd.sinarmas.co.id WHERE id_pega = vvIDPEGA;
--                if cntConverted = 0 and cntDetSales = 1 then 
--                    BEGIN
--                        SELECT trace_message into ResponseMsg from POOLDATA.log_pega_conversion
--                            WHERE id_pega = vvIDPEGA and conversion_sts = '1';
--                        update POOLDATA.json_polis set sts_konversi = '1', err_note = null WHERE idpega = vvIDPEGA;
--                        
--                        --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
--                        COMMIT;
--                        RETURN;
--                    EXCEPTION WHEN OTHERS THEN
--                        cntConverted :=0;-- klo tidak nemu konversi db lagi 
--                    END;
--                END IF;
                
                BEGIN
                    IF vidpegacek LIKE 'ASM-FW-GISFW-WORK NB-%' THEN
                        SELECT COUNT(0) INTO cntJsonPolis FROM JSON_POLIS WHERE NOPOLIS = vpolicynocek AND IDPEGA LIKE 'ASM-FW-GISFW-WORK NB-%' AND IDPEGA <> vidpegacek AND STS_KONVERSI = 1;
                        IF cntJsonPolis > 0 THEN
                            SELECT IDPEGA INTO vIdPegaExists FROM JSON_POLIS WHERE NOPOLIS = vpolicynocek AND IDPEGA LIKE 'ASM-FW-GISFW-WORK NB-%' AND IDPEGA <> vidpegacek AND STS_KONVERSI = 1;
                            Errmsg := 'Error proses data NB; No Polis ' || vpolicynocek || ' sudah masuk dengan ID Pega: ' || vIdPegaExists;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;
                    END IF;
                END;
            END IF;
        EXCEPTION WHEN OTHERS THEN
            Errmsg := 'Error, Object Json Data tidak ada 1 : ' || SQLERRM || dbms_utility.format_error_backtrace;
            ROLLBACK;
            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
            COMMIT;
            RETURN;
        END;
        
        SELECT COUNT(1) into cntConverted from POOLDATA.t_pool_policy WHERE CaseID = vvIDPEGA and policyno = vpolicynocek;
        if cntConverted > 0 then 
            --DELETE FROM POOLDATA.TEMPPROCESS WHERE CASEID = vvIDPEGA;
            --if cntConverted > 0 then 
                --SELECT responsemsg into responsemessage from POOLDATA.t_pool_policy WHERE CaseID = vvIDPEGA;
                SELECT COUNT(1) into cntOutgoRow from POOLDATA.t_pool_outgo WHERE caseid = vvIDPEGA;
                if cntOutgoRow > 0 then
                    DBMS_OUTPUT.PUT_LINE('Re-calculate outgo');
                    --POOLDATA.COUNT_OUTGO_PROD(vvIDPEGA, Errmsg);
                END IF;
                
                SELECT  '{' ||  TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'ResponseMessage', 'OK', ','))               ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'PolicyNo', POLICYNO, ','))                  ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'DigitCheckSum', DIGITCHECKSUM, ','))        ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'StampNumber', STAMPNUMBER, ','))            ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'VirtualAccountNumber', NOVA, ','))          ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'VirtualAccountNumberUSD', NOVAUSD, ','))    ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'VirtualAccountName', NAMAVA, ','))    ||
                               TO_CLOB (
                               POOLDATA.jsonObjectWriter ( 'AgingAmount', '', ',',AGINGAMOUNT,'number2'))    ||
                               To_clob ('"Payment":{') ||
                               To_clob ('"ListInstallment":[') || 
                                  POOLDATA.ConvertInstallmentProduksi(a.CaseID)
                               || To_clob (']')
                               || '},' || 
                               To_clob ('"CoinsList":[') ||
                                    POOLDATA.ConvertCoinsProduksi(a.CaseID)
                               || To_clob (']')
                           || '}'  into RESPONSEMESSAGE from POOLDATA.t_pool_Policy a WHERE caseid = vvIDPEGA and rownum = 1;
                
                ResponseMsg := responsemessage;
                
                if stsConvert <> 1 or stsConvert is null then
                    IF (jsondataclob is not null or jsondataclob <> '') then      
                        BEGIN 
                            l_jsonObject := Json_object_t(jsondataclob);
                        EXCEPTION WHEN OTHERS THEN
                            Errmsg := 'Error, JSON Data invalid : ' || SQLERRM ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END;
                        Policy_IsDeclaration    := l_jsonObject.Get_string('IsDeclaration');
                        Quotation_BusinessCode  := l_jsonObject.Get_object('Quotation').Get_string('BusinessCode');
                        Quotation_BusinessCode  := POOLDATA.getnewID(Quotation_BusinessCode,'M_BUSINESS','ID','kosong', 'kosong', 'OLDID');
                        Quotation_SourceOfBusiness := l_jsonObject.Get_object('Quotation').Get_string('SourceOfBusiness');
                        Quotation_GroupPanel    := l_jsonObject.Get_object('Quotation').Get_string('GroupPanel');
                        POOLDATA.generate_lst_agen(Quotation_SourceOfBusiness, Errmsg);
                        
                        --get total object 
                        IF Quotation_GroupPanel in ('007') then
                            SELECT COUNT(1) into vJumlahObject from POOLDATA.t_vehiclelist WHERE idpega = vvIDPEGA;
                        elsif Quotation_GroupPanel in ('004') then
                            SELECT COUNT(1) into vJumlahObject from POOLDATA.t_cargolist WHERE idpega = vvIDPEGA;
                        elsif Quotation_GroupPanel in ('006') then
                            SELECT COUNT(1) into vJumlahObject from POOLDATA.t_propertylist WHERE idpega = vvIDPEGA;
                        elsif Quotation_GroupPanel in ('003') then
                            SELECT COUNT(1) into vJumlahObject from POOLDATA.t_anekalist WHERE idpega = vvIDPEGA;
                        elsif Quotation_GroupPanel in ('005','002','010') then
                            SELECT COUNT(1) into vJumlahObject from POOLDATA.t_personlist WHERE idpega = vvIDPEGA;
                        END IF;
                    ELSE
                        Errmsg := 'Json data Null';
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF; 
                    
                    if Policy_IsDeclaration = 'True' and Quotation_BusinessCode = '04' then
                        update POOLDATA.json_polis set sts_konversi = 1, tgl_konversi = sysdate WHERE idpega = vvIDPEGA ;
                        COMMIT;
                    else
                        BEGIN
                            if vJumlahObject > 50 OR Quotation_SourceOfBusiness = '10034413' then
                                if vISB2B='1' then
                                        b2b_message := POOLDATA.queue_b2b_type(1,vvIDPEGA,'b2b');
                                        DBMS_AQ.ENQUEUE(
                                        queue_name => 'POOLDATA.Q_PROCESB2B',
                                        enqueue_options => queue_options,
                                        message_properties => message_properties,
                                        payload => b2b_message,
                                        msgid => message_id);
                                
                                elsif visservice='1' then
                                        service_message := POOLDATA.queue_service_type(1,vvIDPEGA,'service');
                                        DBMS_AQ.ENQUEUE(
                                        queue_name => 'POOLDATA.Q_PROCESSERVICE',
                                        enqueue_options => queue_options,
                                        message_properties => message_properties,
                                        payload => service_message,
                                        msgid => message_id);
                                
                                elsif visaro='1' then
                                        aro_message  := POOLDATA.queue_aro_type(1,vvIDPEGA,'aro');
                                        DBMS_AQ.ENQUEUE(
                                        queue_name => 'POOLDATA.Q_PROCESARO',
                                        enqueue_options => queue_options,
                                        message_properties => message_properties,
                                        payload => aro_message,
                                        msgid => message_id);
                                else
                                    if instr(vvIDPEGA,'EDM')>0 then
                                        my_message := POOLDATA.queue_polis_type(1,vvIDPEGA,'edm');
                                    else
                                        my_message := POOLDATA.queue_polis_type(1,vvIDPEGA,'polis');
                                    END IF;
                                    DBMS_AQ.ENQUEUE(
                                        queue_name => 'POOLDATA.Q_PROCESPOLIS',
                                        enqueue_options => queue_options,
                                        message_properties => message_properties,
                                        payload => my_message,
                                        msgid => message_id);
                                END IF;
                            else
                                if vISB2B='1' then
                                    POOLDATA.processqueuedirect (vvIDPEGA,'b2b');
                                elsif visservice='1' then
                                    POOLDATA.processqueuedirect (vvIDPEGA,'service');
                                elsif visaro='1' then
                                    POOLDATA.processqueuedirect (vvIDPEGA,'aro');
                                else
                                    if instr(vvIDPEGA,'EDM')>0 then
                                        POOLDATA.processqueuedirect (vvIDPEGA,'edm');
                                    else
                                        POOLDATA.processqueuedirect (vvIDPEGA,'polis');
                                    END IF;
                                END IF;
                            END IF;
                        END;
                    END IF;
                END IF;
            RETURN;
           -- END IF;
        END IF;
                      
        IF (jsondataclob is not null or jsondataclob <> '') then
            BEGIN 
                l_jsonObject := Json_object_t(jsondataclob);
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error, JSON Data Invalid : ' || SQLERRM ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            -- Data Policy Gathering
            Begin
                Policy_CaseID:=l_jsonObject.Get_string('CaseID');  
                if Policy_CaseID <> vvIDPEGA then
                    Errmsg := 'Data Policy.CaseID :'||Policy_CaseID||' Tidak sama dengan IdPega di Json_Polis :'|| vvIDPEGA;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                Policy_Prodke:=l_jsonObject.Get_string('Prodke');
                Policy_PolicyNo:=TRIM(l_jsonObject.Get_string('PolicyNo'));
                if Policy_PolicyNo is null then
                    Errmsg := 'Data Policy.PolicyNo Tidak boleh NULL ' ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
              Policy_IsDeclaration := l_jsonObject.Get_string('IsDeclaration');
              if Policy_IsDeclaration is null then
                Policy_IsDeclaration := 'False';
              END IF;
              Policy_Currency := l_jsonObject.Get_string('Currency');
              if Policy_Currency is null then
                Errmsg := 'Data Policy.Currency Tidak boleh NULL ' ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                Return ;
              END IF;
              Policy_CurrencyValue := l_jsonObject.Get_number('CurrencyValue');
              if Policy_CurrencyValue is null then
                Errmsg := 'Data Policy.CurrencyValue Tidak boleh NULL ' ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                Return ;
              END IF;
              Policy_CustomerType := l_jsonObject.Get_string('CustomerType');
              if Policy_CustomerType is null then
                Errmsg := 'Data Policy.CustomerType Tidak boleh NULL ' ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                Return ;
              END IF;
              Policy_TypeOfCoins := l_jsonObject.Get_string('TypeOfCoins');
              if Policy_TypeOfCoins is null then
                Errmsg := 'Data Policy.TypeOfCoins Tidak boleh NULL ' ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                Return ;
              END IF;
                Policy_StartDateTime    := TO_DATE(SUBSTR(l_jsonObject.Get_string('StartDateTime'),0,8),'YYYYMMdd');
                IF Policy_StartDateTime <= TO_DATE('01-01-1970','DD-MM-YYYY') THEN
                    Errmsg := 'Format Policy.StartDateTime Invalid';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                Policy_EndDateTime      := TO_DATE(SUBSTR(l_jsonObject.Get_string('EndDateTime'),0,8),'YYYYMMdd');
                IF Policy_EndDateTime >= TO_DATE('01-01-2200','DD-MM-YYYY') THEN
                    Errmsg := 'Format Policy.EndDateTime Invalid';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                Policy_ProdDateTime     := TO_DATE(SUBSTR(l_jsonObject.Get_string('ProdDateTime'),0,8),'YYYYMMdd');
                IF Policy_ProdDateTime is null or Policy_ProdDateTime = '' then
                    SELECT proddatetime into ProdDateJson from POOLDATA.json_polis WHERE idpega=vvIDPEGA;
                    Policy_ProdDateTime:=TO_DATE(Substr(ProdDateJson,0,8),'YYYYMMdd');
                    IF Policy_ProdDateTime is null then
                        Errmsg := 'Data ProdDateTime tidak boleh kosong ';
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                END IF;
                Policy_SyariahStatus    := l_jsonObject.Get_string('SyariahStatus');
                Policy_B2BBatchNo := l_jsonObject.Get_string('B2BBatchNo');
                Policy_IdTransaction := l_jsonObject.Get_string('IdTransaction');
                if trunc(sysdate) > trunc(Policy_ProdDateTime) then
                    Policy_ProdDateTime := sysdate;
                END IF;
              if Policy_StartDateTime is null or Policy_StartDateTime = '' then
                Errmsg := 'Data StartDateTime tidak boleh kosong ';
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                Return ;
              END IF;
              if Policy_EndDateTime is null  or Policy_EndDateTime = '' then
                Errmsg :='Data EndDateTime tidak boleh kosong ';
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                Return ;
              END IF;
                if trunc(Policy_EndDateTime) < trunc(Policy_StartDateTime) then
                    Errmsg := 'Data Policy EndDateTime :' || to_char(Policy_EndDateTime,'dd/MM/yyyy') || ' lebih kecil dari Policy StartDateTime :' || to_Char(Policy_StartDateTime,'dd/MM/yyyy');
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    Return ;
                END IF;
                vTotalYear := ceil(months_between(Policy_EndDateTime, Policy_StartDateTime) /12) ;
              
                Policy_QQName := l_jsonObject.Get_string('QQName');
                IF Policy_QQName IS NULL AND (vvIDPEGA LIKE 'ASM-FW-GISFW-WORK NB-%' OR vvIDPEGA LIKE 'ASM-FW-GISFW-WORK RNW-%') THEN
                    Errmsg := 'Data Policy.QQName tidak boleh kosong';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                Policy_IsAnekaPerYear   := l_jsonObject.Get_string('IsAnekaPerYear');
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error on Policy gathering : ' || SQLERRM || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            LkuId := POOLDATA.getnewID(Policy_Currency,'M_CURRENCY','ID','kosong', 'kosong', 'OLDID');
            -- End Policy Gathering 
            
            -- Quotation Gathering
            Begin
                Quotation_EdmType   := l_jsonObject.Get_object('Quotation').Get_string('EdmType');
                Quotation_DetailEDM := l_jsonObject.Get_object('Quotation').Get_string('DetailEDM');
                Quotation_SobLeader0 := l_jsonObject.Get_object('Quotation').Get_string('SobLeader0');
                if Quotation_SobLeader0 is null then
                    Errmsg := 'Data Quotation.SobLeader0 Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                Quotation_SobLeader1 := l_jsonObject.Get_object('Quotation').Get_string('SobLeader1');
                if Quotation_SobLeader1 is null then
                    Errmsg := 'Data Quotation.SobLeader1 Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                Quotation_BranchCode := l_jsonObject.Get_object('Quotation').Get_string('BranchCode');
                if Quotation_BranchCode is null then
                    Errmsg := 'Data Quotation.BranchCode Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                ELSE
                    if Quotation_BranchCode = '100081' then
                        Errmsg := 'Cabang tidak boleh Kantor Pusat, BranchCode : ' || Quotation_BranchCode;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
                END IF;
                Quotation_BusinessCode := l_jsonObject.Get_object('Quotation').Get_string('BusinessCode');
                if Quotation_BusinessCode is null then
                    Errmsg := 'Data Quotation.BusinessCode Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                SELECT STATUSSYARIAH INTO cekStsSyr FROM POOLDATA.BUSINESS WHERE ID = Quotation_BusinessCode;
                IF cekStsSyr = '1' AND Policy_SyariahStatus = '0' THEN
                    Errmsg := 'Bisnis Syariah, Data Policy.SyariahStatus Tidak Sesuai Kode Bisnis';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                IF Policy_SyariahStatus = '1' THEN
                    SELECT STATUSSYARIAH INTO cekStsSyr FROM POOLDATA.BUSINESS WHERE ID = Quotation_BusinessCode;
                    IF cekStsSyr = 0 THEN
                        Errmsg := 'Data Policy.SyariahStatus Tidak Sesuai Kode Bisnis, SyariahStatus:' || Policy_SyariahStatus;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    ELSE
                        --IF vvIDPEGA LIKE 'ASM-FW-GISFW-WORK NB-%' AND TRUNC(SYSDATE) = TO_DATE('01-02-2026','DD-MM-YYYY') THEN
                        --IF TRUNC(InputDateJson) >= TO_DATE('01-02-2026','DD-MM-YYYY') THEN
                            Errmsg := 'Polis Syariah Diterbitkan Di Server Syariah';
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        --END IF;
                    END IF;
                END IF;
                Quotation_GroupPanel := l_jsonObject.Get_object('Quotation').Get_string('GroupPanel');
                if Quotation_GroupPanel is null then
                    SELECT grouppanel into Quotation_GroupPanel from POOLDATA.business WHERE id = Quotation_BusinessCode;
                END IF;
                Quotation_SobLeader2 := l_jsonObject.Get_object('Quotation').Get_string('SobLeader2');
                Quotation_SourceOfBusiness := l_jsonObject.Get_object('Quotation').Get_string('SourceOfBusiness');
                Quotation_MarketingCode := l_jsonObject.Get_object('Quotation').Get_string('MarketingCode');
                if Quotation_MarketingCode is null then
                    Errmsg := 'Data Quotation.MarketingCode Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                
                SELECT COUNT(1) INTO cekMO FROM POOLDATA.M_MARKETINGOFFICER WHERE id =  Quotation_MarketingCode;  
                if cekMO = 0 THEN
                    Errmsg := 'Kode MO tidak ditemukan di M_MARKETINGOFFICER, Quotation.MarketingCode:'||Quotation_MarketingCode;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                
                if Quotation_SourceOfBusiness is null then
                    Errmsg := 'Data  Quotation.SourceOfBusiness Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                SELECT COUNT(1) into cekAgent from POOLDATA.m_agent WHERE id = Quotation_SourceOfBusiness;
                if cekAgent = 0 then
                    Errmsg := 'SourceOfBusiness tidak ditemukan di M_AGENT, Quotation.SourceOfBusiness:'||Quotation_SourceOfBusiness;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;  
                POOLDATA.generate_lst_agen(Quotation_SourceOfBusiness,Errmsg);
                SourceBiz := POOLDATA.getnewID(Quotation_SourceOfBusiness,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                Quotation_BusinessCode := POOLDATA.getnewID(Quotation_BusinessCode,'M_BUSINESS','ID','kosong', 'kosong', 'OLDID');    
                Quotation_OldLeader0 := POOLDATA.getnewID(Quotation_SobLeader0,'M_AGENT','ID','kosong', 'kosong', 'OLDID');    
                Quotation_OldLeader1 := POOLDATA.getnewID(Quotation_SobLeader1,'M_AGENT','ID','kosong', 'kosong', 'OLDID');  
                
                IF Quotation_EdmType = '7' THEN
                    SELECT NVL(COLLECTION.F_SISA_AGING@ASMD.SINARMAS.CO.ID(vpolicynocek),0) INTO cekAging FROM DUAL;
                    IF cekAging <= 1 THEN
                        Errmsg := 'Gagal Konversi EDM Batal Otomatis, Polis Sudah Lunas!';
                        ROLLBACK;
                        UPDATE POOLDATA.LOG_KONVERSI_PROD SET conversion_note = Errmsg, waktu_selesai = CURRENT_TIMESTAMP
                            WHERE id_pega = vvIDPEGA and conversion_no = ConversionNo;
                        UPDATE POOLDATA.JSON_POLIS SET STS_KONVERSI = 95, ERR_NOTE = Errmsg
                            WHERE idpega = vvIDPEGA;
                        COMMIT;
                        RETURN;
                    END IF;
                END IF;  
            Exception when others then
                Errmsg := 'Error on Quotation gathering : ' || SQLERRM ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            -- End Quotation Gathering
            
            -- Delivery Address Gathering   
            Begin 
                if  l_jsonObject.Get_array('DeliveryAddressList') is not null then
                    l_Delivery_arr :=l_jsonObject.Get_array('DeliveryAddressList');
                    indexDelivery := 0;
                    FOR f IN 0 .. l_Delivery_arr.get_size - 1 LOOP
                        l_Delivery_Obj :=Json_object_t(l_Delivery_arr.Get(indexDelivery));
                        Develivery_ASMAddressType :=l_Delivery_Obj.Get_string('ASMAddressType');
                        if Develivery_ASMAddressType is null then
                            Errmsg := 'Data DeliveryAddressList.ASMAddressType Tidak boleh NULL' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            Return;
                        END IF;
                        indexDelivery := indexDelivery+1;
                    END LOOP;
                END IF;
            Exception when others then
                Errmsg := 'Error on Delivery gathering : '|| SQLERRM || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            
            -- End Delivery Address Gathering  
            
            -- Obligee Gathering
            BEGIN
                IF Quotation_BusinessCode = '50' AND Policy_TypeOfCoins <> 'F' THEN
                    l_AnekaGeneral_Obj := l_jsonObject.Get_object('AnekaGeneral');
                    l_Obligee_Obj :=   l_AnekaGeneral_Obj.Get_object('Obligee');
                    
                    Obligee_KBGDate := to_Date(Substr(l_Obligee_Obj.Get_string('KBGDate'),0,8),'YYYYMMdd');
                    Obligee_BankName := l_Obligee_Obj.Get_string('BankName');
                    Obligee_KBGNo := substr(l_Obligee_Obj.Get_string('KBGNo'),0,98);
                    Obligee_BankGuaranteeNo := l_Obligee_Obj.Get_string('BankGuaranteeNo');
                    
                    if Obligee_KBGDate is null then
                        Errmsg := 'KBGDate Tidak boleh NULL ' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    
                    if Obligee_BankName is null then
                        Errmsg := 'Obligee.BankName Tidak boleh NULL ' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    
                    if Obligee_KBGNo is null then
                        Errmsg :='KBGNo Tidak boleh NULL ' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    
                    if Obligee_BankGuaranteeNo is null then
                        Errmsg :='BankGuaranteeNo Tidak boleh NULL ' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    
                END IF;
            Exception when others then
                Errmsg := 'Error on Obligee gathering : '|| SQLERRM || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            -- End Obligee Gathering
            
            -- CoinsList Gathering
            BEGIN
                if  Policy_TypeOfCoins in ('1','2') then
                    if  l_jsonObject.Get_array('CoinsList') is not null then
                        l_Coins_arr :=l_jsonObject.Get_array('CoinsList');
                        indexCoins := 0;
                        FOR h IN 0 .. l_Coins_arr.get_size - 1 LOOP
                            l_Coins_Obj :=Json_object_t(l_Coins_arr.Get(indexCoins));
                            Coins_CoinsID := l_Coins_Obj.Get_string('CoinsID');
                            if Coins_CoinsID is null then     
                                Errmsg := 'Error on CoinsList gathering : CoinsID Tidak Boleh Kosong';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            BEGIN
                                SELECT COUNT(1) INTO cntCoins FROM M_AGENT WHERE ID = Coins_CoinsID;
                                IF cntCoins = 0 THEN
                                    Errmsg := 'CoinsID tidak ditemukan di M_AGENT, CoinsID: ' || Coins_CoinsID;
                                    ROLLBACK;
                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                    COMMIT;
                                    RETURN;
                                END IF;
                                SELECT ISASURADUR INTO cekAsuradur from POOLDATA.agent WHERE id = Coins_CoinsID;
                                IF cekAsuradur = 0 THEN
                                    Errmsg := 'Error get CoinsList: CoinsID harus menggunakan asuradur, CoinsID: ' || Coins_CoinsID || ', IsAsuradur: ' || cekAsuradur;
                                    ROLLBACK;
                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                    COMMIT;
                                    RETURN;
                                END IF;
                                GENERATE_LST_ASURADUR(Coins_CoinsID);
                            EXCEPTION WHEN OTHERS THEN
                                Errmsg := 'Error Generate Lst_Asuradur, CoinsID: ' || Coins_CoinsID || u'\000A' || SQLERRM || u'\000A' || dbms_utility.format_error_backtrace;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END;
                            indexCoins := indexCoins+1;
                        END LOOP;
                    ELSE 
                        Errmsg := 'Error on CoinsList gathering : Data Coas Tidak ditemukan / Null' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
                END IF;
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error on CoinsList gathering : ' || SQLERRM ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            
            --VALIDATE EDM/RNW
            IF (Policy_CaseID LIKE 'ASM-FW-GISFW-WORK EDM-%' OR Policy_CaseID like 'E%') THEN 
                IF Quotation_EdmType IS NULL THEN
                    Errmsg := 'Case EDM, Quotation.EDMType Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
            ELSIF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK RNW-%' THEN
                Quotation_OldPolicyNo := l_jsonObject.Get_object('Quotation').Get_string('OldPolicyNo');
                if Quotation_OldPolicyNo IS NULL THEN
                    Errmsg := 'Case RNW, Quotation.OldPolicyNo Tidak Boleh NULL';
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
--                else
--                    SELECT COUNT(1) into cntOldPolicy from POOLDATA.json_polis WHERE nopolis= Quotation_OldPolicyNo and idpega like 'ASM-FW-GISFW-WORK NB-%';
--                    if cntOldPolicy = 0 then
--                        Errmsg := 'Produksi RNW, Data New Business tidak ditemukan';
--                        RETURN;
--                    else
--                        SELECT sts_konversi into stsOldPolicy 
--                            from POOLDATA.json_polis WHERE nopolis= Quotation_OldPolicyNo and idpega like 'ASM-FW-GISFW-WORK NB-%';
--                        if stsOldPolicy is null or stsOldPolicy <> 1 then
--                            Errmsg := 'Produksi RNW, Data New Business belum masuk / belum konversi';
--                            RETURN;
--                        END IF;
--                    END IF;
                END IF;    
            END IF;
            
            -- Customer Gathering 
            BEGIN
                l_CIF_Obj := l_jsonObject.Get_object('CIFData') ;
                IF Policy_CustomerType ='1' THEN
                    l_Customer_Obj := l_CIF_Obj.Get_object('Customer_P') ;
                    Customer_ASMClientIDPEGA := l_Customer_Obj.Get_string('ASMClientID') ;
                ELSE 
                    l_Customer_Obj := l_CIF_Obj.Get_object('Customer_C') ;
                    Customer_ASMClientIDPEGA := l_Customer_Obj.Get_string('ASMClientID') ;
                    IF Customer_ASMClientIDPEGA is null then
                        Customer_ASMClientIDPEGA := l_Customer_Obj.Get_string('ASMComID') ;
                    END IF;
                END IF;
                
                IF Customer_ASMClientIDPEGA IS NULL THEN
                    Errmsg := 'Data CIFData ASMClientID / ASMComID  Tidak boleh NULL' ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                
                BEGIN
                    SELECT oldid INTO clientoldid FROM m_client WHERE id = Customer_ASMClientIDPEGA;
                EXCEPTION WHEN NO_DATA_FOUND THEN
                    Errmsg := 'ID Client tidak ditemukan di M_CLIENT : ' || Customer_ASMClientIDPEGA ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;
                
                BEGIN
                    SELECT substr(name, 0, 380) INTO VIRTUALACCOUNTNAME FROM POOLDATA.client WHERE ID = Customer_ASMClientIDPEGA;
                    IF TRIM(VIRTUALACCOUNTNAME) IS NULL THEN
                        Errmsg := 'Nama Client Tidak Boleh NULL, ClientID : ' || Customer_ASMClientIDPEGA ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
                    IF TRIM(VIRTUALACCOUNTNAME) = '-' THEN
                        Errmsg := 'Invalid Client Name: ' || VIRTUALACCOUNTNAME || ', ClientID: ' || Customer_ASMClientIDPEGA;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
                END;
                
                BEGIN
                    POOLDATA.GENERATE_MST_CLIENT(Customer_ASMClientIDPEGA,Quotation_BusinessCode,lkuid,sourcebiz);
--                if errmsg is not null then
--                    Errmsg := 'Error konversi client : ' || Errmsg;
--                    ROLLBACK;
--                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                    COMMIT;
--                    RETURN;
--                END IF;
                EXCEPTION WHEN OTHERS THEN
                    Errmsg := 'Error generate mst_client : ' || SQLERRM ||  dbms_utility.format_error_backtrace ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;
                
                SELECT oldid into clientoldid from m_client WHERE id = Customer_ASMClientIDPEGA;
                
                BEGIN
                    Customer_ASMClientID := POOLDATA.getnewID(Customer_ASMClientIDPEGA,'M_CLIENT','ID','kosong', 'kosong', 'OLDID');
                  --  Customer_ASMClientIDPEGAOLD := POOLDATA.getnewID(Customer_ASMClientIDPEGA,'M_CLIENT','ID','kosong', 'kosong', 'OLDID_PEGA');
                    
--                    if Customer_ASMClientIDPEGAOLD is not null then
 --                       update mst_client@asmd.sinarmas.co.id set mcl_id_pega = Customer_ASMClientIDPEGA WHERE mcl_id_pega = Customer_ASMClientIDPEGAOLD;
--                    END IF;
                    if Customer_ASMClientID is null then
                        BEGIN
                            SELECT mcl_id into Customer_ASMClientID from general.mst_client@asmd.sinarmas.co.id WHERE mcl_id_pega = Customer_ASMClientIDPEGA and rownum=1;
                            update m_client set oldid=Customer_ASMClientID WHERE id=Customer_ASMClientIDPEGA;
                        exception when no_data_found then
                            Errmsg :='Data Customer Client ID : ' || SQLERRM ;
                            Customer_ASMClientID := null;
                        END;                            
--                    else
--                        update mst_client@asmd.sinarmas.co.id set mcl_id_pega = Customer_ASMClientIDPEGA WHERE mcl_id = Customer_ASMClientID;
                    END IF;
                EXCEPTION WHEN OTHERS THEN
                    Errmsg :='Error update mst_client : ' || SQLERRM ||  dbms_utility.format_error_backtrace ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;
                
                if Customer_ASMClientID is not null then
                    mclid := Customer_ASMClientID;
                    --GENERAL.PKG_VIRTUAL_ACCOUNT.P_GENERATE_VA_SYR@asmd.sinarmas.co.id
                    --    (Quotation_BusinessCode, lkuid, sourcebiz, mclid, VIRTUALACCOUNTNO,VIRTUALACCOUNTUSDNO, VIRTUALACCOUNTNAME, errmsgva);
--                    BEGIN
--                    if Quotation_BusinessCode like 'S%' then
--                        SELECT no_va_syr, mcl_name into VIRTUALACCOUNTNO, VIRTUALACCOUNTNAME
--                            from mst_client@asmd.sinarmas.co.id WHERE mcl_id = mclid and rownum=1;
--                            
--                        if VIRTUALACCOUNTNO is null then
--                            cntVA := 1;
--                            WHILE  cntVA > 0 LOOP
--                                SELECT '87995'||SUBSTR(ABS(dbms_random.random)||'00000000000',1,11) INTO VIRTUALACCOUNTNO from dual;
--                                SELECT COUNT(1) INTO cntVA FROM MST_VIRTUAL_ACCOUNT@asmd.sinarmas.co.id WHERE CFRSVA = VIRTUALACCOUNTNO ;
--                            END LOOP;
--                        END IF;
--                    else
--                        SELECT no_va, '8144'||substr(no_va,5,12), mcl_name into VIRTUALACCOUNTNO, VIRTUALACCOUNTUSDNO, VIRTUALACCOUNTNAME
--                            from mst_client@asmd.sinarmas.co.id WHERE mcl_id = mclid and rownum=1;
--                            
--                        if VIRTUALACCOUNTNO is null then
--                            cntVA := 1;
--                            WHILE  cntVA > 0 LOOP
--                                SELECT '8005'||SUBSTR(ABS(dbms_random.random)||'00000000000',1,12) INTO VIRTUALACCOUNTNO from dual;
--                                SELECT COUNT(1) INTO cntVA FROM MST_VIRTUAL_ACCOUNT@asmd.sinarmas.co.id WHERE CFRSVA = VIRTUALACCOUNTNO;
--                            END LOOP;
--                            SELECT '8144' ||substr(VIRTUALACCOUNTNO,5,12) INTO VIRTUALACCOUNTUSDNO from dual;
--                        END IF;
--                    END IF;
--                    EXCEPTION WHEN OTHERS THEN
--                        Errmsg := 'Error on Customer gathering : Gagal Get Virtual Account : ' || SQLERRM ;
--                        RETURN;
--                    end ;
--
--                else
--                    SELECT COUNT(1) into cntConverted from POOLDATA.t_pool_policy WHERE CLIENTIDPEGA = Customer_ASMClientIDPEGA;
--                    if cntConverted >0 then
--                        SELECT clientid,nova,novausd into mclid,VIRTUALACCOUNTNO,VIRTUALACCOUNTUSDNO from POOLDATA.t_pool_policy WHERE CLIENTIDPEGA = Customer_ASMClientIDPEGA and rownum=1;
--                    else
--                        POOLDATA.PKG_COUNTER_PRODUCTION.GET_MCLID_SEQ(mclid);
--                         DBMS_OUTPUT.Put_Line('Mulai create client : ' ||  mclid);
--                        
--                        BEGIN 
--                        if Quotation_BusinessCode like 'S%' then
--                            cntVA := 1;
--                            WHILE  cntVA > 0 LOOP
--                                SELECT '87995'||SUBSTR(ABS(dbms_random.random)||'00000000000',1,11) INTO VIRTUALACCOUNTNO from dual;
--                                SELECT COUNT(1) INTO cntVA FROM MST_VIRTUAL_ACCOUNT@asmd.sinarmas.co.id WHERE CFRSVA = VIRTUALACCOUNTNO;
--                            END LOOP;
--                        else
--                            cntVA := 1;
--                            WHILE  cntVA > 0 LOOP
--                                SELECT '8005'||SUBSTR(ABS(dbms_random.random)||'00000000000',1,12) INTO VIRTUALACCOUNTNO from dual;
--                                SELECT COUNT(1) INTO cntVA FROM MST_VIRTUAL_ACCOUNT@asmd.sinarmas.co.id WHERE CFRSVA = VIRTUALACCOUNTNO;
--                            END LOOP;
--                            SELECT '8144' ||substr(VIRTUALACCOUNTNO,5,12) INTO VIRTUALACCOUNTUSDNO from dual;
--                        END IF;
--                        EXCEPTION WHEN OTHERS THEN
--                            Errmsg := 'Error on Customer gathering : VANO'|| SQLERRM ;
--                            DBMS_OUTPUT.Put_Line('Mulai VANO: ' ||  Errmsg);
--                            RETURN;
--                        end ;
--                    END IF;
                END IF; 

--                BEGIN 
--                    SELECT substr(name,0,180) into VIRTUALACCOUNTNAME from client WHERE id = Customer_ASMClientIDPEGA;
--                exception when no_data_found then
--                    Errmsg := 'Error on Customer gathering : Virtual Account Name : ' || SQLERRM ||dbms_utility.format_error_backtrace;
--                    ROLLBACK;
--                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                    COMMIT;
--                    RETURN;
--                end ;
                
                BEGIN
                    --IF (Quotation_SourceOfBusiness <> '10034413' AND Quotation_BusinessCode <> '03') THEN --TKA Tidak generate VA BSM
                        GENERAL.PKG_VIRTUAL_ACCOUNT.P_GENERATE_VA_SYR@ASMD.SINARMAS.CO.ID(
                         Quotation_BusinessCode, lkuId, sourcebiz, mclid, VIRTUALACCOUNTNO, VIRTUALACCOUNTUSDNO, VIRTUALACCOUNTNAME, Errmsg);
                         
                        IF Quotation_SourceOfBusiness = '10034413' AND Quotation_BusinessCode = '03' THEN
                            VIRTUALACCOUNTNO := NULL;
                            VIRTUALACCOUNTUSDNO := NULL;
                        END IF;
                       
--                    if VIRTUALACCOUNTNO is not null and VIRTUALACCOUNTNO like '8005%' then
--                        SELECT COUNT(1) into cntVa from general.mst_virtual_account@asmd.sinarmas.co.id WHERE cfrsva = VIRTUALACCOUNTNO;
--                        if cntVa = 0 then
--                            Errmsg := 'Error Generate VA , nomor virtual account invalid '|| Errmsg ;
--                            ROLLBACK;
--                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                            COMMIT;
--                            RETURN;
--                        END IF;
--                    END IF;
--                        
--                    if VIRTUALACCOUNTNAME is null and VIRTUALACCOUNTNO is not null then
--                        Errmsg := 'Error Generate VA : Virtual Account Name NULL ' ;
--                        ROLLBACK;
--                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                        COMMIT;
--                        RETURN;
--                    END IF;
                        IF (Quotation_SourceOfBusiness = '10001329' OR Quotation_SobLeader0 = '10000173') AND VIRTUALACCOUNTNO IS NULL THEN
                            Errmsg := 'Error Generate VA : VA NULL' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;
                    --END IF;
                EXCEPTION WHEN OTHERS THEN
                    Errmsg := 'Error Return Virtual Account : ' || SQLERRM || u'\000A' || dbms_utility.format_error_backtrace;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;    
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error on Customer gathering : ' || SQLERRM || u'\000A' || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            -- end customer gathering
            
            -- digit check
            BEGIN
                IF vvIDPEGA LIKE 'ASM-FW-GISFW-WORK EDM-%' THEN
                    BEGIN
                        SELECT T_POOL_POLICY.DIGITCHECKSUM INTO DIGITCHECKSUM2 FROM T_POOL_POLICY WHERE POLICYNO = Policy_PolicyNo AND ROWNUM = 1;
                    EXCEPTION WHEN NO_DATA_FOUND THEN
                        SELECT CEK_DIGIT_NEW INTO DIGITCHECKSUM2 FROM MST_DET_SALES@ASMD.SINARMAS.CO.ID WHERE MDS_NO_POLIS = Policy_PolicyNo AND LJP_ID IN('1','2');
                    END;
                ELSE
                    cntDigitPolis := 1;
                    WHILE cntDigitPolis > 0 LOOP
                        SELECT POOLDATA.CEK_DIGIT_POLIS_2(Policy_PolicyNo) INTO DIGITCHECKSUM2 FROM DUAL;
                        SELECT COUNT(0) INTO cntDigitPolis FROM T_POOL_POLICY WHERE T_POOL_POLICY.DIGITCHECKSUM = DIGITCHECKSUM2;
                    END LOOP;
                END IF;
                DIGITCHECKSUM := DIGITCHECKSUM2;
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error cek digit polis : ' || SQLERRM || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            
            -- Payment Gathering
            BEGIN
                BEGIN
                    l_Payment_Obj := l_jsonObject.Get_object('Payment') ;
                EXCEPTION WHEN OTHERS THEN
                    Errmsg := 'Error on Payment Gathering : Object Policy.Payment tidak ditemukan / Invalid : ' || SQLERRM ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;
                                
                Payment_FlagMaterai :=l_Payment_Obj.Get_string('FlagMaterai');  
                if Payment_FlagMaterai ='true' then
                    Payment_FlagMaterai := 1;
                else
                    Payment_FlagMaterai := 0;
                END IF;  
                Payment_Premium := NVL(l_Payment_Obj.Get_number('Premium'),0);
                Payment_AdminFee := NVL(l_Payment_Obj.Get_number('AdminFee'),0);
                Payment_Commision := NVL(l_Payment_Obj.Get_number('Commision'),0);
                Payment_Diskon := NVL(l_Payment_Obj.Get_number('Diskon'),0);
                Payment_OC := NVL(l_Payment_Obj.Get_number('OC'),0);
                Payment_StampPolicy := NVL(l_Payment_Obj.Get_number('StampPolicy'),0);
                Payment_StampReceipts := NVL(l_Payment_Obj.Get_number('StampReceipts'),0);
                Payment_TotalStamp := Payment_StampReceipts + Payment_StampPolicy;
                if Payment_Premium = 0 and vvIDPEGA like 'ASM-FW-GISFW-WORK EDM-%' and Quotation_EdmType = '3' and 
                 vvIDPEGA not in ('ASM-FW-GISFW-WORK EDM-1137464','ASM-FW-GISFW-WORK EDM-1137470') then
                    Errmsg := 'Edm Pengaktifan, Nilai Premi tidak boleh 0' ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                
                if Payment_TotalStamp > 0 and (Policy_CaseID like 'ASM-FW-GISFW-WORK NB-%' or Policy_CaseID like 'ASM-FW-GISFW-WORK RNW-%') 
                    AND Quotation_SobLeader0 IN ('10000174','10000971','10018096')
                    and Policy_TypeOfCoins <> '1' then    
                    if LkuId = '01' and Payment_TotalStamp < 10000 and Payment_Premium > 5000000 then
                        Errmsg := 'Nilai biaya materai :' || Payment_TotalStamp || ' kurang dari ketentuan (10.000)' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
--                    if LkuId <> '01' and (Payment_StampReceipts + Payment_StampPolicy)*Policy_CurrencyValue < 9500 and Payment_Premium*Policy_CurrencyValue > 5000000 then
--                        Errmsg := 'Nilai biaya materai kurang dari ketentuan' ;
--                        ROLLBACK;
--                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                        COMMIT;
--                        RETURN;    
--                    END IF;
                END IF;
                
                -- Installment Gathering
                BEGIN
                    Begin
                        l_Installment_arr := l_Payment_Obj.Get_array('ListInstallment');
                    EXCEPTION WHEN OTHERS THEN
                        Errmsg := 'Error on Installment Gathering : Object Payment.ListInstallment tidak ditemukan / Invalid : ' || SQLERRM ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END;
                    indexInstallment := 0;
                    TotalInstallmentPercentage:=0;
                    SumInstallment_Premium := 0;
                    SumInstallment_OC := 0;
                    SumInstallment_Commission := 0;
                    SumInstallment_Discount := 0;
                    SumInstallment_AdminFee := 0;
                    SumInstallment_Stamp := 0;
                    SumPaymentTotal := 0;
                    IF (Quotation_EdmType IS NULL OR Quotation_EdmType <> '9') THEN
                    FOR e IN 0 .. l_Installment_arr.get_size - 1 LOOP
                        l_Installment_Obj := Json_object_t(l_Installment_arr.Get(indexInstallment));
                        Installment_DueDate := to_Date(Substr(l_Installment_Obj.Get_string('DueDate'),0,8),'YYYYMMdd');
                        if Installment_DueDate is null and Policy_IsDeclaration <> 'True' then
                            Errmsg := 'Data ListInstallment.DueDate Tidak boleh NULL ' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            Return ;
                        END IF;
                        Installment_InstallmentNo := l_Installment_Obj.Get_number('InstallmentNo');
                        if Installment_InstallmentNo is null then
                            Errmsg := 'Data ListInstallment.InstallmentNo Tidak boleh NULL' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            Return ;
                        END IF;
                        Installment_AdminFee := NVL(l_Installment_Obj.Get_number('AdminFee'),0);
                        Installment_Discount := NVL(l_Installment_Obj.Get_number('Discount'),0);
                        Installment_OC := NVL(l_Installment_Obj.Get_number('OC'),0);
                        Installment_Premium := NVL(l_Installment_Obj.Get_number('Premium'),0);
                        Installment_Stamp := NVL(l_Installment_Obj.Get_number('Stamp'),0);
                        Installment_InstallmentPercentage := l_Installment_Obj.Get_number('InstallmentPercentage');
                        if Installment_InstallmentPercentage is null then
                            Installment_InstallmentPercentage := 100;
                        END IF;
                        Installment_PaymentTotal := l_Installment_Obj.Get_number('PaymentTotal');
                        if Installment_PaymentTotal is null then
                            Errmsg :='Data ListInstallment.PaymentTotal Tidak boleh NULL' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            Return ;
                        END IF;
                        Installment_Commission := NVL(l_Installment_Obj.Get_number('Commission'),0);
                        
                        TotalInstallmentPercentage := TotalInstallmentPercentage + Installment_InstallmentPercentage;
                        SumPaymentTotal := SumPaymentTotal + Installment_PaymentTotal;
                        SumInstallment_AdminFee := SumInstallment_AdminFee + Installment_AdminFee;
                        SumInstallment_Discount := SumInstallment_Discount + Installment_Discount;
                        SumInstallment_Commission := SumInstallment_Commission +Installment_Commission;
                        SumInstallment_Premium := SumInstallment_Premium +Installment_Premium;
                        SumInstallment_OC := SumInstallment_OC + Installment_OC;
                        SumInstallment_Stamp := SumInstallment_Stamp + Installment_Stamp;
                        
                        ReceiptNo := null; -- generate receiptno / mdsnokwi
                        POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWI_SEQ(ReceiptNo);

                        Insert into POOLDATA.T_POOL_INSTALLMENT 
                            (   ReceiptNo, InstallmentNo, DueDate,
                                PolicyNo, CaseID, PaymentTotal, Comm
                            )
                            values
                            (   ReceiptNo, Installment_InstallmentNo,Installment_DueDate,
                                TRIM(Policy_PolicyNo),Policy_CaseID, Installment_PaymentTotal, Installment_Commission
                            );
                        indexInstallment := indexInstallment +1;
                    END LOOP;
                    END IF;
                    if TotalInstallmentPercentage-100 >0.1 and TotalInstallmentPercentage-100 <-0.1  and Quotation_SobLeader2 <>'10001872'then 
                        Errmsg := 'Data Total Installment Percentage Harus 100, Current : ' || TotalInstallmentPercentage ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    if FLOOR(Payment_AdminFee) <> FLOOR(SumInstallment_AdminFee) then
                        Errmsg := 'Nilai Total Installment.AdminFee <> Payment.AdminFee, Installment : ' || SumInstallment_AdminFee  || ', Payment : ' || Payment_AdminFee;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    if FLOOR(Payment_TotalStamp) <> FLOOR(SumInstallment_Stamp) then
                        Errmsg := 'Nilai Total Materai Installment <> Payment, Installment : ' || SumInstallment_Stamp  || ', Payment : ' || Payment_TotalStamp;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    if abs(Payment_Diskon - SumInstallment_Discount) > 1 then
                        Errmsg := 'Nilai Total Installment.Discount <> Payment.Diskon, Installment : ' || SumInstallment_Discount  || ', Payment : ' || Payment_Diskon;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    if abs(Payment_Commision - SumInstallment_Commission) > 1 then
                        Errmsg := 'Nilai Total Installment.Commision <> Payment.Commision, Installment : ' || SumInstallment_Commission  || ', Payment : ' || Payment_Commision;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    if abs(Payment_OC - SumInstallment_OC) > 1 then
                        Errmsg := 'Data Total Installment.OC <> Payment.OC, Installment : ' || SumInstallment_OC  || ', Payment : ' || Payment_OC;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                    if abs(Payment_Premium - SumInstallment_Premium) > 1 and Policy_TypeOfCoins <> 'F' then
                        Errmsg := 'Nilai Total Installment.Premium <> Payment.Premium, Installment : ' || SumInstallment_Premium  || ', Payment : ' || Payment_Premium;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        Return ;
                    END IF;
                EXCEPTION WHEN OTHERS THEN
                    Errmsg := 'Error on Installment gathering : '|| SQLERRM || dbms_utility.format_error_backtrace;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;
                -- End Installment Gathering
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error on Payment gathering : ' || SQLERRM || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            -- End Payment Gathering    
            
            -- CoinsList Gathering 
            BEGIN
            --1 in 2 out
                IF Policy_TypeOfCoins IN ('1','2') THEN
                    if l_jsonObject.Get_array('CoinsList') IS NOT NULL THEN
                        l_Coins_arr := l_jsonObject.Get_array('CoinsList');
                        indexCoins  := 0;
                        TotalPercentCoins   := 0;
                        TotalPremiumCoins   := 0;
                        TotalBrokerageCoins := 0;
                        TotalDiscountCoins  := 0;
                        TotalTSICoins       := 0;
                        cekLeaderCoas       := 0;
                        FOR h IN 0 .. l_Coins_arr.get_size - 1 LOOP
                            l_Coins_Obj :=Json_object_t(l_Coins_arr.Get(indexCoins));
                            
                            Coins_CoinsID := l_Coins_Obj.Get_string('CoinsID');
                            IF Coins_CoinsID IS NULL THEN     
                                Errmsg := 'Error on CoinsList gathering : CoinsID Tidak Boleh Kosong';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            --generate_lst_asuradur(Coins_CoinsID);
                            Coins_Leader := l_Coins_Obj.Get_string('Leader');
                            IF Coins_Leader IS NULL THEN    
                                Errmsg := 'Error on CoinsList gathering : Leader Tidak Boleh Kosong';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            Coins_PremiShare    := NVL(l_Coins_Obj.Get_number('PremiShare'),0);
                            Coins_Brokerage     := NVL(l_Coins_Obj.Get_number('Brokerage'),0);
                            IF Coins_Leader = 'true' THEN
                                Coins_Leader := 1;
                                cekLeaderCoas:= 1;
                            ELSE
                                Coins_Leader := 0;                            
                            END IF;
                            Coins_CoinsName     := l_Coins_Obj.Get_string('CoinsName');  
                            Coins_PercentShare  := l_Coins_Obj.Get_number('PercentShare');
                            if (Coins_PercentShare is null ) and Policy_TypeOfCoins <> '2' and Policy_CaseID not like 'ASM-FW-GISFW-WORK EDM-%' then
                                Errmsg := 'Error insert CoinsList : PercentShare Tidak Boleh Kosong . CoinsID : ' || Coins_CoinsID ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF; 
                            Coins_PercentPPN := l_Coins_Obj.Get_number('PercentPPN');
                            if Coins_PercentPPN is null then 
                                Coins_PercentPPN := 0;
                            END IF;
                            Coins_PercentPPN := Coins_PercentPPN/100;
                            Coins_PercentPPH := l_Coins_Obj.Get_number('PercentPPH');
                            if Coins_PercentPPH is null then
                                Coins_PercentPPH := 0;
                            END IF;
                            Coins_PercentPPH := Coins_PercentPPH/100;
                            Coins_PPH   := l_Coins_Obj.Get_number('PPH');
                            Coins_PPN   := l_Coins_Obj.Get_number('PPN');
                            Coins_FlagDelete := NVL(l_Coins_Obj.Get_string('FlagDelete'),0);
                            Coins_FlagOldData := NVL(l_Coins_Obj.Get_string('FlagOldData'),0);
                            INVOICENO := null;  
                            
                            IF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK EDM-%' THEN
                                IF l_Coins_Obj.Get_array('OldCoins') is not null and Coins_FlagOldData = '1' THEN
                                    l_OldCoins_arr  := l_Coins_Obj.Get_array('OldCoins');
                                    indexOldCoins   := 0;
                                        
                                    FOR p IN 0 .. l_OldCoins_arr.get_size - 1 LOOP
                                        l_OldCoins_Obj :=Json_object_t(l_OldCoins_arr.Get(indexOldCoins));
                                        OldCoins_CoinsID := l_OldCoins_Obj.Get_string('CoinsID');
                                        OldCoins_PremiShare := NVL(l_OldCoins_Obj.Get_number('PremiShare'),0);
                                        OldCoins_Brokerage := NVL(l_OldCoins_Obj.Get_number('Brokerage'),0);
--                                            OldCoins_PPH := NVL(l_OldCoins_Obj.Get_number('PPH'),0);
--                                            OldCoins_PPN := NVL(l_OldCoins_Obj.Get_number('PPN'),0);
                                        if OldCoins_CoinsID is null then     
                                            Errmsg := 'Error get data OldCoins :CoinsID Tidak Boleh Kosong, CoinsID :' || OldCoins_CoinsID;
                                            ROLLBACK;
                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                            COMMIT;
                                            RETURN;
                                        else
                                            if OldCoins_CoinsID <> Coins_CoinsID then
                                                Errmsg := 'Untuk mengubah member coas, harap hapus terlebih dahulu member yang diubah, kemudian input data coas yang baru ';
                                                ROLLBACK;
                                                RETURN;
                                            END IF;
                                        END IF;
                                        
                                        Coins_PremiShare := Coins_PremiShare - OldCoins_PremiShare;
                                        Coins_Brokerage := Coins_Brokerage - OldCoins_Brokerage;
--                                            Coins_PPH := Coins_PPH - OldCoins_PPH;
--                                            Coins_PPN := Coins_PPN - OldCoins_PPN;
--                                            
                                        if Quotation_EdmType = '2' then
                                            Coins_PremiShare    := Coins_PercentShare/100 * Payment_Premium;
                                            Coins_Brokerage     := Coins_PercentShare/100 * Payment_Commision;
--                                                Coins_PPH           := Coins_Brokerage * Coins_PercentPPH;
--                                                Coins_PPN           := Coins_Brokerage * Coins_PercentPPN;
                                        END IF;
                                        indexOldCoins := indexOldCoins+1;
                                    END LOOP;
                                END IF;                              
                            END IF;
                            
                            IF Policy_TypeOfCoins = 1 THEN
                                typeofcoins := '2';
                            ELSIF Policy_TypeOfCoins = 2 THEN
                                typeofcoins := '1';
                            ELSIF Policy_TypeOfCoins = 0 THEN
                                typeofcoins := '0';
                            END IF;
                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_INVNO_SEQ(Policy_PolicyNo, typeofcoins, INVOICENO);
                            
                            -- Kalau COUT itung PPH
                            --IF Policy_TypeOfCoins = 2 THEN
                                IF Coins_Brokerage <> 0 THEN
                                    BEGIN
                                        SELECT POOLDATA.getnewID(A.LEADER0,'M_AGENT','ID','kosong', 'kosong', 'OLDID'),
                                               POOLDATA.getnewID(A.LEADER1,'M_AGENT','ID','kosong', 'kosong', 'OLDID')
                                         INTO vOldLeader0, vOldLeader1 
                                        FROM POOLDATA.T_AGENT A WHERE A.OLDID = sourcebiz AND A.LEADER1 IS NOT NULL;
                                    EXCEPTION WHEN NO_DATA_FOUND THEN
                                        vOldLeader0 := '00000';
                                        vOldLeader1 := '00000';
                                    END;
                                    
                                    SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_GET_COMM_GROSS2@ASMD.sinarmas.co.id(Coins_Brokerage, sourcebiz, vOldLeader0, vOldLeader1),0) 
                                      INTO Coins_BrokerageDPP 
                                      FROM DUAL;
                                END IF;
                                
                                BEGIN
                                    COLLECTION.p_get_pph_new@asmd.sinarmas.co.id(Coins_Brokerage, lkuid, sourcebiz, null, lppid, pctpphcoas, pphnotecoas, Errmsg);
                                    IF pctpphcoas IS NULL THEN
                                        pctpphcoas := 0;
                                    END IF;
                                    Coins_PercentPPH := pctpphcoas;
                                    IF pctpphcoas = 0.05 OR pctpphcoas = 0.06 OR lppid = 'PPH21' THEN
                                        pctpphcoas := pctpphcoas * 0.5;
                                    END IF;
                                EXCEPTION WHEN OTHERS THEN 
                                    Errmsg := 'Error on CoinsList gathering : Get PPH Coas : ' || SQLERRM;
                                    ROLLBACK;
                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                    COMMIT;
                                    RETURN;
                                END;
                                
                                BEGIN
                                    pctppncoas := COLLECTION.f_get_ppn_booking@asmd.sinarmas.co.id(sourcebiz);
                                    IF pctppncoas IS NULL THEN
                                        pctppncoas := 0;
                                    END IF;
                                    
                                    Coins_PercentPPN := pctppncoas;   
                                EXCEPTION WHEN OTHERS THEN 
                                    Errmsg := 'Error on CoinsList gathering : Get PPN Coas : ' || SQLERRM;
                                    ROLLBACK;
                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                    COMMIT;
                                    RETURN;
                                END;
--                            ELSIF Policy_TypeOfCoins = '1' THEN
--                                IF Quotation_SourceOfBusiness = '10000952' THEN
--                                    pctppncoas := 0.022;
--                                    Coins_PercentPPN := pctppncoas;   
--                                END IF;
--                            END IF;
                            
                            IF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK RNW-%' AND Coins_FlagDelete = '1' THEN
                                indexCoins := indexCoins+1;
                                CONTINUE;
                            END IF;
                            TotalPremiumCoins   := TotalPremiumCoins + Coins_PremiShare;
                            TotalBrokerageCoins := TotalBrokerageCoins + Coins_Brokerage;
                            
--                            IF cekLeaderCoas <> 1 THEN
--                                Errmsg := 'Data Leader:True di CoinsList tidak ditemukan';
--                                ROLLBACK;
--                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                                COMMIT;
--                                RETURN;
--                            END IF;
                                
                            BEGIN
                                IF (Coins_CoinsID = '10038996') THEN
                                    Coins_PercentPPH := 0;
                                    Coins_PPH := 0;
                                END IF; 
                                IF Policy_TypeOfCoins = '2' OR Quotation_SourceOfBusiness = '10000952' THEN
                                    Coins_PPH := NVL(Coins_BrokerageDPP*pctpphcoas,0);
                                    --Coins_PPN := NVL(NVL(l_Coins_Obj.Get_number('Brokerage'),0)*pctppncoas,0);
                                    SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_HITUNG_PPN_NEW@ASMD.sinarmas.co.id(Coins_Brokerage, sourcebiz, vOldLeader0, vOldLeader1),0) 
                                      INTO Coins_PPN FROM DUAL;
                                END IF;
                                
                                INSERT INTO POOLDATA.T_POOL_COINS
                                    (   COINSID, COINSNAME, INVOICENO, PERCENTPPH, PERCENTPPN,
                                        POLICYNO, CASEID, LEADER, PPH, PPN, PREMISHARE, BROKERAGE, FLAGDELETE,
                                        BROKERAGEDPP
                                    )
                                VALUES
                                    (   Coins_CoinsID , Coins_CoinsName , INVOICENO, Coins_PercentPPH, Coins_PercentPPN, 
                                        TRIM(Policy_PolicyNo) , Policy_CaseID , Coins_Leader, Coins_PPH, Coins_PPN, Coins_PremiShare, Coins_Brokerage, Coins_FlagDelete,
                                        Coins_BrokerageDPP
                                    );
                                indexCoins := indexCoins+1;
                            EXCEPTION WHEN OTHERS THEN 
                                Errmsg := 'Error on Coinslist insert : ' || SQLERRM || dbms_utility.format_error_backtrace;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END;
                        END LOOP;
                        
                        --IF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK NB-%' OR Policy_CaseID LIKE 'ASM-FW-GISFW-WORK RNW-%' THEN
                            --dbms_output.put_line('TotalPremiumCoins'||TotalPremiumCoins||'Payment_Premium'||Payment_Premium);
                            --dbms_output.put_line('TotalBrokerageCoins'||TotalBrokerageCoins||'Payment_Commision'||Payment_Commision);
                            IF ABS(TotalPremiumCoins - Payment_Premium) > 100 THEN
                                Errmsg := 'Error Cek CoinsList : Nilai Total Premi Share :' || TotalPremiumCoins || ' tidak sama dengan nilai Payment.Premium :'|| Payment_Premium;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
--                            if abs(TotalBrokerageCoins - (Payment_Commision + Payment_OC)) > 100 then
--                                Errmsg := 'Error Cek CoinsList : Nilai Total Brokerage :' || TotalBrokerageCoins || ' tidak sama dengan nilai Payment.Commision :'|| Payment_Commision;
--                                ROLLBACK;
--                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                                COMMIT;
--                                RETURN;
--                            END IF;
                        --END IF;
                        
                        SELECT COUNT(1) INTO cekLeaderCoas
                          FROM POOLDATA.T_POOL_COINS WHERE POLICYNO = vpolicynocek AND CASEID = vvIDPEGA AND LEADER = '1';
                        IF cekLeaderCoas = 0 THEN
                            Errmsg := 'Data Leader:True di CoinsList tidak ditemukan';
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;    
                        
                    ELSE 
                        Errmsg := 'Error on Coinslist gathering : Data Coas Tidak Ditemukan / Null' ;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
                END IF;
            Exception WHEN OTHERS THEN
                Errmsg := 'Error on Coinslist gathering : ' || SQLERRM || dbms_utility.format_error_backtrace ||' idx: '||indexCoins;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            -- End CoinsList Gathering          
            
            DBMS_OUTPUT.Put_Line('Mulai Object ' || CURRENT_TIMESTAMP);
            BEGIN 
                BEGIN 
                    indexobject :=0;
                    if Quotation_GroupPanel in ('007') then -- MBU VehicleList
                        SELECT COUNT(1) into cntObjList from POOLDATA.t_vehiclelist WHERE idpega = vvIDPEGA;
                        if cntObjList = 0 then
                            Errmsg := 'Id Pega '|| vvIDPEGA || ' Tidak ditemukan di T_VEHICLELIST' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;
                        
                        SELECT max(indexobject) into vJumlahObject from POOLDATA.t_vehiclelist WHERE idpega = vvIDPEGA;
                        for var_vehicle_obj in curr_vehicle_obj (vvIDPEGA) loop    
                            if var_vehicle_obj.vehiclelist is null then
                                Errmsg := 'Error Get Data T_VEHICLELIST : VehicleList tidak boleh null' ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;                    
                            l_Object_Obj := Json_object_t(var_vehicle_obj.vehiclelist);
                            if l_Object_obj is null then
                                Errmsg := 'Error Get Data Object : Object data tidak ada : VehicleList' ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            IF var_vehicle_obj.coveragelist IS NULL THEN
                                Errmsg := 'CoverageList di T_VEHICLELIST NULL';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            --Loop masing2 object
                            l_coverage_arr := Json_object_t(var_vehicle_obj.coveragelist).Get_array('CoverageList');

                            Coverage_PremiTotal     :=0;
                            Coverage_DiscountTotal  :=0;
                            Coverage_OCTotal        :=0;
                            Coverage_CommTotal      :=0;
                            Coverage_OutgoTotal     :=0;
                            DELETE POOLDATA.CoverageYearly WHERE caseID = Policy_CaseID;
                            DELETE POOLDATA.OutgoYearly WHERE caseID = Policy_CaseID;
                            indexCoverage :=0;
                            
                            FOR b IN 0 .. l_coverage_arr.get_size - 1 LOOP
                                BEGIN 
                                    Coverage_PremiYearly := 0;
                                    Coverage_DiscountYearly := 0;
                                    Coverage_OutgoYearly := 0;
                                    Coverage_CommYearly := 0;
                                    Coverage_OCYearly := 0;
                                    l_coverage_Obj := Json_object_t(l_coverage_arr.Get(indexCoverage));
                                    
                                    Coverage_BeginDate := TO_DATE(SUBSTR(l_coverage_Obj.Get_string('BeginDate'),0,8),'YYYYMMdd');
                                    Coverage_EndDate := TO_DATE(SUBSTR(l_coverage_Obj.Get_string('EndDate'),0,8),'YYYYMMdd');
                                    Coverage_PremiYearly := Coverage_PremiYearly + l_coverage_Obj.Get_number('Premium');
                                    Coverage_DiscountYearly := Coverage_DiscountYearly + l_coverage_Obj.Get_number('Discount');
                                    -- Begin Gathering Outgo 
                                    IF l_coverage_Obj.Get_array('OutGoList') IS NOT NULL THEN
                                        BEGIN
                                            l_Outgo_arr := l_coverage_Obj.Get_array('OutGoList');
                                            indexOutgo :=0;
                                            FOR d IN 0 .. l_Outgo_arr.get_size - 1 LOOP
                                                l_Outgo_Obj := Json_object_t(l_Outgo_arr.Get(indexOutgo));
                                                Coverage_OutgoYearly := Coverage_OutgoYearly + l_Outgo_Obj.Get_number('OutgoAmount');
                                                Outgo_OutgoAmount := l_Outgo_Obj.Get_number('OutgoAmount');
                                                if Outgo_OutgoAmount IS NULL THEN
                                                    Outgo_OutgoAmount :=0;
                                                END IF;
                                                Outgo_PercentOutgo := NVL(l_Outgo_Obj.Get_number('PercentOutgo'),0);
                                                Outgo_TypeOfOutgo := l_Outgo_Obj.Get_string('TypeOfOutgo');
                                                Outgo_ReceiverOutgo := l_Outgo_Obj.Get_string('ReceiverOutgo');
                                                Outgo_SourceOfBusiness :=  l_Outgo_Obj.Get_string('SourceOfBusiness');
                                                Outgo_OCID := l_Outgo_Obj.Get_string('OCID');
                                                if Outgo_OCID IS NULL THEN
                                                    Outgo_OCID := Outgo_SourceOfBusiness;
                                                    if Outgo_OCID IS NULL AND Payment_Commision <> 0 THEN
                                                        Errmsg := 'OCID di OutGoList Tidak ditemukan.' ;
                                                        ROLLBACK;
                                                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                        COMMIT;
                                                        RETURN;
                                                    END IF;
                                                    
                                                    SELECT COUNT(0) INTO cekAgent FROM m_agent WHERE id = Outgo_OCID;
                                                    IF cekAgent = 0 AND Payment_Commision <> 0 THEN
                                                        SELECT COUNT(0) into cekAgent FROM m_oc WHERE id = Outgo_OCID;
                                                        if cekAgent = 0 then
                                                            Errmsg := 'OCID tidak ditemukan di M_AGENT, OutGoList.OCID:'||Outgo_OCID;
                                                            ROLLBACK;
                                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                            COMMIT;
                                                            RETURN; 
                                                        END IF;
                                                    END IF;  
                                                END IF;
                                                
                                                Outgo_SourceBizName := l_Outgo_Obj.Get_string('SourceBizName');
                                                Outgo_FlagDelete :=  NVL(l_Outgo_Obj.Get_string('FlagDelete'),0);
                                                Outgo_FlagOldData := NVL(l_Outgo_Obj.Get_string('FlagOldData'),0);
                                                Outgo_FlagEditData := NVL(l_Outgo_Obj.Get_string('FlagEditData'),0);
                                                Outgo_SourceOfBusiness := POOLDATA.getnewID(Outgo_SourceOfBusiness,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                IF Policy_CaseID IN ( 'ASM-FW-GISFW-WORK EDM-1138600X','ASM-FW-GISFW-WORK EDM-1138597X') then  
                                                    Outgo_OutgoAmount :=Outgo_OutgoAmount*-1;
                                                    Outgo_FlagOldData := '0';
                                                END IF;
                                                
                                                IF(Outgo_TypeOfOutgo <> '9') THEN
                                                    tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                    IF(tempocid=Outgo_OCID) THEN
                                                        IF Outgo_TypeOfOutgo in ('2','A','B','C','D','E') THEN
                                                            SELECT COUNT(0) INTO cntOC FROM M_OC WHERE ID = tempocid;
                                                            IF cntOC = 0 THEN
                                                                Errmsg := 'OCID tidak ditemukan di table M_OC : ' || tempocid;
                                                                ROLLBACK;
                                                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                                COMMIT;
                                                                RETURN;
                                                            END IF;
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                            IF tempocid IS NULL AND LENGTH(Outgo_OCID)>8 THEN
                                                                GENERATE_LST_OC ( Outgo_OCID, ERRMSG );
                                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                            END IF;
                                                        ELSIF Outgo_TypeOfOutgo in ('4','5','1') THEN
                                                            IF length (TRIM(Outgo_OCID))>8 THEN
                                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                            ELSE
                                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                            END IF;
                                                        END IF;
                                                    END IF;
                                                    
                                                    -- OldOutgo Gathering
                                                    if vISb2b is null then
                                                        vISb2b := 0;
                                                    END IF;
                                                    if Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%'  then
                                                        --if Quotation_GroupPanel in ('002','005','007') and vISb2b = 0 then
                                                            if Outgo_FlagEditData ='1' then
                                                                l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                                indexOldOutgo := 0;
                                                                        
                                                                FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                    l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                    Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                    indexOldOutgo := indexOldOutgo+1;
                                                                END LOOP;
                                                            else
                                                                if Outgo_FlagOldData = '1' then
                                                                    Outgo_OutgoAmount := 0;
                                                                --else
                                                                END IF;
                                                            END IF;
                                                        --else
                                                            --if Outgo_FlagDelete = '0' then
--                                                                if  l_Outgo_Obj.Get_array('OldOutgo') is not null  and Outgo_FlagOldData ='1'then
--                                                                    l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
--                                                                    indexOldOutgo := 0;
--                                                                        
--                                                                    FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
--                                                                        l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
--                                                                        Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);
--
--                                                                        indexOldOutgo := indexOldOutgo+1;
--                                                                    END LOOP;
--                                                                END IF;
                                                            --else
                                                                --Outgo_OutgoAmount := Outgo_OutgoAmount * -1;                                            
                                                            --END IF;                                        
                                                        --END IF;
                                                    END IF;
                                                    -- End OldOutgo Gathering
                                                    
                                                    if Outgo_TypeOfOutgo not  in ('1','4','A','B','C','D','E') then 
                                                        Errmsg :='Error gathering outgo , Type Outgo salah : ' ||  Outgo_TypeOfOutgo || 'Receiver : ' || Outgo_ReceiverOutgo;
                                                        RETURN;
                                                    else
                                                        if Outgo_TypeOfOutgo in ('A','B','C','D','E')  then
                                                            Coverage_OCYearly := Coverage_OCYearly + Outgo_OutgoAmount;
                                                        else
                                                            Coverage_CommYearly := Coverage_CommYearly + Outgo_OutgoAmount;
                                                        END IF;
                                                    END IF;
                                                    
                                                    COMMKWIID := 0; -- generate COMMKWIID
                                                    POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                                    
                                                    SLIPNO := 0;    -- generate slipno / mnknokwikom
                                                    POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                                    
                                                    --DBMS_OUTPUT.Put_Line('param pph OCID: ' || tempocid );
                                                    COLLECTION.p_get_pph_new@asmd.sinarmas.co.id(Outgo_OutgoAmount, lkuid, tempocid, outgo_sourcebizname, lppid, pctpph, pphnote, Errmsg); 
                                                    
                                                    if pctpph is null then 
                                                        pctpph := 0;
                                                    END IF;
                                                    
                                                    -- 01-02-2021 OTO/SOF Refund komisi PPH 0 (Group 73) 
                                                    if Quotation_SobLeader2 in ('10003883',     --OTO MULTIARTHA
                                                                                '10007505',     --SUMMIT OTO FINANCE, PT.
                                                                                '10008129',     --OTO MULTIARTHA AFFINITY
                                                                                '10013958',     --OTO SOF ASSET
                                                                                '10055652',     --SUMMIT OTO FINANCE RENEWAL
                                                                                '10055999')     --OTO MULTIARTHA  RENEWAL
                                                    and Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%' and Payment_Commision < 0 then
                                                        pctpph := 0;
                                                    END IF;
                                                    
                                                    cntpctpph := 0;
                                                    IF pctpph = 0.05 or pctpph = 0.06 THEN
                                                        cntpctpph := pctpph * 0.5;
                                                    ELSE
                                                        cntpctpph := pctpph;
                                                    END IF; 
                                       
                                                    SELECT COLLECTION.f_get_ppn_booking@asmd.sinarmas.co.id(tempocid) INTO pctppn FROM DUAL;
                                                    
                                                    if pctppn is null then
                                                        pctppn := 0;
                                                    END IF;
                                                                                                        
                                                    outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                    outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                    outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                    
                                                    if indexobject = 0 and indexCoverage = 0 and indexOutgo=0 then
                                                        INSERT INTO POOLDATA.t_pool_outgo(ocidold, slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete,pphnote) values
                                                        (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete,pphnote);
                                                    else
                                                        SELECT COUNT(1) into cntTypeOutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and FlagDelete <> '1';
                                                        if cntTypeOutgo > 0  then
                                                            SELECT ocid into cekExistsOCID from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and rownum = 1;
                                                            if cekExistsOCID <> Outgo_OCID and Outgo_FlagDelete <> '1' then
                                                                Errmsg := 'Error insert OutGo OCID '|| Outgo_OCID || ', TypeOfOutgo '|| Outgo_TypeOfOutgo || ' sudah ada dengan OCID : '|| cekExistsOCID;
                                                                ROLLBACK;
                                                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                                COMMIT;
                                                                RETURN;
                                                            END IF;
                                                        END IF;
                                                        SELECT COUNT(1) into cntoutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(Outgo_OCID);
                                                        IF cntoutgo = 0 then
                                                            INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno, pphnote) values
                                                            (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',pphnote);
                                                        ELSE       
                                                            update POOLDATA.t_pool_outgo set
                                                            OutgoAmount = OutgoAmount + Outgo_OutgoAmount,
                                                            pphamount = pphamount + outgo_pphamount,
                                                            ppnamount = ppnamount + outgo_ppnamount,
                                                            totalcomm = totalcomm + outgo_totalcomm
                                                            WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(outgo_ocid);
                                                        END IF;  
                                                    END IF;
                                                END IF;
                                                                                    
                                                indexOutgo := indexOutgo+1;
                                            END LOOP;
                                        EXCEPTION WHEN OTHERS THEN
                                            Errmsg := 'Error gathering OutGoList : ' || SQLERRM  || dbms_utility.format_error_backtrace;
                                            ROLLBACK;
                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                            COMMIT;
                                            RETURN;
                                        END;
                                    END IF;
                                    -- End Gathering Outgo
                                                                        
                                    -- loop additionalCoverage
                                    if l_coverage_Obj.Get_array('AdditionalCoverage') is not null then
                                        Begin 
                                            indexAddCoverage :=0;
                                            l_addCoverage_arr := l_coverage_Obj.Get_array('AdditionalCoverage');
                                            FOR c IN 0 .. l_addCoverage_arr.get_size - 1 LOOP
                                                l_addCoverage_Obj := Json_object_t(l_addCoverage_arr.Get(indexAddCoverage));
                                                Coverage_PremiYearly := Coverage_PremiYearly + l_addCoverage_Obj.Get_number('Premium');
                                                Coverage_DiscountYearly := Coverage_DiscountYearly + l_addCoverage_Obj.Get_number('Discount');
                                                indexAddCoverage := indexAddCoverage+1;
                                            END LOOP;
                                        EXCEPTION WHEN OTHERS THEN
                                            Errmsg := 'Error Gathering AdditionalCoverage : ' || SQLERRM ;
                                            ROLLBACK;
                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                            COMMIT;
                                            RETURN;
                                        end ;
                                    END IF;
                                    Coverage_PremiTotal := Coverage_PremiTotal + Coverage_PremiYearly;
                                    Coverage_DiscountTotal := Coverage_DiscountTotal + Coverage_DiscountYearly;
                                    Coverage_OutgoTotal := Coverage_OutgoTotal + Coverage_OutgoYearly;
                                    Coverage_CommTotal := Coverage_CommTotal + Coverage_CommYearly;
                                    Coverage_OCTotal := Coverage_OCTotal + Coverage_OCYearly;
                                    --DBMS_OUTPUT.Put_Line(vTotalYear || indexobject);
                                    IF vTotalYear >= 1 and Quotation_GroupPanel in ('007') THEN --  ganti grouppanel XXX dengan kode MBU 
                                        IF indexobject = 0 THEN
                                            IF Coverage_BeginDate > Policy_ProdDateTime THEN
                                                INSERT INTO POOLDATA.CoverageYearly (CaseID,proddate, BEGINDATE, ENDDATE , CurrentYear, Premium ,Discount, Outgo , Comm , OC) values 
                                                    (Policy_CaseID,Coverage_BeginDate, Coverage_BeginDate, Coverage_EndDate,indexCoverage +1, NVL(Coverage_PremiYearly,0),Coverage_DiscountYearly,Coverage_OutgoYearly ,Coverage_CommYearly,Coverage_OCYearly);
                                            ELSE
                                                INSERT INTO POOLDATA.CoverageYearly (CaseID,proddate, BEGINDATE, ENDDATE , CurrentYear, Premium ,Discount, Outgo , Comm , OC) values 
                                                    (Policy_CaseID,Policy_ProdDateTime, Coverage_BeginDate, Coverage_EndDate,indexCoverage +1, NVL(Coverage_PremiYearly,0),Coverage_DiscountYearly,Coverage_OutgoYearly ,Coverage_CommYearly,Coverage_OCYearly);
                                            END IF;
                                        ELSE
                                            update  POOLDATA.CoverageYearly set 
                                            Premium = Premium + Coverage_PremiYearly, 
                                            Discount = Discount + Coverage_DiscountYearly, 
                                            Outgo = Outgo + Coverage_OutgoYearly, 
                                            Comm = Comm + Coverage_CommYearly, 
                                            OC = OC + Coverage_OCYearly
                                            WHERE CaseID = Policy_CaseID And CurrentYear= indexCoverage +1;
                                        END IF;
                                    END IF;
                                    indexCoverage := indexCoverage+1;
                                EXCEPTION WHEN OTHERS THEN
                                    Errmsg := 'Error Gathering Coverage : ' || SQLERRM ;
                                    ROLLBACK;
                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                    COMMIT;
                                    RETURN;
                                end ;
                            END LOOP;
                            indexobject:=indexobject+1;
                        END LOOP;
                        
                        --cek installment outgo > 1
                        BEGIN
                            if Policy_TypeOfCoins <> '1' then
                            DBMS_OUTPUT.Put_Line('Object MBU 3' || CURRENT_TIMESTAMP);
                                IF l_Installment_arr.get_size > 1 THEN
                                    flagDeleteOutgo := 0;
                                    FOR var_outgo IN curr_pool_outgo(Policy_CaseID, Policy_PolicyNo) loop
                                        tempocid               := var_outgo.ocidold;
                                        Outgo_OCID             := var_outgo.ocid;
                                        Outgo_SourceBizName := var_outgo.sourcebizname;
                                        pctppn                := var_outgo.ppnpercent;
                                        pctpph                 := var_outgo.pphpercent;
                                        lppid                 := var_outgo.pphtype;
                                        Outgo_TypeOfOutgo     := var_outgo.typeofoutgo;
                                        Outgo_ReceiverOutgo := var_outgo.receiveroutgo;
                                        Outgo_PercentOutgo  := var_outgo.percentoutgo;
                                        Outgo_OutgoAmountv  := var_outgo.outgoamount;
                                        pphnote             := var_outgo.pphnote;
                                                            
                                        if flagDeleteOutgo = 0 then
                                            DELETE FROM POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo);
                                            flagDeleteOutgo := 1;
                                        END IF;
                                                            
                                        --DBMS_OUTPUT.Put_Line('Proses ulang installment : ' || l_Installment_arr.get_size);
                                        indexInstallment := 0;
                                        FOR e IN 0 .. l_Installment_arr.get_size - 1 LOOP
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                            l_Installment_Obj := Json_object_t(l_Installment_arr.Get(indexInstallment));
                                            Installment_InstallmentPercentage := l_Installment_Obj.Get_number('InstallmentPercentage');
                                            Installment_Commission := l_Installment_Obj.Get_number('Commission');
                                            Installment_InstallmentNo := l_Installment_Obj.Get_number('InstallmentNo');
                                            indexInstallment := indexInstallment +1;
                                                                
                                            if Quotation_GroupPanel in ('007') then
                                                Outgo_OutgoAmount := NVL(Installment_Commission,0);
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                    
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            else
                                                Outgo_OutgoAmount := NVL(Outgo_OutgoAmountv,0) * NVL(Installment_InstallmentPercentage,0)/100;
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                    
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            END IF;         
                                        END LOOP;            
                                    END LOOP;
                                END IF; 
                            END IF;                               
                        Exception when others then
                            Errmsg := 'Error insert Outgo Installment > 1 : ' || SQLERRM;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;   
                        END;                                                    
--                    --end cek installment outgo >1 
                    elsif Quotation_GroupPanel in ('004') then --object MARINE
                        SELECT COUNT(1) into vJumlahObject from POOLDATA.t_cargolist WHERE idpega = vvIDPEGA;
                        if vJumlahObject = 0 then
                            Errmsg := 'IdPega '|| vvIDPEGA || ' Tidak ditemukan di T_CARGOLIST' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;
                        for var_cargo_obj in curr_cargo_obj (vvIDPEGA) loop
                            l_Object_Obj := Json_object_t(var_cargo_obj.objectdata);
                            if l_Object_obj is null then
                                Errmsg := 'Error Get Data Object : Object data tidak ada : CargoList' ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            if var_cargo_obj.coveragedata is null then
                                Errmsg := 'CoverageData di T_CARGOLIST NULL';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            --Loop masing2 object
                            l_coverage_arr := Json_object_t(var_cargo_obj.coveragedata).Get_array('CoverageList');

                            Coverage_PremiTotal :=0;
                            Coverage_DiscountTotal :=0;
                            Coverage_OCTotal :=0;
                            Coverage_CommTotal :=0;
                            Coverage_OutgoTotal :=0;
                            delete POOLDATA.CoverageYearly WHERE caseID = Policy_CaseID;
                            delete POOLDATA.OutgoYearly WHERE caseID = Policy_CaseID;
                            indexCoverage :=0;
                            
                            FOR b IN 0 .. l_coverage_arr.get_size - 1 LOOP
                            
                                Coverage_PremiYearly := 0;
                                Coverage_DiscountYearly := 0;
                                Coverage_OutgoYearly := 0;
                                Coverage_CommYearly := 0;
                                Coverage_OCYearly := 0;
                                l_coverage_Obj := Json_object_t(l_coverage_arr.Get(indexCoverage));
                                
                                Coverage_BeginDate:=to_Date(Substr(l_coverage_Obj.Get_string('BeginDate'),0,8),'YYYYMMdd');
                                Coverage_EndDate:=to_Date(Substr(l_coverage_Obj.Get_string('EndDate'),0,8),'YYYYMMdd');
                                Coverage_PremiYearly := Coverage_PremiYearly + NVL(l_coverage_Obj.Get_number('Premium'),0);--((case when Coverage_FlagDelete= '1' then 1 else 1 end ) * ) 
                                Coverage_DiscountYearly := Coverage_DiscountYearly + NVL(l_coverage_Obj.Get_number('Discount'),0); -- ((case when Coverage_FlagDelete= '1' then 1 else 1 end ) *)
                                
                                -- Begin Gathering Outgo 
                                if  l_coverage_Obj.Get_array('OutGoList') is not null then
                                    Begin
                                        l_Outgo_arr := l_coverage_Obj.Get_array('OutGoList');
                                        indexOutgo :=0;
                                        FOR d IN 0 .. l_Outgo_arr.get_size - 1 LOOP
                                            l_Outgo_Obj := Json_object_t(l_Outgo_arr.Get(indexOutgo));
                                            Coverage_OutgoYearly := Coverage_OutgoYearly + l_Outgo_Obj.Get_number('OutgoAmount');
                                            Outgo_OutgoAmount := l_Outgo_Obj.Get_number('OutgoAmount');
                                            if Outgo_OutgoAmount is null then
                                                Outgo_OutgoAmount :=0;
                                            END IF;
                                            Outgo_PercentOutgo := NVL(l_Outgo_Obj.Get_number('PercentOutgo'),0);
                                            Outgo_TypeOfOutgo :=l_Outgo_Obj.Get_string('TypeOfOutgo');
                                            Outgo_ReceiverOutgo := l_Outgo_Obj.Get_string('ReceiverOutgo');
                                            Outgo_SourceOfBusiness :=  l_Outgo_Obj.Get_string('SourceOfBusiness');
                                            Outgo_OCID := l_Outgo_Obj.Get_string('OCID');
                                            if Outgo_OCID is null then
                                                Outgo_OCID := Outgo_SourceOfBusiness;
                                                if Outgo_OCID is null then
                                                    Errmsg := 'OCID di OutGoList Tidak ditemukan.' ;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                END IF;
                                            END IF;
                                            Outgo_SourceBizName := l_Outgo_Obj.Get_string('SourceBizName');
                                            Outgo_FlagDelete :=  NVL(l_Outgo_Obj.Get_string('FlagDelete'),0);
                                            Outgo_FlagOldData := NVL(l_Outgo_Obj.Get_string('FlagOldData'),0);
                                            Outgo_FlagEditData := NVL(l_Outgo_Obj.Get_string('FlagEditData'),0);
                                            Outgo_SourceOfBusiness := POOLDATA.getnewID(Outgo_SourceOfBusiness,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                            
                                            if(Outgo_TypeOfOutgo <> '9') then
                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                if(tempocid=Outgo_OCID) then
                                                    if Outgo_TypeOfOutgo in ('2','A','B','C','D','E') then
                                                        SELECT COUNT(0) INTO cntOC FROM M_OC WHERE ID = tempocid;
                                                        IF cntOC = 0 THEN
                                                            Errmsg := 'OCID tidak ditemukan di table M_OC : ' || tempocid;
                                                            ROLLBACK;
                                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                            COMMIT;
                                                            RETURN;
                                                        END IF;
                                                        tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        IF tempocid IS NULL AND LENGTH(Outgo_OCID)>8 THEN
                                                            GENERATE_LST_OC ( Outgo_OCID, ERRMSG );
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    elsif Outgo_TypeOfOutgo in ('4','5','1') then
                                                        if length (TRIM(Outgo_OCID))>8 then
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        else
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    END IF;
                                                END IF;
                                                
                                                -- OldOutgo Gathering
                                                if vISb2b is null then
                                                    vISb2b := 0;
                                                END IF;
                                                if Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%' then
                                                    if Quotation_GroupPanel in ('002','005','007','010','004') and vISb2b = 0 then
                                                        if Outgo_FlagEditData ='1' then
                                                            l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                            indexOldOutgo := 0;
                                                                    
                                                            FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                indexOldOutgo := indexOldOutgo+1;
                                                            END LOOP;
                                                        else
                                                            if Outgo_FlagOldData = '1' then
                                                                Outgo_OutgoAmount := 0;
                                                            --else
                                                            END IF;
                                                        END IF;
                                                    else
                                                        --if Outgo_FlagDelete = '0' then
                                                            if  l_Outgo_Obj.Get_array('OldOutgo') is not null  and Outgo_FlagOldData ='1'then
                                                                l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                                indexOldOutgo := 0;
                                                                    
                                                                FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                    l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                    Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);
                                                                    indexOldOutgo := indexOldOutgo+1;
                                                                END LOOP;
                                                            END IF;
                                                        --else
                                                            --Outgo_OutgoAmount := Outgo_OutgoAmount * -1;                                            
                                                        --END IF;                                        
                                                    END IF;
                                                END IF;
                                                -- End OldOutgo Gathering
                                                
                                                if Outgo_TypeOfOutgo not  in ('1','4','A','B','C','D','E') then 
                                                    Errmsg := 'Error gathering OutGoList, Type Outgo salah : ' ||  Outgo_TypeOfOutgo || 'Receiver : ' || Outgo_ReceiverOutgo;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                else
                                                    if Outgo_TypeOfOutgo in ('A','B','C','D','E')  then
                                                        Coverage_OCYearly := Coverage_OCYearly + Outgo_OutgoAmount;
                                                    else
                                                        Coverage_CommYearly := Coverage_CommYearly + Outgo_OutgoAmount;
                                                    END IF;
                                                END IF;
                                                
                                                COMMKWIID := 0; -- generate COMMKWIID
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                                
                                                SLIPNO := 0;    -- generate slipno / mnknokwikom
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                                
                                                --DBMS_OUTPUT.Put_Line('param pph OCID: ' || tempocid );
                                                COLLECTION.p_get_pph_new@asmd.sinarmas.co.id(Outgo_OutgoAmount, lkuid, tempocid, outgo_sourcebizname, lppid, pctpph, pphnote, Errmsg);                                                                        
                                                
                                                if pctpph is null then 
                                                    pctpph := 0;
                                                END IF;
                                                
                                                cntpctpph := 0;
                                                IF pctpph = 0.05 or pctpph = 0.06 THEN
                                                    cntpctpph := pctpph * 0.5;
                                                ELSE
                                                    cntpctpph := pctpph;
                                                END IF; 
                                   
                                                SELECT COLLECTION.f_get_ppn_booking@asmd.sinarmas.co.id(tempocid) INTO pctppn FROM DUAL;
                                                
                                                if pctppn is null then
                                                    pctppn := 0;
                                                END IF;
                                                
                                                if Quotation_BusinessCode ='50' and Outgo_TypeOfOutgo = '4' then
                                                    pctppn := 0;
                                                END IF;
                                                
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                
                                                if indexobject =0 and indexCoverage =0 then
                                                    INSERT INTO POOLDATA.t_pool_outgo(ocidold, slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete,pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete,pphnote);
                                                else
                                                    SELECT COUNT(1) into cntTypeOutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and FlagDelete <> '1';
                                                        if cntTypeOutgo > 0  then
                                                            SELECT ocid into cekExistsOCID from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and rownum = 1;
                                                            if cekExistsOCID <> Outgo_OCID and Outgo_FlagDelete <> '1' then
                                                                Errmsg := 'Error insert OutGo OCID '|| Outgo_OCID || ', TypeOfOutgo '|| Outgo_TypeOfOutgo || ' sudah ada dengan OCID : '|| cekExistsOCID;
                                                                ROLLBACK;
                                                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                                COMMIT;
                                                                RETURN;
                                                            END IF;
                                                        END IF;
                                                    SELECT COUNT(1) into cntoutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(Outgo_OCID);
                                                    if cntoutgo=0 then
                                                        INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno, pphnote) values
                                                        (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1', pphnote);
                                                    else                                          
                                                        update POOLDATA.t_pool_outgo set
                                                        OutgoAmount = OutgoAmount + Outgo_OutgoAmount,
                                                        pphamount = pphamount + outgo_pphamount,
                                                        ppnamount = ppnamount + outgo_ppnamount,
                                                        totalcomm = totalcomm + outgo_totalcomm
                                                        WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(outgo_ocid);
                                                    END IF;
                                                END IF;
                                            END IF;
                                                                                
                                            indexOutgo := indexOutgo+1;
                                        End LOOP;
                                    Exception when others then
                                        Errmsg := 'Error gathering OutGoList : ' || SQLERRM  || dbms_utility.format_error_backtrace;
                                        ROLLBACK;
                                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                        COMMIT;
                                        RETURN;
                                    END;
                                END IF;
                                -- End Gathering Outgo
                                indexCoverage:=indexCoverage+1;
                                
                                Coverage_PremiTotal := Coverage_PremiTotal + Coverage_PremiYearly;
                                Coverage_DiscountTotal := Coverage_DiscountTotal + Coverage_DiscountYearly;
                                Coverage_OutgoTotal := Coverage_OutgoTotal + Coverage_OutgoYearly;
                                Coverage_CommTotal := Coverage_CommTotal + Coverage_CommYearly;
                                Coverage_OCTotal := Coverage_OCTotal + Coverage_OCYearly;
                            end loop;
                            indexobject:=indexobject+1;
                        end loop;
                        
                        --cek installment outgo > 1
                        BEGIN
                            if Policy_TypeOfCoins <> '1' then
                                IF l_Installment_arr.get_size > 1 THEN
                                    flagDeleteOutgo := 0;
                                    FOR var_outgo IN curr_pool_outgo(Policy_CaseID, Policy_PolicyNo) loop
                                        tempocid               := var_outgo.ocidold;
                                        Outgo_OCID             := var_outgo.ocid;
                                        Outgo_SourceBizName := var_outgo.sourcebizname;
                                        pctppn                := var_outgo.ppnpercent;
                                        pctpph                 := var_outgo.pphpercent;
                                        lppid                 := var_outgo.pphtype;
                                        Outgo_TypeOfOutgo     := var_outgo.typeofoutgo;
                                        Outgo_ReceiverOutgo := var_outgo.receiveroutgo;
                                        Outgo_PercentOutgo  := var_outgo.percentoutgo;
                                        Outgo_OutgoAmountv  := var_outgo.outgoamount;
                                        pphnote             := var_outgo.pphnote;
                                                                    
                                        if flagDeleteOutgo = 0 then
                                            DELETE FROM POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo);
                                            flagDeleteOutgo := 1;
                                        END IF;
                                                                    
                                        --DBMS_OUTPUT.Put_Line('Proses ulang installment : ' || l_Installment_arr.get_size);
                                        indexInstallment := 0;
                                        FOR e IN 0 .. l_Installment_arr.get_size - 1 LOOP
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                            l_Installment_Obj := Json_object_t(l_Installment_arr.Get(indexInstallment));
                                            Installment_InstallmentPercentage := l_Installment_Obj.Get_number('InstallmentPercentage');
                                            Installment_Commission := l_Installment_Obj.Get_number('Commission');
                                            Installment_InstallmentNo := l_Installment_Obj.Get_number('InstallmentNo');
                                            indexInstallment := indexInstallment +1;
                                                                        
                                            if Quotation_GroupPanel in ('007') then
                                                Outgo_OutgoAmount := NVL(Installment_Commission,0);
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            else
                                                Outgo_OutgoAmount := NVL(Outgo_OutgoAmountv,0) * NVL(Installment_InstallmentPercentage,0)/100;
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            END IF;         
                                        END LOOP;            
                                    END LOOP;
                                END IF; 
                            END IF;                               
                        Exception when others then
                            Errmsg := 'Error insert Outgo Installment > 1 : ' || SQLERRM;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;   
                        END;                                                    
                        --end cek installment outgo >1 
                               
                    elsif Quotation_GroupPanel in ('006') and (Quotation_EdmType <> '9' or Quotation_EdmType is null)  then --object FIRE
                        SELECT COUNT(1) into vJumlahObject from POOLDATA.t_propertylist WHERE idpega = vvIDPEGA;
                        if vJumlahObject = 0 then
                            Errmsg := 'IdPega '|| vvIDPEGA || ' Tidak ditemukan di T_PROPERTYLIST' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;
                        
                        for var_property_obj in curr_property_obj (vvIDPEGA) loop
                            l_Object_Obj := Json_object_t(var_property_obj.propertylist);
                            if l_Object_obj is null then
                                Errmsg := 'Error Get Data Object : Object data tidak ada : PropertyList' ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            if var_property_obj.coveragelist is null then
                                Errmsg := 'Data CoverageList di T_PROPERTYLIST NULL';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            
                            --Loop masing2 object
                            l_coverage_arr := Json_object_t(var_property_obj.coveragelist).Get_array('CoverageList');

                            Coverage_PremiTotal :=0;
                            Coverage_DiscountTotal :=0;
                            Coverage_OCTotal :=0;
                            Coverage_CommTotal :=0;
                            Coverage_OutgoTotal :=0;
                            delete POOLDATA.CoverageYearly WHERE caseID = Policy_CaseID;
                            delete POOLDATA.OutgoYearly WHERE caseID = Policy_CaseID;
                            indexCoverage :=0;
                            
                            FOR b IN 0 .. l_coverage_arr.get_size - 1 LOOP
                            
                                Coverage_PremiYearly := 0;
                                Coverage_DiscountYearly := 0;
                                Coverage_OutgoYearly := 0;
                                Coverage_CommYearly := 0;
                                Coverage_OCYearly := 0;
                                l_coverage_Obj := Json_object_t(l_coverage_arr.Get(indexCoverage));
                                
                                Coverage_BeginDate:=to_Date(Substr(l_coverage_Obj.Get_string('BeginDate'),0,8),'YYYYMMdd');
                                Coverage_EndDate:=to_Date(Substr(l_coverage_Obj.Get_string('EndDate'),0,8),'YYYYMMdd');
                                Coverage_PremiYearly := Coverage_PremiYearly + NVL(l_coverage_Obj.Get_number('Premium'),0);--((case when Coverage_FlagDelete= '1' then 1 else 1 end ) * ) 
                                Coverage_DiscountYearly := Coverage_DiscountYearly + NVL(l_coverage_Obj.Get_number('Discount'),0); -- ((case when Coverage_FlagDelete= '1' then 1 else 1 end ) *)
                                
                                -- Begin Gathering Outgo 
                                if  l_coverage_Obj.Get_array('OutGoList') is not null then
                                    Begin
                                        l_Outgo_arr := l_coverage_Obj.Get_array('OutGoList');
                                        indexOutgo :=0;
                                        FOR d IN 0 .. l_Outgo_arr.get_size - 1 LOOP
                                            l_Outgo_Obj := Json_object_t(l_Outgo_arr.Get(indexOutgo));
                                            Coverage_OutgoYearly := Coverage_OutgoYearly + l_Outgo_Obj.Get_number('OutgoAmount');
                                            Outgo_OutgoAmount := l_Outgo_Obj.Get_number('OutgoAmount');
                                            if Outgo_OutgoAmount is null then
                                                Outgo_OutgoAmount :=0;
                                            END IF;
                                            Outgo_PercentOutgo := NVL(l_Outgo_Obj.Get_number('PercentOutgo'),0);
                                            Outgo_TypeOfOutgo :=l_Outgo_Obj.Get_string('TypeOfOutgo');
                                            Outgo_ReceiverOutgo := l_Outgo_Obj.Get_string('ReceiverOutgo');
                                            Outgo_SourceOfBusiness :=  l_Outgo_Obj.Get_string('SourceOfBusiness');
                                            Outgo_OCID := l_Outgo_Obj.Get_string('OCID');
                                            if Outgo_OCID is null then
                                                Outgo_OCID := Outgo_SourceOfBusiness;
                                            END IF;
                                            Outgo_SourceBizName := l_Outgo_Obj.Get_string('SourceBizName');
                                            Outgo_FlagDelete :=  NVL(l_Outgo_Obj.Get_string('FlagDelete'),0);
                                            Outgo_FlagOldData := NVL(l_Outgo_Obj.Get_string('FlagOldData'),0);
                                            Outgo_FlagEditData := NVL(l_Outgo_Obj.Get_string('FlagEditData'),0);
                                            Outgo_SourceOfBusiness := POOLDATA.getnewID(Outgo_SourceOfBusiness,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                            
                                            if(Outgo_TypeOfOutgo <> '9') then
                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                if(tempocid=Outgo_OCID) then
                                                    if Outgo_TypeOfOutgo in ('2','A','B','C','D','E') then
                                                        SELECT COUNT(0) INTO cntOC FROM M_OC WHERE ID = tempocid;
                                                        IF cntOC = 0 THEN
                                                            Errmsg := 'OCID tidak ditemukan di table M_OC : ' || tempocid;
                                                            ROLLBACK;
                                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                            COMMIT;
                                                            RETURN;
                                                        END IF;
                                                        tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        IF tempocid IS NULL AND LENGTH(Outgo_OCID)>8 THEN
                                                            GENERATE_LST_OC ( Outgo_OCID, ERRMSG );
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    elsif Outgo_TypeOfOutgo in ('4','5','1') then
                                                        if length (TRIM(Outgo_OCID))>8 then
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        else
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    END IF;
                                                END IF;
                                                
                                                -- OldOutgo Gathering
                                                if Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%' then                                                    
                                                    --if Outgo_FlagDelete = '0' then
                                                        if  l_Outgo_Obj.Get_array('OldOutgo') is not null  and Outgo_FlagEditData ='1'then
                                                            l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                            indexOldOutgo := 0;
                                                                    
                                                            FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                indexOldOutgo := indexOldOutgo+1;
                                                            END LOOP;
                                                        else
                                                            if Outgo_FlagOldData ='1' then
                                                                Outgo_OutgoAmount := 0;
                                                            --else
                                                            END IF;
                                                        END IF;
--                                                    else
--                                                        Outgo_OutgoAmount := Outgo_OutgoAmount * -1;                                            
--                                                    END IF; 
                                                END IF;
                                                -- End OldOutgo Gathering
                                                
                                                if Outgo_TypeOfOutgo not  in ('1','4','A','B','C','D','E') then 
                                                    Errmsg := 'Error gathering OutGoList, Type Outgo salah : ' ||  Outgo_TypeOfOutgo || ' Receiver : ' || Outgo_ReceiverOutgo;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                else
                                                    if Outgo_TypeOfOutgo in ('A','B','C','D','E')  then
                                                        Coverage_OCYearly := Coverage_OCYearly + Outgo_OutgoAmount;
                                                    else
                                                        Coverage_CommYearly := Coverage_CommYearly + Outgo_OutgoAmount;
                                                    END IF;
                                                END IF;
                                                
                                                COMMKWIID := 0; -- generate COMMKWIID
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                                
                                                SLIPNO := 0;    -- generate slipno / mnknokwikom
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                                
                                                --DBMS_OUTPUT.Put_Line('param pph OCID: ' || tempocid );
                                                COLLECTION.p_get_pph_new@asmd.sinarmas.co.id(Outgo_OutgoAmount, lkuid, tempocid, outgo_sourcebizname, lppid, pctpph, pphnote, Errmsg);                                                                        
                                                
                                                if pctpph is null then 
                                                    pctpph := 0;
                                                END IF;
                                                
                                                cntpctpph := 0;
                                                IF pctpph = 0.05 or pctpph = 0.06 THEN
                                                    cntpctpph := pctpph * 0.5;
                                                ELSE
                                                    cntpctpph := pctpph;
                                                END IF; 
                                   
                                                SELECT COLLECTION.f_get_ppn_booking@asmd.sinarmas.co.id(tempocid) INTO pctppn FROM DUAL;
                                                
                                                if pctppn is null then
                                                    pctppn := 0;
                                                END IF;
                                                                                                
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                
                                                if indexobject =0 and indexCoverage =0 and indexOutgo=0 then
                                                    INSERT INTO POOLDATA.t_pool_outgo(ocidold, slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete, pphnote);
                                                else 
                                                    SELECT COUNT(1) into cntTypeOutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and FlagDelete <> '1';
                                                        if cntTypeOutgo > 0  then
                                                            SELECT ocid into cekExistsOCID from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and rownum = 1;
                                                            if cekExistsOCID <> Outgo_OCID and Outgo_FlagDelete <> '1' then
                                                                Errmsg := 'Error insert OutGo OCID '|| Outgo_OCID || ', TypeOfOutgo '|| Outgo_TypeOfOutgo || ' sudah ada dengan OCID : '|| cekExistsOCID;
                                                                ROLLBACK;
                                                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                                COMMIT;
                                                                RETURN;
                                                            END IF;
                                                        END IF;
                                                    SELECT COUNT(1) into cntoutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(Outgo_OCID);
                                                    if cntoutgo=0 then
                                                        INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno, pphnote) values
                                                        (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1', pphnote);
                                                    else    
                                                        update POOLDATA.t_pool_outgo set
                                                        OutgoAmount = OutgoAmount + Outgo_OutgoAmount,
                                                        pphamount = pphamount + outgo_pphamount,
                                                        ppnamount = ppnamount + outgo_ppnamount,
                                                        totalcomm = totalcomm + outgo_totalcomm
                                                        WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(outgo_ocid);
                                                    END IF;
                                                END IF;
                                            END IF;
                                            indexOutgo := indexOutgo+1;
                                        End LOOP;
                                    Exception when others then  
                                        Errmsg := 'Error gathering OutGoList ' || SQLERRM  || dbms_utility.format_error_backtrace;
                                        ROLLBACK;
                                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                        COMMIT;
                                        RETURN;
                                    END;
                                END IF;
                                -- End Gathering Outgo
                                indexCoverage:=indexCoverage+1;
                                
                                Coverage_PremiTotal := Coverage_PremiTotal + Coverage_PremiYearly;
                                Coverage_DiscountTotal := Coverage_DiscountTotal + Coverage_DiscountYearly;
                                Coverage_OutgoTotal := Coverage_OutgoTotal + Coverage_OutgoYearly;
                                Coverage_CommTotal := Coverage_CommTotal + Coverage_CommYearly;
                                Coverage_OCTotal := Coverage_OCTotal + Coverage_OCYearly;
                            end loop;
                            indexobject:=indexobject+1;
                        end loop;
                                    
                        --cek installment outgo > 1
                        BEGIN
                            if Policy_TypeOfCoins <> '1' then
                                IF l_Installment_arr.get_size > 1 THEN
                                    flagDeleteOutgo := 0;
                                    FOR var_outgo IN curr_pool_outgo(Policy_CaseID, Policy_PolicyNo) loop
                                        tempocid               := var_outgo.ocidold;
                                        Outgo_OCID             := var_outgo.ocid;
                                        Outgo_SourceBizName := var_outgo.sourcebizname;
                                        pctppn                := var_outgo.ppnpercent;
                                        pctpph                 := var_outgo.pphpercent;
                                        lppid                 := var_outgo.pphtype;
                                        Outgo_TypeOfOutgo     := var_outgo.typeofoutgo;
                                        Outgo_ReceiverOutgo := var_outgo.receiveroutgo;
                                        Outgo_PercentOutgo  := var_outgo.percentoutgo;
                                        Outgo_OutgoAmountv  := var_outgo.outgoamount;
                                        pphnote             := var_outgo.pphnote;
                                                                    
                                        if flagDeleteOutgo = 0 then
                                            DELETE FROM POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo);
                                            flagDeleteOutgo := 1;
                                        END IF;
                                                                    
                                        --DBMS_OUTPUT.Put_Line('Proses ulang installment : ' || l_Installment_arr.get_size);
                                        indexInstallment := 0;
                                        FOR e IN 0 .. l_Installment_arr.get_size - 1 LOOP
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                            l_Installment_Obj := Json_object_t(l_Installment_arr.Get(indexInstallment));
                                            Installment_InstallmentPercentage := l_Installment_Obj.Get_number('InstallmentPercentage');
                                            Installment_Commission := l_Installment_Obj.Get_number('Commission');
                                            Installment_InstallmentNo := l_Installment_Obj.Get_number('InstallmentNo');
                                            indexInstallment := indexInstallment +1;
                                                                        
                                            if Quotation_GroupPanel in ('007') then
                                                Outgo_OutgoAmount := NVL(Installment_Commission,0);
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            else
                                                Outgo_OutgoAmount := NVL(Outgo_OutgoAmountv,0) * NVL(Installment_InstallmentPercentage,0)/100;
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            END IF;         
                                        END LOOP;            
                                    END LOOP;
                                END IF; 
                            END IF;                               
                        Exception when others then
                            Errmsg := 'Error insert Outgo Installment > 1 : ' || SQLERRM;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;   
                        END;                                                    
                        --end cek installment outgo >1 
                           
                    elsif Quotation_GroupPanel in ('003') then -- ANEKA  
                        SELECT COUNT(0) into vJumlahObject from POOLDATA.t_anekalist WHERE idpega = vvIDPEGA; 
                        if vJumlahObject = 0 then
                            Errmsg := 'IdPega ' || vvIDPEGA || ' Tidak ditemukan di T_ANEKALIST' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;   
                        SELECT max(indexobject) into vJumlahObject from POOLDATA.t_anekalist WHERE idpega = vvIDPEGA;                                       
                        for var_aneka_obj in curr_aneka_obj (vvIDPEGA) loop
                            l_Object_Obj := Json_object_t(var_aneka_obj.anekalist);
                            if l_Object_obj is null then
                                Errmsg := 'Error Get Data Object : Object data tidak ada : AnekaList' ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            if var_aneka_obj.coveragelist is null then
                                Errmsg := 'Data CoverageList di T_ANEKALIST NULL';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            --Loop masing2 object
                            l_coverage_arr := Json_object_t(var_aneka_obj.coveragelist).Get_array('CoverageList');

                            Coverage_PremiTotal :=0;
                            Coverage_DiscountTotal :=0;
                            Coverage_OCTotal :=0;
                            Coverage_CommTotal :=0;
                            Coverage_OutgoTotal :=0;
                            delete POOLDATA.CoverageYearly WHERE caseID = Policy_CaseID;
                            delete POOLDATA.OutgoYearly WHERE caseID = Policy_CaseID;
                            indexCoverage :=0;
                            
                            FOR b IN 0 .. l_coverage_arr.get_size - 1 LOOP
                            
                                Coverage_PremiYearly := 0;
                                Coverage_DiscountYearly := 0;
                                Coverage_OutgoYearly := 0;
                                Coverage_CommYearly := 0;
                                Coverage_OCYearly := 0;
                                l_coverage_Obj := Json_object_t(l_coverage_arr.Get(indexCoverage));
                                
                                Coverage_BeginDate:=to_Date(Substr(l_coverage_Obj.Get_string('BeginDate'),0,8),'YYYYMMdd');
                                Coverage_EndDate:=to_Date(Substr(l_coverage_Obj.Get_string('EndDate'),0,8),'YYYYMMdd');
                                Coverage_PremiYearly := Coverage_PremiYearly + NVL(l_coverage_Obj.Get_number('Premium'),0);--((case when Coverage_FlagDelete= '1' then 1 else 1 end ) * ) 
                                Coverage_DiscountYearly := Coverage_DiscountYearly + NVL(l_coverage_Obj.Get_number('Discount'),0); -- ((case when Coverage_FlagDelete= '1' then 1 else 1 end ) *)
                                
                                -- Begin Gathering Outgo 
                                if  l_coverage_Obj.Get_array('OutGoList') is not null then
                                    Begin
                                        l_Outgo_arr := l_coverage_Obj.Get_array('OutGoList');
                                        indexOutgo :=0;
                                        FOR d IN 0 .. l_Outgo_arr.get_size - 1 LOOP
                                            l_Outgo_Obj := Json_object_t(l_Outgo_arr.Get(indexOutgo));
                                            Coverage_OutgoYearly := Coverage_OutgoYearly + l_Outgo_Obj.Get_number('OutgoAmount');
                                            Outgo_OutgoAmount := l_Outgo_Obj.Get_number('OutgoAmount');
                                            if Outgo_OutgoAmount is null then
                                                Outgo_OutgoAmount :=0;
                                            END IF;
                                            Outgo_PercentOutgo := NVL(l_Outgo_Obj.Get_number('PercentOutgo'),0);
                                            Outgo_TypeOfOutgo :=l_Outgo_Obj.Get_string('TypeOfOutgo');
                                            Outgo_ReceiverOutgo := l_Outgo_Obj.Get_string('ReceiverOutgo');
                                            Outgo_SourceOfBusiness :=  l_Outgo_Obj.Get_string('SourceOfBusiness');
                                            Outgo_OCID := l_Outgo_Obj.Get_string('OCID');
                                            if Outgo_OCID is null then
                                                Outgo_OCID := Outgo_SourceOfBusiness;
                                                if Outgo_OCID is null then
                                                    Errmsg := 'OCID di OutGoList Tidak ditemukan.' ;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                END IF;
                                            END IF;
                                            Outgo_SourceBizName := l_Outgo_Obj.Get_string('SourceBizName');
                                            Outgo_FlagDelete :=  NVL(l_Outgo_Obj.Get_string('FlagDelete'),0);
                                            Outgo_FlagOldData := NVL(l_Outgo_Obj.Get_string('FlagOldData'),0);
                                            Outgo_FlagEditData := NVL(l_Outgo_Obj.Get_string('FlagEditData'),0);
                                            Outgo_SourceOfBusiness := POOLDATA.getnewID(Outgo_SourceOfBusiness,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                            
                                            if(Outgo_TypeOfOutgo <> '9') then
                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                if(tempocid=Outgo_OCID) then
                                                    if Outgo_TypeOfOutgo in ('2','A','B','C','D','E') then
                                                        SELECT COUNT(0) INTO cntOC FROM M_OC WHERE ID = tempocid;
                                                        IF cntOC = 0 THEN
                                                            Errmsg := 'OCID tidak ditemukan di table M_OC : ' || tempocid;
                                                            ROLLBACK;
                                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                            COMMIT;
                                                            RETURN;
                                                        END IF;
                                                        tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        IF tempocid IS NULL AND LENGTH(Outgo_OCID)>8 THEN
                                                            GENERATE_LST_OC ( Outgo_OCID, ERRMSG );
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    elsif Outgo_TypeOfOutgo in ('4','5','1') then
                                                        if length (TRIM(Outgo_OCID))>8 then
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        else
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    END IF;
                                                END IF;
                                                
                                                -- OldOutgo Gathering
                                                if vISb2b is null then
                                                    vISb2b := 0;
                                                END IF;
                                                if Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%' THEN
                                                    IF tempocid = '14757' THEN
                                                        IF l_Outgo_Obj.Get_array('OldOutgo') is not null then
                                                            l_OldOutgo_arr := l_Outgo_Obj.Get_array('OldOutgo');
                                                            indexOldOutgo := 0;
                                                                        
                                                            FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                indexOldOutgo := indexOldOutgo+1;
                                                            END LOOP;
                                                        ELSE
                                                            IF Outgo_FlagOldData = '1' THEN
                                                                Outgo_OutgoAmount := 0;
                                                            END IF;
                                                        END IF;
                                                    ELSE
                                                        if Outgo_FlagEditData ='1' then
                                                            l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                            indexOldOutgo := 0;
                                                                        
                                                            FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                indexOldOutgo := indexOldOutgo+1;
                                                            END LOOP;
                                                        else
                                                            if Outgo_FlagOldData = '1' then
                                                                Outgo_OutgoAmount := 0;
                                                            --else
                                                            END IF;
                                                        END IF;
                                                    END IF;
                                                END IF;
                                                -- End OldOutgo Gathering
                                                
                                                if Outgo_TypeOfOutgo not  in ('1','4','A','B','C','D','E') then 
                                                    Errmsg := 'Error gathering OutGoList, Type Outgo salah : ' ||  Outgo_TypeOfOutgo || ' Receiver : ' || Outgo_ReceiverOutgo;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                else
                                                    if Outgo_TypeOfOutgo in ('A','B','C','D','E')  then
                                                        Coverage_OCYearly := Coverage_OCYearly + Outgo_OutgoAmount;
                                                    else
                                                        Coverage_CommYearly := Coverage_CommYearly + Outgo_OutgoAmount;
                                                    END IF;
                                                END IF;
                                                
                                                COMMKWIID := 0; -- generate COMMKWIID
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                                
                                                SLIPNO := 0;    -- generate slipno / mnknokwikom
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                                
                                                --DBMS_OUTPUT.Put_Line('param pph OCID: ' || tempocid );
                                                COLLECTION.p_get_pph_new@asmd.sinarmas.co.id(Outgo_OutgoAmount, lkuid, tempocid, outgo_sourcebizname, lppid, pctpph, pphnote, Errmsg);                                                                        
                                                
                                                if pctpph is null then 
                                                    pctpph := 0;
                                                END IF;
                                                
                                                cntpctpph := 0;
                                                IF pctpph = 0.05 or pctpph = 0.06 THEN
                                                    cntpctpph := pctpph * 0.5;
                                                ELSE
                                                    cntpctpph := pctpph;
                                                END IF; 
                                   
                                                SELECT COLLECTION.f_get_ppn_booking@asmd.sinarmas.co.id(tempocid) INTO pctppn FROM DUAL;
                                                
                                                if pctppn is null then
                                                    pctppn := 0;
                                                END IF;
                                                
                                                if Quotation_BusinessCode ='50' and Outgo_TypeOfOutgo = '4' then
                                                    pctppn := 0;
                                                END IF;
                                                
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                
                                                --dbms_output.put_line('index object:'||indexobject || '  outgoamount:'||Outgo_OutgoAmount);
                                                IF indexobject = 0 AND indexCoverage = 0 AND indexOutgo = 0 THEN
                                                    INSERT INTO POOLDATA.t_pool_outgo(ocidold, slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete, pphnote);
                                                ELSE
                                                    SELECT COUNT(1) into cntTypeOutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and FlagDelete <> '1';
                                                        if cntTypeOutgo > 0  then
                                                            SELECT ocid into cekExistsOCID from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and rownum = 1;
                                                            if cekExistsOCID <> Outgo_OCID and Outgo_FlagDelete <> '1' then
                                                                Errmsg := 'Error insert OutGo OCID '|| Outgo_OCID || ', TypeOfOutgo '|| Outgo_TypeOfOutgo || ' sudah ada dengan OCID : '|| cekExistsOCID;
                                                                ROLLBACK;
                                                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                                COMMIT;
                                                                RETURN;
                                                            END IF;
                                                        END IF;
                                                    SELECT COUNT(1) into cntoutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(Outgo_OCID);
                                                    if cntoutgo=0 then
                                                        INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno, pphnote) values
                                                        (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1', pphnote);
                                                    else                                          
                                                        update POOLDATA.t_pool_outgo set
                                                        OutgoAmount = OutgoAmount + Outgo_OutgoAmount,
                                                        pphamount = pphamount + outgo_pphamount,
                                                        ppnamount = ppnamount + outgo_ppnamount,
                                                        totalcomm = totalcomm + outgo_totalcomm
                                                        WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(outgo_ocid);
                                                    END IF;
                                                END IF;
                                            END IF;
                                                                                
                                            indexOutgo := indexOutgo+1;
                                        End LOOP;
                                    Exception when others then
                                        Errmsg :='Error gathering OutGoList : ' || SQLERRM  || dbms_utility.format_error_backtrace;
                                        ROLLBACK;
                                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                        COMMIT;
                                        RETURN;
                                    END;
                                END IF;
                                -- End Gathering Outgo
                                
                                IF vTotalYear >= 1 AND Policy_IsAnekaPerYear = 'true' THEN --  ganti grouppanel XXX dengan kode MBU 
                                    IF indexobject = 0 THEN
                                        IF Coverage_BeginDate > Policy_ProdDateTime THEN
                                            INSERT INTO pooldata.CoverageYearly (CaseID,proddate, BEGINDATE, ENDDATE , CurrentYear, Premium ,Discount, Outgo , Comm , OC) values 
                                                (Policy_CaseID,Coverage_BeginDate, Coverage_BeginDate, Coverage_EndDate,indexCoverage +1, nvl(Coverage_PremiYearly,0),Coverage_DiscountYearly,Coverage_OutgoYearly ,Coverage_CommYearly,Coverage_OCYearly);
                                        ELSE
                                            INSERT INTO pooldata.CoverageYearly (CaseID,proddate, BEGINDATE, ENDDATE , CurrentYear, Premium ,Discount, Outgo , Comm , OC) values 
                                                (Policy_CaseID,Policy_ProdDateTime, Coverage_BeginDate, Coverage_EndDate,indexCoverage +1, nvl(Coverage_PremiYearly,0),Coverage_DiscountYearly,Coverage_OutgoYearly ,Coverage_CommYearly,Coverage_OCYearly);
                                        END IF;
                                    ELSE
                                        UPDATE pooldata.CoverageYearly set 
                                        Premium = Premium + Coverage_PremiYearly, 
                                        Discount = Discount + Coverage_DiscountYearly, 
                                        Outgo = Outgo + Coverage_OutgoYearly, 
                                        Comm = Comm + Coverage_CommYearly, 
                                        OC = OC + Coverage_OCYearly
                                        where CaseID = Policy_CaseID And CurrentYear= indexCoverage +1;
                                    END IF;
                                END IF;
                                indexCoverage:=indexCoverage+1;
                                
                                Coverage_PremiTotal := Coverage_PremiTotal + Coverage_PremiYearly;
                                Coverage_DiscountTotal := Coverage_DiscountTotal + Coverage_DiscountYearly;
                                Coverage_OutgoTotal := Coverage_OutgoTotal + Coverage_OutgoYearly;
                                Coverage_CommTotal := Coverage_CommTotal + Coverage_CommYearly;
                                Coverage_OCTotal := Coverage_OCTotal + Coverage_OCYearly;
                                
                            END LOOP;
                            indexobject:=indexobject+1;
                        END LOOP;
                        
                        --cek installment outgo > 1
                        BEGIN
                            IF Policy_TypeOfCoins <> '1' THEN
                                IF l_Installment_arr.get_size > 1 THEN
                                    flagDeleteOutgo := 0;
                                    FOR var_outgo IN curr_pool_outgo(Policy_CaseID, Policy_PolicyNo) LOOP
                                        tempocid               := var_outgo.ocidold;
                                        Outgo_OCID             := var_outgo.ocid;
                                        Outgo_SourceBizName := var_outgo.sourcebizname;
                                        pctppn                := var_outgo.ppnpercent;
                                        pctpph                 := var_outgo.pphpercent;
                                        lppid                 := var_outgo.pphtype;
                                        Outgo_TypeOfOutgo     := var_outgo.typeofoutgo;
                                        Outgo_ReceiverOutgo := var_outgo.receiveroutgo;
                                        Outgo_PercentOutgo  := var_outgo.percentoutgo;
                                        Outgo_OutgoAmountv  := var_outgo.outgoamount;
                                        pphnote             := var_outgo.pphnote;
                                                                    
                                        if flagDeleteOutgo = 0 then
                                            DELETE FROM POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo);
                                            flagDeleteOutgo := 1;
                                        END IF;
                                                                    
                                        --DBMS_OUTPUT.Put_Line('Proses ulang installment : ' || l_Installment_arr.get_size);
                                        indexInstallment := 0;
                                        FOR e IN 0 .. l_Installment_arr.get_size - 1 LOOP
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                            l_Installment_Obj := Json_object_t(l_Installment_arr.Get(indexInstallment));
                                            Installment_InstallmentPercentage := l_Installment_Obj.Get_number('InstallmentPercentage');
                                            Installment_Commission := l_Installment_Obj.Get_number('Commission');
                                            Installment_InstallmentNo := l_Installment_Obj.Get_number('InstallmentNo');
                                            indexInstallment := indexInstallment +1;
                                                                        
                                            if Quotation_GroupPanel in ('007') then
                                                Outgo_OutgoAmount := NVL(Installment_Commission,0);
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            else
                                                Outgo_OutgoAmount := NVL(Outgo_OutgoAmountv,0) * NVL(Installment_InstallmentPercentage,0)/100;
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            END IF;         
                                        END LOOP;            
                                    END LOOP;
                                END IF; 
                            END IF;                               
                        Exception when others then
                            Errmsg := 'Error insert Outgo Installment > 1 ' || SQLERRM ;
                            RETURN;   
                        END;                                                    
                        --end cek installment outgo >1 
                          
                    elsif Quotation_GroupPanel in ('005','002','010') then -- TRAVEL, PA 
                        SELECT COUNT(0) into vJumlahObject from POOLDATA.t_personlist WHERE idpega = vvIDPEGA;
                        if vJumlahObject = 0 then
                            Errmsg := 'IdPega ' || vvIDPEGA || ' Tidak ditemukan di T_PERSONLIST' ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;
                        END IF;  
                        SELECT max(indexobject) into vJumlahObject from POOLDATA.t_personlist WHERE idpega = vvIDPEGA;
                        for var_person_obj in curr_person_obj (vvIDPEGA) loop
                            l_Object_Obj := Json_object_t(var_person_obj.objectdata);
                            if l_Object_obj is null then
                                Errmsg := 'Error Get Data Object : Object data tidak ada : PersonList' ;
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            if var_person_obj.coveragedata is null then
                                Errmsg := 'Data CoverageData di T_PERSONLIST NULL';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;
                            END IF;
                            --Loop masing2 object
                            l_coverage_arr := Json_object_t(var_person_obj.coveragedata).Get_array('ASMCoverage');

                            Coverage_PremiTotal :=0;
                            Coverage_DiscountTotal :=0;
                            Coverage_OCTotal :=0;
                            Coverage_CommTotal :=0;
                            Coverage_OutgoTotal :=0;
                            delete POOLDATA.CoverageYearly WHERE caseID = Policy_CaseID;
                            delete POOLDATA.OutgoYearly WHERE caseID = Policy_CaseID;
                            indexCoverage :=0;
                            
                            FOR b IN 0 .. l_coverage_arr.get_size - 1 LOOP
                                Coverage_PremiYearly := 0;
                                Coverage_DiscountYearly := 0;
                                Coverage_OutgoYearly := 0;
                                Coverage_CommYearly := 0;
                                Coverage_OCYearly := 0;
                                l_coverage_Obj := Json_object_t(l_coverage_arr.Get(indexCoverage));
                                Coverage_BeginDate:=to_Date(Substr(l_coverage_Obj.Get_string('BeginDate'),0,8),'YYYYMMdd');
                                Coverage_EndDate:=to_Date(Substr(l_coverage_Obj.Get_string('EndDate'),0,8),'YYYYMMdd');
                                Coverage_PremiYearly := Coverage_PremiYearly + NVL(l_coverage_Obj.Get_number('Premium'),0);--((case when Coverage_FlagDelete= '1' then 1 else 1 end ) * ) 
                                Coverage_DiscountYearly := Coverage_DiscountYearly + NVL(l_coverage_Obj.Get_number('Discount'),0); -- ((case when Coverage_FlagDelete= '1' then 1 else 1 end ) *)
                                
                                -- Begin Gathering Outgo 
                                if  l_coverage_Obj.Get_array('OutGoList') is not null then
                                    Begin
                                        l_Outgo_arr := l_coverage_Obj.Get_array('OutGoList');
                                        indexOutgo :=0;
                                        FOR d IN 0 .. l_Outgo_arr.get_size - 1 LOOP
                                            l_Outgo_Obj := Json_object_t(l_Outgo_arr.Get(indexOutgo));
                                            Coverage_OutgoYearly := Coverage_OutgoYearly + l_Outgo_Obj.Get_number('OutgoAmount');
                                            Outgo_OutgoAmount := l_Outgo_Obj.Get_number('OutgoAmount');
                                            if Outgo_OutgoAmount is null then
                                                Outgo_OutgoAmount :=0;
                                            END IF;
                                            Outgo_PercentOutgo := NVL(l_Outgo_Obj.Get_number('PercentOutgo'),0);
                                            Outgo_TypeOfOutgo :=l_Outgo_Obj.Get_string('TypeOfOutgo');
                                            Outgo_ReceiverOutgo := l_Outgo_Obj.Get_string('ReceiverOutgo');
                                            Outgo_SourceOfBusiness :=  l_Outgo_Obj.Get_string('SourceOfBusiness');
                                            Outgo_OCID := l_Outgo_Obj.Get_string('OCID');
                                            if Outgo_OCID IS NULL THEN
                                                Outgo_OCID := Outgo_SourceOfBusiness;
                                                if Outgo_OCID IS NULL AND Outgo_OutgoAmount <> 0 THEN
                                                    Errmsg := 'OCID di OutGoList Tidak ditemukan.' ;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                END IF;
                                                
                                                SELECT COUNT(0) INTO cekAgent FROM m_agent WHERE id = Outgo_OCID;
                                                IF cekAgent = 0 AND Outgo_OutgoAmount <> 0 THEN
                                                    SELECT COUNT(0) into cekAgent FROM m_oc WHERE id = Outgo_OCID;
                                                    if cekAgent = 0 then
                                                        Errmsg := 'OCID tidak ditemukan di M_AGENT, OutGoList.OCID:'||Outgo_OCID;
                                                        ROLLBACK;
                                                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                        COMMIT;
                                                        RETURN; 
                                                    END IF;
                                                END IF;  
                                            END IF;
                                            Outgo_SourceBizName := l_Outgo_Obj.Get_string('SourceBizName');
                                            Outgo_FlagDelete :=  NVL(l_Outgo_Obj.Get_string('FlagDelete'),0);
                                            Outgo_FlagOldData := NVL(l_Outgo_Obj.Get_string('FlagOldData'),0);
                                            Outgo_FlagEditData := NVL(l_Outgo_Obj.Get_string('FlagEditData'),0);
                                            Outgo_SourceOfBusiness := POOLDATA.getnewID(Outgo_SourceOfBusiness,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                            
                                            if(Outgo_TypeOfOutgo <> '9') then
                                                tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                if(tempocid=Outgo_OCID) then
                                                    if Outgo_TypeOfOutgo in ('2','A','B','C','D','E') then
                                                        SELECT COUNT(0) INTO cntOC FROM M_OC WHERE ID = tempocid;
                                                        IF cntOC = 0 THEN
                                                            Errmsg := 'OCID tidak ditemukan di table M_OC : ' || tempocid;
                                                            ROLLBACK;
                                                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                            COMMIT;
                                                            RETURN;
                                                        END IF;
                                                        tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        IF tempocid IS NULL AND LENGTH(Outgo_OCID)>8 THEN
                                                            GENERATE_LST_OC ( Outgo_OCID, ERRMSG );
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    elsif Outgo_TypeOfOutgo in ('4','5','1') then
                                                        if length (TRIM(Outgo_OCID))>8 then
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_OC','ID','kosong', 'kosong', 'OLDID');
                                                        else
                                                            tempocid := POOLDATA.getnewID(Outgo_OCID,'M_AGENT','ID','kosong', 'kosong', 'OLDID');
                                                        END IF;
                                                    END IF;
                                                END IF;
                                                
                                                -- OldOutgo Gathering
                                                if vISb2b is null then
                                                    vISb2b := 0;
                                                END IF;
                                                if Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%' then
                                                    if Outgo_OCID = '10034413' THEN
                                                        if  l_Outgo_Obj.Get_array('OldOutgo') is not null then
                                                            l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                            indexOldOutgo := 0;
                                                            FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);
                                                                indexOldOutgo := indexOldOutgo+1;
                                                            END LOOP;
                                                        END IF;
                                                    elsif Quotation_GroupPanel in ('002','005','007','010') and vISb2b = 0 then
                                                        if Outgo_FlagEditData ='1' then
                                                            l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                            indexOldOutgo := 0;
                                                                    
                                                            FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                indexOldOutgo := indexOldOutgo+1;
                                                            END LOOP;
                                                        else
                                                            if Outgo_FlagOldData = '1' then
                                                                Outgo_OutgoAmount := 0;
                                                            --else
                                                            END IF;
                                                        END IF;
                                                    else
                                                        --if Outgo_FlagDelete = '0' then
                                                            if  l_Outgo_Obj.Get_array('OldOutgo') is not null  and Outgo_FlagOldData ='1'then
                                                                l_OldOutgo_arr :=l_Outgo_Obj.Get_array('OldOutgo');
                                                                indexOldOutgo := 0;
                                                                    
                                                                FOR q IN 0 .. l_OldOutgo_arr.get_size - 1 LOOP
                                                                    l_OldOutgo_Obj :=Json_object_t(l_OldOutgo_arr.Get(indexOldOutgo));
                                                                    Outgo_OutgoAmount := Outgo_OutgoAmount - NVL(l_OldOutgo_Obj.Get_number('OutgoAmount'),0);

                                                                    indexOldOutgo := indexOldOutgo+1;
                                                                END LOOP;
                                                            END IF;
                                                        --else
                                                            --Outgo_OutgoAmount := Outgo_OutgoAmount * -1;                                            
                                                        --END IF;                                        
                                                    END IF;
                                                END IF;
                                                -- End OldOutgo Gathering
                                                
                                                if Outgo_TypeOfOutgo not  in ('1','4','A','B','C','D','E') then 
                                                    Errmsg := 'Error gathering OutGoList, Type Outgo salah : ' ||  Outgo_TypeOfOutgo || ' Receiver : ' || Outgo_ReceiverOutgo;
                                                    ROLLBACK;
                                                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                    COMMIT;
                                                    RETURN;
                                                else
                                                    if Outgo_TypeOfOutgo in ('A','B','C','D','E')  then
                                                        Coverage_OCYearly := Coverage_OCYearly + Outgo_OutgoAmount;
                                                    else
                                                        Coverage_CommYearly := Coverage_CommYearly + Outgo_OutgoAmount;
                                                    END IF;
                                                END IF;
                                                
                                                COMMKWIID := 0; -- generate COMMKWIID
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                                
                                                SLIPNO := 0;    -- generate slipno / mnknokwikom
                                                POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                                
                                                --DBMS_OUTPUT.Put_Line('param pph OCID: ' || tempocid );
                                                COLLECTION.p_get_pph_new@asmd.sinarmas.co.id(Outgo_OutgoAmount, lkuid, tempocid, outgo_sourcebizname, lppid, pctpph, pphnote, Errmsg);                                                                        
                                                
                                                if pctpph is null then 
                                                    pctpph := 0;
                                                END IF;
                                                
                                            -- 01-02-2021 OTO/SOF Refund komisi PPH 0 (Group 73) 
                                                if Quotation_SobLeader2 in ('10003883',     --OTO MULTIARTHA
                                                                            '10007505',     --SUMMIT OTO FINANCE, PT.
                                                                            '10008129',     --OTO MULTIARTHA AFFINITY
                                                                            '10013958',     --OTO SOF ASSET
                                                                            '10055652',     --SUMMIT OTO FINANCE RENEWAL
                                                                            '10055999')     --OTO MULTIARTHA  RENEWAL
                                                and Policy_CaseID like 'ASM-FW-GISFW-WORK EDM-%' and Payment_Commision < 0 then
                                                    pctpph := 0;
                                                END IF;
                                                
                                                cntpctpph := 0;
                                                IF pctpph = 0.05 or pctpph = 0.06 THEN
                                                    cntpctpph := pctpph * 0.5;
                                                ELSE
                                                    cntpctpph := pctpph;
                                                END IF; 
                                   
                                                SELECT COLLECTION.f_get_ppn_booking@asmd.sinarmas.co.id(tempocid) INTO pctppn FROM DUAL;
                                                
                                                if pctppn is null then
                                                    pctppn := 0;
                                                END IF;
                                                
                                                if Quotation_BusinessCode ='50' and Outgo_TypeOfOutgo = '4' then
                                                    pctppn := 0;
                                                END IF;
                                                
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                
--                                                IF Quotation_EdmType = '2' AND Quotation_GroupPanel = '002' THEN
--                                                    BEGIN
--                                                        SELECT CASE WHEN a.jsondata.CustomProrateRefundFormula = '1' THEN TO_NUMBER(a.jsondata.FormulaPercentage)/100 ELSE 1 END
--                                                         INTO Outgo_FormulaPercentage
--                                                        FROM M_template a
--                                                        WHERE ID = (SELECT templateno FROM t_general WHERE idpega = Policy_CaseID);
--                                                    EXCEPTION WHEN NO_DATA_FOUND THEN
--                                                        Outgo_FormulaPercentage := 1;
--                                                    END;
--                                                    Outgo_OutgoAmount := Outgo_OutgoAmount*NVL(Outgo_FormulaPercentage,0);
--                                                END IF;
                                                
                                                if indexobject =0 and indexCoverage =0 then
                                                    SELECT COUNT(1) into cntoutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(Outgo_OCID);
                                                    if cntoutgo=0 then
                                                        --DBMS_OUTPUT.PUT_LINE('Insert Outgo2, OCID :'||Outgo_OCID);
                                                        INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete, pphnote) values
                                                        (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete, pphnote);
                                                    else                                          
                                                        update POOLDATA.t_pool_outgo set
                                                        OutgoAmount = OutgoAmount + Outgo_OutgoAmount,
                                                        pphamount = pphamount + outgo_pphamount,
                                                        ppnamount = ppnamount + outgo_ppnamount,
                                                        totalcomm = totalcomm + outgo_totalcomm
                                                        WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(outgo_ocid);
                                                    END IF;
--                                                    INSERT INTO POOLDATA.t_pool_outgo(ocidold, slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete, pphnote) values
--                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete, pphnote);
                                                else
                                                    --DBMS_OUTPUT.PUT_LINE('typeofoutgo :'||Outgo_TypeOfOutgo||   '   OCID:'||Outgo_OCID);
                                                    SELECT COUNT(1) into cntTypeOutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and FlagDelete <> '1';
                                                        if cntTypeOutgo > 0  then
                                                            SELECT ocid into cekExistsOCID from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and typeofoutgo = Outgo_TypeOfOutgo and rownum = 1;
                                                            if cekExistsOCID <> Outgo_OCID and Outgo_FlagDelete <> '1' and Outgo_OutgoAmount <> 0 then
                                                                Errmsg := 'Error insert OutGo OCID '|| Outgo_OCID || ', TypeOfOutgo '|| Outgo_TypeOfOutgo || ' sudah ada dengan OCID : '|| cekExistsOCID;
                                                                ROLLBACK;
                                                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                                                COMMIT;
                                                                RETURN;
                                                            END IF;
                                                        END IF;
                                                    SELECT COUNT(1) into cntoutgo from POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(Outgo_OCID);
                                                    if cntoutgo=0 then
                                                        --DBMS_OUTPUT.PUT_LINE('Insert Outgo2, OCID :'||Outgo_OCID);
                                                        INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid,installmentno,sourceofbusiness,flagdelete, pphnote) values
                                                        (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID,'1',Outgo_sourceofbusiness,outgo_flagdelete, pphnote);
                                                    else                                          
                                                        update POOLDATA.t_pool_outgo set
                                                        OutgoAmount = OutgoAmount + Outgo_OutgoAmount,
                                                        pphamount = pphamount + outgo_pphamount,
                                                        ppnamount = ppnamount + outgo_ppnamount,
                                                        totalcomm = totalcomm + outgo_totalcomm
                                                        WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo) and OCID=TRIM(outgo_ocid);
                                                    END IF;
                                                END IF;
                                            END IF;
                                                                                
                                            indexOutgo := indexOutgo+1;
                                        End LOOP;
                                    Exception when others then
                                        Errmsg :='Error gathering OutGoList : ' || SQLERRM  || dbms_utility.format_error_backtrace;
                                        ROLLBACK;
                                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                        COMMIT;
                                        RETURN;
                                    END;
                                END IF;
                                -- End Gathering Outgo
                                indexCoverage:=indexCoverage+1;
                                
                                Coverage_PremiTotal := Coverage_PremiTotal + Coverage_PremiYearly;
                                Coverage_DiscountTotal := Coverage_DiscountTotal + Coverage_DiscountYearly;
                                Coverage_OutgoTotal := Coverage_OutgoTotal + Coverage_OutgoYearly;
                                Coverage_CommTotal := Coverage_CommTotal + Coverage_CommYearly;
                                Coverage_OCTotal := Coverage_OCTotal + Coverage_OCYearly;
                                
                            end loop;
                            indexobject:=indexobject+1;
                        end loop;
--                        if round(Coverage_CommTotal) <> round(Payment_commision) then
--                            Errmsg := 'Error insert OutGoList : Nilai Komisi Outgo :' || Coverage_CommTotal || ' tidak sama dengan Payment.Commision :'||Payment_commision;
--                            ROLLBACK;
--                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                            COMMIT;
--                            RETURN;
--                        END IF;
                        
                        --cek installment outgo > 1
                        DBMS_OUTPUT.Put_Line('Mulai Object PA2' || CURRENT_TIMESTAMP);
                        BEGIN
                            if Policy_TypeOfCoins <> '1' then
                                IF l_Installment_arr.get_size > 1 THEN
                                    flagDeleteOutgo := 0;
                                    FOR var_outgo IN curr_pool_outgo(Policy_CaseID, Policy_PolicyNo) loop
                                        tempocid               := var_outgo.ocidold;
                                        Outgo_OCID             := var_outgo.ocid;
                                        Outgo_SourceBizName := var_outgo.sourcebizname;
                                        pctppn                := var_outgo.ppnpercent;
                                        pctpph                 := var_outgo.pphpercent;
                                        lppid                 := var_outgo.pphtype;
                                        Outgo_TypeOfOutgo     := var_outgo.typeofoutgo;
                                        Outgo_ReceiverOutgo := var_outgo.receiveroutgo;
                                        Outgo_PercentOutgo  := var_outgo.percentoutgo;
                                        Outgo_OutgoAmountv  := var_outgo.outgoamount;
                                        pphnote             := var_outgo.pphnote;
                                                                    
                                        if flagDeleteOutgo = 0 then
                                            DELETE FROM POOLDATA.t_pool_outgo WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo= TRIM(Policy_PolicyNo);
                                            flagDeleteOutgo := 1;
                                        END IF;
                                                                    
                                        --DBMS_OUTPUT.Put_Line('Proses ulang installment : ' || l_Installment_arr.get_size);
                                        indexInstallment := 0;
                                        FOR e IN 0 .. l_Installment_arr.get_size - 1 LOOP
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_COMMID_SEQ(CommKwiId);
                                            POOLDATA.PKG_COUNTER_PRODUCTION.GET_NOKWIKOM_SEQ(SlipNo);
                                            l_Installment_Obj := Json_object_t(l_Installment_arr.Get(indexInstallment));
                                            Installment_InstallmentPercentage := l_Installment_Obj.Get_number('InstallmentPercentage');
                                            Installment_Commission := l_Installment_Obj.Get_number('Commission');
                                            Installment_InstallmentNo := l_Installment_Obj.Get_number('InstallmentNo');
                                            indexInstallment := indexInstallment +1;
                                                                        
                                            if Quotation_GroupPanel in ('007') then
                                                Outgo_OutgoAmount := NVL(Installment_Commission,0);
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            else
                                                Outgo_OutgoAmount := NVL(Outgo_OutgoAmountv,0) * NVL(Installment_InstallmentPercentage,0)/100;
                                                outgo_pphamount := NVL(cntpctpph,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_ppnamount := NVL(pctppn,0) * NVL(Outgo_OutgoAmount,0);
                                                outgo_totalcomm := NVL(Outgo_OutgoAmount,0) - NVL(outgo_pphamount,0) + NVL(outgo_ppnamount,0);
                                                                            
                                                INSERT INTO POOLDATA.t_pool_outgo(ocidold,slipno, commkwiid, ocid, sourcebizname,ppnpercent, ppnamount, pphpercent, pphamount, pphtype, totalcomm, outgoamount, typeofoutgo, receiveroutgo, percentoutgo, policyno, caseid, installmentno, pphnote) values
                                                    (tempocid,slipno, commkwiid, Outgo_OCID, Outgo_SourceBizName, pctppn, outgo_ppnamount, pctpph, outgo_pphamount,lppid, outgo_totalcomm, Outgo_OutgoAmount, Outgo_TypeOfOutgo, Outgo_ReceiverOutgo, Outgo_PercentOutgo, Policy_PolicyNo, Policy_CaseID, Installment_InstallmentNo, pphnote);
                                            END IF;         
                                        END LOOP;            
                                    END LOOP;
                                END IF; 
                            END IF;                               
                        Exception when others then
                            Errmsg := 'Error insert Outgo Installment > 1 ' || SQLERRM ;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;   
                        END;                                                    
                        --end cek installment outgo >1 
                    END IF;
                EXCEPTION WHEN OTHERS THEN
                    Errmsg := 'Error on Object Gathering : Failed to get data  : '|| SQLERRM || ' '|| dbms_utility.format_error_backtrace;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END;
                                
            Exception when others then
                Errmsg := 'Error On Gathering Object : ' || SQLERRM || dbms_utility.format_error_backtrace;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            end ;
            DBMS_OUTPUT.Put_Line('Selesai Object : ' || CURRENT_TIMESTAMP);
            -- End Object Gathering  ( PersonList , VehicleList , PropertyList , AnekaList )
            -- Bentuk Payment Final
            if vTotalYear <=1 and Quotation_GroupPanel not in ('007') then  --  ganti grouppanel XXX dengan kode MBU      
                Coverage_BeginDate :=  Policy_StartDateTime;
                Coverage_EndDate :=  Policy_EndDateTime;
                if Policy_ProdDateTime > Coverage_BeginDate then
                    Yearly_ProdDateTime := Policy_ProdDateTime;
                else
                    Yearly_ProdDateTime := Coverage_BeginDate;
                END IF;
                if trunc(sysdate) > trunc(Yearly_ProdDateTime) then
                    Yearly_ProdDateTime := sysdate;
                END IF;
                
                INSERT INTO POOLDATA.CoverageYearly (CaseID,ProdDate,BeginDate, EndDate, CurrentYear, Premium, Discount, Outgo, Comm , OC ) values 
                    (Policy_CaseID,Yearly_ProdDateTime, Coverage_BeginDate, Coverage_EndDate, 1, NVL(Payment_PremiTotal,0), Payment_Diskon, Payment_Commision+ Payment_OC, Payment_Commision, Payment_OC);
            elsif vTotalYear >1 and Quotation_GroupPanel not in ('007') then
                --DBMS_OUTPUT.Put_Line('Coverage Gathering NON MBU');           
                CNTDAYS := Policy_EndDateTime - Policy_StartDateTime;                
                Coverage_SisaPremiYearly := Payment_PremiTotal;
                Coverage_SisaDiscountYearly := Payment_Diskon;
                Coverage_SisaOutgoYearly := Payment_Commision + Payment_OC;
                Coverage_SisaCommYearly := Payment_Commision;
                Coverage_SisaOCYearly := Payment_OC;
                Coverage_BeginDate :=  Policy_StartDateTime;
                Coverage_EndDate :=  Policy_EndDateTime;
                 FOR x IN 1..vTotalYear  LOOP                    
                    if vTotalYear =1 then
                        Coverage_BeginDate :=  Policy_StartDateTime;
                        Coverage_EndDate :=  Policy_EndDateTime;
                    else 
                        if x = 1 then
                            Coverage_BeginDate :=  Policy_StartDateTime;
                            Coverage_EndDate :=  add_months(Coverage_BeginDate,12);
                        else
                            Coverage_BeginDate :=  add_months(Coverage_BeginDate,12);
                            if Policy_EndDateTime < add_months(Coverage_BeginDate,12) then
                                Coverage_EndDate :=  Policy_EndDateTime;
                            else
                                Coverage_EndDate :=  add_months(Coverage_BeginDate,12);
                            END IF;
                        END IF;
                    END IF;
                    if Policy_ProdDateTime > Coverage_BeginDate then
                        Yearly_ProdDateTime := Policy_ProdDateTime;
                    else
                        Yearly_ProdDateTime := Coverage_BeginDate;
                    END IF;
                    if trunc(sysdate) > trunc(Yearly_ProdDateTime) then
                        Yearly_ProdDateTime := sysdate;
                    END IF;
                    CNTDAYSYEAR := Coverage_EndDate-Coverage_BeginDate;
                    Coverage_PremiYearly := round(Payment_PremiTotal * CNTDAYSYEAR / CNTDAYS,2);
                    Coverage_DiscountYearly := round(Payment_Diskon * CNTDAYSYEAR / CNTDAYS,2);
                    Coverage_OutgoYearly := round((Payment_Commision + Payment_OC) * CNTDAYSYEAR / CNTDAYS,2);
                    Coverage_CommYearly := round(Payment_Commision * CNTDAYSYEAR / CNTDAYS,2);
                    Coverage_OCYearly := round(Payment_OC * CNTDAYSYEAR / CNTDAYS,2);
                    
                    if x = vTotalYear then
                        INSERT INTO POOLDATA.CoverageYearly (CaseID,proddate,BeginDate, EndDate,  CurrentYear, Premium , Discount, Comm , OC, Outgo) values 
                            (Policy_CaseID, Yearly_ProdDateTime, Coverage_BeginDate, Coverage_EndDate, x, NVL(Coverage_SisaPremiYearly,0),Coverage_SisaDiscountYearly,Coverage_SisaCommYearly,Coverage_SisaOCYearly,Coverage_SisaOutgoYearly);
                    else
                        Coverage_EndDate := Coverage_EndDate ;
                        Coverage_SisaPremiYearly  := Coverage_SisaPremiYearly - Coverage_PremiYearly;
                        Coverage_SisaDiscountYearly := Coverage_SisaDiscountYearly - Coverage_DiscountYearly;
                        Coverage_SisaOutgoYearly :=  Coverage_SisaOutgoYearly - Coverage_OutgoYearly ;
                        Coverage_SisaCommYearly := Coverage_SisaCommYearly - Coverage_CommYearly;
                        Coverage_SisaOCYearly := Coverage_SisaOCYearly - Coverage_OCYearly;
                        INSERT INTO POOLDATA.CoverageYearly (CaseID,proddate,BeginDate, EndDate, CurrentYear, Premium, Discount, Comm , OC, Outgo ) values 
                            (Policy_CaseID,Yearly_ProdDateTime,Coverage_BeginDate, Coverage_EndDate,  x, NVL(Coverage_PremiYearly,0),Coverage_DiscountYearly,Coverage_CommYearly,Coverage_OCYearly,Coverage_OutgoYearly);
                    END IF;
                 end loop;
            END IF;
            
            BEGIN
                IF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK NB-%' OR Policy_CaseID LIKE 'ASM-FW-GISFW-WORK RNW-%' THEN
                    AgingAmount := SumPaymentTotal;
                ELSE
                    SELECT COLLECTION.f_sisa_aging_new@asmd.sinarmas.co.id(Policy_PolicyNo, SumPaymentTotal, Policy_CaseID) INTO AgingAmount FROM dual;  
                END IF;
            EXCEPTION WHEN OTHERS THEN
                AgingAmount:=0;
            END;
            
            -- VALIDATE OUTGO & COVERAGE
            BEGIN
                SELECT COUNT(0) INTO CNTOUTGO2 FROM POOLDATA.T_POOL_OUTGO WHERE CASEID = vvIDPEGA;
                IF CNTOUTGO2 > 0 
                 AND (Policy_TypeOfCoins NOT IN ('1','F') OR (Quotation_BusinessCode = '03' AND SourceBiz = '14757')) 
                 AND Policy_PolicyNo NOT IN ('12300000017155', '12300001387622', '12300001394554', '12100000303243',
                                             '12100000303245', '12300002598452', '12300002867687')
                 THEN
                    SELECT NVL(SUM(outgoamount),0) INTO outgoamount_ttl FROM POOLDATA.t_pool_outgo WHERE policyno = Policy_PolicyNo AND caseid = Policy_CaseID;
                    SELECT NVL(Payment_Commision,0) + NVL(Payment_OC,0) into TotalCommOC FROM DUAL;
                    IF ABS(outgoamount_ttl - TotalCommOC) > 1 
                     --AND (Quotation_DetailEDM IS NULL OR Quotation_DetailEDM <> '3') 
                     THEN
                        Errmsg := 'Total OutgoAmount Coverage :'||outgoamount_ttl || ' tidak sama dengan Payment.Commision :' || TotalCommOC;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END IF;
                END IF;
            END;
            -- END VALIDATE OUTGO & COVERAGE
            
            -- MATERAI POLIS
            BEGIN
                if Policy_CaseID LIKE 'ASM-FW-GISFW-WORK NB-%' THEN
                    prodkeDetSales := '00001';
                    POOLDATA.PKG_COUNTER_PRODUCTION.GET_SPPA_SEQ(sppano);
--                    BEGIN 
--                        SELECT msp_no_spak INTO sppano FROM mst_det_sales@asmd.sinarmas.co.id WHERE mds_no_polis = Policy_PolicyNo and mds_prod_ke = prodkeDetSales;
--                    EXCEPTION WHEN OTHERS THEN
--                        sppano := NULL;
--                    END;
--                    IF sppano IS NULL THEN
--                        POOLDATA.PKG_COUNTER_PRODUCTION.GET_SPPA_SEQ(sppano);
--                    END IF;
--                ELSIF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK EDM-%' AND Policy_IsDeclaration <> 'True' THEN
--                    BEGIN
--                        SELECT msp_no_spak, SUBSTR(MAX(mds_prod_ke),0,3) || LPAD(SUBSTR(MAX(mds_prod_ke),4,2)+1,2,0)
--                          INTO sppano, prodkeDetSales FROM mst_det_sales@asmd.sinarmas.co.id WHERE mds_no_polis = Policy_PolicyNo
--                          GROUP BY msp_no_spak;
--                    EXCEPTION WHEN OTHERS THEN
--                        Errmsg := 'Gagal get nospak & prodke EDM :' || SQLERRM || 'IDPEGA : ' || vvIDPEGA;
--                        ROLLBACK;
--                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                        COMMIT;
--                        RETURN;
--                    END;
--                    
--                    BEGIN
--                        SELECT COUNT(1) INTO cntprodke FROM mst_det_sales@asmd.sinarmas.co.id WHERE msp_no_spak = sppano AND mds_prod_ke = prodkeDetSales; 
--                        WHILE cntprodke > 0 LOOP
--                            prodkeDetSales := SUBSTR(prodkeDetSales,0,3) || LPAD(SUBSTR(prodkeDetSales,4,2)+1,2,0);
--                            SELECT COUNT(1) INTO cntprodke FROM mst_det_sales@asmd.sinarmas.co.id WHERE msp_no_spak = sppano AND mds_prod_ke = prodkeDetSales; 
--                        END LOOP;
--                    EXCEPTION WHEN OTHERS THEN
--                        Errmsg := 'Gagal get nospak & prodke EDM 2' || SQLERRM || 'IDPEGA : ' || vvIDPEGA;
--                        ROLLBACK;
--                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                        COMMIT;
--                        RETURN;
--                    END;
                ELSIF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK RNW-%' THEN --Quotation_OldPolicyNo
                    BEGIN
                        SELECT msp_no_spak INTO sppano FROM mst_det_sales@asmd.sinarmas.co.id WHERE mds_no_polis = Quotation_OldPolicyNO AND ljp_id IN ('2','1') AND ROWNUM = 1;
                        SELECT MAX(mds_prod_ke) INTO prodkeDetSales FROM mst_det_sales@asmd.sinarmas.co.id WHERE msp_no_spak = sppano AND ljp_id IN ('2','1');
                        prodkeDetSales := LPAD(TO_NUMBER(prodkeDetSales)+100,5,0);
                    EXCEPTION WHEN OTHERS THEN
                        Errmsg := 'Gagal get nospak & prodke RNW :' || SQLERRM || 'IDPEGA : ' || vvIDPEGA||', No Polis Lama : '||Quotation_OldPolicyNO;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;
                    END;
                END IF;
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Gagal get NoSpak' || SQLERRM || 'IDPEGA : ' || vvIDPEGA;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
            
--            BEGIN
--                SELECT COUNT(1) into cntMaterai from collection.mst_materai@asmd.sinarmas.co.id WHERE msp_no_spak = sppano and prod_ke = prodkeDetSales;
--                if cntMaterai = 0 and Policy_IsDeclaration <> 'True' and Policy_TypeOfCoins <> '1' then
--                    FOR var_installment IN curr_pool_installment(Policy_CaseID, Policy_PolicyNo) loop
--                        tempinstallmentno  := var_installment.installmentno;
--                        tempinstallmentno  := lpad(tempinstallmentno, 2, '0');
--                        
--                        if tempinstallmentno = '01' then
--                            INSERT INTO collection.mst_materai@asmd.sinarmas.co.id (msp_no_spak, prod_ke, mns_cicil_ke, bi_mat_polis, bi_mat_kwi) 
--                                values(sppano, prodkeDetSales, tempinstallmentno, Payment_StampPolicy, Payment_StampReceipts);
--                        else
--                            INSERT INTO collection.mst_materai@asmd.sinarmas.co.id (msp_no_spak, prod_ke, mns_cicil_ke, bi_mat_polis, bi_mat_kwi) 
--                                values(sppano, prodkeDetSales, tempinstallmentno, 0, 0);
--                        END IF;
--                    end loop;
--                END IF;
--            EXCEPTION WHEN OTHERS THEN
--                Errmsg := 'Gagal cek row mst_materai ' || SQLERRM ||dbms_utility.format_error_backtrace;
--                ROLLBACK;
--                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                COMMIT;
--                RETURN;
--            END;
            
--            BEGIN
--                if Payment_FlagMaterai = '0' and (Payment_StampPolicy > 0 or  Payment_StampReceipts > 0) and Policy_TypeOfCoins <> '1' 
--                    AND (Policy_CaseID like 'ASM-FW-GISFW-WORK NB-%' or Policy_CaseID like 'ASM-FW-GISFW-WORK RNW-%')  then
--                    COLLECTION.PKG_MATERAI_NEW.PENCATATAN_MATERAI_POLIS@asmd.sinarmas.co.id
--                        (sppano, prodkeDetSales, Policy_PolicyNo, LkuId, sysdate, Errmsg, MatNoIjin);
--  
--                    if Errmsg is not null and (Errmsg <> 'Error, data materai sudah pernah ada di data' OR Errmsg <> 'Deposit anda sudah tidak mencukupi, harap isi kembali!')
--                    then
--                        Errmsg := 'Gagal get no ijin materai: ' || Errmsg;
--                        ROLLBACK;
--                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                        COMMIT;
--                        RETURN;
--                    END IF;
--
--                END IF;               
--            EXCEPTION WHEN OTHERS THEN
--                Errmsg := 'Gagal get no ijin materai 2' || SQLERRM ;
--                ROLLBACK;
--                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
--                COMMIT;
--                RETURN;
--            END;
            --END MATERAI POLIS
            
            DBMS_OUTPUT.Put_Line('Update Outgo : ' || CURRENT_TIMESTAMP);
            BEGIN
                FOR var_outgo IN curr_pool_outgo(Policy_CaseID, Policy_PolicyNo) LOOP
                    tempocid            := var_outgo.ocidold;
                    Outgo_TypeOfOutgo   := var_outgo.typeofoutgo;
                    Outgo_PercentOutgo  := var_outgo.percentoutgo;
                    Outgo_OutgoAmountv  := var_outgo.outgoamount;
                    Outgo_PPHType       := var_outgo.pphtype;
                    tempinstallmentno   := var_outgo.installmentno;
                    pctppn              := var_outgo.ppnpercent;
                    pctpph              := var_outgo.pphpercent;
                    cntpctpph           := 0;
                    outgo_pphamount     := 0;
                    outgo_ppnamount     := 0;
                    outgo_totalcomm     := 0;
                    Outgo_PPHNote       := NULL;
                    Outgo_PPNNote       := NULL;
                    pphnote             := NULL;
                    BEGIN
                        Outgo_GrossCommision    := Outgo_OutgoAmountv;
                        Outgo_CommisionOri      := Outgo_OutgoAmountv;
                        BEGIN
                            SELECT CASE 
                                WHEN VAT_CALC = 'exc' THEN 1 
                                WHEN VAT_CALC = 'inc' THEN 0 END
                             INTO vPPNSTATUS 
                            FROM GENERAL.LST_AGEN@ASMD.SINARMAS.CO.ID WHERE LAG_AGEN_ID = tempocid;
                        EXCEPTION WHEN NO_DATA_FOUND THEN
                            vPPNSTATUS := 0;
                        END;
                        
                        BEGIN
                            SELECT A.LEADER0, POOLDATA.getnewID(A.LEADER0,'M_AGENT','ID','kosong', 'kosong', 'OLDID'),
                                A.LEADER1, POOLDATA.getnewID(A.LEADER1,'M_AGENT','ID','kosong', 'kosong', 'OLDID'), B.NPWP
                             INTO vLeader0, vOldLeader0, vLeader1, vOldLeader1, vNPWP
                            FROM POOLDATA.T_AGENT A, POOLDATA.T_MCLIENT B WHERE A.CLIENTID = B.ID AND A.OLDID = tempocid AND A.LEADER1 IS NOT NULL;
                        EXCEPTION WHEN NO_DATA_FOUND THEN
                            vOldLeader0 := '00000';
                            vOldLeader1 := '00000';
                        END;
                        
                        BEGIN
                            -- untuk EDM BROKER
                            IF Policy_CaseID LIKE 'ASM-FW-GISFW-WORK EDM-%' AND vOldLeader0 = '00003' AND Outgo_OutgoAmountv < 0 THEN
                                SELECT COUNT(0) INTO cntComm FROM T_POOL_OUTGO WHERE POLICYNO = Policy_PolicyNo AND OCIDOLD = tempocid AND OUTGOAMOUNT > 0 AND PCTPPN = 0.022 AND PPNSTATUS IS NOT NULL;
                                IF cntComm > 0 THEN
                                    SELECT A.PPNSTATUS INTO vPPNSTATUS FROM T_POOL_OUTGO A, JSON_POLIS B 
                                    WHERE A.POLICYNO = B.NOPOLIS AND A.CASEID = B.IDPEGA AND A.OUTGOAMOUNT > 0 AND A.OCIDOLD = tempocid AND A.POLICYNO = Policy_PolicyNo
                                    AND B.PRODKE = (SELECT MAX(D.PRODKE) FROM T_POOL_OUTGO C, JSON_POLIS D WHERE C.POLICYNO = D.NOPOLIS AND C.CASEID = D.IDPEGA AND C.OUTGOAMOUNT > 0 AND C.OCIDOLD = tempocid AND C.POLICYNO = Policy_PolicyNo);
                                    SELECT NVL(GENERAL.PKG_CAL_COMMISION.COMM_GROSS_OLD@ASMD.sinarmas.co.id(Outgo_OutgoAmountv,tempocid, vPPNSTATUS),0) INTO Outgo_GrossCommisionNew FROM DUAL;
                                ELSE
                                    SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_GET_COMM_GROSS2@ASMD.sinarmas.co.id(Outgo_OutgoAmountv,tempocid, vOldLeader0, vOldLeader1),0) INTO Outgo_GrossCommisionNew FROM DUAL;
                                END IF;
                            ELSE
                                SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_GET_COMM_GROSS2@ASMD.sinarmas.co.id(Outgo_OutgoAmountv,tempocid, vOldLeader0, vOldLeader1),0) INTO Outgo_GrossCommisionNew FROM DUAL;
                            END IF;
                        EXCEPTION WHEN OTHERS THEN
                            Errmsg := 'Error set nilai Komisi Gross: ' || SQLERRM || dbms_utility.format_error_backtrace;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;  
                        END;
                        
                        BEGIN
                            IF Outgo_PPHType = 'Salah PPH' AND Policy_TypeOfCoins <> '1' THEN
                                Errmsg := 'Error get Percent PPH!';
                                ROLLBACK;
                                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                                COMMIT;
                                RETURN;        
                            END IF;
                            IF Outgo_PPHType = 'PPH21' THEN
                                cntpctpph := pctpph*0.5;
                                outgo_pphamount := Outgo_GrossCommisionNew*cntpctpph;
                                IF vNPWP IS NULL THEN
                                    SELECT GENERAL.PKG_CAL_COMMISION.F_GET_NPWP@ASMD.sinarmas.co.id(tempocid) INTO vNPWP FROM DUAL;
                                END IF;
                                IF vNPWP IS NULL THEN
                                    cekNPWP := '0';
                                ELSE
                                    cekNPWP := '1';
                                END IF;
                                COLLECTION.NILAI_PPH_PROGRESIF@ASMD.SINARMAS.CO.ID(LKUID, Outgo_GrossCommisionNew, cekNPWP, outgo_pphamount, Outgo_PPHNote, lppid);
                            ELSE
                                IF Quotation_EdmType = '2' AND Quotation_SobLeader2 = '10017990' THEN
                                    pctpph := 0;
                                END IF;
                                cntpctpph := pctpph;
                                outgo_pphamount := Outgo_GrossCommisionNew*cntpctpph;
                            END IF;
                        EXCEPTION WHEN OTHERS THEN
                            Errmsg := 'Error set nilai PPh Komisi: ' || SQLERRM || dbms_utility.format_error_backtrace;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;  
                        END;
                        
                        BEGIN
                            IF pctppn = 0.012 THEN
                                Outgo_PPNNote := '(' || REPLACE(REPLACE(REPLACE(TO_CHAR(Outgo_OutgoAmountv, 'FM99,999,999,999.00'), ',', '#'), '.', ','), '#', '.') || ' x 10% x (11/12) x 12%)';
                            ELSIF pctppn = 0.024 THEN
                                Outgo_PPNNote := '(' || REPLACE(REPLACE(REPLACE(TO_CHAR(Outgo_OutgoAmountv, 'FM99,999,999,999.00'), ',', '#'), '.', ','), '#', '.') || ' x 20% x (11/12) x 12%)';
                            ELSIF pctppn = 0.12 THEN
                                Outgo_PPNNote := '(' || REPLACE(REPLACE(REPLACE(TO_CHAR(Outgo_GrossCommision, 'FM99,999,999,999.00'), ',', '#'), '.', ','), '#', '.') || ' x (11/12) x 12%)';
                            END IF;
                            --outgo_ppnamount := Outgo_GrossCommisionNew*pctppn;
                            SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_HITUNG_PPN_NEW@ASMD.sinarmas.co.id(Outgo_OutgoAmountv,tempocid, vOldLeader0, vOldLeader1),0) 
                              INTO Outgo_PPNAmount FROM DUAL;
                        EXCEPTION WHEN OTHERS THEN
                            Errmsg := 'Error set nilai PPN Note: ' || SQLERRM || dbms_utility.format_error_backtrace;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;  
                        END;
                        
                        BEGIN
                            IF Outgo_PPHNote IS NOT NULL THEN
                                Outgo_PPHNote := '(' || TRIM(REGEXP_REPLACE(REGEXP_SUBSTR(Outgo_PPHNote, '[0-9\.,% x-]+'), '(-?)(\d+),(\d+)\.(\d+)', '\1\2.\3,\4')) || ')';
                            ELSIF Outgo_PPHNote IS NULL AND pctpph > 0 THEN
                                Outgo_PPHNote := '(' || REPLACE(REPLACE(REPLACE(TO_CHAR(Outgo_OutgoAmountv, 'FM99,999,999,999.00'), ',', '#'), '.', ','), '#', '.') || ' x ' || TO_CHAR(pctpph*100) || '%)';  
                            END IF;
                        EXCEPTION WHEN OTHERS THEN
                            Errmsg := 'Error set nilai PPH Note: ' || SQLERRM || dbms_utility.format_error_backtrace;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;  
                        END;
                        
                        
                        BEGIN
                            SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_GET_COMM_NETT2@ASMD.sinarmas.co.id(Outgo_OutgoAmountv,tempocid, vOldLeader0, vOldLeader1),0) INTO outgo_totalcomm FROM DUAL;
                            SELECT NVL(GENERAL.PKG_CAL_COMMISION.F_GET_COMM_DPP@ASMD.sinarmas.co.id(Outgo_OutgoAmountv,tempocid, vOldLeader0, vOldLeader1),0) INTO Outgo_CommisionDPP FROM DUAL;
                        EXCEPTION WHEN OTHERS THEN
                            Errmsg := 'Error set nilai Komisi Nett: ' || SQLERRM || dbms_utility.format_error_backtrace;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;  
                        END;
--                        IF Outgo_PPHType = 'PPH21' THEN
--                            SELECT COUNT(0) INTO cntAgenProgresif FROM GL.MASTER_PPH_PROGRESSIF@ASMD.SINARMAS.CO.ID WHERE KODE_SB = tempocid;
--                            IF cntAgenProgresif > 0 AND (Policy_CaseID LIKE 'ASM-FW-GISFW-WORK NB-%' OR Policy_CaseID LIKE 'ASM-FW-GISFW-WORK RNW-%') THEN
--                                SELECT RATE_MASTER INTO pctrateprogresif FROM GL.MASTER_PPH_PROGRESSIF@ASMD.SINARMAS.CO.ID WHERE KODE_SB = tempocid;
--                                pctpph          := 0.5*pctrateprogresif;
--                                cntpctpph       := pctpph;
--                                outgo_pphamount := outgo_grosscommisionnew*cntpctpph;
--                                outgo_pphnote   := '50% X ' || pctrateprogresif*100 || '%';
--                                outgo_pphamount := outgo_grosscommisionnew*cntpctpph;
--                                outgo_ppnamount := outgo_grosscommisionnew*pctppn;
--                                outgo_totalcomm := outgo_grosscommisionnew-outgo_pphamount;
--                            END IF;
--                        END IF;
                        
                    EXCEPTION WHEN OTHERS THEN
                        Outgo_GrossCommisionNew := Outgo_OutgoAmountv;
                        outgo_pphamount := Outgo_GrossCommisionNew*cntpctpph;
                        outgo_ppnamount := Outgo_GrossCommisionNew*pctppn;
                        outgo_totalcomm := Outgo_GrossCommisionNew-outgo_pphamount+outgo_ppnamount;
                    END;    
                    
                    BEGIN  
                        IF Outgo_GrossCommision IS NOT NULL AND Outgo_CommisionDPP IS NULL THEN
                            Errmsg := 'Error set nilai Komisi DPP: ' || SQLERRM || dbms_utility.format_error_backtrace;
                            ROLLBACK;
                            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                            COMMIT;
                            RETURN;  
                        END IF;
                        UPDATE POOLDATA.T_POOL_OUTGO 
                           SET COMMISIONGROSS = Outgo_GrossCommision, PPNSTATUS = vPPNSTATUS, OUTGOAMOUNT = Outgo_GrossCommisionNew, PPHAMOUNT = outgo_pphamount, PPNAMOUNT = outgo_ppnamount, 
                               TOTALCOMM = outgo_totalcomm, T_POOL_OUTGO.PPNNOTE = Outgo_PPNNote, T_POOL_OUTGO.PPHNOTE = Outgo_PPHNote, T_POOL_OUTGO.PPHPERCENT = pctpph,
                               COMMISIONDPP = Outgo_CommisionDPP, COMMISIONORI = Outgo_CommisionOri
                         WHERE CASEID = Policy_CaseID AND OCIDOLD = tempocid AND TYPEOFOUTGO = Outgo_TypeOfOutgo AND INSTALLMENTNO = tempinstallmentno;
                    EXCEPTION WHEN OTHERS THEN
                        Errmsg := 'Error set nilai Komisi DPP: ' || SQLERRM || dbms_utility.format_error_backtrace;
                        ROLLBACK;
                        POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                        COMMIT;
                        RETURN;  
                    END;
                END LOOP;
            END;
                
            BEGIN
                IF (Quotation_SourceOfBusiness = '10001329' OR Quotation_SobLeader0 = '10000173') AND VIRTUALACCOUNTNO IS NULL THEN
                    Errmsg := 'Error Generate VA : VA NULL' ;
                    ROLLBACK;
                    POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                    COMMIT;
                    RETURN;
                END IF;
                INSERT INTO POOLDATA.T_POOL_POLICY (policyno, caseid, nova, novausd, namava, digitchecksum, AgingAmount, clientid, nospak, prodke)
                VALUES (TRIM(Policy_PolicyNo), Policy_CaseID, VIRTUALACCOUNTNO, VIRTUALACCOUNTUSDNO, VIRTUALACCOUNTNAME, DIGITCHECKSUM, AgingAmount, mclid, sppano, prodkeDetSales);   
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error on Insert T_POOL_POLICY: ' || SQLERRM ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;          
            END;
            
            SELECT  '{' ||  TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'ResponseMessage', 'OK', ','))               ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'PolicyNo', POLICYNO, ','))                  ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'DigitCheckSum', DIGITCHECKSUM, ','))        ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'StampNumber', STAMPNUMBER, ','))            ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'VirtualAccountNumber', NOVA, ','))          ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'VirtualAccountNumberUSD', NOVAUSD, ','))    ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'VirtualAccountName', NAMAVA, ','))          ||
                           TO_CLOB (
                           POOLDATA.jsonObjectWriter ( 'AgingAmount', '', ',',AGINGAMOUNT,'number2'))    ||
                           TO_CLOB ('"Payment":{') ||
                           TO_CLOB ('"ListInstallment":[') || 
                              POOLDATA.CONVERTINSTALLMENTPRODUKSI(a.CaseID)
                           || TO_CLOB (']')
                           || '},' || 
                           TO_CLOB ('"CoinsList":[') ||
                                POOLDATA.ConvertCoinsProduksi(a.CaseID)
                           || TO_CLOB (']')
                       || '}'  INTO RESPONSEMESSAGE FROM POOLDATA.T_POOL_POLICY a WHERE caseid = Policy_CaseID AND ROWNUM = 1; 
                       
            UPDATE POOLDATA.T_POOL_POLICY SET responsemsg = RESPONSEMESSAGE
            WHERE CaseID = TRIM(Policy_CaseID) And PolicyNo = TRIM(Policy_PolicyNo);
           
            BEGIN 
                IF Policy_IsDeclaration = 'True' AND Quotation_BusinessCode = '04' THEN
                    UPDATE POOLDATA.JSON_POLIS SET sts_konversi = 1, tgl_konversi = SYSDATE
                    WHERE idpega = Policy_CaseID AND nopolis = Policy_PolicyNo;
                ELSE
                    BEGIN
                        IF vJumlahObject > 50 OR Quotation_SourceOfBusiness = '10034413' THEN
                            IF vISB2B = '1' THEN
                                b2b_message := POOLDATA.queue_b2b_type(1,Policy_CaseID,'b2b');
                                DBMS_AQ.ENQUEUE(
                                    queue_name => 'POOLDATA.Q_PROCESB2B',
                                    enqueue_options => queue_options,
                                    message_properties => message_properties,
                                    payload => b2b_message,
                                    msgid => message_id);
                            
                            ELSIF visservice = '1' THEN
                                service_message := POOLDATA.queue_service_type(1,Policy_CaseID,'service');
                                DBMS_AQ.ENQUEUE(
                                    queue_name => 'POOLDATA.Q_PROCESSERVICE',
                                    enqueue_options => queue_options,
                                    message_properties => message_properties,
                                    payload => service_message,
                                    msgid => message_id);
                            
                            ELSIF visaro = '1' THEN
                                aro_message  := POOLDATA.queue_aro_type(1,Policy_CaseID,'aro');
                                DBMS_AQ.ENQUEUE(
                                    queue_name => 'POOLDATA.Q_PROCESARO',
                                    enqueue_options => queue_options,
                                    message_properties => message_properties,
                                    payload => aro_message,
                                    msgid => message_id);
                            ELSE
                                IF instr(Policy_CaseID,'EDM') > 0 THEN
                                    my_message := POOLDATA.queue_polis_type(1,Policy_CaseID,'edm');
                                ELSE
                                    my_message := POOLDATA.queue_polis_type(1,Policy_CaseID,'polis');
                                END IF;
                            
                                DBMS_AQ.ENQUEUE(
                                    queue_name => 'POOLDATA.Q_PROCESPOLIS',
                                    enqueue_options => queue_options,
                                    message_properties => message_properties,
                                    payload => my_message,
                                    msgid => message_id);
                            END IF;
                        ELSE
                            IF vISB2B = '1' THEN
                                POOLDATA.PROCESSQUEUEDIRECT (Policy_CaseID,'b2b');
                            ELSIF visservice = '1' THEN
                                POOLDATA.PROCESSQUEUEDIRECT (Policy_CaseID,'service');
                            ELSIF visaro = '1' THEN
                                POOLDATA.PROCESSQUEUEDIRECT (Policy_CaseID,'aro');
                            ELSE
                                IF INSTR(vvIDPEGA,'EDM') > 0 THEN
                                    POOLDATA.PROCESSQUEUEDIRECT (Policy_CaseID,'edm');
                                ELSE
                                    POOLDATA.PROCESSQUEUEDIRECT (Policy_CaseID,'polis');
                                END IF;
                            END IF;
                        END IF;
                    END;
                END IF;
            EXCEPTION WHEN OTHERS THEN
                Errmsg := 'Error cek polis Deklarasi: ' || SQLERRM ;
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END;
             
            ResponseMsg := RESPONSEMESSAGE;
            
            UPDATE POOLDATA.LOG_KONVERSI_PROD
             SET no_polis = Policy_PolicyNo, prod_ke = Policy_ProdKe, trace_message = RESPONSEMESSAGE, conversion_note = Errmsg
            WHERE id_pega = Policy_CaseID AND conversion_no = Conversion_No;
            
            SELECT COUNT(1) INTO cntConverted FROM POOLDATA.t_pool_policy WHERE CLIENTIDPEGA = Customer_ASMClientIDPEGA AND clientid <> mclid AND caseid <> Policy_CaseID;
            IF cntConverted > 1 THEN
                Errmsg := 'Nomor Client Sama, Coba Print Ulang ';
                ROLLBACK;
                POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
                COMMIT;
                RETURN;
            END IF;   
            
            --DELETE FROM POOLDATA.tempprocess WHERE caseid = vvIDPEGA; 
            COMMIT; 
        ELSE
            Errmsg := 'Json data Null';
            ROLLBACK;
            POOLDATA.UPDATE_LOG_KONVERSI(vpolicynocek, vvIDPEGA, Errmsg, ConversionNo);
            COMMIT;
            RETURN;
        END IF;
    EXCEPTION WHEN OTHERS THEN
        Errmsg := 'Error convert jsondata (Invalid JSON/No JSON Data): ' || SQLERRM || dbms_utility.format_error_backtrace;
        ROLLBACK;
        RETURN;
    END;
      
EXCEPTION WHEN OTHERS THEN
    Errmsg := 'Error Process: ' || SQLERRM;
    ROLLBACK;
    RETURN;
END;

/