# Duck Creek Billing: screens and workflows

Date: 2026-10-14. Source: `Safepoint Billing UI Workflow Documentation.docx` (Work documents/Billing). The document has 117 screenshots with short captions. This write-up records what the screenshots show: navigation, fields, dropdown values, business rules and the problems you can see in the data. `#NNN` is the screenshot's position in the document (1–117).

The screenshots come from two Duck Creek OnDemand environments, both at `…duckcreekondemand.com/Billing/default.aspx`:

| Environment | System date | Screenshots | Accounts used |
|---|---|---|---|
| `uncd679` | 10/12/2026 | #001–#054 | CLAD3057412, CLAD3057414 (Dwelling, check payments), STXW3081544 |
| `uncd680` | 01/01/2027 | #055–#117 | SFLH3074412 (FL HO3), STXH3074413 (TX Homeowners) |

Every page shows the system date and has a **Set System Date** control in the footer. Testers move this date to run time-based processes such as invoicing and renewals. Some footers show the Duck Creek page ID, for example `Carrier_Billing_PaymentProcessing`, `Carrier_Billing_BillAccountSetup_Base`, `Carrier_Party_US_3_0_0_0` and `Carrier_Billing_Transactions_1_0_0_0`.

---

## 1. Navigation map

```
Header: brands (SafePoint / Cajun Underwriters / Manatee) · module switcher [Billing ▾ | Party] · quick search [Account Number ▾][____][🔍] · user
Top nav
├── Home
│   ├── Activities: Search For A Payment · Enter A Single Payment · Unidentified Payments · Create A New Account · Import Statements
│   ├── Recently Accessed (account # + name)
│   ├── Reports (9, see §5)
│   ├── Find a Billing Account (Find account by ▾ / searching for / Search)
│   ├── Notifications (Type, Message, Received; View All Notifications)
│   └── Tasks ("You have no tasks due today")
├── Administration
│   ├── View Error Queue
│   ├── Manage Hold Events
│   ├── Disbursement Request (Disbursement Interface Service)
│   └── Billing Transactions
└── Batch Cash Management
    ├── Batches Awaiting Deposit (landing)
    ├── Start A New Batch
    └── Processed Batches

Account page (left menu)
├── Account (summary + activity + policies)
├── Installment Schedule
├── Account Settings → Select Billing Account Options
├── Payment Processing → Make A Payment · Suspense Allocation · Payments List
├── Items In Suspense
├── EFT Maintenance → Scheduled EFT Transactions
├── Hold Activity
├── Account Tasks
└── Billing Transactions
Account side panels: Account Ledger · Account Details · Payee (Change Payee) · Payor (Change Payor)

Policy Term Summary (opened from a term link)
└── Policy Term Actions: Send To Internal Collections · Transfer Policy Term · Transfer To Collections ·
    View/Edit Policy Term Settings · Change renewal plan · Hold Billing for Policy Term · Paid In Full Discount
Term side panels: Term Details · Term Ledger · Agent

Party (People & Places Details), opened from the Payee/Payor name or the "360" link
└── Name Address & Contact · Phones · Email · Locations · Demographics & Other Names ·
    License & Membership · People & Places Relationships
```

---

## 2. Core objects as the UI shows them

| Object | Where it shows | Key attributes |
|---|---|---|
| **Billing account** | Account page header, Account Details panel | Account number. Here it equals the policy number (CLAD3057414, SFLH3074412), so each policy gets its own account. Also: Target Due Days, Account Type (`Account`), Account Class (`SPD8`, `SPD9`, `SPDK`), Collection Method (`PA`), Disbursement Hold, NSF Counter, Currency (USD), Account Plan, Invoice Due Day |
| **Account ledger** | Account Ledger panel | Outstanding Balance, Current Amount Due, Past-Due Amount, In Suspense Amount, Pending Disbursements |
| **Payor / Payee** | Side panels | Each is a link to a Party, with address and email. "Payee and payor are the same party" is ticked by default when an account is created |
| **Policy term** | Policies/Arrangements grid, Policy Term Summary | Reference (`CLAD3057414 Dwelling`, `STXH3074413 Homeowners`), Term (eff–exp), Status, Current Due, Outstanding Balance, Potential Commission, Aggregation Reference |
| **Term details** | Term Details panel | Billing Frequency (`ANNUAL`), Master Company (`DCT`), Line of Business (`HO3`), Equity Date, Currency |
| **Term ledger** | Term Ledger panel | Outstanding, Current Due, Past-Due, Receivable Amount, Paid on Term, Written Off/Waived |
| **Receivable items** | Policy Term Activity | Transaction (`New Business`), Receivable (`Premium`, `Fee`), Receivable Sub Type (`MGA` $25, `EMPA` $2), Amount, Bill In Full, Billing Item Effective Dates |
| **Installment** | Installment Schedule | Install Date, Term, Due Date, Recv, Trans, Item Amt, Install Amt, Cash, Credit, Write-Off, ReDist, Balance |
| **Payment** | Payment Details, Payments List | Amount, Date Entered, Status (`Active`, `Reversed`), Method, Check # / Name on Check / Postmark Date, or Card Number (masked) / Name on Card, Reference # |
| **Allocation** | Payment Details › Allocations | Account #, Direction (policy + term), Transaction Date, Status, Allocation Type (`Scheduled Item`), Amount |
| **Suspense item** | Items In Suspense | Payment Source, Refund Source, Reference, Suspense Date, Suspense Reason (`Overpayment`), Amount, Status (`No Hold`) |
| **Disbursement** | Disburse Funds, reports | Direction, Available Amount, Disbursement Group (`Org Unit 9999 Check`), Amount, Type (`Regular`), Reason. Goes through authorization |
| **Hold** | Hold Activity, Manage Hold Events | Individual or area event; Hold Type, Hold Reason, Start, End (blank = "Indefinite") |
| **Task** | Account Tasks | Title, Account Number, Task Type (`Billing Account`), Due Date, Description, Queue, Assignee, Status |
| **Party** | People & Places | Party Number, type (Person / Organization), names, addresses (Legal, Legacy Client), phones, emails, locations, demographics, other names, licenses, memberships. Every row is effective-dated, open-ended to 12/31/9999 |

### Account activity types seen

`Account Created` · `Policy Term Issued` · `Policy Transaction` ("New Business / Renewal (Policy … ): receivable has been processed") · `Policy Renewed` · `Policy Renewal Configuration Change` (by `SFP Admin` / `Express Admin`) · `Payment Received` · `Reverse Payment` · `Disbursement Pending Authorization` · `Disbursement Authorized` (by `System`) · `Set Hold`.

`Policy Renewal Configuration Change` entries show the payment plans being renamed at renewal: *"Renewal Payment Plan was changed from Full Pay Premium Finance to Full Pay"* (#063) and *"…from Semi-Annual (55% Down) to Full Pay"* (#089).

---

## 3. Workflows

### 3.1 Find an account (#001, #052, #030)
Use **Home › Find a Billing Account** (Find account by `Account Number` + value) or the header quick search. **Recently Accessed** links also open accounts. All three land on the **Account** page.

### 3.2 Search for a payment (#002–#005)
**Search For A Payment** says "Enter at least one piece of information". Fields: Account #, Policy/Arrangement #, Date (Range | Exact), Amount (Range | Exact), Payment Method, Currency, Other Info (first or last name, check number, card number). The **Search** button appears only after you enter a value.
The result opens **Payment Details**: a header (Amount, Date Entered, Status, Method, Check #, Name on Check, Postmark Date) and an **Allocations** grid. Example: a $722 check was split over 7 *Scheduled Items* on CLAD3057412 ($25, $30, $1, $1, $1, $60, $604), which is the fees plus the premium installment.

### 3.3 Enter a single payment, known account (#006–#012)
1. **Enter A Single Payment** › Payment Reference: *"I know the account, policy, arrangement or name"* › Search by `Account Number` › **Account Search**.
2. The results show Account Number, Payor Name and Address, with a **Payment Processing** link.
3. **Make A Payment**:
   - Payment Details: Payment Method, Entry Date (defaults to the system date), and for checks Check Number, Name On Check and Postmark Date.
   - Payment Allocation radio:
     - **Pay To Account** (default): option *Outstanding Balance $X* with Payment Amount pre-filled.
     - **Pay Specific Policies / Arrangements**: grid with Allocation Amt (editable), Reference (policy, plus an `Account Level` row), Effective, Expiration, Current Due, Past Due, Outstanding Balance and Suspense State, then Total Allocated / Total Remaining. Note on screen: *"Any remaining balance will stay on hold in suspense at the account level."*
   - **Submit Payment** or **Exit Payment Processing**.
4. Confirmation on Home: *"Payment for $981.00 entered for Account CLAD3057414."* (#011)

### 3.4 Unidentified payments (#013–#017)
- **Entry**: choose *"This is an unidentified payment, not associated with an account, policy or arrangement"*. Fields: Last Name, First, Middle, Account Reference, Policy/Arrangement Reference, Other Information. Each carries a required marker, but the rule is "enter at least one".
- **List** (Unidentified Payments): filters for Entry Date After/Before/Equal, Amount Greater/Less/Equal, Currency and Payment Method checkboxes. Grid: Entry Date, User Id, Amount, Currency, Payment Method, Identifier Type (`Account`, `Last`), Identifier, **Details**.
- **Detail**: three choices:
  1. **Apply** it: *I know the account…* › Search for Account by › Enter Number › Search.
  2. **Update Reference**: keep it unidentified but change Last/First/Middle, Account reference, Term reference or Other information.
  3. **Reverse This Payment**: reason (`RP-Reverse Payment`) › **Reverse Payment**.

  The detail screen also has a Payment Details strip: Amount, Date Entered, Status, Method, masked Card Number, Name on Card.

  The list contains card payments whose identifier is a valid account number (STXD3081495, CALD3057363, …). These are payments that came in but did not post automatically.

### 3.5 Card payment through the Heartland / Global Payments portal (#066–#070)
1. Make A Payment with Method = `Credit/Debit Card`. There are no check fields. Enter the amount and click **Submit Payment**.
2. An embedded **globalpayments** modal opens:
   - **Payment Information**: Card Number (Visa, MC, Discover, Amex), Name on Card, Month, Year, Security Code. All required.
   - **Card Holder Information**, pre-filled from the payor: First, Middle, Last, Address, City, State, Postal Code, Country, Email, Phone, Mobile.
   - Summary: **Amount $10.00 + Processing Fee $0.27 = $10.27**. The fee is 2.7% and the customer pays it.
3. Terms dialog (**Decline / Accept**): liability for returned payments, a RECEIPT clause (emailed if an address is given) and a CONTACT clause. The CONTACT clause reads *"contact Safepoint Insurance QA"*, which is environment-specific wording.
4. **Payment Successful**: receipt emailed, Transaction Number (`10542530`), Base Amount, Processing Fee, Total; buttons **Email** and **Finish**.
5. Billing records only the **base amount** ($10.00, "Credit/Debit Card payment for $10.00 received"). The policy owed nothing, so the $10 went to **suspense** (#073).

### 3.6 Suspense, refund and authorization (#071, #074–#077, #082)
- **Suspense Allocation** (Payment Processing submenu): Select All, Calculate Total Suspense Selected, Total Selected/Remaining, then allocate to the account or to specific policies.
- **Items In Suspense**: grid (Payment Source `Credit/Debit Card`, Refund Source `Check`, Suspense Date, Reason `Overpayment`, Amount, Status `No Hold`). Actions: **Disburse · Hold For Premium · Place Hold · Release Hold · Transfer · Write Off**.
- **Disburse Funds**: payee (`TEST`, with *Select Another Payee*); grid Direction / Available / Disbursement Group `Org Unit 9999 Check` / Disbursement Amount (editable) / Remaining; Disbursement Type `Regular`; Disbursement Reason (required before **Refund** is enabled):
  `Agency request · Audit Premium · Cancellation · Commission · Customer request · Endorsement premium · Other · Overpayment`.
- Result: *"Disbursement for $10.00 was submitted for authorization. Payee: TEST"*. Ledger › Pending Disbursements = $10. Activity shows `Disbursement Pending Authorization` (user), then `Disbursement Authorized` by **System** about a minute later (#082). Under this threshold or role, approval is automatic.
- The refund source was **Check** even though the money came in by card. Refunds do not go back to the card.

### 3.7 Create a new account (#018–#027)
**Create New Account** (`Carrier_Billing_BillAccountSetup_Base`):
- Account Payor: **Search For An Existing Party** or **Create A New Party**. Checkbox *Payee and payor are the same party* (ticked).
- Billing Options: Bill Account Class (required), Target Due Days (required, default 1), *Set target due day to end of month*.

**Search for a Person or Place** (`Carrier_Party_US_3_0_0_0`):
- *I'm looking for a/an*: `Agency`, `Person` (others likely).
- *I want to search by*: `Name and Address`, `License Number`, `Phone Number` (XXX-XXX-XXXX), `Membership ID`, `Other`.
- Name and Address fields: Organization or Last Name* (Agency) or Last*/First/Middle (Person), Address, City, State*, Zip, *Use phonetic search*.
- Results: Name, Address, DOB, Entity Name, Party Type; actions **View · Select · 360**; **Revise Search · Add New · Cancel**.

### 3.8 Account page (#031, #053–#054)
- **Add A Note** button (top right).
- Summary tiles: **Most Recent Invoice** (Invoice Date, Docs, Amount, Due Date), **Last Payment Received** (Date, Amount link, Method), **Next Scheduled Invoice** (Invoice Date, Amount, Due Date).
- **Recent Account Activity** (Date, Activity Type, Created By, Amount, with a description line). Export, save, refresh and help icons; paged.
- **Policies / Arrangements Associated With This Account**, grouped by reference. The Term link opens the Policy Term Summary.

### 3.9 Policy term actions (#055–#061)
**Policy Term Summary**: Summary (Policy Number, Eff/Exp, Status, Aggregation Reference), Most Recent Invoice, **Next Installment** (Date, Amount, Due Date; invoice 1/4/2027 due 1/24/2027, so invoices go out about 20 days before the due date), Additional Data (Name/Value), Policy Term Activity, Other Terms Associated With This Policy (with **Potential Commission**, $189.53 on $1,805 premium, about 10.5%).

| Action | Screen | Behaviour seen |
|---|---|---|
| Transfer Policy Term | Transfer Policy From Account X: Search for Account by / Enter Number / Search | Moves a term to another billing account |
| Transfer To Collections | — | Blocked: *"The term outstanding balance: $0.00 is not enough to make a transfer to collections."* |
| Send To Internal Collections | not opened | — |
| View/Edit Policy Term Settings | **Term Plan Setup**: Payment Plan radio | Only `Premium Finance Annual` offered |
| Change renewal plan | **Renewal Plan Setup**: Renewal Payment Plan radio | Only `Premium Finance Annual` offered |
| Hold Billing for Policy Term | Hold Type, Hold Reason, Start = system date (read-only), End | **Place Hold** enabled once type and reason are chosen |
| Paid In Full Discount | Term, Currency, Potential Discount, Evaluation Period, Discount Amount*, Reason* (`Paid in Full Discount Request`) | *"Cannot be calculated as the Paid In Full configuration has not been established"*: no PIF configuration in this environment |

### 3.10 Installment schedule (#062)
Date-range filter and refresh. Rows expand (+). Example: one annual installment, Install 1/4/2027, Due 1/24/2027, $1,832 paid in full by cash, balance 0. Side panel has **Schedule History**.

### 3.11 Account settings (#064)
**Select Billing Account Options**: Account Plan (`Paper Invoices ($3 Flat Invoice Charge)`, `Paper Invoices (Mortgagee)`, `Paper Invoices (PremiumFinance)`), Invoice Due Day*, Save Changes.

### 3.12 Holds (#079–#083, #046)
- **Hold Activity For Account**: *This is an individual account hold* or *This hold is part of an area hold event*. The area option lists events to attach to (Event Name, Description, Type, Reason, Start, End).
- Individual: Hold Type (`Hold Follow Up`, …), Hold Reason (`Company request`, `Natural Disaster`, …), Start = system date, End optional.
- Result: banner *"Account hold set. Hold type: Hold Follow Up. Reason: Company request. Beginning: 1/1/2027 Ending: Indefinite"*, a persistent warning with an **Update** link, and a `Set Hold` activity.
- **Edit Hold Activity**: End date plus **Resume Options**: *Adjust due dates* or *Do not adjust due dates*. Buttons Save / **Resume** / Cancel.
- Admin › **Manage Hold Events**: Create Hold Event; Current Hold Events (Event Name, Description, Type, Reason, Start, End, **End** link, **Review** link). Example: `Test Hold / Hold Follow Up / Natural Disaster / 3/19/2025`, still open. Catastrophe holds are applied in bulk this way.

### 3.13 Tasks (#084–#087)
- **Add a Task** › New Task: Title*, Account Number (fixed), Task Type (`Billing Account`), Due Date, Description, Queue* (agency queues such as `All Phase Insurance 500284`), Assignee.
- **My Tasks** and **Tasks Available for Pickup** (unassigned tasks in a queue). Both grids: Queue, Category, Title, Status, Reference #, Created, Due Date, pickup icon. *More Options* filters: Category, State (`Open`), Search (`Policy/Quote #`) + For.

### 3.14 Billing transactions (#048, #088)
- Admin: search by Account Number.
- Account level: Transaction dropdown with one option, **Add Miscellaneous Billing Item**. The item form is not shown.

### 3.15 EFT maintenance (#078)
**Scheduled EFT Transactions** grid (Date, Activity Type, Created By, Amount) is empty. The document does not show how to add bank details.

### 3.16 Party maintenance (#089–#114)
Open it from the Payee/Payor link or "360". The module switcher changes to **Party**.

- Header: *"This party is a Person associated with Internal Users"*, Person Name, Correspondence Name, Preferred method of contact, Country, **Party Number** (95371).
- **Correct vs Replace**. This is the key rule for name, address and other names:
  - **Correct** fixes the current row in place. The effective date stays (9/23/2026).
  - **Replace** creates a new version effective from the system date (10/12/2026) and end-dates the old one.
  - Other Names also have **Remove**.
- Addresses: *Legal Addresses* and *Legacy Client Addresses* (the second set comes from the conversion). Add/Correct form: Country, Address*, Address 2, City*, State*, Zip*, +4, County, Comments, Effective/Expiration, Attention, **Associations** (*"This is the Legal Address as of …"*, *Update associations*).
- Phones: Country, Phone Type*, Number*, Ext, Primary, Contact Hours From/To, Comments.
- Email: Email Type*, Email Address*, Comments, Effective, Primary E-Mail, Edit.
- Locations (physical): same fields as an address, "Physical Address Effective beginning …".
- Demographics: Preferred Language, Gender, Ethnic Origin, DOB, Marital Status, Occupation, SSN, US Citizen? (default Yes), Other ID, Date of Death, Number of Dependents, Profession, Company Personnel, OFAC Check Not Required (`Unanswered`), Party Comments.
- Other Names: Type of Name* (`Formerly Known As`), Prefix/First/Middle/Last*/Suffix, Comments, Effective. Test party has `ZZZLEGACY` and `SAFEPOINT` as FKA rows, which are conversion leftovers.
- Licenses: License Type*, Number*, Status, State, City, County, Effective, Expiration.
- Memberships: Membership Type* (`Club Id`, `Employee Id`, `Membership ID`, `Agency Id`), ID*.
- **Change Payee / Change Payor** (#115–#117) run the party search (Person · Name and Address), then **Select** on the result.

### 3.17 Batch cash management (#049–#051)
- **Batches Awaiting Deposit**: Batch Number (link), Entry Date, User Id, Status (`Processing`, `Processing Error`), Control Total, Batch Total, Currency; description *"BHHC LockBox Transmission <timestamp>"*. These are bank lockbox files.
- **Start A New Payment Batch**: Batch Number, Control Total, Currency, Default Payment Method (`Check`), Processing Org. Unit, Description. **Create** stays disabled until the fields are filled.
- **Processed Batches**: search by Batch Number, User Id, Date Deposited, Date Created; **Display All Items**; grid with **Reverse**.

### 3.18 Administration (#042–#048)
- **Error Queue**: filter Start/End Date, Type, Status; grid Date, Type, Reference, Status, **View Details**, **Resubmit**.
  - Types seen: `Generate Invoice` (*"Error generating the invoice."*), `Automatic Pending Cancellation Process` (*"Error processing an automatic policy term pending cancellation request."*), `Receivable`.
  - **Error Details**: Error Description (here *"Object reference not set to an instance of an object."*) and the raw **XML payload** (`<BillInvoice.generateRq invoiceID="872444" asyncID="…">…<AccountReference>SFLH3070506</AccountReference>`).
  - **Resubmission Details**: editable payload sections (`_BillingInvoice`, `_GenerateInvoice` with `<AccountId>302299</AccountId><DueDateSet>…`), invoiceID, asyncID, **Submit**. Background work runs as asynchronous queue requests that can be retried.
- **Disbursement Interface Service**: **Start Service / Stop Service**. Warning on screen: Start adds a new Async Queue request, and if one is already queued you get duplicates and the interface runs more often than expected.
- **Manage Hold Events**: see §3.12.

---

## 4. Code lists observed

| List | Values |
|---|---|
| Payment methods (filter) | Auto-Pay Bank, Auto-Pay Credit Card, Auto-Pay Debit Card, Cash, Check, Credit Card, Debit Card, eCheck, Expense, Pre-Authorized Card. Grids show `Credit/Debit Card`, `Check` (`CK` in reports) |
| Payment status | Active, Reversed |
| Reverse reason | RP-Reverse Payment |
| Allocation type | Scheduled Item |
| Receivable / sub type | Premium; Fee: MGA, EMPA |
| Suspense actions | Disburse, Hold For Premium, Place Hold, Release Hold, Transfer, Write Off |
| Suspense reason / status | Overpayment / No Hold |
| Disbursement type | Regular |
| Disbursement reason | Agency request, Audit Premium, Cancellation, Commission, Customer request, Endorsement premium, Other, Overpayment |
| Disbursement group | Org Unit 9999 Check |
| Hold types / reasons | Hold Follow Up / Company request, Natural Disaster |
| Hold resume option | Adjust due dates, Do not adjust due dates |
| Account plans | Paper Invoices ($3 Flat Invoice Charge), Paper Invoices (Mortgagee), Paper Invoices (PremiumFinance) |
| Payment plans | Premium Finance Annual; in activity text: Full Pay, Full Pay Premium Finance, Semi-Annual (55% Down) |
| Account classes | SPD8, SPD9, SPDK |
| Collection method | PA |
| Billing frequency | ANNUAL |
| Task type / category / state | Billing Account / Billing Account / Open |
| Party search target | Agency, Person |
| Party search by | Name and Address, License Number, Phone Number, Membership ID, Other |
| Membership types | Club Id, Employee Id, Membership ID, Agency Id |
| Other-name type | Formerly Known As |
| Report export formats | Word, Excel, PowerPoint, PDF, TIFF, MHTML, CSV, XML file with report data, Data Feed |
| Error types | Generate Invoice, Automatic Pending Cancellation Process, Receivable |
| Batch status | Processing, Processing Error |

---

## 5. Reports (Home › Reports / Reporting Home)

Reporting Home has **Report History**, **Export** (formats above) and **Reports** dropdowns. Report headers show *Company Name: Top Insurance Company* (a Duck Creek sample value, not SafePoint), *Processing Branch: Cajun UnderWriters* and *Currency*.

| Report | Columns | What the test data shows |
|---|---|---|
| Aged Receivables | Account, Policy, +30, +60, +90, +180 Days, Total | Mostly +180-day balances (e.g. CALH3047821 $4,275) |
| Agency Open Items | — | Empty ("Open Items", page 1 of 1) |
| Cash In Suspense | Account Reference, Allocation Date, Days Pending, Amount (subtotal per account) | Acct 0000007365: **$237,497.63 pending 983 days** |
| Commission Activity | Agent Number, Bill Account, Policy Number, Premium Amt, Pay Option, Commission Amt | "No Commission Activity" |
| Daily Cash Journal | per Batch (Batch N/A = Operator Entered): Account, Entry User Id, Payment Method, Amount, Batch Total; summary by method | Today's two test checks: 722 + 981 = 1,703 |
| Disbursement Activity | Account, Request Date, Method, Method Detail, User, Description, Amount, Reason, Payee Changed | None |
| Disbursements Pending Authorization | same as above | Many check refunds from **5/21/2025** by SFP Admin up to $242,914; one **Commission $242,755.34 from 12/6/2022** |
| Payment Batch Activity | Batch, Batch Total, Batch Status, Description, Bill Account, Policy Ref, Payment Method, Amount, Detail Status | None |
| Unidentified Cash | Account, Policy, Other Identifier, Entry Date, Name, Amount | **CLAH3048610 1/14/2026 $8,504.00 repeated many times** |

---

## 6. Business rules shown in the screens

1. One billing account per policy. The account number equals the policy number.
2. Overpayments and unallocated amounts go to account-level suspense automatically.
3. A refund (disbursement) from suspense needs a reason and goes through authorization. Small amounts are approved by `System` straight away.
4. Refunds go out by Check, whatever the original payment method was.
5. Card payments go through Global Payments (Heartland) with a 2.7% convenience fee. Only the base amount posts to billing.
6. Transfer to collections needs a positive outstanding balance on the term.
7. Holds start on the system date and cannot be back-dated. A blank end date means indefinite. On resume you choose whether due dates move.
8. Party data is effective-dated. Correct overwrites the current row; Replace versions it from the system date.
9. Unidentified payments need at least one identifier and can be applied, re-referenced or reversed.
10. Background jobs (invoice generation, pending cancellation, receivables, disbursement interface) run as async queue requests. Failures land in the Error Queue with their XML payload and can be resubmitted after editing.
11. Primary action buttons (Create, Place Hold, Refund, Import Statements, Search) stay disabled until the required inputs are filled.

---

## 7. Problems visible in the screenshots

These matter for any migration, rebuild or reconciliation. They are what the screenshots show, not checked further.

| # | Observation | Where | Why it matters |
|---|---|---|---|
| O1 | Error-queue items from **4/26/2024** still `Open` (invoice generation, pending-cancellation, receivable) | #043 | Stale failures; some invoices and cancellations never ran |
| O2 | Lockbox batches from **2018** still in `Processing` / `Processing Error`, with Control Total $0.00 | #049 | Cash never deposited or posted; Control Total not used |
| O3 | Large disbursements pending authorization since 2022 and 2025 (up to ~$243K), including a $242,755 commission | #039 | Unapproved refunds and commissions; must be settled before cut-over |
| O4 | Cash in suspense up to **983 days** old ($237K on one account) | #035 | Unresolved suspense balances to carry over or clean up |
| O5 | The same unidentified-cash row repeated many times (CLAH3048610, $8,504, 1/14/2026) | #041 | Duplicate postings or a reporting defect |
| O6 | Unidentified card payments whose identifier is a valid account number | #015 | Auto-matching gap in the payment intake |
| O7 | Paid In Full configuration "has not been established" | #061 | PIF discount cannot be calculated |
| O8 | Term and renewal plan screens offer only `Premium Finance Annual`, but activity shows Full Pay / Semi-Annual plans | #058–#059, #063, #089 | Plan options depend on configuration or state; check before relying on this screen |
| O9 | Conversion leftovers: "Legacy Client Addresses", FKA name `ZZZLEGACY` | #090, #107 | Earlier migration artifacts on party records |
| O10 | Report header "Top Insurance Company"; card terms say "Safepoint Insurance QA" | #032, #069 | Environment and branding configuration not finished |
| O11 | Error-queue failure is a .NET null reference ("Object reference not set…") | #044 | Product or config defect, not a data validation message |

---

## 8. What the document does not show

To capture in a follow-up walkthrough if billing is in scope:

- Invoice documents (the *Docs* column), statement or invoice generation, and **Schedule History**.
- EFT/ACH enrolment (adding bank details), recurring card autopay.
- NSF / returned-payment handling (NSF Counter exists but is never exercised).
- Cancellation for non-payment, reinstatement, write-off, and the **Write Off / Transfer / Hold For Premium** suspense actions.
- **Add Miscellaneous Billing Item** form, **Send To Internal Collections**, collections agency flow.
- Batch entry detail (adding payments to a batch, deposit, reverse).
- Area **Create Hold Event** form, Notes (*Add A Note*), Notifications detail, **360** view, People & Places Relationships.
- Agency-bill flow: Import Statements was empty, Agency Open Items and Commission Activity had no data.
- Disbursement authorization screen (who approves, thresholds).
