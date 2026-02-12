package repo

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// UserRepository is an interface that provides access to user data
type UserRepository interface {
	// FindByUsername retrieves a user by username
	FindByUsername(username string) (*domain.User, error)

	// FindByID retrieves a user by ID
	FindByID(id int64) (*domain.User, error)

	// Create creates a new user
	Create(user *domain.User) error

	// Update updates a user
	Update(user *domain.User) error

	// Delete deletes a user
	Delete(id int64) error
}

// userRepository is the SQLite implementation of UserRepository
type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

// FindByUsername retrieves a user by username
func (r *userRepository) FindByUsername(username string) (*domain.User, error) {
	var user domain.User
	query := "SELECT * FROM users WHERE username = ?"

	err := r.db.Get(&user, query, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	return &user, nil
}

// FindByID retrieves a user by ID
func (r *userRepository) FindByID(id int64) (*domain.User, error) {
	var user domain.User
	query := "SELECT * FROM users WHERE id = ?"

	err := r.db.Get(&user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return &user, nil
}

// Create creates a new user
func (r *userRepository) Create(user *domain.User) error {
	query := `
		INSERT INTO users (username, password_hash, created_at, updated_at)
		VALUES (:username, :password_hash, :created_at, :updated_at)
	`

	result, err := r.db.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	user.ID = id
	return nil
}

// Update updates a user
func (r *userRepository) Update(user *domain.User) error {
	query := `
		UPDATE users
		SET username = :username, password_hash = :password_hash, updated_at = :updated_at
		WHERE id = :id
	`

	_, err := r.db.NamedExec(query, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete deletes a user
func (r *userRepository) Delete(id int64) error {
	query := "DELETE FROM users WHERE id = ?"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
