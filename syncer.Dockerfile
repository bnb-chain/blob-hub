FROM golang:1.20-alpine as builder

# Set up apk dependencies
ENV PACKAGES make git libc-dev bash gcc linux-headers eudev-dev curl ca-certificates build-base

# Set working directory for the build
WORKDIR /opt/app

# Add source files
COPY . .

# Install minimum necessary dependencies, remove packages
RUN apk add --no-cache $PACKAGES

# For Private REPO
ARG GH_TOKEN=""
RUN go env -w GOPRIVATE="github.com/bnb-chain/*"
RUN git config --global url."https://${GH_TOKEN}@github.com".insteadOf "https://github.com"

RUN make build

# Pull binary into a second stage deploy alpine container
FROM alpine:3.17

ARG USER=app
ARG USER_UID=1000
ARG USER_GID=1000

ENV BLOB_SYNCER_HOME /opt/app
ENV CONFIG_FILE_PATH $BLOB_SYNCER_HOME/config/config.json
ENV PRIVATE_KEY ""
ENV DB_USERNAME ""
ENV DB_PASSWORD ""

ENV PACKAGES ca-certificates libstdc++
ENV WORKDIR=/app

RUN apk add --no-cache $PACKAGES \
  && rm -rf /var/cache/apk/* \
  && addgroup -g ${USER_GID} ${USER} \
  && adduser -u ${USER_UID} -G ${USER} --shell /sbin/nologin --no-create-home -D ${USER} \
  && addgroup ${USER} tty \
  && sed -i -e "s/bin\/sh/bin\/bash/" /etc/passwd

WORKDIR ${WORKDIR}

COPY --from=builder /opt/app/build/blob-syncer ${WORKDIR}/
RUN chown -R ${USER_UID}:${USER_GID} ${WORKDIR}
USER ${USER_UID}:${USER_GID}

VOLUME [ $BLOB_SYNCER_HOME ]

# Run the app
CMD /app/blob-syncer --config-path "$CONFIG_FILE_PATH"