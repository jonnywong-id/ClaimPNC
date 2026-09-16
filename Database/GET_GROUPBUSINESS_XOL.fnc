CREATE OR REPLACE function          GET_GROUPBUSINESS_XOL (tahun varchar2, m_id varchar2, tipe varchar2) return varchar2
is
   idbisnis varchar2(3600);
   jml int;
   
   CURSOR c_M_XOL_PNC IS
    SELECT b.idbusiness,(select note from pooldata.businessgroup where id=b.idbusiness) as note from pooldata.mst_xol_business b, pooldata.mst_xol_pnc a
    WHERE  b.id = m_id and a.tahun= tahun and a.id=b.id;   
                   
begin
 
    jml := 0;
    
    FOR PNC_XOL  IN c_M_XOL_PNC LOOP
    
        if tipe = 'id' then    
    
            if jml = 0 then
                
                idbisnis := ''''||PNC_XOL.idbusiness || '''' ;
                
            elsif jml > 0 then

                idbisnis :=  idbisnis || ',''' || PNC_XOL.idbusiness || '''' ;

            end if;
            
        elsif tipe = 'note' then
        
            IF PNC_XOL.note is null then
                PNC_XOL.note := 'TREATY INWARD';
            ELSE 
                PNC_XOL.note := PNC_XOL.note  ;
            END IF;

            if jml = 0 then
                
                idbisnis := PNC_XOL.note  ;
                
                
            elsif jml > 0 then

                idbisnis :=  idbisnis || ',  ' || PNC_XOL.note ;

            end if;
        
        end if;        
        
        jml := jml + 1;
        
    END LOOP;
    
    return idbisnis;
   
end GET_GROUPBUSINESS_XOL;

/