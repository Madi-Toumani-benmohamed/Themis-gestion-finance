FROM postgres:16-alpine

COPY themis-backup-15.sql /docker-entrypoint-initdb.d/