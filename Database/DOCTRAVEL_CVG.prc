CREATE OR REPLACE PROCEDURE DOCTRAVEL_CVG (NamaDokumen1 in VARCHAR, IDPega in VARCHAR, format OUT VARCHAR, ErrMsg OUT VARCHAR)
AS
formatErr varchar2(500);
id_site M_SITE_DATABASE.ID%type;
id_docTravel varchar(8);
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

        id_docTravel := id_site || lpad(to_Char(DOCTRAVEL_SEQ.nextval),5,'0');

        BEGIN
            INSERT INTO POOLDATA.M_DOCTRAVEL(DOCID,NAMADOKUMEN) VALUES(id_docTravel,NamaDokumen1);
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || id_docTravel;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT M_DOCTRAVEL Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;


    ELSE

            BEGIN
                UPDATE POOLDATA.M_DOCTRAVEL SET NAMADOKUMEN = NamaDokumen1 WHERE DOCID = IDPega;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || IDPega;
                COMMIT;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE M_DOCTRAVEL Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;


    END IF;
    
    EXCEPTION WHEN OTHERS THEN
    format := null;
    formatErr := sqlerrm;
    rollback;
    RETURN;
END;

/