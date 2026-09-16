CREATE OR REPLACE PROCEDURE          PEGA_M_CAUSE_OF_LOSS(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_mcouseofloss_ins varchar2 (4);

BEGIN

    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT M_SITE_DATABASE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_mcouseofloss_ins := id_site || lpad(to_Char(M_CAUSE_SEQ.nextval),3,'0');

        BEGIN
            INSERT INTO POOLDATA.M_CAUSE_OF_LOSS(M_COL_ID,JSON_DATA) VALUES(id_mcouseofloss_ins,replace(Datapega,'UnknownID',id_mcouseofloss_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_mcouseofloss_ins;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT POOLDATA.M_CAUSE_OF_LOSS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

    ELSE

            BEGIN
                UPDATE POOLDATA.M_CAUSE_OF_LOSS SET JSON_DATA = Datapega WHERE M_COL_ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE POOLDATA.M_CAUSE_OF_LOSS Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;

    END IF;

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_M_CAUSE_OF_LOSS Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/