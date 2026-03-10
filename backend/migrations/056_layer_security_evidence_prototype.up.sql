CREATE TABLE IF NOT EXISTS catalog_image_layer_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE,
    layer_digest VARCHAR(255) NOT NULL,
    layer_number INTEGER,
    history_created_by TEXT,
    source_command TEXT,
    diff_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(image_id, layer_digest)
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_evidence_image_id ON catalog_image_layer_evidence(image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_evidence_layer_digest ON catalog_image_layer_evidence(layer_digest);

CREATE TABLE IF NOT EXISTS catalog_image_layer_packages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE,
    layer_digest VARCHAR(255) NOT NULL,
    package_name VARCHAR(255) NOT NULL,
    package_version VARCHAR(100) NOT NULL DEFAULT '',
    package_type VARCHAR(50),
    package_path VARCHAR(500),
    source_command TEXT,
    known_vulnerabilities_count INTEGER DEFAULT 0,
    critical_vulnerabilities_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(image_id, layer_digest, package_name, package_version)
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_packages_image_id ON catalog_image_layer_packages(image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_packages_layer_digest ON catalog_image_layer_packages(layer_digest);
CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_packages_name ON catalog_image_layer_packages(package_name);

CREATE TABLE IF NOT EXISTS catalog_image_layer_vulnerabilities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    image_id UUID NOT NULL REFERENCES catalog_images(id) ON DELETE CASCADE,
    layer_digest VARCHAR(255) NOT NULL,
    package_name VARCHAR(255) NOT NULL,
    package_version VARCHAR(100) NOT NULL DEFAULT '',
    cve_id VARCHAR(20) NOT NULL REFERENCES cve_database(cve_id) ON DELETE CASCADE,
    severity VARCHAR(20),
    cvss_v3_score DECIMAL(4,1),
    reference_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(image_id, layer_digest, package_name, package_version, cve_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_vuln_image_id ON catalog_image_layer_vulnerabilities(image_id);
CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_vuln_layer_digest ON catalog_image_layer_vulnerabilities(layer_digest);
CREATE INDEX IF NOT EXISTS idx_catalog_image_layer_vuln_cve_id ON catalog_image_layer_vulnerabilities(cve_id);

CREATE TRIGGER update_catalog_image_layer_evidence_updated_at
    BEFORE UPDATE ON catalog_image_layer_evidence
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
