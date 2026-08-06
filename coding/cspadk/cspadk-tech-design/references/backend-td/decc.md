# DECC / Cross-Region Compliance

**Load this file only if the feature moves data across RoW / EU TTP / US TTP (including CN<->RoW, BOE<->RoW).**

Cross-region data transfer at ByteDance is governed by **DECC** (Data Export Compliance Control), the control plane over DES-RPC/DES-HTTP. **Unregistered fields in real traffic block the entire request** (not filtered) — so this section must be settled before code merges.

Reference: `https://bytedance.larkoffice.com/wiki/OMNKwlYG9ir0hjkNyFTcR243nlc`. DECC web UI: `https://decc.tiktok-row.net/v3/des-rpc`. Support: DECC Oncall (`/des_rpc`, `https://deccsupport.aipa.bytedance.net`).

## Scope of this section
This section covers **RPC/HTTP** cross-region calls via DES-RPC. **RDS cross-region sync** and **MQ cross-region sync** go through separate gateways (DES-RDS / DES-MQ) with their own approval flows — if the feature uses those, flag it explicitly and route the user to DECC Oncall for the current process rather than guessing.

## Required fields in the TD

| Field | Content |
|---|---|
| Applicable Scenario(s) | Texas (US TTP) / Clover (EU TTP) / CN跨境 / TT<>Non-TT / RoW<>BOE |
| Physical Direction | e.g. `RoW-TT -> CN`, `BOE -> RoW-TT` (must be a SUBSET of the labeled direction) |
| Callee PSM + Service Type | `RPC-PSM` / `HTTP-PSM` / `HTTP-DOMAIN` |
| IDL Location in BAM | `https://cloud.bytedance.net/bam/` entry + git source; list methods to register |
| Field-level Label Plan | per field: Data Catalog, Data Subject, User Region, Data Type (`Default` / `Aggregated User Data` / `BinaryData` / `ExemptionData`), Description (English), 同步原因 |
| Caller PSM(s) + Topology | caller→callee, VGeo direction |
| Caller Code Changes | kitex: `callopt.WithIDC(env.DC_USEAST5)` / `callopt.WithVRegion(...)`; HTTP mesh: `destination-idc` / `destination-vregion` headers; Mesh switch enabled on TCE/CronJob/FaaS |
| Approval Expectation | auto vs manual (US is always manual; channel p0 triggers data-plane review) |
| Data-type Risk Callouts | binary data NOT allowed for TT<>NonTT / CN<>SG / BOE<>ROW; TT user data NOT allowed on `RoW-TT -> CN` or `RoW-TT -> Non-TT` |

## Mandatory design-time actions
1. **Classify scenario** — propose Texas / Clover / CN跨境 / TT-NonTT / RoW-BOE via `AskUserQuestion` if ambiguous; get user confirmation before proceeding.
2. **Enumerate every field crossing the boundary** and draft the label plan table above. Do not defer this — the label plan shapes the IDL.
3. **Identify unsupported payloads early**: `thrift binary` / `list<byte>` is rejected by TT<>NonTT, CN<>SG, BOE<>ROW. Route binary/file transfer via DES-TOS, not DES-RPC.
4. **Check for TT user data in forbidden directions**: `RoW-TT -> CN` and `RoW-TT -> Non-TT` auto-reject TT normal user data. If the feature requires this, flag as a blocker — design needs rework before TD review.
5. **Pre-existing registrations**: check whether callee PSM is already registered for the target scenario. Some legacy PSMs live on the v2 platform; if the new platform says "please go to v2", note this in the TD.

**Cross-team work required**

| Team / PSM Owner | What they need to do | When |
|---|---|---|

## DECC pitfalls to call out in the TD

- **Adding IDL fields without a new label version.** Cross-region fields that aren't in the Applied label version block the *entire* request, not just the unregistered field. IDL changes on cross-region services need a label-plan update *before* code merges.
- **Forgetting caller code changes.** Registering topology alone doesn't route traffic. Kitex callers need `callopt.WithIDC` / `WithVRegion`; HTTP-mesh callers need `destination-idc` / `destination-vregion` / `destination-domain` headers; TCE/CronJob/FaaS need the **Mesh switch** enabled.
- **Binary/file transfer over DES-RPC.** TT<>NonTT, CN<>SG, and BOE<>ROW reject `thrift binary` / `list<byte>`. Route file transfer via DES-TOS, not as RPC fields.
- **TT user data in forbidden directions.** `RoW-TT -> CN` and `RoW-TT -> Non-TT` auto-reject TT normal user data. Catch this at design time, not at approval.
- **HTTP services skipping BAM IDL registration.** HTTP endpoints still need BAM IDL registration with `api.post` / `api.body` / `api.query` / `api.header` / `api.path` annotations; commonly forgotten.
- **Assuming RDS and MQ cross-region sync work the same as RPC.** This section only documents DES-RPC/HTTP. RDS and MQ cross-region sync go through separate gateways with their own flows — route users to DECC Oncall rather than guessing.
