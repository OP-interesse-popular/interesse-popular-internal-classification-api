FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /internal-classification-api .

FROM alpine:3.22
RUN adduser -D -H appuser
USER appuser
COPY --from=build /internal-classification-api /internal-classification-api
EXPOSE 3001
ENTRYPOINT ["/internal-classification-api"]
