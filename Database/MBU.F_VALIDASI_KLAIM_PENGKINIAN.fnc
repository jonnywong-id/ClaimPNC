CREATE OR REPLACE FUNCTION MBU.F_VALIDASI_KLAIM_PENGKINIAN( vJENIS_DOC IN VARCHAR2, vNOMOR_DOC IN VARCHAR2, vWRONG OUT VARCHAR2 ) RETURN INTEGER 
IS     
   vRETURN INTEGER;
       
BEGIN
   vRETURN := 0;
   IF vJENIS_DOC = 'NO_KTP' THEN
      IF LENGTH(vNOMOR_DOC) < 14 OR LENGTH(vNOMOR_DOC) > 16 THEN
         vWRONG  := 'Panjang karakter No. KTP (' || vNOMOR_DOC || ') tidak valid !';             
      ELSIF TRANSLATE(vNOMOR_DOC,'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ+_-., ','0123456789') <> vNOMOR_DOC THEN
         vWRONG  := 'Format No. KTP (' || vNOMOR_DOC || ') tidak valid !'; 
      ELSIF INSTR(vNOMOR_DOC,'0000000') > 0 OR INSTR(vNOMOR_DOC,'1111111') > 0 OR INSTR(vNOMOR_DOC,'22222') > 0 OR INSTR(vNOMOR_DOC,'33333') > 0 
         OR INSTR(vNOMOR_DOC,'44444') > 0 OR INSTR(vNOMOR_DOC,'55555') > 0 OR INSTR(vNOMOR_DOC,'66666') > 0 OR INSTR(vNOMOR_DOC,'77777') > 0 
         OR INSTR(vNOMOR_DOC,'88888') > 0 OR INSTR(vNOMOR_DOC,'99999') > 0 THEN
         vWRONG  := 'Format No. KTP (' || vNOMOR_DOC || ') tidak valid, ada pengulangan angka !';
      ELSIF INSTR(vNOMOR_DOC,'12345') > 0 OR INSTR(vNOMOR_DOC,'23456') > 0 OR INSTR(vNOMOR_DOC,'34567') > 0 OR INSTR(vNOMOR_DOC,'45678') > 0 
         OR INSTR(vNOMOR_DOC,'56789') > 0 OR INSTR(vNOMOR_DOC,'67890') > 0 THEN
         vWRONG  := 'Format No. KTP (' || vNOMOR_DOC || ') tidak valid, angka berurutan !';   
      ELSE 
         vRETURN := 1;       
      END IF;   
   ELSIF vJENIS_DOC = 'NO_HP' THEN
      IF LENGTH(vNOMOR_DOC) < 10 OR LENGTH(vNOMOR_DOC) > 14 THEN
         vWRONG  := 'Panjang karakter No. HP (' || vNOMOR_DOC || ') tidak valid !';
      ELSIF TRANSLATE(vNOMOR_DOC,'0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ+_-., ','0123456789') <> vNOMOR_DOC THEN
         vWRONG  := 'Format No. HP (' || vNOMOR_DOC || ') tidak valid !';   
      ELSIF ( vNOMOR_DOC NOT LIKE '08%' AND vNOMOR_DOC NOT LIKE '628%' ) THEN
         vWRONG  := 'Format No. HP (' || vNOMOR_DOC || ') tidak valid !';
      ELSE
         vRETURN := 1;
      END IF;     
   ELSIF vJENIS_DOC = 'EMAIL' THEN
      IF vNOMOR_DOC NOT LIKE '%@%' THEN
         vWRONG  := 'Format Email (' || vNOMOR_DOC || ') tidak valid !'; 
      ELSIF INSTR(vNOMOR_DOC,' ') > 0 OR LOWER(SUBSTR(vNOMOR_DOC, -2)) = 'co' THEN 
         vWRONG  := 'Format Email (' || vNOMOR_DOC || ') tidak valid !';
      ELSIF LOWER(vNOMOR_DOC) LIKE '%sinarmas.co.id%' or LOWER(vNOMOR_DOC) LIKE '%simasinsurtech.com%' THEN 
         vWRONG  := 'Email ' || vNOMOR_DOC || ' tidak valid. Tidak boleh menggunakan email internal !';   
      ELSE    
         vRETURN := 1;
      END IF;     
   ELSE 
      vRETURN := 0;
   END IF;
       
   RETURN vRETURN;   
END;
/
