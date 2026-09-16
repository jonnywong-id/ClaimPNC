CREATE OR REPLACE function          GET_POSISI_PROGRESS2(p_claimno varchar2, id varchar2, PJSON CLOB ) return varchar2
is

   jml_prog2 int;
   temp_prog2 varchar2(1200);
   STS_PROGRESS VARCHAR2(100);
   pjson2 JSON_OBJECT_T;
   TES3   JSON_ARRAY_T;
   TES CLOB;
   tes2 JSON_OBJECT_T;
   tess VARCHAR2(600);
   

     
 BEGIN    
 
--    TES := PJSON;
    pjson2 := JSON_OBJECT_T.parse(PJSON);
    TES3 := pjson2.get_array('ObjectList');
    
    jml_prog2 :=0;
     FOR y IN 0 .. TES3.get_size - 1 LOOP
            
            tes2     :=  JSON_OBJECT_T(TES3.get(y));
            STS_PROGRESS := tes2.get_string('BranchName');
            select STS_PROGRESS2 into tess from POOLDATA.GCNM_MST_PROGRESS where id_mst = STS_PROGRESS; 
         IF (jml_prog2 = 0) THEN

            temp_prog2 := tess;
        ELSE
            temp_prog2 :=temp_prog2 || ', ' || tess; 

        END IF;
        jml_prog2 := jml_prog2 + 1;
        END loop;
        
return temp_prog2; 
end GET_POSISI_PROGRESS2;

/