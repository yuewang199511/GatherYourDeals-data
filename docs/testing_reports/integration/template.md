# Integration Test Report — Template

**Agent:** Integration Agent
**Timestamp:** [ISO timestamp]
**Status:** [completed | issue_found | escalation_required | blocked]

---

### 1. What I Did
[Summary — which test suite was run, against which environment]

### 2. What I See Wrong
[Assertion errors observed — or "All tests passed"]

### 3. What I Suggest Next
[Proposed fix or next step — or "No action needed, see logs at load_testing/integration/"]

### 4. Why This Fix Will Work
[Reasoning — or "N/A"]

---

### Extension

#### Test Run Summary
**Test run ID:** [for Honeycomb correlation]
**Environment:** [local | staging | docker compose profile]
**Tests passed:** [N]
**Tests failed:** [N]

#### Failed Test Details (repeat per failure)
**Test file:** [file path]
**Test case:** [test function name]
**Endpoint:** [HTTP method + path]
**User context:** [user role / user ID]
**Expected:** [expected response]
**Actual:** [actual response]
**Assertion error:** [exact error message]
**Issue location:** [test code | service code]
