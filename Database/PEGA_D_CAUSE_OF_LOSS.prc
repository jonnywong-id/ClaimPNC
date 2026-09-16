CREATE OR REPLACE PROCEDURE          PEGA_D_CAUSE_OF_LOSS(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_dcol_ins POOLDATA.D_CAUSE_OF_LOSS.D_COL_ID%type;

BEGIN

    IF IDPega = 'UnknownID' then

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_D_CAUSE_OF_LOSS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_dcol_ins := id_site || lpad(to_Char(D_CAUSE_SEQ.nextval),4,'0');

        BEGIN
            INSERT INTO POOLDATA.D_CAUSE_OF_LOSS(D_COL_ID,JSONDATA) VALUES(id_dcol_ins,replace(DataPega,'UnknownID',id_dcol_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_dcol_ins;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_D_CAUSE_OF_LOSS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
    ELSE

            BEGIN
                UPDATE POOLDATA.D_CAUSE_OF_LOSS SET JSONDATA = DataPega WHERE D_COL_ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_D_CAUSE_OF_LOSS Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;
    END IF;


EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_D_CAUSE_OF_LOSS Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/