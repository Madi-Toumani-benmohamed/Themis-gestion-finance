FROM python:3.12-slim

WORKDIR /app

RUN apt-get update \
 && apt-get install -y --no-install-recommends gcc libffi-dev \
 && rm -rf /var/lib/apt/lists/*

COPY memory/requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY --chmod=644 memory/main.py memory/inspection.py ./

RUN groupadd --gid 10001 themis \
 && useradd --uid 10001 --gid themis --create-home themis
USER themis

EXPOSE 8100

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8100"]
