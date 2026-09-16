CREATE OR REPLACE PROCEDURE          PEGA_M_PANEL_HE(DataPega IN CLOB, IDPanel IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_panel_he_ins M_PANEL_HE.ID%type;


BEGIN
    
    IF IDPanel = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_PANEL_HE_CLAIM Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
        
        id_panel_he_ins := id_site || lpad(to_Char(PANEL_HE_SEQ.nextval),6,'0');
        
        BEGIN
            INSERT INTO POOLDATA.M_PANEL_HE(ID,JSONDATA) VALUES(id_panel_he_ins,replace(DataPega,'UnknownID',id_panel_he_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_panel_he_ins ;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_PANEL_HE_CLAIM Error : ' || sqlerrm || DataPega || id_panel_he_ins;
              ROLLBACK;
              RETURN;
        END;
        
    ELSE
        
            BEGIN
                UPDATE POOLDATA.M_PANEL_HE SET JSONDATA = DataPega WHERE ID = IDPanel;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPanel;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_PANEL_HE_CLAIM Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END; 
            
    END IF;
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_PANEL_HE_CLAIM Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/