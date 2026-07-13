ALTER TABLE links DROP CONSTRAINT IF EXISTS links_original_url_check;
ALTER TABLE links DROP CONSTRAINT IF EXISTS links_original_url_length_check;
ALTER TABLE links ADD CONSTRAINT links_original_url_length_check CHECK (length(original_url) <= 65535);
