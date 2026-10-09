"""Explicit source-to-application coverage, including non-input controls.

The HTML is data only. Carrier-backed actions remain disabled until a service is
connected; unavailable calculated values are never substituted with zero.
"""
from collections import Counter
import json


def complete(pages, catalog, root):
    by_id = {p['id']: p for p in pages}
    destinations = {'account': 'account', 'policydetail': 'newquote',
                    'locationdetail': 'dwellinginfo', 'claimshistory': 'claimshistory',
                    'insurancehistory': 'insurancehistory', 'additionalinterests': 'additionalinterests',
                    'pricing': 'summary', 'dwellinginfo': 'dwellinginfo', 'dwellingcoverage': 'dwellingcoverage'}

    def add(page, key, label, kind='text', **kw):
        f = dict(key=key, label=label, type=kind, required=False, **kw)
        by_id[page]['fields'].append(f)
        return f

    def unavailable(page, key, label, reason, **kw):
        return add(page, key, label, readOnly=True, unavailable=reason, **kw)

    def action(page, key, label, reason, kind='unavailable'):
        a = dict(key=key, label=label, kind=kind, reason=reason)
        by_id[page].setdefault('actions', []).append(a)
        return a

    # Fields described only in the narrative have local keys, not guessed carrier mappings.
    narrative = []
    def extra(page, key, label, kind='text', **kw):
        if kind in ['number','integer']: kw.setdefault('min',0)
        f = add(page, key, label, kind, **kw)
        f['narrativeSource'] = label
        narrative.append(dict(page=page, key=key, label=label, status='read-only' if f.get('readOnly') else 'editable'))
        return f

    extra('account', 'Applicant.EntityType', 'Entity type', group='Applicant Info', options=['Individual', 'Trust', 'Corporation', 'Partnership', 'Other'], note='Captured locally; carrier entity codes require confirmation.')
    extra('account', 'Applicant.DateOfBirth', 'Date of birth', 'date', group='Applicant Info')
    extra('account', 'Applicant.Producer', 'Producer', group='Producer', note='Enter the producer name or code. Producer-directory lookup is not connected.')
    extra('account', 'Applicant.Territory', 'Territory', readOnly=True, unavailable='Awaiting geocoding service', group='Geocoding')
    extra('account', 'Applicant.FireDistrict', 'Fire district', readOnly=True, unavailable='Awaiting geocoding service', group='Geocoding')
    for key, label, kind in [
        ('PriorPolicyNumber', 'Prior policy number', 'text'), ('PriorCarrier', 'Prior carrier', 'text'),
        ('EPolicyDiscount', 'E-policy discount', 'boolean'), ('SubscriptionAgreement', 'Subscription agreement flag', 'boolean'),
        ('WritingCompany', 'Writing company', 'text'), ('ConsentToRate', 'Consent to rate', 'boolean'),
        ('CTRFactor', 'Consent-to-rate factor', 'number'), ('CitizensLegacyPremium', 'Citizens conversion legacy premium', 'number'),
        ('CitizensCapping', 'Citizens conversion capping details', 'textarea'), ('RiskOrigin', 'Risk origin', 'text')]:
        extra('newquote', 'PolicyDetails.'+key, label, kind, group='Policy information',
              **({'readOnly':True,'unavailable':'Awaiting signed subscriber agreement'} if key=='SubscriptionAgreement' else {}))
    for key, label, kind in [
        ('SwimmingPool', 'Swimming pool on premises?', 'boolean'), ('HotTub', 'Hot tub on premises?', 'boolean'),
        ('ForSale', 'Property for sale?', 'boolean'), ('Disrepair', 'Property in disrepair?', 'boolean'),
        ('DayCare', 'Day care use?', 'boolean'), ('Farming', 'Farming use?', 'boolean'),
        ('IncidentalOccupancy', 'Incidental occupancy?', 'boolean'), ('Foreclosure', 'Foreclosure history?', 'boolean'),
        ('Bankruptcy', 'Bankruptcy history?', 'boolean'), ('LeadPaint', 'Known lead paint?', 'boolean'),
        ('Landfill', 'Landfill exposure?', 'boolean'), ('RenovationCompletionDate', 'Renovation completion date', 'date'),
        ('RenovationValue', 'Renovation dollar value', 'number')]:
        extra('underwriting', 'Underwriting.'+key, label, kind, group='Additional underwriting details',
              **({'showWhen':{'key':'Underwriting.renovation','values':[True]}} if key.startswith('Renovation') else {}))
    extra('insurancehistory', 'History.InsuredPast30Days', 'Insurance within the past 30 days?', 'boolean', group='Prior insurance')
    by_id['insurancehistory']['collection']['fields'].append(dict(key='PriorInsurance.PolicyNumber',label='Prior policy number',type='text',required=False,narrativeSource='Prior policy number'))
    narrative.append(dict(page='insurancehistory',key='PriorInsurance.PolicyNumber',label='Prior policy number',status='editable collection'))
    by_id['claimshistory']['collection']['fields'].append(dict(key='LossInput.Source',label='Loss source',type='text',required=False,default='Applicant disclosure',narrativeSource='Loss source'))
    narrative.append(dict(page='claimshistory',key='LossInput.Source',label='Loss source',status='editable collection'))
    for key,label in [('InsuranceScore','Insurance score'),('InsuranceScoreOverride','Insurance score override'),('AdverseActionReasons','Adverse action reasons'),('CLUEStatus','CLUE report status')]:
        extra('claimshistory','Reports.'+key,label,readOnly=True,unavailable='Not available — report provider and authorized-role service are not connected',group='Reports')
    for label in ['Order CLUE report','Order insurance score','FCRA disclosure','FCRA NCF disclosure','Run eligibility check']:
        action('claimshistory','Reports.'+label,label,'Report provider, disclosure text and authorized-role service are not connected.')
    for key,label in [('DeclineMessages','Decline messages'),('ReferralMessages','Referral messages'),('SubscriberAgreement','Subscriber agreement and LPOA')]:
        extra('summary','Rating.'+key,label,readOnly=True,unavailable='Not evaluated — carrier service is not connected',group='Carrier review')
    for label in ['Premium detail','Coverage calculator','Quote sheet','Sign subscriber agreement and LPOA','Bind']:
        action('summary','Rating.'+label,label,'Requires carrier rating, approved documents, e-signature or binding integration.')
    extra('billing','Billing.InitialPaymentMethod','Initial payment method',options=['EFT','Credit card'],group='Initial payment',note='Preference only. No bank or card details are collected and no payment is processed.')
    for key,label in [('InitialPaymentAmount','Initial payment amount'),('PaymentStatus','Payment status'),('InstallmentDueDates','Installment due dates'),('InstallmentAmounts','Installment amounts')]:
        extra('billing','Billing.'+key,label,readOnly=True,unavailable='Awaiting billing provider',group='Billing summary')
    action('billing','Billing.Pay','Make a payment','Billing provider is not connected. No payment will be taken.')

    local_controls = {
        ('account',8): 'collection-add', ('account',13): 'collection-remove',
        ('claimshistory',7): 'collection-remove', ('claimshistory',8): 'collection-add',
        ('insurancehistory',6): 'collection-remove', ('insurancehistory',7): 'collection-add',
        ('additionalinterests',2): 'collection-details', ('additionalinterests',3): 'collection-remove', ('additionalinterests',4): 'collection-add',
    }
    alias = {('additionalinterests',5): 'name'}
    computed = {
        'AccountInput.Name': ('fullName', None),
        'PolicyInput.Term': ('constant', 12),
        'PolicyInput.ExpirationDate': ('expirationDate', None),
        'AccountOutputNonShredded.PrimaryInsuredText': ('constant', 'Primary Insured'),
        'PersonOutputNonShredded.CoapplicantLabel': ('coapplicantLabel', None),
        'LossOutputNonShredded.NumberDisplay': ('rowNumber', None),
        'AdditionalOtherInterestInput.Description': ('interestDescription', None),
        'WaterBackupAndSumpOverflowInput.Limit': ('constant', 10000),
        'IdentityFraudExpenseCoverageInput.Limit': ('constant', 15000),
        'CoverageDLossOfUseOutput.IncludedLimit': ('lossOfUseIncluded', None),
        'Geocode.GeocodeStatusDisplay': ('constant', 'Unverified'),
        'LocationGeocode.GeocodeStatusDisplay': ('constant', 'Unverified'),
    }
    reasons = {'account':'Party search and PBBI geocoding are not connected.',
               'policydetail':'Carrier rate-version service is not connected.',
               'locationdetail':'Location geocoding is not connected.',
               'pricing':'Carrier rating and document-generation services are not connected.'}
    entries=[]
    for source_page in catalog['pages']:
        source_id=source_page['id']; target=destinations[source_id]; p=by_id[target]
        for source in source_page['fields']:
            ref=f"{source_id}:{source['id']}"; key=source['tech']; label=source['label']
            row=dict(sourcePage=source_id, sourceID=source['id'], sourceKey=key, label=label, targetPage=target)
            if (source_id,source['id']) in local_controls:
                role=local_controls[(source_id,source['id'])]
                p['collection'].setdefault('sourceControls',{})[role]=ref
                row.update(status='local action',target=role)
            elif source['type']=='action':
                kind='unavailable'
                if source_id=='locationdetail' and source['id'] in [11,12,13,14]:kind={11:'save-location',12:'cancel-location',13:'clear-location',14:'copy-address'}[source['id']]
                a=action(target,key,label,reasons.get(source_id,'Carrier service is not connected.'),kind)
                a['sourceRefs']=[ref];row.update(status='local action' if kind!='unavailable' else 'unavailable service action',target=key)
            elif source['type']=='static':
                f=add(target,key,label,readOnly=True,default='Prior losses',group=source['group'])
                f['sourceRefs']=[ref];row.update(status='read-only',target=key)
            else:
                collection=p.get('collection',{})
                collection_fields=collection.get('fields',[])
                lookup=alias.get((source_id,source['id']),key)
                f=next((f for f in p['fields']+collection_fields if f['key']==lookup),None)
                if f is None:
                    typ={'int':'integer','float':'number','boolean':'boolean','date':'date'}.get(source['type'],'text')
                    f=dict(key=key,label=label,type=typ,required=False,group=source['group'])
                    if key in computed:
                        name,value=computed[key];f.update(readOnly=True,derive=name)
                        if value is not None:f['default']=value
                    elif key in ['AccountPrivate.XHTMLMapsBing','AccountPrivate.XHTMLMapsGoogle']:
                        f.update(readOnly=True,link='bing' if key.endswith('Bing') else 'google')
                    elif key=='AccountPBBI.Latitude / AccountPBBI.Longitude':
                        f.update(readOnly=True,hidden=True,unavailable='Awaiting PBBI coordinates')
                    elif key in ['DwellingInput.Form','DwellingInput.Deductible','DwellingInput.HurricaneDeductible']:
                        f.update(readOnly=True,note='Edit on New quote.' if key=='DwellingInput.Form' else 'Edit on Dwelling coverage.')
                    elif key.startswith('Coverage') and 'IncludedLimit' in key:
                        f.update(readOnly=True,unavailable='Awaiting carrier included-factor table')
                    elif source['rw']=='Read' or source['enabled'].startswith('Read-Only'):
                        f.update(readOnly=True,unavailable='Awaiting carrier rating service')
                    elif typ=='boolean':f['default']=False
                    if key=='LineInput.CoveragePackage':f.update(options=['Standard','Deluxe','Plus'],default='Standard',note='Selection is saved. Package limit defaults require the carrier CoverageDefaults table.')
                    if key=='RiskInput.UseDeductibleByPeril':f['note']='Selection is saved. Peril-specific deductibles require carrier configuration.'
                    if key in ['UnscheduledJewelryInput.Indicator','IncidentalFarmingPersonalLiabilityInput.Indicator']:f['note']='Selection is captured for review. Automatic reconciliation requires carrier minimum-limit or farming-type rules.'
                    if key=='PolicyAdmin.UseDCTFormsAndMessages':f['default']=True
                    if key=='AccountInput.Name':f['hidden']=True
                    if key in ['PersonOutputNonShredded.CoapplicantLabel','LossOutputNonShredded.NumberDisplay','AdditionalOtherInterestInput.Description']:
                        collection_fields.append(f)
                    else:p['fields'].append(f)
                f.setdefault('sourceRefs',[]).append(ref)
                row.update(status='internal' if f.get('hidden') else 'unavailable service value' if f.get('unavailable') else 'read-only' if f.get('readOnly') else 'editable',target=f['key'])
            entries.append(row)

    # Respect the table's explicit visibility rather than guessing from a label.
    risk=by_id['dwellinginfo']['fields']
    for f in risk:
        if f['key']=='DwellingInput.NumberOfFloor':f['showWhen']={'key':'DwellingInput.Form','values':['HO3']}
        if f['key']=='DwellingInput.MonthsUnoccupied':f['showWhen']={'key':'DwellingInput.UseType','values':['Primary','FarmRanch','UnderConstruction','Rental']}
        if f['key'] in ['DwellingInput.PoolSlide','DwellingInput.PoolApprovedFence','DwellingInput.PoolDivingBoard']:f.pop('showWhen',None)
    for p in pages:
        for f in p['fields']:
            if f['key']=='LineInput.CoveragePackage':f.update(options=['Standard','Deluxe','Plus'],default='Standard',note='Selection is saved. Package limit defaults require the carrier CoverageDefaults table.')
            if f['key']=='RiskInput.UseDeductibleByPeril':f['note']='Selection is saved. Peril-specific deductibles require carrier configuration.'
            if f['key'] in ['UnscheduledJewelryInput.Indicator','IncidentalFarmingPersonalLiabilityInput.Indicator']:f['note']='Selection is captured for review. Automatic reconciliation requires carrier minimum-limit or farming-type rules.'
            if f['key']=='DwellingInput.Deductible':f['disabledWhen']={'key':'RiskInput.UseDeductibleByPeril','values':[True]}
    # Separate provider panels are visible on their parent pages; no fake report result can unlock a journey page.
    by_id['claimshistory']['servicePanels']=[
        dict(title='CLUE loss history report',text='Appears after a successful CLUE response with at least one claim. No report has been ordered.',fields=['Claim date','Cause of loss','Individual payment amounts','Claim detail'],actions=['Detail','Delete CLUE claim','Supplement CLUE claim','OK','Cancel']),
        dict(title='Insurance score detail',text='Available only after a provider response. Score overrides require an authorized underwriter or manager.',fields=['Insurance score','Adverse action reasons'],actions=['Override score'])]
    by_id['summary']['servicePanels']=[dict(title='Subscriber agreement and LPOA',text='Approved agreement text and e-signature service are not connected. No acknowledgement or signature has been recorded.',fields=['Insured email address','Agreement acknowledgement'],actions=['Acknowledge and send for signature'])]
    # Fixed limits mentioned in calculation notes are displayed beside their selection.
    for key,label,value in [('WaterBackupOfSewersOrDrainsInput.Limit','Water backup of sewers or drains limit',10000),('PersonalInjuryInput.Limit','Personal injury limit',5000),('IncidentalFarmingPersonalLiabilityInput.Limit','Incidental farming liability limit',10000),('SpecialComputerCoverageInput.Limit','Special computer coverage limit',3000)]:
        extra('dwellingcoverage',key,label,'number',readOnly=True,default=value,derive='constant',group='Additional coverage limits')

    audit=dict(source=catalog['source'],sha256=catalog['sha256'],entries=entries,narrative=narrative)
    (root/'web/src/lib/personalhome/field-audit.json').write_text(json.dumps(audit,indent=2)+'\n')
    lines=['# PersonalHome field coverage audit','',f"Source SHA-256: `{catalog['sha256']}`.",'',
           'All 170 detailed inventory rows are accounted for below. This is field/control coverage, not a claim that carrier integrations are complete.',
           'Policy Information is suppressed by the source LOB: its fields appear under New quote. Pricing is suppressed: its fields appear under Coverage summary. Location Detail appears inside Risk schedule. Existing interest names retain the local `name` storage key for compatibility.',
           'Carrier outputs remain unavailable (not zero or fabricated). Hidden/internal source fields remain hidden in the form. Local source controls use the existing collection editor; AOI Details expands an editable row. Location OK saves a draft, Cancel restores saved values, and Delete clears the draft address only.',
           'The source table explicitly says Floor Unit is visible when Form is NOT HO6; this overrides the previous implementation. Pool detail fields are always visible as documented. Months unoccupied is hidden for secondary/seasonal usage.', '',
           '| Source page | Inventory rows | Editable | Read-only | Internal | Local actions | Unavailable values/actions |','|---|---:|---:|---:|---:|---:|---:|']
    for sp in catalog['pages']:
        counts=Counter(r['status'] for r in entries if r['sourcePage']==sp['id'])
        lines.append(f"| {sp['name']} | {len(sp['fields'])} | {counts['editable']} | {counts['read-only']} | {counts['internal']} | {counts['local action']} | {counts['unavailable service value']+counts['unavailable service action']} |")
    for sp in catalog['pages']:
        lines += ['', '## '+sp['name'],'','| ID | Source field/control | Application page | Implementation |','|---|---|---|---|']
        for r in entries:
            if r['sourcePage']==sp['id']:lines.append(f"| {r['sourceID']} | {r['label']} (`{r['sourceKey']}`) | {r['targetPage']} | {r['status']} → `{r['target']}` |")
    lines += ['','## Narrative-only coverage','', 'Narrative-only fields use local application keys unless a technical key is supplied. No carrier mapping is inferred. These additions are optional so existing saved drafts remain usable.','', '| Page | Field | Implementation |','|---|---|---|']
    lines += [f"| {r['page']} | {r['label']} (`{r['key']}`) | {r['status']} |" for r in narrative]
    lines += ['','## Unavailable services and unresolved source detail','',
      '- Applicant: producer-directory/party search and Maprisk/PBBI geocoding are unavailable. Producer can be entered manually; no address is marked verified.',
      '- Underwriting: the document supplies topics but not exact carrier questions/rules. All named topics are captured locally; decline/referral results are explicitly unavailable.',
      '- Insurance history: the source does not provide its carrier dropdown list. Carrier name remains free text. Advantage-only liability limits are outside this FL Select product.',
      '- Claims, CLUE and insurance score: ordering, FCRA consent text, role permissions and response contracts are missing. Report panels expose documented fields/actions in an unavailable state; they do not claim a report exists. Flood-specific navigation gating requires a carrier rule contract.',
      '- Coverage: Cov B/C included-factor tables, package defaults, jewelry minimum, farming-type rules and peril deductible definitions are not supplied. Selections are saved for review, with explanatory notes. Included D uses the documented 20% HO3 / 40% HO6 factors.',
      '- Coverage summary/Pricing: all four premium displays, form controls, messages, subscriber agreement and Bind are present; real rating, PDFs, signatures and binding remain unavailable.',
      '- Payment/Billing summary: bill class, plan, paperless and method preferences are stored. Due dates, amounts and payment execution require the billing provider. No payment credentials are collected.',
      '- Commit: the local Review page submits for rating only. Carrier policy number, In-Force status, IVANS/Insvista notifications, inspection, Zesty and Data Insights are not fabricated.',
      '- Narrative capping fields are not individually named. A local capping-details input captures notes; exact carrier fields require the authoritative schema. Subscriber/FCRA legal text is also absent.', '']
    (root/'docs/PERSONALHOME_FIELD_AUDIT.md').write_text('\n'.join(lines))
