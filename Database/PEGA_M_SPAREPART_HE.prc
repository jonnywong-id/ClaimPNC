CREATE OR REPLACE PROCEDURE          PEGA_M_SPAREPART_HE(DataPega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id M_SPAREPART_HE_BU.ID%type;

BEGIN
    
    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_M_SPAREPART_HE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
        
        id := id_site || lpad(to_Char(SPAREPART_HE_SEQ.nextval),10,'0');
        
        BEGIN
            INSERT INTO POOLDATA.M_SPAREPART_HE_BU(ID,JSONDATA) VALUES(id,replace(DataPega,'UnknownID',id));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id ;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_M_SPAREPART_HE Error : ' || sqlerrm || DataPega || id;
              ROLLBACK;
              RETURN;
        END;
        
    ELSE
        
            BEGIN
                UPDATE POOLDATA.M_SPAREPART_HE_BU SET JSONDATA = DataPega WHERE ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_M_SPAREPART_HE Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END; 
            
    END IF;
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_M_SPAREPART_HE Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/