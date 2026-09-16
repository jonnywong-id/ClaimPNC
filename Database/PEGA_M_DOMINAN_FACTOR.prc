CREATE OR REPLACE PROCEDURE          PEGA_M_DOMINAN_FACTOR(KETERANGAN IN VARCHAR2, ID IN VARCHAR2, STATUS IN VARCHAR2,ErrMsg OUT VARCHAR2)
AS
id_site VARCHAR (5);
id_dominan_factor varchar2 (4);


BEGIN

    
   IF STATUS = 'Insert' THEN

        BEGIN
                select nvl(max(to_number(ID)),0)+1 into id_site from M_DOMINAN_FACTOR;
                
          INSERT INTO POOLDATA.M_DOMINAN_FACTOR(ID,NAME) VALUES(ID_SITE,KETERANGAN);
            ErrMsg := 'Data Sudah Disimpan dengan ID : ' || ID_SITE;
            COMMIT;
        EXCEPTION
        WHEN OTHERS THEN
              ErrMsg := 'INSERT POOLDATA.M_DOMINAN_FACTOR Error : ' || sqlerrm;
              ROLLBACK;
              RETURN;
        END;
   -- END IF;
    
    ELSE
         BEGIN
                UPDATE POOLDATA.M_DOMINAN_FACTOR SET NAME = KETERANGAN WHERE ID = id;
                ErrMsg := 'Data Sudah Diupdate dengan ID : ' || ID;
                COMMIT;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'UPDATE POOLDATA.M_DOMIAN_FACTOR Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;
    END IF;

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition PEGA_M_DOMINAN_FACTOR : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/