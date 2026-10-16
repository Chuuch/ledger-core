CREATE TABLE journal_entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reference TEXT,
  description TEXT NOT NULL CHECK (length(trim(description)) > 0),
  status TEXT NOT NULL CHECK (status IN ('posted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX journal_entries_reference_uidx
  ON journal_entries (reference)
  WHERE reference IS NOT NULL;

CREATE INDEX journal_entries_created_at_idx
  ON journal_entries (created_at DESC);

CREATE TABLE postings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  journal_entry_id UUID NOT NULL REFERENCES journal_entries (id),
  account_id UUID NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
  amount BIGINT NOT NULL CHECK (amount > 0),
  currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX postings_journal_entry_id_idx
  ON postings (journal_entry_id);

CREATE INDEX postings_account_id_idx
  ON postings (account_id);

CREATE INDEX postings_account_id_created_at_idx
  ON postings (account_id, created_at DESC);

CREATE TABLE account_balances (
  account_id UUID NOT NULL,
  currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
  balance BIGINT NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  PRIMARY KEY (account_id, currency)
);

