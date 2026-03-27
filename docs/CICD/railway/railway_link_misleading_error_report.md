# Forensic Report: `railway link` Returns "Unauthorized" on Invalid Argument

**Date:** 2026-03-26
**Railway CLI version:** 4.35.0
**Reported to:** Railway dev team
**Severity:** Medium — causes incorrect diagnosis and wasted investigation time

---

## Summary

`railway link "<project-id>"` (positional argument syntax) returns an "Unauthorized" error message after a CLI breaking change removed support for positional project IDs. The correct error should be an argument parsing error. The misleading message caused multiple rounds of unnecessary token debugging.

---

## Exact Error Observed

```
$ railway link "7639518c-e0a7-4cd4-8227-6c2575458620"

Unauthorized. Please check that your RAILWAY_TOKEN is valid and has access
to the resource you're trying to use.
```

Exit code: 1

---

## Why This Led to Wrong Diagnosis

The error message says `Unauthorized` and explicitly tells the user to check their `RAILWAY_TOKEN`. This language has exactly one interpretation: **authentication failure**. There is no indication that the argument format is wrong.

The investigation that followed:

| Step | Action | Time spent | Outcome |
|---|---|---|---|
| 1 | Assumed token expired — asked user to regenerate | ~5 min | Token was valid all along |
| 2 | Confirmed local `railway whoami` works | 2 min | Ruled out token invalidity |
| 3 | Tried `railway link -p <id>` (flag syntax) | 1 min | Worked — revealed the real issue |
| 4 | Realized it was a CLI syntax change, not auth | — | Root cause found |

Total time lost on incorrect diagnosis: **~10 minutes** plus user friction around token regeneration.

---

## Root Cause

`railway link` previously accepted a positional project ID:
```bash
railway link "7639518c-e0a7-4cd4-8227-6c2575458620"   # old syntax
```

At some point (version unknown, observed broken on v4.35.0) this was changed to require a flag:
```bash
railway link -p "7639518c-e0a7-4cd4-8227-6c2575458620"   # new syntax
```

The CLI did not print a deprecation warning in any prior version. When the positional argument is passed, the CLI appears to skip local argument parsing and attempt authentication with an incorrect or empty project context — which then returns `Unauthorized` from the Railway API.

**Note:** Earlier in the same session, a *different* error did appear for the same command with a different CLI version (also v4.35.0 on a different runner):

```
error: unexpected argument '7639518c-e0a7-4cd4-8227-6c2575458620' found
Usage: railway link [OPTIONS]
```

This is the correct, clear error. Its presence on one run and absence on another suggests the behavior may be non-deterministic, or the two runs were using slightly different builds.

---

## Comparison: Good vs Bad Error

**Run A (correct error):**
```
error: unexpected argument '7639518c-e0a7-4cd4-8227-6c2575458620' found

Usage: railway link [OPTIONS]

For more information, try '--help'.
```

**Run B (misleading error):**
```
Unauthorized. Please check that your RAILWAY_TOKEN is valid and has access
to the resource you're trying to use.
```

Both runs used `railway link "7639518c-..."` with the same `RAILWAY_TOKEN`. The token was valid in both cases.

---

## Impact

1. **Incorrect user action:** User was asked to regenerate a valid Railway project token unnecessarily.
2. **Wasted CI runs:** 3 CI runs failed while diagnosing a token issue that did not exist.
3. **Misleading documentation risk:** Any agent, human, or documentation that sees this error will correctly follow the advice ("check your token") and reach a dead end.

---

## Suggested Fix

When `railway link` receives an unrecognized positional argument, it should:

1. Print the argument parsing error **before** attempting any API call
2. **Never** return an authentication error for an argument parsing failure
3. Optionally: print a migration hint, e.g.:
   ```
   error: positional project ID is no longer supported.
   Use: railway link -p "7639518c-..."
   ```

---

## Reproduction

```bash
# Requires: valid RAILWAY_TOKEN set in environment
# CLI version: 4.35.0

export RAILWAY_TOKEN=<valid-project-token>
railway link "7639518c-e0a7-4cd4-8227-6c2575458620"

# Expected: argument parsing error
# Actual:   Unauthorized. Please check that your RAILWAY_TOKEN is valid...
```

---

## Workaround (applied)

Replace `railway link` entirely. Project tokens already carry project/environment context — `railway link` is not needed:

```bash
# Before (broken)
railway link "7639518c-e0a7-4cd4-8227-6c2575458620"
railway environment "load-test"
railway service "GatherYourDeals-data"
railway up --detach

# After (working)
railway up --service "GatherYourDeals-data"
```
