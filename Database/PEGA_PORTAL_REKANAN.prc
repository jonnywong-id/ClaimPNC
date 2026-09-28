CREATE OR REPLACE PROCEDURE          PEGA_PORTAL_REKANAN (status IN VARCHAR2, IDPega IN VARCHAR2, tlogin IN VARCHAR2, tnopolis IN VARCHAR2, tclaimno IN VARCHAR2,
 tstart IN DATE, tend IN DATE, tqqname IN VARCHAR2, tserialno IN VARCHAR2, tquotation IN VARCHAR2,
 tbill IN VARCHAR2, timei IN VARCHAR2,tnohp IN VARCHAR2, tgaransi IN VARCHAR2, tpic IN VARCHAR2, tanalisa IN VARCHAR2, tserivcefee IN NUMBER, tsparepart IN NUMBER, ttax IN NUMBER,
 totherfee IN NUMBER, tsts_approval IN VARCHAR2, tdeliveryfee IN NUMBER, ttotalfee IN NUMBER, tremark IN VARCHAR2, 
 tPrincipal_Bill_No IN VARCHAR2, tInsurance IN VARCHAR2, tInsurance_Bill_No IN VARCHAR2, tCollect_Point IN VARCHAR2, tRepair_Point IN VARCHAR2, tProd_Group IN VARCHAR2, 
 tProd_Category IN VARCHAR2, tBrand IN VARCHAR2, tModel IN VARCHAR2, tColour IN VARCHAR2, tIs_Delivery IN VARCHAR2, tQuotation_Amount IN NUMBER, tReason IN VARCHAR2,                                 
 tCancelled_Reason IN VARCHAR2, tEstimated_Pickup_Date IN DATE, tPickup_Courier_Date IN DATE, tDevice IN VARCHAR2, tSymptom_Code IN VARCHAR2, tSymptom_Desc IN VARCHAR2,    
 tCust_Arrival IN VARCHAR2, tCust_Name IN VARCHAR2, tStatus IN VARCHAR2, tDownPayment IN NUMBER, tDP_No IN VARCHAR2, tDP_Method IN VARCHAR2,                              
 tAcknowledge_Date IN DATE, tAssigned_Date IN DATE, tCompleted_Date IN DATE, tRelease_Date IN DATE, tInvoice_Date IN DATE, tDeskCharger IN VARCHAR2, tCarKit IN VARCHAR2,   
 tRemovableAntenna IN VARCHAR2, tHeadSet IN VARCHAR2, tBattery IN VARCHAR2, tSimCard IN VARCHAR2, tExtraCover IN VARCHAR2, tBatteryCover IN VARCHAR2, tLCD_text IN VARCHAR2, 
 tType in VARCHAR2, tDol in DATE, tExcess in NUMBER, tObject in VARCHAR2, tNote in VARCHAR2,tRepairID in VARCHAR2,tkomite in VARCHAR2 ,tDeductible in NUMBER,
 tnoktp in VARCHAR2,tittemwarranty in VARCHAR2,taccesorlainya in VARCHAR2,tcase in VARCHAR2,tsboxunit in VARCHAR2,
tchargerchabel in VARCHAR2,tExcessap in NUMBER,tDeductibleap in NUMBER,tsukucadangap in NUMBER,tppnap in NUMBER,
ttaxap in NUMBER,tdeliveryfeeap in NUMBER,ttotalfeeap in NUMBER,tsukucadang in NUMBER,tppn in NUMBER,tpartjson clob,ErrMsg OUT VARCHAR2)
AS
idcount number;

BEGIN 

    IF tType = 'SC' THEN
    
        IF status = 'insert' THEN
            
            BEGIN
                INSERT INTO POOLDATA.T_KLAIM_PORTAL_REKANAN a(ID,INPUTDATE,LOGIN,NOPOLIS,STARTDATE,ENDDATE,QQNAME,SERIALNO,QUOTATIONNO,BILLNO,IMEI,
                                                            NOHP,JENIS_GARANSI,PIC,ANALISA,SERVICE_FEE,SPAREPART_FEE,TAX_FEE,
                                                            OTHER_FEE,STS_APPROVAL,DELIVERY_FEE, TOTAL_FEE, REMARK, Principal_Bill_No, Insurance, 
                                                            Insurance_Bill_No, Collect_Point, Repair_Point, Prod_Group, Prod_Category, Brand, Model, 
                                                            Colour, Is_Delivery, Quotation_Amount, Reason, Cancelled_Reason, Estimated_Pickup_Date, 
                                                            Pickup_Courier_Date, Device, Symptom_Code, Symptom_Desc, Cust_Arrival, Cust_Name, Status, 
                                                            DownPayment, DP_No, DP_Method, Acknowledge_Date, Assigned_Date, Completed_Date, Release_Date, 
                                                            Invoice_Date, DeskCharger, CarKit, RemovableAntenna, HeadSet, Battery, SimCard, ExtraCover, BatteryCover, LCD_text, Type,KOMITEAPPROVE,
                                                            A.NOKTP,A.ITEMWARRANTY,A.ACCESSORIESLAINYA,A.CASE,A.BOXUNIT,A.CGARGERCABLE,A.EXCESSAPPROVE,A.DEDUCAPPROVE,A.SUKUCADANGAPPROVE,A.PPNAPPROVE,
                                                            A.TAX_FEEAPPROVE,A.DELIVERY_FEEAPPROVE,A.TOTAL_FEEAPPROVE,A.SUKUCADANG,A.PPN,A.DETAILPART,EXCESS,DEDUCTIBLE,A.OBJECT,A.REPAIRID) 
                VALUES(IDPega,sysdate,tlogin,tnopolis,tstart,tend,tqqname,tserialno,tquotation,tbill,timei,tnohp,tgaransi,tpic,tanalisa,
                        tserivcefee,tsparepart,ttax,totherfee,tsts_approval,tdeliveryfee,ttotalfee, tremark,tPrincipal_Bill_No, tInsurance, 
                        tInsurance_Bill_No, tCollect_Point,tRepair_Point, tProd_Group, tProd_Category, tBrand, tModel, tColour, tIs_Delivery, tQuotation_Amount, 
                        tReason, tCancelled_Reason, tEstimated_Pickup_Date, tPickup_Courier_Date, tDevice, tSymptom_Code, tSymptom_Desc, tCust_Arrival, tCust_Name, 
                        tStatus, tDownPayment, tDP_No, tDP_Method, tAcknowledge_Date, tAssigned_Date, tCompleted_Date, tRelease_Date, tInvoice_Date, 
                        tDeskCharger, tCarKit, tRemovableAntenna, tHeadSet, tBattery, tSimCard, tExtraCover, tBatteryCover, tLCD_text, tType,tkomite,tnoktp,tittemwarranty,taccesorlainya,tcase,tsboxunit,tchargerchabel,
                        tExcessap,tDeductibleap,tsukucadangap,tppnap,ttaxap,tdeliveryfeeap,ttotalfeeap,tsukucadang,tppn,tpartjson,tExcess,tDeductible,tObject,tRepairID);
                ErrMsg := 'Success';
                COMMIT;
            EXCEPTION
            WHEN OTHERS THEN
                  ErrMsg := 'INSERT Error : ' || sqlerrm;
                  ROLLBACK;
                  RETURN;
            END;
            
        ELSIF status = 'update' THEN
            
                BEGIN
                    UPDATE POOLDATA.T_KLAIM_PORTAL_REKANAN a SET NOPOLIS = tnopolis, STARTDATE = tstart, ENDDATE = tend, QQNAME = tqqname, 
                                                     SERIALNO = tserialno, QUOTATIONNO = tquotation, BILLNO = tbill, IMEI = timei, NOHP = tnohp, 
                                                     JENIS_GARANSI = tgaransi, PIC = tpic, ANALISA = tanalisa, SERVICE_FEE = tserivcefee, SPAREPART_FEE = tsparepart, TAX_FEE = 
                                                     ttax, OTHER_FEE = totherfee, DELIVERY_FEE = tdeliveryfee, TOTAL_FEE = ttotalfee, REMARK = tremark, Principal_Bill_No = tPrincipal_Bill_No,
                                                     Insurance = tInsurance, Insurance_Bill_No = tInsurance_Bill_No, Collect_Point = tCollect_Point, Repair_Point = tRepair_Point,
                                                     Prod_Group = tProd_Group, Prod_Category = tProd_Category, Brand = tBrand, Model = tModel, Colour = tColour, Is_Delivery = tIs_Delivery,
                                                     Quotation_Amount = tQuotation_Amount, Reason = tReason, Cancelled_Reason = tCancelled_Reason, Estimated_Pickup_Date = tEstimated_Pickup_Date,
                                                     Pickup_Courier_Date = tPickup_Courier_Date, Device = tDevice, Symptom_Code = tSymptom_Code, Symptom_Desc = tSymptom_Desc, Cust_Arrival = tCust_Arrival,
                                                     Cust_Name = tCust_Name, Status = tStatus, DownPayment = tDownPayment, DP_No  = tDP_No, DP_Method = tDP_Method, Acknowledge_Date = tAcknowledge_Date,
                                                     Assigned_Date = tAssigned_Date, Completed_Date = tCompleted_Date, Release_Date = tRelease_Date, Invoice_Date = tInvoice_Date, DeskCharger = tDeskCharger,
                                                     CarKit = tCarKit, RemovableAntenna = tRemovableAntenna, HeadSet = tHeadSet, Battery = tBattery, SimCard = tSimCard, ExtraCover = tExtraCover,
                                                     BatteryCover = tBatteryCover, LCD_text = tLCD_text,STS_APPROVAL=tsts_approval,A.NOKTP= tnoktp,A.ITEMWARRANTY= tittemwarranty,A.ACCESSORIESLAINYA= taccesorlainya,
                                                     A.CASE=tcase,A.BOXUNIT = tsboxunit,A.CGARGERCABLE =tchargerchabel,A.EXCESSAPPROVE= tExcessap,A.DEDUCAPPROVE= tDeductibleap,A.SUKUCADANGAPPROVE= tsukucadangap,
                                                     A.PPNAPPROVE= tppnap,A.TAX_FEEAPPROVE= ttaxap,A.DELIVERY_FEEAPPROVE= tdeliveryfeeap,A.TOTAL_FEEAPPROVE= ttotalfeeap,A.SUKUCADANG=tsukucadang,
                                                     A.PPN= tppn,A.DETAILPART= tpartjson,EXCESS = tExcess,DEDUCTIBLE = tDeductible,A.OBJECT=tObject,A.REPAIRID=tRepairID
                    WHERE ID = IDPega;
                    ErrMsg := 'Success';
                EXCEPTION
                WHEN OTHERS THEN
                      ErrMsg := 'UPDATE Error : ' || sqlerrm;
                      ROLLBACK;
                      RETURN;
                END; 
                
        ELSIF status = '1' OR status = '2' OR status = '3' THEN
        
                BEGIN
                    UPDATE POOLDATA.T_KLAIM_PORTAL_REKANAN SET STS_APPROVAL = tsts_approval, REMARK = tremark
                    WHERE ID = IDPega;
                    ErrMsg := 'Success';
                EXCEPTION
                WHEN OTHERS THEN
                      ErrMsg := 'Error : ' || sqlerrm;
                      ROLLBACK;
                      RETURN;
                END;
                
        END IF;
    
    ELSIF tType = 'BROKER' THEN
        
        IF status = 'insert' THEN
        
            select count(1) into idcount from POOLDATA.T_KLAIM_PORTAL_REKANAN WHERE type='BROKER' and id=IDPega;
            
            if idcount = 0 then
            
        
                BEGIN
                    INSERT INTO POOLDATA.T_KLAIM_PORTAL_REKANAN (ID, INPUTDATE, TOTAL_FEE, DATEOFLOSS, INSURANCE, NOPOLIS, REMARK, LOGIN,
                                                                 EXCESS, OBJECT, CANCELLED_REASON, NOTE, STATUS, QQNAME, Type, DEDUCTIBLE,REPAIRID)
                    VALUES(IDPega, sysdate, ttotalfee, tDol, tInsurance, tnopolis, tremark, tlogin, tExcess, tObject, tCancelled_Reason, tNote, tStatus, tqqname, tType, tDeductible,tRepairID);
                    ErrMsg := 'Success';
                    COMMIT;
                EXCEPTION
                WHEN OTHERS THEN
                      ErrMsg := 'INSERT Error : ' || sqlerrm;
                      ROLLBACK;
                      RETURN;
                END;
            else 
                
                BEGIN
                    UPDATE POOLDATA.T_KLAIM_PORTAL_REKANAN SET TOTAL_FEE = ttotalfee, DATEOFLOSS = tDol, INSURANCE = tInsurance, NOPOLIS = tnopolis, DEDUCTIBLE = tDeductible,
                    REMARK = tremark, EXCESS = tExcess, OBJECT = tObject, CANCELLED_REASON = tCancelled_Reason, NOTE = tNote, STATUS = tStatus, QQNAME = tqqname 
                    WHERE ID = IDPega;
                    ErrMsg := 'Success';
                EXCEPTION
                WHEN OTHERS THEN
                      ErrMsg := 'Update Error : ' || sqlerrm;
                      ROLLBACK;
                      RETURN;
                END;
            
            end if;
            
        ELSIF status = 'update' THEN
        
            BEGIN
                    UPDATE POOLDATA.T_KLAIM_PORTAL_REKANAN SET TOTAL_FEE = ttotalfee, DATEOFLOSS = tDol, INSURANCE = tInsurance, NOPOLIS = tnopolis, DEDUCTIBLE = tDeductible,
                    REMARK = tremark, EXCESS = tExcess, OBJECT = tObject, CANCELLED_REASON = tCancelled_Reason, NOTE = tNote, STATUS = tStatus, QQNAME = tqqname 
                    WHERE ID = IDPega;
                    ErrMsg := 'Success';
                EXCEPTION
                WHEN OTHERS THEN
                      ErrMsg := 'Update Error : ' || sqlerrm;
                      ROLLBACK;
                      RETURN;
            END;
        
        END IF;
        
    
    END IF;
        

EXCEPTION
        WHEN OTHERS THEN
        ErrMsg := 'Proc Condition Error : ' || sqlerrm;
        ROLLBACK;
        RETURN;
END;

/