package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"natapadu-app/backend/db"
	"natapadu-app/backend/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db          *db.Database
	currentAuth *models.UserSession
}

func NewAuthService(database *db.Database) *AuthService {
	svc := &AuthService{
		db: database,
	}
	_ = svc.EnsureDefaultAdmin()
	return svc
}

// EnsureDefaultAdmin creates an initial admin account if users table is empty
func (s *AuthService) EnsureDefaultAdmin() error {
	var count int
	err := s.db.Conn().QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user := models.User{
			ID:           uuid.New().String(),
			Username:     "admin",
			PasswordHash: string(hash),
			DisplayName:  "Administrator Natapadu",
			Role:         "ADMIN",
			Status:       "ACTIVE",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err = s.db.Conn().Exec(`
			INSERT INTO users (id, username, password_hash, display_name, role, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			user.ID, user.Username, user.PasswordHash, user.DisplayName, user.Role, user.Status, user.CreatedAt, user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to seed default admin: %w", err)
		}
	}
	return nil
}

// Login verifies credentials and creates a user session
func (s *AuthService) Login(username, password string) (*models.UserSession, error) {
	var u models.User
	var createdAtStr, updatedAtStr string

	err := s.db.Conn().QueryRow(`
		SELECT id, username, password_hash, display_name, role, status, created_at, updated_at
		FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Role, &u.Status, &createdAtStr, &updatedAtStr)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("username atau password salah")
		}
		return nil, err
	}

	if u.Status != "ACTIVE" {
		return nil, errors.New("akun anda sedang dinonaktifkan")
	}

	// Verify password hash
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("username atau password salah")
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	session := &models.UserSession{
		User:      u,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	s.currentAuth = session
	return session, nil
}

// GetCurrentSession returns the active session
func (s *AuthService) GetCurrentSession() *models.UserSession {
	return s.currentAuth
}

// Logout clears the current session
func (s *AuthService) Logout() bool {
	s.currentAuth = nil
	return true
}

// GetAllUsers lists all registered users
func (s *AuthService) GetAllUsers() ([]models.User, error) {
	rows, err := s.db.Conn().Query(`
		SELECT id, username, display_name, role, status, created_at, updated_at
		FROM users ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var cr, up string
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Role, &u.Status, &cr, &up); err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, cr)
		u.UpdatedAt, _ = time.Parse(time.RFC3339, up)
		users = append(users, u)
	}
	return users, nil
}

// CreateUser adds a new user to system
func (s *AuthService) CreateUser(username, password, displayName, role string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := models.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		Role:         role,
		Status:       "ACTIVE",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = s.db.Conn().Exec(`
		INSERT INTO users (id, username, password_hash, display_name, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.DisplayName, u.Role, u.Status, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat user: %w", err)
	}

	return &u, nil
}

// UpdatePassword updates user password
func (s *AuthService) UpdatePassword(userID, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.db.Conn().Exec(`
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		string(hash), time.Now(), userID,
	)
	return err
}
