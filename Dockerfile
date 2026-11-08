# SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
# SPDX-License-Identifier: Apache-2.0
FROM alpine:latest

# The dms binary is built by goreleaser and provided in the build context.
COPY dms /dms

# Include compliance details about the container and what it contains.
COPY Dockerfile /
COPY NOTICE     /
COPY LICENSE    /

USER nobody

ENTRYPOINT ["/dms"]

EXPOSE 8080
