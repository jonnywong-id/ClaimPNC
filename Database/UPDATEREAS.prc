CREATE OR REPLACE PROCEDURE          UPDATEREAS (tREINSID in VARCHAR2, tREINSNAME in VARCHAR2, tLOGIN in VARCHAR2, tEMAIl in VARCHAR2, tCOUNTRY in VARCHAR2, tTYPE in VARCHAR2, ErrMsg OUT VARCHAR2)
AS
REAS NUMBER; 
JUMLAH_TIPE NUMBER; 
JENIS NUMBER;
NEGARA_ID VARCHAR2 (1000); 

BEGIN
 
   SELECT COUNT(1) INTO REAS FROM T_REINSURER 
   WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME ;   

   IF REAS > 0 THEN 
   
        SELECT COUNT(1) INTO JUMLAH_TIPE FROM T_REINSURER 
        WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = tTYPE ;
        
        IF JUMLAH_TIPE > 0 THEN 
            
            UPDATE POOLDATA.T_REINSURER 
            SET EMAIl=tEMAIl
            WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = tTYPE;  
            ErrMsg := 'Data Diperbarui/1'; 
            COMMIT;     
            
        ELSE          
            
            SELECT COUNT(TYPE) INTO JENIS FROM T_REINSURER 
            WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = '1' ; 
                
            IF JENIS > 0 THEN 
                    
                UPDATE POOLDATA.T_REINSURER 
                SET EMAIl=tEMAIl, TYPE=tTYPE 
                WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = '1';  
                ErrMsg := 'Data Diperbarui/2'; 
                COMMIT;    
                    
            ELSE 
            
                SELECT ID INTO NEGARA_ID FROM COUNTRY 
                WHERE COUNTRY = tCOUNTRY ;
                                    
                INSERT INTO POOLDATA.T_REINSURER (REINSURERID, REINSURERNAME, LOGIN, EMAIl, COUNTRY, COUNTRYID, TYPE)
                VALUES (tREINSID, tREINSNAME, tLOGIN, tEMAIl, tCOUNTRY, NEGARA_ID, tTYPE); 
                ErrMsg := 'Data Tersimpan/2';
                COMMIT;
                    
            END IF; 
                       
        END IF;        
        
   ELSE
   
        SELECT ID INTO NEGARA_ID FROM COUNTRY 
        WHERE COUNTRY = tCOUNTRY ; 
    
        INSERT INTO POOLDATA.T_REINSURER (REINSURERID, REINSURERNAME, LOGIN, EMAIl, COUNTRY, COUNTRYID, TYPE)
        VALUES (tREINSID, tREINSNAME, tLOGIN, tEMAIl, tCOUNTRY, NEGARA_ID, tTYPE); 
        ErrMsg := 'Data Tersimpan/1';
        COMMIT; 
        
   END IF; 
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Insert/Update Error : ' || sqlerrm;
        ROLLBACK;
        RETURN; 
END;

/