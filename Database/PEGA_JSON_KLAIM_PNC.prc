CREATE OR REPLACE PROCEDURE          PEGA_JSON_KLAIM_PNC(No_Klaim IN VARCHAR2, PolisNO IN VARCHAR2, PegaID IN VARCHAR2, DataPega IN CLOB, 
                                                            PolisTreaty in VARCHAR2, ErrMsg OUT VARCHAR2, StsSimpan OUT NUMBER)
AS
id_count INTEGER;
Claimdata            JSON_OBJECT_T;
userteknis              Varchar(500);

BEGIN
    BEGIN
        SELECT count(1) INTO id_count FROM JSON_KLAIM WHERE IDPEGA = PegaID;
        EXCEPTION
        WHEN OTHERS THEN
                ErrMsg := 'SELECT JSON_KLAIM COUNT Error : ' || sqlerrm;
                ROLLBACK;
                RETURN;
    END;
    
    IF id_count > 0 THEN
            BEGIN
                UPDATE JSON_KLAIM SET tgl_input=sysdate,MNK_NO_KLAIM=No_Klaim, IDPROD=0,DATA_JSONBLOB = clobToBlob(DataPega) WHERE IDPEGA = PegaID;
            EXCEPTION
                WHEN OTHERS THEN
                ErrMsg := 'UPDATE JSON_KLAIM Error : ' || sqlerrm;
                StsSimpan := 0;
                ROLLBACK;
                RETURN;
                
                Claimdata       :=  JSON_OBJECT_T.parse(DataPega);
                userteknis      := Claimdata.get_string('UserTeknis');
                
                BEGIN
                UPDATE POOLDATA.PEGA_DASHBOARDPNC SET PIC = userteknis WHERE NOKLAIM = SUBSTR(PegaID,20,29);
                EXCEPTION
                WHEN OTHERS THEN
                ErrMsg := 'UPDATE PEGA_DASHBOARDPNC Error : ' || sqlerrm;
                StsSimpan := 0;
                ROLLBACK;
                RETURN;
                END;
        
            END;    
    ELSE
       
        BEGIN
            INSERT INTO JSON_KLAIM (MNK_NO_KLAIM, IDPEGA, NOPOLIS, IDPROD,DATA_JSONBLOB, NOPOLIS_TREATY,tgl_input) 
            VALUES (No_Klaim, PegaID, PolisNO,0,clobToBlob(DataPega),PolisTreaty,sysdate);
        EXCEPTION
            WHEN OTHERS THEN
                ErrMsg := 'JSON_KLAIM Error : ' || sqlerrm;
                StsSimpan := 0;
                ROLLBACK;
                RETURN;
        END;
    END IF;
     /*
    ErrMsg := 'Data sudah di simpan';
    StsSimpan := 1;
    COMMIT;
   -- di jalanin di procedure PEGA_CONVERT_PNC pakai job, supaya tidak lemot di pega
    BEGIN
        POOLDATA.PEGA_CONVERT_JSONKLAIM_PNC(PegaID, ErrMsg);
    END;
    */
    IF ErrMsg IS NULL THEN
        ErrMsg := 'Data sudah di simpan';
        StsSimpan := 1;
    ELSE
        StsSimpan := 0;
    END IF;
    
EXCEPTION
    WHEN OTHERS THEN
        ErrMsg := 'PEGA_JSON_KLAIM_PNC Error : ' || sqlerrm;
        StsSimpan := 0;
        ROLLBACK;
        RETURN;
END;

/