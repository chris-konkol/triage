CREATE TABLE analytics_snapshots (
    id BIGSERIAL PRIMARY KEY,
    snapshot_date DATE NOT NULL,
    tickets_created INT DEFAULT 0,
    tickets_resolved INT DEFAULT 0,
    tickets_by_priority JSONB DEFAULT '{}',
    tickets_by_category JSONB DEFAULT '{}',
    avg_resolution_hours DOUBLE PRECISION,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_analytics_date ON analytics_snapshots(snapshot_date);
