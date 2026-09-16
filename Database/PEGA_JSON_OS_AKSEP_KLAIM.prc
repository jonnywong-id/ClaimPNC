CREATE OR REPLACE PROCEDURE          PEGA_JSON_OS_AKSEP_KLAIM(NoClaim IN VARCHAR2, PolisNO IN VARCHAR2, PegaID IN VARCHAR2, DataPega IN CLOB, StatusReject IN VARCHAR2, STS_PLA IN VARCHAR2, ErrMsg OUT VARCHAR2)
AS

BEGIN
    
    BEGIN
        INSERT INTO OS_AKSEPTASI_KLAIM (CASEID, NOCLAIM, DATA_JSON, NOPOLIS, TANGGAL, STS_REJECT,STS_DLA) VALUES (PegaID, NoClaim, DataPega, PolisNO, CURRENT_TIMESTAMP, StatusReject,STS_PLA);
        ErrMsg := 'Data sudah di simpan';
    EXCEPTION
        WHEN OTHERS THEN
            ErrMsg := 'OS_AKSEPTASI_KLAIM Error : ' || sqlerrm;
            ROLLBACK;
            RETURN;
    END;
        
EXCEPTION
    WHEN OTHERS THEN
        ErrMsg := 'PEGA_JSON_OS_AKSEP_KLAIM Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/