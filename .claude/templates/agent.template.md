---
name: worker-[type]
description: [Brief description of worker purpose. Used for parallel [task type].]
permissionMode: acceptEdits
model: haiku
# skills: skill-name-1, skill-name-2
---

<!--
Agents are lightweight swarm workers spawned via Task tool.

Required fields:
  - name: lowercase, hyphens only, max 64 chars
  - description: max 1024 chars, include "when to use" triggers

Optional fields:
  - permissionMode: default|acceptEdits|bypassPermissions|plan
      Use "acceptEdits" for workers that read and write files (most workers).
      Use "bypassPermissions" only when the orchestrator has already validated safety.
  - model: haiku (fast), sonnet (balanced), opus (complex)
  - skills: Comma-separated skill names to auto-load

NOTE: Use permissionMode to control agent access, not "tools:". Restricting
individual tools via "tools:" is fragile; permissionMode provides the right
granularity for swarm workers.
-->

# [Worker Type] Worker

[One-line description of worker focus]

## Focus

- [Primary task 1]
- [Primary task 2]
- [Primary task 3]

## Output Format

```
[Field 1]: [description]
[Field 2]: [description]
[Field 3]: [description]
```

## Constraints

- [Constraint 1 - e.g., "Read-only operations"]
- [Constraint 2 - e.g., "Single task focus"]
- [Constraint 3 - e.g., "Timeout at 5 min"]

## On Completion

Report: [what to report back to orchestrator]
