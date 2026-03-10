package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/srikarm/image-factory/internal/domain/mfa"
)

// MFARepository implements the mfa.Repository interface using in-memory storage for development
type MFARepository struct {
	methods    map[uuid.UUID]*mfa.MFAMethodConfig
	challenges map[uuid.UUID]*mfa.MFAChallenge
	logger     *zap.Logger
}

// NewMFARepository creates a new MFA repository
func NewMFARepository(logger *zap.Logger) *MFARepository {
	return &MFARepository{
		methods:    make(map[uuid.UUID]*mfa.MFAMethodConfig),
		challenges: make(map[uuid.UUID]*mfa.MFAChallenge),
		logger:     logger,
	}
}

// SaveMFAMethod saves an MFA method
func (r *MFARepository) SaveMFAMethod(ctx context.Context, method *mfa.MFAMethodConfig) error {
	r.methods[method.ID] = method
	r.logger.Info("MFA method saved", 
		zap.String("method_id", method.ID.String()),
		zap.String("user_id", method.UserID.String()),
		zap.String("method", string(method.Method)))
	return nil
}

// UpdateMFAMethod updates an existing MFA method
func (r *MFARepository) UpdateMFAMethod(ctx context.Context, method *mfa.MFAMethodConfig) error {
	if _, exists := r.methods[method.ID]; !exists {
		return fmt.Errorf("MFA method not found")
	}
	r.methods[method.ID] = method
	return nil
}

// DeleteMFAMethod deletes an MFA method
func (r *MFARepository) DeleteMFAMethod(ctx context.Context, id uuid.UUID) error {
	if _, exists := r.methods[id]; !exists {
		return fmt.Errorf("MFA method not found")
	}
	delete(r.methods, id)
	r.logger.Info("MFA method deleted", zap.String("method_id", id.String()))
	return nil
}

// FindMFAMethodByID finds an MFA method by ID
func (r *MFARepository) FindMFAMethodByID(ctx context.Context, id uuid.UUID) (*mfa.MFAMethodConfig, error) {
	method, exists := r.methods[id]
	if !exists {
		return nil, fmt.Errorf("MFA method not found")
	}
	return method, nil
}

// FindMFAMethodsByUserID finds all MFA methods for a user
func (r *MFARepository) FindMFAMethodsByUserID(ctx context.Context, userID uuid.UUID) ([]*mfa.MFAMethodConfig, error) {
	var methods []*mfa.MFAMethodConfig
	for _, method := range r.methods {
		if method.UserID == userID {
			methods = append(methods, method)
		}
	}
	return methods, nil
}

// FindPrimaryMFAMethod finds the primary MFA method for a user
func (r *MFARepository) FindPrimaryMFAMethod(ctx context.Context, userID uuid.UUID) (*mfa.MFAMethodConfig, error) {
	for _, method := range r.methods {
		if method.UserID == userID && method.IsPrimary {
			return method, nil
		}
	}
	return nil, fmt.Errorf("primary MFA method not found")
}

// SaveMFAChallenge saves an MFA challenge
func (r *MFARepository) SaveMFAChallenge(ctx context.Context, challenge *mfa.MFAChallenge) error {
	r.challenges[challenge.ID] = challenge
	r.logger.Info("MFA challenge saved", 
		zap.String("challenge_id", challenge.ID.String()),
		zap.String("user_id", challenge.UserID.String()))
	return nil
}

// UpdateMFAChallenge updates an existing MFA challenge
func (r *MFARepository) UpdateMFAChallenge(ctx context.Context, challenge *mfa.MFAChallenge) error {
	if _, exists := r.challenges[challenge.ID]; !exists {
		return fmt.Errorf("MFA challenge not found")
	}
	r.challenges[challenge.ID] = challenge
	return nil
}

// FindMFAChallenge finds an MFA challenge by ID
func (r *MFARepository) FindMFAChallenge(ctx context.Context, id uuid.UUID) (*mfa.MFAChallenge, error) {
	challenge, exists := r.challenges[id]
	if !exists {
		return nil, fmt.Errorf("MFA challenge not found")
	}
	return challenge, nil
}

// FindActiveChallengesByUserID finds all active challenges for a user
func (r *MFARepository) FindActiveChallengesByUserID(ctx context.Context, userID uuid.UUID) ([]*mfa.MFAChallenge, error) {
	var challenges []*mfa.MFAChallenge
	now := time.Now()
	
	for _, challenge := range r.challenges {
		if challenge.UserID == userID && 
		   challenge.Status == mfa.MFAStatusPending && 
		   challenge.ExpiresAt.After(now) {
			challenges = append(challenges, challenge)
		}
	}
	return challenges, nil
}

// DeleteExpiredChallenges deletes all expired challenges
func (r *MFARepository) DeleteExpiredChallenges(ctx context.Context) error {
	now := time.Now()
	expiredCount := 0
	
	for id, challenge := range r.challenges {
		if challenge.ExpiresAt.Before(now) {
			delete(r.challenges, id)
			expiredCount++
		}
	}
	
	if expiredCount > 0 {
		r.logger.Info("Expired MFA challenges deleted", zap.Int("count", expiredCount))
	}
	return nil
}