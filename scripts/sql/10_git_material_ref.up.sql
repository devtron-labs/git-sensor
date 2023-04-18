---- add ref_git_material_id column
ALTER TABLE git_material ADD COLUMN IF NOT EXISTS ref_git_material_id INTEGER;

DO $$
DECLARE
    material_id integer;
    repo_url varchar;
    xnum integer;
BEGIN
    FOR material_id, repo_url, xnum IN SELECT * FROM (
                        SELECT id, url, ROW_NUMBER() OVER (PARTITION BY url ORDER BY id ASC) AS num FROM git_material WHERE deleted = false
                        ) materials where num <= 1
    LOOP
        UPDATE git_material SET ref_git_material_id = material_id WHERE url = repo_url;
        RAISE NOTICE 'updated ref_git_material_id for url: % to id: %', repo_url, material_id;
    END LOOP;
END$$;
