# Health Watcher: Data Model

> **Status**: revision 1 - 2026-10-22 (D9)
> **Purpose**: how the records model lines up with HL7 FHIR R4, so a later FHIR API, export and import (HW-INT-1..3) are a mapping layer and need no schema change. [REQUIREMENTS.md §4](./REQUIREMENTS.md#4-records-model) lists what each record holds; this file says which FHIR resource and element each part becomes.

## 1. Rules

- **FHIR R4** (4.0.1). VN Core, the Vietnamese profile set at [hl7.org.vn](https://hl7.org.vn/en/knowledge/fhir-emr-vietnam/), is R4. So are Apple Health clinical records and most hospital systems.
- **Typed tables, not resource JSON.** Each record keeps its own SQLite table with constraints, because trends, search and approval query single columns. A FHIR resource is built from the rows when a client asks for it.
- **App words stay.** The UI and the JSON API say person, visit, provider, document and finding. The table in §2 maps each one to exactly one FHIR resource, and nothing in the model needs a resource that the table leaves out.
- **Codes come from the standard systems** in §3. A value read off a paper keeps the text as printed next to its code, as FHIR's `CodeableConcept.text` does, so a wrong code never hides what the paper said.
- **Only approved records leave the app.** Drafts and rejected readings are workflow state (`RecordStatus`), not clinical data. An approved record maps to `final`, an edited one to `amended`, a deleted one to `entered-in-error`.
- **IDs** are UUIDv7 strings, valid as FHIR logical ids. A resource keeps its id for life.
- **No national identifiers yet.** CCCD, BHYT and similar numbers are not stored (D9). The VN Core Patient profile needs one, so exported patients will conform to base R4 only. An identifier table can be added later without touching other tables.

## 2. Records and their FHIR resources

| App record                            | FHIR R4 resource              | Notes                                                                                                                                                                                                         |
| ------------------------------------- | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Tenant                                | none                          | Access boundary only. Becomes the compartment a token can read.                                                                                                                                               |
| Person                                | `Patient`                     | `name.text` = display name, `birthDate`, `gender` (§4.1).                                                                                                                                                     |
| Provider                              | `Organization`                | `name`, `type` from the provider kind (§4.2).                                                                                                                                                                 |
| Visit                                 | `Encounter`                   | `status` = `finished`, `class` and `type` from the visit type (§4.3), `period` from date and end date, `serviceProvider` = provider.                                                                          |
| Document                              | `DocumentReference`           | `type` = LOINC document code from the kind (§4.4), `date` = the paper's date, `content[]` = one `Attachment` per file, `context.encounter` = visit.                                                           |
| Lab result or imaging report document | `DiagnosticReport` too        | Built from the document: `category` LAB or RAD, `result[]` = its observations, `conclusion` = its imaging report findings, `presentedForm` = the original.                                                    |
| File                                  | `Binary`                      | Served as the attachment's `url`. `Attachment.hash` is SHA-1 in R4; the app keys files by SHA-256, so the hash goes in an extension.                                                                          |
| Picture                               | `Media`                       | `type` = `image`, `modality` from the kind (X-ray `DX`, ultrasound `US`, endoscopy `ES`, ECG `ECG`, CT `CT`, MRI `MR`), `content` = the cropped JPEG, `note` = the caption, `partOf` = the document's report. |
| Observation                           | `Observation`                 | §4.5.                                                                                                                                                                                                         |
| Finding `diagnosis`                   | `Condition`                   | `code` = ICD-10 (§3) with the text, `category` = `encounter-diagnosis`, `verificationStatus` (§4.6).                                                                                                          |
| Finding `prescription`                | `MedicationRequest`           | `intent` = `order`, `medicationCodeableConcept.text` = drug, `dosageInstruction.text` = dose, frequency and duration as written.                                                                              |
| Finding `procedure`                   | `Procedure`                   | `status` = `completed`, `code.text`, `performedDateTime` = visit date.                                                                                                                                        |
| Finding `allergy`                     | `AllergyIntolerance`          | `code.text` = the substance as written.                                                                                                                                                                       |
| Finding `vaccination`                 | `Immunization`                | `status` = `completed`, `vaccineCode.text` = the vaccine, `doseQuantity` = the amount as written, `occurrenceDateTime` = the dose's date, else the visit date.                                                |
| Finding `exam`                        | `Observation`                 | `category` = `exam`, `code.text` = section, `valueString` = text.                                                                                                                                             |
| Finding `imaging_report`              | `DiagnosticReport.conclusion` | Part of the document's report (row above), not a resource of its own.                                                                                                                                         |
| Finding `advice`                      | `CarePlan`                    | `status` = `active`, `intent` = `plan`, `activity.detail.description` = text, `activity.detail.scheduledString` = its date when it has one, such as a vaccine dose that is due.                               |
| Finding `note`                        | `Encounter` annotation        | Extension on the encounter; FHIR R4 `Encounter` has no `note`.                                                                                                                                                |
| Record versions                       | `meta.versionId`, `_history`  | §5.                                                                                                                                                                                                           |
| Who and what made a record            | `Provenance`                  | §5.                                                                                                                                                                                                           |

## 3. Code systems

`domain/terminology.go` holds these URIs and the fixed maps from app values to codes.

| Use                                         | System                                                                                                           |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Lab tests and vital signs                   | LOINC `http://loinc.org` (catalog, HW-MET-1)                                                                     |
| Document types                              | LOINC document codes                                                                                             |
| Units                                       | UCUM `http://unitsofmeasure.org`                                                                                 |
| Diagnoses                                   | ICD-10 `http://hl7.org/fhir/sid/icd-10`; VN Core's Vietnamese ICD-10 system can replace it once its URI is final |
| Result interpretation                       | `http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation`                                             |
| Observation category                        | `http://terminology.hl7.org/CodeSystem/observation-category`                                                     |
| Encounter class                             | `http://terminology.hl7.org/CodeSystem/v3-ActCode`                                                               |
| Organization type                           | `http://terminology.hl7.org/CodeSystem/organization-type`                                                        |
| Condition statuses                          | `http://terminology.hl7.org/CodeSystem/condition-clinical`, `condition-ver-status`                               |
| Report category                             | `http://terminology.hl7.org/CodeSystem/v2-0074`                                                                  |
| App-only values (visit type, document kind) | `https://health-watcher.local/fhir/CodeSystem/<name>`                                                            |

## 4. Value maps

### 4.1 Person sex to `Patient.gender`

`female` → `female`, `male` → `male`, empty → `unknown`. The app keeps sex at birth for reference ranges; FHIR's `gender` is administrative, which is the same for this app's use.

### 4.2 Provider kind to `Organization.type`

`hospital`, `clinic`, `dental`, `lab` and `pharmacy` → `prov` (healthcare provider), `other` → `other`. The app kind also goes in as a second coding under the app system, since the HL7 organization type list has no finer codes for these.

### 4.3 Visit type to `Encounter.class`

`inpatient` → `IMP`, `emergency` → `EMER`, everything else → `AMB`. The visit type itself is `Encounter.type` under the app system. An inpatient stay has an end date (`visits.end_date`); other visits have only a start.

### 4.4 Document kind to LOINC document code

Every code was checked against the NLM Clinical Tables LOINC service on 2026-10-22.

| Kind                | LOINC                                          |
| ------------------- | ---------------------------------------------- |
| `lab_result`        | 11502-2 Laboratory report                      |
| `imaging_report`    | 18748-4 Diagnostic imaging study               |
| `imaging_film`      | 18748-4, with the film photo as the attachment |
| `prescription`      | 57833-6 Prescription for medication            |
| `discharge_summary` | 18842-5 Discharge summary                      |
| `dental_record`     | 28618-7 Dentistry Note                         |
| `record_book_page`  | 11506-3 Progress note                          |
| `invoice`           | 94093-2 Itemized bill                          |
| `other`             | 34109-9 Note                                   |

### 4.5 Observation

| Column                              | FHIR                                                                                                                  |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `analyte`                           | `code.text` (as printed)                                                                                              |
| `metric_code`                       | `code.coding` with the catalog entry's LOINC code                                                                     |
| (derived)                           | `category`: `vital-signs` for a catalog vital sign, else `laboratory` (`ObservationCategory`)                         |
| `value_num`, `unit`                 | `valueQuantity`: `value`, `unit` as printed, `system` UCUM and `code` from `UCUMUnit` when the catalog knows the unit |
| `value` with a comparator (`< 0.5`) | `valueQuantity` with `comparator` (`<`, `<=`, `>=`, `>`) from `ParseComparator`                                       |
| `value` when not a number           | `valueString`                                                                                                         |
| `ref_low`, `ref_high`, `ref_text`   | `referenceRange.low`, `.high` in the same unit, `.text`                                                               |
| `flag`                              | `interpretation`: `H` → `H`, `L` → `L`, `*` → `A`                                                                     |
| `sample_date`                       | `effectiveDateTime`; the visit date when empty                                                                        |
| `document_id`, `page_id`            | `derivedFrom` the document; the page goes into Provenance. Vitals typed without a paper (HW-MET-2) have neither       |

Category, UCUM code and comparator are derived from the stored columns when a resource is built, not stored. `value_num` stays empty for a value with a comparator, so charts never draw `< 0.5` as 0.5.

### 4.6 Diagnosis to `Condition.verificationStatus`

Vietnamese papers mark a suspected diagnosis with `TD` (theo dõi, under observation), and that maps to `provisional`. Every other diagnosis maps to `confirmed`. `ConditionVerification` derives it from the text, so a corrected text corrects the status.

## 5. Versions and provenance

Every observation and finding has a `version` number, starting at 1 and raised by each edit of an approved record. Edits of a draft keep its version. The `audit_log` table keeps each earlier version of an approved record: who changed or deleted it, when, the version, and the old JSON. That is enough for `vread` and `_history` later.

Each observation and finding records where it came from:

| Column                       | Meaning                                                                                                                              | FHIR `Provenance`                |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------- |
| `source`                     | `reading` (AI reading of a document), `manual` (typed by a person) or `import` (another system)                                      | `activity`                       |
| `source_detail`              | The model for a reading, the import id for an import                                                                                 | `agent` (device or organization) |
| `page_id`                    | The page it was read from                                                                                                            | `entity` (source)                |
| `approved_by`, `approved_at` | Who approved the draft, and when. A record typed by hand is approved by its author when saved. Empty for records approved before D9. | `agent` (verifier), `recorded`   |

## 6. Export gaps

Some fields are optional while a person works but a complete export needs them. `domain/export.go` lists them, and each visit, observation and finding reports its own in `exportGaps`:

| Gap        | Record                                  | How to fill it                                                                                                                                          |
| ---------- | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `end_date` | An inpatient visit without its last day | _Last day of the stay_ in the visit form.                                                                                                               |
| `approver` | An approved record with no approver     | _I checked them_ on the document page records the editor as approver of the document's records that have none. The values and version stay as they are. |

The document date is never a gap: the visit date stands in for it. The export (HW-INT-2) lists every gap and starts only when none is left.

## 7. Not modelled yet

- National identifiers (D9).
- Coded medications. No free Vietnamese drug code list was found; drugs stay as text.
- Structured dosage (`Timing`). Dose, frequency and duration stay as written.
- FHIR JSON itself. Nothing builds resources yet; when HW-INT-1 does, tests check its output against the official R4 JSON schema (D9).
- Blood pressure as one panel with two components. It stays two observations (systolic, diastolic), which FHIR also accepts.
