# Multi-stage Dockerfile-as-code
# Equivalent to a classic Go multi-stage build.

def build():
    # --- builder stage ---
    builder = from_("golang:1.22-alpine", as_="builder")
    builder = builder.workdir("/src")
    builder = builder.env("CGO_ENABLED", "0")
    builder = builder.copy(src=".", dest="/src")
    builder = builder.run("go build -o /out/app ./cmd/app")

    # --- final stage ---
    final = from_("alpine:3.20")
    final = final.run("apk add --no-cache ca-certificates")
    # COPY --from=builder
    final = final.copy(src="/out/app", dest="/usr/local/bin/app", from_=builder)
    final = final.user("nobody")
    final = final.entrypoint(["/usr/local/bin/app"])
    return final
