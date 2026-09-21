# CLAUDE.md — Project Conventions for new-api

## MANDATORY: Read AGENTS.md with the Read tool

Do not treat `@AGENTS.md` as loaded. Claude Code does not reliably inline that import.

Before any planning, coding, reviewing, or answering a project question, you MUST call the Read tool on the repo-root file `AGENTS.md` and wait for the full contents. This is the first action of every session and every new task.

Rules:

- Do not start from memory, summaries, or this file alone.
- Do not skip the Read because a previous turn mentioned AGENTS.md.
- Do not replace the Read with a grep, glob, or partial skim.
- After reading, follow every rule in `AGENTS.md` for the rest of the work.
- If the task touches `web/`, also Read `web/AGENTS.md` before editing frontend files.
- If the task touches billing as defined under **Billing rules (mandatory read gate)** in `AGENTS.md`, also Read `.agents/rules/billing.md` in full before planning or editing. Tasks outside that definition may skip it.

## MANDATORY: 二次开发变更记录

本仓库是上游 new-api 的 fork，`main` 只用于同步官方版本，二次开发的改动最终合并进 `production`。

- 任何改动上游代码的任务，开始前先用 Read 工具读仓库根目录的 `FORK_CHANGES.md`，了解已有的二次开发改动与约束。
- 改完代码后，**必须**在 `FORK_CHANGES.md` 的「变更记录」中按其记录模板追加一条记录，与代码放在同一个提交里。只提交代码不记录视为未完成。
- 改动必须能长期跟随上游更新：功能逻辑放进新增文件，对上游既有文件只做追加式的最小改动，不重排、不重构、不顺手修改无关代码行（包括既存的 lint 告警）。
