# Unofficial Postgres Exporter for OTEL

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/destrex271/postgresexporter)
![GitHub Repo stars](https://img.shields.io/github/stars/destrex271/postgresexporter)

*The official development for this component is being carried out in this pull request -> https://github.com/open-telemetry/opentelemetry-collector-contrib/pull/36403*
<hr/>

This repository contains an unofficial implementation for an <a href="https://opentelemetry.io/">Open Telemetry</a> Exporter to export otel logs, traces and metrics to PostgreSQL.
The original issue for the development of this exporter can be found here: https://github.com/paradedb/paradedb/issues/1632

## Usage

To try this exporter you can build your custom OTEL Collector.

1. Clone the repository
```
git clone https://github.com/destrex271/postgresexporter
```

2. In the same directory where your repository is cloned, add the following distribution config in a file named `builder_config.yml`

```yaml
dist:
  name: otellocalcol
  description: OpenTelemetry Collector Local for tests.
  version: 1.0.0
  output_path: . 

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.124.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.124.0
  - gomod: go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.124.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.124.0
  - gomod: github.com/destrex271/postgresexporter main
    path: "./postgresexporter" # if you have repo locally

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.28.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v1.28.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpprovider v1.28.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.28.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.28.0

extensions:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.124.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckv2extension v0.124.0
```

3. Install the otel builder utility
```
go install go.opentelemetry.io/collector/cmd/builder@latest
```
*NOTE*: Builder version should be same as the otel collector version i.e. 0.124.0 in this case. You can replace `latest` by the version number

4. Run the builder
```
builder --config=builder-config.yml
```
This will generate an executable named `otellocalcol`

5. Run the collector to start collecting measurements

```
./otellocalcol --config="./postgresexporter/example/otel-collector-config.yml"
```
Here we are using the example config provided with the exporter. The contents of this file look something like this:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  postgresexporter:
    username: "postgres"
    password: "postgres"
    database: "postgres"
    logs_table_name: "otellogs"
    traces_table_name: "oteltraces"
    metrics_table_name: "otelmetrics"
    port: 5432
    host: "localhost"
  debug:

service:
  pipelines:
    logs:
      receivers: [otlp]
      exporters:
        - debug
        - postgresexporter
    traces:
      receivers: [otlp]
      exporters:
        - debug
        - postgresexporter
    # metrics:
    #   receivers: [otlp]
    #   exporters:
    #     - debug
    #     - postgresexporter
```

You can modify the values according to your requirements

6. Although the logs will clarify if the exporter worked or not, you can also check if the tables were
created your in postgres database. For this you can open up psql and run the following commands:

```psql
postgres=# \d
                         List of relations
 Schema |           Name            |       Type        |  Owner   
--------+---------------------------+-------------------+----------
 public | otellogs                  | table             | postgres
 public | oteltraces                | table             | postgres
 public | oteltraces_trace_id_ts    | table             | postgres
 public | oteltraces_trace_id_ts_mv | materialized view | postgres

```

Here we can see that otellogs and oteltraces tables were created.

For example we can see the columns in the otellogs table:

```psql
postgres=# \d+ otellogs
                                                                           Table "public.otellogs"
       Column       |              Type              | Collation | Nullable |                 Default                  | Storage  | Compression | Stats target | Description 
--------------------+--------------------------------+-----------+----------+------------------------------------------+----------+-------------+--------------+-------------
 Timestamp          | timestamp(6) without time zone |           | not null |                                          | plain    |             |              | 
 TimestampTime      | timestamp without time zone    |           | not null | generated always as ("Timestamp") stored | plain    |             |              | 
 TraceId            | text                           |           |          |                                          | extended |             |              | 
 SpanId             | text                           |           |          |                                          | extended |             |              | 
 TraceFlags         | smallint                       |           |          |                                          | plain    |             |              | 
 SeverityText       | text                           |           |          |                                          | extended |             |              | 
 SeverityNumber     | smallint                       |           |          |                                          | plain    |             |              | 
 ServiceName        | text                           |           | not null |                                          | extended |             |              | 
 Body               | text                           |           |          |                                          | extended |             |              | 
 ResourceSchemaUrl  | text                           |           |          |                                          | extended |             |              | 
 ResourceAttributes | jsonb                          |           |          |                                          | extended |             |              | 
 ScopeSchemaUrl     | text                           |           |          |                                          | extended |             |              | 
 ScopeName          | text                           |           |          |                                          | extended |             |              | 
 ScopeVersion       | text                           |           |          |                                          | extended |             |              | 
 ScopeAttributes    | jsonb                          |           |          |                                          | extended |             |              | 
 LogAttributes      | jsonb                          |           |          |                                          | extended |             |              | 
Indexes:
    "otellogs_pkey" PRIMARY KEY, btree ("ServiceName", "TimestampTime")
Access method: heap

```

## Supported Functionalities

 - Export OTEL Logs to Postgres
 - Export OTEL Traces to Postgres
 <hr/>
 - *Inprogress*: Export OTEL Metrics to Postgres

## Contributing
Please check out the issues section of the repository to contribute to this project
