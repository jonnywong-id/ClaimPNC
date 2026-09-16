CREATE OR REPLACE PROCEDURE          PEGA_LST_DET_TYPE_DOC_BUSINESS(IDPega IN VARCHAR2, IDBusiness IN VARCHAR2, IdObjDoc IN VARCHAR2, IdTypeDoc IN VARCHAR2,
IdTypeDTDoc IN VARCHAR2, DetailDocument IN VARCHAR2, statusWajib IN VARCHAR2, MinDoc IN VARCHAR2, TglEdit IN VARCHAR2, UserEdit IN VARCHAR2, 
tCvg IN VARCHAR2,claimno in varchar2, tID OUT VARCHAR2, ErrMsg OUT VARCHAR2)
AS
id_site M_SITE_DATABASE.ID%type;
id_detype_ins POOLDATA.LST_TYPE_DOC_BUSINESS.ID%type;
countcvg number;

BEGIN

    IF IDPega = 'UnknownID' AND tCvg IS NULL THEN

        BEGIN
                SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
        EXCEPTION
              WHEN OTHERS THEN
              ErrMsg := 'SELECT PEGA_LST_DET_TYPE_DOC_BUSINESS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;

        id_detype_ins := id_site || lpad(to_Char(LST_TYPE_DOC_BUSINESS_SEQ.nextval),4,'0');

        BEGIN
            INSERT INTO POOLDATA.LST_TYPE_DOC_BUSINESS(ID,BUSINESSID, DOCUMENT_TYPE_ID, OBJECT_DOC_ID, DOC_TYPE_DT_ID, DETAIL_DOKUMEN, STS_WAJIB, MIN_DOC, EDIT_DATE,USER_EDIT) 
            VALUES(id_detype_ins, IDBusiness, IdTypeDoc, IdObjDoc, IdTypeDTDoc, DetailDocument, statusWajib, MinDoc, TglEdit, UserEdit);
            ErrMsg := 'Data Berhasil Disimpan ';
            tID := id_detype_ins;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT Tabel LST_TYPE_DOC_BUSINESS Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
        
    ELSIF IDPega != 'UnknownID' AND tCvg IS NULL THEN

            BEGIN
                UPDATE POOLDATA.LST_TYPE_DOC_BUSINESS SET DOCUMENT_TYPE_ID = IdTypeDoc, OBJECT_DOC_ID = IdObjDoc, 
                DOC_TYPE_DT_ID = IdTypeDTDoc, DETAIL_DOKUMEN = DetailDocument,  STS_WAJIB= statusWajib, MIN_DOC = MinDoc,  EDIT_DATE=TglEdit, USER_EDIT =UserEdit
                WHERE ID = IDPega;
                ErrMsg := 'Data berhasil di update Sudah Disimpan: ';
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE Tabel LST_TYPE_DOC_BUSINESS Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;
    END IF;
    
    IF tCvg IS NOT NULL and claimno is null THEN
    
        select count(id) into countcvg from POOLDATA.COVERAGE_DOC_BUSINESS where coverageid = tCvg and id = IDPega;
        
        if countcvg = 0 then
        
            insert into pooldata.coverage_doc_business (id, businessid, coverageid)
            values (IDPega,IDBusiness,tCvg);
        
        end if;
    
    ELSIF claimno is not null then
    
        select count(id) into countcvg from POOLDATA.COVERAGE_DOC_BUSINESS where coverageid = tCvg and id = IDPega and NOKLAIM=claimno;
        
        if countcvg = 0 then
        
            insert into pooldata.coverage_doc_business (id, businessid, coverageid,NOKLAIM)
            values (IDPega,IDBusiness,tCvg,claimno);
        
        end if;
    
    END IF;
    
    

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition LST_TYPE_DOC_BUSINESS Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/