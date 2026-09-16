CREATE OR REPLACE PROCEDURE          PEGA_D_SURVEYORS(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_surv_ins POOLDATA.D_SURVEYORS.D_SURVEY_ID%type;

BEGIN

    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_M_SURVEYOR Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_surv_ins := id_site || lpad(to_Char(D_SURVEYORS_SEQ.nextval),6,'0');

        BEGIN
            INSERT INTO POOLDATA.D_SURVEYORS(D_SURVEY_ID,JSON_DATA) VALUES(id_surv_ins,replace(DataPega,'UnknownID',id_surv_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_surv_ins;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT PEGA_M_SURVEYOR Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
    ELSE

            BEGIN
                UPDATE POOLDATA.D_SURVEYORS SET JSON_DATA = DataPega WHERE D_SURVEY_ID = IDPega;
                ErrMsg := 'Data Sudah Disimpan dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE PEGA_M_SURVEYOR Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;
    END IF;

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_M_SURVEYOR Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/