package build

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConfigTemplateRepository is a mock implementation for testing
type MockConfigTemplateRepository struct {
	templates       map[uuid.UUID]*ConfigTemplate
	shares          map[uuid.UUID][]*ConfigTemplateShare
	lastSavedError  error
	lastLoadedError error
}

func NewMockConfigTemplateRepository() *MockConfigTemplateRepository {
	return &MockConfigTemplateRepository{
		templates: make(map[uuid.UUID]*ConfigTemplate),
		shares:    make(map[uuid.UUID][]*ConfigTemplateShare),
	}
}

func (m *MockConfigTemplateRepository) SaveTemplate(ctx context.Context, template *ConfigTemplate) error {
	if m.lastSavedError != nil {
		return m.lastSavedError
	}
	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}
	template.UpdatedAt = time.Now()
	m.templates[template.ID] = template
	return nil
}

func (m *MockConfigTemplateRepository) GetTemplate(ctx context.Context, id uuid.UUID) (*ConfigTemplate, error) {
	if m.lastLoadedError != nil {
		return nil, m.lastLoadedError
	}
	template, ok := m.templates[id]
	if !ok {
		return nil, ErrTemplateNotFound
	}
	return template, nil
}

func (m *MockConfigTemplateRepository) ListTemplatesByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*ConfigTemplate, int, error) {
	var templates []*ConfigTemplate
	for _, t := range m.templates {
		if t.ProjectID == projectID {
			templates = append(templates, t)
		}
	}
	return templates, len(templates), nil
}

func (m *MockConfigTemplateRepository) UpdateTemplate(ctx context.Context, template *ConfigTemplate) error {
	if _, ok := m.templates[template.ID]; !ok {
		return ErrTemplateNotFound
	}
	template.UpdatedAt = time.Now()
	m.templates[template.ID] = template
	return nil
}

func (m *MockConfigTemplateRepository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.templates[id]; !ok {
		return ErrTemplateNotFound
	}
	delete(m.templates, id)
	return nil
}

func (m *MockConfigTemplateRepository) ShareTemplate(ctx context.Context, share *ConfigTemplateShare) error {
	m.shares[share.TemplateID] = append(m.shares[share.TemplateID], share)
	return nil
}

func (m *MockConfigTemplateRepository) GetSharesByTemplate(ctx context.Context, templateID uuid.UUID) ([]*ConfigTemplateShare, error) {
	return m.shares[templateID], nil
}

func (m *MockConfigTemplateRepository) GetSharesByUser(ctx context.Context, userID uuid.UUID) ([]*ConfigTemplateShare, error) {
	var shares []*ConfigTemplateShare
	for _, s := range m.shares {
		for _, share := range s {
			if share.SharedWithUserID == userID {
				shares = append(shares, share)
			}
		}
	}
	return shares, nil
}

func (m *MockConfigTemplateRepository) DeleteShare(ctx context.Context, templateID, userID uuid.UUID) error {
	shares, ok := m.shares[templateID]
	if !ok {
		return ErrTemplateNotFound
	}
	for i, share := range shares {
		if share.SharedWithUserID == userID {
			m.shares[templateID] = append(shares[:i], shares[i+1:]...)
			return nil
		}
	}
	return ErrShareNotFound
}

// Tests

func TestSaveAsTemplate_Success(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	projectID := uuid.New()
	userID := uuid.New()
	req := &SaveTemplateRequest{
		ProjectID:   projectID,
		Name:        "Python Build",
		Description: "Python app build config",
		Method:      "buildx",
		TemplateData: map[string]interface{}{
			"dockerfile": "Dockerfile",
			"platforms":  []string{"linux/amd64", "linux/arm64"},
		},
	}

	template, err := service.SaveAsTemplate(context.Background(), req, userID)
	require.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "Python Build", template.Name)
	assert.Equal(t, projectID, template.ProjectID)
	assert.Equal(t, userID, template.CreatedByUserID)
	assert.Equal(t, "buildx", template.Method)
}

func TestSaveAsTemplate_InvalidMethod(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	projectID := uuid.New()
	userID := uuid.New()
	req := &SaveTemplateRequest{
		ProjectID:    projectID,
		Name:         "Bad Template",
		Method:       "invalid_method",
		TemplateData: map[string]interface{}{},
	}

	_, err := service.SaveAsTemplate(context.Background(), req, userID)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidBuildMethod, err)
}

func TestSaveAsTemplate_MissingName(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	projectID := uuid.New()
	userID := uuid.New()
	req := &SaveTemplateRequest{
		ProjectID:    projectID,
		Name:         "",
		Method:       "docker",
		TemplateData: map[string]interface{}{},
	}

	_, err := service.SaveAsTemplate(context.Background(), req, userID)
	assert.Error(t, err)
}

func TestLoadTemplate_Success(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	// First save a template
	template := &ConfigTemplate{
		ID:               uuid.New(),
		ProjectID:        uuid.New(),
		CreatedByUserID:  uuid.New(),
		Name:             "Saved Template",
		Method:           "kaniko",
		TemplateData:     map[string]interface{}{"registry": "gcr.io"},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	repo.SaveTemplate(context.Background(), template)

	// Load it
	loaded, err := service.LoadTemplate(context.Background(), template.ID)
	require.NoError(t, err)
	assert.Equal(t, template.Name, loaded.Name)
	assert.Equal(t, template.Method, loaded.Method)
}

func TestLoadTemplate_NotFound(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	_, err := service.LoadTemplate(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrTemplateNotFound, err)
}

func TestListTemplates_ByProject(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	projectID := uuid.New()
	userID := uuid.New()

	// Save multiple templates
	for i := 0; i < 3; i++ {
		req := &SaveTemplateRequest{
			ProjectID:    projectID,
			Name:         "Template " + string(rune(i+'0')),
			Method:       "docker",
			TemplateData: map[string]interface{}{},
		}
		_, err := service.SaveAsTemplate(context.Background(), req, userID)
		require.NoError(t, err)
	}

	// List templates
	templates, total, err := service.ListTemplatesByProject(context.Background(), projectID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, templates, 3)
}

func TestShareTemplate_Success(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	templateID := uuid.New()
	template := &ConfigTemplate{
		ID:              templateID,
		ProjectID:       uuid.New(),
		CreatedByUserID: uuid.New(),
		Name:            "Shared Template",
		Method:          "packer",
		TemplateData:    map[string]interface{}{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	repo.SaveTemplate(context.Background(), template)

	userID := uuid.New()
	shareReq := &ShareTemplateRequest{
		TemplateID:      templateID,
		SharedWithUserID: userID,
		CanUse:          true,
		CanEdit:         false,
		CanDelete:       false,
	}

	share, err := service.ShareTemplate(context.Background(), shareReq)
	require.NoError(t, err)
	assert.NotNil(t, share)
	assert.Equal(t, templateID, share.TemplateID)
	assert.Equal(t, userID, share.SharedWithUserID)
	assert.True(t, share.CanUse)
	assert.False(t, share.CanEdit)
}

func TestShareTemplate_TemplateNotFound(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	shareReq := &ShareTemplateRequest{
		TemplateID:       uuid.New(),
		SharedWithUserID: uuid.New(),
		CanUse:           true,
	}

	_, err := service.ShareTemplate(context.Background(), shareReq)
	assert.Error(t, err)
	assert.Equal(t, ErrTemplateNotFound, err)
}

func TestUpdateTemplate_Success(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	template := &ConfigTemplate{
		ID:              uuid.New(),
		ProjectID:       uuid.New(),
		CreatedByUserID: uuid.New(),
		Name:            "Original Name",
		Method:          "nix",
		TemplateData:    map[string]interface{}{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	repo.SaveTemplate(context.Background(), template)

	// Update
	updateReq := &UpdateTemplateRequest{
		TemplateID:  template.ID,
		Name:        "Updated Name",
		Description: "New description",
		TemplateData: map[string]interface{}{
			"flake": "github:owner/repo",
		},
	}

	updated, err := service.UpdateTemplate(context.Background(), updateReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "New description", updated.Description)
}

func TestDeleteTemplate_Success(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	template := &ConfigTemplate{
		ID:              uuid.New(),
		ProjectID:       uuid.New(),
		CreatedByUserID: uuid.New(),
		Name:            "To Delete",
		Method:          "docker",
		TemplateData:    map[string]interface{}{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	repo.SaveTemplate(context.Background(), template)

	// Delete
	err := service.DeleteTemplate(context.Background(), template.ID)
	require.NoError(t, err)

	// Verify it's gone
	_, err = repo.GetTemplate(context.Background(), template.ID)
	assert.Error(t, err)
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	err := service.DeleteTemplate(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, ErrTemplateNotFound, err)
}

func TestGetSharedTemplates_ForUser(t *testing.T) {
	repo := NewMockConfigTemplateRepository()
	service := NewConfigTemplateServiceImpl(repo)

	templateID := uuid.New()
	template := &ConfigTemplate{
		ID:              templateID,
		ProjectID:       uuid.New(),
		CreatedByUserID: uuid.New(),
		Name:            "Shared",
		Method:          "buildx",
		TemplateData:    map[string]interface{}{},
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	repo.SaveTemplate(context.Background(), template)

	userID := uuid.New()
	share := &ConfigTemplateShare{
		ID:               uuid.New(),
		TemplateID:       templateID,
		SharedWithUserID: userID,
		CanUse:           true,
		CanEdit:          true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	repo.ShareTemplate(context.Background(), share)

	// Get shared templates
	shares, err := service.GetSharedTemplatesForUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, shares, 1)
	assert.Equal(t, templateID, shares[0].TemplateID)
}
