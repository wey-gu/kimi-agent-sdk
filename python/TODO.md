# Exception Normalization

## Exceptions not based on KimiAgentException (KimiCLIException)

### SDK-sourced
- ValueError: prompt() validation for yolo/approval_handler_fn in src/kimi_agent_sdk/_prompt.py:80, :82
- RuntimeError: Session.prompt() when closed or already running in src/kimi_agent_sdk/_session.py:224, :226

### Propagated from Kimi CLI / Kosong
- FileNotFoundError: agent file missing (KimiCLI.create contract)
- LLMNotSet: raised in kimi_cli.soul
- LLMNotSupported: raised in kimi_cli.soul
- MaxStepsReached: raised in kimi_cli.soul
- RunCancelled: raised in kimi_cli.soul
- ChatProviderError (and subclasses): raised by chat providers in kosong

## Path vs KaosPath in public APIs

- Distinguish KaosPath vs Path in public API signatures.
- work_dir should be KaosPath (not coerced from Path/str).
- skills_dir should be KaosPath (agent uses KaosPath for iteration/exec).
- config should be Path (Config/Path overload to be revisited).
- agent_file should remain Path.
- Remove implicit coercion for work_dir/skills_dir (no _coerce_work_dir).

## Documentation & Release

- Update README to reflect KaosPath/Path distinctions and finalized API surface.
- Update project description (pyproject) to match SDK positioning.
- Add module-level docstrings for public modules.
- Set up CI for lint/typecheck/test.
- Add unit tests for public APIs and error handling.
- Add release workflow (tag -> build -> publish).
- Add docs (API reference, usage guide).
