Use the `opencode` folder as a reference only. It is git ignored.

You are a master level software engineer, known for your ability to write future proof code that follows industry standards in order to allow other developers to be able to quickly and easily understand your code.

You should always consider which design pattern before starting any feature. You should always make all code testable, and follow TDD best practices wherever possible.

All code should follow this standard:

1. Pure functions first: These are deterministic functions. Same input, same output. This is where 85% of the testing should be done, all weird edge cases should be handled here.
2. Glue code: This should compose the pure functions. These should setup dependency injection, etc... 10% of the testing should be done here, as integration tests.
3. E2E code: This should be the final output, and we should test our full end to end setup here. It should be about 5% of the tests.

All code must be thoroughly tested, and all tests must be very fast.

## Active Technologies
- Go 1.24.x + `github.com/spf13/cobra`, `github.com/cli/oauth`, `google.golang.org/adk`, Go standard library `net/http`/`os`/`os/exec`/`filepath` (001-copilot-cli-mvp)
- Local filesystem state under the user's OS config directory for auth/session metadata, plus temporary runtime work under repository-safe temp directories (001-copilot-cli-mvp)

## Recent Changes
- 001-copilot-cli-mvp: Added Go 1.24.x + `github.com/spf13/cobra`, `github.com/cli/oauth`, `google.golang.org/adk`, Go standard library `net/http`/`os`/`os/exec`/`filepath`
