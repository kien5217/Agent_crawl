CREATE TABLE workflow_executions (
    id            TEXT PRIMARY KEY,
    workflow_name TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'pending',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at  TIMESTAMPTZ,
    error_msg     TEXT        NOT NULL DEFAULT ''
);

CREATE TABLE step_executions (
    id                   TEXT PRIMARY KEY,
    workflow_id          TEXT        NOT NULL REFERENCES workflow_executions(id),
    step_name            TEXT        NOT NULL,
    status               TEXT        NOT NULL DEFAULT 'pending',
    attempts             INT         NOT NULL DEFAULT 0,
    result_summary_json  TEXT        NOT NULL DEFAULT '{}',
    error_msg            TEXT        NOT NULL DEFAULT '',
    started_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at         TIMESTAMPTZ
);

CREATE INDEX idx_step_executions_workflow_id ON step_executions(workflow_id);
CREATE INDEX idx_workflow_executions_started_at ON workflow_executions(started_at DESC);