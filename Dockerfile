FROM golang:1.24.12

# Install Go CI/CD tools with pinned versions
RUN go install honnef.co/go/tools/cmd/staticcheck@2025.1 && \
    go install github.com/kisielk/errcheck@v1.9.0 && \
    go install github.com/jstemmer/go-junit-report/v2@v2.1.0 && \
    go install github.com/boumenot/gocover-cobertura@v1.4.0 && \
    go install github.com/securego/gosec/v2/cmd/gosec@v2.22.11 && \
    go install golang.org/x/vuln/cmd/govulncheck@v1.1.4 && \
    go install github.com/google/go-licenses@latest

# Add /go/bin to PATH (tools are installed there)
ENV PATH="/go/bin:${PATH}"

# Verify tools are available
RUN command -v staticcheck && \
    command -v errcheck && \
    command -v go-junit-report && \
    command -v gosec && \
    command -v govulncheck && \
    command -v go-licenses
