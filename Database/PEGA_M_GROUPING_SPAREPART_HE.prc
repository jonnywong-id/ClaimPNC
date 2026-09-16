CREATE OR REPLACE PROCEDURE          PEGA_M_GROUPING_SPAREPART_HE(DataPega IN CLOB, IDPega IN VARCHAR2,  ErrMsg OUT VARCHAR2)
AS
id m_sparepart_he_vin_key.ID%type;
id_count NUMBER;

BEGIN  

    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT NVL(MAX(ID),0) + 1 INTO id_count from POOLDATA.m_sparepart_he_vin_key;
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_M_GROUPING_SPAREPART_HE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
        
        BEGIN
            INSERT INTO POOLDATA.m_sparepart_he_vin_key(ID,JSONDATA) VALUES(id_count,replace(DataPega,'UnknownID',id_count));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_count ;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT m_sparepart_he_vin_key Error : ' || sqlerrm || DataPega || id_count;
              ROLLBACK;
              RETURN;
        END;
        
    ELSE
        
            BEGIN
                UPDATE POOLDATA.m_sparepart_he_vin_key SET JSONDATA = DataPega WHERE ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_PANEL_HE_CLAIM Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;  
            
    END IF;
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition m_sparepart_he_vin_key Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/