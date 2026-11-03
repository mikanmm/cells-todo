FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
# Generate go.sum if it is missing; this step requires network access during the build.
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server .

FROM alpine:3.20
RUN apk add --no-cache tzdata ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/server /app/server
USER app
EXPOSE 8081
ENTRYPOINT ["/app/server"]
CMD ["serve"]
