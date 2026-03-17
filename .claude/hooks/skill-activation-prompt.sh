#!/usr/bin/env bash
# Skill activation hook - suggests relevant skills based on prompt keywords

cd "$CLAUDE_PROJECT_DIR/.claude/hooks"
cat | npx tsx skill-activation-prompt.ts || true
