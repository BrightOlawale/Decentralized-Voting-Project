package repositories

import (
	"database/sql"
	"voting-system/internal/database"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *database.User) error {
	query := `
        INSERT INTO users (username, email, password_hash, first_name, last_name, role, permissions)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `
	result, err := r.db.Exec(query, user.Username, user.Email, user.PasswordHash,
		user.FirstName, user.LastName, user.Role, user.Permissions)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = id
	return nil
}

func (r *UserRepository) GetByUsername(username string) (*database.User, error) {
	query := `
        SELECT id, username, email, password_hash, first_name, last_name, role, 
               permissions, is_active, last_login, created_at, updated_at
        FROM users
        WHERE username = ? AND is_active = true
    `

	var user database.User
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Role, &user.Permissions,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateLastLogin(userID int64) error {
	query := `
        UPDATE users 
        SET last_login = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
	_, err := r.db.Exec(query, userID)
	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(userID int64) (*database.User, error) {
	query := `
        SELECT id, username, email, password_hash, first_name, last_name, role, 
               permissions, is_active, last_login, created_at, updated_at
        FROM users
        WHERE id = ?
    `

	var user database.User
	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Role, &user.Permissions,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*database.User, error) {
	query := `
        SELECT id, username, email, password_hash, first_name, last_name, role, 
               permissions, is_active, last_login, created_at, updated_at
        FROM users
        WHERE email = ? AND is_active = true
    `

	var user database.User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.Role, &user.Permissions,
		&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ListUsers retrieves all users with pagination and filtering
func (r *UserRepository) ListUsers(role string, isActive *bool, limit, offset int) ([]database.User, error) {
	query := `
        SELECT id, username, email, password_hash, first_name, last_name, role, 
               permissions, is_active, last_login, created_at, updated_at
        FROM users
        WHERE 1=1
    `
	args := []interface{}{}

	if role != "" {
		query += " AND role = ?"
		args = append(args, role)
	}

	if isActive != nil {
		query += " AND is_active = ?"
		args = append(args, *isActive)
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []database.User
	for rows.Next() {
		var user database.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.FirstName, &user.LastName, &user.Role, &user.Permissions,
			&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateUser updates user information
func (r *UserRepository) UpdateUser(user *database.User) error {
	query := `
        UPDATE users 
        SET username = ?, email = ?, first_name = ?, last_name = ?, 
            role = ?, permissions = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
	_, err := r.db.Exec(query, user.Username, user.Email, user.FirstName, user.LastName,
		user.Role, user.Permissions, user.IsActive, user.ID)
	return err
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, passwordHash, userID)
	return err
}

// DeactivateUser deactivates a user
func (r *UserRepository) DeactivateUser(userID int64) error {
	query := `UPDATE users SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}

// ActivateUser activates a user
func (r *UserRepository) ActivateUser(userID int64) error {
	query := `UPDATE users SET is_active = true, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}

// GetUsersByRole gets all users with a specific role
func (r *UserRepository) GetUsersByRole(role string) ([]database.User, error) {
	query := `
        SELECT id, username, email, password_hash, first_name, last_name, role, 
               permissions, is_active, last_login, created_at, updated_at
        FROM users
        WHERE role = ? AND is_active = true
        ORDER BY created_at DESC
    `

	rows, err := r.db.Query(query, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []database.User
	for rows.Next() {
		var user database.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.FirstName, &user.LastName, &user.Role, &user.Permissions,
			&user.IsActive, &user.LastLogin, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
