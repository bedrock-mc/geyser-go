# Geyser-Go agent instructions

This repository is the Go Geyser port and is separate from `rust-mcbe`/Cinnabar.
The target is practical, version-matched Geyser parity using the pinned Lunar
Gophertunnel fork, complete Cloudburst/Lunar vanilla registries, and verified
Java-server/Bedrock-client interoperability.

## Mandatory model routing

- Use `gpt-5.6-luna` for the coordinator, writers, reviewers, and research
  workers. Do not use `gpt-5.6-sol`, Terra, or another model for this project.
- Prefer `model_reasoning_effort="max"` when the runtime accepts it. Otherwise
  use `model_reasoning_effort="xhigh"` and `service_tier="fast"` when that
  setting is exposed. Never silently claim a setting the CLI did not confirm.
- A typical CLI invocation is:

  ```powershell
  codex -a never exec -m gpt-5.6-luna -c 'model_reasoning_effort="max"' -C <worktree> -
  ```

## Parallel work

- Parallelize only disjoint tranches. Every writing worker gets its own linked
  worktree and branch; the coordinator owns integration, plan updates, pushes,
  and live acceptance.
- Workers commit locally but do not push or modify the authoritative checkout.
- Review the complete worker commit range before cherry-picking it. Preserve
  clean worktrees and record focused tests plus remaining native/live gates.

## Fidelity and evidence

- Treat Geyser, CloudburstMC/Data, Lunar/Dragonfly, and the negotiated protocol
  as source material; do not infer support from Dragonfly gameplay coverage
  alone when complete generated vanilla data exists.
- Keep wire-malformed input fatal and semantically odd but well-formed remote
  data lenient, bounded, logged, and session-preserving.
- Do not call the port complete based on unit tests alone. Native Bedrock,
  Paper/BDS interoperability, performance, and feature-family gates remain
  separate acceptance requirements.
