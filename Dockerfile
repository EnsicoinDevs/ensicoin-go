FROM golang:latest AS build-env

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -mod=vendor -o eccd

FROM alpine

WORKDIR /app
COPY --from=build-env /src/eccd .

RUN mkdir /data

EXPOSE 4224
EXPOSE 4225

ENTRYPOINT [ "./eccd" ]
CMD [ "--datadir" "/data/" ]
