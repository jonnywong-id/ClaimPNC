CREATE OR REPLACE PROCEDURE          PEGA_LST_DOC_TYPE(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_lst_doc_type POOLDATA.LST_DOC_TYPE.ID%type;

BEGIN

    IF IDPega = 'UnknownID' then

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT M_SITE_DATABASE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_lst_doc_type := id_site || lpad(to_Char(SET_LST_DOC_TYPE.nextval),4,'0');

        BEGIN
            INSERT INTO POOLDATA.LST_DOC_TYPE(ID,JSON_DATA) VALUES(id_lst_doc_type,replace(DataPega,'UnknownID',id_lst_doc_type));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_lst_doc_type;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT POOLDATA.LST_DOC_TYPE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

    ELSE

            BEGIN
                UPDATE POOLDATA.LST_DOC_TYPE SET JSON_DATA = DataPega WHERE ID = IDPega;
                ErrMsg := 'Data Sudah Disimpan dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE POOLDATA.LST_DOC_TYPE Error : ' || sqlerrm;
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