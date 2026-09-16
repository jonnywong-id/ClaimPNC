CREATE OR REPLACE PROCEDURE          PEGA_LST_DET_TYPE_DOC(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_detype_ins POOLDATA.LST_DET_TYPE_DOC.ID%type;

BEGIN

    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_LST_DOC_TYPE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_detype_ins := id_site || lpad(to_Char(LST_DET_TYPE_DOC_SEQ.nextval),4,'0');

        BEGIN
            INSERT INTO POOLDATA.LST_DET_TYPE_DOC(ID,JSON_DATA) VALUES(id_detype_ins,replace(DataPega,'UnknownID',id_detype_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_detype_ins;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_LST_DOC_TYPE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
    ELSE

            BEGIN
                UPDATE POOLDATA.LST_DET_TYPE_DOC SET JSON_DATA = DataPega WHERE ID = IDPega;
                ErrMsg := 'Data Sudah Disimpan dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_LST_DOC_TYPE Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;
    END IF;

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_LST_DOC_TYPE Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/