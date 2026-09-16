CREATE OR REPLACE PROCEDURE          PEGA_M_SURVEYORS(Datapega IN CLOB, IDPega IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_surv_ins varchar2 (5);

BEGIN

    IF IDPega = 'UnknownID' THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT M_SURVEYORS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_surv_ins := id_site || lpad(to_Char(M_SURVEYORS_SEQ.nextval),3,'0');

        BEGIN
            INSERT INTO POOLDATA.M_SURVEYORS(M_SURVEY_ID,JSON_DATA) VALUES(id_surv_ins,replace(DataPega,'UnknownID',id_surv_ins));
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_surv_ins;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT M_SURVEYORS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;


    ELSE

            BEGIN
                UPDATE POOLDATA.M_SURVEYORS SET JSON_DATA = DataPega WHERE M_SURVEY_ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE M_SURVEYORS Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;


    END IF;




EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition Pega_M_SURVEYORS Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/