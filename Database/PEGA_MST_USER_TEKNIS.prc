CREATE OR REPLACE PROCEDURE          PEGA_MST_USER_TEKNIS(IDPega IN VARCHAR2,TNAMA IN VARCHAR2,TIPEBISNIS IN VARCHAR2, tEmail1 IN VARCHAR2, 
                                                          TJOB IN NUMBER, STS IN VARCHAR2, TGROUP IN VARCHAR2, TJOB2 IN NUMBER,
                                                          tATASAN IN VARCHAR2, ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_mst_user_teknis POOLDATA.mst_user_teknik.OPERATOR_ID%type;
temp_operator varchar2(100);

BEGIN
    BEGIN
        SELECT operator_id into temp_operator from mst_user_teknik where operator_id = IDPega;
    EXCEPTION
        WHEN NO_DATA_FOUND
        THEN
          temp_operator := NULL;
        WHEN OTHERS THEN
        ErrMsg := 'SELECT TEMP_OPERATOR Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
    END;

    IF temp_operator is NULL then

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT M_SITE_DATABASE Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_mst_user_teknis := id_site || lpad(to_Char(MST_USER_TEKNIS_SEQ.nextval),6,'0');

        BEGIN
            INSERT INTO POOLDATA.mst_user_teknik(OPERATOR_ID,COUNTER_QUOTA, TYPE_BUSINESS, EMAIL, STS_AKTIF, TEAM_GROUP, ATASAN,COUNTER_QUOTA2,MCL_NAME) 
            VALUES(IDPega, TJOB, TIPEBISNIS, tEmail1, STS, TGROUP, tATASAN,TJOB2,TNAMA);
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || IDPega;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT POOLDATA.mst_user_teknik Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

    ELSE

            BEGIN
                UPDATE POOLDATA.mst_user_teknik SET OPERATOR_ID = IDPega,
                                                    COUNTER_QUOTA = TJOB, 
                                                    COUNTER_QUOTA2 = TJOB2,
                                                    TYPE_BUSINESS = TIPEBISNIS, 
                                                    EMAIL = tEmail1, 
                                                    STS_AKTIF =STS, 
                                                    TEAM_GROUP = TGROUP, 
                                                    ATASAN =tATASAN,
                                                    MCL_NAME= TNAMA
                WHERE OPERATOR_ID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE POOLDATA.mst_user_teknik Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;

    END IF;

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_MST_USER_TEKNIS Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/