DROP INDEX IF EXISTS idx_inbound_email_attachments_blob_key;
DROP INDEX IF EXISTS idx_inbound_email_raws_blob_key;

ALTER TABLE inbound_email_attachments
    ALTER COLUMN content SET NOT NULL;

ALTER TABLE inbound_email_attachments
    DROP COLUMN IF EXISTS blob_key;

ALTER TABLE inbound_email_raws
    ALTER COLUMN raw_source SET NOT NULL;

ALTER TABLE inbound_email_raws
    DROP COLUMN IF EXISTS blob_key;
