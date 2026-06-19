CREATE TABLE deployments (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    release_id  BIGINT      NOT NULL REFERENCES releases (id),
    environment TEXT        NOT NULL CHECK (environment IN ('staging', 'production')),
    status      TEXT        NOT NULL CHECK (status IN ('pending', 'succeeded', 'failed')),
    deployed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX deployments_env_deployed_at_idx ON deployments (environment, deployed_at DESC);
