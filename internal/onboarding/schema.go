package onboarding

type Condition struct {
	Condition string      `json:"condition"`
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
}

type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon,omitempty"`
	Sub   string `json:"sub,omitempty"`
}

type SchemaNode struct {
	ID         string     `json:"id"`
	Screen     string     `json:"screen"`
	Code       string     `json:"code"`
	Type       string     `json:"type"`
	Label      string     `json:"label"`
	Hint       string     `json:"hint,omitempty"`
	Required   bool       `json:"required,omitempty"`
	Group      string     `json:"group,omitempty"`
	RequiredIf *Condition `json:"required_if,omitempty"`
	ShowIf     *Condition `json:"show_if,omitempty"`
	Options    []Option   `json:"options,omitempty"`
}

func eq(field string, value interface{}) *Condition {
	return &Condition{Condition: "==", Field: field, Value: value}
}
func neq(field string, value interface{}) *Condition {
	return &Condition{Condition: "!=", Field: field, Value: value}
}
func has(field string, value interface{}) *Condition {
	return &Condition{Condition: "contains", Field: field, Value: value}
}

var nigerianStates = []Option{
	{ID: "abia", Label: "Abia"},
	{ID: "adamawa", Label: "Adamawa"},
	{ID: "akwa_ibom", Label: "Akwa Ibom"},
	{ID: "anambra", Label: "Anambra"},
	{ID: "bauchi", Label: "Bauchi"},
	{ID: "bayelsa", Label: "Bayelsa"},
	{ID: "benue", Label: "Benue"},
	{ID: "borno", Label: "Borno"},
	{ID: "cross_river", Label: "Cross River"},
	{ID: "delta", Label: "Delta"},
	{ID: "ebonyi", Label: "Ebonyi"},
	{ID: "edo", Label: "Edo"},
	{ID: "ekiti", Label: "Ekiti"},
	{ID: "enugu", Label: "Enugu"},
	{ID: "fct_abuja", Label: "FCT — Abuja"},
	{ID: "gombe", Label: "Gombe"},
	{ID: "imo", Label: "Imo"},
	{ID: "jigawa", Label: "Jigawa"},
	{ID: "kaduna", Label: "Kaduna"},
	{ID: "kano", Label: "Kano"},
	{ID: "katsina", Label: "Katsina"},
	{ID: "kebbi", Label: "Kebbi"},
	{ID: "kogi", Label: "Kogi"},
	{ID: "kwara", Label: "Kwara"},
	{ID: "lagos", Label: "Lagos"},
	{ID: "nasarawa", Label: "Nasarawa"},
	{ID: "niger", Label: "Niger"},
	{ID: "ogun", Label: "Ogun"},
	{ID: "ondo", Label: "Ondo"},
	{ID: "osun", Label: "Osun"},
	{ID: "oyo", Label: "Oyo"},
	{ID: "plateau", Label: "Plateau"},
	{ID: "rivers", Label: "Rivers"},
	{ID: "sokoto", Label: "Sokoto"},
	{ID: "taraba", Label: "Taraba"},
	{ID: "yobe", Label: "Yobe"},
	{ID: "zamfara", Label: "Zamfara"},
}

var countries = []Option{
	{ID: "nigeria", Label: "Nigeria"},
	{ID: "ghana", Label: "Ghana"},
	{ID: "kenya", Label: "Kenya"},
	{ID: "south_africa", Label: "South Africa"},
	{ID: "cameroon", Label: "Cameroon"},
	{ID: "benin", Label: "Benin Republic"},
	{ID: "togo", Label: "Togo"},
	{ID: "ivory_coast", Label: "Côte d'Ivoire"},
	{ID: "senegal", Label: "Senegal"},
	{ID: "ethiopia", Label: "Ethiopia"},
	{ID: "tanzania", Label: "Tanzania"},
	{ID: "rwanda", Label: "Rwanda"},
	{ID: "egypt", Label: "Egypt"},
	{ID: "uk", Label: "United Kingdom"},
	{ID: "usa", Label: "United States"},
	{ID: "canada", Label: "Canada"},
	{ID: "germany", Label: "Germany"},
	{ID: "france", Label: "France"},
	{ID: "netherlands", Label: "Netherlands"},
	{ID: "uae", Label: "United Arab Emirates"},
	{ID: "saudi_arabia", Label: "Saudi Arabia"},
	{ID: "china", Label: "China"},
	{ID: "india", Label: "India"},
	{ID: "australia", Label: "Australia"},
	{ID: "other", Label: "Other"},
}

// GetSchema returns the full NTA 2025 individual onboarding question corpus.
// 210 questions across 31 screens (00–30). The typical salaried employee sees
// roughly 25 screens; conditional branching suppresses the rest automatically.
func GetSchema() []SchemaNode {
	return []SchemaNode{

		// ── SCREEN 00 — Start ──────────────────────────────────────────────────

		{ID: "Q000", Screen: "00", Code: "sentiment", Type: "sentiment_picker",
			Label: "First, how are you feeling about filing your taxes today?", Required: true,
			Options: []Option{
				{ID: "confident", Label: "Confident", Icon: "sentiment_satisfied"},
				{ID: "neutral", Label: "Okay", Icon: "sentiment_neutral"},
				{ID: "anxious", Label: "Anxious", Icon: "sentiment_dissatisfied"},
			}},

		{ID: "Q001", Screen: "00", Code: "intention", Type: "single_select",
			Label: "What would you like to do?", Required: true,
			Options: []Option{
				{ID: "file_return", Label: "File my individual tax return"},
				{ID: "calculate_pit", Label: "Calculate my personal income tax"},
				{ID: "check_payment", Label: "Check whether I've paid enough tax"},
				{ID: "calculate_paye", Label: "Calculate my PAYE"},
				{ID: "register_tax_id", Label: "Register/get my Tax ID"},
				{ID: "check_compliance", Label: "Check my tax compliance"},
				{ID: "help_previous", Label: "Get help with a previous tax return"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q002", Screen: "00", Code: "tax_year", Type: "single_select",
			Label: "Which tax year are you filing for?", Required: true,
			Options: []Option{
				{ID: "2026", Label: "2026"},
				{ID: "2025", Label: "2025"},
				{ID: "2024", Label: "2024"},
				{ID: "other", Label: "Other year"},
			}},

		{ID: "Q003", Screen: "00", Code: "filing_type", Type: "single_select",
			Label: "Are you filing an individual return?", Required: true,
			Options: []Option{
				{ID: "individual", Label: "Yes — I'm an individual"},
				{ID: "company", Label: "No — I'm filing for a company"},
				{ID: "partnership", Label: "No — I'm filing for a partnership, trust or estate"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q004", Screen: "00", Code: "first_time_user", Type: "boolean",
			Label: "Is this your first time using this service?", Required: true},

		// ── SCREEN 01 — Your identity ──────────────────────────────────────────

		{ID: "Q005", Screen: "01", Code: "first_name", Type: "text",
			Label: "What is your legal first name?", Group: "Personal details", Required: true},

		{ID: "Q006", Screen: "01", Code: "middle_name", Type: "text",
			Label: "What is your middle name?", Group: "Personal details"},

		{ID: "Q007", Screen: "01", Code: "last_name", Type: "text",
			Label: "What is your surname?", Group: "Personal details", Required: true},

		{ID: "Q008", Screen: "01", Code: "preferred_name", Type: "text",
			Label: "What name do you normally go by?", Group: "Personal details"},

		{ID: "Q009", Screen: "01", Code: "dob", Type: "date",
			Label: "What is your date of birth?", Group: "Personal details", Required: true},

		{ID: "Q010", Screen: "01", Code: "gender", Type: "single_select",
			Label: "What is your gender?", Group: "Personal details",
			Options: []Option{
				{ID: "male", Label: "Male"},
				{ID: "female", Label: "Female"},
				{ID: "prefer_not_to_say", Label: "Prefer not to say"},
			}},

		{ID: "Q011", Screen: "01", Code: "phone", Type: "text",
			Label: "What is your phone number?", Group: "Contact information", Required: true},

		{ID: "Q012", Screen: "01", Code: "email", Type: "text",
			Label: "What email address should we use?", Group: "Contact information", Required: true},

		{ID: "Q013", Screen: "01", Code: "has_nin", Type: "single_select",
			Label: "Do you have a National Identification Number (NIN)?", Group: "National identity", Required: true,
			Hint: "Your 11-digit NIN is printed on your NIMC ID card or slip. If you have never registered, visit a NIMC centre or use the NIMC mobile app.",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "applied", Label: "I have applied"},
				{ID: "dont_know", Label: "I don't know"},
			}},

		{ID: "Q014", Screen: "01", Code: "nin_value", Type: "text",
			Label:      "Enter your NIN.",
			Group:      "National identity",
			Hint:       "Enter exactly 11 digits as shown on your NIMC card. No spaces, dashes or letters.",
			ShowIf:     eq("has_nin", "yes"),
			RequiredIf: eq("has_nin", "yes")},

		// ── SCREEN 02 — Tax identity ───────────────────────────────────────────

		{ID: "Q015", Screen: "02", Code: "has_tax_id", Type: "single_select",
			Label: "Do you already have a Nigerian Tax ID, {first_name}?", Required: true,
			Hint: "A Tax ID (formerly TIN) is different from your NIN. It is issued by the NRS and may appear on old FIRS letters, payslips, or previous tax documents.",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q016", Screen: "02", Code: "tax_id_type", Type: "single_select",
			Label:  "What type of Tax ID do you have?",
			ShowIf: eq("has_tax_id", "yes"),
			Options: []Option{
				{ID: "nrs_tax_id", Label: "NRS Tax ID"},
				{ID: "previous_firs", Label: "Previous FIRS / TaxPro Max-linked identifier"},
				{ID: "state_identifier", Label: "State tax identifier"},
				{ID: "other", Label: "Other"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q017", Screen: "02", Code: "tax_id_value", Type: "text",
			Label:  "Enter your Tax ID.",
			Hint:   "Your Tax ID is typically 10 digits. It may appear on NRS correspondence, your employer's PAYE schedule, or previous tax assessment notices.",
			ShowIf: eq("has_tax_id", "yes")},

		{ID: "Q018", Screen: "02", Code: "retrieve_tax_id", Type: "boolean",
			Label:  "Would you like us to help retrieve your Tax ID?",
			ShowIf: neq("has_tax_id", "yes")},

		// ── SCREEN 03 — Address & residence ───────────────────────────────────

		{ID: "Q019", Screen: "03", Code: "current_location", Type: "single_select",
			Label: "{first_name}, where do you currently live?", Required: true,
			Options: []Option{
				{ID: "nigeria", Label: "Nigeria"},
				{ID: "outside_nigeria", Label: "Outside Nigeria"},
				{ID: "both", Label: "I live between Nigeria and another country"},
			}},

		{ID: "Q020", Screen: "03", Code: "country_of_residence", Type: "single_select",
			Label: "What is your country of residence?", Required: true,
			Options: countries},

		{ID: "Q021", Screen: "03", Code: "nigerian_state", Type: "single_select",
			Label:  "Which Nigerian state do you live in?",
			ShowIf: eq("current_location", "nigeria"),
			Options: nigerianStates},

		{ID: "Q022", Screen: "03", Code: "lga", Type: "text",
			Label:  "Which local government area do you live in?",
			ShowIf: eq("current_location", "nigeria")},

		{ID: "Q023", Screen: "03", Code: "residential_address", Type: "text",
			Label: "What is your residential address?", Required: true},

		{ID: "Q024", Screen: "03", Code: "home_available_nigeria", Type: "single_select",
			Label: "Did you have a home available to you in Nigeria during the tax year?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "part_year", Label: "Part of the year"},
			}},

		{ID: "Q025", Screen: "03", Code: "days_183_in_nigeria", Type: "single_select",
			Label: "Did you spend 183 days or more in Nigeria during the relevant period?",
			Hint:  "183 days is the primary residency test under NTA 2025 s.12. Count all days physically present in Nigeria, including arrival and departure days.",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q026", Screen: "03", Code: "spouse_family_in_nigeria", Type: "single_select",
			Label: "Was your spouse or family primarily based in Nigeria?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_applicable", Label: "Not applicable"},
				{ID: "prefer_not_to_say", Label: "Prefer not to say"},
			}},

		{ID: "Q027", Screen: "03", Code: "main_economic_interests_nigeria", Type: "single_select",
			Label: "Were your main economic or business interests in Nigeria?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "both", Label: "Both Nigeria and another country"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q028", Screen: "03", Code: "nigerian_govt_employee_abroad", Type: "boolean",
			Label: "Were you a Nigerian government or diplomatic employee posted abroad?"},

		{ID: "Q029", Screen: "03", Code: "believes_tax_resident", Type: "single_select",
			Label: "Do you believe you were a Nigerian tax resident during the year?", Required: true,
			Hint:  "Nigerian tax residents pay tax on their worldwide income. Non-residents pay tax only on Nigerian-source income. If unsure, select 'Not sure' — we will help you determine your status.",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		// ── SCREEN 04 — Marital & family situation ─────────────────────────────

		{ID: "Q030", Screen: "04", Code: "marital_status", Type: "single_select",
			Label: "What was your marital status during the tax year?", Required: true,
			Options: []Option{
				{ID: "single", Label: "Single"},
				{ID: "married", Label: "Married"},
				{ID: "divorced", Label: "Divorced"},
				{ID: "widowed", Label: "Widowed"},
				{ID: "separated", Label: "Separated"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q031", Screen: "04", Code: "married_during_year", Type: "boolean",
			Label:  "Did you marry during the tax year?",
			ShowIf: eq("marital_status", "married")},

		{ID: "Q032", Screen: "04", Code: "marriage_ended_during_year", Type: "boolean",
			Label:  "Did your marriage end during the tax year?",
			ShowIf: eq("marital_status", "divorced")},

		{ID: "Q033", Screen: "04", Code: "spouse_had_income", Type: "single_select",
			Label:  "Did your spouse have income?",
			ShowIf: neq("marital_status", "single"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "dont_know", Label: "Don't know"},
				{ID: "not_applicable", Label: "Not applicable"},
			}},

		{ID: "Q034", Screen: "04", Code: "spouse_resident_nigeria", Type: "single_select",
			Label:  "Was your spouse resident in Nigeria?",
			ShowIf: neq("marital_status", "single"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "part_year", Label: "Part of the year"},
				{ID: "not_applicable", Label: "Not applicable"},
			}},

		{ID: "Q035", Screen: "04", Code: "spouse_has_tax_id", Type: "single_select",
			Label:  "Does your spouse have a Nigerian Tax ID?",
			ShowIf: neq("marital_status", "single"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "dont_know", Label: "Don't know"},
				{ID: "not_applicable", Label: "Not applicable"},
			}},

		// ── SCREEN 05 — Dependants ─────────────────────────────────────────────

		{ID: "Q036", Screen: "05", Code: "has_dependants", Type: "single_select",
			Label: "Do you financially support anyone?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q037", Screen: "05", Code: "dependant_count", Type: "number",
			Label:  "How many dependants do you support?",
			ShowIf: eq("has_dependants", "yes")},

		{ID: "Q038", Screen: "05", Code: "dependant_types", Type: "multi_select",
			Label:  "Who do you support?",
			ShowIf: eq("has_dependants", "yes"),
			Options: []Option{
				{ID: "child", Label: "Child"},
				{ID: "stepchild", Label: "Stepchild"},
				{ID: "adopted_child", Label: "Adopted child"},
				{ID: "foster_child", Label: "Foster child"},
				{ID: "spouse", Label: "Spouse"},
				{ID: "parent", Label: "Parent"},
				{ID: "grandparent", Label: "Grandparent"},
				{ID: "sibling", Label: "Sibling"},
				{ID: "other_relative", Label: "Other relative"},
				{ID: "special_care", Label: "Person requiring special care"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q039", Screen: "05", Code: "dependant_lives_with_you", Type: "single_select",
			Label:  "Does any dependant live with you?",
			ShowIf: eq("has_dependants", "yes"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "some", Label: "Some of them"},
			}},

		{ID: "Q040", Screen: "05", Code: "dependant_has_income", Type: "single_select",
			Label:  "Does any dependant have their own income?",
			ShowIf: eq("has_dependants", "yes"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "dont_know", Label: "Don't know"},
			}},

		{ID: "Q041", Screen: "05", Code: "paid_school_fees", Type: "boolean",
			Label:  "Did you pay school fees for a dependant?",
			ShowIf: eq("has_dependants", "yes")},

		{ID: "Q042", Screen: "05", Code: "paid_medical_for_dependant", Type: "boolean",
			Label:  "Did you pay medical expenses for a dependant?",
			ShowIf: eq("has_dependants", "yes")},

		// ── SCREEN 06 — Work situation ─────────────────────────────────────────

		{ID: "Q043", Screen: "06", Code: "work_situation", Type: "multi_select",
			Label: "{first_name}, which of these describe how you earned money during the year?", Required: true,
			Options: []Option{
				{ID: "employee", Label: "Employee", Icon: "work"},
				{ID: "freelancer", Label: "Freelancer", Icon: "laptop_mac"},
				{ID: "contractor", Label: "Contractor", Icon: "handyman"},
				{ID: "self_employed", Label: "Self-employed", Icon: "self_improvement"},
				{ID: "business_owner", Label: "Business owner", Icon: "store"},
				{ID: "professional", Label: "Professional", Icon: "local_library"},
				{ID: "landlord", Label: "Landlord", Icon: "real_estate_agent"},
				{ID: "investor", Label: "Investor", Icon: "trending_up"},
				{ID: "farmer", Label: "Farmer", Icon: "agriculture"},
				{ID: "gig_worker", Label: "Gig worker", Icon: "directions_bike"},
				{ID: "content_creator", Label: "Content creator / influencer", Icon: "video_camera_front"},
				{ID: "retiree", Label: "Retiree / pensioner", Icon: "elderly"},
				{ID: "student_with_income", Label: "Student with income", Icon: "school"},
				{ID: "other", Label: "Other", Icon: "more_horiz"},
			}},

		// ── SCREEN 07 — Employment ─────────────────────────────────────────────

		{ID: "Q044", Screen: "07", Code: "had_employment", Type: "boolean",
			Label:  "Did you work for an employer?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q045", Screen: "07", Code: "employer_count", Type: "single_select",
			Label:  "How many employers did you have?",
			ShowIf: has("work_situation", "employee"),
			Options: []Option{
				{ID: "1", Label: "1"},
				{ID: "2", Label: "2"},
				{ID: "3", Label: "3"},
				{ID: "4_plus", Label: "4 or more"},
				{ID: "dont_know", Label: "Don't know"},
			}},

		{ID: "Q046", Screen: "07", Code: "employer_name", Type: "text",
			Label:      "What was your employer's name?",
			ShowIf:     has("work_situation", "employee"),
			RequiredIf: has("work_situation", "employee")},

		{ID: "Q047", Screen: "07", Code: "employer_address", Type: "text",
			Label:  "What was your employer's address?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q048", Screen: "07", Code: "nigerian_employer", Type: "single_select",
			Label:  "Did you work for a Nigerian employer?",
			ShowIf: has("work_situation", "employee"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "both", Label: "Both Nigerian and foreign employers"},
			}},

		{ID: "Q049", Screen: "07", Code: "worked_physically_nigeria", Type: "single_select",
			Label:  "Did you work physically in Nigeria?",
			ShowIf: has("work_situation", "employee"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "partly", Label: "Partly"},
			}},

		{ID: "Q050", Screen: "07", Code: "worked_remotely_for_foreign_employer", Type: "boolean",
			Label:  "Did you work remotely from Nigeria for a foreign employer?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q051", Screen: "07", Code: "employment_basic_salary", Type: "currency",
			Label:      "What was your total basic salary?",
			Hint:       "Basic salary is your fixed contractual pay before allowances or deductions. Find it on your payslip or employment contract — not the 'Gross' or 'Net' figure.",
			ShowIf:     has("work_situation", "employee"),
			RequiredIf: has("work_situation", "employee")},

		{ID: "Q052", Screen: "07", Code: "employment_gross_salary", Type: "currency",
			Label:  "What was your total gross salary?",
			Hint:   "Gross salary = basic salary + all allowances + bonuses. Use the 'Gross Emoluments' figure on your annual PAYE statement (Form H1) if you have it.",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q053", Screen: "07", Code: "received_bonuses", Type: "boolean",
			Label:  "Did you receive bonuses?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q054", Screen: "07", Code: "bonus_amount", Type: "currency",
			Label:  "How much did you receive in bonuses?",
			ShowIf: eq("received_bonuses", true)},

		{ID: "Q055", Screen: "07", Code: "received_commissions", Type: "boolean",
			Label:  "Did you receive commissions?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q056", Screen: "07", Code: "commission_amount", Type: "currency",
			Label:  "How much commission did you receive?",
			ShowIf: eq("received_commissions", true)},

		{ID: "Q057", Screen: "07", Code: "received_allowances", Type: "boolean",
			Label:  "Did you receive allowances?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q058", Screen: "07", Code: "allowance_types", Type: "multi_select",
			Label:  "Which allowances did you receive?",
			ShowIf: eq("received_allowances", true),
			Options: []Option{
				{ID: "housing", Label: "Housing"},
				{ID: "transport", Label: "Transport"},
				{ID: "meal", Label: "Meal / feeding"},
				{ID: "utility", Label: "Utility"},
				{ID: "communication", Label: "Communication"},
				{ID: "medical", Label: "Medical"},
				{ID: "leave", Label: "Leave"},
				{ID: "entertainment", Label: "Entertainment"},
				{ID: "responsibility", Label: "Responsibility"},
				{ID: "shift", Label: "Shift"},
				{ID: "hazard", Label: "Hazard"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q059", Screen: "07", Code: "taxable_allowances_total", Type: "currency",
			Label:  "What was the total value of your taxable allowances?",
			Hint:   "Some allowances may be fully or partially exempt from tax. Only include the taxable portion your employer declared. Your annual PAYE statement will show a 'Taxable Allowances' figure.",
			ShowIf: eq("received_allowances", true)},

		{ID: "Q060", Screen: "07", Code: "employer_provided_accommodation", Type: "boolean",
			Label:  "Did your employer provide you with accommodation?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q061", Screen: "07", Code: "employer_provided_vehicle", Type: "boolean",
			Label:  "Did your employer provide you with a vehicle or other benefit in kind?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q062", Screen: "07", Code: "other_employer_benefits", Type: "boolean",
			Label:  "Did you receive any other benefits from your employer?",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q063", Screen: "07", Code: "employment_paye_deducted", Type: "boolean",
			Label:  "Did your employer deduct PAYE?",
			Hint:   "PAYE (Pay As You Earn) is income tax your employer deducts from your salary each month and remits directly to the tax authority on your behalf.",
			ShowIf: has("work_situation", "employee")},

		{ID: "Q064", Screen: "07", Code: "employment_paye_amount", Type: "currency",
			Label:      "How much PAYE was deducted?",
			Hint:       "Find the total PAYE for the year on your annual employer statement (Form H1) or add up the 'Tax' column on each monthly payslip.",
			ShowIf:     eq("employment_paye_deducted", true),
			RequiredIf: eq("employment_paye_deducted", true)},

		{ID: "Q065", Screen: "07", Code: "has_annual_paye_statement", Type: "single_select",
			Label:  "Do you have your annual employer / PAYE statement?",
			ShowIf: has("work_situation", "employee"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_provided", Label: "Employer hasn't provided it"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q066", Screen: "07", Code: "paye_statement_file", Type: "file",
			Label:  "Upload your PAYE / employer statement.",
			ShowIf: eq("has_annual_paye_statement", "yes")},

		// ── SCREEN 08 — Freelance & gig income ────────────────────────────────

		{ID: "Q067", Screen: "08", Code: "has_freelance_income", Type: "boolean",
			Label:  "Did you earn money from freelance or gig work?",
			ShowIf: has("work_situation", "freelancer")},

		{ID: "Q068", Screen: "08", Code: "freelance_work_types", Type: "multi_select",
			Label:  "What type of freelance or gig work did you do?",
			ShowIf: has("work_situation", "freelancer"),
			Options: []Option{
				{ID: "technology", Label: "Technology"},
				{ID: "design", Label: "Design"},
				{ID: "writing", Label: "Writing"},
				{ID: "consulting", Label: "Consulting"},
				{ID: "marketing", Label: "Marketing"},
				{ID: "sales", Label: "Sales"},
				{ID: "photography_video", Label: "Photography / video"},
				{ID: "education", Label: "Education"},
				{ID: "healthcare", Label: "Healthcare"},
				{ID: "engineering", Label: "Engineering"},
				{ID: "legal", Label: "Legal"},
				{ID: "accounting", Label: "Accounting"},
				{ID: "transportation", Label: "Transportation"},
				{ID: "delivery", Label: "Delivery"},
				{ID: "content_creation", Label: "Content creation"},
				{ID: "social_media", Label: "Social media"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q069", Screen: "08", Code: "freelance_activity_count", Type: "number",
			Label:  "How many separate freelance or gig activities did you have?",
			ShowIf: has("work_situation", "freelancer")},

		{ID: "Q070", Screen: "08", Code: "freelance_gross_income", Type: "currency",
			Label:      "What was your total gross freelance income?",
			ShowIf:     has("work_situation", "freelancer"),
			RequiredIf: has("work_situation", "freelancer")},

		{ID: "Q071", Screen: "08", Code: "freelance_client_origin", Type: "single_select",
			Label:  "Was any of this income paid by Nigerian clients?",
			ShowIf: has("work_situation", "freelancer"),
			Options: []Option{
				{ID: "nigerian", Label: "Yes — Nigerian clients only"},
				{ID: "foreign", Label: "No — foreign clients only"},
				{ID: "both", Label: "Both Nigerian and foreign clients"},
			}},

		{ID: "Q072", Screen: "08", Code: "freelance_wht_deducted", Type: "boolean",
			Label:  "Was any tax withheld from your freelance payments?",
			ShowIf: has("work_situation", "freelancer")},

		{ID: "Q073", Screen: "08", Code: "freelance_wht_amount", Type: "currency",
			Label:  "How much was withheld?",
			Hint:   "Nigerian organisations typically deduct 5% WHT from freelance payments and issue a WHT credit note. Keep these — the amount becomes a credit against your final tax bill.",
			ShowIf: eq("freelance_wht_deducted", true)},

		{ID: "Q074", Screen: "08", Code: "has_freelance_expenses", Type: "boolean",
			Label:  "Did you incur expenses directly related to earning this income?",
			ShowIf: has("work_situation", "freelancer")},

		{ID: "Q075", Screen: "08", Code: "freelance_expense_types", Type: "multi_select",
			Label:  "What categories of expenses did you have?",
			ShowIf: eq("has_freelance_expenses", true),
			Options: []Option{
				{ID: "internet", Label: "Internet / data"},
				{ID: "phone", Label: "Phone"},
				{ID: "software", Label: "Software"},
				{ID: "equipment", Label: "Equipment"},
				{ID: "transport", Label: "Transport"},
				{ID: "office", Label: "Office"},
				{ID: "advertising", Label: "Advertising"},
				{ID: "professional_fees", Label: "Professional fees"},
				{ID: "bank_charges", Label: "Bank / payment charges"},
				{ID: "contractors", Label: "Contractors"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q076", Screen: "08", Code: "has_freelance_expense_records", Type: "single_select",
			Label:  "Do you have records or receipts for these expenses?",
			ShowIf: eq("has_freelance_expenses", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "some", Label: "Some"},
				{ID: "no", Label: "No"},
			}},

		// ── SCREEN 09 — Business ───────────────────────────────────────────────

		{ID: "Q077", Screen: "09", Code: "has_business_income", Type: "boolean",
			Label:  "Do you operate a business?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q078", Screen: "09", Code: "business_registration_type", Type: "single_select",
			Label:  "How is the business registered?",
			ShowIf: has("work_situation", "business_owner"),
			Options: []Option{
				{ID: "cac_company", Label: "CAC company"},
				{ID: "cac_business_name", Label: "CAC business name"},
				{ID: "partnership", Label: "Partnership"},
				{ID: "unregistered", Label: "Unregistered / informal"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q079", Screen: "09", Code: "business_name", Type: "text",
			Label:  "What is the business name?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q080", Screen: "09", Code: "business_activity", Type: "text",
			Label:  "What does the business do?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q081", Screen: "09", Code: "business_total_revenue", Type: "currency",
			Label:  "What was the business's total revenue?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q082", Screen: "09", Code: "business_allowable_expenses", Type: "currency",
			Label:  "How much did the business spend on allowable business expenses?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q083", Screen: "09", Code: "business_paid_employees", Type: "boolean",
			Label:  "Did the business pay employees?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q084", Screen: "09", Code: "business_paid_contractors", Type: "boolean",
			Label:  "Did you pay contractors?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q085", Screen: "09", Code: "business_paid_rent", Type: "boolean",
			Label:  "Did the business pay rent?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q086", Screen: "09", Code: "business_bought_assets", Type: "boolean",
			Label:  "Did the business buy equipment or assets?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q087", Screen: "09", Code: "business_incurred_loss", Type: "boolean",
			Label:  "Did the business incur a loss?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q088", Screen: "09", Code: "has_business_records", Type: "single_select",
			Label:  "Do you have business financial records?",
			ShowIf: has("work_situation", "business_owner"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "partial", Label: "Partial"},
				{ID: "no", Label: "No"},
			}},

		{ID: "Q089", Screen: "09", Code: "has_profit_loss_statement", Type: "boolean",
			Label:  "Do you have an income statement or profit-and-loss statement?",
			ShowIf: has("work_situation", "business_owner")},

		{ID: "Q090", Screen: "09", Code: "has_balance_sheet", Type: "boolean",
			Label:  "Do you have a statement of assets and liabilities?",
			ShowIf: has("work_situation", "business_owner")},

		// ── SCREEN 10 — Professional income ───────────────────────────────────

		{ID: "Q091", Screen: "10", Code: "has_professional_income", Type: "boolean",
			Label:  "Did you earn income from a profession?",
			ShowIf: has("work_situation", "professional")},

		{ID: "Q092", Screen: "10", Code: "profession_type", Type: "single_select",
			Label:  "Which profession?",
			ShowIf: has("work_situation", "professional"),
			Options: []Option{
				{ID: "lawyer", Label: "Lawyer"},
				{ID: "accountant", Label: "Accountant"},
				{ID: "doctor", Label: "Doctor"},
				{ID: "engineer", Label: "Engineer"},
				{ID: "architect", Label: "Architect"},
				{ID: "consultant", Label: "Consultant"},
				{ID: "surveyor", Label: "Surveyor"},
				{ID: "pharmacist", Label: "Pharmacist"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q093", Screen: "10", Code: "professional_gross_income", Type: "currency",
			Label:      "What was your gross professional income?",
			ShowIf:     has("work_situation", "professional"),
			RequiredIf: has("work_situation", "professional")},

		{ID: "Q094", Screen: "10", Code: "has_professional_expenses", Type: "boolean",
			Label:  "Did you have professional expenses?",
			ShowIf: has("work_situation", "professional")},

		{ID: "Q095", Screen: "10", Code: "professional_expenses_total", Type: "currency",
			Label:  "What were your total professional expenses?",
			ShowIf: eq("has_professional_expenses", true)},

		{ID: "Q096", Screen: "10", Code: "professional_wht_deducted", Type: "boolean",
			Label:  "Did any client deduct withholding tax?",
			ShowIf: has("work_situation", "professional")},

		{ID: "Q097", Screen: "10", Code: "professional_wht_amount", Type: "currency",
			Label:  "What was the total withholding tax deducted?",
			Hint:   "WHT deducted by clients (typically 5%) is a credit against your personal income tax. Collect WHT credit notes from every client — you will need them when filing.",
			ShowIf: eq("professional_wht_deducted", true)},

		// ── SCREEN 11 — Rental income ──────────────────────────────────────────

		{ID: "Q098", Screen: "11", Code: "receives_rental_income", Type: "boolean",
			Label:  "Did you receive rent from property?",
			ShowIf: has("work_situation", "landlord")},

		{ID: "Q099", Screen: "11", Code: "rental_property_count", Type: "number",
			Label:  "How many properties generated rent?",
			ShowIf: has("work_situation", "landlord")},

		{ID: "Q100", Screen: "11", Code: "rental_property_type", Type: "single_select",
			Label:  "What type of property?",
			ShowIf: has("work_situation", "landlord"),
			Options: []Option{
				{ID: "residential", Label: "Residential"},
				{ID: "commercial", Label: "Commercial"},
				{ID: "land", Label: "Land"},
				{ID: "short_let", Label: "Short-let"},
				{ID: "mixed_use", Label: "Mixed-use"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q101", Screen: "11", Code: "rental_property_location", Type: "single_select",
			Label:  "Where is the property?",
			ShowIf: has("work_situation", "landlord"),
			Options: []Option{
				{ID: "nigeria", Label: "Nigeria"},
				{ID: "outside_nigeria", Label: "Outside Nigeria"},
			}},

		{ID: "Q102", Screen: "11", Code: "rental_property_state", Type: "single_select",
			Label:   "Which Nigerian state?",
			ShowIf:  eq("rental_property_location", "nigeria"),
			Options: nigerianStates},

		{ID: "Q103", Screen: "11", Code: "total_rental_income", Type: "currency",
			Label:      "What was your total rental income?",
			Hint:       "Include all rent received across all properties and all months. If you received advance rent covering future periods, include the full amount in the year you received it.",
			ShowIf:     has("work_situation", "landlord"),
			RequiredIf: has("work_situation", "landlord")},

		{ID: "Q104", Screen: "11", Code: "tenant_wht_deducted", Type: "boolean",
			Label:  "Did a tenant, agent or organisation withhold tax?",
			ShowIf: has("work_situation", "landlord")},

		{ID: "Q105", Screen: "11", Code: "tenant_wht_amount", Type: "currency",
			Label:  "How much withholding tax was deducted?",
			ShowIf: eq("tenant_wht_deducted", true)},

		{ID: "Q106", Screen: "11", Code: "has_property_expenses", Type: "boolean",
			Label:  "Did you incur expenses related to the property?",
			ShowIf: has("work_situation", "landlord")},

		{ID: "Q107", Screen: "11", Code: "property_expense_types", Type: "multi_select",
			Label:  "What property expenses did you pay?",
			ShowIf: eq("has_property_expenses", true),
			Options: []Option{
				{ID: "repairs", Label: "Repairs"},
				{ID: "management", Label: "Property management"},
				{ID: "insurance", Label: "Insurance"},
				{ID: "security", Label: "Security"},
				{ID: "service_charges", Label: "Service charges"},
				{ID: "mortgage_interest", Label: "Mortgage / loan interest"},
				{ID: "other", Label: "Other"},
			}},

		// ── SCREEN 12 — Your own rent (rent relief) ────────────────────────────

		{ID: "Q108", Screen: "12", Code: "pays_own_rent", Type: "boolean",
			Label: "{first_name}, did you pay rent for your own residence during the year?"},

		{ID: "Q109", Screen: "12", Code: "annual_rent_paid", Type: "currency",
			Label:  "How much rent did you actually pay?",
			Hint:   "Enter the total rent paid across all months of the year. Rent relief under NTA s.30 is capped at the lesser of 20% of this amount or ₦500,000.",
			ShowIf: eq("pays_own_rent", true)},

		{ID: "Q110", Screen: "12", Code: "paid_rent_self", Type: "single_select",
			Label:  "Did you pay the rent yourself?",
			Hint:   "To claim rent relief, you must have personally paid the rent — employer-provided accommodation or rent paid by a third party does not qualify.",
			ShowIf: eq("pays_own_rent", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "partly", Label: "Partly"},
			}},

		{ID: "Q111", Screen: "12", Code: "has_rent_proof", Type: "single_select",
			Label:  "Do you have proof of payment?",
			ShowIf: eq("pays_own_rent", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "some", Label: "Some"},
				{ID: "no", Label: "No"},
			}},

		{ID: "Q112", Screen: "12", Code: "has_tenancy_agreement", Type: "boolean",
			Label:  "Do you have a tenancy agreement or other supporting document?",
			ShowIf: eq("pays_own_rent", true)},

		{ID: "Q113", Screen: "12", Code: "rent_is_own_residence", Type: "boolean",
			Label:  "Was the property your own residence?",
			ShowIf: eq("pays_own_rent", true)},

		// ── SCREEN 13 — Bank interest & investment income ──────────────────────

		{ID: "Q114", Screen: "13", Code: "received_interest_income", Type: "boolean",
			Label: "Did you receive interest income?"},

		{ID: "Q115", Screen: "13", Code: "interest_income_source", Type: "multi_select",
			Label:  "Where did the interest come from?",
			ShowIf: eq("received_interest_income", true),
			Options: []Option{
				{ID: "bank_account", Label: "Bank account"},
				{ID: "fixed_deposit", Label: "Fixed deposit"},
				{ID: "treasury_bills", Label: "Treasury bills"},
				{ID: "bonds", Label: "Bonds"},
				{ID: "loan_to_person", Label: "Loan to another person or business"},
				{ID: "foreign_account", Label: "Foreign account"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q116", Screen: "13", Code: "total_interest_received", Type: "currency",
			Label:  "What was your total interest received?",
			ShowIf: eq("received_interest_income", true)},

		{ID: "Q117", Screen: "13", Code: "interest_tax_deducted", Type: "boolean",
			Label:  "Was tax deducted from the interest?",
			ShowIf: eq("received_interest_income", true)},

		{ID: "Q118", Screen: "13", Code: "interest_tax_amount", Type: "currency",
			Label:  "How much tax was deducted?",
			ShowIf: eq("interest_tax_deducted", true)},

		{ID: "Q119", Screen: "13", Code: "received_dividends", Type: "boolean",
			Label: "Did you receive dividends?"},

		{ID: "Q120", Screen: "13", Code: "dividends_from_nigerian_companies", Type: "single_select",
			Label:  "Were the dividends from Nigerian companies?",
			ShowIf: eq("received_dividends", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "both", Label: "Both"},
			}},

		{ID: "Q121", Screen: "13", Code: "total_dividends_received", Type: "currency",
			Label:  "What was your total dividends received?",
			ShowIf: eq("received_dividends", true)},

		// ── SCREEN 14 — Shares & investments ──────────────────────────────────

		{ID: "Q122", Screen: "14", Code: "sold_investments", Type: "boolean",
			Label: "Did you sell shares or investments?"},

		{ID: "Q123", Screen: "14", Code: "investment_types_sold", Type: "multi_select",
			Label:  "What did you sell?",
			ShowIf: eq("sold_investments", true),
			Options: []Option{
				{ID: "nigerian_shares", Label: "Nigerian shares"},
				{ID: "foreign_shares", Label: "Foreign shares"},
				{ID: "mutual_funds", Label: "Mutual funds"},
				{ID: "etfs", Label: "ETFs"},
				{ID: "bonds", Label: "Bonds"},
				{ID: "other_securities", Label: "Other securities"},
				{ID: "other_assets", Label: "Other assets"},
			}},

		{ID: "Q124", Screen: "14", Code: "nigerian_shares_included", Type: "boolean",
			Label:  "Were any shares in Nigerian companies?",
			ShowIf: eq("sold_investments", true)},

		{ID: "Q125", Screen: "14", Code: "nigerian_share_proceeds", Type: "currency",
			Label:  "What were your total Nigerian share-disposal proceeds?",
			Hint:   "Under NTA s.34, gains on Nigerian company shares may be exempt if total proceeds are below ₦150m AND the total gain is below ₦10m — or if proceeds are fully reinvested in other Nigerian shares within the same assessment year.",
			ShowIf: eq("nigerian_shares_included", true)},

		{ID: "Q126", Screen: "14", Code: "nigerian_share_gain", Type: "currency",
			Label:  "What was your total gain from those disposals?",
			Hint:   "Gain = disposal proceeds minus original acquisition cost (plus qualifying improvement costs). If you received the shares as a gift, use the market value at the date of the gift as your acquisition cost.",
			ShowIf: eq("nigerian_shares_included", true)},

		{ID: "Q127", Screen: "14", Code: "reinvested_proceeds", Type: "single_select",
			Label:  "Did you reinvest proceeds into shares in the same assessment year?",
			ShowIf: eq("nigerian_shares_included", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "partly", Label: "Partly"},
			}},

		{ID: "Q128", Screen: "14", Code: "reinvestment_amount", Type: "currency",
			Label:  "How much was reinvested?",
			ShowIf: eq("reinvested_proceeds", "yes")},

		{ID: "Q129", Screen: "14", Code: "sold_at_loss", Type: "boolean",
			Label:  "Did you sell any investment at a loss?",
			ShowIf: eq("sold_investments", true)},

		{ID: "Q130", Screen: "14", Code: "has_investment_statements", Type: "boolean",
			Label:  "Do you have brokerage or investment statements?",
			ShowIf: eq("sold_investments", true)},

		// ── SCREEN 15 — Cryptocurrency & digital assets ────────────────────────

		{ID: "Q131", Screen: "15", Code: "crypto_activity", Type: "single_select",
			Label: "Did you transact in cryptocurrency or other virtual assets?",
			Hint:  "The NRS treats virtual asset transactions as potentially taxable under NTA 2025. This includes selling, exchanging, mining, staking, airdrops, hard forks, and receiving crypto as payment or compensation.",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "only_held", Label: "I only held them"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q132", Screen: "15", Code: "crypto_activities_done", Type: "multi_select",
			Label:  "What did you do?",
			ShowIf: eq("crypto_activity", "yes"),
			Options: []Option{
				{ID: "bought", Label: "Bought"},
				{ID: "sold", Label: "Sold"},
				{ID: "exchanged", Label: "Exchanged"},
				{ID: "transferred", Label: "Transferred"},
				{ID: "received_as_payment", Label: "Received as payment"},
				{ID: "received_as_compensation", Label: "Received as compensation"},
				{ID: "staked", Label: "Staked"},
				{ID: "mined", Label: "Mined"},
				{ID: "airdrop", Label: "Received an airdrop"},
				{ID: "hard_fork", Label: "Received a hard fork"},
				{ID: "received_rewards", Label: "Received rewards"},
				{ID: "used_to_buy", Label: "Used crypto to buy goods/services"},
			}},

		{ID: "Q133", Screen: "15", Code: "crypto_asset_types", Type: "text",
			Label:  "Which assets did you transact in?",
			ShowIf: eq("crypto_activity", "yes")},

		{ID: "Q134", Screen: "15", Code: "total_crypto_disposal_value", Type: "currency",
			Label:  "What was the total value of disposals or transfers?",
			ShowIf: eq("crypto_activity", "yes")},

		{ID: "Q135", Screen: "15", Code: "crypto_acquisition_cost", Type: "currency",
			Label:  "What was your acquisition cost?",
			ShowIf: eq("crypto_activity", "yes")},

		{ID: "Q136", Screen: "15", Code: "received_crypto_for_work", Type: "boolean",
			Label:  "Did you receive crypto for work or services?",
			ShowIf: eq("crypto_activity", "yes")},

		{ID: "Q137", Screen: "15", Code: "received_crypto_staking_mining", Type: "boolean",
			Label:  "Did you receive crypto from staking, mining or rewards?",
			ShowIf: eq("crypto_activity", "yes")},

		{ID: "Q138", Screen: "15", Code: "has_crypto_records", Type: "single_select",
			Label:  "Do you have exchange or wallet transaction records?",
			ShowIf: eq("crypto_activity", "yes"),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "partial", Label: "Partial"},
				{ID: "no", Label: "No"},
			}},

		// ── SCREEN 16 — Foreign income ─────────────────────────────────────────

		{ID: "Q139", Screen: "16", Code: "earned_foreign_income", Type: "boolean",
			Label: "Did you earn income outside Nigeria?"},

		{ID: "Q140", Screen: "16", Code: "foreign_income_countries", Type: "text",
			Label:  "Which countries did the income come from?",
			ShowIf: eq("earned_foreign_income", true)},

		{ID: "Q141", Screen: "16", Code: "foreign_income_types", Type: "multi_select",
			Label:  "What type of foreign income?",
			ShowIf: eq("earned_foreign_income", true),
			Options: []Option{
				{ID: "salary", Label: "Salary"},
				{ID: "freelance", Label: "Freelance"},
				{ID: "business", Label: "Business"},
				{ID: "rent", Label: "Rent"},
				{ID: "interest", Label: "Interest"},
				{ID: "dividends", Label: "Dividends"},
				{ID: "pension", Label: "Pension"},
				{ID: "investment_gains", Label: "Investment gains"},
				{ID: "digital_assets", Label: "Digital assets"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q142", Screen: "16", Code: "total_foreign_income", Type: "currency",
			Label:  "What was your total foreign income?",
			ShowIf: eq("earned_foreign_income", true)},

		{ID: "Q143", Screen: "16", Code: "paid_tax_in_other_country", Type: "boolean",
			Label:  "Did you pay tax in another country on this income?",
			ShowIf: eq("earned_foreign_income", true)},

		{ID: "Q144", Screen: "16", Code: "foreign_tax_paid_amount", Type: "currency",
			Label:  "How much foreign tax did you pay?",
			ShowIf: eq("paid_tax_in_other_country", true)},

		{ID: "Q145", Screen: "16", Code: "has_foreign_tax_proof", Type: "boolean",
			Label:  "Do you have proof of foreign tax paid?",
			ShowIf: eq("paid_tax_in_other_country", true)},

		// ── SCREEN 17 — Pension ────────────────────────────────────────────────

		{ID: "Q146", Screen: "17", Code: "makes_pension_contributions", Type: "boolean",
			Label: "Did you make pension contributions, {first_name}?"},

		{ID: "Q147", Screen: "17", Code: "pension_contribution_amount", Type: "currency",
			Label:  "How much did you contribute?",
			Hint:   "Qualifying pension under the Contributory Pension Scheme (CPS) is an eligible deduction under NTA s.30. For most employees this is 8% of monthly emoluments deducted automatically from salary.",
			ShowIf: eq("makes_pension_contributions", true)},

		{ID: "Q148", Screen: "17", Code: "pension_through_employer", Type: "single_select",
			Label:  "Were the contributions made through your employer?",
			ShowIf: eq("makes_pension_contributions", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "both", Label: "Both"},
			}},

		{ID: "Q149", Screen: "17", Code: "makes_voluntary_pension", Type: "boolean",
			Label:  "Did you make voluntary pension contributions?",
			Hint:   "Voluntary Additional Contributions (VACs) to your Retirement Savings Account (RSA) may also qualify as eligible deductions under NTA s.30. Confirm with your Pension Fund Administrator (PFA).",
			ShowIf: eq("makes_pension_contributions", true)},

		{ID: "Q150", Screen: "17", Code: "receives_pension_income", Type: "boolean",
			Label: "Did you receive pension income?"},

		{ID: "Q151", Screen: "17", Code: "pension_income_amount", Type: "currency",
			Label:  "How much pension income did you receive?",
			ShowIf: eq("receives_pension_income", true)},

		// ── SCREEN 18 — NHIS & NHF ─────────────────────────────────────────────

		{ID: "Q152", Screen: "18", Code: "makes_nhis_contributions", Type: "boolean",
			Label: "Did you make National Health Insurance (NHIS) contributions?",
			Hint:  "NHIS contributions are typically 5% of basic salary for employees, deducted monthly. Look for 'NHIS' or 'Health Insurance' on your payslip. Qualifying contributions are eligible deductions under NTA s.30."},

		{ID: "Q153", Screen: "18", Code: "nhis_contribution_amount", Type: "currency",
			Label:  "How much did you contribute?",
			Hint:   "Enter the total NHIS deducted across all 12 months. Your payslip shows the monthly figure — multiply by the number of months you were employed.",
			ShowIf: eq("makes_nhis_contributions", true)},

		{ID: "Q154", Screen: "18", Code: "makes_nhf_contributions", Type: "boolean",
			Label: "Did you contribute to the National Housing Fund (NHF)?",
			Hint:  "NHF contributions are 2.5% of your monthly basic salary, deducted by your employer and remitted to the Federal Mortgage Bank of Nigeria (FMBN). Look for 'NHF' or 'Housing Fund' on your payslip."},

		{ID: "Q155", Screen: "18", Code: "nhf_contribution_amount", Type: "currency",
			Label:  "How much did you contribute?",
			Hint:   "Use the total NHF deducted across the year. Your employer or the FMBN can issue an NHF contribution statement if you need confirmation.",
			ShowIf: eq("makes_nhf_contributions", true)},

		// ── SCREEN 19 — Life insurance ─────────────────────────────────────────

		{ID: "Q156", Screen: "19", Code: "has_life_insurance", Type: "boolean",
			Label: "Did you pay premiums for life insurance?"},

		{ID: "Q157", Screen: "19", Code: "life_insured_person", Type: "single_select",
			Label:  "Whose life was insured?",
			ShowIf: eq("has_life_insurance", true),
			Options: []Option{
				{ID: "self", Label: "Mine"},
				{ID: "spouse", Label: "My spouse's"},
				{ID: "both", Label: "Both"},
			}},

		{ID: "Q158", Screen: "19", Code: "has_deferred_annuity", Type: "boolean",
			Label:  "Did you pay premiums for a deferred annuity?",
			ShowIf: eq("has_life_insurance", true)},

		{ID: "Q159", Screen: "19", Code: "qualifying_premiums_total", Type: "currency",
			Label:  "What were your total qualifying premiums paid?",
			ShowIf: eq("has_life_insurance", true)},

		{ID: "Q160", Screen: "19", Code: "has_premium_proof", Type: "boolean",
			Label:  "Do you have proof of payment?",
			ShowIf: eq("has_life_insurance", true)},

		// ── SCREEN 20 — Home loan ──────────────────────────────────────────────

		{ID: "Q161", Screen: "20", Code: "has_home_loan", Type: "boolean",
			Label: "Did you take a loan to develop your owner-occupied residential house?"},

		{ID: "Q162", Screen: "20", Code: "home_loan_property_owner_occupied", Type: "boolean",
			Label:  "Is the property your owner-occupied residence?",
			ShowIf: eq("has_home_loan", true)},

		{ID: "Q163", Screen: "20", Code: "home_loan_interest_paid", Type: "currency",
			Label:  "How much interest did you pay during the year?",
			Hint:   "Only interest on loans for building or completing your own owner-occupied home qualifies under NTA s.30. Loans for property purchase, investment property, or commercial property do not qualify.",
			ShowIf: eq("has_home_loan", true)},

		{ID: "Q164", Screen: "20", Code: "has_lender_statement", Type: "boolean",
			Label:  "Do you have a lender statement showing the interest?",
			ShowIf: eq("has_home_loan", true)},

		// ── SCREEN 21 — Other income ───────────────────────────────────────────

		{ID: "Q165", Screen: "21", Code: "has_other_income", Type: "boolean",
			Label: "Did you receive any other money or economic benefit not already reported?"},

		{ID: "Q166", Screen: "21", Code: "other_income_types", Type: "multi_select",
			Label:  "What type?",
			ShowIf: eq("has_other_income", true),
			Options: []Option{
				{ID: "prize", Label: "Prize"},
				{ID: "winnings", Label: "Winnings"},
				{ID: "award", Label: "Award"},
				{ID: "grant", Label: "Grant"},
				{ID: "honorarium", Label: "Honorarium"},
				{ID: "royalty", Label: "Royalty"},
				{ID: "commission", Label: "Commission"},
				{ID: "annuity", Label: "Annuity"},
				{ID: "compensation", Label: "Compensation"},
				{ID: "settlement", Label: "Settlement"},
				{ID: "inheritance", Label: "Inheritance"},
				{ID: "gift", Label: "Gift"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q167", Screen: "21", Code: "other_income_total", Type: "currency",
			Label:  "How much did you receive?",
			ShowIf: eq("has_other_income", true)},

		{ID: "Q168", Screen: "21", Code: "other_income_tax_deducted", Type: "boolean",
			Label:  "Was tax deducted from the payment?",
			ShowIf: eq("has_other_income", true)},

		{ID: "Q169", Screen: "21", Code: "other_income_is_compensation", Type: "boolean",
			Label:  "Was the amount compensation or damages for personal injury or loss of office/employment?",
			ShowIf: eq("has_other_income", true)},

		{ID: "Q170", Screen: "21", Code: "compensation_within_50m", Type: "boolean",
			Label:  "Was the total compensation ₦50 million or less?",
			Hint:   "Under NTA s.25, the first ₦50m of qualifying compensation for personal injury or loss of office is excluded from chargeable gains. Only the amount above ₦50m may be taxable.",
			ShowIf: eq("other_income_is_compensation", true)},

		// ── SCREEN 22 — Asset disposals ────────────────────────────────────────

		{ID: "Q171", Screen: "22", Code: "disposed_major_assets", Type: "boolean",
			Label: "Did you sell, transfer or otherwise dispose of any major assets?"},

		{ID: "Q172", Screen: "22", Code: "disposed_asset_types", Type: "multi_select",
			Label:  "What did you dispose of?",
			ShowIf: eq("disposed_major_assets", true),
			Options: []Option{
				{ID: "land", Label: "Land"},
				{ID: "house", Label: "House"},
				{ID: "apartment", Label: "Apartment"},
				{ID: "commercial_property", Label: "Commercial property"},
				{ID: "shares", Label: "Shares"},
				{ID: "vehicle", Label: "Vehicle"},
				{ID: "business", Label: "Business"},
				{ID: "equipment", Label: "Equipment"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q173", Screen: "22", Code: "asset_is_principal_residence", Type: "boolean",
			Label:  "Was the property your principal private residence?",
			ShowIf: eq("disposed_major_assets", true)},

		{ID: "Q174", Screen: "22", Code: "asset_used_for_business", Type: "single_select",
			Label:  "Was the asset used for your business?",
			ShowIf: eq("disposed_major_assets", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "partly", Label: "Partly"},
			}},

		{ID: "Q175", Screen: "22", Code: "asset_acquisition_date", Type: "date",
			Label:  "When did you acquire it?",
			ShowIf: eq("disposed_major_assets", true)},

		{ID: "Q176", Screen: "22", Code: "asset_acquisition_cost", Type: "currency",
			Label:  "How much did you pay to acquire it?",
			ShowIf: eq("disposed_major_assets", true)},

		{ID: "Q177", Screen: "22", Code: "asset_disposal_proceeds", Type: "currency",
			Label:  "How much did you receive when you disposed of it?",
			ShowIf: eq("disposed_major_assets", true)},

		{ID: "Q178", Screen: "22", Code: "incurred_disposal_costs", Type: "boolean",
			Label:  "Did you incur costs to acquire, sell or improve the asset?",
			ShowIf: eq("disposed_major_assets", true)},

		{ID: "Q179", Screen: "22", Code: "total_disposal_costs", Type: "currency",
			Label:  "What were your total qualifying costs?",
			ShowIf: eq("incurred_disposal_costs", true)},

		// ── SCREEN 23 — Tax already paid ──────────────────────────────────────

		{ID: "Q180", Screen: "23", Code: "paid_tax_during_year", Type: "boolean",
			Label: "Did you already pay tax during the year?"},

		{ID: "Q181", Screen: "23", Code: "tax_payment_types", Type: "multi_select",
			Label:  "What type of tax or payment?",
			ShowIf: eq("paid_tax_during_year", true),
			Options: []Option{
				{ID: "paye", Label: "PAYE"},
				{ID: "withholding_tax", Label: "Withholding tax"},
				{ID: "direct_assessment", Label: "Direct assessment"},
				{ID: "self_assessment", Label: "Self-assessment"},
				{ID: "foreign_tax", Label: "Foreign income tax"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q182", Screen: "23", Code: "paye_deducted_total", Type: "currency",
			Label:  "How much PAYE was deducted?",
			Hint:   "This should match the 'Tax Deducted' or 'PAYE' figure on your employer's annual deduction certificate (Form H1). If you had multiple employers, add their PAYE figures together.",
			ShowIf: eq("paid_tax_during_year", true)},

		{ID: "Q183", Screen: "23", Code: "wht_deducted_total", Type: "currency",
			Label:  "How much withholding tax was deducted?",
			Hint:   "Add up all WHT credit notes received from clients, tenants, or financial institutions during the year. WHT is a credit against your personal income tax liability.",
			ShowIf: eq("paid_tax_during_year", true)},

		{ID: "Q184", Screen: "23", Code: "direct_assessment_paid", Type: "currency",
			Label:  "How much direct or self-assessment tax did you pay?",
			ShowIf: eq("paid_tax_during_year", true)},

		{ID: "Q185", Screen: "23", Code: "has_tax_receipts", Type: "single_select",
			Label:  "Do you have payment receipts or certificates?",
			ShowIf: eq("paid_tax_during_year", true),
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "some", Label: "Some"},
				{ID: "no", Label: "No"},
			}},

		{ID: "Q186", Screen: "23", Code: "tax_certificate_file", Type: "file",
			Label:  "Upload your tax certificates or receipts.",
			ShowIf: eq("paid_tax_during_year", true)},

		// ── SCREEN 24 — Previous filing & compliance ───────────────────────────

		{ID: "Q187", Screen: "24", Code: "has_filed_before", Type: "boolean",
			Label: "Have you filed a Nigerian tax return before?"},

		{ID: "Q188", Screen: "24", Code: "last_return_year", Type: "text",
			Label:  "When was your last return? (year)",
			ShowIf: eq("has_filed_before", true)},

		{ID: "Q189", Screen: "24", Code: "how_filed_previously", Type: "single_select",
			Label:  "How did you file it?",
			ShowIf: eq("has_filed_before", true),
			Options: []Option{
				{ID: "portal", Label: "Tax authority portal"},
				{ID: "accountant", Label: "Accountant"},
				{ID: "consultant", Label: "Tax consultant"},
				{ID: "employer", Label: "Employer"},
				{ID: "manually", Label: "Manually"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q190", Screen: "24", Code: "has_copy_previous_return", Type: "boolean",
			Label:  "Do you have a copy?",
			ShowIf: eq("has_filed_before", true)},

		{ID: "Q191", Screen: "24", Code: "has_outstanding_assessments", Type: "boolean",
			Label: "Do you have any outstanding tax assessments?"},

		{ID: "Q192", Screen: "24", Code: "has_unpaid_tax", Type: "boolean",
			Label: "Do you have unpaid tax?"},

		{ID: "Q193", Screen: "24", Code: "in_tax_audit_or_dispute", Type: "boolean",
			Label: "Are you currently involved in a tax audit or dispute?"},

		{ID: "Q194", Screen: "24", Code: "received_demand_notice", Type: "boolean",
			Label: "Have you received a tax demand or assessment notice?"},

		// ── SCREEN 25 — Documents ──────────────────────────────────────────────

		{ID: "Q195", Screen: "25", Code: "available_documents", Type: "multi_select",
			Label: "Which documents do you have?",
			Options: []Option{
				{ID: "nin", Label: "NIN"},
				{ID: "tax_id", Label: "Tax ID"},
				{ID: "payslips", Label: "Payslips"},
				{ID: "annual_paye_statement", Label: "Annual PAYE statement"},
				{ID: "bank_statements", Label: "Bank statements"},
				{ID: "investment_statements", Label: "Investment statements"},
				{ID: "wht_certificates", Label: "WHT certificates"},
				{ID: "pension_statement", Label: "Pension statement"},
				{ID: "nhis_evidence", Label: "NHIS evidence"},
				{ID: "nhf_evidence", Label: "NHF evidence"},
				{ID: "rent_agreement", Label: "Rent agreement"},
				{ID: "rent_receipts", Label: "Rent receipts"},
				{ID: "life_insurance_receipts", Label: "Life-insurance receipts"},
				{ID: "mortgage_statement", Label: "Mortgage statement"},
				{ID: "business_accounts", Label: "Business accounts"},
				{ID: "business_receipts", Label: "Business receipts"},
				{ID: "property_purchase_docs", Label: "Property purchase documents"},
				{ID: "property_sale_docs", Label: "Property sale documents"},
				{ID: "crypto_transaction_report", Label: "Crypto transaction report"},
				{ID: "foreign_tax_documents", Label: "Foreign tax documents"},
				{ID: "previous_tax_return", Label: "Previous tax return"},
				{ID: "other", Label: "Other"},
			}},

		{ID: "Q196", Screen: "25", Code: "wants_to_upload_now", Type: "single_select",
			Label: "Would you like to upload your documents now?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "later", Label: "Later"},
			}},

		{ID: "Q197", Screen: "25", Code: "has_bank_statements", Type: "single_select",
			Label: "Do you have bank statements covering your income period?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "partial", Label: "Partial"},
				{ID: "no", Label: "No"},
			}},

		{ID: "Q198", Screen: "25", Code: "wants_to_import_transactions", Type: "single_select",
			Label: "Would you like to connect or import your financial transactions?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "later", Label: "Later"},
			}},

		// ── SCREEN 26 — Data verification ─────────────────────────────────────

		{ID: "Q199", Screen: "26", Code: "name_is_correct", Type: "boolean",
			Label: "Is your name correct, {first_name}?"},

		{ID: "Q200", Screen: "26", Code: "tax_id_is_correct", Type: "single_select",
			Label: "Is your Tax ID correct?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "no_tax_id", Label: "I don't have one"},
			}},

		{ID: "Q201", Screen: "26", Code: "address_is_correct", Type: "boolean",
			Label: "Is your address correct?"},

		{ID: "Q202", Screen: "26", Code: "residency_info_correct", Type: "boolean",
			Label: "Is your residency information correct?"},

		{ID: "Q203", Screen: "26", Code: "declared_all_income_sources", Type: "single_select",
			Label: "Have you declared every source of income?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		{ID: "Q204", Screen: "26", Code: "included_foreign_income", Type: "single_select",
			Label: "Have you included income received outside Nigeria?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no", Label: "No"},
				{ID: "not_applicable", Label: "Not applicable"},
				{ID: "not_sure", Label: "I'm not sure"},
			}},

		// ── SCREEN 27 — Tax calculation preview ───────────────────────────────

		{ID: "Q205", Screen: "27", Code: "calculation_looks_correct", Type: "single_select",
			Label: "Does this calculation look correct?",
			Options: []Option{
				{ID: "yes", Label: "Yes"},
				{ID: "no_missing", Label: "No — something is missing"},
				{ID: "need_help", Label: "I need help understanding it"},
			}},

		// ── SCREEN 28 — Deductions review ─────────────────────────────────────

		{ID: "Q206", Screen: "28", Code: "wants_deductions_review", Type: "boolean",
			Label: "Do you want to review your deductions?"},

		// ── SCREEN 29 — Final declarations ────────────────────────────────────

		{ID: "Q207", Screen: "29", Code: "income_from_all_sources_declared", Type: "boolean",
			Label: "{first_name}, have you provided income from all sources?", Required: true,
			Hint:  "Under NTAA s.13, you must declare income from every source — employment, freelance, business, rent, investments and foreign income — even where tax was already deducted at source."},

		{ID: "Q208", Screen: "29", Code: "information_true_and_complete", Type: "boolean",
			Label: "Is the information you provided true and complete to the best of your knowledge?", Required: true},

		{ID: "Q209", Screen: "29", Code: "authorises_submission", Type: "boolean",
			Label: "Do you authorise this return to be submitted on your behalf?", Required: true},

		// ── SCREEN 30 — Filing status ──────────────────────────────────────────

		{ID: "Q210", Screen: "30", Code: "filing_approach", Type: "single_select",
			Label: "How would you like to proceed?",
			Options: []Option{
				{ID: "self", Label: "File myself"},
				{ID: "agent", Label: "Have an accredited tax agent assist me"},
				{ID: "save_later", Label: "Save and finish later"},
				{ID: "need_expert", Label: "I need expert help"},
			}},
	}
}
