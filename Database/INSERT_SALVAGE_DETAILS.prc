CREATE OR REPLACE PROCEDURE          INSERT_SALVAGE_DETAILS(t_status varchar2, tnoakseptasi varchar2, tclaimno varchar2, tidsalvage varchar2,
                                                            tNamabarang varchar2, tnilaiklaim number,tremarks varchar2,tpemenang varchar2,
                                                            tterjual varchar2, idtetails varchar2, tsatuan varchar2, tnoteitem varchar2,tidbalalelang varchar2,tTOTALHARGA number,tTIPEPENJUALAN varchar2,
                                                            tsalvagedate Date,thargaterjual number,ErrMsg OUT VARCHAR2)
AS

jum_data number;
tiddetailssalvage varchar2(3000);
tsnamabarang varchar2(3000);
tssatuan varchar2(3000);
salvageid varchar2(2000);
noklaims varchar2(1000);


BEGIN
   
    dbms_output.put_line('satu');
    
    IF t_status = '1' THEN         
        
        select count(1) into jum_data from POOLDATA.DETAIL_PNC_SALVAGE where IDSALVAGE=tidsalvage and NOKLAIM=tclaimno;
        
        jum_data := jum_data+1;
        tiddetailssalvage := tclaimno||'/'||tidsalvage||'/'||jum_data;
      
        INSERT INTO POOLDATA.DETAIL_PNC_SALVAGE (INSERTDATE,STATUSTERJUAL, NOAKSEPTASI,NILAIAKSEPTASI,TGLAKSEP,REMARK,IDSALVAGE,NOKLAIM,IDDETAILSALVAGE,NAMABARANG,SATUAN,NOTE_ITEM,HARGAITEM,IDOBJECT)
        VALUES (sysdate,'3', '-', tnilaiklaim, null, tremarks,tidsalvage,tclaimno,tiddetailssalvage,tNamabarang,tsatuan,tnoteitem,tTOTALHARGA,jum_data); 
        ErrMsg := tiddetailssalvage;
        COMMIT;     
    
    ELSIF t_status = '0' THEN
        dbms_output.put_line('dua');
        BEGIN
        
        select IDDETAILSALVAGE into tiddetailssalvage from POOLDATA.DETAIL_PNC_SALVAGE where IDDETAILSALVAGE=idtetails;
        dbms_output.put_line('tiga');
        
        EXCEPTION WHEN NO_DATA_FOUND THEN  
                tiddetailssalvage := null;  
                
        END;
        
          
        IF tiddetailssalvage is not null THEN
                
            UPDATE POOLDATA.DETAIL_PNC_SALVAGE SET TGLAKSEP=sysdate,NOAKSEPTASI=tnoakseptasi,NILAIAKSEPTASI=tnilaiklaim,REMARK=tremarks,STATUSTERJUAL=tterjual,PEMENANGSALVAGE=tpemenang,IDBALAILELANG=tidbalalelang,
            TIPEPENJUALAN=tTIPEPENJUALAN,SALVAGEDATE=tsalvagedate,HARGATERJUAL=thargaterjual
            WHERE IDDETAILSALVAGE=tiddetailssalvage;   
            
            dbms_output.put_line('empat');
            ErrMsg := tiddetailssalvage;     
            
            IF tterjual ='2' then
                UPDATE POOLDATA.PNC_SALVAGE set STSTRANSFER='5' where IDSALVAGE = tidsalvage;
            
            end if;  
            
            COMMIT;

        END IF;

    ELSIF t_status = '2' THEN
    
        BEGIN
        
        select IDDETAILSALVAGE,A.NAMABARANG,A.SATUAN,A.NOKLAIM,A.IDSALVAGE into tiddetailssalvage,tsnamabarang,tssatuan,noklaims,salvageid from POOLDATA.DETAIL_PNC_SALVAGE a where IDDETAILSALVAGE=idtetails;
        dbms_output.put_line('tiga');
        
        EXCEPTION WHEN NO_DATA_FOUND THEN  
                tiddetailssalvage := null;  
                
        END;
    
        IF tiddetailssalvage is not null THEN
            select count(1) into jum_data from POOLDATA.DETAIL_PNC_SALVAGE where IDSALVAGE=salvageid and NOKLAIM=noklaims;
            jum_data := jum_data+1;
            tiddetailssalvage := noklaims||'/'||salvageid||'/'||jum_data;
            
            INSERT INTO POOLDATA.DETAIL_PNC_SALVAGE (INSERTDATE,NOAKSEPTASI,NILAIAKSEPTASI,TGLAKSEP,REMARK,IDSALVAGE,NOKLAIM,IDDETAILSALVAGE,NAMABARANG,SATUAN,STATUSTERJUAL,NOTE_ITEM,IDOBJECT)
            VALUES (sysdate, tnoakseptasi, tnilaiklaim, sysdate, tremarks,salvageid,noklaims,tiddetailssalvage,tsnamabarang,tssatuan,tterjual,'PENYESUAIAN SALVAGE',jum_data); 
            ErrMsg := tiddetailssalvage;
            COMMIT;   
            
        END IF;
        
    END IF;    
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT SALVAGE : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/