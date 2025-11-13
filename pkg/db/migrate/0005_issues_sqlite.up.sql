CREATE TABLE IF NOT EXISTS issues (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  state INTEGER NOT NULL DEFAULT 0,
  author_id INTEGER NOT NULL,
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
  CONSTRAINT closed_by_fk
  FOREIGN KEY(closed_by) REFERENCES users(id)
  ON DELETE SET NULL
  ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_issues_repo_id ON issues(repo_id);
CREATE INDEX IF NOT EXISTS idx_issues_author_id ON issues(author_id);
CREATE INDEX IF NOT EXISTS idx_issues_state ON issues(state);
