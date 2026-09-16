CREATE OR REPLACE PROCEDURE          INSERT_PLADLA(TCLAIMID in varchar2, TOBJID in varchar2, TCVGID in varchar2,TADJID in varchar2, TNOPLADLA in varchar2, 
                                                   TTGLPLADLA in date, TNILAIDLA in varchar2, TREINSURER in varchar2,TTIPE in varchar2, TNOTE in varchar2,
                                                   TREVISI in varchar2, PLADLA in varchar2, TNOAKSEP in varchar2, TREINSCODE in varchar2, tCURRENCY in varchar2,
                                                   tCURRPOLIS in varchar2, tSHARE in varchar2, tPERCENT in varchar2, tNILAIKLAIM in varchar2, tQSPQS in varchar2,
                                                   tQSRI in varchar2, tKirim in varchar2, tEmail2 in varchar2, tESTSHARE in varchar2, tTGLAKSEP in date,tjsonpla clob,tlocationid in varchar2,
                                                   ErrMsg OUT VARCHAR2)
AS
idcount number;
countreas number;
tlogin varchar2(200);
tcountry varchar2(60);
temail varchar2(1000);
tnotess varchar2(3000);
tcatatan varchar2(3000);
countplas number;
countpla number;
nomoroldes varchar2(100);
tgloldest varchar2(100);
nomorp varchar2(100);
tanggalp varchar2(100);
cekdla number;
cekpla number;
salvagenotes varchar2(2000);
countaksep number;

BEGIN
--    ErrMsg :=tKirim || PLADLA;
--    RETURN;
    IF PLADLA='PLA' THEN         
        select count(1) into idcount from POOLDATA.T_PLALIST where claimid=TCLAIMID and NOPLA=TNOPLADLA 
        and REVISI=TREVISI and OBJECTID=TOBJID and OBJECTCOVERAGEID=TCVGID;
              
            IF idcount < 1 THEN
                
                select count(1) into countreas from t_reinsurer a where reinsurerid = TREINSCODE and REINSURERNAME=TREINSURER fetch next 1 row only;
                select count(1) into cekpla from POOLDATA.T_PLALIST d where claimid=TCLAIMID and REINSCODE =TREINSCODE;
                
                IF countreas > 0 then
                    select login, country, email 
                    into tlogin, tcountry, temail from t_reinsurer where reinsurerid = TREINSCODE and REINSURERNAME=TREINSURER fetch next 1 row only;             
                
                elsif countreas =0 then
                    select count(1) into countreas from t_reinsurer a where reinsurerid = TREINSCODE fetch next 1 row only;
                    IF countreas > 0 then
                        select login, country, email 
                        into tlogin, tcountry, temail from t_reinsurer where reinsurerid = TREINSCODE fetch next 1 row only;             
                    end if;
                
                end if;
                
                If cekpla = 0 then 
                   tcatatan := 'Estimation only. Only when we have detailed information, we would revise accordingly.';
                
                      
                elsIf cekpla>0 then
                    select D.NOPLA,to_char(D.tglpla,'dd/mm/yyyy') into nomorp,tanggalp from POOLDATA.T_PLALIST d where claimid=TCLAIMID and REINSCODE =TREINSCODE order by D.TGLPLA desc fetch next 1 row only;
                    if nomorp is not null then
                        tcatatan := '- Please see our PLA No.:'||nomorp||' with DD: '||tanggalp;
                        
                    end if;
                end if;
                
            
                      
                INSERT INTO POOLDATA.T_PLALIST (CLAIMID, OBJECTID, OBJECTCOVERAGEID, NOPLA, NILAIPLA, PLAREINSURER, REVISI, TIPEPLA, TGLPLA, NOTES, REINSCODE,
                                                CURRENCY, CURRENCYPOLIS, SHARESPREADING, PERCENTPLA, ESTIMASI, ESTIMASISHARE, QSPQS,QSRI, EMAILPLA, LOGIN, COUNTRY,JSON_PLA,LOCATIONID)
                VALUES (TCLAIMID, TOBJID, TCVGID, TNOPLADLA, TNILAIDLA, TREINSURER, TREVISI, TTIPE, SYSDATE, tcatatan,TREINSCODE,
                        tCURRENCY, tCURRPOLIS, tSHARE, tPERCENT, tNILAIKLAIM, tESTSHARE, tQSPQS,tQSRI, temail, tlogin, tcountry,tjsonpla,tlocationid); 
                COMMIT;  
        
            ELSIF idcount > 0 AND TNOTE IS NOT NULL AND tKIRIM IS NULL THEN
    
                UPDATE T_PLALIST SET NOTES=TNOTE, ISPLA='1' WHERE CLAIMID=TCLAIMID AND NOPLA=TNOPLADLA and REVISI=TREVISI;
                COMMIT;
            
            ELSIF idcount > 0 AND tKIRIM IS NOT NULL THEN
    
                UPDATE T_PLALIST SET ISKIRIM = tKIRIM, TGLKIRIM =SYSDATE, EMAILPLA=tEmail2  WHERE CLAIMID=TCLAIMID AND NOPLA=TNOPLADLA and REVISI=TREVISI;
                COMMIT;
            
            END IF;      
    
    ELSIF PLADLA='DLA' THEN
        select count(1) into idcount from POOLDATA.T_DLALIST where claimid=TCLAIMID and NODLA=TNOPLADLA 
        and REVISI=TREVISI and OBJECTID=TOBJID and OBJECTCOVERAGEID=TCVGID and ADJUSTMENTID=TADJID;
           --DBMS_OUTPUT.PUT_LINE('masuk sini');
            IF idcount < 1 and TNILAIDLA is not null THEN
                 
                select count(1) into cekdla from POOLDATA.T_DLALIST c where claimid=TCLAIMID and REINSCODE =TREINSCODE;
                
                select count(1) into countreas from t_reinsurer where reinsurerid = TREINSCODE and REINSURERNAME=TREINSURER fetch next 1 row only;
                
                 IF countreas > 0 then
                    select login, country, email 
                    into tlogin, tcountry, temail from t_reinsurer where reinsurerid = TREINSCODE and REINSURERNAME=TREINSURER fetch next 1 row only;             
                
                elsif countreas =0 then
                    select count(1) into countreas from t_reinsurer a where reinsurerid = TREINSCODE fetch next 1 row only;
                    IF countreas > 0 then
                        select login, country, email 
                        into tlogin, tcountry, temail from t_reinsurer where reinsurerid = TREINSCODE fetch next 1 row only;             
                    end if;
                
                end if;
                
                if cekdla < 1 then
                    select count(1) into countplas from T_PLALIST c where claimid=TCLAIMID and REINSCODE =TREINSCODE;
                     
                    if countplas >0 then
                        select nopla,to_char(tglpla,'dd/mm/yyyy') into nomoroldes,tgloldest from T_PLALIST c where claimid=TCLAIMID and REINSCODE =TREINSCODE order by tglpla desc fetch next 1 row only;
                        if nomoroldes is not null then
                            tnotess := '- Please see our PLA No.:'||nomoroldes||' with DD: '||tgloldest;
                        
                        end if;
                    end if;
                    
                    
                ELSIF cekdla>0 then
                    select C.NODLA,C.TGLDLA into nomoroldes,tgloldest from POOLDATA.T_DLALIST c where claimid=TCLAIMID and REINSCODE =TREINSCODE order by C.TGLDLA desc fetch next 1 row only;
                    if nomoroldes is not null then
                        tnotess := '- Please see our DLA No.:'||nomoroldes||' with DD: '||tgloldest;
                        
                    end if;
                end if;
                
                if TNOTE is not null or TNOTE !='' then
                    tnotess:=TNOTE;
                end if;
                
                
                
                
                --DBMS_OUTPUT.PUT_LINE('masuk sini 222 '||countreas||' '||cekdla);          
                INSERT INTO POOLDATA.T_DLALIST (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID,NODLA, DLAREINSURER, REVISI, NOAKSEP, TIPEDLA, TGLDLA, NILAIDLA, NOTES, REINSCODE,
                                                CURRENCY, CURRENCYPOLIS, SHARESPREADING, PERCENTDLA, KLAIMAMOUNT,QS_PQS,QSRI, EMAILDLA, LOGIN, COUNTRY, TGLAKSEP,LOCATIONID)
                VALUES (TCLAIMID, TOBJID, TCVGID, TADJID,TNOPLADLA, TREINSURER, TREVISI,TNOAKSEP , TTIPE, SYSDATE, TNILAIDLA, tnotess, TREINSCODE,
                        tCURRENCY, tCURRPOLIS, tSHARE, tPERCENT, tNILAIKLAIM,tQSPQS,tQSRI, temail, tlogin, tcountry, tTGLAKSEP,tlocationid);        
                COMMIT;       
              
            ELSIF idcount > 0 AND TNOTE IS NOT NULL AND tKIRIM IS NULL THEN
    
                UPDATE T_DLALIST SET NOTES=TNOTE, ISDLA='1' WHERE CLAIMID=TCLAIMID AND NODLA=TNOPLADLA and REVISI=TREVISI;   
                COMMIT;
            
            ELSIF idcount > 0 AND tKIRIM IS NOT NULL THEN
    
                UPDATE T_DLALIST SET ISKIRIM = tKIRIM,NOTES=TNOTE, TGLKIRIM =SYSDATE, EMAILDLA=tEmail2 WHERE CLAIMID=TCLAIMID AND NODLA=TNOPLADLA and REVISI=TREVISI;
                COMMIT;
           
            END IF;
    
    ELSIF PLADLA='PREDLA' THEN
        select count(1) into idcount from POOLDATA.T_PREDLALIST where claimid=TCLAIMID and NODLA=TNOPLADLA 
        and REVISI=TREVISI and OBJECTID=TOBJID and OBJECTCOVERAGEID=TCVGID and ADJUSTMENTID=TADJID;
            --DBMS_OUTPUT.PUT_LINE('masuk sini');
            
            IF idcount < 1 and TNILAIDLA is not null THEN
                
                select count(1) into countreas from t_reinsurer where reinsurerid = TREINSCODE and REINSURERNAME=TREINSURER fetch next 1 row only;
                
                 IF countreas > 0 then
                    select login, country, email 
                    into tlogin, tcountry, temail from t_reinsurer where reinsurerid = TREINSCODE and REINSURERNAME=TREINSURER fetch next 1 row only;             
                
                elsif countreas =0 then
                    select count(1) into countreas from t_reinsurer a where reinsurerid = TREINSCODE fetch next 1 row only;
                    IF countreas > 0 then
                        select login, country, email 
                        into tlogin, tcountry, temail from t_reinsurer where reinsurerid = TREINSCODE fetch next 1 row only;             
                    end if;
                
                end if;
                
            
                INSERT INTO POOLDATA.T_PREDLALIST (CLAIMID, OBJECTID, OBJECTCOVERAGEID, ADJUSTMENTID, NODLA, DLAREINSURER, REVISI, TIPEDLA, TGLDLA, NILAIDLA,NOTES,REINSCODE,
                                                CURRENCY, CURRENCYPOLIS, SHARESPREADING, PERCENTDLA, KLAIMAMOUNT,QS_PQS,QSRI,EMAILDLA, LOGIN, COUNTRY, TGLAKSEP,LOCATIONID)
                VALUES (TCLAIMID, TOBJID, TCVGID, TADJID,TNOPLADLA, TREINSURER, TREVISI, TTIPE, SYSDATE, TNILAIDLA, TNOTE, TREINSCODE,
                        tCURRENCY, tCURRPOLIS, tSHARE, tPERCENT, tNILAIKLAIM,tQSPQS,tQSRI,temail, tlogin, tcountry, tTGLAKSEP,tlocationid);        
                COMMIT;       
              
            ELSIF idcount > 0 AND TNOTE IS NOT NULL AND tKIRIM IS NULL THEN
                --DBMS_OUTPUT.PUT_LINE('masuk sini 22 '|| TNOTE || TNOPLADLA || TREVISI || TCLAIMID);
                UPDATE T_PREDLALIST SET NOTES=TNOTE, ISDLA='1' WHERE CLAIMID=TCLAIMID AND NODLA=TNOPLADLA and REVISI=TREVISI;   
                COMMIT;
            
            ELSIF idcount > 0 AND tKIRIM IS NOT NULL THEN
    
                UPDATE T_PREDLALIST SET ISKIRIM = tKIRIM, TGLKIRIM =SYSDATE WHERE CLAIMID=TCLAIMID AND NODLA=TNOPLADLA and REVISI=TREVISI;
                COMMIT;
                    
            END IF;

    END IF;    
    
EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Error INSERT PLA/DLA : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/