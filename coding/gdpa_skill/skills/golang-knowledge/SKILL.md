---
name: golang-knowledge
status: WIP
user-invocable: false
---

## Add Third-Party Package Dependencies

To safely add a new third-party package dependency:

- Run `go list -m <package_path> || go get <package_path>` This will only add the package if it's not already present.
- When go get fails:
    - Verify the package path is correct.
    - If you are sure the package path is correct, but it cannot be resolved, you can refuse the task with an
      explanation.

## Handling Errors in Third-Party Libraries

* For `sonic`, `frugal`, `choleraehyq/pid`, try to update them to latest version.
    - Since you don't know the exact version, just use `latest`. If `latest` doesn't work, try `master`, especially for
      `overpass/x`.
* For `thrift`, `dynamicgo` related 3rd library, try to add
  `replace github.com/apache/thrift => github.com/apache/thrift v0.13.0` in go.mod.
* For `code.byted.org/gdp/` related 3rd library, try to update them to `latest` version (for `code.byted.org/gdp/*` use
  `latest` rather than `master`).
* For `tiktok/apimodels/commons`: unknown revision commons/v0.0.0
    - Run `go mod edit -droprequire code.byted.org/tiktok/dtoconv && go get code.byted.org/tiktok/dtoconv` if
      `tiktok/dtoconv` is included.
    - Run
      `go mod edit -droprequire code.byted.org/tiktok/apimodels/commons && go get code.byted.org/tiktok/apimodels/commons`