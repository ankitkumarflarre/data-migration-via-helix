package quote

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newQuote(t *testing.T) (*Store, *Quote) {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	q, err := s.Create(map[string]any{"PolicyInput.EffectiveDate": "2027-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	return s, q
}
func completeValues(q *Quote, p Page) map[string]any {
	out := map[string]any{}
	for _, f := range p.Fields {
		if f.ReadOnly {
			continue
		}
		if v, ok := q.Values[f.Key]; ok {
			out[f.Key] = v
			continue
		}
		if !f.Required {
			continue
		}
		if len(f.Options) > 0 {
			out[f.Key] = f.Options[0]
			continue
		}
		switch f.Type {
		case "boolean":
			out[f.Key] = false
		case "integer", "number":
			if f.Min != nil {
				out[f.Key] = *f.Min
			} else {
				out[f.Key] = float64(1)
			}
		case "date":
			out[f.Key] = "2027-01-01"
		case "email":
			out[f.Key] = "test@example.com"
		case "zip":
			out[f.Key] = "33101"
		default:
			out[f.Key] = "Test"
		}
	}
	if p.ID == "account" {
		out["AccountInput.PrimaryPhone"] = "3055550100"
	}
	if p.ID == "dwellinginfo" {
		out["DwellingInput.YearBuilt"] = float64(1990)
		out["DwellingInput.RoofGeometry"] = "Gable"
	}
	return out
}
func advance(t *testing.T, s *Store, q *Quote) *Quote {
	t.Helper()
	p := ActivePages(q)[index(ActivePages(q), q.CurrentPage)]
	next, err := s.Save(q.ID, Input{Version: q.Version, Page: p.ID, Values: completeValues(q, p), Advance: true})
	if err != nil {
		t.Fatalf("%s: %v %#v", p.ID, err, err)
	}
	return next
}
func TestCreatePersistsAndResumes(t *testing.T) {
	s, q := newQuote(t)
	if q.CurrentPage != "account" || q.Status != "draft" {
		t.Fatal(q)
	}
	reopened, err := Open(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := reopened.Get(q.ID)
	if err != nil || saved.Number != q.Number || saved.Values["PolicyInput.EffectiveDate"] != "2027-01-01" {
		t.Fatal(saved, err)
	}
	all, err := reopened.List()
	if err != nil || len(all) != 1 {
		t.Fatal(all, err)
	}
}
func TestDraftAndAdvanceValidation(t *testing.T) {
	s, q := newQuote(t)
	saved, err := s.Save(q.ID, Input{Version: q.Version, Page: "account", Values: map[string]any{"AccountInput.FirstName": "Alex"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(q.ID, Input{Version: saved.Version, Page: "account", Advance: true})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["AccountInput.LastName"] == "" {
		t.Fatal(err)
	}
	actual, _ := s.Get(q.ID)
	if actual.Version != saved.Version {
		t.Fatal("failed validation changed stored version")
	}
}
func TestOptimisticConcurrency(t *testing.T) {
	s, q := newQuote(t)
	_, err := s.Save(q.ID, Input{Version: q.Version, Page: "account"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "account"})
	if !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}
func TestCannotSkipOrForgeReadOnly(t *testing.T) {
	s, q := newQuote(t)
	_, err := s.Save(q.ID, Input{Version: q.Version, Page: "review", Advance: true})
	if !errors.Is(err, ErrTransition) {
		t.Fatal(err)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "account", Values: map[string]any{"PolicyPremiums.Premium": 1}})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal(err)
	}
	_, err = s.Get("../../secret")
	if !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}
func TestFullJourneyAndEarlierEditInvalidatesSubmission(t *testing.T) {
	s, q := newQuote(t)
	for q.CurrentPage != "review" {
		q = advance(t, s, q)
	}
	submitted, err := s.Submit(q.ID, q.Version)
	if err != nil || submitted.Status != "ready_for_rating" {
		t.Fatal(submitted, err)
	}
	edited, err := s.Save(q.ID, Input{Version: submitted.Version, Page: "account", Values: map[string]any{"AccountInput.FirstName": "Changed"}})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Status != "draft" || len(edited.Completed) != 1 {
		t.Fatal(edited)
	}
	_, err = s.Submit(edited.ID, edited.Version)
	if !errors.Is(err, ErrTransition) {
		t.Fatal(err)
	}
}
func TestCollectionDatesAndRequiredRows(t *testing.T) {
	s, q := newQuote(t)
	for q.CurrentPage != "insurancehistory" {
		q = advance(t, s, q)
	}
	_, err := s.Save(q.ID, Input{Version: q.Version, Page: "insurancehistory", Values: map[string]any{"History.HasPrior": true}, Advance: true})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["priorInsurance"] == "" {
		t.Fatal(err)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "insurancehistory", Values: map[string]any{"History.HasPrior": true}, Rows: []map[string]any{{"PriorInsurance.CarrierName": "Example", "PriorInsurance.EffectiveDate": "2026-06-01", "PriorInsurance.ExpirationDate": "2026-01-01"}}, Advance: true})
	if !errors.As(err, &validation) || validation.Fields["priorInsurance.0.PriorInsurance.ExpirationDate"] == "" {
		t.Fatal(err)
	}
}
func TestConditionalHO6AndBilling(t *testing.T) {
	s, q := newQuote(t)
	q, err := s.Save(q.ID, Input{Version: q.Version, Page: "newquote", Values: map[string]any{"DwellingInput.Form": "HO6", "Quote.DCTBilling": true}, Advance: true})
	if err != nil {
		t.Fatal(err)
	}
	if index(ActivePages(q), "billing") < 0 || q.Values["DwellingInput.RoofGeometry"] != "NA" {
		t.Fatal(q)
	}
	q = advance(t, s, q)
	p := ActivePages(q)[index(ActivePages(q), "dwellinginfo")]
	values := completeValues(q, p)
	delete(values, "DwellingInput.YearBuilt")
	values["DwellingInput.RoofGeometry"] = "NA"
	q, err = s.Save(q.ID, Input{Version: q.Version, Page: p.ID, Values: values, Advance: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "dwellingcoverage", Values: map[string]any{"CoverageCPersonalPropertyHO46Input.Limit": float64(9999)}, Advance: true})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["CoverageCPersonalPropertyHO46Input.Limit"] == "" {
		t.Fatal(err)
	}
}
func TestHTTPValidationAndRoundtrip(t *testing.T) {
	s, _ := newQuote(t)
	mux := http.NewServeMux()
	Register(mux, s)
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"values":{}}`, 422}, {`{"values":{"PolicyInput.EffectiveDate":"2027-02-30"}}`, 422}, {`{"values":{"PolicyInput.EffectiveDate":"2027-02-01"}}`, 201}, {`{"values":{}} {}`, 400}, {`{"unexpected":true}`, 400}} {
		r := httptest.NewRequest("POST", "/api/quotes", strings.NewReader(tc.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.body, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/quotes", nil))
	var list []Quote
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatal(w.Body.String(), err)
	}
}

func TestCoverageDefaultsAndMinimums(t *testing.T) {
	s, q := newQuote(t)
	q = advance(t, s, q)
	q = advance(t, s, q)
	q, err := s.Save(q.ID, Input{Version: q.Version, Page: "dwellingcoverage", Values: map[string]any{"CoverageADwellingInput.Limit": float64(300000)}, Advance: true})
	if err != nil {
		t.Fatal(err)
	}
	if q.Values["CoverageCPersonalPropertyHO3Input.Limit"] != float64(150000) || q.Values["CoverageDLossOfUseInput.Limit"] != float64(60000) || q.Values["ReplacementCostDwellingInput.ReplacementCostValue"] != float64(300000) {
		t.Fatal(q.Values)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "dwellingcoverage", Values: map[string]any{"CoverageCPersonalPropertyHO3Input.Limit": float64(1)}, Advance: true})
	var v *ValidationError
	if !errors.As(err, &v) || v.Fields["CoverageCPersonalPropertyHO3Input.Limit"] == "" {
		t.Fatal(err)
	}
}
func TestRemovingConditionalCollection(t *testing.T) {
	s, q := newQuote(t)
	for q.CurrentPage != "insurancehistory" {
		q = advance(t, s, q)
	}
	q, err := s.Save(q.ID, Input{Version: q.Version, Page: "insurancehistory", Values: map[string]any{"History.HasPrior": true}, Rows: []map[string]any{{"PriorInsurance.CarrierName": "Test carrier"}}})
	if err != nil {
		t.Fatal(err)
	}
	q, err = s.Save(q.ID, Input{Version: q.Version, Page: "insurancehistory", Values: map[string]any{"History.HasPrior": false}, Rows: q.Collections["priorInsurance"], Advance: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Collections["priorInsurance"]) != 0 {
		t.Fatal("hidden prior policies retained")
	}
}
func TestBillingEscrowNeedsMortgagee(t *testing.T) {
	s, q := newQuote(t)
	q, err := s.Save(q.ID, Input{Version: q.Version, Page: "newquote", Values: map[string]any{"Quote.DCTBilling": true}, Advance: true})
	if err != nil {
		t.Fatal(err)
	}
	for q.CurrentPage != "billing" {
		q = advance(t, s, q)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "billing", Values: map[string]any{"Billing.BillClass": "Mortgagee Escrow", "Billing.PaymentPlan": "Annual"}, Advance: true})
	var v *ValidationError
	if !errors.As(err, &v) || v.Fields["Billing.BillClass"] == "" {
		t.Fatal(err)
	}
	q, err = s.Save(q.ID, Input{Version: q.Version, Page: "billing", Values: map[string]any{"Billing.BillClass": "Direct Bill", "Billing.PaymentPlan": "Annual"}, Advance: true})
	if err != nil || q.CurrentPage != "review" {
		t.Fatal(err)
	}
}

func TestSourceFieldsPersistAndReadOnlyCalculations(t *testing.T) {
	s, q := newQuote(t)
	q, err := s.Save(q.ID, Input{Version: q.Version, Page: "account", Values: map[string]any{"AccountInput.FirstName": "Avery", "AccountInput.LastName": "Demo", "Applicant.EntityType": "Individual", "Applicant.DateOfBirth": "1985-04-12", "Applicant.Producer": "Demo agency"}, Rows: []map[string]any{{"PersonInput.FirstName": "Casey", "PersonInput.LastName": "Demo"}}})
	if err != nil {
		t.Fatal(err)
	}
	q, err = s.Get(q.ID)
	if err != nil || q.Values["AccountInput.Name"] != "Avery Demo" || q.Values["Applicant.Producer"] != "Demo agency" || q.Collections["coapplicants"][0]["PersonOutputNonShredded.CoapplicantLabel"] != "Co-applicant #1" {
		t.Fatal(q, err)
	}
	if q.Values["PolicyInput.Term"] != float64(12) || q.Values["PolicyInput.ExpirationDate"] != "2028-01-01" {
		t.Fatal(q.Values)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "account", Values: map[string]any{"AccountInput.Name": "Forged"}})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal("computed name accepted", err)
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "account", Rows: []map[string]any{{"PersonOutputNonShredded.CoapplicantLabel": "Forged"}}})
	if !errors.As(err, &validation) {
		t.Fatal("computed collection field accepted", err)
	}
}
func TestNewCoverageSelectionsAndDisabledDeductible(t *testing.T) {
	s, q := newQuote(t)
	q = advance(t, s, q)
	q = advance(t, s, q)
	values := map[string]any{"CoverageADwellingInput.Limit": float64(400000), "LineInput.CoveragePackage": "Deluxe", "UnscheduledJewelryInput.Indicator": true, "IncidentalFarmingPersonalLiabilityInput.Indicator": true, "RiskInput.UseDeductibleByPeril": true}
	q, err := s.Save(q.ID, Input{Version: q.Version, Page: "dwellingcoverage", Values: values})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(q.ID)
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range values {
		if got.Values[key] != value {
			t.Fatalf("%s did not persist", key)
		}
	}
	if got.Values["CoverageDLossOfUseOutput.IncludedLimit"] != float64(80000) || got.Values["WaterBackupAndSumpOverflowInput.Limit"] != float64(10000) {
		t.Fatal(got.Values)
	}
	if got.Values["CoverageAOutput.Premium"] != nil || got.Values["CoverageBOtherStructuresOutput.IncludedLimit"] != nil {
		t.Fatal("unavailable rating output fabricated")
	}
	_, err = s.Save(q.ID, Input{Version: q.Version, Page: "dwellingcoverage", Values: map[string]any{"DwellingInput.Deductible": "2500"}})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatal("disabled deductible accepted", err)
	}
}
func TestExpirationClampsLeapDayAndOldDraftHydrates(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	q, err := s.Create(map[string]any{"PolicyInput.EffectiveDate": "2028-02-29"})
	if err != nil {
		t.Fatal(err)
	}
	if q.Values["PolicyInput.ExpirationDate"] != "2029-02-28" {
		t.Fatal(q.Values)
	}
	// Missing new optional fields do not invalidate old saved applications.
	for q.CurrentPage != "review" {
		q = advance(t, s, q)
	}
	if _, err = s.Submit(q.ID, q.Version); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyCollectionRowCannotPanicDerivedFields(t *testing.T) {
	s, q := newQuote(t)
	saved, err := s.Save(q.ID, Input{Version: q.Version, Page: "account", Rows: []map[string]any{nil}})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Collections["coapplicants"][0]["PersonOutputNonShredded.CoapplicantLabel"] != "Co-applicant #1" {
		t.Fatal(saved)
	}
	_, err = s.Save(saved.ID, Input{Version: saved.Version, Page: "account", Values: completeValues(saved, Schema.Pages[1]), Rows: []map[string]any{nil}, Advance: true})
	var v *ValidationError
	if !errors.As(err, &v) || v.Fields["coapplicants.0.PersonInput.FirstName"] == "" {
		t.Fatal(err)
	}
}
