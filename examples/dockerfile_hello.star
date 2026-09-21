# Dockerfile-as-code example
# Equivalent to:
#
#   FROM alpine:3.20
#   RUN apk add --no-cache curl
#   WORKDIR /app
#   COPY . /app
#   ENV APP_ENV=production
#   EXPOSE 8080
#   USER nobody
#   ENTRYPOINT ["/app/server"]
#   CMD ["--help"]

def build():
    s = from_("alpine:3.20")
    s = s.run("apk add --no-cache curl")
    s = s.workdir("/app")
    s = s.copy(src=".", dest="/app")
    s = s.env("APP_ENV", "production")
    s = s.expose("8080")
    s = s.user("nobody")
    s = s.entrypoint(["/app/server"])
    s = s.cmd(["--help"])
    return s
