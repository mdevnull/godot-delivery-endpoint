FROM golang:1.17-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git wget
RUN wget https://downloads.tuxfamily.org/godotengine/3.4.4/mono/Godot_v3.4.4-stable_mono_x11_64.zip && \
    unzip Godot_v3.4.4-stable_mono_x11_64.zip && rm Godot_v3.4.4-stable_mono_x11_64.zip && \
    cp Godot_v3.4.4-stable_mono_linux_server_64/Godot_v3.4.4-stable_mono_linux_server.64 /app/godot

COPY . .

COPY ./netrc /root/.netrc
RUN chmod 600 /root/.netrc

RUN go mod download && \
    go build -o server main.go

FROM alpine:3 AS runner

COPY --from=builder /app/server /server
COPY --from=builder /app/godot /godot

ENTRYPOINT ["/server"]