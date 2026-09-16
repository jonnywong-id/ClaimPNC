CREATE OR REPLACE FUNCTION          Get_InterpolasiPNC (ad_value number)
   RETURN NUMBER
IS
    les_index number;
    max_index number;
    less_amount number;
    ehasil number;
    max_fee number;
    less_fee number;
    data_ex number;
    selisih_fee number;
    selisih_lossamount number;
    max_amount number;
    jum_data number;
Begin
    data_ex := ad_value;
    select count(1) into jum_data FROM POOLDATA.GCNM_FEE_SCALE where LOSS_AMOUNT >= data_ex and INDEX_FEE = (select min(INDEX_FEE) FROM POOLDATA.GCNM_FEE_SCALE where LOSS_AMOUNT >= data_ex);
    
    if jum_data> 0 then
        select LOSS_AMOUNT,INDEX_FEE,FEE into max_amount,max_index,max_fee FROM POOLDATA.GCNM_FEE_SCALE where LOSS_AMOUNT >= data_ex and INDEX_FEE = (select min(INDEX_FEE) FROM POOLDATA.GCNM_FEE_SCALE where LOSS_AMOUNT >= data_ex);
        if max_index > 1 then
            select LOSS_AMOUNT,INDEX_FEE,FEE into less_amount,les_index,less_fee FROM POOLDATA.GCNM_FEE_SCALE where INDEX_FEE=max_index-1;
            selisih_lossamount := max_amount-less_amount;
            selisih_fee := max_fee - less_fee;
            ehasil := less_fee + ((data_ex - less_amount) * (selisih_fee/selisih_lossamount));
            
            ---dbms_output.put_line('Hasil '||ehasil);
            
        else
            ehasil := 1650000;
            ---dbms_output.put_line(ehasil);
            
        end if;           
    
    else
        select LOSS_AMOUNT,FEE into max_amount,max_fee FROM POOLDATA.GCNM_FEE_SCALE where INDEX_FEE=17;
        ehasil := (max_fee+((data_ex-max_amount)*(2/100)));
        ---dbms_output.put_line('Hasil '||ehasil);
    end if;
    
    return ehasil;
    
end;

/