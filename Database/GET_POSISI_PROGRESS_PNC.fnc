CREATE OR REPLACE function          GET_POSISI_PROGRESS_PNC (p_claimno varchar2, kategori varchar2) return varchar2
is
   temp_posisi varchar2(4000);
   jml_posisi int;
   
   CURSOR c_M_WORK_PNC IS
    SELECT q.posisi,
           q.claimno,
           (select max(sts_progress1) from gcnm_progress_claim a,GCNM_MST_PROGRESS_KLAIM b where a.status_progress1 = b.id_progress 
                   and pnccaseid=q.claimno and posisiid=q.id) as "stsprog1",
            (select max(sts_progress2) from gcnm_progress_claim a,GCNM_MST_PROGRESS b where a.status_progress2= b.id_mst
                   and pnccaseid=q.claimno and posisiid=q.id) as "stsprog2",
            (select to_char(max(next_followup),'DD-MM-YYYY hh:mm:ss')from gcnm_progress_claim a where pnccaseid = q.claimno and posisiid = q.id) as "tglfu"
        FROM GCNM_PROGRESS_POSISI_PNC q
    WHERE  q.claimno = p_claimno and q.statusposisi = 'On Progress';                  
 begin
    jml_posisi := 0;
    FOR PEGA_WORKPNC  IN c_M_WORK_PNC LOOP
        if kategori='POSISI' then
            if (jml_posisi = 0 ) then
                temp_posisi := PEGA_WORKPNC.posisi;
            else 
                temp_posisi :=  temp_posisi || ',' || PEGA_WORKPNC.posisi;
            end if;
        elsif kategori='sts_prg1' then
            if (jml_posisi = 0 ) then
                temp_posisi := PEGA_WORKPNC."stsprog1";
            else 
                temp_posisi :=  temp_posisi || ',' || PEGA_WORKPNC."stsprog1";
            end if;
         elsif kategori='sts_prg2' then
            if (jml_posisi = 0 ) then
                temp_posisi := PEGA_WORKPNC."stsprog2";
            else 
                temp_posisi :=  temp_posisi || ',' || PEGA_WORKPNC."stsprog2";
            end if;
        elsif kategori='nextfu' then
            if (jml_posisi = 0 ) then
                temp_posisi := PEGA_WORKPNC."tglfu";
            else 
                temp_posisi :=  temp_posisi || ',' || PEGA_WORKPNC."tglfu";
            end if;
        end if;
        jml_posisi := jml_posisi + 1;
    END LOOP;
   return temp_posisi;
end GET_POSISI_PROGRESS_PNC;

/