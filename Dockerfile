FROM postgres:16-alpine

COPY themis-backup-14.sql /docker-entrypoint-initdb.d/