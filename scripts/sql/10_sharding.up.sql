-- Sequence and defined type
CREATE SEQUENCE IF NOT EXISTS id_seq_git_material_node_mapping;

-- Create GitSensorNode mapping table
CREATE TABLE git_material_node_mapping(
                                        "id" INTEGER PRIMARY KEY DEFAULT nextval('id_seq_git_material_node_mapping'::regclass),
                                        "git_material_id" INTEGER NOT NULL,
                                        "ordinal_index" INTEGER NOT NULL,
                                        "created_on" TIMESTAMPTZ,
                                        "created_by" INTEGER,
                                        "updated_on" TIMESTAMPTZ,
                                        "updated_by" INTEGER
);