"""Seed fictional quotes through the local API; never overwrite existing demos.

Run with the quote server on 127.0.0.1:8080. No external services are called.
"""
import json
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import HTTPError

ROOT = Path(__file__).resolve().parents[1]
BASE = 'http://127.0.0.1:8080/api/quotes'
PAGES = json.loads((ROOT / 'internal/quote/schema.json').read_text())['pages']
INDEX = ROOT / 'data/demo-seed-index.json'


def call(method, path='', body=None):
    request = Request(BASE + path, data=None if body is None else json.dumps(body).encode(),
                      headers={'Content-Type': 'application/json'}, method=method)
    try:
        with urlopen(request, timeout=20) as response:
            return json.load(response)
    except HTTPError as error:
        raise RuntimeError(f'{method} {path}: {error.read().decode()}') from error


def save_index(index):
    INDEX.parent.mkdir(parents=True, exist_ok=True)
    temporary = INDEX.with_suffix('.tmp')
    temporary.write_text(json.dumps(index, indent=2) + '\n')
    temporary.replace(INDEX)


def fixture(name, form, city, county, zip_code, address, phone, billing=False):
    values = {
        'Quote.Product': 'Manatee FL Select PersonalHome', 'Quote.State': 'FL',
        'DwellingInput.Form': form, 'PolicyInput.EffectiveDate': '2027-01-01',
        'Quote.DCTBilling': billing,
        'AccountInput.FirstName': name, 'AccountInput.LastName': 'Demo',
        'AccountInput.PrimaryPhone': phone, 'AccountInput.Email': name.lower()+'.demo@example.com',
        'AccountInput.Address1': address, 'AccountInput.City': city,
        'AccountInput.State': 'FL', 'AccountInput.ZipCode': zip_code, 'AccountInput.County': county,
        'DwellingInput.UseType': 'Primary', 'DwellingInput.OccupancyType': 'Owner',
        'DwellingInput.BuildingType': 'Condo' if form == 'HO6' else 'Dwelling',
        'DwellingInput.YearBuilt': 2008, 'DwellingInput.Construction': 'Frame',
        'DwellingInput.RatedProtectionClass': '05', 'DwellingInput.NumberOfFamilies': 1,
        'DwellingInput.NumberOfUnits': 1, 'DwellingInput.SquareFeet': 2200,
        'DwellingInput.FoundationType': 'Closed', 'DwellingInput.SwimmingPool': 'None',
        'DwellingInput.NumberOfStories': 1, 'DwellingInput.PrimaryHeatType': 'Electric',
        'DwellingInput.BuildingCodeEffectivenessGrading': 3,
        'DwellingInput.ElectricalSystem': 'Breaker', 'DwellingInput.Wiring': 'Copper',
        'DwellingInput.NumberOfAmps': 200, 'DwellingInput.PlumbingType': 'PEPEX',
        'DwellingInput.NumberOfBathrooms': '2', 'DwellingInput.OpeningProtection': 'ClassA',
        'DwellingInput.RoofGeometry': 'NA' if form == 'HO6' else 'Hip',
        'DwellingInput.SecondaryWaterResistance': 'Yes', 'DwellingInput.RoofToWallAttachment': 'Clips',
        'DwellingInput.RoofCover': 'FBCEquivalent', 'DwellingInput.Terrain': 'B',
        'DwellingInput.WindSpeedDesign': '120 mph+', 'DwellingInput.WindSpeedLocation': '120+',
        'DwellingInput.RoofDeckAttachment': 'C8d@66', 'DwellingInput.RoofType': 'Asphalt',
        'DwellingInput.Sprinkler': 'None', 'DwellingInput.LockedSecurityGate': False,
        'DwellingInput.SecurityAttendant': False, 'DwellingInput.DistanceToFireStation': 'Within5miles',
        'DwellingInput.DistanceToHydrant': 'Within1000ft',
        'DwellingInput.RespondingFireDepartment': 'Demo Municipal Fire Department',
        'DwellingInput.DwellingAccessibleToFireEquipment': 'Y',
        'UpdatedServicesInfo.RemainingRoofLife': 18, 'UpdatedServicesInfo.RoofUpdateYear': 2022,
        'UpdatedServicesInfo.ElectricSystemUpdateYear': 2020,
        'UpdatedServicesInfo.PlumbingUpdateYear': 2020,
        'DwellingInput.Deductible': '1000', 'DwellingInput.HurricaneDeductible': '2%',
        'ReplacementCostDwellingInput.Indicator': form == 'HO3',
        'OrdinanceOrLawInput.Limit': '25', 'LimitedFungiBacteriaInput.Limit': '10/50',
        'ReplacementCostContentsInput.Indicator': True, 'RiskInput.EquipmentBreakdown': True,
        'WaterBackupAndSumpOverflowInput.Indicator': True,
        'IdentityFraudExpenseCoverageInput.Indicator': True,
        'History.HasPrior': True, 'Claims.HasLosses': False,
        'Billing.BillClass': 'Direct Bill', 'Billing.PaymentPlan': 'Annual', 'Billing.Paperless': True,
    }
    for key in ['Address1','City','State','ZipCode','County']:
        values['LocationInput.'+key] = values['AccountInput.'+key]
    for key in ['animals','trampoline','disrepair','renovation','business','financialHistory','criminalHistory','leadOrLandfill','declined','sinkhole']:
        values['Underwriting.'+key] = False
    values['Underwriting.Notes'] = 'FICTIONAL DEMO DATA — not a real applicant or insurance request.'
    if form == 'HO3':
        values.update({'CoverageADwellingInput.Limit':400000, 'CoverageBOtherStructuresInput.Limit':40000,
                       'CoverageCPersonalPropertyHO3Input.Limit':200000, 'CoverageDLossOfUseInput.Limit':80000,
                       'ReplacementCostDwellingInput.ReplacementCostValue':400000})
    else:
        values.update({'CoverageCPersonalPropertyHO46Input.Limit':75000,
                       'CoverageDLossOfUseInput.Limit':15000, 'DwellingInput.NumberOfFloor':'4',
                       'DwellingInput.SquareFeet':1250, 'LossAssessmentInput.Indicator':True,
                       'LossAssessmentInput.Limit':25000, 'SpecialPersonalPropertyCoverageInput.Indicator':True})
    collections = {'priorInsurance':[{'PriorInsurance.CarrierName':'Demo Harbor Insurance (fictional)',
        'PriorInsurance.EffectiveDate':'2026-01-01', 'PriorInsurance.ExpirationDate':'2027-01-01',
        'PriorInsurance.Limit':400000 if form == 'HO3' else 75000, 'PriorInsurance.Deductible':1000}]}
    return {'key':name.lower()+'-'+form.lower(), 'values':values, 'collections':collections, 'stop':'review'}


def demos():
    home = fixture('Avery','HO3','Tampa','Hillsborough','33602','100 Demo Harbor Lane','8135550101',True)
    home['collections']['coapplicants'] = [{'PersonInput.FirstName':'Casey','PersonInput.LastName':'Demo'}]
    home['collections']['interests'] = [{'type':'Mortgagee','name':'Demo Community Bank (fictional)',
        'address':'1 Fictional Bank Plaza, Tampa, FL 33602','rank':1,'reference':'DEMO-LOAN-1001'}]
    home['values'].update({'DwellingInput.SwimmingPool':'Inground','DwellingInput.PoolSlide':'N',
        'DwellingInput.PoolApprovedFence':'Y','DwellingInput.PoolDivingBoard':'N',
        'Billing.BillClass':'Mortgagee Escrow'})
    condo = fixture('Morgan','HO6','Miami','Miami-Dade','33101','200 Demo Bay Avenue, Unit 4A','3055550102')
    condo['values']['Claims.HasLosses'] = True
    condo['collections']['losses'] = [{'LossInput.LossType':'Water damage', 'LossInput.DateOfLoss':'2025-06-15',
        'LossInput.AmountPaid':2250, 'LossInput.Description':'DEMO: repaired appliance supply-line leak; fictional prior loss.'}]
    condo['collections']['interests'] = [{'type':'Additional Interest','name':'Demo Bay Condominium Association',
        'address':'200 Demo Bay Avenue, Miami, FL 33101','reference':'DEMO-UNIT-4A'}]
    draft = fixture('Riley','HO3','Orlando','Orange','32801','300 Demo Garden Way','4075550103')
    draft['stop'] = 'underwriting'
    draft['values'].update({'Underwriting.renovation':True,
        'Underwriting.Notes':'DEMO ONLY: kitchen cabinets being replaced; no structural work. Demonstrates an application awaiting underwriting review.'})
    return [home, condo, draft]


def main():
    index = json.loads(INDEX.read_text()) if INDEX.exists() else {}
    existing = {q['id']:q for q in call('GET')}
    for demo in demos():
        key = demo['key']; entry = index.get(key, {})
        if entry.get('id') in existing and entry.get('complete'):
            q = existing[entry['id']]
            print(f"Already present: {q['number']} — {demo['values']['AccountInput.FirstName']} Demo")
            continue
        if entry.get('id') in existing:
            q = existing[entry['id']]
        else:
            # Also detect the reserved demo email if a seed index was lost.
            match = next((q for q in existing.values() if q['values'].get('AccountInput.Email') == demo['values']['AccountInput.Email']), None)
            if match:
                index[key] = {'id':match['id'],'complete':True};save_index(index)
                print(f"Preserved existing demo: {match['number']}")
                continue
            keys = {f['key'] for f in PAGES[0]['fields']}
            q = call('POST', body={'values':{k:v for k,v in demo['values'].items() if k in keys}})
            index[key] = {'id':q['id'],'complete':False};save_index(index)
        while True:
            page = next(p for p in PAGES if p['id'] == q['current_page'])
            if page['id'] == 'review':
                q = call('POST','/'+q['id']+'/submit',{'version':q['version']});break
            keys = {f['key'] for f in page['fields'] if not f.get('readOnly')}
            stop = page['id'] == demo['stop']
            q = call('PUT','/'+q['id'],{'version':q['version'],'page':page['id'],
                'values':{k:v for k,v in demo['values'].items() if k in keys},
                'rows':demo['collections'].get(page.get('collection',{}).get('key'),[]),'advance':not stop})
            if stop:break
        index[key] = {'id':q['id'],'complete':True};save_index(index)
        print(f"Seeded {q['number']} — {demo['values']['AccountInput.FirstName']} Demo — {q['status']} / {q['current_page']}")


if __name__ == '__main__':
    main()
