FROM golang:1.17-alpine AS build-env

RUN apk add ca-certificates

WORKDIR /go/src/app

COPY . .

RUN GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build -o /app


FROM scratch


ENV GITLAB_LDAP_GROUP_MAPPER_LDAP_BINDPASSWORD change-me
ENV GITLAB_LDAP_GROUP_MAPPER_LDAP_BINDUSERNAME change-me
ENV GITLAB_LDAP_GROUP_MAPPER_LDAP_FQDN example.com
ENV GITLAB_LDAP_GROUP_MAPPER_LDAP_BASEDN dc=example,dc=com
ENV GITLAB_LDAP_GROUP_MAPPER_LDAP_FILTER (&(objectCategory=user)(memberOf:1.2.840.113556.1.4.1941:=CN=%s,OU=Something,DC=example,dc=com))
ENV GITLAB_LDAP_GROUP_MAPPER_GITLAB_TOKEN change-me
ENV GITLAB_LDAP_GROUP_MAPPER_GITLAB_DOMAIN https://gitlab.com/api/v4

COPY --from=build-env /etc/ssl/certs /etc/ssl/certs
COPY --from=build-env /app /app

ENTRYPOINT ["/app"]
