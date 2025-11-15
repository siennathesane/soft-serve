CREATE TABLE IF NOT EXISTS issue_dependencies (
  id SERIAL PRIMARY KEY,
  issue_id INTEGER NOT NULL,
  depends_on_id INTEGER NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT issue_id_fk
  FOREIGN KEY(issue_id) REFERENCES issues(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT depends_on_id_fk
  FOREIGN KEY(depends_on_id) REFERENCES issues(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT unique_dependency
  UNIQUE(issue_id, depends_on_id),
  CONSTRAINT no_self_dependency
  CHECK(issue_id != depends_on_id)
);

CREATE INDEX IF NOT EXISTS idx_issue_dependencies_issue_id ON issue_dependencies(issue_id);
CREATE INDEX IF NOT EXISTS idx_issue_dependencies_depends_on_id ON issue_dependencies(depends_on_id);
