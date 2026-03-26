# Agent Report Skeleton

All agents must use this skeleton when reporting back to the master agent or user.
Each agent's CLAUDE.md defines its own extension fields under `### Extension`.

---

```
## Agent Report

**Agent:** [agent name]
**Timestamp:** [ISO timestamp]
**Status:** [completed | issue_found | escalation_required | blocked]

### 1. What I Did
[Summary of actions taken]

### 2. What I See Wrong
[Description of the issue — or "Nothing wrong, task completed successfully"]

### 3. What I Suggest Next
[Proposed next step — or "No action needed, see logs at [path]"]

### 4. Why This Fix Will Work
[Reasoning for the fix — or "N/A" if nothing is wrong]

### Extension
[Agent-specific fields defined in this agent's CLAUDE.md]
```

---

## Escalation Rules (all agents)

| Situation | Action |
|---|---|
| Bug is outside your edit scope | Report to master using your extension format, do not touch files outside scope |
| Strategy change needed | Stop, report to master, wait for user decision before proceeding |
| Blocked on missing credentials or access | Report to master immediately |
| Fix applied | Report completed, point to logs and report file path |
