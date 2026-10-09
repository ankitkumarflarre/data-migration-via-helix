# PersonalHome field coverage audit

Source SHA-256: `ea47f49fe8c759c88e47067892e318a9d96f154065fd981c4afac103d7ecc296`.

All 170 detailed inventory rows are accounted for below. This is field/control coverage, not a claim that carrier integrations are complete.
Policy Information is suppressed by the source LOB: its fields appear under New quote. Pricing is suppressed: its fields appear under Coverage summary. Location Detail appears inside Risk schedule. Existing interest names retain the local `name` storage key for compatibility.
Carrier outputs remain unavailable (not zero or fabricated). Hidden/internal source fields remain hidden in the form. Local source controls use the existing collection editor; AOI Details expands an editable row. Location OK saves a draft, Cancel restores saved values, and Delete clears the draft address only.
The source table explicitly says Floor Unit is visible when Form is NOT HO6; this overrides the previous implementation. Pool detail fields are always visible as documented. Months unoccupied is hidden for secondary/seasonal usage.

| Source page | Inventory rows | Editable | Read-only | Internal | Local actions | Unavailable values/actions |
|---|---:|---:|---:|---:|---:|---:|
| Applicant | 28 | 14 | 5 | 2 | 2 | 5 |
| Policy Information | 6 | 2 | 2 | 0 | 0 | 2 |
| Location Detail | 14 | 6 | 1 | 0 | 4 | 3 |
| Claims History | 8 | 4 | 2 | 0 | 2 | 0 |
| Insurance History | 7 | 5 | 0 | 0 | 2 | 0 |
| Additional Interests | 5 | 1 | 1 | 0 | 3 | 0 |
| Pricing | 9 | 1 | 0 | 0 | 0 | 8 |
| Dwelling Info (Risk Schedule) | 51 | 48 | 3 | 0 | 0 | 0 |
| Dwelling Coverage | 42 | 31 | 5 | 0 | 0 | 6 |

## Applicant

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Primary Insured Text (`AccountOutputNonShredded.PrimaryInsuredText`) | account | read-only → `AccountOutputNonShredded.PrimaryInsuredText` |
| 2 | First Name (`AccountInput.FirstName`) | account | editable → `AccountInput.FirstName` |
| 3 | Middle Initial (MI) (`AccountInput.MiddleName`) | account | editable → `AccountInput.MiddleName` |
| 4 | Last Name (`AccountInput.LastName`) | account | editable → `AccountInput.LastName` |
| 5 | Name (Calculated) (`AccountInput.Name`) | account | internal → `AccountInput.Name` |
| 6 | Party Search (Manual) (`Action – partyStart:partySearch`) | account | unavailable service action → `Action – partyStart:partySearch` |
| 7 | Auto Search (`Action – partyStart:searchResults`) | account | unavailable service action → `Action – partyStart:searchResults` |
| 8 | Add Co-Applicant (`Action – addObjectRecordRq (Person)`) | account | local action → `collection-add` |
| 9 | Co-Applicant Label (`PersonOutputNonShredded.CoapplicantLabel`) | account | read-only → `PersonOutputNonShredded.CoapplicantLabel` |
| 10 | Co-Applicant First Name (`PersonInput.FirstName`) | account | editable → `PersonInput.FirstName` |
| 11 | Co-Applicant MI (`PersonInput.MiddleName`) | account | editable → `PersonInput.MiddleName` |
| 12 | Co-Applicant Last Name (`PersonInput.LastName`) | account | editable → `PersonInput.LastName` |
| 13 | Delete Co-Applicant (`Action – delete Person`) | account | local action → `collection-remove` |
| 14 | Phone Number (`AccountInput.PrimaryPhone`) | account | editable → `AccountInput.PrimaryPhone` |
| 15 | E-mail Address (`AccountInput.Email`) | account | editable → `AccountInput.Email` |
| 16 | Address 1 (`AccountInput.Address1`) | account | editable → `AccountInput.Address1` |
| 17 | Address 2 (`AccountInput.Address2`) | account | editable → `AccountInput.Address2` |
| 18 | City (`AccountInput.City`) | account | editable → `AccountInput.City` |
| 19 | State (`AccountInput.State`) | account | editable → `AccountInput.State` |
| 20 | Zip Code (`AccountInput.ZipCode`) | account | editable → `AccountInput.ZipCode` |
| 21 | County (`AccountInput.County`) | account | editable → `AccountInput.County` |
| 22 | Geocoding Status (`Geocode.GeocodeStatusDisplay`) | account | read-only → `Geocode.GeocodeStatusDisplay` |
| 23 | Verify Address (`Action – PBBI Geocode`) | account | unavailable service action → `Action – PBBI Geocode` |
| 24 | Edit Verified Address (`Action – ClearLastVerified`) | account | unavailable service action → `Action – ClearLastVerified` |
| 25 | View Map (`Action – PBBI Interactive Map`) | account | unavailable service action → `Action – PBBI Interactive Map` |
| 26 | Bing Map Link (`AccountPrivate.XHTMLMapsBing`) | account | read-only → `AccountPrivate.XHTMLMapsBing` |
| 27 | Google Map Link (`AccountPrivate.XHTMLMapsGoogle`) | account | read-only → `AccountPrivate.XHTMLMapsGoogle` |
| 28 | Latitude / Longitude (`AccountPBBI.Latitude / AccountPBBI.Longitude`) | account | internal → `AccountPBBI.Latitude / AccountPBBI.Longitude` |

## Policy Information

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Description (`PolicyInput.Description`) | newquote | editable → `PolicyInput.Description` |
| 2 | Effective Date (`PolicyInput.EffectiveDate`) | newquote | editable → `PolicyInput.EffectiveDate` |
| 3 | Get Latest Rates Notice (`GetLatestRates (Spacer + Action)`) | newquote | unavailable service action → `GetLatestRates (Spacer + Action)` |
| 4 | Get Latest Rates Button (`Action – GetLatestRates.Value`) | newquote | unavailable service action → `Action – GetLatestRates.Value` |
| 5 | Term (`PolicyInput.Term`) | newquote | read-only → `PolicyInput.Term` |
| 6 | Expiration Date (`PolicyInput.ExpirationDate`) | newquote | read-only → `PolicyInput.ExpirationDate` |

## Location Detail

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Address 1 (`LocationInput.Address1`) | dwellinginfo | editable → `LocationInput.Address1` |
| 2 | Address 2 (`LocationInput.Address2`) | dwellinginfo | editable → `LocationInput.Address2` |
| 3 | City (`LocationInput.City`) | dwellinginfo | editable → `LocationInput.City` |
| 4 | State (`LocationInput.State`) | dwellinginfo | editable → `LocationInput.State` |
| 5 | ZIP Code (`LocationInput.ZipCode`) | dwellinginfo | editable → `LocationInput.ZipCode` |
| 6 | County (`LocationInput.County`) | dwellinginfo | editable → `LocationInput.County` |
| 7 | Geocoding Status (`LocationGeocode.GeocodeStatusDisplay`) | dwellinginfo | read-only → `LocationGeocode.GeocodeStatusDisplay` |
| 8 | Verify Location (`Action – PBBI Location Geocoding`) | dwellinginfo | unavailable service action → `Action – PBBI Location Geocoding` |
| 9 | Edit Verified Address (`Action – ClearLastVerified (Location)`) | dwellinginfo | unavailable service action → `Action – ClearLastVerified (Location)` |
| 10 | View Map (`Action – Interactive Map (Location)`) | dwellinginfo | unavailable service action → `Action – Interactive Map (Location)` |
| 11 | OK (`Action – close/return`) | dwellinginfo | local action → `Action – close/return` |
| 12 | Cancel (`Action – cancelChanges`) | dwellinginfo | local action → `Action – cancelChanges` |
| 13 | Delete Location (`Action – delete (add mode)`) | dwellinginfo | local action → `Action – delete (add mode)` |
| 14 | Reset to Applicant Address (`Action – LocationPrivate.ResetToApplicantAddress`) | dwellinginfo | local action → `Action – LocationPrivate.ResetToApplicantAddress` |

## Claims History

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Prior Losses Header (`Spacer/Static Text`) | claimshistory | read-only → `Spacer/Static Text` |
| 2 | Loss Number (`LossOutputNonShredded.NumberDisplay`) | claimshistory | read-only → `LossOutputNonShredded.NumberDisplay` |
| 3 | Loss Type (`LossInput.LossType`) | claimshistory | editable → `LossInput.LossType` |
| 4 | Date Of Loss (`LossInput.DateOfLoss`) | claimshistory | editable → `LossInput.DateOfLoss` |
| 5 | Amount Paid (`LossInput.AmountPaid`) | claimshistory | editable → `LossInput.AmountPaid` |
| 6 | Description (`LossInput.Description`) | claimshistory | editable → `LossInput.Description` |
| 7 | Delete Loss (`Action – delete (Loss row)`) | claimshistory | local action → `collection-remove` |
| 8 | Add Loss (`Action – add (Loss)`) | claimshistory | local action → `collection-add` |

## Insurance History

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Carrier Name (`PriorInsurance.CarrierName`) | insurancehistory | editable → `PriorInsurance.CarrierName` |
| 2 | Effective Date (`PriorInsurance.EffectiveDate`) | insurancehistory | editable → `PriorInsurance.EffectiveDate` |
| 3 | Expiration Date (`PriorInsurance.ExpirationDate`) | insurancehistory | editable → `PriorInsurance.ExpirationDate` |
| 4 | Limit (`PriorInsurance.Limit`) | insurancehistory | editable → `PriorInsurance.Limit` |
| 5 | Deductible (`PriorInsurance.Deductible`) | insurancehistory | editable → `PriorInsurance.Deductible` |
| 6 | Delete Carrier (`Action – delete (PriorInsurance row)`) | insurancehistory | local action → `collection-remove` |
| 7 | Add Carrier (`Action – add (PriorInsurance)`) | insurancehistory | local action → `collection-add` |

## Additional Interests

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | AOI Description (`AdditionalOtherInterestInput.Description`) | additionalinterests | read-only → `AdditionalOtherInterestInput.Description` |
| 2 | Details Popup (`Action – specificIter (AdditionalOtherInterestDetail)`) | additionalinterests | local action → `collection-details` |
| 3 | Delete AOI (`Action – delete`) | additionalinterests | local action → `collection-remove` |
| 4 | Add Additional Other Interest (`Action – add (AOI popup)`) | additionalinterests | local action → `collection-add` |
| 5 | AOI Name (`AdditionalOtherInterestInput.Name`) | additionalinterests | editable → `name` |

## Pricing

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Rating Worksheet Button (`Action – printjob:Worksheet`) | summary | unavailable service action → `Action – printjob:Worksheet` |
| 2 | Quote Letter Button (`Action – printjob:Quote`) | summary | unavailable service action → `Action – printjob:Quote` |
| 3 | Application Button (`Action – printjob:Application`) | summary | unavailable service action → `Action – printjob:Application` |
| 4 | Include Forms (`PolicyAdmin.UseDCTFormsAndMessages`) | summary | editable → `PolicyAdmin.UseDCTFormsAndMessages` |
| 5 | View Forms Button (`Action – previewAsyncPrintJob`) | summary | unavailable service action → `Action – previewAsyncPrintJob` |
| 6 | Premium (`PolicyPremiums.Premium`) | summary | unavailable service value → `PolicyPremiums.Premium` |
| 7 | Premium Change (`PolicyPremiums.PremiumChange`) | summary | unavailable service value → `PolicyPremiums.PremiumChange` |
| 8 | Premium Prior (`PolicyPremiums.PremiumPrior`) | summary | unavailable service value → `PolicyPremiums.PremiumPrior` |
| 9 | Premium Written (`PolicyPremiums.PremiumWritten`) | summary | unavailable service value → `PolicyPremiums.PremiumWritten` |

## Dwelling Info (Risk Schedule)

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Policy Form (`DwellingInput.Form`) | dwellinginfo | read-only → `DwellingInput.Form` |
| 2 | Usage (`DwellingInput.UseType`) | dwellinginfo | editable → `DwellingInput.UseType` |
| 3 | Occupancy (`DwellingInput.OccupancyType`) | dwellinginfo | editable → `DwellingInput.OccupancyType` |
| 4 | # of Months Unoccupied (`DwellingInput.MonthsUnoccupied`) | dwellinginfo | editable → `DwellingInput.MonthsUnoccupied` |
| 5 | Structure (`DwellingInput.BuildingType`) | dwellinginfo | editable → `DwellingInput.BuildingType` |
| 6 | Year Built (`DwellingInput.YearBuilt`) | dwellinginfo | editable → `DwellingInput.YearBuilt` |
| 7 | Construction (`DwellingInput.Construction`) | dwellinginfo | editable → `DwellingInput.Construction` |
| 8 | Protection Class (`DwellingInput.RatedProtectionClass`) | dwellinginfo | editable → `DwellingInput.RatedProtectionClass` |
| 9 | # of Families (`DwellingInput.NumberOfFamilies`) | dwellinginfo | editable → `DwellingInput.NumberOfFamilies` |
| 10 | # of Units in Fire Division (`DwellingInput.NumberOfUnits`) | dwellinginfo | editable → `DwellingInput.NumberOfUnits` |
| 11 | Floor Unit (Floor #) (`DwellingInput.NumberOfFloor`) | dwellinginfo | editable → `DwellingInput.NumberOfFloor` |
| 12 | Foundation Type (`DwellingInput.FoundationType`) | dwellinginfo | editable → `DwellingInput.FoundationType` |
| 13 | Square Feet (`DwellingInput.SquareFeet`) | dwellinginfo | editable → `DwellingInput.SquareFeet` |
| 14 | Pool on Premises? (`DwellingInput.SwimmingPool`) | dwellinginfo | editable → `DwellingInput.SwimmingPool` |
| 15 | Slide (Pool) (`DwellingInput.PoolSlide`) | dwellinginfo | editable → `DwellingInput.PoolSlide` |
| 16 | Is pool surrounded by fence? (`DwellingInput.PoolApprovedFence`) | dwellinginfo | editable → `DwellingInput.PoolApprovedFence` |
| 17 | Diving Board (Pool) (`DwellingInput.PoolDivingBoard`) | dwellinginfo | editable → `DwellingInput.PoolDivingBoard` |
| 18 | Number of Stories (`DwellingInput.NumberOfStories`) | dwellinginfo | editable → `DwellingInput.NumberOfStories` |
| 19 | Primary Heat Type (`DwellingInput.PrimaryHeatType`) | dwellinginfo | editable → `DwellingInput.PrimaryHeatType` |
| 20 | Supplemental Heat Type (`DwellingInput.SupplementalHeatType`) | dwellinginfo | editable → `DwellingInput.SupplementalHeatType` |
| 21 | Building Code Grade (BCEG) (`DwellingInput.BuildingCodeEffectivenessGrading`) | dwellinginfo | editable → `DwellingInput.BuildingCodeEffectivenessGrading` |
| 22 | Electrical System (`DwellingInput.ElectricalSystem`) | dwellinginfo | editable → `DwellingInput.ElectricalSystem` |
| 23 | Opening Protection (`DwellingInput.OpeningProtection`) | dwellinginfo | editable → `DwellingInput.OpeningProtection` |
| 24 | Roof Type (`DwellingInput.RoofType`) | dwellinginfo | editable → `DwellingInput.RoofType` |
| 25 | Sprinkler System (`DwellingInput.Sprinkler`) | dwellinginfo | editable → `DwellingInput.Sprinkler` |
| 26 | Locked Security Gate (`DwellingInput.LockedSecurityGate`) | dwellinginfo | editable → `DwellingInput.LockedSecurityGate` |
| 27 | Security Guard (`DwellingInput.SecurityAttendant`) | dwellinginfo | editable → `DwellingInput.SecurityAttendant` |
| 28 | Wiring (`DwellingInput.Wiring`) | dwellinginfo | editable → `DwellingInput.Wiring` |
| 29 | # of Amps (`DwellingInput.NumberOfAmps`) | dwellinginfo | editable → `DwellingInput.NumberOfAmps` |
| 30 | Plumbing Pipe Material (`DwellingInput.PlumbingType`) | dwellinginfo | editable → `DwellingInput.PlumbingType` |
| 31 | Roof Shape (`DwellingInput.RoofGeometry`) | dwellinginfo | editable → `DwellingInput.RoofGeometry` |
| 32 | Miles To Fire Station (`DwellingInput.DistanceToFireStation`) | dwellinginfo | editable → `DwellingInput.DistanceToFireStation` |
| 33 | Feet To Hydrant (`DwellingInput.DistanceToHydrant`) | dwellinginfo | editable → `DwellingInput.DistanceToHydrant` |
| 34 | Responding Fire Department (`DwellingInput.RespondingFireDepartment`) | dwellinginfo | editable → `DwellingInput.RespondingFireDepartment` |
| 35 | Remaining Useful Life of Roof (`UpdatedServicesInfo.RemainingRoofLife`) | dwellinginfo | editable → `UpdatedServicesInfo.RemainingRoofLife` |
| 36 | Year Electrical Updated (`UpdatedServicesInfo.ElectricSystemUpdateYear`) | dwellinginfo | editable → `UpdatedServicesInfo.ElectricSystemUpdateYear` |
| 37 | Year Heating Updated (`UpdatedServicesInfo.HeatingUpdateYear`) | dwellinginfo | editable → `UpdatedServicesInfo.HeatingUpdateYear` |
| 38 | Year Plumbing Updated (`UpdatedServicesInfo.PlumbingUpdateYear`) | dwellinginfo | editable → `UpdatedServicesInfo.PlumbingUpdateYear` |
| 39 | Year Roof Updated (`UpdatedServicesInfo.RoofUpdateYear`) | dwellinginfo | editable → `UpdatedServicesInfo.RoofUpdateYear` |
| 40 | Secondary Water Resistance (`DwellingInput.SecondaryWaterResistance`) | dwellinginfo | editable → `DwellingInput.SecondaryWaterResistance` |
| 41 | Accessible all year (`DwellingInput.DwellingAccessibleToFireEquipment`) | dwellinginfo | editable → `DwellingInput.DwellingAccessibleToFireEquipment` |
| 42 | Roof to Wall Attachment (`DwellingInput.RoofToWallAttachment`) | dwellinginfo | editable → `DwellingInput.RoofToWallAttachment` |
| 43 | Roof Cover (`DwellingInput.RoofCover`) | dwellinginfo | editable → `DwellingInput.RoofCover` |
| 44 | Terrain (`DwellingInput.Terrain`) | dwellinginfo | editable → `DwellingInput.Terrain` |
| 45 | Floor to Foundation Attachment (`DwellingInput.FloorToFoundationAttachment`) | dwellinginfo | editable → `DwellingInput.FloorToFoundationAttachment` |
| 46 | Wind Speed Design (`DwellingInput.WindSpeedDesign`) | dwellinginfo | editable → `DwellingInput.WindSpeedDesign` |
| 47 | Wind Speed Location (`DwellingInput.WindSpeedLocation`) | dwellinginfo | editable → `DwellingInput.WindSpeedLocation` |
| 48 | Roof Deck Attachment (`DwellingInput.RoofDeckAttachment`) | dwellinginfo | editable → `DwellingInput.RoofDeckAttachment` |
| 49 | Hurricane Deductible (`DwellingInput.HurricaneDeductible`) | dwellinginfo | read-only → `DwellingInput.HurricaneDeductible` |
| 50 | All-Peril Deductible (`DwellingInput.Deductible`) | dwellinginfo | read-only → `DwellingInput.Deductible` |
| 51 | Number Of Bathrooms (`DwellingInput.NumberOfBathrooms`) | dwellinginfo | editable → `DwellingInput.NumberOfBathrooms` |

## Dwelling Coverage

| ID | Source field/control | Application page | Implementation |
|---|---|---|---|
| 1 | Dwelling Limit (A) (`CoverageADwellingInput.Limit`) | dwellingcoverage | editable → `CoverageADwellingInput.Limit` |
| 2 | Other Structures Limit (B) (`CoverageBOtherStructuresInput.Limit`) | dwellingcoverage | editable → `CoverageBOtherStructuresInput.Limit` |
| 3 | Cov B Included Limit (`CoverageBOtherStructuresOutput.IncludedLimit`) | dwellingcoverage | unavailable service value → `CoverageBOtherStructuresOutput.IncludedLimit` |
| 4 | Personal Property Limit (C) – HO3 (`CoverageCPersonalPropertyHO3Input.Limit`) | dwellingcoverage | editable → `CoverageCPersonalPropertyHO3Input.Limit` |
| 5 | Personal Property Limit (C) – HO6 (`CoverageCPersonalPropertyHO46Input.Limit`) | dwellingcoverage | editable → `CoverageCPersonalPropertyHO46Input.Limit` |
| 6 | Cov C Included Limit (`CoverageCPersonalPropertyHO3Output.IncludedLimit`) | dwellingcoverage | unavailable service value → `CoverageCPersonalPropertyHO3Output.IncludedLimit` |
| 7 | Loss of Use Limit (D) (`CoverageDLossOfUseInput.Limit`) | dwellingcoverage | editable → `CoverageDLossOfUseInput.Limit` |
| 8 | Cov D Included Limit (`CoverageDLossOfUseOutput.IncludedLimit`) | dwellingcoverage | read-only → `CoverageDLossOfUseOutput.IncludedLimit` |
| 9 | Personal Liability Limit (E) (`CoverageELiabilityInput.Limit`) | dwellingcoverage | read-only → `CoverageELiabilityInput.Limit` |
| 10 | Medical Payments Limit (F) (`CoverageFMedicalInput.Limit`) | dwellingcoverage | read-only → `CoverageFMedicalInput.Limit` |
| 11 | All-Peril Deductible (`DwellingInput.Deductible`) | dwellingcoverage | editable → `DwellingInput.Deductible` |
| 12 | Hurricane Deductible (`DwellingInput.HurricaneDeductible`) | dwellingcoverage | editable → `DwellingInput.HurricaneDeductible` |
| 13 | Replacement Cost – Dwelling (`ReplacementCostDwellingInput.Indicator`) | dwellingcoverage | editable → `ReplacementCostDwellingInput.Indicator` |
| 14 | Replacement Cost Value (`ReplacementCostDwellingInput.ReplacementCostValue`) | dwellingcoverage | editable → `ReplacementCostDwellingInput.ReplacementCostValue` |
| 15 | Sinkhole Indicator (`SinkholeInput.Indicator`) | dwellingcoverage | editable → `SinkholeInput.Indicator` |
| 16 | Ordinance Or Law (`OrdinanceOrLawInput.Limit`) | dwellingcoverage | editable → `OrdinanceOrLawInput.Limit` |
| 17 | Water Back Up and Sump Overflow (`WaterBackupAndSumpOverflowInput.Indicator`) | dwellingcoverage | editable → `WaterBackupAndSumpOverflowInput.Indicator` |
| 18 | Water Backup Limit (`WaterBackupAndSumpOverflowInput.Limit`) | dwellingcoverage | read-only → `WaterBackupAndSumpOverflowInput.Limit` |
| 19 | Limited Fungi or Bacteria (`LimitedFungiBacteriaInput.Limit`) | dwellingcoverage | editable → `LimitedFungiBacteriaInput.Limit` |
| 20 | Loss Assessment Coverage (`LossAssessmentInput.Indicator`) | dwellingcoverage | editable → `LossAssessmentInput.Indicator` |
| 21 | Loss Assessment Limit (`LossAssessmentInput.Limit`) | dwellingcoverage | editable → `LossAssessmentInput.Limit` |
| 22 | Carports, Pool Cages & Screen Enclosures (`CarportsPoolCagesInput.Limit`) | dwellingcoverage | editable → `CarportsPoolCagesInput.Limit` |
| 23 | Replacement Cost – Contents (`ReplacementCostContentsInput.Indicator`) | dwellingcoverage | editable → `ReplacementCostContentsInput.Indicator` |
| 24 | Identity Fraud Expense (`IdentityFraudExpenseCoverageInput.Indicator`) | dwellingcoverage | editable → `IdentityFraudExpenseCoverageInput.Indicator` |
| 25 | Identity Fraud Limit (`IdentityFraudExpenseCoverageInput.Limit`) | dwellingcoverage | read-only → `IdentityFraudExpenseCoverageInput.Limit` |
| 26 | Unscheduled Jewelry Indicator (`UnscheduledJewelryInput.Indicator`) | dwellingcoverage | editable → `UnscheduledJewelryInput.Indicator` |
| 27 | Unscheduled Jewelry Limit (`UnscheduledJewelryInput.Limit`) | dwellingcoverage | editable → `UnscheduledJewelryInput.Limit` |
| 28 | Water Backup of Sewers or Drains (`WaterBackupOfSewersOrDrainsInput.Indicator`) | dwellingcoverage | editable → `WaterBackupOfSewersOrDrainsInput.Indicator` |
| 29 | Personal Property Other Residence (`PersonalPropertyOtherResidenceInput.Indicator`) | dwellingcoverage | editable → `PersonalPropertyOtherResidenceInput.Indicator` |
| 30 | Personal Injury (`PersonalInjuryInput.Indicator`) | dwellingcoverage | editable → `PersonalInjuryInput.Indicator` |
| 31 | Unit-Owners Rental to Others (`UnitsRegularlyRentedToOthersInput.Indicator`) | dwellingcoverage | editable → `UnitsRegularlyRentedToOthersInput.Indicator` |
| 32 | Incidental Farming Personal Liability (`IncidentalFarmingPersonalLiabilityInput.Indicator`) | dwellingcoverage | editable → `IncidentalFarmingPersonalLiabilityInput.Indicator` |
| 33 | Choose Deductibles by Peril (`RiskInput.UseDeductibleByPeril`) | dwellingcoverage | editable → `RiskInput.UseDeductibleByPeril` |
| 34 | Coverage Tier / Package (`LineInput.CoveragePackage`) | dwellingcoverage | editable → `LineInput.CoveragePackage` |
| 35 | Equipment Breakdown (`RiskInput.EquipmentBreakdown`) | dwellingcoverage | editable → `RiskInput.EquipmentBreakdown` |
| 36 | Permitted Incidental Occupancy (`RiskInput.PermittedIncidentalOccupancy`) | dwellingcoverage | editable → `RiskInput.PermittedIncidentalOccupancy` |
| 37 | Special Computer Coverage (`SpecialComputerCoverageInput.Indicator`) | dwellingcoverage | editable → `SpecialComputerCoverageInput.Indicator` |
| 38 | Unit-Owners Special Coverage C (`SpecialPersonalPropertyCoverageInput.Indicator`) | dwellingcoverage | editable → `SpecialPersonalPropertyCoverageInput.Indicator` |
| 39 | Coverage A Premium (display) (`CoverageAOutput.Premium`) | dwellingcoverage | unavailable service value → `CoverageAOutput.Premium` |
| 40 | Coverage B Premium (`CoverageBOtherStructuresOutput.Premium`) | dwellingcoverage | unavailable service value → `CoverageBOtherStructuresOutput.Premium` |
| 41 | Coverage C Premium (`CoverageCPersonalPropertyHO3Output.Amount`) | dwellingcoverage | unavailable service value → `CoverageCPersonalPropertyHO3Output.Amount` |
| 42 | Coverage D Premium (`CoverageDLossOfUseOutput.Premium`) | dwellingcoverage | unavailable service value → `CoverageDLossOfUseOutput.Premium` |

## Narrative-only coverage

Narrative-only fields use local application keys unless a technical key is supplied. No carrier mapping is inferred. These additions are optional so existing saved drafts remain usable.

| Page | Field | Implementation |
|---|---|---|
| account | Entity type (`Applicant.EntityType`) | editable |
| account | Date of birth (`Applicant.DateOfBirth`) | editable |
| account | Producer (`Applicant.Producer`) | editable |
| account | Territory (`Applicant.Territory`) | read-only |
| account | Fire district (`Applicant.FireDistrict`) | read-only |
| newquote | Prior policy number (`PolicyDetails.PriorPolicyNumber`) | editable |
| newquote | Prior carrier (`PolicyDetails.PriorCarrier`) | editable |
| newquote | E-policy discount (`PolicyDetails.EPolicyDiscount`) | editable |
| newquote | Subscription agreement flag (`PolicyDetails.SubscriptionAgreement`) | read-only |
| newquote | Writing company (`PolicyDetails.WritingCompany`) | editable |
| newquote | Consent to rate (`PolicyDetails.ConsentToRate`) | editable |
| newquote | Consent-to-rate factor (`PolicyDetails.CTRFactor`) | editable |
| newquote | Citizens conversion legacy premium (`PolicyDetails.CitizensLegacyPremium`) | editable |
| newquote | Citizens conversion capping details (`PolicyDetails.CitizensCapping`) | editable |
| newquote | Risk origin (`PolicyDetails.RiskOrigin`) | editable |
| underwriting | Swimming pool on premises? (`Underwriting.SwimmingPool`) | editable |
| underwriting | Hot tub on premises? (`Underwriting.HotTub`) | editable |
| underwriting | Property for sale? (`Underwriting.ForSale`) | editable |
| underwriting | Property in disrepair? (`Underwriting.Disrepair`) | editable |
| underwriting | Day care use? (`Underwriting.DayCare`) | editable |
| underwriting | Farming use? (`Underwriting.Farming`) | editable |
| underwriting | Incidental occupancy? (`Underwriting.IncidentalOccupancy`) | editable |
| underwriting | Foreclosure history? (`Underwriting.Foreclosure`) | editable |
| underwriting | Bankruptcy history? (`Underwriting.Bankruptcy`) | editable |
| underwriting | Known lead paint? (`Underwriting.LeadPaint`) | editable |
| underwriting | Landfill exposure? (`Underwriting.Landfill`) | editable |
| underwriting | Renovation completion date (`Underwriting.RenovationCompletionDate`) | editable |
| underwriting | Renovation dollar value (`Underwriting.RenovationValue`) | editable |
| insurancehistory | Insurance within the past 30 days? (`History.InsuredPast30Days`) | editable |
| insurancehistory | Prior policy number (`PriorInsurance.PolicyNumber`) | editable collection |
| claimshistory | Loss source (`LossInput.Source`) | editable collection |
| claimshistory | Insurance score (`Reports.InsuranceScore`) | read-only |
| claimshistory | Insurance score override (`Reports.InsuranceScoreOverride`) | read-only |
| claimshistory | Adverse action reasons (`Reports.AdverseActionReasons`) | read-only |
| claimshistory | CLUE report status (`Reports.CLUEStatus`) | read-only |
| summary | Decline messages (`Rating.DeclineMessages`) | read-only |
| summary | Referral messages (`Rating.ReferralMessages`) | read-only |
| summary | Subscriber agreement and LPOA (`Rating.SubscriberAgreement`) | read-only |
| billing | Initial payment method (`Billing.InitialPaymentMethod`) | editable |
| billing | Initial payment amount (`Billing.InitialPaymentAmount`) | read-only |
| billing | Payment status (`Billing.PaymentStatus`) | read-only |
| billing | Installment due dates (`Billing.InstallmentDueDates`) | read-only |
| billing | Installment amounts (`Billing.InstallmentAmounts`) | read-only |
| dwellingcoverage | Water backup of sewers or drains limit (`WaterBackupOfSewersOrDrainsInput.Limit`) | read-only |
| dwellingcoverage | Personal injury limit (`PersonalInjuryInput.Limit`) | read-only |
| dwellingcoverage | Incidental farming liability limit (`IncidentalFarmingPersonalLiabilityInput.Limit`) | read-only |
| dwellingcoverage | Special computer coverage limit (`SpecialComputerCoverageInput.Limit`) | read-only |

## Unavailable services and unresolved source detail

- Applicant: producer-directory/party search and Maprisk/PBBI geocoding are unavailable. Producer can be entered manually; no address is marked verified.
- Underwriting: the document supplies topics but not exact carrier questions/rules. All named topics are captured locally; decline/referral results are explicitly unavailable.
- Insurance history: the source does not provide its carrier dropdown list. Carrier name remains free text. Advantage-only liability limits are outside this FL Select product.
- Claims, CLUE and insurance score: ordering, FCRA consent text, role permissions and response contracts are missing. Report panels expose documented fields/actions in an unavailable state; they do not claim a report exists. Flood-specific navigation gating requires a carrier rule contract.
- Coverage: Cov B/C included-factor tables, package defaults, jewelry minimum, farming-type rules and peril deductible definitions are not supplied. Selections are saved for review, with explanatory notes. Included D uses the documented 20% HO3 / 40% HO6 factors.
- Coverage summary/Pricing: all four premium displays, form controls, messages, subscriber agreement and Bind are present; real rating, PDFs, signatures and binding remain unavailable.
- Payment/Billing summary: bill class, plan, paperless and method preferences are stored. Due dates, amounts and payment execution require the billing provider. No payment credentials are collected.
- Commit: the local Review page submits for rating only. Carrier policy number, In-Force status, IVANS/Insvista notifications, inspection, Zesty and Data Insights are not fabricated.
- Narrative capping fields are not individually named. A local capping-details input captures notes; exact carrier fields require the authoritative schema. Subscriber/FCRA legal text is also absent.
