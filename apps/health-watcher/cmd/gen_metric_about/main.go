// Command gen_metric_about writes domain/catalog_about_data.go: for each
// catalog metric, a word-for-word quote from MedlinePlus, or from the NCI
// Dictionary of Cancer Terms where MedlinePlus has none, that says what the
// metric is (HW-MET-6). Run it from apps/health-watcher:
//
//	go run ./cmd/gen_metric_about
//
// The quotes are curated below. The command fetches each page summary from
// MedlinePlus Connect by LOINC code, or each NCI term's definition, and
// fails if a quote is not found verbatim in that text, so a changed page is
// caught before the file is rewritten.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"html"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"apps/health-watcher/domain"
)

const connectURL = "https://connect.medlineplus.gov/service"

// about picks the quote for one metric: the Connect entry titled Title,
// looked up by LOINC (the catalog's code when empty), or, when NCITerm is
// set, that term's definition in the NCI Dictionary of Cancer Terms.
type about struct {
	Code    string
	LOINC   string
	Title   string
	NCITerm string
	Quote   string
}

var abouts = []about{
	{Code: "glucose", Title: "Blood Glucose Test", Quote: `A blood glucose test measures the glucose levels in your blood. Glucose is a type of sugar. It is your body's main source of energy.`},
	{Code: "hba1c", Title: "Hemoglobin A1C (HbA1c) Test", Quote: `A hemoglobin A1C (HbA1C) test is a blood test that shows your average level of blood glucose, also called blood sugar, over the past two to three months.`},
	{Code: "insulin", Title: "Insulin in Blood Test", Quote: `An insulin in blood test measures the amount of insulin in a sample of your blood. Insulin is a hormone that your pancreas makes.`},
	{Code: "cholesterol_total", Title: "Cholesterol", Quote: `Cholesterol is a waxy, fat-like substance that's found in all the cells in your body. Your body needs some cholesterol to make hormones, vitamin D, and substances that help you digest foods.`},
	{Code: "triglycerides", Title: "Triglycerides", Quote: `Triglycerides are a type of fat. They are the most common type of fat in your body. They come from foods, especially butter, oils, and other fats you eat.`},
	{Code: "hdl", Title: "Cholesterol", Quote: `HDL stands for high-density lipoprotein. It is sometimes called "good" cholesterol because it helps your body get rid of cholesterol. It carries cholesterol from other parts of your body back to your liver.`},
	{Code: "ldl", Title: "Cholesterol", Quote: `LDL stands for low-density lipoprotein. It is sometimes called "bad" cholesterol because a high LDL level leads to the buildup of plaque in your arteries.`},
	// MedlinePlus has no page for urea itself; the BUN page explains it.
	{Code: "urea", LOINC: "3094-0", Title: "BUN (Blood Urea Nitrogen)", Quote: `Urea nitrogen is a waste product that forms as your body breaks down proteins. It's carried in your blood and then removed by your kidneys when you urinate (pee).`},
	{Code: "bun", Title: "BUN (Blood Urea Nitrogen)", Quote: `A BUN, or blood urea nitrogen test, measures the amount of urea nitrogen in your blood. Urea nitrogen is a waste product that forms as your body breaks down proteins.`},
	{Code: "creatinine", Title: "Creatinine Test", Quote: `Creatinine is a normal waste product in your body. It's made when you use your muscles and some of the muscle tissue breaks down. Normally, your kidneys filter creatinine from your blood and remove it from your body in your urine.`},
	{Code: "egfr", Title: "Estimated Glomerular Filtration Rate (eGFR) and GFR Tests", Quote: `Estimated glomerular filtration rate (eGFR) and GFR tests check how well your kidneys are working.`},
	{Code: "uric_acid", LOINC: "3084-1", Title: "Uric Acid Test", Quote: `Uric acid is a normal waste product that your body makes when it breaks down chemicals called purines. These chemicals come from your cells when they die and can be found in certain foods and beverages.`},
	{Code: "acr", Title: "Microalbumin Creatinine Ratio", Quote: `A microalbumin creatinine ratio urine test checks the levels of the protein albumin in your urine (pee). Abnormal amounts of this protein may be an early sign of kidney disease.`},
	{Code: "ast", Title: "AST Test", Quote: `AST (aspartate aminotransferase) is an enzyme, a protein that speeds up certain chemical reactions in your body. It is found mainly in your liver but also in your heart, muscles, and other tissues.`},
	{Code: "alt", Title: "ALT Blood Test", Quote: `ALT (alanine transaminase) is an enzyme, a protein that speeds up certain chemical reactions in your body. It is found mainly in your liver.`},
	{Code: "ggt", Title: "Gamma-glutamyl Transferase (GGT) Test", Quote: `The GGT enzyme is found throughout your body, but it is mainly found in your liver. If your liver or bile duct is damaged, GGT may leak into your bloodstream.`},
	{Code: "alp", Title: "Alkaline Phosphatase", Quote: `An alkaline phosphatase (ALP) test measures the amount of ALP in your blood. ALP is an enzyme, a protein that speeds up certain chemical reactions in your body. ALP is found in all your body tissues, but higher amounts can be found in your liver, bile ducts, and bones.`},
	{Code: "bilirubin_total", Title: "Bilirubin Blood Test", Quote: `A bilirubin blood test measures the levels of bilirubin in your blood. Bilirubin is a yellowish substance made during your body's normal process of breaking down old red blood cells.`},
	{Code: "bilirubin_direct", Title: "Bilirubin Blood Test", Quote: `Bilirubin is a yellowish substance made during your body's normal process of breaking down old red blood cells. Bilirubin is found in bile, a fluid your liver makes to help you digest food.`},
	{Code: "albumin", Title: "Albumin Blood Test", Quote: `An albumin blood test measures the amount of albumin in your blood. Albumin is a protein made by your liver.`},
	{Code: "protein_total", Title: "Liver Function Tests", Quote: `Total protein. This test measures the total amount of protein in your blood, which includes albumin and globulins. These proteins are mainly made in your liver.`},
	{Code: "wbc", Title: "White Blood Count (WBC)", Quote: `A white blood count measures the number of white blood cells (WBCs) in your blood. White blood cells, also called leukocytes, are part of your immune system.`},
	{Code: "rbc", Title: "Red Blood Cell (RBC) Count", Quote: `A red blood cell (RBC) count measures the number of red blood cells, also known as erythrocytes, in your blood. Red blood cells are made in your bone marrow, the spongy tissue inside your large bones. They contain hemoglobin, an iron-rich protein that carries oxygen from your lungs to every cell in your body.`},
	{Code: "hemoglobin", Title: "Hemoglobin Test", Quote: `Hemoglobin is an iron-rich protein in your red blood cells. It carries oxygen from your lungs to the rest of your body.`},
	{Code: "hematocrit", Title: "Hematocrit Test", Quote: `A hematocrit test is a blood test that measures the amount (percent) of your blood that is made up of red blood cells. Red blood cells carry oxygen from your lungs to the rest of your body.`},
	{Code: "mcv", Title: "MCV (Mean Corpuscular Volume)", Quote: `MCV stands for mean corpuscular volume. An MCV blood test measures the average size of your red blood cells.`},
	{Code: "mch", Title: "Red Blood Cell (RBC) Indices", Quote: `Mean corpuscular hemoglobin (MCH), which measures the average amount of hemoglobin in a single red blood cell.`},
	{Code: "mchc", Title: "Red Blood Cell (RBC) Indices", Quote: `Mean corpuscular hemoglobin concentration (MCHC), which measures how concentrated (close together) the hemoglobin is in your red blood cells.`},
	{Code: "rdw", Title: "RDW (Red Cell Distribution Width)", Quote: `A red cell distribution width (RDW) test measures how much the volume and size of your red blood cells (erythrocytes) varies.`},
	{Code: "platelets", Title: "Platelet Tests", Quote: `Platelets, also known as thrombocytes, are small blood cells. They form in your bone marrow, the sponge-like tissue in your bones. Platelets are essential for blood clotting.`},
	{Code: "mpv", Title: "MPV Blood Test", Quote: `An MPV blood test measures the mean (average) size of your platelets. Looking at the size of platelets provides information about how well they're working.`},
	{Code: "neutrophils_pct", Title: "Blood Differential", Quote: `Neutrophils are the most common type of white blood cell. They are your body's main defense against infection when bacteria, viruses, or other germs enter your body.`},
	{Code: "neutrophils_abs", Title: "Blood Differential", Quote: `Neutrophils are the most common type of white blood cell. They are your body's main defense against infection when bacteria, viruses, or other germs enter your body.`},
	{Code: "lymphocytes_pct", Title: "Blood Differential", Quote: `Lymphocytes include two main types of white blood cells: B cells and T cells. B cells fight off invading viruses, bacteria, or toxins. Certain T cells can target and destroy your body's own cells, such as cancer cells and cells that have been infected by viruses.`},
	{Code: "lymphocytes_abs", Title: "Blood Differential", Quote: `Lymphocytes include two main types of white blood cells: B cells and T cells. B cells fight off invading viruses, bacteria, or toxins. Certain T cells can target and destroy your body's own cells, such as cancer cells and cells that have been infected by viruses.`},
	{Code: "monocytes_pct", Title: "Blood Differential", Quote: `Monocytes kill bacteria, viruses, and other germs that may make you sick. They also boost your body's immune response and clear away dead cells.`},
	{Code: "eosinophils_pct", Title: "Blood Differential", Quote: `Eosinophils defend against parasites and infections. They are also involved in allergies and help control inflammation (swelling and redness).`},
	{Code: "basophils_pct", Title: "Blood Differential", Quote: `Basophils release enzymes during allergic reactions and asthma attacks.`},
	{Code: "tsh", Title: "TSH (Thyroid-stimulating hormone) Test", Quote: `TSH stands for thyroid stimulating hormone. Another name for it is thyrotropin. A TSH test is a blood test that measures this hormone.`},
	{Code: "ft4", Title: "Thyroxine (T4) Test", Quote: `T4 is the main hormone that your thyroid makes. There are two forms of it in your blood: Free T4 is the active form of thyroxine hormone that enters your body tissues where you need it.`},
	{Code: "ft3", Title: "Triiodothyronine (T3) Tests", Quote: `There are two forms of T3 in your blood: Free T3 is the active form that enters your tissues where it's needed.`},
	{Code: "sodium", Title: "Sodium Blood Test", Quote: `A sodium blood test measures the amount of sodium in your blood. Sodium is a type of electrolyte. Electrolytes are minerals that have an electrical charge when they are dissolved in water or body fluids.`},
	{Code: "potassium", Title: "Potassium Blood Test", Quote: `Potassium is a type of electrolyte. Electrolytes are electrically charged minerals that help control the amount of fluids and the balance of acids and bases (pH balance) in your body.`},
	{Code: "chloride", Title: "Chloride Blood Test", Quote: `Chloride is a type of electrolyte. Electrolytes are electrically charged minerals that help control the amount of fluids and the balance of acids and bases (pH balance) in your body.`},
	{Code: "calcium", Title: "Calcium Blood Test", Quote: `Calcium is one of the most important minerals in your body. About 1% of the calcium in your body is in your blood. The rest is stored in your bones and teeth.`},
	{Code: "crp", Title: "C-Reactive Protein (CRP) Test", Quote: `A c-reactive protein test measures the level of c-reactive protein (CRP) in a sample of your blood. Your liver makes CRP in response to inflammation.`},
	{Code: "hscrp", Title: "C-Reactive Protein (CRP) Test", Quote: `An hs-CRP test measures very tiny increases in your CRP levels. It is used to estimate your risk of heart disease.`},
	{Code: "esr", Title: "Erythrocyte Sedimentation Rate (ESR)", Quote: `Erythrocytes are red blood cells. The sedimentation rate is the time it takes for your red blood cells to settle at the bottom of a test tube. An erythrocyte sedimentation rate (ESR) is a blood test that can show if you have inflammation in your body.`},
	{Code: "ferritin", Title: "Ferritin Blood Test", Quote: `A ferritin blood test measures the level of ferritin in your blood. Ferritin is a protein that binds to iron and stores it in your body.`},
	{Code: "iron", Title: "Iron Tests", Quote: `Serum iron test, which measures the amount of iron in the blood`},
	{Code: "vitamin_d", Title: "Vitamin D Test", Quote: `A vitamin D test measures the levels of vitamin D in your blood. Vitamin D helps your body absorb calcium to build healthy bones and teeth. It also helps keep your muscles, nerves, and immune system working normally.`},
	{Code: "vitamin_b12", Title: "Vitamin B Test", Quote: `B12, (cobalamin) gives your body energy and helps keep your nerves and blood cells healthy.`},
	{Code: "folate", Title: "Vitamin B Test", Quote: `B9, (folate or folic acid) helps your body make healthy new cells.`},
	{Code: "hbsag", Title: "Hepatitis Testing", Quote: `Hepatitis B Surface Antigen (HBsAg), an antigen on the surface of the hepatitis B virus.`},
	{Code: "anti_hbs", Title: "Hepatitis Testing", Quote: `Hepatitis B Surface-Antibody (Anti-HBs), an antibody that your immune system makes to fight HBV after a hepatitis B infection or vaccination.`},
	// The page has no sentence on the hepatitis C antibody alone.
	{Code: "anti_hcv", Title: "Hepatitis Testing", Quote: `Hepatitis antibodies, which are proteins that your immune system makes to help fight a hepatitis infection. If you have hepatitis antibodies in your blood, it may mean that you have an acute or chronic infection or that you had an infection in the past.`},
	{Code: "psa", Title: "Prostate-Specific Antigen (PSA) Test", Quote: `A prostate-specific antigen (PSA) test is a blood test that measures the level of PSA in a sample of your blood. PSA is a protein made by your prostate.`},
	// The AFP page covers pregnancy screening; adults read AFP as a tumor marker.
	{Code: "afp", LOINC: "2039-6", Title: "Tumor Marker Tests", Quote: `AFP (Alpha-fetoprotein) Tumor marker for: liver cancer, and cancers of the ovary or testicles`},
	{Code: "cea", Title: "CEA Test", Quote: `A CEA test measures how much of a specific protein called carcinoembryonic antigen (CEA) you have.`},
	{Code: "pt", Title: "Prothrombin Time Test and INR (PT/INR)", Quote: `A prothrombin time (PT) test measures how many seconds it takes for a clot to form in a blood sample.`},
	{Code: "inr", Title: "Prothrombin Time Test and INR (PT/INR)", Quote: `Since the results of this test can vary with different labs or test methods, your results may be used to create an INR (international normalized ratio). Using this ratio makes it easier for health care providers to compare and better understand results from different labs or test methods.`},
	{Code: "aptt", Title: "Partial Thromboplastin Time (PTT) Test", Quote: `A partial thromboplastin time (PTT) test uses a blood sample to measure the number of seconds it takes for your blood to clot.`},
	// Fibrinogen is a clotting factor; the page names it but does not define it alone.
	{Code: "fibrinogen", Title: "Coagulation Factor Tests", Quote: `Coagulation factors are proteins in your blood. They help form blood clots to stop bleeding when you have an injury.`},
	{Code: "amylase", Title: "Amylase Test", Quote: `Amylase is an enzyme, a protein that speeds up chemical reactions in your body. Amylase helps you digest carbohydrates. Most of the amylase in your body is made by your pancreas and salivary (spit) glands.`},
	{Code: "lipase", Title: "Lipase Tests", Quote: `A lipase test measures the level of lipase in your blood. Lipase helps your body digest fats. It's a type of digestive enzyme.`},
	{Code: "ldh", Title: "Lactate Dehydrogenase (LDH) Test", Quote: `LDH helps your cells make energy. It is found in almost all the tissues in your body.`},
	{Code: "ck", Title: "Creatine Kinase", Quote: `CK helps make energy. Most of the CK in your body is in your skeletal muscles. These are the muscles you use to move.`},
	{Code: "urine_protein", LOINC: "2888-6", Title: "Protein in Urine", Quote: `A protein in urine test measures how much protein is in your urine (pee). Normally, you have very little protein in your urine.`},
	{Code: "urine_glucose", Title: "Glucose in Urine Test", Quote: `A glucose in urine test measures the amount of glucose in your urine (pee).`},
	{Code: "urine_ketones", Title: "Ketones in Urine", Quote: `If your cells can't get enough glucose, your body breaks down fat for energy instead. This produces an acid called ketones, which can build up in your blood and urine.`},
	{Code: "urine_ph", Title: "Urinalysis", Quote: `The pH level (acidity)`},
	// The nitrites page is the one that speaks of white blood cells in urine.
	{Code: "urine_leukocytes", LOINC: "5802-4", Title: "Nitrites in Urine", Quote: `White blood cells in urine may also mean you have a UTI.`},
	{Code: "urine_nitrite", Title: "Nitrites in Urine", Quote: `Nitrites in your urine may be a sign of a urinary tract infection (UTI). Different types of bacteria can cause UTIs. Many of these types of bacteria change a normal chemical in your urine, called nitrates, into another chemical, called nitrites.`},
	{Code: "urine_blood", LOINC: "5799-2", Title: "Blood in Urine", Quote: `Having blood in your urine is called hematuria. The blood may make your urine or the water in the toilet bowl turn pink, red, or reddish brown. But you might not always be able to see the blood without a lab test.`},
	{Code: "urine_bilirubin", Title: "Bilirubin in Urine", Quote: `A bilirubin in urine test measures the levels of bilirubin in your urine. Normally, bilirubin isn't found in urine. If there is bilirubin in your urine (called bilirubinuria), it may be an early sign of a liver condition.`},
	{Code: "urine_urobilinogen", Title: "Urobilinogen in Urine", Quote: `A urobilinogen in urine test measures the amount of urobilinogen in your urine (pee). Urobilinogen comes from bilirubin.`},
	{Code: "weight", Title: "Body Weight", Quote: `Body weight is the total mass of your body. Keeping track of your weight is one way to check for changes in your health, but it can't tell you the whole story.`},
	{Code: "bmi", Title: "Body Weight", Quote: `BMI is a measure of how much you weigh compared to how tall you are. The higher your BMI, the higher your risk for certain diseases.`},
	{Code: "bp_systolic", Title: "Measuring Blood Pressure", Quote: `Systolic blood pressure (the first and higher number) measures pressure inside your arteries when your heart beats.`},
	{Code: "bp_diastolic", Title: "Measuring Blood Pressure", Quote: `Diastolic blood pressure (the second and lower number) measures the pressure inside the artery when your heart rests between beats.`},
	{Code: "heart_rate", Title: "Vital Signs", Quote: `Heart rate, or pulse, which measures how fast your heart is beating.`},
	{Code: "temperature", Title: "Vital Signs", Quote: `Temperature, which measures how hot your body is.`},
	{Code: "spo2", Title: "Pulse Oximetry", Quote: `Pulse oximetry is a test that uses a small, clip-like device called a pulse oximeter (Pulse Ox) to measure oxygen levels in your blood.`},
	// MedlinePlus never says what granulocytes are; the NCI dictionary does.
	// Neither source defines the Mid cells of a 3-part analyser.
	{Code: "granulocytes_pct", NCITerm: "granulocyte", Title: "granulocyte", Quote: `A type of immune cell that has granules (small particles) with enzymes that are released during infections, allergic reactions, and asthma. Neutrophils, eosinophils, and basophils are granulocytes.`},
	{Code: "granulocytes_abs", NCITerm: "granulocyte", Title: "granulocyte", Quote: `A type of immune cell that has granules (small particles) with enzymes that are released during infections, allergic reactions, and asthma. Neutrophils, eosinophils, and basophils are granulocytes.`},
	// Connect has no entry for the waist LOINC codes; the body weight page covers it.
	{Code: "waist", LOINC: "29463-7", Title: "Body Weight", Quote: `Waist circumference. If most of your body fat is around your abdomen (belly) rather than your hips, you may be at a higher risk for heart disease and type 2 diabetes.`},
}

// entry is one page in a Connect response.
type entry struct {
	Title   string
	URL     string
	Summary string // plain text
}

func main() {
	out := flag.String("out", "domain/catalog_about_data.go", "file to write")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "gen_metric_about:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	byCode, err := checkTable()
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	pages := map[string][]entry{}
	resolved := map[string]domain.MetricAbout{}
	var problems []string
	for _, a := range abouts {
		if a.NCITerm != "" {
			e, err := fetchNCI(client, a.NCITerm)
			if err != nil {
				return err
			}
			if !strings.Contains(e.Summary, a.Quote) {
				problems = append(problems, fmt.Sprintf("%s: quote not found verbatim in the NCI definition of %q (%s)", a.Code, a.NCITerm, e.URL))
				continue
			}
			resolved[a.Code] = domain.MetricAbout{Quote: a.Quote, Source: domain.AboutSourceNCI, Title: e.Title, URL: e.URL}
			continue
		}
		loinc := a.LOINC
		if loinc == "" {
			m, _ := domain.DefaultCatalog.Metric(a.Code)
			loinc = m.LOINC
		}
		entries, ok := pages[loinc]
		if !ok {
			entries, err = fetch(client, loinc)
			if err != nil {
				return err
			}
			pages[loinc] = entries
		}
		e, ok := findEntry(entries, a.Title)
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: LOINC %s has no entry titled %q", a.Code, loinc, a.Title))
			continue
		}
		if !strings.Contains(e.Summary, a.Quote) {
			problems = append(problems, fmt.Sprintf("%s: quote not found verbatim in %q (%s)", a.Code, e.Title, e.URL))
			continue
		}
		resolved[a.Code] = domain.MetricAbout{Quote: a.Quote, Title: e.Title, URL: e.URL}
	}
	if len(problems) > 0 {
		return errors.New("quotes do not match their source:\n  " + strings.Join(problems, "\n  "))
	}
	src, err := render(resolved, time.Now().Format(time.DateOnly))
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("wrote %s: %d of %d metrics described\n", out, len(resolved), len(domain.DefaultCatalog.Metrics()))
	for _, m := range domain.DefaultCatalog.Metrics() {
		if _, ok := byCode[m.Code]; !ok {
			fmt.Printf("no description: %s (%s, LOINC %s)\n", m.Code, m.Name, m.LOINC)
		}
	}
	return nil
}

// checkTable rejects codes missing from the catalog and duplicate rows.
func checkTable() (map[string]about, error) {
	byCode := map[string]about{}
	for _, a := range abouts {
		if _, ok := domain.DefaultCatalog.Metric(a.Code); !ok {
			return nil, fmt.Errorf("code %q is not in the catalog", a.Code)
		}
		if _, dup := byCode[a.Code]; dup {
			return nil, fmt.Errorf("code %q is listed twice", a.Code)
		}
		if strings.TrimSpace(a.Quote) == "" || a.Title == "" {
			return nil, fmt.Errorf("code %q needs a title and a quote", a.Code)
		}
		byCode[a.Code] = a
	}
	return byCode, nil
}

// fetch asks MedlinePlus Connect for the pages about a LOINC code.
func fetch(client *http.Client, loinc string) ([]entry, error) {
	q := url.Values{}
	q.Set("mainSearchCriteria.v.cs", "2.16.840.1.113883.6.1")
	q.Set("mainSearchCriteria.v.c", loinc)
	q.Set("knowledgeResponseType", "application/json")
	resp, err := client.Get(connectURL + "?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("fetch LOINC %s: %w", loinc, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch LOINC %s: status %s", loinc, resp.Status)
	}
	var body struct {
		Feed struct {
			Entry []struct {
				Title struct {
					Value string `json:"_value"`
				} `json:"title"`
				Link []struct {
					Href string `json:"href"`
				} `json:"link"`
				Summary struct {
					Value string `json:"_value"`
				} `json:"summary"`
			} `json:"entry"`
		} `json:"feed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode LOINC %s: %w", loinc, err)
	}
	entries := make([]entry, 0, len(body.Feed.Entry))
	for _, e := range body.Feed.Entry {
		if len(e.Link) == 0 {
			continue
		}
		href, _, _ := strings.Cut(e.Link[0].Href, "?")
		entries = append(entries, entry{Title: e.Title.Value, URL: href, Summary: plainText(e.Summary.Value)})
	}
	return entries, nil
}

// nciURL is the NCI Dictionary of Cancer Terms API for patients, in English.
const nciURL = "https://webapis.cancer.gov/glossary/v1/Terms/Cancer.gov/Patient/en/"

// fetchNCI reads a term's definition from the NCI Dictionary of Cancer Terms.
func fetchNCI(client *http.Client, term string) (entry, error) {
	resp, err := client.Get(nciURL + url.PathEscape(term))
	if err != nil {
		return entry{}, fmt.Errorf("fetch NCI term %s: %w", term, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return entry{}, fmt.Errorf("fetch NCI term %s: status %s", term, resp.Status)
	}
	var body struct {
		TermName   string `json:"termName"`
		PrettyURL  string `json:"prettyUrlName"`
		Definition struct {
			Text string `json:"text"`
		} `json:"definition"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return entry{}, fmt.Errorf("decode NCI term %s: %w", term, err)
	}
	return entry{Title: body.TermName, URL: "https://www.cancer.gov/publications/dictionaries/cancer-terms/def/" + body.PrettyURL,
		Summary: plainText(body.Definition.Text)}, nil
}

func findEntry(entries []entry, title string) (entry, bool) {
	for _, e := range entries {
		if e.Title == title {
			return e, true
		}
	}
	return entry{}, false
}

var (
	blockTag  = regexp.MustCompile(`(?i)</?(p|h[1-6]|li|ul|ol|br|div|table|tr|td|th|dl|dt|dd)\b[^>]*>`)
	inlineTag = regexp.MustCompile(`<[^>]*>`)
)

// plainText strips the summary's HTML the way a reader sees it: block
// tags separate words, inline tags vanish, entities are decoded and
// whitespace is collapsed.
func plainText(s string) string {
	s = blockTag.ReplaceAllString(s, " ")
	s = inlineTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

// render writes the generated Go file, entries in catalog order.
func render(resolved map[string]domain.MetricAbout, fetched string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("// Code generated by cmd/gen_metric_about; DO NOT EDIT.\n\npackage domain\n\n")
	b.WriteString("// AboutFetched is the day the quotes were checked against their sources.\n")
	fmt.Fprintf(&b, "const AboutFetched = %s\n\n", strconv.Quote(fetched))
	b.WriteString("var metricAbout = map[string]MetricAbout{\n")
	for _, m := range domain.DefaultCatalog.Metrics() {
		a, ok := resolved[m.Code]
		if !ok {
			continue
		}
		source := ""
		if a.Source == domain.AboutSourceNCI {
			source = "Source: AboutSourceNCI,\n"
		}
		fmt.Fprintf(&b, "%s: {\n%sTitle: %s,\nURL: %s,\nQuote: %s,\n},\n",
			strconv.Quote(m.Code), source, strconv.Quote(a.Title), strconv.Quote(a.URL), strconv.Quote(a.Quote))
	}
	b.WriteString("}\n")
	src, err := format.Source(b.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated file: %w", err)
	}
	return src, nil
}
