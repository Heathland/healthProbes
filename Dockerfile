# Define the go version
ARG GO_VERSION=1.14

##
## First stage - Building the application
##

# Builder Container
FROM golang:${GO_VERSION}-alpine AS builder

# Git is required for fetching the dependencies.
RUN apk add --no-cache git gcc musl-dev

# Set the working directory outside $GOPATH to enable the support for modules.
WORKDIR /src

# Fetch dependencies first; they are less susceptible to change on every build
# and will therefore be cached for speeding up the next build
COPY ./go.* ./
RUN go mod download

# Import the code from the context.
COPY ./ ./

# Build the executable to `/app`. Mark the build as statically linked.
RUN CGO_ENABLED=0 go build \
    -installsuffix 'static' \
-o /app .


##
## Final stage - Containerization of the application
##
FROM alpine AS final
LABEL maintainer="r.vanderheide@wearetriple.com"

# Setting default values
ENV WAIT_STARTUP_TIME 20
ENV WAIT_LIVENESS_TIME 35
ENV JOB_DURATION_TIME 20

# Import the compiled executable from the first stage.
COPY --from=builder /app /app

# Declare the port on which the webserver will be exposed.
# As we're going to run the executable as an unprivileged user, we can't bind
# to ports below 1024.
EXPOSE 8080

# Run the compiled binary.
ENTRYPOINT ["/app"]
