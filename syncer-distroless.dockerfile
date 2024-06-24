FROM golang:1.22-alpine as builder

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

RUN make build_syncer


FROM alpine:3.17

ARG USER=app
ARG USER_UID=1000
ARG USER_GID=1000

ENV PACKAGES ca-certificates libstdc++ curl
ENV WORKDIR=/app

RUN apk add --no-cache $PACKAGES \
  && rm -rf /var/cache/apk/* \
  && addgroup -g ${USER_GID} ${USER} \
  && adduser -u ${USER_UID} -G ${USER} --shell /sbin/nologin --no-create-home -D ${USER} \
  && addgroup ${USER} tty \
  && sed -i -e "s/bin\/sh/bin\/bash/" /etc/passwd

WORKDIR ${WORKDIR}
RUN chown -R ${USER_UID}:${USER_GID} ${WORKDIR}
USER ${USER_UID}:${USER_GID}

ENV CONFIG_FILE_PATH /opt/app/config/config.json

ENV WORKDIR=/app
WORKDIR ${WORKDIR}
COPY --from=builder /opt/app/build/syncer ${WORKDIR}

# Run the app
CMD /app/syncer --config-path "$CONFIG_FILE_PATH"