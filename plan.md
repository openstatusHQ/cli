# Sync plan: `openstatus terraform generate` ↔ terraform-provider-openstatus

**Goal.** Bring the HCL produced by `openstatus terraform generate` into one-to-one alignment with the schema of `terraform-provider-openstatus@v0.2.0`. Every workspace resource must round-trip through `terraform import → terraform plan` with **no drift** and **no validation errors**.

**Inputs.**
- Local: `internal/terraform/` (`generate.go`, `fetch.go`, `hcl.go`, `enums.go`, `regions.go`, `naming.go`, `generate_test.go`).
- Provider: `github.com/openstatusHQ/terraform-provider-openstatus` @ `main` (v0.2.0).
- Proto API: `github.com/openstatusHQ/openstatus/packages/proto/api/openstatus/v1` (monitor, notification, status_page, plus unused maintenance / status_report).
- Pinned SDK after `go get -u`: `buf.build/gen/go/openstatus/api/...@v1.36.11-20260512200453-7d7b7047611f.1`. Every symbol referenced below is verified present in this pin.

**Out of scope.** Provider-side changes; new RPCs; non-export commands. The generator is read-only: it consumes List+Get RPCs and writes HCL.

---

## 1. What syncs and what doesn't

| Provider resource (v0.2.0) | Provider attrs / blocks | Generator today | Action |
|---|---|---|---|
| `openstatus_http_monitor` | name, url, periodicity, method, body, timeout, degraded_at, retry, follow_redirects, active, public, description, regions; blocks: headers, status_code_assertions, body_assertions, header_assertions, **open_telemetry** | All scalar fields ✓; all blocks except `open_telemetry` ✓ | §3.1 add `open_telemetry` |
| `openstatus_tcp_monitor` | name, uri, periodicity, timeout, degraded_at, retry, active, public, description, regions; **open_telemetry** | Scalars ✓; no blocks | §3.1 add `open_telemetry` |
| `openstatus_dns_monitor` | …+ record_assertions, **open_telemetry** | Scalars + record_assertions ✓ | §3.1 add `open_telemetry` |
| `openstatus_notification` | name, provider_type, monitor_ids; 13 inner blocks incl. **ms_teams** | 12 inner blocks; `ms_teams` missing | §3.2 |
| `openstatus_status_page` | title, slug, description, homepage_url, contact_url, icon, custom_domain, access_type, password, **auth_email_domains, allowed_ip_ranges, theme, default_locale, locales, allow_index** | Only the first 8 + conditional password | §3.3 |
| `openstatus_status_page_component_group` | page_id, name, **default_open** | page_id, name only | §3.4 |
| `openstatus_status_page_component` | page_id, type, monitor_id, name, description, order, group_id, group_order | Complete ✓ | none |

**No provider resource exists** for maintenances or status reports. The generator ignores them entirely — no fetch, no summary, no sidecar files. Document upstream if/when the provider gains them.

**Provider version constraint** in generated `provider.tf` is still `~> 0.1.0`; the v0.2.0 schema additions require `~> 0.2`. Fix in §3.0.

---

## 2. Strings to keep verbatim

These are the exact tf-string values the provider's `OneOf` validators accept (cross-checked against `internal/monitor/common.go` and `internal/statuspage/*` in the provider repo). The generator's existing `enums.go`/`regions.go` helpers already emit these correctly except where flagged:

- `periodicity`: `30s`, `1m`, `5m`, `10m`, `30m`, `1h`
- `method`: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`, `TRACE`, `CONNECT`
- regions: 28 values — `fly-{ams,arn,bom,cdg,dfw,ewr,fra,gru,iad,jnb,lax,lhr,nrt,ord,sjc,sin,syd,yyz}`, `koyeb-{fra,par,sfo,sin,tyo,was}`, `railway-{us-west2,us-east4,europe-west4,asia-southeast1}`
- number comparator: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`
- string comparator: + `contains`, `not_contains`, `empty`, `not_empty`
- record comparator: `eq`, `neq`, `contains`, `not_contains`
- DNS record: `A`, `AAAA`, `CNAME`, `MX`, `TXT`
- notification `provider_type`: `discord`, `email`, `slack`, `pagerduty`, `opsgenie`, `webhook`, `telegram`, `sms`, `whatsapp`, `google_chat`, `grafana_oncall`, `ntfy`, **`ms_teams`** (missing today)
- opsgenie region: `us`, `eu`
- page component type: `monitor`, `static`
- status page `access_type`: `public`, `password`, `email-domain`, **`ip`** (missing today)
- status page `theme`: `system`, `light`, `dark`
- locale: `en`, `fr`, `de`

---

## 2b. Determinism rules

The generator must produce byte-identical output when re-run against an unchanged workspace, so re-export diffs stay readable. Rule:

- **Sort** (sets — order is irrelevant to the provider): `monitor_ids`, `regions`.
- **Sort** (lists where order is presentational only): `locales`, `auth_email_domains`.
- **Preserve API order** (lists where order may be meaningful to the user): assertion lists (`status_code_assertions`, `body_assertions`, `header_assertions`, `record_assertions`), monitor `headers`, webhook headers, OTEL headers.
- **Preserve API order** for top-level resources (monitors, notifications, pages) — the API returns them in roughly `created_at` order, which is stable enough.

Apply via `sort.Strings(...)` before each affected emission. Helper not needed; ~5 LOC total.

---

## 3. Sync checklist

Each item is independent and can ship as a separate PR. Suggested order is roughly safety-first (correctness bugs before drift fixes).

### 3.0 Bump provider version constraint in generated `provider.tf`

`internal/terraform/hcl.go:91-103` — `GenerateProviderFile`.

```diff
-      version = "~> 0.1.0"
+      version = "~> 0.2"
```

Test: update `TestGenerateProviderFile` in `generate_test.go:13-18`.

Pair with §3.6 init-upgrade hint so users on a previously-generated workspace know to run `terraform init -upgrade` after re-running the command.

---

### 3.1 Monitors: emit `open_telemetry` block on HTTP, TCP, DNS

Three monitor builders in `internal/terraform/hcl.go` lines 109, 151, 180. Add a helper alongside `writeRegions`:

```go
// hcl.go — new helper, place near writeHeaders
func writeOpenTelemetry(b *hclwrite.Body, ot *monitorv1.OpenTelemetryConfig) {
    if ot == nil {
        return
    }
    endpoint := ot.GetEndpoint()
    headers := ot.GetHeaders()
    if endpoint == "" && len(headers) == 0 {
        return
    }
    otb := b.AppendNewBlock("open_telemetry", nil).Body()
    if endpoint != "" {
        otb.SetAttributeValue("endpoint", cty.StringVal(endpoint))
    }
    for _, h := range headers {
        if h.GetKey() == "" {
            continue
        }
        hb := otb.AppendNewBlock("headers", nil).Body()
        hb.SetAttributeValue("key", cty.StringVal(h.GetKey()))
        hb.SetAttributeValue("value", cty.StringVal(h.GetValue()))
    }
}
```

Call it from each monitor branch, after the assertion writers (HTTP) or after `writeRegions` (TCP/DNS).

SDK getters used: `*HTTPMonitor.GetOpenTelemetry()`, `*TCPMonitor.GetOpenTelemetry()`, `*DNSMonitor.GetOpenTelemetry()` — confirmed present in pinned SDK.

Tests: extend `TestGenerateMonitorsFile_HTTP` (and add TCP/DNS variants) to assert that an HTTP monitor with `OpenTelemetryConfig{Endpoint:"https://otel.example.com/v1/metrics", Headers:[{X-Api-Key,secret}]}` produces the block shown in `examples/resources/openstatus_http_monitor/resource.tf`.

---

### 3.2 Notifications: refactor + correctness fixes

A single block of work covering four related changes. All edits in `internal/terraform/hcl.go` (`writeNotificationProvider` and `GenerateNotificationsFile`) and `internal/terraform/enums.go` (`notificationProviderToString`).

**A. Single source of truth driven by the data oneof.** Replace the two-source approach (`notificationProviderToString(n.GetProvider())` for the `provider_type` attribute plus a parallel `switch d := data.Data` for the block) with a single switch on `data.Data` that yields both the provider string and the emitted block. Prevents server-side mismatches between `provider` and `data` from producing broken HCL.

```go
func writeNotificationProvider(b *hclwrite.Body, n *notificationv1.Notification) (providerType string, ok bool) {
    data := n.GetData()
    if data == nil {
        return "", false
    }
    switch d := data.Data.(type) {
    case *notificationv1.NotificationData_Discord:
        pb := b.AppendNewBlock("discord", nil).Body()
        pb.SetAttributeValue("webhook_url", cty.StringVal(d.Discord.GetWebhookUrl()))
        return "discord", true
    // …one case per provider type, including ms_teams (new)…
    case *notificationv1.NotificationData_MsTeams:
        pb := b.AppendNewBlock("ms_teams", nil).Body()
        pb.SetAttributeValue("webhook_url", cty.StringVal(d.MsTeams.GetWebhookUrl()))
        return "ms_teams", true
    }
    return "", false
}
```

Caller (`GenerateNotificationsFile`) inverts to "block first, then attributes": peek the data oneof to decide whether to emit at all, write the resource header, then call `writeNotificationProvider` which returns the inferred `provider_type` and emits the inner block in one pass.

**B. Add `ms_teams`** — covered by (A). Pinned SDK confirmed to include `NotificationProvider_NOTIFICATION_PROVIDER_MS_TEAMS = 13`, `MsTeamsData{WebhookUrl}`, and the `NotificationData_MsTeams` oneof case.

**C. Fix webhook headers — `ListNestedAttribute`, not block.** The provider's webhook schema declares `headers` as `schema.ListNestedAttribute`. The current generator emits `headers { key=… value=… }` block syntax which the provider rejects. Replace with list-attribute syntax:

```go
case *notificationv1.NotificationData_Webhook:
    pb := b.AppendNewBlock("webhook", nil).Body()
    pb.SetAttributeValue("endpoint", cty.StringVal(d.Webhook.GetEndpoint()))
    if hs := d.Webhook.GetHeaders(); len(hs) > 0 {
        vals := make([]cty.Value, 0, len(hs))
        for _, h := range hs {
            if h.GetKey() == "" { continue }
            vals = append(vals, cty.ObjectVal(map[string]cty.Value{
                "key":   cty.StringVal(h.GetKey()),
                "value": cty.StringVal(h.GetValue()),
            }))
        }
        if len(vals) > 0 {
            pb.SetAttributeValue("headers", cty.ListVal(vals))
        }
    }
    return "webhook", true
```

**D. Emit `monitor_ids` as traversals when the id is in `monitorRefs`; fall back to plain string when it isn't.** Matches the pattern `setTraversalOrString` already uses for singular cross-refs. The string fallback intentionally preserves the id in HCL even when the monitor isn't in the workspace (race between `ListMonitors` and `ListNotifications`, or monitor deleted out-of-band) — terraform will surface the inconsistency at plan time rather than the generator silently dropping it.

Build the set manually using hclwrite tokens so traversals and string literals can coexist in one set. Skeleton:

```go
if ids := n.GetMonitorIds(); len(ids) > 0 {
    tokens := hclwrite.Tokens{ /* '[' */ }
    for i, id := range ids {
        if i > 0 { tokens = append(tokens, commaToken) }
        if ref, found := g.monitorRefs[id]; found {
            tokens = append(tokens, identTraversal(ref.ResourceType, ref.Name, "id")...)
        } else {
            tokens = append(tokens, stringLiteral(id)...)
        }
    }
    tokens = append(tokens, /* ']' */)
    b.SetAttributeRaw("monitor_ids", tokens)
}
```

**E. Skip + warn on unknown / UNSPECIFIED providers.** `writeNotificationProvider` returning `ok=false` triggers the caller to:
- Print `warning: skipping notification %q — unknown provider type (CLI may be outdated)` to stderr.
- Continue past this notification without writing a resource block.
- Exclude the notification's id from `GenerateImportsFile`.

Tests:
- `TestGenerateNotificationsFile_MsTeams` — provider type, `ms_teams { webhook_url }` block.
- `TestGenerateNotificationsFile_WebhookHeaders` — assert `headers = [{key = "X", value = "Y"}]` attribute syntax, not block.
- `TestGenerateNotificationsFile_MonitorIdsTraversal` — workspace with one matching monitor and one unknown id → list has `openstatus_http_monitor.foo.id` and `"unknown-id"` mixed.
- `TestGenerateNotificationsFile_UnknownProviderSkipped` — UNSPECIFIED provider → no resource block in output, no import in imports.tf.

---

### 3.3 Status pages: correctness pack

Two correctness bugs and four drift gaps in one resource. All edits land in `internal/terraform/hcl.go:308-345` (`GenerateStatusPagesFile`) and `internal/terraform/enums.go:167-178` (`pageAccessTypeToString`).

**A. Fix `access_type = "ip"` being silently dropped.** Today the `pageAccessTypeToString` switch has no `IP_RESTRICTED` case and falls through to `"public"`, which (a) loses the user's choice and (b) drops the required `allowed_ip_ranges`. Add:

```go
case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED:
    return "ip"
```

**B. Emit `auth_email_domains` and `allowed_ip_ranges`.** These are required by the provider's `ValidateConfig` when `access_type` is `email-domain` / `ip`. Without them, the generated HCL **fails plan**. Replace the `access_type != "public"` block in `GenerateStatusPagesFile` (around hcl.go:336-343) with:

```go
switch accessType {
case "password":
    b.SetAttributeValue("access_type", cty.StringVal("password"))
    appendTODOComment(b)
    b.SetAttributeValue("password", cty.StringVal("REPLACE_ME"))
case "email-domain":
    b.SetAttributeValue("access_type", cty.StringVal("email-domain"))
    domains := page.GetAuthEmailDomains()
    vals := make([]cty.Value, len(domains))
    for i, d := range domains {
        vals[i] = cty.StringVal(d)
    }
    b.SetAttributeValue("auth_email_domains", cty.ListVal(vals)) // safe: provider requires ≥1
case "ip":
    b.SetAttributeValue("access_type", cty.StringVal("ip"))
    b.SetAttributeValue("allowed_ip_ranges", cty.StringVal(page.GetAllowedIpRanges()))
}
```

(`access_type = "public"` continues to be omitted as a default.)

**C. Emit `theme`, `default_locale`, `locales`, `allow_index`** with skip-default rules to avoid drift on the next plan:

```go
if theme := pageThemeToString(page.GetTheme()); theme != "" && theme != "system" {
    b.SetAttributeValue("theme", cty.StringVal(theme))
}
if dl := localeToString(page.GetDefaultLocale()); dl != "" && dl != "en" {
    b.SetAttributeValue("default_locale", cty.StringVal(dl))
}
if locs := page.GetLocales(); len(locs) > 0 {
    vals := make([]cty.Value, 0, len(locs))
    for _, l := range locs {
        if s := localeToString(l); s != "" {
            vals = append(vals, cty.StringVal(s))
        }
    }
    if len(vals) > 0 {
        b.SetAttributeValue("locales", cty.ListVal(vals))
    }
}
if page.GetAllowIndex() {
    b.SetAttributeValue("allow_index", cty.BoolVal(true))
}
```

Helpers to add in `enums.go`:

```go
func pageThemeToString(t status_pagev1.PageTheme) string {
    switch t {
    case status_pagev1.PageTheme_PAGE_THEME_SYSTEM:
        return "system"
    case status_pagev1.PageTheme_PAGE_THEME_LIGHT:
        return "light"
    case status_pagev1.PageTheme_PAGE_THEME_DARK:
        return "dark"
    }
    return ""
}

func localeToString(l status_pagev1.Locale) string {
    switch l {
    case status_pagev1.Locale_LOCALE_EN:
        return "en"
    case status_pagev1.Locale_LOCALE_FR:
        return "fr"
    case status_pagev1.Locale_LOCALE_DE:
        return "de"
    }
    return ""
}
```

SDK confirmed: `PageTheme = {UNSPECIFIED, SYSTEM, LIGHT, DARK}`, `Locale = {UNSPECIFIED, EN, FR, DE}`, `PageAccessType.IP_RESTRICTED = 4`, and `StatusPage.GetTheme/DefaultLocale/Locales/AllowIndex/AuthEmailDomains/AllowedIpRanges` all present.

Tests: add cases to `generate_test.go`:
- `TestGenerateStatusPagesFile_IPAccess` — IP_RESTRICTED → `access_type = "ip"` + `allowed_ip_ranges`.
- `TestGenerateStatusPagesFile_EmailDomainAccess` — AUTHENTICATED → `access_type = "email-domain"` + `auth_email_domains = [...]`.
- `TestGenerateStatusPagesFile_ThemeLocaleAllowIndex` — dark/fr-locale page emits the three attrs; default page emits none.

---

### 3.4 Component groups: emit `default_open`

`internal/terraform/hcl.go:348-354`. Skip-default rule (provider default is `false`):

```go
if grp.GetDefaultOpen() {
    gb.SetAttributeValue("default_open", cty.BoolVal(true))
}
```

SDK confirmed: `PageComponentGroup.GetDefaultOpen() bool`.

Test: extend `TestGenerateStatusPagesFile` to include a group with `default_open=true` and assert the line is emitted.

---

### 3.5 CLI ergonomics: `--force` and init-upgrade hint

`internal/terraform/generate.go`.

**A. `--force` flag, refuse-by-default overwrites.** Today `writeFile` truncates blindly. Add a stat-and-bail step before any write: if any of `provider.tf`, `monitors.tf`, `notifications.tf`, `status_pages.tf`, `imports.tf` already exists in `--output-dir` and `--force` is not set, exit with:

```
error: refusing to overwrite existing file %s; pass --force to replace
```

Sketch:

```go
&cli.BoolFlag{
    Name:    "force",
    Usage:   "Overwrite existing files in --output-dir",
    Aliases: []string{"f"},
},
// ...
if !cmd.Bool("force") {
    for _, name := range []string{"provider.tf", "monitors.tf", "notifications.tf", "status_pages.tf", "imports.tf"} {
        if _, err := os.Stat(filepath.Join(outputDir, name)); err == nil {
            return cli.Exit(fmt.Sprintf("refusing to overwrite existing file %s; pass --force to replace", name), 1)
        }
    }
}
```

The check happens after the API fetch fails-fast but before any disk writes, so partial output is impossible.

**B. Init-upgrade hint.** Always append to `printSummary` (`generate.go:130`):

```
Note: provider version pinned to ~> 0.2. Run 'terraform init -upgrade' if you previously ran this command.
```

Tests: extend `generate_test.go` (or add a small `cli_test.go`) covering:
- Refusal when a target file exists and `--force` is unset.
- Overwrite when `--force` is passed.
- Init-upgrade hint present in `printSummary` output.

---

## 4. Test additions checklist

Append to `internal/terraform/generate_test.go`:

- [ ] Update `TestGenerateProviderFile` for `~> 0.2`.
- [ ] HTTP monitor `open_telemetry` block.
- [ ] TCP monitor `open_telemetry` block (also adds first TCP test).
- [ ] DNS monitor `open_telemetry` block (existing DNS test extends).
- [ ] Notification `ms_teams` block.
- [ ] Status page IP access (`access_type = "ip"` + `allowed_ip_ranges`).
- [ ] Status page email-domain access (`access_type = "email-domain"` + `auth_email_domains`).
- [ ] Status page theme/default_locale/locales/allow_index emission.
- [ ] Component group `default_open = true`.
- [ ] Notification refactor: provider/data oneof drives both attr and block (no mismatch path).
- [ ] Webhook headers emitted as `headers = [{...}]` attribute (not block).
- [ ] `monitor_ids` emit as traversals for known IDs, plain strings for unknown.
- [ ] UNSPECIFIED / unknown notification provider → resource skipped, no import block, stderr warning.
- [ ] `--force` refuses to overwrite by default; allows overwrite when set.
- [ ] `printSummary` includes the `terraform init -upgrade` hint.

---

## 5. Reference: provider import IDs

Confirmed against provider `internal/.../ImportState` parsers:

| Resource | Import ID format | Generator today |
|---|---|---|
| `openstatus_http_monitor` / `_tcp_monitor` / `_dns_monitor` | `<id>` | ✓ |
| `openstatus_notification` | `<id>` | ✓ |
| `openstatus_status_page` | `<id>` | ✓ |
| `openstatus_status_page_component` | `<page_id>/<component_id>` | ✓ |
| `openstatus_status_page_component_group` | `<page_id>/<group_id>` | ✓ |

No changes needed in `GenerateImportsFile` (`hcl.go:386-414`).

---

## 6. File-level summary of edits

| File | Edit |
|---|---|
| `internal/terraform/hcl.go` | `GenerateProviderFile` (version bump); HTTP/TCP/DNS monitor branches (add `writeOpenTelemetry`); `writeNotificationProvider` (add `MsTeams` case); `GenerateStatusPagesFile` (rewrite access-type branch; add theme/locale/allow_index emission); component group branch (`default_open`); new `writeOpenTelemetry` helper |
| `internal/terraform/enums.go` | `notificationProviderToString` (add `MS_TEAMS`); `pageAccessTypeToString` (add `IP_RESTRICTED`); add `pageThemeToString`, `localeToString`. `notificationProviderToString` may move/become driven by `NotificationData` oneof per Q5b. |
| `internal/terraform/generate.go` | `--force` flag + pre-write existence check; init-upgrade hint in `printSummary` |
| `internal/terraform/generate_test.go` | New cases per §4 |
| `internal/cmd/app.go` | Bump `Version` to `"v1.1.0"` |
| `docs/openstatus-docs.md`, `docs/openstatus.1` | Regenerated (see §7) |

No new files **except** the opt-in smoke test (§9, phase 7). No SDK pin bumps required beyond the dep refresh that has already shipped (`buf.build/gen/go/openstatus/api/...@v1.36.11-20260512200453-7d7b7047611f.1`).

---

## 7. Docs regeneration (last commit of the PR)

After all code is in and tests pass, regenerate the auto-generated docs from the urfave/cli command tree (the new `--force` flag changes the rendered help text):

```sh
go run cmd/docs/docs.go
cd docs && pandoc -s -t man openstatus-docs.md -o openstatus.1
```

Commit both `docs/openstatus-docs.md` and `docs/openstatus.1` as the final commit. README is left alone — the team hasn't documented individual `terraform generate` flags in it so far.

---

## 8. Unrelated note from the dep refresh

`github.com/urfave/cli/v3` moved from `v3.0.0-alpha9.2` → `v3.9.0`, changing `BeforeFunc` to `func(context.Context, *Command) (context.Context, error)`. Already patched at `internal/cmd/app.go:63` — build and tests green. Mentioned here only so reviewers don't wonder why that file changed in the same branch.

---

## 9. Implementation todo list

Each phase is one commit. Phases are ordered so that earlier work doesn't conflict with later work, and so the build stays green commit-by-commit (a reviewer can `git bisect` cleanly). Run `go build ./... && go test ./...` at the end of every phase before committing.

### Phase 0 — Pre-flight ✅

- [x] Cut a feature branch off `main` (e.g. `feat/tf-generate-sync-v0.2`). _(jj: working on an anonymous change off main; chore: refresh deps committed)_
- [x] Confirm working tree is clean (`git status`); dep refresh already landed in a previous commit.
- [x] `go build ./...` and `go test ./...` green from baseline.

### Phase 1 — Bump generated provider version (commit 1: `chore(terraform): pin generated provider to ~> 0.2`) ✅

- [x] `internal/terraform/hcl.go` — `GenerateProviderFile`: change `version = "~> 0.1.0"` → `version = "~> 0.2"`.
- [x] `internal/terraform/generate_test.go` — update `TestGenerateProviderFile` assertion to `~> 0.2`.
- [x] `go test ./internal/terraform/...` green.

### Phase 2 — Status page correctness pack (commit 2: `fix(terraform): emit access_type=ip + auth_email_domains/allowed_ip_ranges; add theme/locale/allow_index`) ✅

Order matters within this phase: add helpers first, then call them.

- [x] `internal/terraform/enums.go` — extend `pageAccessTypeToString` with `case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED: return "ip"`.
- [x] `internal/terraform/enums.go` — add `pageThemeToString(t status_pagev1.PageTheme) string` (returns `""` for UNSPECIFIED, `"system"|"light"|"dark"` otherwise).
- [x] `internal/terraform/enums.go` — add `localeToString(l status_pagev1.Locale) string` (returns `""` for UNSPECIFIED, `"en"|"fr"|"de"` otherwise).
- [x] `internal/terraform/hcl.go` — `GenerateStatusPagesFile`: replace the current `accessType != "public"` block with the four-case switch (`public` omits / `password` keeps current TODO+REPLACE_ME / `email-domain` emits `auth_email_domains` sorted via `sort.Strings` or TODO+REPLACE_ME / `ip` emits `allowed_ip_ranges` or TODO+REPLACE_ME).
- [x] `internal/terraform/hcl.go` — `GenerateStatusPagesFile`: after the access-type block, emit `theme` (when not `system`), `default_locale` (when not `en`), `locales` (sorted, when non-empty), `allow_index` (when `true`), using skip-default rules from Q1.
- [x] `internal/terraform/generate_test.go` — `TestGenerateStatusPagesFile_IPAccess` with non-empty `AllowedIpRanges`.
- [x] `internal/terraform/generate_test.go` — `TestGenerateStatusPagesFile_IPAccessEmptyFallback`: IP_RESTRICTED + empty `allowed_ip_ranges` → asserts `# TODO:` comment and `REPLACE_ME` value.
- [x] `internal/terraform/generate_test.go` — `TestGenerateStatusPagesFile_EmailDomainAccess` with non-empty domains.
- [x] `internal/terraform/generate_test.go` — `TestGenerateStatusPagesFile_EmailDomainEmptyFallback`.
- [x] `internal/terraform/generate_test.go` — `TestGenerateStatusPagesFile_ThemeLocaleAllowIndex`: dark + fr default_locale + locales=[en,fr] + allow_index=true → all four emitted; default page → none emitted.
- [x] `go test ./internal/terraform/...` green.

### Phase 3 — Component group `default_open` (commit 3: `feat(terraform): emit default_open on status page component groups`) ✅

- [x] `internal/terraform/hcl.go` — component-group branch in `GenerateStatusPagesFile`: `if grp.GetDefaultOpen() { gb.SetAttributeValue("default_open", cty.BoolVal(true)) }`.
- [x] `internal/terraform/generate_test.go` — extend `TestGenerateStatusPagesFile` (or add a focused test) to include a group with `DefaultOpen: true` and assert the line.
- [x] `go test ./internal/terraform/...` green.

### Phase 4 — Monitor `open_telemetry` (commit 4: `feat(terraform): emit open_telemetry block on HTTP/TCP/DNS monitors`) ✅

- [x] `internal/terraform/hcl.go` — modify the existing `writeRegions` helper to sort regions alphabetically (per §2b — set semantics, deterministic output).
- [x] `internal/terraform/hcl.go` — new helper `writeOpenTelemetry(b *hclwrite.Body, ot *monitorv1.OpenTelemetryConfig)` per Q2: skip iff `ot == nil` OR (`endpoint == "" && len(headers) == 0`); inside, emit `endpoint` only when non-empty; emit one `headers { key/value }` block per header (no sort — preserves API order per §2b).
- [x] Call `writeOpenTelemetry(b, m.GetOpenTelemetry())` in each of: HTTP monitor branch (`hcl.go:109` block, after the assertion writers), TCP monitor branch (`hcl.go:151`, after `writeRegions`), DNS monitor branch (`hcl.go:180`, after `record_assertions`).
- [x] `internal/terraform/generate_test.go` — `TestGenerateMonitorsFile_HTTP_OpenTelemetry`: endpoint + one header → block present.
- [x] `internal/terraform/generate_test.go` — `TestGenerateMonitorsFile_TCP_OpenTelemetry` (also adds first TCP-only test).
- [x] `internal/terraform/generate_test.go` — `TestGenerateMonitorsFile_DNS_OpenTelemetry`.
- [x] `internal/terraform/generate_test.go` — `TestGenerateMonitorsFile_OpenTelemetry_SkippedWhenEmpty`: `OpenTelemetryConfig{Endpoint:"", Headers:nil}` → no block.
- [x] `go test ./internal/terraform/...` green.

### Phase 5 — Notification refactor + fixes (commit 5: `fix(terraform): notification provider type from data oneof; ms_teams; webhook headers attribute; monitor_ids traversals`) ✅

This is the largest phase. Land in one commit (per Q7) but write it incrementally.

- [x] `internal/terraform/hcl.go` — add the `*notificationv1.NotificationData_MsTeams` case emitting `ms_teams { webhook_url = … }`.
- [x] `internal/terraform/hcl.go` — rewrite the `*notificationv1.NotificationData_Webhook` case so `headers` is emitted as a `cty.ListVal([]cty.Value{cty.ObjectVal({key,value})})` set via `SetAttributeValue("headers", …)`, not as nested `headers { … }` blocks.
- [x] `internal/terraform/hcl.go` — replace the current `monitor_ids` plain-string emission with a token-list builder (`writeMonitorIds`). Sort via `sort.Strings` first; traversal tokens for known refs, string-literal tokens otherwise.
- [x] `internal/terraform/hcl.go` — add `traversalTokensInline(parts ...string)` (no trailing newline) and refactor `traversalTokens` to call it + append newline. Add `stringLitTokens(s)` helper.
- [x] `internal/terraform/hcl.go` — add `renderableNotification(n) (providerType string, ok bool)` switch returning the tf-string per oneof case.
- [x] `internal/terraform/hcl.go` — `Generator` struct: add `skippedNotifications map[string]bool` field; initialize in `NewGenerator`.
- [x] `internal/terraform/hcl.go` — `NewGenerator` notifications loop: skip + warn on `!ok` via `renderableNotification`.
- [x] `internal/terraform/hcl.go` — `GenerateNotificationsFile`: skip when `g.skippedNotifications[n.GetId()]`; provider_type comes from `renderableNotification`.
- [x] `internal/terraform/hcl.go` — `GenerateImportsFile`: same skip guard for the notification import block.
- [x] `internal/terraform/enums.go` — remove `notificationProviderToString` (now unused).
- [x] `internal/terraform/generate_test.go` — `TestGenerateNotificationsFile_MsTeams`.
- [x] `internal/terraform/generate_test.go` — `TestGenerateNotificationsFile_WebhookHeaders`.
- [x] `internal/terraform/generate_test.go` — `TestGenerateNotificationsFile_MonitorIdsTraversal`.
- [x] `internal/terraform/generate_test.go` — `TestGenerateNotificationsFile_UnknownProviderSkipped`.
- [x] `go test ./internal/terraform/...` green.

### Phase 6 — CLI ergonomics (commit 6: `feat(terraform): --force flag and terraform init -upgrade hint`) ✅

- [x] `internal/terraform/generate.go` — add `&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Usage: "Overwrite existing files in --output-dir"}` to `GetTerraformGenerateCmd().Flags`.
- [x] `internal/terraform/generate.go` — extract `checkExistingFiles(outputDir, force)` helper; invoke before `MkdirAll`. Stats each of `provider.tf`, `monitors.tf`, `notifications.tf`, `status_pages.tf`, `imports.tf` and returns an error mentioning the filename if any exists.
- [x] `internal/terraform/generate.go` — extend `printSummary` with the `terraform init -upgrade` hint.
- [x] `internal/terraform/cli_test.go` — `TestCheckExistingFiles_RefusesExisting` / `_OverwritesWithForce` / `_EmptyDir` / `_NonexistentDir`.
- [x] `internal/terraform/cli_test.go` — `TestPrintSummary_IncludesInitUpgradeHint` (captures stdout).
- [x] `go test ./internal/terraform/...` green.

### Phase 7 — Opt-in smoke test (commit 7: `test(terraform): add terraform-validate smoke test behind build tag`) ✅

- [x] New file `internal/terraform/smoke_test.go` with `//go:build smoke` build tag at top.
- [x] In the file: build a representative `WorkspaceData` covering HTTP/TCP/DNS monitors (HTTP includes OTEL + assertions); slack + ms_teams + webhook-with-headers notifications; status page with theme/locales/allow_index + default_open group + monitor component; second status page with IP access.
- [x] Write all generated files to `t.TempDir()`. Skip the test (`t.Skipf`) if `terraform` is not on `PATH`.
- [x] Exec `terraform init -upgrade` then `terraform validate` against the temp dir; fail the test on non-zero exit.
- [x] Document at the top of the file: `// Run with: go test -tags=smoke ./internal/terraform/`.
- [x] `go test ./internal/terraform/...` (no tag) still green and does NOT invoke terraform.
- [x] Manually verified: `go test -tags=smoke ./internal/terraform/` passes locally.

### Phase 8 — Version + docs (commit 8: `chore: bump cli to v1.1.0 and regenerate docs`) ✅

- [x] `internal/cmd/app.go` — change `Version: "v1.0.5"` → `Version: "v1.1.0"`. Also updated `internal/cmd/app_test.go` to match.
- [x] From repo root: `go run cmd/docs/docs.go` (updates `docs/openstatus-docs.md`).
- [x] `cd docs && pandoc -s -t man openstatus-docs.md -o openstatus.1` (updates the manpage).
- [x] Verified that `--force` flag entry appears in both `openstatus-docs.md` and `openstatus.1`.
- [x] `go build ./... && go vet ./... && go test ./...` all green.

### Phase 9 — Submit ✅

- [x] Push the branch. _(Pushed `feat/tf-generate-sync-v0.2` to origin via `jj git push --bookmark feat/tf-generate-sync-v0.2 --allow-new`.)_
- [ ] Open PR. Title suggestion: `terraform generate: sync with provider v0.2 (open_telemetry, ms_teams, ip access, theme/locales, --force)`. _(Author to open — GitHub provided: https://github.com/openstatusHQ/cli/pull/new/feat/tf-generate-sync-v0.2)_
- [ ] PR description: bullet list of user-visible changes; include "Closes #…" if upstream has tracking issues; call out the two correctness fixes (webhook headers, IP access type) since they affect real users.
- [ ] Watch CI; address review feedback.
- [ ] Merge strategy is the author's call (repo allows squash, merge, rebase).

### Definition of done

- All boxes above checked.
- `go build ./...`, `go test ./...`, and `go test -tags=smoke ./internal/terraform/` all pass locally.
- The PR's diff on `internal/terraform/` matches the file-level summary in §6, plus the new `smoke_test.go`.
- `docs/openstatus-docs.md` and `docs/openstatus.1` show the `--force` flag.
- `internal/cmd/app.go` shows `v1.1.0`.
- No new TODOs, no commented-out code, no orphaned helpers (e.g. old `notificationProviderToString` removed if Phase 5 removed it).
