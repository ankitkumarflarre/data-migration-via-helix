package billing

import (
	"fmt"
	"strconv"
)

// SeedDate is the system date of the first walkthrough environment (#001–#054).
// Move the date to 1/1/2027 to replay the second one (#055–#117).
const SeedDate = "2026-10-12"

// Seed loads the accounts, payments, suspense, batches, errors, hold events
// and parties of the screenshots. Real-looking personal data (names of the
// testers, emails, street addresses, phone numbers) is replaced with obvious
// test values. It refuses to run on a store that already has data unless
// force is set, which starts from an empty store.
func (s *Store) Seed(force bool) error {
	s.mu.Lock()
	empty := s.st.Empty()
	s.mu.Unlock()
	if !empty && !force {
		return rule("The billing store already has data. Seed only loads into an empty store.")
	}
	return s.update(func(t *tx) error {
		*t.State = *emptyState(SeedDate)
		seed(t)
		t.SystemDate, t.as = SeedDate, ""
		return nil
	})
}

const (
	billingTester = "Billing Tester" // the tester shown as "Geethika D"
	policyTester  = "Policy Tester"  // the tester shown as "Janani Sai"
)

func seed(t *tx) {
	at := func(date, clock, user string) { t.SystemDate, t.clock, t.as = date, clock, user }
	at("2026-09-23", "09:00", billingTester)

	// Parties. TX Test SAFEPOINT is created first so it gets party number 95371.
	tx := t.newParty("Person", map[string]string{"first": "TX Test", "last": "SAFEPOINT", "correspondence": "TX Test SAFEPOINT"}, "2026-09-23")
	t.addRow(tx, "address", "2026-09-23", addr("Legal", "13802 Test Ave", "Corpus Christi", "TX", "78418"))
	t.addRow(tx, "address", "2026-09-23", addr("Legacy Client", "13802 Test Ave", "Corpus Christi", "TX", "78418"))
	t.addRow(tx, "email", "2026-09-23", map[string]string{"type": "Home", "address": "tx.test@example.test", "primary": "Yes"})
	t.addRow(tx, "othername", "2026-09-23", map[string]string{"type": "Formerly Known As", "last": "ZZZLEGACY"})
	t.addRow(tx, "othername", "2026-09-23", map[string]string{"type": "Formerly Known As", "last": "SAFEPOINT"})

	fl := t.newParty("Person", map[string]string{"first": "FL", "last": "Test safepoint"}, "2026-09-23")
	t.addRow(fl, "address", "2026-09-23", addr("Legal", "100 Test Street", "Saint Cloud", "FL", "34772"))
	t.addRow(fl, "email", "2026-09-23", map[string]string{"type": "Home", "address": "fl.test@example.test", "primary": "Yes"})
	t.addRow(fl, "phone", "2026-09-23", map[string]string{"country": "United States", "type": "Home", "number": "555-010-0001", "primary": "Yes"})
	payeeTest := t.newParty("Organization", map[string]string{"organization": "TEST"}, "2026-09-23")
	t.addRow(payeeTest, "address", "2026-09-23", addr("Legal", "200 Test Street", "Saint Cloud", "FL", "34772"))

	payor := t.newParty("Person", map[string]string{"first": "Test", "last": "Payor"}, "2026-09-23")
	t.addRow(payor, "address", "2026-09-23", addr("Legal", "100 Test Street", "Alexandria", "LA", "71302"))
	t.addRow(payor, "email", "2026-09-23", map[string]string{"type": "Home", "address": "test.payor@example.test", "primary": "Yes"})
	est := t.newParty("Person", map[string]string{"first": "Test", "last": "est"}, "2026-02-15")
	t.addRow(est, "address", "2026-02-15", addr("Legal", "300 Test Street", "Austin", "TX", "78701"))
	org := t.newParty("Organization", map[string]string{"organization": "Test Risk Orign"}, "2026-01-01")
	t.addRow(org, "license", "2026-01-01", map[string]string{"type": "Agency License", "number": "TX-0001", "status": "Active", "state": "TX"})
	t.addRow(org, "membership", "2026-01-01", map[string]string{"type": "Agency Id", "id": "AG-1001"})
	agent := &Agent{Name: "Latest347", Address: "333 Test Ave"}

	full := func(class string, due int, p *Party, payee *Party) func(number string) *Account {
		return func(number string) *Account {
			a := t.newAccount(number, p.ID, payee.ID, class, due)
			return a
		}
	}

	// CLAD3057412: $722 check split over seven scheduled items (#005).
	at("2026-10-12", "06:10", billingTester)
	a412 := full("SPD8", 12, payor, payor)("CLAD3057412")
	t412 := &Term{PolicyNumber: "CLAD3057412", Product: "Dwelling", Effective: "2026-10-12", Expiration: "2027-10-12", Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Full Pay", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "DP3", EquityDate: "2027-10-12", Agent: agent, PotentialCommission: 6972}
	t.issueTerm(a412, t412, []*Receivable{
		rcv("2026-10-12", "Fee", "MGA", 2500, t412), rcv("2026-10-12", "Fee", "Policy", 3000, t412), rcv("2026-10-12", "Fee", "EMPA", 100, t412),
		rcv("2026-10-12", "Fee", "FIGA", 100, t412), rcv("2026-10-12", "Fee", "FHCF", 100, t412), rcv("2026-10-12", "Premium", "", 6000, t412), rcv("2026-10-12", "Premium", "", 60400, t412),
	}, "2026-10-12")
	at("2026-10-12", "06:11", "SFP Admin")
	t.activity(a412.Number, "Policy Renewal Configuration Change", "SFP Admin", nil, "On 10/12/2026, the following configuration changes were made to the renewal term for policy CLAD3057412:")
	at("2026-10-12", "07:30", billingTester)
	t.post(t.newPayment(&PaymentInput{Method: "Check", Amount: 72200, EntryDate: "2026-10-12", PostmarkDate: "2026-10-12"}), a412, &PaymentInput{Allocation: "account"})

	// CLAD3057414: $981 outstanding, ready for Enter A Single Payment (#006–#012).
	at("2026-10-12", "06:18", billingTester)
	a414 := full("SPD8", 12, payor, payor)("CLAD3057414")
	t414 := &Term{PolicyNumber: "CLAD3057414", Product: "Dwelling", Effective: "2026-10-12", Expiration: "2027-10-12", Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Full Pay", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "DP3", EquityDate: "2027-10-12", Agent: agent, PotentialCommission: 9927}
	t.issueTerm(a414, t414, []*Receivable{rcv("2026-10-12", "Premium", "", 95400, t414), rcv("2026-10-12", "Fee", "MGA", 2500, t414), rcv("2026-10-12", "Fee", "EMPA", 200, t414)}, "2026-10-12")
	t.as = "SFP Admin"
	t.activity(a414.Number, "Policy Renewal Configuration Change", "SFP Admin", nil, "On 10/12/2026, the following configuration changes were made to the renewal term for policy CLAD3057414:")

	// STXW3081544 "Test est": new business and renewal paid by card (#031).
	at("2027-03-01", "05:26", billingTester)
	a544 := full("SPD8", 1, est, est)("STXW3081544")
	t544 := &Term{PolicyNumber: "STXW3081544", Product: "Homeowners", Effective: "2026-03-01", Expiration: "2027-03-01", Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Full Pay", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "HO3", EquityDate: "2027-03-01", Agent: agent}
	t.issueTerm(a544, t544, []*Receivable{rcv("2026-03-01", "Premium", "", 539400, t544), rcv("2026-03-01", "Fee", "MGA", 2500, t544), rcv("2026-03-01", "Fee", "EMPA", 200, t544)}, "2026-03-24")
	t.as = "SFP Admin"
	t.activity(a544.Number, "Policy Renewal Configuration Change", "SFP Admin", nil, "On 3/1/2027, the following configuration changes were made to the renewal term for policy STXW3081544:")
	at("2027-03-01", "05:29", billingTester)
	card(t, a544, 542100, "4012002000060016", "Test est")
	at("2027-03-01", "05:32", "SFP Admin")
	t.activity(a544.Number, "Policy Renewed", "SFP Admin", nil, "Policy STXW3081544 was renewed.")
	at("2027-03-01", "05:32", billingTester)
	r544 := &Term{PolicyNumber: "STXW3081544", Product: "Homeowners", Effective: "2027-03-01", Expiration: "2028-03-01", Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Full Pay", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "HO3", EquityDate: "2028-03-01", Agent: agent}
	t.issueTerm(a544, r544, []*Receivable{rcvT("2027-03-01", "Renewal", "Premium", "", 598900, r544), rcvT("2027-03-01", "Renewal", "Fee", "MGA", 2500, r544), rcvT("2027-03-01", "Renewal", "Fee", "EMPA", 200, r544)}, "2027-03-24")
	at("2027-03-01", "05:32", "SFP Admin")
	t.activity(a544.Number, "Policy Renewal Configuration Change", "SFP Admin", nil, "On 3/1/2027, the following configuration changes were made to the renewal term for policy STXW3081544:")
	at("2027-03-01", "05:36", billingTester)
	card(t, a544, 320000, "5473500000000014", "Test est")

	// SFLH3074412 "FL Test safepoint": $10 card payment in suspense, ready for
	// Disburse Funds and the hold (#055–#083).
	at("2027-01-01", "03:06", policyTester)
	a412fl := full("SPD9", 1, fl, payeeTest)("SFLH3074412")
	tfl := &Term{PolicyNumber: "SFLH3074412", Product: "Homeowners", Effective: "2027-01-01", Expiration: "2028-01-01", Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Full Pay Premium Finance", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "HO3", EquityDate: "2028-01-01", Agent: agent, PotentialCommission: 18953}
	t.issueTerm(a412fl, tfl, []*Receivable{rcv("2027-01-01", "Premium", "", 180500, tfl), rcv("2027-01-01", "Fee", "MGA", 2500, tfl), rcv("2027-01-01", "Fee", "EMPA", 200, tfl)}, "2027-01-24")
	t.as = "Express Admin"
	tfl.RenewalPlan = "Full Pay"
	t.activity(a412fl.Number, "Policy Renewal Configuration Change", "Express Admin", nil, "On 1/1/2027, the following configuration changes were made to the renewal term for policy SFLH3074412: Renewal Payment Plan was changed from Full Pay Premium Finance to Full Pay.")
	at("2027-01-01", "05:04", policyTester)
	p1000 := card(t, a412fl, 100000, "4012002000060016", "FL Test safepoint")
	at("2027-02-03", "07:43", billingTester)
	reverse(t, p1000)
	at("2027-03-01", "07:47", billingTester)
	t.post(t.newPayment(&PaymentInput{Method: "Check", Amount: 183200, EntryDate: "2027-03-01", PostmarkDate: "2027-03-01"}), a412fl, &PaymentInput{Allocation: "account"})
	at("2027-01-01", "08:47", billingTester)
	p10 := card(t, a412fl, 1000, "4012002000060016", "FL Test safepoint")
	p10.TransactionNumber = "10542530"
	t.Seq["TXN"] = 1

	// STXH3074413 "TX Test SAFEPOINT" (#089–#117).
	at("2027-01-01", "03:18", policyTester)
	a413 := full("SPDK", 1, tx, tx)("STXH3074413")
	t413 := &Term{PolicyNumber: "STXH3074413", Product: "Homeowners", Effective: "2027-01-01", Expiration: "2028-01-01", Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Semi-Annual (55% Down)", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "HO3", EquityDate: "2028-01-01", Agent: agent, PotentialCommission: 259677}
	t.issueTerm(a413, t413, []*Receivable{rcv("2027-01-01", "Premium", "", 2470400, t413), rcv("2027-01-01", "Fee", "MGA", 2500, t413), rcv("2027-01-01", "Fee", "EMPA", 200, t413)}, "2027-01-24")
	t.as = "Express Admin"
	t413.RenewalPlan = "Full Pay"
	t.activity(a413.Number, "Policy Renewal Configuration Change", "Express Admin", nil, "On 1/1/2027, the following configuration changes were made to the renewal term for policy STXH3074413: Renewal Payment Plan was changed from Semi-Annual (55% Down) to Full Pay.")
	at("2027-01-01", "05:05", policyTester)
	p15 := card(t, a413, 1573100, "4012002000060016", "TX Test SAFEPOINT")
	at("2027-02-03", "07:44", billingTester)
	reverse(t, p15)
	at("2027-03-01", "07:50", billingTester)
	t.post(t.newPayment(&PaymentInput{Method: "Check", Amount: 2473100, EntryDate: "2027-03-01", PostmarkDate: "2027-03-01"}), a413, &PaymentInput{Allocation: "account"})

	// Report data from the first environment (O3, O4, O5).
	at(SeedDate, "06:00", "Admin")
	n := 0
	stub := func(number string) *Account {
		if a := t.Accounts[number]; a != nil {
			return a
		}
		n++
		p := t.newParty("Person", map[string]string{"first": "Test", "last": "Insured " + strconv.Itoa(n)}, "2024-01-01")
		t.addRow(p, "address", "2024-01-01", addr("Legal", fmt.Sprintf("%d Test Street", 100+n), "Tampa", "FL", "33601"))
		a := &Account{Number: number, Version: 1, Status: "Active", PayorID: p.ID, PayeeID: p.ID, TargetDueDays: 1, AccountType: "Account", AccountClass: "SPD9", CollectionMethod: "PA", Currency: "USD", AccountPlan: Codes.AccountPlans[0], CreatedOn: "2024-01-01"}
		t.Accounts[number] = a
		return a
	}
	for _, r := range []struct {
		number string
		amount int64
		due    string
	}{
		{"CALH3047821", 427500, "2026-03-01"}, {"CALH3048694", 501900, "2026-03-01"}, {"CALH3049157", 6200, "2026-07-01"},
		{"CLAD2098072", 227600, "2026-03-01"}, {"CLAD3044377", 92800, "2026-03-01"}, {"CLAD3044378", 99800, "2026-03-01"},
		{"CLAD3044405", 49000, "2026-03-01"}, {"CLAD3044409", 382900, "2026-03-01"}, {"CLAD3045410", 134800, "2026-03-01"},
		{"CLAD3045416", 137700, "2026-03-01"}, {"CLAD3045721", 91100, "2026-03-01"},
	} {
		a := stub(r.number)
		eff := addMonths(r.due, -1)
		tm := &Term{PolicyNumber: r.number, Product: "Homeowners", Effective: eff, Expiration: addMonths(eff, 12), Status: "Active", PaymentPlan: "Full Pay", RenewalPlan: "Full Pay", BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: "HO3", EquityDate: addMonths(eff, 12)}
		t.SystemDate = eff
		t.issueTerm(a, tm, []*Receivable{rcv(eff, "Premium", "", r.amount, tm)}, r.due)
		for _, in := range tm.Installments {
			t.SystemDate = in.InstallDate
			t.invoice(tm, in, SystemUser)
			in.PastDue = true
		}
	}
	at(SeedDate, "06:00", "Admin")
	for _, r := range []struct {
		number, date string
		amount       int64
	}{
		{"0000007365", "2024-02-02", 23749763}, {"0000008412", "2025-11-01", 42570}, {"0000008412", "2026-01-01", 9435},
		{"0000008414", "2026-09-01", 56835}, {"0000008454", "2026-09-01", 12140},
	} {
		a := stub(r.number)
		t.SystemDate = r.date
		t.post(t.newPayment(&PaymentInput{Method: "Check", Amount: r.amount, EntryDate: r.date}), a, &PaymentInput{Allocation: "account"})
	}
	for _, r := range []struct {
		number, date, user, desc string
		amount                   int64
	}{
		{"SFLW3071351", "2025-05-21", "SFP Admin", "Other", 24291400}, {"0000000008", "2022-12-06", "SFPFullName292", "Commission", 24275534},
		{"SFLW3071064", "2025-05-21", "SFP Admin", "Other", 12387000}, {"SALH3070538", "2025-05-21", "SFP Admin", "Other", 9382600},
		{"MFLH3001327", "2025-05-21", "SFP Admin", "Other", 2909500}, {"SFLW3071099", "2025-05-21", "SFP Admin", "Other", 2768400},
		{"SFLH3071212", "2025-05-21", "SFP Admin", "Other", 2727400}, {"SFLH3071539", "2025-05-21", "SFP Admin", "Other", 2444600},
		{"SFLD3071230", "2025-05-21", "SFP Admin", "Other", 2242600}, {"MFLW3000076", "2025-05-21", "SFP Admin", "Other", 2115200},
		{"SLAH0387843", "2025-05-21", "SFP Admin", "Other", 1968680}, {"SFLW3071916", "2025-05-21", "SFP Admin", "Other", 1934100},
		{"SFLW3072996", "2025-05-21", "SFP Admin", "Other", 1934100},
	} {
		a := stub(r.number)
		d := &Disbursement{ID: t.next("DSB"), AccountNumber: a.Number, PayeePartyID: a.PayeeID, PayeeName: t.partyName(a.PayeeID), Amount: r.amount, Type: "Regular", Group: Codes.DisbursementGroups[0], Method: "Check", Description: r.desc, Status: "Pending Authorization", RequestDate: r.date, User: r.user, Sources: []DisbursementSource{}}
		t.Disbursements[d.ID] = d
	}
	for _, r := range []struct {
		date, user, method, last4, kind, ident string
		amount                                 int64
	}{
		{"2026-09-19", "110315", "Credit/Debit Card", "0006", "account", "STXD3081495", 348700},
		{"2026-09-15", "110255", "Credit/Debit Card", "0016", "account", "CMSH3057359", 98600},
		{"2026-09-15", "110255", "Credit/Debit Card", "0016", "account", "CMSD3057361", 200900},
		{"2026-09-15", "110255", "Credit/Debit Card", "0016", "account", "CALD3057363", 165848},
		{"2026-08-31", "110255", "Credit/Debit Card", "0016", "account", "CLAH3057266", 98624},
		{"2026-08-13", "tester1@example.test", "Check", "", "account", "STXW3081304", 300000},
		{"2026-07-01", "tester2@example.test", "Check", "", "last", "Test", 10000},
		{"2026-07-01", "Admin", "Check", "", "account", "MFLD2101987", 172300},
		{"2026-07-01", "Admin", "Check", "", "account", "MFLH3014108", 90500},
		{"2026-07-01", "Admin", "Check", "", "account", "MFLH3013750", 577900},
		{"2026-01-14", "Admin", "Check", "", "account", "CLAH3048610", 850400},
		{"2026-01-12", "Admin", "Check", "", "account", "CLAD3048608", 11100},
		{"2026-01-12", "Admin", "Check", "", "account", "CLAD3048608", 92300},
		{"2025-12-31", "Admin", "Check", "", "account", "CLAH3048534", 659000},
		{"2025-12-30", "Admin", "Check", "", "account", "CLAH3048518", 820900},
		{"2025-11-14", "Admin", "Check", "", "account", "CLAD3048184", 98910},
	} {
		at(r.date, "09:00", r.user)
		p := t.newPayment(&PaymentInput{Method: r.method, Amount: r.amount, EntryDate: r.date})
		p.Unidentified = true
		p.Identity = &Identity{}
		if r.kind == "last" {
			p.Identity.Last = r.ident
		} else {
			p.Identity.AccountRef = r.ident
		}
		if r.last4 != "" {
			p.CardBrand, p.CardLast4, p.NameOnCard = "Visa", r.last4, "Test Cardholder"
		}
		t.Payments[p.ID] = p
	}

	// Lockbox batches never deposited (O2), and one deposited sample batch so
	// Processed Batches › Reverse can be shown.
	for _, b := range []struct {
		number, date, clock, status string
		total                       int64
	}{
		{"076", "2018-07-25", "4:18:37 PM", "Processing Error", 21840}, {"999", "2018-08-05", "12:04:57 PM", "Processing", 599700},
		{"208", "2018-09-18", "7:30:02 AM", "Processing Error", 306900}, {"193", "2018-09-19", "10:15:02 AM", "Processing Error", 88700},
		{"951", "2018-09-27", "8:54:52 PM", "Processing Error", 246600}, {"336", "2018-11-07", "4:30:02 PM", "Processing Error", 642800},
		{"385", "2018-12-05", "2:30:05 PM", "Processing Error", 83200}, {"397", "2018-12-11", "4:45:04 PM", "Processing Error", 243500},
		{"423", "2018-12-28", "1:45:02 PM", "Processing Error", 440400},
	} {
		t.Batches[b.number] = &Batch{Number: b.number, Version: 1, EntryDate: b.date, CreatedAt: b.date + "T00:00", User: "admin", Status: b.status, BatchTotal: b.total, Currency: "US Dollars ($)", DefaultMethod: "Check", OrgUnit: "Org Unit 9999", Description: "BHHC LockBox Transmission " + US(b.date) + " " + b.clock}
	}
	t.Batches["1001"] = &Batch{Number: "1001", Version: 1, EntryDate: "2026-10-01", CreatedAt: "2026-10-01T09:00", User: "admin", Status: "Deposited", ControlTotal: 125000, BatchTotal: 125000, Currency: "US Dollars ($)", DefaultMethod: "Check", OrgUnit: "Org Unit 9999", Description: "Sample deposited batch (test data)", DepositedOn: "2026-10-02"}

	// Error queue (O1), with the payload kept for resubmission.
	invoiceSections := []PayloadSection{
		{"_BillingInvoice", `<Invoice id="I4CAD2D7061014553AAC18006310770A8"><AccountReference>SFLH3070506</AccountReference></Invoice>`},
		{"_GenerateInvoice", `<GenerateInvoice id="GA189055BBB69413B8ADB3F58E7B337B4"><AccountId>302299</AccountId><DueDateSet>2024-05-16</DueDateSet></GenerateInvoice>`},
	}
	errs := []*ErrorItem{
		{At: "2024-04-26T09:38", Type: "Generate Invoice", Summary: "Error generating the invoice.", Sections: invoiceSections, InvoiceID: "872444", AsyncID: "A786026B9F64483795CD2849908684E9"},
		{At: "2024-04-26T09:44", Type: "Automatic Pending Cancellation Process", Reference: "CLAD3044370", Summary: "Error processing an automatic policy term pending cancellation request."},
		{At: "2024-04-26T09:50", Type: "Automatic Pending Cancellation Process", Reference: "CLAD3044370", Summary: "Error processing an automatic policy term pending cancellation request."},
		{At: "2024-04-26T09:58", Type: "Automatic Pending Cancellation Process", Reference: "MFLH3000020", Summary: "Error processing an automatic policy term pending cancellation request."},
		{At: "2024-04-26T09:59", Type: "Automatic Pending Cancellation Process", Reference: "MFLH3000020", Summary: "Error processing an automatic policy term pending cancellation request."},
		{At: "2024-04-28T09:09", Type: "Receivable", Reference: "SALH3014952", Summary: "Error processing a receivable request."},
	}
	for i, e := range errs {
		e.ID, e.Version, e.Status, e.Description = t.next("ERR"), 1, "Open", "Object reference not set to an instance of an object."
		if e.Sections == nil {
			async := fmt.Sprintf("B%031X", i+1)
			e.AsyncID = async
			e.Sections = []PayloadSection{{"_PolicyTerm", "<PolicyTerm><PolicyReference>" + e.Reference + "</PolicyReference></PolicyTerm>"}}
		}
		e.Payload = payload(e)
		t.Errors[e.ID] = e
	}

	t.HoldEvents["EVT1001"] = &HoldEvent{ID: "EVT1001", Version: 1, Name: "Test Hold", Description: "testing", Type: "Hold Follow Up", Reason: "Natural Disaster", Start: "2025-03-19"}
	t.Seq["EVT"] = 1
	t.Notifications = []*Notification{
		{"", "TEST INSURED has been approved on 2026-09-30.", "9/30/2026 7:09:26 AM"},
		{"", "Test MS HO has been approved on 2026-09-18.", "9/18/2026 6:18:50 AM"},
		{"Cancellation", "Hello, we have issued policies STXD3102035 and STXD3102036.", "6/13/2025 6:34:08 PM"},
	}
	t.Recent = []string{"STXW3081544", "CLAD3057414", "CLAD3057412", "SFLH3074412", "STXH3074413"}
}

// payload rebuilds the request XML from its sections.
func payload(e *ErrorItem) string {
	root := map[string]string{"Generate Invoice": "BillInvoice.generateRq", "Automatic Pending Cancellation Process": "PolicyTerm.pendingCancellationRq", "Receivable": "Receivable.processRq"}[e.Type]
	body := ""
	for _, s := range e.Sections {
		body += "<" + s.Name + ">" + s.Value + "</" + s.Name + ">"
	}
	attrs := fmt.Sprintf(`asyncID="%s"`, e.AsyncID)
	if e.InvoiceID != "" {
		attrs = fmt.Sprintf(`invoiceID="%s" %s`, e.InvoiceID, attrs)
	}
	return "<" + root + " " + attrs + ">" + body + "</" + root + ">"
}

func addr(kind, line1, city, state, zip string) map[string]string {
	return map[string]string{"kind": kind, "country": "United States", "line1": line1, "city": city, "state": state, "zip": zip}
}

func rcv(date, recv, sub string, amount int64, tm *Term) *Receivable {
	return rcvT(date, "New Business", recv, sub, amount, tm)
}

func rcvT(date, trans, recv, sub string, amount int64, tm *Term) *Receivable {
	return &Receivable{Date: date, Transaction: trans, Receivable: recv, SubType: sub, Amount: amount, Effective: tm.Effective, Expiration: tm.Expiration}
}

func card(t *tx, a *Account, amount int64, number, name string) *Payment {
	p := t.newPayment(&PaymentInput{Method: "Credit/Debit Card", Amount: amount, EntryDate: t.SystemDate})
	p.CardBrand, p.CardLast4, p.NameOnCard, p.Fee = cardBrand(number), number[len(number)-4:], name, CardFee(amount)
	t.post(p, a, &PaymentInput{Allocation: "account"})
	return p
}

func reverse(t *tx, p *Payment) {
	for _, a := range p.Allocations {
		if it := t.findItem(a.TermID, a.ItemID); it != nil {
			it.Cash -= a.Amount
		}
		if s := t.Suspense[a.SuspenseID]; s != nil {
			s.Amount -= a.Amount
		}
		a.Status = "Reversed"
	}
	p.Status, p.ReverseReason, p.ReversedOn = "Reversed", "RP-Reverse Payment", t.SystemDate
	t.activity(p.AccountNumber, "Reverse Payment", t.user(), cents(p.Amount), USD(p.Amount)+" Payment was reversed. Reason: Reverse Payment").PaymentID = p.ID
}
