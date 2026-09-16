CREATE OR REPLACE PROCEDURE          PEGA_M_BENGKEL_HE(DataPega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_bengkel M_BENGKEL_HE.ID%type;

BEGIN
    
    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_M_BENGKEL_HE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
        
        id_bengkel := id_site || lpad(to_Char(BENGKEL_HE_SEQ.nextval),10,'0');
        
        BEGIN
            INSERT INTO POOLDATA.M_BENGKEL_HE(ID,JSONDATA) VALUES(id_bengkel,replace(DataPega,'UnknownID',id_bengkel));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_bengkel ;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_M_BENGKEL_HE Error : ' || sqlerrm || DataPega || id_bengkel;
              ROLLBACK;
              RETURN;
        END;
        
    ELSE
        
            BEGIN
                UPDATE POOLDATA.M_BENGKEL_HE SET JSONDATA = DataPega WHERE ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_M_BENGKEL_HE Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END; 
            
    END IF;
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_M_BENGKEL_HE Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/