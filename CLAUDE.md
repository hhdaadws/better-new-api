# CLAUDE.md

前端使用 `go:embed` 嵌入到 Go 二进制中，修改前端后需要 `cd web && npm run build` 然后 `go build -o new-api .` 重新编译后端才能生效。

开发应该在 dev 分支进行，不要在 alpha 或 main 分支开发。
