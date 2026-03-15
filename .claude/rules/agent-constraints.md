# Agent Constraints

Rules that apply to all agent workflows.

## Anti-Patterns

- NO loading full context into workers
- NO sharing state between workers (use Beads)
- NO workers spawning workers (single-level only)
- NO long-running workers (timeout at 5 min)
- NO opus for simple tasks (cost optimization)
- NO skipping git push (work is NOT complete until pushed)
