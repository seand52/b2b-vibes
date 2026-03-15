---
description: [Short description for autocomplete - what this command does]
argument-hint: [task-description]
# model: sonnet
# disable-model-invocation: false
---

<!--
Commands define expert modes invoked via slash commands (e.g., /swarm-plan).

Required fields:
  - description: Short text shown in autocomplete

Optional fields:
  - argument-hint: Help text for command arguments
  - model: sonnet|opus|haiku (defaults to current model)
  - disable-model-invocation: true to prevent SlashCommand tool use

NOTE: Commands do not use tool restrictions. Tool access is controlled via
permissionMode in the agent definition, not in command frontmatter.
-->

# You are now the [Command Name]

[Clear identity statement describing who this command is and their primary focus. 2-3 sentences.]

## Responsibilities

- **[Responsibility 1]**: [Brief description]
- **[Responsibility 2]**: [Brief description]
- **[Responsibility 3]**: [Brief description]

## Methods

### [Method Category 1]

[Description of approach or methodology]

### [Method Category 2]

[Description of approach or methodology]

## Quality Standards

- [Standard 1]
- [Standard 2]
- [Standard 3]

## Constraints

- **DO NOT** [constraint 1]
- **DO NOT** [constraint 2]
- **ALWAYS** [requirement 1]
- **ALWAYS** [requirement 2]

## Related Skills

- `[skill-name]` - [When to use]
- `[skill-name]` - [When to use]

## Handoff Protocol

- To **[Command]**: [What to create/prepare before handoff]
- To **[Command]**: [What to create/prepare before handoff]

$ARGUMENTS
