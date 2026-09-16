CREATE OR REPLACE PROCEDURE          ProcessNewEDMObjectData(p_IDPEGA IN VARCHAR2, p_NOPOLIS IN VARCHAR2, ERRMSG OUT VARCHAR2)
    IS
       v_idpega             VARCHAR2(200);
       v_FlagOldData        VARCHAR2(10);
       v_FlagEditData       VARCHAR2(10);
       v_INDEXOBJECT        number;
       CNT_IndexObject      number;  
       isExistData          VARCHAR2(10);
       PERSONLISTDATA       JSON_OBJECT_T   DEFAULT NULL; 
       PERSONLISTBLOB       BLOB;
       COVERAGELIST_OBJECTDATA  JSON_OBJECT_T   DEFAULT NULL;
       COVERAGELISTDATA     JSON_OBJECT_T   DEFAULT NULL;   /*Ini Untuk Loop dan ambil data*/
       COVERAGELIST         JSON_ARRAY_T    DEFAULT NULL; 
       COVERAGELISTBLOB     BLOB;
       COVERAGELISTDATA_NEW    JSON_OBJECT_T   DEFAULT NULL; /*Ini Untuk Storing dan betuk coverage baru*/
       COVERAGELIST_OBJNEW  JSON_OBJECT_T   DEFAULT NULL;
       COVERAGELIST_ARRNEW  JSON_ARRAY_T    DEFAULT NULL; 
       OUTGOLISTDATA        JSON_OBJECT_T   DEFAULT NULL;
       OUTGOLIST            JSON_ARRAY_T    DEFAULT NULL; 
       OUTGOLIST_OBJNEW     JSON_OBJECT_T    DEFAULT NULL; 
       OUTGOLIST_ARRNEW     JSON_ARRAY_T    DEFAULT NULL; 
       SPREADINGLISTDATA        JSON_OBJECT_T   DEFAULT NULL;
       SPREADINGLIST            JSON_ARRAY_T    DEFAULT NULL; 
       SPREADINGLIST_OBJNEW     JSON_OBJECT_T    DEFAULT NULL; 
       SPREADINGLIST_ARRNEW     JSON_ARRAY_T    DEFAULT NULL; 
CURSOR Data_Person IS
    select NOPOLIS,
           PRODKE,
           INDEXOBJECT,
           UNAMEDBASESPARTICIPANT,
           ASMCCAMOUNT,
           PYFULLNAME,
           ASMIDCARD,
           ASMDATEOFBIRTH,
           ASMPARTICIPANTSTATUS,
           ASMGENDER,
           ASMJOBNAME,
           ASMJOBDESC,
           ASMCLASS,
           ASMCLASSID,
           ASMHEIGHT,
           ASMWEIGHT,
           ASMLEFTHANDED,
           HEIRTYPE,
           STARTDATETIME,
           ENDDATETIME,
           GROUPPANEL,
           FLAGOLDDATA,
           FLAGDELETE,
           FLAGEDITDATA,
           OBJECTDATA,
           COVERAGEDATA,
           HEIRDATA 
           from t_personlist where NOPOLIS = p_NOPOLIS 
           and to_number(prodke) = (select max(to_number(prodke)) 
                                    from t_personlist where nopolis = p_NOPOLIS);
   BEGIN
   
   select count(*) into isExistData from t_personlist where idpega = p_IDPEGA; 
   IF isExistData = 0 THEN
       v_idpega := p_IDPEGA;
       v_FlagOldData := '1';
       v_FlagEditData := '0';

       FOR Rec_Person  IN Data_Person
       LOOP
            IF Rec_Person.FLAGDELETE <> '1' OR Rec_Person.FLAGDELETE is null THEN 
            
            PERSONLISTDATA := JSON_OBJECT_T();
            PERSONLISTDATA :=  JSON_OBJECT_T.parse(Rec_Person.OBJECTDATA);
            -- Untuk Bentuk Object Data Person JSON EDM Baru
            BEGIN 
                /*Setting adjust index object */
                    SELECT max(indexobject)
                    into   CNT_IndexObject
                    FROM   T_PERSONLIST a
                    WHERE  (a.IDPEGA = p_IDPEGA);
                
                    /*Untuk data pertama jika index kosong*/
                    IF CNT_IndexObject IS NULL THEN
                        CNT_IndexObject := 0;
                    END IF;
    
               
                IF (PERSONLISTDATA.get_string('IndexObject')IS NULL) THEN
                        v_INDEXOBJECT         :=  CNT_IndexObject+1; 
                        PERSONLISTDATA.put('IndexObject',v_INDEXOBJECT);
                ELSE
                        v_INDEXOBJECT         :=  PERSONLISTDATA.get_number('IndexObject');
                END IF;
            
                   PERSONLISTDATA.put('FlagDelete','0');
                   PERSONLISTDATA.put('FlagEditData','0');
                   PERSONLISTDATA.put('FlagOldData','1');
                   PERSONLISTDATA.remove('OldPerson'); 
                   PERSONLISTDATA.remove('ASMHeir'); 
                PERSONLISTBLOB := pooldata.clobtoblob(PERSONLISTDATA.to_clob());
            exception when others then
                    ERRMSG := 'Person list data kosong';
            END;
            
            -- Untuk Bentuk Coverage Data JSON EDM Baru
            BEGIN
               COVERAGELIST_OBJECTDATA := JSON_OBJECT_T();
               COVERAGELIST := JSON_ARRAY_T();
               COVERAGELISTDATA := JSON_OBJECT_T();
               COVERAGELIST_OBJECTDATA :=  JSON_OBJECT_T.parse(Rec_Person.COVERAGEDATA);
               IF(COVERAGELIST_OBJECTDATA.get_array('ASMCoverage') is not null) then  
               COVERAGELIST_OBJNEW := Json_object_t();  
               COVERAGELIST_ARRNEW := Json_array_t(); 
               COVERAGELIST    :=  COVERAGELIST_OBJECTDATA.get_array('ASMCoverage');
                          
                    FOR j IN 0 .. COVERAGELIST.get_size - 1 LOOP
                        COVERAGELISTDATA_NEW := Json_object_t(); 
                        COVERAGELISTDATA    :=  JSON_OBJECT_T(COVERAGELIST.get(j));
                        COVERAGELISTDATA_NEW := JSON_OBJECT_T(COVERAGELISTDATA);
                            COVERAGELISTDATA_NEW.put('FlagDelete','0');
                            COVERAGELISTDATA_NEW.put('FlagEditData','0');
                            COVERAGELISTDATA_NEW.put('FlagOldData','1');
                            COVERAGELISTDATA_NEW.remove('OldCoverage');
                            
                        -- Untuk Bentuk Outgo Data JSON EDM Baru    
                        BEGIN  
                            IF(COVERAGELISTDATA.get_array('OutGoList')) is not null then  
                                 OUTGOLIST_ARRNEW := Json_array_t(); 
                                 OUTGOLISTDATA := Json_object_t();  
                                 OUTGOLIST := Json_array_t();
                                 OUTGOLIST   :=  COVERAGELISTDATA.get_array('OutGoList');
                                 FOR j IN 0 .. OUTGOLIST.get_size - 1 LOOP
                                    OUTGOLISTDATA   :=  JSON_OBJECT_T(OUTGOLIST.get(j));
                                    OUTGOLISTDATA.put('FlagDelete','0');
                                    OUTGOLISTDATA.put('FlagEditData','0');
                                    OUTGOLISTDATA.put('FlagOldData','1');
                                    OUTGOLISTDATA.remove('OldOutgo');
                                 END LOOP;
                                 
                                 OUTGOLIST_ARRNEW.Append(OUTGOLISTDATA);
                           
                             --Masukin outgo baru ke dalam outgo
                            COVERAGELISTDATA_NEW.put('OutGoList',OUTGOLIST_ARRNEW);
                             
                            end if;
                        exception when others then
                            ERRMSG := 'Coverage list data kosong';
                        END;
                        
                         -- Untuk Bentuk Spreading Data JSON EDM Baru    
                        BEGIN  
                            IF(COVERAGELISTDATA.get_array('SpreadingList')) is not null then
                                 SPREADINGLIST_OBJNEW := Json_object_t();  
                                 SPREADINGLIST_ARRNEW := Json_array_t(); 
                                 SPREADINGLISTDATA := Json_object_t();  
                                 SPREADINGLIST := Json_array_t();
                                 SPREADINGLIST   :=  COVERAGELISTDATA.get_array('SpreadingList');
                                 FOR j IN 0 .. SPREADINGLIST.get_size - 1 LOOP
                                    SPREADINGLISTDATA   :=  JSON_OBJECT_T(SPREADINGLIST.get(j));
                                    SPREADINGLISTDATA.put('FlagDelete','0');
                                    SPREADINGLISTDATA.put('FlagEditData','0');
                                    SPREADINGLISTDATA.put('FlagOldData','1');
                                    SPREADINGLISTDATA.remove('OldSpreading');
                                 END LOOP;
                                 
                                 SPREADINGLIST_ARRNEW.Append(SPREADINGLISTDATA);
                            
                             --Masukin outgo baru ke dalam outgo
                            COVERAGELISTDATA_NEW.put('SpreadingList',SPREADINGLIST_ARRNEW);
                            
                            end if;
                        exception when others then
                            ERRMSG := 'Spreading list data kosong';
                        END;
                        
                    COVERAGELIST_ARRNEW.Append(COVERAGELISTDATA_NEW);    
                   -- dbms_output.put_line(COVERAGELISTDATA.stringify());
                    END LOOP;
                    COVERAGELIST_OBJNEW.Put('ASMCoverage',COVERAGELIST_ARRNEW);
                    COVERAGELISTBLOB    :=  pooldata.clobtoblob(COVERAGELIST_OBJNEW.to_clob());
                      
               END IF;
           exception when others then
                    ERRMSG := ERRMSG || SQLERRM;     
           END;
                
            BEGIN
                INSERT INTO t_personlist ( IDPEGA,
                                           NOPOLIS,
                                           PRODKE,
                                           INDEXOBJECT,
                                           UNAMEDBASESPARTICIPANT,
                                           ASMCCAMOUNT,
                                           PYFULLNAME,
                                           ASMIDCARD,
                                           ASMDATEOFBIRTH,
                                           ASMPARTICIPANTSTATUS,
                                           ASMGENDER,
                                           ASMJOBNAME,
                                           ASMJOBDESC,
                                           ASMCLASS,
                                           ASMCLASSID,
                                           ASMHEIGHT,
                                           ASMWEIGHT,
                                           ASMLEFTHANDED,
                                           HEIRTYPE,
                                           STARTDATETIME,
                                           ENDDATETIME,
                                           GROUPPANEL,
                                           FLAGOLDDATA,
                                           FLAGDELETE,
                                           FLAGEDITDATA,
                                           OBJECTDATA,
                                           COVERAGEDATA,
                                           HEIRDATA )
                 VALUES (p_IDPEGA,
                         Rec_Person.NOPOLIS,
                         Rec_Person.PRODKE,
                         v_INDEXOBJECT,
                         Rec_Person.UNAMEDBASESPARTICIPANT,
                         Rec_Person.ASMCCAMOUNT,
                         Rec_Person.PYFULLNAME,
                         Rec_Person.ASMIDCARD,
                         Rec_Person.ASMDATEOFBIRTH,
                         Rec_Person.ASMPARTICIPANTSTATUS,
                         Rec_Person.ASMGENDER,
                         Rec_Person.ASMJOBNAME,
                         Rec_Person.ASMJOBDESC,
                         Rec_Person.ASMCLASS,
                         Rec_Person.ASMCLASSID,
                         Rec_Person.ASMHEIGHT,
                         Rec_Person.ASMWEIGHT,
                         Rec_Person.ASMLEFTHANDED,
                         Rec_Person.HEIRTYPE,
                         Rec_Person.STARTDATETIME,
                         Rec_Person.ENDDATETIME,
                         Rec_Person.GROUPPANEL,
                         v_FlagOldData,
                         nvl(Rec_Person.FLAGDELETE,0),
                         v_FlagEditData,
                         PERSONLISTBLOB,
                         COVERAGELISTBLOB,
                         Rec_Person.HEIRDATA);  
            EXCEPTION
            WHEN OTHERS
            THEN
                ERRMSG := 'ERROR INSERT T_PERSON LIST';         
            END;
            
            END IF;   
            COMMIT;
       END LOOP;
   END IF;
   EXCEPTION
      WHEN OTHERS
      THEN
            ERRMSG := 'Error Copy Data dengan IDPEGA : '||p_IDPEGA||' dan Nopolis ' || p_nopolis;
   END;

/