CREATE OR REPLACE PROCEDURE          PNC_INSERT_EMAIL_ADJUSTER(CaseSurvey IN VARCHAR2, SubjectEmail IN VARCHAR2, BodyEmail IN VARCHAR2, 
Sender IN VARCHAR2, Recipient IN VARCHAR2, Picteknis IN VARCHAR2, Picadjuster IN VARCHAR2, Typesender IN VARCHAR2, ErrMsg OUT VARCHAR2)
AS

BEGIN
    
    BEGIN
        INSERT INTO POOLDATA.LIST_EMAIL_ADJUSTER (CASESURVEY, SUBJECTEMAIL, BODYEMAIL, SENDER, RECIPIENT, PICTEKNIS, PICADJUSTER, TYPESENDER) 
        VALUES (CaseSurvey, SubjectEmail, BodyEmail, Sender, Recipient, Picteknis, Picadjuster, Typesender);
        ErrMsg := 'Data sudah di simpan';
    EXCEPTION
        WHEN OTHERS THEN
            ErrMsg := 'Error Send Email Adjuster: ' || sqlerrm;
            ROLLBACK;
            RETURN;
    END;
        
EXCEPTION
    WHEN OTHERS THEN
        ErrMsg := 'PROCEDURE Email Adjuster Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/