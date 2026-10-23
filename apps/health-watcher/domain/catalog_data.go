package domain

// CatalogVersion changes whenever DefaultCatalog's entries change.
const CatalogVersion = "2026-10-23.1"

// Metric categories.
const (
	CatGlucose      MetricCategory = "glucose"
	CatLipids       MetricCategory = "lipids"
	CatKidney       MetricCategory = "kidney"
	CatLiver        MetricCategory = "liver"
	CatBloodCount   MetricCategory = "blood_count"
	CatThyroid      MetricCategory = "thyroid"
	CatElectrolytes MetricCategory = "electrolytes"
	CatInflammation MetricCategory = "inflammation"
	CatIronVitamins MetricCategory = "iron_vitamins"
	CatHepatitis    MetricCategory = "hepatitis"
	CatTumorMarkers MetricCategory = "tumor_markers"
	CatCoagulation  MetricCategory = "coagulation"
	CatEnzymes      MetricCategory = "enzymes"
	CatUrine        MetricCategory = "urine"
	CatVitals       MetricCategory = "vitals"
)

func conv(from string, factor float64) Conversion { return Conversion{From: from, Factor: factor} }

// DefaultCatalog is the curated metric list shipped with the app (HW-MET-1).
// Conversion factors are the usual clinical ones (e.g. glucose mg/dL ÷
// 18.016). LOINC codes were entered by hand from the LOINC table; check a
// code at loinc.org before relying on it outside this app.
var DefaultCatalog = NewCatalog(CatalogVersion, []Metric{
	// Glucose
	{Code: "glucose", LOINC: "14749-6", Name: "Glucose", NameVI: "Glucose máu", Category: CatGlucose, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 1/18.016)},
		Aliases:     []string{"glu", "đường huyết", "đường máu", "glucose đói", "glucose lúc đói", "fasting glucose", "fpg", "blood glucose", "glucose huyết tương"}},
	{Code: "hba1c", LOINC: "4548-4", Name: "HbA1c", NameVI: "HbA1c", Category: CatGlucose, Unit: "%",
		Conversions: []Conversion{{From: "mmol/mol", Factor: 0.09148, Offset: 2.152}},
		Aliases:     []string{"a1c", "hemoglobin a1c", "hb a1c", "glycated hemoglobin", "glycohemoglobin"}},
	{Code: "insulin", LOINC: "20448-7", Name: "Insulin", NameVI: "Insulin", Category: CatGlucose, Unit: "µIU/mL",
		Conversions: []Conversion{conv("pmol/L", 0.144), conv("mIU/L", 1)}},

	// Lipids
	{Code: "cholesterol_total", LOINC: "14647-2", Name: "Total cholesterol", NameVI: "Cholesterol toàn phần", Category: CatLipids, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 0.02586)},
		Aliases:     []string{"cholesterol", "cholesterol tp", "chol", "tc", "total cholesterol"}},
	{Code: "triglycerides", LOINC: "14927-8", Name: "Triglycerides", NameVI: "Triglycerid", Category: CatLipids, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 0.01129)},
		Aliases:     []string{"triglyceride", "triglycerid", "tg", "trig"}},
	{Code: "hdl", LOINC: "14646-4", Name: "HDL cholesterol", NameVI: "HDL-Cholesterol", Category: CatLipids, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 0.02586)},
		Aliases:     []string{"hdl", "hdl-c", "cholesterol hdl", "hdl cholesterol"}},
	{Code: "ldl", LOINC: "22748-8", Name: "LDL cholesterol", NameVI: "LDL-Cholesterol", Category: CatLipids, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 0.02586)},
		Aliases:     []string{"ldl", "ldl-c", "cholesterol ldl", "ldl cholesterol"}},

	// Kidney
	{Code: "urea", LOINC: "22664-7", Name: "Urea", NameVI: "Urê máu", Category: CatKidney, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 0.1665)},
		Aliases:     []string{"ure", "urea", "urê", "ure máu", "urea máu"}},
	{Code: "bun", LOINC: "3094-0", Name: "Blood urea nitrogen", NameVI: "BUN", Category: CatKidney, Unit: "mg/dL",
		Aliases: []string{"bun", "urea nitrogen", "nitơ ure"}},
	{Code: "creatinine", LOINC: "14682-9", Name: "Creatinine", NameVI: "Creatinin", Category: CatKidney, Unit: "µmol/L",
		Conversions: []Conversion{conv("mg/dL", 88.42)},
		Aliases:     []string{"creatinin", "creatinine", "cre", "crea", "creatinin máu"}},
	{Code: "egfr", LOINC: "62238-1", Name: "eGFR", NameVI: "Mức lọc cầu thận ước tính", Category: CatKidney, Unit: "mL/min/1.73m²",
		Aliases: []string{"egfr", "gfr", "mlct", "mức lọc cầu thận", "egfr ckd-epi", "egfr (ckd-epi)", "ckd-epi"}},
	{Code: "uric_acid", LOINC: "14933-6", Name: "Uric acid", NameVI: "Acid uric", Category: CatKidney, Unit: "µmol/L",
		Conversions: []Conversion{conv("mg/dL", 59.48)},
		Aliases:     []string{"acid uric", "axit uric", "uric acid", "ua", "a. uric", "urat"}},
	{Code: "acr", LOINC: "9318-7", Name: "Urine albumin/creatinine ratio", NameVI: "Tỷ số albumin/creatinin niệu", Category: CatKidney, Unit: "mg/g",
		Conversions: []Conversion{conv("mg/mmol", 8.84)},
		Aliases:     []string{"acr", "uacr", "microalbumin/creatinin", "tỷ số albumin creatinin", "albumin/creatinin"}},

	// Liver
	{Code: "ast", LOINC: "1920-8", Name: "AST (GOT)", NameVI: "AST (GOT)", Category: CatLiver, Unit: "U/L",
		Aliases: []string{"ast", "got", "sgot", "aspartate aminotransferase"}},
	{Code: "alt", LOINC: "1742-6", Name: "ALT (GPT)", NameVI: "ALT (GPT)", Category: CatLiver, Unit: "U/L",
		Aliases: []string{"alt", "gpt", "sgpt", "alanine aminotransferase"}},
	{Code: "ggt", LOINC: "2324-2", Name: "GGT", NameVI: "GGT", Category: CatLiver, Unit: "U/L",
		Aliases: []string{"ggt", "gamma gt", "γgt", "gamma glutamyl transferase"}},
	{Code: "alp", LOINC: "6768-6", Name: "Alkaline phosphatase", NameVI: "Phosphatase kiềm", Category: CatLiver, Unit: "U/L",
		Aliases: []string{"alp", "pal", "phosphatase kiềm", "alkaline phosphatase"}},
	{Code: "bilirubin_total", LOINC: "14631-6", Name: "Total bilirubin", NameVI: "Bilirubin toàn phần", Category: CatLiver, Unit: "µmol/L",
		Conversions: []Conversion{conv("mg/dL", 17.1)},
		Aliases:     []string{"bilirubin", "bilirubin tp", "total bilirubin", "tbil", "bil-t", "bilirubin t"}},
	{Code: "bilirubin_direct", LOINC: "14629-0", Name: "Direct bilirubin", NameVI: "Bilirubin trực tiếp", Category: CatLiver, Unit: "µmol/L",
		Conversions: []Conversion{conv("mg/dL", 17.1)},
		Aliases:     []string{"bilirubin tt", "direct bilirubin", "dbil", "bil-d", "bilirubin d"}},
	{Code: "albumin", LOINC: "1751-7", Name: "Albumin", NameVI: "Albumin", Category: CatLiver, Unit: "g/L",
		Conversions: []Conversion{conv("g/dL", 10)},
		Aliases:     []string{"alb", "albumin máu"}},
	{Code: "protein_total", LOINC: "2885-2", Name: "Total protein", NameVI: "Protein toàn phần", Category: CatLiver, Unit: "g/L",
		Conversions: []Conversion{conv("g/dL", 10)},
		Aliases:     []string{"protein tp", "total protein", "protid", "protein máu"}},

	// Blood count
	{Code: "wbc", LOINC: "6690-2", Name: "White blood cells", NameVI: "Số lượng bạch cầu", Category: CatBloodCount, Unit: "G/L",
		Aliases: []string{"wbc", "bạch cầu", "white blood cell", "white blood cells", "leukocytes", "sl bạch cầu"}, WholeName: true},
	{Code: "rbc", LOINC: "789-8", Name: "Red blood cells", NameVI: "Số lượng hồng cầu", Category: CatBloodCount, Unit: "T/L",
		Aliases: []string{"rbc", "hồng cầu", "red blood cell", "red blood cells", "erythrocytes", "sl hồng cầu"}, WholeName: true},
	{Code: "hemoglobin", LOINC: "718-7", Name: "Hemoglobin", NameVI: "Huyết sắc tố", Category: CatBloodCount, Unit: "g/L",
		Conversions: []Conversion{conv("g/dL", 10)},
		Aliases:     []string{"hgb", "hb", "hemoglobin", "haemoglobin", "lượng huyết sắc tố", "hst"}, WholeName: true},
	{Code: "hematocrit", LOINC: "4544-3", Name: "Hematocrit", NameVI: "Thể tích khối hồng cầu", Category: CatBloodCount, Unit: "%",
		Conversions: []Conversion{conv("L/L", 100)},
		Aliases:     []string{"hct", "hematocrit", "thể tích hồng cầu"}},
	{Code: "mcv", LOINC: "787-2", Name: "MCV", NameVI: "Thể tích trung bình hồng cầu", Category: CatBloodCount, Unit: "fL",
		Aliases: []string{"mcv"}},
	{Code: "mch", LOINC: "785-6", Name: "MCH", NameVI: "Lượng huyết sắc tố trung bình hồng cầu", Category: CatBloodCount, Unit: "pg",
		Aliases: []string{"mch", "lượng hst trung bình hồng cầu"}},
	{Code: "mchc", LOINC: "786-4", Name: "MCHC", NameVI: "Nồng độ huyết sắc tố trung bình hồng cầu", Category: CatBloodCount, Unit: "g/L",
		Conversions: []Conversion{conv("g/dL", 10)},
		Aliases:     []string{"mchc", "nồng độ hst trung bình hồng cầu"}},
	{Code: "rdw", LOINC: "788-0", Name: "RDW", NameVI: "Độ phân bố kích thước hồng cầu", Category: CatBloodCount, Unit: "%",
		Aliases: []string{"rdw", "rdw-cv", "dải phân bố hồng cầu"}},
	{Code: "platelets", LOINC: "777-3", Name: "Platelets", NameVI: "Số lượng tiểu cầu", Category: CatBloodCount, Unit: "G/L",
		Aliases: []string{"plt", "tiểu cầu", "platelet", "platelet count", "sl tiểu cầu"}, WholeName: true},
	{Code: "mpv", LOINC: "32623-1", Name: "MPV", NameVI: "Thể tích trung bình tiểu cầu", Category: CatBloodCount, Unit: "fL",
		Aliases: []string{"mpv"}},
	{Code: "neutrophils_pct", LOINC: "770-8", Name: "Neutrophils %", NameVI: "Tỷ lệ bạch cầu trung tính", Category: CatBloodCount, Unit: "%",
		Aliases: []string{"neu%", "neut%", "neu", "neut", "neutrophil %", "neutrophils %", "bạch cầu trung tính %", "trung tính %", "trung tính"}},
	{Code: "neutrophils_abs", LOINC: "751-8", Name: "Neutrophils", NameVI: "Số lượng bạch cầu trung tính", Category: CatBloodCount, Unit: "G/L",
		Aliases: []string{"neu#", "neut#", "neu", "neut", "neutrophils #", "bạch cầu trung tính #", "trung tính #", "trung tính"}},
	{Code: "lymphocytes_pct", LOINC: "736-9", Name: "Lymphocytes %", NameVI: "Tỷ lệ bạch cầu lympho", Category: CatBloodCount, Unit: "%",
		Aliases: []string{"lym%", "lymph%", "lym", "lymph", "lymphocytes %", "bạch cầu lympho %", "lympho %", "lympho"}},
	{Code: "lymphocytes_abs", LOINC: "731-0", Name: "Lymphocytes", NameVI: "Số lượng bạch cầu lympho", Category: CatBloodCount, Unit: "G/L",
		Aliases: []string{"lym#", "lymph#", "lym", "lymph", "lymphocytes #", "bạch cầu lympho #", "lympho #", "lympho"}},
	{Code: "monocytes_pct", LOINC: "5905-5", Name: "Monocytes %", NameVI: "Tỷ lệ bạch cầu mono", Category: CatBloodCount, Unit: "%",
		Aliases: []string{"mono%", "mono", "monocytes %", "bạch cầu mono %", "bạch cầu mono"}},
	{Code: "eosinophils_pct", LOINC: "713-8", Name: "Eosinophils %", NameVI: "Tỷ lệ bạch cầu ưa acid", Category: CatBloodCount, Unit: "%",
		Aliases: []string{"eos%", "eos", "eosinophils %", "bạch cầu ưa acid", "bạch cầu ái toan", "tỷ lệ bạch cầu ái toan"}},
	{Code: "basophils_pct", LOINC: "706-2", Name: "Basophils %", NameVI: "Tỷ lệ bạch cầu ưa base", Category: CatBloodCount, Unit: "%",
		Aliases: []string{"baso%", "baso", "basophils %", "bạch cầu ưa base", "bạch cầu ái kiềm", "tỷ lệ bạch cầu ái kiềm"}},

	// Thyroid
	{Code: "tsh", LOINC: "3016-3", Name: "TSH", NameVI: "TSH", Category: CatThyroid, Unit: "mIU/L",
		Conversions: []Conversion{conv("µIU/mL", 1)},
		Aliases:     []string{"tsh", "thyrotropin"}},
	{Code: "ft4", LOINC: "14920-3", Name: "Free T4", NameVI: "FT4", Category: CatThyroid, Unit: "pmol/L",
		Conversions: []Conversion{conv("ng/dL", 12.87)},
		Aliases:     []string{"ft4", "free t4", "t4 tự do"}},
	{Code: "ft3", LOINC: "14928-6", Name: "Free T3", NameVI: "FT3", Category: CatThyroid, Unit: "pmol/L",
		Conversions: []Conversion{conv("pg/mL", 1.536)},
		Aliases:     []string{"ft3", "free t3", "t3 tự do"}},

	// Electrolytes
	{Code: "sodium", LOINC: "2951-2", Name: "Sodium", NameVI: "Natri", Category: CatElectrolytes, Unit: "mmol/L",
		Conversions: []Conversion{conv("mEq/L", 1)},
		Aliases:     []string{"na", "na+", "natri", "sodium"}},
	{Code: "potassium", LOINC: "2823-3", Name: "Potassium", NameVI: "Kali", Category: CatElectrolytes, Unit: "mmol/L",
		Conversions: []Conversion{conv("mEq/L", 1)},
		Aliases:     []string{"k", "k+", "kali", "potassium"}},
	{Code: "chloride", LOINC: "2075-0", Name: "Chloride", NameVI: "Clo", Category: CatElectrolytes, Unit: "mmol/L",
		Conversions: []Conversion{conv("mEq/L", 1)},
		Aliases:     []string{"cl", "cl-", "clo", "chloride", "clorua"}},
	{Code: "calcium", LOINC: "2000-8", Name: "Calcium", NameVI: "Calci toàn phần", Category: CatElectrolytes, Unit: "mmol/L",
		Conversions: []Conversion{conv("mg/dL", 0.2495)},
		Aliases:     []string{"ca", "calci", "canxi", "calcium", "ca toàn phần", "calci tp"}},

	// Inflammation
	{Code: "crp", LOINC: "1988-5", Name: "C-reactive protein", NameVI: "CRP", Category: CatInflammation, Unit: "mg/L",
		Conversions: []Conversion{conv("mg/dL", 10)},
		Aliases:     []string{"crp", "crp định lượng", "protein c phản ứng", "c-reactive protein"}},
	{Code: "hscrp", LOINC: "30522-7", Name: "hs-CRP", NameVI: "CRP siêu nhạy", Category: CatInflammation, Unit: "mg/L",
		Conversions: []Conversion{conv("mg/dL", 10)},
		Aliases:     []string{"hs-crp", "hscrp", "crp-hs", "crp siêu nhạy", "high sensitivity crp"}},
	{Code: "esr", LOINC: "30341-2", Name: "ESR", NameVI: "Tốc độ máu lắng", Category: CatInflammation, Unit: "mm/h",
		Aliases: []string{"esr", "máu lắng", "vs", "vss", "tốc độ lắng máu"}},

	// Iron and vitamins
	{Code: "ferritin", LOINC: "2276-4", Name: "Ferritin", NameVI: "Ferritin", Category: CatIronVitamins, Unit: "ng/mL",
		Conversions: []Conversion{conv("µg/L", 1)},
		Aliases:     []string{"ferritin"}},
	{Code: "iron", LOINC: "14798-3", Name: "Serum iron", NameVI: "Sắt huyết thanh", Category: CatIronVitamins, Unit: "µmol/L",
		Conversions: []Conversion{conv("µg/dL", 0.1791)},
		Aliases:     []string{"sắt", "fe", "iron", "serum iron", "sắt huyết thanh"}},
	{Code: "vitamin_d", LOINC: "62292-8", Name: "25-OH vitamin D", NameVI: "Vitamin D (25-OH)", Category: CatIronVitamins, Unit: "ng/mL",
		Conversions: []Conversion{conv("nmol/L", 0.4006)},
		Aliases:     []string{"vitamin d", "vit d", "25-oh vitamin d", "25-oh d", "25(oh)d", "vitamin d3", "vitamin d total"}},
	{Code: "vitamin_b12", LOINC: "2132-9", Name: "Vitamin B12", NameVI: "Vitamin B12", Category: CatIronVitamins, Unit: "pg/mL",
		Conversions: []Conversion{conv("pmol/L", 1.355)},
		Aliases:     []string{"b12", "vit b12", "cobalamin"}},
	{Code: "folate", LOINC: "2284-8", Name: "Folate", NameVI: "Acid folic", Category: CatIronVitamins, Unit: "ng/mL",
		Conversions: []Conversion{conv("nmol/L", 0.4413)},
		Aliases:     []string{"folate", "acid folic", "axit folic", "folic acid"}},

	// Hepatitis
	{Code: "hbsag", LOINC: "5196-1", Name: "HBsAg", NameVI: "HBsAg", Category: CatHepatitis, Unit: "",
		Aliases: []string{"hbsag", "hbs ag", "hbsag test nhanh", "kháng nguyên bề mặt viêm gan b"}},
	{Code: "anti_hbs", LOINC: "16935-9", Name: "Anti-HBs", NameVI: "Anti-HBs", Category: CatHepatitis, Unit: "mIU/mL",
		Conversions: []Conversion{conv("U/L", 1)},
		Aliases:     []string{"anti-hbs", "anti hbs", "hbsab", "hbs ab", "kháng thể hbs"}},
	{Code: "anti_hcv", LOINC: "16128-1", Name: "Anti-HCV", NameVI: "Anti-HCV", Category: CatHepatitis, Unit: "",
		Aliases: []string{"anti-hcv", "anti hcv", "hcv ab", "kháng thể hcv"}},

	// Tumour markers
	{Code: "psa", LOINC: "2857-1", Name: "PSA", NameVI: "PSA toàn phần", Category: CatTumorMarkers, Unit: "ng/mL",
		Aliases: []string{"psa", "total psa", "tpsa", "psa tp"}},
	{Code: "afp", LOINC: "1834-1", Name: "AFP", NameVI: "AFP", Category: CatTumorMarkers, Unit: "ng/mL",
		Aliases: []string{"afp", "alpha fetoprotein", "alpha-fetoprotein"}},
	{Code: "cea", LOINC: "2039-6", Name: "CEA", NameVI: "CEA", Category: CatTumorMarkers, Unit: "ng/mL",
		Aliases: []string{"cea"}},

	// Coagulation
	{Code: "pt", LOINC: "5902-2", Name: "Prothrombin time", NameVI: "Thời gian prothrombin", Category: CatCoagulation, Unit: "s",
		Aliases: []string{"pt", "tq", "prothrombin time", "thời gian prothrombin", "pt (giây)"}},
	{Code: "inr", LOINC: "6301-6", Name: "INR", NameVI: "INR", Category: CatCoagulation, Unit: "",
		Aliases: []string{"inr", "pt-inr", "pt inr"}},
	{Code: "aptt", LOINC: "3173-2", Name: "APTT", NameVI: "APTT", Category: CatCoagulation, Unit: "s",
		Aliases: []string{"aptt", "tck", "thời gian thromboplastin từng phần hoạt hóa"}},
	{Code: "fibrinogen", LOINC: "3255-7", Name: "Fibrinogen", NameVI: "Fibrinogen", Category: CatCoagulation, Unit: "g/L",
		Conversions: []Conversion{conv("mg/dL", 0.01)},
		Aliases:     []string{"fibrinogen", "fib"}},

	// Enzymes
	{Code: "amylase", LOINC: "1798-8", Name: "Amylase", NameVI: "Amylase", Category: CatEnzymes, Unit: "U/L", Aliases: []string{"amylase", "amylase máu"}},
	{Code: "lipase", LOINC: "3040-3", Name: "Lipase", NameVI: "Lipase", Category: CatEnzymes, Unit: "U/L", Aliases: []string{"lipase"}},
	{Code: "ldh", LOINC: "2532-0", Name: "LDH", NameVI: "LDH", Category: CatEnzymes, Unit: "U/L", Aliases: []string{"ldh", "lactate dehydrogenase"}},
	{Code: "ck", LOINC: "2157-6", Name: "Creatine kinase", NameVI: "CK", Category: CatEnzymes, Unit: "U/L", Aliases: []string{"ck", "cpk", "creatine kinase"}},

	// Urine test strip
	{Code: "urine_protein", LOINC: "20454-5", Name: "Urine protein", NameVI: "Protein niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"protein", "pro", "prot", "protein nước tiểu"}},
	{Code: "urine_glucose", LOINC: "25428-4", Name: "Urine glucose", NameVI: "Glucose niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"glucose", "glu", "đường niệu"}},
	{Code: "urine_ketones", LOINC: "2514-8", Name: "Urine ketones", NameVI: "Thể ceton niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"ketone", "ketones", "ket", "ceton", "thể ceton", "cetonic"}},
	{Code: "urine_ph", LOINC: "5803-2", Name: "Urine pH", NameVI: "pH niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"ph"}},
	{Code: "urine_sg", LOINC: "5811-5", Name: "Urine specific gravity", NameVI: "Tỷ trọng niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"sg", "tỷ trọng", "specific gravity"}},
	{Code: "urine_leukocytes", LOINC: "5799-2", Name: "Urine leukocytes", NameVI: "Bạch cầu niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"leu", "bạch cầu", "leukocytes", "leukocyte esterase"}, WholeName: true},
	{Code: "urine_nitrite", LOINC: "5802-4", Name: "Urine nitrite", NameVI: "Nitrit niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"nit", "nitrit", "nitrite"}},
	{Code: "urine_blood", LOINC: "5794-3", Name: "Urine blood", NameVI: "Hồng cầu niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"ery", "bld", "hồng cầu", "blood", "máu"}, WholeName: true},
	{Code: "urine_bilirubin", LOINC: "5770-3", Name: "Urine bilirubin", NameVI: "Bilirubin niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"bil", "bilirubin"}},
	{Code: "urine_urobilinogen", LOINC: "5818-0", Name: "Urine urobilinogen", NameVI: "Urobilinogen niệu", Category: CatUrine, Unit: "", Urine: true,
		Aliases: []string{"ubg", "uro", "urobilinogen"}},

	// Vitals and body
	{Code: "weight", LOINC: "29463-7", Name: "Body weight", NameVI: "Cân nặng", Category: CatVitals, Unit: "kg",
		Conversions: []Conversion{conv("lb", 0.45359237)},
		Aliases:     []string{"weight", "cân nặng", "trọng lượng", "body weight"}},
	{Code: "height", LOINC: "8302-2", Name: "Body height", NameVI: "Chiều cao", Category: CatVitals, Unit: "cm",
		Conversions: []Conversion{conv("m", 100), conv("in", 2.54)},
		Aliases:     []string{"height", "chiều cao", "body height"}},
	{Code: "bmi", LOINC: "39156-5", Name: "BMI", NameVI: "Chỉ số khối cơ thể", Category: CatVitals, Unit: "kg/m²",
		Aliases: []string{"bmi", "chỉ số bmi", "body mass index"}},
	{Code: "bp_systolic", LOINC: "8480-6", Name: "Systolic blood pressure", NameVI: "Huyết áp tâm thu", Category: CatVitals, Unit: "mmHg",
		Aliases: []string{"huyết áp tâm thu", "ha tâm thu", "systolic", "sbp", "systolic blood pressure"}},
	{Code: "bp_diastolic", LOINC: "8462-4", Name: "Diastolic blood pressure", NameVI: "Huyết áp tâm trương", Category: CatVitals, Unit: "mmHg",
		Aliases: []string{"huyết áp tâm trương", "ha tâm trương", "diastolic", "dbp", "diastolic blood pressure"}},
	{Code: "heart_rate", LOINC: "8867-4", Name: "Heart rate", NameVI: "Nhịp tim", Category: CatVitals, Unit: "/min",
		Aliases: []string{"heart rate", "nhịp tim", "mạch", "pulse", "hr", "tần số tim"}},
	{Code: "temperature", LOINC: "8310-5", Name: "Body temperature", NameVI: "Nhiệt độ", Category: CatVitals, Unit: "°C",
		Conversions: []Conversion{{From: "°F", Factor: 5.0 / 9.0, Offset: -32 * 5.0 / 9.0}},
		Aliases:     []string{"temperature", "nhiệt độ", "thân nhiệt", "body temperature"}},
	{Code: "spo2", LOINC: "59408-5", Name: "SpO2", NameVI: "SpO2", Category: CatVitals, Unit: "%",
		Aliases: []string{"spo2", "sp02", "độ bão hòa oxy"}},
	{Code: "waist", LOINC: "8280-0", Name: "Waist circumference", NameVI: "Vòng eo", Category: CatVitals, Unit: "cm",
		Aliases: []string{"waist", "vòng eo", "waist circumference", "vòng bụng"}},
})
