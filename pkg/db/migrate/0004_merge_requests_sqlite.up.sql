CREATE TABLE IF NOT EXISTS merge_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  source_branch TEXT NOT NULL,
  target_branch TEXT NOT NULL,
  state INTEGER NOT NULL DEFAULT 0,
  author_id INTEGER NOT NULL,
  merged_by INTEGER,
  merged_at DATETIME,
  closed_by INTEGER,
  closed_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL,
  CONSTRAINT repo_id_fk
  FOREIGN KEY(repo_id) REFERENCES repos(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT author_id_fk
  FOREIGN KEY(author_id) REFERENCES users(id)
  ON DELETE CASCADE
  ON UPDATE CASCADE,
  CONSTRAINT merged_by_fk
  FOREIGN KEY(merged_by) REFERENCES users(id)
  ON DELETE SET NULL
  ON UPDATE CASCADE,
  CONSTRAINT closed_by_fk
  FOREIGN KEY(closed_by) REFERENCES users(id)
  ON DELETE SET NULL
  ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_merge_requests_repo_id ON merge_requests(repo_id);
CREATE INDEX IF NOT EXISTS idx_merge_requests_author_id ON merge_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_merge_requests_state ON merge_requests(state);
