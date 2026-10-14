# Project Run Command Rule

## Trigger Phrases
When the user says any of these phrases:
- "run the project"
- "rebuild the project"
- "rerun the project"

## Action
1. First, check if there's a currently running process (typically from a previous `make run` command)
2. If a process is running, stop it (using Ctrl+C or appropriate termination)
3. Execute `make run` to start the project fresh

## Implementation Notes
- This is a restart operation - it ensures a clean state by stopping any existing session
- The `make run` command should be executed from the project root directory
- Wait for the previous process to fully terminate before starting the new one
