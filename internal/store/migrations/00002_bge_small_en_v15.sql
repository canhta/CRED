-- +goose Up
-- +goose StatementBegin

INSERT INTO embedding_models (id, name, dimensions, status)
VALUES (1, 'bge-small-en-v1.5', 384, 'PRESENT');

CREATE TABLE claim_embeddings_m1 PARTITION OF claim_embeddings
    FOR VALUES IN (1);

-- Vectors are normalized at write time, so inner product is the distance and
-- the norm is an invariant worth asserting. A provider that silently changes
-- normalization is caught here at write time rather than as a slow, unexplained
-- decline in recall quality.
ALTER TABLE claim_embeddings_m1
    ADD CONSTRAINT claim_embeddings_m1_is_normalized
    CHECK (abs(l2_norm(embedding::halfvec(384)) - 1) < 1e-2);

-- The dimension lives on the index expression, not on the column. The planner
-- only chooses an expression index at scale — ignored at 500 rows, used at
-- ~50k — so verify with EXPLAIN on realistic data before trusting it.
CREATE INDEX claim_embeddings_m1_hnsw
    ON claim_embeddings_m1
    USING hnsw ((embedding::halfvec(384)) halfvec_ip_ops);

-- The single local principal.
--
-- This exists so `docker compose up` reaches a working instance with no
-- additional steps: a read path that demanded a principal be created first
-- would fail PRD acceptance criterion 11 on the first command. It is a real
-- row in a real table, not a bypass — recall evaluates it through the same
-- intersection every other principal will.
INSERT INTO principals (id, kind, display_name)
VALUES ('local', 'user', 'Local user');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM principals WHERE id = 'local';
DROP TABLE IF EXISTS claim_embeddings_m1;
DELETE FROM embedding_models WHERE id = 1;
-- +goose StatementEnd
