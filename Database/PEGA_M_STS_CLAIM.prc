CREATE OR REPLACE PROCEDURE          PEGA_M_STS_CLAIM(DataPega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_stclaim_ins M_STS_CLAIM.LSC_ID%type;

BEGIN
    
    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_M_STS_CLAIM Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
        
        id_stclaim_ins := id_site || lpad(to_Char(M_STS_CLAIM_SEQ.nextval),3,'0');
        
        BEGIN
            INSERT INTO POOLDATA.M_STS_CLAIM(LSC_ID,JSONDATA) VALUES(id_stclaim_ins,replace(DataPega,'UnknownID',id_stclaim_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_stclaim_ins ;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_M_STS_CLAIM Error : ' || sqlerrm || DataPega || id_stclaim_ins;
              ROLLBACK;
              RETURN;
        END;
        
    ELSE
        
            BEGIN
                UPDATE POOLDATA.M_STS_CLAIM SET JSONDATA = DataPega WHERE LSC_ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_M_STS_CLAIM Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END; 
            
    END IF;
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_M_STS_CLAIM Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/