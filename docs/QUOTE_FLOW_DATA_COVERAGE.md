# Quote flow: what the rater data can fill

Date: 2026-10-09. Compares every input of the PersonalHome quote flow (`internal/quote/schema.json`, 163 editable fields on 10 pages) with the 87 columns of the HO rater's Policy Data sheet (Manatee FL Select HO 12.1.25). Read-only, calculated and service-only fields are left out. Regenerate the counts with `python3 scripts/quote_data_coverage.py <out.json>`.

| Category | Meaning |
|---|---|
| In the sheet | The value is a sheet column, as is or through a value translation |
| Derived (assumption) | Worked out from sheet columns with a stated assumption |
| Partly in the sheet | The column exists but is empty or does not fit for some rows |
| Fixed / default | No data needed: a fixed product value or the quote's own default |
| Not in the data | Not in the rater data |
| Not applicable | In a section the data says does not apply (no losses, no pool, …) |

## Summary

- **Required inputs that apply: 54.** In the sheet 14 · Derived (assumption) 6 · Fixed / default 5 · Partly in the sheet 1 · Not in the data 28.
- **All inputs that apply: 145.** In the sheet 37 · Derived (assumption) 13 · Fixed / default 11 · Partly in the sheet 5 · Not in the data 79.
- Not applicable: 18.

## Field by field

### New quote

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Product | required | Fixed / default |  | Single product: Manatee FL Select PersonalHome. |
| Risk state | required | Fixed / default |  | FL for every row. |
| Policy form | required | In the sheet | B Form | HO3 / HO6 match the options. |
| Effective date | required | In the sheet | F Effective Date |  |
| Prior policy number | optional | Partly in the sheet | BV Citizens Policy Number | Column exists but is empty in every test row. |
| Prior carrier | optional | Derived (assumption) | G Risk Origin | FLCPIC → Citizens Property Insurance Corporation. |
| E-policy discount | optional | Not in the data |  |  |
| Writing company | optional | Not in the data |  | Could be a fixed company value once agreed. |
| Consent to rate | optional | In the sheet | AX Consent to Rate | 0/1 → No/Yes (all 0 in test data). |
| Consent-to-rate factor | optional | Not in the data |  | Rater has only the flag. |
| Citizens conversion legacy premium | optional | Partly in the sheet | AY FL CPIC Expiring Premium (Term 1) | Only term-1 rows (698 of 1,394) have it. |
| Citizens conversion capping details | optional | Derived (assumption) | AZ, BA, BB, BC, BD | Free-text summary built from the capping columns. |
| Risk origin | optional | In the sheet | G Risk Origin |  |
| Description | optional | Not in the data |  | Optional free text. |

### Applicant

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| First Name | required | Not in the data |  | Rater has no insured name. |
| Middle Initial (MI) | optional | Not in the data |  |  |
| Last Name | required | Not in the data |  | Rater has no insured name. |
| Phone Number | required | Not in the data |  |  |
| E-mail Address | required | Not in the data |  |  |
| Address 1 | required | Not in the data |  | No mailing address. |
| Address 2 | optional | Not in the data |  |  |
| City | required | Not in the data |  |  |
| State | required | Fixed / default |  | FL default. |
| Zip Code | required | Not in the data |  |  |
| County | optional | Derived (assumption) | C County | Only if the mailing county is the risk county (assumption). |
| Entity type | optional | Fixed / default |  | Individual. |
| Date of birth | optional | Not in the data |  |  |
| Producer | optional | Not in the data |  |  |
| Co-Applicant First Name (coapplicants) | required | Not applicable |  | Co-applicants: optional, none in the data. |
| Co-Applicant MI (coapplicants) | optional | Not applicable |  |  |
| Co-Applicant Last Name (coapplicants) | required | Not applicable |  |  |

### Risk schedule

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Address 1 | required | Not in the data |  | Risk address is not in the rater (only county). |
| Address 2 | optional | Not in the data |  |  |
| City | required | Not in the data |  |  |
| State | required | Fixed / default |  | FL. |
| ZIP Code | required | Not in the data |  |  |
| County | required | In the sheet | C County | Upper-case to match the county list. |
| Usage | required | Partly in the sheet | H Occupancy | Primary/Secondary map; Seasonal must be split into Seasonal3to6 / Seasonal6Mths (not in data). |
| Occupancy | required | In the sheet | I Occupied By | O → Owner, T → Tenant. |
| # of Months Unoccupied | optional | Not in the data |  | Optional. |
| Structure | required | Derived (assumption) | B Form | HO6 → Condo, HO3 → Dwelling (assumption: no townhouse/rowhouse distinction). |
| Year Built | required | In the sheet | J Year Built |  |
| Construction | required | In the sheet | K Construction | Masonry Veneer → MasonryVeneer; all values fit. |
| Protection Class | required | In the sheet | L Protection Class |  |
| # of Families | optional | Fixed / default |  | Default 1. |
| # of Units in Fire Division | required | In the sheet | O Units Within the Fire Division |  |
| Floor Unit (Floor #) | optional | Not in the data |  | The unit's floor is not in the rater (column M holds the building's floors for HO6). |
| Foundation Type | optional | Not in the data |  |  |
| Square Feet | required | Not in the data |  | Required; not in the rater. |
| Pool on Premises? | optional | Not in the data |  |  |
| Slide (Pool) | required | Not applicable |  | Only asked when there is a pool. |
| Is pool surrounded by fence? | required | Not applicable |  | Only asked when there is a pool. |
| Diving Board (Pool) | required | Not applicable |  | Only asked when there is a pool. |
| Number of Stories | optional | In the sheet | M Number of Stories | HO3 rows; for HO6 column M is the building's floors. |
| Primary Heat Type | optional | Not in the data |  |  |
| Supplemental Heat Type | optional | Not in the data |  |  |
| Building Code Grade (BCEG) | optional | Partly in the sheet | S BCEG | Whole numbers only: 1,002 rows say “Ungraded”. |
| Electrical System | optional | Not in the data |  |  |
| Opening Protection | optional | In the sheet | Z Opening Protection | Class A/Class B/None → ClassA/ClassB/None. |
| Roof Type | required | Not in the data |  | Required; the rater has roof covering compliance, not roof material. |
| Sprinkler System | optional | In the sheet | R Sprinkler System | Neither → None, Class B → ClassB. |
| Locked Security Gate | optional | Fixed / default |  | Default No. |
| Security Guard | optional | Fixed / default |  | Default No. |
| Wiring | optional | Not in the data |  |  |
| # of Amps | optional | Not in the data |  |  |
| Plumbing Pipe Material | optional | Not in the data |  |  |
| Roof Shape | required | In the sheet | W Roof Shape | Hip Roof → Hip. |
| Miles To Fire Station | required | Not in the data |  | Required; geocoding/protection data, not in the rater. |
| Feet To Hydrant | required | Not in the data |  | Required; not in the rater. |
| Responding Fire Department | required | Not in the data |  | Required; not in the rater. |
| Remaining Useful Life of Roof | optional | Not in the data |  |  |
| Year Electrical Updated | optional | Not in the data |  |  |
| Year Heating Updated | optional | Not in the data |  |  |
| Year Plumbing Updated | optional | Not in the data |  |  |
| Year Roof Updated | optional | Not in the data |  |  |
| Secondary Water Resistance | optional | In the sheet | Y Secondary Water Resistance | SWR → Yes, No SWR → No. |
| Accessible all year | optional | Not in the data |  |  |
| Roof to Wall Attachment | optional | In the sheet | U Roof Wall Connection | Values match the options. |
| Roof Cover | optional | In the sheet | T Roof Covering | All three rater values have an option (incl. ReinforcedConcreteRoofDeck). |
| Terrain | required | In the sheet | X Terrain Exposure | B / C. |
| Floor to Foundation Attachment | optional | Not in the data |  |  |
| Wind Speed Design | optional | In the sheet | AA Wind Speed Design | “Greater than or equal to 100” → 100, …120 → 120 mph+. |
| Wind Speed Location | optional | In the sheet | AB Wind Speed Location | …120 and WBDR → 120WBDR. |
| Roof Deck Attachment | optional | In the sheet | V Roof Deck Attachment | Level A/B/C, Reinforced Concrete, Wood/Metal deck all have an option. |
| Number Of Bathrooms | optional | Not in the data |  |  |

### Dwelling coverage

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Dwelling Limit (A) | conditional | In the sheet | AC Coverage A |  |
| Other Structures Limit (B) | optional | In the sheet | AD Coverage B |  |
| Personal Property Limit (C) – HO3 | optional | In the sheet | AE Coverage C | HO3 rows. |
| Personal Property Limit (C) – HO6 | conditional | In the sheet | AE Coverage C | HO6 rows. |
| Loss of Use Limit (D) | optional | Not in the data |  | Optional; the quote can calculate it from A. |
| All-Peril Deductible | required | In the sheet | AG All Peril Deductible | 500 / 1000 / 2500. |
| Hurricane Deductible | required | In the sheet | AH Hurricane Deductible | 0.02 → 2%, 0.05 → 5%, 0.1 → 10%, 500; Ex-Wind → N (4 rows). |
| Replacement Cost – Dwelling | optional | Not in the data |  | Defaults to Yes in the quote, which then requires a replacement cost value. |
| Replacement Cost Value | conditional | Not in the data |  | Required while the indicator is Yes (the default); no RCE value in the rater. |
| Sinkhole Indicator | optional | In the sheet | AP Sinkhole Coverage |  |
| Ordinance Or Law | optional | In the sheet | AK Ordinance or Law | 0.25 → 25, 0.5 → 50. |
| Water Back Up and Sump Overflow | optional | In the sheet | AO Water Back Up | Same rater column also fits the sewers/drains cover; one has to be chosen. |
| Limited Fungi or Bacteria | optional | In the sheet | AQ Limited Fungi Sect. 1 | $10,000 Included → 10/50 (every row). |
| Loss Assessment Coverage | optional | Derived (assumption) | AL Loss Assessment | “Increase to $2,000” → Yes; “$1,000 Included” → No. |
| Loss Assessment Limit | optional | In the sheet | AL Loss Assessment | 2000 when increased. |
| Carports, Pool Cages & Screen Enclosures | optional | In the sheet | AM Screened Enclosure | 0 in every row. |
| Replacement Cost – Contents | optional | In the sheet | AN Replacement Cost on Contents |  |
| Identity Fraud Expense | optional | Not in the data |  | Optional add-on. |
| Unscheduled Jewelry Indicator | optional | Not in the data |  |  |
| Unscheduled Jewelry Limit | optional | Not in the data |  |  |
| Water Backup of Sewers or Drains | optional | Derived (assumption) | AO Water Back Up | Only if the rater column means this cover rather than sump overflow. |
| Personal Property Other Residence | optional | Not in the data |  |  |
| Personal Injury | optional | Not in the data |  |  |
| Unit-Owners Rental to Others | optional | In the sheet | AT HO6 Units Regularly Rented to Others |  |
| Incidental Farming Personal Liability | optional | Not in the data |  |  |
| Choose Deductibles by Peril | optional | Fixed / default |  | Default No. |
| Coverage Tier / Package | optional | Not in the data |  | Rater has no package; default Standard may be wrong. |
| Equipment Breakdown | optional | In the sheet | AS Equipment Breakdown |  |
| Permitted Incidental Occupancy | optional | In the sheet | AI Permitted Incidental Occupancy |  |
| Special Computer Coverage | optional | Not in the data |  |  |
| Unit-Owners Special Coverage C | optional | Not in the data |  | Rater has “HO6 Special Coverage A” (AU), not Special Coverage C. |

### Underwriting

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Restricted dogs, exotic pets or guard dogs? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Trampoline on the premises? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Property for sale or in disrepair? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Property under renovation? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Business, day care or farming use? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Foreclosure or bankruptcy history? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Criminal indictment history? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Known lead paint or landfill exposure? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Prior insurance declined or cancelled? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Known sinkhole activity? | required | Not in the data |  | Required eligibility answer; not in the rater. |
| Details for any Yes answers | optional | Not in the data |  |  |
| Swimming pool on premises? | optional | Not in the data |  |  |
| Hot tub on premises? | optional | Not in the data |  |  |
| Property for sale? | optional | Not in the data |  |  |
| Property in disrepair? | optional | Not in the data |  |  |
| Day care use? | optional | Not in the data |  |  |
| Farming use? | optional | Not in the data |  |  |
| Incidental occupancy? | optional | Derived (assumption) | AI Permitted Incidental Occupancy | Same fact as the coverage flag. |
| Foreclosure history? | optional | Not in the data |  |  |
| Bankruptcy history? | optional | Not in the data |  |  |
| Known lead paint? | optional | Not in the data |  |  |
| Landfill exposure? | optional | Not in the data |  |  |
| Renovation completion date | optional | Not applicable |  | Only when under renovation. |
| Renovation dollar value | optional | Not applicable |  | Only when under renovation. |

### Insurance history

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Prior insurance to disclose? | required | Derived (assumption) | G Risk Origin | Citizens take-out (FLCPIC) → Yes. |
| Insurance within the past 30 days? | optional | Derived (assumption) | G Risk Origin | Take-outs are insured until the Manatee effective date → Yes. |
| Carrier Name (priorInsurance) | required | Derived (assumption) | G Risk Origin | Citizens. |
| Effective Date (priorInsurance) | required | Derived (assumption) | F Effective Date | Assumes a 1-year Citizens term ending at the Manatee effective date. |
| Expiration Date (priorInsurance) | required | Derived (assumption) | F Effective Date | The Manatee effective date (take-out). |
| Limit (priorInsurance) | optional | Not in the data |  |  |
| Deductible (priorInsurance) | optional | Not in the data |  |  |
| Prior policy number (priorInsurance) | optional | Partly in the sheet | BV Citizens Policy Number | Empty in every test row. |

### Claims history

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Prior losses to disclose? | required | Derived (assumption) | AW Prior Claims | 0 prior claims → No (every row). |
| Loss Type (losses) | required | Not applicable |  | No losses in the data. |
| Date Of Loss (losses) | required | Not applicable |  | No losses in the data. |
| Amount Paid (losses) | required | Not applicable |  | No losses in the data. |
| Description (losses) | required | Not applicable |  | No losses in the data. |
| Loss source (losses) | optional | Not applicable |  | No losses in the data. |

### Coverage summary

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Include Forms | optional | Fixed / default |  | Default Yes. |

### Additional interests

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| Interest type (interests) | required | Not applicable |  | Optional; no mortgagee data in the rater. |
| Name (interests) | required | Not applicable |  | Optional; no mortgagee data in the rater. |
| Mailing address (interests) | required | Not applicable |  | Optional; no mortgagee data in the rater. |
| Mortgage rank (interests) | optional | Not applicable |  | Optional; no mortgagee data in the rater. |
| Loan / reference number (interests) | optional | Not applicable |  | Optional; no mortgagee data in the rater. |

### Payment details

Filled in after the application is submitted for rating.

| Field | Need | Category | Rater column | Note |
|---|---|---|---|---|
| How will this policy be billed? | required | Fixed / default |  | Customer is the only option. |
| Payment Plan | required | Not in the data |  | Rater has no billing data. |
| What day of the month will the invoice be due? | required | Not in the data |  | Rater has no billing data. |
