"""Classifies every quote-flow input (internal/quote/schema.json) against the HO rater's Policy Data sheet.

Categories:
  sheet     – the value is in the sheet (as is, or with a value translation)
  derived   – worked out from sheet columns with a stated assumption
  partial   – the sheet has the column, but values are missing or don't fit for some rows
  default   – no data needed: a fixed product value or the quote's own default is right
  missing   – not in the rater data at all
  n/a       – inside a section that the data says does not apply (e.g. no losses)
"""
import json, collections, sys

S = 'internal/quote/schema.json'
C = {
    # ---- New quote ----
    'Quote.Product': ('default', '', 'Single product: Manatee FL Select PersonalHome.'),
    'Quote.State': ('default', '', 'FL for every row.'),
    'DwellingInput.Form': ('sheet', 'B Form', 'HO3 / HO6 match the options.'),
    'PolicyInput.EffectiveDate': ('sheet', 'F Effective Date', ''),
    'PolicyDetails.PriorPolicyNumber': ('partial', 'BV Citizens Policy Number', 'Column exists but is empty in every test row.'),
    'PolicyDetails.PriorCarrier': ('derived', 'G Risk Origin', 'FLCPIC → Citizens Property Insurance Corporation.'),
    'PolicyDetails.EPolicyDiscount': ('missing', '', ''),
    'PolicyDetails.WritingCompany': ('missing', '', 'Could be a fixed company value once agreed.'),
    'PolicyDetails.ConsentToRate': ('sheet', 'AX Consent to Rate', '0/1 → No/Yes (all 0 in test data).'),
    'PolicyDetails.CTRFactor': ('missing', '', 'Rater has only the flag.'),
    'PolicyDetails.CitizensLegacyPremium': ('partial', 'AY FL CPIC Expiring Premium (Term 1)', 'Only term-1 rows (698 of 1,394) have it.'),
    'PolicyDetails.CitizensCapping': ('derived', 'AZ, BA, BB, BC, BD', 'Free-text summary built from the capping columns.'),
    'PolicyDetails.RiskOrigin': ('sheet', 'G Risk Origin', ''),
    'PolicyInput.Description': ('missing', '', 'Optional free text.'),
    # ---- Applicant ----
    'AccountInput.FirstName': ('missing', '', 'Rater has no insured name.'),
    'AccountInput.MiddleName': ('missing', '', ''),
    'AccountInput.LastName': ('missing', '', 'Rater has no insured name.'),
    'AccountInput.PrimaryPhone': ('missing', '', ''),
    'AccountInput.Email': ('missing', '', ''),
    'AccountInput.Address1': ('missing', '', 'No mailing address.'),
    'AccountInput.Address2': ('missing', '', ''),
    'AccountInput.City': ('missing', '', ''),
    'AccountInput.State': ('default', '', 'FL default.'),
    'AccountInput.ZipCode': ('missing', '', ''),
    'AccountInput.County': ('derived', 'C County', 'Only if the mailing county is the risk county (assumption).'),
    'Applicant.EntityType': ('default', '', 'Individual.'),
    'Applicant.DateOfBirth': ('missing', '', ''),
    'Applicant.Producer': ('missing', '', ''),
    'PersonInput.FirstName': ('n/a', '', 'Co-applicants: optional, none in the data.'),
    'PersonInput.MiddleName': ('n/a', '', ''),
    'PersonInput.LastName': ('n/a', '', ''),
    # ---- Risk schedule ----
    'LocationInput.Address1': ('missing', '', 'Risk address is not in the rater (only county).'),
    'LocationInput.Address2': ('missing', '', ''),
    'LocationInput.City': ('missing', '', ''),
    'LocationInput.State': ('default', '', 'FL.'),
    'LocationInput.ZipCode': ('missing', '', ''),
    'LocationInput.County': ('sheet', 'C County', 'Upper-case to match the county list.'),
    'DwellingInput.UseType': ('partial', 'H Occupancy', 'Primary/Secondary map; Seasonal must be split into Seasonal3to6 / Seasonal6Mths (not in data).'),
    'DwellingInput.OccupancyType': ('sheet', 'I Occupied By', 'O → Owner, T → Tenant.'),
    'DwellingInput.MonthsUnoccupied': ('missing', '', 'Optional.'),
    'DwellingInput.BuildingType': ('derived', 'B Form', 'HO6 → Condo, HO3 → Dwelling (assumption: no townhouse/rowhouse distinction).'),
    'DwellingInput.YearBuilt': ('sheet', 'J Year Built', ''),
    'DwellingInput.Construction': ('sheet', 'K Construction', 'Masonry Veneer → MasonryVeneer; all values fit.'),
    'DwellingInput.RatedProtectionClass': ('sheet', 'L Protection Class', ''),
    'DwellingInput.NumberOfFamilies': ('default', '', 'Default 1.'),
    'DwellingInput.NumberOfUnits': ('sheet', 'O Units Within the Fire Division', ''),
    'DwellingInput.NumberOfFloor': ('missing', '', 'The unit\'s floor is not in the rater (column M holds the building\'s floors for HO6).'),
    'DwellingInput.FoundationType': ('missing', '', ''),
    'DwellingInput.SquareFeet': ('missing', '', 'Required; not in the rater.'),
    'DwellingInput.SwimmingPool': ('missing', '', ''),
    'DwellingInput.PoolSlide': ('n/a', '', 'Only asked when there is a pool.'),
    'DwellingInput.PoolApprovedFence': ('n/a', '', 'Only asked when there is a pool.'),
    'DwellingInput.PoolDivingBoard': ('n/a', '', 'Only asked when there is a pool.'),
    'DwellingInput.NumberOfStories': ('sheet', 'M Number of Stories', 'HO3 rows; for HO6 column M is the building\'s floors.'),
    'DwellingInput.PrimaryHeatType': ('missing', '', ''),
    'DwellingInput.SupplementalHeatType': ('missing', '', ''),
    'DwellingInput.BuildingCodeEffectivenessGrading': ('partial', 'S BCEG', 'Whole numbers only: 1,002 rows say “Ungraded”.'),
    'DwellingInput.ElectricalSystem': ('missing', '', ''),
    'DwellingInput.OpeningProtection': ('sheet', 'Z Opening Protection', 'Class A/Class B/None → ClassA/ClassB/None.'),
    'DwellingInput.RoofType': ('missing', '', 'Required; the rater has roof covering compliance, not roof material.'),
    'DwellingInput.Sprinkler': ('sheet', 'R Sprinkler System', 'Neither → None, Class B → ClassB.'),
    'DwellingInput.LockedSecurityGate': ('default', '', 'Default No.'),
    'DwellingInput.SecurityAttendant': ('default', '', 'Default No.'),
    'DwellingInput.Wiring': ('missing', '', ''),
    'DwellingInput.NumberOfAmps': ('missing', '', ''),
    'DwellingInput.PlumbingType': ('missing', '', ''),
    'DwellingInput.RoofGeometry': ('sheet', 'W Roof Shape', 'Hip Roof → Hip.'),
    'DwellingInput.DistanceToFireStation': ('missing', '', 'Required; geocoding/protection data, not in the rater.'),
    'DwellingInput.DistanceToHydrant': ('missing', '', 'Required; not in the rater.'),
    'DwellingInput.RespondingFireDepartment': ('missing', '', 'Required; not in the rater.'),
    'UpdatedServicesInfo.RemainingRoofLife': ('missing', '', ''),
    'UpdatedServicesInfo.ElectricSystemUpdateYear': ('missing', '', ''),
    'UpdatedServicesInfo.HeatingUpdateYear': ('missing', '', ''),
    'UpdatedServicesInfo.PlumbingUpdateYear': ('missing', '', ''),
    'UpdatedServicesInfo.RoofUpdateYear': ('missing', '', ''),
    'DwellingInput.SecondaryWaterResistance': ('sheet', 'Y Secondary Water Resistance', 'SWR → Yes, No SWR → No.'),
    'DwellingInput.DwellingAccessibleToFireEquipment': ('missing', '', ''),
    'DwellingInput.RoofToWallAttachment': ('sheet', 'U Roof Wall Connection', 'Values match the options.'),
    'DwellingInput.RoofCover': ('sheet', 'T Roof Covering', 'All three rater values have an option (incl. ReinforcedConcreteRoofDeck).'),
    'DwellingInput.Terrain': ('sheet', 'X Terrain Exposure', 'B / C.'),
    'DwellingInput.FloorToFoundationAttachment': ('missing', '', ''),
    'DwellingInput.WindSpeedDesign': ('sheet', 'AA Wind Speed Design', '“Greater than or equal to 100” → 100, …120 → 120 mph+.'),
    'DwellingInput.WindSpeedLocation': ('sheet', 'AB Wind Speed Location', '…120 and WBDR → 120WBDR.'),
    'DwellingInput.RoofDeckAttachment': ('sheet', 'V Roof Deck Attachment', 'Level A/B/C, Reinforced Concrete, Wood/Metal deck all have an option.'),
    'DwellingInput.NumberOfBathrooms': ('missing', '', ''),
    # ---- Dwelling coverage ----
    'CoverageADwellingInput.Limit': ('sheet', 'AC Coverage A', ''),
    'CoverageBOtherStructuresInput.Limit': ('sheet', 'AD Coverage B', ''),
    'CoverageCPersonalPropertyHO3Input.Limit': ('sheet', 'AE Coverage C', 'HO3 rows.'),
    'CoverageCPersonalPropertyHO46Input.Limit': ('sheet', 'AE Coverage C', 'HO6 rows.'),
    'CoverageDLossOfUseInput.Limit': ('missing', '', 'Optional; the quote can calculate it from A.'),
    'DwellingInput.Deductible': ('sheet', 'AG All Peril Deductible', '500 / 1000 / 2500.'),
    'DwellingInput.HurricaneDeductible': ('sheet', 'AH Hurricane Deductible', '0.02 → 2%, 0.05 → 5%, 0.1 → 10%, 500; Ex-Wind → N (4 rows).'),
    'ReplacementCostDwellingInput.Indicator': ('missing', '', 'Defaults to Yes in the quote, which then requires a replacement cost value.'),
    'ReplacementCostDwellingInput.ReplacementCostValue': ('missing', '', 'Required while the indicator is Yes (the default); no RCE value in the rater.'),
    'SinkholeInput.Indicator': ('sheet', 'AP Sinkhole Coverage', ''),
    'OrdinanceOrLawInput.Limit': ('sheet', 'AK Ordinance or Law', '0.25 → 25, 0.5 → 50.'),
    'WaterBackupAndSumpOverflowInput.Indicator': ('sheet', 'AO Water Back Up', 'Same rater column also fits the sewers/drains cover; one has to be chosen.'),
    'LimitedFungiBacteriaInput.Limit': ('sheet', 'AQ Limited Fungi Sect. 1', '$10,000 Included → 10/50 (every row).'),
    'LossAssessmentInput.Indicator': ('derived', 'AL Loss Assessment', '“Increase to $2,000” → Yes; “$1,000 Included” → No.'),
    'LossAssessmentInput.Limit': ('sheet', 'AL Loss Assessment', '2000 when increased.'),
    'CarportsPoolCagesInput.Limit': ('sheet', 'AM Screened Enclosure', '0 in every row.'),
    'ReplacementCostContentsInput.Indicator': ('sheet', 'AN Replacement Cost on Contents', ''),
    'IdentityFraudExpenseCoverageInput.Indicator': ('missing', '', 'Optional add-on.'),
    'UnscheduledJewelryInput.Indicator': ('missing', '', ''),
    'UnscheduledJewelryInput.Limit': ('missing', '', ''),
    'WaterBackupOfSewersOrDrainsInput.Indicator': ('derived', 'AO Water Back Up', 'Only if the rater column means this cover rather than sump overflow.'),
    'PersonalPropertyOtherResidenceInput.Indicator': ('missing', '', ''),
    'PersonalInjuryInput.Indicator': ('missing', '', ''),
    'UnitsRegularlyRentedToOthersInput.Indicator': ('sheet', 'AT HO6 Units Regularly Rented to Others', ''),
    'IncidentalFarmingPersonalLiabilityInput.Indicator': ('missing', '', ''),
    'RiskInput.UseDeductibleByPeril': ('default', '', 'Default No.'),
    'LineInput.CoveragePackage': ('missing', '', 'Rater has no package; default Standard may be wrong.'),
    'RiskInput.EquipmentBreakdown': ('sheet', 'AS Equipment Breakdown', ''),
    'RiskInput.PermittedIncidentalOccupancy': ('sheet', 'AI Permitted Incidental Occupancy', ''),
    'SpecialComputerCoverageInput.Indicator': ('missing', '', ''),
    'SpecialPersonalPropertyCoverageInput.Indicator': ('missing', '', 'Rater has “HO6 Special Coverage A” (AU), not Special Coverage C.'),
    # ---- Underwriting ----
    **{k: ('missing', '', 'Required eligibility answer; not in the rater.') for k in [
        'Underwriting.animals', 'Underwriting.trampoline', 'Underwriting.disrepair', 'Underwriting.renovation', 'Underwriting.business',
        'Underwriting.financialHistory', 'Underwriting.criminalHistory', 'Underwriting.leadOrLandfill', 'Underwriting.declined', 'Underwriting.sinkhole']},
    'Underwriting.Notes': ('missing', '', ''),
    **{k: ('missing', '', '') for k in ['Underwriting.SwimmingPool', 'Underwriting.HotTub', 'Underwriting.ForSale', 'Underwriting.Disrepair',
        'Underwriting.DayCare', 'Underwriting.Farming', 'Underwriting.Foreclosure', 'Underwriting.Bankruptcy', 'Underwriting.LeadPaint', 'Underwriting.Landfill']},
    'Underwriting.IncidentalOccupancy': ('derived', 'AI Permitted Incidental Occupancy', 'Same fact as the coverage flag.'),
    'Underwriting.RenovationCompletionDate': ('n/a', '', 'Only when under renovation.'),
    'Underwriting.RenovationValue': ('n/a', '', 'Only when under renovation.'),
    # ---- Insurance history ----
    'History.HasPrior': ('derived', 'G Risk Origin', 'Citizens take-out (FLCPIC) → Yes.'),
    'History.InsuredPast30Days': ('derived', 'G Risk Origin', 'Take-outs are insured until the Manatee effective date → Yes.'),
    'PriorInsurance.CarrierName': ('derived', 'G Risk Origin', 'Citizens.'),
    'PriorInsurance.EffectiveDate': ('derived', 'F Effective Date', 'Assumes a 1-year Citizens term ending at the Manatee effective date.'),
    'PriorInsurance.ExpirationDate': ('derived', 'F Effective Date', 'The Manatee effective date (take-out).'),
    'PriorInsurance.Limit': ('missing', '', ''),
    'PriorInsurance.Deductible': ('missing', '', ''),
    'PriorInsurance.PolicyNumber': ('partial', 'BV Citizens Policy Number', 'Empty in every test row.'),
    # ---- Claims ----
    'Claims.HasLosses': ('derived', 'AW Prior Claims', '0 prior claims → No (every row).'),
    **{k: ('n/a', '', 'No losses in the data.') for k in ['LossInput.LossType', 'LossInput.DateOfLoss', 'LossInput.AmountPaid', 'LossInput.Description', 'LossInput.Source']},
    # ---- Summary / interests ----
    'PolicyAdmin.UseDCTFormsAndMessages': ('default', '', 'Default Yes.'),
    **{k: ('n/a', '', 'Optional; no mortgagee data in the rater.') for k in ['type', 'name', 'address', 'rank', 'reference']},
    # ---- Payment details (after submission) ----
    'Billing.BilledTo': ('default', '', 'Customer is the only option.'),
    'Billing.PaymentPlan': ('missing', '', 'Rater has no billing data.'),
    'Billing.InvoiceDueDate': ('missing', '', 'Rater has no billing data.'),
}

schema = json.load(open(S))
rows = []
for p in schema['pages']:
    items = [(None, f) for f in p['fields']]
    if p.get('collection'):
        items += [(p['collection']['key'], f) for f in p['collection']['fields']]
    for coll, f in items:
        if f.get('readOnly') or f.get('unavailable') or f.get('hidden') or f.get('derive'):
            continue
        if f['key'] not in C:
            sys.exit('unclassified: ' + f['key'])
        cat, src, note = C[f['key']]
        req = 'required' if f['required'] and not f.get('showWhen') else ('conditional' if f['required'] or f.get('requiredWhen') else 'optional')
        rows.append(dict(page=p['title'], group=coll or '', key=f['key'], label=f['label'], need=req, category=cat, source=src, note=note))

json.dump(rows, open(sys.argv[1], 'w'), indent=1)
by = collections.Counter(r['category'] for r in rows)
print('all', len(rows), dict(by))
for need in ('required', 'conditional', 'optional'):
    print(need, sum(1 for r in rows if r['need'] == need), dict(collections.Counter(r['category'] for r in rows if r['need'] == need)))
print()
for page in dict.fromkeys(r['page'] for r in rows):
    pr = [r for r in rows if r['page'] == page]
    print(f"{page:24} {len(pr):3}", dict(collections.Counter(r['category'] for r in pr)))
print()
print('REQUIRED + missing:')
for r in rows:
    if r['need'] == 'required' and r['category'] in ('missing', 'partial'):
        print(f"  {r['page']:18} {r['label']:45} {r['category']}")
