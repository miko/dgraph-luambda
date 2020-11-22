FROM golang:alpine AS builder
RUN apk add --update git gcc libc-dev
WORKDIR /src
COPY ./go* ./
RUN go mod download
COPY . .
ARG TAG
RUN go build -ldflags "-X main.TAG=$TAG" -o /tmp/app

FROM alpine:latest
WORKDIR /app
RUN apk add --update curl
RUN mkdir /app/events
ENV STATIC=/app/public
COPY --from=builder /tmp/app /app/app
COPY env.env /app/.env
COPY *.lua /app/
COPY schema.graphql /app/
COPY init.graphql /app/
ENTRYPOINT /app/app
EXPOSE 8686
