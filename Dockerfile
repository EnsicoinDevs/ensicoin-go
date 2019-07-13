FROM golang:alpine AS build-env

RUN apk update
RUN apk add git

RUN mkdir /src
WORKDIR /src
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o eccd

FROM alpine

WORKDIR /app
COPY --from=build-env /src/eccd .

RUN mkdir /data

EXPOSE 4224
EXPOSE 4225

ENTRYPOINT [ "./eccd" ]
CMD [ "--datadir" "/data/" ]
