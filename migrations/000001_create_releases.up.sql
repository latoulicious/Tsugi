CREATE TABLE releases (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version             TEXT        NOT NULL UNIQUE,
    commit_sha          TEXT        NOT NULL,
    previous_commit_sha TEXT        NOT NULL DEFAULT '',
    changelog           TEXT        NOT NULL DEFAULT '',
    status              TEXT        NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'created', 'staging', 'production', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
