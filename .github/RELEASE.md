
## Changelog

### New Features

1. **Distributed Load Testing in Perftest**: Added distributed testing mode with real-time performance metrics visualization on the dashboard, enabling large-scale load testing.
2. **Gin Session Usage Tips**: Added guidance for using sessions in the Gin framework, making integration easier for developers.

### Bug Fixes

1. **SQL Code Generation Fix**: Fixed an issue where SQL-to-code generation failed when SQL comments contained line breaks.
2. **SSE Client Release Fix**: Fixed an issue where the SSE server didn’t properly release connected clients upon shutdown. [#136](https://github.com/go-dev-frame/sponge/issues/136)
