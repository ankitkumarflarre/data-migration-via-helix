"""Build explicit local quote forms from the reviewed PersonalHome inventory."""
import json,re
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
cat=json.loads((ROOT/'web/src/lib/personalhome/catalog.json').read_text())

def field(key,label,kind='text',required=False,**kw):
 return dict(key=key,label=label,type=kind,required=required,**kw)
def page(id,title,fields,**kw):return dict(id=id,title=title,fields=fields,**kw)
def inventory(id):return next(p for p in cat['pages'] if p['id']==id)
def extract(id):
 out=[]
 for f in inventory(id)['fields']:
  if f['type'] in ['action','static'] or f['rw']!='Read and Written' or 'Output' in f['tech'] or 'Private' in f['tech'] or f['enabled']=='Read-Only' or 'Hidden' in f['visibility'] or 'Internal' in f['visibility'] or f['control']=='Computed':continue
  typ={'int':'integer','float':'number','date':'date','boolean':'boolean'}.get(f['type'],'text')
  x=field(f['tech'],f['label'],typ,f['required']=='Required',group=f['group'],hint=f['validation'] if f['validation']!='—' else '',source=f['tech'])
  limit=re.search(r'MaxLength: (\d+)',f['validation'])
  if limit:x['maxLength']=int(limit[1])
  if typ in ['number','integer']:x['min']=0
  if typ=='boolean':x['default']=False
  out.append(x)
 return out

def patch(fields,key,**kw):
 f=next(f for f in fields if f['key']==key);f.update(kw)
def cond(key,*values):return dict(key=key,values=list(values))
account=extract('account'); co=[f for f in account if f['key'].startswith('PersonInput.')];account=[f for f in account if not f['key'].startswith('PersonInput.')]
for k in ['FirstName','LastName','PrimaryPhone','Email','Address1','City','ZipCode']:patch(account,'AccountInput.'+k,required=True)
for f in co:
 if f['key'] in ['PersonInput.FirstName','PersonInput.LastName']:f['required']=True
patch(account,'AccountInput.Email',type='email');patch(account,'AccountInput.ZipCode',type='zip');patch(account,'AccountInput.State',default='FL',maxLength=2)
location=extract('locationdetail')
for f in location:f['group']='Risk address';f['required']=not f['key'].endswith('Address2')
patch(location,'LocationInput.State',options=['FL'],default='FL');patch(location,'LocationInput.ZipCode',type='zip')
risk=extract('dwellinginfo')
risk=[f for f in risk if f['key'] not in ['DwellingInput.Form','DwellingInput.HurricaneDeductible','DwellingInput.Deductible']]
options={
'UseType':['Primary','Secondary','Seasonal3to6','Seasonal6Mths','FarmRanch','UnderConstruction','Rental'],
'OccupancyType':['Owner','Tenant','Unoccupied','Vacant'],'BuildingType':['Condo','Dwelling','Rowhouse','Townhouse','Modular/Prefab'],
'Construction':['Frame','Aluminum','PlasticSiding','Masonry','MasonryVeneer','Superior'],
'FoundationType':['Closed','Open'],'SwimmingPool':['None','Inground','AboveGround'],
'PoolSlide':['N','Y'],'PoolApprovedFence':['N','Y'],'PoolDivingBoard':['N','Y'],
'PrimaryHeatType':['Electric','Other','Gas','Unknown'],'SupplementalHeatType':['FireplaceInsert','Electric','Gas','Other','Unknown'],
'ElectricalSystem':['Breaker','Fuse','Other','Unknown'],'OpeningProtection':['ClassA','ClassB','ClassC','None','Unknown'],
'RoofType':['Architectural','Asphalt','Builtup','Concrete','Fiberglass','Metal','Other','Slate','TileBarrelTile','TileFlatTile','Wood'],
'Sprinkler':['None','SinglePoint','WholeHome','Shutoff','ClassB','ClassA'],'Wiring':['Copper','Aluminum','KnobAndTube','Unknown'],
'PlumbingType':['PVCCPVC','PEPEX','Copper','Polybutylene','GalvanizedSteel','Other'],'RoofGeometry':['Hip','Gable','Flat','Other','NA'],
'DistanceToFireStation':['Within5miles','Greaterthan5miles'],'DistanceToHydrant':['Within1000ft','Greaterthan1000ft'],
'SecondaryWaterResistance':['Yes','No','Unknown'],'DwellingAccessibleToFireEquipment':['N','Y','U'],
'RoofToWallAttachment':['ToeNails','Clips','SingleWraps','DoubleWraps','Unknown','NA'],
'RoofCover':['NonFBCEquivalent','FBCEquivalent','ReinforcedConcreteRoofDeck','NA'],'Terrain':['B','C','H','Unknown'],
'WindSpeedDesign':['100','110','120 mph+','Unknown'],'WindSpeedLocation':['100','110','120+','120WBDR','Unknown'],
'RoofDeckAttachment':['A6d@612','B8d@612','C8d@66','WoodDeck','MetalDeck','Reinforce','NA']}
for k,v in options.items():patch(risk,'DwellingInput.'+k,options=v)
for k,v in {'UseType':'Primary','OccupancyType':'Owner','Construction':'Frame','SwimmingPool':'None','ElectricalSystem':'Unknown','RoofGeometry':'Gable','Terrain':'Unknown','NumberOfStories':1,'NumberOfFamilies':1}.items():patch(risk,'DwellingInput.'+k,default=v)
patch(risk,'DwellingInput.YearBuilt',required=True,min=1600,requiredWhen=cond('DwellingInput.Form','HO3'))
patch(risk,'DwellingInput.NumberOfUnits',min=1,max=9);patch(risk,'DwellingInput.NumberOfFamilies',min=1,max=4);patch(risk,'DwellingInput.MonthsUnoccupied',min=0,max=12)
patch(risk,'DwellingInput.SquareFeet',min=1)
for k in ['PoolSlide','PoolApprovedFence','PoolDivingBoard']:patch(risk,'DwellingInput.'+k,showWhen=cond('DwellingInput.SwimmingPool','Inground','AboveGround'))
patch(risk,'DwellingInput.NumberOfFloor',showWhen=cond('DwellingInput.Form','HO6'))
coverage=extract('dwellingcoverage')
# Fields whose calculation depends on unavailable lookup tables are not editable inputs.
coverage=[f for f in coverage if f['key'] not in ['UnscheduledJewelryInput.Indicator','IncidentalFarmingPersonalLiabilityInput.Indicator','RiskInput.UseDeductibleByPeril','LineInput.CoveragePackage']]
patch(coverage,'CoverageADwellingInput.Limit',showWhen=cond('DwellingInput.Form','HO3'),required=True,min=40000)
patch(coverage,'CoverageCPersonalPropertyHO3Input.Limit',showWhen=cond('DwellingInput.Form','HO3'))
patch(coverage,'CoverageCPersonalPropertyHO46Input.Limit',showWhen=cond('DwellingInput.Form','HO6'),required=True,min=10000)
patch(coverage,'ReplacementCostDwellingInput.Indicator',showWhen=cond('DwellingInput.Form','HO3'),default=True)
patch(coverage,'ReplacementCostDwellingInput.ReplacementCostValue',showWhen=cond('ReplacementCostDwellingInput.Indicator',True),required=True,max=1000000)
patch(coverage,'DwellingInput.Deductible',options=['500','1000','2500'],default='500',required=True)
patch(coverage,'DwellingInput.HurricaneDeductible',options=['N','500','2%','5%','10%'],default='N',required=True)
patch(coverage,'OrdinanceOrLawInput.Limit',options=['N','25','50'],default='25')
patch(coverage,'LimitedFungiBacteriaInput.Limit',options=['N','10/50','25/50','50/50'],default='10/50')
patch(coverage,'LossAssessmentInput.Limit',showWhen=cond('LossAssessmentInput.Indicator',True),min=1000,max=50000)
patch(coverage,'CarportsPoolCagesInput.Limit',max=75000)
patch(coverage,'UnitsRegularlyRentedToOthersInput.Indicator',showWhen=cond('DwellingInput.Form','HO6'))
patch(coverage,'SpecialPersonalPropertyCoverageInput.Indicator',showWhen=cond('DwellingInput.Form','HO6'))
# Product document fixes these displayed coverages; they remain server derived.
coverage += [field('CoverageELiabilityInput.Limit','Personal Liability (E)','text',readOnly=True,default='300000',group='Base coverages'),field('CoverageFMedicalInput.Limit','Medical Payments (F)','text',readOnly=True,default='3000',group='Base coverages')]
uw=[]
for key,label in [('animals','Restricted dogs, exotic pets or guard dogs?'),('trampoline','Trampoline on the premises?'),('disrepair','Property for sale or in disrepair?'),('renovation','Property under renovation?'),('business','Business, day care or farming use?'),('financialHistory','Foreclosure or bankruptcy history?'),('criminalHistory','Criminal indictment history?'),('leadOrLandfill','Known lead paint or landfill exposure?'),('declined','Prior insurance declined or cancelled?'),('sinkhole','Known sinkhole activity?')]:
 uw.append(field('Underwriting.'+key,label,'boolean',True,group='Underwriting declarations',hint='Yes requires an underwriter to review; no automatic eligibility decision.'))
uw.append(field('Underwriting.Notes','Details for any Yes answers','textarea',group='Additional details'))
history=extract('insurancehistory')
for f in history:f['required']=f['key'] in ['PriorInsurance.CarrierName','PriorInsurance.EffectiveDate','PriorInsurance.ExpirationDate']
losses=extract('claimshistory')
for f in losses:f['required']=True
interests=[field('type','Interest type',required=True,options=['Mortgagee','Additional Insured','Additional Interest','Premium Finance']),field('name','Name',required=True),field('address','Mailing address',required=True),field('rank','Mortgage rank','integer',min=1,max=4),field('reference','Loan / reference number')]
pages=[
 page('newquote','New quote',[field('Quote.Product','Product',required=True,options=['Manatee FL Select PersonalHome'],default='Manatee FL Select PersonalHome'),field('Quote.State','Risk state',required=True,options=['FL'],default='FL'),field('DwellingInput.Form','Policy form',required=True,options=['HO3','HO6'],default='HO3'),field('PolicyInput.EffectiveDate','Effective date','date',True),field('Quote.DCTBilling','Include billing instructions','boolean',default=False)]),
 page('account','Applicant',account,collection=dict(key='coapplicants',label='Co-applicants',fields=co)),
 page('dwellinginfo','Risk schedule',location+risk),page('dwellingcoverage','Dwelling coverage',coverage),
 page('underwriting','Underwriting',uw,notice='These declarations capture the topics in the source document. Carrier underwriting rules and the full questionnaire still require integration.'),
 page('insurancehistory','Insurance history',[field('History.HasPrior','Prior insurance to disclose?','boolean',True)],collection=dict(key='priorInsurance',label='Prior policies',fields=history,when=cond('History.HasPrior',True))),
 page('claimshistory','Claims history',[field('Claims.HasLosses','Prior losses to disclose?','boolean',True)],collection=dict(key='losses',label='Prior losses',fields=losses,when=cond('Claims.HasLosses',True)),notice='Enter known losses manually. CLUE and insurance-score reports are not ordered by this application.'),
 page('summary','Coverage summary',[],notice='Coverage selections are saved. Premium calculation and carrier eligibility require the rating integration; this is not a priced or bindable quote.'),
 page('additionalinterests','Additional interests',[],collection=dict(key='interests',label='Additional interests',fields=interests)),
 page('billing','Billing instructions',[field('Billing.BillClass','Bill class',required=True,options=['Direct Bill','Mortgagee Escrow','Agency Bill']),field('Billing.PaymentPlan','Payment plan',required=True,options=['Annual','Semi-Annual','Quarterly']),field('Billing.Paperless','Paperless invoices','boolean',default=False)],when=cond('Quote.DCTBilling',True),notice='Save billing preferences only. No payment is collected and no installment schedule is generated.'),
 page('review','Review application',[],notice='Review and submit the captured application for rating. Submission records a local workflow status; it does not bind insurance or send data to a carrier.')]
schema=dict(version=1,pages=pages)
(ROOT/'internal/quote/schema.json').write_text(json.dumps(schema,indent=2)+'\n')
print('Created quote form schema:',sum(len(p['fields']) for p in pages),'fields')
