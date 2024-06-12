# Build Stage
FROM golang:1.22-alpine3.18 AS builder
WORKDIR /app

COPY . .

RUN go build -o battleship cmd/main.go


# Run stage
FROM alpine:3.18 AS runtime
WORKDIR /app

COPY --from=builder /app/battleship /app
# COPY --from=builder /app/db /app
# COPY .env /app 

EXPOSE 1313
CMD [ "./battleship" ]