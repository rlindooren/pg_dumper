FROM postgres:latest

RUN apt update && apt install -y golang-go curl

WORKDIR /pg_dumper
COPY pg_dumper-linux pg_dumper

CMD /pg_dumper/pg_dumper
