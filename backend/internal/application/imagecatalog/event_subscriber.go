package imagecatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/image"
	"github.com/srikarm/image-factory/internal/infrastructure/messaging"
	"go.uber.org/zap"
)

type BuildRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*build.Build, error)
}

type ExecutionReader interface {
	GetExecution(ctx context.Context, executionID uuid.UUID) (*build.BuildExecution, error)
}

type BuildExecutionLister interface {
	GetBuildExecutions(ctx context.Context, buildID uuid.UUID, limit, offset int) ([]build.BuildExecution, int64, error)
}

type ImageRepository interface {
	FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*image.Image, error)
	Save(ctx context.Context, image *image.Image) error
	Update(ctx context.Context, image *image.Image) error
}

type ImageVersionRepository interface {
	FindByImageAndVersion(ctx context.Context, imageID uuid.UUID, version string) (*image.ImageVersion, error)
	Save(ctx context.Context, version *image.ImageVersion) error
	Update(ctx context.Context, version *image.ImageVersion) error
}

type EventSubscriber struct {
	buildRepo                        BuildRepository
	executions                       ExecutionReader
	imageRepo                        ImageRepository
	versionRepo                      ImageVersionRepository
	evidenceRepo                     BuildEvidenceRepository
	logger                           *zap.Logger
	strictExecutionIDResolution      atomic.Bool
	eventsReceived                   atomic.Int64
	missingExecutionID               atomic.Int64
	explicitExecutionLookupFailures  atomic.Int64
	fallbackExecutionLookupAttempts  atomic.Int64
	fallbackExecutionLookupSuccesses atomic.Int64
	fallbackExecutionLookupSkipped   atomic.Int64
	ingestsMissingEvidence           atomic.Int64
	alertsEmitted                    atomic.Int64
	alertDeliveryFailures            atomic.Int64
	alertNotifier                    IngestAlertNotifier
}

type Snapshot struct {
	EventsReceived                  int64
	MissingExecutionID              int64
	ExplicitExecutionLookupFailures int64
	FallbackExecutionLookupAttempts int64
	FallbackExecutionLookupSuccess  int64
	FallbackExecutionLookupSkipped  int64
	IngestsMissingEvidence          int64
	AlertsEmitted                   int64
	AlertDeliveryFailures           int64
}

type IngestAlertNotifier interface {
	NotifyCatalogIngestIssue(
		ctx context.Context,
		tenantID uuid.UUID,
		buildID uuid.UUID,
		imageID uuid.UUID,
		title string,
		message string,
		metadata map[string]interface{},
	) error
}

func NewEventSubscriber(
	buildRepo BuildRepository,
	executions ExecutionReader,
	imageRepo ImageRepository,
	versionRepo ImageVersionRepository,
	logger *zap.Logger,
) *EventSubscriber {
	return &EventSubscriber{
		buildRepo:   buildRepo,
		executions:  executions,
		imageRepo:   imageRepo,
		versionRepo: versionRepo,
		logger:      logger,
	}
}

func (s *EventSubscriber) SetBuildEvidenceRepository(repo BuildEvidenceRepository) {
	if s == nil {
		return
	}
	s.evidenceRepo = repo
}

func (s *EventSubscriber) SetStrictExecutionResolution(strict bool) {
	if s == nil {
		return
	}
	s.strictExecutionIDResolution.Store(strict)
}

func (s *EventSubscriber) Snapshot() Snapshot {
	if s == nil {
		return Snapshot{}
	}
	return Snapshot{
		EventsReceived:                  s.eventsReceived.Load(),
		MissingExecutionID:              s.missingExecutionID.Load(),
		ExplicitExecutionLookupFailures: s.explicitExecutionLookupFailures.Load(),
		FallbackExecutionLookupAttempts: s.fallbackExecutionLookupAttempts.Load(),
		FallbackExecutionLookupSuccess:  s.fallbackExecutionLookupSuccesses.Load(),
		FallbackExecutionLookupSkipped:  s.fallbackExecutionLookupSkipped.Load(),
		IngestsMissingEvidence:          s.ingestsMissingEvidence.Load(),
		AlertsEmitted:                   s.alertsEmitted.Load(),
		AlertDeliveryFailures:           s.alertDeliveryFailures.Load(),
	}
}

func (s *EventSubscriber) SetAlertNotifier(notifier IngestAlertNotifier) {
	if s == nil {
		return
	}
	s.alertNotifier = notifier
}

func RegisterEventSubscriber(bus messaging.EventBus, subscriber *EventSubscriber) []func() {
	if bus == nil || subscriber == nil {
		return nil
	}
	return []func(){
		bus.Subscribe(messaging.EventTypeBuildExecutionCompleted, subscriber.HandleBuildExecutionCompleted),
	}
}

func (s *EventSubscriber) HandleBuildExecutionCompleted(ctx context.Context, event messaging.Event) {
	if s == nil || s.buildRepo == nil || s.imageRepo == nil || s.versionRepo == nil {
		return
	}
	s.eventsReceived.Add(1)

	if !strings.EqualFold(strings.TrimSpace(stringPayload(event.Payload, "status")), "success") {
		return
	}

	buildID, err := parseUUIDPayload(event.Payload, "build_id")
	if err != nil {
		s.logDebug("image catalog subscriber ignored event with invalid build_id", zap.Error(err))
		return
	}

	b, err := s.buildRepo.FindByID(ctx, buildID)
	if err != nil || b == nil {
		s.logWarn("image catalog subscriber failed to load build",
			zap.String("build_id", buildID.String()),
			zap.Error(err))
		return
	}

	imageRef := inferImageReference(b)
	if imageRef == "" {
		s.logDebug("image catalog subscriber skipped build without registry repository",
			zap.String("build_id", buildID.String()))
		return
	}

	repoNoTag, tag, digest := splitImageReference(imageRef)
	imageName := deriveImageName(repoNoTag)
	if imageName == "" {
		imageName = strings.TrimSpace(b.Manifest().Name)
	}
	if imageName == "" {
		s.logDebug("image catalog subscriber skipped build without derivable image name",
			zap.String("build_id", buildID.String()))
		return
	}
	if tag == "" {
		tag = inferTag(b)
	}
	if tag == "" {
		tag = "latest"
	}
	executionID, executionIDSource := parseExecutionID(event.Payload)
	if executionID == "" {
		s.missingExecutionID.Add(1)
		s.logWarn("image catalog subscriber received event without execution_id",
			zap.String("build_id", b.ID().String()))
		s.emitAlert(ctx, b.TenantID(), b.ID(), uuid.Nil,
			"Image catalog ingest missing execution_id",
			"Build completion event did not include execution_id; strict ingest may skip fallback resolution.",
			map[string]interface{}{"reason": "missing_execution_id"})
	}
	execution, executionSource := s.resolveExecution(ctx, b.ID(), executionID)
	if executionID != "" && execution == nil {
		s.explicitExecutionLookupFailures.Add(1)
		s.logWarn("image catalog subscriber could not resolve execution_id from event payload",
			zap.String("build_id", b.ID().String()),
			zap.String("execution_id", executionID),
			zap.String("execution_id_source", executionIDSource))
		s.emitAlert(ctx, b.TenantID(), b.ID(), uuid.Nil,
			"Image catalog ingest execution resolution failed",
			"Build completion event contained execution_id but the execution record could not be resolved.",
			map[string]interface{}{"execution_id": executionID, "execution_id_source": executionIDSource})
	}
	if executionSource == "fallback" {
		s.logWarn("image catalog subscriber used execution fallback resolution",
			zap.String("build_id", b.ID().String()),
			zap.String("execution_id_source", executionIDSource),
			zap.Bool("strict_execution_id_resolution", s.strictExecutionIDResolution.Load()))
	}
	if digest == "" && b.Result() != nil {
		digest = strings.TrimSpace(b.Result().ImageDigest)
	}
	if digest == "" {
		digest = inferDigestFromArtifacts(parseExecutionArtifacts(execution))
	}

	sizePtr := inferImageSize(b)
	createdBy := uuid.Nil
	if by := b.CreatedBy(); by != nil {
		createdBy = *by
	}

	img, err := s.upsertCatalogImage(ctx, b, imageName, repoNoTag, createdBy)
	if err != nil || img == nil {
		s.logWarn("image catalog subscriber failed to upsert catalog image",
			zap.String("build_id", b.ID().String()),
			zap.String("image_name", imageName),
			zap.Error(err))
		return
	}

	ev := deriveEvidence(b, img.ID(), execution)
	if ev != nil && len(ev.Layers) == 0 && ev.SBOM == nil && ev.VulnerabilityScan == nil {
		s.ingestsMissingEvidence.Add(1)
		s.logWarn("image catalog subscriber ingested build without layer/sbom/vulnerability evidence",
			zap.String("build_id", b.ID().String()),
			zap.String("image_id", img.ID().String()),
			zap.String("execution_resolution", executionSource))
		s.emitAlert(ctx, b.TenantID(), b.ID(), img.ID(),
			"Image catalog ingest missing evidence",
			"Build ingest completed without layers, SBOM, or vulnerability evidence; metadata is marked stale/unavailable.",
			map[string]interface{}{"execution_resolution": executionSource})
	}
	if err := s.upsertCatalogVersion(ctx, img.ID(), tag, digest, sizePtr, createdBy, b, executionID); err != nil {
		s.logWarn("image catalog subscriber failed to upsert image version",
			zap.String("build_id", b.ID().String()),
			zap.String("image_id", img.ID().String()),
			zap.String("version", tag),
			zap.Error(err))
		s.emitAlert(ctx, b.TenantID(), b.ID(), img.ID(),
			"Image catalog version upsert failed",
			"Image catalog failed to upsert image version; evidence persistence may be incomplete.",
			map[string]interface{}{"version": tag, "error": err.Error()})
		if s.evidenceRepo != nil && ev != nil {
			if persistErr := s.evidenceRepo.PersistBuildEvidence(ctx, ev); persistErr != nil {
				s.logWarn("image catalog subscriber failed to persist evidence after version upsert failure",
					zap.String("build_id", b.ID().String()),
					zap.String("image_id", img.ID().String()),
					zap.Error(persistErr))
			}
		}
		return
	}
	if s.evidenceRepo != nil && ev != nil {
		if persistErr := s.evidenceRepo.PersistBuildEvidence(ctx, ev); persistErr != nil {
			s.logWarn("image catalog subscriber failed to persist evidence",
				zap.String("build_id", b.ID().String()),
				zap.String("image_id", img.ID().String()),
				zap.Error(persistErr))
			s.emitAlert(ctx, b.TenantID(), b.ID(), img.ID(),
				"Image catalog evidence persistence failed",
				"Image catalog could not persist normalized evidence for this build.",
				map[string]interface{}{"error": persistErr.Error()})
		}
	}

	s.logInfo("image catalog auto-ingested successful build",
		zap.String("build_id", b.ID().String()),
		zap.String("image_id", img.ID().String()),
		zap.String("image_name", imageName),
		zap.String("version", tag))
}

func (s *EventSubscriber) getExecution(ctx context.Context, executionID string) *build.BuildExecution {
	if s == nil || s.executions == nil || strings.TrimSpace(executionID) == "" {
		return nil
	}
	id, err := uuid.Parse(strings.TrimSpace(executionID))
	if err != nil {
		return nil
	}
	execution, err := s.executions.GetExecution(ctx, id)
	if err != nil {
		return nil
	}
	return execution
}

func (s *EventSubscriber) resolveExecution(ctx context.Context, buildID uuid.UUID, executionID string) (*build.BuildExecution, string) {
	if execution := s.getExecution(ctx, executionID); execution != nil {
		return execution, "explicit"
	}
	if s == nil || s.executions == nil {
		return nil, "none"
	}
	if s.strictExecutionIDResolution.Load() {
		s.fallbackExecutionLookupSkipped.Add(1)
		return nil, "strict_skip"
	}
	lister, ok := s.executions.(BuildExecutionLister)
	if !ok {
		return nil, "none"
	}
	s.fallbackExecutionLookupAttempts.Add(1)
	executions, _, err := lister.GetBuildExecutions(ctx, buildID, 10, 0)
	if err != nil || len(executions) == 0 {
		return nil, "none"
	}
	for _, execution := range executions {
		if execution.Status == build.ExecutionSuccess {
			candidate := execution
			s.fallbackExecutionLookupSuccesses.Add(1)
			return &candidate, "fallback"
		}
	}
	candidate := executions[0]
	s.fallbackExecutionLookupSuccesses.Add(1)
	return &candidate, "fallback"
}

func deriveEvidence(b *build.Build, imageID uuid.UUID, execution *build.BuildExecution) *BuildEvidence {
	if b == nil || imageID == uuid.Nil {
		return nil
	}
	evidence := &BuildEvidence{
		BuildID: b.ID(),
		ImageID: imageID,
	}
	if completedAt := b.CompletedAt(); completedAt != nil {
		ts := completedAt.UTC()
		evidence.BuildCompletedAt = &ts
	}
	if b.Result() != nil {
		evidence.ImageDigest = strings.TrimSpace(b.Result().ImageDigest)
		if b.Result().Size > 0 {
			size := b.Result().Size
			evidence.ImageSizeBytes = &size
		}
		if b.Result().Duration > 0 {
			evidence.BuildDurationSeconds = int(b.Result().Duration.Seconds())
		}
		if scan := deriveVulnerabilityScanFromResult(b.Result()); scan != nil {
			evidence.VulnerabilityScan = scan
			evidence.ScanTool = scan.Tool
		}
		if sbom := deriveSBOMFromResult(b.Result()); sbom != nil {
			evidence.SBOM = sbom
		}
	}
	artifacts := parseExecutionArtifacts(execution)
	if len(artifacts) > 0 {
		evidence.Artifacts = append(evidence.Artifacts, artifacts...)
		if evidence.ImageDigest == "" {
			evidence.ImageDigest = inferDigestFromArtifacts(artifacts)
		}
	}
	if execution != nil {
		evidence.Layers = append(evidence.Layers, parseLayerEvidence(execution.Artifacts)...)
		if evidence.SBOM == nil {
			evidence.SBOM = parseSBOMEvidence(execution.Artifacts)
		}
		if evidence.VulnerabilityScan == nil {
			evidence.VulnerabilityScan = parseVulnerabilityEvidence(execution.Artifacts)
		}
	}
	if evidence.VulnerabilityScan != nil && evidence.ScanTool == "" {
		evidence.ScanTool = evidence.VulnerabilityScan.Tool
	}
	return evidence
}

func (s *EventSubscriber) emitAlert(ctx context.Context, tenantID, buildID, imageID uuid.UUID, title, message string, metadata map[string]interface{}) {
	if s == nil || s.alertNotifier == nil || tenantID == uuid.Nil || buildID == uuid.Nil {
		return
	}
	if err := s.alertNotifier.NotifyCatalogIngestIssue(ctx, tenantID, buildID, imageID, title, message, metadata); err != nil {
		s.alertDeliveryFailures.Add(1)
		s.logWarn("image catalog subscriber failed to deliver ingest alert",
			zap.String("build_id", buildID.String()),
			zap.Error(err))
		return
	}
	s.alertsEmitted.Add(1)
}

func parseExecutionArtifacts(execution *build.BuildExecution) []BuildArtifactEvidence {
	if execution == nil || len(execution.Artifacts) == 0 {
		return nil
	}
	var raw []map[string]interface{}
	if err := json.Unmarshal(execution.Artifacts, &raw); err == nil {
		return toBuildArtifactEvidence(raw)
	}
	var names []string
	if err := json.Unmarshal(execution.Artifacts, &names); err == nil {
		result := make([]BuildArtifactEvidence, 0, len(names))
		for _, name := range names {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			result = append(result, BuildArtifactEvidence{
				Type:        "build_output",
				Name:        trimmed,
				Location:    trimmed,
				IsAvailable: true,
			})
		}
		return result
	}
	return nil
}

func toBuildArtifactEvidence(raw []map[string]interface{}) []BuildArtifactEvidence {
	result := make([]BuildArtifactEvidence, 0, len(raw))
	for _, item := range raw {
		name := stringField(item, "name")
		location := firstNonEmpty(
			stringField(item, "value"),
			stringField(item, "path"),
			stringField(item, "location"),
		)
		result = append(result, BuildArtifactEvidence{
			Type:        firstNonEmpty(stringField(item, "artifact_type"), stringField(item, "type"), "build_output"),
			Name:        firstNonEmpty(name, location),
			Version:     firstNonEmpty(stringField(item, "version"), stringField(item, "artifact_version")),
			Location:    location,
			MimeType:    firstNonEmpty(stringField(item, "content_type"), stringField(item, "mime_type")),
			SizeBytes:   int64PtrField(item, "size"),
			SHA256:      firstNonEmpty(stringField(item, "sha256"), stringField(item, "checksum"), stringField(item, "digest")),
			IsAvailable: true,
		})
	}
	return result
}

func parseLayerEvidence(rawPayload []byte) []LayerEvidence {
	if len(rawPayload) == 0 {
		return nil
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(rawPayload, &rows); err != nil {
		return nil
	}
	layers := make([]LayerEvidence, 0, 8)
	seenDigests := make(map[string]struct{}, 8)
	for _, row := range rows {
		hint := strings.ToLower(firstNonEmpty(stringField(row, "name"), stringField(row, "type")))
		if !strings.Contains(hint, "layer") {
			continue
		}
		rawValue := firstNonEmpty(stringField(row, "value"), stringField(row, "path"), stringField(row, "name"))
		if !strings.HasPrefix(rawValue, "[") {
			continue
		}
		var rawLayers []map[string]interface{}
		if err := json.Unmarshal([]byte(rawValue), &rawLayers); err != nil {
			continue
		}
		for _, layer := range rawLayers {
			digest := firstNonEmpty(stringField(layer, "digest"), stringField(layer, "layer_digest"))
			if digest == "" {
				continue
			}
			digest = strings.TrimSpace(digest)
			if _, exists := seenDigests[digest]; exists {
				continue
			}
			seenDigests[digest] = struct{}{}
			layerNumber := len(layers)
			layers = append(layers, LayerEvidence{
				LayerNumber:      layerNumber,
				Digest:           digest,
				SizeBytes:        int64PtrField(layer, "size"),
				MediaType:        firstNonEmpty(stringField(layer, "mediaType"), stringField(layer, "media_type")),
				HistoryCreatedBy: firstNonEmpty(stringField(layer, "createdBy"), stringField(layer, "history_created_by")),
				SourceCommand:    firstNonEmpty(stringField(layer, "command"), stringField(layer, "source_command")),
				DiffID:           firstNonEmpty(stringField(layer, "diffID"), stringField(layer, "diff_id")),
			})
		}
	}
	return layers
}

func parseSBOMEvidence(rawPayload []byte) *SBOMEvidence {
	if len(rawPayload) == 0 {
		return nil
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(rawPayload, &rows); err != nil {
		return nil
	}
	var best *SBOMEvidence
	bestScore := -1
	bestContentLen := -1
	for _, row := range rows {
		name := strings.ToLower(firstNonEmpty(stringField(row, "name"), stringField(row, "type")))
		if !strings.Contains(name, "sbom") {
			continue
		}
		content := firstNonEmpty(stringField(row, "value"), stringField(row, "path"))
		if strings.TrimSpace(content) == "" {
			continue
		}
		sbom := &SBOMEvidence{
			Format:      "spdx",
			Content:     content,
			GeneratedBy: firstNonEmpty(stringField(row, "tool"), "unknown"),
			Status:      "valid",
		}
		score := 0
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err == nil {
			sbom.Content = content
			sbom.Format = firstNonEmpty(stringField(parsed, "bomFormat"), stringField(parsed, "spdxVersion"), "spdx")
			sbom.Version = firstNonEmpty(stringField(parsed, "specVersion"), stringField(parsed, "version"))
			sbom.GeneratedBy = firstNonEmpty(stringField(parsed, "generator"), sbom.GeneratedBy)
			sbom.Packages = parseSBOMPackages(parsed)
			score += len(sbom.Packages) * 10
			if intField(parsed, "package_count") > 0 {
				score += intField(parsed, "package_count")
			}
			if _, ok := parsed["packages"]; ok {
				score += 5
			}
			if _, ok := parsed["components"]; ok {
				score += 5
			}
		}
		contentLen := len(content)
		if score > bestScore || (score == bestScore && contentLen > bestContentLen) {
			best = sbom
			bestScore = score
			bestContentLen = contentLen
		}
	}
	return best
}

func parseSBOMPackages(parsed map[string]interface{}) []SBOMPackageEvidence {
	pkgsRaw, ok := parsed["packages"]
	if !ok {
		pkgsRaw = parsed["components"]
	}
	rows, ok := pkgsRaw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]SBOMPackageEvidence, 0, len(rows))
	for _, row := range rows {
		pkg, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		name := firstNonEmpty(stringField(pkg, "name"), stringField(pkg, "purl"))
		if name == "" {
			continue
		}
		packagePath, layerDigest, sourceCommand := parseSBOMPackageSource(pkg)
		result = append(result, SBOMPackageEvidence{
			Name:          name,
			Version:       stringField(pkg, "version"),
			Type:          firstNonEmpty(stringField(pkg, "type"), stringField(pkg, "package_type")),
			PackageURL:    firstNonEmpty(stringField(pkg, "purl"), stringField(pkg, "package_url")),
			HomepageURL:   stringField(pkg, "homepage"),
			LicenseName:   stringField(pkg, "licenseDeclared"),
			PackagePath:   packagePath,
			LayerDigest:   layerDigest,
			SourceCommand: sourceCommand,
		})
	}
	return result
}

func parseSBOMPackageSource(pkg map[string]interface{}) (packagePath, layerDigest, sourceCommand string) {
	if pkg == nil {
		return "", "", ""
	}
	packagePath = firstNonEmpty(
		stringField(pkg, "package_path"),
		stringField(pkg, "path"),
		stringField(pkg, "filePath"),
	)
	layerDigest = firstNonEmpty(
		stringField(pkg, "layer_digest"),
		stringField(pkg, "layerID"),
		stringField(pkg, "layerId"),
	)
	sourceCommand = firstNonEmpty(
		stringField(pkg, "source_command"),
		stringField(pkg, "sourceCommand"),
	)
	if locations, ok := pkg["locations"].([]interface{}); ok {
		for _, locationRaw := range locations {
			location, ok := locationRaw.(map[string]interface{})
			if !ok {
				continue
			}
			if packagePath == "" {
				packagePath = firstNonEmpty(stringField(location, "path"), stringField(location, "filePath"))
			}
			if layerDigest == "" {
				layerDigest = firstNonEmpty(
					stringField(location, "layerID"),
					stringField(location, "layerId"),
					stringField(location, "layer_digest"),
				)
			}
			if sourceCommand == "" {
				sourceCommand = firstNonEmpty(
					stringField(location, "source_command"),
					stringField(location, "sourceCommand"),
				)
			}
			if packagePath != "" && layerDigest != "" && sourceCommand != "" {
				break
			}
		}
	}
	if properties, ok := pkg["properties"].([]interface{}); ok {
		for _, raw := range properties {
			property, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			key := strings.ToLower(firstNonEmpty(stringField(property, "name"), stringField(property, "key")))
			value := firstNonEmpty(stringField(property, "value"), stringField(property, "val"))
			if key == "" || value == "" {
				continue
			}
			switch {
			case strings.Contains(key, "layer") && layerDigest == "":
				layerDigest = value
			case (strings.Contains(key, "path") || strings.Contains(key, "location")) && packagePath == "":
				packagePath = value
			case strings.Contains(key, "command") && sourceCommand == "":
				sourceCommand = value
			}
		}
	}
	return packagePath, layerDigest, sourceCommand
}

func parseVulnerabilityEvidence(rawPayload []byte) *VulnerabilityScanEvidence {
	if len(rawPayload) == 0 {
		return nil
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(rawPayload, &rows); err != nil {
		return nil
	}
	for _, row := range rows {
		name := strings.ToLower(firstNonEmpty(stringField(row, "name"), stringField(row, "type")))
		if !strings.Contains(name, "scan") && !strings.Contains(name, "vuln") {
			continue
		}
		content := firstNonEmpty(stringField(row, "value"), stringField(row, "path"))
		scan := &VulnerabilityScanEvidence{
			Tool:   firstNonEmpty(stringField(row, "tool"), "unknown"),
			Status: "completed",
		}
		if strings.TrimSpace(content) == "" {
			return scan
		}
		scan.ReportJSON = content
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(content), &parsed); err == nil {
			v := deriveVulnerabilityCounts(parsed)
			scan.Critical = v.Critical
			scan.High = v.High
			scan.Medium = v.Medium
			scan.Low = v.Low
			scan.Negligible = v.Negligible
			scan.Unknown = v.Unknown
			scan.PassFail = v.PassFail
		}
		return scan
	}
	return nil
}

func deriveSBOMFromResult(result *build.BuildResult) *SBOMEvidence {
	if result == nil || len(result.SBOM) == 0 {
		return nil
	}
	raw, err := json.Marshal(result.SBOM)
	if err != nil {
		return nil
	}
	evidence := &SBOMEvidence{
		Format:      firstNonEmpty(stringField(result.SBOM, "bomFormat"), stringField(result.SBOM, "format"), "spdx"),
		Version:     firstNonEmpty(stringField(result.SBOM, "specVersion"), stringField(result.SBOM, "version")),
		Content:     string(raw),
		GeneratedBy: firstNonEmpty(stringField(result.SBOM, "generator"), "unknown"),
		Status:      "valid",
	}
	evidence.Packages = parseSBOMPackages(result.SBOM)
	return evidence
}

func deriveVulnerabilityScanFromResult(result *build.BuildResult) *VulnerabilityScanEvidence {
	if result == nil || len(result.ScanResults) == 0 {
		return nil
	}
	raw, _ := json.Marshal(result.ScanResults)
	counts := deriveVulnerabilityCounts(result.ScanResults)
	return &VulnerabilityScanEvidence{
		Tool:       firstNonEmpty(stringField(result.ScanResults, "tool"), "unknown"),
		Status:     "completed",
		Critical:   counts.Critical,
		High:       counts.High,
		Medium:     counts.Medium,
		Low:        counts.Low,
		Negligible: counts.Negligible,
		Unknown:    counts.Unknown,
		PassFail:   counts.PassFail,
		ReportJSON: string(raw),
	}
}

type vulnerabilityCounts struct {
	Critical   int
	High       int
	Medium     int
	Low        int
	Negligible int
	Unknown    int
	PassFail   string
}

func deriveVulnerabilityCounts(payload map[string]interface{}) vulnerabilityCounts {
	counts := vulnerabilityCounts{}
	raw := payload
	if nested, ok := payload["vulnerabilities"].(map[string]interface{}); ok {
		raw = nested
	}
	counts.Critical = intField(raw, "critical")
	counts.High = intField(raw, "high")
	counts.Medium = intField(raw, "medium")
	counts.Low = intField(raw, "low")
	counts.Negligible = intField(raw, "negligible")
	counts.Unknown = intField(raw, "unknown")
	if passed, ok := payload["passed"].(bool); ok {
		if passed {
			counts.PassFail = "PASS"
		} else {
			counts.PassFail = "FAIL"
		}
	}
	return counts
}

func inferDigestFromArtifacts(artifacts []BuildArtifactEvidence) string {
	for _, artifact := range artifacts {
		digest := strings.TrimSpace(artifact.SHA256)
		if strings.HasPrefix(digest, "sha256:") {
			return digest
		}
		if len(digest) == 64 {
			return "sha256:" + digest
		}
		candidates := []string{artifact.Name, artifact.Location}
		for _, candidate := range candidates {
			value := strings.TrimSpace(candidate)
			if strings.HasPrefix(value, "sha256:") {
				return value
			}
		}
	}
	return ""
}

func stringField(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch cast := value.(type) {
	case string:
		return strings.TrimSpace(cast)
	default:
		return strings.TrimSpace(fmt.Sprint(cast))
	}
}

func intField(data map[string]interface{}, key string) int {
	if data == nil {
		return 0
	}
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch cast := value.(type) {
	case int:
		return cast
	case int64:
		return int(cast)
	case float64:
		return int(cast)
	case json.Number:
		n, _ := cast.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(cast))
		return n
	default:
		return 0
	}
}

func int64PtrField(data map[string]interface{}, key string) *int64 {
	v := intField(data, key)
	if v <= 0 {
		return nil
	}
	value := int64(v)
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *EventSubscriber) upsertCatalogImage(
	ctx context.Context,
	b *build.Build,
	imageName, repoNoTag string,
	createdBy uuid.UUID,
) (*image.Image, error) {
	existing, err := s.imageRepo.FindByTenantAndName(ctx, b.TenantID(), imageName)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		description := fmt.Sprintf("Auto-ingested from build %s", b.ID().String())
		img, createErr := image.NewImage(b.TenantID(), imageName, description, image.VisibilityTenant, createdBy)
		if createErr != nil {
			return nil, createErr
		}
		_ = img.UpdateStatus(image.StatusPublished)
		img.SetUpdatedBy(createdBy)
		img.UpdateMetadata(map[string]interface{}{
			"source":        "build_auto_ingest",
			"build_id":      b.ID().String(),
			"project_id":    b.ProjectID().String(),
			"registry_repo": repoNoTag,
			"last_ingested": time.Now().UTC().Format(time.RFC3339),
		})
		if saveErr := s.imageRepo.Save(ctx, img); saveErr != nil {
			return nil, saveErr
		}
		return img, nil
	}

	_ = existing.UpdateStatus(image.StatusPublished)
	existing.SetUpdatedBy(createdBy)
	meta := existing.Metadata()
	if meta == nil {
		meta = map[string]interface{}{}
	}
	meta["source"] = "build_auto_ingest"
	meta["build_id"] = b.ID().String()
	meta["project_id"] = b.ProjectID().String()
	meta["registry_repo"] = repoNoTag
	meta["last_ingested"] = time.Now().UTC().Format(time.RFC3339)
	existing.UpdateMetadata(meta)
	if err := s.imageRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *EventSubscriber) upsertCatalogVersion(
	ctx context.Context,
	imageID uuid.UUID,
	version string,
	digest string,
	sizeBytes *int64,
	createdBy uuid.UUID,
	b *build.Build,
	executionID string,
) error {
	digestPtr := nullableString(digest)

	existing, err := s.versionRepo.FindByImageAndVersion(ctx, imageID, version)
	if err != nil {
		return err
	}

	metadata := map[string]interface{}{
		"source":        "build_auto_ingest",
		"build_id":      b.ID().String(),
		"project_id":    b.ProjectID().String(),
		"execution_id":  executionID,
		"last_ingested": time.Now().UTC().Format(time.RFC3339),
	}

	if existing == nil {
		v, createErr := image.NewImageVersion(imageID, version, digestPtr, sizeBytes, createdBy)
		if createErr != nil {
			return createErr
		}
		v.SetMetadata(metadata)
		return s.versionRepo.Save(ctx, v)
	}

	existing.SetDigest(digestPtr)
	existing.SetSizeBytes(sizeBytes)
	existing.SetMetadata(metadata)
	existing.SetPublishedAt(time.Now().UTC())
	return s.versionRepo.Update(ctx, existing)
}

func inferImageReference(b *build.Build) string {
	if b == nil {
		return ""
	}
	if cfg := b.Manifest().BuildConfig; cfg != nil {
		if ref := strings.TrimSpace(cfg.RegistryRepo); ref != "" {
			return ref
		}
	}
	if cfg := b.Config(); cfg != nil && cfg.Metadata != nil {
		if ref, ok := cfg.Metadata["registry_repo"].(string); ok {
			if trimmed := strings.TrimSpace(ref); trimmed != "" {
				return trimmed
			}
		}
	}
	if b.Result() != nil {
		return strings.TrimSpace(b.Result().ImageID)
	}
	return ""
}

func inferTag(b *build.Build) string {
	if b == nil {
		return ""
	}
	tags := b.Manifest().Tags
	if len(tags) == 0 {
		return ""
	}
	return strings.TrimSpace(tags[0])
}

func inferImageSize(b *build.Build) *int64 {
	if b == nil || b.Result() == nil || b.Result().Size <= 0 {
		return nil
	}
	size := b.Result().Size
	return &size
}

func deriveImageName(repo string) string {
	r := strings.TrimSpace(repo)
	if r == "" {
		return ""
	}
	parts := strings.Split(r, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func splitImageReference(ref string) (repoNoTag, tag, digest string) {
	remainder := strings.TrimSpace(ref)
	if remainder == "" {
		return "", "", ""
	}

	if at := strings.Index(remainder, "@"); at != -1 {
		digest = strings.TrimSpace(remainder[at+1:])
		remainder = strings.TrimSpace(remainder[:at])
	}

	lastSlash := strings.LastIndex(remainder, "/")
	lastColon := strings.LastIndex(remainder, ":")
	if lastColon > lastSlash {
		tag = strings.TrimSpace(remainder[lastColon+1:])
		repoNoTag = strings.TrimSpace(remainder[:lastColon])
	} else {
		repoNoTag = remainder
	}
	return repoNoTag, tag, digest
}

func parseExecutionID(payload map[string]interface{}) (string, string) {
	if direct := firstNonEmpty(
		fmt.Sprint(payload["execution_id"]),
		fmt.Sprint(payload["executionId"]),
	); direct != "" && direct != "<nil>" {
		if strings.TrimSpace(fmt.Sprint(payload["execution_id"])) != "" && strings.TrimSpace(fmt.Sprint(payload["execution_id"])) != "<nil>" {
			return strings.TrimSpace(direct), "payload.execution_id"
		}
		return strings.TrimSpace(direct), "payload.executionId"
	}
	rawMeta, ok := payload["metadata"].(map[string]interface{})
	if !ok {
		return "", ""
	}
	for _, key := range []string{"execution_id", "executionId"} {
		if value := strings.TrimSpace(fmt.Sprint(rawMeta[key])); value != "" && value != "<nil>" {
			return value, "metadata." + key
		}
	}
	return "", ""
}

func parseUUIDPayload(payload map[string]interface{}, key string) (uuid.UUID, error) {
	raw, ok := payload[key]
	if !ok {
		return uuid.Nil, fmt.Errorf("missing %s", key)
	}
	return uuid.Parse(strings.TrimSpace(fmt.Sprint(raw)))
}

func stringPayload(payload map[string]interface{}, key string) string {
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func nullableString(v string) *string {
	value := strings.TrimSpace(v)
	if value == "" {
		return nil
	}
	return &value
}

func (s *EventSubscriber) logDebug(msg string, fields ...zap.Field) {
	if s != nil && s.logger != nil {
		s.logger.Debug(msg, fields...)
	}
}

func (s *EventSubscriber) logInfo(msg string, fields ...zap.Field) {
	if s != nil && s.logger != nil {
		s.logger.Info(msg, fields...)
	}
}

func (s *EventSubscriber) logWarn(msg string, fields ...zap.Field) {
	if s != nil && s.logger != nil {
		s.logger.Warn(msg, fields...)
	}
}
