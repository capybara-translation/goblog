package repo

import (
	"database/sql"
	"fmt"

	"github.com/capybara-translation/goblog/internal/domain"
	"github.com/jmoiron/sqlx"
)

// UserRepository はユーザーデータへのアクセスを提供するインターフェースです
type UserRepository interface {
	// FindByUsername はユーザー名でユーザーを取得します
	FindByUsername(username string) (*domain.User, error)

	// FindByID はIDでユーザーを取得します
	FindByID(id int64) (*domain.User, error)

	// Create は新しいユーザーを作成します
	Create(user *domain.User) error

	// Update はユーザーを更新します
	Update(user *domain.User) error

	// Delete はユーザーを削除します
	Delete(id int64) error
}

// userRepository はUserRepositoryのSQLite実装です
type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository は新しいUserRepositoryを作成します
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{db: db}
}

// FindByUsername はユーザー名でユーザーを取得します
func (r *userRepository) FindByUsername(username string) (*domain.User, error) {
	var user domain.User
	query := "SELECT * FROM users WHERE username = ?"

	err := r.db.Get(&user, query, username)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}

	return &user, nil
}

// FindByID はIDでユーザーを取得します
func (r *userRepository) FindByID(id int64) (*domain.User, error) {
	var user domain.User
	query := "SELECT * FROM users WHERE id = ?"

	err := r.db.Get(&user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return &user, nil
}

// Create は新しいユーザーを作成します
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

// Update はユーザーを更新します
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

// Delete はユーザーを削除します
func (r *userRepository) Delete(id int64) error {
	query := "DELETE FROM users WHERE id = ?"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
