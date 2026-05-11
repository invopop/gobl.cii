# GOBL ↔ CDAR (Flow 6) mapping reference

This document describes how `bill.Status` documents in GOBL with the
`fr-ctc-v1` addon round-trip to and from the French CDAR
(*Compte Rendu d'Avis et de Réception* — XP Z12-012 Annex A v1.3) XML
exchanged between *Plateformes Agréées* (PAs).

The mapping is bidirectional: the same convention drives the writer
(`gobl.cii`) on the way out and the parser on the way back in.

---

## 1. Context vs lifecycle category

Two independent axes:

| Axis | Lives in | Values | Drives |
|---|---|---|---|
| **GOBL lifecycle category** | `bill.Status.Type` | `response` / `update` / `system` | The flow6 process-code lookup (`ctc.CDARProcessCodeFor(key, type)`) |
| **CDAR phase / wire ack** | `cii.Context.GuidelineID` | `urn.cpro.gouv.fr:1p0:CDV:invoice` (treatment, ack 23) or `…einvoicingF2` (transmission, ack 305) | `<ram:TypeCode>` and the trade-party shape |

`bill.Status.Type` and the ack TypeCode are **not the same thing** —
any (key, type) status can be converted under either Context. The
flow6 process table happens to align them by convention, but the wire
phase is a per-message authoring decision the caller makes via
`WithContext(...)`.

```go
// Treatment phase (ack TypeCode 23) — end-party CDV
cii.Convert(env, cii.WithContext(cii.ContextCDARFlow6))

// Transmission phase (ack TypeCode 305) — to the PPF
cii.Convert(env, cii.WithContext(cii.ContextCDARFlow6PPF))
```

---

## 2. Party slot mapping

### Wire slots

The CDAR `<rsm:ExchangedDocument>` carries four party slots and the
referenced-document block one more:

| MDT / MDG | Wire path | Required? |
|---|---|---|
| MDT-21 (`SenderTradeParty/RoleCode`) | `ExchangedDocument/SenderTradeParty` | yes |
| MDG-16 (`IssuerTradeParty`) | `ExchangedDocument/IssuerTradeParty` | yes |
| MDG-23 (`RecipientTradeParty`) | `ExchangedDocument/RecipientTradeParty` | yes |
| MDT-129 (`IssuerAssignedID` of ref.) | `ReferenceReferencedDocument/IssuerTradeParty/GlobalID` | yes (BR-FR-CDV-13) |

### `bill.Status` source

| Wire slot | `bill.Status` field | Notes |
|---|---|---|
| `SenderTradeParty` (MDT-21) | bare `WK` party by default; `WithSenderTradeParty(p)` on `Convert` to override | The platform identity is anonymous in the UC1 corpus; the option lets a caller put their identified PA on the wire (Name + GlobalID + Inbox + Role) when needed. |
| `IssuerTradeParty` (MDG-16) | `Issuer` | Direct, by-name. The role on this party tells the schematron which side is reporting (BY/SE for ack 23; WK for ack 305). |
| `RecipientTradeParty` (MDG-23) | `Recipient` | Direct, by-name. URIID required if role ≠ WK/DFH (BR-FR-CDV-08). |
| `ref.IssuerTradeParty/GlobalID` (MDT-129) | `Supplier`'s SIREN identity | The seller of the referenced invoice. Carries only the SIREN on the wire. |

### Required `bill.Status` fields

- `Supplier` — direct source for `ref.IssuerTradeParty/GlobalID`
  (MDT-129, BR-FR-CDV-13). Must carry an identity with
  `iso-scheme-id = 0002` (SIREN).
- `Issuer` — direct source for `IssuerTradeParty`. Must carry
  `ext[fr-ctc-role]` (BR-FR-CDV-CL-03 list).
- `Recipient` — direct source for `RecipientTradeParty`. Must carry
  `ext[fr-ctc-role]` plus an `inboxes` entry when role ≠ WK / DFH
  (BR-FR-CDV-08).
- `Lines[0]` — exactly one (CDAR carries one status per CDV).

`Customer` is optional invoice metadata. The flow6 addon does **not**
auto-derive any of these fields — the caller writes them explicitly.

### Parser-side fallback

The parser populates `Supplier` from `ref.IssuerTradeParty` (the
seller's canonical SIREN slot per MDT-129) when the JSON document
doesn't already carry one. That keeps a parsed status round-trippable
through the writer without forcing the caller to repeat the seller in
the JSON when they only want what was on the wire. (Auto-fill of the
*JSON* fields from each other has been removed — only the
`ref → Supplier` extraction at parse time remains.)

### Allowed roles per slot

| Slot | Ack 23 (treatment) | Ack 305 (PPF) | Source |
|---|---|---|---|
| `SenderTradeParty.RoleCode` (MDT-21) | one of `BY, DL, SE, AB, SR, WK, PE, PR, II, IV` | must be `WK` | BR-FR-CDV-CL-02 |
| `IssuerTradeParty.RoleCode` (MDT-40) | one of `BY, AB, DL, SE, SR, PE, PR, II, IV` (no WK) | must be `WK` | BR-FR-CDV-CL-03 |
| `RecipientTradeParty.RoleCode` (MDT-59) | one of `BY, DL, SE, AB, SR, PE, PR, II, IV, WK, DFH` | `DFH` (the PPF) | BR-FR-CDV-CL-04 |

### PPF identity

The Public Portal (PPF) is the unique recipient on ack 305 messages.
Its identity is fixed by the spec — `SIREN 9998`, scheme `0238`,
role `DFH`, name `PPF`. The `flow6` addon does not ship a constructor
for it; downstream French integrations supply the party (or build it
inline) when they construct an ack 305 status.

---

## 3. (`StatusLine.Key`, `Status.Type`) → `ProcessConditionCode`

Each Flow 6 process code (MDT-105) maps 1:1 to a (key, type) pair.
The key set is partly drawn from GOBL's `bill.Status*` constants; the
ones not in stock GOBL are exported by `flow6` itself.

| `bill.StatusLine.Key` | `bill.Status.Type` | Process code | Libellé | Source const |
|---|---|---|---|---|
| `issued` | `update` | **200** | Déposée | `bill.StatusEventIssued` |
| `issued` | `response` | **201** | Émise par la plateforme | `bill.StatusEventIssued` |
| `acknowledged` | `response` | **202** | Reçue | `bill.StatusEventAcknowledged` |
| `made-available` | `response` | **203** | Mise à disposition | `ctc.StatusEventMadeAvailable` |
| `processing` | `response` | **204** | Prise en charge | `bill.StatusEventProcessing` |
| `accepted` | `response` | **205** | Approuvée | `bill.StatusEventAccepted` |
| `partially-accepted` | `response` | **206** | Approuvée Partiellement | `ctc.StatusEventPartiallyAccepted` |
| `disputed` | `response` | **207** | En litige | `ctc.StatusEventDisputed` |
| `querying` | `response` | **208** | Suspendue | `bill.StatusEventQuerying` |
| `completed` | `response` | **209** | Complétée | `ctc.StatusEventCompleted` |
| `rejected` | `response` | **210** | Refusée | `bill.StatusEventRejected` |
| `paid` | `update` | **211** | Paiement transmis | `bill.StatusEventPaid` (shared) |
| `paid` | `response` | **212** | Encaissée | `bill.StatusEventPaid` (shared) |
| `error` | `response` | **213** | Rejetée (sémantique) | `bill.StatusEventError` |

`paid` is the only key that pairs with both `Status.Type` values: the
same payment event is reported under different CDAR codes depending
on whether it travels in the transmission phase (PA→PPF, code 211) or
the treatment phase (seller-side confirms receipt, code 212). When
the caller uses the `paid` key, `Type` must be set explicitly — the
flow6 normaliser cannot default it because the key alone is
ambiguous.

Helpers:

```go
flow6.CDARProcessCodeFor(line.Key, st.Type)  // → "212", true
flow6.StatusKeyFor("212")                     // → "paid", "response", true
```

---

## 4. Status line content mapping

| CDAR slot | `bill.StatusLine` field | Notes |
|---|---|---|
| `ProcessConditionCode` (MDT-105) | derived from `Key` + `Status.Type` | See table §3 |
| `IssuerAssignedID` (MDT-87) | `Doc.Code` | The referenced invoice's BT-1 |
| `FormattedIssueDateTime` (MDG-35) | `Doc.IssueDate` | BT-2 |
| `TypeCode` (MDT-91) | `Doc.Type` | UNTDID 1001 invoice type |
| `ReceiptDateTime` | not currently mapped | |
| `SpecifiedDocumentStatus[]` | one entry per `(Reason × Action)` pair, then characteristic-only entries | See §5 |

### Status-required Reasons (BR-FR-CDV-15)

A `bill.StatusLine` whose key is one of `rejected`, `error`,
`disputed`, `partially-accepted`, `suspended` MUST carry at least one
`Reason`.

### Per-status allowed `ReasonCode` (BR-FR-CDV-CL-09)

| Process code | Status | Allowed `fr-ctc-reason-code` values |
|---|---|---|
| 200 | Déposée | `NON_TRANSMISE` |
| 206 | Approuvée Part. | `AUTRE`, `CMD_ERR`, `SIRET_ERR`, `CODE_ROUTAGE_ERR`, `REF_CT_ABSENT`, `REF_ERR`, `PU_ERR`, `REM_ERR`, `QTE_ERR`, `ART_ERR`, `MODPAI_ERR`, `QUALITE_ERR`, `LIVR_INCOMP` |
| 207 | En litige | `AUTRE`, `COORD_BANC_ERR`, `TX_TVA_ERR`, `MONTANTTOTAL_ERR`, `CALCUL_ERR`, `NON_CONFORME`, `DOUBLON`, `DEST_ERR`, `TRANSAC_INC`, `EMMET_INC`, `CONTRAT_TERM`, `DOUBLE_FACT`, `CMD_ERR`, `ADR_ERR`, `SIRET_ERR`, `CODE_ROUTAGE_ERR`, `REF_CT_ABSENT`, `REF_ERR`, `PU_ERR`, `REM_ERR`, `QTE_ERR`, `ART_ERR`, `MODPAI_ERR`, `QUALITE_ERR`, `LIVR_INCOMP` |
| 208 | Suspendue | `JUSTIF_ABS`, `COORD_BANC_ERR`, `CMD_ERR`, `SIRET_ERR`, `CODE_ROUTAGE_ERR`, `REF_CT_ABSENT`, `REF_ERR` |
| 210 | Refusée | `TX_TVA_ERR`, `MONTANTTOTAL_ERR`, `CALCUL_ERR`, `NON_CONFORME`, `DOUBLON`, `DEST_ERR`, `TRANSAC_INC`, `EMMET_INC`, `CONTRAT_TERM`, `DOUBLE_FACT`, `CMD_ERR`, `ADR_ERR`, `REF_CT_ABSENT` |
| 213 | Rejetée sém. | `MONTANTTOTAL_ERR`, `CALCUL_ERR`, `DOUBLON`, `ADR_ERR`, `REJ_SEMAN`, `REJ_UNI`, `REJ_COH`, `REJ_ADR`, `REJ_CONT_B2G`, `REJ_REF_PJ`, `REJ_ASS_PJ` |

Other process codes do not constrain the reason set (the rule is
silent for them in Annexe A).

### Action codes (UNCL CDAR-style, MDT-121)

| `bill.Action.Key` | CDAR `RequestedActionCode` |
|---|---|
| `none` | `NOA` |
| `provide` | `PIN` |
| `reissue` | `NIN` |
| `credit-full` | `CNF` |
| `credit-partial` | `CNP` |
| `credit-amount` | `CNA` |
| `other` | `OTH` |

### Characteristics — `<ram:SpecifiedDocumentCharacteristic>`

A `ctc.Characteristic` complement on a status line maps to one
`SpecifiedDocumentCharacteristic`:

| CDAR field | Characteristic field |
|---|---|
| `ID` | `ID` |
| `TypeCode` (MDT-207) | `TypeCode` (one of MEN, MPA, RAP, ESC, RAB, REM, MAP, MAPTTC, MNA, MNATTC, CBB, DIV, DVA, MAJ) |
| `Name` | `Name` |
| `Location` | `Location` |
| `ValueAmount` (with `currencyID`) | `Amount` (`currency.Amount`) |
| `ValuePercent` | `Percent` |
| `ValueChangedIndicator` | `Changed` (bool) |

Special rule (BR-FR-CDV-14): a `paid` (212) status line MUST carry a
characteristic with `TypeCode = MEN` (Montant Encaissé) and
`Amount.{Value, Currency}` populated.

When a Reason and a Characteristic are linked by their CDAR ReasonCode
the writer keeps them in the same `SpecifiedDocumentStatus` entry; the
`Characteristic.ReasonCode` field, when set, must match the
`fr-ctc-reason-code` of some sibling Reason on the same line.

---

## 5. Validations enforced by `flow6` (mirror of the schematron)

The `fr-ctc-v1` addon validates everything that the deployed
schematron currently *warns* about (warnings are treated as errors —
they will become hard errors in future spec releases):

| Rule ID (gobl) | Schematron / spec | What it checks |
|---|---|---|
| BILL-STATUS-01 | — | `Type ∈ {response, update}` |
| BILL-STATUS-02 | BR-FR-CDV-13 | `Supplier` present (after normalisation) |
| BILL-STATUS-03 | — | `Supplier` carries SIREN identity |
| BILL-STATUS-04 | — | exactly one status line |
| BILL-STATUS-05 | BR-FR-CDV-10 | each line has `Doc` |
| BILL-STATUS-06 | — | line key is a known Flow 6 event |
| BILL-STATUS-07 | BR-FR-CDV-14 | `paid` (212) line has MEN characteristic with Amount |
| BILL-STATUS-08 | — | `Status.Type` consistent with line keys |
| BILL-STATUS-09 | — | `Characteristic.ReasonCode` resolves to sibling Reason |
| BILL-STATUS-10 | BR-FR-CDV-CL-11 | `Characteristic.TypeCode` is in the MDT-207 set |
| BILL-STATUS-11 | BR-FR-CDV-10 | `Doc.Code` present |
| BILL-STATUS-12 | BR-FR-CDV-11 | `Doc.IssueDate` present |
| BILL-STATUS-13 | BR-FR-CDV-15 | reasons mandatory for 206/207/208/210/213 |
| BILL-STATUS-14 | BR-FR-CDV-CL-03 | `Issuer` present (MDG-16) |
| BILL-STATUS-15 | BR-FR-CDV-CL-03 | `Issuer.ext.fr-ctc-role` present |
| BILL-STATUS-16 | BR-FR-CDV-CL-04 | `Recipient` present (MDG-23) |
| BILL-STATUS-17 | BR-FR-CDV-CL-04 | `Recipient.ext.fr-ctc-role` present |
| BILL-STATUS-18 | BR-FR-CDV-08 | `Recipient` has inbox when role ≠ WK / DFH |
| BILL-STATUS-19 | BR-FR-CDV-CL-09 | reason codes allowed for the line's process code |
| BILL-REASON-01 | — | `Reason.Key` is a known `bill.ReasonKey` |
| BILL-REASON-02 | — | `fr-ctc-reason-code` known and matches `Reason.Key` bucket |
| BILL-ACTION-01 | — | `Action.Key` is a known `bill.ActionKey` |
| ORG-PARTY-01 | BR-FR-CDV-CL-02..04 | `fr-ctc-role` is in the UNCL 3035 subset |
| ORG-PARTY-02 | — | identity scheme is in the ICD 6523 subset |

---

## 6. Worked examples

### Buyer-issued ack 23 (CDV-205, "Approuvée")

```yaml
$schema: "https://gobl.org/draft-0/bill/status"
$addons: ["fr-ctc-v1"]
issue_date: "2026-04-16"
code: "STA-2026-0001"

supplier:
  name: "VENDEUR SARL"
  identities: [{type: "SIREN", code: "732829320", ext: {iso-scheme-id: "0002"}}]
  ext: {fr-ctc-role: "SE"}

issuer:
  name: "ACHETEUR SARL"
  identities: [{type: "SIREN", code: "200000008", ext: {iso-scheme-id: "0002"}}]
  ext: {fr-ctc-role: "BY"}

recipient:
  name: "VENDEUR SARL"
  identities: [{type: "SIREN", code: "732829320", ext: {iso-scheme-id: "0002"}}]
  inboxes: [{scheme: "0225", code: "732829320_PEP"}]
  ext: {fr-ctc-role: "SE"}

lines:
  - key: "accepted"
    doc: {code: "2026-00042", issue_date: "2026-04-15"}
```

### Seller-issued ack 23 (CDV-212, "Encaissée")

Same parties, swapped roles: `Issuer` = the seller (SE), `Recipient` =
the buyer (BY).

```yaml
supplier:
  name: "VENDEUR SARL"
  identities: [{type: "SIREN", code: "732829320", ext: {iso-scheme-id: "0002"}}]
  ext: {fr-ctc-role: "SE"}

issuer:
  name: "VENDEUR SARL"
  identities: [{type: "SIREN", code: "732829320", ext: {iso-scheme-id: "0002"}}]
  ext: {fr-ctc-role: "SE"}

recipient:
  name: "ACHETEUR SARL"
  identities: [{type: "SIREN", code: "200000008", ext: {iso-scheme-id: "0002"}}]
  inboxes: [{scheme: "0225", code: "200000008_PEP"}]
  ext: {fr-ctc-role: "BY"}

lines:
  - key: "paid"
    doc: {code: "2026-00042", issue_date: "2026-04-15"}
    complements:
      - $schema: "https://gobl.org/draft-0/addons/fr/ctc/characteristic"
        type_code: "MEN"
        amount: {currency: "EUR", value: "1200.00"}
```

### PPF transmission (CDV-200, "Déposée")

`Issuer` is the WK platform; `Recipient` is the PPF; `Supplier` is the
seller of the referenced invoice (no SE party in `Issuer`/`Recipient`,
so the caller must set it explicitly).

```go
st := &bill.Status{
    Type:      bill.StatusTypeUpdate,
    Code:      "STATUS-200",
    IssueDate: cal.MakeDate(2026, time.May, 2),
    Supplier:  &org.Party{Name: "VENDEUR", Identities: ..., Ext: SE},
    Issuer:    &org.Party{Name: "PA-FR", Identities: ..., Ext: WK},
    Recipient: &org.Party{
        Name:       "PPF",
        Identities: []*org.Identity{{Code: "9998", Ext: ...0238}},
        Ext:        ...flow6.RoleDFH,
    },
    Lines:     []*bill.StatusLine{{Key: bill.StatusEventIssued, Doc: ...}},
}
cii.Convert(env, cii.WithContext(cii.ContextCDARFlow6PPF))
```
