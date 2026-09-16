CREATE OR REPLACE PROCEDURE          InsertMasterRecoveryKlaim(tidmaster in number,tnamaprincipal in varchar2,ttahun in varchar2,tnilaiklaim in number,tnilairecovery in number,tPEMBAYARAN in number,tsisa in number,tketerangan in varchar2,tposisi in varchar2,
                                                               tdokumeid in varchar2,tNOVA in varchar2,tUSERNAME in varchar2,tCLIENTID in varchar2,tJSON_POLIS clob,tNOHPLL in varchar2,tNOPOLIS in varchar2, tLBUID in varchar2, tLDCID in varchar2, tLAGAGENID in varchar2, tLMOID in varchar2, ErrMsg out varchar2)
AS
    count_master number;
                                                                
BEGIN
    
    select count(1) into count_master from POOLDATA.MST_RECOVERY_ASM_PENJAMINAN a where A.BATCH=tidmaster;
    
    IF count_master<=0 then
        insert into POOLDATA.MST_RECOVERY_ASM_PENJAMINAN(
            BATCH
            ,NAMAPRINCIPAL
            ,TAHUN
            ,NILAIKLAIM
            ,NILAIRECOVERY
            ,PEMBAYARAN
            ,SISAKLAIM
            ,KETERANGAN
            ,POSISIKASUS,DOKUMENID,NOVA,USERNAME,CLIENTID,JSON_POLIS,NOHPLL,NOPOLIS,LBU_ID,LDC_ID,LAG_AGEN_ID,LMO_ID)
        VALUES(tidmaster,tnamaprincipal,ttahun,tnilaiklaim,tnilairecovery,tPEMBAYARAN,tsisa,tketerangan,tposisi,tdokumeid,tNOVA,tUSERNAME,tCLIENTID,tJSON_POLIS,tNOHPLL,tNOPOLIS,tLBUID,tLDCID,tLAGAGENID,tLMOID );
        commit;
    
    END IF;

    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT DATA RECOVERY : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/