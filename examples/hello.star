def build():
    alpine = llb.image("docker.io/library/alpine:3.20")
    return alpine.run("echo 'Hello from starllb!'").root()
