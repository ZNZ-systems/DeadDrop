ALTER TABLE inbound_email_raws
    ADD COLUMN blob_key TEXT NOT NULL DEFAULT '';

ALTER TABLE inbound_email_raws
    ALTER COLUMN raw_source DROP NOT NULL;

ALTER TABLE inbound_email_attachments
    ADD COLUMN blob_key TEXT NOT NULL DEFAULT '';

ALTER TABLE inbound_email_attachments
    ALTER COLUMN content DROP NOT NULL;

CREATE INDEX idx_inbound_email_raws_blob_key
ON inbound_email_raws(blob_key)
WHERE blob_key <> '';

CREATE INDEX idx_inbound_email_attachments_blob_key
ON inbound_email_attachments(blob_key)
WHERE blob_key <> '';
