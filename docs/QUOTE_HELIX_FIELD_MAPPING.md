# PersonalHome quote form → Helix field mapping

Generated 2026-10-13 from the mapping code: `internal/quotesync/mapping.go` (`links`), `internal/quotesync/supplement.go` (`supplements`, `lists`, `remarkAnswers`, quote-only templates), the reviewed rule set `internal/rules/rulesets/manatee_fl_select_ho_12_1_25.json` with its templates, and `data/rules` (Rules-tab edits: none saved). Regenerate it whenever those files change.

## How a value reaches Helix

Each save of the quote is synced to Helix in the background (`quotesync.Syncer`). The quote is turned into one rater-shaped row, then planned and written like a migrated spreadsheet row:

1. **Rater column fields.** The value is translated to the rater's spelling and placed in a rater column (`links`). That column's reviewed rule, or a template field that reads the column, gives the Helix target.
2. **Quote-only fields.** Fields the rater never had go straight to a Helix field through a best-effort rule (`supplements`). These targets are proposals, not reviewed mappings.
3. **Lists.** Losses, additional interests and co-applicants become one Helix record per entry (`lists`).

How to read the tables:

- **Helix target** is `variant#instance.field`. The instance tells apart several records of one variant on a policy, such as one coverage instance per coverage.
- **Value conversion** shows UI value → rater cell → Helix value. When a step doesn't change the value it isn't repeated. "as entered" means the value is copied and coerced to the Helix field's type.
- **Not written** means the field is kept only in the local quote (`data/quotes/<id>.json`).
- Read-only and service-only fields on the form are left out. They are never written.

General limits:

- Writes only add or patch values, so clearing a field doesn't clear it in Helix. Removing a list entry leaves its record in Helix.
- Shared records, such as the issuer, product and product coverages, are reused and never updated by a quote.
- A quote-only rule whose target Helix doesn't have is skipped and reported as a sync warning. The last sync of Q-4BF1613705 had no warnings.

## Values the quote supplies without a form field

| Source | Rater column / rule | Helix target |
|---|---|---|
| Policy number: the imported policy's number, or the quote number for a new quote | A → Rule HO-01 (High) | `policy.property.us-fl.personal.safepoint.policy_number` |
| Term number: 1 for a new quote | D → Rule HO-04 (Medium/Low) | `policy_term.term_number` |
| Quote number (for example Q-4BF1613705) | quote-only rule | `policy.property.us-fl.personal.safepoint.quote_number` |

Templates also write fixed values, references and generated keys, such as statuses, coverage codes and `PH-<policy number>` party references. They aren't listed here: see the rule set's templates file and `supplementTemplates()` in `supplement.go`.

**Count:** 123 editable form fields reach Helix and 42 don't. List fields are counted once per column.

## New quote (`newquote`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Product | `Quote.Product` | **Not written** | | |  |
| Risk state | `Quote.State` | **Not written** | | |  |
| Policy form | `DwellingInput.Form` | `policy.property.us-fl.personal.safepoint.personal_policy_form` | Rater column B → Template field | HO3 → ho_3; HO6 → ho_6 |  |
| Effective date | `PolicyInput.EffectiveDate` | `policy.property.us-fl.personal.safepoint.effective_date` | Rater column F → Rule HO-06 (High) | as entered |  |
| ↳ |  | `policy.property.us-fl.personal.safepoint.issue_date` | Rater column F → Template field | as entered | Issue date is not in the sheet; the Effective Date is used (D6, S11). |
| ↳ |  | `location.property.us.personal.effective_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `wind_mitigation_verification.property.us-fl.personal.safepoint.inspection_date` | Rater column F → Template field | as entered | Inspection date is not in the sheet; the Effective Date is used (D9, S13). |
| ↳ |  | `policy_term.effective_end_date` | Rater column F → Template field + 1 year | as entered | The term end is not in the sheet; Effective Date + 1 year is used. |
| ↳ |  | `policy_term.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `contract_wording.effective_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `policy_version.effective_end_date` | Rater column F → Template field + 1 year | as entered |  |
| ↳ |  | `policy_version.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `dwelling_feature.property.personal#units_in_fire_division.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `dwelling_feature.property.personal#burglar_alarm.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `dwelling_feature.property.personal#fire_alarm.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `dwelling_feature.property.personal#sprinkler_system.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `dwelling_feature.property.personal#terrain_exposure.effective_start_date` | Rater column F → Template field | as entered |  |
| ↳ |  | `premium_transaction#surplus_contribution.effective_from` | Rater column F → Template field | as entered |  |
| ↳ |  | `premium_transaction#surplus_contribution.effective_to` | Rater column F → Template field + 1 year | as entered |  |
| ↳ |  | `premium_transaction#figa.effective_from` | Rater column F → Template field | as entered |  |
| ↳ |  | `premium_transaction#figa.effective_to` | Rater column F → Template field + 1 year | as entered |  |
| ↳ |  | `effective_date` of 23 shared records: product, coverage.property.us.personal#coverage_a, coverage.property.us.personal#coverage_b, coverage.property.us.personal#coverage_c, coverage.property.us.personal#coverage_e, coverage.property.us.personal#business_use, coverage.property.us.personal#other_structures_rented, coverage.property.us.personal#ordinance_or_law, coverage.property.us.personal#loss_assessment, coverage.property.us.personal#screened_enclosure, coverage.property.us.personal#replacement_cost_contents, coverage.property.us.personal#water_back_up, coverage.property.us.personal#limited_fungi_section_1, coverage.property.us.personal#limited_fungi_section_2, coverage.property.us.personal#ho6_units_rented, coverage.property.us.personal#ho6_special_coverage_a, coverage.property.us.personal#other_structures_increased, product_version, coverage.property.us.personal#coverage_d, coverage.property.us.personal#coverage_f, coverage.property.us.personal#identity_fraud, coverage.property.us.personal#unscheduled_jewelry, coverage.property.us.personal#personal_injury | Rater column F → Template fields (earliest value; shared records are written once and never updated) | as entered |  |
| Prior policy number | `PolicyDetails.PriorPolicyNumber` | `policy.property.us-fl.personal.safepoint.previous_policy_number` | Quote-only rule (best effort) | as entered | Rater column BV rule HO-74 → `residual_market_placement.property.us.personal.external_policy_number` is disabled: Empty in every test row, and a residual market placement needs a property asset record the migrator does not write. |
| Prior carrier | `PolicyDetails.PriorCarrier` | **Not written** | | |  |
| E-policy discount | `PolicyDetails.EPolicyDiscount` | **Not written** | | |  |
| Writing company | `PolicyDetails.WritingCompany` | **Not written** | | |  |
| Consent to rate | `PolicyDetails.ConsentToRate` | `line.property.us.personal.consent_to_rate` | Rater column AX → Rule HO-50 (High) | No → 0 → false; Yes → 1 → true | Read as a yes/no flag (0/1), but the UI shows Consent to Rate as an amount and a factor (line.ctrfactor). Every test row is 0; any other value blocks the row. |
| Consent-to-rate factor | `PolicyDetails.CTRFactor` | **Not written** | | |  |
| Citizens conversion legacy premium | `PolicyDetails.CitizensLegacyPremium` | **Not written** | | |  |
| Citizens conversion capping details | `PolicyDetails.CitizensCapping` | **Not written** | | |  |
| Risk origin | `PolicyDetails.RiskOrigin` | **Not written** | | | Rater column G rule HO-07 → `residual_market_placement.property.us.personal.placement_basis_code` is disabled: A residual market placement needs a property asset record the migrator does not write, and every row is FLCPIC. Open question: are there origins other than Citizens? |
| Description | `PolicyInput.Description` | **Not written** | | |  |

## Applicant (`account`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| First Name | `AccountInput.FirstName` | `person.given_name` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `person.recorded_name` | Quote-only rule, part of "First Middle Last" | joined with spaces |  |
| ↳ |  | `person.display_name` | Quote-only rule, part of "First Middle Last" | joined with spaces |  |
| Middle Initial (MI) | `AccountInput.MiddleName` | `person.recorded_name` | Quote-only rule, part of "First Middle Last" | joined with spaces |  |
| ↳ |  | `person.display_name` | Quote-only rule, part of "First Middle Last" | joined with spaces |  |
| Last Name | `AccountInput.LastName` | `person.family_name` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `person.recorded_name` | Quote-only rule, part of "First Middle Last" | joined with spaces |  |
| ↳ |  | `person.display_name` | Quote-only rule, part of "First Middle Last" | joined with spaces |  |
| Phone Number | `AccountInput.PrimaryPhone` | `contact_point#phone.channel_value` | Quote-only rule (best effort) | as entered |  |
| E-mail Address | `AccountInput.Email` | `contact_point#email.channel_value` | Quote-only rule (best effort) | as entered |  |
| Address 1 | `AccountInput.Address1` | `mailing_address.property.us.personal.address1` | Quote-only rule (best effort) | as entered |  |
| Address 2 | `AccountInput.Address2` | `mailing_address.property.us.personal.address2` | Quote-only rule (best effort) | as entered |  |
| City | `AccountInput.City` | `mailing_address.property.us.personal.city` | Quote-only rule (best effort) | as entered |  |
| State | `AccountInput.State` | `mailing_address.property.us.personal.state` | Quote-only rule (best effort) | as entered |  |
| Zip Code | `AccountInput.ZipCode` | `mailing_address.property.us.personal.zip` | Quote-only rule (best effort) | as entered |  |
| County | `AccountInput.County` | **Not written** | | |  |
| Entity type | `Applicant.EntityType` | **Not written** | | |  |
| Date of birth | `Applicant.DateOfBirth` | `person.date_of_birth` | Quote-only rule (best effort) | as entered |  |
| Producer | `Applicant.Producer` | **Not written** | | |  |

### Co-applicants (list `coapplicants`)

Each entry *n* becomes one `person` record, instance `coapplicant_n`.
The person record also gets `party_reference` = `PH-<policy number>`, `record_status` = active and `vital_status` = living.

| UI field | Key | Helix target | Notes |
|---|---|---|---|
| Co-Applicant First Name | `PersonInput.FirstName` | `person#coapplicant_n.given_name` | Also part of `recorded_name` ("First Middle Last"). |
| Co-Applicant MI | `PersonInput.MiddleName` | **Not written** | Also part of `recorded_name` ("First Middle Last"). |
| Co-Applicant Last Name | `PersonInput.LastName` | `person#coapplicant_n.family_name` | Also part of `recorded_name` ("First Middle Last"). |

## Risk schedule (`dwellinginfo`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Address 1 | `LocationInput.Address1` | `location_address.property.us.personal.address1` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `location.property.us.personal.address_text` | Quote-only rule, part of the risk address text | "Address1, Address2, City, State Zip" |  |
| Address 2 | `LocationInput.Address2` | `location_address.property.us.personal.address2` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `location.property.us.personal.address_text` | Quote-only rule, part of the risk address text | "Address1, Address2, City, State Zip" |  |
| City | `LocationInput.City` | `location_address.property.us.personal.city` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `location.property.us.personal.address_text` | Quote-only rule, part of the risk address text | "Address1, Address2, City, State Zip" |  |
| State | `LocationInput.State` | `location_address.property.us.personal.state` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `location.property.us.personal.address_text` | Quote-only rule, part of the risk address text | "Address1, Address2, City, State Zip" |  |
| ZIP Code | `LocationInput.ZipCode` | `location_address.property.us.personal.zip_code` | Quote-only rule (best effort) | as entered |  |
| ↳ |  | `location.property.us.personal.address_text` | Quote-only rule, part of the risk address text | "Address1, Address2, City, State Zip" |  |
| County | `LocationInput.County` | `location_address.property.us.personal.county` | Rater column C → Rule HO-03 (High) | as entered | upper-cased; upper-cased |
| ↳ |  | `location.property.us.personal.address_text` | Rater column C → Template format `{C}, FL` | as entered | upper-cased; Only the county is known; the address is "<County>, FL" (D6, S11). |
| Usage | `DwellingInput.UseType` | `policy.property.us-fl.personal.safepoint.residence_occupancy` | Rater column H → Rule HO-08 (Medium/Low) | Primary → owner_occupied; Seasonal3to6 → Seasonal → seasonal; Seasonal6Mths → Seasonal → seasonal; Secondary → secondary | only when Occupancy = Owner |
| ↳ |  | `policy.property.us-fl.personal.safepoint.residence_occupancy` | Rater column H → Rule HO-08b (Medium/Low) | Primary → tenant_occupied; Seasonal3to6 → Seasonal → seasonal; Seasonal6Mths → Seasonal → seasonal; Secondary → secondary | only when Occupancy = Tenant; Seasonal or secondary homes occupied by a tenant are written as seasonal / secondary: Helix has one occupancy value, so the tenant is not recorded for them (2 rows in the HO test file). |
| ↳ |  | `dwelling.property.us.personal.use_type` | Quote-only rule (best effort) | as entered |  |
| Occupancy | `DwellingInput.OccupancyType` | `dwelling.property.us.personal.occupancy_type` | Quote-only rule (best effort) | as entered | Rater column I has no rule of its own: Read as the condition of HO-08 and HO-08b (Occupancy): owner (O) or tenant (T) decides the occupancy written. |
| # of Months Unoccupied | `DwellingInput.MonthsUnoccupied` | `dwelling.property.us.personal.months_unoccupied` | Quote-only rule (best effort) | as entered |  |
| Structure | `DwellingInput.BuildingType` | `dwelling.property.us.personal.building_type` | Quote-only rule (best effort) | as entered |  |
| Year Built | `DwellingInput.YearBuilt` | `dwelling_asset.property.personal.construction_year` | Rater column J → Rule HO-10 (High) | as entered |  |
| ↳ |  | `dwelling.property.us.personal.year_built` | Quote-only rule (best effort) | as entered |  |
| Construction | `DwellingInput.Construction` | `dwelling_asset.property.personal.construction_type` | Rater column K → Rule HO-11 (High) | Aluminum; Frame; Masonry; MasonryVeneer → Masonry Veneer; PlasticSiding → Plastic Siding; Superior |  |
| ↳ |  | `dwelling.property.us.personal.construction` | Quote-only rule (best effort) | as entered |  |
| Protection Class | `DwellingInput.RatedProtectionClass` | `dwelling_asset.property.personal.protection_class` | Rater column L → Rule HO-12 (High) | as entered |  |
| ↳ |  | `dwelling.property.us.personal.rated_protection_class` | Quote-only rule (best effort) | as entered |  |
| # of Families | `DwellingInput.NumberOfFamilies` | `dwelling.property.us.personal.number_of_families` | Quote-only rule (best effort) | as entered |  |
| # of Units in Fire Division | `DwellingInput.NumberOfUnits` | `dwelling_feature.property.personal#units_in_fire_division.value` | Rater column O → Rule HO-15 (Medium/Low) | as entered |  |
| Floor Unit (Floor #) | `DwellingInput.NumberOfFloor` | **Not written** | | |  |
| Foundation Type | `DwellingInput.FoundationType` | `dwelling.property.us.personal.foundation_type` | Quote-only rule (best effort) | as entered |  |
| Square Feet | `DwellingInput.SquareFeet` | `dwelling.property.us.personal.square_feet` | Quote-only rule (best effort) | as entered |  |
| Pool on Premises? | `DwellingInput.SwimmingPool` | **Not written** | | |  |
| Slide (Pool) | `DwellingInput.PoolSlide` | `dwelling.property.us.personal.pool_slide` | Quote-only rule (best effort) | as entered |  |
| Is pool surrounded by fence? | `DwellingInput.PoolApprovedFence` | `dwelling.property.us.personal.pool_approved_fence` | Quote-only rule (best effort) | as entered |  |
| Diving Board (Pool) | `DwellingInput.PoolDivingBoard` | `dwelling.property.us.personal.pool_diving_board` | Quote-only rule (best effort) | as entered |  |
| Number of Stories | `DwellingInput.NumberOfStories` | `dwelling.property.us.personal.number_of_stories` | Rater column M → Rule HO-13 (High) | as entered | only when Policy form = HO3 |
| ↳ |  | `dwelling.property.us.personal.number_of_floor` | Rater column M → Rule HO-13b (High) | as entered | only when Policy form = HO6 |
| Primary Heat Type | `DwellingInput.PrimaryHeatType` | `dwelling.property.us.personal.primary_heat_type` | Quote-only rule (best effort) | as entered |  |
| Supplemental Heat Type | `DwellingInput.SupplementalHeatType` | `dwelling.property.us.personal.supplemental_heat_type` | Quote-only rule (best effort) | as entered |  |
| Building Code Grade (BCEG) | `DwellingInput.BuildingCodeEffectivenessGrading` | `dwelling.property.us.personal.building_code_effectiveness_grading` | Rater column S → Rule HO-19 (High) | as entered |  |
| Electrical System | `DwellingInput.ElectricalSystem` | `dwelling.property.us.personal.electrical_system` | Quote-only rule (best effort) | as entered |  |
| Opening Protection | `DwellingInput.OpeningProtection` | `wind_mitigation_verification.property.us-fl.personal.safepoint.opening_protection` | Rater column Z → Rule HO-26 (High) | ClassA → Class A → hurricane_rated; ClassB → Class B → basic; None → none; Unknown → unknown | Opening Protection classes are mapped Class A → hurricane_rated, Class B → basic (UI lists Basic / Hurricane Rated / None). Confirm the class meaning with underwriting. |
| ↳ |  | `dwelling.property.us.personal.opening_protection` | Quote-only rule (best effort) | as entered |  |
| Roof Type | `DwellingInput.RoofType` | `dwelling.property.us.personal.roof_type` | Quote-only rule (best effort) | as entered |  |
| Sprinkler System | `DwellingInput.Sprinkler` | `dwelling_feature.property.personal#sprinkler_system.value` | Rater column R → Rule HO-18 (Medium/Low) | ClassA → Class A; ClassB → Class B; None → Neither → none; Shutoff; SinglePoint; WholeHome |  |
| ↳ |  | `dwelling.property.us.personal.sprinkler` | Quote-only rule (best effort) | as entered |  |
| Locked Security Gate | `DwellingInput.LockedSecurityGate` | `dwelling.property.us.personal.locked_security_gate` | Quote-only rule (best effort) | Yes / No |  |
| Security Guard | `DwellingInput.SecurityAttendant` | `dwelling.property.us.personal.security_attendant` | Quote-only rule (best effort) | Yes / No |  |
| Wiring | `DwellingInput.Wiring` | `dwelling.property.us.personal.wiring` | Quote-only rule (best effort) | as entered |  |
| # of Amps | `DwellingInput.NumberOfAmps` | `dwelling.property.us.personal.amps` | Quote-only rule (best effort) | as entered |  |
| Plumbing Pipe Material | `DwellingInput.PlumbingType` | `dwelling.property.us.personal.plumbing_type` | Quote-only rule (best effort) | as entered |  |
| Roof Shape | `DwellingInput.RoofGeometry` | `wind_mitigation_verification.property.us-fl.personal.safepoint.roof_shape` | Rater column W → Rule HO-23 (High) | Flat → flat; Gable → gable; Hip → hip; Other → other | Writing Roof Shape also creates a location, a dwelling asset and a wind mitigation verification with placeholder inspection fields (D4, S7). |
| ↳ |  | `dwelling.property.us.personal.roof_geometry` | Quote-only rule (best effort) | as entered |  |
| Miles To Fire Station | `DwellingInput.DistanceToFireStation` | `dwelling.property.us.personal.distance_to_fire_station` | Quote-only rule (best effort) | as entered |  |
| Feet To Hydrant | `DwellingInput.DistanceToHydrant` | `dwelling.property.us.personal.distance_to_hydrant` | Quote-only rule (best effort) | as entered |  |
| Responding Fire Department | `DwellingInput.RespondingFireDepartment` | `dwelling.property.us.personal.responding_fire_department` | Quote-only rule (best effort) | as entered |  |
| Remaining Useful Life of Roof | `UpdatedServicesInfo.RemainingRoofLife` | `dwelling.property.us.personal.remaining_roof_life` | Quote-only rule (best effort) | as entered |  |
| Year Electrical Updated | `UpdatedServicesInfo.ElectricSystemUpdateYear` | `dwelling.property.us.personal.electric_system_update_year` | Quote-only rule (best effort) | as entered |  |
| Year Heating Updated | `UpdatedServicesInfo.HeatingUpdateYear` | `dwelling.property.us.personal.heating_update_year` | Quote-only rule (best effort) | as entered |  |
| Year Plumbing Updated | `UpdatedServicesInfo.PlumbingUpdateYear` | `dwelling.property.us.personal.plumbing_update_year` | Quote-only rule (best effort) | as entered |  |
| Year Roof Updated | `UpdatedServicesInfo.RoofUpdateYear` | `dwelling.property.us.personal.roof_update_year` | Quote-only rule (best effort) | as entered |  |
| Secondary Water Resistance | `DwellingInput.SecondaryWaterResistance` | `wind_mitigation_verification.property.us-fl.personal.safepoint.secondary_water_resistance_flag` | Rater column Y → Rule HO-25 (High) | No → No SWR → false; Yes → SWR → true |  |
| ↳ |  | `dwelling.property.us.personal.secondary_water_resistance` | Quote-only rule (best effort) | as entered |  |
| Accessible all year | `DwellingInput.DwellingAccessibleToFireEquipment` | `dwelling.property.us.personal.dwelling_accessible_to_fire_equipment` | Quote-only rule (best effort) | as entered |  |
| Roof to Wall Attachment | `DwellingInput.RoofToWallAttachment` | `wind_mitigation_verification.property.us-fl.personal.safepoint.roof_to_wall_connection` | Rater column U → Rule HO-21 (Medium/Low) | Clips → clips; DoubleWraps → double_wraps; SingleWraps → single_wraps; ToeNails → toe_nails; Unknown → unknown |  |
| ↳ |  | `dwelling.property.us.personal.roof_to_wall_attachment` | Quote-only rule (best effort) | as entered |  |
| Roof Cover | `DwellingInput.RoofCover` | `wind_mitigation_verification.property.us-fl.personal.safepoint.roof_covering_code_compliance` | Rater column T → Rule HO-20 (Medium/Low) | FBCEquivalent → FBC Equivalent → meets_2001_fbc; NonFBCEquivalent → Non-FBC Equivalent → pre_2001_fbc; ReinforcedConcreteRoofDeck → Reinforced Concrete Roof Deck → (not written) | “Reinforced Concrete Roof Deck” has no equivalent in Helix (meets_2001_fbc, pre_2001_fbc, unknown); the field is left empty for those rows, with a warning. |
| ↳ |  | `dwelling.property.us.personal.roof_cover` | Quote-only rule (best effort) | as entered |  |
| Terrain | `DwellingInput.Terrain` | `dwelling_feature.property.personal#terrain_exposure.value` | Rater column X → Rule HO-24 (Medium/Low) | B; C; H |  |
| ↳ |  | `dwelling.property.us.personal.terrain` | Quote-only rule (best effort) | as entered |  |
| Floor to Foundation Attachment | `DwellingInput.FloorToFoundationAttachment` | `dwelling.property.us.personal.floor_to_foundation_attachment` | Quote-only rule (best effort) | as entered |  |
| Wind Speed Design | `DwellingInput.WindSpeedDesign` | `dwelling.property.us.personal.wind_speed_design` | Rater column AA → Rule HO-27 (High) | 100 → Greater than or equal to 100; 110 → Greater than or equal to 110; 120 mph+ → Greater than or equal to 120 |  |
| Wind Speed Location | `DwellingInput.WindSpeedLocation` | `dwelling.property.us.personal.wind_speed_location` | Rater column AB → Rule HO-28 (High) | 100; 110; 120+ → 120; 120WBDR → 120 and WBDR |  |
| Roof Deck Attachment | `DwellingInput.RoofDeckAttachment` | `dwelling.property.us.personal.roof_deck_attachment` | Rater column V → Rule HO-22 (High) | A6d@612 → Level A: 6d @ 6/12; B8d@612 → Level B: 8d @ 6/12; C8d@66 → Level C: 8d @ 6/6; MetalDeck → Metal Deck; Reinforce → Reinforced Concrete Roof Deck; WoodDeck → Wood Deck |  |
| Number Of Bathrooms | `DwellingInput.NumberOfBathrooms` | **Not written** | | |  |

## Dwelling coverage (`dwellingcoverage`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Dwelling Limit (A) | `CoverageADwellingInput.Limit` | `coverage_instance.property.us.personal#coverage_a.limit_amount` | Rater column AC → Rule HO-29 (Medium/Low) | as entered |  |
| Other Structures Limit (B) | `CoverageBOtherStructuresInput.Limit` | `coverage_instance.property.us.personal#coverage_b.limit_amount` | Rater column AD → Rule HO-30 (Medium/Low) | as entered | only when Other Structures Limit (B) =  |
| Personal Property Limit (C) – HO3 | `CoverageCPersonalPropertyHO3Input.Limit` | `coverage_instance.property.us.personal#coverage_c.limit_amount` | Rater column AE → Rule HO-31 (Medium/Low) | as entered | HO3 only; only when Personal Property Limit (C) – HO3 =  |
| Personal Property Limit (C) – HO6 | `CoverageCPersonalPropertyHO46Input.Limit` | `coverage_instance.property.us.personal#coverage_c.limit_amount` | Rater column AE → Rule HO-31 (Medium/Low) | as entered | HO6 only; only when Personal Property Limit (C) – HO3 =  |
| Loss of Use Limit (D) | `CoverageDLossOfUseInput.Limit` | `coverage_instance.property.us.personal#coverage_d.limit_amount` | Quote-only rule (best effort) | as entered |  |
| All-Peril Deductible | `DwellingInput.Deductible` | `coverage_deductible.property.us-fl.personal.safepoint#all_other_perils.deductible_amount` | Rater column AG → Rule HO-33 (Medium/Low) | as entered |  |
| Hurricane Deductible | `DwellingInput.HurricaneDeductible` | `coverage_deductible.property.us-fl.personal.safepoint#hurricane.florida_hurricane_deductible_option` | Rater column AH → Rule HO-34 (Medium/Low) | 10% → 0.1 → percent_10; 2% → 0.02 → percent_2; 5% → 0.05 → percent_5; 500 → amount_500; N → Ex-Wind → (not written) | “Ex-Wind” (wind excluded) has no hurricane deductible option, so no hurricane deductible is written for those rows; 0.03 and 0.04 would also have no option. |
| ↳ |  | `coverage_deductible.property.us-fl.personal.safepoint#hurricane.deductible_basis` | Rater column AH → Rule HO-34b (Medium/Low) | 10% → 0.1 → percentage_of_dwelling_limit; 2% → 0.02 → percentage_of_dwelling_limit; 5% → 0.05 → percentage_of_dwelling_limit; 500 → flat_amount; N → Ex-Wind → (not written) |  |
| ↳ |  | `coverage_deductible.property.us-fl.personal.safepoint#hurricane.deductible_percentage` | Rater column AH → Rule HO-34c (Medium/Low) | 10% → 0.1 → 10; 2% → 0.02 → 2; 5% → 0.05 → 5; 500 → (not written); N → Ex-Wind → (not written) | The hurricane deductible percentage is written as a percent (0.02 → 2). Confirm Helix expects percent, not a fraction. |
| Replacement Cost – Dwelling | `ReplacementCostDwellingInput.Indicator` | `policy.property.us-fl.personal.safepoint.property_valuation_basis` | Quote-only rule (best effort) | Yes → replacement_cost; No → actual_cash_value |  |
| Replacement Cost Value | `ReplacementCostDwellingInput.ReplacementCostValue` | **Not written** | | |  |
| Sinkhole Indicator | `SinkholeInput.Indicator` | `policy.property.us-fl.personal.safepoint.sinkhole_coverage_option` | Rater column AP → Template field | No → catastrophic_ground_cover_collapse_only; Yes → sinkhole_loss |  |
| Ordinance Or Law | `OrdinanceOrLawInput.Limit` | `coverage_instance.property.us.personal#ordinance_or_law.percentage_of_dwelling_limit` | Rater column AK → Rule HO-37 (High (for name-based absence)) | 25 → 0.25 → 25; 50 → 0.5 → 50 | Ordinance or Law is a share of Coverage A; 0.25 is written as 25 (percent). Confirm Helix expects percent, not a fraction. |
| Water Back Up and Sump Overflow | `WaterBackupAndSumpOverflowInput.Indicator` | `coverage_instance.property.us.personal#water_back_up.coverage_status` | Rater column AO → Rule HO-41 (Medium/Low) | No; Yes → active | only when Water Back Up and Sump Overflow = Yes |
| Limited Fungi or Bacteria | `LimitedFungiBacteriaInput.Limit` | `coverage_instance.property.us.personal#limited_fungi_section_1.limit_amount` | Rater column AQ → Rule HO-43 (High (for name-based absence)) | 10/50 → $10,000 Included → 10000; 25/50 → 25000; 50/50 → 50000 |  |
| Loss Assessment Coverage | `LossAssessmentInput.Indicator` | `coverage_instance.property.us.personal#loss_assessment.limit_amount` | Rater column AL → Rule HO-38 (High (for name-based absence)) | No → $1,000 Included → 1000; Yes → Increase to $2,000 → 2000 |  |
| Loss Assessment Limit | `LossAssessmentInput.Limit` | **Not written** | | |  |
| Carports, Pool Cages & Screen Enclosures | `CarportsPoolCagesInput.Limit` | `coverage_instance.property.us.personal#screened_enclosure.limit_amount` | Rater column AM → Rule HO-39 (High (for name-based absence)) | as entered | only when Carports, Pool Cages & Screen Enclosures =  |
| Replacement Cost – Contents | `ReplacementCostContentsInput.Indicator` | `coverage_instance.property.us.personal#replacement_cost_contents.coverage_status` | Rater column AN → Rule HO-40 (Medium/Low) | No; Yes → active | only when Replacement Cost – Contents = Yes |
| Identity Fraud Expense | `IdentityFraudExpenseCoverageInput.Indicator` | `coverage_instance.property.us.personal#identity_fraud.coverage_status` | Quote-only rule (best effort) | Yes → active | only when Identity Fraud Expense = Yes |
| Unscheduled Jewelry Indicator | `UnscheduledJewelryInput.Indicator` | `coverage_instance.property.us.personal#unscheduled_jewelry.coverage_status` | Quote-only rule (best effort) | Yes → active | only when Unscheduled Jewelry Indicator = Yes |
| Unscheduled Jewelry Limit | `UnscheduledJewelryInput.Limit` | `coverage_instance.property.us.personal#unscheduled_jewelry.limit_amount` | Quote-only rule (best effort) | as entered | only when Unscheduled Jewelry Indicator = Yes |
| Water Backup of Sewers or Drains | `WaterBackupOfSewersOrDrainsInput.Indicator` | **Not written** | | |  |
| Personal Property Other Residence | `PersonalPropertyOtherResidenceInput.Indicator` | **Not written** | | |  |
| Personal Injury | `PersonalInjuryInput.Indicator` | `coverage_instance.property.us.personal#personal_injury.coverage_status` | Quote-only rule (best effort) | Yes → active | only when Personal Injury = Yes |
| Unit-Owners Rental to Others | `UnitsRegularlyRentedToOthersInput.Indicator` | `coverage_instance.property.us.personal#ho6_units_rented.coverage_status` | Rater column AT → Rule HO-46 (High (for name-based absence)) | No; Yes → active | only when Unit-Owners Rental to Others = Yes |
| Incidental Farming Personal Liability | `IncidentalFarmingPersonalLiabilityInput.Indicator` | **Not written** | | |  |
| Choose Deductibles by Peril | `RiskInput.UseDeductibleByPeril` | **Not written** | | |  |
| Coverage Tier / Package | `LineInput.CoveragePackage` | **Not written** | | |  |
| Equipment Breakdown | `RiskInput.EquipmentBreakdown` | `section_icoverages.property.us.personal.equipment_breakdown` | Rater column AS → Rule HO-45 (High) | No; Yes |  |
| Permitted Incidental Occupancy | `RiskInput.PermittedIncidentalOccupancy` | `coverage_instance.property.us.personal#business_use.coverage_status` | Rater column AI → Rule HO-35 (Medium/Low) | No; Yes → active | only when Permitted Incidental Occupancy = Yes |
| Special Computer Coverage | `SpecialComputerCoverageInput.Indicator` | **Not written** | | |  |
| Unit-Owners Special Coverage C | `SpecialPersonalPropertyCoverageInput.Indicator` | **Not written** | | |  |

## Underwriting (`underwriting`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Restricted dogs, exotic pets or guard dogs? | `Underwriting.animals` | `underwriting_personal_home.property.us.personal.any_animals_or_exotic_pets` | Quote-only rule (best effort) | Yes / No |  |
| Trampoline on the premises? | `Underwriting.trampoline` | `underwriting_personal_home.property.us.personal.is_there_atrampoline_on_premises` | Quote-only rule (best effort) | Yes / No |  |
| Property for sale or in disrepair? | `Underwriting.disrepair` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Property under renovation? | `Underwriting.renovation` | `dwelling_asset.property.personal.is_under_renovation` | Quote-only rule (best effort) | Yes / No |  |
| Business, day care or farming use? | `Underwriting.business` | `underwriting_personal_home.property.us.personal.is_business_conducted_on_premises` | Quote-only rule (best effort) | Yes / No |  |
| Foreclosure or bankruptcy history? | `Underwriting.financialHistory` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Criminal indictment history? | `Underwriting.criminalHistory` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Known lead paint or landfill exposure? | `Underwriting.leadOrLandfill` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Prior insurance declined or cancelled? | `Underwriting.declined` | `underwriting_personal_home.property.us.personal.any_coverage_declined_cancelled` | Quote-only rule (best effort) | Yes / No |  |
| Known sinkhole activity? | `Underwriting.sinkhole` | `underwriting_personal_home.property.us.personal.is_applicant_aware_of_sinkhole_claim` | Quote-only rule (best effort) | Yes / No |  |
| Details for any Yes answers | `Underwriting.Notes` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: the notes text | text |  |
| Swimming pool on premises? | `Underwriting.SwimmingPool` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Hot tub on premises? | `Underwriting.HotTub` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Property for sale? | `Underwriting.ForSale` | `underwriting_personal_home.property.us.personal.is_home_for_sale` | Quote-only rule (best effort) | Yes / No |  |
| Property in disrepair? | `Underwriting.Disrepair` | `underwriting_personal_home.property.us.personal.is_property_disrepair` | Quote-only rule (best effort) | Yes / No |  |
| Day care use? | `Underwriting.DayCare` | `underwriting_personal_home.property.us.personal.is_day_care` | Quote-only rule (best effort) | Yes / No |  |
| Farming use? | `Underwriting.Farming` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Incidental occupancy? | `Underwriting.IncidentalOccupancy` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Foreclosure history? | `Underwriting.Foreclosure` | `underwriting_personal_home.property.us.personal.had_recent_foreclosure` | Quote-only rule (best effort) | Yes / No |  |
| Bankruptcy history? | `Underwriting.Bankruptcy` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Known lead paint? | `Underwriting.LeadPaint` | `underwriting_personal_home.property.us.personal.any_lead_paint` | Quote-only rule (best effort) | Yes / No |  |
| Landfill exposure? | `Underwriting.Landfill` | `underwriting_personal_home.property.us.personal.is_swelling_located_on_landfill` | Quote-only rule (best effort) | Yes / No |  |
| Renovation completion date | `Underwriting.RenovationCompletionDate` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |
| Renovation dollar value | `Underwriting.RenovationValue` | `underwriting_personal_home.property.us.personal.agent_remarks` | Quote-only rule: a line "<question> Yes/No" under "--- Other underwriting answers ---" | text | No Helix question of its own |

## Insurance history (`insurancehistory`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Prior insurance to disclose? | `History.HasPrior` | **Not written** | | |  |
| Insurance within the past 30 days? | `History.InsuredPast30Days` | **Not written** | | |  |

### Prior policies (list `priorInsurance`)

**No Helix mapping:** the entries are kept only in the local quote.
The one exception: if *Prior policy number* on the New quote page is empty, the first entry's policy number is used for it. It then goes to that field's targets.

| UI field | Key |
|---|---|
| Carrier Name | `PriorInsurance.CarrierName` |
| Effective Date | `PriorInsurance.EffectiveDate` |
| Expiration Date | `PriorInsurance.ExpirationDate` |
| Limit | `PriorInsurance.Limit` |
| Deductible | `PriorInsurance.Deductible` |
| Prior policy number | `PriorInsurance.PolicyNumber` |

## Claims history (`claimshistory`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Prior losses to disclose? | `Claims.HasLosses` | **Not written** | | |  |

### Prior losses (list `losses`)

Each entry *n* becomes one `loss.property.us.personal` record, instance `loss_n`.

| UI field | Key | Helix target | Notes |
|---|---|---|---|
| Loss Type | `LossInput.LossType` | `loss.property.us.personal#loss_n.loss_type` |  |
| Date Of Loss | `LossInput.DateOfLoss` | `loss.property.us.personal#loss_n.date_of_loss` |  |
| Amount Paid | `LossInput.AmountPaid` | `loss.property.us.personal#loss_n.amount_paid` |  |
| Description | `LossInput.Description` | `loss.property.us.personal#loss_n.description` |  |
| Loss source | `LossInput.Source` | `loss.property.us.personal#loss_n.source` |  |

## Coverage summary (`summary`)

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| Include Forms | `PolicyAdmin.UseDCTFormsAndMessages` | **Not written** | | |  |

## Additional interests (`additionalinterests`)

### Additional interests (list `interests`)

Each entry *n* becomes one `additional_other_interest.property.us.personal` record, instance `interest_n`.

| UI field | Key | Helix target | Notes |
|---|---|---|---|
| Interest type | `type` | `additional_other_interest.property.us.personal#interest_n.interest_type` |  |
| Name | `name` | `additional_other_interest.property.us.personal#interest_n.name` |  |
| Mailing address | `address` | `additional_other_interest.property.us.personal#interest_n.address_line1` |  |
| Mortgage rank | `rank` | **Not written** |  |
| Loan / reference number | `reference` | `additional_other_interest.property.us.personal#interest_n.loan_number` |  |

## Payment details (`payment`)

Opens only after the application is submitted for rating. Helix has no billing or payment fields, so nothing on this page is written.

| UI field | Key | Helix target | Path | Value conversion | Notes |
|---|---|---|---|---|---|
| How will this policy be billed? | `Billing.BilledTo` | **Not written** | | |  |
| Payment Plan | `Billing.PaymentPlan` | **Not written** | | |  |
| What day of the month will the invoice be due? | `Billing.InvoiceDueDate` | **Not written** | | |  |

## Disabled rules for columns the quote fills

- HO-07 column G → `residual_market_placement.property.us.personal.placement_basis_code`: A residual market placement needs a property asset record the migrator does not write, and every row is FLCPIC. Open question: are there origins other than Citizens?
- HO-74 column BV → `residual_market_placement.property.us.personal.external_policy_number`: Empty in every test row, and a residual market placement needs a property asset record the migrator does not write.

## Open points

- The quote-only targets and the coverage codes COD, COF, IDENTITY_FRAUD, UNSCHEDULED_JEWELRY and PERSONAL_INJURY are proposals that need agreement (S18).
- Some fields reach Helix twice, once through a rater column rule and once through a quote-only rule. Year built is an example: it goes to `dwelling_asset.construction_year` and to `dwelling.year_built`. Pick one target per field once the targets are agreed.
