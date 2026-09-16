import re, os, glob, json
UP="/mnt/user-data/uploads"
FILES={os.path.basename(p):p for p in glob.glob(UP+"/*.xml")}
TYPES={'Rule-Obj-Activity':'Activity','Rule-RDB-SQL':'RDB List','Rule-Obj-Model':'Data Transform','Rule-HTML-Section':'Section','Rule-Obj-Flow':'Flow'}
SUFFIX={'Harness':'-Harness','Section':'-Section','Activity':'-Act','Data Transform':'-DT','RDB List':'-SQL','Flow':'-Flow'}

# --- Option 1: PEGA engine / OOTB activities to exclude entirely ---
ENGINE={n.lower() for n in [
 'Save','SaveNew','SaveSetup','PreSave','Validate','StandardValidate','ValidateNew','DictionaryValidation',
 'WorkLock','WorkUnlock','NewDefaults','NewInternalDefaults','createWorkPage','CreateWorkPage','CreateInstance',
 'View','Show-Harness','performAssignmentCheck','setPageErrors','setProcessState','setOutput',
 'GenerateID','AttachToWork','AddToFolder','acquireWorkObject','CheckForWarnings','CheckForCustomWarnings',
 'CheckCache','CheckKeys','CheckDuplicates','commitWithErrorHandling','AddWork','AddCoveredWork',
 'AllCoveredResolved','Resolve','UpdateStatus','SuspendFlows','RecalculateAndSave']}

def is_ootb(n): return n[:2].lower() in ('px','py','pz') or n=='CustomActivePage'
def is_engine(t,n): return t=='Activity' and n.lower() in ENGINE
def clean_rdb(n): return n.split(' GCNM ')[-1].strip().split(' ')[-1] if ' GCNM ' in n else n
def fname(name,jenis):
    c=f"{name}{SUFFIX[jenis]}.xml"; return c if c in FILES else None
def children_of(name,jenis):
    fn=fname(name,jenis)
    if not fn: return []
    x=open(FILES[fn],encoding='utf-8',errors='replace').read(); out=[];seen=set()
    for b in re.findall(r'<rowdata REPEATINGINDEX="\d+">.*?</rowdata>',x,re.S):
        if 'Embed-Reference-Rule' not in b: continue
        o=re.search(r'<pxRuleObjClass>([^<]*)</pxRuleObjClass>',b); n=re.search(r'<pyRuleName>([^<]*)</pyRuleName>',b)
        if not o or not n or o.group(1) not in TYPES or not n.group(1).strip(): continue
        t=TYPES[o.group(1)]; d=clean_rdb(n.group(1)) if t=='RDB List' else n.group(1)
        if is_ootb(d) or is_engine(t,d): continue
        k=(t,d)
        if k in seen: continue
        seen.add(k); out.append((t,d))
    return out

level2=[("Inbox_AS_KREDIT_Sect","Section"),("BrowseAutoKlaim","Section"),("ButtonPagingInbox","Section"),
 ("BrowseAutoKlaim_act","Activity"),("GetLinkAppClaim","Activity"),("GetListKorwil","Activity"),
 ("CreateCasePNCAgent_AsuransiKredit","Activity"),("INBOX_AS_KREDIT_ACT_CLAIMKREDIT","Activity"),
 ("GetReportClaimAsuransiKredit","Activity"),("REPORT_ASURANSI_KREDIT_ACT","Activity"),("DETAIL_ASURANSI_KREDIT","Activity"),
 ("CreateCasePNCAgent_AutoClaim","Activity"),("INBOX_AS_KREDIT_ACT_AUTOCLAIM","Activity"),
 ("GetReportClaimAutoClaim","Activity"),("REPORT_AUTO_CLAIM_ACT","Activity"),("DETAIL_AUTO_CLAIM","Activity"),
 ("CreateCasePNCAgent_Travel","Activity"),("INBOX_AS_KREDIT_ACT_TRAVEL","Activity"),
 ("GetReportClaimTravel","Activity"),("REPORT_TRAVEL_ACT","Activity"),("DETAIL_TRAVEL","Activity"),
 ("GenerateDLA_Askredit","Activity"),("CekPremi","Activity"),("ValidationInitial_act","Activity"),
 ("RemovePaginationFU","Activity"),("RemovePaginationOS","Activity")]

# --- Option 2: global dedup, expand each rule once at first occurrence ---
rows=[]        # (level,no,name,jenis,dupref)
first_no={}    # (type,name)->No where first expanded
def add(l,no,nm,jn,dup): rows.append((l,no,nm,jn,dup))
def recurse(name,jenis,no,level):
    key=(jenis,name)
    if key in first_no:
        add(level,no,name,jenis,first_no[key]); return   # duplicate -> mark, don't expand
    first_no[key]=no
    add(level,no,name,jenis,None)
    for i,(ct,cn) in enumerate(children_of(name,jenis),1):
        recurse(cn,ct,f"{no}.{i}",level+1)
add(0,"1","InboxAutoClaim","Harness",None)
first_no[("Harness","InboxAutoClaim")]="1"
for i,(nm,jn) in enumerate(level2,1):
    recurse(nm,jn,f"1.{i}",1)
json.dump(rows,open("rows.json","w"))

# --- needed list (custom, non-engine, no file) ---
refby={}; seenwalk=set()
def walk(name,jenis):
    if (jenis,name) in seenwalk: return
    seenwalk.add((jenis,name))
    for ct,cn in children_of(name,jenis):
        refby.setdefault((ct,cn),set()).add(name); walk(cn,ct)
for nm,jn in level2:
    refby.setdefault((jn,nm),set()).add("InboxAutoClaim"); walk(nm,jn)
needed=[]
for (t,n),parents in refby.items():
    if fname(n,t) is None: needed.append((t,n,f"{n}{SUFFIX[t]}.xml",sorted(parents)))
order={'Activity':0,'Section':1,'Data Transform':2,'Flow':3,'RDB List':4}
needed.sort(key=lambda r:(order.get(r[0],9),r[1].lower()))
json.dump(needed,open("needed.json","w"))
from collections import Counter
print("tree rows:",len(rows),dict(Counter(j for _,_,_,j,_ in rows)))
dups=sum(1 for *_,d in rows if d)
print("  of which duplicate-refs:",dups,"| unique expanded:",len(rows)-dups)
print("needed:",len(needed),dict(Counter(t for t,_,_,_ in needed)))
print("activities still needed:",[n for t,n,_,_ in needed if t=="Activity"])
