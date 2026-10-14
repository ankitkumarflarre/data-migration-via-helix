// Package billing is a durable local copy of the Duck Creek Billing screens:
// accounts, policy terms, payments, suspense, disbursements, holds, tasks,
// parties, batches, the error queue and the reports. Money is integer cents
// and dates are YYYY-MM-DD strings. Nothing is sent to Helix or a gateway.
package billing

// CurrentUser is the person working in the local workspace. System is the
// user that automatic steps (auto-authorization, due processes) run as.
const (
	CurrentUser = "Local User"
	SystemUser  = "System"
	OpenEnded   = "9999-12-31"
)

// State is everything the billing workspace stores. It is saved as one file.
type State struct {
	SystemDate    string                   `json:"system_date"`
	Version       int                      `json:"version"` // version of the system date and settings record
	Settings      Settings                 `json:"settings"`
	Seq           map[string]int           `json:"seq"`
	Accounts      map[string]*Account      `json:"accounts"`
	Terms         map[string]*Term         `json:"terms"`
	Parties       map[string]*Party        `json:"parties"`
	Payments      map[string]*Payment      `json:"payments"`
	Suspense      map[string]*SuspenseItem `json:"suspense"`
	Disbursements map[string]*Disbursement `json:"disbursements"`
	Holds         map[string]*Hold         `json:"holds"`
	HoldEvents    map[string]*HoldEvent    `json:"hold_events"`
	Tasks         map[string]*Task         `json:"tasks"`
	Batches       map[string]*Batch        `json:"batches"`
	Errors        map[string]*ErrorItem    `json:"errors"`
	Invoices      map[string]*Invoice      `json:"invoices"`
	Statements    map[string]*Statement    `json:"statements"`
	Activities    []*Activity              `json:"activities"`
	Notifications []*Notification          `json:"notifications"`
	Recent        []string                 `json:"recent"`
}

// Settings are the workspace switches the screens depend on.
type Settings struct {
	AutoAuthorizeLimit  int64        `json:"auto_authorize_limit"` // disbursements at or under this are authorized by System
	PIF                 *PIFConfig   `json:"pif,omitempty"`        // nil: "configuration has not been established"
	OfferedPlans        []string     `json:"offered_plans"`        // plans Term Plan Setup and Renewal Plan Setup offer
	DisbursementService ServiceState `json:"disbursement_service"`
}

type PIFConfig struct {
	DiscountBasisPoints int64 `json:"discount_basis_points"` // 100 = 1% of the term premium
	EvaluationDays      int   `json:"evaluation_days"`       // days after the effective date the term must be paid in full
}

// ServiceState is the Disbursement Interface Service. Queued is the pending
// async request; a second Start Service while one is queued is refused.
type ServiceState struct {
	Status   string `json:"status"` // Running | Stopped
	Queued   bool   `json:"queued"`
	QueuedAt string `json:"queued_at,omitempty"`
	LastRun  string `json:"last_run,omitempty"`
}

type Account struct {
	Number           string `json:"number"`
	Version          int    `json:"version"`
	Status           string `json:"status"`
	PayorID          string `json:"payor_id"`
	PayeeID          string `json:"payee_id"`
	TargetDueDays    int    `json:"target_due_days"`
	EndOfMonth       bool   `json:"end_of_month"`
	AccountType      string `json:"account_type"`
	AccountClass     string `json:"account_class"`
	CollectionMethod string `json:"collection_method"`
	DisbursementHold string `json:"disbursement_hold"`
	NSFCounter       int    `json:"nsf_counter"`
	Currency         string `json:"currency"`
	AccountPlan      string `json:"account_plan"`
	InvoiceDueDay    int    `json:"invoice_due_day,omitempty"`
	CreatedOn        string `json:"created_on"`
	Source           string `json:"source,omitempty"` // quote:<id> when created from a quote
}

type Agent struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Term struct {
	ID                  string         `json:"id"`
	AccountNumber       string         `json:"account_number"`
	PolicyNumber        string         `json:"policy_number"`
	Product             string         `json:"product"` // Dwelling, Homeowners
	Effective           string         `json:"effective"`
	Expiration          string         `json:"expiration"`
	Status              string         `json:"status"`
	PaymentPlan         string         `json:"payment_plan"`
	RenewalPlan         string         `json:"renewal_plan"`
	BillingFrequency    string         `json:"billing_frequency"`
	MasterCompany       string         `json:"master_company"`
	LineOfBusiness      string         `json:"line_of_business"`
	EquityDate          string         `json:"equity_date"`
	AggregationRef      string         `json:"aggregation_reference"`
	PotentialCommission int64          `json:"potential_commission"`
	Agent               *Agent         `json:"agent,omitempty"`
	AdditionalData      []NameValue    `json:"additional_data"`
	Receivables         []*Receivable  `json:"receivables"`
	Installments        []*Installment `json:"installments"`
	History             []ScheduleNote `json:"history"`
}

type ScheduleNote struct {
	Date string `json:"date"`
	Text string `json:"text"`
}

type Receivable struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	Transaction string `json:"transaction"` // New Business, Renewal
	Receivable  string `json:"receivable"`  // Premium, Fee
	SubType     string `json:"sub_type"`    // MGA, EMPA
	Amount      int64  `json:"amount"`
	BillInFull  bool   `json:"bill_in_full"`
	Effective   string `json:"effective"`
	Expiration  string `json:"expiration"`
}

type Installment struct {
	ID          string         `json:"id"`
	InstallDate string         `json:"install_date"` // the invoice date
	DueDate     string         `json:"due_date"`
	InvoiceID   string         `json:"invoice_id,omitempty"`
	PastDue     bool           `json:"past_due,omitempty"` // the due process has flagged it
	Items       []*InstallItem `json:"items"`
}

type InstallItem struct {
	ID           string `json:"id"`
	ReceivableID string `json:"receivable_id"`
	Recv         string `json:"recv"`  // PREM, FEE
	Trans        string `json:"trans"` // NBUS, RENW
	Amount       int64  `json:"amount"`
	Cash         int64  `json:"cash"`
	Credit       int64  `json:"credit"`
	WriteOff     int64  `json:"write_off"`
	Redist       int64  `json:"redist"`
}

func (i *InstallItem) Balance() int64 { return i.Amount - i.Cash - i.Credit - i.WriteOff }

func (in *Installment) Balance() (b int64) {
	for _, it := range in.Items {
		b += it.Balance()
	}
	return
}

func (in *Installment) Total() (t int64) {
	for _, it := range in.Items {
		t += it.Amount
	}
	return
}

type Identity struct {
	Last       string `json:"last"`
	First      string `json:"first"`
	Middle     string `json:"middle"`
	AccountRef string `json:"account_ref"`
	PolicyRef  string `json:"policy_ref"`
	Other      string `json:"other"`
}

type Payment struct {
	ID                string        `json:"id"`
	Version           int           `json:"version"`
	AccountNumber     string        `json:"account_number,omitempty"`
	Amount            int64         `json:"amount"`
	Currency          string        `json:"currency"`
	Method            string        `json:"method"`
	EntryDate         string        `json:"entry_date"`
	PostmarkDate      string        `json:"postmark_date,omitempty"`
	CheckNumber       string        `json:"check_number,omitempty"`
	NameOnCheck       string        `json:"name_on_check,omitempty"`
	CardBrand         string        `json:"card_brand,omitempty"`
	CardLast4         string        `json:"card_last4,omitempty"`
	NameOnCard        string        `json:"name_on_card,omitempty"`
	Fee               int64         `json:"fee,omitempty"`
	TransactionNumber string        `json:"transaction_number,omitempty"`
	Reference         string        `json:"reference,omitempty"`
	Status            string        `json:"status"`
	CreatedBy         string        `json:"created_by"`
	CreatedAt         string        `json:"created_at"`
	BatchNumber       string        `json:"batch_number,omitempty"`
	Unidentified      bool          `json:"unidentified,omitempty"`
	Identity          *Identity     `json:"identity,omitempty"`
	AppliedOn         string        `json:"applied_on,omitempty"`
	ReverseReason     string        `json:"reverse_reason,omitempty"`
	ReversedOn        string        `json:"reversed_on,omitempty"`
	Allocations       []*Allocation `json:"allocations"`
}

// Allocation records where part of a payment went: a scheduled item or account
// suspense. Amount is what is still allocated; reversal re-opens it.
type Allocation struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	TermID        string `json:"term_id,omitempty"`
	Direction     string `json:"direction"`
	Date          string `json:"date"`
	Status        string `json:"status"`
	Type          string `json:"type"` // Scheduled Item, Suspense
	Amount        int64  `json:"amount"`
	InstallmentID string `json:"installment_id,omitempty"`
	ItemID        string `json:"item_id,omitempty"`
	SuspenseID    string `json:"suspense_id,omitempty"`
}

type SuspenseItem struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	PaymentID     string `json:"payment_id,omitempty"`
	PaymentSource string `json:"payment_source"`
	RefundSource  string `json:"refund_source"`
	Reference     string `json:"reference"`
	Date          string `json:"date"`
	Reason        string `json:"reason"`
	Original      int64  `json:"original"`
	Amount        int64  `json:"amount"` // still in suspense
	Status        string `json:"status"` // No Hold, Hold
}

type DisbursementSource struct {
	SuspenseID string `json:"suspense_id"`
	Amount     int64  `json:"amount"`
}

type Disbursement struct {
	ID            string               `json:"id"`
	AccountNumber string               `json:"account_number"`
	PayeeName     string               `json:"payee_name"`
	PayeePartyID  string               `json:"payee_party_id,omitempty"`
	Amount        int64                `json:"amount"`
	Type          string               `json:"type"`
	Reason        string               `json:"reason"`
	Group         string               `json:"group"`
	Method        string               `json:"method"`
	MethodDetail  string               `json:"method_detail"`
	Description   string               `json:"description"`
	Status        string               `json:"status"` // Pending Authorization, Authorized, Issued, Rejected
	RequestDate   string               `json:"request_date"`
	User          string               `json:"user"`
	AuthorizedBy  string               `json:"authorized_by,omitempty"`
	AuthorizedOn  string               `json:"authorized_on,omitempty"`
	IssuedOn      string               `json:"issued_on,omitempty"`
	PayeeChanged  bool                 `json:"payee_changed"`
	Sources       []DisbursementSource `json:"sources"`
}

type Hold struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	TermID        string `json:"term_id,omitempty"`
	EventID       string `json:"event_id,omitempty"`
	Type          string `json:"type"`
	Reason        string `json:"reason"`
	Start         string `json:"start"`
	End           string `json:"end,omitempty"` // blank: Indefinite
	Status        string `json:"status"`        // Active, Released
	ReleasedOn    string `json:"released_on,omitempty"`
	ResumeOption  string `json:"resume_option,omitempty"`
	AdjustedDays  int    `json:"adjusted_days,omitempty"`
}

type HoldEvent struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Reason      string `json:"reason"`
	Start       string `json:"start"`
	End         string `json:"end,omitempty"`
}

type Task struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	AccountNumber string `json:"account_number"`
	Type          string `json:"type"`
	Category      string `json:"category"`
	DueDate       string `json:"due_date"`
	Description   string `json:"description"`
	Queue         string `json:"queue"`
	Assignee      string `json:"assignee,omitempty"`
	Status        string `json:"status"`
	Created       string `json:"created"`
}

type Batch struct {
	Number        string `json:"number"`
	Version       int    `json:"version"`
	EntryDate     string `json:"entry_date"`
	CreatedAt     string `json:"created_at"`
	User          string `json:"user"`
	Status        string `json:"status"` // Open, Processing, Processing Error, Deposited, Reversed
	ControlTotal  int64  `json:"control_total"`
	BatchTotal    int64  `json:"batch_total"`
	Currency      string `json:"currency"`
	DefaultMethod string `json:"default_method"`
	OrgUnit       string `json:"org_unit"`
	Description   string `json:"description"`
	DepositedOn   string `json:"deposited_on,omitempty"`
	ReversedOn    string `json:"reversed_on,omitempty"`
}

func (b *Batch) Processed() bool { return b.Status == "Deposited" || b.Status == "Reversed" }

type PayloadSection struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ErrorItem is a failed async queue request. It keeps its payload so it can be
// corrected and resubmitted.
type ErrorItem struct {
	ID          string           `json:"id"`
	Version     int              `json:"version"`
	At          string           `json:"at"`
	Type        string           `json:"type"`
	Reference   string           `json:"reference"`
	Status      string           `json:"status"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	Payload     string           `json:"payload"`
	Sections    []PayloadSection `json:"sections"`
	InvoiceID   string           `json:"invoice_id"`
	AsyncID     string           `json:"async_id"`
	Attempts    int              `json:"attempts"`
	ResolvedAt  string           `json:"resolved_at,omitempty"`
}

type Invoice struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	TermID        string `json:"term_id"`
	InstallmentID string `json:"installment_id"`
	InvoiceDate   string `json:"invoice_date"`
	DueDate       string `json:"due_date"`
	Amount        int64  `json:"amount"`
}

type Statement struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Imported string `json:"imported,omitempty"`
}

type Activity struct {
	ID            string `json:"id"`
	AccountNumber string `json:"account_number"`
	At            string `json:"at"` // YYYY-MM-DDTHH:MM
	Type          string `json:"type"`
	CreatedBy     string `json:"created_by"`
	Amount        *int64 `json:"amount"` // nil shows N/A
	Text          string `json:"text"`
	PaymentID     string `json:"payment_id,omitempty"`
	TermID        string `json:"term_id,omitempty"`
}

type Notification struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Received string `json:"received"`
}

// Party is a person or organization. Every row is effective-dated: a row is
// current from Effective (inclusive) until Expiration (exclusive);
// 9999-12-31 means open-ended.
type Party struct {
	ID             string                 `json:"id"`
	Number         int                    `json:"number"`
	Version        int                    `json:"version"`
	Kind           string                 `json:"kind"` // Person, Organization
	AssociatedWith string                 `json:"associated_with"`
	Rows           map[string][]*PartyRow `json:"rows"`
	Demographics   map[string]string      `json:"demographics"`
}

type PartyRow struct {
	ID         string            `json:"id"`
	Effective  string            `json:"effective"`
	Expiration string            `json:"expiration"`
	Fields     map[string]string `json:"fields"`
}
