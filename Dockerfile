FROM postgres:16-alpine

COPY themis-backup-7.sql /docker-entrypoint-initdb.d/